#!/usr/bin/env python3
"""Exercise the real WebKit/Wails window via Pika's CLI in an isolated profile.
Requires a graphical session. Does not change shortcuts or user config.
"""
import json
import os
import re
import shutil
from pathlib import Path
import subprocess
import tempfile
import time

binary = Path(__file__).resolve().parents[1] / "build/bin/pika"
# Independently observe X11 geometry, not Pika's requested/remembered position.
check_geometry = bool(os.environ.get("DISPLAY")) and all(shutil.which(tool) for tool in ("xwininfo", "xprop", "xrandr"))
geometry_checks = 0
def assert_centered():
    global geometry_checks
    if not check_geometry:
        return
    monitors = subprocess.check_output(["xrandr", "--listmonitors"], text=True, timeout=3)
    bounds = [tuple(map(int, m)) for m in re.findall(r"(\d+)/\d+x(\d+)/\d+([+-]\d+)([+-]\d+)", monitors)]
    assert bounds, "Could not read monitor geometry"
    tree = subprocess.check_output(["xwininfo", "-root", "-tree"], text=True, timeout=3)
    for line in tree.splitlines():
        match = re.match(r'\s*(0x[0-9a-f]+) "Pika":.*?\s(\d+)x(\d+)[+-]\d+[+-]\d+\s+([+-]\d+)([+-]\d+)', line)
        if not match:
            continue
        xid, width, height, x, y = match.groups()
        prop = subprocess.check_output(["xprop", "-id", xid, "_NET_WM_PID"], text=True, timeout=3)
        pid = re.search(r"= (\d+)", prop)
        if not pid:
            continue
        try:
            if Path(f"/proc/{pid[1]}/exe").resolve(strict=True) != binary:
                continue
        except FileNotFoundError:
            continue
        width, height, x, y = map(int, (width, height, x, y))
        assert any(abs(x - (mx + (mw-width)//2)) <= 1 and abs(y - (my + (mh-height)//2)) <= 1
                   for mw, mh, mx, my in bounds), f"Visible Pika not centered: {(x,y,width,height)}, monitors={bounds}"
        geometry_checks += 1
        return
    raise AssertionError("Could not find isolated Pika's native window")
with tempfile.TemporaryDirectory(prefix="pika-smoke-") as tmp:
    root = Path(tmp)
    env = os.environ.copy()
    for key, name in [("XDG_RUNTIME_DIR", "runtime"), ("XDG_CONFIG_HOME", "config"),
                      ("XDG_DATA_HOME", "data"), ("XDG_STATE_HOME", "state"), ("XDG_CACHE_HOME", "cache")]:
        path = root / name
        path.mkdir(mode=0o700)
        env[key] = str(path)
    def cli(command, check=True):
        return subprocess.run([str(binary), command], env=env, capture_output=True, text=True, timeout=8, check=check)
    def wait_focus(visible):
        deadline = time.monotonic() + 3
        last = {}
        while time.monotonic() < deadline:
            last = json.loads(cli("focus-state").stdout)
            if visible and last["visible"] and last["mapped"] and last["surface_ready"]:
                assert_centered()
            if visible and all(last.values()):
                return last
            if not visible and not last["mapped"] and not last["visible"] and not last["window_active"]:
                return last
            time.sleep(.05)
        raise AssertionError(f"Focus state did not settle (visible={visible}): {last}")
    with (root / "native.log").open("w+") as log:
        started = time.perf_counter()
        gui = subprocess.Popen([str(binary), "--background"], env=env, stdout=log, stderr=log)
        try:
            deadline = time.monotonic() + 12
            while time.monotonic() < deadline:
                result = cli("stats", check=False)
                if result.returncode == 0:
                    break
                if gui.poll() is not None:
                    raise RuntimeError("Native process exited before frontend readiness")
                time.sleep(.1)
            else:
                raise RuntimeError("Native frontend did not become ready")
            ready_ms = (time.perf_counter() - started) * 1000
            stats = json.loads(result.stdout)
            initial = wait_focus(False)
            assert not initial["surface_ready"], "Cold GTK surface was exposed before the first frontend paint"
            # This is the same IPC toggle command as the Cinnamon shortcut.
            # Check native keyboard ownership AND the WebKit search input.
            for _ in range(10):
                cli("toggle")
                wait_focus(True)
                cli("hide")
                wait_focus(False)
            for _ in range(100):
                cli("toggle")
            # A pending fade/fallback from an older hide must not unmap a new show.
            cli("show")
            wait_focus(True)
            time.sleep(.45)
            wait_focus(True)
            fade_started = time.perf_counter()
            cli("hide")
            wait_focus(False)
            hide_ms = (time.perf_counter() - fade_started) * 1000
            assert hide_ms >= 80, f"Hide skipped its exit animation: {hide_ms:.1f} ms"
            cli("reload-config")
            cli("reindex")
            assert gui.poll() is None, "Window lifecycle killed the process"
            cli("quit")
            gui.wait(timeout=8)
            assert gui.returncode == 0, "Native process failed during shutdown"
            # Test the shortcut path when there is no running GUI instance.
            cli("toggle")
            wait_focus(True)
            cli("stats")
            cli("hide")
            wait_focus(False)
            cli("quit")
            deadline = time.monotonic() + 8
            while (root / "runtime/pika/control.sock").exists() and time.monotonic() < deadline:
                time.sleep(.05)
            assert not (root / "runtime/pika/control.sock").exists(), "Cold-start process did not shut down"
            print(json.dumps({"frontend_ready_ms": round(ready_ms, 1), "toggle_calls": 100,
                              "apps": stats["apps"], "focus_cycles": 10, "cold_toggle": "search focused",
                              "startup_surface": "guarded", "hide_ms": round(hide_ms, 1),
                              "stale_hide": "cancelled", "shutdown": "clean"}, indent=2))
            print(f"Native geometry checks: {geometry_checks}" if check_geometry else "Native geometry checks: skipped (X11 tools unavailable)")
        finally:
            if gui.poll() is None:
                cli("quit", check=False)
                try:
                    gui.wait(timeout=4)
                except subprocess.TimeoutExpired:
                    gui.terminate()
                    gui.wait(timeout=4)
            log.seek(0)
            text = log.read()
            if text:
                print(text[-4000:])

#!/usr/bin/env python3
"""Exercise the real WebKit/Wails window via Pika's CLI in an isolated profile.
Requires a graphical session. Does not change shortcuts or user config.
"""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

binary = Path(__file__).resolve().parents[1] / "build/bin/pika"
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

#!/usr/bin/env python3
"""Install per-user Pika assets. No sudo; preserve config and usage state."""
import os
from pathlib import Path
import shutil
import sys
import tempfile

repo = Path(__file__).resolve().parents[1]
home = Path.home()
def xdg(key, fallback):
    value = Path(os.environ.get(key, str(home / fallback)))
    if not value.is_absolute():
        raise SystemExit(f"{key} must be absolute")
    return value

data = xdg("XDG_DATA_HOME", ".local/share")
config = xdg("XDG_CONFIG_HOME", ".config")
binary = home / ".local/bin/pika"
desktop = data / "applications/pika.desktop"
icon = data / "icons/hicolor/scalable/apps/pika.svg"
autostart = config / "autostart/pika.desktop"
manifest = data / "pika/installation.txt"

def quote_exec(path):
    # Desktop Entry escaping, including the extra escaping layer for Exec.
    value = str(path).replace("\\", "\\\\\\\\").replace('"', '\\"').replace("`", "\\`").replace("$", "\\$").replace("%", "%%")
    return '"' + value + '"'

def entry(background=False):
    return f"""[Desktop Entry]
Type=Application
Name=Pika
Comment=Search apps, files and personal commands
Exec={quote_exec(binary)} {'--background' if background else 'show'}
Icon=pika
Terminal=false
Categories=Utility;
StartupWMClass=pika
X-Pika-Managed=true
"""

def atomic_write(path, content, mode=0o644):
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, temp = tempfile.mkstemp(prefix=".pika-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(content)
        os.chmod(temp, mode)
        os.replace(temp, path)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)

def remove_entry(path):
    if path.exists():
        if "X-Pika-Managed=true" not in path.read_text():
            raise SystemExit(f"Refusing to remove unmanaged entry: {path}")
        path.unlink()

def ensure_managed(path):
    if path.exists() and "X-Pika-Managed=true" not in path.read_text():
        raise SystemExit(f"Existing unmanaged desktop entry: {path}")

mode = sys.argv[1] if len(sys.argv) > 1 else ""
if mode == "install":
    source = repo / "build/bin/pika"
    if not source.is_file():
        raise SystemExit("Run make build first")
    ensure_managed(desktop)
    if binary.exists() and not manifest.exists():
        raise SystemExit(f"Existing binary was not installed by this script: {binary}")
    atomic_write(binary, source.read_bytes(), 0o755)
    atomic_write(icon, (repo / "build/pika.svg").read_bytes())
    atomic_write(desktop, entry().encode())
    atomic_write(manifest, (str(binary) + "\n" + str(icon) + "\n").encode(), 0o600)
    print(f"Installed: {binary}\nRun: {binary} show\nAutostart (optional): make autostart\nCinnamon shortcut command: {binary} toggle")
elif mode == "autostart":
    if not binary.exists():
        raise SystemExit("Run make install first")
    ensure_managed(autostart)
    atomic_write(autostart, entry(background=True).encode())
    print(f"Autostart enabled: {autostart}")
elif mode == "autostart-off":
    remove_entry(autostart)
    print("Autostart disabled")
elif mode == "uninstall":
    # Validate entries before removing anything.
    ensure_managed(desktop)
    ensure_managed(autostart)
    if not manifest.exists():
        raise SystemExit("No Pika installation manifest; nothing removed")
    remove_entry(desktop)
    remove_entry(autostart)
    binary.unlink(missing_ok=True)
    icon.unlink(missing_ok=True)
    manifest.unlink()
    print("Uninstalled. Config and usage database preserved. Remove the Cinnamon shortcut manually.")
else:
    raise SystemExit("Usage: install.py install|uninstall|autostart|autostart-off")

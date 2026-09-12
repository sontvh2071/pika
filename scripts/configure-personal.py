#!/usr/bin/env python3
"""Prepare/apply the requested Mint + wallpaper palette and personal search roots.
Preserves unrelated config fields; validates with the newly built Pika binary.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import time
import tomllib

repo = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--apply', action='store_true', help='Back up and update the personal config')
parser.add_argument('--output', type=Path, default=repo / 'docs/config.personal.toml')
args = parser.parse_args()
personal_home = Path.home()
config_base = Path(os.environ.get('XDG_CONFIG_HOME', str(personal_home / '.config')))
if not config_base.is_absolute():
    config_base = personal_home / '.config'
target = (config_base / 'pika/config.toml').resolve()
original = target.read_bytes() if target.exists() else (repo / 'docs/config.example.toml').read_bytes()
body = original.decode()
tomllib.loads(body)
editor = shutil.which('code')
if not editor:
    raise SystemExit('VS Code executable `code` was not found; install it before applying this profile.')

def set_value(section, key, value):
    global body
    lines = body.splitlines(keepends=True)
    start, end = None, len(lines)
    for i, line in enumerate(lines):
        text = line.split('#', 1)[0].strip()
        if text.startswith('['):
            if start is not None:
                end = i
                break
            if text == f'[{section}]':
                start = i + 1
    replacement = f'{key} = {json.dumps(value, ensure_ascii=False)}\n'
    if start is None:
        body = body.rstrip() + f'\n\n[{section}]\n' + replacement
        return
    for i in range(start, end):
        if re.match(r'^\s*' + re.escape(key) + r'\s*=', lines[i]):
            lines[i] = replacement
            body = ''.join(lines)
            return
    lines.insert(start, replacement)
    body = ''.join(lines)

set_value('appearance', 'theme', 'custom')
set_value('appearance', 'inner_inset', 8)
set_value('appearance', 'font_size', 16)
for key, value in dict(background='#101010', surface='#151515', text='#f3f3f3', muted='#b8b8b8',
                       selection='#343434', accent='#ffffff', border='#535353', error='#d0d0d0').items():
    set_value('appearance.colors', key, value)
set_value('search', 'include_commands', False)
set_value('index', 'roots', [str(personal_home)])
exclusions = tomllib.loads(body).get('index', {}).get('exclude_paths', [])
excluded_go = [str(personal_home / 'go'), str(personal_home / 'workspace/go')]
set_value('index', 'exclude_paths', list(dict.fromkeys([*exclusions, *excluded_go])))
set_value('index', 'max_depth', 32)
set_value('index', 'max_candidates', 100000)
set_value('watcher', 'max_directories', 16384)
set_value('open', 'workspace_root', str(personal_home / 'workspace'))
set_value('open', 'workspace_executable', editor)
# Cache/dependency exclusions and include_hidden follow the existing user config.
tomllib.loads(body)
with tempfile.TemporaryDirectory(prefix='pika-profile-check-') as tmp:
    validation_config = Path(tmp) / 'pika/config.toml'
    validation_config.parent.mkdir()
    validation_config.write_text(body)
    env = dict(os.environ, XDG_CONFIG_HOME=tmp)
    subprocess.run([str(repo / 'build/bin/pika'), '--check-config'], env=env, check=True,
                   capture_output=True, text=True)
args.output.write_text(body)
print('Validated profile:', args.output)
if args.apply:
    target.parent.mkdir(parents=True, exist_ok=True)
    if target.exists() and target.read_bytes() != original:
        raise SystemExit('Config changed during preparation; retry.')
    backup = target.with_name(target.name + '.before-personal-' + time.strftime('%Y%m%d-%H%M%S'))
    if target.exists():
        with backup.open('xb') as stream:
            stream.write(original)
        backup.chmod(0o600)
    with tempfile.NamedTemporaryFile(mode='w', dir=target.parent, prefix='.pika-profile-', delete=False) as stream:
        temporary = Path(stream.name)
        try:
            stream.write(body)
            stream.flush()
            os.fsync(stream.fileno())
            if target.exists() and target.read_bytes() != original:
                raise SystemExit('Config changed while saving; retry.')
            os.replace(temporary, target)
        finally:
            temporary.unlink(missing_ok=True)
    print('Applied:', target)
    print('Backup:', backup)
print('Profile SHA256:', hashlib.sha256(body.encode()).hexdigest())

# Pika

A personal, local-first launcher for Linux Mint / Cinnamon, built with Go + Wails + TypeScript.

Tokyo Night search and two result/detail panels, separated by **12 px**, on a transparent native window. Search applications, opt-in files/folders and configured commands; launch with the keyboard; keep usage ranking on your machine.

## Quick start

```sh
make deps
make build
./build/bin/pika
```

Requires Go compatible with `go.mod`, Node 22.12+ (22.x), Wails 2.15.0, GTK 3 and WebKitGTK 4.1. Build/dev commands include `-tags webkit2_41` for Linux Mint 22.x.

## Personal installation

```sh
make install
~/.local/bin/pika show
make autostart  # optional
```

Set a Cinnamon custom shortcut to `/home/lilmint/.local/bin/pika toggle`, then bind Alt+Space. Adapt the home path if installing for a different user. Keep Super+Space for IBus.

**[Hướng dẫn setup và sử dụng chi tiết bằng tiếng Việt](docs/usage.vi.md)** — dependencies, hotkey conflicts, config, themes, file roots, custom commands, upgrades and troubleshooting.

- [Current config example](docs/config.example.toml)
- [Validation and known limitations](docs/validation.md)
- [Original implementation plan](docs/implementation-plan.vi.md)

## Keyboard

Up/Down select · Enter opens · Escape hides · Ctrl+, opens settings · Ctrl+Shift+, reloads config · Ctrl+R reindexes.

## Development

```sh
make dev
make test
make race
make bench
```

`make preview` serves a **visual preview with sample data**. Run the Wails desktop build to search real items and open applications. File search is opt-in: add roots in `~/.config/pika/config.toml` and reload. Config and usage survive reinstall/uninstall.

Current release: **0.1.0**. Core features are implemented; long-term daily-use validation and the full v1 performance targets in the original plan remain to be measured.

Personal Mint profile (monochrome black/white, home search excluding ~/go and ~/workspace/go, workspace → VS Code): see [the Vietnamese guide](docs/usage.vi.md) and [config.personal.toml](docs/config.personal.toml). Prepare with `python3 scripts/configure-personal.py`; apply with `--apply` after building.

The full native window stays centered at its configured size. Its 40px search bar stays fixed at the top of that frame; result/detail panels appear below only for a nonblank query with matches. Empty and no-match states hide the panels without resizing or recentering the window. The unused area is transparent; settings uses the same frame.

Startup guards the native GTK surface until the styled frontend is ready. Show/hide uses a reversible slide-and-scale transition (120/80 ms): expand from 92% scale and 12px below the resting position, then retract along the same path. The top-center origin keeps it inside the native frame. Reduced motion skips it; rapid toggles cancel stale hides and preserve the current pose. At rest, text is untransformed and the search anchor stays fixed while results change. Motion regressions: `cd frontend && npm test` (Node 22.6+); real window lifecycle: `python3 scripts/native-smoke.py`.


Built-in `lock`, `logout`, and `shutdown` search results use Cinnamon's screen lock and native session dialogs; they remain available with personal commands disabled. Desktop apps launch in-process through GIO with a live parent for polkit authentication. The personal profile uses opaque monochrome panels, firmer type, and 8px panel spacing.


ChatGPT/Codex app details include live remaining 5-hour and weekly quotas, reset times and available reset credits from the official Codex app-server API. Refreshes every 30 seconds while selected; no reset is consumed and no agent turn is created. See the Vietnamese guide for troubleshooting.

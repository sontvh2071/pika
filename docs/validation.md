# Validation — Pika 0.1.0

Tested on 2026-09-11, Linux Mint 22.3 / Cinnamon X11, Intel Core i7-3770, GTK 3.24.41, WebKitGTK 4.1 (2.52.6), Go 1.27.1, Wails 2.15.0, Node 22.23.2.

## Completed

- `wails build -tags webkit2_41`: release binary generated at `build/bin/pika` (approximately 11 MiB).
- Frontend TypeScript type check and Vite release build passed.
- `go test -race ./internal/...`: search/ranking, Vietnamese normalization, desktop entry precedence, file exclusions and symlink handling, config validation, IPC ownership/idempotency, SQLite persistence, watcher refresh and concurrent search/usage/reindex passed.
- `go vet -tags webkit2_41 ./...`: passed.
- Native Wails/WebKit smoke: isolated config/data/runtime profile, frontend readiness, 100 toggle commands, hide, reload-config, reindex and clean quit passed. Cold `pika toggle` also started a new instance successfully after the previous process exited. Runs discovered 98 applications; frontend readiness was about 1.1 s initially and 0.53 s in the final run (individual observations, not percentiles).
- Installer exercised in a temporary home/profile: install, reinstall, enable/disable autostart, uninstall. User config preserved.
- Look layout redesign: browser preview inspected at 800×660, 720×520 and minimum 480×380. Both panels and the Settings footer remain within the viewport; result/details panels scroll independently. Those measurements preceded the later removal of the outer wrapper.
- Interaction checks: ArrowDown updates selection and details together; single click selects without launching; quick search filters results; empty search results hide stale details; command prefix filters the preview; the action button reports preview limitations. Settings opens and closes. The theme bar and gradient background were subsequently removed at the user’s request; themes remain configurable through TOML.
- Theme persistence tests cover all supported palettes, preserving commands and comments, inserting missing keys/sections, and rejecting invalid configs/themes without writing.
- Redesigned release native smoke: 98 apps, frontend ready in 664 ms, 100 toggle calls, reload/reindex, cold toggle and clean shutdown passed. Browser checks use preview data; native smoke covers lifecycle rather than pixel comparison or physical IME input.

## Search microbenchmark

Fixture: generated prepared file candidates, fuzzy query `prj`, top 10. These numbers measure only the Go engine, not IPC, UI rendering or the Cinnamon shortcut.

| Dataset | Mean time/op | Bytes/op | Allocs/op |
|---|---:|---:|---:|
| 10,000 candidates | 3.018 ms | 2,840 | 4 |
| 50,000 candidates | 15.473 ms | 2,840 | 4 |

Reproduce with `make bench`. These are benchmark averages, **not p95 latency guarantees**. The original plan's complete resource/performance budgets have not been established.

## Current behavior and limits

- App search, custom commands, configured file/folder roots, usage ranking, pins, config reload, six configurable dark themes plus light/custom themes, IPC and bounded directory watches are implemented.
- Files are opt-in. Search covers names/paths, not file contents. Recursive file walking does not follow directory symlinks.
- Watch events coalesce into a full background refresh of configured sources. Per-subtree/delta indexing from the long-term plan remains future work. Keep roots bounded and exclude dependency trees.
- Some desktop icon themes use paths not covered by the initial resolver; those apps receive a generic icon.
- Config is edited in a text editor. Settings exposes status, edit/reload config, reindex and quit; it is not a full graphical config editor.
- Minimal layout: only the search bar and two result/detail panels are visible. No outer background, border, footer or quick-query chips. `inner_inset` now sets panel spacing (12 px default). Linux `WindowIsTranslucent` and an alpha-zero WebView background enable transparent gaps/corners; compositor support is required. No desktop blur.
- Minimal layout browser check at 720×520: search at (0,0), result panels at y=52, widths 354 each with 12 px spacing; html/body/app/launcher/inner-frame backgrounds all rgba(0,0,0,0), no outer border or shadow. Ctrl+, opens Settings; ArrowDown updates details. Native lifecycle smoke passed with 98 apps, 512.4 ms readiness and 100 toggles. Desktop alpha compositing was configured via Wails but not visually measured by this browser check.
- Runtime/native smoke validates frontend readiness and CLI lifecycle, not physical key delivery from every app. Manually verify Cinnamon Alt+Space, focus, your IBus/IME, multiple monitors, suspend/resume and logout/login after installation.
- The user's real shortcut, autostart and installation were not changed during automated validation. Native tests used temporary profiles.
- Seven-day daily-use testing, sustained CPU/total PSS measurements and end-to-end latency percentiles remain outstanding before calling this a fully validated v1.

Use `docs/config.example.toml` and `docs/usage.vi.md` for the implemented configuration schema. The earlier architecture plan contains proposed fields that are not all part of version 0.1.0.

## Personal polish — 2026-09-12

- Reference inspected: Look commit `3392db65c59ebdc149b1ae25d78a41f960de46a2`, especially [motion.css](https://github.com/kunkka19xx/look/blob/3392db65c59ebdc149b1ae25d78a41f960de46a2/apps/linows/src/css/motion.css), [results.css](https://github.com/kunkka19xx/look/blob/3392db65c59ebdc149b1ae25d78a41f960de46a2/apps/linows/src/css/components/results.css) and [preview.js](https://github.com/kunkka19xx/look/blob/3392db65c59ebdc149b1ae25d78a41f960de46a2/apps/linows/src/js/components/preview.js). Pika implements its own motion and rendering code, using transform/opacity, a sliding selection layer and retained rows.
- Desktop read: Mint-L-Dark GTK/Cinnamon, Mint-L icons, Ubuntu 10, GTK selection `#8fa876`; wallpaper black/gray/copper wormhole astronaut. Personal palette applied as custom, not automatic wallpaper tracking. Outer native surface remains transparent.
- Build/TypeScript, `go test -race ./internal/...` and `go vet -tags webkit2_41 ./...` passed. Tests include home scope, symlink folder classification, exact workspace path boundaries, literal argv, disabled commands, package version parsing and ignoring the Desktop Entry specification Version.
- `go run ./scripts/check-profile`: actual personal profile indexed 99 apps and 30,041 files/folders in 1,070 ms, 0 commands, no index warnings. This diagnostic disables watchers and uses a temporary SQLite DB. Downloads/Desktop/Pictures route to file manager; workspace routes to VS Code. VS Code package version resolved as `1.136.1-1788413865`. These counts/times are one observation and change with filesystem contents.
- Native isolated smoke passed 100 toggles, cold start and shutdown, with frontend ready in 611.3 ms. No 60 fps or end-to-end latency percentile claim is made; browser inspection confirms transition configuration, not GPU frame timing.
- Personal config was backed up before applying the new colors, home root, workspace editor and commands-disabled profile. Exclusions and existing window size retained.
- Browser: metadata labels, workspace/normal-folder actions, selection/details synchronization, hidden Commands tab, and empty results verified. Version absence displays honestly; preview has no installed package database.

- Final native personal instance: 99 apps, 30,041 files/folders, 0 commands, 6,191 watches, no warnings/state error, last scan 647 ms. Narrow 480×380 preview has no page overflow; Settings and full path/action remain available. Reindex status updates retain the selected details instead of replaying their animation.

## Compact search and exact exclusion — 2026-09-12

- Personal surface changed to neutral near-black `#181818`, selection `#303030`; no green tint in either color.
- Frontend hides both lower panels for blank/whitespace queries or zero matches. Empty queries do not invoke catalog search; Enter cannot execute a hidden suggestion. Settings expands separately.
- Native window starts at 40 px height, expands to configured height for results/Settings, and recenters on show and after GTK resize has reached the frontend. This avoids centering a large transparent rectangle while displaying only its top search bar.
- Browser at 720×520: compact bar bounds x=0,y=240,width=720,height=40, exactly vertically centered; matching query opens both panels; no-match/clear/whitespace collapse them. Settings round-trip and 480×380 expanded layout verified, no page overflow. Native positioning across physical monitors was not visually measured by the browser check.
- `index.exclude_paths` prunes an exact directory and subtree from candidates and directory watches. Tests include explicit excluded roots and verify `workspace/go-tools`, `Documents/go` and unrelated projects remain indexed. Config validation rejects relative paths without ~/.
- Race tests, frontend/release build and Go vet passed. Native smoke: readiness 545.5 ms, 100 toggles, cold toggle and clean exit. Personal diagnostic confirms configured exclusion has no candidate; 99 apps/30,041 files indexed without warnings in this observed run.
- Config backup created before updating personal colors and adding `/home/lilmint/workspace/go` to exclusions.

## Whole-launcher centering — 2026-09-12

- Look reference: `recenter_window` in `apps/linows/src-tauri/src/main.rs` computes position from the complete expected window bounds. Pika preview now represents the same full-window coordinate model rather than stretching results across the browser viewport.
- Preview measurement at 1280×900: compact bounds (280,430,720,40), center (640,450); expanded bounds (280,190,720,520), same center (640,450). Combined search-top to results-bottom has ~190 px clearance above/below (subpixel difference during the 180 ms entry animation). No-match returns to the same centered compact bounds.
- At 480×380, preview clamps to 456×356 with center (240,190), no overflow. Native remains governed by its own window bounds; preview-only stage margins are not applied to the desktop.
- Native resize acknowledgements include viewport width/height. Center requests for previous dimensions are ignored, and centering occurs after the requested full size is allocated rather than centering an old-height window immediately after requesting resize.
- Release/TypeScript build and Go vet passed. Native lifecycle smoke separately checks startup and toggling; browser geometry measurements are not a visual measurement of the native window manager.


## 2026-09-12 — Fixed search anchor, monochrome palette, both Go exclusions

This revision supersedes the earlier compact-window/recentering behavior. Native startup now uses the configured full frame (personal profile: 720 × 520); search transitions only hide/show the result panels. `SetExpanded` and its resize queue were removed. Search entrance animates opacity only.

- Browser preview at 1280 × 900: search bounds stayed x=280, y=190, width=720, height=40 for empty query, a matching query, no matches, clearing, whitespace and returning from Settings. Full frame remained 720 × 520 centered. Result panels were hidden for empty/no-match/whitespace states.
- Preview screenshot inspected; UI palette uses grayscale. Hovered selected result text measured rgb(229, 229, 229). Personal config and preview both use accent #f0f0f0, surface #181818 and selection #303030.
- Personal profile excludes both /home/lilmint/go and /home/lilmint/workspace/go. Regression fixture checks descendants cannot become candidates or watches, even if explicitly added as roots; go-tools and Documents/go remain searchable.
- `go test -race ./internal/...`, `go vet -tags webkit2_41 ./...`, and `wails build -tags webkit2_41` passed.
- `python3 scripts/native-smoke.py`: frontend ready 526.3 ms, 100 toggle calls, cold toggle ready, clean shutdown. Native physical screen coordinates were not visually measured; geometry above was measured in the browser preview.
- Applied the profile with a backup, then restarted build/bin/pika. Live stats: 99 apps, 96 files/folders, 36 watched directories, 102 ms scan, no warnings or state error. Read-only profile diagnostic confirmed excluded roots absent, Downloads/Desktop/Pictures open with the file manager, workspace opens with VS Code.


## 2026-09-12 — Alt+Space keyboard focus

Reproduced against the installed background process: Pika was mapped and always-on-top, while `_NET_ACTIVE_WINDOW` remained Brave. Wails WindowUnminimise calls gtk_window_present without a fresh input timestamp; the IPC shortcut provides no GTK key event. DOM input.focus alone could not obtain desktop keyboard ownership.

Added a GTK-main-thread activation helper using a fresh X11 server timestamp, gtk_window_present_with_time and WebView widget focus. Pending activation is invalidated on hide/quit. The frontend finishes focusing search on the window focus event, with hidden/settings guards and no repeated focus stealing timer. The focus-state CLI diagnostic reports native and DOM focus booleans without query text.

Validation: Wails production build, go vet and internal race tests passed. Native smoke passed 10 show/hide focus cycles (all six focus flags true on show), 100 rapid toggles, cold toggle with query focus, and clean shutdown; frontend ready 564 ms. Tests use the same IPC command as the Cinnamon shortcut and inspect native/DOM state; they do not synthesize physical Alt+Space or IME text.

GTK reference: https://docs.gtk.org/gtk3/method.Window.present_with_time.html


## 2026-09-12 — Polkit launch, Cinnamon actions, stronger monochrome UI

Reproduced Login Window failure with `gtk-launch lightdm-settings.desktop`: exit 0 but stderr `Refusing to render service to dead parents.` Replaced the short-lived gtk-launch helper with in-process GIO desktop launching using G_SPAWN_DO_NOT_REAP_CHILD and a Go waiter per child. The launcher now hides before handing off to app/auth/session dialogs. Regression test launches a real harmless desktop probe and asserts its parent PID equals the Pika/test host and literal desktop arguments are preserved. Authentication completion with the user's password was not performed.

Added built-in Lock, Log Out and Shut Down candidates independent of personal commands. Exact action keywords outrank matching application keywords (including Screensaver's `lock` keyword). Unit tests validate matching and exact Cinnamon argv; no lock, logout, suspend or shutdown was executed during validation.

Browser preview checked all three action details/icons, opaque rgb(21,21,21) surfaces, 8px gap, 600-weight titles, and hidden panels after clearing. Search remains anchored. Removed scale animation on the text-bearing workspace and made type larger/stronger.

Passed: internal race tests, go vet, Wails production build. Native smoke: 10 full focus cycles, 100 rapid toggles, cold launch search focus and clean shutdown, frontend readiness 438.4 ms. Real-profile diagnostic: 100 apps, 102 files/folders, 3 system actions, no warnings; confirms common home folders, VS Code routing and Go exclusions. Applied personal profile with backup and installed to ~/.local/bin/pika.

Sources: https://docs.gtk.org/gio-unix/method.DesktopAppInfo.launch_uris_as_manager.html and installed Cinnamon menu/session-quit source for the native session commands.

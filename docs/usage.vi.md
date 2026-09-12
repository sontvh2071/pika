# Pika — Cài đặt và sử dụng trên Linux Mint

Pika 0.1.0 là launcher cá nhân dùng Go, Wails và TypeScript. Giao diện bám bố cục Look trong ảnh tham chiếu: chỉ có ô tìm kiếm cùng hai cột kết quả và chi tiết màu đen–trắng; vùng bên ngoài các khung trong suốt. Mã giao diện được triển khai riêng trong repo này.

Giao diện đã bỏ nền và viền bao ngoài, thanh theme, chân trang và các chip Try. `appearance.inner_inset = 8` hiện điều chỉnh khoảng cách 8px giữa các khung. `radius` điều chỉnh bo góc từng khung, tối đa 12px trong bố cục gọn này.

## Cấu hình cá nhân đã áp dụng ngày 12/09/2026

Cửa sổ giữ cố định kích thước `window.width × window.height` (profile hiện tại: 720 × 520), căn giữa theo toàn bộ bố cục. Thanh search cao 40px luôn nằm ở đầu khung đó, giữ nguyên vị trí cả khi chưa gõ, có kết quả hoặc không có kết quả. Chỉ khi truy vấn có nội dung và có kết quả mới hiện hai cột bên dưới; xóa query, nhập toàn khoảng trắng hoặc không có kết quả sẽ ẩn hai cột, không đổi kích thước hay căn giữa lại. Phần trống bên dưới trong suốt. Settings vẫn mở bằng `Ctrl+,`. Palette đen–trắng dùng nền đặc `#151515`, highlight xám `#343434`, chữ `#f3f3f3`, điểm nhấn `#ffffff`; hover và focus cũng dùng thang xám. Font Ubuntu như desktop. Đây là palette cố định, không tự đổi theo wallpaper.

- Search ứng dụng Linux và file/thư mục trong `/home/lilmint`. Profile này tắt commands bằng `search.include_commands = false`; tab Commands và Ctrl+4 không hoạt động trong profile này.
- `/home/lilmint/workspace` và mọi đường dẫn con mở bằng `/usr/bin/code --reuse-window -- <path>`, gồm cả file và folder. Việc khớp dựa trên thành phần đường dẫn: `workspace-backup` không thuộc `workspace`.
- File ngoài workspace mở bằng `xdg-open`, dùng ứng dụng mặc định theo loại file. Folder ngoài workspace mở bằng file manager mặc định; trên máy này là Nemo (`nemo.desktop`). Không sửa file association của hệ điều hành.
- Cột phải hiển thị **Kind, Path, Version** và nút mở tương ứng. Path của app là file `.desktop` đã được index; file/folder dùng đường dẫn đầy đủ. Version app được đọc bất đồng bộ từ Debian package, Flatpak hoặc metadata AppImage khi có; không chạy lệnh Exec của app để đoán version. `Not available` = không lấy được version; file/folder hiện `Not applicable`.
- Highlight trượt 230 ms, icon nảy nhẹ 230 ms, cửa sổ xuất hiện 260–300 ms. Kết quả giữ nguyên trong lúc tìm, tái sử dụng các hàng còn khớp và cập nhật ngay khi có phản hồi; không chạy lại hiệu ứng cả danh sách sau mỗi phím gõ. Chờ 25 ms để gom gõ nhanh, ngăn mở kết quả cũ khi đang tìm. Tôn trọng `prefers-reduced-motion`.

Profile đã áp dụng: `~/.config/pika/config.toml`. Bản tham chiếu trong repo: [config.personal.toml](config.personal.toml). Script tạo bản sao lưu cạnh config trước khi ghi; tên bắt đầu `config.toml.before-personal-`.

Nếu cần áp dụng lại sau này:

```sh
cd /home/lilmint/workspace/me/pika
make build
python3 scripts/configure-personal.py --apply
./build/bin/pika quit
./build/bin/pika show
```

Chuẩn bị profile để xem trước mà chưa ghi vào config cá nhân:

```sh
python3 scripts/configure-personal.py
```

Kiểm tra index và luật mở mà không launch candidate:

```sh
go run ./scripts/check-profile
```

Profile thêm `index.exclude_paths = ["/home/lilmint/go", "/home/lilmint/workspace/go"]`, bỏ cả hai thư mục đó và mọi nội dung bên trong khỏi candidates và watches. Các thư mục như `workspace/go-tools` hoặc `Documents/go` vẫn được tìm. Profile giữ các exclusions đang có: file/thư mục ẩn, `.git`, `node_modules`, `.cache`, `vendor`, `dist`, `build`, `.venv` được loại trừ theo config. Độ sâu tối đa 32, giới hạn 100.000 candidates và 16.384 directory watches. Không theo symlink thư mục khi index; bản thân symlink vẫn tìm thấy và mở theo đường dẫn đã chọn. Cửa sổ Settings (`Ctrl+,`) báo nếu gặp giới hạn hoặc đường dẫn không đọc được.

## 1. Chuẩn bị môi trường

Môi trường đã dùng để build/kiểm thử: Linux Mint 22.3, Cinnamon/X11, x86-64, GTK 3, WebKitGTK 4.1, Node 22.23.2, Wails 2.15.0. Repo khai báo Go 1.25; lần kiểm thử dùng Go 1.27.1.

Kiểm tra công cụ:

```sh
go version
node --version
npm --version
wails version
pkg-config --modversion gtk+-3.0 webkit2gtk-4.1
command -v gtk-launch
command -v xdg-open
```

Nếu thiếu native dependencies trên Mint 22.x:

```sh
sudo apt update
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev libgtk-3-bin xdg-utils python3
```

Dùng Go phù hợp `go.mod`, Node 22.12+ trong nhánh 22 (máy hiện có 22.23.2), và cài Wails đúng phiên bản:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
```

Nếu `wails` không tìm thấy, thêm thư mục `go/bin` của bạn vào PATH trong cấu hình shell. Ví dụ với đường dẫn mặc định:

```sh
export PATH="$PATH:$HOME/go/bin:$HOME/.local/bin"
```

Có thể thêm dòng đó vào `~/.zshrc` nếu dùng zsh hoặc `~/.bashrc` nếu dùng bash. Shortcut Cinnamon ở phần sau dùng absolute path nên không phụ thuộc shell PATH.

## 2. Build và chạy lần đầu

```sh
cd /home/lilmint/workspace/me/pika
make deps
make build
./build/bin/pika
```

Lần chạy đầu sinh `~/.config/pika/config.toml`. Ô tìm kiếm tự nhận focus; ứng dụng được thu thập tự động. Config mới từ lần chạy đầu chưa bật file roots; script profile cá nhân ở phần trên bật `/home/lilmint`.

`make build` tương đương `wails build -tags webkit2_41`. Không bỏ tag trên máy Mint này: WebKitGTK 4.0 không có trong môi trường đã kiểm tra.

Trong lúc phát triển:

```sh
make dev
```

Nên thoát bản release trước khi chạy dev, vì cả hai dùng chung single-instance socket/profile. Không chạy thêm `npm run dev` độc lập cùng lúc với Wails dev trừ khi chỉ muốn xem preview.

Chạy riêng preview:

```sh
make preview
```

Preview tại `http://127.0.0.1:5173` mô phỏng một cửa sổ 720×520 nằm giữa vùng trình duyệt, thu về 720×40 khi query rỗng hoặc không có kết quả. Cả hai trạng thái giữ cùng tâm; search không được dùng làm mốc cố định để thả kết quả xuống dưới. Preview dùng dữ liệu minh họa, có nhãn **Visual preview** trong Settings, không mở app/file thật. Bản desktop có dữ liệu thực và nhãn tổng số items.

## 3. Cài vào tài khoản cá nhân

```sh
make install
~/.local/bin/pika show
```

Không dùng sudo cho `make install`. Script cài:

| File | Chức năng |
|---|---|
| `~/.local/bin/pika` | Binary |
| `~/.local/share/applications/pika.desktop` | Mục Pika trong menu ứng dụng |
| `~/.local/share/icons/hicolor/scalable/apps/pika.svg` | Icon |
| `~/.local/share/pika/installation.txt` | Ghi nhận cài đặt để gỡ đúng file |

Script tôn trọng XDG_DATA_HOME/XDG_CONFIG_HOME nếu đã cấu hình. Config và SQLite không bị ghi đè khi cài lại. Nếu có binary `pika` do công cụ khác cài ở cùng đích, script dừng và báo rõ.

Muốn mở sẵn trong nền sau khi đăng nhập:

```sh
make autostart
```

Lệnh này tạo `~/.config/autostart/pika.desktop`, chạy `pika --background`. Pika dựng WebView nhưng ẩn cửa sổ, chờ bạn gọi.

Tắt tự khởi động:

```sh
make autostart-off
```

## 4. Gán Alt+Space trong Cinnamon

1. Mở **System Settings → Keyboard → Shortcuts**.
2. Kiểm tra shortcut đang dùng **Alt+Space**. Nếu nó mở window menu, bỏ hoặc đổi binding đó trong nhóm Windows. Nếu Look đang bắt Alt+Space thì thoát Look hoặc đổi hotkey của Look.
3. Vào **Custom Shortcuts → Add custom shortcut**.
4. Điền Name: **Pika**.
5. Điền Command:

```text
/home/lilmint/.local/bin/pika toggle
```

6. Chọn shortcut vừa tạo, đặt keyboard binding **Alt+Space**.
7. Thử từ terminal, browser và editor: launcher hiện, gõ được ngay; nhấn lại để ẩn.

Giữ **Super+Space** cho IBus như workflow hiện tại của bạn. Pika không tự sửa hay reset phím tắt hệ thống.

Nếu dùng repo mà chưa cài, Command tạm thời là:

```text
/home/lilmint/workspace/me/pika/build/bin/pika toggle
```

## 5. Cách sử dụng

| Thao tác | Kết quả |
|---|---|
| Gõ tên/alias | Tìm trong tab hiện tại; hỗ trợ tên tiếng Việt không dấu |
| `↑` / `↓` | Chuyển kết quả; danh sách tự cuộn |
| `Enter` | Mở kết quả đang chọn rồi ẩn Pika |
| Click kết quả | Chọn và xem chi tiết ở cột phải |
| Nhấp đúp kết quả / bấm Action ở cột phải | Mở kết quả đang chọn |
| `Escape` | Đóng Settings nếu đang mở; nếu không thì ẩn Pika |
| `Ctrl+1` | All |
| `Ctrl+2` | Apps |
| `Ctrl+3` | Files, gồm cả folders |
| `Ctrl+4` | Commands, nếu `search.include_commands = true` |
| `Ctrl+,` | Mở/đóng Settings |
| `Ctrl+Shift+,` | Đọc lại config từ đĩa |
| `Ctrl+R` | Yêu cầu reindex nền |
| `> tên-lệnh` | Chỉ tìm commands, dù tab hiện tại là All |

Không cần có dấu tiếng Việt khi tìm: `dieu khien` có thể tìm `Điều khiển`. Fuzzy hỗ trợ kiểu viết tắt theo thứ tự ký tự, như `frfx` cho Firefox; không phải bộ sửa mọi lỗi chính tả.

Giao diện không hiện kết quả khi query rỗng, kể cả khi chọn tab Files. Gõ tên rồi dùng phím mũi tên và Enter để mở. Candidate khớp tên chính xác đứng trước fuzzy match dù fuzzy item có usage cao hơn.

Đang nhập tiếng Việt bằng IME, Enter để chốt composition không được dùng để mở ứng dụng. Hãy thử với bộ gõ thực tế của bạn sau khi cài.

## 6. Tùy chỉnh Tokyo Night và khoảng cách khung

Nhấn phím tắt `Ctrl+,`, sau đó **Edit config**. File mở bằng ứng dụng mặc định cho TOML trên máy. Bạn cũng có thể mở trực tiếp bằng editor.

Trong config hiện có, sửa các section tương ứng; không thêm một section trùng tên:

```toml
[window]
width = 800
height = 660
remember_query = false

[appearance]
theme = "tokyo-night"
inner_inset = 12
radius = 20
font_size = 15
```

Lưu file rồi bấm **Reload config**, hoặc:

```sh
~/.local/bin/pika reload-config
```

Kích thước window và theme được áp dụng ngay. Config sai báo lỗi và giữ cấu hình tốt đang chạy. Kiểm tra trước khi mở Pika:

```sh
~/.local/bin/pika --check-config
```

Giao diện không có thanh chọn theme hay nền chuyển sắc. Mặc định dùng Tokyo Night; nếu cần, vẫn có thể đổi `appearance.theme` trong config thành `catppuccin`, `rose-pine`, `gruvbox`, `dracula`, `kanagawa`, `light` hoặc `custom`, rồi Reload config.

Kích thước mặc định mới là **800×660** để hai cột thoáng như ảnh Look. Config cũ không bị tự ghi đè: nếu đang dùng 720×520, sửa `[window]` theo mẫu trên rồi Reload config. Bố cục vẫn hỗ trợ cửa sổ nhỏ tới 480×380.

Dùng `Ctrl+R` hoặc **Reindex** trong Settings để làm mới dữ liệu. Nút lọc bên phải ô tìm kiếm lần lượt là All, Apps, Files, Commands (Ctrl+1…4). Muốn đổi từng màu, đặt `theme = "custom"` rồi sửa:

```toml
[appearance.colors]
background = "#1a1b26"
surface = "#24283b"
text = "#c0caf5"
muted = "#9aa5ce"
selection = "#283457"
accent = "#7aa2f7"
border = "#414868"
error = "#f7768e"
```

Mỗi màu dùng định dạng `#RRGGBB`. Cửa sổ native và vùng ngoài các khung dùng nền trong suốt; ô search và hai cột có nền đặc để dễ đọc. Cần compositor của desktop hoạt động để hiển thị alpha; không dùng blur desktop. Cần thoát và mở lại binary mới để thay đổi nền native có hiệu lực. Các key trong file mẫu cũ ở tài liệu thiết kế có thể khác implementation; dùng [config.example.toml](config.example.toml) làm tham chiếu hiện hành.

## 7. Chọn thư mục tìm kiếm

Sửa section `[index]` đã có:

```toml
[index]
roots = ["~/Documents", "~/Downloads", "~/workspace/me"]
exclude_dirs = [".git", "node_modules", ".cache", "vendor", "dist", "build", ".venv"]
max_depth = 8
max_candidates = 50000
include_hidden = false
```

Reload config, chờ trạng thái **Updating index…** kết thúc, rồi chuyển sang tab Files. Roots không tồn tại sẽ được báo trong Settings.

Pika chỉ tìm tên/path, không tìm nội dung file. `exclude_dirs` khớp tên thư mục/thành phần đường dẫn; `exclude_paths` khớp đường dẫn tuyệt đối hoặc `~/…` cùng cây con. Cả hai không dùng glob. File tạm `.swp`, `.tmp`, `.crdownload`, tên kết thúc `~` được bỏ qua. Không recurse qua symlink thư mục.

Giới hạn 50k áp dụng cho file/folder; apps/commands được thu thập riêng. Khi chạm cap, Settings báo index chưa đủ và gợi ý thu hẹp phạm vi.

Watcher mặc định:

```toml
[watcher]
enabled = true
max_directories = 4096
reconcile_seconds = 900
```

Thay đổi create/rename/delete được gom và reindex nền; sau các đợt event có cooldown. Directory mới được thêm watch. Có lượt scan đối soát mỗi 15 phút; đường dẫn mất kết nối hoặc vượt số watches có thể chỉ được cập nhật ở lần scan tiếp theo. `Ctrl+R` để làm mới ngay.

Phiên bản này refresh toàn bộ các roots được cấu hình trong một background worker khi có thay đổi cần index. Chưa có cập nhật delta theo từng nhánh như thiết kế dài hạn; nên chọn roots nhỏ và exclude dependency trees.

## 8. Thêm command mở project

Thêm ở cuối config:

```toml
[[commands]]
id = "open-pika"
name = "Open Pika project"
aliases = ["pika", "project pika"]
executable = "code"
args = ["/home/lilmint/workspace/me/pika"]
cwd = "/home/lilmint/workspace/me/pika"
```

Nếu muốn dùng commands trong profile cá nhân, đặt `search.include_commands = true`. Reload → gõ `> pika` → Enter. `code` phải tìm thấy trong PATH của desktop session; nếu không, dùng đường dẫn tuyệt đối từ `command -v code`.

Mở terminal trong project, nếu máy có `gnome-terminal`:

```toml
[[commands]]
id = "terminal-pika"
name = "Terminal · Pika"
aliases = ["term pika"]
executable = "gnome-terminal"
args = ["--working-directory=/home/lilmint/workspace/me/pika"]
```

Lệnh có pipeline hoặc nhiều bước nên đặt trong một script do bạn quản lý, rồi dùng đường dẫn script làm executable. Pika truyền argv trực tiếp, không chạy chuỗi query qua shell và không tự expand `~`/`$VAR` trong `args`. Dùng absolute path trong args.

Để pin command trên cùng khi query rỗng, sửa `pinned_ids` ở đầu file, trước các section:

```toml
pinned_ids = ["command:open-pika"]
```

App ID có dạng `app:firefox.desktop`; file/folder ID có dạng `file:/absolute/path` hoặc `directory:/absolute/path`. Có thể xem desktop ID từ tên file `.desktop` trong thư mục applications của XDG.

## 9. Các lệnh quản lý

```sh
~/.local/bin/pika show          # hiện và focus; tự start nếu chưa chạy
~/.local/bin/pika toggle        # đảo hiện/ẩn; tự start nếu chưa chạy
~/.local/bin/pika hide          # ẩn, không thoát
~/.local/bin/pika --background  # chạy nền ẩn, process vẫn sống
~/.local/bin/pika reindex       # queue scan nền
~/.local/bin/pika reload-config # validate rồi apply
~/.local/bin/pika stats         # counts, warnings, watches, thời gian scan
~/.local/bin/pika quit          # thoát thật và flush usage
```

`pika` không có argument chạy foreground, tiện xem lỗi trong terminal. `pika show/toggle` có thể spawn background process và trả lại terminal sau khi frontend ready.

## 10. Lưu dữ liệu, cập nhật và gỡ

| Đường dẫn mặc định | Nội dung |
|---|---|
| `~/.config/pika/config.toml` | Config người dùng |
| `~/.local/share/pika/state.db` | Usage count + last used, SQLite |
| `~/.local/state/pika/pika.log` | Log khi start qua show/toggle |
| `$XDG_RUNTIME_DIR/pika/control.sock` | IPC session |

Usage cập nhật RAM ngay sau dispatch được chấp nhận, ghi SQLite theo đợt khoảng 1 giây; Quit flush trước khi đóng. Crash có thể mất phần chưa flush. DB lỗi không ngăn tìm kiếm, nhưng Settings sẽ báo persistence degraded. “Mở thành công” nghĩa là dispatch/helper được chấp nhận; Pika không kiểm chứng mọi app đích đã dựng cửa sổ.

Nâng cấp binary sau khi sửa code:

```sh
~/.local/bin/pika quit
make install
~/.local/bin/pika show
```

Gỡ:

```sh
~/.local/bin/pika quit
make uninstall
```

Sau đó xóa custom shortcut Pika trong Cinnamon. Config và SQLite được giữ để dùng lại. Không có cơ chế purge tự động.

## 11. Khắc phục lỗi thường gặp

| Hiện tượng | Cách kiểm tra |
|---|---|
| Alt+Space vẫn mở window menu/Look | Kiểm tra binding Cinnamon và hotkey Look; thử `~/.local/bin/pika toggle` trong terminal trước |
| Pika không hiện | Chạy binary không argument để đọc lỗi; xem `pika.log`; kiểm tra `pika --check-config` |
| `webkit2gtk-4.0` missing | Dùng `make build`/`make dev` có `-tags webkit2_41` |
| CLI báo XDG_RUNTIME_DIR | Chạy từ phiên desktop đang login, không qua sudo; thư mục runtime phải thuộc user và có mode 0700 |
| File không tìm thấy | Kiểm tra roots, exclusions, depth/cap trong Settings; chạy reindex |
| Command `code` không chạy | PATH desktop khác terminal; dùng absolute executable và absolute args |
| Settings Edit config mở sai app | Mở TOML bằng editor trực tiếp hoặc chỉnh file association của hệ thống |
| Icon generic | Một số icon theme không nằm trong các đường dẫn resolver hiện hỗ trợ; search/open vẫn hoạt động |
| Dev mở bản cũ | Thoát bản release trước `make dev`; chỉ một instance trên session |

Kiểm thử dành cho phát triển:

```sh
make test
make race
make bench
make check
make smoke
```

`make smoke` mở native window bằng profile tạm, gọi toggle 100 lần rồi quit; không đổi config/autostart/hotkey thật. Test này xác nhận readiness và các lệnh lifecycle, nay kiểm tra thêm 10 chu kỳ nhận focus của cửa sổ native, WebView và ô search, cùng focus khi khởi động lạnh. Việc nhập tiếng Việt qua IME vẫn cần thử trực tiếp trong phiên dùng hằng ngày.


### Chẩn đoán focus sau Alt+Space

Bản sửa ngày 12/09/2026 dùng thời gian hiện tại của X11 khi kích hoạt cửa sổ, đặt focus vào WebView và ô search sau khi cửa sổ nhận bàn phím. Điều này xử lý trường hợp Pika hiện phía trên nhưng bàn phím vẫn thuộc ứng dụng trước đó.

Nếu cần kiểm tra, chạy:

```sh
~/.local/bin/pika show
~/.local/bin/pika focus-state
```

Sau khi cửa sổ đã hiện, các giá trị `visible`, `mapped`, `window_active`, `webview_focused`, `document_focused`, `query_focused` phải là `true`. Chạy lệnh kiểm tra ngay sau `show` có thể thấy trạng thái trung gian vì GTK kích hoạt bất đồng bộ; đọc lại sau một nhịp. Lệnh chỉ trả về trạng thái focus, không trả về nội dung tìm kiếm.


### Ứng dụng cần mật khẩu, mục hệ thống và giao diện rõ hơn

Pika mở file `.desktop` trực tiếp bằng GIO và giữ tiến trình cha sống trong lúc ứng dụng yêu cầu xác thực. Lỗi `Refusing to render service to dead parents` của luồng `gtk-launch` cũ đã được xử lý. Khi mở **Login Window**, Pika ẩn trước, rồi polkit của Linux hiển thị yêu cầu mật khẩu như khi mở từ menu hệ thống. Nhập mật khẩu trong hộp thoại của Linux; Pika không xử lý hay lưu mật khẩu.

Nhấn Alt+Space, gõ một trong các từ sau, chọn kết quả và Enter:

| Từ tìm kiếm | Kết quả |
|---|---|
| `lock` hoặc `khoa man hinh` | Khóa màn hình |
| `logout` hoặc `log out` | Mở hộp thoại Cinnamon: Log Out, Switch User, Cancel |
| `shutdown` hoặc `shut down` | Mở hộp thoại Cinnamon: Suspend, Restart, Shut Down, Cancel |

Các mục này thuộc **System action**, tìm được trong All hoặc Apps ngay cả khi `include_commands = false`. `logout` và `shutdown` gọi hộp thoại hệ thống, không dùng lệnh tắt máy trực tiếp hoặc tùy chọn bỏ qua xác nhận. Lựa chọn cụ thể phụ thuộc khả năng và cấu hình Cinnamon; ví dụ Suspend có thể bị vô hiệu hóa khi hệ thống không hỗ trợ.

Profile hiện tại dùng khoảng cách 8px, cỡ chữ cơ sở 16px, chữ tiêu đề đậm hơn và viền xám rõ. Các khung dùng nền đặc; phần ngoài khung vẫn trong suốt. Hiệu ứng co giãn cả vùng chữ đã được bỏ để giảm nhòe khi launcher hiện lên.


### Quota Codex trong chi tiết ChatGPT

Tìm `ChatGPT` (ứng dụng Codex) rồi chọn ứng dụng. Cột phải có **Usage remaining**:

- **5 hours**: phần trăm quota 5 giờ còn lại và thời điểm làm mới.
- **Weekly**: phần trăm quota tuần còn lại và thời điểm làm mới.
- **Resets available**: số lượt reset còn lại do Codex cung cấp.

Pika tự cập nhật mỗi 30 giây khi bạn đang xem mục này; nút ↻ làm mới theo yêu cầu (giới hạn tối thiểu 5 giây giữa các lần đọc). Pika không dùng lượt reset. Không có dữ liệu sẽ hiện `Not available`; lỗi kết nối giữ dữ liệu lần trước kèm `Last known usage` và thời điểm cập nhật.

Nguồn là API chỉ đọc `account/rateLimits/read` của [Codex App Server](https://learn.chatgpt.com/docs/app-server). Pika ưu tiên CLI đi kèm app tại `/usr/lib/chatgpt/resources/codex`, dùng đăng nhập Codex hiện có. Không cần API key riêng và không tạo task/turn để đọc quota. Trường `availableCount` là số reset chính thức, không phải số tiền credits hay số dòng trong danh sách reset.

Nếu không thấy số liệu, mở Codex kiểm tra đăng nhập và kết nối. Có thể chạy chẩn đoán chỉ đọc:

```sh
cd /home/lilmint/workspace/me/pika
go run ./scripts/check-codex-usage
```

Preview trình duyệt dùng số minh họa và có nhãn `Preview example`; bản desktop lấy số thực từ tài khoản.

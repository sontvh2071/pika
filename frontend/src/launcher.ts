import './launcher.css';
import { reveal, selectIcon, revealDetails } from './motion';
type Item = {
    id: string;
    kind: string;
    name: string;
    subtitle: string;
    pinned: boolean;
    path: string;
};
type Config = {
    window: {
        width: number;
        height: number;
        remember_query: boolean;
    };
    appearance: {
        theme: string;
        inner_inset: number;
        radius: number;
        font_size: number;
        colors: Record<string, string>;
    };
    search: { include_commands: boolean };
    open: { workspace_root: string; workspace_executable: string };
    index: {
        roots: string[];
    };
};
type Status = {
    apps: number;
    files: number;
    commands: number;
    system: number;
    indexing: boolean;
    watches: number;
    warnings: string[];
    state_error: string;
    duration_ms: number;
};
type AppState = {
    config: Config;
    status: Status;
    config_path: string;
};
type Response = {
    request_id: number;
    results: Item[];
    version: number;
    duration_ms: number;
};
type API = {
    GetState(): Promise<AppState>;
    Search(q: string, kind: string, id: number): Promise<Response>;
    Execute(id: string): Promise<void>;
    Icon(id: string): Promise<string>;
    Details(id: string): Promise<{kind: string; path: string; version: string; opener: string}>;
    Hide(): Promise<void>;
    FrontendReady(): Promise<void>;
    ReportFocus(documentFocused: boolean, queryFocused: boolean): Promise<void>;
    CenterWindow(width: number, height: number): Promise<void>;
    OpenConfig(): Promise<void>;
    ReloadConfig(): Promise<void>;
    SetTheme(name: string): Promise<void>;
    Reindex(): Promise<void>;
    Quit(): Promise<void>;
};
declare global {
    interface Window {
        go?: {
            main: {
                App: API;
            };
        };
        runtime?: {
            EventsOn(name: string, fn: () => void): () => void;
        };
    }
}
const icons: Record<string, string> = {
    search: '<circle cx="10.8" cy="10.8" r="6.8"/><path d="m16 16 4.5 4.5"/>',
    app: '<rect x="3" y="3" width="7" height="7" rx="2"/><rect x="14" y="3" width="7" height="7" rx="2"/><rect x="3" y="14" width="7" height="7" rx="2"/><rect x="14" y="14" width="7" height="7" rx="2"/>',
    file: '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6M8 14h8M8 17h5"/>',
    directory: '<path d="M3 7V5a2 2 0 0 1 2-2h5l3 4h6a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z"/>',
    lock: '<rect x="5" y="10" width="14" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3M12 14v3"/>',
    logout: '<path d="M9 4H4v16h5M13 7l5 5-5 5M8 12h12"/>',
    shutdown: '<path d="M12 3v9M7 5.5a8 8 0 1 0 10 0"/>',
    command: '<path d="m4 6 6 6-6 6M13 18h7"/>',
    settings: '<path d="M4 7h16M4 17h16"/><circle cx="9" cy="7" r="3"/><circle cx="15" cy="17" r="3"/>',
    arrow: '<path d="M19 5v8H5m5-5-5 5 5 5"/>',
    refresh: '<path d="M20 7v5h-5M4 17v-5h5M6.1 6.1A8 8 0 0 1 20 12M4 12a8 8 0 0 0 13.9 5.9"/>',
    close: '<path d="m6 6 12 12M18 6 6 18"/>',
    bolt: '<path d="m13 2-9 12h7l-1 8 10-13h-7l1-7Z"/>',
    pin: '<path d="m16 3 5 5-5 2-2 5-2-2-6 6-1-1 6-6-2-2 5-2z"/>',
    globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c5 5 5 13 0 18-5-5-5-13 0-18Z"/>',
};
function itemIcon(item: Item): string { return item.kind === 'system' ? item.id.split(':')[1] : item.kind; }
function svg(name: string, cls = ''): string { return `<svg class="${cls}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${icons[name] || icons.app}</svg>`; }
const native = Boolean(window.go?.main?.App);
document.documentElement.dataset.host = native ? 'native' : 'preview';
const demo: Item[] = [
    {path:'/usr/bin/cinnamon-screensaver-command',id:'system:lock',kind:'system',name:'Lock',subtitle:'Lock the screen',pinned:false},
    {path:'/usr/bin/cinnamon-session-quit',id:'system:logout',kind:'system',name:'Log Out',subtitle:'Choose Log Out, Switch User or Cancel',pinned:false},
    {path:'/usr/bin/cinnamon-session-quit',id:'system:shutdown',kind:'system',name:'Shut Down',subtitle:'Choose Suspend, Restart, Shut Down or Cancel',pinned:false},
    { path: '/usr/share/applications/firefox.desktop', id: 'preview:firefox', kind: 'app', name: 'Firefox', subtitle: 'Web Browser', pinned: true },
    { path: '/usr/share/applications/code.desktop', id: 'preview:code', kind: 'app', name: 'Visual Studio Code', subtitle: 'Code Editor', pinned: false },
    { path: '/usr/share/applications/org.gnome.Terminal.desktop', id: 'preview:terminal', kind: 'app', name: 'Terminal', subtitle: 'Use the command line', pinned: false },
    { path: '/usr/share/applications/nemo.desktop', id: 'preview:files', kind: 'app', name: 'Files', subtitle: 'Browse your files and folders', pinned: false },
    { path: '/home/lilmint/workspace/me/pika', id: 'preview:project', kind: 'directory', name: 'pika', subtitle: '~/workspace/me', pinned: false },
    { path: '/home/lilmint/Documents', id: 'preview:documents', kind: 'directory', name: 'Documents', subtitle: '~/Documents', pinned: false },
];
const previewState: AppState = { config: { window: { width: 720, height: 520, remember_query: false }, appearance: { theme: 'custom', inner_inset: 8, radius: 20, font_size: 16, colors: {background:'#101010',surface:'#151515',text:'#f3f3f3',muted:'#b8b8b8',selection:'#343434',accent:'#ffffff',border:'#535353',error:'#d0d0d0'} }, search: {include_commands:false}, open: {workspace_root:'/home/lilmint/workspace',workspace_executable:'/usr/bin/code'}, index: { roots: ['/home/lilmint'] } }, status: { apps: 4, files: 2, commands: 0, system: 3, indexing: false, watches: 0, warnings: [], state_error: '', duration_ms: 0 }, config_path: '~/.config/pika/config.toml' };
const api: API = window.go?.main.App || {
    GetState: async () => previewState,
    ReportFocus: async () => {},
    CenterWindow: async () => {},
    Details: async id => { const x = demo.find(item => item.id === id)!; return {kind:x.kind,path:x.path,version:'',opener:x.kind === 'system' ? (x.id === 'system:lock' ? 'Lock screen' : 'Show options') : x.path.startsWith('/home/lilmint/workspace/') ? 'VS Code' : x.kind === 'directory' ? 'File manager' : 'Launch application'}; },
    SetTheme: async name => { previewState.config.appearance.theme = name; },
    Search: async (q, k, id) => ({ request_id: id, results: demo.filter(x => (!q.trim().startsWith('>') || x.kind === 'command') && (k === 'all' || x.kind === k || (k === 'file' && x.kind === 'directory') || (k === 'app' && x.kind === 'system')) && x.name.toLowerCase().replaceAll(' ', '').includes(q.replace(/^>\s*/, '').toLowerCase().replaceAll(' ', ''))), version: 1, duration_ms: 0 }),
    Execute: async () => { throw Error('Open the desktop build to launch applications. This is a visual preview.'); },
    Icon: async () => '', Hide: async () => { toast('Escape hides the window in the desktop app.'); }, FrontendReady: async () => { },
    OpenConfig: async () => { throw Error('Config editing is available in the desktop app.'); }, ReloadConfig: async () => { }, Reindex: async () => { }, Quit: async () => { },
};
const themes: Record<string, string> = { catppuccin: 'Catppuccin', 'tokyo-night': 'Tokyo Night', 'rose-pine': 'Rose Pine', gruvbox: 'Gruvbox', dracula: 'Dracula', kanagawa: 'Kanagawa' };
document.querySelector<HTMLDivElement>('#app')!.innerHTML = `
 <main class="launcher" aria-label="Pika launcher">
  <div class="inner-frame">
   <header class="search-area">
    <div class="search-symbol">${svg('search')}</div>
    <input id="query" role="combobox" aria-label="Search apps, files and commands" aria-autocomplete="list" aria-controls="results" aria-expanded="false" placeholder="Search apps, files, commands…" autocomplete="off" spellcheck="false" autofocus />
    <button id="clear" class="icon-button clear-button" aria-label="Clear search" title="Clear search" hidden>${svg('close')}</button>
    <div class="tabs" role="tablist" aria-label="Filter results">
     ${[['all','search','All'],['app','app','Apps'],['file','file','Files'],['command','command','Commands']].map(([id, icon, label], i) => `<button class="tab ${i === 0 ? 'active' : ''}" role="tab" data-kind="${id}" aria-label="${label}" title="${label} · Ctrl+${i + 1}" aria-selected="${i === 0}">${svg(icon)}<sup>${i + 1}</sup></button>`).join('')}
    </div>
   </header>
   <div class="workspace" hidden>
    <section class="results-area" aria-label="Search results">
     <div class="section-heading"><span id="section-label">SUGGESTED</span><span id="result-count"></span></div>
     <div id="results" role="listbox" aria-label="Results"><div id="selection-highlight" aria-hidden="true" hidden></div></div>
     <div id="empty" class="empty" hidden></div>
    </section>
    <section class="detail-panel" aria-label="Selected result details">
     <div id="detail-content" hidden>
      <header class="detail-header"><div id="detail-icon" class="detail-icon"></div><div class="detail-heading"><h2 id="detail-name"></h2><span id="detail-kind" class="type-badge"></span><span id="detail-pin" class="type-badge" hidden>Pinned</span></div></header>
      <dl class="detail-properties"><div><dt>Kind</dt><dd id="detail-type"></dd></div><div class="path-property"><dt>Path</dt><dd id="detail-path"></dd></div><div><dt>Version</dt><dd id="detail-version"></dd></div></dl><button id="detail-open" class="detail-action"></button>
      <div id="detail-description" class="detail-description"><span id="detail-label"></span><p id="detail-subtitle"></p></div>
     </div>
     <div id="detail-empty" class="detail-empty">${svg('search')}<p>Select a result to see its details</p></div>
    </section>
   </div>
   <div id="message" class="message" role="status" aria-live="polite" hidden></div>
  </div>
  <section id="settings-panel" class="settings-panel" aria-label="Settings and index status" hidden>
   <header class="panel-header"><div><span class="eyebrow">MAKE IT YOURS</span><h1>Settings</h1></div><button id="settings-close" class="icon-button" aria-label="Close settings">${svg('close')}</button></header>
   <div class="settings-content">
    <p id="status-text" class="settings-copy" role="status"></p>
    <div class="theme-card"><div class="theme-mark">${svg('bolt')}</div><div><strong id="theme-name">Tokyo Night</strong><p id="theme-info">Panel spacing · 12 px</p></div><div class="swatches"><i></i><i></i><i></i><i></i></div></div>
    <p class="settings-copy">Your launcher, your rules. Edit the config to change colors, search folders, pins and personal commands.</p>
    <code id="config-path" class="config-path"></code>
    <div class="button-row"><button id="edit-config" class="primary-button">Edit config ${svg('arrow')}</button><button id="reload-config" class="secondary-button">Reload config</button></div>
    <div class="stats-grid"><div><strong id="app-count">0</strong><span>applications</span></div><div><strong id="file-count">0</strong><span>files & folders</span></div><div><strong id="command-count">0</strong><span>commands</span></div><div><strong id="system-count">0</strong><span>system actions</span></div></div>
    <div class="index-detail"><span id="index-detail">Index ready</span><button id="reindex" class="text-button">${svg('refresh')} Reindex</button></div>
    <p id="roots-help" class="settings-copy"></p><div id="warnings" class="warnings"></div>
   </div>
   <footer class="settings-footer"><span>PIKA <span class="version">0.1.0</span> · LOCAL FIRST</span><button id="quit" class="text-button">Quit Pika</button></footer>
  </section>
 </main>`;
const $ = <T extends HTMLElement = HTMLElement>(id: string) => document.getElementById(id) as T;
const input = $<HTMLInputElement>('query');
let state: AppState = previewState, kind = 'all', results: Item[] = [], selected = 0, version = 0, composing = false, busy = false, panel = false, hidden = false;
let pending = false, searchTimer: ReturnType<typeof setTimeout>, detailTimer: ReturnType<typeof setTimeout>, detailToken = 0, detailedID = '';
const rowCache = new Map<string, HTMLElement>();
const highlight = document.getElementById('selection-highlight')!;
const shell = document.querySelector<HTMLElement>('.launcher')!;
if (native) shell.classList.add('is-armed');
let toastTimer: ReturnType<typeof setTimeout>;
const iconCache = new Map<string, string>();
function toast(message: string) { clearTimeout(toastTimer); $('message').textContent = message; $('message').hidden = false; toastTimer = setTimeout(() => { $('message').hidden = true; }, 5000); }
function report(e: unknown) { toast(e instanceof Error ? e.message : String(e)); }
function applyTheme() { const a = state.config.appearance; const root = document.documentElement; root.dataset.theme = a.theme; root.style.setProperty('--launcher-width', `${state.config.window.width}px`); root.style.setProperty('--launcher-height', `${state.config.window.height}px`); root.style.setProperty('--inset', `${a.inner_inset}px`); root.style.setProperty('--radius', `${a.radius}px`); root.style.setProperty('--font-size', `${a.font_size}px`); for (const key of ['background', 'surface', 'text', 'muted', 'selection', 'accent', 'border', 'error']) {
    root.style.removeProperty(`--${key}`);
    if (a.theme === 'custom' && a.colors[key])
        root.style.setProperty(`--${key}`, a.colors[key]);
} ; $('theme-name').textContent = themes[a.theme] || (a.theme === 'light' ? 'Daylight' : 'Custom theme'); $('theme-info').textContent = `Panel spacing · ${a.inner_inset} px · Radius ${Math.min(a.radius, 12)} px`; }
function renderStatus() { const s = state.status; $('status-text').textContent = !native ? 'Visual preview' : s.indexing ? 'Updating index…' : s.warnings.length || s.state_error ? 'Index needs attention' : `${s.apps + s.files + s.commands + s.system} items · local`; $('app-count').textContent = String(s.apps); $('file-count').textContent = String(s.files); $('command-count').textContent = String(s.commands); $('system-count').textContent = String(s.system); $('config-path').textContent = state.config_path; $('index-detail').textContent = s.indexing ? 'Indexing in background…' : `${s.watches} watched directories · ${s.duration_ms} ms last scan`; $('roots-help').textContent = state.config.index.roots.length ? `Searching: ${state.config.index.roots.join(', ')}` : 'File search is opt-in. Add directories to [index].roots in your config.'; $('warnings').textContent = [...s.warnings, s.state_error].filter(Boolean).join('\n'); }
async function refreshState() { try {
    state = await api.GetState();
    applyTheme();
    renderStatus();
    const commandTab = document.querySelector<HTMLButtonElement>('[data-kind="command"]')!;
    commandTab.hidden = !state.config.search.include_commands;
    if (kind === 'command' && !state.config.search.include_commands) kind = 'all';
    input.placeholder = state.config.search.include_commands ? 'Search apps, files, commands…' : 'Search apps, files and folders…';
    input.setAttribute('aria-label', input.placeholder.replace('…', ''));
}
catch (e) {
    report(e);
} }
function setPanel(open: boolean) { panel = open; syncLayout(); $('settings-panel').hidden = !open; document.querySelector<HTMLElement>('.inner-frame')!.inert = open; if (open)
    $('settings-close').focus();
else
    input.focus(); }
const workspacePanel = document.querySelector<HTMLElement>('.workspace')!;
function syncLayout() {
    const visible = Boolean(input.value.trim()) && results.length > 0;
    workspacePanel.hidden = !visible;
    input.setAttribute('aria-expanded', String(visible && !panel));
    // Hide only the painted panels; never resize or recenter while typing.
}
let centerFrame = 0;
function centerWholeWindow() {
    if (!native) return;
    cancelAnimationFrame(centerFrame);
    centerFrame = requestAnimationFrame(() => { void api.CenterWindow(innerWidth, innerHeight).catch(report); });
}
window.addEventListener('resize', centerWholeWindow);
function setKind(next: string) { if (next === 'command' && !state.config.search.include_commands) return; kind = next; for (const el of document.querySelectorAll<HTMLButtonElement>('[data-kind]')) {
    const active = el.dataset.kind === kind;
    el.classList.toggle('active', active);
    el.setAttribute('aria-selected', String(active));
} ; input.focus(); void search(); }
function selection(animate = true) {
    const rows = document.querySelectorAll<HTMLElement>('.result');
    const previous = document.querySelector<HTMLElement>('.result.selected');
    rows.forEach((row, i) => { row.classList.toggle('selected', i === selected); row.setAttribute('aria-selected', String(i === selected)); });
    const row = rows[selected];
    highlight.hidden = !row;
    if (row) {
        input.setAttribute('aria-activedescendant', row.id);
        highlight.classList.toggle('instant', !animate);
        highlight.style.height = `${row.offsetHeight}px`;
        highlight.style.transform = `translateY(${row.offsetTop}px)`;
        if (!animate) { void highlight.offsetHeight; highlight.classList.remove('instant'); }
        row.scrollIntoView({ block: 'nearest', behavior: 'auto' });
        if (animate && previous !== row) selectIcon(row.querySelector<HTMLElement>('.result-icon')!);
    } else input.removeAttribute('aria-activedescendant');
    renderDetails();
}
new ResizeObserver(() => selection(false)).observe($('results'));
function renderDetails() {
    const item = results[selected];
    $('detail-content').hidden = !item;
    $('detail-empty').hidden = Boolean(item);
    if (!item) { detailedID = ''; detailToken++; clearTimeout(detailTimer); return; }
    const fingerprint = JSON.stringify([item.id, item.path, item.subtitle, item.pinned]);
    if (detailedID === fingerprint) return;
    detailedID = fingerprint;
    const token = ++detailToken;
    clearTimeout(detailTimer);
    const labels: Record<string, string> = { app: 'Application', file: 'File', directory: 'Folder', command: 'Command', system: 'System action' };
    $('detail-name').textContent = item.name;
    $('detail-kind').textContent = labels[item.kind] || item.kind;
    $('detail-type').textContent = labels[item.kind] || item.kind;
    $('detail-path').textContent = item.path || 'Not available';
    $('detail-version').textContent = item.kind === 'app' ? 'Looking up…' : 'Not applicable';
    $('detail-pin').hidden = !item.pinned;
    const workspace = state.config.open.workspace_root.replace(/\/$/, '');
    const inWorkspace = item.path === workspace || item.path.startsWith(workspace + '/');
    $('detail-open').textContent = (item.kind === 'system' ? (item.id === 'system:lock' ? 'Lock screen' : 'Show options') : inWorkspace ? 'Open in VS Code' : item.kind === 'app' ? 'Launch application' : item.kind === 'command' ? 'Run command' : item.kind === 'directory' ? 'Open in Files' : 'Open with default app') + ' ↵';
    $('detail-label').textContent = item.kind === 'command' ? 'COMMAND' : 'ABOUT';
    $('detail-subtitle').textContent = item.subtitle;
    $('detail-description').hidden = item.kind !== 'app' && item.kind !== 'command' && item.kind !== 'system';
    const badge = $('detail-icon');
    badge.innerHTML = svg(itemIcon(item));
    revealDetails($('detail-content'));
    if (native && item.kind === 'app') {
        void loadIcon(item.id).then(url => {
            if (!url || token !== detailToken) return;
            const image = document.createElement('img'); image.src = url; image.alt = '';
            image.onerror = () => { if (token === detailToken) badge.innerHTML = svg(itemIcon(item)); };
            badge.replaceChildren(image);
        }).catch(() => {});
    }
    detailTimer = setTimeout(() => { void api.Details(item.id).then(detail => {
        if (token !== detailToken || hidden) return;
        $('detail-path').textContent = detail.path || 'Not available';
        $('detail-version').textContent = detail.version || (item.kind === 'app' ? 'Not available' : 'Not applicable');
        $('detail-open').textContent = (detail.opener === 'VS Code' ? 'Open in VS Code' : detail.opener === 'File manager' ? 'Open in Files' : detail.opener === 'Default application' ? 'Open with default app' : detail.opener) + ' ↵';
    }).catch(() => { if (token === detailToken) $('detail-version').textContent = 'Not available'; }); }, 65);
}
const iconRequests = new Map<string, Promise<string>>();
function loadIcon(id: string): Promise<string> {
    if (iconCache.has(id)) return Promise.resolve(iconCache.get(id)!);
    if (iconRequests.has(id)) return iconRequests.get(id)!;
    const request = api.Icon(id).then(url => { iconCache.set(id, url); return url; }).finally(() => iconRequests.delete(id));
    iconRequests.set(id, request); return request;
}
function render() {
    syncLayout();
    const box = $('results');
    for (const [id, row] of rowCache) { if (!results.some(item => item.id === id)) { row.remove(); rowCache.delete(id); } }
    $('section-label').textContent = input.value.trim() ? 'RESULTS' : kind === 'all' ? 'SUGGESTED' : kind === 'app' ? 'APPLICATIONS' : kind === 'file' ? 'FILES & FOLDERS' : 'YOUR COMMANDS';
    $('result-count').textContent = results.length ? `${results.length} results` : '';
    for (const [i, item] of results.entries()) {
        const cached = rowCache.get(item.id);
        if (cached) {
            cached.id = `result-${i}`;
            cached.querySelector('.result-title')!.textContent = item.name;
            cached.querySelector('.result-subtitle')!.textContent = item.subtitle;
            box.append(cached);
            continue;
        }
        const row = document.createElement('div');
        rowCache.set(item.id, row);
        row.className = `result ${i === selected ? 'selected' : ''}`;
        row.id = `result-${i}`;
        row.role = 'option';
        row.setAttribute('aria-selected', String(i === selected));
        row.dataset.id = item.id;
        const badge = document.createElement('div');
        badge.className = `result-icon ${item.kind}`;
        badge.innerHTML = svg(itemIcon(item));
        row.append(badge);
        const text = document.createElement('div');
        text.className = 'result-text';
        const title = document.createElement('span');
        title.className = 'result-title';
        title.textContent = item.name;
        const sub = document.createElement('span');
        sub.className = 'result-subtitle';
        sub.textContent = item.subtitle;
        text.append(title, sub);
        row.append(text);
        const meta = document.createElement('div');
        meta.className = 'result-meta';
        if (item.pinned)
            meta.innerHTML = svg('pin', 'pin');
        const label = document.createElement('span');
        label.textContent = item.kind === 'system' ? 'System action' : item.kind === 'app' ? 'Application' : item.kind === 'directory' ? 'Folder' : item.kind === 'command' ? 'Command' : 'File';
        meta.append(label);
        const enter = document.createElement('span');
        enter.className = 'result-enter';
        enter.innerHTML = svg('arrow');
        meta.append(enter);
        row.append(meta);
        row.addEventListener('mousedown', e => e.preventDefault());
        row.addEventListener('click', () => { selected = results.findIndex(x => x.id === item.id); selection(); input.focus(); });
        row.addEventListener('dblclick', () => { selected = results.findIndex(x => x.id === item.id); selection(); void execute(); });
        box.append(row);
        if (native && item.kind === 'app') {
            const show = (url: string) => { if (url && row.isConnected) {
                const image = document.createElement('img');
                image.src = url;
                image.alt = '';
                image.addEventListener('error', () => { badge.innerHTML = svg(itemIcon(item)); }, { once: true });
                badge.replaceChildren(image);
            } };
            if (iconCache.has(item.id))
                show(iconCache.get(item.id)!);
            else
                void loadIcon(item.id).then(show).catch(() => { });
        }
    }
    $('empty').hidden = results.length > 0;
    if (!results.length) {
        const message = state.status.indexing ? 'Building your index…' : kind === 'file' && !state.config.index.roots.length ? 'A place for your files' : kind === 'command' ? 'Your shortcuts start here' : 'No matches found';
        const detail = state.status.indexing ? 'Applications will appear first. You can keep typing.' : kind === 'file' && !state.config.index.roots.length ? 'Add folders to your config to make them searchable.' : kind === 'command' ? 'Add a [[commands]] entry to your config, then reload.' : 'Try a shorter name or another category.';
        $('empty').replaceChildren();
        const symbol = document.createElement('div');
        symbol.className = 'empty-icon';
        symbol.innerHTML = svg(kind === 'file' ? 'directory' : kind === 'command' ? 'command' : 'search');
        const h = document.createElement('strong');
        h.textContent = message;
        const p = document.createElement('p');
        p.textContent = detail;
        $('empty').append(symbol, h, p);
    }
    selection(false);
}
async function search(preserveSelection = false) {
    clearTimeout(searchTimer);
    const id = ++version;
    const previous = preserveSelection ? results[selected]?.id : '';
    pending = true;
    $('results').setAttribute('aria-busy', 'true');
    $<HTMLButtonElement>('detail-open').disabled = true;
    $('clear').hidden = !input.value;
    try {
        if (!input.value.trim()) { results = []; selected = 0; render(); return; }
        const r = await api.Search(input.value, kind, id);
        if (id !== version || hidden) return;
        results = r.results;
        selected = Math.max(0, results.findIndex(x => x.id === previous));
        render();
    } catch (e) {
        if (id === version) { results = []; render(); report(e); }
    } finally {
        if (id === version) {
            pending = false;
            $('results').setAttribute('aria-busy', 'false');
            $<HTMLButtonElement>('detail-open').disabled = false;
        }
    }
}
function queueSearch() {
    version++;
    pending = true;
    clearTimeout(searchTimer);
    $('results').setAttribute('aria-busy', 'true');
    $<HTMLButtonElement>('detail-open').disabled = true;
    $('clear').hidden = !input.value;
    if (!input.value.trim()) { results = []; selected = 0; render(); }
    if (!composing) searchTimer = setTimeout(() => { void search(); }, 25);
}
async function execute() { if (busy || pending || composing || !results[selected])
    return; busy = true; const id = results[selected].id; try {
    await api.Execute(id);
}
catch (e) {
    report(e);
}
finally {
    busy = false;
} }
input.addEventListener('input', queueSearch);
input.addEventListener('compositionstart', () => { composing = true; queueSearch(); });
input.addEventListener('compositionend', () => { composing = false; void search(); });
$('clear').onclick = () => { input.value = ''; input.focus(); void search(); };
for (const tab of document.querySelectorAll<HTMLButtonElement>('[data-kind]'))
    tab.onclick = () => setKind(tab.dataset.kind!);
$('detail-open').onclick = () => { void execute(); };
$('settings-close').onclick = () => setPanel(false);
$('edit-config').onclick = () => { void api.OpenConfig().catch(report); };
async function reload() { try {
    await api.ReloadConfig();
    await refreshState();
    void search();
    toast('Configuration reloaded');
}
catch (e) {
    report(e);
} }
$('reload-config').onclick = () => { void reload(); };
$('reindex').onclick = () => { void api.Reindex().then(() => toast('Refreshing your index…')).catch(report); };
$('quit').onclick = () => { void api.Quit().catch(report); };
document.addEventListener('keydown', e => {
    if (e.isComposing || composing || e.keyCode === 229)
        return;
    if (e.key === 'Escape') {
        e.preventDefault();
        if (panel)
            setPanel(false);
        else {
            version++;
            void api.Hide().catch(report);
        }
        ;
        return;
    }
    if (e.ctrlKey && (e.code === 'Comma' || e.key === ',')) {
        e.preventDefault();
        if (e.shiftKey)
            void reload();
        else
            setPanel(!panel);
        return;
    }
    if (e.ctrlKey && e.key.toLowerCase() === 'r') {
        e.preventDefault();
        void api.Reindex().catch(report);
        return;
    }
    if (e.ctrlKey && ['1', '2', '3', '4'].includes(e.key)) {
        e.preventDefault();
        if (panel)
            setPanel(false);
        setKind(['all', 'app', 'file', 'command'][Number(e.key) - 1]);
        return;
    }
    if (panel)
        return;
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault();
        if (results.length) {
            selected = (selected + (e.key === 'ArrowDown' ? 1 : -1) + results.length) % results.length;
            selection();
        }
    }
    if (e.key === 'Enter' && document.activeElement === input) {
        e.preventDefault();
        void execute();
    }
});
function reportFocus() {
    void api.ReportFocus(document.hasFocus(), document.activeElement === input && document.hasFocus()).catch(() => {});
}
let focusFrame = 0;
function focusSearchAfterActivation() {
    cancelAnimationFrame(focusFrame);
    focusFrame = requestAnimationFrame(() => {
        if (hidden || panel || !document.hasFocus()) return;
        input.focus({preventScroll:true});
        reportFocus();
    });
}
// Native window activation is asynchronous. DOM autofocus alone cannot acquire
// the desktop keyboard; finish focusing the input when WebKit receives it.
window.addEventListener('focus', focusSearchAfterActivation);
window.addEventListener('blur', () => { cancelAnimationFrame(focusFrame); reportFocus(); });
input.addEventListener('focus', reportFocus);
input.addEventListener('blur', reportFocus);
window.runtime?.EventsOn('pika:shown', () => {
    hidden = false; version++; detailedID = ''; shell.classList.remove('is-armed'); reveal();
    if (!state.config.window.remember_query) { input.value = ''; kind = 'all'; results = []; }
    setPanel(false); focusSearchAfterActivation();
    void refreshState().then(() => setKind(kind));
});
window.runtime?.EventsOn('pika:hidden', () => { hidden = true; cancelAnimationFrame(focusFrame); shell.classList.add('is-armed'); version++; detailToken++; clearTimeout(searchTimer); clearTimeout(detailTimer); });
window.runtime?.EventsOn('pika:index', () => { void refreshState().then(() => { if (!hidden && !composing) void search(true); }); });
window.runtime?.EventsOn('pika:config', () => { void refreshState().then(() => { if (!hidden && !composing) { detailedID = ''; void search(true); } }); });
void (async () => { await refreshState(); await search(); await api.FrontendReady(); if (!native) reveal(); input.focus(); })().catch(report);

export namespace catalog {
	
	export class Result {
	    path: string;
	    id: string;
	    kind: string;
	    name: string;
	    subtitle: string;
	    pinned: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.subtitle = source["subtitle"];
	        this.pinned = source["pinned"];
	    }
	}

}

export namespace codexusage {
	
	export class Window {
	    remaining_percent: number;
	    resets_at?: number;
	
	    static createFrom(source: any = {}) {
	        return new Window(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.remaining_percent = source["remaining_percent"];
	        this.resets_at = source["resets_at"];
	    }
	}
	export class Snapshot {
	    five_hour?: Window;
	    weekly?: Window;
	    reset_credits?: number;
	    updated_at: number;
	    stale: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.five_hour = this.convertValues(source["five_hour"], Window);
	        this.weekly = this.convertValues(source["weekly"], Window);
	        this.reset_credits = source["reset_credits"];
	        this.updated_at = source["updated_at"];
	        this.stale = source["stale"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace config {
	
	export class Appearance {
	    theme: string;
	    inner_inset: number;
	    radius: number;
	    font_size: number;
	    colors: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Appearance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.inner_inset = source["inner_inset"];
	        this.radius = source["radius"];
	        this.font_size = source["font_size"];
	        this.colors = source["colors"];
	    }
	}
	export class Command {
	    id: string;
	    name: string;
	    aliases: string[];
	    executable: string;
	    args: string[];
	    cwd: string;
	
	    static createFrom(source: any = {}) {
	        return new Command(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.aliases = source["aliases"];
	        this.executable = source["executable"];
	        this.args = source["args"];
	        this.cwd = source["cwd"];
	    }
	}
	export class Watcher {
	    enabled: boolean;
	    max_directories: number;
	    reconcile_seconds: number;
	
	    static createFrom(source: any = {}) {
	        return new Watcher(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.max_directories = source["max_directories"];
	        this.reconcile_seconds = source["reconcile_seconds"];
	    }
	}
	export class Index {
	    exclude_paths: string[];
	    roots: string[];
	    exclude_dirs: string[];
	    max_depth: number;
	    max_candidates: number;
	    include_hidden: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Index(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exclude_paths = source["exclude_paths"];
	        this.roots = source["roots"];
	        this.exclude_dirs = source["exclude_dirs"];
	        this.max_depth = source["max_depth"];
	        this.max_candidates = source["max_candidates"];
	        this.include_hidden = source["include_hidden"];
	    }
	}
	export class Open {
	    workspace_root: string;
	    workspace_executable: string;
	
	    static createFrom(source: any = {}) {
	        return new Open(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace_root = source["workspace_root"];
	        this.workspace_executable = source["workspace_executable"];
	    }
	}
	export class Search {
	    include_commands: boolean;
	    max_results: number;
	
	    static createFrom(source: any = {}) {
	        return new Search(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.include_commands = source["include_commands"];
	        this.max_results = source["max_results"];
	    }
	}
	export class Window {
	    width: number;
	    height: number;
	    remember_query: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Window(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.width = source["width"];
	        this.height = source["height"];
	        this.remember_query = source["remember_query"];
	    }
	}
	export class Config {
	    schema_version: number;
	    window: Window;
	    appearance: Appearance;
	    search: Search;
	    open: Open;
	    index: Index;
	    watcher: Watcher;
	    commands: Command[];
	    pinned_ids: string[];
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.window = this.convertValues(source["window"], Window);
	        this.appearance = this.convertValues(source["appearance"], Appearance);
	        this.search = this.convertValues(source["search"], Search);
	        this.open = this.convertValues(source["open"], Open);
	        this.index = this.convertValues(source["index"], Index);
	        this.watcher = this.convertValues(source["watcher"], Watcher);
	        this.commands = this.convertValues(source["commands"], Command);
	        this.pinned_ids = source["pinned_ids"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	

}

export namespace launcher {
	
	export class Details {
	    usage_provider?: string;
	    kind: string;
	    path: string;
	    version: string;
	    opener: string;
	
	    static createFrom(source: any = {}) {
	        return new Details(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.usage_provider = source["usage_provider"];
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.version = source["version"];
	        this.opener = source["opener"];
	    }
	}
	export class Response {
	    request_id: number;
	    results: catalog.Result[];
	    version: number;
	    duration_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.request_id = source["request_id"];
	        this.results = this.convertValues(source["results"], catalog.Result);
	        this.version = source["version"];
	        this.duration_ms = source["duration_ms"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Status {
	    apps: number;
	    files: number;
	    commands: number;
	    system: number;
	    indexing: boolean;
	    version: number;
	    last_index: string;
	    duration_ms: number;
	    watches: number;
	    warnings: string[];
	    state_error: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apps = source["apps"];
	        this.files = source["files"];
	        this.commands = source["commands"];
	        this.system = source["system"];
	        this.indexing = source["indexing"];
	        this.version = source["version"];
	        this.last_index = source["last_index"];
	        this.duration_ms = source["duration_ms"];
	        this.watches = source["watches"];
	        this.warnings = source["warnings"];
	        this.state_error = source["state_error"];
	    }
	}

}

export namespace main {
	
	export class AppState {
	    config: config.Config;
	    status: launcher.Status;
	    config_path: string;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], config.Config);
	        this.status = this.convertValues(source["status"], launcher.Status);
	        this.config_path = source["config_path"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}


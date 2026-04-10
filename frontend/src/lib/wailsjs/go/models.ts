export namespace config {
	
	export class AppConfig {
	    repository: string;
	    setup: string[];
	    cleanup: string[];
	    recentDirectories: string[];
	    fluxLicenseKey: string;
	    fluxUsername: string;
	    fluxComposerUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repository = source["repository"];
	        this.setup = source["setup"];
	        this.cleanup = source["cleanup"];
	        this.recentDirectories = source["recentDirectories"];
	        this.fluxLicenseKey = source["fluxLicenseKey"];
	        this.fluxUsername = source["fluxUsername"];
	        this.fluxComposerUrl = source["fluxComposerUrl"];
	    }
	}

}

export namespace db {
	
	export class Installation {
	    id: number;
	    pathHash: string;
	    projectPath: string;
	    projectName: string;
	    repository: string;
	    siteName: string;
	    dbName: string;
	    installedAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Installation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pathHash = source["pathHash"];
	        this.projectPath = source["projectPath"];
	        this.projectName = source["projectName"];
	        this.repository = source["repository"];
	        this.siteName = source["siteName"];
	        this.dbName = source["dbName"];
	        this.installedAt = source["installedAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}

export namespace features {
	
	export class ConfigOption {
	    value: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}
	export class ConfigField {
	    key: string;
	    label: string;
	    type: string;
	    default: string;
	    placeholder: string;
	    options: ConfigOption[];
	
	    static createFrom(source: any = {}) {
	        return new ConfigField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.type = source["type"];
	        this.default = source["default"];
	        this.placeholder = source["placeholder"];
	        this.options = this.convertValues(source["options"], ConfigOption);
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
	
	export class Hooks {
	    preClone: string[];
	    postClone: string[];
	    preHerd: string[];
	    postHerd: string[];
	    prePatch: string[];
	    postPatch: string[];
	    preInstall: string[];
	    postInstall: string[];
	    preCleanup: string[];
	    postCleanup: string[];
	
	    static createFrom(source: any = {}) {
	        return new Hooks(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preClone = source["preClone"];
	        this.postClone = source["postClone"];
	        this.preHerd = source["preHerd"];
	        this.postHerd = source["postHerd"];
	        this.prePatch = source["prePatch"];
	        this.postPatch = source["postPatch"];
	        this.preInstall = source["preInstall"];
	        this.postInstall = source["postInstall"];
	        this.preCleanup = source["preCleanup"];
	        this.postCleanup = source["postCleanup"];
	    }
	}
	export class Patch {
	    file: string;
	    mode: string;
	    format: string;
	    instruction: string;
	    diff: string;

	    static createFrom(source: any = {}) {
	        return new Patch(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file = source["file"];
	        this.mode = source["mode"];
	        this.format = source["format"];
	        this.instruction = source["instruction"];
	        this.diff = source["diff"];
	    }
	}
	export class Instruction {
	    text: string;
	    copy: string;

	    static createFrom(source: any = {}) {
	        return new Instruction(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.copy = source["copy"];
	    }
	}
	export class Feature {
	    id: string;
	    name: string;
	    description: string;
	    requires: string[];
	    incompatible: string[];
	    patches: Patch[];
	    instructions: Instruction[];
	    config: ConfigField[];
	    hooks: Hooks;
	
	    static createFrom(source: any = {}) {
	        return new Feature(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.requires = source["requires"];
	        this.incompatible = source["incompatible"];
	        this.patches = this.convertValues(source["patches"], Patch);
	        this.instructions = this.convertValues(source["instructions"], Instruction);
	        this.config = this.convertValues(source["config"], ConfigField);
	        this.hooks = this.convertValues(source["hooks"], Hooks);
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
	
	
	export class Registry {
	    Features: Feature[];
	    DependencyMap: Record<string, Array<string>>;
	    IncompatMap: Record<string, Array<string>>;
	
	    static createFrom(source: any = {}) {
	        return new Registry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Features = this.convertValues(source["Features"], Feature);
	        this.DependencyMap = source["DependencyMap"];
	        this.IncompatMap = source["IncompatMap"];
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

export namespace installer {
	
	export class InstallRequest {
	    projectName: string;
	    workingDir: string;
	    selectedIds: string[];
	    configValues: Record<string, string>;
	    tempClonePath: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectName = source["projectName"];
	        this.workingDir = source["workingDir"];
	        this.selectedIds = source["selectedIds"];
	        this.configValues = source["configValues"];
	        this.tempClonePath = source["tempClonePath"];
	    }
	}

}

export namespace main {
	
	export class CompatResult {
	    compatible: boolean;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new CompatResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.compatible = source["compatible"];
	        this.reason = source["reason"];
	    }
	}
	export class StartupContext {
	    projectName: string;
	    workingDir: string;
	    hasContext: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StartupContext(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectName = source["projectName"];
	        this.workingDir = source["workingDir"];
	        this.hasContext = source["hasContext"];
	    }
	}
	export class StartupResult {
	    done: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new StartupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.done = source["done"];
	        this.error = source["error"];
	    }
	}

}


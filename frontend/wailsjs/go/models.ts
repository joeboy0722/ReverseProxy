export namespace main {
	
	export class CustomCert {
	    certPath: string;
	    keyPath: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomCert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.certPath = source["certPath"];
	        this.keyPath = source["keyPath"];
	    }
	}
	export class NavConfig {
	    navTitle: string;
	    navSubtitle: string;
	    themeColor: string;
	
	    static createFrom(source: any = {}) {
	        return new NavConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.navTitle = source["navTitle"];
	        this.navSubtitle = source["navSubtitle"];
	        this.themeColor = source["themeColor"];
	    }
	}
	export class ServerStatus {
	    isRunning: boolean;
	    bindAddr: string;
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new ServerStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isRunning = source["isRunning"];
	        this.bindAddr = source["bindAddr"];
	        this.port = source["port"];
	    }
	}

}

export namespace proxy {
	
	export class RequestLog {
	    id: string;
	    // Go type: time
	    timestamp: any;
	    method: string;
	    path: string;
	    ruleSource: string;
	    targetURL: string;
	    statusCode: number;
	    latencyMs: number;
	    reqHeaders: Record<string, string>;
	    reqBody: string;
	    respHeaders: Record<string, string>;
	    respBody: string;
	    reqBodyTrunc: boolean;
	    respBodyTrunc: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RequestLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.method = source["method"];
	        this.path = source["path"];
	        this.ruleSource = source["ruleSource"];
	        this.targetURL = source["targetURL"];
	        this.statusCode = source["statusCode"];
	        this.latencyMs = source["latencyMs"];
	        this.reqHeaders = source["reqHeaders"];
	        this.reqBody = source["reqBody"];
	        this.respHeaders = source["respHeaders"];
	        this.respBody = source["respBody"];
	        this.reqBodyTrunc = source["reqBodyTrunc"];
	        this.respBodyTrunc = source["respBodyTrunc"];
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
	export class RouteRule {
	    id: string;
	    source: string;
	    type: string;
	    target: string;
	    active: boolean;
	    headers: Record<string, string>;
	    healthy: boolean;
	    keepPrefix: boolean;
	    injectBase: boolean;
	    redirectSlash: boolean;
	    healthCheckEnabled?: boolean;
	    healthCheckPath: string;
	    showInIndex: boolean;
	    title: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new RouteRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source = source["source"];
	        this.type = source["type"];
	        this.target = source["target"];
	        this.active = source["active"];
	        this.headers = source["headers"];
	        this.healthy = source["healthy"];
	        this.keepPrefix = source["keepPrefix"];
	        this.injectBase = source["injectBase"];
	        this.redirectSlash = source["redirectSlash"];
	        this.healthCheckEnabled = source["healthCheckEnabled"];
	        this.healthCheckPath = source["healthCheckPath"];
	        this.showInIndex = source["showInIndex"];
	        this.title = source["title"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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


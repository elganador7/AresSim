export namespace main {
	
	export class BridgeResult {
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new BridgeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}

}


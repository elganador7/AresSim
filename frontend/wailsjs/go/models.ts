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
	export class EffectiveRelationshipPreview {
	    fromCountry: string;
	    toCountry: string;
	    shareIntel: boolean;
	    airspaceTransitAllowed: boolean;
	    airspaceStrikeAllowed: boolean;
	    defensivePositioningAllowed: boolean;
	    maritimeTransitAllowed: boolean;
	    maritimeStrikeAllowed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EffectiveRelationshipPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fromCountry = source["fromCountry"];
	        this.toCountry = source["toCountry"];
	        this.shareIntel = source["shareIntel"];
	        this.airspaceTransitAllowed = source["airspaceTransitAllowed"];
	        this.airspaceStrikeAllowed = source["airspaceStrikeAllowed"];
	        this.defensivePositioningAllowed = source["defensivePositioningAllowed"];
	        this.maritimeTransitAllowed = source["maritimeTransitAllowed"];
	        this.maritimeStrikeAllowed = source["maritimeStrikeAllowed"];
	    }
	}
	export class PathViolationPreview {
	    blocked: boolean;
	    country?: string;
	    legIndex?: number;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new PathViolationPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.blocked = source["blocked"];
	        this.country = source["country"];
	        this.legIndex = source["legIndex"];
	        this.reason = source["reason"];
	    }
	}

}


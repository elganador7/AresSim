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
	export class EngagementOptionPreview {
	    targetUnitId: string;
	    targetDisplayName: string;
	    targetTeamId: string;
	    readyToFire: boolean;
	    canAssign: boolean;
	    weaponId?: string;
	    reason?: string;
	    reasonCode?: string;
	    rangeToTargetM?: number;
	    weaponRangeM?: number;
	    fireProbability?: number;
	    desiredEffectSupport: boolean;
	    inStrikeCooldown: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EngagementOptionPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetUnitId = source["targetUnitId"];
	        this.targetDisplayName = source["targetDisplayName"];
	        this.targetTeamId = source["targetTeamId"];
	        this.readyToFire = source["readyToFire"];
	        this.canAssign = source["canAssign"];
	        this.weaponId = source["weaponId"];
	        this.reason = source["reason"];
	        this.reasonCode = source["reasonCode"];
	        this.rangeToTargetM = source["rangeToTargetM"];
	        this.weaponRangeM = source["weaponRangeM"];
	        this.fireProbability = source["fireProbability"];
	        this.desiredEffectSupport = source["desiredEffectSupport"];
	        this.inStrikeCooldown = source["inStrikeCooldown"];
	    }
	}
	export class EngagementPreview {
	    readyToFire: boolean;
	    canAssign: boolean;
	    weaponId?: string;
	    reason?: string;
	    reasonCode?: string;
	    rangeToTargetM?: number;
	    weaponRangeM?: number;
	    fireProbability?: number;
	    desiredEffectSupport: boolean;
	    inStrikeCooldown: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EngagementPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.readyToFire = source["readyToFire"];
	        this.canAssign = source["canAssign"];
	        this.weaponId = source["weaponId"];
	        this.reason = source["reason"];
	        this.reasonCode = source["reasonCode"];
	        this.rangeToTargetM = source["rangeToTargetM"];
	        this.weaponRangeM = source["weaponRangeM"];
	        this.fireProbability = source["fireProbability"];
	        this.desiredEffectSupport = source["desiredEffectSupport"];
	        this.inStrikeCooldown = source["inStrikeCooldown"];
	    }
	}
	export class draftPointInput {
	    lat: number;
	    lon: number;
	
	    static createFrom(source: any = {}) {
	        return new draftPointInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lat = source["lat"];
	        this.lon = source["lon"];
	    }
	}
	export class PathViolationPreview {
	    blocked: boolean;
	    country?: string;
	    legIndex?: number;
	    reason?: string;
	    routePoints?: draftPointInput[];
	
	    static createFrom(source: any = {}) {
	        return new PathViolationPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.blocked = source["blocked"];
	        this.country = source["country"];
	        this.legIndex = source["legIndex"];
	        this.reason = source["reason"];
	        this.routePoints = this.convertValues(source["routePoints"], draftPointInput);
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
	export class TargetEngagementDebugSummary {
	    playerTeam: string;
	    targetUnitId: string;
	    targetDisplayName: string;
	    friendlyUnitCount: number;
	    readyShooterCount: number;
	    assignableShooterCount: number;
	    blockedShooterCount: number;
	    nonOperationalCount: number;
	    nonHostileCount: number;
	
	    static createFrom(source: any = {}) {
	        return new TargetEngagementDebugSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.playerTeam = source["playerTeam"];
	        this.targetUnitId = source["targetUnitId"];
	        this.targetDisplayName = source["targetDisplayName"];
	        this.friendlyUnitCount = source["friendlyUnitCount"];
	        this.readyShooterCount = source["readyShooterCount"];
	        this.assignableShooterCount = source["assignableShooterCount"];
	        this.blockedShooterCount = source["blockedShooterCount"];
	        this.nonOperationalCount = source["nonOperationalCount"];
	        this.nonHostileCount = source["nonHostileCount"];
	    }
	}
	export class TargetEngagementOptionPreview {
	    shooterUnitId: string;
	    shooterDisplayName: string;
	    shooterTeamId: string;
	    loadoutConfigurationId?: string;
	    readyToFire: boolean;
	    canAssign: boolean;
	    weaponId?: string;
	    reason?: string;
	    reasonCode?: string;
	    rangeToTargetM?: number;
	    weaponRangeM?: number;
	    fireProbability?: number;
	    desiredEffectSupport: boolean;
	    inStrikeCooldown: boolean;
	    pathBlocked: boolean;
	    pathReason?: string;
	    engagementCostUsd?: number;
	    expectedTargetValueUsd?: number;
	    expectedValueExchangeUsd?: number;
	
	    static createFrom(source: any = {}) {
	        return new TargetEngagementOptionPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.shooterUnitId = source["shooterUnitId"];
	        this.shooterDisplayName = source["shooterDisplayName"];
	        this.shooterTeamId = source["shooterTeamId"];
	        this.loadoutConfigurationId = source["loadoutConfigurationId"];
	        this.readyToFire = source["readyToFire"];
	        this.canAssign = source["canAssign"];
	        this.weaponId = source["weaponId"];
	        this.reason = source["reason"];
	        this.reasonCode = source["reasonCode"];
	        this.rangeToTargetM = source["rangeToTargetM"];
	        this.weaponRangeM = source["weaponRangeM"];
	        this.fireProbability = source["fireProbability"];
	        this.desiredEffectSupport = source["desiredEffectSupport"];
	        this.inStrikeCooldown = source["inStrikeCooldown"];
	        this.pathBlocked = source["pathBlocked"];
	        this.pathReason = source["pathReason"];
	        this.engagementCostUsd = source["engagementCostUsd"];
	        this.expectedTargetValueUsd = source["expectedTargetValueUsd"];
	        this.expectedValueExchangeUsd = source["expectedValueExchangeUsd"];
	    }
	}

}


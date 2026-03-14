package scenario

import enginev1 "github.com/aressim/internal/gen/engine/v1"

// DefaultWeaponDefinitions returns the global weapon catalog seeded on startup.
// Each entry describes a munition type independent of who carries it.
func DefaultWeaponDefinitions() []*enginev1.WeaponDefinition {
	L := enginev1.UnitDomain_DOMAIN_LAND
	A := enginev1.UnitDomain_DOMAIN_AIR
	S := enginev1.UnitDomain_DOMAIN_SEA
	U := enginev1.UnitDomain_DOMAIN_SUBSURFACE

	return []*enginev1.WeaponDefinition{
		// ── Land-domain guns ─────────────────────────────────────────────────────
		{
			Id: "gnd-120mm-apfsds", Name: "120mm APFSDS",
			Description:       "M829A3 kinetic-energy penetrator for MBTs",
			DomainTargets:     []enginev1.UnitDomain{L},
			SpeedMps:          1700, RangeM: 3500, ProbabilityOfHit: 0.85,
		},
		{
			Id: "gnd-25mm-hei", Name: "25mm HEI",
			Description:       "M919 25mm high-explosive incendiary for IFV autocannons",
			DomainTargets:     []enginev1.UnitDomain{L, A},
			SpeedMps:          1100, RangeM: 2500, ProbabilityOfHit: 0.70,
		},
		{
			Id: "gnd-155mm-he", Name: "155mm HE Shell",
			Description:       "M795 HE projectile for 155mm howitzers",
			DomainTargets:     []enginev1.UnitDomain{L},
			SpeedMps:          550, RangeM: 30000, ProbabilityOfHit: 0.75,
		},
		{
			Id: "gnd-gmlrs", Name: "GMLRS 227mm",
			Description:       "Guided MLRS rocket with GPS/INS precision guidance",
			DomainTargets:     []enginev1.UnitDomain{L},
			SpeedMps:          1075, RangeM: 84000, ProbabilityOfHit: 0.72,
		},
		{
			Id: "gnd-tow", Name: "BGM-71 TOW",
			Description:       "Wire-guided anti-tank missile",
			DomainTargets:     []enginev1.UnitDomain{L},
			SpeedMps:          318, RangeM: 4200, ProbabilityOfHit: 0.88,
		},
		{
			Id: "gnd-javelin", Name: "FGM-148 Javelin",
			Description:       "Fire-and-forget man-portable ATGM",
			DomainTargets:     []enginev1.UnitDomain{L},
			SpeedMps:          300, RangeM: 4000, ProbabilityOfHit: 0.90,
		},
		{
			Id: "gnd-556mm-rifle", Name: "5.56mm Rifle",
			Description:       "Standard infantry rifle ammunition",
			DomainTargets:     []enginev1.UnitDomain{L},
			SpeedMps:          940, RangeM: 500, ProbabilityOfHit: 0.60,
		},
		{
			Id: "gnd-at4", Name: "AT4 Rocket",
			Description:       "Single-shot 84mm unguided rocket launcher",
			DomainTargets:     []enginev1.UnitDomain{L},
			SpeedMps:          290, RangeM: 300, ProbabilityOfHit: 0.65,
		},
		// ── Air-to-air missiles (Western / US) ───────────────────────────────────
		{
			Id: "aam-aim120c", Name: "AIM-120C AMRAAM",
			Description:       "Beyond-visual-range active-radar AAM",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1450, RangeM: 105000, ProbabilityOfHit: 0.80,
		},
		{
			Id: "aam-aim9x", Name: "AIM-9X Sidewinder",
			Description:       "Short-range infrared-guided AAM",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1000, RangeM: 35000, ProbabilityOfHit: 0.78,
		},
		// ── Air-to-air missiles (Russian) ─────────────────────────────────────
		{
			Id: "aam-r77-1", Name: "R-77-1 (AA-12 Adder B)",
			Description:       "Russian active-radar BVR AAM; primary armament of Flanker/Felon family",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1480, RangeM: 110000, ProbabilityOfHit: 0.78,
		},
		{
			Id: "aam-r74m", Name: "R-74M (AA-11 Archer)",
			Description:       "Russian high-off-boresight short-range IR-guided AAM",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          900, RangeM: 40000, ProbabilityOfHit: 0.76,
		},
		{
			Id: "aam-r37m", Name: "R-37M (AA-13 Axehead)",
			Description:       "Russian very-long-range interceptor missile; primary armament of MiG-31BM",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1500, RangeM: 300000, ProbabilityOfHit: 0.72,
		},
		// ── Air-to-air missiles (Chinese) ─────────────────────────────────────
		{
			Id: "aam-pl15", Name: "PL-15 BVR Missile",
			Description:       "Chinese long-range active-radar BVR AAM for J-20/J-16/J-10C",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1500, RangeM: 200000, ProbabilityOfHit: 0.78,
		},
		{
			Id: "aam-pl10", Name: "PL-10 WVR Missile",
			Description:       "Chinese high-off-boresight short-range IR-guided AAM",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1000, RangeM: 20000, ProbabilityOfHit: 0.80,
		},
		{
			Id: "aam-pl12", Name: "PL-12 BVR Missile",
			Description:       "Chinese active-radar BVR AAM; primary on J-11B/J-15",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1400, RangeM: 100000, ProbabilityOfHit: 0.76,
		},
		// ── Air-to-air missiles (European) ────────────────────────────────────
		{
			Id: "aam-meteor", Name: "Meteor BVR Missile",
			Description:       "European ramjet-powered BVR AAM with very large no-escape zone; Typhoon/Rafale/Gripen E",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1500, RangeM: 200000, ProbabilityOfHit: 0.85,
		},
		{
			Id: "aam-mica-em", Name: "MICA-EM BVR Missile",
			Description:       "French active-radar BVR/WVR AAM; primary armament of Rafale and Mirage 2000-5",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1200, RangeM: 80000, ProbabilityOfHit: 0.80,
		},
		{
			Id: "aam-mica-ir", Name: "MICA-IR WVR Missile",
			Description:       "French IR-guided short-range AAM for Rafale and Mirage 2000-5",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1000, RangeM: 25000, ProbabilityOfHit: 0.82,
		},
		{
			Id: "aam-iris-t", Name: "IRIS-T WVR Missile",
			Description:       "German high-off-boresight short-range IR-guided AAM; Eurofighter/Gripen standard",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1000, RangeM: 25000, ProbabilityOfHit: 0.82,
		},
		// ── Air-to-air missiles (Israeli) ─────────────────────────────────────
		{
			Id: "aam-python5", Name: "Python-5 WVR Missile",
			Description:       "Israeli imaging-IR short-range AAM; exceptional off-boresight capability",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          900, RangeM: 20000, ProbabilityOfHit: 0.85,
		},
		{
			Id: "aam-derby", Name: "Derby / I-Derby ER BVR",
			Description:       "Israeli active-radar BVR AAM; I-Derby ER variant extends range to 100 km",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1300, RangeM: 100000, ProbabilityOfHit: 0.78,
		},
		// ── Air-to-air missiles (Indian) ──────────────────────────────────────
		{
			Id: "aam-astra-mk1", Name: "Astra Mk1 BVR",
			Description:       "Indian domestically developed active-radar BVR AAM for Su-30MKI and Tejas",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1400, RangeM: 80000, ProbabilityOfHit: 0.74,
		},
		// ── Air-to-ground / air-to-surface ───────────────────────────────────────
		{
			Id: "agm-114-hellfire", Name: "AGM-114 Hellfire",
			Description:       "Laser/RF-guided anti-armor missile for helicopters",
			DomainTargets:     []enginev1.UnitDomain{L, S},
			SpeedMps:          475, RangeM: 8000, ProbabilityOfHit: 0.90,
		},
		{
			Id: "agm-65-maverick", Name: "AGM-65 Maverick",
			Description:       "EO/IR-guided air-to-ground missile",
			DomainTargets:     []enginev1.UnitDomain{L, S},
			SpeedMps:          340, RangeM: 27000, ProbabilityOfHit: 0.82,
		},
		{
			Id: "agm-84-harpoon", Name: "AGM-84 Harpoon",
			Description:       "Air-launched anti-ship cruise missile",
			DomainTargets:     []enginev1.UnitDomain{S},
			SpeedMps:          310, RangeM: 290000, ProbabilityOfHit: 0.85,
		},
		{
			Id: "agm-88-harm", Name: "AGM-88 HARM",
			Description:       "High-speed anti-radiation missile",
			DomainTargets:     []enginev1.UnitDomain{L},
			SpeedMps:          1480, RangeM: 150000, ProbabilityOfHit: 0.80,
		},
		{
			Id: "gbu-32-jdam", Name: "GBU-32 JDAM",
			Description:       "GPS/INS-guided 1,000-lb precision bomb",
			DomainTargets:     []enginev1.UnitDomain{L, S},
			SpeedMps:          280, RangeM: 28000, ProbabilityOfHit: 0.88,
		},
		// ── Air-domain guns ───────────────────────────────────────────────────────
		{
			Id: "gun-m61a2-20mm", Name: "M61A2 Vulcan 20mm",
			Description:       "20mm rotary cannon for fighter aircraft",
			DomainTargets:     []enginev1.UnitDomain{A, L},
			SpeedMps:          1050, RangeM: 1800, ProbabilityOfHit: 0.75,
		},
		{
			Id: "gun-m230-30mm", Name: "M230 30mm Chain Gun",
			Description:       "30mm cannon for AH-64 Apache",
			DomainTargets:     []enginev1.UnitDomain{L},
			SpeedMps:          805, RangeM: 4000, ProbabilityOfHit: 0.78,
		},
		{
			Id: "gun-gsh301-30mm", Name: "GSh-30-1 30mm Cannon",
			Description:       "Single-barrel 30mm cannon on Russian Flanker/Felon family fighters",
			DomainTargets:     []enginev1.UnitDomain{A, L},
			SpeedMps:          900, RangeM: 2000, ProbabilityOfHit: 0.72,
		},
		{
			Id: "gun-mauser-bk27", Name: "Mauser BK-27 27mm Cannon",
			Description:       "27mm revolver cannon on Eurofighter Typhoon and Gripen",
			DomainTargets:     []enginev1.UnitDomain{A, L},
			SpeedMps:          1025, RangeM: 2000, ProbabilityOfHit: 0.74,
		},
		{
			Id: "gun-defa554-30mm", Name: "DEFA 554 30mm Cannon",
			Description:       "30mm revolver cannon on Rafale and Mirage 2000",
			DomainTargets:     []enginev1.UnitDomain{A, L},
			SpeedMps:          820, RangeM: 2000, ProbabilityOfHit: 0.70,
		},
		// ── Ship-launched surface-to-surface ─────────────────────────────────────
		{
			Id: "ssm-rgm84-harpoon", Name: "RGM-84 Harpoon",
			Description:       "Ship/sub-launched anti-ship cruise missile",
			DomainTargets:     []enginev1.UnitDomain{S},
			SpeedMps:          310, RangeM: 315000, ProbabilityOfHit: 0.82,
		},
		// ── Ship-launched surface-to-air ─────────────────────────────────────────
		{
			Id: "sam-sm2-block3", Name: "SM-2 Block III",
			Description:       "Long-range area air defense missile",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          900, RangeM: 167000, ProbabilityOfHit: 0.80,
		},
		{
			Id: "sam-sm6", Name: "SM-6",
			Description:       "Extended-range air defense / ballistic missile defense",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1200, RangeM: 370000, ProbabilityOfHit: 0.78,
		},
		{
			Id: "gun-phalanx-20mm", Name: "Phalanx CIWS 20mm",
			Description:       "Close-in weapon system; last-ditch anti-missile defense",
			DomainTargets:     []enginev1.UnitDomain{A},
			SpeedMps:          1100, RangeM: 3500, ProbabilityOfHit: 0.65,
		},
		{
			Id: "gun-mk45-5in", Name: "Mk 45 5\"/62 Naval Gun",
			Description:       "127mm naval gun for surface and shore fire",
			DomainTargets:     []enginev1.UnitDomain{L, S},
			SpeedMps:          920, RangeM: 24000, ProbabilityOfHit: 0.72,
		},
		// ── Anti-submarine warfare ────────────────────────────────────────────────
		{
			Id: "asw-vlasroc", Name: "VL-ASROC",
			Description:       "Vertically-launched rocket-propelled Mk 54 torpedo",
			DomainTargets:     []enginev1.UnitDomain{U},
			SpeedMps:          350, RangeM: 22000, ProbabilityOfHit: 0.75,
		},
		{
			Id: "torp-mk48", Name: "Mk 48 ADCAP Torpedo",
			Description:       "Heavy-weight wire-guided torpedo for submarines",
			DomainTargets:     []enginev1.UnitDomain{S, U},
			SpeedMps:          28, RangeM: 55000, ProbabilityOfHit: 0.80,
		},
		{
			Id: "torp-mk54", Name: "Mk 54 LHT Torpedo",
			Description:       "Lightweight hybrid torpedo for surface ships and aircraft",
			DomainTargets:     []enginev1.UnitDomain{U, S},
			SpeedMps:          28, RangeM: 18000, ProbabilityOfHit: 0.75,
		},
	}
}

// weaponSlot is a shorthand used in the loadout table below.
type weaponSlot struct {
	WeaponID    string
	MaxQty      int32
	InitialQty  int32
}

// defaultLoadouts maps UnitGeneralType (int32) to the default weapon loadout
// for units of that type. Initialized once on first call via loadoutsOnce.
// Loadouts reflect realistic standard configurations; no custom configs needed.
var defaultLoadouts = map[int32][]weaponSlot{
	// ── Fighter (GENERAL_TYPE_FIGHTER = 10) ─────────────────────────────────
	10: {
		{"aam-aim120c", 6, 6},
		{"aam-aim9x", 2, 2},
		{"gun-m61a2-20mm", 511, 511},
	},
	// ── Multirole (11) ────────────────────────────────────────────────────────
	11: {
		{"aam-aim120c", 4, 4},
		{"aam-aim9x", 2, 2},
		{"agm-65-maverick", 2, 2},
		{"gbu-32-jdam", 4, 4},
		{"gun-m61a2-20mm", 511, 511},
	},
	// ── Attack aircraft (12) ─────────────────────────────────────────────────
	12: {
		{"agm-65-maverick", 6, 6},
		{"gbu-32-jdam", 4, 4},
		{"agm-88-harm", 2, 2},
		{"aam-aim9x", 2, 2},
	},
	// ── Bomber (13) ──────────────────────────────────────────────────────────
	13: {
		{"gbu-32-jdam", 24, 24},
		{"agm-88-harm", 4, 4},
	},
	// ── Attack helicopter (GENERAL_TYPE_ATTACK_HELICOPTER = 20) ─────────────
	20: {
		{"agm-114-hellfire", 16, 16},
		{"gun-m230-30mm", 1200, 1200},
	},
	// ── Aircraft carrier (40) ────────────────────────────────────────────────
	40: {
		{"sam-sm2-block3", 24, 24},
		{"gun-phalanx-20mm", 20000, 20000},
	},
	// ── Destroyer (42) ───────────────────────────────────────────────────────
	42: {
		{"sam-sm6", 24, 24},
		{"sam-sm2-block3", 28, 28},
		{"ssm-rgm84-harpoon", 8, 8},
		{"asw-vlasroc", 16, 16},
		{"gun-mk45-5in", 600, 600},
		{"gun-phalanx-20mm", 20000, 20000},
	},
	// ── Frigate (43) ─────────────────────────────────────────────────────────
	43: {
		{"sam-sm2-block3", 16, 16},
		{"ssm-rgm84-harpoon", 8, 8},
		{"asw-vlasroc", 8, 8},
		{"torp-mk54", 4, 4},
		{"gun-mk45-5in", 300, 300},
	},
	// ── Corvette (44) ────────────────────────────────────────────────────────
	44: {
		{"ssm-rgm84-harpoon", 4, 4},
		{"sam-sm2-block3", 8, 8},
	},
	// ── Patrol vessel (45) ───────────────────────────────────────────────────
	45: {
		{"ssm-rgm84-harpoon", 2, 2},
		{"gun-mk45-5in", 200, 200},
	},
	// ── Attack submarine (50) ────────────────────────────────────────────────
	50: {
		{"torp-mk48", 26, 26},
		{"ssm-rgm84-harpoon", 12, 12},
	},
	// ── MBT (60) ─────────────────────────────────────────────────────────────
	60: {
		{"gnd-120mm-apfsds", 42, 42},
	},
	// ── IFV (61) ─────────────────────────────────────────────────────────────
	61: {
		{"gnd-25mm-hei", 900, 900},
		{"gnd-tow", 7, 7},
	},
	// ── APC (62) ─────────────────────────────────────────────────────────────
	62: {
		{"gnd-556mm-rifle", 3000, 3000},
	},
	// ── Recon vehicle (63) ───────────────────────────────────────────────────
	63: {
		{"gnd-25mm-hei", 300, 300},
	},
	// ── Self-propelled artillery (70) ────────────────────────────────────────
	70: {
		{"gnd-155mm-he", 39, 39},
	},
	// ── Towed artillery (71) ─────────────────────────────────────────────────
	71: {
		{"gnd-155mm-he", 30, 30},
	},
	// ── Rocket artillery MLRS (72) ───────────────────────────────────────────
	72: {
		{"gnd-gmlrs", 12, 12},
	},
	// ── Air defense (73) — generic SAM battery ────────────────────────────────
	73: {
		{"sam-sm2-block3", 8, 8},
	},
	// ── Special forces (80) ──────────────────────────────────────────────────
	80: {
		{"gnd-556mm-rifle", 1000, 1000},
		{"gnd-javelin", 2, 2},
	},
	// ── Light infantry (81) ──────────────────────────────────────────────────
	81: {
		{"gnd-556mm-rifle", 840, 840},
		{"gnd-at4", 4, 4},
	},
	// ── Airborne infantry (82) ───────────────────────────────────────────────
	82: {
		{"gnd-556mm-rifle", 840, 840},
		{"gnd-at4", 4, 4},
	},
	// ── Marine infantry (83) ─────────────────────────────────────────────────
	83: {
		{"gnd-556mm-rifle", 840, 840},
		{"gnd-at4", 4, 4},
	},
}

// InitUnitWeapons sets the weapons field on a unit if it has none, based on
// the unit definition's general type. Call this on every unit when loading a
// scenario for the first time.
func InitUnitWeapons(unit interface {
	GetWeapons() []*enginev1.WeaponState
	GetDefinitionId() string
}, generalType int32) []*enginev1.WeaponState {
	slots, ok := defaultLoadouts[generalType]
	if !ok || len(slots) == 0 {
		return nil
	}
	states := make([]*enginev1.WeaponState, 0, len(slots))
	for _, s := range slots {
		states = append(states, &enginev1.WeaponState{
			WeaponId:   s.WeaponID,
			CurrentQty: s.InitialQty,
			MaxQty:     s.MaxQty,
		})
	}
	return states
}

package scenario

import enginev1 "github.com/aressim/internal/gen/engine/v1"

// DefaultWeaponDefinitions returns the global weapon catalog seeded on startup.
// Each entry describes a munition type independent of who carries it.
func DefaultWeaponDefinitions() []*enginev1.WeaponDefinition {
	L := enginev1.UnitDomain_DOMAIN_LAND
	A := enginev1.UnitDomain_DOMAIN_AIR
	S := enginev1.UnitDomain_DOMAIN_SEA
	U := enginev1.UnitDomain_DOMAIN_SUBSURFACE

	unguided := enginev1.GuidanceType_GUIDANCE_UNGUIDED
	gps := enginev1.GuidanceType_GUIDANCE_GPS
	radar := enginev1.GuidanceType_GUIDANCE_RADAR
	ir := enginev1.GuidanceType_GUIDANCE_IR
	laser := enginev1.GuidanceType_GUIDANCE_LASER
	wire := enginev1.GuidanceType_GUIDANCE_WIRE
	sonar := enginev1.GuidanceType_GUIDANCE_SONAR

	defs := []*enginev1.WeaponDefinition{
		// ── Land-domain guns ─────────────────────────────────────────────────────
		{
			Id: "gnd-120mm-apfsds", Name: "120mm APFSDS",
			Description:   "M829A3 kinetic-energy penetrator for MBTs",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      1700, RangeM: 3500, ProbabilityOfHit: 0.85,
			Guidance: unguided,
		},
		{
			Id: "gnd-25mm-hei", Name: "25mm HEI",
			Description:   "M919 25mm high-explosive incendiary for IFV autocannons",
			DomainTargets: []enginev1.UnitDomain{L, A},
			SpeedMps:      1100, RangeM: 2500, ProbabilityOfHit: 0.70,
			Guidance: unguided,
		},
		{
			Id: "gnd-155mm-he", Name: "155mm HE Shell",
			Description:   "M795 HE projectile for 155mm howitzers",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      550, RangeM: 30000, ProbabilityOfHit: 0.75,
			Guidance: unguided,
		},
		{
			Id: "gnd-gmlrs", Name: "GMLRS 227mm",
			Description:   "Guided MLRS rocket with GPS/INS precision guidance",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      1075, RangeM: 84000, ProbabilityOfHit: 0.72,
			Guidance: gps,
		},
		{
			Id: "gnd-tow", Name: "BGM-71 TOW",
			Description:   "Wire-guided anti-tank missile",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      318, RangeM: 4200, ProbabilityOfHit: 0.88,
			Guidance: wire,
		},
		{
			Id: "gnd-javelin", Name: "FGM-148 Javelin",
			Description:   "Fire-and-forget man-portable ATGM with IR seeker",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      300, RangeM: 4000, ProbabilityOfHit: 0.90,
			Guidance: ir,
		},
		{
			Id: "gnd-spike-nlos", Name: "Spike NLOS",
			Description:   "Israeli long-range precision missile for helicopter, vehicle, and ground-launch strike",
			DomainTargets: []enginev1.UnitDomain{L, S},
			SpeedMps:      360, RangeM: 32000, ProbabilityOfHit: 0.87,
			Guidance: ir,
		},
		{
			Id: "gnd-556mm-rifle", Name: "5.56mm Rifle",
			Description:   "Standard infantry rifle ammunition",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      940, RangeM: 500, ProbabilityOfHit: 0.60,
			Guidance: unguided,
		},
		{
			Id: "gnd-at4", Name: "AT4 Rocket",
			Description:   "Single-shot 84mm unguided rocket launcher",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      290, RangeM: 300, ProbabilityOfHit: 0.65,
			Guidance: unguided,
		},
		// ── Air-to-air missiles (Western / US) ───────────────────────────────────
		{
			Id: "aam-aim120c", Name: "AIM-120C AMRAAM",
			Description:   "Beyond-visual-range active-radar AAM",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1450, RangeM: 105000, ProbabilityOfHit: 0.80,
			Guidance: radar,
		},
		{
			Id: "aam-aim9x", Name: "AIM-9X Sidewinder",
			Description:   "Short-range infrared-guided AAM",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1000, RangeM: 35000, ProbabilityOfHit: 0.78,
			Guidance: ir,
		},
		// ── Air-to-air missiles (Russian) ─────────────────────────────────────
		{
			Id: "aam-r77-1", Name: "R-77-1 (AA-12 Adder B)",
			Description:   "Russian active-radar BVR AAM; primary armament of Flanker/Felon family",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1480, RangeM: 110000, ProbabilityOfHit: 0.78,
			Guidance: radar,
		},
		{
			Id: "aam-r74m", Name: "R-74M (AA-11 Archer)",
			Description:   "Russian high-off-boresight short-range IR-guided AAM",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      900, RangeM: 40000, ProbabilityOfHit: 0.76,
			Guidance: ir,
		},
		{
			Id: "aam-r37m", Name: "R-37M (AA-13 Axehead)",
			Description:   "Russian very-long-range interceptor missile; primary armament of MiG-31BM",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1500, RangeM: 300000, ProbabilityOfHit: 0.72,
			Guidance: radar,
		},
		// ── Air-to-air missiles (Chinese) ─────────────────────────────────────
		{
			Id: "aam-pl15", Name: "PL-15 BVR Missile",
			Description:   "Chinese long-range active-radar BVR AAM for J-20/J-16/J-10C",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1500, RangeM: 200000, ProbabilityOfHit: 0.78,
			Guidance: radar,
		},
		{
			Id: "aam-pl10", Name: "PL-10 WVR Missile",
			Description:   "Chinese high-off-boresight short-range IR-guided AAM",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1000, RangeM: 20000, ProbabilityOfHit: 0.80,
			Guidance: ir,
		},
		{
			Id: "aam-pl12", Name: "PL-12 BVR Missile",
			Description:   "Chinese active-radar BVR AAM; primary on J-11B/J-15",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1400, RangeM: 100000, ProbabilityOfHit: 0.76,
			Guidance: radar,
		},
		// ── Air-to-air missiles (European) ────────────────────────────────────
		{
			Id: "aam-meteor", Name: "Meteor BVR Missile",
			Description:   "European ramjet-powered BVR AAM with very large no-escape zone; Typhoon/Rafale/Gripen E",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1500, RangeM: 200000, ProbabilityOfHit: 0.85,
			Guidance: radar,
		},
		{
			Id: "aam-mica-em", Name: "MICA-EM BVR Missile",
			Description:   "French active-radar BVR/WVR AAM; primary armament of Rafale and Mirage 2000-5",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1200, RangeM: 80000, ProbabilityOfHit: 0.80,
			Guidance: radar,
		},
		{
			Id: "aam-mica-ir", Name: "MICA-IR WVR Missile",
			Description:   "French IR-guided short-range AAM for Rafale and Mirage 2000-5",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1000, RangeM: 25000, ProbabilityOfHit: 0.82,
			Guidance: ir,
		},
		{
			Id: "aam-iris-t", Name: "IRIS-T WVR Missile",
			Description:   "German high-off-boresight short-range IR-guided AAM; Eurofighter/Gripen standard",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1000, RangeM: 25000, ProbabilityOfHit: 0.82,
			Guidance: ir,
		},
		// ── Air-to-air missiles (Israeli) ─────────────────────────────────────
		{
			Id: "aam-python5", Name: "Python-5 WVR Missile",
			Description:   "Israeli imaging-IR short-range AAM; exceptional off-boresight capability",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      900, RangeM: 20000, ProbabilityOfHit: 0.85,
			Guidance: ir,
		},
		{
			Id: "aam-derby", Name: "Derby / I-Derby ER BVR",
			Description:   "Israeli active-radar BVR AAM; I-Derby ER variant extends range to 100 km",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1300, RangeM: 100000, ProbabilityOfHit: 0.78,
			Guidance: radar,
		},
		// ── Air-to-air missiles (Indian) ──────────────────────────────────────
		{
			Id: "aam-astra-mk1", Name: "Astra Mk1 BVR",
			Description:   "Indian domestically developed active-radar BVR AAM for Su-30MKI and Tejas",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1400, RangeM: 80000, ProbabilityOfHit: 0.74,
			Guidance: radar,
		},
		// ── Air-to-ground / air-to-surface ───────────────────────────────────────
		{
			Id: "agm-114-hellfire", Name: "AGM-114 Hellfire",
			Description:   "Semi-active laser or radar-guided anti-armor missile for helicopters",
			DomainTargets: []enginev1.UnitDomain{L, S},
			SpeedMps:      475, RangeM: 8000, ProbabilityOfHit: 0.90,
			Guidance: laser,
		},
		{
			Id: "agm-65-maverick", Name: "AGM-65 Maverick",
			Description:   "EO/IR-guided air-to-ground missile",
			DomainTargets: []enginev1.UnitDomain{L, S},
			SpeedMps:      340, RangeM: 27000, ProbabilityOfHit: 0.82,
			Guidance: ir,
		},
		{
			Id: "agm-84-harpoon", Name: "AGM-84 Harpoon",
			Description:   "Air-launched anti-ship cruise missile with active radar seeker",
			DomainTargets: []enginev1.UnitDomain{S},
			SpeedMps:      310, RangeM: 290000, ProbabilityOfHit: 0.85,
			Guidance: radar,
		},
		{
			Id: "agm-88-harm", Name: "AGM-88 HARM",
			Description:   "High-speed anti-radiation missile; homes on radar emissions",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      1480, RangeM: 150000, ProbabilityOfHit: 0.80,
			Guidance: radar,
		},
		{
			Id: "gbu-32-jdam", Name: "GBU-32 JDAM",
			Description:   "GPS/INS-guided 1,000-lb precision bomb",
			DomainTargets: []enginev1.UnitDomain{L, S},
			SpeedMps:      280, RangeM: 28000, ProbabilityOfHit: 0.88,
			Guidance: gps,
		},
		{
			Id: "agm-154-jsow", Name: "AGM-154 JSOW",
			Description:   "Standoff glide weapon for precision strike against fixed and relocatable targets",
			DomainTargets: []enginev1.UnitDomain{L, S},
			SpeedMps:      250, RangeM: 130000, ProbabilityOfHit: 0.84,
			Guidance: gps,
		},
		{
			Id: "agm-158-jassm-er", Name: "AGM-158B JASSM-ER",
			Description:   "Low-observable extended-range cruise missile for land and maritime strike",
			DomainTargets: []enginev1.UnitDomain{L, S},
			SpeedMps:      280, RangeM: 925000, ProbabilityOfHit: 0.86,
			Guidance: gps,
		},
		{
			Id: "agm-158c-lrasm", Name: "AGM-158C LRASM",
			Description:   "Long-range anti-ship cruise missile derived from JASSM-ER",
			DomainTargets: []enginev1.UnitDomain{S},
			SpeedMps:      280, RangeM: 500000, ProbabilityOfHit: 0.88,
			Guidance: radar,
		},
		{
			Id: "asm-am39-exocet", Name: "AM39 Exocet",
			Description:   "French sea-skimming anti-ship missile used by aircraft and surface combatants",
			DomainTargets: []enginev1.UnitDomain{S},
			SpeedMps:      315, RangeM: 180000, ProbabilityOfHit: 0.83,
			Guidance: radar,
		},
		{
			Id: "asm-nsm", Name: "Naval Strike Missile",
			Description:   "Imaging-seeker anti-ship and land-attack missile with low-observable profile",
			DomainTargets: []enginev1.UnitDomain{S, L},
			SpeedMps:      300, RangeM: 250000, ProbabilityOfHit: 0.86,
			Guidance: ir,
		},
		{
			Id: "asm-gabriel-v", Name: "Gabriel V",
			Description:   "Israeli long-range sea-skimming anti-ship missile for naval and coastal strike",
			DomainTargets: []enginev1.UnitDomain{S, L},
			SpeedMps:      300, RangeM: 200000, ProbabilityOfHit: 0.85,
			Guidance: radar,
		},
		{
			Id: "asm-brahmos", Name: "BrahMos",
			Description:   "Supersonic cruise missile for ship, land, and maritime strike missions",
			DomainTargets: []enginev1.UnitDomain{S, L},
			SpeedMps:      850, RangeM: 450000, ProbabilityOfHit: 0.84,
			Guidance: radar,
		},
		{
			Id: "asm-noor", Name: "Noor / Qader",
			Description:   "Iranian anti-ship cruise missile family used from shore batteries, fast attack craft, and larger combatants",
			DomainTargets: []enginev1.UnitDomain{S},
			SpeedMps:      280, RangeM: 200000, ProbabilityOfHit: 0.79,
			Guidance: radar,
		},
		{
			Id: "asm-abu-mahdi", Name: "Abu Mahdi",
			Description:   "Iranian long-range anti-ship cruise missile for coastal denial and maritime strike",
			DomainTargets: []enginev1.UnitDomain{S},
			SpeedMps:      260, RangeM: 700000, ProbabilityOfHit: 0.76,
			Guidance: radar,
		},
		{
			Id: "agm-kh31p", Name: "Kh-31P",
			Description:   "Russian anti-radiation missile for suppression of enemy air defenses",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      1000, RangeM: 110000, ProbabilityOfHit: 0.78,
			Guidance: radar,
		},
		// ── Air-domain guns ───────────────────────────────────────────────────────
		{
			Id: "gun-m61a2-20mm", Name: "M61A2 Vulcan 20mm",
			Description:   "20mm rotary cannon for fighter aircraft",
			DomainTargets: []enginev1.UnitDomain{A, L},
			SpeedMps:      1050, RangeM: 1800, ProbabilityOfHit: 0.75,
			Guidance: unguided,
		},
		{
			Id: "gun-m230-30mm", Name: "M230 30mm Chain Gun",
			Description:   "30mm cannon for AH-64 Apache",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      805, RangeM: 4000, ProbabilityOfHit: 0.78,
			Guidance: unguided,
		},
		{
			Id: "gun-gsh301-30mm", Name: "GSh-30-1 30mm Cannon",
			Description:   "Single-barrel 30mm cannon on Russian Flanker/Felon family fighters",
			DomainTargets: []enginev1.UnitDomain{A, L},
			SpeedMps:      900, RangeM: 2000, ProbabilityOfHit: 0.72,
			Guidance: unguided,
		},
		{
			Id: "gun-mauser-bk27", Name: "Mauser BK-27 27mm Cannon",
			Description:   "27mm revolver cannon on Eurofighter Typhoon and Gripen",
			DomainTargets: []enginev1.UnitDomain{A, L},
			SpeedMps:      1025, RangeM: 2000, ProbabilityOfHit: 0.74,
			Guidance: unguided,
		},
		{
			Id: "gun-defa554-30mm", Name: "DEFA 554 30mm Cannon",
			Description:   "30mm revolver cannon on Rafale and Mirage 2000",
			DomainTargets: []enginev1.UnitDomain{A, L},
			SpeedMps:      820, RangeM: 2000, ProbabilityOfHit: 0.70,
			Guidance: unguided,
		},
		// ── Ship-launched surface-to-surface ─────────────────────────────────────
		{
			Id: "ssm-rgm84-harpoon", Name: "RGM-84 Harpoon",
			Description:   "Ship/sub-launched anti-ship cruise missile with active radar seeker",
			DomainTargets: []enginev1.UnitDomain{S},
			SpeedMps:      310, RangeM: 315000, ProbabilityOfHit: 0.82,
			Guidance: radar,
		},
		// ── Ship-launched surface-to-air ─────────────────────────────────────────
		{
			Id: "sam-sm2-block3", Name: "SM-2 Block III",
			Description:   "Long-range area air defense missile with semi-active/active radar",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      900, RangeM: 167000, ProbabilityOfHit: 0.80,
			Guidance: radar,
		},
		{
			Id: "sam-sm6", Name: "SM-6",
			Description:   "Extended-range air defense / ballistic missile defense; active radar seeker",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1200, RangeM: 370000, ProbabilityOfHit: 0.78,
			Guidance: radar,
		},
		{
			Id: "sam-patriot-pac3", Name: "Patriot PAC-3 MSE",
			Description:   "Land-based hit-to-kill interceptor for aircraft and missile defense",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1600, RangeM: 120000, ProbabilityOfHit: 0.81,
			Guidance: radar,
		},
		{
			Id: "sam-nasams-amraam-er", Name: "NASAMS AMRAAM-ER",
			Description:   "Medium-range networked surface-to-air missile using AMRAAM-ER interceptors",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1100, RangeM: 50000, ProbabilityOfHit: 0.79,
			Guidance: radar,
		},
		{
			Id: "sam-vl-mica", Name: "VL MICA",
			Description:   "Vertical-launch MICA family interceptor for point and local-area air defense",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      950, RangeM: 25000, ProbabilityOfHit: 0.77,
			Guidance: radar,
		},
		{
			Id: "sam-s400-40n6", Name: "S-400 40N6",
			Description:   "Very-long-range Russian surface-to-air missile for high-value aerial targets",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1400, RangeM: 380000, ProbabilityOfHit: 0.77,
			Guidance: radar,
		},
		{
			Id: "sam-irist-slm", Name: "IRIS-T SLM",
			Description:   "Short-to-medium range air defense missile with imaging infrared terminal seeker",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1000, RangeM: 40000, ProbabilityOfHit: 0.8,
			Guidance: ir,
		},
		{
			Id: "sam-hq9b", Name: "HQ-9B",
			Description:   "Chinese long-range area air defense missile system",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1350, RangeM: 250000, ProbabilityOfHit: 0.78,
			Guidance: radar,
		},
		{
			Id: "sam-s300pmu2", Name: "S-300PMU-2",
			Description:   "Long-range Russian-origin strategic surface-to-air missile used for high-value air-defense coverage",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1450, RangeM: 200000, ProbabilityOfHit: 0.79,
			Guidance: radar,
		},
		{
			Id: "sam-sayyad4b", Name: "Sayyad-4B",
			Description:   "Iranian long-range SAM used by Bavar-373 for strategic air and missile defense",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1500, RangeM: 300000, ProbabilityOfHit: 0.77,
			Guidance: radar,
		},
		{
			Id: "sam-3rd-khordad", Name: "3rd Khordad",
			Description:   "Iranian medium-range mobile SAM for air-defense ambush and high-value target engagement",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1200, RangeM: 105000, ProbabilityOfHit: 0.75,
			Guidance: radar,
		},
		{
			Id: "sam-tor-m1", Name: "Tor-M1",
			Description:   "Short-range point-defense SAM for cruise missile, aircraft, and guided munition interception",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      850, RangeM: 16000, ProbabilityOfHit: 0.74,
			Guidance: radar,
		},
		{
			Id: "sam-barak-8", Name: "Barak-8",
			Description:   "Israeli area air defense missile for naval and land-based interception",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1200, RangeM: 100000, ProbabilityOfHit: 0.80,
			Guidance: radar,
		},
		{
			Id: "sam-tamir", Name: "Tamir",
			Description:   "Iron Dome interceptor for short-range rockets, artillery, and aircraft threats",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      750, RangeM: 70000, ProbabilityOfHit: 0.82,
			Guidance: radar,
		},
		{
			Id: "sam-stunner", Name: "Stunner",
			Description:   "David's Sling interceptor for cruise missiles, aircraft, and theater ballistic threats",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1400, RangeM: 250000, ProbabilityOfHit: 0.80,
			Guidance: radar,
		},
		{
			Id: "sam-arrow3", Name: "Arrow-3",
			Description:   "Israeli exo-atmospheric interceptor for long-range ballistic missile defense",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      2500, RangeM: 500000, ProbabilityOfHit: 0.76,
			Guidance: radar,
		},
		{
			Id: "sam-arrow2", Name: "Arrow-2",
			Description:   "Israeli upper-tier interceptor for endo-atmospheric ballistic missile defense",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      2200, RangeM: 150000, ProbabilityOfHit: 0.79,
			Guidance: radar,
		},
		{
			Id: "sam-spyder-python5", Name: "SPYDER Python-5",
			Description:   "Israeli short-range quick-reaction interceptor using the Python-5 seeker",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1000, RangeM: 20000, ProbabilityOfHit: 0.80,
			Guidance: ir,
		},
		{
			Id: "sam-spyder-derby-er", Name: "SPYDER Derby-ER",
			Description:   "Israeli medium-range networked interceptor derived from the I-Derby ER missile",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1100, RangeM: 50000, ProbabilityOfHit: 0.78,
			Guidance: radar,
		},
		{
			Id: "sam-ihawk", Name: "I-Hawk",
			Description:   "Legacy medium-range semi-active radar-guided surface-to-air missile",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      900, RangeM: 40000, ProbabilityOfHit: 0.68,
			Guidance: radar,
		},
		{
			Id: "gun-phalanx-20mm", Name: "Phalanx CIWS 20mm",
			Description:   "Close-in weapon system; last-ditch anti-missile defense",
			DomainTargets: []enginev1.UnitDomain{A},
			SpeedMps:      1100, RangeM: 3500, ProbabilityOfHit: 0.65,
			Guidance: unguided,
		},
		{
			Id: "gun-mk45-5in", Name: "Mk 45 5\"/62 Naval Gun",
			Description:   "127mm naval gun for surface and shore fire",
			DomainTargets: []enginev1.UnitDomain{L, S},
			SpeedMps:      920, RangeM: 24000, ProbabilityOfHit: 0.72,
			Guidance: unguided,
		},
		// ── Anti-submarine warfare ────────────────────────────────────────────────
		{
			Id: "asw-vlasroc", Name: "VL-ASROC",
			Description:   "Vertically-launched rocket-propelled Mk 54 torpedo; GPS midcourse, sonar terminal",
			DomainTargets: []enginev1.UnitDomain{U},
			SpeedMps:      350, RangeM: 22000, ProbabilityOfHit: 0.75,
			Guidance: sonar,
		},
		{
			Id: "torp-mk48", Name: "Mk 48 ADCAP Torpedo",
			Description:   "Heavy-weight wire-guided torpedo for submarines",
			DomainTargets: []enginev1.UnitDomain{S, U},
			SpeedMps:      28, RangeM: 55000, ProbabilityOfHit: 0.80,
			Guidance: sonar,
		},
		{
			Id: "torp-mk54", Name: "Mk 54 LHT Torpedo",
			Description:   "Lightweight hybrid torpedo for surface ships and aircraft",
			DomainTargets: []enginev1.UnitDomain{U, S},
			SpeedMps:      28, RangeM: 18000, ProbabilityOfHit: 0.75,
			Guidance: sonar,
		},
		{
			Id: "ssm-tomahawk", Name: "Tomahawk Land Attack Missile",
			Description:   "Long-range subsonic land-attack cruise missile for surface ships and submarines",
			DomainTargets: []enginev1.UnitDomain{L, S},
			SpeedMps:      245, RangeM: 1600000, ProbabilityOfHit: 0.84,
			Guidance: gps,
		},
		{
			Id: "ssm-kalibr", Name: "3M-14 Kalibr",
			Description:   "Russian long-range land-attack and antiship cruise missile family",
			DomainTargets: []enginev1.UnitDomain{L, S},
			SpeedMps:      250, RangeM: 1500000, ProbabilityOfHit: 0.82,
			Guidance: gps,
		},
		{
			Id: "ssm-fateh110", Name: "Fateh-110",
			Description:   "Iranian road-mobile short-range ballistic missile for regional precision strike",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      1400, RangeM: 300000, ProbabilityOfHit: 0.74,
			Guidance: gps,
		},
		{
			Id: "ssm-qiam1", Name: "Qiam-1",
			Description:   "Iranian liquid-fuel SRBM used for larger regional salvos against fixed targets",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      1500, RangeM: 800000, ProbabilityOfHit: 0.70,
			Guidance: gps,
		},
		{
			Id: "ssm-kheibar-shekan", Name: "Kheibar Shekan",
			Description:   "Iranian solid-fuel MRBM intended for deep regional strike against defended bases and infrastructure",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      1800, RangeM: 1450000, ProbabilityOfHit: 0.72,
			Guidance: gps,
		},
		{
			Id: "ssm-sejjil", Name: "Sejjil",
			Description:   "Iranian two-stage solid-fuel MRBM for long-range theater strike",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      1900, RangeM: 2000000, ProbabilityOfHit: 0.68,
			Guidance: gps,
		},
		{
			Id: "ssm-paveh", Name: "Paveh",
			Description:   "Iranian long-range land-attack cruise missile for terrain-following regional strike",
			DomainTargets: []enginev1.UnitDomain{L},
			SpeedMps:      240, RangeM: 1650000, ProbabilityOfHit: 0.80,
			Guidance: gps,
		},
	}
	for _, wd := range defs {
		wd.EffectType = defaultWeaponEffectType(wd.Id, wd.DomainTargets)
	}
	return defs
}

// weaponSlot is a shorthand used in the loadout table below.
type weaponSlot struct {
	WeaponID   string
	MaxQty     int32
	InitialQty int32
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

func defaultWeaponEffectType(id string, domains []enginev1.UnitDomain) enginev1.WeaponEffectType {
	switch {
	case id == "torp-mk48":
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_TORPEDO
	case hasPrefix(id, "sam-"):
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_INTERCEPTOR
	case hasPrefix(id, "aam-"):
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_AIR
	case hasPrefix(id, "asm-"), id == "agm-84-harpoon", id == "agm-158c-lrasm", id == "ssm-rgm84-harpoon":
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_SHIP
	case id == "agm-88-harm", id == "agm-kh31p":
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_SEAD
	case hasPrefix(id, "ssm-"):
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE
	case id == "gnd-120mm-apfsds", id == "gnd-tow", id == "gnd-javelin":
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_ARMOR
	case hasPrefix(id, "gun-"), id == "gnd-25mm-hei", id == "gnd-556mm-rifle", id == "gnd-at4":
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_GUNFIRE
	default:
		for _, domain := range domains {
			if domain == enginev1.UnitDomain_DOMAIN_LAND {
				return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE
			}
		}
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_UNSPECIFIED
	}
}

func hasPrefix(value, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}

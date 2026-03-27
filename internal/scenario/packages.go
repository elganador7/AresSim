package scenario

import (
	"fmt"
	"strings"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"google.golang.org/protobuf/proto"
)

// PackageTemplate is a reusable bundle of units that can be composed into
// scenarios. Units are authored around a reference anchor and translated at
// instantiation time.
type PackageTemplate struct {
	ID           string
	Name         string
	Category     string
	Description  string
	Version      string
	Priority     int32
	ReferenceLat float64
	ReferenceLon float64
	Accuracy     PackageAccuracyProfile
	Units        []*enginev1.Unit
}

type PackageInstanceOptions struct {
	Prefix         string
	AnchorLat      float64
	AnchorLon      float64
	TeamID         string
	CoalitionID    string
	OperatorTeamID string
}

type PackageAccuracyProfile struct {
	OperationalRole string
	Assumptions     []string
	ValidationGoals []string
	Sources         []PackageSource
}

type PackageSource struct {
	Label string
	URL   string
	Kind  string
}

func PackageTemplateOrder() []string {
	return []string{
		"pkg-isr-iron-dome-battery",
		"pkg-isr-layered-defense-dan",
		"pkg-isr-counterstrike-cell",
		"pkg-usa-al-udeid-airbase",
		"pkg-are-al-dhafra-airbase",
		"pkg-are-abu-dhabi-coastal-defense",
		"pkg-omn-musandam-strait-guard",
		"pkg-bhr-naval-support-bahrain",
		"pkg-irn-hormuz-coastal-denial",
		"pkg-irn-western-missile-regiment",
	}
}

func PackageTemplates() map[string]PackageTemplate {
	return map[string]PackageTemplate{
		"pkg-isr-iron-dome-battery":        packageISRIronDomeBattery(),
		"pkg-isr-layered-defense-dan":      packageISRLayeredDefenseDan(),
		"pkg-isr-counterstrike-cell":       packageISRCounterstrikeCell(),
		"pkg-usa-al-udeid-airbase":         packageUSAAlUdeidAirbase(),
		"pkg-are-al-dhafra-airbase":        packageAREAlDhafraAirbase(),
		"pkg-are-abu-dhabi-coastal-defense": packageAREAbuDhabiCoastalDefense(),
		"pkg-omn-musandam-strait-guard":    packageOMNMusandamStraitGuard(),
		"pkg-bhr-naval-support-bahrain":    packageBHRNavalSupportBahrain(),
		"pkg-irn-hormuz-coastal-denial":    packageIRNHormuzCoastalDenial(),
		"pkg-irn-western-missile-regiment": packageIRNWesternMissileRegiment(),
	}
}

func PackageTemplateByID(id string) (PackageTemplate, bool) {
	pkg, ok := PackageTemplates()[strings.TrimSpace(id)]
	return pkg, ok
}

func InstantiatePackage(template PackageTemplate, opts PackageInstanceOptions) ([]*enginev1.Unit, error) {
	if strings.TrimSpace(template.ID) == "" {
		return nil, fmt.Errorf("package template missing id")
	}
	if len(template.Units) == 0 {
		return nil, fmt.Errorf("package template %s has no units", template.ID)
	}
	prefix := strings.TrimSpace(opts.Prefix)
	if prefix == "" {
		prefix = template.ID
	}
	anchorLat := opts.AnchorLat
	anchorLon := opts.AnchorLon
	if anchorLat == 0 && anchorLon == 0 {
		anchorLat = template.ReferenceLat
		anchorLon = template.ReferenceLon
	}
	dLat := anchorLat - template.ReferenceLat
	dLon := anchorLon - template.ReferenceLon

	idMap := make(map[string]string, len(template.Units))
	out := make([]*enginev1.Unit, 0, len(template.Units))
	for _, unit := range template.Units {
		if unit == nil {
			continue
		}
		clone := proto.Clone(unit).(*enginev1.Unit)
		oldID := clone.GetId()
		if oldID == "" {
			return nil, fmt.Errorf("package template %s contains unit with empty id", template.ID)
		}
		newID := prefix + "-" + oldID
		idMap[oldID] = newID
		clone.Id = newID
		if opts.TeamID != "" {
			clone.TeamId = opts.TeamID
		}
		if opts.CoalitionID != "" {
			clone.CoalitionId = opts.CoalitionID
		}
		if opts.OperatorTeamID != "" {
			clone.OperatorTeamId = opts.OperatorTeamID
		}
		if clone.GetPosition() != nil {
			clone.Position = &enginev1.Position{
				Lat:     clone.GetPosition().GetLat() + dLat,
				Lon:     clone.GetPosition().GetLon() + dLon,
				AltMsl:  clone.GetPosition().GetAltMsl(),
				Heading: clone.GetPosition().GetHeading(),
				Speed:   clone.GetPosition().GetSpeed(),
			}
		}
		out = append(out, clone)
	}
	for _, unit := range out {
		if mapped, ok := idMap[unit.GetHostBaseId()]; ok {
			unit.HostBaseId = mapped
		}
		if order := unit.GetAttackOrder(); order != nil {
			if mapped, ok := idMap[order.GetTargetUnitId()]; ok {
				order.TargetUnitId = mapped
			}
		}
	}
	return out, nil
}

func ValidatePackageTemplate(template PackageTemplate) error {
	if strings.TrimSpace(template.ID) == "" {
		return fmt.Errorf("package template missing id")
	}
	if strings.TrimSpace(template.Name) == "" {
		return fmt.Errorf("package template %s missing name", template.ID)
	}
	if template.Priority <= 0 {
		return fmt.Errorf("package template %s missing priority", template.ID)
	}
	if strings.TrimSpace(template.Version) == "" {
		return fmt.Errorf("package template %s missing version", template.ID)
	}
	if strings.TrimSpace(template.Accuracy.OperationalRole) == "" {
		return fmt.Errorf("package template %s missing operational role", template.ID)
	}
	if len(template.Accuracy.ValidationGoals) == 0 {
		return fmt.Errorf("package template %s missing validation goals", template.ID)
	}
	if len(template.Units) == 0 {
		return fmt.Errorf("package template %s has no units", template.ID)
	}
	seen := map[string]struct{}{}
	for _, unit := range template.Units {
		if unit == nil {
			return fmt.Errorf("package template %s contains nil unit", template.ID)
		}
		id := strings.TrimSpace(unit.GetId())
		if id == "" {
			return fmt.Errorf("package template %s contains unit with empty id", template.ID)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("package template %s contains duplicate unit id %s", template.ID, id)
		}
		seen[id] = struct{}{}
		if strings.TrimSpace(unit.GetDefinitionId()) == "" {
			return fmt.Errorf("package template %s unit %s missing definition id", template.ID, id)
		}
	}
	return nil
}

func packageISRIronDomeBattery() PackageTemplate {
	return PackageTemplate{
		ID:           "pkg-isr-iron-dome-battery",
		Name:         "ISR Iron Dome Battery",
		Category:     "air_defense",
		Description:  "Single Iron Dome battery with representative magazine depth for local area defense.",
		Version:      "1.1.0",
		Priority:     1,
		ReferenceLat: 31.89,
		ReferenceLon: 34.79,
		Accuracy: PackageAccuracyProfile{
			OperationalRole: "Short-range point and area defense against rockets, UAVs, cruise missiles, and low-altitude air threats around a defended urban or infrastructure cluster.",
			Assumptions: []string{
				"One package instance represents one deployed Iron Dome firing battery rather than the full national network.",
				"The package uses the library default 60-round Tamir magazine, representing a three-launcher public-reference battery depth rather than a partially expended unit.",
				"Battle-management, radar, and launch elements remain abstracted into the battery definition until separate subcomponents exist in the library.",
			},
			ValidationGoals: []string{
				"Intercept a majority of a small inbound raid without creating perfect leakage-free defense.",
				"Exhaust magazine depth under sustained pressure before layered upper-tier systems do.",
			},
			Sources: []PackageSource{
				{Label: "Israel MOD public Iron Dome system releases", URL: "https://www.mod.gov.il", Kind: "official"},
				{Label: "Rafael Iron Dome overview", URL: "https://www.rafael.co.il", Kind: "official"},
				{Label: "U.S. MDA cooperative missile-defense releases", URL: "https://www.mda.mil", Kind: "official"},
			},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("radar", "Iron Dome Radar", "Iron Dome Radar Section Package", "ISR", "COALITION_WEST", "iron-dome-radar-section", 31.90, 34.80, 0, 0, 0),
			provingGroundUnit("bmc", "Iron Dome BMC", "Iron Dome Battle Management Package", "ISR", "COALITION_WEST", "iron-dome-battle-management-center", 31.88, 34.77, 0, 0, 0),
			provingGroundUnit("battery", "Iron Dome", "Iron Dome Battery Package", "ISR", "COALITION_WEST", "iron-dome-battery", 31.89, 34.79, 0, 0, 0),
		},
	}
}

func packageISRLayeredDefenseDan() PackageTemplate {
	return PackageTemplate{
		ID:           "pkg-isr-layered-defense-dan",
		Name:         "ISR Layered Defense Dan",
		Category:     "air_defense",
		Description:  "Representative layered missile-defense posture for central Israel with Arrow, David's Sling, and Iron Dome elements.",
		Version:      "1.5.0",
		Priority:     2,
		ReferenceLat: 31.95,
		ReferenceLon: 34.78,
		Accuracy: PackageAccuracyProfile{
			OperationalRole: "Layered missile and air defense for Israel's central population, airbase, and command belt using upper-tier, mid-tier, and point-defense systems.",
			Assumptions: []string{
				"This package is a defended-region abstraction for the Dan area, not a literal ORBAT of every battery in the district.",
				"Arrow batteries represent strategic upper-tier BMD, David's Sling provides medium-range missile and cruise-missile defense, and Iron Dome batteries absorb short-range leakage and raid saturation.",
				"A national C2 node and early-warning radar are included because the package is intended to represent an operationally integrated defended region, not an isolated battery cluster.",
				"Battery-level launcher counts remain abstracted, but Arrow, David's Sling, Barak MX, and Iron Dome now include explicit local sensor and battle-management components in the package.",
				"An airborne Eitam is included as a representative on-station AEW layer because the package is intended as an operational defended-region default state rather than a parked peacetime posture.",
				"The package geometry is deliberately spread across the Tel Aviv-Jerusalem corridor so upper-tier, mid-tier, and point-defense nodes are not unrealistically co-located.",
			},
			ValidationGoals: []string{
				"Intercept most mixed ballistic and cruise threats in small proving grounds.",
				"Still allow leakage under saturation and magazine depletion.",
				"Enable counterstrike scenarios to evaluate attacker losses against a defended target set.",
			},
			Sources: []PackageSource{
				{Label: "Israel MOD Arrow / David's Sling releases", URL: "https://www.mod.gov.il", Kind: "official"},
				{Label: "U.S. MDA cooperative missile-defense releases", URL: "https://www.mda.mil", Kind: "official"},
			},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("national-c2", "National Air Defense C2", "Israeli National C2 - Dan Package", "ISR", "COALITION_WEST", "israel-national-c2-site", 32.04, 34.84, 0, 0, 0),
			provingGroundUnit("early-warning-radar", "Early Warning Radar", "Israeli Early Warning Radar - Dan Package", "ISR", "COALITION_WEST", "israel-early-warning-radar-site", 31.61, 35.24, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingAircraft("eitam", "Eitam", "Israeli Eitam AEW - Dan Package", "ISR", "COALITION_WEST", "g550-eitam", 32.38, 35.02, 10500, 180, 220)
				u.HostBaseId = ""
				return u
			}(),
			provingGroundUnit("arrow-radar", "Green Pine", "Green Pine Radar - Dan Package", "ISR", "COALITION_WEST", "green-pine-radar-section", 31.66, 35.31, 0, 0, 0),
			provingGroundUnit("arrow-bmc", "Arrow BMC", "Arrow Battle Management - Dan Package", "ISR", "COALITION_WEST", "arrow-battle-management-center", 31.70, 35.18, 0, 0, 0),
			provingGroundUnit("arrow3", "Arrow-3", "Arrow-3 Package Battery", "ISR", "COALITION_WEST", "arrow3-battery", 31.72, 35.23, 0, 0, 0),
			provingGroundUnit("arrow2", "Arrow-2", "Arrow-2 Package Battery", "ISR", "COALITION_WEST", "arrow2-battery", 31.86, 35.07, 0, 0, 0),
			provingGroundUnit("sling-radar", "David's Sling Radar", "David's Sling Radar - Dan Package", "ISR", "COALITION_WEST", "davids-sling-radar-section", 32.11, 34.99, 0, 0, 0),
			provingGroundUnit("sling-bmc", "David's Sling BMC", "David's Sling Battle Management - Dan Package", "ISR", "COALITION_WEST", "davids-sling-battle-management-center", 32.08, 34.95, 0, 0, 0),
			provingGroundUnit("davids-sling", "David's Sling", "David's Sling Package Battery", "ISR", "COALITION_WEST", "davids-sling-battery", 32.10, 34.97, 0, 0, 0),
			provingGroundUnit("barak-radar", "Barak Radar", "Barak MX Radar - Dan Package", "ISR", "COALITION_WEST", "barak-mx-radar-section", 32.22, 34.67, 0, 0, 0),
			provingGroundUnit("barak-bmc", "Barak BMC", "Barak MX Battle Management - Dan Package", "ISR", "COALITION_WEST", "barak-mx-battle-management-center", 32.19, 34.70, 0, 0, 0),
			provingGroundUnit("barak-mx", "Barak MX", "Barak MX Package Battery", "ISR", "COALITION_WEST", "barak-mx-battery", 32.20, 34.72, 0, 0, 0),
			provingGroundUnit("iron-dome-dan-radar", "Iron Dome Dan Radar", "Iron Dome Dan Radar - Dan Package", "ISR", "COALITION_WEST", "iron-dome-radar-section", 32.12, 34.85, 0, 0, 0),
			provingGroundUnit("iron-dome-dan-bmc", "Iron Dome Dan BMC", "Iron Dome Dan Battle Management - Dan Package", "ISR", "COALITION_WEST", "iron-dome-battle-management-center", 32.14, 34.83, 0, 0, 0),
			provingGroundUnit("iron-dome-dan", "Iron Dome Dan", "Iron Dome Dan Package Battery", "ISR", "COALITION_WEST", "iron-dome-battery", 32.16, 34.84, 0, 0, 0),
			provingGroundUnit("iron-dome-jerusalem-radar", "Iron Dome Jerusalem Radar", "Iron Dome Jerusalem Radar - Dan Package", "ISR", "COALITION_WEST", "iron-dome-radar-section", 31.79, 35.27, 0, 0, 0),
			provingGroundUnit("iron-dome-jerusalem-bmc", "Iron Dome Jerusalem BMC", "Iron Dome Jerusalem Battle Management - Dan Package", "ISR", "COALITION_WEST", "iron-dome-battle-management-center", 31.81, 35.22, 0, 0, 0),
			provingGroundUnit("iron-dome-jerusalem", "Iron Dome Jerusalem", "Iron Dome Jerusalem Package Battery", "ISR", "COALITION_WEST", "iron-dome-battery", 31.82, 35.24, 0, 0, 0),
		},
	}
}

func packageUSAAlUdeidAirbase() PackageTemplate {
	return PackageTemplate{
		ID:           "pkg-usa-al-udeid-airbase",
		Name:         "USA Al Udeid Airbase",
		Category:     "airbase",
		Description:  "Representative U.S.-operated Al Udeid package with host-nation sovereignty, hosted strike aircraft, airlift, support infrastructure, and local defensive cover.",
		Version:      "1.4.0",
		Priority:     3,
		ReferenceLat: 25.12,
		ReferenceLon: 51.31,
		Accuracy: PackageAccuracyProfile{
			OperationalRole: "U.S.-operated Gulf expeditionary airbase for hosted fighter/strike sortie generation under Qatari sovereignty.",
			Assumptions: []string{
				"The base remains sovereign Qatari territory while presenting as U.S.-operated through operator_team_id.",
				"The package is centered on a representative 379th Air Expeditionary Wing-style mix rather than a literal day-specific ORBAT.",
				"The package includes a small hosted strike element instead of singletons so proving grounds can exercise launch, recovery, reserve aircraft, and re-sortie behavior.",
				"The package includes core command, fuel, munitions, maintenance, tanker, AEW, airlift, and local Patriot defense elements, but still abstracts the full theater support tail.",
				"Qatari sovereign basing remains intact while U.S.-specific assets retain U.S. team ownership.",
				"A host-nation Qatari national AOC is included to represent the joint-base support environment rather than treating the installation as purely American.",
			},
			ValidationGoals: []string{
				"Hosted aircraft launch, recover, replenish, and re-sortie correctly.",
				"Player-facing UI shows the base as U.S.-operated without losing Qatari sovereignty and permissions.",
				"Support assets create a more realistic target set around the airbase than a runway plus two fighters alone.",
				"At least two hosted strike sorties can be generated from the package without inventing ad hoc extra aircraft in the scenario.",
			},
			Sources: []PackageSource{
				{Label: "AFCENT / Al Udeid official releases", URL: "https://www.afcent.af.mil", Kind: "official"},
				{Label: "DVIDS Al Udeid / 379 AEW coverage", URL: "https://www.dvidshub.net", Kind: "official"},
			},
		},
		Units: []*enginev1.Unit{
			func() *enginev1.Unit {
				u := provingGroundUnit("airbase", "Al Udeid", "U.S. Package Airbase - Al Udeid", "QAT", "COALITION_WEST", "qatari-expeditionary-airbase", 25.12, 51.31, 0, 0, 0)
				u.OperatorTeamId = "USA"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("aoc", "Air Operations Center", "Combined Air Operations - Al Udeid Package", "QAT", "COALITION_WEST", "theater-air-operations-center", 25.14, 51.29, 0, 0, 0)
				u.OperatorTeamId = "USA"
				return u
			}(),
			provingGroundUnit("qatari-aoc", "Qatari NAOC", "Qatari National Air Operations Center - Al Udeid Package", "QAT", "COALITION_WEST", "qatari-national-air-operations-center", 25.16, 51.30, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingGroundUnit("fuel", "Fuel Terminal", "Regional Fuel Terminal - Al Udeid Package", "QAT", "COALITION_WEST", "regional-fuel-terminal", 25.10, 51.34, 0, 0, 0)
				u.OperatorTeamId = "USA"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("munitions", "Munitions Area", "Theater Munitions Storage - Al Udeid Package", "QAT", "COALITION_WEST", "theater-munitions-storage-area", 25.11, 51.28, 0, 0, 0)
				u.OperatorTeamId = "USA"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("maintenance", "Maintenance Complex", "Aircraft Maintenance Complex - Al Udeid Package", "QAT", "COALITION_WEST", "expeditionary-aircraft-maintenance-complex", 25.13, 51.33, 0, 0, 0)
				u.OperatorTeamId = "USA"
				return u
			}(),
			provingGroundUnit("patriot", "Patriot Battery", "USAF/US Army Patriot - Al Udeid Package", "USA", "COALITION_WEST", "patriot-pac3-battery", 25.17, 51.26, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingAircraft("e3", "E-3G", "USAF E-3G Package AEW Aircraft", "USA", "COALITION_WEST", "e3g-sentry", 25.12, 51.31, 0, 0, 0)
				u.HostBaseId = "airbase"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("kc46", "KC-46A", "USAF KC-46A Package Tanker", "USA", "COALITION_WEST", "kc46a-pegasus", 25.12, 51.31, 0, 0, 0)
				u.HostBaseId = "airbase"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("c17", "C-17A", "USAF C-17A Package Airlift Aircraft", "USA", "COALITION_WEST", "c17a-globemaster-iii", 25.12, 51.31, 0, 0, 0)
				u.HostBaseId = "airbase"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f15e-lead", "F-15E Lead", "USAF F-15E Lead Package Strike Aircraft", "USA", "COALITION_WEST", "f15e-strike-eagle", 25.12, 51.31, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "deep_strike"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f15e-wing", "F-15E Wing", "USAF F-15E Wing Package Strike Aircraft", "USA", "COALITION_WEST", "f15e-strike-eagle", 25.12, 51.31, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "deep_strike"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f35a-lead", "F-35A Lead", "USAF F-35A Lead Package Stealth Aircraft", "USA", "COALITION_WEST", "f35a-lightning", 25.12, 51.31, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "internal_strike"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f35a-wing", "F-35A Wing", "USAF F-35A Wing Package Stealth Aircraft", "USA", "COALITION_WEST", "f35a-lightning", 25.12, 51.31, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "internal_strike"
				return u
			}(),
		},
	}
}

func packageISRCounterstrikeCell() PackageTemplate {
	return PackageTemplate{
		ID:           "pkg-isr-counterstrike-cell",
		Name:         "ISR Counterstrike Cell",
		Category:     "strike",
		Description:  "Representative Israeli standoff counterstrike cell with paired strike and SEAD aircraft, tanker, and AEW support based from a strategic airbase.",
		Version:      "1.1.0",
		Priority:     3,
		ReferenceLat: 31.89,
		ReferenceLon: 34.69,
		Accuracy: PackageAccuracyProfile{
			OperationalRole: "Israeli standoff strike and suppression package for regional counterforce missions against missile, air-defense, and strategic fixed targets.",
			Assumptions: []string{
				"The package represents a compact operational strike cell built around two-ship strike and SEAD elements rather than a full wing-sized air order of battle.",
				"It includes one tanker and one AEW aircraft as representative enabling support for long-range counterstrike missions.",
				"The package is base-hosted by default and can be pushed airborne in proving grounds or scenarios that need an already-launched posture.",
			},
			ValidationGoals: []string{
				"Support a realistic Israeli suppression and counterstrike slice against defended regional targets.",
				"Provide a reusable package for proving grounds that need real Israeli strike aircraft rather than borrowed coalition stand-ins.",
			},
			Sources: []PackageSource{
				{Label: "Israeli Air Force public capability releases", URL: "https://www.iaf.org.il", Kind: "official"},
				{Label: "Israeli MOD and open-source order-of-battle reporting", URL: "https://www.mod.gov.il", Kind: "official"},
			},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("airbase", "Palmachim AB", "Israeli Counterstrike Package Host Base", "ISR", "COALITION_WEST", "israel-strategic-airbase", 31.89, 34.69, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingAircraft("eitam", "Eitam", "IAF Eitam Package AEW Aircraft", "ISR", "COALITION_WEST", "g550-eitam", 31.89, 34.69, 0, 0, 0)
				u.HostBaseId = "airbase"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("reem", "Re'em", "IAF Re'em Package Tanker", "ISR", "COALITION_WEST", "boeing707-reem", 31.89, 34.69, 0, 0, 0)
				u.HostBaseId = "airbase"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f15i-lead", "F-15I Lead", "IAF F-15I Lead Package Strike Aircraft", "ISR", "COALITION_WEST", "f15i-raam", 31.89, 34.69, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "deep_strike"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f15i-wing", "F-15I Wing", "IAF F-15I Wing Package Strike Aircraft", "ISR", "COALITION_WEST", "f15i-raam", 31.89, 34.69, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "deep_strike"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f16i-lead", "F-16I Lead", "IAF F-16I Lead Package SEAD Aircraft", "ISR", "COALITION_WEST", "f16i-sufa", 31.89, 34.69, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "sead"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f16i-wing", "F-16I Wing", "IAF F-16I Wing Package SEAD Aircraft", "ISR", "COALITION_WEST", "f16i-sufa", 31.89, 34.69, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "sead"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f35i-lead", "F-35I Lead", "IAF F-35I Lead Package Penetrating Strike Aircraft", "ISR", "COALITION_WEST", "f35i-adir", 31.89, 34.69, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "internal_strike"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f35i-wing", "F-35I Wing", "IAF F-35I Wing Package Penetrating Strike Aircraft", "ISR", "COALITION_WEST", "f35i-adir", 31.89, 34.69, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "internal_strike"
				return u
			}(),
		},
	}
}

func packageAREAlDhafraAirbase() PackageTemplate {
	return PackageTemplate{
		ID:           "pkg-are-al-dhafra-airbase",
		Name:         "ARE Al Dhafra Airbase",
		Category:     "airbase",
		Description:  "Representative Al Dhafra package with sovereign Emirati infrastructure, U.S.-operated hosted aircraft, command support, and local missile defense.",
		Version:      "1.0.0",
		Priority:     4,
		ReferenceLat: 24.25,
		ReferenceLon: 54.55,
		Accuracy: PackageAccuracyProfile{
			OperationalRole: "Joint UAE-U.S. expeditionary airbase for Gulf counterair, strike support, and regional command integration.",
			Assumptions: []string{
				"The base remains sovereign Emirati territory while presenting as U.S.-operated through operator_team_id on selected coalition infrastructure.",
				"The package is built as a compact but operationally meaningful Gulf support node rather than a literal day-specific ramp census.",
				"It emphasizes counterair and penetrating-strike utility with one F-22 pair, one F-35 pair, tanker support, AEW support, Patriot defense, and host-nation air-defense command.",
				"Hosted aircraft are U.S.-owned while the airbase and host-nation ADOC remain Emirati.",
			},
			ValidationGoals: []string{
				"Support Gulf counterair proving grounds with a realistic sovereign/operator split.",
				"Provide a reusable UAE-based package for composed regional posture scenarios.",
				"Keep enough hosted capability to validate launch, recovery, and coordinated support behavior without building a full wing-sized package.",
			},
			Sources: []PackageSource{
				{Label: "U.S. Air Force Al Dhafra public releases", URL: "https://www.afcent.af.mil", Kind: "official"},
				{Label: "UAE MOD / national defense releases", URL: "https://www.mod.gov.ae", Kind: "official"},
			},
		},
		Units: []*enginev1.Unit{
			func() *enginev1.Unit {
				u := provingGroundUnit("airbase", "Al Dhafra", "U.S. Package Airbase - Al Dhafra", "ARE", "COALITION_WEST", "emirati-strategic-airbase", 24.25, 54.55, 0, 0, 0)
				u.OperatorTeamId = "USA"
				return u
			}(),
			provingGroundUnit("adoc", "Emirati ADOC", "Emirati Air Defense Operations Center - Al Dhafra Package", "ARE", "COALITION_WEST", "emirati-air-defense-operations-center", 24.28, 54.58, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingGroundUnit("aoc", "Coalition AOC", "Coalition Air Operations Center - Al Dhafra Package", "ARE", "COALITION_WEST", "theater-air-operations-center", 24.23, 54.52, 0, 0, 0)
				u.OperatorTeamId = "USA"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("fuel", "Fuel Terminal", "Regional Fuel Terminal - Al Dhafra Package", "ARE", "COALITION_WEST", "regional-fuel-terminal", 24.21, 54.56, 0, 0, 0)
				u.OperatorTeamId = "USA"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("maintenance", "Maintenance Complex", "Aircraft Maintenance Complex - Al Dhafra Package", "ARE", "COALITION_WEST", "expeditionary-aircraft-maintenance-complex", 24.24, 54.50, 0, 0, 0)
				u.OperatorTeamId = "USA"
				return u
			}(),
			provingGroundUnit("patriot", "Patriot Battery", "Patriot Battery - Al Dhafra Package", "USA", "COALITION_WEST", "patriot-pac3-battery", 24.31, 54.46, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingAircraft("e3", "E-3G", "USAF E-3G Package AEW Aircraft - Al Dhafra", "USA", "COALITION_WEST", "e3g-sentry", 24.25, 54.55, 0, 0, 0)
				u.HostBaseId = "airbase"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("kc46", "KC-46A", "USAF KC-46A Package Tanker - Al Dhafra", "USA", "COALITION_WEST", "kc46a-pegasus", 24.25, 54.55, 0, 0, 0)
				u.HostBaseId = "airbase"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f22-lead", "F-22 Lead", "USAF F-22A Lead Package Fighter - Al Dhafra", "USA", "COALITION_WEST", "f22a-raptor", 24.25, 54.55, 0, 0, 0)
				u.HostBaseId = "airbase"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f22-wing", "F-22 Wing", "USAF F-22A Wing Package Fighter - Al Dhafra", "USA", "COALITION_WEST", "f22a-raptor", 24.25, 54.55, 0, 0, 0)
				u.HostBaseId = "airbase"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f35a-lead", "F-35A Lead", "USAF F-35A Lead Package Aircraft - Al Dhafra", "USA", "COALITION_WEST", "f35a-lightning", 24.25, 54.55, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "internal_strike"
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("f35a-wing", "F-35A Wing", "USAF F-35A Wing Package Aircraft - Al Dhafra", "USA", "COALITION_WEST", "f35a-lightning", 24.25, 54.55, 0, 0, 0)
				u.HostBaseId = "airbase"
				u.LoadoutConfigurationId = "internal_strike"
				return u
			}(),
		},
	}
}

func packageAREAbuDhabiCoastalDefense() PackageTemplate {
	return PackageTemplate{
		ID:           "pkg-are-abu-dhabi-coastal-defense",
		Name:         "ARE Abu Dhabi Coastal Defense",
		Category:     "coastal_defense",
		Description:  "Representative Emirati coastal-defense and offshore-infrastructure package for the western Gulf, pairing national command nodes with a Baynunah corvette and local missile defense.",
		Version:      "1.1.0",
		Priority:     5,
		ReferenceLat: 24.52,
		ReferenceLon: 54.37,
		Accuracy: PackageAccuracyProfile{
			OperationalRole: "Emirati western-Gulf coastal defense for offshore infrastructure protection, littoral patrol, and local missile-defense support.",
			Assumptions: []string{
				"The package represents an Abu Dhabi-area coastal-defense posture rather than a single named base complex.",
				"It centers on a two-corvette Baynunah local sea-defense slice, national command and missile-defense support, and nearby critical infrastructure to produce a realistic defended coastal target set.",
				"Patriot is included as a representative high-end aligned defensive layer alongside Emirati command infrastructure.",
			},
			ValidationGoals: []string{
				"Provide a reusable native Emirati maritime/coastal package for larger Gulf compositions.",
				"Create a western-Gulf counterpart to the Iranian Hormuz coastal-denial package.",
				"Support littoral proving grounds without relying only on U.S. and Bahraini surface presence.",
			},
			Sources: []PackageSource{
				{Label: "UAE MOD / national defense releases", URL: "https://www.mod.gov.ae", Kind: "official"},
				{Label: "EDGE / Baynunah program reporting", URL: "https://www.edgegroup.ae", Kind: "official"},
			},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("adoc", "Emirati ADOC", "Abu Dhabi Coastal Defense ADOC", "ARE", "COALITION_WEST", "emirati-air-defense-operations-center", 24.48, 54.39, 0, 0, 0),
			provingGroundUnit("export-terminal", "Export Terminal", "Abu Dhabi Coastal Defense Export Terminal", "ARE", "COALITION_WEST", "emirati-export-terminal", 24.55, 54.44, 0, 0, 0),
			provingGroundUnit("desalination", "Desalination Complex", "Abu Dhabi Coastal Defense Desalination Complex", "ARE", "COALITION_WEST", "emirati-desalination-complex", 24.50, 54.28, 0, 0, 0),
			provingGroundUnit("patriot", "Patriot Battery", "Abu Dhabi Coastal Defense Patriot", "ARE", "COALITION_WEST", "patriot-pac3-battery", 24.60, 54.31, 0, 0, 0),
			provingGroundUnit("baynunah", "Baynunah", "Abu Dhabi Coastal Defense Corvette", "ARE", "COALITION_WEST", "baynunah-corvette-uae", 24.62, 54.47, 0, 60, 0),
			provingGroundUnit("baynunah-wing", "Baynunah Wing", "Abu Dhabi Coastal Defense Corvette Wing", "ARE", "COALITION_WEST", "baynunah-corvette-uae", 24.66, 54.51, 0, 60, 0),
		},
	}
}

func packageOMNMusandamStraitGuard() PackageTemplate {
	return PackageTemplate{
		ID:           "pkg-omn-musandam-strait-guard",
		Name:         "OMN Musandam Strait Guard",
		Category:     "coastal_defense",
		Description:  "Representative Omani Musandam package for southern Strait of Hormuz surveillance, escort, and local sea control.",
		Version:      "1.0.0",
		Priority:     6,
		ReferenceLat: 26.20,
		ReferenceLon: 56.30,
		Accuracy: PackageAccuracyProfile{
			OperationalRole: "Omani southern-Hormuz surveillance and local escort package for the Musandam / Gulf of Oman approaches.",
			Assumptions: []string{
				"The package represents a reusable Musandam-area maritime-security posture rather than a literal order of battle snapshot at one pier.",
				"It combines a coastal radar and maritime support node with one Khareef corvette and one Musandam OPV as the core local surface-defense element.",
				"The package is intentionally national and modest; it is meant to complement UAE, Bahraini, and U.S. Gulf packages rather than replace them.",
			},
			ValidationGoals: []string{
				"Provide a reusable southern-Hormuz package for Gulf composition scenarios.",
				"Support routing, escort, and littoral contest scenarios in the Strait of Hormuz and Gulf of Oman.",
				"Represent Omani local maritime presence without relying on coalition-only assets.",
			},
			Sources: []PackageSource{
				{Label: "Royal Navy of Oman public reporting", URL: "https://mod.gov.om", Kind: "official"},
				{Label: "Oman Ministry of Defence public reporting", URL: "https://mod.gov.om", Kind: "official"},
			},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("airbase", "Musandam Air Base", "Musandam Strait Guard Maritime Air Base", "OMN", "COALITION_WEST", "omani-maritime-airbase", 26.18, 56.24, 0, 0, 0),
			provingGroundUnit("port", "Khasab Port", "Musandam Strait Guard Logistics Port", "OMN", "COALITION_WEST", "omani-logistics-port", 26.20, 56.25, 0, 0, 0),
			provingGroundUnit("radar", "Musandam Coastal Radar", "Musandam Strait Guard Coastal Radar", "OMN", "COALITION_WEST", "omani-coastal-radar-site", 26.23, 56.28, 0, 0, 0),
			provingGroundUnit("khareef", "Khareef", "Musandam Strait Guard Corvette", "OMN", "COALITION_WEST", "khareef-corvette-oman", 26.24, 56.34, 0, 55, 0),
			provingGroundUnit("musandam", "Musandam", "Musandam Strait Guard OPV", "OMN", "COALITION_WEST", "musandam-opv-oman", 26.15, 56.38, 0, 55, 0),
		},
	}
}

func packageBHRNavalSupportBahrain() PackageTemplate {
	return PackageTemplate{
		ID:           "pkg-bhr-naval-support-bahrain",
		Name:         "Bahrain Naval Support Package",
		Category:     "naval_base",
		Description:  "Representative Bahrain naval support package with sovereign host-nation infrastructure, Bahraini surface combatants, and a small U.S. littoral presence.",
		Version:      "1.0.0",
		Priority:     4,
		ReferenceLat: 26.23,
		ReferenceLon: 50.61,
		Accuracy: PackageAccuracyProfile{
			OperationalRole: "Joint Gulf naval support node for maritime security, escort, patrol, and coalition presence around Bahrain and the central Gulf.",
			Assumptions: []string{
				"The package represents Naval Support Activity Bahrain and adjacent host-nation naval infrastructure as a reusable operational default, not a literal pier assignment snapshot.",
				"The base remains sovereign Bahraini territory while presenting as U.S.-supported through operator_team_id on selected infrastructure.",
				"The surface element is intentionally modest: one U.S. littoral combatant plus representative Bahraini frigate and corvette coverage for local escort and patrol tasks.",
			},
			ValidationGoals: []string{
				"Support Gulf maritime presence scenarios without needing to hand-build Bahrain every time.",
				"Provide a reusable littoral-defense and escort package for routing, movement, and small maritime engagement proving grounds.",
				"Preserve Bahraini sovereignty while still surfacing U.S. operational presence in the UI.",
			},
			Sources: []PackageSource{
				{Label: "NAVCENT / NSA Bahrain official releases", URL: "https://www.cusnc.navy.mil", Kind: "official"},
				{Label: "Bahrain Defence Force public reporting", URL: "https://www.bdf.bh", Kind: "official"},
			},
		},
		Units: []*enginev1.Unit{
			func() *enginev1.Unit {
				u := provingGroundUnit("naval-base", "NSA Bahrain", "Bahrain Naval Support Package Base", "BHR", "COALITION_WEST", "bahraini-naval-support-base", 26.23, 50.61, 0, 0, 0)
				u.OperatorTeamId = "USA"
				return u
			}(),
			provingGroundUnit("refinery", "Sitra Refinery", "Bahrain Naval Support Package Refinery", "BHR", "COALITION_WEST", "bahraini-refinery-terminal", 26.15, 50.64, 0, 0, 0),
			provingGroundUnit("utility", "Power and Water Plant", "Bahrain Naval Support Package Utility Plant", "BHR", "COALITION_WEST", "bahraini-power-water-plant", 26.19, 50.56, 0, 0, 0),
			provingGroundUnit("airbase", "Bahrain Air Base", "Bahrain Naval Support Package Air Base", "BHR", "COALITION_WEST", "bahraini-airbase", 26.27, 50.64, 0, 0, 0),
			provingGroundUnit("frigate", "RBNS Sabha", "Bahraini Frigate - Naval Support Package", "BHR", "COALITION_WEST", "rbns-sabha-frigate-bahrain", 26.21, 50.58, 0, 90, 0),
			provingGroundUnit("corvette", "Muray Jib", "Bahraini Corvette - Naval Support Package", "BHR", "COALITION_WEST", "muray-jib-corvette-bahrain", 26.19, 50.60, 0, 90, 0),
			provingGroundUnit("lcs", "USS Freedom", "U.S. Littoral Combat Ship - Bahrain Package", "USA", "COALITION_WEST", "freedom-lcs", 26.17, 50.57, 0, 90, 0),
			provingGroundUnit("patrol", "Cyclone Patrol Ship", "U.S. Patrol Ship - Bahrain Package", "USA", "COALITION_WEST", "cyclone-patrol-ship", 26.18, 50.55, 0, 90, 0),
		},
	}
}

func packageIRNHormuzCoastalDenial() PackageTemplate {
	return PackageTemplate{
		ID:           "pkg-irn-hormuz-coastal-denial",
		Name:         "IRN Hormuz Coastal Denial",
		Category:     "coastal_denial",
		Description:  "Representative eastern Gulf and Strait of Hormuz coastal-denial package built around coastal anti-ship fires, local air defense, surface swarm elements, and a midget submarine.",
		Version:      "1.0.0",
		Priority:     5,
		ReferenceLat: 27.03,
		ReferenceLon: 56.24,
		Accuracy: PackageAccuracyProfile{
			OperationalRole: "Iranian littoral sea-denial and chokepoint pressure package for Bandar Abbas and the Strait of Hormuz.",
			Assumptions: []string{
				"The package is a representative Hormuz-area coastal-denial bundle rather than a literal single-base ORBAT.",
				"It combines coastal anti-ship missile fires, local and area air defense, swarm-command surface combatants, and a small submarine threat.",
				"It is optimized for Gulf chokepoint and near-shore pressure, not blue-water fleet action.",
			},
			ValidationGoals: []string{
				"Threaten a mixed Gulf transit group with multiple domains rather than a single raider.",
				"Provide a reusable eastern-Gulf adversary package for larger Gulf compositions.",
				"Force western packages to handle both shore-based and afloat littoral threats.",
			},
			Sources: []PackageSource{
				{Label: "IRGCN / Iranian coastal-defense open reporting", URL: "https://www.csis.org", Kind: "reputable_analysis"},
				{Label: "Iran Watch regional maritime and missile reporting", URL: "https://www.iranwatch.org", Kind: "reputable_analysis"},
			},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("oil-terminal", "Bandar Abbas Export Terminal", "Hormuz Coastal Denial Oil Export Terminal", "IRN", "COALITION_IRAN", "iran-gulf-oil-export-terminal", 27.11, 56.23, 0, 0, 0),
			provingGroundUnit("missile-complex", "Underground Missile Complex", "Hormuz Coastal Denial Missile Complex", "IRN", "COALITION_IRAN", "iran-underground-missile-complex", 27.28, 56.12, 0, 0, 0),
			provingGroundUnit("coastal-battery", "Raad Coastal Battery", "Hormuz Coastal Denial Raad Battery", "IRN", "COALITION_IRAN", "raad-coastal-battery", 26.97, 56.18, 0, 140, 0),
			provingGroundUnit("sam", "3rd Khordad", "Hormuz Coastal Denial Local Air Defense", "IRN", "COALITION_IRAN", "third-khordad-battery", 27.05, 56.31, 0, 180, 0),
			provingGroundUnit("area-sam", "Bavar-373", "Hormuz Coastal Denial Area Air Defense", "IRN", "COALITION_IRAN", "bavar373-battery", 27.22, 56.03, 0, 180, 0),
			provingGroundUnit("swarm", "IRGCN Swarm", "Hormuz Coastal Denial Swarm Group", "IRN", "COALITION_IRAN", "irgcn-swarm-group", 26.86, 56.34, 0, 250, 0),
			provingGroundUnit("soleimani", "Shahid Soleimani", "Hormuz Coastal Denial Corvette", "IRN", "COALITION_IRAN", "shahid-soleimani-corvette", 26.92, 56.46, 0, 245, 0),
			provingGroundUnit("ghadir-sub", "Ghadir Sub", "Hormuz Coastal Denial Midget Submarine", "IRN", "COALITION_IRAN", "ghadir-midget-submarine", 26.78, 56.28, -25, 240, 0),
		},
	}
}

func packageIRNWesternMissileRegiment() PackageTemplate {
	return PackageTemplate{
		ID:           "pkg-irn-western-missile-regiment",
		Name:         "IRN Western Missile Regiment",
		Category:     "missile_force",
		Description:  "Representative western Iranian launcher bundle for regional ballistic and cruise-missile strike operations with local support, deception, and point air defense.",
		Version:      "1.5.0",
		Priority:     4,
		ReferenceLat: 34.25,
		ReferenceLon: 45.55,
		Accuracy: PackageAccuracyProfile{
			OperationalRole: "Western Iranian land-attack missile package for regional fixed-target strike into Israel and the Gulf.",
			Assumptions: []string{
				"The package is a representative strike regiment abstraction, not a literal named brigade disposition.",
				"Kheibar Shekan provides the heavier ballistic-strike component while Paveh represents the long-range cruise-missile component.",
				"The package now includes representative command, reload, deception, and local SAM support, but still abstracts deeper tunnel-storage and national-level cueing infrastructure.",
				"Launch elements are intentionally dispersed across western Iran rather than parked in one compact cluster, so the package behaves more like a survivable operating area than a parade lineup.",
				"The geometry separates launchers, command, reload, and deception elements across multiple valleys and road axes to avoid turning the package into a single aimpoint.",
			},
			ValidationGoals: []string{
				"Generate enough projectile volume to stress regional missile defenses.",
				"Remain vulnerable to counterstrike once located.",
				"Present a more realistic target set than launchers alone, including local air defense and sustainment nodes.",
			},
			Sources: []PackageSource{
				{Label: "Iran Watch missile force background", URL: "https://www.iranwatch.org", Kind: "reputable_analysis"},
				{Label: "CSIS Missile Threat / regional analysis", URL: "https://www.csis.org", Kind: "reputable_analysis"},
			},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("command", "Missile Brigade C2", "Western Iran Missile Package Command Post", "IRN", "COALITION_IRAN", "missile-brigade-command-post", 34.74, 46.18, 0, 180, 0),
			provingGroundUnit("support", "Missile Support Column", "Western Iran Missile Package Support Column", "IRN", "COALITION_IRAN", "missile-support-column", 33.76, 44.92, 0, 180, 0),
			provingGroundUnit("reload", "Reload Transporter", "Western Iran Missile Package Reload Transporter", "IRN", "COALITION_IRAN", "missile-reload-transporter", 34.62, 44.64, 0, 180, 0),
			provingGroundUnit("decoy", "Decoy Detachment", "Western Iran Missile Package Decoy Detachment", "IRN", "COALITION_IRAN", "missile-decoy-detachment", 33.96, 46.56, 0, 180, 0),
			provingGroundUnit("sam", "Khordad-15", "Western Iran Missile Package Local Air Defense", "IRN", "COALITION_IRAN", "khordad15-battery", 34.03, 45.68, 0, 180, 0),
			provingGroundUnit("area-sam", "Bavar-373", "Western Iran Missile Package Area Air Defense", "IRN", "COALITION_IRAN", "bavar373-battery", 34.86, 45.14, 0, 180, 0),
			func() *enginev1.Unit {
				u := provingGroundUnit("kheibar-1", "Kheibar Brigade 1", "Western Iran Kheibar Package Brigade 1", "IRN", "COALITION_IRAN", "kheibar-shekan-brigade", 33.72, 45.11, 0, 240, 0)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "ssm-kheibar-shekan", CurrentQty: 8, MaxQty: 8}}
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("kheibar-2", "Kheibar Brigade 2", "Western Iran Kheibar Package Brigade 2", "IRN", "COALITION_IRAN", "kheibar-shekan-brigade", 35.01, 45.82, 0, 240, 0)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "ssm-kheibar-shekan", CurrentQty: 8, MaxQty: 8}}
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("paveh-1", "Paveh Regiment 1", "Western Iran Paveh Package Regiment 1", "IRN", "COALITION_IRAN", "paveh-cruise-missile-regiment", 34.49, 44.56, 0, 240, 0)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "ssm-paveh", CurrentQty: 8, MaxQty: 8}}
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("paveh-2", "Paveh Regiment 2", "Western Iran Paveh Package Regiment 2", "IRN", "COALITION_IRAN", "paveh-cruise-missile-regiment", 34.19, 46.42, 0, 240, 0)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "ssm-paveh", CurrentQty: 8, MaxQty: 8}}
				return u
			}(),
		},
	}
}

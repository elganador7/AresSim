package scenario

import (
	"math"
	"testing"

	"github.com/aressim/internal/library"
)

func TestPackageTemplatesValidate(t *testing.T) {
	for id, template := range PackageTemplates() {
		if err := ValidatePackageTemplate(template); err != nil {
			t.Fatalf("ValidatePackageTemplate(%s) failed: %v", id, err)
		}
	}
}

func TestInstantiatePackage_RemapsIDsAndHostBase(t *testing.T) {
	template, ok := PackageTemplateByID("pkg-usa-al-udeid-airbase")
	if !ok {
		t.Fatalf("package template not found")
	}
	units, err := InstantiatePackage(template, PackageInstanceOptions{
		Prefix:    "test-au",
		AnchorLat: 26.00,
		AnchorLon: 52.00,
	})
	if err != nil {
		t.Fatalf("InstantiatePackage failed: %v", err)
	}
	if len(units) != 14 {
		t.Fatalf("expected 14 units, got %d", len(units))
	}
	var airbaseFound bool
	hostedFound := map[string]bool{}
	supportFound := map[string]bool{}
	for _, unit := range units {
		switch unit.GetId() {
		case "test-au-f15e-lead", "test-au-f15e-wing", "test-au-f35a-lead", "test-au-f35a-wing", "test-au-c17", "test-au-e3", "test-au-kc46":
			hostedFound[unit.GetId()] = true
			if unit.GetHostBaseId() != "test-au-airbase" {
				t.Fatalf("expected hosted aircraft %s to remap host base, got %s", unit.GetId(), unit.GetHostBaseId())
			}
		}
		switch unit.GetId() {
		case "test-au-airbase":
			airbaseFound = true
			if unit.GetTeamId() != "QAT" || unit.GetOperatorTeamId() != "USA" {
				t.Fatalf("expected sovereign QAT / operator USA for Al Udeid, got team=%s operator=%s", unit.GetTeamId(), unit.GetOperatorTeamId())
			}
		case "test-au-aoc", "test-au-qatari-aoc", "test-au-fuel", "test-au-munitions", "test-au-maintenance", "test-au-patriot", "test-au-c17", "test-au-e3", "test-au-kc46":
			supportFound[unit.GetId()] = true
		}
	}
	if !airbaseFound || !hostedFound["test-au-f15e-lead"] || !hostedFound["test-au-f35a-lead"] || !hostedFound["test-au-c17"] || !supportFound["test-au-aoc"] || !supportFound["test-au-qatari-aoc"] || !supportFound["test-au-patriot"] {
		t.Fatalf("expected remapped package ids to be present")
	}
}

func TestInstantiatePackage_BahrainNavalSupportPreservesSovereigntyAndPresence(t *testing.T) {
	template, ok := PackageTemplateByID("pkg-bhr-naval-support-bahrain")
	if !ok {
		t.Fatalf("package template not found")
	}
	units, err := InstantiatePackage(template, PackageInstanceOptions{
		Prefix:    "test-bh",
		AnchorLat: 26.30,
		AnchorLon: 50.70,
	})
	if err != nil {
		t.Fatalf("InstantiatePackage failed: %v", err)
	}
	if len(units) != 8 {
		t.Fatalf("expected 8 units, got %d", len(units))
	}
	found := map[string]bool{}
	for _, unit := range units {
		switch unit.GetId() {
		case "test-bh-naval-base":
			found[unit.GetId()] = true
			if unit.GetTeamId() != "BHR" || unit.GetOperatorTeamId() != "USA" {
				t.Fatalf("expected Bahraini sovereignty and U.S. operator on naval base, got team=%s operator=%s", unit.GetTeamId(), unit.GetOperatorTeamId())
			}
		case "test-bh-frigate", "test-bh-corvette":
			found[unit.GetId()] = true
			if unit.GetTeamId() != "BHR" {
				t.Fatalf("expected Bahraini combatant team, got %s", unit.GetTeamId())
			}
		case "test-bh-lcs", "test-bh-patrol":
			found[unit.GetId()] = true
			if unit.GetTeamId() != "USA" {
				t.Fatalf("expected U.S. maritime presence team, got %s", unit.GetTeamId())
			}
		}
	}
	for _, id := range []string{"test-bh-naval-base", "test-bh-frigate", "test-bh-corvette", "test-bh-lcs", "test-bh-patrol"} {
		if !found[id] {
			t.Fatalf("expected package unit %s", id)
		}
	}
}

func TestInstantiatePackage_AlDhafraPreservesSovereigntyAndCounterairPresence(t *testing.T) {
	template, ok := PackageTemplateByID("pkg-are-al-dhafra-airbase")
	if !ok {
		t.Fatalf("package template not found")
	}
	units, err := InstantiatePackage(template, PackageInstanceOptions{
		Prefix:    "test-ad",
		AnchorLat: 24.40,
		AnchorLon: 54.70,
	})
	if err != nil {
		t.Fatalf("InstantiatePackage failed: %v", err)
	}
	if len(units) != 12 {
		t.Fatalf("expected 12 units, got %d", len(units))
	}
	found := map[string]bool{}
	for _, unit := range units {
		switch unit.GetId() {
		case "test-ad-airbase":
			found[unit.GetId()] = true
			if unit.GetTeamId() != "ARE" || unit.GetOperatorTeamId() != "USA" {
				t.Fatalf("expected Emirati sovereignty and U.S. operator on Al Dhafra, got team=%s operator=%s", unit.GetTeamId(), unit.GetOperatorTeamId())
			}
		case "test-ad-adoc":
			found[unit.GetId()] = true
			if unit.GetTeamId() != "ARE" {
				t.Fatalf("expected Emirati ADOC team, got %s", unit.GetTeamId())
			}
		case "test-ad-f22-lead", "test-ad-f22-wing", "test-ad-f35a-lead", "test-ad-f35a-wing", "test-ad-e3", "test-ad-kc46":
			found[unit.GetId()] = true
			if unit.GetHostBaseId() != "test-ad-airbase" {
				t.Fatalf("expected hosted air unit %s to remap host base, got %s", unit.GetId(), unit.GetHostBaseId())
			}
		}
	}
	for _, id := range []string{"test-ad-airbase", "test-ad-adoc", "test-ad-f22-lead", "test-ad-f22-wing", "test-ad-f35a-lead", "test-ad-f35a-wing"} {
		if !found[id] {
			t.Fatalf("expected package unit %s", id)
		}
	}
}

func TestInstantiatePackage_HormuzCoastalDenialIncludesLittoralLayers(t *testing.T) {
	template, ok := PackageTemplateByID("pkg-irn-hormuz-coastal-denial")
	if !ok {
		t.Fatalf("package template not found")
	}
	units, err := InstantiatePackage(template, PackageInstanceOptions{
		Prefix:    "test-hcd",
		AnchorLat: 27.20,
		AnchorLon: 56.40,
	})
	if err != nil {
		t.Fatalf("InstantiatePackage failed: %v", err)
	}
	if len(units) != 8 {
		t.Fatalf("expected 8 units, got %d", len(units))
	}
	found := map[string]bool{}
	for _, unit := range units {
		switch unit.GetId() {
		case "test-hcd-coastal-battery", "test-hcd-sam", "test-hcd-area-sam", "test-hcd-swarm", "test-hcd-soleimani", "test-hcd-ghadir-sub":
			found[unit.GetId()] = true
			if unit.GetTeamId() != "IRN" {
				t.Fatalf("expected Iranian coastal-denial unit team, got %s", unit.GetTeamId())
			}
		}
	}
	for _, id := range []string{"test-hcd-coastal-battery", "test-hcd-sam", "test-hcd-area-sam", "test-hcd-swarm", "test-hcd-soleimani", "test-hcd-ghadir-sub"} {
		if !found[id] {
			t.Fatalf("expected package unit %s", id)
		}
	}
}

func TestInstantiatePackage_UAECoastalDefenseIncludesNativeLittoralCore(t *testing.T) {
	template, ok := PackageTemplateByID("pkg-are-abu-dhabi-coastal-defense")
	if !ok {
		t.Fatalf("package template not found")
	}
	units, err := InstantiatePackage(template, PackageInstanceOptions{
		Prefix:    "test-uacd",
		AnchorLat: 24.60,
		AnchorLon: 54.45,
	})
	if err != nil {
		t.Fatalf("InstantiatePackage failed: %v", err)
	}
	if len(units) != 6 {
		t.Fatalf("expected 6 units, got %d", len(units))
	}
	found := map[string]bool{}
	for _, unit := range units {
		switch unit.GetId() {
		case "test-uacd-adoc", "test-uacd-export-terminal", "test-uacd-desalination", "test-uacd-patriot", "test-uacd-baynunah", "test-uacd-baynunah-wing":
			found[unit.GetId()] = true
			if unit.GetTeamId() != "ARE" {
				t.Fatalf("expected Emirati coastal-defense team, got %s", unit.GetTeamId())
			}
		}
	}
	for _, id := range []string{"test-uacd-adoc", "test-uacd-export-terminal", "test-uacd-desalination", "test-uacd-patriot", "test-uacd-baynunah", "test-uacd-baynunah-wing"} {
		if !found[id] {
			t.Fatalf("expected package unit %s", id)
		}
	}
}

func TestInstantiatePackage_OMNMusandamStraitGuardIncludesSouthernHormuzCore(t *testing.T) {
	template, ok := PackageTemplateByID("pkg-omn-musandam-strait-guard")
	if !ok {
		t.Fatalf("package template not found")
	}
	units, err := InstantiatePackage(template, PackageInstanceOptions{
		Prefix:    "test-omsg",
		AnchorLat: 26.20,
		AnchorLon: 56.30,
	})
	if err != nil {
		t.Fatalf("InstantiatePackage failed: %v", err)
	}
	if len(units) != 5 {
		t.Fatalf("expected 5 units, got %d", len(units))
	}
	found := map[string]bool{}
	for _, unit := range units {
		switch unit.GetId() {
		case "test-omsg-airbase", "test-omsg-port", "test-omsg-radar", "test-omsg-khareef", "test-omsg-musandam":
			found[unit.GetId()] = true
			if unit.GetTeamId() != "OMN" {
				t.Fatalf("expected Omani strait-guard team, got %s", unit.GetTeamId())
			}
		}
	}
	for _, id := range []string{"test-omsg-airbase", "test-omsg-port", "test-omsg-radar", "test-omsg-khareef", "test-omsg-musandam"} {
		if !found[id] {
			t.Fatalf("expected package unit %s", id)
		}
	}
}

func TestInstantiatePackage_ISRCounterstrikeCellHostsAircraft(t *testing.T) {
	template, ok := PackageTemplateByID("pkg-isr-counterstrike-cell")
	if !ok {
		t.Fatalf("package template not found")
	}
	units, err := InstantiatePackage(template, PackageInstanceOptions{
		Prefix:    "test-isr",
		AnchorLat: 32.10,
		AnchorLon: 34.95,
	})
	if err != nil {
		t.Fatalf("InstantiatePackage failed: %v", err)
	}
	if len(units) != 9 {
		t.Fatalf("expected 9 units, got %d", len(units))
	}
	hosted := map[string]bool{}
	for _, unit := range units {
		switch unit.GetId() {
		case "test-isr-f15i-lead", "test-isr-f15i-wing", "test-isr-f16i-lead", "test-isr-f16i-wing", "test-isr-f35i-lead", "test-isr-f35i-wing", "test-isr-eitam", "test-isr-reem":
			if unit.GetHostBaseId() != "test-isr-airbase" {
				t.Fatalf("expected hosted unit %s to remap host base, got %s", unit.GetId(), unit.GetHostBaseId())
			}
			hosted[unit.GetId()] = true
		}
	}
	if !hosted["test-isr-f15i-lead"] || !hosted["test-isr-f15i-wing"] || !hosted["test-isr-f16i-lead"] || !hosted["test-isr-f16i-wing"] || !hosted["test-isr-f35i-lead"] || !hosted["test-isr-f35i-wing"] {
		t.Fatalf("expected strike aircraft in package instance")
	}
}

func TestPackageTemplateOrderMatchesCatalog(t *testing.T) {
	order := PackageTemplateOrder()
	templates := PackageTemplates()
	if len(order) != len(templates) {
		t.Fatalf("expected package order to cover all templates: order=%d templates=%d", len(order), len(templates))
	}
	seen := map[string]struct{}{}
	for _, id := range order {
		if _, ok := templates[id]; !ok {
			t.Fatalf("package order contains unknown template %s", id)
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("package order contains duplicate template %s", id)
		}
		seen[id] = struct{}{}
	}
}

func TestLibraryDeepening_F35AHasInternalStrikeConfig(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	var found bool
	for _, def := range defs {
		if def.ID != "f35a-lightning" {
			continue
		}
		found = true
		if def.DefaultWeaponConfiguration != "internal_strike" {
			t.Fatalf("expected f35a-lightning default weapon config internal_strike, got %q", def.DefaultWeaponConfiguration)
		}
		var configFound bool
		for _, cfg := range def.WeaponConfigurations {
			if cfg.ID == "internal_strike" && len(cfg.Loadout) > 0 {
				configFound = true
				break
			}
		}
		if !configFound {
			t.Fatalf("expected f35a-lightning internal_strike config")
		}
	}
	if !found {
		t.Fatal("expected f35a-lightning definition")
	}
}

func TestLibraryDeepening_NewSupportDefinitionsPresent(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	required := map[string]bool{
		"green-pine-radar-section":                   false,
		"arrow-battle-management-center":             false,
		"davids-sling-radar-section":                 false,
		"davids-sling-battle-management-center":      false,
		"barak-mx-radar-section":                     false,
		"barak-mx-battle-management-center":          false,
		"expeditionary-aircraft-maintenance-complex": false,
		"theater-munitions-storage-area":             false,
		"missile-reload-transporter":                 false,
		"missile-decoy-detachment":                   false,
	}
	for _, def := range defs {
		if _, ok := required[def.ID]; ok {
			required[def.ID] = true
		}
	}
	for id, found := range required {
		if !found {
			t.Fatalf("expected library definition %s", id)
		}
	}
}

func TestPackageScenarioBuiltinsPresent(t *testing.T) {
	ids := []string{
		"pg-package-iron-dome-drill",
		"pg-package-al-udeid-sortie",
		"pg-package-layered-israel-defense",
		"pg-package-bahrain-maritime-presence",
		"pg-package-al-dhafra-forward-strike",
		"pg-package-gulf-regional-support-posture",
		"pg-package-hormuz-coastal-denial",
		"pg-package-uae-coastal-defense",
		"pg-package-oman-musandam-strait-guard",
		"pg-package-strait-regional-control",
	}
	for _, id := range ids {
		if scen := BuiltinByID(id); scen == nil {
			t.Fatalf("expected builtin scenario %s", id)
		}
	}
}

func TestPackageLayeredIsraelDefense_HasAirborneCounterstrikeWave(t *testing.T) {
	scen := BuiltinByID("pg-package-layered-israel-defense")
	if scen == nil {
		t.Fatal("expected builtin scenario")
	}
	required := map[string]bool{
		"pg-pkg-lad-isrstrike-f15i-lead": false,
		"pg-pkg-lad-isrstrike-f15i-wing": false,
		"pg-pkg-lad-isrstrike-f35i-lead": false,
		"pg-pkg-lad-isrstrike-f35i-wing": false,
	}
	for _, unit := range scen.GetUnits() {
		if _, ok := required[unit.GetId()]; !ok {
			continue
		}
		required[unit.GetId()] = true
		if unit.GetPosition().GetAltMsl() <= 0 {
			t.Fatalf("expected %s to be airborne", unit.GetId())
		}
		if unit.GetAttackOrder() == nil {
			t.Fatalf("expected %s to have an attack order", unit.GetId())
		}
	}
	for id, found := range required {
		if !found {
			t.Fatalf("expected airborne counterstrike unit %s", id)
		}
	}
}

func TestPackageGeometry_ISRLayeredDefenseNodesAreNotCollapsed(t *testing.T) {
	template, ok := PackageTemplateByID("pkg-isr-layered-defense-dan")
	if !ok {
		t.Fatal("expected layered defense package")
	}
	var arrow3, davidsSling, ironDomeDan, ironDomeJerusalem *libraryPoint
	for _, unit := range template.Units {
		switch unit.GetId() {
		case "arrow3":
			arrow3 = pointOf(unit.GetPosition().GetLat(), unit.GetPosition().GetLon())
		case "davids-sling":
			davidsSling = pointOf(unit.GetPosition().GetLat(), unit.GetPosition().GetLon())
		case "iron-dome-dan":
			ironDomeDan = pointOf(unit.GetPosition().GetLat(), unit.GetPosition().GetLon())
		case "iron-dome-jerusalem":
			ironDomeJerusalem = pointOf(unit.GetPosition().GetLat(), unit.GetPosition().GetLon())
		}
	}
	if arrow3 == nil || davidsSling == nil || ironDomeDan == nil || ironDomeJerusalem == nil {
		t.Fatal("expected key layered-defense nodes")
	}
	if kmBetween(*arrow3, *davidsSling) < 20 {
		t.Fatalf("expected Arrow and David's Sling nodes to be meaningfully separated")
	}
	if kmBetween(*ironDomeDan, *ironDomeJerusalem) < 25 {
		t.Fatalf("expected Dan and Jerusalem Iron Dome batteries to cover distinct defended areas")
	}
}

func TestPackageGeometry_IRNMissileRegimentIsDispersed(t *testing.T) {
	template, ok := PackageTemplateByID("pkg-irn-western-missile-regiment")
	if !ok {
		t.Fatal("expected Iranian missile regiment package")
	}
	points := map[string]*libraryPoint{}
	for _, unit := range template.Units {
		switch unit.GetId() {
		case "kheibar-1", "kheibar-2", "paveh-1", "paveh-2", "command", "reload", "decoy":
			points[unit.GetId()] = pointOf(unit.GetPosition().GetLat(), unit.GetPosition().GetLon())
		}
	}
	for _, id := range []string{"kheibar-1", "kheibar-2", "paveh-1", "paveh-2", "command", "reload", "decoy"} {
		if points[id] == nil {
			t.Fatalf("expected point for %s", id)
		}
	}
	if kmBetween(*points["kheibar-1"], *points["kheibar-2"]) < 80 {
		t.Fatalf("expected Kheibar launchers to be dispersed across western Iran")
	}
	if kmBetween(*points["paveh-1"], *points["paveh-2"]) < 120 {
		t.Fatalf("expected Paveh launchers to be dispersed across western Iran")
	}
	if kmBetween(*points["command"], *points["decoy"]) < 90 {
		t.Fatalf("expected decoy and command nodes to be well separated")
	}
}

type libraryPoint struct {
	lat float64
	lon float64
}

func pointOf(lat, lon float64) *libraryPoint {
	return &libraryPoint{lat: lat, lon: lon}
}

func kmBetween(a, b libraryPoint) float64 {
	const earthRadiusKm = 6371.0
	dLat := (b.lat - a.lat) * math.Pi / 180
	dLon := (b.lon - a.lon) * math.Pi / 180
	lat1 := a.lat * math.Pi / 180
	lat2 := b.lat * math.Pi / 180
	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(h))
}

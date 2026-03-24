package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/aressim/internal/geo"
	"github.com/aressim/internal/sim"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"google.golang.org/protobuf/proto"
)

type PathViolationPreview struct {
	Blocked  bool   `json:"blocked"`
	Country  string `json:"country,omitempty"`
	LegIndex int    `json:"legIndex,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type EngagementPreview struct {
	ReadyToFire          bool    `json:"readyToFire"`
	CanAssign            bool    `json:"canAssign"`
	WeaponId             string  `json:"weaponId,omitempty"`
	Reason               string  `json:"reason,omitempty"`
	ReasonCode           string  `json:"reasonCode,omitempty"`
	RangeToTargetM       float64 `json:"rangeToTargetM,omitempty"`
	WeaponRangeM         float64 `json:"weaponRangeM,omitempty"`
	FireProbability      float64 `json:"fireProbability,omitempty"`
	DesiredEffectSupport bool    `json:"desiredEffectSupport"`
	InStrikeCooldown     bool    `json:"inStrikeCooldown"`
}

type EngagementOptionPreview struct {
	TargetUnitId         string  `json:"targetUnitId"`
	TargetDisplayName    string  `json:"targetDisplayName"`
	TargetTeamId         string  `json:"targetTeamId"`
	ReadyToFire          bool    `json:"readyToFire"`
	CanAssign            bool    `json:"canAssign"`
	WeaponId             string  `json:"weaponId,omitempty"`
	Reason               string  `json:"reason,omitempty"`
	ReasonCode           string  `json:"reasonCode,omitempty"`
	RangeToTargetM       float64 `json:"rangeToTargetM,omitempty"`
	WeaponRangeM         float64 `json:"weaponRangeM,omitempty"`
	FireProbability      float64 `json:"fireProbability,omitempty"`
	DesiredEffectSupport bool    `json:"desiredEffectSupport"`
	InStrikeCooldown     bool    `json:"inStrikeCooldown"`
}

type TargetEngagementOptionPreview struct {
	ShooterUnitId            string  `json:"shooterUnitId"`
	ShooterDisplayName       string  `json:"shooterDisplayName"`
	ShooterTeamId            string  `json:"shooterTeamId"`
	LoadoutConfigurationId   string  `json:"loadoutConfigurationId,omitempty"`
	ReadyToFire              bool    `json:"readyToFire"`
	CanAssign                bool    `json:"canAssign"`
	WeaponId                 string  `json:"weaponId,omitempty"`
	Reason                   string  `json:"reason,omitempty"`
	ReasonCode               string  `json:"reasonCode,omitempty"`
	RangeToTargetM           float64 `json:"rangeToTargetM,omitempty"`
	WeaponRangeM             float64 `json:"weaponRangeM,omitempty"`
	FireProbability          float64 `json:"fireProbability,omitempty"`
	DesiredEffectSupport     bool    `json:"desiredEffectSupport"`
	InStrikeCooldown         bool    `json:"inStrikeCooldown"`
	PathBlocked              bool    `json:"pathBlocked"`
	PathReason               string  `json:"pathReason,omitempty"`
	EngagementCostUsd        float64 `json:"engagementCostUsd,omitempty"`
	ExpectedTargetValueUsd   float64 `json:"expectedTargetValueUsd,omitempty"`
	ExpectedValueExchangeUsd float64 `json:"expectedValueExchangeUsd,omitempty"`
}

type TargetEngagementDebugSummary struct {
	PlayerTeam             string `json:"playerTeam"`
	TargetUnitId           string `json:"targetUnitId"`
	TargetDisplayName      string `json:"targetDisplayName"`
	FriendlyUnitCount      int    `json:"friendlyUnitCount"`
	ReadyShooterCount      int    `json:"readyShooterCount"`
	AssignableShooterCount int    `json:"assignableShooterCount"`
	BlockedShooterCount    int    `json:"blockedShooterCount"`
	NonOperationalCount    int    `json:"nonOperationalCount"`
	NonHostileCount        int    `json:"nonHostileCount"`
}

const targetEngagementReasonNotOperational = "not_operational"
const targetEngagementReasonNotHostile = "not_hostile"

func scenarioCoalitionByTeam(units []*enginev1.Unit) map[string]string {
	coalitions := make(map[string]string, len(units))
	for _, unit := range units {
		if unit == nil {
			continue
		}
		teamID := sim.CountryDisplayCode(unit.GetTeamId())
		if teamID == "" {
			continue
		}
		if coalitionID := sim.CountryDisplayCode(unit.GetCoalitionId()); coalitionID != "" {
			coalitions[teamID] = coalitionID
		}
	}
	return coalitions
}

func friendlyToPlayerTeam(unit *enginev1.Unit, playerTeam string, coalitionByTeam map[string]string) bool {
	if unit == nil {
		return false
	}
	unitTeam := sim.CountryDisplayCode(unit.GetTeamId())
	if unitTeam == "" || playerTeam == "" {
		return false
	}
	if unitTeam == playerTeam {
		return true
	}
	playerCoalition := coalitionByTeam[playerTeam]
	unitCoalition := coalitionByTeam[unitTeam]
	return playerCoalition != "" && unitCoalition != "" && playerCoalition == unitCoalition
}

func (a *App) canonicalPlayerTeam(explicitTeamID string) string {
	if human := a.getHumanControlledTeam(); human != "" {
		return human
	}
	return sim.CountryDisplayCode(explicitTeamID)
}

func (a *App) buildTargetEngagementOptions(
	target *enginev1.Unit,
	playerTeam string,
	desiredEffect enginev1.DesiredEffect,
	strikeUntilEffect bool,
) ([]TargetEngagementOptionPreview, error) {
	if a.currentScenario == nil {
		return nil, fmt.Errorf("no scenario loaded")
	}
	if target == nil {
		return nil, fmt.Errorf("target not found")
	}
	if playerTeam == "" {
		return nil, fmt.Errorf("no player team selected")
	}
	defs := a.getCachedDefs()
	targetDef := defs[extractRecordID(target.GetDefinitionId())]
	weapons := a.getCachedWeaponCatalog()
	rules := a.relationshipRules()
	targetValue := estimateUnitValueUSD(target, targetDef)
	coalitionByTeam := scenarioCoalitionByTeam(a.currentScenario.GetUnits())
	options := make([]TargetEngagementOptionPreview, 0)
	for _, shooter := range a.currentScenario.GetUnits() {
		if shooter == nil || shooter.GetId() == target.GetId() {
			continue
		}
		if !friendlyToPlayerTeam(shooter, playerTeam, coalitionByTeam) {
			continue
		}
		option := TargetEngagementOptionPreview{
			ShooterUnitId:          shooter.GetId(),
			ShooterDisplayName:     shooter.GetDisplayName(),
			ShooterTeamId:          sim.CountryDisplayCode(shooter.GetTeamId()),
			LoadoutConfigurationId: shooter.GetLoadoutConfigurationId(),
			ExpectedTargetValueUsd: targetValue,
		}
		if !sim.UnitsAreHostileForUI(shooter, target) {
			option.Reason = "Target is not hostile to this unit."
			option.ReasonCode = targetEngagementReasonNotHostile
			options = append(options, option)
			continue
		}
		if !sim.UnitCanOperateForUI(shooter) {
			option.Reason = "Unit is not operational."
			option.ReasonCode = targetEngagementReasonNotOperational
			options = append(options, option)
			continue
		}
		decision := sim.EvaluateCurrentEngagement(
			shooter,
			target,
			a.currentScenario.GetUnits(),
			defs,
			weapons,
			rules,
			desiredEffect,
			strikeUntilEffect,
			a.getSimSeconds(),
		)
		canAssign := decision.WeaponID != "" && decision.Reason != sim.EngagementReasonHoldFire
		pathBlocked := false
		pathReason := ""
		if canAssign {
			if err := a.validateStrikeWithWeapon(shooter, target, decision.WeaponID); err != nil {
				pathBlocked = true
				pathReason = err.Error()
			}
		}
		engagementCost := estimateWeaponCostUSD(decision.WeaponID)
		option.ReadyToFire = decision.CanFire && !pathBlocked
		option.CanAssign = canAssign && !pathBlocked
		option.WeaponId = decision.WeaponID
		option.Reason = decision.ReasonText()
		option.ReasonCode = string(decision.Reason)
		option.RangeToTargetM = decision.RangeToTargetM
		option.WeaponRangeM = decision.WeaponRangeM
		option.FireProbability = decision.FireProbability
		option.DesiredEffectSupport = decision.DesiredEffectSupport
		option.InStrikeCooldown = decision.InStrikeCooldown
		option.PathBlocked = pathBlocked
		option.PathReason = pathReason
		option.EngagementCostUsd = engagementCost
		option.ExpectedValueExchangeUsd = decision.FireProbability*targetValue - engagementCost
		if pathBlocked {
			option.Reason = pathReason
			option.ReasonCode = "path_blocked"
		}
		options = append(options, option)
	}
	slices.SortFunc(options, func(a, b TargetEngagementOptionPreview) int {
		if a.ReadyToFire != b.ReadyToFire {
			if a.ReadyToFire {
				return -1
			}
			return 1
		}
		if a.CanAssign != b.CanAssign {
			if a.CanAssign {
				return -1
			}
			return 1
		}
		if a.ReasonCode == targetEngagementReasonNotOperational && b.ReasonCode != targetEngagementReasonNotOperational {
			return 1
		}
		if b.ReasonCode == targetEngagementReasonNotOperational && a.ReasonCode != targetEngagementReasonNotOperational {
			return -1
		}
		if a.ReasonCode == targetEngagementReasonNotHostile && b.ReasonCode != targetEngagementReasonNotHostile {
			return 1
		}
		if b.ReasonCode == targetEngagementReasonNotHostile && a.ReasonCode != targetEngagementReasonNotHostile {
			return -1
		}
		if a.ExpectedValueExchangeUsd != b.ExpectedValueExchangeUsd {
			if a.ExpectedValueExchangeUsd > b.ExpectedValueExchangeUsd {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ShooterDisplayName, b.ShooterDisplayName)
	})
	return options, nil
}

func estimateWeaponCostUSD(weaponID string) float64 {
	weaponID = strings.TrimSpace(strings.ToLower(weaponID))
	switch {
	case strings.Contains(weaponID, "tomahawk"):
		return 2_000_000
	case strings.Contains(weaponID, "jassm"):
		return 1_500_000
	case strings.Contains(weaponID, "harpoon"):
		return 1_400_000
	case strings.Contains(weaponID, "amraam"), strings.Contains(weaponID, "aim120"):
		return 1_200_000
	case strings.Contains(weaponID, "phoenix"), strings.Contains(weaponID, "aim54"):
		return 1_100_000
	case strings.Contains(weaponID, "sparrow"), strings.Contains(weaponID, "aim7"):
		return 500_000
	case strings.Contains(weaponID, "sidewinder"), strings.Contains(weaponID, "aim9"):
		return 450_000
	case strings.Contains(weaponID, "pac3"), strings.Contains(weaponID, "patriot"):
		return 4_000_000
	case strings.Contains(weaponID, "thaad"), strings.Contains(weaponID, "arrow"):
		return 3_000_000
	case strings.Contains(weaponID, "torp"), strings.Contains(weaponID, "mk48"):
		return 6_000_000
	case strings.Contains(weaponID, "shahed"), strings.Contains(weaponID, "arash"), strings.Contains(weaponID, "uav"):
		return 150_000
	case strings.Contains(weaponID, "qiam"), strings.Contains(weaponID, "fateh"), strings.Contains(weaponID, "kheibar"), strings.Contains(weaponID, "sejjil"):
		return 700_000
	default:
		return 1_000_000
	}
}

func estimateUnitValueUSD(unit *enginev1.Unit, def sim.DefStats) float64 {
	if unit == nil {
		return 0
	}
	remainingFraction := 1.0 - damageLossFraction(unit.GetDamageState())
	if remainingFraction < 0 {
		remainingFraction = 0
	}
	human := float64(def.AuthorizedPersonnel) * casualtyFraction(def, enginev1.DamageState_DAMAGE_STATE_DESTROYED, unit.GetStatus().GetPersonnelStrength()) * valueOfStatisticalLifeUSD
	return remainingFraction * (def.ReplacementCostUSD + def.StrategicValueUSD + def.EconomicValueUSD + human)
}

func (a *App) previewShooterWithLoadout(unitID string, loadoutConfigurationID string) (*enginev1.Unit, map[string]sim.DefStats, map[string]sim.WeaponStats, sim.RelationshipRules, error) {
	if a.currentScenario == nil {
		return nil, nil, nil, nil, fmt.Errorf("no scenario loaded")
	}
	var shooter *enginev1.Unit
	for _, candidate := range a.currentScenario.GetUnits() {
		if candidate.GetId() == unitID {
			shooter = candidate
			break
		}
	}
	if shooter == nil {
		return nil, nil, nil, nil, fmt.Errorf("unit %s not found", unitID)
	}
	previewShooter := proto.Clone(shooter).(*enginev1.Unit)
	if strings.TrimSpace(loadoutConfigurationID) != "" && strings.TrimSpace(loadoutConfigurationID) != strings.TrimSpace(previewShooter.GetLoadoutConfigurationId()) {
		def, found := a.libDefsCache[extractRecordID(previewShooter.GetDefinitionId())]
		if !found {
			return nil, nil, nil, nil, fmt.Errorf("unit definition not found")
		}
		selectedID, slots := selectWeaponConfiguration(def, loadoutConfigurationID)
		if len(slots) == 0 {
			return nil, nil, nil, nil, fmt.Errorf("selected loadout has no weapons")
		}
		previewShooter.LoadoutConfigurationId = selectedID
		previewShooter.Weapons = loadoutToWeaponStates(slots)
	}
	return previewShooter, a.getCachedDefs(), a.getCachedWeaponCatalog(), a.relationshipRules(), nil
}

type EffectiveRelationshipPreview struct {
	FromCountry                 string `json:"fromCountry"`
	ToCountry                   string `json:"toCountry"`
	ShareIntel                  bool   `json:"shareIntel"`
	AirspaceTransitAllowed      bool   `json:"airspaceTransitAllowed"`
	AirspaceStrikeAllowed       bool   `json:"airspaceStrikeAllowed"`
	DefensivePositioningAllowed bool   `json:"defensivePositioningAllowed"`
	MaritimeTransitAllowed      bool   `json:"maritimeTransitAllowed"`
	MaritimeStrikeAllowed       bool   `json:"maritimeStrikeAllowed"`
}

func isDisplayCountryCode(code string) bool {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return false
	}
	switch code {
	case "BLUE", "RED", "NEUTRAL", "DEBUG", "NON_ALIGNED":
		return false
	}
	return !strings.HasPrefix(code, "COALITION_")
}

func isMaritimeDomain(domain enginev1.UnitDomain) bool {
	return domain == enginev1.UnitDomain_DOMAIN_SEA || domain == enginev1.UnitDomain_DOMAIN_SUBSURFACE
}

type draftRelationshipInput struct {
	FromCountry                 string `json:"fromCountry"`
	ToCountry                   string `json:"toCountry"`
	ShareIntel                  bool   `json:"shareIntel"`
	AirspaceTransitAllowed      bool   `json:"airspaceTransitAllowed"`
	AirspaceStrikeAllowed       bool   `json:"airspaceStrikeAllowed"`
	DefensivePositioningAllowed bool   `json:"defensivePositioningAllowed"`
	MaritimeTransitAllowed      bool   `json:"maritimeTransitAllowed"`
	MaritimeStrikeAllowed       bool   `json:"maritimeStrikeAllowed"`
}

type draftPointInput struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func buildEffectiveRelationships(countries []string, rules sim.RelationshipRules, countryCoalitions map[string]string) []EffectiveRelationshipPreview {
	normalized := make([]string, 0, len(countries))
	seen := make(map[string]bool, len(countries))
	for _, country := range countries {
		code := geo.CountryCode(country)
		if !isDisplayCountryCode(code) || seen[code] {
			continue
		}
		seen[code] = true
		normalized = append(normalized, code)
	}
	slices.Sort(normalized)

	previews := make([]EffectiveRelationshipPreview, 0, len(normalized)*len(normalized))
	for _, from := range normalized {
		for _, to := range normalized {
			if from == to {
				continue
			}
			rule := sim.GetRelationshipRuleWithCoalitions(rules, countryCoalitions, from, to)
			previews = append(previews, EffectiveRelationshipPreview{
				FromCountry:                 from,
				ToCountry:                   to,
				ShareIntel:                  rule.ShareIntel,
				AirspaceTransitAllowed:      rule.AirspaceTransitAllowed,
				AirspaceStrikeAllowed:       rule.AirspaceStrikeAllowed,
				DefensivePositioningAllowed: rule.DefensivePositioningAllowed,
				MaritimeTransitAllowed:      rule.MaritimeTransitAllowed,
				MaritimeStrikeAllowed:       rule.MaritimeStrikeAllowed,
			})
		}
	}
	return previews
}

func currentScenarioCountries(scen *enginev1.Scenario) []string {
	if scen == nil {
		return nil
	}
	countries := make([]string, 0, len(scen.GetUnits())+len(scen.GetRelationships())*2)
	for _, unit := range scen.GetUnits() {
		if unit == nil {
			continue
		}
		if code := sim.CountryDisplayCode(unit.GetTeamId()); isDisplayCountryCode(code) {
			countries = append(countries, code)
		}
	}
	for _, relationship := range scen.GetRelationships() {
		if relationship == nil {
			continue
		}
		if code := geo.CountryCode(relationship.GetFromCountry()); isDisplayCountryCode(code) {
			countries = append(countries, code)
		}
		if code := geo.CountryCode(relationship.GetToCountry()); isDisplayCountryCode(code) {
			countries = append(countries, code)
		}
	}
	return countries
}

func previewTransitPath(ownerCountry string, maritime bool, points []geo.Point, rules sim.RelationshipRules, countryCoalitions map[string]string) *PathViolationPreview {
	ownerCountry = sim.CountryDisplayCode(ownerCountry)
	if ownerCountry == "" || len(points) < 2 {
		return nil
	}
	if maritime {
		for idx := 1; idx < len(points); idx++ {
			if geo.IsLandPoint(points[idx]) {
				return &PathViolationPreview{
					Blocked:  true,
					LegIndex: idx,
					Reason:   fmt.Sprintf("%s naval units cannot route onto land", ownerCountry),
				}
			}
			if geo.SegmentCrossesLand(points[idx-1], points[idx]) {
				return &PathViolationPreview{
					Blocked:  true,
					LegIndex: idx,
					Reason:   fmt.Sprintf("%s naval route crosses land", ownerCountry),
				}
			}
		}
	}
	for idx, segment := range geo.SamplePath(points) {
		var countries []string
		if maritime {
			countries = segment.SeaZoneOwners
		} else {
			countries = segment.AirspaceOwners
		}
		for _, country := range countries {
			if country == "" || country == ownerCountry {
				continue
			}
			rule := sim.GetRelationshipRuleWithCoalitions(rules, countryCoalitions, ownerCountry, country)
			allowed := rule.AirspaceTransitAllowed
			reasonFmt := "%s cannot transit %s airspace"
			if maritime {
				allowed = rule.MaritimeTransitAllowed
				reasonFmt = "%s cannot transit %s territorial waters"
			}
			if !allowed {
				return &PathViolationPreview{
					Blocked:  true,
					Country:  country,
					LegIndex: idx + 1,
					Reason:   fmt.Sprintf(reasonFmt, ownerCountry, country),
				}
			}
		}
	}
	return nil
}

func previewStrikePath(ownerCountry string, maritime bool, points []geo.Point, rules sim.RelationshipRules, countryCoalitions map[string]string) *PathViolationPreview {
	ownerCountry = sim.CountryDisplayCode(ownerCountry)
	if ownerCountry == "" || len(points) < 2 {
		return nil
	}
	if maritime {
		last := points[len(points)-1]
		if geo.IsLandPoint(last) {
			return &PathViolationPreview{
				Blocked:  true,
				LegIndex: len(points) - 1,
				Reason:   fmt.Sprintf("%s naval units cannot target land positions", ownerCountry),
			}
		}
		for idx := 1; idx < len(points); idx++ {
			if geo.SegmentCrossesLand(points[idx-1], points[idx]) {
				return &PathViolationPreview{
					Blocked:  true,
					LegIndex: idx,
					Reason:   fmt.Sprintf("%s naval strike path crosses land", ownerCountry),
				}
			}
		}
	}
	for idx, segment := range geo.SamplePath(points) {
		var countries []string
		if maritime {
			countries = segment.SeaZoneOwners
		} else {
			countries = segment.AirspaceOwners
		}
		for _, country := range countries {
			if country == "" || country == ownerCountry {
				continue
			}
			rule := sim.GetRelationshipRuleWithCoalitions(rules, countryCoalitions, ownerCountry, country)
			transitAllowed := rule.AirspaceTransitAllowed
			strikeAllowed := rule.AirspaceStrikeAllowed
			transitFmt := "%s cannot transit %s airspace"
			strikeFmt := "%s cannot conduct strike operations in %s airspace"
			if maritime {
				transitAllowed = rule.MaritimeTransitAllowed
				strikeAllowed = rule.MaritimeStrikeAllowed
				transitFmt = "%s cannot transit %s territorial waters"
				strikeFmt = "%s cannot conduct strike operations in %s territorial waters"
			}
			if !transitAllowed {
				return &PathViolationPreview{
					Blocked:  true,
					Country:  country,
					LegIndex: idx + 1,
					Reason:   fmt.Sprintf(transitFmt, ownerCountry, country),
				}
			}
			if !strikeAllowed {
				return &PathViolationPreview{
					Blocked:  true,
					Country:  country,
					LegIndex: idx + 1,
					Reason:   fmt.Sprintf(strikeFmt, ownerCountry, country),
				}
			}
		}
	}
	return nil
}

func previewDefensiveAirPath(ownerCountry string, points []geo.Point, rules sim.RelationshipRules, countryCoalitions map[string]string) *PathViolationPreview {
	ownerCountry = sim.CountryDisplayCode(ownerCountry)
	if ownerCountry == "" || len(points) < 2 {
		return nil
	}
	for idx, segment := range geo.SamplePath(points) {
		for _, country := range segment.AirspaceOwners {
			if country == "" || country == ownerCountry {
				continue
			}
			rule := sim.GetRelationshipRuleWithCoalitions(rules, countryCoalitions, ownerCountry, country)
			if !rule.AirspaceTransitAllowed {
				return &PathViolationPreview{
					Blocked:  true,
					Country:  country,
					LegIndex: idx + 1,
					Reason:   fmt.Sprintf("%s cannot transit %s airspace", ownerCountry, country),
				}
			}
			if !rule.DefensivePositioningAllowed {
				return &PathViolationPreview{
					Blocked:  true,
					Country:  country,
					LegIndex: idx + 1,
					Reason:   fmt.Sprintf("%s cannot conduct defensive air operations in %s airspace", ownerCountry, country),
				}
			}
		}
	}
	return nil
}

func (a *App) validateTransit(ownerCountry string, domain enginev1.UnitDomain, startLat, startLon, endLat, endLon float64) error {
	if violation := previewTransitPath(
		ownerCountry,
		isMaritimeDomain(domain),
		[]geo.Point{{Lat: startLat, Lon: startLon}, {Lat: endLat, Lon: endLon}},
		a.relationshipRules(),
		a.countryCoalitions(),
	); violation != nil {
		return fmt.Errorf("%s", violation.Reason)
	}
	return nil
}

func (a *App) validateStrike(shooter, target *enginev1.Unit) error {
	return a.validateStrikeWithWeapon(shooter, target, "")
}

func (a *App) shooterCanUseBallisticStrikeWeapon(shooter, target *enginev1.Unit, preferredWeaponID string) bool {
	if shooter == nil || target == nil {
		return false
	}
	targetDef := a.getCachedDefs()[extractRecordID(target.GetDefinitionId())]
	if targetDef.Domain == enginev1.UnitDomain_DOMAIN_UNSPECIFIED {
		return false
	}
	weapons := a.getCachedWeaponCatalog()
	preferredWeaponID = strings.TrimSpace(preferredWeaponID)
	for _, ws := range shooter.GetWeapons() {
		if ws.GetCurrentQty() <= 0 {
			continue
		}
		if preferredWeaponID != "" && ws.GetWeaponId() != preferredWeaponID {
			continue
		}
		weapon, ok := weapons[ws.GetWeaponId()]
		if !ok {
			continue
		}
		if weapon.EffectType != enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE {
			continue
		}
		if !weaponTargetsDomain(weapon.DomainTargets, targetDef.Domain) {
			continue
		}
		return true
	}
	return false
}

func weaponTargetsDomain(targets []enginev1.UnitDomain, domain enginev1.UnitDomain) bool {
	for _, candidate := range targets {
		if candidate == domain {
			return true
		}
	}
	return false
}

func weaponStatsForID(weapons map[string]sim.WeaponStats, weaponID string) (sim.WeaponStats, bool) {
	weaponID = strings.TrimSpace(weaponID)
	if weaponID == "" {
		return sim.WeaponStats{}, false
	}
	weapon, ok := weapons[weaponID]
	return weapon, ok
}

func isDefensiveAirInterceptMission(targetDef sim.DefStats, weapon sim.WeaponStats) bool {
	if targetDef.Domain == enginev1.UnitDomain_DOMAIN_UNSPECIFIED {
		return false
	}
	if weapon.EffectType != enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_AIR &&
		weapon.EffectType != enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_INTERCEPTOR {
		return false
	}
	return targetDef.Domain == enginev1.UnitDomain_DOMAIN_AIR
}

func (a *App) validateStrikeWithWeapon(shooter, target *enginev1.Unit, weaponID string) error {
	if shooter == nil || target == nil {
		return nil
	}
	if a.shooterCanUseBallisticStrikeWeapon(shooter, target, weaponID) {
		return nil
	}
	ownerCountry := unitCountryCode(shooter)
	if ownerCountry == "" {
		return nil
	}
	domain := a.getCachedDefs()[extractRecordID(shooter.DefinitionId)].Domain
	targetDef := a.getCachedDefs()[extractRecordID(target.DefinitionId)]
	points := [][2]float64{{shooter.GetPosition().GetLat(), shooter.GetPosition().GetLon()}}
	for _, wp := range shooter.GetMoveOrder().GetWaypoints() {
		points = append(points, [2]float64{wp.GetLat(), wp.GetLon()})
	}
	points = append(points, [2]float64{target.GetPosition().GetLat(), target.GetPosition().GetLon()})
	pathPoints := make([]geo.Point, 0, len(points))
	for _, point := range points {
		pathPoints = append(pathPoints, geo.Point{Lat: point[0], Lon: point[1]})
	}
	weapons := a.getCachedWeaponCatalog()
	if weapon, ok := weaponStatsForID(weapons, weaponID); ok && isDefensiveAirInterceptMission(targetDef, weapon) {
		if violation := previewDefensiveAirPath(ownerCountry, pathPoints, a.relationshipRules(), a.countryCoalitions()); violation != nil {
			return fmt.Errorf("%s", violation.Reason)
		}
		return nil
	}
	if violation := previewStrikePath(ownerCountry, isMaritimeDomain(domain), pathPoints, a.relationshipRules(), a.countryCoalitions()); violation != nil {
		return fmt.Errorf("%s", violation.Reason)
	}
	return nil
}

func (a *App) PreviewCurrentTransitPath(unitID string) (*PathViolationPreview, error) {
	if a.currentScenario == nil {
		return nil, fmt.Errorf("no scenario loaded")
	}
	var unit *enginev1.Unit
	for _, candidate := range a.currentScenario.GetUnits() {
		if candidate.GetId() == unitID {
			unit = candidate
			break
		}
	}
	if unit == nil {
		return nil, fmt.Errorf("unit %s not found", unitID)
	}
	if unit.GetMoveOrder() == nil || len(unit.GetMoveOrder().GetWaypoints()) == 0 {
		return &PathViolationPreview{Blocked: false}, nil
	}
	domain := a.getCachedDefs()[unit.DefinitionId].Domain
	points := make([]geo.Point, 0, len(unit.GetMoveOrder().GetWaypoints())+1)
	points = append(points, geo.Point{Lat: unit.GetPosition().GetLat(), Lon: unit.GetPosition().GetLon()})
	for _, wp := range unit.GetMoveOrder().GetWaypoints() {
		points = append(points, geo.Point{Lat: wp.GetLat(), Lon: wp.GetLon()})
	}
	violation := previewTransitPath(unitCountryCode(unit), isMaritimeDomain(domain), points, a.relationshipRules(), a.countryCoalitions())
	if violation == nil {
		return &PathViolationPreview{Blocked: false}, nil
	}
	return violation, nil
}

func (a *App) PreviewCurrentStrikePath(unitID string) (*PathViolationPreview, error) {
	if a.currentScenario == nil {
		return nil, fmt.Errorf("no scenario loaded")
	}
	var unit *enginev1.Unit
	var target *enginev1.Unit
	for _, candidate := range a.currentScenario.GetUnits() {
		if candidate.GetId() == unitID {
			unit = candidate
		}
	}
	if unit == nil {
		return nil, fmt.Errorf("unit %s not found", unitID)
	}
	if unit.GetAttackOrder() == nil || unit.GetAttackOrder().GetTargetUnitId() == "" {
		return &PathViolationPreview{Blocked: false}, nil
	}
	domain := a.getCachedDefs()[unit.DefinitionId].Domain
	for _, candidate := range a.currentScenario.GetUnits() {
		if candidate.GetId() == unit.GetAttackOrder().GetTargetUnitId() {
			target = candidate
			break
		}
	}
	if target == nil {
		return &PathViolationPreview{
			Blocked: true,
			Reason:  "Assigned target is no longer available.",
		}, nil
	}
	points := make([]geo.Point, 0, len(unit.GetMoveOrder().GetWaypoints())+2)
	points = append(points, geo.Point{Lat: unit.GetPosition().GetLat(), Lon: unit.GetPosition().GetLon()})
	for _, wp := range unit.GetMoveOrder().GetWaypoints() {
		points = append(points, geo.Point{Lat: wp.GetLat(), Lon: wp.GetLon()})
	}
	points = append(points, geo.Point{Lat: target.GetPosition().GetLat(), Lon: target.GetPosition().GetLon()})
	decision := sim.EvaluateCurrentEngagement(
		unit,
		target,
		a.currentScenario.GetUnits(),
		a.getCachedDefs(),
		a.getCachedWeaponCatalog(),
		a.relationshipRules(),
		unit.GetAttackOrder().GetDesiredEffect(),
		unit.GetAttackOrder().GetOrderType() == enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT,
		a.getSimSeconds(),
	)
	var violation *PathViolationPreview
	if weapon, ok := weaponStatsForID(a.getCachedWeaponCatalog(), decision.WeaponID); ok && isDefensiveAirInterceptMission(a.getCachedDefs()[extractRecordID(target.DefinitionId)], weapon) {
		violation = previewDefensiveAirPath(unitCountryCode(unit), points, a.relationshipRules(), a.countryCoalitions())
	} else {
		violation = previewStrikePath(unitCountryCode(unit), isMaritimeDomain(domain), points, a.relationshipRules(), a.countryCoalitions())
	}
	if violation == nil {
		return &PathViolationPreview{Blocked: false}, nil
	}
	return violation, nil
}

func (a *App) PreviewCurrentEngagement(unitID string) (*EngagementPreview, error) {
	shooter, _, _, _, err := a.previewShooterWithLoadout(unitID, "")
	if err != nil {
		return nil, err
	}
	order := shooter.GetAttackOrder()
	if order == nil || order.GetTargetUnitId() == "" {
		return &EngagementPreview{Reason: "No target assigned.", ReasonCode: string(sim.EngagementReasonNoTarget)}, nil
	}
	var target *enginev1.Unit
	for _, candidate := range a.currentScenario.GetUnits() {
		if candidate.GetId() == order.GetTargetUnitId() {
			target = candidate
			break
		}
	}
	if target == nil {
		return &EngagementPreview{Reason: "Assigned target no longer exists.", ReasonCode: string(sim.EngagementReasonNoTarget)}, nil
	}
	decision := sim.EvaluateCurrentEngagement(
		shooter,
		target,
		a.currentScenario.GetUnits(),
		a.getCachedDefs(),
		a.getCachedWeaponCatalog(),
		a.relationshipRules(),
		order.GetDesiredEffect(),
		order.GetOrderType() == enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT,
		a.getSimSeconds(),
	)
	return &EngagementPreview{
		ReadyToFire:          decision.CanFire,
		CanAssign:            decision.WeaponID != "" && decision.Reason != sim.EngagementReasonHoldFire,
		WeaponId:             decision.WeaponID,
		Reason:               decision.ReasonText(),
		ReasonCode:           string(decision.Reason),
		RangeToTargetM:       decision.RangeToTargetM,
		WeaponRangeM:         decision.WeaponRangeM,
		FireProbability:      decision.FireProbability,
		DesiredEffectSupport: decision.DesiredEffectSupport,
		InStrikeCooldown:     decision.InStrikeCooldown,
	}, nil
}

func (a *App) PreviewEngagementOptionsForLoadout(unitID string, loadoutConfigurationID string) ([]EngagementOptionPreview, error) {
	shooter, defs, weapons, rules, err := a.previewShooterWithLoadout(unitID, loadoutConfigurationID)
	if err != nil {
		return nil, err
	}
	options := make([]EngagementOptionPreview, 0)
	for _, target := range a.currentScenario.GetUnits() {
		if target == nil || target.GetId() == shooter.GetId() || !sim.UnitCanOperateForUI(target) {
			continue
		}
		if !sim.UnitsAreHostileForUI(shooter, target) {
			continue
		}
		decision := sim.EvaluateCurrentEngagement(
			shooter,
			target,
			a.currentScenario.GetUnits(),
			defs,
			weapons,
			rules,
			enginev1.DesiredEffect_DESIRED_EFFECT_UNSPECIFIED,
			false,
			a.getSimSeconds(),
		)
		canAssign := decision.WeaponID != "" && decision.Reason != sim.EngagementReasonHoldFire
		options = append(options, EngagementOptionPreview{
			TargetUnitId:         target.GetId(),
			TargetDisplayName:    target.GetDisplayName(),
			TargetTeamId:         sim.CountryDisplayCode(target.GetTeamId()),
			ReadyToFire:          decision.CanFire,
			CanAssign:            canAssign,
			WeaponId:             decision.WeaponID,
			Reason:               decision.ReasonText(),
			ReasonCode:           string(decision.Reason),
			RangeToTargetM:       decision.RangeToTargetM,
			WeaponRangeM:         decision.WeaponRangeM,
			FireProbability:      decision.FireProbability,
			DesiredEffectSupport: decision.DesiredEffectSupport,
			InStrikeCooldown:     decision.InStrikeCooldown,
		})
	}
	slices.SortFunc(options, func(a, b EngagementOptionPreview) int {
		if a.ReadyToFire != b.ReadyToFire {
			if a.ReadyToFire {
				return -1
			}
			return 1
		}
		if a.CanAssign != b.CanAssign {
			if a.CanAssign {
				return -1
			}
			return 1
		}
		if a.TargetTeamId != b.TargetTeamId {
			return strings.Compare(a.TargetTeamId, b.TargetTeamId)
		}
		return strings.Compare(a.TargetDisplayName, b.TargetDisplayName)
	})
	return options, nil
}

func (a *App) PreviewEngagementOptions(unitID string) ([]EngagementOptionPreview, error) {
	return a.PreviewEngagementOptionsForLoadout(unitID, "")
}

func (a *App) PreviewTargetEngagementOptions(targetUnitID string, playerTeamID string) ([]TargetEngagementOptionPreview, error) {
	if a.currentScenario == nil {
		return nil, fmt.Errorf("no scenario loaded")
	}
	playerTeam := a.canonicalPlayerTeam(playerTeamID)
	if playerTeam == "" {
		return nil, fmt.Errorf("no player team selected")
	}
	var target *enginev1.Unit
	for _, candidate := range a.currentScenario.GetUnits() {
		if candidate.GetId() == targetUnitID {
			target = candidate
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("target %s not found", targetUnitID)
	}
	return a.buildTargetEngagementOptions(
		target,
		playerTeam,
		enginev1.DesiredEffect_DESIRED_EFFECT_UNSPECIFIED,
		false,
	)
}

func (a *App) PreviewTargetEngagementSummary(targetUnitID string, playerTeamID string) (*TargetEngagementDebugSummary, error) {
	if a.currentScenario == nil {
		return nil, fmt.Errorf("no scenario loaded")
	}
	playerTeam := a.canonicalPlayerTeam(playerTeamID)
	if playerTeam == "" {
		return nil, fmt.Errorf("no player team selected")
	}
	var target *enginev1.Unit
	for _, candidate := range a.currentScenario.GetUnits() {
		if candidate.GetId() == targetUnitID {
			target = candidate
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("target %s not found", targetUnitID)
	}
	options, err := a.buildTargetEngagementOptions(
		target,
		playerTeam,
		enginev1.DesiredEffect_DESIRED_EFFECT_UNSPECIFIED,
		false,
	)
	if err != nil {
		return nil, err
	}
	summary := &TargetEngagementDebugSummary{
		PlayerTeam:        playerTeam,
		TargetUnitId:      target.GetId(),
		TargetDisplayName: target.GetDisplayName(),
	}
	for _, option := range options {
		summary.FriendlyUnitCount++
		if option.ReadyToFire {
			summary.ReadyShooterCount++
			continue
		}
		if option.CanAssign {
			summary.AssignableShooterCount++
			continue
		}
		summary.BlockedShooterCount++
		switch option.ReasonCode {
		case targetEngagementReasonNotOperational:
			summary.NonOperationalCount++
		case targetEngagementReasonNotHostile:
			summary.NonHostileCount++
		}
	}
	return summary, nil
}

func parseDraftRelationshipRules(relationshipsJSON string) (sim.RelationshipRules, error) {
	if strings.TrimSpace(relationshipsJSON) == "" {
		return nil, nil
	}
	var rels []draftRelationshipInput
	if err := json.Unmarshal([]byte(relationshipsJSON), &rels); err != nil {
		return nil, fmt.Errorf("decode relationships: %w", err)
	}
	protoRels := make([]*enginev1.CountryRelationship, 0, len(rels))
	for _, rel := range rels {
		protoRels = append(protoRels, &enginev1.CountryRelationship{
			FromCountry:                 rel.FromCountry,
			ToCountry:                   rel.ToCountry,
			ShareIntel:                  rel.ShareIntel,
			AirspaceTransitAllowed:      rel.AirspaceTransitAllowed,
			AirspaceStrikeAllowed:       rel.AirspaceStrikeAllowed,
			DefensivePositioningAllowed: rel.DefensivePositioningAllowed,
			MaritimeTransitAllowed:      rel.MaritimeTransitAllowed,
			MaritimeStrikeAllowed:       rel.MaritimeStrikeAllowed,
		})
	}
	return sim.BuildRelationshipRules(protoRels), nil
}

func parseDraftCountryCoalitions(countryCoalitionsJSON string) (map[string]string, error) {
	if strings.TrimSpace(countryCoalitionsJSON) == "" {
		return nil, nil
	}
	var raw map[string]string
	if err := json.Unmarshal([]byte(countryCoalitionsJSON), &raw); err != nil {
		return nil, fmt.Errorf("decode country coalitions: %w", err)
	}
	normalized := make(map[string]string, len(raw))
	for country, coalition := range raw {
		country = geo.CountryCode(country)
		coalition = geo.CountryCode(coalition)
		if country == "" || coalition == "" {
			continue
		}
		normalized[country] = coalition
	}
	return normalized, nil
}

func parseDraftPoints(pointsJSON string) ([]geo.Point, error) {
	if strings.TrimSpace(pointsJSON) == "" {
		return nil, nil
	}
	var raw []draftPointInput
	if err := json.Unmarshal([]byte(pointsJSON), &raw); err != nil {
		return nil, fmt.Errorf("decode points: %w", err)
	}
	points := make([]geo.Point, 0, len(raw))
	for _, point := range raw {
		points = append(points, geo.Point{Lat: point.Lat, Lon: point.Lon})
	}
	return points, nil
}

func parseDraftCountries(countriesJSON string) ([]string, error) {
	if strings.TrimSpace(countriesJSON) == "" {
		return nil, nil
	}
	var countries []string
	if err := json.Unmarshal([]byte(countriesJSON), &countries); err != nil {
		return nil, fmt.Errorf("decode countries: %w", err)
	}
	return countries, nil
}

func (a *App) PreviewDraftTransitPath(ownerCountry string, maritime bool, relationshipsJSON, countryCoalitionsJSON, pointsJSON string) (*PathViolationPreview, error) {
	rules, err := parseDraftRelationshipRules(relationshipsJSON)
	if err != nil {
		return nil, err
	}
	countryCoalitions, err := parseDraftCountryCoalitions(countryCoalitionsJSON)
	if err != nil {
		return nil, err
	}
	points, err := parseDraftPoints(pointsJSON)
	if err != nil {
		return nil, err
	}
	violation := previewTransitPath(ownerCountry, maritime, points, rules, countryCoalitions)
	if violation == nil {
		return &PathViolationPreview{Blocked: false}, nil
	}
	return violation, nil
}

func (a *App) PreviewDraftStrikePath(ownerCountry string, maritime bool, relationshipsJSON, countryCoalitionsJSON, pointsJSON string) (*PathViolationPreview, error) {
	rules, err := parseDraftRelationshipRules(relationshipsJSON)
	if err != nil {
		return nil, err
	}
	countryCoalitions, err := parseDraftCountryCoalitions(countryCoalitionsJSON)
	if err != nil {
		return nil, err
	}
	points, err := parseDraftPoints(pointsJSON)
	if err != nil {
		return nil, err
	}
	violation := previewStrikePath(ownerCountry, maritime, points, rules, countryCoalitions)
	if violation == nil {
		return &PathViolationPreview{Blocked: false}, nil
	}
	return violation, nil
}

func (a *App) PreviewDraftPlacement(ownerCountry string, maritime bool, employmentRole, relationshipsJSON, countryCoalitionsJSON string, lat, lon float64) (*PathViolationPreview, error) {
	ownerCountry = geo.CountryCode(ownerCountry)
	if ownerCountry == "" {
		return &PathViolationPreview{Blocked: false}, nil
	}
	rules, err := parseDraftRelationshipRules(relationshipsJSON)
	if err != nil {
		return nil, err
	}
	countryCoalitions, err := parseDraftCountryCoalitions(countryCoalitionsJSON)
	if err != nil {
		return nil, err
	}
	ctx := geo.LookupPoint(geo.Point{Lat: lat, Lon: lon})
	hostCountry := ctx.AirspaceOwner
	hostLabel := "airspace"
	if maritime {
		hostCountry = ctx.SeaZoneOwner
		hostLabel = "territorial waters"
	}
	if hostCountry == "" || hostCountry == ownerCountry {
		return &PathViolationPreview{Blocked: false}, nil
	}
	rule := sim.GetRelationshipRuleWithCoalitions(rules, countryCoalitions, ownerCountry, hostCountry)
	role := strings.TrimSpace(strings.ToLower(employmentRole))
	if role == "defensive" {
		if !rule.DefensivePositioningAllowed {
			return &PathViolationPreview{
				Blocked: true,
				Country: hostCountry,
				Reason:  fmt.Sprintf("%s cannot position defensive assets inside %s %s", ownerCountry, hostCountry, hostLabel),
			}, nil
		}
		return &PathViolationPreview{Blocked: false}, nil
	}
	offensiveAllowed := rule.AirspaceStrikeAllowed
	if maritime {
		offensiveAllowed = rule.MaritimeStrikeAllowed
	}
	if !offensiveAllowed {
		return &PathViolationPreview{
			Blocked: true,
			Country: hostCountry,
			Reason:  fmt.Sprintf("%s cannot position offensive-capable assets inside %s %s", ownerCountry, hostCountry, hostLabel),
		}, nil
	}
	return &PathViolationPreview{Blocked: false}, nil
}

func (a *App) PreviewCurrentRelationships() ([]EffectiveRelationshipPreview, error) {
	if a.currentScenario == nil {
		return nil, fmt.Errorf("no scenario loaded")
	}
	return buildEffectiveRelationships(
		currentScenarioCountries(a.currentScenario),
		a.relationshipRules(),
		a.countryCoalitions(),
	), nil
}

func (a *App) PreviewDraftRelationships(relationshipsJSON, countryCoalitionsJSON, countriesJSON string) ([]EffectiveRelationshipPreview, error) {
	rules, err := parseDraftRelationshipRules(relationshipsJSON)
	if err != nil {
		return nil, err
	}
	countryCoalitions, err := parseDraftCountryCoalitions(countryCoalitionsJSON)
	if err != nil {
		return nil, err
	}
	countries, err := parseDraftCountries(countriesJSON)
	if err != nil {
		return nil, err
	}
	return buildEffectiveRelationships(countries, rules, countryCoalitions), nil
}

package scenario

import "strings"

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
		"pkg-isr-iron-dome-battery":         packageISRIronDomeBattery(),
		"pkg-isr-layered-defense-dan":       packageISRLayeredDefenseDan(),
		"pkg-isr-counterstrike-cell":        packageISRCounterstrikeCell(),
		"pkg-usa-al-udeid-airbase":          packageUSAAlUdeidAirbase(),
		"pkg-are-al-dhafra-airbase":         packageAREAlDhafraAirbase(),
		"pkg-are-abu-dhabi-coastal-defense": packageAREAbuDhabiCoastalDefense(),
		"pkg-omn-musandam-strait-guard":     packageOMNMusandamStraitGuard(),
		"pkg-bhr-naval-support-bahrain":     packageBHRNavalSupportBahrain(),
		"pkg-irn-hormuz-coastal-denial":     packageIRNHormuzCoastalDenial(),
		"pkg-irn-western-missile-regiment":  packageIRNWesternMissileRegiment(),
	}
}

func PackageTemplateByID(id string) (PackageTemplate, bool) {
	pkg, ok := PackageTemplates()[strings.TrimSpace(id)]
	return pkg, ok
}

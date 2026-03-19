package scenario

import (
	"testing"
	"time"
)

func TestIranCoalitionWarSkeletonStartsAtOpeningStrike(t *testing.T) {
	scen := IranCoalitionWarSkeleton()
	got := time.Unix(int64(scen.GetStartTimeUnix()), 0).UTC()
	want := time.Date(2026, 2, 28, 1, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("expected scenario start %v, got %v", want, got)
	}
}

func TestIranCoalitionWarSkeletonAssignsHostBases(t *testing.T) {
	scen := IranCoalitionWarSkeleton()
	byID := make(map[string]string, len(scen.GetUnits()))
	for _, unit := range scen.GetUnits() {
		byID[unit.GetId()] = unit.GetHostBaseId()
	}
	cases := map[string]string{
		"isr-f35i-nevatim":     "isr-airbase-nevatim",
		"isr-f15i-hatzor":      "isr-airbase-hatzor",
		"isr-f16i-ramon":       "isr-airbase-ramon",
		"usa-f35a-al-udeid":    "qat-airbase-al-udeid",
		"usa-f15e-al-dhafra":   "uae-airbase-al-dhafra",
		"usa-b1b-diego-garcia": "usa-airbase-diego-garcia",
		"usa-kc46-gulf":        "qat-airbase-al-udeid",
		"irn-f14-tehran":       "irn-airbase-tehran",
		"irn-f4-bandar-abbas":  "irn-airbase-bandar-abbas",
		"isr-eitam-central":    "isr-airbase-nevatim",
		"isr-oron-central":     "isr-airbase-hatzor",
		"isr-reem-support":     "isr-airbase-nevatim",
		"sau-f15sa-khamis":     "sau-airbase-khamis",
		"sau-e3a-riyadh":       "sau-airbase-khamis",
		"uae-f16-block60":      "uae-airbase-al-dhafra",
		"uae-globaleye":        "uae-airbase-al-dhafra",
		"qat-f15qa":            "qat-airbase-al-udeid",
		"omn-f16-seeb":         "omn-airbase-seeb",
		"bhr-f16v-isa":         "bhr-airbase-isa",
		"jord-f16-central":     "jor-airbase-central",
	}
	for unitID, wantBaseID := range cases {
		if got := byID[unitID]; got != wantBaseID {
			t.Fatalf("expected %s host base %s, got %s", unitID, wantBaseID, got)
		}
	}
}

func TestIranCoalitionWarSkeletonSeedsOpeningStrikeActions(t *testing.T) {
	scen := IranCoalitionWarSkeleton()
	if len(scen.GetOpeningStrikeActions()) < 3 {
		t.Fatalf("expected opening strike actions to be seeded, got %d", len(scen.GetOpeningStrikeActions()))
	}
	if scen.GetOpeningStrikeActions()[0].GetUnitId() == "" || scen.GetOpeningStrikeActions()[0].GetTargetUnitId() == "" {
		t.Fatal("expected opening strike actions to include shooter and target ids")
	}
}

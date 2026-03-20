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
		"isr-f35i-nevatim":      "isr-airbase-nevatim",
		"isr-f35i-nevatim-2":    "isr-airbase-nevatim",
		"isr-f15i-hatzor":       "isr-airbase-hatzor",
		"isr-f15i-hatzor-2":     "isr-airbase-hatzor",
		"isr-f16i-ramon":        "isr-airbase-ramon",
		"isr-f16i-ramon-2":      "isr-airbase-ramon",
		"usa-f35a-al-udeid":     "qat-airbase-al-udeid",
		"usa-f22a-al-udeid":     "qat-airbase-al-udeid",
		"usa-p8a-gulf":          "qat-airbase-al-udeid",
		"usa-f15e-al-dhafra":    "uae-airbase-al-dhafra",
		"usa-b1b-diego-garcia":  "usa-airbase-diego-garcia",
		"usa-c17a-diego-garcia": "usa-airbase-diego-garcia",
		"usa-kc46-gulf":         "qat-airbase-al-udeid",
		"irn-f14-tehran":        "irn-airbase-tehran",
		"irn-f14-tehran-2":      "irn-airbase-tehran",
		"irn-su24-tehran":       "irn-airbase-tehran",
		"irn-707-tehran":        "irn-airbase-tehran",
		"irn-f4-bandar-abbas":   "irn-airbase-bandar-abbas",
		"irn-p3f-bandar-abbas":  "irn-airbase-bandar-abbas",
		"isr-eitam-central":     "isr-airbase-nevatim",
		"isr-oron-central":      "isr-airbase-hatzor",
		"isr-reem-support":      "isr-airbase-nevatim",
		"sau-f15sa-khamis":      "sau-airbase-khamis",
		"sau-f15sa-khamis-2":    "sau-airbase-khamis",
		"sau-e3a-riyadh":        "sau-airbase-khamis",
		"uae-f16-block60":       "uae-airbase-al-dhafra",
		"uae-f16-block60-2":     "uae-airbase-al-dhafra",
		"uae-globaleye":         "uae-airbase-al-dhafra",
		"uae-dash8-mpa":         "uae-airbase-al-dhafra",
		"qat-f15qa":             "qat-airbase-al-udeid",
		"qat-f15qa-2":           "qat-airbase-al-udeid",
		"omn-f16-seeb":          "omn-airbase-seeb",
		"omn-cn235-mpa":         "omn-airbase-seeb",
		"omn-super-lynx":        "omn-airbase-seeb",
		"bhr-f16v-isa":          "bhr-airbase-isa",
		"bhr-bell412":           "bhr-airbase-isa",
		"kwt-fa18-ahmad":        "kwt-airbase-ahmad",
		"kwt-kc130j-ahmad":      "kwt-airbase-ahmad",
		"jord-f16-central":      "jor-airbase-central",
	}
	for unitID, wantBaseID := range cases {
		if got := byID[unitID]; got != wantBaseID {
			t.Fatalf("expected %s host base %s, got %s", unitID, wantBaseID, got)
		}
	}
}

func TestIranCoalitionWarSkeletonHasExpandedInitialOrderOfBattle(t *testing.T) {
	scen := IranCoalitionWarSkeleton()
	if got := len(scen.GetUnits()); got < 60 {
		t.Fatalf("expected expanded initial order of battle, got only %d units", got)
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

func TestIranCoalitionWarSkeletonStagesIranianRetaliation(t *testing.T) {
	scen := IranCoalitionWarSkeleton()
	ready := map[string]float64{}
	for _, unit := range scen.GetUnits() {
		switch unit.GetId() {
		case "irn-qiam-central", "irn-kheibar-west", "irn-paveh-south", "irn-shahed-central", "irn-arash-west":
			ready[unit.GetId()] = unit.GetNextStrikeReadySeconds()
		}
	}
	if ready["irn-qiam-central"] <= 0 {
		t.Fatal("expected qiam brigade to have delayed retaliatory readiness")
	}
	if !(ready["irn-qiam-central"] < ready["irn-kheibar-west"] &&
		ready["irn-kheibar-west"] < ready["irn-paveh-south"] &&
		ready["irn-paveh-south"] < ready["irn-arash-west"]) {
		t.Fatalf("expected staggered iranian strike readiness, got %+v", ready)
	}
}

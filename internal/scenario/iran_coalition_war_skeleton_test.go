package scenario

import (
	"testing"
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
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
		"isr-f35i-nevatim":        "isr-airbase-nevatim",
		"isr-f35i-nevatim-2":      "isr-airbase-nevatim",
		"isr-f35i-nevatim-3":      "isr-airbase-nevatim",
		"isr-f35i-nevatim-4":      "isr-airbase-nevatim",
		"isr-f15i-hatzor":         "isr-airbase-hatzor",
		"isr-f15i-hatzor-2":       "isr-airbase-hatzor",
		"isr-f15i-hatzor-3":       "isr-airbase-hatzor",
		"isr-f16i-ramon":          "isr-airbase-ramon",
		"isr-f16i-ramon-2":        "isr-airbase-ramon",
		"isr-f16i-ramon-3":        "isr-airbase-ramon",
		"isr-f16i-ramat-david":    "isr-airbase-ramat-david",
		"isr-f16i-telnof":         "isr-airbase-telnof",
		"usa-f35c-ford":           "usa-cvn78-redsea",
		"usa-fa18e-ford":          "usa-cvn78-redsea",
		"usa-fa18f-ford":          "usa-cvn78-redsea",
		"usa-ea18g-ford":          "usa-cvn78-redsea",
		"usa-e2d-ford":            "usa-cvn78-redsea",
		"usa-mh60r-ford":          "usa-cvn78-redsea",
		"usa-fa18e-nimitz":        "usa-cvn68-arabian-sea",
		"usa-ea18g-nimitz":        "usa-cvn68-arabian-sea",
		"usa-e2d-nimitz":          "usa-cvn68-arabian-sea",
		"usa-f35a-al-udeid":       "qat-airbase-al-udeid",
		"usa-f35a-dhafra":         "uae-airbase-al-dhafra",
		"usa-f22a-al-udeid":       "qat-airbase-al-udeid",
		"usa-f22a-al-udeid-2":     "qat-airbase-al-udeid",
		"usa-p8a-gulf":            "qat-airbase-al-udeid",
		"usa-p8a-dhafra":          "uae-airbase-al-dhafra",
		"usa-f15e-al-dhafra":      "uae-airbase-al-dhafra",
		"usa-f15ex-prince-sultan": "sau-airbase-prince-sultan",
		"usa-b1b-diego-garcia":    "usa-airbase-diego-garcia",
		"usa-c17a-diego-garcia":   "usa-airbase-diego-garcia",
		"usa-c17a-qatar":          "qat-airbase-al-udeid",
		"usa-kc46-gulf":           "qat-airbase-al-udeid",
		"usa-kc46-dhafra":         "uae-airbase-al-dhafra",
		"usa-rq4-prince-sultan":   "sau-airbase-prince-sultan",
		"usa-mq9-dhafra":          "uae-airbase-al-dhafra",
		"irn-f14-tehran":          "irn-airbase-tehran",
		"irn-f14-tehran-2":        "irn-airbase-tehran",
		"irn-f14-tabriz":          "irn-airbase-tabriz",
		"irn-f14-esfahan":         "irn-airbase-esfahan",
		"irn-su24-tehran":         "irn-airbase-tehran",
		"irn-su24-shiraz":         "irn-airbase-shiraz",
		"irn-707-tehran":          "irn-airbase-tehran",
		"irn-707-esfahan":         "irn-airbase-esfahan",
		"irn-f4-bandar-abbas":     "irn-airbase-bandar-abbas",
		"irn-f4-bushehr":          "irn-airbase-bushehr",
		"irn-p3f-bandar-abbas":    "irn-airbase-bandar-abbas",
		"irn-p3f-bushehr":         "irn-airbase-bushehr",
		"isr-eitam-central":       "isr-airbase-nevatim",
		"isr-eitam-central-2":     "isr-airbase-telnof",
		"isr-oron-central":        "isr-airbase-hatzor",
		"isr-oron-central-2":      "isr-airbase-nevatim",
		"isr-reem-support":        "isr-airbase-nevatim",
		"isr-reem-support-2":      "isr-airbase-nevatim",
		"isr-heron-nevatim":       "isr-airbase-nevatim",
		"isr-heron-palmachim":     "isr-airbase-palmachim",
		"isr-hermes900-palmachim": "isr-airbase-palmachim",
		"isr-hermes450-ramon":     "isr-airbase-ramon",
		"sau-f15sa-khamis":        "sau-airbase-khamis",
		"sau-f15sa-khamis-2":      "sau-airbase-khamis",
		"sau-e3a-riyadh":          "sau-airbase-khamis",
		"uae-f16-block60":         "uae-airbase-al-dhafra",
		"uae-f16-block60-2":       "uae-airbase-al-dhafra",
		"uae-globaleye":           "uae-airbase-al-dhafra",
		"uae-dash8-mpa":           "uae-airbase-al-dhafra",
		"qat-f15qa":               "qat-airbase-al-udeid",
		"qat-f15qa-2":             "qat-airbase-al-udeid",
		"omn-f16-seeb":            "omn-airbase-seeb",
		"omn-cn235-mpa":           "omn-airbase-seeb",
		"omn-super-lynx":          "omn-airbase-seeb",
		"bhr-f16v-isa":            "bhr-airbase-isa",
		"bhr-bell412":             "bhr-airbase-isa",
		"kwt-fa18-ahmad":          "kwt-airbase-ahmad",
		"kwt-kc130j-ahmad":        "kwt-airbase-ahmad",
		"jord-f16-central":        "jor-airbase-central",
	}
	for unitID, wantBaseID := range cases {
		if got := byID[unitID]; got != wantBaseID {
			t.Fatalf("expected %s host base %s, got %s", unitID, wantBaseID, got)
		}
	}
}

func TestIranCoalitionWarSkeletonHasDeepIsraeliOrderOfBattle(t *testing.T) {
	scen := IranCoalitionWarSkeleton()
	count := countUnitsWithPrefix(scen, "isr-")
	if count < 30 {
		t.Fatalf("expected deep Israeli order of battle, got only %d israeli units", count)
	}
}

func TestIranCoalitionWarSkeletonHasDeepUnitedStatesOrderOfBattle(t *testing.T) {
	scen := IranCoalitionWarSkeleton()
	count := countUnitsWithPrefix(scen, "usa-")
	if count < 20 {
		t.Fatalf("expected deep U.S. order of battle, got only %d U.S. units", count)
	}
}

func TestIranCoalitionWarSkeletonHasDeepIranianOrderOfBattle(t *testing.T) {
	scen := IranCoalitionWarSkeleton()
	count := countUnitsWithPrefix(scen, "irn-")
	if count < 30 {
		t.Fatalf("expected deep Iranian order of battle, got only %d Iranian units", count)
	}
}

func TestIranCoalitionWarSkeletonHasExpandedInitialOrderOfBattle(t *testing.T) {
	scen := IranCoalitionWarSkeleton()
	if got := len(scen.GetUnits()); got < 90 {
		t.Fatalf("expected expanded initial order of battle, got only %d units", got)
	}
}

func TestIranCoalitionWarSkeletonLeavesIranianRetaliationToAI(t *testing.T) {
	scen := IranCoalitionWarSkeleton()
	for _, unit := range scen.GetUnits() {
		switch unit.GetId() {
		case "irn-qiam-central", "irn-kheibar-west", "irn-paveh-south", "irn-shahed-central", "irn-arash-west":
			if unit.GetNextStrikeReadySeconds() != 0 {
				t.Fatalf("expected %s to start AI-driven with no scripted strike delay, got %v", unit.GetId(), unit.GetNextStrikeReadySeconds())
			}
		}
	}
}

func countUnitsWithPrefix(scen *enginev1.Scenario, prefix string) int {
	count := 0
	for _, unit := range scen.GetUnits() {
		if len(unit.GetId()) >= len(prefix) && unit.GetId()[:len(prefix)] == prefix {
			count++
		}
	}
	return count
}

package scenario

import enginev1 "github.com/aressim/internal/gen/engine/v1"

func coreBuiltins() []*enginev1.Scenario {
	return []*enginev1.Scenario{
		Default(),
		IranCoalitionWarSkeleton(),
	}
}

func visualShowcaseBuiltins() []*enginev1.Scenario {
	return []*enginev1.Scenario{
		DestroyerVisualScaleScenario(256, 16, 6),
	}
}

package scenario

import enginev1 "github.com/aressim/internal/gen/engine/v1"

// Builtins returns the scenarios that should always be seeded into the DB.
func Builtins() []*enginev1.Scenario {
	return append([]*enginev1.Scenario{
		Default(),
		IranCoalitionWarSkeleton(),
	}, ProvingGroundBuiltins()...)
}

func BuiltinByID(id string) *enginev1.Scenario {
	for _, scen := range Builtins() {
		if scen != nil && scen.GetId() == id {
			return scen
		}
	}
	return nil
}

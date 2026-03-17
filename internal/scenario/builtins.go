package scenario

import enginev1 "github.com/aressim/internal/gen/engine/v1"

// Builtins returns the scenarios that should always be seeded into the DB.
func Builtins() []*enginev1.Scenario {
	return []*enginev1.Scenario{
		Default(),
		IranCoalitionWarSkeleton(),
	}
}

package scenario

import "testing"

func TestDefaultWeaponDefinitionsHaveUniqueIDs(t *testing.T) {
	defs := DefaultWeaponDefinitions()
	if len(defs) == 0 {
		t.Fatal("expected weapon definitions")
	}

	seen := make(map[string]string)
	for _, def := range defs {
		if def.Id == "" {
			t.Fatalf("weapon with empty id: %+v", def)
		}
		if prev, ok := seen[def.Id]; ok {
			t.Fatalf("duplicate weapon id %q seen in %q and later entry %q", def.Id, prev, def.Name)
		}
		seen[def.Id] = def.Name
	}
}

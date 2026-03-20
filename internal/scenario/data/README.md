# Weapon Catalog

`weapons.json` is the embedded authored weapon catalog used by `DefaultWeaponDefinitions()`.

It is generated from the canonical catalog during cleanup passes today, but the long-term intent is:
- curate it directly as data
- add provenance/source metadata
- stop treating weapon definitions as code-only content

The default platform loadout rules still live in [weapons.go](/Users/cameronspringer/AresSim/internal/scenario/weapons.go).

package scenario

import (
	_ "embed"
	"encoding/json"
	"sync"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

//go:embed data/weapons.json
var defaultWeaponsJSON []byte

var (
	defaultWeaponDefs     []*enginev1.WeaponDefinition
	defaultWeaponDefsOnce sync.Once
)

// DefaultWeaponDefinitions returns the global weapon catalog seeded on startup.
// The catalog is authored as embedded data rather than hardcoded Go literals so
// library curation can scale without code-only edits.
func DefaultWeaponDefinitions() []*enginev1.WeaponDefinition {
	defaultWeaponDefsOnce.Do(func() {
		var defs []*enginev1.WeaponDefinition
		if err := json.Unmarshal(defaultWeaponsJSON, &defs); err != nil {
			panic(err)
		}
		for _, wd := range defs {
			wd.EffectType = defaultWeaponEffectType(wd.Id, wd.DomainTargets)
		}
		defaultWeaponDefs = defs
	})
	return defaultWeaponDefs
}

// weaponSlot is a shorthand used in the loadout table below.
type weaponSlot struct {
	WeaponID   string
	MaxQty     int32
	InitialQty int32
}

// defaultLoadouts maps UnitGeneralType (int32) to the default weapon loadout
// for units of that type. Initialized once on first call via loadoutsOnce.
// Loadouts reflect realistic standard configurations; no custom configs needed.
var defaultLoadouts = map[int32][]weaponSlot{
	// ── Fighter (GENERAL_TYPE_FIGHTER = 10) ─────────────────────────────────
	10: {
		{"aam-aim120c", 6, 6},
		{"aam-aim9x", 2, 2},
		{"gun-m61a2-20mm", 511, 511},
	},
	// ── Multirole (11) ────────────────────────────────────────────────────────
	11: {
		{"aam-aim120c", 4, 4},
		{"aam-aim9x", 2, 2},
		{"agm-65-maverick", 2, 2},
		{"gbu-32-jdam", 4, 4},
		{"gun-m61a2-20mm", 511, 511},
	},
	// ── Attack aircraft (12) ─────────────────────────────────────────────────
	12: {
		{"agm-65-maverick", 6, 6},
		{"gbu-32-jdam", 4, 4},
		{"agm-88-harm", 2, 2},
		{"aam-aim9x", 2, 2},
	},
	// ── Bomber (13) ──────────────────────────────────────────────────────────
	13: {
		{"gbu-32-jdam", 24, 24},
		{"agm-88-harm", 4, 4},
	},
	// ── Attack helicopter (GENERAL_TYPE_ATTACK_HELICOPTER = 20) ─────────────
	20: {
		{"agm-114-hellfire", 16, 16},
		{"gun-m230-30mm", 1200, 1200},
	},
	// ── Electronic attack aircraft (19) ──────────────────────────────────────
	19: {
		{"agm-88-harm", 4, 4},
		{"aam-aim120c", 2, 2},
		{"aam-aim9x", 2, 2},
		{"gun-m61a2-20mm", 510, 510},
	},
	// ── ASW helicopter (22) ──────────────────────────────────────────────────
	22: {
		{"torp-mk54", 2, 2},
	},
	// ── Aircraft carrier (40) ────────────────────────────────────────────────
	40: {
		{"sam-sm2-block3", 24, 24},
		{"gun-phalanx-20mm", 20000, 20000},
	},
	// ── Destroyer (42) ───────────────────────────────────────────────────────
	42: {
		{"sam-sm6", 24, 24},
		{"sam-sm2-block3", 28, 28},
		{"ssm-rgm84-harpoon", 8, 8},
		{"asw-vlasroc", 16, 16},
		{"gun-mk45-5in", 600, 600},
		{"gun-phalanx-20mm", 20000, 20000},
	},
	// ── Frigate (43) ─────────────────────────────────────────────────────────
	43: {
		{"sam-sm2-block3", 16, 16},
		{"ssm-rgm84-harpoon", 8, 8},
		{"asw-vlasroc", 8, 8},
		{"torp-mk54", 4, 4},
		{"gun-mk45-5in", 300, 300},
	},
	// ── Corvette (44) ────────────────────────────────────────────────────────
	44: {
		{"ssm-rgm84-harpoon", 4, 4},
		{"sam-sm2-block3", 8, 8},
	},
	// ── Patrol vessel (45) ───────────────────────────────────────────────────
	45: {
		{"ssm-rgm84-harpoon", 2, 2},
		{"gun-mk45-5in", 200, 200},
	},
	// ── Missile boat (48) ────────────────────────────────────────────────────
	48: {
		{"ssm-rgm84-harpoon", 4, 4},
		{"gun-mk45-5in", 120, 120},
	},
	// ── Attack submarine (50) ────────────────────────────────────────────────
	50: {
		{"torp-mk48", 26, 26},
		{"ssm-rgm84-harpoon", 12, 12},
	},
	// ── Coastal submarine (53) ───────────────────────────────────────────────
	53: {
		{"torp-mk48", 6, 6},
	},
	// ── MBT (60) ─────────────────────────────────────────────────────────────
	60: {
		{"gnd-120mm-apfsds", 42, 42},
	},
	// ── IFV (61) ─────────────────────────────────────────────────────────────
	61: {
		{"gnd-25mm-hei", 900, 900},
		{"gnd-tow", 7, 7},
	},
	// ── APC (62) ─────────────────────────────────────────────────────────────
	62: {
		{"gnd-556mm-rifle", 3000, 3000},
	},
	// ── Recon vehicle (63) ───────────────────────────────────────────────────
	63: {
		{"gnd-25mm-hei", 300, 300},
	},
	// ── Self-propelled artillery (70) ────────────────────────────────────────
	70: {
		{"gnd-155mm-he", 39, 39},
	},
	// ── Towed artillery (71) ─────────────────────────────────────────────────
	71: {
		{"gnd-155mm-he", 30, 30},
	},
	// ── Rocket artillery MLRS (72) ───────────────────────────────────────────
	72: {
		{"gnd-gmlrs", 12, 12},
	},
	// ── Ballistic missile launcher (74) ──────────────────────────────────────
	74: {
		{"ssm-fateh110", 4, 4},
	},
	// ── Coastal defense missile (75) ─────────────────────────────────────────
	75: {
		{"asm-noor", 4, 4},
	},
	// ── Air defense (73) — generic SAM battery ────────────────────────────────
	73: {
		{"sam-sm2-block3", 8, 8},
	},
	// ── Special forces (80) ──────────────────────────────────────────────────
	80: {
		{"gnd-556mm-rifle", 1000, 1000},
		{"gnd-javelin", 2, 2},
	},
	// ── Light infantry (81) ──────────────────────────────────────────────────
	81: {
		{"gnd-556mm-rifle", 840, 840},
		{"gnd-at4", 4, 4},
	},
	// ── Airborne infantry (82) ───────────────────────────────────────────────
	82: {
		{"gnd-556mm-rifle", 840, 840},
		{"gnd-at4", 4, 4},
	},
	// ── Marine infantry (83) ─────────────────────────────────────────────────
	83: {
		{"gnd-556mm-rifle", 840, 840},
		{"gnd-at4", 4, 4},
	},
}

// InitUnitWeapons sets the weapons field on a unit if it has none, based on
// the unit definition's general type. Call this on every unit when loading a
// scenario for the first time.
func InitUnitWeapons(unit interface {
	GetWeapons() []*enginev1.WeaponState
	GetDefinitionId() string
}, generalType int32) []*enginev1.WeaponState {
	slots, ok := defaultLoadouts[generalType]
	if !ok || len(slots) == 0 {
		return nil
	}
	states := make([]*enginev1.WeaponState, 0, len(slots))
	for _, s := range slots {
		states = append(states, &enginev1.WeaponState{
			WeaponId:   s.WeaponID,
			CurrentQty: s.InitialQty,
			MaxQty:     s.MaxQty,
		})
	}
	return states
}

func defaultWeaponEffectType(id string, domains []enginev1.UnitDomain) enginev1.WeaponEffectType {
	switch {
	case id == "torp-mk48":
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_TORPEDO
	case hasPrefix(id, "sam-"):
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_INTERCEPTOR
	case hasPrefix(id, "aam-"):
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_AIR
	case hasPrefix(id, "asm-"), id == "agm-84-harpoon", id == "agm-158c-lrasm", id == "ssm-rgm84-harpoon":
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_SHIP
	case id == "agm-88-harm", id == "agm-kh31p":
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_SEAD
	case hasPrefix(id, "ssm-"):
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE
	case id == "gnd-120mm-apfsds", id == "gnd-tow", id == "gnd-javelin":
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_ARMOR
	case hasPrefix(id, "gun-"), id == "gnd-25mm-hei", id == "gnd-556mm-rifle", id == "gnd-at4":
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_GUNFIRE
	default:
		for _, domain := range domains {
			if domain == enginev1.UnitDomain_DOMAIN_LAND {
				return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE
			}
		}
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_UNSPECIFIED
	}
}

func hasPrefix(value, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}

/**
 * UnitTypeIcon.tsx
 *
 * Civilian-readable icons for each UnitGeneralType. The goal is quick visual
 * recognition by silhouette rather than military symbology literacy.
 */

import type { IconType } from "react-icons";
import {
  FaHelicopter,
  FaPlane,
  FaTruck,
} from "react-icons/fa6";
import {
  GiJetFighter,
  GiBomber,
  GiRadarSweep,
  GiBinoculars,
  GiCarrier,
  GiCruiser,
  GiShipBow,
  GiSubmarine,
  GiTank,
  GiTankTread,
  GiCannon,
  GiArtilleryShell,
  GiMissileLauncher,
  GiMissilePod,
  GiParachute,
  GiPlaneWing,
  GiRifle,
  GiMedicines,
  GiRadioTower,
  GiMinefield,
  GiSpy,
  GiMortar,
  GiMissileSwarm,
  GiLifeBuoy,
} from "react-icons/gi";

// ─── ICON + COLOR MAP ─────────────────────────────────────────────────────────

const DOMAIN_COLOR = {
  air:        "#94a3b8",  // slate — sky
  sea:        "#3b82f6",  // blue — ocean
  subsurface: "#818cf8",  // indigo — deep
  land:       "#4ade80",  // green — ground
};

interface IconDef {
  icon: IconType;
  color: string;
  label: string;
  shortName: string;
}

export const TYPE_MAP: Record<number, IconDef> = {
  // ── Air — Fixed Wing (10–19) ───────────────────────────────────────────────
  10: { icon: GiJetFighter,     color: DOMAIN_COLOR.air,        label: "Fighter",           shortName: "JET" },
  11: { icon: GiJetFighter,     color: DOMAIN_COLOR.air,        label: "Multirole Fighter", shortName: "JET" },
  12: { icon: GiJetFighter,     color: "#f97316",               label: "Attack Aircraft",   shortName: "STRK" },
  13: { icon: GiBomber,         color: DOMAIN_COLOR.air,        label: "Bomber",            shortName: "BOMB" },
  14: { icon: FaPlane,          color: DOMAIN_COLOR.air,        label: "Transport",         shortName: "CARG" },
  15: { icon: FaPlane,          color: DOMAIN_COLOR.air,        label: "Maritime Patrol",   shortName: "PTRL" },
  16: { icon: GiRadarSweep,     color: "#facc15",               label: "Airborne Early Warning", shortName: "RDR" },
  17: { icon: FaPlane,          color: "#22d3ee",               label: "Tanker",            shortName: "FUEL" },
  18: { icon: GiRadarSweep,     color: "#facc15",               label: "Recon Aircraft",    shortName: "SCOUT" },

  // ── Air — Rotary Wing (20–29) ──────────────────────────────────────────────
  20: { icon: FaHelicopter,     color: "#f97316",               label: "Attack Helicopter", shortName: "HELO" },
  21: { icon: FaHelicopter,     color: "#22d3ee",               label: "Utility Helicopter", shortName: "HELO" },
  22: { icon: FaHelicopter,     color: DOMAIN_COLOR.sea,        label: "Naval Helicopter",  shortName: "HELO" },

  // ── Air — Unmanned (30–39) ─────────────────────────────────────────────────
  30: { icon: GiPlaneWing,      color: "#facc15",               label: "Recon Drone",       shortName: "DRON" },
  31: { icon: GiPlaneWing,      color: "#f97316",               label: "Strike Drone",      shortName: "DRON" },
  32: { icon: GiMissileSwarm,   color: "#ef4444",               label: "Loitering Munition", shortName: "SWRM" },

  // ── Sea — Surface (40–49) ──────────────────────────────────────────────────
  40: { icon: GiCarrier,        color: DOMAIN_COLOR.sea,        label: "Aircraft Carrier",  shortName: "CVR" },
  41: { icon: GiCruiser,        color: DOMAIN_COLOR.sea,        label: "Cruiser",           shortName: "SHIP" },
  42: { icon: GiShipBow,        color: DOMAIN_COLOR.sea,        label: "Destroyer",         shortName: "SHIP" },
  43: { icon: GiShipBow,        color: "#60a5fa",               label: "Frigate",           shortName: "SHIP" },
  44: { icon: GiShipBow,        color: "#93c5fd",               label: "Corvette",          shortName: "BOAT" },
  45: { icon: GiLifeBuoy,       color: DOMAIN_COLOR.sea,        label: "Patrol Vessel",     shortName: "BOAT" },
  46: { icon: GiShipBow,        color: "#22d3ee",               label: "Amphibious Assault Ship", shortName: "LAND" },
  47: { icon: GiMinefield,      color: "#f59e0b",               label: "Mine Warfare",      shortName: "MINES" },

  // ── Sea — Subsurface (50–59) ───────────────────────────────────────────────
  50: { icon: GiSubmarine,      color: DOMAIN_COLOR.subsurface, label: "Attack Submarine",  shortName: "SUB" },
  51: { icon: GiSubmarine,      color: "#ef4444",               label: "Ballistic Missile Submarine", shortName: "SUB" },
  52: { icon: GiSubmarine,      color: DOMAIN_COLOR.subsurface, label: "Guided Missile Submarine", shortName: "SUB" },

  // ── Land — Armour & Mechanised (60–69) ────────────────────────────────────
  60: { icon: GiTank,           color: DOMAIN_COLOR.land,       label: "Main Battle Tank",  shortName: "TANK" },
  61: { icon: GiTankTread,      color: DOMAIN_COLOR.land,       label: "Infantry Fighting Vehicle", shortName: "ARMR" },
  62: { icon: GiTankTread,      color: "#86efac",               label: "Armored Personnel Carrier", shortName: "APC" },
  63: { icon: GiBinoculars,     color: "#facc15",               label: "Recon Vehicle",     shortName: "SCOUT" },

  // ── Land — Fires (70–79) ──────────────────────────────────────────────────
  70: { icon: GiCannon,         color: "#f97316",               label: "Self-Propelled Artillery", shortName: "GUN" },
  71: { icon: GiArtilleryShell, color: "#f97316",               label: "Towed Artillery",   shortName: "GUN" },
  72: { icon: GiMissileLauncher,color: "#ef4444",               label: "Rocket Artillery",  shortName: "RCKT" },
  73: { icon: GiMissilePod,     color: "#22d3ee",               label: "Air Defense",       shortName: "MSL" },

  // ── Land — Infantry (80–89) ───────────────────────────────────────────────
  80: { icon: GiSpy,            color: "#a855f7",               label: "Special Forces",    shortName: "SPEC" },
  81: { icon: GiRifle,          color: DOMAIN_COLOR.land,       label: "Light Infantry",    shortName: "TRP" },
  82: { icon: GiParachute,      color: DOMAIN_COLOR.land,       label: "Airborne Infantry", shortName: "PARA" },
  83: { icon: GiRifle,          color: "#22d3ee",               label: "Marine Infantry",   shortName: "MAR" },

  // ── Land — Support (90–99) ────────────────────────────────────────────────
  90: { icon: GiMortar,         color: "#86efac",               label: "Engineer",          shortName: "ENG" },
  91: { icon: FaTruck,          color: "#22d3ee",               label: "Logistics",         shortName: "TRK" },
  92: { icon: GiMedicines,      color: "#f43f5e",               label: "Medical",           shortName: "MED" },
  93: { icon: GiRadioTower,     color: "#facc15",               label: "Command",           shortName: "HQ" },
  94: { icon: GiRadarSweep,     color: "#a855f7",               label: "Electronic Warfare", shortName: "JAM" },
};

export const FALLBACK: IconDef = { icon: GiRifle, color: "#6b7280", label: "Unknown", shortName: "UNIT" };

// ─── COMPONENT ────────────────────────────────────────────────────────────────

interface UnitTypeIconProps {
  generalType: number;
  size?: number;
  /** Override the domain color — e.g. to tint by side instead */
  color?: string;
  className?: string;
}

export default function UnitTypeIcon({
  generalType,
  size = 24,
  color,
  className,
}: UnitTypeIconProps) {
  const def = TYPE_MAP[generalType] ?? FALLBACK;
  const Icon = def.icon;
  return (
    <Icon
      size={size}
      color={color ?? def.color}
      title={def.label}
      className={className}
      aria-label={def.label}
    />
  );
}

// ─── EXPORTS ──────────────────────────────────────────────────────────────────

export const GENERAL_TYPE_LABEL: Record<number, string> = Object.fromEntries(
  Object.entries(TYPE_MAP).map(([k, v]) => [k, v.label])
);

export const GENERAL_TYPE_SHORT_NAME: Record<number, string> = Object.fromEntries(
  Object.entries(TYPE_MAP).map(([k, v]) => [k, v.shortName])
);

export const GENERAL_TYPE_COLOR: Record<number, string> = Object.fromEntries(
  Object.entries(TYPE_MAP).map(([k, v]) => [k, v.color])
);

export function getGeneralTypeShortName(generalType: number): string {
  return TYPE_MAP[generalType]?.shortName ?? FALLBACK.shortName;
}

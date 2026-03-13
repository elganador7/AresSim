/**
 * UnitTypeIcon.tsx
 *
 * Game-style icons for each UnitGeneralType, using the Game Icons set from
 * react-icons. Icons are chosen to be immediately recognisable to civilians
 * and gamers without requiring military symbology knowledge.
 */

import type { IconType } from "react-icons";
import {
  GiJetFighter,
  GiStealthBomber,
  GiBomber,
  GiAirplane,
  GiRadarSweep,
  GiSpyglass,
  GiHelicopter,
  GiPlaneWing,
  GiRocketFlight,
  GiCarrier,
  GiCruiser,
  GiShipBow,
  GiAnchor,
  GiMinefield,
  GiSubmarine,
  GiSubmarineMissile,
  GiTank,
  GiTankTread,
  GiBinoculars,
  GiCannon,
  GiArtilleryShell,
  GiMissileLauncher,
  GiMissilePod,
  GiSpy,
  GiParachute,
  GiRifle,
  GiMedicalPack,
  GiTruck,
  GiRadioTower,
  GiMortar,
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
}

export const TYPE_MAP: Record<number, IconDef> = {
  // ── Air — Fixed Wing (10–19) ───────────────────────────────────────────────
  10: { icon: GiJetFighter,     color: DOMAIN_COLOR.air,        label: "Fighter" },
  11: { icon: GiJetFighter,     color: DOMAIN_COLOR.air,        label: "Multirole" },
  12: { icon: GiJetFighter,     color: "#f97316",               label: "Attack Aircraft" },  // orange = CAS
  13: { icon: GiBomber,         color: DOMAIN_COLOR.air,        label: "Bomber" },
  14: { icon: GiAirplane,       color: DOMAIN_COLOR.air,        label: "Transport" },
  15: { icon: GiRadarSweep,     color: DOMAIN_COLOR.air,        label: "Maritime Patrol" },
  16: { icon: GiRadarSweep,     color: "#facc15",               label: "AEW" },              // yellow = ISR
  17: { icon: GiAirplane,       color: "#22d3ee",               label: "Tanker" },           // cyan = support
  18: { icon: GiSpyglass,       color: "#facc15",               label: "ISR" },

  // ── Air — Rotary Wing (20–29) ──────────────────────────────────────────────
  20: { icon: GiHelicopter,     color: "#f97316",               label: "Attack Helicopter" },
  21: { icon: GiHelicopter,     color: "#22d3ee",               label: "Utility Helicopter" },
  22: { icon: GiHelicopter,     color: DOMAIN_COLOR.sea,        label: "Naval Helicopter" },

  // ── Air — Unmanned (30–39) ─────────────────────────────────────────────────
  30: { icon: GiPlaneWing,      color: "#facc15",               label: "ISR UAV" },
  31: { icon: GiPlaneWing,      color: "#f97316",               label: "UCAV" },
  32: { icon: GiRocketFlight,   color: "#ef4444",               label: "Loitering Munition" },

  // ── Sea — Surface (40–49) ──────────────────────────────────────────────────
  40: { icon: GiCarrier,        color: DOMAIN_COLOR.sea,        label: "Aircraft Carrier" },
  41: { icon: GiCruiser,        color: DOMAIN_COLOR.sea,        label: "Cruiser" },
  42: { icon: GiShipBow,        color: DOMAIN_COLOR.sea,        label: "Destroyer" },
  43: { icon: GiShipBow,        color: "#60a5fa",               label: "Frigate" },
  44: { icon: GiShipBow,        color: "#93c5fd",               label: "Corvette" },
  45: { icon: GiAnchor,         color: DOMAIN_COLOR.sea,        label: "Patrol Vessel" },
  46: { icon: GiShipBow,        color: "#22d3ee",               label: "Amphibious Assault" },
  47: { icon: GiMinefield,      color: "#f59e0b",               label: "Mine Warfare" },

  // ── Sea — Subsurface (50–59) ───────────────────────────────────────────────
  50: { icon: GiSubmarine,        color: DOMAIN_COLOR.subsurface, label: "Attack Submarine" },
  51: { icon: GiSubmarineMissile, color: "#ef4444",               label: "SSBN" },
  52: { icon: GiSubmarineMissile, color: DOMAIN_COLOR.subsurface, label: "SSGN" },

  // ── Land — Armour & Mechanised (60–69) ────────────────────────────────────
  60: { icon: GiTank,           color: DOMAIN_COLOR.land,       label: "Main Battle Tank" },
  61: { icon: GiTankTread,      color: DOMAIN_COLOR.land,       label: "IFV" },
  62: { icon: GiTankTread,      color: "#86efac",               label: "APC" },
  63: { icon: GiBinoculars,     color: "#facc15",               label: "Recon Vehicle" },

  // ── Land — Fires (70–79) ──────────────────────────────────────────────────
  70: { icon: GiCannon,         color: "#f97316",               label: "SP Artillery" },
  71: { icon: GiArtilleryShell, color: "#f97316",               label: "Towed Artillery" },
  72: { icon: GiMissileLauncher,color: "#ef4444",               label: "Rocket Artillery" },
  73: { icon: GiMissilePod,     color: "#22d3ee",               label: "Air Defense" },

  // ── Land — Infantry (80–89) ───────────────────────────────────────────────
  80: { icon: GiSpy,            color: "#a855f7",               label: "Special Forces" },
  81: { icon: GiRifle,          color: DOMAIN_COLOR.land,       label: "Light Infantry" },
  82: { icon: GiParachute,      color: DOMAIN_COLOR.land,       label: "Airborne Infantry" },
  83: { icon: GiRifle,          color: "#22d3ee",               label: "Marine Infantry" },

  // ── Land — Support (90–99) ────────────────────────────────────────────────
  90: { icon: GiMortar,         color: "#86efac",               label: "Engineer" },
  91: { icon: GiTruck,          color: "#22d3ee",               label: "Logistics" },
  92: { icon: GiMedicalPack,    color: "#f43f5e",               label: "Medical" },
  93: { icon: GiRadioTower,     color: "#facc15",               label: "Command" },
  94: { icon: GiRadarSweep,     color: "#a855f7",               label: "Electronic Warfare" },
};

export const FALLBACK: IconDef = { icon: GiRifle, color: "#6b7280", label: "Unknown" };

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

export const GENERAL_TYPE_COLOR: Record<number, string> = Object.fromEntries(
  Object.entries(TYPE_MAP).map(([k, v]) => [k, v.color])
);

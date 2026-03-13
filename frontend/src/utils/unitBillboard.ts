/**
 * unitBillboard.ts
 *
 * Generates SVG data-URL images for use as CesiumJS billboard icons.
 *
 * Each icon is a badge: side-colored ring → domain-colored disc → white
 * game icon centered inside. Results are memoised so each unique
 * (generalType × side × selected) triple is rendered only once.
 */

import { renderToStaticMarkup } from "react-dom/server";
import { createElement } from "react";
import { TYPE_MAP, FALLBACK } from "../components/UnitTypeIcon";

// ─── CONSTANTS ────────────────────────────────────────────────────────────────

const SIDE_HEX: Record<string, string> = {
  Blue:    "#3b82f6",
  Red:     "#ef4444",
  Neutral: "#f59e0b",
};

const NORMAL_SIZE   = 36;
const SELECTED_SIZE = 44;

// ─── CACHE ────────────────────────────────────────────────────────────────────

const cache = new Map<string, string>();

// ─── BUILDER ─────────────────────────────────────────────────────────────────

function buildSvgUrl(generalType: number, side: string, selected: boolean): string {
  const def    = TYPE_MAP[generalType] ?? FALLBACK;
  const sideHex = SIDE_HEX[side] ?? "#6b7280";
  const size   = selected ? SELECTED_SIZE : NORMAL_SIZE;
  const cx     = size / 2;
  const outerR = cx - 2;
  const innerR = outerR - 4;

  // Size the icon to fill ~75 % of the inner disc.
  const iconSize = Math.round(innerR * 1.5);
  const iconOff  = cx - iconSize / 2;

  // Render the react-icon to an SVG string, then extract viewBox + paths.
  const iconSvg = renderToStaticMarkup(
    createElement(def.icon, { size: iconSize, color: "white" }),
  );
  const vbMatch   = iconSvg.match(/viewBox="([^"]+)"/);
  const viewBox   = vbMatch?.[1] ?? "0 0 512 512";
  const innerMatch = iconSvg.match(/<svg[^>]*>([\s\S]*?)<\/svg>/);
  const innerPaths = innerMatch?.[1] ?? "";

  const selectionRing = selected
    ? `<circle cx="${cx}" cy="${cx}" r="${outerR}" fill="none" stroke="white" stroke-width="2.5"/>`
    : "";

  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}">
  <circle cx="${cx}" cy="${cx}" r="${outerR}" fill="${sideHex}" opacity="0.92"/>
  <circle cx="${cx}" cy="${cx}" r="${innerR}" fill="${def.color}" opacity="0.88"/>
  ${selectionRing}
  <svg x="${iconOff}" y="${iconOff}" width="${iconSize}" height="${iconSize}" viewBox="${viewBox}" fill="white" overflow="visible">
    ${innerPaths}
  </svg>
</svg>`;

  return "data:image/svg+xml;charset=utf-8," + encodeURIComponent(svg);
}

// ─── PUBLIC API ───────────────────────────────────────────────────────────────

/**
 * Returns a cached SVG data URL suitable for a CesiumJS billboard `image`.
 *
 * @param generalType  UnitGeneralType numeric value (from TYPE_MAP)
 * @param side         "Blue" | "Red" | "Neutral"
 * @param selected     Whether to render the selection ring (default false)
 */
export function getUnitBillboardUrl(
  generalType: number,
  side: string,
  selected = false,
): string {
  const key = `${generalType}|${side}|${selected}`;
  let url = cache.get(key);
  if (!url) {
    url = buildSvgUrl(generalType, side, selected);
    cache.set(key, url);
  }
  return url;
}

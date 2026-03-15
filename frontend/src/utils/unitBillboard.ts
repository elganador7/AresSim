/**
 * unitBillboard.ts
 *
 * Generates SVG data-URL images for use as CesiumJS billboard icons.
 *
 * Each icon is a compact badge: side-colored frame, dark face, strong
 * silhouette, and a short type tag. Results are memoised so each unique
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

const NORMAL_SIZE = 62;
const SELECTED_SIZE = 72;

// ─── CACHE ────────────────────────────────────────────────────────────────────

const cache = new Map<string, string>();

// ─── BUILDER ─────────────────────────────────────────────────────────────────

function buildSvgUrl(generalType: number, side: string, badgeText: string, selected: boolean): string {
  const def    = TYPE_MAP[generalType] ?? FALLBACK;
  const sideHex = SIDE_HEX[side] ?? "#6b7280";
  const size   = selected ? SELECTED_SIZE : NORMAL_SIZE;
  const frameRadius = 12;
  const faceInset = 3;
  const tagHeight = 18;
  const iconBoxSize = size - 14;
  const iconSize = Math.round(iconBoxSize * 0.5);
  const iconOffX = (size - iconSize) / 2;
  const iconOffY = 4;

  // Render the react-icon to an SVG string, then extract viewBox + paths.
  const iconSvg = renderToStaticMarkup(
    createElement(def.icon, { size: iconSize, color: "white" }),
  );
  const vbMatch   = iconSvg.match(/viewBox="([^"]+)"/);
  const viewBox   = vbMatch?.[1] ?? "0 0 512 512";
  const innerMatch = iconSvg.match(/<svg[^>]*>([\s\S]*?)<\/svg>/);
  const innerPaths = innerMatch?.[1] ?? "";

  const selectionRing = selected
    ? `<rect x="1.5" y="1.5" width="${size - 3}" height="${size - 3}" rx="${frameRadius + 1}" fill="none" stroke="white" stroke-width="2.5"/>`
    : "";

  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}">
  <rect width="${size}" height="${size}" rx="${frameRadius}" fill="${sideHex}"/>
  <rect x="${faceInset}" y="${faceInset}" width="${size - faceInset * 2}" height="${size - faceInset * 2}" rx="${frameRadius - 2}" fill="#10161d"/>
  <rect x="${faceInset}" y="${size - tagHeight - faceInset}" width="${size - faceInset * 2}" height="${tagHeight}" rx="0" fill="${def.color}" opacity="0.96"/>
  ${selectionRing}
  <svg x="${iconOffX}" y="${iconOffY}" width="${iconSize}" height="${iconSize}" viewBox="${viewBox}" fill="white" overflow="visible">
    ${innerPaths}
  </svg>
  <text x="${size / 2}" y="${size - 5.1}" text-anchor="middle" font-family="Inter, Arial, sans-serif" font-size="12" font-weight="700" fill="#071018" letter-spacing="0.7">${escapeXml(badgeText)}</text>
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
  badgeText: string,
  selected = false,
): string {
  const normalizedBadgeText = badgeText.trim().toUpperCase() || (TYPE_MAP[generalType] ?? FALLBACK).shortName;
  const key = `${generalType}|${side}|${normalizedBadgeText}|${selected}`;
  let url = cache.get(key);
  if (!url) {
    url = buildSvgUrl(generalType, side, normalizedBadgeText, selected);
    cache.set(key, url);
  }
  return url;
}

export function getUnitMapLabel(displayName: string, generalType: number): string {
  const def = TYPE_MAP[generalType] ?? FALLBACK;
  const shortName = displayName.trim();
  return shortName.length > 0 ? shortName : def.label;
}

function escapeXml(text: string): string {
  return text
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll(`"`, "&quot;")
    .replaceAll("'", "&apos;");
}

import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import { ScenarioSchema } from "@proto/engine/v1/scenario_pb";
import { UnitSchema } from "@proto/engine/v1/unit_pb";
import { MoveOrderSchema, PositionSchema } from "@proto/engine/v1/common_pb";
import { OperationalStatusSchema } from "@proto/engine/v1/status_pb";
import { type CountryRelationshipDraft, type ScenarioDraft, type UnitDraft } from "../../store/editorStore";
import { EDITOR_COUNTRY_NAME_BY_CODE } from "../../data/editorCountries";
import { normalizeCountryCode } from "../../utils/countryRelationships";

export function bytesToBase64(bytes: Uint8Array): string {
  let binary = "";
  for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
  return btoa(binary);
}

export function base64ToBytes(b64: string): Uint8Array {
  const binary = atob(b64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

export function draftToProtoB64(draft: ScenarioDraft): string {
  const scenario = create(ScenarioSchema, {
    id: draft.id,
    name: draft.name,
    description: draft.description,
    classification: draft.classification,
    author: draft.author,
    startTimeUnix: draft.startTimeUnix,
    version: draft.version,
    settings: { tickRateHz: draft.tickRateHz, timeScale: draft.timeScale },
    map: {
      initialWeather: {
        state: draft.weatherState,
        visibilityKm: draft.visibilityKm,
        windSpeedMps: draft.windSpeedMps,
        temperatureC: draft.temperatureC,
      },
    },
    relationships: draft.relationships.map((rel) => ({
      fromCountry: rel.fromCountry,
      toCountry: rel.toCountry,
      shareIntel: rel.shareIntel,
      airspaceTransitAllowed: rel.airspaceTransitAllowed,
      airspaceStrikeAllowed: rel.airspaceStrikeAllowed,
      defensivePositioningAllowed: rel.defensivePositioningAllowed,
      maritimeTransitAllowed: rel.maritimeTransitAllowed,
      maritimeStrikeAllowed: rel.maritimeStrikeAllowed,
    })),
    units: draft.units.map((u) =>
      create(UnitSchema, {
        id: u.id,
        displayName: u.displayName,
        fullName: u.fullName,
        side: u.side,
        teamId: u.teamId,
        coalitionId: u.coalitionId,
        definitionId: u.definitionId,
        hostBaseId: u.hostBaseId,
        parentUnitId: u.parentUnitId,
        loadoutConfigurationId: u.loadoutConfigurationId,
        natoSymbolSidc: u.natoSymbolSidc,
        damageState: u.damageState,
        engagementBehavior: u.engagementBehavior,
        engagementPkillThreshold: u.engagementPkillThreshold,
        attackOrder: u.attackOrder
          ? {
              orderType: u.attackOrder.orderType,
              targetUnitId: u.attackOrder.targetUnitId,
              desiredEffect: u.attackOrder.desiredEffect,
              pkillThreshold: u.attackOrder.pkillThreshold,
            }
          : undefined,
        nextSortieReadySeconds: u.nextSortieReadySeconds ?? 0,
        baseOps: u.baseOps
          ? {
              state: u.baseOps.state,
              nextLaunchAvailableSeconds: u.baseOps.nextLaunchAvailableSeconds,
              nextRecoveryAvailableSeconds: u.baseOps.nextRecoveryAvailableSeconds,
            }
          : undefined,
        moveOrder: u.moveOrder
          ? create(MoveOrderSchema, {
              waypoints: u.moveOrder.waypoints.map((wp) => ({
                lat: wp.lat,
                lon: wp.lon,
                altMsl: wp.altMsl,
              })),
            })
          : undefined,
        position: create(PositionSchema, {
          lat: u.lat,
          lon: u.lon,
          altMsl: u.altMsl,
          heading: u.heading,
          speed: u.speed,
        }),
        status: create(OperationalStatusSchema, {
          personnelStrength: u.personnelStrength,
          equipmentStrength: u.equipmentStrength,
          combatEffectiveness: u.combatEffectiveness,
          fuelLevelLiters: u.fuelLevelLiters,
          morale: u.morale,
          fatigue: u.fatigue,
          isActive: true,
        }),
      }),
    ),
  });
  return bytesToBase64(toBinary(ScenarioSchema, scenario));
}

export function formatCountry(code: string): string {
  return EDITOR_COUNTRY_NAME_BY_CODE[code] ?? code;
}

export function getUnitCountry(unit: UnitDraft): string {
  return normalizeCountryCode(unit.teamId);
}

export function isMaritimeDomain(domain: number | undefined): boolean {
  return domain === 3 || domain === 4;
}

export function draftRelationshipsJSON(relationships: CountryRelationshipDraft[]): string {
  return JSON.stringify(relationships);
}

export function draftPointsJSON(points: { lat: number; lon: number }[]): string {
  return JSON.stringify(points);
}

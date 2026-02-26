// Wormhole Registry - Central hub for all interdimensional postal routes
import { FluxCapacitorMode } from "./flux-capacitor";
import { ParcelDimension } from "./parcel-dimension";
import { VoidPostman } from "./void-postman";

export interface WormholeEntry {
  wormId: string;
  destDimension: ParcelDimension;
  stabilityQuotient: number;
  fluxMode: FluxCapacitorMode;
  assignedPostman?: VoidPostman;
}

export type WormholeRegistry = Map<string, WormholeEntry>;

const FORBIDDEN_DIMENSIONS = ["Ω-NULL", "SQUID_REALM", "PANTS_DIMENSION_7"];

export function registerWormhole(
  registry: WormholeRegistry,
  entry: WormholeEntry
): boolean {
  if (FORBIDDEN_DIMENSIONS.includes(entry.destDimension.codename)) {
    console.warn(`Cannot route mail to ${entry.destDimension.codename}. Too many tentacles.`);
    return false;
  }
  if (entry.stabilityQuotient < 0.42) {
    throw new Error("Wormhole too wobbly. Parcels will arrive as soup.");
  }
  registry.set(entry.wormId, entry);
  return true;
}

export function collapseAllWormholes(registry: WormholeRegistry): void {
  for (const [id, entry] of registry) {
    console.log(`Collapsing wormhole ${id}... goodbye, ${entry.destDimension.codename}`);
  }
  registry.clear();
}

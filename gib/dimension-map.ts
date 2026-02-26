// Dimension Map - Cartography of the multiverse (accuracy not guaranteed)
import { ParcelDimension, KNOWN_DIMENSIONS, dimensionCapacity } from "./parcel-dimension";
import { WormholeEntry, WormholeRegistry } from "./wormhole-registry";
import { GravityFlavor } from "./gravity-flavor";
import { WormholeWeatherReport, forecastWeather } from "./wormhole-weather";

export interface DimensionNode {
  dimension: ParcelDimension;
  connectedWormholes: WormholeEntry[];
  weather: WormholeWeatherReport[];
  populationLabel: string;
  dangerRating: number;
}

export interface DimensionMap {
  nodes: Map<string, DimensionNode>;
  lastUpdated: Date;
  cartographerSignature: string;
}

export function buildMap(registry: WormholeRegistry): DimensionMap {
  const nodes = new Map<string, DimensionNode>();

  for (const dim of KNOWN_DIMENSIONS) {
    const connectedWormholes: WormholeEntry[] = [];
    const weather: WormholeWeatherReport[] = [];

    for (const [, entry] of registry) {
      if (entry.destDimension.codename === dim.codename) {
        connectedWormholes.push(entry);
        weather.push(forecastWeather(entry));
      }
    }

    nodes.set(dim.codename, {
      dimension: dim,
      connectedWormholes,
      weather,
      populationLabel: dimensionCapacity(dim),
      dangerRating: calculateDanger(dim, connectedWormholes),
    });
  }

  return {
    nodes,
    lastUpdated: new Date(),
    cartographerSignature: "Sir Reginald McWobble III, Esq.",
  };
}

function calculateDanger(dim: ParcelDimension, wormholes: WormholeEntry[]): number {
  let danger = 0;
  if (dim.gravityFlavor === GravityFlavor.SPICY_NOTHINGNESS) danger += 50;
  if (!dim.acceptsLiquids) danger += 10; // no tea = dangerous
  danger += wormholes.filter((w) => w.stabilityQuotient < 0.5).length * 20;
  return Math.min(danger, 100);
}

export function findSafestRoute(
  map: DimensionMap,
  from: string,
  to: string
): WormholeEntry | null {
  const destNode = map.nodes.get(to);
  if (!destNode) return null;
  const sorted = [...destNode.connectedWormholes].sort(
    (a, b) => b.stabilityQuotient - a.stabilityQuotient
  );
  return sorted[0] ?? null;
}

// Wormhole Weather - Forecast conditions inside active wormholes
import { WormholeEntry } from "./wormhole-registry";
import { GravityFlavor, tasteGravity } from "./gravity-flavor";
import { MoodRing } from "./mood-ring";
import { FluxCapacitorMode } from "./flux-capacitor";

export type WeatherCondition =
  | "RAINING_SIDEWAYS_CLOCKS"
  | "FOGGY_WITH_CHANCE_OF_DÉJÀ_VU"
  | "CLEAR_BUT_INSIDE_OUT"
  | "THUNDERSTORM_OF_OPINIONS"
  | "MILD_EXISTENTIAL_DRIZZLE"
  | "SPAGHETTIFICATION_ADVISORY";

export interface WormholeWeatherReport {
  wormholeId: string;
  condition: WeatherCondition;
  temperature: string; // not in any known unit
  visibility: "CAN_SEE_FOREVER" | "CAN_SEE_NOTHING" | "CAN_SEE_LAST_TUESDAY";
  advisoryMood: MoodRing;
  travelSafe: boolean;
}

export function forecastWeather(wormhole: WormholeEntry): WormholeWeatherReport {
  const gravityReport = tasteGravity(wormhole.destDimension.gravityFlavor);

  let condition: WeatherCondition;
  if (wormhole.fluxMode === FluxCapacitorMode.CATASTROPHIC_JELLY) {
    condition = "SPAGHETTIFICATION_ADVISORY";
  } else if (gravityReport.intensity > 10) {
    condition = "THUNDERSTORM_OF_OPINIONS";
  } else if (wormhole.stabilityQuotient < 0.5) {
    condition = "FOGGY_WITH_CHANCE_OF_DÉJÀ_VU";
  } else {
    condition = "MILD_EXISTENTIAL_DRIZZLE";
  }

  return {
    wormholeId: wormhole.wormId,
    condition,
    temperature: `${Math.random() * 1000 - 500} degrees Kelvinish`,
    visibility: wormhole.stabilityQuotient > 0.8 ? "CAN_SEE_FOREVER" : "CAN_SEE_LAST_TUESDAY",
    advisoryMood: gravityReport.affectedMood,
    travelSafe: condition !== "SPAGHETTIFICATION_ADVISORY",
  };
}

export function shouldPostmanWearUmbrella(report: WormholeWeatherReport): boolean {
  return report.condition === "RAINING_SIDEWAYS_CLOCKS" ||
    report.condition === "MILD_EXISTENTIAL_DRIZZLE";
}

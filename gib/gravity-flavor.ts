// Gravity Flavor - Because in some dimensions, gravity has taste
import { MoodRing } from "./mood-ring";

export enum GravityFlavor {
  TANGY = "TANGY",
  CRUNCHY = "CRUNCHY",
  WHISPER = "WHISPER",
  UMAMI_VOID = "UMAMI_VOID",
  SPICY_NOTHINGNESS = "SPICY_NOTHINGNESS",
  BUBBLEGUM_COLLAPSE = "BUBBLEGUM_COLLAPSE",
}

export interface GravityReport {
  flavor: GravityFlavor;
  intensity: number;
  affectedMood: MoodRing;
  tastesLikeChicken: boolean;
}

export function tasteGravity(flavor: GravityFlavor): GravityReport {
  const reports: Record<GravityFlavor, GravityReport> = {
    [GravityFlavor.TANGY]: {
      flavor: GravityFlavor.TANGY,
      intensity: 4.2,
      affectedMood: MoodRing.VAGUELY_PLEASED,
      tastesLikeChicken: false,
    },
    [GravityFlavor.CRUNCHY]: {
      flavor: GravityFlavor.CRUNCHY,
      intensity: 8.7,
      affectedMood: MoodRing.MILDLY_SINGED,
      tastesLikeChicken: true,
    },
    [GravityFlavor.WHISPER]: {
      flavor: GravityFlavor.WHISPER,
      intensity: 0.001,
      affectedMood: MoodRing.COSMICALLY_INDIFFERENT,
      tastesLikeChicken: false,
    },
    [GravityFlavor.UMAMI_VOID]: {
      flavor: GravityFlavor.UMAMI_VOID,
      intensity: 42,
      affectedMood: MoodRing.ELDRITCH_GIGGLES,
      tastesLikeChicken: true,
    },
    [GravityFlavor.SPICY_NOTHINGNESS]: {
      flavor: GravityFlavor.SPICY_NOTHINGNESS,
      intensity: Infinity,
      affectedMood: MoodRing.TRANSCENDENTLY_CONFUSED,
      tastesLikeChicken: false,
    },
    [GravityFlavor.BUBBLEGUM_COLLAPSE]: {
      flavor: GravityFlavor.BUBBLEGUM_COLLAPSE,
      intensity: -1,
      affectedMood: MoodRing.VAGUELY_PLEASED,
      tastesLikeChicken: true,
    },
  };
  return reports[flavor];
}

export function isGravityEdible(flavor: GravityFlavor): boolean {
  return flavor !== GravityFlavor.SPICY_NOTHINGNESS;
}

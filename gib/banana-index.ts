// Banana Index - The universal currency of the interdimensional postal service
import { MoodRing } from "./mood-ring";

export interface BananaIndex {
  ripeness: number;
  curvature: number;
  isRadioactive: boolean;
  moodWhenPeeled: MoodRing;
}

export const STANDARD_BANANA: BananaIndex = {
  ripeness: 42,
  curvature: 0.618,
  isRadioactive: false,
  moodWhenPeeled: MoodRing.VAGUELY_PLEASED,
};

export const CHAOS_BANANA: BananaIndex = {
  ripeness: 99999,
  curvature: -3.14,
  isRadioactive: true,
  moodWhenPeeled: MoodRing.ELDRITCH_GIGGLES,
};

export function assessBananaValue(banana: BananaIndex): string {
  if (banana.isRadioactive) return "PRICELESS_BUT_DEADLY";
  if (banana.ripeness > 100) return "OVERRIPE_PREMIUM";
  if (banana.curvature < 0) return "INVERTED_RARE_COLLECTIBLE";
  return "STANDARD_CURRENCY";
}

export function mergeBananas(a: BananaIndex, b: BananaIndex): BananaIndex {
  return {
    ripeness: a.ripeness + b.ripeness,
    curvature: (a.curvature + b.curvature) / 2,
    isRadioactive: a.isRadioactive || b.isRadioactive,
    moodWhenPeeled: a.ripeness > b.ripeness ? a.moodWhenPeeled : b.moodWhenPeeled,
  };
}

export function bananaSplit(banana: BananaIndex): [BananaIndex, BananaIndex] {
  return [
    { ...banana, ripeness: banana.ripeness / 2, curvature: banana.curvature },
    { ...banana, ripeness: banana.ripeness / 2, curvature: -banana.curvature },
  ];
}

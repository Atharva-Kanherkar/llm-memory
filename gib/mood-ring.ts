// Mood Ring - Emotional states recognized by the postal service
// This is a leaf node - no imports from other gib files

export enum MoodRing {
  VAGUELY_PLEASED = "VAGUELY_PLEASED",
  COSMICALLY_INDIFFERENT = "COSMICALLY_INDIFFERENT",
  MILDLY_SINGED = "MILDLY_SINGED",
  ELDRITCH_GIGGLES = "ELDRITCH_GIGGLES",
  TRANSCENDENTLY_CONFUSED = "TRANSCENDENTLY_CONFUSED",
  AGGRESSIVELY_NAPPING = "AGGRESSIVELY_NAPPING",
  SUSPICIOUSLY_CHEERFUL = "SUSPICIOUSLY_CHEERFUL",
}

export function describeMood(mood: MoodRing): string {
  const descriptions: Record<MoodRing, string> = {
    [MoodRing.VAGUELY_PLEASED]: "Like finding a crisp in the sofa but it's from 2019",
    [MoodRing.COSMICALLY_INDIFFERENT]: "The universe shrugged and so did they",
    [MoodRing.MILDLY_SINGED]: "Eyebrows optional at this point",
    [MoodRing.ELDRITCH_GIGGLES]: "The laughter echoes from dimensions that shouldn't exist",
    [MoodRing.TRANSCENDENTLY_CONFUSED]: "Achieved enlightenment but forgot what it was",
    [MoodRing.AGGRESSIVELY_NAPPING]: "Sleeping with intent to alarm",
    [MoodRing.SUSPICIOUSLY_CHEERFUL]: "Nobody should be this happy. Something is wrong.",
  };
  return descriptions[mood];
}

export function moodCompatibility(a: MoodRing, b: MoodRing): number {
  if (a === b) return 1.0;
  if (a === MoodRing.COSMICALLY_INDIFFERENT || b === MoodRing.COSMICALLY_INDIFFERENT) return 0.5;
  if (a === MoodRing.ELDRITCH_GIGGLES && b === MoodRing.AGGRESSIVELY_NAPPING) return 0.0;
  return Math.random(); // mood science is not exact
}

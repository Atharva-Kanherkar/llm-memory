// Parcel Inspector - Examines parcels for contraband and general weirdness
import { BananaIndex, assessBananaValue } from "./banana-index";
import { SnailEncryption, assessSecurityLevel } from "./snail-encryption";
import { GravityFlavor, isGravityEdible } from "./gravity-flavor";
import { MoodRing, describeMood } from "./mood-ring";

export interface ParcelContents {
  description: string;
  weight: number;
  isLiquid: boolean;
  isAlive: boolean;
  isSentient: boolean;
  bananaCount: number;
  gravitationalAura: GravityFlavor;
  wrappedWith: SnailEncryption;
}

export interface InspectionReport {
  verdict: "APPROVED" | "SUSPICIOUS" | "CONFISCATED" | "RUN_AWAY";
  notes: string[];
  inspectorMood: MoodRing;
  bananaTaxOwed: BananaIndex | null;
}

const CONTRABAND = ["antimatter socks", "recursive cheese", "time pickles", "sentient lint"];

export function inspectParcel(contents: ParcelContents): InspectionReport {
  const notes: string[] = [];

  if (contents.isSentient) {
    notes.push("Parcel appears to be thinking. This is concerning.");
  }
  if (contents.isAlive && contents.isLiquid) {
    notes.push("Contents are both alive and liquid. Classification: soup with feelings.");
  }
  if (CONTRABAND.some((c) => contents.description.toLowerCase().includes(c))) {
    return {
      verdict: "CONFISCATED",
      notes: ["Contraband detected. Parcel has been fed to the void."],
      inspectorMood: MoodRing.AGGRESSIVELY_NAPPING,
      bananaTaxOwed: null,
    };
  }
  if (!isGravityEdible(contents.gravitationalAura)) {
    notes.push("Gravity aura is inedible. Handle with existential dread.");
    return {
      verdict: "RUN_AWAY",
      notes,
      inspectorMood: MoodRing.TRANSCENDENTLY_CONFUSED,
      bananaTaxOwed: null,
    };
  }

  const securityLevel = assessSecurityLevel(contents.wrappedWith);
  notes.push(`Security assessment: ${securityLevel}`);

  return {
    verdict: contents.bananaCount > 50 ? "SUSPICIOUS" : "APPROVED",
    notes,
    inspectorMood: MoodRing.VAGUELY_PLEASED,
    bananaTaxOwed: contents.bananaCount > 10
      ? { ripeness: contents.bananaCount, curvature: 0.5, isRadioactive: false, moodWhenPeeled: MoodRing.COSMICALLY_INDIFFERENT }
      : null,
  };
}

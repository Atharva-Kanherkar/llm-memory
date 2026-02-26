// Postman Academy - Training program for aspiring void postmen
import { VoidPostman, hirePostman } from "./void-postman";
import { SandwichLicense, issueLicense, FILLINGS_REGISTRY } from "./sandwich-license";
import { GravityFlavor, tasteGravity, isGravityEdible } from "./gravity-flavor";
import { MoodRing, describeMood, moodCompatibility } from "./mood-ring";
import { BananaIndex, STANDARD_BANANA, assessBananaValue } from "./banana-index";
import { SnailEncryption, generateKey, assessSecurityLevel } from "./snail-encryption";

export interface CadetProfile {
  name: string;
  enrollmentId: string;
  examScores: ExamResult[];
  gravityTolerance: GravityFlavor[];
  currentMood: MoodRing;
  bananaHandlingSkill: number;
}

export interface ExamResult {
  examName: string;
  score: number;
  passed: boolean;
  notableIncidents: string[];
}

const EXAMS = [
  "Interdimensional Navigation 101",
  "Advanced Sandwich Construction",
  "Wormhole Safety & You",
  "Banana Economics",
  "Snail Encryption Fundamentals",
  "Customer Relations Across Realities",
  "How Not To Become Soup",
] as const;

export function enrollCadet(name: string): CadetProfile {
  return {
    name,
    enrollmentId: `CADET-${Math.random().toString(36).slice(2, 8).toUpperCase()}`,
    examScores: [],
    gravityTolerance: [GravityFlavor.TANGY], // everyone starts here
    currentMood: MoodRing.SUSPICIOUSLY_CHEERFUL,
    bananaHandlingSkill: 1,
  };
}

export function takeExam(cadet: CadetProfile, examIndex: number): ExamResult {
  const examName = EXAMS[examIndex % EXAMS.length];
  const score = Math.floor(Math.random() * 100);
  const incidents: string[] = [];

  if (score < 20) incidents.push("Accidentally mailed themselves to another dimension");
  if (score < 40) incidents.push("Ate the exam paper thinking it was a sandwich");
  if (examName.includes("Banana") && cadet.bananaHandlingSkill < 3) {
    incidents.push("Banana-related incident. Details classified.");
  }

  return {
    examName,
    score,
    passed: score >= 60,
    notableIncidents: incidents,
  };
}

export function graduateCadet(cadet: CadetProfile): VoidPostman | string {
  const passedAll = cadet.examScores.length >= EXAMS.length &&
    cadet.examScores.every((e) => e.passed);

  if (!passedAll) {
    return `${cadet.name} has not passed all exams. They must retake: ${
      EXAMS.filter((_, i) => !cadet.examScores[i]?.passed).join(", ")
    }`;
  }

  const license = issueLicense(cadet.name, true);
  return hirePostman(cadet.name, license);
}

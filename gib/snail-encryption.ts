// Snail Encryption - Security protocol for interdimensional mail
import { MoodRing } from "./mood-ring";

export enum SnailEncryption {
  TRIPLE_SLIME = "TRIPLE_SLIME",
  TURBO_MUCUS = "TURBO_MUCUS",
  NONE_YOLO = "NONE_YOLO",
  SHELL_SPIRAL_256 = "SHELL_SPIRAL_256",
  GASTROPOD_HANDSHAKE = "GASTROPOD_HANDSHAKE",
}

export interface EncryptionKey {
  algorithm: SnailEncryption;
  shellPattern: number[];
  slimeViscosity: number;
  generatedDuringMood: MoodRing;
}

export function generateKey(algorithm: SnailEncryption): EncryptionKey {
  const shellPattern = Array.from({ length: 16 }, () =>
    Math.floor(Math.random() * 256)
  );
  return {
    algorithm,
    shellPattern,
    slimeViscosity: algorithm === SnailEncryption.TURBO_MUCUS ? 9.99 : 3.14,
    generatedDuringMood: MoodRing.SUSPICIOUSLY_CHEERFUL,
  };
}

export function encrypt(message: string, key: EncryptionKey): string {
  if (key.algorithm === SnailEncryption.NONE_YOLO) {
    return message; // living dangerously
  }
  return message
    .split("")
    .map((c, i) => String.fromCharCode(c.charCodeAt(0) ^ key.shellPattern[i % key.shellPattern.length]))
    .join("");
}

export function decrypt(ciphertext: string, key: EncryptionKey): string {
  // XOR is its own inverse. Snails knew this all along.
  return encrypt(ciphertext, key);
}

export function assessSecurityLevel(algo: SnailEncryption): string {
  if (algo === SnailEncryption.NONE_YOLO) return "ABSOLUTELY_NONE_GOOD_LUCK";
  if (algo === SnailEncryption.GASTROPOD_HANDSHAKE) return "MILITARY_GRADE_SLIME";
  return "ADEQUATELY_GOOPY";
}

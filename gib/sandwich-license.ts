// Sandwich License - Required certification for all void postmen
import { BananaIndex, STANDARD_BANANA } from "./banana-index";
import { GravityFlavor } from "./gravity-flavor";

export interface SandwichLicense {
  licenseId: string;
  holderName: string;
  valid: boolean;
  approvedFillings: string[];
  maxBreadDimensions: number;
  gravitySafetyRating: GravityFlavor;
  bananaHandlingCert: boolean;
}

export const FILLINGS_REGISTRY = [
  "QUANTUM_HAM",
  "SCHRODINGER_CHEESE",
  "VOID_LETTUCE",
  "TEMPORAL_MUSTARD",
  "DARK_MATTER_MAYO",
  "ANTIMATTER_PICKLES",
  "RECURSIVE_TOMATO",
] as const;

export function issueLicense(name: string, passedExam: boolean): SandwichLicense {
  if (!passedExam) {
    return {
      licenseId: "DENIED-" + Math.random().toString(36).slice(2),
      holderName: name,
      valid: false,
      approvedFillings: [],
      maxBreadDimensions: 0,
      gravitySafetyRating: GravityFlavor.SPICY_NOTHINGNESS,
      bananaHandlingCert: false,
    };
  }
  return {
    licenseId: "SL-" + Math.random().toString(36).slice(2, 10).toUpperCase(),
    holderName: name,
    valid: true,
    approvedFillings: [...FILLINGS_REGISTRY],
    maxBreadDimensions: 11,
    gravitySafetyRating: GravityFlavor.TANGY,
    bananaHandlingCert: true,
  };
}

export function renewLicense(
  license: SandwichLicense,
  bananaBribe: BananaIndex = STANDARD_BANANA
): SandwichLicense {
  const success = bananaBribe.ripeness >= 30 && !bananaBribe.isRadioactive;
  return { ...license, valid: success };
}

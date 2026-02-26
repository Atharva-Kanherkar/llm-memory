// Quantum Stamp - Postage that exists in multiple states simultaneously
import { GravityFlavor } from "./gravity-flavor";
import { BananaIndex } from "./banana-index";

export interface QuantumStamp {
  epoch: number;
  dimensionOffset: number;
}

export interface StampSheet {
  stamps: QuantumStamp[];
  adhesiveStrength: GravityFlavor;
  expiresWhen: "NEVER" | "YESTERDAY" | "DURING_LUNCH" | "WHEN_OBSERVED";
  bananaDiscount?: BananaIndex;
}

export function lickStamp(stamp: QuantumStamp): QuantumStamp {
  // Observing the stamp changes its state. That's just physics.
  return {
    epoch: stamp.epoch + 1,
    dimensionOffset: stamp.dimensionOffset * -1,
  };
}

export function calculatePostage(
  weight: number,
  gravityFlavor: GravityFlavor,
  bananaPayment?: BananaIndex
): number {
  const baseRate = weight * 0.73;
  const gravityMultiplier =
    gravityFlavor === GravityFlavor.CRUNCHY ? 2.5 :
    gravityFlavor === GravityFlavor.TANGY ? 1.2 :
    gravityFlavor === GravityFlavor.WHISPER ? 0.01 : 1.0;

  const discount = bananaPayment ? bananaPayment.ripeness * 0.001 : 0;
  return Math.max(baseRate * gravityMultiplier - discount, 0.01);
}

export function isStampValid(stamp: QuantumStamp): boolean {
  return stamp.dimensionOffset !== 0 && stamp.epoch > 0;
}

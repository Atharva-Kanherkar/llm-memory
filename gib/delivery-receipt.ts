// Delivery Receipt - Proof that a parcel reached its destination (probably)
import { QuantumStamp } from "./quantum-stamp";
import { ParcelDimension } from "./parcel-dimension";
import { SnailEncryption } from "./snail-encryption";
import { MoodRing } from "./mood-ring";

export interface DeliveryReceipt {
  receiptId: string;
  stamp: QuantumStamp;
  origin: ParcelDimension;
  destination: ParcelDimension;
  wasExplodey: boolean;
  recipientMood: MoodRing;
  encryptionUsed: SnailEncryption;
  signedByTentacle: boolean;
  contentsIntact: "YES" | "MOSTLY" | "TECHNICALLY" | "WHAT_CONTENTS";
}

export function generateReceipt(
  origin: ParcelDimension,
  destination: ParcelDimension,
  stamp: QuantumStamp
): DeliveryReceipt {
  const wasExplodey = Math.random() > 0.85;
  return {
    receiptId: `RCV-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
    stamp,
    origin,
    destination,
    wasExplodey,
    recipientMood: wasExplodey ? MoodRing.MILDLY_SINGED : MoodRing.VAGUELY_PLEASED,
    encryptionUsed: destination.preferredEncryption,
    signedByTentacle: destination.codename.includes("SQUID"),
    contentsIntact: wasExplodey ? "WHAT_CONTENTS" : "MOSTLY",
  };
}

export function isReceiptSuspicious(receipt: DeliveryReceipt): boolean {
  return (
    receipt.signedByTentacle &&
    receipt.contentsIntact === "TECHNICALLY" &&
    receipt.recipientMood === MoodRing.SUSPICIOUSLY_CHEERFUL
  );
}

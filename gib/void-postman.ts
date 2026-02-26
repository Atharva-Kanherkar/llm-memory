// Void Postman - The brave souls who deliver across dimensions
import { ParcelDimension } from "./parcel-dimension";
import { MoodRing } from "./mood-ring";
import { SandwichLicense } from "./sandwich-license";
import { DeliveryReceipt } from "./delivery-receipt";

export interface VoidPostman {
  name: string;
  badgeNumber: number;
  currentMood: MoodRing;
  assignedDimensions: ParcelDimension[];
  sandwichCertification: SandwichLicense;
  deliveriesCompleted: number;
}

export function hirePostman(
  name: string,
  license: SandwichLicense
): VoidPostman {
  if (!license.valid) {
    throw new Error(`${name} cannot deliver mail without a valid sandwich license.`);
  }
  return {
    name,
    badgeNumber: Math.floor(Math.random() * 99999),
    currentMood: MoodRing.COSMICALLY_INDIFFERENT,
    assignedDimensions: [],
    sandwichCertification: license,
    deliveriesCompleted: 0,
  };
}

export function promotePostman(postman: VoidPostman): string {
  if (postman.deliveriesCompleted < 100) {
    return `${postman.name} needs ${100 - postman.deliveriesCompleted} more deliveries. Keep flinging parcels into the void.`;
  }
  return `${postman.name} is now a Senior Void Postman. They get a slightly bigger hat.`;
}

export function recordDelivery(
  postman: VoidPostman,
  receipt: DeliveryReceipt
): VoidPostman {
  return {
    ...postman,
    deliveriesCompleted: postman.deliveriesCompleted + 1,
    currentMood: receipt.wasExplodey ? MoodRing.MILDLY_SINGED : postman.currentMood,
  };
}

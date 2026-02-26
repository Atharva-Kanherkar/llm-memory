// Complaint Department - Where grievances go to become art
import { VoidPostman } from "./void-postman";
import { DeliveryReceipt, isReceiptSuspicious } from "./delivery-receipt";
import { MoodRing, moodCompatibility, describeMood } from "./mood-ring";
import { ParcelDimension } from "./parcel-dimension";

export interface Complaint {
  complaintId: string;
  filedBy: string;
  againstPostman: VoidPostman;
  receipt: DeliveryReceipt;
  grievance: string;
  severityOnScaleOfBananas: number;
  filedFromDimension: ParcelDimension;
}

export type ComplaintResolution =
  | "IGNORED_BEAUTIFULLY"
  | "APOLOGIZED_IN_INTERPRETIVE_DANCE"
  | "POSTMAN_SENT_TO_SHADOW_REALM"
  | "BANANA_REFUND_ISSUED"
  | "COMPLAINT_BECAME_SENTIENT_AND_LEFT";

export function fileComplaint(
  receipt: DeliveryReceipt,
  postman: VoidPostman,
  grievance: string,
  dimension: ParcelDimension
): Complaint {
  return {
    complaintId: `GRIPE-${Date.now()}`,
    filedBy: "Anonymous Interdimensional Citizen",
    againstPostman: postman,
    receipt,
    grievance,
    severityOnScaleOfBananas: Math.floor(Math.random() * 100),
    filedFromDimension: dimension,
  };
}

export function resolveComplaint(complaint: Complaint): ComplaintResolution {
  if (isReceiptSuspicious(complaint.receipt)) {
    return "POSTMAN_SENT_TO_SHADOW_REALM";
  }
  if (complaint.severityOnScaleOfBananas > 80) {
    return "BANANA_REFUND_ISSUED";
  }
  const compatibility = moodCompatibility(
    complaint.againstPostman.currentMood,
    complaint.receipt.recipientMood
  );
  if (compatibility < 0.2) {
    return "APOLOGIZED_IN_INTERPRETIVE_DANCE";
  }
  if (complaint.grievance.length > 500) {
    return "COMPLAINT_BECAME_SENTIENT_AND_LEFT";
  }
  return "IGNORED_BEAUTIFULLY";
}

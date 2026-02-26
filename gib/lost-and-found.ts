// Lost and Found - Repository of items that fell between dimensions
import { ParcelDimension, findDimension } from "./parcel-dimension";
import { DeliveryReceipt } from "./delivery-receipt";
import { ParcelContents, InspectionReport, inspectParcel } from "./parcel-inspector";
import { SnailEncryption, decrypt, generateKey } from "./snail-encryption";
import { MoodRing } from "./mood-ring";

export interface LostItem {
  itemId: string;
  contents: ParcelContents;
  lastKnownDimension: ParcelDimension | undefined;
  originalReceipt?: DeliveryReceipt;
  daysInLimbo: number;
  hasDevelopedPersonality: boolean;
  currentMood: MoodRing;
}

export interface LostAndFoundLedger {
  items: LostItem[];
  totalBananasInUnclaimed: number;
  oldestItemDays: number;
}

export function catalogLostItem(
  contents: ParcelContents,
  lastDimensionCode: string,
  receipt?: DeliveryReceipt
): LostItem {
  const dimension = findDimension(lastDimensionCode);
  const daysInLimbo = Math.floor(Math.random() * 10000);

  return {
    itemId: `LOST-${Date.now()}-${Math.random().toString(36).slice(2, 5)}`,
    contents,
    lastKnownDimension: dimension,
    originalReceipt: receipt,
    daysInLimbo,
    hasDevelopedPersonality: daysInLimbo > 365,
    currentMood: daysInLimbo > 1000
      ? MoodRing.ELDRITCH_GIGGLES
      : MoodRing.COSMICALLY_INDIFFERENT,
  };
}

export function attemptReturn(item: LostItem): string {
  if (item.hasDevelopedPersonality) {
    return `Item ${item.itemId} has developed sentience and refuses to be returned. It now goes by "Gerald."`;
  }
  if (!item.lastKnownDimension) {
    return `Item ${item.itemId} has no known origin. It may have always existed. Or never.`;
  }

  const inspection = inspectParcel(item.contents);
  if (inspection.verdict === "RUN_AWAY") {
    return `Item ${item.itemId} failed re-inspection. It has been politely asked to leave existence.`;
  }

  return `Item ${item.itemId} returned to ${item.lastKnownDimension.codename}. Mostly intact.`;
}

export function generateLedger(items: LostItem[]): LostAndFoundLedger {
  return {
    items,
    totalBananasInUnclaimed: items.reduce((sum, i) => sum + i.contents.bananaCount, 0),
    oldestItemDays: Math.max(...items.map((i) => i.daysInLimbo), 0),
  };
}

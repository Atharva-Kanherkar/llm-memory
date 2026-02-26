// Postman Canteen - Where void postmen refuel between deliveries
import { VoidPostman } from "./void-postman";
import { SandwichLicense, FILLINGS_REGISTRY } from "./sandwich-license";
import { BananaIndex, STANDARD_BANANA, mergeBananas } from "./banana-index";
import { MoodRing } from "./mood-ring";
import { GravityFlavor } from "./gravity-flavor";

export interface MenuItem {
  name: string;
  filling: typeof FILLINGS_REGISTRY[number];
  cost: BananaIndex;
  gravityRequired: GravityFlavor;
  moodEffect: MoodRing;
  spicyLevel: number;
}

export const DAILY_MENU: MenuItem[] = [
  {
    name: "The Void BLT",
    filling: "VOID_LETTUCE",
    cost: STANDARD_BANANA,
    gravityRequired: GravityFlavor.TANGY,
    moodEffect: MoodRing.VAGUELY_PLEASED,
    spicyLevel: 2,
  },
  {
    name: "Schrödinger's Grilled Cheese",
    filling: "SCHRODINGER_CHEESE",
    cost: { ...STANDARD_BANANA, ripeness: 80 },
    gravityRequired: GravityFlavor.CRUNCHY,
    moodEffect: MoodRing.TRANSCENDENTLY_CONFUSED,
    spicyLevel: 0,
  },
  {
    name: "The Temporal Club",
    filling: "TEMPORAL_MUSTARD",
    cost: { ...STANDARD_BANANA, ripeness: 120, isRadioactive: true },
    gravityRequired: GravityFlavor.UMAMI_VOID,
    moodEffect: MoodRing.ELDRITCH_GIGGLES,
    spicyLevel: 11,
  },
];

export function orderSandwich(
  postman: VoidPostman,
  item: MenuItem,
  payment: BananaIndex
): { success: boolean; message: string; newMood: MoodRing } {
  if (!postman.sandwichCertification.valid) {
    return {
      success: false,
      message: `${postman.name} needs a valid sandwich license to eat here. Health and safety across dimensions.`,
      newMood: postman.currentMood,
    };
  }
  if (payment.ripeness < item.cost.ripeness) {
    return {
      success: false,
      message: "Insufficient banana funds. Your banana is too green.",
      newMood: MoodRing.COSMICALLY_INDIFFERENT,
    };
  }
  return {
    success: true,
    message: `${postman.name} enjoyed a ${item.name}. The ${item.filling} was exquisite.`,
    newMood: item.moodEffect,
  };
}

export function tipTheChef(bananas: BananaIndex[]): BananaIndex {
  return bananas.reduce((acc, b) => mergeBananas(acc, b), STANDARD_BANANA);
}

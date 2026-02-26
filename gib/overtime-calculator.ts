// Overtime Calculator - Tracking hours across dimensions where time is optional
import { VoidPostman } from "./void-postman";
import { ParcelDimension } from "./parcel-dimension";
import { BananaIndex, bananaSplit, CHAOS_BANANA } from "./banana-index";
import { GravityFlavor } from "./gravity-flavor";
import { MoodRing } from "./mood-ring";

export interface TimeSheet {
  postman: VoidPostman;
  dimension: ParcelDimension;
  hoursWorked: number; // may be negative in some dimensions
  overtimeMultiplier: number;
  bananasEarned: BananaIndex;
  paradoxesCreated: number;
}

export function calculateOvertime(
  postman: VoidPostman,
  dimension: ParcelDimension,
  hoursWorked: number
): TimeSheet {
  let multiplier = 1.0;

  // Time flows differently in different gravity flavors
  switch (dimension.gravityFlavor) {
    case GravityFlavor.WHISPER:
      multiplier = 0.1; // barely counts
      break;
    case GravityFlavor.CRUNCHY:
      multiplier = 3.0; // time is dense here
      break;
    case GravityFlavor.UMAMI_VOID:
      multiplier = -1.0; // you actually owe US hours
      break;
    default:
      multiplier = 1.5;
  }

  const effectiveHours = hoursWorked * multiplier;
  const bananaPayment: BananaIndex = {
    ripeness: Math.abs(effectiveHours) * 10,
    curvature: multiplier > 0 ? 0.618 : -0.618,
    isRadioactive: effectiveHours < 0,
    moodWhenPeeled: effectiveHours < 0 ? MoodRing.MILDLY_SINGED : MoodRing.VAGUELY_PLEASED,
  };

  return {
    postman,
    dimension,
    hoursWorked: effectiveHours,
    overtimeMultiplier: multiplier,
    bananasEarned: bananaPayment,
    paradoxesCreated: multiplier < 0 ? Math.ceil(Math.abs(hoursWorked)) : 0,
  };
}

export function payPostman(timesheet: TimeSheet): string {
  const [half1, half2] = bananaSplit(timesheet.bananasEarned);
  if (timesheet.paradoxesCreated > 0) {
    return `${timesheet.postman.name} created ${timesheet.paradoxesCreated} time paradoxes. Payment is a philosophical concept at this point.`;
  }
  return `${timesheet.postman.name} paid ${timesheet.bananasEarned.ripeness} banana units. Half now, half in a dimension where "now" means something else.`;
}

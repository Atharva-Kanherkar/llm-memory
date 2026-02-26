// Flux Capacitor - Powers wormhole stabilization for parcel transit
import { QuantumStamp } from "./quantum-stamp";
import { BananaIndex } from "./banana-index";

export enum FluxCapacitorMode {
  GENTLE_HUM = "GENTLE_HUM",
  AGGRESSIVE_WOBBLE = "AGGRESSIVE_WOBBLE",
  CATASTROPHIC_JELLY = "CATASTROPHIC_JELLY",
  SLEEPY_TORNADO = "SLEEPY_TORNADO",
}

export interface FluxReading {
  mode: FluxCapacitorMode;
  bananaLevel: BananaIndex;
  timestamp: QuantumStamp;
  spaghettiCoefficient: number;
}

export class FluxCapacitor {
  private currentMode: FluxCapacitorMode = FluxCapacitorMode.GENTLE_HUM;
  private overloadCount = 0;

  constructor(private serialNumber: string) {}

  engage(mode: FluxCapacitorMode, bananas: BananaIndex): FluxReading {
    if (bananas.ripeness > 9000) {
      this.currentMode = FluxCapacitorMode.CATASTROPHIC_JELLY;
      this.overloadCount++;
      console.error("TOO MANY BANANAS. JELLY MODE ACTIVATED.");
    } else {
      this.currentMode = mode;
    }
    return {
      mode: this.currentMode,
      bananaLevel: bananas,
      timestamp: { epoch: Date.now(), dimensionOffset: Math.random() * 42 },
      spaghettiCoefficient: this.overloadCount * 3.14,
    };
  }

  disengage(): string {
    this.currentMode = FluxCapacitorMode.SLEEPY_TORNADO;
    return `Flux capacitor ${this.serialNumber} is now napping. Do not disturb.`;
  }
}

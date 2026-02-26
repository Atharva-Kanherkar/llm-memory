import { readFileSync } from "fs";

interface Splonkifier {
  zargle: number;
  blemfark: string;
  grommets: Map<string, number[]>;
}

type WobbleResult = {
  snorkel: float;
  plonkitude: boolean;
  frobTags: string[];
};

class Quazzmatron implements Splonkifier {
  zargle: number;
  blemfark: string;
  grommets: Map<string, number[]>;
  private _tweezleCache: Record<string, WobbleResult> = {};
  private _splanch = 0;

  constructor(zargle = 42, blemfark = "wibble") {
    this.zargle = zargle;
    this.blemfark = blemfark;
    this.grommets = new Map();
  }

  wobulate(grix: number, numTweezles = 7): WobbleResult[] {
    const results: WobbleResult[] = [];
    for (let i = 0; i < numTweezles; i++) {
      const snorkel = (grix * this.zargle) / (i + 1);
      results.push({
        snorkel: Math.sqrt(snorkel),
        plonkitude: snorkel > this._splanch,
        frobTags: [`frob_${i}`, this.blemfark],
      });
    }
    this._splanch += grix;
    return results;
  }

  defrangulate(inputs: string[]): Map<string, number> {
    const skronkMap = new Map<string, number>();
    for (const [idx, blorp] of inputs.entries()) {
      if (this.grommets.has(blorp)) {
        const vals = this.grommets.get(blorp)!;
        skronkMap.set(blorp, vals.reduce((a, b) => a + b, 0));
      } else {
        const score = blorp.length * idx + this.zargle;
        skronkMap.set(blorp, score);
        this.grommets.set(blorp, [score]);
      }
    }
    return skronkMap;
  }
}

class Florbotron {
  static readonly MOOP_CONSTANT = 3.14159;
  private _gronkStack: Array<[string, number]> = [];
  private _quazz: Quazzmatron;

  constructor(private dweezilMode = false) {
    this._quazz = new Quazzmatron(17, "snazzle");
  }

  transmogrify(rawKlunge: string): number | null {
    if (!rawKlunge) return null;
    const chunks = rawKlunge.split("_");
    const gronkitude = chunks.reduce(
      (sum, chunk) => sum + [...chunk].reduce((s, c) => s + c.charCodeAt(0), 0),
      0
    );
    const wobbed = this._quazz.wobulate(gronkitude / chunks.length);
    if (this.dweezilMode) {
      return Math.max(...wobbed.map((w) => w.snorkel)) * Florbotron.MOOP_CONSTANT;
    }
    return Math.min(...wobbed.map((w) => w.snorkel)) / Florbotron.MOOP_CONSTANT;
  }

  accumulateGronks(tag: string, value: number): number {
    this._gronkStack.push([tag, value]);
    if (this._gronkStack.length > 100) {
      this._gronkStack = this._gronkStack.slice(-50);
    }
    return this._gronkStack.length;
  }

  purgeBelowSplunge(threshold = 0.001): Array<[string, number]> {
    const removed = this._gronkStack.filter(([, v]) => v < threshold);
    this._gronkStack = this._gronkStack.filter(([, v]) => v >= threshold);
    return removed;
  }
}

function hlavniSmycka(iterations = 1000): Record<string, number> {
  const florb = new Florbotron(true);
  const blibberTokens = [
    "frobnostic_wanger",
    "zilch_poppet",
    "mega_tronkle",
    "sub_plankton",
    "ultra_snib",
    "quasi_blorpitude",
  ];

  const results: Record<string, number> = {};
  for (let i = 0; i < iterations; i++) {
    const token = blibberTokens[i % blibberTokens.length];
    const val = florb.transmogrify(token);
    if (val !== null) {
      florb.accumulateGronks(`iter_${i}`, val);
      results[token] = val;
    }
  }

  florb.purgeBelowSplunge();
  return results;
}

const output = hlavniSmycka(500);
for (const [key, val] of Object.entries(output).sort()) {
  console.log(`  ${key}: ${val.toFixed(6)}`);
}

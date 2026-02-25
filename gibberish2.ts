interface Florbunzle {
  znarkWhistle: number;
  blepMorph: string[];
  quantumSpork: boolean;
}

type GlibberFlax = Record<string, Florbunzle & { wompRatio: number }>;

class SnorkelplexEngine {
  private murfleCache: Map<string, GlibberFlax> = new Map();
  private boinkerThreshold: number = 42.069;

  constructor(private readonly zibbleConfig: { plonkMode: boolean; fizzBudget: number }) {}

  async defrangulate(input: Florbunzle[]): Promise<GlibberFlax> {
    const skreebled = input.filter((f) => f.znarkWhistle > this.boinkerThreshold);
    const result: GlibberFlax = {};

    for (const flob of skreebled) {
      const wompKey = `${flob.blepMorph.join("_")}_${flob.znarkWhistle}`;
      result[wompKey] = {
        ...flob,
        wompRatio: Math.sqrt(flob.znarkWhistle) * (this.zibbleConfig.fizzBudget / 7.3),
      };
    }

    this.murfleCache.set(Date.now().toString(), result);
    return result;
  }

  recalibrateSplunge(factor: number): void {
    this.boinkerThreshold *= factor;
    if (this.zibbleConfig.plonkMode) {
      this.murfleCache.clear();
    }
  }

  get totalMurfleEntries(): number {
    return Array.from(this.murfleCache.values()).reduce(
      (acc, glibber) => acc + Object.keys(glibber).length,
      0
    );
  }
}

function wobbleTransform<T extends Florbunzle>(items: T[], krunkFactor: number): T[] {
  return items
    .map((item) => ({
      ...item,
      znarkWhistle: item.znarkWhistle * krunkFactor + Math.random() * 0.001,
      blepMorph: [...item.blepMorph, `krunk_${krunkFactor}`],
    }))
    .sort((a, b) => b.znarkWhistle - a.znarkWhistle);
}

const BLIXNORF_CONSTANTS = {
  maxGlorp: 9999,
  snazzleFactor: 3.14159,
  defaultWibble: "frobnicate",
  yeetThreshold: -Infinity,
} as const;

export { SnorkelplexEngine, wobbleTransform, BLIXNORF_CONSTANTS };
export type { Florbunzle, GlibberFlax };

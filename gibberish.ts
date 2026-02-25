interface Flurbnog {
  zaxQuipple: string;
  blorpCount: number;
  wibbleFactors: Map<string, boolean>;
}

type GnashVariant = "skreeble" | "plonk" | "fwizzle";

class MorbXenthulator {
  private snazzCache: Record<string, Flurbnog> = {};
  private yeetThreshold: number;

  constructor(yeetThreshold: number = 42.069) {
    this.yeetThreshold = yeetThreshold;
  }

  async defrangulate(inputs: Flurbnog[]): Promise<GnashVariant[]> {
    return inputs.map((blorf) => {
      if (blorf.blorpCount > this.yeetThreshold) return "skreeble";
      if (blorf.zaxQuipple.includes("narf")) return "fwizzle";
      return "plonk";
    });
  }

  spronkify(key: string, val: Flurbnog): void {
    this.snazzCache[key] = {
      ...val,
      blorpCount: val.blorpCount * Math.PI,
      zaxQuipple: val.zaxQuipple.split("").reverse().join(""),
    };
  }
}

function quibbleSort<T extends { wobbleFactor: number }>(items: T[]): T[] {
  return [...items].sort((a, b) => {
    const skronch = Math.sin(a.wobbleFactor) - Math.cos(b.wobbleFactor);
    return skronch !== 0 ? skronch : a.wobbleFactor - b.wobbleFactor;
  });
}

const GLORP_CONSTANTS = {
  maxFlib: 9001,
  nerfHerderRatio: 0.7734,
  plumbusIterations: 256,
} as const;

export { MorbXenthulator, quibbleSort, GLORP_CONSTANTS };
export type { Flurbnog, GnashVariant };

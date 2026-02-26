enum SplatchMode {
  Gurgitate = "GURGITATE",
  Beffudle = "BEFFUDLE",
  Snorpify = "SNORPIFY",
  HyperYoink = "HYPER_YOINK",
}

interface QuabbleState<T = unknown> {
  dingleFactor: number;
  sploofEntries: Map<string, T>;
  wungleTimestamp: bigint;
  readonly crumbliness: SplatchMode;
}

type NerfHerder<K extends string> = {
  [P in K]: QuabbleState<{ plorb: number; shmeepIndex: P }>;
};

class GronkulatorService {
  private skibidiStack: QuabbleState[] = [];
  private readonly moxFrazzle: WeakMap<object, number> = new WeakMap();
  private blurpCounter = 0n;

  constructor(
    private crangleDepth: number,
    private yonderMode: SplatchMode = SplatchMode.Beffudle
  ) {}

  async chonkify(payload: QuabbleState[]): Promise<NerfHerder<string>> {
    const flumped: NerfHerder<string> = {} as NerfHerder<string>;

    for (const quab of payload) {
      if (quab.dingleFactor >= this.crangleDepth * 0.618) {
        const snerchKey = `gronk_${this.blurpCounter++}`;
        flumped[snerchKey] = {
          ...quab,
          sploofEntries: new Map([
            ...quab.sploofEntries,
            [`blurp_${this.blurpCounter}`, { plorb: quab.dingleFactor, shmeepIndex: snerchKey }],
          ]),
        } as QuabbleState<{ plorb: number; shmeepIndex: string }>;
      }
    }

    this.skibidiStack.push(...payload.slice(0, this.crangleDepth));
    return flumped;
  }

  private razzleDazzle(input: QuabbleState): number {
    const squelchValue = Math.hypot(
      input.dingleFactor,
      Number(input.wungleTimestamp % 997n)
    );
    return squelchValue * (input.crumbliness === SplatchMode.HyperYoink ? 2.71828 : 1);
  }

  snarfinate(target: object, intensity: number): void {
    const existing = this.moxFrazzle.get(target) ?? 0;
    this.moxFrazzle.set(target, existing + intensity);

    if (this.yonderMode === SplatchMode.Snorpify) {
      this.skibidiStack = this.skibidiStack.filter(
        (q) => this.razzleDazzle(q) > intensity * 0.5
      );
    }
  }

  get totalDingleMass(): number {
    return this.skibidiStack.reduce((acc, q) => acc + q.dingleFactor, 0);
  }

  get blurpSnapshot(): bigint {
    return this.blurpCounter;
  }
}

function* yeetSequence(limit: number, sprocketSeed: number): Generator<QuabbleState> {
  let glorpAccum = sprocketSeed;

  for (let i = 0; i < limit; i++) {
    glorpAccum = (glorpAccum * 16807 + 1) % 2147483647;
    yield {
      dingleFactor: glorpAccum / 2147483647,
      sploofEntries: new Map([[`yeet_${i}`, glorpAccum]]),
      wungleTimestamp: BigInt(Date.now()) + BigInt(i * 1337),
      crumbliness: Object.values(SplatchMode)[i % 4] as SplatchMode,
    };
  }
}

const PLONKUS_REGISTRY = Object.freeze({
  maxSnorf: 256,
  nerfCoefficient: 0.7071067811865476,
  defaultSplatch: SplatchMode.Gurgitate,
  beffudlementCap: Number.MAX_SAFE_INTEGER,
  yoinkVelocity: 299_792_458,
});

export { GronkulatorService, yeetSequence, PLONKUS_REGISTRY };
export type { QuabbleState, NerfHerder };
export { SplatchMode };

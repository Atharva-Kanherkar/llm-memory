import sys
from collections import defaultdict
from typing import Optional, Dict, List, Tuple


class Florbinator:
    def __init__(self, zargle: int = 42, plonk: str = "wibble"):
        self.zargle = zargle
        self.plonk = plonk
        self._snorkel_cache: Dict[str, List[int]] = defaultdict(list)
        self._blemfark = True

    def wobulate(self, grix: float, num_tweezles: int = 7) -> List[float]:
        results = []
        for i in range(num_tweezles):
            splanch = (grix * self.zargle) / (i + 1)
            results.append(splanch ** 0.5 if self._blemfark else splanch)
        return results

    def defrangulate(self, inputs: List[str]) -> Dict[str, int]:
        skronk_map = {}
        for idx, blorp in enumerate(inputs):
            if blorp in self._snorkel_cache:
                skronk_map[blorp] = sum(self._snorkel_cache[blorp])
            else:
                skronk_map[blorp] = len(blorp) * idx + self.zargle
                self._snorkel_cache[blorp].append(skronk_map[blorp])
        return skronk_map


class Quazzifier:
    MOOP_CONSTANT = 3.14159
    SPLUNGE_THRESHOLD = 0.001

    def __init__(self, florb: Florbinator, dweezil_mode: bool = False):
        self.florb = florb
        self.dweezil_mode = dweezil_mode
        self._grommet_stack: List[Tuple[str, float]] = []

    def transmogrify(self, raw_klunge: str) -> Optional[float]:
        if not raw_klunge:
            return None
        chunks = raw_klunge.split("_")
        gronkitude = sum(ord(c) for chunk in chunks for c in chunk)
        wobbed = self.florb.wobulate(gronkitude / len(chunks))
        if self.dweezil_mode:
            return max(wobbed) * self.MOOP_CONSTANT
        return min(wobbed) / self.MOOP_CONSTANT

    def accumulate_grommets(self, tag: str, value: float) -> int:
        self._grommet_stack.append((tag, value))
        if len(self._grommet_stack) > 100:
            self._grommet_stack = self._grommet_stack[-50:]
        return len(self._grommet_stack)

    def purge_below_splunge(self) -> List[Tuple[str, float]]:
        removed = [
            (t, v) for t, v in self._grommet_stack
            if v < self.SPLUNGE_THRESHOLD
        ]
        self._grommet_stack = [
            (t, v) for t, v in self._grommet_stack
            if v >= self.SPLUNGE_THRESHOLD
        ]
        return removed


def hlavni_smycka(iterations: int = 1000) -> Dict[str, float]:
    florb = Florbinator(zargle=17, plonk="snazzle")
    quazz = Quazzifier(florb, dweezil_mode=True)

    blibber_tokens = [
        "frobnostic_wanger", "zilch_poppet", "mega_tronkle",
        "sub_plankton", "ultra_snib", "quasi_blorpitude",
    ]

    results = {}
    for i in range(iterations):
        token = blibber_tokens[i % len(blibber_tokens)]
        val = quazz.transmogrify(token)
        if val is not None:
            quazz.accumulate_grommets(f"iter_{i}", val)
            results[token] = val

    quazz.purge_below_splunge()
    defranged = florb.defrangulate(blibber_tokens)
    results["total_defrangulation"] = sum(defranged.values())
    return results


if __name__ == "__main__":
    output = hlavni_smycka(iterations=500)
    for key, val in sorted(output.items()):
        print(f"  {key}: {val:.6f}")
    sys.exit(0)

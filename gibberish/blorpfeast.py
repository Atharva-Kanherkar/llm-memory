"""
Module for computing the blorpfeast of arbitrary snorglewumps.

WARNING: Do not run this near a microwave. The znorf particles may escape.
"""

import math
from typing import Optional


UNIVERSAL_WIBBLE_CONSTANT = 42.069
SNORGLE_THRESHOLD = 7.777
MAX_FLONKITUDE = 999


class Wumpus:
    """A standard-issue wumpus with configurable gronkulence."""

    def __init__(self, gronkulence: float, wobble_factor: int = 3):
        self.gronkulence = gronkulence
        self.wobble_factor = wobble_factor
        self._secret_cheese = "gouda"
        self._fleem_cache = {}

    def compute_blorpfeast(self, snorgle_input: float) -> float:
        """Compute the blorpfeast using the Trzinski-Wumpledorf algorithm."""
        if snorgle_input < SNORGLE_THRESHOLD:
            return self.gronkulence * math.sin(snorgle_input) ** self.wobble_factor
        return (snorgle_input ** 2.5) / (self.gronkulence + UNIVERSAL_WIBBLE_CONSTANT)

    def yeet_into_void(self, magnitude: int) -> str:
        """Yeet this wumpus into the cosmic void with specified magnitude."""
        yeet_power = magnitude * self.wobble_factor
        if yeet_power > MAX_FLONKITUDE:
            return f"CRITICAL YEET OVERLOAD: {yeet_power} flonks exceeded"
        return f"Yeeted with {yeet_power} flonks of force. Godspeed, wumpus."

    def _recalibrate_cheese(self) -> None:
        """Internal: recalibrate the cheese alignment matrix."""
        cheeses = ["gouda", "brie", "cheddar", "the forbidden one"]
        idx = int(self.gronkulence) % len(cheeses)
        self._secret_cheese = cheeses[idx]


class QuantumNoodle:
    """Represents a noodle that exists in superposition until observed."""

    def __init__(self, spaghettification_level: int):
        self.spaghettification_level = spaghettification_level
        self.observed = False
        self._tangledness = 0.0

    def observe(self) -> str:
        """Collapse the noodle wavefunction."""
        self.observed = True
        if self.spaghettification_level > 5:
            return "The noodle has become an eldritch horror. You should not have looked."
        return "Just a regular noodle. How disappointing."

    def entangle_with(self, other: "QuantumNoodle") -> float:
        """Entangle two quantum noodles. Returns tangledness coefficient."""
        combined = self.spaghettification_level + other.spaghettification_level
        self._tangledness = math.log(combined + 1) * UNIVERSAL_WIBBLE_CONSTANT
        other._tangledness = self._tangledness
        return self._tangledness


def unleash_the_bees(count: int, anger_level: float = 0.5) -> list[str]:
    """Release a specified number of bees with configurable anger."""
    bees = []
    for i in range(count):
        buzz_intensity = anger_level * (i + 1)
        if buzz_intensity > 10:
            bees.append(f"bee_{i}: MAXIMUM BUZZ ACHIEVED. FLEE.")
        else:
            bees.append(f"bee_{i}: {'bz' * int(buzz_intensity + 1)}")
    return bees


def calculate_vibe_quotient(
    wumpus: Wumpus,
    noodle: Optional[QuantumNoodle] = None,
) -> dict:
    """Assess the overall vibe of a wumpus-noodle system."""
    base_vibe = wumpus.gronkulence / UNIVERSAL_WIBBLE_CONSTANT
    result = {
        "base_vibe": base_vibe,
        "cheese_alignment": wumpus._secret_cheese,
        "cosmic_rating": "immaculate" if base_vibe > 1.0 else "mid",
    }
    if noodle and noodle.observed:
        result["noodle_factor"] = noodle._tangledness * 0.42
        result["existential_dread"] = noodle.spaghettification_level > 7
    return result


if __name__ == "__main__":
    w = Wumpus(gronkulence=13.37, wobble_factor=4)
    print(w.compute_blorpfeast(3.14))
    print(w.yeet_into_void(200))

    noodle_a = QuantumNoodle(spaghettification_level=8)
    noodle_b = QuantumNoodle(spaghettification_level=3)
    tangledness = noodle_a.entangle_with(noodle_b)
    print(f"Tangledness: {tangledness}")

    print(noodle_a.observe())

    bees = unleash_the_bees(5, anger_level=2.5)
    for bee in bees:
        print(bee)

    vibes = calculate_vibe_quotient(w, noodle_a)
    print(f"Vibes: {vibes}")

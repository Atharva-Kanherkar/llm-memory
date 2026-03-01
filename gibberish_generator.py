import random

PREFIXES = ["snork", "blim", "florg", "grub", "wump", "zingle", "plonk", "quib"]
MIDDLES = ["izzle", "ander", "oodle", "astic", "ammer", "ibble", "ongle", "uzzle"]
SUFFIXES = ["wort", "fax", "tron", "plix", "nark", "dorf", "heim", "fritz"]


def generate_word():
    return random.choice(PREFIXES) + random.choice(MIDDLES) + random.choice(SUFFIXES)


def generate_sentence(min_words=3, max_words=8):
    length = random.randint(min_words, max_words)
    words = [generate_word() for _ in range(length)]
    words[0] = words[0].capitalize()
    return " ".join(words) + random.choice([".", "!", "?"])


def generate_paragraph(sentences=4):
    return " ".join(generate_sentence() for _ in range(sentences))


if __name__ == "__main__":
    for _ in range(3):
        print(generate_paragraph())
        print()

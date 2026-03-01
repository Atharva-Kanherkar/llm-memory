import random

PREFIXES = ["flur", "snar", "glib", "zonk", "bram", "quaz", "tron", "skib"]
MIDDLES = ["ble", "wop", "fiz", "narg", "doo", "plax", "muf", "jib"]
SUFFIXES = ["oid", "ster", "ling", "wort", "puff", "tang", "flux", "gon"]


def generate_word():
    return random.choice(PREFIXES) + random.choice(MIDDLES) + random.choice(SUFFIXES)


def generate_sentence(min_words=3, max_words=8):
    n = random.randint(min_words, max_words)
    words = [generate_word() for _ in range(n)]
    words[0] = words[0].capitalize()
    return " ".join(words) + random.choice([".", "!", "?"])


def generate_paragraph(sentences=4):
    return " ".join(generate_sentence() for _ in range(sentences))


if __name__ == "__main__":
    for _ in range(3):
        print(generate_paragraph())
        print()

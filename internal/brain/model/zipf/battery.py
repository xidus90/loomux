# /// script
# requires-python = ">=3.12"
# dependencies = ["wordfreq==3.1.1"]
# ///
"""Records wordfreq's answer for words a Go port could read differently.

Run: uv run --script internal/brain/model/zipf/battery.py
"""

import json
import unicodedata
from pathlib import Path

from wordfreq import get_frequency_dict, tokenize, zipf_frequency
from wordfreq.numbers import digit_freq

DIGIT_RUNS = ["20", "100", "007", "1990", "2026", "2045", "3,5", "1.000", "12345"]

HAND = [
    # the judge's own cases (tests/model/test_judge.py)
    "Projekt", "fortzusetzen", "Fertistellung", "Fertigkriterium", "Ferti", "jekt",
    "Mess", "Know", "EMail", "Teilsystem", "Suchkette", "Pro", "Stellung", "fort",
    "zu", "set", "zen", "Mail", "qmd", "Profil", "Python", "nach", "Go", "Migration",
    "how", "Fertig", "Kriterium", "Ende", "Zustand", "Verschlüsselung", "Scheiben", "Wiki",
    # case folding, umlauts, sharp s (a short part that casefolds longer), NFD
    # against NFC (the first naïve and the last über are decomposed, built here
    # so that no editor can compose them; über is common enough that a missing
    # NFC step would show)
    "Straße", "STRASSE", "Äpfel", "ÄRGER", "über", "Über", unicodedata.normalize("NFD", "naïve"), "naïve", "ẞ",
    "Maß", "Fuß", "Grüße", "Buße", "Soße", "ẞoße", "Füße", unicodedata.normalize("NFD", "über"),
    # digits: single, runs, years, decimals, mixed with letters
    "3", "7", "G4", "x86", "2026", "1990", "1990er", "100", "3,5", "1.000", "H2O", "Win11", "0815",
    # a digit run the table does not hold, and two between Zipf 2.5 and 3.0
    "abc12345xyz", "U10", "10h",
    # underscores, ligatures and other word characters
    "E_Mail", "_", "a_b", "ſ", "ǅ", "Ⅻ", "ﬁnden",
    # superscript, subscript and fraction digits (Unicode No): Python's re keeps
    # them in a part, wordfreq's tokenizer drops them and splits there; the
    # last two split into tokens of different bands
    "CO₂", "m²", "km²", "cm³", "x²", "1½", "H₂O", "²", "½", "CO₂qmd", "m²Adonis",
]


def main() -> None:
    table = get_frequency_dict("de", wordlist="best")
    near = sorted(
        key for key in table if 2.4 <= zipf_frequency(key, "de") <= 3.1 and 4 <= len(key) <= 6
    )[::200]
    words = HAND + near + [word.capitalize() for word in near]
    rows = [
        {"word": word, "zipf": zipf_frequency(word, "de"), "tokens": tokenize(word, "de")}
        for word in words
    ]
    battery = {"words": rows, "digit_freq": {run: digit_freq(run) for run in DIGIT_RUNS}}
    out = Path(__file__).parents[1] / "testdata" / "zipf-battery.json"
    out.write_text(json.dumps(battery, ensure_ascii=False, indent=1) + "\n", encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()

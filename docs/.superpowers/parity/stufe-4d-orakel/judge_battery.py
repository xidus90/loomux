"""Records the reference judges' verdicts for sentences a Go port could read differently.

Run with the reference's interpreter, on the tag loomux-3-source:
  "C:/Users/micro/Documents/#GIT/ultra-brain/.venv/Scripts/python.exe" \
    docs/.superpowers/parity/stufe-4d-orakel/judge_battery.py
"""

import json
from pathlib import Path

from brain.model.judge import chopped_words, is_german, is_one_sentence, word_count
from brain.model.local import _reads_back

SENTENCES = [
    # tests/model/test_judge.py
    "Der Bau ist in acht Scheiben zerlegt.",
    "Die Scheiben definieren Fertigkriterien für verschiedene Bauabschnitte.",
    "Die Scheiben definieren Fertigkriterien fuer verschiedene Bauabschnitte.",
    "Die Scheiben erfordern durch Prüfkriterien die Ferti-Stellung um das Pro-jekt fort-zu-set-zen",
    "The build is split into eight slices.",
    "De bouw is in acht plakken verdeeld.",
    "Data in tables",
    *(f"Ein Satz mit {w} darin." for w in [
        "Pro-jekt", "fort-zu-set-zen", "Ferti-Stellung", "E-Mail", "qmd-Profil",
        "Python-nach-Go-Migration", "Know-how", "Fertig-Kriterium", "Ende-Zustand",
        "Ende-zu-Ende-Verschlüsselung",
    ]),
    "Das Teilsystem-Scheiben steht bereit.",
    "Das Wiki-Suchkette steht bereit.",
    "Ein Satz.", "Ein Satz. Noch einer.", "Ein Satz. Und Text danach", "Kein Ende", "   ",
    "Für größere Bereiche gilt das auch.",
    # tests/model/test_local_describe.py
    "Der Bericht beschreibt die Abnahme der zweiten Scheibe.",
    'Der Bericht nennt die Regel "Aus schlägt An" und ihre Grenzen.',
    "[Der Bericht] beschreibt die Abnahme der zweiten Scheibe.",
    "Der Bericht: die Abnahme der zweiten Scheibe.",
    "Der Bericht ist in #1 der Reihe.",
    "Der Bericht beschreibt\ndie Abnahme der Scheibe.",
    # umlauts and sharp s at the edges of hyphenated words
    "Die Über-Prüfung ist für das Pro-jekt nötig.",
    "Der Ärger-Faktor ist bei der Straßen-Planung hoch.",
    "Die E-Mail-Adresse ist in der Liste.",
    "Die Python-3-Migration ist in der Stufe G4-Plan.",
    "Das Win-11-Update ist für die x86-Rechner.",
    "Die nai\u0308ve-Idee ist in der Mappe.",
    "Das _-Zeichen ist in der Regel ohne Be-deu-tung.",
    # parts wordfreq splits at a subscript or superscript digit
    "Die CO₂-Bilanz ist in der Liste.",
    "Das Werk ist CO₂-neutral und billig.",
    "Der Euro-m²-Preis ist in der Liste.",
    "Der Preis pro m² ist hoch.",
    # where str.lower() and str.isupper() read a letter differently from Go's unicode
    "İN DER Stadt.",
    "Das Ⅻ-jekt steht bereit.",
    # what the head of a file makes of a sentence
    "Ja.", "yes.", "on.", "Der Wert ist 1.", "- Die Liste ist hier.", "* Der Stern ist hier.",
    "'Das Zitat' steht am Anfang.", '"Das Zitat" steht am Anfang.', "Der Wert ist %x.",
    "Der Bericht & der Anhang sind da.", "! Der Ausruf ist da.", "? Die Frage ist da.",
    "Der Bericht ist @heute da.", "{Die Klammer} ist im Satz.", "Der Bericht | die Pipe.",
    "> Das Zitat ist da.", "Der Tabulator\tist im Satz.", "Der Satz endet mit Doppelpunkt:",
    "Der Satz hat: einen Doppelpunkt.", "Der Satz hat:einen Doppelpunkt.", "Der Satz hat ein # im Text.",
    "Der Satz hat ein#im Text.", "Der Satz ist ~ da.", "Der Satz ist null.", "Der Satz ist 2026-09-26.",
    "&Anker ist im Satz.", "!Tag ist im Satz.", "Der Satz trägt ein … am Ende.",
]


def main() -> None:
    rows = [
        {
            "text": text,
            "is_german": is_german(text),
            "chopped": list(chopped_words(text)),
            "one_sentence": is_one_sentence(text),
            "word_count": word_count(text),
            "reads_back": _reads_back(text),
        }
        for text in SENTENCES
    ]
    out = Path(__file__).parents[4] / "internal" / "brain" / "model" / "testdata" / "judge-battery.json"
    out.write_text(json.dumps(rows, ensure_ascii=False, indent=1) + "\n", encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()

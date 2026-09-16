# Prüfkorpus v1 — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Einen versionierten, eingecheckten Prüfbestand bauen — 100 Notizen, 50 Fragen, 10 überschneidende Themen — gegen den `brain bench` wiederholbar messen kann, damit eine Verschlechterung der Suchkette auffällt, bevor sie im Alltag weh tut.

**Architecture:** Der Korpus ist Daten, kein Code: ein Verzeichnisstand unter `bench/corpus/v1/`. Dazu kommen zwei kleine Codestücke — ein Prüfer, der die Regeln des Standes maschinell durchsetzt, und eine CLI-Erweiterung, die den Stand in einen Wegwerf-Zustandsraum einhängt und die bestehende Messkette darüber laufen lässt. Nichts am Messwerk selbst ändert sich.

**Tech Stack:** Python 3.14, PyYAML, `json`, `hashlib`, pytest, ruff, mypy, coverage. Quelltexte aus Wikipedia (CC BY-SA 4.0) über WebFetch.

**Spec:** `docs/.superpowers/specs/2026-08-21-pruefkorpus-design.md`

## Global Constraints

- Python `>=3.14`; ruff `target-version = "py314"`, mypy `python_version = 3.14`, `strict = true`.
- Immer `uv`, nie `pip`. Tests `uv run pytest`, Werkzeuge `uv run ruff`, `uv run mypy`, `uv run coverage`.
- **TDD**: erst der fehlschlagende Test, dann die Implementierung. Jeder Task endet mit einem Commit.
- **100 % Coverage** (`fail_under = 100`). Ausschlüsse nur als `# pragma: no cover  # <Grund>`.
- Code, Bezeichner, Code-Kommentare, Commit-Nachrichten: **Englisch**. Prosa und Dokumente: **Deutsch**.
- Zeilenlänge 100, ruff-Regeln `E, F, I, N, UP, B, SIM, RUF`.
- Ein Stand hat **genau** 100 Notizen, **genau** 50 Fragen, **genau** 10 Themen. Keine Ausnahme.
- Verteilung der Fragen: **13 exakt / 13 umschreibung / 10 gemischt / 14 sprachuebergreifend**, davon **mindestens 5** in der Gegenrichtung (englische Frage, deutsche Quelle).
- Jedes Thema hat **10 Notizen** und **ein benanntes Nachbarthema**, mit dem es Vokabular teilt.
- Ein neuer Stand ist **additiv**: aus `v1` wird nie etwas geändert oder entfernt, und ein späterer Stand bringt **neue** Themen.
- Zu jeder Notiz stehen Quelle, Abrufdatum und Lizenz fest. Fremder Text ohne Herkunftsangabe kommt nicht ins Repository.
- **Korpuszahlen tragen nie eine Architekturentscheidung.** Sie beantworten „ist es schlechter geworden", nie „was ist gut".

## Abhängigkeit zum echten Fragensatz

`brain.bench.questions.DEFAULT_SHAPE` steht nach der Erweiterung des echten
Satzes auf 13/13/10/14 — dieselbe Verteilung, die der Korpus verlangt. Der Plan
setzt das voraus und führt **kein** eigenes `--shape` ein. Steht `DEFAULT_SHAPE`
beim Ausführen noch auf 8/8/6/8, ist das die erste Zeile, die zu ändern ist,
nicht ein zweiter Weg daneben.

---

## Dateien im Überblick

| Datei | Verantwortung |
|---|---|
| `bench/corpus/v1/notes/*.md` | die 100 Notizen, je mit Herkunftskopf |
| `bench/corpus/v1/themes.yaml` | die 10 Themen, ihr Nachbarthema, ihre Notizen |
| `bench/corpus/v1/questions.yaml` | die 50 Fragen, Form wie der echte Satz |
| `bench/corpus/v1/manifest.json` | Prüfsumme, Quelle und Lizenz je Notiz |
| `bench/corpus/v1/HERKUNFT.md` | dasselbe für Menschen lesbar |
| `src/brain/bench/corpus.py` | den Stand lesen und **alle** Regelverstöße melden |
| `src/brain/cli.py` | `--corpus` an `brain bench` |
| `tests/test_bench_corpus.py` | der Prüfer, gegen gebaute Stände |
| `tests/test_corpus_v1.py` | der echte Stand `v1` besteht den Prüfer |

---

### Task 1: Der Prüfer

**Files:**
- Create: `src/brain/bench/corpus.py`
- Test: `tests/test_bench_corpus.py`

**Interfaces:**
- Consumes: `brain.bench.questions.Kind`, `load`, `QuestionSetError`
- Produces: `CORPUS_SHAPE: Mapping[Kind, int]`, `REVERSE_MINIMUM = 5`, `NOTES = 100`, `THEMES = 10`, `Theme(name: str, neighbour: str, notes: tuple[str, ...])`, `CorpusError(problems: tuple[str, ...])`, `check(stand: Path, *, earlier: Sequence[Path] = ()) -> None`

`earlier` sind die Stände, die es vor diesem gab. Ein Thema, das dort schon vorkam, ist ein Verstoß: Wiederholte Themen erzeugen Störer über Standgrenzen hinweg, und ein Lauf über zwei Stände ergäbe dann etwas anderes als die Summe der Einzelläufe — genau die Vergleichbarkeit, für die die Nummerierung da ist (Spec §4). Für `v1` ist die Liste leer; der Parameter existiert, damit `v2` ihn nicht erst einführen muss.

`check` prüft einen Stand und wirft `CorpusError` mit **allen** Verstößen, oder kehrt still zurück. Dieselbe Haltung wie beim Fragensatz: Der erste Verstoß ist nicht der interessante, die Liste ist es.

- [ ] **Step 1: Write the failing test**

**Wo `_stand` hingehört:** in `tests/conftest.py`, unter dem Namen
`sound_stand`, **nicht** in das Testmodul. Task 6 baut damit denselben Stand,
und eine Hilfsfunktion zweimal zu schreiben ist genau die Verdopplung, die eine
Prüfung später meldet. Unten steht sie der Lesbarkeit halber im Testmodul —
lege sie beim Umsetzen nach `conftest.py` und importiere sie hier.

Create `tests/test_bench_corpus.py`:

```python
"""A stand that does not pass the checker does not exist (spec 5)."""

import hashlib
import json
from pathlib import Path

import pytest

from brain.bench.corpus import CorpusError, check
from brain.bench.questions import Kind

_SORTS = [
    *["exakt"] * 13,
    *["umschreibung"] * 13,
    *["gemischt"] * 10,
    *["sprachuebergreifend"] * 14,
]


def _opener(number: int, reverse: int) -> str:
    """English only on cross-lingual questions — they are the last fourteen.

    Putting it on the first N would leave the reverse count at zero however
    high `reverse` is: an English opener on an `exakt` question is not a
    reverse question, and `_is_reverse` is right to ignore it.
    """
    first_cross = len(_SORTS) - 14
    return "Which" if first_cross <= number < first_cross + reverse else "Welche"


def _stand(tmp_path: Path, *, notes: int = 100, themes: int = 10, reverse: int = 5) -> Path:
    """A sound stand, built from the smallest thing that satisfies every rule."""
    root = tmp_path / "v1"
    (root / "notes").mkdir(parents=True)
    entries = []
    for number in range(notes):
        note = root / "notes" / f"n{number:03d}.md"
        note.write_text(f"Beleg {number} steht hier.\n", encoding="utf-8")
        entries.append(note)
    manifest = {
        note.name: {
            "sha256": hashlib.sha256(note.read_bytes()).hexdigest(),
            "quelle": "https://de.wikipedia.org/wiki/Beispiel",
            "abgerufen": "2026-08-21",
            "lizenz": "CC BY-SA 4.0",
        }
        for note in entries
    }
    (root / "manifest.json").write_text(json.dumps(manifest, indent=2), encoding="utf-8")
    names = [f"n{n:03d}.md" for n in range(notes)]
    (root / "themes.yaml").write_text(
        "".join(
            f"- name: thema{t}\n  nachbar: thema{(t + 1) % themes}\n  notizen:\n"
            + "".join(f"    - {name}\n" for name in names[t * 10 : t * 10 + 10])
            for t in range(themes)
        ),
        encoding="utf-8",
    )
    (root / "questions.yaml").write_text(
        "".join(
            f"- id: q{number:02d}\n  sort: {sort}\n"
            f'  query: "{_opener(number, reverse)} Frage {number}?"\n'
            f'  expect: "notes/n{number:03d}.md"\n'
            f'  beleg: "Beleg {number} steht hier."\n'
            for number, sort in enumerate(_SORTS)
        ),
        encoding="utf-8",
    )
    (root / "HERKUNFT.md").write_text("# Herkunft\n\nBeispiel, CC BY-SA 4.0\n", encoding="utf-8")
    return root


def test_a_sound_stand_passes(tmp_path: Path) -> None:
    check(_stand(tmp_path))


def test_the_note_count_is_exact(tmp_path: Path) -> None:
    """99 is not "nearly a stand" — a fixed number is what makes two stands comparable."""
    root = _stand(tmp_path)
    (root / "notes" / "n099.md").unlink()

    with pytest.raises(CorpusError) as raised:
        check(root)

    assert any("99 notes, expected 100" in p for p in raised.value.problems)


def test_a_changed_note_is_caught_by_its_checksum(tmp_path: Path) -> None:
    """An unnoticed edit to an old stand silently invalidates every earlier number."""
    root = _stand(tmp_path)
    (root / "notes" / "n000.md").write_text("etwas anderes\n", encoding="utf-8")

    with pytest.raises(CorpusError) as raised:
        check(root)

    assert any("n000.md: checksum" in p for p in raised.value.problems)


def test_a_note_without_provenance_is_refused(tmp_path: Path) -> None:
    root = _stand(tmp_path)
    manifest = json.loads((root / "manifest.json").read_text(encoding="utf-8"))
    del manifest["n001.md"]["lizenz"]
    (root / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")

    with pytest.raises(CorpusError) as raised:
        check(root)

    assert any("n001.md: missing 'lizenz'" in p for p in raised.value.problems)


def test_a_note_missing_from_the_manifest_is_refused(tmp_path: Path) -> None:
    root = _stand(tmp_path)
    manifest = json.loads((root / "manifest.json").read_text(encoding="utf-8"))
    del manifest["n002.md"]
    (root / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")

    with pytest.raises(CorpusError) as raised:
        check(root)

    assert any("n002.md: not in the manifest" in p for p in raised.value.problems)


def test_every_theme_carries_ten_notes(tmp_path: Path) -> None:
    root = _stand(tmp_path)
    text = (root / "themes.yaml").read_text(encoding="utf-8")
    (root / "themes.yaml").write_text(text.replace("    - n009.md\n", "", 1), encoding="utf-8")

    with pytest.raises(CorpusError) as raised:
        check(root)

    assert any("thema0: 9 notes, expected 10" in p for p in raised.value.problems)


def test_a_neighbour_that_is_not_a_theme_is_refused(tmp_path: Path) -> None:
    """The neighbour is what makes the distractors credible; a dangling name has none."""
    root = _stand(tmp_path)
    text = (root / "themes.yaml").read_text(encoding="utf-8")
    (root / "themes.yaml").write_text(
        text.replace("nachbar: thema1", "nachbar: erfunden", 1), encoding="utf-8"
    )

    with pytest.raises(CorpusError) as raised:
        check(root)

    assert any("thema0: neighbour 'erfunden' is not a theme" in p for p in raised.value.problems)


def test_a_theme_that_is_its_own_neighbour_is_refused(tmp_path: Path) -> None:
    root = _stand(tmp_path)
    text = (root / "themes.yaml").read_text(encoding="utf-8")
    (root / "themes.yaml").write_text(
        text.replace("nachbar: thema1", "nachbar: thema0", 1), encoding="utf-8"
    )

    with pytest.raises(CorpusError) as raised:
        check(root)

    assert any("thema0: is its own neighbour" in p for p in raised.value.problems)


def test_a_note_in_two_themes_is_refused(tmp_path: Path) -> None:
    """Ten themes of ten notes must partition the hundred, or a note counts twice."""
    root = _stand(tmp_path)
    text = (root / "themes.yaml").read_text(encoding="utf-8")
    (root / "themes.yaml").write_text(text.replace("- n010.md", "- n000.md", 1), encoding="utf-8")

    with pytest.raises(CorpusError) as raised:
        check(root)

    assert any("n000.md: claimed by two themes" in p for p in raised.value.problems)
    assert any("n010.md: claimed by no theme" in p for p in raised.value.problems)


def test_the_question_distribution_is_enforced(tmp_path: Path) -> None:
    root = _stand(tmp_path)
    text = (root / "questions.yaml").read_text(encoding="utf-8")
    (root / "questions.yaml").write_text(
        text.replace("sort: gemischt", "sort: exakt", 1), encoding="utf-8"
    )

    with pytest.raises(CorpusError) as raised:
        check(root)

    assert any("exakt: 14 questions, expected 13" in p for p in raised.value.problems)


def test_the_reverse_direction_has_a_floor(tmp_path: Path) -> None:
    """Four English questions would leave the reverse direction barely measured."""
    with pytest.raises(CorpusError) as raised:
        check(_stand(tmp_path, reverse=4))

    assert any("4 questions in the reverse direction, expected at least 5" in p
               for p in raised.value.problems)


def test_a_missing_file_is_named_rather_than_crashing(tmp_path: Path) -> None:
    root = _stand(tmp_path)
    (root / "HERKUNFT.md").unlink()

    with pytest.raises(CorpusError) as raised:
        check(root)

    assert any("HERKUNFT.md is missing" in p for p in raised.value.problems)


def test_every_problem_is_reported_at_once(tmp_path: Path) -> None:
    root = _stand(tmp_path)
    (root / "notes" / "n099.md").unlink()
    (root / "HERKUNFT.md").unlink()

    with pytest.raises(CorpusError) as raised:
        check(root)

    assert len(raised.value.problems) >= 2


def test_a_theme_from_an_earlier_stand_is_refused(tmp_path: Path) -> None:
    """Repeated themes make two stands interfere; the numbering then buys nothing."""
    first = _stand(tmp_path / "a")
    second = _stand(tmp_path / "b")

    with pytest.raises(CorpusError) as raised:
        check(second, earlier=[first])

    assert any("thema0: already used in an earlier stand" in p for p in raised.value.problems)


def test_an_earlier_stand_with_other_themes_is_no_problem(tmp_path: Path) -> None:
    first = _stand(tmp_path / "a")
    text = (first / "themes.yaml").read_text(encoding="utf-8")
    (first / "themes.yaml").write_text(text.replace("thema", "alt"), encoding="utf-8")

    check(_stand(tmp_path / "b"), earlier=[first])


def test_the_shape_is_the_fifty_of_the_spec() -> None:
    from brain.bench.corpus import CORPUS_SHAPE, NOTES, REVERSE_MINIMUM, THEMES

    assert sum(CORPUS_SHAPE.values()) == 50
    assert CORPUS_SHAPE[Kind.CROSS_LINGUAL] == 14
    assert (NOTES, THEMES, REVERSE_MINIMUM) == (100, 10, 5)
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `uv run pytest tests/test_bench_corpus.py -v`
Expected: FAIL — `ModuleNotFoundError: No module named 'brain.bench.corpus'`

- [ ] **Step 3: Write the implementation**

Create `src/brain/bench/corpus.py`:

```python
"""The rules of a corpus stand, enforced rather than remembered.

A stand that does not pass this check does not exist (spec 5). The rules are
few and blunt on purpose: fixed counts, a partition, provenance for every file,
and a checksum per note. What they buy is that two numbers measured a year
apart are numbers about the same thing.
"""

import hashlib
import json
from collections.abc import Mapping, Sequence
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml

from brain.bench.questions import Kind, Question, QuestionSetError
from brain.bench.questions import load as load_questions

NOTES = 100
THEMES = 10
REVERSE_MINIMUM = 5
CORPUS_SHAPE: Mapping[Kind, int] = {
    Kind.EXACT: 13,
    Kind.PARAPHRASE: 13,
    Kind.MIXED: 10,
    Kind.CROSS_LINGUAL: 14,
}
_PROVENANCE = ("sha256", "quelle", "abgerufen", "lizenz")
# An English question against a German source. Recognised by its first word
# rather than by language detection: the set is written by hand, and a word
# list that a human can check beats a model nobody can question.
_ENGLISH_OPENERS = frozenset(
    {"which", "why", "what", "how", "when", "where", "who", "does", "is", "can"}
)


@dataclass(frozen=True, slots=True)
class Theme:
    name: str
    neighbour: str
    notes: tuple[str, ...]


class CorpusError(Exception):
    """Everything wrong with the stand, in one exception."""

    def __init__(self, problems: tuple[str, ...]) -> None:
        self.problems = problems
        super().__init__("\n".join(problems))


def check(stand: Path, *, earlier: Sequence[Path] = ()) -> None:
    """Raise with every violation, or return silently.

    `earlier` names the stands that came before this one. A theme that already
    appeared in one of them is refused: repeated themes place distractors
    across stand borders, and a run over two stands would then measure
    something other than the sum of the single runs (spec 4).
    """
    problems: list[str] = []
    for name in ("notes", "themes.yaml", "questions.yaml", "manifest.json", "HERKUNFT.md"):
        if not (stand / name).exists():
            problems.append(f"{name} is missing")
    if problems:
        raise CorpusError(tuple(problems))
    notes = sorted(path.name for path in (stand / "notes").glob("*.md"))
    if len(notes) != NOTES:
        problems.append(f"{len(notes)} notes, expected {NOTES}")
    problems += _manifest_problems(stand, notes)
    problems += _theme_problems(stand, notes)
    problems += _earlier_problems(stand, earlier)
    problems += _question_problems(stand)
    if problems:
        raise CorpusError(tuple(problems))


def _manifest_problems(stand: Path, notes: list[str]) -> list[str]:
    raw: dict[str, Any] = json.loads((stand / "manifest.json").read_text(encoding="utf-8"))
    problems = []
    for name in notes:
        entry = raw.get(name)
        if entry is None:
            problems.append(f"{name}: not in the manifest")
            continue
        problems += [f"{name}: missing {key!r}" for key in _PROVENANCE if key not in entry]
        if "sha256" not in entry:
            continue
        digest = hashlib.sha256((stand / "notes" / name).read_bytes()).hexdigest()
        if digest != entry["sha256"]:
            problems.append(f"{name}: checksum {digest} does not match the manifest")
    return problems


def _earlier_problems(stand: Path, earlier: Sequence[Path]) -> list[str]:
    seen = {theme.name for path in earlier for theme in _themes(path)}
    return [
        f"{theme.name}: already used in an earlier stand"
        for theme in _themes(stand)
        if theme.name in seen
    ]


def _theme_problems(stand: Path, notes: list[str]) -> list[str]:
    themes = _themes(stand)
    problems = []
    if len(themes) != THEMES:
        problems.append(f"{len(themes)} themes, expected {THEMES}")
    names = {theme.name for theme in themes}
    claimed: dict[str, int] = {}
    for theme in themes:
        if len(theme.notes) != 10:
            problems.append(f"{theme.name}: {len(theme.notes)} notes, expected 10")
        if theme.neighbour == theme.name:
            problems.append(f"{theme.name}: is its own neighbour")
        elif theme.neighbour not in names:
            problems.append(f"{theme.name}: neighbour {theme.neighbour!r} is not a theme")
        for note in theme.notes:
            claimed[note] = claimed.get(note, 0) + 1
    problems += [f"{note}: claimed by two themes" for note, count in claimed.items() if count > 1]
    problems += [f"{note}: claimed by no theme" for note in notes if note not in claimed]
    return problems


def _themes(stand: Path) -> list[Theme]:
    raw = yaml.safe_load((stand / "themes.yaml").read_text(encoding="utf-8")) or []
    return [
        Theme(str(entry["name"]), str(entry["nachbar"]), tuple(str(n) for n in entry["notizen"]))
        for entry in raw
    ]


def _question_problems(stand: Path) -> list[str]:
    try:
        questions = load_questions(stand / "questions.yaml", shape=CORPUS_SHAPE)
    except QuestionSetError as broken:
        return list(broken.problems)
    reverse = sum(_is_reverse(question) for question in questions)
    if reverse < REVERSE_MINIMUM:
        return [
            f"{reverse} questions in the reverse direction, "
            f"expected at least {REVERSE_MINIMUM}"
        ]
    return []


def _is_reverse(question: Question) -> bool:
    if question.kind is not Kind.CROSS_LINGUAL:
        return False
    first = question.query.split(maxsplit=1)[0].strip("\"'").lower()
    return first in _ENGLISH_OPENERS
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `uv run pytest tests/test_bench_corpus.py -v`
Expected: PASS.

- [ ] **Step 5: Lint, types, coverage**

Run:
```bash
uv run ruff check src/brain/bench tests/test_bench_corpus.py
uv run ruff format --check src/brain/bench tests/test_bench_corpus.py
uv run mypy
uv run coverage run -m pytest tests/test_bench_corpus.py && uv run coverage report --include "src/brain/bench/corpus.py"
```
Expected: sauber, 100 %. Fehlt eine Zeile, schreibe den Test — kein `pragma`.

- [ ] **Step 6: Commit**

```bash
git add src/brain/bench/corpus.py tests/test_bench_corpus.py
git commit -m "Enforce the rules of a corpus stand instead of remembering them"
```

---

### Task 2: Die zehn Themen festlegen

Kein Prüfcode, aber die Entscheidung, an der alles Folgende hängt. Ergebnis ist eine Datei, die die späteren Tasks abarbeiten.

**Files:**
- Create: `bench/corpus/v1/themes.yaml`

**Interfaces:**
- Produces: die zehn Themennamen und ihre Nachbarpaare, auf die Task 3 und 4 sich beziehen

- [ ] **Step 1: Die fünf Nachbarpaare wählen**

Zehn Themen, fünf Paare. Innerhalb eines Paares teilen die Themen Vokabular — das macht die Störer glaubhaft. Vorschlag, an dem sich die Auswahl orientiert (Wikipedia führt jedes davon in beiden Sprachen):

| # | Thema | Nachbar | prüft vor allem |
|---|---|---|---|
| 1 | Netzwerkprotokolle | Netzwerksicherheit | exakte Bezeichner (Portnummern, RFC-Nummern) |
| 2 | Netzwerksicherheit | Netzwerkprotokolle | dieselbe Sorte, überschneidendes Vokabular |
| 3 | Baustatik | Baustoffe | exakte Größen mit Einheiten |
| 4 | Baustoffe | Baustatik | Umschreibung (Alltagssprache über Material) |
| 5 | Meeresbiologie | Meereskunde | Umschreibung |
| 6 | Meereskunde | Meeresbiologie | sprachübergreifend (Fachwörter in beiden Sprachen verschieden) |
| 7 | Vertragsrecht | Gesellschaftsrecht | Umschreibung, lange Prosa |
| 8 | Gesellschaftsrecht | Vertragsrecht | exakte Paragraphen |
| 9 | Kaffeeanbau | Teeanbau | sprachübergreifend, Alltagsnähe |
| 10 | Teeanbau | Kaffeeanbau | gemischt (deutsche Prosa, englische Sortennamen) |

Die Auswahl darf abweichen; verbindlich ist nur: zehn Themen, fünf Nachbarpaare, jedes Thema in beiden Sprachen bei der Quelle vorhanden, und die vier Fragesorten über die Themen abgedeckt.

- [ ] **Step 2: `themes.yaml` anlegen**

```yaml
- name: netzwerkprotokolle
  nachbar: netzwerksicherheit
  notizen: []   # von Task 3 gefüllt
- name: netzwerksicherheit
  nachbar: netzwerkprotokolle
  notizen: []
```

…für alle zehn. Die `notizen`-Listen bleiben in diesem Task leer; Task 3 trägt die Dateinamen ein, sobald die Notizen existieren.

- [ ] **Step 3: Commit**

```bash
git add bench/corpus/v1/themes.yaml
git commit -m "Choose the ten themes of corpus v1 and their neighbours"
```

---

### Task 3: Die ersten fünfzig Notizen (Themen 1–5)

**Files:**
- Create: `bench/corpus/v1/notes/*.md` (50 Stück)
- Modify: `bench/corpus/v1/themes.yaml` (die `notizen`-Listen der Themen 1–5)
- Modify: `bench/corpus/v1/manifest.json`, `bench/corpus/v1/HERKUNFT.md`

**Interfaces:**
- Consumes: die Themennamen aus Task 2
- Produces: 50 Notizen und ihre Herkunftseinträge

- [ ] **Step 1: Je Thema zehn Artikel wählen**

Zehn Artikel je Thema, aus Wikipedia. Verbindlich:

- **Sprache bewusst wählen.** Ein Thema, das sprachübergreifend prüfen soll, bekommt seine Notizen in **einer** Sprache — die Frage stellt später die andere. Für die übrigen Themen mischen, wie es der echte Bestand auch tut.
- **Länge:** 150 bis 600 Wörter je Notiz. Ein ganzer Wikipedia-Artikel ist zu lang; schneide einen zusammenhängenden Abschnitt heraus, statt Sätze zusammenzuwürfeln.
- **Kein Umschreiben.** Der Text wird übernommen, nicht verbessert — sonst steht im Korpus etwas, das nirgends herkommt, und die Lizenzangabe wird falsch.

- [ ] **Step 2: Notizformat**

Jede Notiz trägt ihren Herkunftskopf im Frontmatter, damit sie auch außerhalb des Standes deutbar bleibt:

```markdown
---
titel: Transmission Control Protocol
quelle: https://de.wikipedia.org/wiki/Transmission_Control_Protocol
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerkprotokolle
---

<der übernommene Abschnitt>
```

Dateiname: `<thema>-<zweistellige-nummer>.md`, etwa `netzwerkprotokolle-01.md`.

- [ ] **Step 3: `themes.yaml` füllen**

Trage die zehn Dateinamen je Thema ein. Kein Name darf in zwei Themen stehen — der Prüfer aus Task 1 weist das ab.

- [ ] **Step 4: Manifest und Herkunft schreiben**

`manifest.json`, ein Eintrag je Notiz:

```json
{
  "netzwerkprotokolle-01.md": {
    "sha256": "<sha256 der Datei>",
    "quelle": "https://de.wikipedia.org/wiki/Transmission_Control_Protocol",
    "abgerufen": "2026-08-21",
    "lizenz": "CC BY-SA 4.0"
  }
}
```

Die Prüfsumme über die Datei, wie sie auf der Platte liegt:

```bash
uv run python -c "
import hashlib, json, pathlib
notes = sorted(pathlib.Path('bench/corpus/v1/notes').glob('*.md'))
print(json.dumps({p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in notes}, indent=2))"
```

`HERKUNFT.md` führt dasselbe als Liste für Menschen, gruppiert nach Thema, mit einem einleitenden Absatz, der die Lizenz nennt und was sie verlangt.

- [ ] **Step 5: Zwischenstand prüfen**

Der Prüfer wird noch fehlschlagen — 50 statt 100 Notizen, keine Fragen. Das ist erwartet. Was jetzt schon stimmen muss, prüfst du gezielt:

```bash
uv run python -c "
from pathlib import Path
from brain.bench.corpus import CorpusError, check
try:
    check(Path('bench/corpus/v1'))
except CorpusError as broken:
    for p in broken.problems: print(p)"
```

Erwartet: **nur** Meldungen über die fehlende Hälfte (`50 notes, expected 100`, `claimed by no theme` für nichts, fehlende `questions.yaml`). Jede andere Meldung — eine falsche Prüfsumme, eine fehlende Lizenz, ein doppelt beanspruchter Name — ist jetzt zu beheben, nicht später.

- [ ] **Step 6: Commit**

```bash
git add bench/corpus/v1
git commit -m "Add the first fifty corpus notes: themes one to five"
```

---

### Task 4: Die zweiten fünfzig Notizen (Themen 6–10)

**Files:**
- Create: `bench/corpus/v1/notes/*.md` (50 weitere)
- Modify: `bench/corpus/v1/themes.yaml`, `manifest.json`, `HERKUNFT.md`

**Interfaces:**
- Consumes: das Format und die Konventionen aus Task 3
- Produces: die vollständigen 100 Notizen

- [ ] **Step 1: Wie Task 3, für die Themen 6 bis 10**

Dieselben Regeln, wörtlich: zehn Artikel je Thema, 150–600 Wörter, zusammenhängender Abschnitt, kein Umschreiben, Herkunftskopf im Frontmatter, Dateiname `<thema>-<nummer>.md`.

Notizformat zur Erinnerung — es steht hier noch einmal, weil du diesen Task auch ohne Task 3 lesen können sollst:

```markdown
---
titel: <Artikelname>
quelle: <URL>
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: <themenname>
---

<der übernommene Abschnitt>
```

- [ ] **Step 2: `themes.yaml`, `manifest.json` und `HERKUNFT.md` vervollständigen**

Prüfsummen wie in Task 3:

```bash
uv run python -c "
import hashlib, json, pathlib
notes = sorted(pathlib.Path('bench/corpus/v1/notes').glob('*.md'))
print(json.dumps({p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in notes}, indent=2))"
```

- [ ] **Step 3: Der Notizteil muss jetzt vollständig durchgehen**

```bash
uv run python -c "
from pathlib import Path
from brain.bench.corpus import CorpusError, check
try:
    check(Path('bench/corpus/v1'))
except CorpusError as broken:
    for p in broken.problems: print(p)"
```

Erwartet: **ausschließlich** Meldungen über die fehlende `questions.yaml`. Steht dort noch eine Zeile über Notizen, Themen oder Prüfsummen, ist dieser Task nicht fertig.

- [ ] **Step 4: Commit**

```bash
git add bench/corpus/v1
git commit -m "Add the second fifty corpus notes: themes six to ten"
```

---

### Task 5: Die fünfzig Fragen

**Files:**
- Create: `bench/corpus/v1/questions.yaml`

**Interfaces:**
- Consumes: die 100 Notizen und `themes.yaml`
- Produces: den prüffähigen Stand — nach diesem Task läuft `check` grün

- [ ] **Step 1: Die Verteilung aufteilen**

50 Fragen: **13 exakt, 13 umschreibung, 10 gemischt, 14 sprachuebergreifend**, davon **mindestens 5** mit englischer Frage auf deutsche Quelle. Verteile sie über die zehn Themen, damit kein Thema ungeprüft bleibt und keines die Messung dominiert — fünf Fragen je Thema geht auf.

- [ ] **Step 2: Die Fragen schreiben**

Form wie beim echten Satz; `expect` ist **relativ zum Stand** (`notes/<datei>.md`), weil `load` relative Pfade gegen das Verzeichnis der Fragendatei auflöst:

```yaml
- id: c01
  sort: exakt
  query: "RFC 793 Verbindungsaufbau Dreiwege-Handschlag"
  expect: "notes/netzwerkprotokolle-01.md"
  beleg: "Der Verbindungsaufbau erfolgt über einen Drei-Wege-Handschlag."
  hinweis: "Die RFC-Nummer steht nur in dieser Notiz."
```

Die Regeln je Sorte, dieselben wie beim echten Satz:
- **exakt:** ein seltener Bezeichner, eine Nummer, eine Größe mit Einheit; er kommt in genau einer Notiz vor.
- **umschreibung:** Frage und Notiz teilen möglichst kein Stichwort — nur Bedeutung trägt.
- **gemischt:** deutsche Frage mit englischem Fachwort aus der Notiz.
- **sprachuebergreifend:** Frage in der einen, Notiz in der anderen Sprache, ohne gemeinsame Eigennamen als Brücke.

**Die Störer sind der Zweck.** Zu jeder Frage muss es Notizen im selben Thema und im Nachbarthema geben, die dasselbe Vokabular tragen und die Antwort **nicht** enthalten. Prüfe das bei jeder Frage: Wäre die Zielnotiz die einzige zum Thema, prüft die Frage nichts.

- [ ] **Step 3: Der Stand muss jetzt grün sein**

```bash
uv run python -c "
from pathlib import Path
from brain.bench.corpus import check
check(Path('bench/corpus/v1'))
print('v1 besteht den Prüfer')"
```

Erwartet: `v1 besteht den Prüfer`. Jede Meldung ist zu beheben, nicht wegzuargumentieren.

- [ ] **Step 4: Den Stand in die Prüfstrecke hängen**

Create `tests/test_corpus_v1.py`:

```python
"""The checked-in stand passes its own checker, on every run of the suite.

Without this test the rules of spec 5 hold only when somebody remembers to run
them by hand — and a stand nobody checks is a stand that drifts.
"""

from pathlib import Path

from brain.bench.corpus import check


def test_the_committed_stand_v1_passes_the_checker() -> None:
    check(Path(__file__).resolve().parent.parent / "bench" / "corpus" / "v1")
```

- [ ] **Step 5: Die Prüfstrecke laufen lassen**

Run: `uv run pytest tests/test_corpus_v1.py -v && uv run pytest`
Expected: beide grün.

- [ ] **Step 6: Commit**

```bash
git add bench/corpus/v1/questions.yaml tests/test_corpus_v1.py
git commit -m "Add the fifty corpus questions and pin the stand in the suite"
```

---

### Task 6: `brain bench --corpus`

**Files:**
- Modify: `src/brain/cli.py`
- Test: `tests/test_cli.py`

**Interfaces:**
- Consumes: `brain.bench.corpus.check`, `brain.search.qmd_config.sync_collections`, `CollectionSpec`, `brain.registry`, das bestehende `_bench`
- Produces: `brain bench --corpus <pfad>`

**Was der Schalter tut**, in dieser Reihenfolge:

1. `check(pfad)` — ein Stand, der die Regeln verletzt, wird nicht gemessen.
2. Einen **Wegwerf-Zustandsraum** anlegen (temporäres Verzeichnis) mit einer Registry, die genau einen schreibgeschützten Bereich `corpus` auf `<pfad>/notes` führt. Damit läuft die Messung durch **dieselbe** Kette wie im Alltag — Sichtbarkeit, Entdopplung, Datenschutzfilter —, statt an ihr vorbei.
3. Die qmd-Sammlung `corpus` auf `<pfad>/notes` setzen (`sync_collections`), `refresh` und `embed` fahren.
4. Den normalen Messweg laufen lassen: `--scope corpus`, Fragensatz `<pfad>/questions.yaml`.
5. Im Protokollkopf den **Stand** nennen (`bench/corpus/v1`) und den Satz, dass diese Zahlen **Regression** messen und **keine Entscheidung tragen**.

`--corpus` schließt `--scope` und `--questions` aus: Der Stand bestimmt beide. Wer sie zusammen angibt, bekommt eine Fehlermeldung, die das sagt.

- [ ] **Step 1: Write the failing test**

Append to `tests/test_cli.py`:

```python
def test_bench_corpus_refuses_a_stand_that_breaks_its_rules(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    """An unsound stand is not measured — its numbers would mean nothing."""
    stand = tmp_path / "v1"
    (stand / "notes").mkdir(parents=True)
    monkeypatch.setattr("brain.cli._port", lambda: FakePort(results=[]))

    code = main(["bench", "--corpus", str(stand), "--out", str(tmp_path)])

    assert code == 1
    assert "themes.yaml is missing" in capsys.readouterr().err
    assert not list(tmp_path.glob("bench-*"))


def test_bench_corpus_and_scope_together_are_refused(
    tmp_path: Path, capsys: pytest.CaptureFixture[str]
) -> None:
    """The stand names its own scope; two answers to one question is a bug report."""
    code = main(["bench", "--corpus", str(tmp_path), "--scope", "knowledge"])

    assert code == 1
    assert "--corpus" in capsys.readouterr().err
```

Dazu ein Test, der den vollständigen Weg gegen einen gebauten Stand fährt. Er baut denselben Stand wie `tests/test_bench_corpus.py::_stand` — importiere dessen Hilfsfunktion nicht, sondern lege sie nach `tests/conftest.py`, damit beide Testmodule sie teilen, statt sie zu verdoppeln:

```python
def test_bench_corpus_measures_through_the_ordinary_chain(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    stand = sound_stand(tmp_path)          # aus conftest
    hits = [[_corpus_hit(stand, number)] for number in range(50)]
    port = FakePort(results=hits, indexed={"corpus": [f"n{n:03d}.md" for n in range(100)]})
    monkeypatch.setattr("brain.cli._port", lambda: port)
    monkeypatch.setattr("brain.cli._now", lambda: "2026-08-21-1500")
    monkeypatch.setattr("brain.cli._qmd_version", lambda: "qmd 2.8.3")
    monkeypatch.setattr("brain.cli.qmd_config_path", lambda: tmp_path / "qmd" / "index.yml")

    code = main(["bench", "--corpus", str(stand), "--out", str(tmp_path)])

    assert code == 0
    text = (tmp_path / "bench-2026-08-21-1500-fast.md").read_text(encoding="utf-8")
    assert "| gesamt | 50/50 |" in text
    assert "Regression" in text and "keine Entscheidung" in text
    assert str(stand) in text
    assert port.refreshed and port.embedded          # der Stand wurde indexiert
```

`_corpus_hit` baut einen `SearchHit` auf `collection="corpus"` und `relative=f"n{number:03d}.md"`; schreibe ihn neben die anderen Hilfsfunktionen in `tests/test_cli.py`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `uv run pytest tests/test_cli.py -v -k corpus`
Expected: FAIL — `argparse` kennt `--corpus` nicht, `SystemExit: 2`.

- [ ] **Step 3: Write the implementation**

In `src/brain/cli.py`, Parser:

```python
    bench_parser.add_argument("--corpus", type=Path, default=None)
```

Und in `_bench`, vor allem anderen:

```python
def _corpus_run(args: argparse.Namespace) -> tuple[Path, Path, str]:
    """Prepare a stand for measurement: check it, register it, index it.

    A throwaway state directory rather than the user's own: the corpus is not
    part of anybody's knowledge, and a run that measures it must not leave a
    registered area behind. Measuring through a registered area at all — rather
    than asking the engine directly — is deliberate: the number is only
    comparable to an everyday number if it went through the everyday chain.
    """
    stand = args.corpus.resolve()
    check(stand)
    state_dir = Path(tempfile.mkdtemp(prefix="brain-corpus-"))
    (state_dir / "registry.toml").write_text(
        f'[[area]]\nscope = "corpus"\npath = "{(stand / "notes").as_posix()}"\nreadonly = true\n',
        encoding="utf-8",
    )
    area_state = area_state_dir(state_dir, "corpus")
    area_state.mkdir(parents=True)
    (area_state / ".brain.toml").write_text(
        '[area]\nscope = "corpus"\n\n[index]\ninclude = ["**/*.md"]\n', encoding="utf-8"
    )
    sync_collections(
        qmd_config_path(), {"corpus": CollectionSpec(path=stand / "notes", pattern="**/*.md")}
    )
    return stand, state_dir, "corpus"
```

Die genaue Form von `CollectionSpec` liest du in `src/brain/search/qmd_config.py` nach, bevor du sie aufrufst — sie trägt heute die Felder, die `sync_collections` erwartet, und dieser Plan rät sie nicht.

`_bench` ruft `_corpus_run` auf, wenn `args.corpus` gesetzt ist, benutzt den zurückgegebenen Zustandsraum und Bereich statt `args.scope`/`args.state_dir`, ruft `port.refresh(("corpus",))` und `port.embed(("corpus",))` vor der Messung, und reicht den Stand in den Kopf durch. Der Ausschluss:

```python
    if args.corpus is not None and (args.scope != _SCOPE_DEFAULT or args.questions is not None):
        raise BenchError("--corpus names its own scope and question set; drop --scope/--questions")
```

Im Protokoll (`report.py`) bekommt `Environment` ein Feld `corpus: str | None`; ist es gesetzt, druckt der Kopf den Stand und darunter den Satz:

> Diese Zahlen messen **Regression** gegen einen künstlichen Bestand. Sie sagen, ob die Kette schlechter geworden ist als beim letzten Stand — und **keine** von ihnen trägt eine Architekturentscheidung. Dafür ist der echte Fragensatz da.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `uv run pytest tests/test_cli.py -v -k corpus`
Expected: PASS.

- [ ] **Step 5: Volles Tor**

Run:
```bash
uv run pytest
uv run ruff check src tests
uv run ruff format --check src tests
uv run mypy
uv run coverage run -m pytest && uv run coverage report
```
Expected: alles grün, Coverage 100 %.

- [ ] **Step 6: Commit**

```bash
git add src/brain/cli.py src/brain/bench/report.py tests/test_cli.py tests/conftest.py
git commit -m "Measure a corpus stand through the ordinary chain"
```

---

### Task 7: Der Ausgangswert

Kein Code. Der erste echte Lauf gegen `v1`, und das Festhalten seiner Zahl als **Ausgangswert** — nicht als Bewertung.

**Files:**
- Create: das Protokoll (Ablageort nach Wahl des Ausführenden, per `--out`)
- Modify: `docs/.superpowers/specs/2026-08-21-pruefkorpus-design.md` (Fertig-Kriterium abhaken)

- [ ] **Step 1: Messen**

```bash
uv run brain bench --corpus bench/corpus/v1 --out "<messverzeichnis>"
uv run brain bench --corpus bench/corpus/v1 --profile full --out "<messverzeichnis>"
```

Der erste Lauf ist kalt und indexiert den Stand — er dauert. Der zweite zeigt, ob die beiden Profile sich auf künstlichem Material anders verhalten als auf echtem.

- [ ] **Step 2: Den Ausgangswert festhalten**

Trage die Zahl in die Spec ein, mit dem Datum, dem qmd-Stand und den Modellnamen aus dem Protokollkopf. **Formuliere sie als Ausgangswert**, nicht als Note: „v1 liegt bei x/50 unter qmd 2.8.3 mit <Modellen>; jede spätere Messung wird hiergegen gehalten." Ob x/50 gut ist, sagt diese Zahl nicht und soll sie nicht sagen.

- [ ] **Step 3: Fertig-Kriterium abhaken**

Die fünf Punkte in §7 der Spec durchgehen und abhaken, was erfüllt ist.

- [ ] **Step 4: Commit**

```bash
git add docs/.superpowers/specs/2026-08-21-pruefkorpus-design.md
git commit -m "Record the corpus v1 baseline"
```

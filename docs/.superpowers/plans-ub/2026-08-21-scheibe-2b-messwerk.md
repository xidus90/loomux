# Scheibe 2b — Messwerk: Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `brain bench` bauen — ein Messwerk, das Trefferqualität an einem geprüften Fragensatz und Latenz je Operation misst und beides protokolliert, damit die offenen Entscheidungen aus §7.3, §13 und §16.3 an Zahlen fallen statt an Vermutungen.

**Architecture:** Vier Module unter `src/brain/bench/`, jedes mit einer Aufgabe und ohne Kenntnis von qmd. Das Messwerk bekommt Funktionen hereingereicht — eine Frage-Funktion, eine Uhr —, nie einen Prozess. Der CLI-Unterbefehl setzt beides zusammen, holt den Umgebungskopf und schreibt Protokoll und JSON.

**Tech Stack:** Python 3.14, argparse, PyYAML, `statistics.median`, pytest, ruff, mypy, coverage. Keine neue Abhängigkeit.

**Spec:** `docs/.superpowers/specs/2026-08-21-scheibe-2b-messwerk-design.md`

## Global Constraints

- Python `>=3.14`; ruff `target-version = "py314"`, mypy `python_version = 3.14`, `strict = true`.
- Immer `uv`, nie `pip`. Tests: `uv run pytest`, Werkzeuge: `uv run ruff`, `uv run mypy`, `uv run coverage`.
- **TDD**: erst der fehlschlagende Test, dann die Implementierung. Jeder Task endet mit einem Commit.
- **100 % Coverage** (`fail_under = 100`). Ausschlüsse nur als `# pragma: no cover  # <Grund>`.
- Code, Bezeichner, Code-Kommentare, Commit-Nachrichten: **Englisch**. Prosa und Dokumente: **Deutsch**.
- Imports stehen auf Modulebene. Ein lokaler Import braucht einen begründenden Kommentar.
- Zeilenlänge 100. Lint-Regeln `E, F, I, N, UP, B, SIM, RUF`.
- Kein Test startet einen Prozess oder liest den echten Vault. `tests/conftest.py::never_the_real_qmd` läuft automatisch.
- Datenfelder der YAML-Datei bleiben deutsch (`sort`, `beleg`, `hinweis`) — sie sind das Dateiformat, nicht der Code. Die Python-Bezeichner daneben sind englisch.

---

## Dateien im Überblick

| Datei | Verantwortung |
|---|---|
| `src/brain/bench/__init__.py` | leer, macht das Paket |
| `src/brain/bench/questions.py` | Fragensatz lesen, **alle** Regelverstöße auf einmal melden |
| `src/brain/bench/quality.py` | je Frage: Rang der Zielquelle, Treffer ja/nein, verbrauchte Zeit |
| `src/brain/bench/latency.py` | je Operation: ein kalter Lauf, `repeat` warme, Median/Min/Max |
| `src/brain/bench/report.py` | Auswertung nach Sorte, Markdown-Protokoll, JSON, Vorbehalte |
| `src/brain/search/qmd_config.py` | erweitert um `models()` — Modellnamen für den Protokollkopf |
| `src/brain/cli.py` | erweitert um den Unterbefehl `bench` |
| `tests/test_bench_questions.py` … `test_bench_report.py`, `test_cli.py` | die Tests dazu |

**Was der Fragensatzprüfer nicht fest verdrahtet:** die Satzgröße. Vorgabe ist 30 in der Verteilung 8/8/6/8 (§13), aber sie kommt als Parameter herein — der Prüfkorpus der nächsten Scheibe verlangt 50 in 13/13/10/14, und ein hartkodiertes 30 wäre dort eine Änderung an geprüftem Code statt eines Arguments.

---

### Task 1: Fragensatz lesen und prüfen

**Files:**
- Create: `src/brain/bench/__init__.py`
- Create: `src/brain/bench/questions.py`
- Test: `tests/test_bench_questions.py`

**Interfaces:**
- Consumes: nichts
- Produces: `Kind` (StrEnum: `EXACT="exakt"`, `PARAPHRASE="umschreibung"`, `MIXED="gemischt"`, `CROSS_LINGUAL="sprachuebergreifend"`), `Question(id: str, kind: Kind, query: str, expect: Path, evidence: str, note: str | None)`, `DEFAULT_SHAPE: Mapping[Kind, int]`, `QuestionSetError(problems: tuple[str, ...])`, `load(path: Path, *, shape: Mapping[Kind, int] = DEFAULT_SHAPE) -> tuple[Question, ...]`

- [ ] **Step 1: Write the failing test**

Create `tests/test_bench_questions.py`:

```python
"""The question set is checked before anything is measured, and completely."""

from pathlib import Path

import pytest

from brain.bench.questions import Kind, QuestionSetError, load

_SHAPE = {Kind.EXACT: 1, Kind.PARAPHRASE: 1, Kind.MIXED: 1, Kind.CROSS_LINGUAL: 1}


def _entry(tmp_path: Path, number: int, kind: Kind, evidence: str = "the line") -> str:
    """One valid entry plus the file it points at."""
    target = tmp_path / f"note{number}.md"
    target.write_text(f"before\n{evidence}\nafter\n", encoding="utf-8")
    return (
        f"- id: q{number:02d}\n"
        f"  sort: {kind.value}\n"
        f'  query: "frage {number}"\n'
        f'  expect: "{target.as_posix()}"\n'
        f'  beleg: "{evidence}"\n'
    )


def _write(tmp_path: Path, body: str) -> Path:
    path = tmp_path / "questions.yaml"
    path.write_text(body, encoding="utf-8")
    return path


def _four(tmp_path: Path) -> str:
    return "".join(_entry(tmp_path, n, kind) for n, kind in enumerate(Kind, start=1))


def test_a_sound_set_loads(tmp_path: Path) -> None:
    questions = load(_write(tmp_path, _four(tmp_path)), shape=_SHAPE)

    assert [q.id for q in questions] == ["q01", "q02", "q03", "q04"]
    assert questions[0].kind is Kind.EXACT
    assert questions[0].expect.is_absolute()
    assert questions[0].note is None


def test_the_optional_hint_is_carried(tmp_path: Path) -> None:
    body = _four(tmp_path) + '  hinweis: "weil"\n'

    assert load(_write(tmp_path, body), shape=_SHAPE)[3].note == "weil"


def test_every_violation_is_reported_at_once(tmp_path: Path) -> None:
    """The first violation is not the interesting one — the list is."""
    body = (
        _entry(tmp_path, 1, Kind.EXACT)
        + _entry(tmp_path, 2, Kind.EXACT)
        + "- id: q02\n"
        "  sort: erfunden\n"
        '  query: "x"\n'
        f'  expect: "{(tmp_path / "gone.md").as_posix()}"\n'
        '  beleg: "y"\n'
    )

    with pytest.raises(QuestionSetError) as raised:
        load(_write(tmp_path, body), shape=_SHAPE)

    problems = raised.value.problems
    assert any("unknown sort 'erfunden'" in p for p in problems)
    assert any("duplicate id 'q02'" in p for p in problems)
    assert any("expect does not exist" in p for p in problems)
    assert any("umschreibung: 0 questions, expected 1" in p for p in problems)
    assert any("exakt: 2 questions, expected 1" in p for p in problems)


def test_evidence_must_stand_verbatim_in_the_target(tmp_path: Path) -> None:
    """A set whose evidence has drifted measures a corpus that no longer exists."""
    body = _four(tmp_path).replace('beleg: "the line"', 'beleg: "not in there"', 1)

    with pytest.raises(QuestionSetError) as raised:
        load(_write(tmp_path, body), shape=_SHAPE)

    assert any("evidence not found verbatim" in p for p in raised.value.problems)


def test_a_missing_field_names_the_entry(tmp_path: Path) -> None:
    body = _four(tmp_path) + "- id: q05\n  sort: exakt\n"

    with pytest.raises(QuestionSetError) as raised:
        load(_write(tmp_path, body), shape=_SHAPE)

    assert any("q05: missing field 'query'" in p for p in raised.value.problems)
    assert any("q05: missing field 'expect'" in p for p in raised.value.problems)


def test_an_entry_without_an_id_is_named_by_position(tmp_path: Path) -> None:
    body = _four(tmp_path) + '- sort: exakt\n  query: "x"\n'

    with pytest.raises(QuestionSetError) as raised:
        load(_write(tmp_path, body), shape=_SHAPE)

    assert any("entry 5: missing field 'id'" in p for p in raised.value.problems)


def test_a_file_that_is_not_a_list_is_refused(tmp_path: Path) -> None:
    with pytest.raises(QuestionSetError) as raised:
        load(_write(tmp_path, "id: q01\n"), shape=_SHAPE)

    assert raised.value.problems == ("the question set must be a list of entries, found dict",)


def test_the_default_shape_is_the_thirty_of_the_spec() -> None:
    from brain.bench.questions import DEFAULT_SHAPE

    assert sum(DEFAULT_SHAPE.values()) == 30
    assert DEFAULT_SHAPE[Kind.CROSS_LINGUAL] == 8


def test_the_message_carries_every_problem() -> None:
    error = QuestionSetError(("a", "b"))

    assert str(error) == "a\nb"
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `uv run pytest tests/test_bench_questions.py -v`
Expected: FAIL — `ModuleNotFoundError: No module named 'brain.bench'`

- [ ] **Step 3: Write the implementation**

Create `src/brain/bench/__init__.py` — empty file.

Create `src/brain/bench/questions.py`:

```python
"""The question set, read and checked before a single measurement runs.

Every violation is collected and reported together. Stopping at the first one
would turn fixing a set into a queue of runs, and the person fixing it cannot
see how much is wrong until the last problem is gone.
"""

from collections.abc import Mapping
from dataclasses import dataclass
from enum import StrEnum
from pathlib import Path
from typing import Any

import yaml


class Kind(StrEnum):
    """The four sorts of spec 13. The values are the file format, hence German."""

    EXACT = "exakt"
    PARAPHRASE = "umschreibung"
    MIXED = "gemischt"
    CROSS_LINGUAL = "sprachuebergreifend"


DEFAULT_SHAPE: Mapping[Kind, int] = {
    Kind.EXACT: 8,
    Kind.PARAPHRASE: 8,
    Kind.MIXED: 6,
    Kind.CROSS_LINGUAL: 8,
}


@dataclass(frozen=True, slots=True)
class Question:
    id: str
    kind: Kind
    query: str
    expect: Path
    evidence: str
    note: str | None = None


class QuestionSetError(Exception):
    """Everything wrong with the set, in one exception."""

    def __init__(self, problems: tuple[str, ...]) -> None:
        self.problems = problems
        super().__init__("\n".join(problems))


def load(path: Path, *, shape: Mapping[Kind, int] = DEFAULT_SHAPE) -> tuple[Question, ...]:
    """Read the set and check it, or raise with every problem found.

    `shape` is a parameter rather than the constant of spec 13 because the
    versioned test corpus of the next slice has a different one; hard-coding
    thirty here would make that a code change instead of an argument.
    """
    raw = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(raw, list):
        raise QuestionSetError(
            (f"the question set must be a list of entries, found {type(raw).__name__}",)
        )
    problems: list[str] = []
    questions: list[Question] = []
    seen: set[str] = set()
    for position, entry in enumerate(raw, start=1):
        question = _entry(entry, position, seen, problems)
        if question is not None:
            questions.append(question)
    problems += _shape_problems(questions, shape)
    if problems:
        raise QuestionSetError(tuple(problems))
    return tuple(questions)


def _entry(
    entry: object, position: int, seen: set[str], problems: list[str]
) -> Question | None:
    if not isinstance(entry, dict):
        problems.append(f"entry {position}: must be a mapping, found {type(entry).__name__}")
        return None
    fields: dict[str, Any] = entry
    name = str(fields["id"]) if "id" in fields else f"entry {position}"
    missing = [key for key in ("id", "sort", "query", "expect", "beleg") if key not in fields]
    for key in missing:
        problems.append(f"{name}: missing field {key!r}")
    if missing:
        return None
    if name in seen:
        problems.append(f"{name}: duplicate id {name!r}")
    seen.add(name)
    kind = _kind(str(fields["sort"]), name, problems)
    expect = Path(str(fields["expect"]))
    evidence = str(fields["beleg"])
    _target_problems(expect, evidence, name, problems)
    if kind is None:
        return None
    note = str(fields["hinweis"]) if "hinweis" in fields else None
    return Question(name, kind, str(fields["query"]), expect, evidence, note)


def _kind(value: str, name: str, problems: list[str]) -> Kind | None:
    try:
        return Kind(value)
    except ValueError:
        problems.append(f"{name}: unknown sort {value!r}")
        return None


def _target_problems(expect: Path, evidence: str, name: str, problems: list[str]) -> None:
    if not expect.is_file():
        problems.append(f"{name}: expect does not exist: {expect}")
        return
    if evidence not in expect.read_text(encoding="utf-8"):
        problems.append(f"{name}: evidence not found verbatim in {expect}")


def _shape_problems(questions: list[Question], shape: Mapping[Kind, int]) -> list[str]:
    counted = {kind: 0 for kind in Kind}
    for question in questions:
        counted[question.kind] += 1
    return [
        f"{kind.value}: {counted[kind]} questions, expected {wanted}"
        for kind, wanted in shape.items()
        if counted[kind] != wanted
    ]
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `uv run pytest tests/test_bench_questions.py -v`
Expected: PASS, alle Tests.

- [ ] **Step 5: Lint, types, coverage of this module**

Run:
```bash
uv run ruff check src/brain/bench tests/test_bench_questions.py
uv run ruff format --check src/brain/bench tests/test_bench_questions.py
uv run mypy
uv run coverage run -m pytest tests/test_bench_questions.py && uv run coverage report --include "src/brain/bench/questions.py"
```
Expected: keine Befunde, `questions.py` bei 100 %. Fehlt eine Zeile, schreibe den fehlenden Test — nicht ein `pragma`.

- [ ] **Step 6: Commit**

```bash
git add src/brain/bench tests/test_bench_questions.py
git commit -m "Read the bench question set and report every violation at once"
```

---

### Task 2: Trefferqualität

**Files:**
- Create: `src/brain/bench/quality.py`
- Test: `tests/test_bench_quality.py`

**Interfaces:**
- Consumes: `brain.bench.questions.Question`, `Kind`
- Produces: `TOP = 3`, `Ask = Callable[[str], tuple[Path, ...]]`, `Outcome(question: Question, rank: int | None, hit: bool, elapsed_ms: float)`, `run(questions: Sequence[Question], *, ask: Ask, clock: Callable[[], float]) -> tuple[Outcome, ...]`

`ask` gibt die Trefferpfade in Rangfolge zurück, absolut. Wer sie beschafft, entscheidet der Aufrufer — im Test eine Liste, in der CLI die echte Suchkette. `clock` liefert Sekunden wie `time.perf_counter`.

- [ ] **Step 1: Write the failing test**

Create `tests/test_bench_quality.py`:

```python
"""Success is the rank of the expected file, never the score (spec 13)."""

from collections.abc import Iterator
from pathlib import Path

import pytest

from brain.bench.questions import Kind, Question
from brain.bench.quality import TOP, Outcome, run
from brain.search.port import SearchUnavailable


def _question(number: int, expect: Path) -> Question:
    return Question(f"q{number:02d}", Kind.EXACT, f"frage {number}", expect, "beleg")


def _clock(steps: list[float]) -> "Iterator[float]":
    return iter(steps)


def test_the_expected_file_in_first_place_is_a_hit(tmp_path: Path) -> None:
    target = tmp_path / "a.md"
    ticks = _clock([1.0, 1.25])

    outcomes = run(
        [_question(1, target)],
        ask=lambda _: (target, tmp_path / "b.md"),
        clock=lambda: next(ticks),
    )

    assert outcomes == (Outcome(_question(1, target), rank=1, hit=True, elapsed_ms=250.0),)


def test_rank_three_still_counts_and_rank_four_does_not(tmp_path: Path) -> None:
    """The promise of spec 13 is the first three, not the first page."""
    target = tmp_path / "a.md"
    others = [tmp_path / f"o{n}.md" for n in range(4)]
    ticks = _clock([0.0, 0.0, 0.0, 0.0])

    third, fourth = run(
        [_question(1, target), _question(2, target)],
        ask=lambda query: (
            (*others[:2], target) if query == "frage 1" else (*others[:3], target)
        ),
        clock=lambda: next(ticks),
    )

    assert (third.rank, third.hit) == (3, True)
    assert (fourth.rank, fourth.hit) == (4, False)


def test_a_file_that_never_appears_has_no_rank(tmp_path: Path) -> None:
    """No rank and rank four are different findings and stay apart."""
    ticks = _clock([0.0, 0.0])

    outcome = run(
        [_question(1, tmp_path / "a.md")],
        ask=lambda _: (tmp_path / "b.md",),
        clock=lambda: next(ticks),
    )[0]

    assert outcome.rank is None
    assert outcome.hit is False


def test_paths_are_compared_case_insensitively_and_resolved(tmp_path: Path) -> None:
    """Windows hands back the spelling on disk; the set carries the one typed."""
    target = tmp_path / "Alpha.md"
    target.write_text("x", encoding="utf-8")
    ticks = _clock([0.0, 0.0])

    outcome = run(
        [_question(1, tmp_path / "sub" / ".." / "Alpha.md")],
        ask=lambda _: (target,),
        clock=lambda: next(ticks),
    )[0]

    assert outcome.rank == 1


def test_an_engine_failure_is_not_swallowed(tmp_path: Path) -> None:
    """A partial run that looks whole is worse than none (spec 6)."""

    def refuse(_: str) -> tuple[Path, ...]:
        raise SearchUnavailable("engine down")

    with pytest.raises(SearchUnavailable):
        run([_question(1, tmp_path / "a.md")], ask=refuse, clock=lambda: 0.0)


def test_top_is_the_three_of_the_spec() -> None:
    assert TOP == 3
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `uv run pytest tests/test_bench_quality.py -v`
Expected: FAIL — `ModuleNotFoundError: No module named 'brain.bench.quality'`

- [ ] **Step 3: Write the implementation**

Create `src/brain/bench/quality.py`:

```python
"""How well the chain finds — measured as rank, never as score.

A high score is no proof of truth (spec 13). The only question asked here is
whether the expected source sits in the first three, and where it sat if it
did not.
"""

import os
from collections.abc import Callable, Sequence
from dataclasses import dataclass
from pathlib import Path

from brain.bench.questions import Question

TOP = 3

type Ask = Callable[[str], tuple[Path, ...]]


@dataclass(frozen=True, slots=True)
class Outcome:
    question: Question
    rank: int | None
    hit: bool
    elapsed_ms: float


def run(
    questions: Sequence[Question], *, ask: Ask, clock: Callable[[], float]
) -> tuple[Outcome, ...]:
    """One measurement per question, in the order the set carries them.

    `SearchUnavailable` is deliberately not caught: the caller must not be able
    to mistake a partial run for a whole one, so the run ends and no protocol
    is written (spec 6).
    """
    outcomes: list[Outcome] = []
    for question in questions:
        started = clock()
        ranked = ask(question.query)
        elapsed_ms = (clock() - started) * 1000
        rank = _rank(question.expect, ranked)
        outcomes.append(Outcome(question, rank, rank is not None and rank <= TOP, elapsed_ms))
    return tuple(outcomes)


def _rank(expect: Path, ranked: Sequence[Path]) -> int | None:
    wanted = _comparable(expect)
    for position, candidate in enumerate(ranked, start=1):
        if _comparable(candidate) == wanted:
            return position
    return None


def _comparable(path: Path) -> str:
    """One spelling for two paths that mean the same file.

    `resolve` settles `..` and the case the file system actually stores;
    `normcase` settles the rest on Windows, where the question set carries the
    spelling somebody typed and the engine hands back the one on disk.
    """
    return os.path.normcase(str(path.resolve()))
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `uv run pytest tests/test_bench_quality.py -v`
Expected: PASS.

- [ ] **Step 5: Lint, types, coverage**

Run:
```bash
uv run ruff check src/brain/bench tests/test_bench_quality.py
uv run ruff format --check src/brain/bench tests/test_bench_quality.py
uv run mypy
uv run coverage run -m pytest tests/test_bench_quality.py && uv run coverage report --include "src/brain/bench/quality.py"
```
Expected: sauber, 100 %.

- [ ] **Step 6: Commit**

```bash
git add src/brain/bench/quality.py tests/test_bench_quality.py
git commit -m "Measure retrieval quality as the rank of the expected source"
```

---

### Task 3: Latenz

**Files:**
- Create: `src/brain/bench/latency.py`
- Test: `tests/test_bench_latency.py`

**Interfaces:**
- Consumes: nichts
- Produces: `DEFAULT_REPEAT = 10`, `Operation(name: str, call: Callable[[], object])`, `Timing(name: str, cold_ms: float, warm_ms: tuple[float, ...])` mit den Eigenschaften `median_ms`, `minimum_ms`, `maximum_ms`, `measure(operations: Sequence[Operation], *, repeat: int = DEFAULT_REPEAT, clock: Callable[[], float]) -> tuple[Timing, ...]`

- [ ] **Step 1: Write the failing test**

Create `tests/test_bench_latency.py`:

```python
"""Ten warm runs and one cold one, because a single value measures noise."""

from collections.abc import Iterator

import pytest

from brain.bench.latency import DEFAULT_REPEAT, Operation, measure


def _ticks(values: list[float]) -> "Iterator[float]":
    return iter(values)


def test_the_first_run_is_cold_and_stands_apart() -> None:
    """Cold happens once per session; averaging it in would hide both numbers."""
    ticks = _ticks([0.0, 2.0, 10.0, 10.1, 20.0, 20.3])
    calls: list[int] = []

    timing = measure(
        [Operation("search", lambda: calls.append(1))], repeat=2, clock=lambda: next(ticks)
    )[0]

    assert timing.cold_ms == 2000.0
    assert timing.warm_ms == (100.0, 300.0)
    assert len(calls) == 3


def test_median_minimum_and_maximum_are_reported() -> None:
    """The median carries the decision, the span says whether it is trustworthy."""
    ticks = _ticks([0.0, 0.0, 0.0, 0.1, 0.0, 0.5, 0.0, 0.3])

    timing = measure([Operation("read", lambda: None)], repeat=3, clock=lambda: next(ticks))[0]

    assert timing.median_ms == 300.0
    assert timing.minimum_ms == 100.0
    assert timing.maximum_ms == 500.0


def test_every_operation_is_measured_in_order() -> None:
    ticks = _ticks([0.0, 0.0] * 4)

    timings = measure(
        [Operation("a", lambda: None), Operation("b", lambda: None)],
        repeat=1,
        clock=lambda: next(ticks),
    )

    assert [t.name for t in timings] == ["a", "b"]


def test_a_repeat_below_one_is_refused() -> None:
    """Without a warm run there is no median, and the protocol would show none."""
    with pytest.raises(ValueError, match="repeat must be at least 1, got 0"):
        measure([Operation("a", lambda: None)], repeat=0, clock=lambda: 0.0)


def test_the_default_repeat_is_ten() -> None:
    assert DEFAULT_REPEAT == 10
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `uv run pytest tests/test_bench_latency.py -v`
Expected: FAIL — `ModuleNotFoundError: No module named 'brain.bench.latency'`

- [ ] **Step 3: Write the implementation**

Create `src/brain/bench/latency.py`:

```python
"""How long an operation takes — once cold, then repeatedly warm.

"Warm" here means the engine has already searched in this session and its model
cache is filled. That is not the daemon warmth spec 13 defines; the daemon
arrives in slice 2c, and the protocol says so above every table.
"""

import statistics
from collections.abc import Callable, Sequence
from dataclasses import dataclass

DEFAULT_REPEAT = 10


@dataclass(frozen=True, slots=True)
class Operation:
    name: str
    call: Callable[[], object]


@dataclass(frozen=True, slots=True)
class Timing:
    name: str
    cold_ms: float
    warm_ms: tuple[float, ...]

    @property
    def median_ms(self) -> float:
        return statistics.median(self.warm_ms)

    @property
    def minimum_ms(self) -> float:
        return min(self.warm_ms)

    @property
    def maximum_ms(self) -> float:
        return max(self.warm_ms)


def measure(
    operations: Sequence[Operation],
    *,
    repeat: int = DEFAULT_REPEAT,
    clock: Callable[[], float],
) -> tuple[Timing, ...]:
    if repeat < 1:
        raise ValueError(f"repeat must be at least 1, got {repeat}")
    return tuple(
        Timing(
            operation.name,
            _once(operation, clock),
            tuple(_once(operation, clock) for _ in range(repeat)),
        )
        for operation in operations
    )


def _once(operation: Operation, clock: Callable[[], float]) -> float:
    started = clock()
    operation.call()
    return (clock() - started) * 1000
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `uv run pytest tests/test_bench_latency.py -v`
Expected: PASS.

- [ ] **Step 5: Lint, types, coverage**

Run:
```bash
uv run ruff check src/brain/bench tests/test_bench_latency.py
uv run ruff format --check src/brain/bench tests/test_bench_latency.py
uv run mypy
uv run coverage run -m pytest tests/test_bench_latency.py && uv run coverage report --include "src/brain/bench/latency.py"
```
Expected: sauber, 100 %.

- [ ] **Step 6: Commit**

```bash
git add src/brain/bench/latency.py tests/test_bench_latency.py
git commit -m "Time each operation once cold and repeatedly warm"
```

---

### Task 4: Auswertung, Protokolltext und JSON

**Files:**
- Create: `src/brain/bench/report.py`
- Test: `tests/test_bench_report.py`

**Interfaces:**
- Consumes: `Outcome`, `Timing`, `Kind`, `brain.search.port.Profile`
- Produces: `Environment(qmd_version: str, models: tuple[str, ...], platform: str, documents: int, question_set: str)`, `Report(profile: Profile, environment: Environment, outcomes: tuple[Outcome, ...], timings: tuple[Timing, ...], stamp: str)`, `tally(outcomes) -> dict[Kind, tuple[int, int]]`, `as_markdown(report) -> str`, `as_json(report) -> str`

Der Vorbehalt je Profil ist Teil des Protokolls, nicht der Prosa drumherum: `--profile vector` und `--profile fast` beantworten verschiedene Fragen (§16.3), und die Verwechslung passiert beim Lesen.

- [ ] **Step 1: Write the failing test**

Create `tests/test_bench_report.py`:

```python
"""What the protocol has to say, whether or not anyone reads it carefully."""

import json
from pathlib import Path

from brain.bench.latency import Timing
from brain.bench.questions import Kind, Question
from brain.bench.quality import Outcome
from brain.bench.report import Environment, Report, as_json, as_markdown, tally
from brain.search.port import Profile

_ENVIRONMENT = Environment(
    qmd_version="qmd 2.8.3",
    models=("embed-me", "rerank-me"),
    platform="Windows-11",
    documents=264,
    question_set="98 Messung/questions.yaml",
)


def _outcome(number: int, kind: Kind, *, rank: int | None, ms: float) -> Outcome:
    question = Question(f"q{number:02d}", kind, "frage", Path("/n.md"), "beleg")
    return Outcome(question, rank, rank is not None and rank <= 3, ms)


def _report(profile: Profile = Profile.FULL, timings: tuple[Timing, ...] = ()) -> Report:
    return Report(
        profile=profile,
        environment=_ENVIRONMENT,
        outcomes=(
            _outcome(1, Kind.EXACT, rank=1, ms=100.0),
            _outcome(2, Kind.EXACT, rank=7, ms=900.0),
            _outcome(3, Kind.CROSS_LINGUAL, rank=None, ms=700.0),
        ),
        timings=timings,
        stamp="2026-08-21-1430",
    )


def test_the_tally_counts_hits_per_sort() -> None:
    counted = tally(_report().outcomes)

    assert counted[Kind.EXACT] == (1, 2)
    assert counted[Kind.CROSS_LINGUAL] == (0, 1)
    assert counted[Kind.MIXED] == (0, 0)


def test_the_table_names_every_sort_and_the_total() -> None:
    """A sort with no questions is a finding about the set, not a row to omit."""
    text = as_markdown(_report())

    assert "| exakt | 1/2 |" in text
    assert "| gemischt | 0/0 |" in text
    assert "| gesamt | 1/3 |" in text


def test_a_missed_question_is_listed_with_its_rank() -> None:
    """Rank seven and 'not found at all' are different findings."""
    text = as_markdown(_report())

    assert "q02" in text and "Rang 7" in text
    assert "q03" in text and "nicht gefunden" in text
    assert "q01" not in text.split("## Fehlliste")[1]


def test_hits_and_misses_are_timed_separately() -> None:
    """The retry on an empty result doubles exactly the failing queries (2a)."""
    text = as_markdown(_report())

    assert "Median Treffer: 100 ms" in text
    assert "Median Fehlschläge: 800 ms" in text


def test_the_head_carries_what_makes_the_run_readable_later() -> None:
    text = as_markdown(_report())

    for expected in ("qmd 2.8.3", "embed-me", "Windows-11", "264", "98 Messung/questions.yaml"):
        assert expected in text


def test_the_vector_run_says_what_it_does_not_answer() -> None:
    text = as_markdown(_report(Profile.VECTOR))

    assert "Anteil des Einbettungsmodells" in text
    assert "sagt nichts darüber, ob `fast` sprachübergreifend trägt" in text


def test_the_fast_run_says_what_it_does_not_answer() -> None:
    text = as_markdown(_report(Profile.FAST))

    assert "ob `fast` für sprachübergreifende Fragen eingesetzt werden darf" in text
    assert "Anteil eines einzelnen Modells" in text


def test_a_run_without_a_caveat_carries_none() -> None:
    assert "beantwortet **nicht**" not in as_markdown(_report(Profile.FULL))


def test_the_latency_table_carries_the_warmth_caveat() -> None:
    """The daemon arrives in 2c; until then warm is a substitute measure."""
    text = as_markdown(_report(timings=(Timing("keyword", 900.0, (20.0, 30.0, 40.0)),)))

    assert "| keyword | 900 ms | 30 ms | 20 ms | 40 ms |" in text
    assert "kein Beleg für die Budgets" in text


def test_a_run_without_timings_has_no_latency_section() -> None:
    assert "## Latenz" not in as_markdown(_report())


def test_the_json_carries_every_question_and_the_head() -> None:
    """The table is for reading; two runs are compared on this."""
    data = json.loads(as_json(_report(timings=(Timing("keyword", 900.0, (20.0,)),))))

    assert data["profile"] == "full"
    assert data["stamp"] == "2026-08-21-1430"
    assert data["environment"]["documents"] == 264
    assert data["questions"][1] == {
        "id": "q02",
        "sort": "exakt",
        "rank": 7,
        "hit": False,
        "elapsed_ms": 900.0,
    }
    assert data["latency"][0] == {
        "operation": "keyword",
        "cold_ms": 900.0,
        "median_ms": 20.0,
        "minimum_ms": 20.0,
        "maximum_ms": 20.0,
        "warm_ms": [20.0],
    }
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `uv run pytest tests/test_bench_report.py -v`
Expected: FAIL — `ModuleNotFoundError: No module named 'brain.bench.report'`

- [ ] **Step 3: Write the implementation**

Create `src/brain/bench/report.py`:

```python
"""What a run leaves behind: a table to read and a file to compare against.

The head is not decoration. Two numbers produced under different models are not
a series, and without the head nobody can tell that a year from now.
"""

import json
import statistics
from dataclasses import dataclass
from typing import Any

from brain.bench.latency import Timing
from brain.bench.questions import Kind
from brain.bench.quality import Outcome
from brain.search.port import Profile

# What each of the two runs of spec 16.3 answers, and what it does not. Kept
# beside the numbers because the confusion happens while reading the table.
_CAVEAT = {
    Profile.VECTOR: (
        "Dieser Lauf beziffert den **Anteil des Einbettungsmodells** an der "
        "Sprachbrücke. Er beantwortet **nicht**, ob `fast` sprachübergreifend "
        "trägt — er sagt nichts darüber, ob `fast` sprachübergreifend trägt."
    ),
    Profile.FAST: (
        "Dieser Lauf beantwortet, **ob `fast` für sprachübergreifende Fragen "
        "eingesetzt werden darf**. Er beantwortet **nicht** den Anteil eines "
        "einzelnen Modells an der Sprachbrücke."
    ),
}
_WARMTH = (
    "„Warm\" heißt hier: qmd hat in dieser Sitzung schon gesucht. Das ist ein "
    "Ersatzmaß und **kein Beleg für die Budgets** aus §13 — die verlangen den "
    "Daemon, und der entsteht in 2c."
)


@dataclass(frozen=True, slots=True)
class Environment:
    qmd_version: str
    models: tuple[str, ...]
    platform: str
    documents: int
    question_set: str


@dataclass(frozen=True, slots=True)
class Report:
    profile: Profile
    environment: Environment
    outcomes: tuple[Outcome, ...]
    timings: tuple[Timing, ...]
    stamp: str


def tally(outcomes: tuple[Outcome, ...]) -> dict[Kind, tuple[int, int]]:
    """Hits and total per sort. Every sort appears, empty ones included."""
    counted = {kind: [0, 0] for kind in Kind}
    for outcome in outcomes:
        counted[outcome.question.kind][1] += 1
        counted[outcome.question.kind][0] += int(outcome.hit)
    return {kind: (hits, total) for kind, (hits, total) in counted.items()}


def as_markdown(report: Report) -> str:
    lines = [
        f"# Messlauf {report.stamp} — Profil `{report.profile.value}`",
        "",
        f"- qmd: {report.environment.qmd_version}",
        f"- Modelle: {', '.join(report.environment.models)}",
        f"- Rechner: {report.environment.platform}",
        f"- indexierte Dokumente: {report.environment.documents}",
        f"- Fragensatz: {report.environment.question_set}",
        "",
    ]
    caveat = _CAVEAT.get(report.profile)
    if caveat is not None:
        lines += [caveat, ""]
    lines += _quality_section(report)
    if report.timings:
        lines += _latency_section(report)
    return "\n".join(lines) + "\n"


def _quality_section(report: Report) -> list[str]:
    counted = tally(report.outcomes)
    hits = sum(pair[0] for pair in counted.values())
    lines = ["## Trefferqualität", "", "| Sorte | Treffer |", "|---|---|"]
    lines += [f"| {kind.value} | {counted[kind][0]}/{counted[kind][1]} |" for kind in Kind]
    lines += [
        f"| gesamt | {hits}/{len(report.outcomes)} |",
        "",
        f"Median Treffer: {_median([o for o in report.outcomes if o.hit])}",
        f"Median Fehlschläge: {_median([o for o in report.outcomes if not o.hit])}",
        "",
        "## Fehlliste",
        "",
    ]
    misses = [outcome for outcome in report.outcomes if not outcome.hit]
    if not misses:
        return [*lines, "keine", ""]
    lines += [
        f"- {outcome.question.id} ({outcome.question.kind.value}): "
        + ("nicht gefunden" if outcome.rank is None else f"Rang {outcome.rank}")
        + f" — {outcome.question.query}"
        for outcome in misses
    ]
    return [*lines, ""]


def _median(outcomes: list[Outcome]) -> str:
    """No number rather than a zero: an empty group has no median."""
    if not outcomes:
        return "—"
    return f"{round(statistics.median(o.elapsed_ms for o in outcomes))} ms"


def _latency_section(report: Report) -> list[str]:
    lines = [
        "## Latenz",
        "",
        _WARMTH,
        "",
        "| Operation | kalt | Median warm | min | max |",
        "|---|---|---|---|---|",
    ]
    lines += [
        f"| {t.name} | {round(t.cold_ms)} ms | {round(t.median_ms)} ms | "
        f"{round(t.minimum_ms)} ms | {round(t.maximum_ms)} ms |"
        for t in report.timings
    ]
    return [*lines, ""]


def as_json(report: Report) -> str:
    payload: dict[str, Any] = {
        "stamp": report.stamp,
        "profile": report.profile.value,
        "environment": {
            "qmd_version": report.environment.qmd_version,
            "models": list(report.environment.models),
            "platform": report.environment.platform,
            "documents": report.environment.documents,
            "question_set": report.environment.question_set,
        },
        "questions": [
            {
                "id": outcome.question.id,
                "sort": outcome.question.kind.value,
                "rank": outcome.rank,
                "hit": outcome.hit,
                "elapsed_ms": outcome.elapsed_ms,
            }
            for outcome in report.outcomes
        ],
        "latency": [
            {
                "operation": timing.name,
                "cold_ms": timing.cold_ms,
                "median_ms": timing.median_ms,
                "minimum_ms": timing.minimum_ms,
                "maximum_ms": timing.maximum_ms,
                "warm_ms": list(timing.warm_ms),
            }
            for timing in report.timings
        ],
    }
    return json.dumps(payload, ensure_ascii=False, indent=2, sort_keys=True) + "\n"
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `uv run pytest tests/test_bench_report.py -v`
Expected: PASS. Schlägt `test_a_missed_question_is_listed_with_its_rank` an der Zeile `assert "q01" not in …` fehl, liegt es daran, dass ein Treffer in der Fehlliste steht — das ist der Fehler, nicht der Test.

- [ ] **Step 5: Lint, types, coverage**

Run:
```bash
uv run ruff check src/brain/bench tests/test_bench_report.py
uv run ruff format --check src/brain/bench tests/test_bench_report.py
uv run mypy
uv run coverage run -m pytest tests/test_bench_report.py && uv run coverage report --include "src/brain/bench/report.py"
```
Expected: sauber, 100 %. Fehlt die Zeile „keine" in der Fehlliste, ergänze einen Test mit einem Lauf ohne Fehlschlag.

- [ ] **Step 6: Commit**

```bash
git add src/brain/bench/report.py tests/test_bench_report.py
git commit -m "Render a bench run as a table to read and a file to compare"
```

---

### Task 5: Modellnamen für den Protokollkopf

**Files:**
- Modify: `src/brain/search/qmd_config.py`
- Test: `tests/test_qmd_config.py`

**Interfaces:**
- Consumes: `qmd_config.qmd_config_path()`, das vorhandene `_load`
- Produces: `models(config: Path) -> tuple[str, ...]`

Kleiner Task, aber ein eigener: Er ändert ein Modul, das nichts mit `bench` zu tun hat, und ein Prüfer soll ihn getrennt annehmen oder ablehnen können.

- [ ] **Step 1: Write the failing test**

Append to `tests/test_qmd_config.py`:

```python
def test_the_model_names_are_read_for_the_protocol_head(tmp_path: Path) -> None:
    """Two numbers produced under different models are not a series."""
    config = tmp_path / "index.yml"
    config.write_text(
        "embedding:\n  model: embed-me\nrerank:\n  model: rerank-me\n"
        "query_expansion:\n  model: expand-me\n",
        encoding="utf-8",
    )

    assert models(config) == ("embed-me", "expand-me", "rerank-me")


def test_a_missing_model_entry_is_named_rather_than_dropped(tmp_path: Path) -> None:
    """A silently shorter list would read as 'that model was not used'."""
    config = tmp_path / "index.yml"
    config.write_text("embedding:\n  model: embed-me\n", encoding="utf-8")

    assert models(config) == ("embed-me", "unknown", "unknown")


def test_an_absent_configuration_yields_unknowns(tmp_path: Path) -> None:
    assert models(tmp_path / "gone.yml") == ("unknown", "unknown", "unknown")
```

Ergänze den Import oben in der Datei um `models`:

```python
from brain.search.qmd_config import CollectionSpec, QmdConfigError, models, sync_collections
```

(Der vorhandene Import ist zu erweitern, nicht zu ersetzen — die anderen Namen bleiben, wie sie in der Datei stehen.)

- [ ] **Step 2: Run the tests to verify they fail**

Run: `uv run pytest tests/test_qmd_config.py -v`
Expected: FAIL — `ImportError: cannot import name 'models'`

- [ ] **Step 3: Write the implementation**

Append to `src/brain/search/qmd_config.py`:

```python
# The three models the chain uses, in the order a reader expects them: what
# embeds, what expands the question, what reranks.
_MODEL_KEYS = ("embedding", "query_expansion", "rerank")


def models(config: Path) -> tuple[str, ...]:
    """The model names for a protocol head, or `unknown` where none is set.

    A missing entry becomes `unknown` rather than a shorter list: a list of two
    would read as "that stage did not run", which is a claim this function has
    no way to make.
    """
    if not config.is_file():
        return tuple("unknown" for _ in _MODEL_KEYS)
    raw = _load(config, config.read_text(encoding="utf-8"))
    return tuple(_model(raw.get(key)) for key in _MODEL_KEYS)


def _model(section: object) -> str:
    if isinstance(section, dict) and isinstance(section.get("model"), str):
        return str(section["model"])
    return "unknown"
```

Prüfe die Signatur von `_load` in der Datei, bevor du sie aufrufst; ist sie `_load(config: Path, raw: str) -> dict[object, object]`, passt der Aufruf oben. `raw.get(key)` verlangt dann keinen Umbau, weil die Schlüssel Strings sind.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `uv run pytest tests/test_qmd_config.py -v`
Expected: PASS.

- [ ] **Step 5: Lint, types, coverage**

Run:
```bash
uv run ruff check src tests
uv run mypy
uv run coverage run -m pytest && uv run coverage report --include "src/brain/search/qmd_config.py"
```
Expected: sauber, 100 %.

- [ ] **Step 6: Commit**

```bash
git add src/brain/search/qmd_config.py tests/test_qmd_config.py
git commit -m "Read the qmd model names for the bench protocol head"
```

---

### Task 6: Der Unterbefehl `brain bench`

**Files:**
- Modify: `src/brain/cli.py`
- Test: `tests/test_cli.py`

**Interfaces:**
- Consumes: alles aus Task 1–5, `brain.core.search`, `brain.core.catalog`, `brain.core.read`, `brain.registry.read_registry`, `brain.paths.registry_path`
- Produces: `brain bench [--profile {full,fast,vector,keyword}] [--questions PATH] [--scope SCOPE] [--latency] [--repeat N] [--out DIR] [--channel] [--state-dir]`; die Seams `brain.cli._now()` und `brain.cli._qmd_version()`

**Verhalten, das die Tests festhalten:**

| Fall | Verhalten |
|---|---|
| Vorgaben | `--profile full`, `--scope knowledge`, `--repeat 10` |
| `--out` fehlt | `<Bereichspfad>/98 Messung`; existiert das Verzeichnis nicht, Abbruch mit Nennung von `--out` |
| `--questions` fehlt | `<out>/questions.yaml` |
| Protokolldatei existiert schon | Abbruch, nichts wird überschrieben |
| Fragensatz fehlerhaft | Abbruch **vor** der Messung, jeder Verstoß auf stderr |
| `SearchUnavailable` im Lauf | Abbruch, **kein** Protokoll |
| Ergebnis | zwei Dateien plus die Tabelle auf stdout, Rückgabewert 0 |

- [ ] **Step 1: Write the failing test**

Append to `tests/test_cli.py`:

```python
# The protocol head counts what the engine holds, so every bench test has to
# script a listing — an unscripted one is an AssertionError in FakePort.
_LISTED = {"knowledge": ["10 Rohquellen/alpha.md"]}


def _bench_world(tmp_path: Path) -> tuple[Path, Path]:
    """One area `knowledge` with a measurement folder and a two-question set."""
    state = tmp_path / "state"
    state.mkdir()
    area = tmp_path / "knowledge"
    (area / "10 Rohquellen").mkdir(parents=True)
    (area / "98 Messung").mkdir()
    (area / ".brain.toml").write_text(
        '[area]\nscope = "knowledge"\n\n[index]\ninclude = ["**/*.md"]\n', encoding="utf-8"
    )
    (state / "registry.toml").write_text(
        f'[[area]]\nscope = "knowledge"\npath = "{area.as_posix()}"\nreadonly = false\n',
        encoding="utf-8",
    )
    target = area / "10 Rohquellen" / "alpha.md"
    target.write_text("---\ntitle: Alpha\n---\n\nder beleg steht hier\n", encoding="utf-8")
    (area / "98 Messung" / "questions.yaml").write_text(
        "".join(
            f"- id: q{n:02d}\n  sort: {sort}\n  query: \"frage {n}\"\n"
            f'  expect: "{target.as_posix()}"\n  beleg: "der beleg steht hier"\n'
            for n, sort in enumerate(("exakt", "umschreibung"), start=1)
        ),
        encoding="utf-8",
    )
    return state, area


def _bench_hit(area: Path) -> SearchHit:
    return SearchHit(
        collection="knowledge",
        relative="10 Rohquellen/alpha.md",
        line=1,
        title="Alpha",
        snippet="…",
        score=0.9,
        content_key="#a",
    )


def test_bench_writes_both_files_and_prints_the_table(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    state, area = _bench_world(tmp_path)
    port = FakePort(results=[[_bench_hit(area)], [_bench_hit(area)]], indexed=_LISTED)
    monkeypatch.setattr("brain.cli._port", lambda: port)
    monkeypatch.setattr("brain.cli._now", lambda: "2026-08-21-1430")
    monkeypatch.setattr("brain.cli._qmd_version", lambda: "qmd 2.8.3")

    code = main(["bench", "--state-dir", str(state)])

    assert code == 0
    protocol = area / "98 Messung" / "bench-2026-08-21-1430-full.md"
    assert protocol.is_file()
    assert (area / "98 Messung" / "bench-2026-08-21-1430-full.json").is_file()
    assert "| gesamt | 2/2 |" in protocol.read_text(encoding="utf-8")
    assert "| gesamt | 2/2 |" in capsys.readouterr().out


def test_bench_asks_with_the_profile_it_was_given(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """The two runs of spec 16.3 are two values of this one argument."""
    state, area = _bench_world(tmp_path)
    port = FakePort(results=[[_bench_hit(area)], [_bench_hit(area)]], indexed=_LISTED)
    monkeypatch.setattr("brain.cli._port", lambda: port)
    monkeypatch.setattr("brain.cli._now", lambda: "2026-08-21-1431")
    monkeypatch.setattr("brain.cli._qmd_version", lambda: "qmd 2.8.3")

    main(["bench", "--profile", "vector", "--state-dir", str(state)])

    assert {call[2] for call in port.calls} == {Profile.VECTOR}
    text = (area / "98 Messung" / "bench-2026-08-21-1431-vector.md").read_text(encoding="utf-8")
    assert "Anteil des Einbettungsmodells" in text


def test_bench_refuses_a_faulty_question_set_before_measuring(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    state, area = _bench_world(tmp_path)
    (area / "98 Messung" / "questions.yaml").write_text(
        '- id: q01\n  sort: exakt\n  query: "x"\n  expect: "/nowhere.md"\n  beleg: "y"\n',
        encoding="utf-8",
    )
    port = FakePort(results=[])
    monkeypatch.setattr("brain.cli._port", lambda: port)

    code = main(["bench", "--state-dir", str(state)])

    assert code == 1
    assert port.calls == []
    assert "expect does not exist" in capsys.readouterr().err
    assert not list((area / "98 Messung").glob("bench-*"))


def test_bench_writes_nothing_when_the_engine_fails(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    """A partial measurement that looks whole is worse than none."""
    state, area = _bench_world(tmp_path)
    monkeypatch.setattr(
        "brain.cli._port",
        lambda: FakePort(results=[SearchUnavailable("engine down")], indexed=_LISTED),
    )
    monkeypatch.setattr("brain.cli._now", lambda: "2026-08-21-1432")
    monkeypatch.setattr("brain.cli._qmd_version", lambda: "qmd 2.8.3")

    code = main(["bench", "--state-dir", str(state)])

    assert code == 1
    assert "engine down" in capsys.readouterr().err
    assert not list((area / "98 Messung").glob("bench-*"))


def test_bench_never_overwrites_an_existing_protocol(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    state, area = _bench_world(tmp_path)
    existing = area / "98 Messung" / "bench-2026-08-21-1433-full.md"
    existing.write_text("older", encoding="utf-8")
    monkeypatch.setattr("brain.cli._port", lambda: FakePort(results=[]))
    monkeypatch.setattr("brain.cli._now", lambda: "2026-08-21-1433")

    code = main(["bench", "--state-dir", str(state)])

    assert code == 1
    assert existing.read_text(encoding="utf-8") == "older"
    assert "already exists" in capsys.readouterr().err


def test_bench_names_the_flag_when_the_measurement_folder_is_missing(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    state, area = _bench_world(tmp_path)
    (area / "98 Messung" / "questions.yaml").unlink()
    (area / "98 Messung").rmdir()
    monkeypatch.setattr("brain.cli._port", lambda: FakePort(results=[]))

    code = main(["bench", "--state-dir", str(state)])

    assert code == 1
    assert "--out" in capsys.readouterr().err


def test_bench_latency_measures_every_operation(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    state, area = _bench_world(tmp_path)
    monkeypatch.setattr(
        "brain.cli._port",
        lambda: FakePort(results=[[_bench_hit(area)]] * 40, indexed=_LISTED),
    )
    monkeypatch.setattr("brain.cli._now", lambda: "2026-08-21-1434")
    monkeypatch.setattr("brain.cli._qmd_version", lambda: "qmd 2.8.3")

    code = main(["bench", "--latency", "--repeat", "2", "--state-dir", str(state)])

    assert code == 0
    text = (area / "98 Messung" / "bench-2026-08-21-1434-full.md").read_text(encoding="utf-8")
    for operation in ("catalog", "read", "keyword", "fast", "full"):
        assert f"| {operation} |" in text
    assert "kein Beleg für die Budgets" in text


def test_bench_refuses_a_repeat_below_one(tmp_path: Path) -> None:
    state, _ = _bench_world(tmp_path)

    with pytest.raises(SystemExit):
        main(["bench", "--latency", "--repeat", "0", "--state-dir", str(state)])
```

Ergänze oben in `tests/test_cli.py` die Importe, die noch fehlen (`SearchUnavailable`, `Profile`, `SearchHit`, `FakePort` — prüfe, welche schon dastehen, und füge nur die fehlenden hinzu).

- [ ] **Step 2: Run the tests to verify they fail**

Run: `uv run pytest tests/test_cli.py -v -k bench`
Expected: FAIL — `argparse` kennt `bench` nicht, `SystemExit: 2`.

- [ ] **Step 3: Write the implementation**

Ergänze in `src/brain/cli.py` die Importe:

```python
import platform
import subprocess
import time
from datetime import UTC, datetime

from brain.bench.latency import DEFAULT_REPEAT, Operation, measure
from brain.bench.questions import QuestionSetError
from brain.bench.questions import load as load_questions
from brain.bench.quality import run as run_quality
from brain.bench.report import Environment, Report, as_json, as_markdown
from brain.registry import read_registry
from brain.search.qmd_config import models, qmd_config_path
```

(`read_registry`, `qmd_config_path` und `Profile` stehen möglicherweise schon oben — dann nicht doppelt importieren.)

Geprüft werden muss außerdem, ob `SearchUnavailable`, `SearchPort` und
`collections.abc.Callable` schon importiert sind; `_bench` und `_asking` brauchen
alle drei.

Füge die beiden Seams und den Befehl hinzu:

```python
def _now() -> str:
    """The stamp in every protocol name. A seam, so tests get a fixed one."""
    return datetime.now(UTC).strftime("%Y-%m-%d-%H%M")


def _qmd_version() -> str:
    """What the engine calls itself, for the protocol head.

    Its own seam rather than a port method: the version belongs in a document,
    not in the contract the core searches through, and widening the port for a
    line of prose would put a reporting need into the search interface.
    """
    finished = subprocess.run(
        ["qmd", "--version"], capture_output=True, text=True, check=False
    )
    return finished.stdout.strip() or "unknown"


def _area_path(state_dir: Path, scope: str) -> Path:
    for area in read_registry(registry_path(state_dir)):
        if area.scope == scope:
            return area.path
    raise BenchError(f"no area named {scope!r} in the registry")


class BenchError(Exception):
    """Something the run cannot proceed with, said in one line."""


def _bench(args: argparse.Namespace) -> int:
    state_dir = resolve_state_dir(args.state_dir)
    channel = Channel(args.channel)
    profile = Profile(args.profile)
    try:
        out = _bench_out(args, state_dir)
        questions = load_questions(args.questions or out / "questions.yaml")
        targets = _bench_targets(out, _now(), profile)
        port = _port()
        outcomes = run_quality(
            questions,
            ask=_asking(port, args.scope, profile, channel, state_dir),
            clock=time.perf_counter,
        )
        timings = (
            measure(
                _bench_operations(port, args.scope, channel, state_dir),
                repeat=args.repeat,
                clock=time.perf_counter,
            )
            if args.latency
            else ()
        )
        report = Report(
            profile=profile,
            environment=_bench_environment(port, args.scope, out),
            outcomes=outcomes,
            timings=timings,
            stamp=targets[0].stem.split("-", 1)[1].rsplit("-", 1)[0],
        )
    except QuestionSetError as broken:
        for problem in broken.problems:
            print(f"error: {problem}", file=sys.stderr)
        return 1
    except (BenchError, SearchUnavailable, OSError) as refused:
        print(f"error: {refused}", file=sys.stderr)
        return 1
    text = as_markdown(report)
    targets[0].write_text(text, encoding="utf-8")
    targets[1].write_text(as_json(report), encoding="utf-8")
    print(text, end="")
    return 0


def _bench_out(args: argparse.Namespace, state_dir: Path) -> Path:
    out = args.out if args.out is not None else _area_path(state_dir, args.scope) / "98 Messung"
    if not out.is_dir():
        raise BenchError(f"no measurement folder at {out} — name one with --out")
    return out


def _bench_targets(out: Path, stamp: str, profile: Profile) -> tuple[Path, Path]:
    """Both names at once, and neither may exist.

    Checked before the first query rather than before the first write: a run
    that measures for a minute and then refuses to save it has wasted the
    minute.
    """
    targets = (
        out / f"bench-{stamp}-{profile.value}.md",
        out / f"bench-{stamp}-{profile.value}.json",
    )
    for target in targets:
        if target.exists():
            raise BenchError(f"{target} already exists")
    return targets


def _asking(
    port: SearchPort, scope: str, profile: Profile, channel: Channel, state_dir: Path
) -> Callable[[str], tuple[Path, ...]]:
    """The chain above the seam, reduced to ranked absolute paths.

    Measured with deduplication and the privacy filter in place, because that
    is the chain that answers in daily use (spec 3.1).
    """
    root = _area_path(state_dir, scope)

    def ask(query: str) -> tuple[Path, ...]:
        answer = search(
            query,
            scope=scope,
            profile=profile,
            n=10,
            channel=channel,
            port=port,
            state_dir=state_dir,
        )
        return tuple(root / result.relative for result in answer.results)

    return ask


def _bench_operations(
    port: SearchPort, scope: str, channel: Channel, state_dir: Path
) -> list[Operation]:
    relative = "10 Rohquellen/alpha.md"

    def searching(profile: Profile) -> Callable[[], object]:
        return lambda: search(
            "latenz",
            scope=scope,
            profile=profile,
            n=5,
            channel=channel,
            port=port,
            state_dir=state_dir,
        )

    return [
        Operation("catalog", lambda: catalog(scope=scope, channel=channel, state_dir=state_dir)),
        Operation(
            "read",
            lambda: read(
                scope=scope,
                relative=relative,
                section=None,
                channel=channel,
                state_dir=state_dir,
            ),
        ),
        Operation("keyword", searching(Profile.KEYWORD)),
        Operation("fast", searching(Profile.FAST)),
        Operation("full", searching(Profile.FULL)),
    ]


def _bench_environment(port: SearchPort, scope: str, out: Path) -> Environment:
    return Environment(
        qmd_version=_qmd_version(),
        models=models(qmd_config_path()),
        platform=platform.platform(),
        documents=len(port.indexed(scope)),
        question_set=str(out / "questions.yaml"),
    )
```

Ergänze in `_build_parser`:

```python
    bench_parser = sub.add_parser("bench", help="measure retrieval quality and latency")
    bench_parser.add_argument(
        "--profile", choices=[p.value for p in Profile], default=Profile.FULL.value
    )
    bench_parser.add_argument("--questions", type=Path, default=None)
    bench_parser.add_argument("--scope", default="knowledge")
    bench_parser.add_argument("--latency", action="store_true")
    bench_parser.add_argument("--repeat", type=_at_least_one, default=DEFAULT_REPEAT)
    bench_parser.add_argument("--out", type=Path, default=None)
    _add_common(bench_parser)
```

Ergänze in `_dispatch`, vor der `status`-Rückgabe:

```python
    if args.command == "bench":
        return _bench(args)
```

`_at_least_one` wird hier wiederverwendet: Die Begründung, warum eine Zahl unter eins abgelehnt gehört, ist bei `--repeat` dieselbe wie bei `-n` — ohne warmen Lauf gibt es keinen Median, und das Protokoll zeigte trotzdem eine Tabelle.

**Wenn `stamp` in `_bench` unhandlich wirkt:** Er wird aus dem Dateinamen zurückgerechnet, damit Name und Kopf nicht auseinanderlaufen können. Reiche stattdessen den Zeitstempel als Variable durch `_bench` — beides ist zulässig, solange `_now()` genau einmal je Lauf aufgerufen wird. Zweimal aufgerufen, liegt der Kopf eine Minute neben dem Dateinamen.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `uv run pytest tests/test_cli.py -v -k bench`
Expected: PASS.

- [ ] **Step 5: Run the whole suite and the full gate**

Run:
```bash
uv run pytest
uv run ruff check src tests
uv run ruff format --check src tests
uv run mypy
uv run coverage run -m pytest && uv run coverage report
```
Expected: alle Tests grün, keine Lint- oder Typbefunde, Coverage 100 %.

- [ ] **Step 6: Commit**

```bash
git add src/brain/cli.py tests/test_cli.py
git commit -m "Add the bench subcommand: measure, protocol, refuse to overwrite"
```

---

### Task 7: Der echte Messlauf und die Spec-Nachträge

Kein Code. Dieser Task ist die Abnahme; er läuft von Hand gegen den echten Vault und endet damit, dass die Spec sagt, was gemessen wurde.

**Files:**
- Modify: `docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md`
- Create: die Protokolle im Vault (nicht im Repo)

- [ ] **Step 1: Die vier Qualitätsläufe**

```bash
uv run brain bench --profile full
uv run brain bench --profile fast
uv run brain bench --profile vector
uv run brain bench --profile keyword
```

Bricht ein Lauf am Fragensatz ab, ist der Fragensatz zu reparieren — nicht die Prüfung. Der Satz ist seit Scheibe 0 nicht mehr gegen den Bestand gehalten worden; abgewanderte Belege sind zu erwarten.

- [ ] **Step 2: Der Latenzlauf**

```bash
uv run brain bench --latency
```

- [ ] **Step 3: Den qmd-Aufrufaufschlag beziffern**

Halte die gemessene `keyword`-Zeit gegen die Zeit, die qmd selbst meldet. Die Differenz ist der Prozessstart. Notiere sie als Zahl, nicht als Eindruck.

- [ ] **Step 4: Die Spec nachziehen**

Trage ein, was gemessen wurde:
- §13, Latenztabelle: die gemessenen Werte, mit dem benannten Vorbehalt zur Ersatz-Wärme
- §13, Absatz „Wie qmd aufgerufen wird": die Entscheidung, mit der Zahl aus Schritt 3
- §16.3: die Ergebnisse der zwei Mehrsprachigkeitsläufe, jeder mit seiner Frage
- §7.3 und Entscheidung 46: bestätigt oder gedreht, mit der Zahl aus dreißig Fragen
- §17, Zeile 2b: abgehakt

- [ ] **Step 5: Commit**

```bash
git add docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md
git commit -m "Record what slice 2b measured, and decide the qmd call form"
```

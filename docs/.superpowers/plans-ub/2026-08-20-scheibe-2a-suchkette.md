# Scheibe 2a — Suchkette ohne Daemon · Implementierungsplan

> **Für ausführende Agenten:** ERFORDERLICHER SUB-SKILL: `superpowers:subagent-driven-development` (empfohlen) oder `superpowers:executing-plans`, um diesen Plan Aufgabe für Aufgabe abzuarbeiten. Schritte nutzen Checkbox-Syntax (`- [ ]`).

**Ziel:** Die fünf schreibfreien Werkzeuge (`catalog`, `search`, `read`, `neighbors`, `status`) laufen als direkte CLI über die drei externen Bestände — mit Datenschutzgatter, Entdopplung und der Unterscheidung „leer" von „fehlgeschlagen".

**Vorgehen:** qmd bleibt die Suchmaschine, wird aber nie direkt angesprochen. Der Kern redet mit einer engen Schnittstelle (`SearchPort`); eine Umsetzung ruft qmd als Prozess auf, eine zweite ist die Attrappe für Tests. Die drei Zusagen der Spec liegen **oberhalb** dieser Naht, damit sie über jeder Umsetzung gelten. Kein Daemon, kein MCP, keine Messung — die kommen in 2b und 2c.

**Werkzeuge:** Python ≥ 3.14, `uv`, `pytest`, `ruff`, `mypy`, `coverage`. Extern: `qmd` 2.8.3 (im Pfad unter `%APPDATA%/npm`).

**Spec:** `docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md` — maßgeblich sind §5.5, §5.5.1, §5.8, §7.2, §7.2.1, §7.4, §7.5, §13, §14, §17 (Zeile 2a) und die Entscheidungen 36–41.

## Globale Rahmenbedingungen

Aus der Spec und den globalen Regeln. Sie gelten für **jede** Aufgabe:

- **Python ≥ 3.14 überall explizit:** `requires-python = ">=3.14"`, ruff `target-version = "py314"`, mypy `python_version = "3.14"`.
- **`uv` für alles.** Kein `pip`, kein `python -m venv`. Aufrufe über `uv run`.
- **TDD.** Erst der fehlschlagende Test, dann die Implementierung.
- **100 % Coverage, gemessen.** Ausschlüsse nur mit begründendem Kommentar: `# pragma: no cover  # <Grund>`, nie nackt.
- **Typisierung vollständig.** Kein `Any`, kein `# type: ignore` ohne begründenden Kommentar. `mypy --strict` läuft ohne Fehler.
- **Imports stehen oben**, auf Modulebene. Ein lokaler Import braucht einen Kommentar mit dem Grund.
- **Sprachen:** Quellcode, Bezeichner, Code-Kommentare, Commit-Nachrichten und Log-/Fehlermeldungen **englisch**; Prosa in Dokumenten deutsch.
- **Zeilenenden immer LF** beim Schreiben, Pfade in Ausgaben immer mit Vorwärts-Schrägstrichen.
- **Keine Suchmaschine, keine Modelle, kein Netz in Tests** (§14). Einzige Ausnahme ist Aufgabe 11, die ausdrücklich als eigener Lauf geführt wird und nicht in der normalen Testrunde steckt.
- **Kein Sprachmodell in Tests.**

### Was in dieser Teilscheibe ausdrücklich nicht gebaut wird

Daemon, IPC, MCP-Adapter (2c). `brain bench` in jeder Form, Latenzmessung, die zwei Mehrsprachigkeitsläufe (2b). `approve`, `cases`, `lint`, `hook`, `promote`, `reconcile` (Scheiben 3 und 5). Eine eigene Frageerweiterung oder ein eigener Reranker (im Brainstorming als Weg B verworfen).

### Die Falle, die man kennen muss

**Es gibt jetzt zwei Indizes über denselben Bestand.** Unseren (`_identities.tsv`, `graph.json`, `index.md`) und den von qmd. Sie können auseinanderlaufen, und die Spec verlangt, dass das **sichtbar** wird statt still zu bleiben (Entscheidung 40). Wer beim Entdoppeln einen unbekannten Pfad einfach verwirft, baut genau den stillen Fehler ein, den die Scheibe verhindern soll: Der Treffer verschwände aus der Liste, ohne dass irgendwo stünde, warum.

Die zweite Falle liegt daneben: **`n` wird nach dem Entdoppeln angewandt, nie davor.** Sonst verschenkt eine Liste mit fünf Plätzen Plätze an Wiederholungen, und „unter den ersten drei" ist eine schwächere Aussage, als sie aussieht (§16.12).

### Was von qmd gemessen feststeht

Nicht raten — das hier wurde gegen `qmd 2.8.3` geprüft:

- `--json` liefert auf **stdout** ein sauberes JSON-Array. Diagnosezeilen („Searching 2 vector queries…", die Hyde-Erweiterung) gehen nach **stderr**. `json.loads(stdout)` genügt.
- Ein Treffer ist ein Objekt mit `docid` (`"#d45d28"` — qmds Inhaltshash), `score` (Fließkomma 0–1), `file` (`"qmd://<collection>/<pfad>"`), `line`, `title`, `snippet`.
- Ein leeres Ergebnis ist `[]` **mit Rückgabewert 0**. Ein leeres Ergebnis ist damit von einem Fehlschlag nur über den Rückgabewert und über stderr zu unterscheiden — genau der Grund für §16.14.
- `-c <name>` filtert auf eine Sammlung, `-n <zahl>` begrenzt.
- `qmd collection add <pfad> [--name NAME] [--mask GLOB]`, `qmd collection list`, `qmd update` zum Neuindizieren.
- Die drei bestehenden Sammlungen heißen `space`, `iam-wiki`, `obsidian-ai` und tragen bereits die Ignore-Liste, die in Scheibe 1 zu `DEFAULT_EXCLUDES` wurde.

---

## Dateistruktur

```
src/brain/
  paths.py                   NEU  Zustandsverzeichnis auflösen, Orte ableiten
  privacy.py                 NEU  Channel, Modus, never — die beiden Gatter
  core.py                    NEU  die fünf Werkzeuge als Funktionen
  search/
    __init__.py              NEU
    port.py                  NEU  SearchPort, SearchHit, die Ausnahmen
    qmd.py                   NEU  die qmd-Umsetzung: Prozess, JSON, Fehler
    fake.py                  NEU  die Attrappe (liegt in src, weil Aufgabe 11 sie prüft)
  models.py                  ändern: Manifest um readonly/privacy erweitern
  manifest.py                ändern: [privacy] und readonly lesen
  registry.py                ändern: Manifest- und Artefaktort je Bereich
  walk.py                    ändern: DEFAULT_EXCLUDES, index.intro.md
  catalog.py                 ändern: Einleitung einfügen
  graph.py                   ändern: links-Block
  cli.py                     ändern: neue Befehle, --channel, --state-dir
tests/
  test_paths.py  test_privacy.py  test_search_port.py  test_search_qmd.py
  test_core_search.py  test_core_tools.py
  test_qmd_contract.py       eigener Lauf, nicht in der normalen Runde
  (bestehende Testmodule wachsen mit den geänderten Modulen)
```

`core.py` bleibt frei von Prozessaufrufen und frei von Argumentzerlegung: Das ist die Form, in die 2c den Daemon einsetzt, ohne `core` anzufassen.

---

## Aufgabe 1: Zustandsverzeichnis und Orte

Jede folgende Aufgabe schreibt oder liest irgendwo. Ohne einen einzigen Ort, der das entscheidet, verteilt sich die Pfadlogik über zehn Module — und Tests müssten das echte Verzeichnis anfassen.

**Files:**
- Create: `src/brain/paths.py`, `tests/test_paths.py`

**Interfaces:**
- Consumes: nichts
- Produces:
  - `resolve_state_dir(explicit: Path | None = None) -> Path`
  - `registry_path(state_dir: Path) -> Path`
  - `area_state_dir(state_dir: Path, scope: str) -> Path`

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

`tests/test_paths.py`:

```python
"""The one place that decides where anything lives."""

from pathlib import Path

import pytest

from brain.paths import area_state_dir, registry_path, resolve_state_dir


def test_explicit_wins_over_environment(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("BRAIN_STATE_DIR", str(tmp_path / "from_env"))
    assert resolve_state_dir(tmp_path / "explicit") == tmp_path / "explicit"


def test_environment_wins_over_platform_default(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setenv("BRAIN_STATE_DIR", str(tmp_path / "from_env"))
    assert resolve_state_dir() == tmp_path / "from_env"


def test_platform_default_is_absolute_and_named(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("BRAIN_STATE_DIR", raising=False)
    resolved = resolve_state_dir()
    assert resolved.is_absolute()
    assert resolved.name == "brain"


def test_derived_locations(tmp_path: Path) -> None:
    assert registry_path(tmp_path) == tmp_path / "registry.toml"
    assert area_state_dir(tmp_path, "engineering/python") == tmp_path / "areas" / "engineering-python"


def test_scope_separator_never_escapes_the_state_dir(tmp_path: Path) -> None:
    """A scope is user input; it must not be able to climb out of the directory."""
    resolved = area_state_dir(tmp_path, "../../etc")
    assert tmp_path in resolved.parents
```

- [ ] **Schritt 2: Test laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_paths.py -q`
Expected: FAIL mit `ModuleNotFoundError: No module named 'brain.paths'`

- [ ] **Schritt 3: Umsetzung schreiben**

`src/brain/paths.py`:

```python
"""Where everything lives. The only module that knows a platform path."""

import os
import re
import sys
from pathlib import Path

_ENV = "BRAIN_STATE_DIR"
# A scope is a user-supplied string with slashes in it. Anything that is not a
# plain name becomes a dash, so a scope can never climb out of the state
# directory or collide with a path separator on either platform.
_UNSAFE = re.compile(r"[^A-Za-z0-9_.-]+")


def resolve_state_dir(explicit: Path | None = None) -> Path:
    """Command line beats environment beats platform default (spec 5.1)."""
    if explicit is not None:
        return explicit
    from_env = os.environ.get(_ENV)
    if from_env:
        return Path(from_env)
    return _platform_default()


def _platform_default() -> Path:
    if sys.platform == "win32":
        base = os.environ.get("LOCALAPPDATA")
        # A Windows without LOCALAPPDATA is broken, but falling back beats
        # raising: the caller can still override the location explicitly.
        return Path(base) / "brain" if base else Path.home() / "AppData" / "Local" / "brain"
    return Path(os.environ.get("XDG_STATE_HOME", Path.home() / ".local" / "state")) / "brain"


def registry_path(state_dir: Path) -> Path:
    return state_dir / "registry.toml"


def area_state_dir(state_dir: Path, scope: str) -> Path:
    """Where a read-only area's artefacts live (spec 5.5, 5.8)."""
    return state_dir / "areas" / _UNSAFE.sub("-", scope).strip("-")
```

- [ ] **Schritt 4: Tests laufen lassen**

Run: `uv run pytest tests/test_paths.py -q && uv run ruff check && uv run mypy`
Expected: PASS, keine Meldungen

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/paths.py tests/test_paths.py
git commit -m "Resolve the state directory in one place"
```

---

## Aufgabe 2: `readonly`-Bereiche und `[privacy]` im Manifest

Ohne diese Aufgabe kann der Indexer die drei externen Bestände nicht anfassen, ohne die Bestandsregel zu brechen — und das Datenschutzgatter hätte nichts zu lesen.

**Files:**
- Modify: `src/brain/models.py`, `src/brain/manifest.py`, `src/brain/registry.py`
- Test: `tests/test_manifest.py`, `tests/test_registry.py`

**Interfaces:**
- Consumes: `brain.paths.area_state_dir`
- Produces:
  - `Manifest` zusätzlich mit `readonly: bool = False`, `privacy_mode: str = "manual_cloud"`, `never: tuple[str, ...] = ()`
  - `Area` zusätzlich mit `readonly: bool = False`
  - `area_artifact_dir(area: Area, state_dir: Path) -> Path` in `registry.py`
  - `manifest_path(area: Area, state_dir: Path) -> Path` in `registry.py`

- [ ] **Schritt 1: Die fehlschlagenden Tests schreiben**

Anhängen an `tests/test_manifest.py`:

```python
def test_reads_privacy_and_readonly(tmp_path: Path) -> None:
    path = tmp_path / ".brain.toml"
    path.write_text(
        '[area]\nscope = "space"\nreadonly = true\n\n'
        '[privacy]\nmode = "local_only"\nnever = ["**/*.env"]\n',
        encoding="utf-8",
    )
    manifest = read_manifest(path)
    assert manifest.readonly is True
    assert manifest.privacy_mode == "local_only"
    assert manifest.never == ("**/*.env",)


def test_privacy_defaults_when_absent(tmp_path: Path) -> None:
    path = tmp_path / ".brain.toml"
    path.write_text('[area]\nscope = "k"\n', encoding="utf-8")
    manifest = read_manifest(path)
    assert manifest.readonly is False
    assert manifest.privacy_mode == "manual_cloud"
    assert manifest.never == ()


def test_unknown_privacy_mode_is_an_error(tmp_path: Path) -> None:
    """A typo must not silently downgrade to the permissive default."""
    path = tmp_path / ".brain.toml"
    path.write_text('[area]\nscope = "k"\n\n[privacy]\nmode = "local-only"\n', encoding="utf-8")
    with pytest.raises(ManifestError, match="mode"):
        read_manifest(path)
```

Anhängen an `tests/test_registry.py`:

```python
def test_readonly_area_keeps_artefacts_in_the_state_dir(tmp_path: Path) -> None:
    registry = tmp_path / "registry.toml"
    registry.write_text(
        '[[area]]\nscope = "space"\npath = "C:/corpus"\nreadonly = true\n', encoding="utf-8"
    )
    (area,) = read_registry(registry)
    assert area.readonly is True
    assert area_artifact_dir(area, tmp_path) == tmp_path / "areas" / "space"
    assert manifest_path(area, tmp_path) == tmp_path / "areas" / "space" / ".brain.toml"


def test_writable_area_keeps_everything_in_its_tree(tmp_path: Path) -> None:
    registry = tmp_path / "registry.toml"
    registry.write_text('[[area]]\nscope = "k"\npath = "C:/vault"\n', encoding="utf-8")
    (area,) = read_registry(registry)
    assert area.readonly is False
    assert area_artifact_dir(area, tmp_path) == Path("C:/vault")
    assert manifest_path(area, tmp_path) == Path("C:/vault/.brain.toml")
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_manifest.py tests/test_registry.py -q`
Expected: FAIL — `AttributeError` auf `readonly` bzw. `ImportError` für `area_artifact_dir`

- [ ] **Schritt 3: `models.py` erweitern**

In `Manifest` ergänzen:

```python
    readonly: bool = False
    privacy_mode: str = "manual_cloud"
    never: tuple[str, ...] = ()
```

In `Area` ergänzen:

```python
    readonly: bool = False
```

- [ ] **Schritt 4: `manifest.py` erweitern**

In `read_manifest`, vor dem `return`:

```python
    privacy = data.get("privacy", {})
    mode = privacy.get("mode", "manual_cloud")
    if mode not in _PRIVACY_MODES:
        raise ManifestError(
            f"{path}: [privacy] mode must be one of {', '.join(sorted(_PRIVACY_MODES))}, "
            f"found {mode!r}"
        )
```

und im `Manifest(...)`-Aufruf `readonly=bool(area.get("readonly", False))`, `privacy_mode=mode`, `never=_globs(path, "never", privacy.get("never", []))` ergänzen. Oben im Modul:

```python
# A misspelt mode must fail loudly: silently falling back would turn the
# strictest setting in the system into the most permissive one.
_PRIVACY_MODES = frozenset({"local_only", "manual_cloud", "automatic_cloud"})
```

- [ ] **Schritt 5: `registry.py` erweitern**

`readonly` im `[[area]]`-Eintrag lesen (`bool(entry.get("readonly", False))`, an `Area` durchreichen) und die beiden Ortsfunktionen ergänzen:

```python
def area_artifact_dir(area: Area, state_dir: Path) -> Path:
    """Where this area's generated files go.

    A read-only area is one we index but do not own; writing a catalog into it
    would be the first forbidden write (spec 5.5, decision 36).
    """
    return area_state_dir(state_dir, area.scope) if area.readonly else area.path


def manifest_path(area: Area, state_dir: Path) -> Path:
    """The manifest follows the artefacts, for the same reason they moved."""
    return area_artifact_dir(area, state_dir) / ".brain.toml"
```

- [ ] **Schritt 6: Tests laufen lassen**

Run: `uv run pytest -q && uv run ruff check && uv run mypy`
Expected: PASS. Die bestehenden Indexer-Tests laufen unverändert mit, weil alle neuen Felder Vorgaben haben.

- [ ] **Schritt 7: Commit**

```bash
git add src/brain/models.py src/brain/manifest.py src/brain/registry.py tests/
git commit -m "Let an area declare itself read-only and set its privacy mode"
```

---

## Aufgabe 3: `cli.reindex` schreibt an den richtigen Ort

Aufgabe 2 hat den Ort bestimmt, benutzt ihn aber noch niemand. Bis hierher schreibt `reindex` weiterhin in den fremden Baum.

**Files:**
- Modify: `src/brain/cli.py`
- Test: `tests/test_cli.py`

**Interfaces:**
- Consumes: `area_artifact_dir`, `manifest_path`, `resolve_state_dir`, `registry_path`
- Produces: `reindex(registry_path: Path, state_dir: Path) -> int` (Signatur um `state_dir` erweitert)

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

Anhängen an `tests/test_cli.py`:

```python
def test_readonly_area_is_not_written_to(tmp_path: Path) -> None:
    """The whole point of decision 36: the corpus stays untouched."""
    corpus = tmp_path / "corpus"
    (corpus / "notes").mkdir(parents=True)
    (corpus / "notes" / "alpha.md").write_text("---\ntitle: Alpha\n---\n\nText\n", encoding="utf-8")
    before = sorted(p.relative_to(corpus).as_posix() for p in corpus.rglob("*"))

    state = tmp_path / "state"
    area_dir = state / "areas" / "corpus"
    area_dir.mkdir(parents=True)
    (area_dir / ".brain.toml").write_text(
        '[area]\nscope = "corpus"\n\n[index]\ninclude = ["**/*.md"]\n', encoding="utf-8"
    )
    registry = state / "registry.toml"
    registry.write_text(
        f'[[area]]\nscope = "corpus"\npath = "{corpus.as_posix()}"\nreadonly = true\n',
        encoding="utf-8",
    )

    assert main(["reindex", "--state-dir", str(state)]) == 0

    after = sorted(p.relative_to(corpus).as_posix() for p in corpus.rglob("*"))
    assert after == before
    assert (area_dir / "index.md").exists()
    assert (area_dir / "graph.json").exists()
    assert (area_dir / "_identities.tsv").exists()
```

- [ ] **Schritt 2: Test laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_cli.py::test_readonly_area_is_not_written_to -q`
Expected: FAIL — `main` kennt `--state-dir` nicht

- [ ] **Schritt 3: Umsetzung**

In `cli.py`: `_index_area(area, areas)` wird zu `_index_area(area, areas, state_dir)`. Jedes `area.path / "<datei>"` beim **Schreiben** wird zu `target / "<datei>"` mit `target = area_artifact_dir(area, state_dir)`; `read_manifest` und `read_identities` lesen ebenfalls von dort. Das **Suchen** der Dateien (`find_files`) bleibt bei `area.path` — die Quellen liegen weiter im Baum, nur die Ausgabe wandert.

In `_write_catalogs` wird `area.path` beim Zusammensetzen des Ziels durch den Artefaktort ersetzt; die relativen Pfade der Dokumente bleiben unverändert.

In `main`:

```python
    parser.add_argument("--state-dir", type=Path, default=None)
    ...
    state_dir = resolve_state_dir(args.state_dir)
    registry = args.registry if args.registry is not None else registry_path(state_dir)
```

`--registry` verliert damit `required=True` und wird zur Ausnahme, die auf eine andere Datei zeigt, nicht auf ein anderes Verzeichnis.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `uv run pytest -q && uv run ruff check && uv run mypy`
Expected: PASS. Die Reproduzierbarkeitstests aus Scheibe 1 laufen unverändert — sie benutzen schreibbare Bereiche.

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/cli.py tests/test_cli.py
git commit -m "Keep generated files out of read-only areas"
```

---

## Aufgabe 4: Katalog-Einleitung und die Ausschlussliste

Zwei Korrekturen aus dem Rauchtest von Scheibe 1, die zusammen in einen Commit gehören, weil beide dasselbe berühren: was in den Katalog kommt.

**Files:**
- Modify: `src/brain/catalog.py`, `src/brain/walk.py`, `src/brain/cli.py`
- Test: `tests/test_catalog.py`, `tests/test_walk.py`

**Interfaces:**
- Consumes: nichts Neues
- Produces: `render_catalog(title, entries, subdirectories, intro: str = "") -> str`

- [ ] **Schritt 1: Die fehlschlagenden Tests schreiben**

Anhängen an `tests/test_catalog.py`:

```python
def test_intro_sits_between_heading_and_areas() -> None:
    rendered = render_catalog("knowledge", [], ["98 Messung"], intro="Erster Anlaufpunkt.\n")
    assert rendered.index("Erster Anlaufpunkt.") < rendered.index("## Bereiche")
    assert rendered.index("# knowledge") < rendered.index("Erster Anlaufpunkt.")


def test_absent_intro_leaves_no_gap() -> None:
    """A missing intro must not show up as a blank line nobody wrote."""
    assert render_catalog("k", [], [], intro="") == render_catalog("k", [], [])


def test_intro_is_copied_verbatim() -> None:
    """Whatever a human wrote stays as written — no escaping, no reflowing."""
    intro = "Siehe [CLAUDE.md](<CLAUDE.md>) — *nicht* überlesen.\n"
    assert intro in render_catalog("k", [], [], intro=intro)
```

Anhängen an `tests/test_walk.py`:

```python
def test_superpowers_specs_and_plans_are_sources(tmp_path: Path) -> None:
    """Spec 5.3: sdd/ is ignored, specs/ and plans/ are sources — the contract."""
    (tmp_path / ".brain.toml").write_text(
        '[area]\nscope = "p"\n\n[index]\ninclude = ["**/*.md"]\n', encoding="utf-8"
    )
    for relative in (
        "docs/.superpowers/specs/design.md",
        "docs/.superpowers/plans/plan.md",
        "docs/.superpowers/sdd/progress.md",
    ):
        target = tmp_path / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text("x\n", encoding="utf-8")
    manifest = read_manifest(tmp_path / ".brain.toml")
    found = [p.relative_to(tmp_path).as_posix() for p in find_files(Area(scope="p", path=tmp_path), manifest)]
    assert "docs/.superpowers/specs/design.md" in found
    assert "docs/.superpowers/plans/plan.md" in found
    assert "docs/.superpowers/sdd/progress.md" not in found


def test_intro_file_is_never_indexed(tmp_path: Path) -> None:
    """It is the indexer's input, like index.md is its output."""
    (tmp_path / ".brain.toml").write_text(
        '[area]\nscope = "p"\n\n[index]\ninclude = ["**/*.md"]\n', encoding="utf-8"
    )
    (tmp_path / "index.intro.md").write_text("Hallo\n", encoding="utf-8")
    (tmp_path / "note.md").write_text("x\n", encoding="utf-8")
    manifest = read_manifest(tmp_path / ".brain.toml")
    found = [p.name for p in find_files(Area(scope="p", path=tmp_path), manifest)]
    assert found == ["note.md"]
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_catalog.py tests/test_walk.py -q`
Expected: FAIL — `render_catalog` kennt `intro` nicht; `sdd/progress.md` wird noch mit dem ganzen Ordner ausgeschlossen

- [ ] **Schritt 3: `walk.py` ändern**

`"**/.superpowers/**"` wird zu `"**/.superpowers/sdd/**"`, mit angepasstem Kommentar, und `"**/index.intro.md"` kommt zur Liste der eigenen Dateien:

```python
    # The indexer's own files. index.intro.md is its input, the other three are
    # its output; indexing any of them would make run 2 see what run 1 wrote.
    "**/index.md",
    "**/index.intro.md",
    "**/graph.json",
    "**/_identities.tsv",
```

- [ ] **Schritt 4: `catalog.py` ändern**

```python
def render_catalog(
    title: str,
    entries: Sequence[Document],
    subdirectories: Sequence[str],
    intro: str = "",
) -> str:
    """Render one catalog level. Sorted throughout, so reruns produce the same bytes."""
    parts: list[str] = [f"# {title}", ""]
    if intro:
        # Copied verbatim: it is prose a human wrote, and the generator has no
        # business reflowing, escaping or otherwise improving it.
        parts += [intro.rstrip("\n"), ""]
    ...
```

- [ ] **Schritt 5: `cli.py` — die Einleitung einlesen**

In `_write_catalogs`, je Ebene vor dem Rendern:

```python
        intro = _read_intro(area.path / directory / "index.intro.md" if directory
                            else area.path / "index.intro.md")
```

und dazu, auf Modulebene:

```python
def _read_intro(path: Path) -> str:
    """An intro that exists must say something.

    An empty one is a user error, not a special case: it would render as a gap
    nobody wrote, and the writer would never learn that their file was ignored.
    """
    if not path.exists():
        return ""
    text = path.read_text(encoding="utf-8")
    if not text.strip():
        raise ManifestError(f"{path}: intro file is empty; delete it or write something in it")
    return text
```

Die Einleitung wird aus `area.path` gelesen, **nicht** aus dem Artefaktort: Sie ist von Menschen geschrieben und gehört zum Bestand. Bei einem `readonly`-Bereich, in dem niemand schreiben darf, liegt sie folgerichtig nur dann vor, wenn der fremde Bestand selbst eine mitbringt.

- [ ] **Schritt 6: Tests laufen lassen**

Run: `uv run pytest -q && uv run ruff check && uv run mypy`
Expected: PASS

- [ ] **Schritt 7: Commit**

```bash
git add src/brain/catalog.py src/brain/walk.py src/brain/cli.py tests/
git commit -m "Give catalogs a hand-written intro and stop excluding specs and plans"
```

---

## Aufgabe 5: Der `links`-Block in `graph.json`

Der Fund aus dem Rauchtest: 0 statt 959 Kanten, und nichts an der Ausgabe sagte es.

**Files:**
- Modify: `src/brain/graph.py`
- Test: `tests/test_graph.py`

**Interfaces:**
- Consumes: nichts Neues
- Produces: `graph.json` zusätzlich mit `"links": {"total": int, "resolved": int, "dropped": {"<grund>": int}}`

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

Anhängen an `tests/test_graph.py`:

```python
import json

def test_link_statistics_separate_no_links_from_all_dropped() -> None:
    """The smoke test of slice 1 could not tell these two apart."""
    linkless = json.loads(render_graph("s", [_doc("a.md", links=())]))
    assert linkless["links"] == {"total": 0, "resolved": 0, "dropped": {}}

    all_dropped = json.loads(
        render_graph("s", [_doc("a.md", links=("/elsewhere/b.md", "https://example.org"))])
    )
    assert all_dropped["edges"] == []
    assert all_dropped["links"]["total"] == 2
    assert all_dropped["links"]["resolved"] == 0
    assert all_dropped["links"]["dropped"] == {"external": 1, "unknown_target": 1}


def test_resolved_links_are_counted_once_per_link_not_per_edge() -> None:
    """Two spellings of the same edge are one edge but two resolved links."""
    docs = [_doc("a.md", links=("b.md", "./b.md")), _doc("b.md", links=())]
    graph = json.loads(render_graph("s", docs))
    assert len(graph["edges"]) == 1
    assert graph["links"]["resolved"] == 2
```

Dazu die Hilfsfunktion `_doc(relative, *, links)`, die ein `Document` mit leeren Feldern und dem gegebenen `links`-Tupel baut — falls `tests/test_graph.py` sie noch nicht hat, hier anlegen.

- [ ] **Schritt 2: Test laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_graph.py -q`
Expected: FAIL mit `KeyError: 'links'`

- [ ] **Schritt 3: Umsetzung**

In `render_graph` beim Durchlaufen der Links mitzählen. Die Gründe sind abschließend und tragen englische Namen:

```python
    counts: dict[str, int] = {}
    total = 0
    resolved = 0
    for doc in sorted(documents, key=lambda item: item.relative):
        for link in doc.links:
            total += 1
            reason = _drop_reason(doc.relative, link, known)
            if reason is None:
                resolved += 1
            else:
                counts[reason] = counts.get(reason, 0) + 1
            ...
```

mit

```python
def _drop_reason(source: str, link: str, known: set[str]) -> str | None:
    """Why this link produced no edge — `None` means it did.

    Named reasons rather than a single count: "every link dropped" and "every
    link points outside" are different diagnoses, and the second one is what a
    mismatched area root looks like.
    """
    if link.startswith(("http://", "https://", "mailto:")):
        return "external"
    if link.startswith("#"):
        return "anchor"
    target = _resolve(source, link)
    if target is None:
        return "outside_area"
    if not any(candidate in known for candidate in (target, f"{target}.md")):
        return "unknown_target"
    return None
```

Der Block kommt sortiert in die Nutzlast: `"links": {"total": total, "resolved": resolved, "dropped": dict(sorted(counts.items()))}`. `json.dumps` läuft ohnehin mit `sort_keys=True`, die Reproduzierbarkeit bleibt.

Zusätzlich der Zusage-Kommentar aus §5.8 in den Modul-Docstring: `edges` ist eine Menge von Beziehungen — das erledigt die zweite bekannte Einschränkung aus Scheibe 1.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `uv run pytest -q && uv run ruff check && uv run mypy`
Expected: PASS

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/graph.py tests/test_graph.py
git commit -m "Count links so an empty graph is not mistaken for a corpus without links"
```

---

## Aufgabe 6: Die Naht — `SearchPort`, `SearchHit`, Attrappe

Ab hier wird Neuland gebaut. Die Naht zuerst, weil jede folgende Aufgabe entweder über ihr oder unter ihr liegt.

**Files:**
- Create: `src/brain/search/__init__.py`, `src/brain/search/port.py`, `src/brain/search/fake.py`, `tests/test_search_port.py`

**Interfaces:**
- Consumes: nichts
- Produces:
  - `class SearchHit` — `collection: str`, `relative: str`, `line: int`, `title: str`, `snippet: str`, `score: float`, `content_key: str`
  - `class Profile(StrEnum)` — `FAST`, `FULL`, `KEYWORD`, `VECTOR`
  - `class SearchPort(Protocol)` — `search(query, collections, profile, n) -> tuple[SearchHit, ...]`, `refresh(collections) -> None`
  - `class SearchUnavailable(Exception)` — die Suchmaschine hat nicht geantwortet
  - `class FakePort` — programmierbare Attrappe für Tests

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

`tests/test_search_port.py`:

```python
"""The seam. Everything above it is ours, everything below it is qmd's."""

import pytest

from brain.search.fake import FakePort
from brain.search.port import Profile, SearchHit, SearchUnavailable


def _hit(relative: str, *, key: str = "#aaa", score: float = 0.5) -> SearchHit:
    return SearchHit(
        collection="space",
        relative=relative,
        line=1,
        title="T",
        snippet="…",
        score=score,
        content_key=key,
    )


def test_fake_returns_what_it_was_given() -> None:
    port = FakePort(results=[[_hit("a.md")]])
    assert port.search("q", ("space",), Profile.FAST, 5) == (_hit("a.md"),)


def test_fake_records_the_call() -> None:
    port = FakePort(results=[[]])
    port.search("q", ("space", "iam-wiki"), Profile.FULL, 3)
    assert port.calls == [("q", ("space", "iam-wiki"), Profile.FULL, 3)]


def test_fake_can_fail_like_the_real_thing() -> None:
    port = FakePort(results=[SearchUnavailable("qmd exited with 1")])
    with pytest.raises(SearchUnavailable):
        port.search("q", ("space",), Profile.FAST, 5)


def test_fake_runs_out_of_scripted_results() -> None:
    """A test that calls more often than it scripted is a broken test."""
    port = FakePort(results=[[]])
    port.search("q", ("space",), Profile.FAST, 5)
    with pytest.raises(AssertionError):
        port.search("q", ("space",), Profile.FAST, 5)
```

- [ ] **Schritt 2: Test laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_search_port.py -q`
Expected: FAIL mit `ModuleNotFoundError: No module named 'brain.search'`

- [ ] **Schritt 3: `port.py` schreiben**

```python
"""The seam between our promises and somebody else's search engine.

Nothing below this module knows about privacy, deduplication or the difference
between an empty result and a failed one — those are ours and live above it
(spec 7.2). Nothing above it knows that qmd exists.
"""

from dataclasses import dataclass
from enum import StrEnum
from typing import Protocol


class SearchUnavailable(Exception):
    """The engine did not answer. This is never an empty result (spec 16.14)."""


class Profile(StrEnum):
    FAST = "fast"
    FULL = "full"
    KEYWORD = "keyword"
    VECTOR = "vector"


@dataclass(frozen=True, slots=True)
class SearchHit:
    """One hit, in our vocabulary rather than the engine's.

    `content_key` is what the engine believes identifies this document's
    content. We keep it to compare against our own register, not to group by
    it — the grouping is ours (decision 40).
    """

    collection: str
    relative: str
    line: int
    title: str
    snippet: str
    score: float
    content_key: str


class SearchPort(Protocol):
    def search(
        self, query: str, collections: tuple[str, ...], profile: Profile, n: int
    ) -> tuple[SearchHit, ...]: ...

    def refresh(self, collections: tuple[str, ...]) -> None: ...
```

- [ ] **Schritt 4: `fake.py` schreiben**

```python
"""A search engine that answers from a script. Spec 14 requires it."""

from brain.search.port import Profile, SearchHit, SearchUnavailable

type _Scripted = list[SearchHit] | SearchUnavailable


class FakePort:
    """Hands back scripted results and records what it was asked.

    Running out of script raises rather than returning empty: a test that calls
    more often than it scripted has stopped testing what it thinks it tests.
    """

    def __init__(self, results: list[_Scripted]) -> None:
        self._results = list(results)
        self.calls: list[tuple[str, tuple[str, ...], Profile, int]] = []
        self.refreshed: list[tuple[str, ...]] = []

    def search(
        self, query: str, collections: tuple[str, ...], profile: Profile, n: int
    ) -> tuple[SearchHit, ...]:
        self.calls.append((query, collections, profile, n))
        assert self._results, "FakePort ran out of scripted results"
        scripted = self._results.pop(0)
        if isinstance(scripted, SearchUnavailable):
            raise scripted
        return tuple(scripted)

    def refresh(self, collections: tuple[str, ...]) -> None:
        self.refreshed.append(collections)
```

- [ ] **Schritt 5: Tests laufen lassen**

Run: `uv run pytest tests/test_search_port.py -q && uv run mypy`
Expected: PASS

- [ ] **Schritt 6: Commit**

```bash
git add src/brain/search tests/test_search_port.py
git commit -m "Add the search seam and a scripted engine for tests"
```

---

## Aufgabe 7: Die qmd-Umsetzung

**Files:**
- Create: `src/brain/search/qmd.py`, `tests/test_search_qmd.py`

**Interfaces:**
- Consumes: `Profile`, `SearchHit`, `SearchUnavailable`
- Produces: `class QmdPort` mit `__init__(self, executable: str = "qmd", runner: Runner | None = None)` und `type Runner = Callable[[list[str]], CompletedProcess[str]]`

- [ ] **Schritt 1: Die fehlschlagenden Tests schreiben**

`tests/test_search_qmd.py`:

```python
"""Translating qmd's answers into ours — and its silences into exceptions."""

import json
from subprocess import CompletedProcess

import pytest

from brain.search.port import Profile, SearchUnavailable
from brain.search.qmd import QmdPort

_ROW = {
    "docid": "#d45d28",
    "score": 0.84,
    "file": "qmd://space/wiki/balancing/zeitraffer.md",
    "line": 35,
    "title": "Formel",
    "snippet": "@@ -34,4 @@\nText\r\n",
}


def _runner(*calls: CompletedProcess[str]):
    queue = list(calls)
    recorded: list[list[str]] = []

    def run(argv: list[str]) -> CompletedProcess[str]:
        recorded.append(argv)
        return queue.pop(0)

    return run, recorded


def test_parses_a_hit(monkeypatch: pytest.MonkeyPatch) -> None:
    run, _ = _runner(CompletedProcess([], 0, json.dumps([_ROW]), ""))
    (hit,) = QmdPort(runner=run).search("q", ("space",), Profile.FULL, 5)
    assert hit.collection == "space"
    assert hit.relative == "wiki/balancing/zeitraffer.md"
    assert hit.line == 35
    assert hit.content_key == "#d45d28"
    assert hit.score == pytest.approx(0.84)


def test_profile_chooses_the_subcommand() -> None:
    run, recorded = _runner(CompletedProcess([], 0, "[]", ""), CompletedProcess([], 0, "[]", ""))
    port = QmdPort(runner=run)
    port.search("q", ("space",), Profile.KEYWORD, 5)
    port.search("q", ("space",), Profile.FULL, 5)
    assert recorded[0][1] == "search"
    assert recorded[1][1] == "query"


def test_every_collection_is_passed(monkeypatch: pytest.MonkeyPatch) -> None:
    run, recorded = _runner(CompletedProcess([], 0, "[]", ""))
    QmdPort(runner=run).search("q", ("space", "iam-wiki"), Profile.FAST, 5)
    assert recorded[0].count("-c") == 2


def test_nonzero_exit_is_unavailable() -> None:
    run, _ = _runner(CompletedProcess([], 1, "", "index locked"))
    with pytest.raises(SearchUnavailable, match="index locked"):
        QmdPort(runner=run).search("q", ("space",), Profile.FAST, 5)


def test_unparseable_output_is_unavailable() -> None:
    """Broken JSON is a failure, never an empty result — spec 16.14."""
    run, _ = _runner(CompletedProcess([], 0, "Searching…", ""))
    with pytest.raises(SearchUnavailable):
        QmdPort(runner=run).search("q", ("space",), Profile.FAST, 5)


def test_empty_output_is_an_empty_result_not_a_failure() -> None:
    run, _ = _runner(CompletedProcess([], 0, "[]", ""))
    assert QmdPort(runner=run).search("q", ("space",), Profile.FAST, 5) == ()


def test_missing_executable_is_unavailable() -> None:
    def run(argv: list[str]) -> CompletedProcess[str]:
        raise FileNotFoundError(argv[0])

    with pytest.raises(SearchUnavailable, match="qmd"):
        QmdPort(runner=run).search("q", ("space",), Profile.FAST, 5)
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_search_qmd.py -q`
Expected: FAIL mit `ModuleNotFoundError: No module named 'brain.search.qmd'`

- [ ] **Schritt 3: Umsetzung**

```python
"""Calling qmd and translating its answers.

Measured against qmd 2.8.3: with `--json`, stdout carries a JSON array and
nothing else — progress lines go to stderr. An empty result is `[]` with exit
code 0, which is why an unparseable stdout has to be a failure: it is the only
remaining way to tell a broken run from an honest zero.

The `runner` seam exists so tests never start a process. Slice 2b replaces it
with a held subprocess if the numbers say so (spec 13); nothing above this
module changes when it does.
"""

import json
import subprocess
from collections.abc import Callable
from subprocess import CompletedProcess

from brain.search.port import Profile, SearchHit, SearchUnavailable

type Runner = Callable[[list[str]], CompletedProcess[str]]

_SUBCOMMAND = {
    Profile.KEYWORD: "search",
    Profile.VECTOR: "vsearch",
    Profile.FAST: "vsearch",
    Profile.FULL: "query",
}
_URI_PREFIX = "qmd://"


def _default_runner(argv: list[str]) -> CompletedProcess[str]:
    return subprocess.run(argv, capture_output=True, text=True, encoding="utf-8", check=False)


class QmdPort:
    def __init__(self, executable: str = "qmd", runner: Runner | None = None) -> None:
        self._executable = executable
        self._run = runner if runner is not None else _default_runner

    def search(
        self, query: str, collections: tuple[str, ...], profile: Profile, n: int
    ) -> tuple[SearchHit, ...]:
        argv = [self._executable, _SUBCOMMAND[profile], query, "--json", "-n", str(n)]
        for collection in collections:
            argv += ["-c", collection]
        return tuple(_parse(self._invoke(argv)))

    def refresh(self, collections: tuple[str, ...]) -> None:
        self._invoke([self._executable, "update"])

    def _invoke(self, argv: list[str]) -> str:
        try:
            completed = self._run(argv)
        except OSError as error:
            # A missing qmd is the same class of event as a crashed one: the
            # engine did not answer. Catalog and read must keep working (7.5).
            raise SearchUnavailable(f"cannot run {argv[0]}: {error}") from error
        if completed.returncode != 0:
            raise SearchUnavailable(
                f"{argv[0]} exited with {completed.returncode}: {completed.stderr.strip()}"
            )
        return completed.stdout


def _parse(stdout: str) -> list[SearchHit]:
    try:
        rows = json.loads(stdout)
    except json.JSONDecodeError as error:
        raise SearchUnavailable(f"could not read the search output: {error}") from error
    if not isinstance(rows, list):
        raise SearchUnavailable(f"expected a list of hits, found {type(rows).__name__}")
    return [_hit(row) for row in rows]


def _hit(row: object) -> SearchHit:
    if not isinstance(row, dict):
        raise SearchUnavailable(f"expected a hit object, found {row!r}")
    collection, relative = _split_uri(str(row.get("file", "")))
    return SearchHit(
        collection=collection,
        relative=relative,
        line=int(row.get("line", 0)),
        title=str(row.get("title", "")),
        snippet=str(row.get("snippet", "")),
        score=float(row.get("score", 0.0)),
        content_key=str(row.get("docid", "")),
    )


def _split_uri(uri: str) -> tuple[str, str]:
    """`qmd://space/wiki/a.md` becomes `("space", "wiki/a.md")`."""
    if not uri.startswith(_URI_PREFIX):
        raise SearchUnavailable(f"expected a {_URI_PREFIX} location, found {uri!r}")
    collection, _, relative = uri[len(_URI_PREFIX) :].partition("/")
    return collection, relative
```

- [ ] **Schritt 4: Tests laufen lassen**

Run: `uv run pytest -q && uv run ruff check && uv run mypy`
Expected: PASS

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/search/qmd.py tests/test_search_qmd.py
git commit -m "Speak qmd behind the seam"
```

---

## Aufgabe 8: Datenschutz — Kanal und die beiden Gatter

**Files:**
- Create: `src/brain/privacy.py`, `tests/test_privacy.py`

**Interfaces:**
- Consumes: `Manifest`
- Produces:
  - `class Channel(StrEnum)` — `LOCAL`, `CLOUD`
  - `is_visible(manifest: Manifest, channel: Channel) -> bool`
  - `is_readable(manifest: Manifest, relative: str) -> bool`
  - `class AccessDenied(Exception)`

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

`tests/test_privacy.py`:

```python
"""The two gates. Spec 7.2.1 and decision 31."""

from brain.models import Manifest
from brain.privacy import Channel, is_readable, is_visible


def _manifest(mode: str = "manual_cloud", never: tuple[str, ...] = ()) -> Manifest:
    return Manifest(scope="s", privacy_mode=mode, never=never)


def test_local_only_is_invisible_to_the_cloud() -> None:
    assert is_visible(_manifest("local_only"), Channel.CLOUD) is False


def test_local_only_is_fully_visible_locally() -> None:
    assert is_visible(_manifest("local_only"), Channel.LOCAL) is True


def test_the_other_modes_are_visible_on_both_channels() -> None:
    for mode in ("manual_cloud", "automatic_cloud"):
        for channel in Channel:
            assert is_visible(_manifest(mode), channel) is True


def test_never_blocks_a_path_on_every_channel() -> None:
    manifest = _manifest(never=("**/secrets/**", "**/*.env"))
    assert is_readable(manifest, "notes/secrets/key.md") is False
    assert is_readable(manifest, "deploy/.env") is False
    assert is_readable(manifest, "notes/public.md") is True


def test_never_matches_at_the_area_root() -> None:
    """`**/` has to match at depth zero, or a root-level secret slips through."""
    assert is_readable(_manifest(never=("**/*.env",)), ".env") is False
```

- [ ] **Schritt 2: Test laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_privacy.py -q`
Expected: FAIL mit `ModuleNotFoundError: No module named 'brain.privacy'`

- [ ] **Schritt 3: Umsetzung**

```python
"""Who may see what, on which channel.

Both gates live here rather than in the MCP adapter, because a gate in the
adapter is a gate the next client forgets (decision 38). They are pure
functions of the manifest, so a test proves them without a search engine.
"""

from enum import StrEnum
from pathlib import PurePosixPath

from brain.models import Manifest


class AccessDenied(Exception):
    """Refused on purpose. Never returned as empty content (spec 16.14)."""


class Channel(StrEnum):
    LOCAL = "local"
    CLOUD = "cloud"


def is_visible(manifest: Manifest, channel: Channel) -> bool:
    """`local_only` means the area does not exist for the cloud model."""
    return not (manifest.privacy_mode == "local_only" and channel is Channel.CLOUD)


def is_readable(manifest: Manifest, relative: str) -> bool:
    """`never` excludes a path everywhere, on every channel (spec 5.5).

    Checked on read as well as on index, for the case where a path falls under
    `never` after it was indexed and the index has not caught up.
    """
    candidate = PurePosixPath(relative)
    return not any(candidate.full_match(pattern) for pattern in manifest.never)
```

- [ ] **Schritt 4: Tests laufen lassen**

Run: `uv run pytest tests/test_privacy.py -q && uv run mypy`
Expected: PASS

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/privacy.py tests/test_privacy.py
git commit -m "Gate visibility by channel and paths by never"
```

---

## Aufgabe 9: `core.search` — Entdopplung, Wiederholung, Befunde

Die eigentliche Scheibe. Alles, was die Spec als „nicht verhandelbar" bezeichnet, liegt in dieser Aufgabe.

**Files:**
- Create: `src/brain/core.py`, `tests/test_core_search.py`

**Interfaces:**
- Consumes: `SearchPort`, `Channel`, `is_visible`, `is_readable`, `read_identities`, `read_manifest`, `read_registry`, `area_artifact_dir`, `manifest_path`
- Produces:
  - `@dataclass Result` — `relative`, `scope`, `doc_id: str | None`, `title`, `snippet`, `score`, `line`, `also_at: tuple[str, ...]`
  - `@dataclass SearchAnswer` — `results: tuple[Result, ...]`, `findings: tuple[str, ...]`
  - `search(query, *, scope, profile, n, channel, port, registry, state_dir) -> SearchAnswer`

- [ ] **Schritt 1: Die fehlschlagenden Tests schreiben**

`tests/test_core_search.py` — jeder Test setzt einen Bestand aus zwei Bereichen auf, einen davon `local_only`. Die Fixture legt Registrierung, Manifeste und `_identities.tsv` unter `tmp_path` an; der `FakePort` liefert die Trefferliste.

```python
"""What the core promises above the seam."""

from pathlib import Path

import pytest

from brain.core import search
from brain.privacy import Channel
from brain.search.fake import FakePort
from brain.search.port import Profile, SearchHit, SearchUnavailable


@pytest.fixture
def world(tmp_path: Path) -> Path:
    """Two areas: `open` (manual_cloud) and `closed` (local_only)."""
    state = tmp_path / "state"
    rows = []
    for scope, mode in (("open", "manual_cloud"), ("closed", "local_only")):
        area_dir = state / "areas" / scope
        area_dir.mkdir(parents=True)
        (area_dir / ".brain.toml").write_text(
            f'[area]\nscope = "{scope}"\n\n[index]\ninclude = ["**/*.md"]\n\n'
            f'[privacy]\nmode = "{mode}"\nnever = ["**/secrets/**"]\n',
            encoding="utf-8",
        )
        (area_dir / "_identities.tsv").write_text(
            "doc_id\tpfad\tcontent_hash\trevision\n"
            f"01AAA\ta.md\tsha256:1\t1\n"
            f"01AAA\tcopy/a.md\tsha256:1\t1\n"
            f"01BBB\tsecrets/key.md\tsha256:2\t1\n",
            encoding="utf-8",
        )
        rows.append(
            f'[[area]]\nscope = "{scope}"\npath = "{(tmp_path / scope).as_posix()}"\n'
            "readonly = true\n"
        )
        (tmp_path / scope).mkdir()
    (state / "registry.toml").write_text("\n".join(rows), encoding="utf-8")
    return state


def _hit(collection: str, relative: str, key: str = "#aaa", score: float = 0.5) -> SearchHit:
    return SearchHit(
        collection=collection, relative=relative, line=1, title="T",
        snippet="…", score=score, content_key=key,
    )


def test_same_document_under_two_paths_appears_once(world: Path) -> None:
    port = FakePort(results=[[_hit("open", "a.md", score=0.9), _hit("open", "copy/a.md", score=0.7)]])
    answer = search("q", scope="all", profile=Profile.FULL, n=5,
                    channel=Channel.LOCAL, port=port, state_dir=world)
    assert [r.relative for r in answer.results] == ["a.md"]
    assert answer.results[0].also_at == ("copy/a.md",)


def test_n_is_applied_after_deduplication(world: Path) -> None:
    """Otherwise a list of one gives up its only slot to a repetition."""
    port = FakePort(results=[[
        _hit("open", "a.md", score=0.9), _hit("open", "copy/a.md", score=0.8),
    ]])
    answer = search("q", scope="all", profile=Profile.FULL, n=1,
                    channel=Channel.LOCAL, port=port, state_dir=world)
    assert len(answer.results) == 1
    assert answer.results[0].also_at == ("copy/a.md",)


def test_local_only_area_is_never_asked_on_the_cloud_channel(world: Path) -> None:
    port = FakePort(results=[[]])
    search("q", scope="all", profile=Profile.FULL, n=5,
           channel=Channel.CLOUD, port=port, state_dir=world)
    (_, collections, _, _) = port.calls[0]
    assert collections == ("open",)


def test_local_channel_asks_both(world: Path) -> None:
    port = FakePort(results=[[]])
    search("q", scope="all", profile=Profile.FULL, n=5,
           channel=Channel.LOCAL, port=port, state_dir=world)
    (_, collections, _, _) = port.calls[0]
    assert collections == ("closed", "open")


def test_never_paths_are_dropped_on_every_channel(world: Path) -> None:
    port = FakePort(results=[[_hit("open", "secrets/key.md")]])
    answer = search("q", scope="all", profile=Profile.FULL, n=5,
                    channel=Channel.LOCAL, port=port, state_dir=world)
    assert answer.results == ()


def test_empty_result_is_retried_exactly_once(world: Path) -> None:
    """qmd goes quiet now and then — spec 16.14."""
    port = FakePort(results=[[], [_hit("open", "a.md")]])
    answer = search("q", scope="all", profile=Profile.FULL, n=5,
                    channel=Channel.LOCAL, port=port, state_dir=world)
    assert len(port.calls) == 2
    assert [r.relative for r in answer.results] == ["a.md"]


def test_twice_empty_is_an_empty_answer_not_an_error(world: Path) -> None:
    port = FakePort(results=[[], []])
    answer = search("q", scope="all", profile=Profile.FULL, n=5,
                    channel=Channel.LOCAL, port=port, state_dir=world)
    assert answer.results == ()
    assert len(port.calls) == 2


def test_engine_failure_propagates(world: Path) -> None:
    port = FakePort(results=[SearchUnavailable("qmd exited with 1")])
    with pytest.raises(SearchUnavailable):
        search("q", scope="all", profile=Profile.FULL, n=5,
               channel=Channel.LOCAL, port=port, state_dir=world)


def test_unknown_path_is_kept_and_reported(world: Path) -> None:
    """Decision 40: divergence is a finding, not a silently dropped hit."""
    port = FakePort(results=[[_hit("open", "brand-new.md")]])
    answer = search("q", scope="all", profile=Profile.FULL, n=5,
                    channel=Channel.LOCAL, port=port, state_dir=world)
    assert [r.relative for r in answer.results] == ["brand-new.md"]
    assert answer.results[0].doc_id is None
    assert any("brand-new.md" in finding for finding in answer.findings)


def test_engine_grouping_that_disagrees_is_reported(world: Path) -> None:
    """Same doc_id in our register, two different content keys at qmd."""
    port = FakePort(results=[[
        _hit("open", "a.md", key="#one"), _hit("open", "copy/a.md", key="#two"),
    ]])
    answer = search("q", scope="all", profile=Profile.FULL, n=5,
                    channel=Channel.LOCAL, port=port, state_dir=world)
    assert [r.relative for r in answer.results] == ["a.md"]
    assert any("#one" in f and "#two" in f for f in answer.findings)


def test_scope_narrows_to_one_area(world: Path) -> None:
    port = FakePort(results=[[]])
    search("q", scope="open", profile=Profile.FULL, n=5,
           channel=Channel.LOCAL, port=port, state_dir=world)
    (_, collections, _, _) = port.calls[0]
    assert collections == ("open",)
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_core_search.py -q`
Expected: FAIL mit `ModuleNotFoundError: No module named 'brain.core'`

- [ ] **Schritt 3: Umsetzung**

`src/brain/core.py`, der Suchteil:

```python
"""The five write-free tools, as plain functions.

No process starts here and no argument is parsed here: this is the shape slice
2c puts a daemon underneath without touching it (spec 7.1).
"""

from dataclasses import dataclass, replace
from pathlib import Path

from brain.identity import read_identities
from brain.manifest import read_manifest
from brain.models import Area, Identity, Manifest
from brain.paths import registry_path
from brain.privacy import AccessDenied, Channel, is_readable, is_visible
from brain.registry import area_artifact_dir, manifest_path, read_registry
from brain.search.port import Profile, SearchHit, SearchPort


@dataclass(frozen=True, slots=True)
class Result:
    relative: str
    scope: str
    doc_id: str | None
    title: str
    snippet: str
    score: float
    line: int
    also_at: tuple[str, ...] = ()


@dataclass(frozen=True, slots=True)
class SearchAnswer:
    """Results and what went sideways while producing them.

    `findings` is deliberately not an error: a divergence between the two
    indexes degrades the answer, it does not invalidate it (decision 40).
    """

    results: tuple[Result, ...]
    findings: tuple[str, ...]


def search(
    query: str,
    *,
    scope: str,
    profile: Profile,
    n: int,
    channel: Channel,
    port: SearchPort,
    state_dir: Path,
) -> SearchAnswer:
    areas = _visible_areas(scope, channel, state_dir)
    hits = _ask(port, query, tuple(sorted(a.scope for a, _ in areas)), profile, n)
    return _assemble(hits, areas, state_dir, n)


def _ask(
    port: SearchPort, query: str, collections: tuple[str, ...], profile: Profile, n: int
) -> tuple[SearchHit, ...]:
    """One retry on empty, then believe it.

    qmd occasionally returns an empty set without failing — measured at four in
    roughly 120 queries during slice 0. `n * 4` is asked for because
    deduplication happens afterwards and would otherwise return fewer than the
    caller asked for.
    """
    hits = port.search(query, collections, profile, n * 4)
    if not hits:
        hits = port.search(query, collections, profile, n * 4)
    return hits
```

Die Zusammensetzung, sorgfältig, weil hier die drei Zusagen sitzen:

```python
def _assemble(
    hits: tuple[SearchHit, ...],
    areas: list[tuple[Area, Manifest]],
    state_dir: Path,
    n: int,
) -> SearchAnswer:
    manifests = {area.scope: manifest for area, manifest in areas}
    registers = {
        area.scope: read_identities(area_artifact_dir(area, state_dir) / "_identities.tsv")
        for area, _ in areas
    }
    findings: list[str] = []
    ordered: list[Result] = []
    by_document: dict[tuple[str, str], int] = {}
    keys_seen: dict[tuple[str, str], set[str]] = {}

    for hit in hits:
        manifest = manifests.get(hit.collection)
        if manifest is None or not is_readable(manifest, hit.relative):
            continue
        identity = registers[hit.collection].get(hit.relative)
        if identity is None:
            findings.append(
                f"{hit.collection}/{hit.relative}: hit is not in the register; "
                "counted as its own document — reindex to catch up"
            )
        key = (hit.collection, identity.doc_id if identity else f"path:{hit.relative}")
        keys_seen.setdefault(key, set()).add(hit.content_key)
        position = by_document.get(key)
        if position is None:
            by_document[key] = len(ordered)
            ordered.append(_result(hit, identity))
        else:
            previous = ordered[position]
            ordered[position] = replace(previous, also_at=previous.also_at + (hit.relative,))

    for (collection, doc), engine_keys in keys_seen.items():
        if len(engine_keys) > 1:
            findings.append(
                f"{collection}/{doc}: our register groups these paths, the engine does not "
                f"({', '.join(sorted(engine_keys))})"
            )
    return SearchAnswer(tuple(ordered[:n]), tuple(findings))


def _result(hit: SearchHit, identity: Identity | None) -> Result:
    return Result(
        relative=hit.relative,
        scope=hit.collection,
        doc_id=identity.doc_id if identity else None,
        title=hit.title,
        snippet=hit.snippet,
        score=hit.score,
        line=hit.line,
    )


def _visible_areas(scope: str, channel: Channel, state_dir: Path) -> list[tuple[Area, Manifest]]:
    """Areas the caller may see, resolved before anything is asked.

    Filtering afterwards would mean the invisible area was queried — and a
    query is already a disclosure of the question (spec 7.2.1).
    """
    visible: list[tuple[Area, Manifest]] = []
    for area in read_registry(registry_path(state_dir)):
        if scope not in ("all", area.scope):
            continue
        manifest = read_manifest(manifest_path(area, state_dir))
        if is_visible(manifest, channel):
            visible.append((area, manifest))
    return visible
```

- [ ] **Schritt 4: Tests laufen lassen**

Run: `uv run pytest -q && uv run ruff check && uv run mypy`
Expected: PASS

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/core.py tests/test_core_search.py
git commit -m "Deduplicate by our own register and report where the indexes disagree"
```

---

## Aufgabe 10: `catalog`, `read`, `neighbors`, `status`

**Files:**
- Modify: `src/brain/core.py`
- Test: `tests/test_core_tools.py`

**Interfaces:**
- Produces:
  - `catalog(*, scope: str, channel: Channel, state_dir: Path) -> str`
  - `read(*, scope: str, relative: str, section: str | None, channel: Channel, state_dir: Path) -> str`
  - `neighbors(*, scope: str, relative: str, channel: Channel, state_dir: Path) -> tuple[tuple[str, ...], tuple[str, ...]]` — eingehend, ausgehend
  - `status(*, channel: Channel, state_dir: Path) -> tuple[str, ...]`

- [ ] **Schritt 1: Die fehlschlagenden Tests schreiben**

`tests/test_core_tools.py` benutzt dieselbe `world`-Fixture wie Aufgabe 9 (in `tests/conftest.py` verschieben, wenn sie dort noch nicht liegt) und ergänzt Katalog- und Graph-Dateien im Artefaktverzeichnis.

```python
def test_catalog_omits_an_invisible_area(world: Path) -> None:
    rendered = catalog(scope="all", channel=Channel.CLOUD, state_dir=world)
    assert "open" in rendered
    assert "closed" not in rendered


def test_catalog_shows_everything_locally(world: Path) -> None:
    rendered = catalog(scope="all", channel=Channel.LOCAL, state_dir=world)
    assert "closed" in rendered


def test_read_refuses_an_invisible_area_explicitly(world: Path) -> None:
    with pytest.raises(AccessDenied):
        read(scope="closed", relative="a.md", section=None,
             channel=Channel.CLOUD, state_dir=world)


def test_read_refuses_a_never_path_on_both_channels(world: Path) -> None:
    for channel in Channel:
        with pytest.raises(AccessDenied):
            read(scope="open", relative="secrets/key.md", section=None,
                 channel=channel, state_dir=world)


def test_read_reports_a_missing_file_instead_of_returning_nothing(world: Path) -> None:
    """Spec 16.14 for read: a failed read is never empty content."""
    with pytest.raises(FileNotFoundError):
        read(scope="open", relative="gone.md", section=None,
             channel=Channel.LOCAL, state_dir=world)


def test_read_returns_one_section(world: Path) -> None:
    (world.parent / "open" / "a.md").write_text(
        "# Titel\n\nEins\n\n## Zwei\n\nZwei-Text\n\n## Drei\n\nDrei-Text\n", encoding="utf-8"
    )
    assert read(scope="open", relative="a.md", section="Zwei",
                channel=Channel.LOCAL, state_dir=world).strip() == "## Zwei\n\nZwei-Text"


def test_neighbors_reads_both_directions(world: Path) -> None:
    (world / "areas" / "open" / "graph.json").write_text(
        '{"scope":"open","nodes":[],"edges":[{"from":"b.md","to":"a.md"},'
        '{"from":"a.md","to":"c.md"}],"links":{"total":2,"resolved":2,"dropped":{}}}\n',
        encoding="utf-8",
    )
    incoming, outgoing = neighbors(scope="open", relative="a.md",
                                   channel=Channel.LOCAL, state_dir=world)
    assert incoming == ("b.md",)
    assert outgoing == ("c.md",)


def test_status_names_a_missing_area_path(world: Path) -> None:
    (world.parent / "open").rmdir()
    assert any("open" in line for line in status(channel=Channel.LOCAL, state_dir=world))


def test_status_flags_a_low_link_resolution_rate(world: Path) -> None:
    (world / "areas" / "open" / "graph.json").write_text(
        '{"scope":"open","nodes":[],"edges":[],'
        '"links":{"total":979,"resolved":0,"dropped":{"unknown_target":979}}}\n',
        encoding="utf-8",
    )
    assert any("979" in line for line in status(channel=Channel.LOCAL, state_dir=world))
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_core_tools.py -q`
Expected: FAIL mit `ImportError: cannot import name 'catalog' from 'brain.core'`

- [ ] **Schritt 3: Umsetzung**

Anhängen an `core.py`. Der Kern liest, was der Indexer geschrieben hat, und wendet dieselben zwei Gatter an:

```python
def catalog(*, scope: str, channel: Channel, state_dir: Path) -> str:
    """The root catalog, or one area's — never a line about an invisible area."""
    areas = _visible_areas(scope, channel, state_dir)
    if scope != "all":
        area, _ = _single(areas, scope)
        return (area_artifact_dir(area, state_dir) / "index.md").read_text(encoding="utf-8")
    lines = ["# brain", ""]
    lines += [f"* [{area.scope}](brain://{area.scope}/)" for area, _ in sorted(
        areas, key=lambda pair: pair[0].scope
    )]
    return "\n".join(lines) + "\n"


def read(
    *, scope: str, relative: str, section: str | None, channel: Channel, state_dir: Path
) -> str:
    """Exactly one file, optionally one section of it.

    Every refusal raises. Returning empty content for a refused or missing read
    would be indistinguishable from an empty file (spec 16.14).
    """
    area, manifest = _single(_visible_areas(scope, channel, state_dir), scope)
    if not is_readable(manifest, relative):
        raise AccessDenied(f"{scope}/{relative} is excluded by [privacy] never")
    text = (area.path / relative).read_text(encoding="utf-8")
    return text if section is None else _section(text, section)


def _single(areas: list[tuple[Area, Manifest]], scope: str) -> tuple[Area, Manifest]:
    for area, manifest in areas:
        if area.scope == scope:
            return area, manifest
    # Indistinguishable from "does not exist" on purpose: telling a cloud caller
    # that an area exists but is hidden would leak the thing being hidden.
    raise AccessDenied(f"no visible area named {scope!r}")


def _section(text: str, heading: str) -> str:
    """From the matching heading to the next heading of the same or higher level."""
    lines = text.splitlines()
    start = next(
        (i for i, line in enumerate(lines)
         if line.startswith("#") and line.lstrip("#").strip() == heading),
        None,
    )
    if start is None:
        raise LookupError(f"no section titled {heading!r}")
    level = len(lines[start]) - len(lines[start].lstrip("#"))
    end = next(
        (i for i in range(start + 1, len(lines))
         if lines[i].startswith("#")
         and len(lines[i]) - len(lines[i].lstrip("#")) <= level),
        len(lines),
    )
    return "\n".join(lines[start:end]).rstrip() + "\n"
```

```python
def neighbors(
    *, scope: str, relative: str, channel: Channel, state_dir: Path
) -> tuple[tuple[str, ...], tuple[str, ...]]:
    """Incoming and outgoing links of one page, from the graph the indexer wrote."""
    area, _ = _single(_visible_areas(scope, channel, state_dir), scope)
    edges = _graph(area, state_dir)["edges"]
    incoming = tuple(sorted(e["from"] for e in edges if e["to"] == relative))
    outgoing = tuple(sorted(e["to"] for e in edges if e["from"] == relative))
    return incoming, outgoing


def status(*, channel: Channel, state_dir: Path) -> tuple[str, ...]:
    """One line per thing a reader should know before trusting an answer."""
    lines: list[str] = []
    for area, _ in _visible_areas("all", channel, state_dir):
        if not area.path.exists():
            lines.append(f"{area.scope}: {area.path} does not exist; skipped")
            continue
        if not (area_artifact_dir(area, state_dir) / "graph.json").exists():
            lines.append(f"{area.scope}: never indexed; run `brain reindex`")
            continue
        links = _graph(area, state_dir)["links"]
        total, resolved = links["total"], links["resolved"]
        # Half is a deliberately loud threshold: the failure this catches is
        # total, not gradual — a mismatched area root drops every single link.
        if total and resolved * 2 < total:
            dropped = ", ".join(f"{k}={v}" for k, v in sorted(links["dropped"].items()))
            lines.append(f"{area.scope}: only {resolved} of {total} links resolved ({dropped})")
    return tuple(lines)


def _graph(area: Area, state_dir: Path) -> dict[str, Any]:
    """Read the area's graph.

    `Any` is unavoidable at the json boundary and confined to this one function;
    every caller narrows what it takes out immediately.
    """
    path = area_artifact_dir(area, state_dir) / "graph.json"
    loaded: dict[str, Any] = json.loads(path.read_text(encoding="utf-8"))
    return loaded
```

Dafür kommen `import json` und `from typing import Any` oben ins Modul.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `uv run pytest -q && uv run ruff check && uv run mypy`
Expected: PASS

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/core.py tests/test_core_tools.py
git commit -m "Add catalog, read, neighbors and status behind the same two gates"
```

---

## Aufgabe 11: Die CLI

**Files:**
- Modify: `src/brain/cli.py`
- Test: `tests/test_cli.py`

**Interfaces:**
- Produces: `brain search|catalog|read|neighbors|status` mit `--scope`, `--profile`, `-n`, `--section`, `--channel`, `--state-dir`

- [ ] **Schritt 1: Die fehlschlagenden Tests schreiben**

```python
def test_search_prints_findings_to_stderr_not_into_the_results(
    world: Path, capsys: pytest.CaptureFixture[str], monkeypatch: pytest.MonkeyPatch
) -> None:
    """A finding must not be mistaken for a hit by whoever reads stdout."""
    monkeypatch.setattr("brain.cli._port", lambda: FakePort(results=[[_hit("open", "new.md")]]))
    assert main(["search", "q", "--state-dir", str(world)]) == 0
    captured = capsys.readouterr()
    assert "new.md" in captured.out
    assert "register" in captured.err


def test_search_reports_an_engine_failure_and_fails(
    world: Path, capsys: pytest.CaptureFixture[str], monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setattr(
        "brain.cli._port", lambda: FakePort(results=[SearchUnavailable("qmd exited with 1")])
    )
    assert main(["search", "q", "--state-dir", str(world)]) == 1
    assert "qmd exited with 1" in capsys.readouterr().err


def test_empty_search_is_success_with_an_explicit_line(
    world: Path, capsys: pytest.CaptureFixture[str], monkeypatch: pytest.MonkeyPatch
) -> None:
    """Spec 16.14 at the surface: the user must see the difference."""
    monkeypatch.setattr("brain.cli._port", lambda: FakePort(results=[[], []]))
    assert main(["search", "q", "--state-dir", str(world)]) == 0
    assert "no matches" in capsys.readouterr().out


def test_channel_flag_reaches_the_gate(
    world: Path, capsys: pytest.CaptureFixture[str]
) -> None:
    assert main(["catalog", "--channel", "cloud", "--state-dir", str(world)]) == 0
    assert "closed" not in capsys.readouterr().out
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_cli.py -q`
Expected: FAIL — `invalid choice: 'search'`

- [ ] **Schritt 3: Umsetzung**

Je Befehl ein Unterparser; `--channel` mit `choices=[c.value for c in Channel]` und Vorgabe `local`; `--profile` mit Vorgabe `full`. Alle Ausnahmen aus `core` und `port` werden an einer Stelle gefangen:

```python
def _port() -> SearchPort:
    """One seam for tests to replace; nothing else constructs a port."""
    return QmdPort()
```

```python
    except (AccessDenied, LookupError, SearchUnavailable, OSError,
            IdentityError, ManifestError, RegistryError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 1
```

Die Ausgabe von `search`:

```python
def _print_search(answer: SearchAnswer) -> int:
    """Results to stdout, findings to stderr.

    Mixing them would let a diagnostic line be read as a hit by anything that
    parses stdout — including the daemon of the next slice.
    """
    if not answer.results:
        print("no matches")
    for result in answer.results:
        print(
            f"brain://{result.scope}/{result.relative}:{result.line}  "
            f"{result.score:.0%}  {result.title}"
        )
        for line in result.snippet.splitlines():
            print(f"    {line}")
        if result.also_at:
            print(f"    also at: {', '.join(result.also_at)}")
        print()
    for finding in answer.findings:
        print(f"note: {finding}", file=sys.stderr)
    return 0
```

`no matches` bei Rückgabewert 0 gegenüber `error: …` bei Rückgabewert 1 ist die Stelle, an der §16.14 für einen Menschen sichtbar wird.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `uv run pytest -q && uv run ruff check && uv run mypy && uv run coverage run -m pytest && uv run coverage report`
Expected: PASS, Coverage 100 %

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/cli.py tests/test_cli.py
git commit -m "Expose the five tools on the command line"
```

---

## Aufgabe 12: `reindex` treibt qmds Sammlung

Erst hier bekommt die Ausschlussliste ihre eine Quelle (Entscheidung 39).

**Gemessen, nicht vermutet:** qmd hält seine Sammlungen in `~/.config/qmd/index.yml`, nicht in einer Datenbank. Die Datei trägt je Sammlung `path`, `pattern` und `ignore`, daneben einen `models:`-Block mit den drei Modellpfaden. Über die CLI ist die Ignore-Liste **nicht** setzbar: `collection add` kennt nur `--name` und `--mask`, und `collection list --json` gibt es nicht — der Schalter wird stillschweigend ignoriert und die Menschentabelle gedruckt. Wer hier eine JSON-Ausgabe einplant, baut auf Sand.

Deshalb schreibt `brain` die Datei. **Nur den `collections`-Block, und darin nur die registrierten Bereiche** — `models:` und fremde Sammlungen bleiben, wie sie sind.

**Files:**
- Create: `src/brain/search/qmd_config.py`, `tests/test_qmd_config.py`
- Modify: `src/brain/cli.py`
- Test: `tests/test_cli.py`

**Interfaces:**
- Consumes: `Manifest`, `Area`, `DEFAULT_EXCLUDES`
- Produces:
  - `qmd_config_path() -> Path`
  - `sync_collections(config: Path, wanted: dict[str, CollectionSpec]) -> tuple[str, ...]` — gibt die Namen der geänderten Sammlungen zurück
  - `@dataclass CollectionSpec` — `path: Path`, `pattern: str`, `ignore: tuple[str, ...]`

- [ ] **Schritt 1: Die fehlschlagenden Tests schreiben**

`tests/test_qmd_config.py`:

```python
"""Writing somebody else's configuration file, as narrowly as possible."""

from pathlib import Path

import yaml

from brain.search.qmd_config import CollectionSpec, sync_collections

_EXISTING = """collections:
  space:
    path: C:/corpus
    pattern: "**/*.md"
    ignore:
      - "**/.git/**"
  foreign:
    path: C:/elsewhere
    pattern: "**/*.md"
    ignore: []
models:
  embed: hf:some/model.gguf
"""


def _write(tmp_path: Path) -> Path:
    config = tmp_path / "index.yml"
    config.write_text(_EXISTING, encoding="utf-8")
    return config


def test_models_block_survives(tmp_path: Path) -> None:
    """Clobbering this would break the user's search until they noticed."""
    config = _write(tmp_path)
    sync_collections(
        config, {"space": CollectionSpec(Path("C:/corpus"), "**/*.md", ("**/.git/**",))}
    )
    loaded = yaml.safe_load(config.read_text(encoding="utf-8"))
    assert loaded["models"] == {"embed": "hf:some/model.gguf"}


def test_collection_we_do_not_manage_survives(tmp_path: Path) -> None:
    config = _write(tmp_path)
    sync_collections(config, {"space": CollectionSpec(Path("C:/corpus"), "**/*.md", ())})
    assert "foreign" in yaml.safe_load(config.read_text(encoding="utf-8"))["collections"]


def test_ignore_list_is_taken_over(tmp_path: Path) -> None:
    config = _write(tmp_path)
    changed = sync_collections(
        config,
        {
            "space": CollectionSpec(
                Path("C:/corpus"), "**/*.md", ("**/.git/**", "**/node_modules/**")
            )
        },
    )
    entry = yaml.safe_load(config.read_text(encoding="utf-8"))["collections"]["space"]
    assert entry["ignore"] == ["**/.git/**", "**/node_modules/**"]
    assert changed == ("space",)


def test_unchanged_config_is_not_rewritten(tmp_path: Path) -> None:
    """reindex runs often; a rewrite per run would churn the file for nothing."""
    config = _write(tmp_path)
    before = config.read_bytes()
    changed = sync_collections(
        config, {"space": CollectionSpec(Path("C:/corpus"), "**/*.md", ("**/.git/**",))}
    )
    assert changed == ()
    assert config.read_bytes() == before


def test_a_backup_is_kept_before_the_first_write(tmp_path: Path) -> None:
    config = _write(tmp_path)
    sync_collections(config, {"space": CollectionSpec(Path("C:/corpus"), "**/*.md", ())})
    assert (tmp_path / "index.yml.brain-backup").read_text(encoding="utf-8") == _EXISTING


def test_missing_config_is_created_with_only_our_collections(tmp_path: Path) -> None:
    config = tmp_path / "index.yml"
    sync_collections(config, {"space": CollectionSpec(Path("C:/corpus"), "**/*.md", ())})
    loaded = yaml.safe_load(config.read_text(encoding="utf-8"))
    assert list(loaded["collections"]) == ["space"]
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_qmd_config.py -q`
Expected: FAIL mit `ModuleNotFoundError: No module named 'brain.search.qmd_config'`

- [ ] **Schritt 3: Umsetzung**

`src/brain/search/qmd_config.py`:

```python
"""qmd's own configuration, edited as narrowly as we can manage.

Measured against qmd 2.8.3: `~/.config/qmd/index.yml` holds `collections` and
`models`. The ignore list is reachable no other way — `collection add` has no
flag for it — and two separately maintained ignore lists would drift, which the
core would then report as a standing finding forever (decision 39).

We rewrite only the entries of areas we manage. `models` and anybody else's
collections are read and written back untouched. Comments and formatting are
not preserved: yaml round-trips data, not layout. Hence the backup.
"""

import os
from dataclasses import dataclass
from pathlib import Path

import yaml

_BACKUP_SUFFIX = ".brain-backup"


@dataclass(frozen=True, slots=True)
class CollectionSpec:
    path: Path
    pattern: str
    ignore: tuple[str, ...]

    def as_entry(self) -> dict[str, object]:
        return {"path": str(self.path), "pattern": self.pattern, "ignore": list(self.ignore)}


def qmd_config_path() -> Path:
    """Where qmd keeps it. XDG on every platform — qmd does not use APPDATA."""
    base = os.environ.get("XDG_CONFIG_HOME")
    return (Path(base) if base else Path.home() / ".config") / "qmd" / "index.yml"


def sync_collections(config: Path, wanted: dict[str, CollectionSpec]) -> tuple[str, ...]:
    """Bring our collections in line. Returns the names actually changed."""
    raw = config.read_text(encoding="utf-8") if config.exists() else ""
    loaded = yaml.safe_load(raw) if raw.strip() else {}
    document: dict[str, object] = loaded if isinstance(loaded, dict) else {}
    collections = document.get("collections")
    if not isinstance(collections, dict):
        collections = {}

    changed = tuple(
        sorted(name for name, spec in wanted.items() if collections.get(name) != spec.as_entry())
    )
    if not changed:
        return ()

    if raw:
        config.with_name(config.name + _BACKUP_SUFFIX).write_text(raw, encoding="utf-8")
    for name in changed:
        collections[name] = wanted[name].as_entry()
    document["collections"] = collections
    config.parent.mkdir(parents=True, exist_ok=True)
    config.write_text(
        yaml.safe_dump(document, allow_unicode=True, sort_keys=True),
        encoding="utf-8",
        newline="",
    )
    return changed
```

- [ ] **Schritt 4: `cli.reindex` anbinden**

Nach dem eigenen Indexlauf, vor `refresh`:

```python
    specs = {
        area.scope: CollectionSpec(
            path=area.path,
            # qmd knows exactly one pattern per collection. A manifest with
            # several include globs cannot be expressed; status says so rather
            # than letting the two indexes quietly select different files.
            pattern=manifest.include[0] if manifest.include else "**/*.md",
            ignore=manifest.exclude + DEFAULT_EXCLUDES,
        )
        for area, manifest in indexed
    }
    changed = sync_collections(qmd_config_path(), specs)
    if changed:
        print(f"updated qmd collections: {', '.join(changed)}")
```

und danach `port.refresh(tuple(sorted(specs)))`. Ein `SearchUnavailable` beendet den Lauf **nicht**: Der eigene Index steht dann bereits, und ein fehlendes qmd darf `catalog` und `read` nicht mitreißen (§7.5). Die Meldung geht nach stderr, der Rückgabewert wird 1.

- [ ] **Schritt 5: Der CLI-Test**

```python
def test_reindex_drives_both_indexes(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """One command, two indexes — and never the real config file."""
    port = FakePort(results=[])
    monkeypatch.setattr("brain.cli._port", lambda: port)
    monkeypatch.setattr("brain.cli.qmd_config_path", lambda: tmp_path / "index.yml")
    state = _state_with_one_readonly_area(tmp_path)
    assert main(["reindex", "--state-dir", str(state)]) == 0
    assert port.refreshed == [("corpus",)]
    assert (tmp_path / "index.yml").exists()
```

`_state_with_one_readonly_area` ist der Aufbau aus Aufgabe 3, Schritt 1; er gehört nach `tests/conftest.py`, sobald ihn der zweite Test braucht.

- [ ] **Schritt 6: Tests laufen lassen**

Run: `uv run pytest -q && uv run ruff check && uv run mypy && uv run coverage run -m pytest && uv run coverage report`
Expected: PASS, Coverage 100 %

- [ ] **Schritt 7: Commit**

```bash
git add src/brain/search/qmd_config.py src/brain/cli.py tests/
git commit -m "Drive both indexes from one exclude list"
```

---

## Aufgabe 13: Der Vertragstest gegen das echte qmd

Ohne diese Aufgabe prüfen alle vorherigen Tests eine Fiktion: Die Attrappe antwortet so, wie wir glauben, dass qmd antwortet.

**Files:**
- Create: `tests/test_qmd_contract.py`
- Modify: `pyproject.toml` (Markierung `contract`, aus der Vorgaberunde ausgeschlossen)

- [ ] **Schritt 1: Den Test schreiben**

```python
"""Holds the fake against the real thing. Not part of the normal test run.

Run with: uv run pytest -m contract
This is the only test in slice 2a that starts a process and loads a model.
"""

import shutil

import pytest
import yaml

from brain.search.port import Profile
from brain.search.qmd import QmdPort
from brain.search.qmd_config import qmd_config_path

pytestmark = [
    pytest.mark.contract,
    pytest.mark.skipif(shutil.which("qmd") is None, reason="qmd is not on PATH"),
]


def test_keyword_search_answers_in_the_shape_we_parse() -> None:
    hits = QmdPort().search("Bahnmechanik", ("space",), Profile.KEYWORD, 3)
    assert hits, "the corpus contains this word; an empty answer means the contract moved"
    first = hits[0]
    assert first.collection == "space"
    assert first.relative.endswith(".md")
    assert first.content_key.startswith("#")
    assert 0.0 <= first.score <= 1.0
    assert first.line > 0


def test_a_query_without_matches_is_empty_and_not_an_error() -> None:
    assert QmdPort().search("zzqqxxwwvv", ("space",), Profile.KEYWORD, 3) == ()


def test_diagnostics_do_not_end_up_on_stdout() -> None:
    """vsearch prints its expansion to stderr; if that ever moves, we misparse."""
    assert QmdPort().search("Bahnmechanik", ("space",), Profile.FAST, 1) is not None


def test_the_config_still_has_the_shape_we_write() -> None:
    """We edit this file (task 12). A format change must fail here, not silently."""
    config = qmd_config_path()
    if not config.exists():
        pytest.skip("no qmd config on this machine")
    loaded = yaml.safe_load(config.read_text(encoding="utf-8"))
    assert isinstance(loaded.get("collections"), dict)
    for entry in loaded["collections"].values():
        assert {"path", "pattern"} <= set(entry)
    assert "models" in loaded, "the block we must not clobber is gone or renamed"
```

- [ ] **Schritt 2: Markierung eintragen**

In `pyproject.toml`:

```toml
[tool.pytest.ini_options]
markers = ["contract: runs against the real qmd; excluded from the default run"]
addopts = "-m 'not contract'"
```

- [ ] **Schritt 3: Beide Läufe ausführen**

Run: `uv run pytest -q` → die Vertragstests werden übersprungen
Run: `uv run pytest -m contract -q` → sie laufen gegen das echte qmd
Expected: beide grün

- [ ] **Schritt 4: Commit**

```bash
git add tests/test_qmd_contract.py pyproject.toml
git commit -m "Hold the fake against the real qmd in a separate run"
```

---

## Aufgabe 14: Die drei Bestände registrieren und das Fertig-Kriterium abnehmen

**Files:**
- Create: `docs/.superpowers/plans/2026-08-20-scheibe-2a-abnahme.md` (das Protokoll dieses Laufs)

Kein Code. Diese Aufgabe führt das System zum ersten Mal gegen den echten Bestand aus.

- [ ] **Schritt 1: Die drei Bestände registrieren**

In `%LOCALAPPDATA%/brain/registry.toml` je einen Eintrag mit `readonly = true` und den Namen, die qmds Sammlungen bereits tragen — `space`, `iam-wiki`, `obsidian-ai`. Die Namen müssen übereinstimmen, sonst legt `ensure_collection` Sammlungen doppelt an.

Je Bereich ein Manifest unter `%LOCALAPPDATA%/brain/areas/<scope>/.brain.toml` mit `include = ["**/*.md"]`. **In keinen der drei Bäume wird geschrieben** — das ist die Bestandsregel, und Aufgabe 3 hat sie zu einer Eigenschaft des Bereichs gemacht.

- [ ] **Schritt 2: Indizieren und die Zahlen festhalten**

```bash
uv run brain reindex
```

Erwartet: die drei Bäume bleiben unverändert (`git status` in jedem der drei ist sauber), die Artefakte stehen unter `areas/<scope>/`.

- [ ] **Schritt 3: Das Fertig-Kriterium Punkt für Punkt abnehmen**

Jeder Punkt wird ausgeführt und die Ausgabe ins Protokoll übernommen:

1. Einen Bereich versuchsweise auf `mode = "local_only"` setzen. `brain search "…" --channel cloud` liefert daraus nichts, `brain catalog --channel cloud` nennt ihn nicht, `brain read` darauf verweigert ausdrücklich. Mit `--channel local` liefert dasselbe alles. Danach zurückstellen.
2. Einen `never`-Pfad setzen und prüfen, dass er auf **beiden** Kanälen weder in der Trefferliste noch über `read` erscheint.
3. Eine Datei suchen, die unter zwei Pfaden liegt — im `space`-Bestand gibt es solche —, und prüfen, dass sie einmal erscheint, der zweite Pfad als `also at`.
4. `qmd` vorübergehend aus dem Pfad nehmen: `brain search` meldet den Ausfall und gibt 1 zurück, `brain catalog` und `brain read` arbeiten weiter.
5. Eine Suche ohne Treffer: `no matches`, Rückgabewert 0. Der Unterschied zu Punkt 4 muss an der Ausgabe erkennbar sein.
6. Eine Frage mit bekannter Zielquelle stellen und prüfen, dass sie unter den ersten drei liegt.

- [ ] **Schritt 4: Das Protokoll schreiben und committen**

Die Zahlen und die Ausgaben, dazu jeder Befund, den `status` meldet. Ein Befund ist kein Fehlschlag der Abnahme — aber er gehört ins Protokoll, weil er der Ausgangspunkt von 2b ist.

```bash
git add docs/.superpowers/plans/2026-08-20-scheibe-2a-abnahme.md
git commit -m "Record the acceptance run of slice 2a"
```

---

## Bekannte Einschränkungen

Punkte, die 2a bewusst offen lässt.

**`reindex` ist nicht mehr netz- und modellfrei.** Entscheidung 39 nimmt das in Kauf. Die Folge für Scheibe 1 ist benannt: „zwei Läufe, byteweise identisch" gilt ab jetzt für den eigenen Teil der Ausgabe. Ob `qmd update` bei jedem Lauf mitlaufen soll oder nur auf Verlangen, ist eine Frage der Kosten — und die misst 2b.

**Die Wiederholung bei leerem Ergebnis kostet im Regelfall doppelte Zeit.** Eine echte Suche ohne Treffer wird immer zweimal ausgeführt. Bei `fast` und `full` sind das im warmen Zustand Millisekunden, im kalten Sekunden. Ob das tragbar ist, entscheidet 2b anhand der Latenzzahlen.

**`n * 4` bei der Anfrage an die Suchmaschine ist geraten.** Der Faktor soll verhindern, dass die Entdopplung eine kürzere Liste zurückgibt als angefordert. Wie viele Wiederholungen im echten Bestand tatsächlich auftreten, weiß erst 2b — und dort gehört der Faktor an eine Zahl gebunden statt an eine Vermutung.

**`brain` schreibt die Konfigurationsdatei eines fremden Werkzeugs.** Das ist der Preis von Entscheidung 39, und er ist echt: Ändert qmd das Format von `index.yml`, schreiben wir Unsinn hinein. Drei Riegel dagegen — eine Sicherungskopie vor dem ersten Schreiben, ein Vertragstest, der das Format prüft (Aufgabe 13), und die Beschränkung, dass nur die Einträge der registrierten Bereiche angefasst werden. Beseitigt ist das Risiko damit nicht.

**qmd kennt je Sammlung genau ein Einschlussmuster.** Ein Manifest mit mehreren `include`-Globs lässt sich qmd nicht mitteilen; der Kern nimmt das erste und meldet den Rest als Befund. Für die drei heutigen Bestände ist das folgenlos — alle drei benutzen `**/*.md` —, für ein Projekt-Repo mit der Vorgabe aus §5.5 (README, CHANGELOG, ADRs, Spec, Plan) wäre es das nicht. Die Scheibe, die das erste Code-Repo indexiert, muss das lösen.

**`brain status` startet einen Prozess je Bereich.** Die Meldung über die Dokumente, die die Suchmaschine nicht kennt, braucht deren Dateiliste, und die kommt aus `qmd ls <sammlung>` — ein Aufruf je registriertem Bereich. Bei vier Sammlungen ist `status` damit spürbar langsamer als vorher; gemessen ist es nicht. Ein Daemon, der die Verbindung hält (§13), nimmt den Aufwand weg, ohne dass sich oberhalb der Naht etwas ändert — bis dahin ist es der Preis dafür, dass die Divergenz überhaupt sichtbar wird.

**Der Kanal ist auf der CLI frei wählbar.** Das ist für den Nachweis nötig und in 2a richtig. Sobald 2c den MCP-Adapter baut, ist zu prüfen, ob das Flag auf der CLI bleiben soll: Es erlaubt einem Menschen, sich freiwillig einzuschränken, aber es erlaubt niemandem, ein Gatter zu öffnen — `--channel local` auf der CLI ist ohnehin der Vorgabezustand.

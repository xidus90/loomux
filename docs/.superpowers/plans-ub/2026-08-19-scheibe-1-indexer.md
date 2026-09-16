# Scheibe 1 — Indexer · Implementierungsplan

> **Für ausführende Agenten:** ERFORDERLICHER SUB-SKILL: `superpowers:subagent-driven-development` (empfohlen) oder `superpowers:executing-plans`, um diesen Plan Aufgabe für Aufgabe abzuarbeiten. Schritte nutzen Checkbox-Syntax (`- [ ]`).

**Ziel:** Ein deterministisches Programm, das durch die registrierten Bereiche läuft und daraus `index.md` je Ebene, `graph.json` und das Identitätsregister `_identities.tsv` erzeugt — bei unverändertem Bestand byteweise identisch.

**Vorgehen:** Kein Modell, kein Daemon, kein Netz. Reine Textverarbeitung: Dateien finden, Frontmatter und Links lesen, Kennungen vergeben, sortiert ausgeben. Acht Aufgaben, jede mit eigenem Testzyklus.

**Werkzeuge:** Python ≥ 3.14 (verfügbar: 3.14.7 über `uv`), `uv` für alles, `pytest`, `ruff`, `mypy`, `coverage`.

**Spec:** `docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md`

## Globale Rahmenbedingungen

Aus der Spec und den globalen Regeln, wörtlich übernommen. Sie gelten für **jede** Aufgabe:

- **Python ≥ 3.14 überall explizit:** `requires-python = ">=3.14"`, ruff `target-version = "py314"`, mypy `python_version = "3.14"`.
- **`uv` für alles.** Kein `pip`, kein `python -m venv`, keine `requirements.txt`. Aufrufe über `uv run`.
- **TDD.** Erst der fehlschlagende Test, dann die Implementierung.
- **100 % Coverage, gemessen.** Ausschlüsse nur mit begründendem Kommentar: `# pragma: no cover  # <Grund>`, nie nackt.
- **Typisierung vollständig.** Kein `Any`, kein `# type: ignore` ohne begründenden Kommentar. `mypy --strict` läuft ohne Fehler.
- **Imports stehen oben**, auf Modulebene. Ein lokaler Import braucht einen Kommentar mit dem Grund.
- **Kein Modell, kein Netz, keine Nebenläufigkeit** in dieser Scheibe (§4.2: der Indexer ist Code, nicht KI).
- **Sprachen:** Quellcode, Bezeichner, Code-Kommentare, Commit-Nachrichten und Log-/Fehlermeldungen **englisch**; Prosa in Dokumenten deutsch.
- **Reproduzierbarkeit ist die Hauptanforderung** (§14): gleiche Eingabe → byteweise gleiche Ausgabe. Stabile Sortierung, keine Zeitstempel im Inhalt, keine Mengen mit undefinierter Reihenfolge.
- **Zeilenenden immer LF** beim Schreiben, unabhängig von der Plattform (§5.6).
- **Pfade in Ausgaben immer mit Vorwärts-Schrägstrichen** und relativ zur Bereichswurzel — sonst ist die Ausgabe plattformabhängig.

### Was in dieser Scheibe ausdrücklich nicht gebaut wird

Daemon, MCP, Suche, Wiki-Schicht, Konverter, Wartung, Web-App. Der Indexer ist ein Programm, das man aufruft und das danach beendet ist.

### Die Reproduzierbarkeitsfalle, die man kennen muss

`doc_id` ist eine **ULID und enthält einen Zeitstempel** — zwei Vergaben derselben Datei ergäben verschiedene Kennungen. Das Fertig-Kriterium „zwei Läufe, byteweise identisch" hält trotzdem, aber **nur** weil der zweite Lauf das Register `_identities.tsv` *liest* und Kennungen ausschließlich für **neue** Dateien vergibt. Wer die Kennung bei jedem Lauf neu erzeugt, bricht die Hauptanforderung dieser Scheibe.

---

## Dateistruktur

```
pyproject.toml               Projekt, Abhängigkeiten, ruff/mypy/coverage
src/brain/
  __init__.py
  models.py                  Datenklassen: Area, Manifest, Document, Identity
  manifest.py                .brain.toml lesen
  registry.py                Registrierung lesen (Bereich -> Pfad)
  walk.py                    Dateien finden, Ausschlüsse anwenden
  document.py                Frontmatter und Links aus einer Markdown-Datei
  identity.py                ULID, content_hash, _identities.tsv
  catalog.py                 index.md je Ebene rendern
  graph.py                   graph.json rendern
  writer.py                  Dateien schreiben: LF, UTF-8, nur bei Änderung
  cli.py                     Einstiegspunkt `brain`
tests/
  conftest.py                Fixtures: synthetische Bereichsbäume
  test_manifest.py  test_registry.py  test_walk.py  test_document.py
  test_identity.py  test_catalog.py   test_graph.py  test_writer.py
  test_cli.py                Ende-zu-Ende inklusive Reproduzierbarkeit
```

Jede Datei hat eine Aufgabe. `models.py` trägt nur Datenklassen, damit die übrigen Module sich nicht gegenseitig importieren müssen.

---

## Aufgabe 1: Projektgerüst und Werkzeugkette

Ohne beweisbar laufende Werkzeugkette ist jede folgende Aufgabe blind.

**Dateien:**
- Anlegen: `pyproject.toml`, `src/brain/__init__.py`, `tests/test_smoke.py`

**Liefert:** ein Projekt, in dem `uv run pytest`, `uv run ruff check`, `uv run mypy` und die Coverage-Schwelle durchlaufen.

- [ ] **Schritt 1: `pyproject.toml` anlegen**

```toml
[project]
name = "brain"
version = "0.1.0"
description = "Deterministic indexer for a local knowledge system"
requires-python = ">=3.14"
dependencies = []

[project.scripts]
brain = "brain.cli:main"

[dependency-groups]
dev = ["pytest>=8", "ruff>=0.6", "mypy>=1.11", "coverage[toml]>=7"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["src/brain"]

[tool.ruff]
target-version = "py314"
line-length = 100
src = ["src", "tests"]

[tool.ruff.lint]
select = ["E", "F", "I", "N", "UP", "B", "SIM", "RUF"]

[tool.mypy]
python_version = "3.14"
strict = true
files = ["src", "tests"]

[tool.coverage.run]
branch = true
source = ["src/brain"]

[tool.coverage.report]
fail_under = 100
show_missing = true
```

- [ ] **Schritt 2: Paket und fehlschlagenden Test anlegen**

`src/brain/__init__.py`:

```python
"""Deterministic indexer for a local knowledge system."""

__version__ = "0.1.0"
```

`tests/test_smoke.py`:

```python
from brain import __version__


def test_version_is_exposed() -> None:
    assert __version__ == "0.1.0"
```

- [ ] **Schritt 3: Werkzeugkette laufen lassen**

```bash
uv sync
uv run pytest -q
uv run ruff check .
uv run mypy
uv run coverage run -m pytest -q && uv run coverage report
```

Erwartung: alle vier grün, Coverage 100 %. Schlägt `uv sync` fehl, weil 3.14 fehlt: `uv python install 3.14`.

- [ ] **Schritt 4: `.gitignore` ergänzen**

An `.gitignore` anhängen:

```gitignore
# Python
__pycache__/
*.py[cod]
.venv/
.pytest_cache/
.mypy_cache/
.ruff_cache/
.coverage
```

- [ ] **Schritt 5: Commit**

```bash
git add pyproject.toml uv.lock src tests .gitignore
git commit -m "Add project scaffolding with strict typing and full coverage gate"
```

---

## Aufgabe 2: Datenklassen, Manifest und Registrierung

**Dateien:**
- Anlegen: `src/brain/models.py`, `src/brain/manifest.py`, `src/brain/registry.py`
- Anlegen: `tests/test_manifest.py`, `tests/test_registry.py`, `tests/conftest.py`

**Produziert für spätere Aufgaben:**
- `Manifest(scope: str, wiki: bool, layout: dict[str, str], include: tuple[str, ...], exclude: tuple[str, ...])`
- `Area(scope: str, path: Path, wiki_path: Path | None)`
- `read_manifest(path: Path) -> Manifest`
- `read_registry(path: Path) -> tuple[Area, ...]`

- [ ] **Schritt 1: Fehlschlagende Tests schreiben**

`tests/conftest.py`:

```python
from pathlib import Path

import pytest


@pytest.fixture
def area_tree(tmp_path: Path) -> Path:
    """A minimal area: manifest plus two notes, one of them excluded."""
    (tmp_path / ".brain.toml").write_text(
        '[area]\nscope = "knowledge"\nwiki = true\n\n'
        '[layout]\nsources = "10 Rohquellen"\n\n'
        '[index]\ninclude = ["10 Rohquellen/**/*.md"]\nexclude = ["**/.claude/**"]\n',
        encoding="utf-8",
    )
    notes = tmp_path / "10 Rohquellen"
    notes.mkdir()
    (notes / "alpha.md").write_text("---\ntitle: Alpha\n---\n\nText\n", encoding="utf-8")
    hidden = notes / ".claude"
    hidden.mkdir()
    (hidden / "copy.md").write_text("---\ntitle: Alpha\n---\n\nText\n", encoding="utf-8")
    return tmp_path
```

`tests/test_manifest.py`:

```python
from pathlib import Path

import pytest

from brain.manifest import ManifestError, read_manifest


def test_reads_scope_and_globs(area_tree: Path) -> None:
    manifest = read_manifest(area_tree / ".brain.toml")
    assert manifest.scope == "knowledge"
    assert manifest.wiki is True
    assert manifest.include == ("10 Rohquellen/**/*.md",)
    assert manifest.exclude == ("**/.claude/**",)
    assert manifest.layout["sources"] == "10 Rohquellen"


def test_missing_scope_is_an_error(tmp_path: Path) -> None:
    bad = tmp_path / ".brain.toml"
    bad.write_text("[area]\nwiki = true\n", encoding="utf-8")
    with pytest.raises(ManifestError, match="scope"):
        read_manifest(bad)


def test_defaults_when_sections_absent(tmp_path: Path) -> None:
    bare = tmp_path / ".brain.toml"
    bare.write_text('[area]\nscope = "knowledge"\n', encoding="utf-8")
    manifest = read_manifest(bare)
    assert manifest.wiki is False
    assert manifest.include == ()
    assert manifest.exclude == ()
    assert manifest.layout == {}
```

`tests/test_registry.py`:

```python
from pathlib import Path

import pytest

from brain.registry import RegistryError, read_registry


def test_reads_areas_in_declared_order(tmp_path: Path) -> None:
    registry = tmp_path / "registry.toml"
    registry.write_text(
        '[[area]]\nscope = "knowledge"\npath = "C:/vault"\n\n'
        '[[area]]\nscope = "project/x"\npath = "C:/repo"\nwiki = "C:/vault/91/x"\n',
        encoding="utf-8",
    )
    areas = read_registry(registry)
    assert [a.scope for a in areas] == ["knowledge", "project/x"]
    assert areas[0].wiki_path is None
    assert areas[1].wiki_path == Path("C:/vault/91/x")


def test_duplicate_scope_is_an_error(tmp_path: Path) -> None:
    registry = tmp_path / "registry.toml"
    registry.write_text(
        '[[area]]\nscope = "knowledge"\npath = "C:/a"\n\n'
        '[[area]]\nscope = "knowledge"\npath = "C:/b"\n',
        encoding="utf-8",
    )
    with pytest.raises(RegistryError, match="duplicate scope"):
        read_registry(registry)
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

```bash
uv run pytest tests/test_manifest.py tests/test_registry.py -q
```

Erwartung: FAIL — `ModuleNotFoundError: No module named 'brain.manifest'`.

- [ ] **Schritt 3: Implementieren**

`src/brain/models.py`:

```python
"""Data carried between the indexer's stages."""

from dataclasses import dataclass, field
from pathlib import Path


@dataclass(frozen=True, slots=True)
class Manifest:
    """What an area declares about itself."""

    scope: str
    wiki: bool = False
    layout: dict[str, str] = field(default_factory=dict)
    include: tuple[str, ...] = ()
    exclude: tuple[str, ...] = ()


@dataclass(frozen=True, slots=True)
class Area:
    """A registered area: where its sources live, and where its wiki lives."""

    scope: str
    path: Path
    wiki_path: Path | None = None
```

`src/brain/manifest.py`:

```python
"""Reading `.brain.toml`, the file in which an area declares itself."""

import tomllib
from pathlib import Path

from brain.models import Manifest


class ManifestError(Exception):
    """The manifest is unusable and the run must not continue on a guess."""


def read_manifest(path: Path) -> Manifest:
    data = tomllib.loads(path.read_text(encoding="utf-8"))
    area = data.get("area", {})
    scope = area.get("scope")
    if not isinstance(scope, str) or not scope:
        raise ManifestError(f"{path}: [area] scope is required and must be a non-empty string")
    index = data.get("index", {})
    return Manifest(
        scope=scope,
        wiki=bool(area.get("wiki", False)),
        layout=dict(data.get("layout", {})),
        include=tuple(index.get("include", ())),
        exclude=tuple(index.get("exclude", ())),
    )
```

`src/brain/registry.py`:

```python
"""Reading the registry: the single place mapping an area name to a path."""

import tomllib
from pathlib import Path

from brain.models import Area


class RegistryError(Exception):
    """The registry is unusable and the run must not continue on a guess."""


def read_registry(path: Path) -> tuple[Area, ...]:
    data = tomllib.loads(path.read_text(encoding="utf-8"))
    areas: list[Area] = []
    seen: set[str] = set()
    for entry in data.get("area", []):
        scope = entry["scope"]
        if scope in seen:
            raise RegistryError(f"{path}: duplicate scope {scope!r}")
        seen.add(scope)
        wiki = entry.get("wiki")
        areas.append(
            Area(
                scope=scope,
                path=Path(entry["path"]),
                wiki_path=Path(wiki) if wiki else None,
            )
        )
    return tuple(areas)
```

- [ ] **Schritt 4: Tests laufen lassen**

```bash
uv run pytest tests/test_manifest.py tests/test_registry.py -q
uv run ruff check . && uv run mypy
```

Erwartung: alle PASS, Linter und Typechecker sauber.

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/models.py src/brain/manifest.py src/brain/registry.py tests/
git commit -m "Read area manifests and the registry"
```

---

## Aufgabe 3: Dateien finden, Ausschlüsse anwenden

Hier steckt der Befund aus Scheibe 0: Ohne Ausschlüsse misst und katalogisiert das System Kopien statt Wissen (§16.12).

**Dateien:**
- Anlegen: `src/brain/walk.py`, `tests/test_walk.py`

**Verbraucht:** `Manifest`, `Area` aus Aufgabe 2.
**Produziert:** `find_files(area: Area, manifest: Manifest, nested: tuple[Path, ...] = ()) -> tuple[Path, ...]` — **sortierte**, absolute Pfade.

- [ ] **Schritt 1: Fehlschlagende Tests schreiben**

`tests/test_walk.py`:

```python
from pathlib import Path

from brain.manifest import read_manifest
from brain.models import Area
from brain.walk import DEFAULT_EXCLUDES, find_files


def test_applies_include_and_exclude(area_tree: Path) -> None:
    manifest = read_manifest(area_tree / ".brain.toml")
    found = find_files(Area(scope="knowledge", path=area_tree), manifest)
    assert [p.name for p in found] == ["alpha.md"]


def test_result_is_sorted(tmp_path: Path) -> None:
    (tmp_path / ".brain.toml").write_text(
        '[area]\nscope = "k"\n\n[index]\ninclude = ["**/*.md"]\n', encoding="utf-8"
    )
    for name in ("gamma.md", "alpha.md", "beta.md"):
        (tmp_path / name).write_text("x\n", encoding="utf-8")
    manifest = read_manifest(tmp_path / ".brain.toml")
    found = find_files(Area(scope="k", path=tmp_path), manifest)
    assert [p.name for p in found] == ["alpha.md", "beta.md", "gamma.md"]


def test_default_excludes_apply_without_being_declared(tmp_path: Path) -> None:
    (tmp_path / ".brain.toml").write_text(
        '[area]\nscope = "k"\n\n[index]\ninclude = ["**/*.md"]\n', encoding="utf-8"
    )
    for folder in (".git", ".obsidian", ".claude", ".tools"):
        sub = tmp_path / folder
        sub.mkdir()
        (sub / "noise.md").write_text("x\n", encoding="utf-8")
    (tmp_path / "real.md").write_text("x\n", encoding="utf-8")
    manifest = read_manifest(tmp_path / ".brain.toml")
    found = find_files(Area(scope="k", path=tmp_path), manifest)
    assert [p.name for p in found] == ["real.md"]
    assert "**/.claude/**" in DEFAULT_EXCLUDES


def test_nested_areas_are_excluded_from_the_parent(tmp_path: Path) -> None:
    (tmp_path / ".brain.toml").write_text(
        '[area]\nscope = "k"\n\n[index]\ninclude = ["**/*.md"]\n', encoding="utf-8"
    )
    (tmp_path / "own.md").write_text("x\n", encoding="utf-8")
    child = tmp_path / "91 Projekte" / "x"
    child.mkdir(parents=True)
    (child / "foreign.md").write_text("x\n", encoding="utf-8")
    manifest = read_manifest(tmp_path / ".brain.toml")
    found = find_files(Area(scope="k", path=tmp_path), manifest, nested=(child,))
    assert [p.name for p in found] == ["own.md"]
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

```bash
uv run pytest tests/test_walk.py -q
```

Erwartung: FAIL — `No module named 'brain.walk'`.

- [ ] **Schritt 3: Implementieren**

`src/brain/walk.py`:

```python
"""Finding the files of an area: includes, excludes, nested areas."""

from pathlib import Path, PurePosixPath

from brain.models import Area, Manifest

DEFAULT_EXCLUDES: tuple[str, ...] = (
    "**/.git/**",
    "**/.obsidian/**",
    # Worktree copies: byte-near twins of real notes, measured in slice 0 (spec 16.12).
    "**/.claude/**",
    # Licence texts and tool ballast, not knowledge.
    "**/.tools/**",
    "**/.superpowers/**",
    "**/node_modules/**",
    "**/tests/fixtures/**",
)


def _matches_any(relative: PurePosixPath, patterns: tuple[str, ...]) -> bool:
    return any(relative.full_match(pattern) for pattern in patterns)


def find_files(
    area: Area,
    manifest: Manifest,
    nested: tuple[Path, ...] = (),
) -> tuple[Path, ...]:
    """Return the area's files, sorted, with every exclusion applied.

    Sorting is not cosmetic: the whole output of the indexer must be
    reproducible byte for byte, and an unordered filesystem walk would
    make it depend on the platform.
    """
    excludes = manifest.exclude + DEFAULT_EXCLUDES
    nested_resolved = tuple(child.resolve() for child in nested)
    found: list[Path] = []
    for candidate in area.path.rglob("*"):
        if not candidate.is_file():
            continue
        resolved = candidate.resolve()
        if any(resolved.is_relative_to(child) for child in nested_resolved):
            continue
        relative = PurePosixPath(candidate.relative_to(area.path).as_posix())
        if not _matches_any(relative, manifest.include):
            continue
        if _matches_any(relative, excludes):
            continue
        found.append(candidate)
    return tuple(sorted(found, key=lambda p: p.relative_to(area.path).as_posix()))
```

- [ ] **Schritt 4: Tests laufen lassen**

```bash
uv run pytest tests/test_walk.py -q
uv run ruff check . && uv run mypy
```

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/walk.py tests/test_walk.py
git commit -m "Find area files with exclusions for copies and tool directories"
```

---

## Aufgabe 4: Ein Dokument lesen

**Dateien:**
- Anlegen: `src/brain/document.py`, `tests/test_document.py`
- Ändern: `src/brain/models.py` — `Document` ergänzen
- Ändern: `pyproject.toml` — Abhängigkeit `pyyaml` und `types-pyyaml`

**Produziert:** `read_document(path: Path, root: Path) -> Document` mit
`Document(relative: str, title: str, description: str, tags: tuple[str, ...], doc_type: str | None, links: tuple[str, ...])`.

- [ ] **Schritt 1: Fehlschlagende Tests schreiben**

`tests/test_document.py`:

```python
from pathlib import Path

from brain.document import read_document


def _write(tmp_path: Path, name: str, text: str) -> Path:
    target = tmp_path / name
    target.write_text(text, encoding="utf-8")
    return target


def test_reads_frontmatter(tmp_path: Path) -> None:
    path = _write(
        tmp_path,
        "note.md",
        "---\ntitle: Alpha\ndescription: One line\ntags: [a, b]\ntype: Topic\n---\n\nBody\n",
    )
    doc = read_document(path, tmp_path)
    assert doc.title == "Alpha"
    assert doc.description == "One line"
    assert doc.tags == ("a", "b")
    assert doc.doc_type == "Topic"
    assert doc.relative == "note.md"


def test_title_falls_back_to_filename(tmp_path: Path) -> None:
    path = _write(tmp_path, "no-frontmatter.md", "Just text\n")
    doc = read_document(path, tmp_path)
    assert doc.title == "no-frontmatter"
    assert doc.description == ""
    assert doc.tags == ()
    assert doc.doc_type is None


def test_collects_markdown_and_wiki_links_sorted_without_duplicates(tmp_path: Path) -> None:
    path = _write(
        tmp_path,
        "links.md",
        "See [beta](./beta.md) and [[gamma]] and [beta again](./beta.md) and [[alpha|Alias]].\n",
    )
    doc = read_document(path, tmp_path)
    assert doc.links == ("./beta.md", "alpha", "gamma")


def test_broken_frontmatter_is_tolerated(tmp_path: Path) -> None:
    path = _write(tmp_path, "broken.md", "---\ntitle: [unclosed\n---\n\nBody\n")
    doc = read_document(path, tmp_path)
    assert doc.title == "broken"
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

```bash
uv run pytest tests/test_document.py -q
```

Erwartung: FAIL — `No module named 'brain.document'`.

- [ ] **Schritt 3: Abhängigkeit ergänzen**

```bash
uv add pyyaml
uv add --dev types-pyyaml
```

- [ ] **Schritt 4: Implementieren**

In `src/brain/models.py` ergänzen:

```python
@dataclass(frozen=True, slots=True)
class Document:
    """One markdown file, reduced to what the catalog and the graph need."""

    relative: str
    title: str
    description: str
    tags: tuple[str, ...]
    doc_type: str | None
    links: tuple[str, ...]
```

`src/brain/document.py`:

```python
"""Reducing a markdown file to the few facts the indexer needs."""

import re
from pathlib import Path
from typing import Any

import yaml

from brain.models import Document

_FRONTMATTER = re.compile(r"\A---\r?\n(.*?)\r?\n---\r?\n", re.DOTALL)
_MARKDOWN_LINK = re.compile(r"(?<!!)\[[^\]]*\]\(([^)\s]+)\)")
_WIKI_LINK = re.compile(r"\[\[([^\]|#]+)")


def _parse_frontmatter(text: str) -> dict[str, Any]:
    match = _FRONTMATTER.match(text)
    if match is None:
        return {}
    try:
        loaded = yaml.safe_load(match.group(1))
    except yaml.YAMLError:
        # Broken frontmatter must not stop an index run: the file still exists
        # and still belongs in the catalog, just without its metadata.
        return {}
    return loaded if isinstance(loaded, dict) else {}


def _as_str(value: object) -> str:
    return value if isinstance(value, str) else ""


def read_document(path: Path, root: Path) -> Document:
    text = path.read_text(encoding="utf-8", errors="replace")
    meta = _parse_frontmatter(text)
    raw_tags = meta.get("tags", [])
    tags = tuple(str(tag) for tag in raw_tags) if isinstance(raw_tags, list) else ()
    links = sorted(
        {*(m.group(1) for m in _MARKDOWN_LINK.finditer(text)),
         *(m.group(1).strip() for m in _WIKI_LINK.finditer(text))}
    )
    doc_type = meta.get("type")
    return Document(
        relative=path.relative_to(root).as_posix(),
        title=_as_str(meta.get("title")) or path.stem,
        description=_as_str(meta.get("description")),
        tags=tags,
        doc_type=doc_type if isinstance(doc_type, str) else None,
        links=tuple(links),
    )
```

- [ ] **Schritt 5: Tests laufen lassen**

```bash
uv run pytest tests/test_document.py -q
uv run ruff check . && uv run mypy
```

- [ ] **Schritt 6: Commit**

```bash
git add pyproject.toml uv.lock src/brain/document.py src/brain/models.py tests/test_document.py
git commit -m "Read frontmatter and links from a markdown document"
```

---

## Aufgabe 5: Identität — Kennung, Hash, Register

Die Grundlage der späteren Wartungsschicht (§5.6). Zwei Eigenschaften sind nicht verhandelbar: Der Hash läuft über **LF-normalisierten** Inhalt, und Kennungen werden nur für **neue** Dateien vergeben.

**Dateien:**
- Anlegen: `src/brain/identity.py`, `tests/test_identity.py`
- Ändern: `src/brain/models.py` — `Identity` ergänzen

**Produziert:**
- `content_hash(path: Path) -> str` → `"sha256:<hex>"`
- `new_doc_id() -> str` → ULID, 26 Zeichen
- `read_identities(path: Path) -> dict[str, Identity]` (Schlüssel: relativer Pfad)
- `render_identities(identities: dict[str, Identity]) -> str`
- `Identity(doc_id: str, relative: str, content_hash: str, revision: int)`

- [ ] **Schritt 1: Fehlschlagende Tests schreiben**

`tests/test_identity.py`:

```python
from pathlib import Path

from brain.identity import content_hash, new_doc_id, read_identities, render_identities
from brain.models import Identity


def test_hash_ignores_line_endings(tmp_path: Path) -> None:
    lf = tmp_path / "lf.md"
    crlf = tmp_path / "crlf.md"
    lf.write_bytes(b"line one\nline two\n")
    crlf.write_bytes(b"line one\r\nline two\r\n")
    assert content_hash(lf) == content_hash(crlf)
    assert content_hash(lf).startswith("sha256:")


def test_hash_changes_with_content(tmp_path: Path) -> None:
    a = tmp_path / "a.md"
    a.write_bytes(b"one\n")
    before = content_hash(a)
    a.write_bytes(b"two\n")
    assert content_hash(a) != before


def test_doc_ids_are_unique_and_of_fixed_length() -> None:
    ids = {new_doc_id() for _ in range(50)}
    assert len(ids) == 50
    assert all(len(value) == 26 for value in ids)


def test_register_round_trips_sorted_by_doc_id(tmp_path: Path) -> None:
    identities = {
        "b.md": Identity(doc_id="01BBBB", relative="b.md", content_hash="sha256:2", revision=3),
        "a.md": Identity(doc_id="01AAAA", relative="a.md", content_hash="sha256:1", revision=1),
    }
    rendered = render_identities(identities)
    assert rendered.splitlines()[0] == "doc_id\tpfad\tcontent_hash\trevision"
    assert rendered.splitlines()[1].startswith("01AAAA")
    target = tmp_path / "_identities.tsv"
    target.write_text(rendered, encoding="utf-8", newline="")
    assert read_identities(target) == identities


def test_missing_register_reads_as_empty(tmp_path: Path) -> None:
    assert read_identities(tmp_path / "absent.tsv") == {}
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

```bash
uv run pytest tests/test_identity.py -q
```

Erwartung: FAIL — `No module named 'brain.identity'`.

- [ ] **Schritt 3: Implementieren**

In `src/brain/models.py` ergänzen:

```python
@dataclass(frozen=True, slots=True)
class Identity:
    """A file's stable identity, kept beside the notes rather than inside them."""

    doc_id: str
    relative: str
    content_hash: str
    revision: int
```

`src/brain/identity.py`:

```python
"""Stable identity for a file: ULID, content hash, and the register holding both."""

import hashlib
import os
import secrets
import time
from pathlib import Path

from brain.models import Identity

_CROCKFORD = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
_HEADER = "doc_id\tpfad\tcontent_hash\trevision"


def content_hash(path: Path) -> str:
    """Hash a file's content with line endings normalised to LF.

    Without normalising, a checkout on another platform would rewrite every
    line ending, change every hash, and make the maintenance layer report
    every page as changed (spec 5.6).
    """
    raw = path.read_bytes().replace(b"\r\n", b"\n")
    return f"sha256:{hashlib.sha256(raw).hexdigest()}"


def _encode(value: int, length: int) -> str:
    out = ""
    for _ in range(length):
        value, remainder = divmod(value, 32)
        out = _CROCKFORD[remainder] + out
    return out


def new_doc_id() -> str:
    """A ULID: 48 bits of milliseconds, 80 bits of randomness.

    Time-ordered so the register sorts chronologically, and independent of any
    coordination so two runs never need to agree on a counter.
    """
    return _encode(int(time.time() * 1000), 10) + _encode(
        int.from_bytes(secrets.token_bytes(10)), 16
    )


def read_identities(path: Path) -> dict[str, Identity]:
    if not path.exists():
        return {}
    identities: dict[str, Identity] = {}
    for line in path.read_text(encoding="utf-8").splitlines()[1:]:
        if not line.strip():
            continue
        doc_id, relative, digest, revision = line.split("\t")
        identities[relative] = Identity(
            doc_id=doc_id,
            relative=relative,
            content_hash=digest,
            revision=int(revision),
        )
    return identities


def render_identities(identities: dict[str, Identity]) -> str:
    """Render sorted by doc_id so the file diffs line by line and never churns."""
    rows = sorted(identities.values(), key=lambda item: item.doc_id)
    lines = [_HEADER]
    lines.extend(f"{r.doc_id}\t{r.relative}\t{r.content_hash}\t{r.revision}" for r in rows)
    return "\n".join(lines) + "\n"
```

Die Importzeile lautet entsprechend nur `import hashlib`, `import secrets`, `import time` — `os` wird nicht gebraucht.

- [ ] **Schritt 4: Tests laufen lassen**

```bash
uv run pytest tests/test_identity.py -q
uv run ruff check . && uv run mypy
```

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/identity.py src/brain/models.py tests/test_identity.py
git commit -m "Assign stable document ids and hash content over normalised line endings"
```

---

## Aufgabe 6: Katalog rendern

Format nach OKF §8: Abschnitte mit Überschrift, darunter Einträge aus Link und Beschreibung.

**Dateien:**
- Anlegen: `src/brain/catalog.py`, `tests/test_catalog.py`

**Produziert:** `render_catalog(title: str, entries: Sequence[Document], subdirectories: Sequence[str]) -> str`

- [ ] **Schritt 1: Fehlschlagenden Test schreiben**

`tests/test_catalog.py`:

```python
from brain.catalog import render_catalog
from brain.models import Document


def _doc(relative: str, title: str, description: str = "") -> Document:
    return Document(
        relative=relative,
        title=title,
        description=description,
        tags=(),
        doc_type=None,
        links=(),
    )


def test_renders_sections_and_entries() -> None:
    rendered = render_catalog(
        title="Katalog",
        entries=[_doc("b.md", "Beta", "Second"), _doc("a.md", "Alpha", "First")],
        subdirectories=["projekte", "archiv"],
    )
    assert rendered == (
        "# Katalog\n"
        "\n"
        "## Bereiche\n"
        "\n"
        "* [archiv](archiv/)\n"
        "* [projekte](projekte/)\n"
        "\n"
        "## Dateien\n"
        "\n"
        "* [Alpha](a.md) - First\n"
        "* [Beta](b.md) - Second\n"
        "\n"
        "> Jeder neue Bereich bekommt sofort eine Zeile in diesem Katalog.\n"
    )


def test_omits_empty_sections() -> None:
    rendered = render_catalog(title="Katalog", entries=[], subdirectories=[])
    assert "## Bereiche" not in rendered
    assert "## Dateien" not in rendered


def test_entry_without_description_has_no_dash() -> None:
    rendered = render_catalog(title="K", entries=[_doc("a.md", "Alpha")], subdirectories=[])
    assert "* [Alpha](a.md)\n" in rendered
```

- [ ] **Schritt 2: Test laufen lassen, Fehlschlag bestätigen**

```bash
uv run pytest tests/test_catalog.py -q
```

Erwartung: FAIL — `No module named 'brain.catalog'`.

- [ ] **Schritt 3: Implementieren**

`src/brain/catalog.py`:

```python
"""Rendering `index.md`, the catalog that is read before anything is searched."""

from collections.abc import Sequence

from brain.models import Document

_RULE = "> Jeder neue Bereich bekommt sofort eine Zeile in diesem Katalog."


def render_catalog(
    title: str,
    entries: Sequence[Document],
    subdirectories: Sequence[str],
) -> str:
    """Render one catalog level. Sorted throughout, so reruns produce the same bytes."""
    parts: list[str] = [f"# {title}", ""]
    if subdirectories:
        parts += ["## Bereiche", ""]
        parts += [f"* [{name}]({name}/)" for name in sorted(subdirectories)]
        parts += [""]
    if entries:
        parts += ["## Dateien", ""]
        for doc in sorted(entries, key=lambda item: item.relative):
            line = f"* [{doc.title}]({doc.relative})"
            parts.append(f"{line} - {doc.description}" if doc.description else line)
        parts += [""]
    parts += [_RULE]
    return "\n".join(parts) + "\n"
```

- [ ] **Schritt 4: Tests laufen lassen**

```bash
uv run pytest tests/test_catalog.py -q
uv run ruff check . && uv run mypy
```

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/catalog.py tests/test_catalog.py
git commit -m "Render the catalog level by level"
```

---

## Aufgabe 7: Graph rendern und Dateien schreiben

**Dateien:**
- Anlegen: `src/brain/graph.py`, `src/brain/writer.py`
- Anlegen: `tests/test_graph.py`, `tests/test_writer.py`

**Produziert:**
- `render_graph(area_scope: str, documents: Sequence[Document]) -> str` — JSON mit Zeilenumbruch am Ende
- `write_if_changed(path: Path, text: str) -> bool` — schreibt LF und UTF-8, gibt zurück, ob geschrieben wurde

- [ ] **Schritt 1: Fehlschlagende Tests schreiben**

`tests/test_graph.py`:

```python
import json

from brain.graph import render_graph
from brain.models import Document


def _doc(relative: str, links: tuple[str, ...] = ()) -> Document:
    return Document(
        relative=relative,
        title=relative.removesuffix(".md"),
        description="",
        tags=(),
        doc_type=None,
        links=links,
    )


def test_nodes_and_edges_are_sorted() -> None:
    rendered = render_graph("knowledge", [_doc("b.md", ("a.md",)), _doc("a.md")])
    data = json.loads(rendered)
    assert [n["id"] for n in data["nodes"]] == ["a.md", "b.md"]
    assert data["edges"] == [{"from": "b.md", "to": "a.md"}]
    assert data["scope"] == "knowledge"


def test_links_to_unknown_targets_are_dropped() -> None:
    rendered = render_graph("k", [_doc("a.md", ("nowhere.md", "https://example.com"))])
    assert json.loads(rendered)["edges"] == []


def test_output_ends_with_newline_and_has_no_timestamp() -> None:
    rendered = render_graph("k", [_doc("a.md")])
    assert rendered.endswith("\n")
    assert "generated" not in rendered
```

`tests/test_writer.py`:

```python
from pathlib import Path

from brain.writer import write_if_changed


def test_writes_lf_even_on_windows(tmp_path: Path) -> None:
    target = tmp_path / "out.md"
    assert write_if_changed(target, "one\ntwo\n") is True
    assert target.read_bytes() == b"one\ntwo\n"


def test_second_write_of_same_text_is_skipped(tmp_path: Path) -> None:
    target = tmp_path / "out.md"
    write_if_changed(target, "same\n")
    assert write_if_changed(target, "same\n") is False


def test_creates_missing_parents(tmp_path: Path) -> None:
    target = tmp_path / "deep" / "out.md"
    assert write_if_changed(target, "x\n") is True
    assert target.exists()
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

```bash
uv run pytest tests/test_graph.py tests/test_writer.py -q
```

Erwartung: FAIL — Module fehlen.

- [ ] **Schritt 3: Implementieren**

`src/brain/graph.py`:

```python
"""Rendering `graph.json`, a derived view that may be deleted at any time."""

import json
from collections.abc import Sequence
from pathlib import PurePosixPath

from brain.models import Document


def _resolve(source: str, target: str) -> str:
    """Resolve a link relative to the linking document, as a posix path."""
    base = PurePosixPath(source).parent
    return str(base.joinpath(target)) if not target.startswith("/") else target.lstrip("/")


def render_graph(area_scope: str, documents: Sequence[Document]) -> str:
    """Render nodes and edges, sorted, without any timestamp.

    A timestamp would make every run differ from the last and drown real
    changes in noise (spec 14).
    """
    known = {doc.relative for doc in documents}
    nodes = [
        {"id": doc.relative, "title": doc.title, "tags": list(doc.tags)}
        for doc in sorted(documents, key=lambda item: item.relative)
    ]
    edges: list[dict[str, str]] = []
    for doc in sorted(documents, key=lambda item: item.relative):
        for link in doc.links:
            target = _resolve(doc.relative, link)
            candidates = (target, f"{target}.md")
            match = next((c for c in candidates if c in known), None)
            if match is not None:
                edges.append({"from": doc.relative, "to": match})
    payload = {"scope": area_scope, "nodes": nodes, "edges": edges}
    return json.dumps(payload, ensure_ascii=False, indent=2, sort_keys=True) + "\n"
```

`src/brain/writer.py`:

```python
"""Writing generated files: always UTF-8, always LF, only when something changed."""

from pathlib import Path


def write_if_changed(path: Path, text: str) -> bool:
    """Write only on change, so untouched files keep their timestamps.

    `newline=""` keeps Python from translating LF to CRLF on Windows; the
    content hash of the maintenance layer depends on the bytes, not on the
    platform (spec 5.6).
    """
    if path.exists() and path.read_text(encoding="utf-8") == text:
        return False
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8", newline="")
    return True
```

- [ ] **Schritt 4: Tests laufen lassen**

```bash
uv run pytest tests/test_graph.py tests/test_writer.py -q
uv run ruff check . && uv run mypy
```

- [ ] **Schritt 5: Commit**

```bash
git add src/brain/graph.py src/brain/writer.py tests/test_graph.py tests/test_writer.py
git commit -m "Render the graph and write generated files deterministically"
```

---

## Aufgabe 8: Der Befehl `brain reindex` und der Reproduzierbarkeitsnachweis

Die Aufgabe, die alles zusammenführt — und das Fertig-Kriterium der Scheibe beweist.

**Dateien:**
- Anlegen: `src/brain/cli.py`, `tests/test_cli.py`

**Verbraucht:** alles aus den Aufgaben 2 bis 7.
**Produziert:** `main(argv: Sequence[str] | None = None) -> int` und `reindex(registry_path: Path) -> int`

- [ ] **Schritt 1: Fehlschlagende Tests schreiben**

`tests/test_cli.py`:

```python
from pathlib import Path

from brain.cli import main


def _make_area(root: Path) -> Path:
    (root / ".brain.toml").write_text(
        '[area]\nscope = "knowledge"\n\n[index]\ninclude = ["**/*.md"]\n',
        encoding="utf-8",
    )
    (root / "alpha.md").write_text(
        "---\ntitle: Alpha\ndescription: First\n---\n\nSee [[beta]]\n", encoding="utf-8"
    )
    (root / "beta.md").write_text("---\ntitle: Beta\n---\n\nText\n", encoding="utf-8")
    return root


def _registry(tmp_path: Path, area: Path) -> Path:
    registry = tmp_path / "registry.toml"
    registry.write_text(
        f'[[area]]\nscope = "knowledge"\npath = "{area.as_posix()}"\n', encoding="utf-8"
    )
    return registry


def test_writes_catalog_graph_and_register(tmp_path: Path) -> None:
    area = tmp_path / "vault"
    area.mkdir()
    _make_area(area)
    registry = _registry(tmp_path, area)
    assert main(["reindex", "--registry", str(registry)]) == 0
    assert (area / "index.md").exists()
    assert (area / "graph.json").exists()
    assert (area / "_identities.tsv").exists()
    assert "Alpha" in (area / "index.md").read_text(encoding="utf-8")


def test_every_directory_gets_its_own_catalog(tmp_path: Path) -> None:
    area = tmp_path / "vault"
    area.mkdir()
    _make_area(area)
    deep = area / "themen" / "technik"
    deep.mkdir(parents=True)
    (deep / "suche.md").write_text("---\ntitle: Suche\n---\n\nText\n", encoding="utf-8")
    registry = _registry(tmp_path, area)
    main(["reindex", "--registry", str(registry)])

    root = (area / "index.md").read_text(encoding="utf-8")
    assert "* [themen](themen/)" in root
    assert "[Alpha](alpha.md)" in root

    middle = (area / "themen" / "index.md").read_text(encoding="utf-8")
    assert "* [technik](technik/)" in middle
    assert "## Dateien" not in middle

    leaf = (area / "themen" / "technik" / "index.md").read_text(encoding="utf-8")
    assert "* [Suche](suche.md)" in leaf


def test_two_runs_produce_identical_bytes(tmp_path: Path) -> None:
    area = tmp_path / "vault"
    area.mkdir()
    _make_area(area)
    registry = _registry(tmp_path, area)
    main(["reindex", "--registry", str(registry)])
    first = {
        name: (area / name).read_bytes()
        for name in ("index.md", "graph.json", "_identities.tsv")
    }
    main(["reindex", "--registry", str(registry)])
    second = {
        name: (area / name).read_bytes()
        for name in ("index.md", "graph.json", "_identities.tsv")
    }
    assert first == second


def test_doc_ids_survive_a_second_run(tmp_path: Path) -> None:
    area = tmp_path / "vault"
    area.mkdir()
    _make_area(area)
    registry = _registry(tmp_path, area)
    main(["reindex", "--registry", str(registry)])
    before = (area / "_identities.tsv").read_text(encoding="utf-8")
    main(["reindex", "--registry", str(registry)])
    assert (area / "_identities.tsv").read_text(encoding="utf-8") == before


def test_revision_increases_only_when_content_changes(tmp_path: Path) -> None:
    area = tmp_path / "vault"
    area.mkdir()
    _make_area(area)
    registry = _registry(tmp_path, area)
    main(["reindex", "--registry", str(registry)])
    (area / "beta.md").write_text("---\ntitle: Beta\n---\n\nChanged\n", encoding="utf-8")
    main(["reindex", "--registry", str(registry)])
    rows = (area / "_identities.tsv").read_text(encoding="utf-8").splitlines()[1:]
    revisions = {row.split("\t")[1]: int(row.split("\t")[3]) for row in rows}
    assert revisions["beta.md"] == 2
    assert revisions["alpha.md"] == 1


def test_missing_registry_reports_an_error(tmp_path: Path) -> None:
    assert main(["reindex", "--registry", str(tmp_path / "absent.toml")]) == 1
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

```bash
uv run pytest tests/test_cli.py -q
```

Erwartung: FAIL — `No module named 'brain.cli'`.

- [ ] **Schritt 3: Implementieren**

`src/brain/cli.py`:

```python
"""The `brain` command. In this slice it does exactly one thing: reindex."""

import argparse
from collections.abc import Sequence
from dataclasses import replace
from pathlib import Path

from brain.catalog import render_catalog
from brain.document import read_document
from brain.graph import render_graph
from brain.identity import content_hash, new_doc_id, read_identities, render_identities
from brain.manifest import ManifestError, read_manifest
from brain.models import Area, Document, Identity
from brain.registry import RegistryError, read_registry
from brain.walk import find_files
from brain.writer import write_if_changed


def _nested_areas(area: Area, areas: Sequence[Area]) -> tuple[Path, ...]:
    """Every other registered area living inside this one is not this one's business."""
    return tuple(
        other.path
        for other in areas
        if other.scope != area.scope and other.path.resolve().is_relative_to(area.path.resolve())
    )


def _index_area(area: Area, areas: Sequence[Area]) -> None:
    manifest = read_manifest(area.path / ".brain.toml")
    files = find_files(area, manifest, nested=_nested_areas(area, areas))
    documents = [read_document(path, area.path) for path in files]

    identities = read_identities(area.path / "_identities.tsv")
    updated: dict[str, Identity] = {}
    for path, doc in zip(files, documents, strict=True):
        digest = content_hash(path)
        previous = identities.get(doc.relative)
        if previous is None:
            updated[doc.relative] = Identity(new_doc_id(), doc.relative, digest, 1)
        elif previous.content_hash != digest:
            updated[doc.relative] = Identity(
                previous.doc_id, doc.relative, digest, previous.revision + 1
            )
        else:
            updated[doc.relative] = previous

    _write_catalogs(area, documents)
    write_if_changed(area.path / "graph.json", render_graph(area.scope, documents))
    write_if_changed(area.path / "_identities.tsv", render_identities(updated))


def _write_catalogs(area: Area, documents: Sequence[Document]) -> None:
    """One catalog per directory, as the spec requires (section 8).

    A single catalog at the root would grow into a book; the ladder's first
    rung is meant to stay small enough to be read whole every time.
    """
    def parent_of(relative: str) -> str:
        return relative.rsplit("/", 1)[0] if "/" in relative else ""

    by_directory: dict[str, list[Document]] = {"": []}
    for doc in documents:
        directory = parent_of(doc.relative)
        by_directory.setdefault(directory, []).append(doc)
        # Register every ancestor as well, so an intermediate directory that
        # holds only subdirectories still gets a catalog.
        while directory:
            by_directory.setdefault(directory, [])
            directory = parent_of(directory)

    for directory, docs in sorted(by_directory.items()):
        children = sorted(
            other.rsplit("/", 1)[-1]
            for other in by_directory
            if other and parent_of(other) == directory
        )
        entries = [replace(doc, relative=doc.relative.rsplit("/", 1)[-1]) for doc in docs]
        target = (area.path / directory / "index.md") if directory else (area.path / "index.md")
        write_if_changed(target, render_catalog(directory or area.scope, entries, children))


def reindex(registry_path: Path) -> int:
    areas = read_registry(registry_path)
    for area in areas:
        if not area.path.exists():
            print(f"skipping {area.scope}: {area.path} does not exist")
            continue
        _index_area(area, areas)
    return 0


def main(argv: Sequence[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="brain")
    sub = parser.add_subparsers(dest="command", required=True)
    reindex_parser = sub.add_parser("reindex", help="rebuild catalogs, graph and identities")
    reindex_parser.add_argument("--registry", required=True, type=Path)
    args = parser.parse_args(argv)
    try:
        return reindex(args.registry)
    except (OSError, ManifestError, RegistryError) as error:
        print(f"error: {error}")
        return 1
```

- [ ] **Schritt 4: Tests laufen lassen**

```bash
uv run pytest -q
uv run ruff check . && uv run mypy
uv run coverage run -m pytest -q && uv run coverage report
```

Erwartung: alles grün, Coverage 100 %.

- [ ] **Schritt 5: Rauchtest gegen den echten Vault**

Der Vault ist bis auf `98 Messung/` leer — der Lauf soll trotzdem sauber durchgehen und **nichts** außerhalb des Vaults anfassen.

```bash
mkdir -p /c/Users/micro/AppData/Local/brain
cat > /c/Users/micro/AppData/Local/brain/registry.toml <<'EOF'
[[area]]
scope = "knowledge"
path = "C:/Users/micro/Documents/#GIT/brain-knowledge"
EOF
uv run brain reindex --registry /c/Users/micro/AppData/Local/brain/registry.toml
git -C "/c/Users/micro/Documents/#GIT/brain-knowledge" status --short
```

Erwartung: `index.md`, `graph.json` und `_identities.tsv` erscheinen im Vault. **Wichtig:** Der Vault hat bereits eine handgeschriebene `index.md` aus Scheibe 0 — sie wird überschrieben. Sichere sie vorher (`git -C … stash` oder Kopie), vergleiche beide und entscheide bewusst, welche gilt. Ein Indexer, der eine gepflegte Datei kommentarlos ersetzt, ist ein Fehler, keine Funktion.

- [ ] **Schritt 6: Zweiter Lauf, Nachweis der Reproduzierbarkeit**

```bash
uv run brain reindex --registry /c/Users/micro/AppData/Local/brain/registry.toml
git -C "/c/Users/micro/Documents/#GIT/brain-knowledge" status --short
```

Erwartung: **leere Ausgabe** — der zweite Lauf ändert keine Datei.

- [ ] **Schritt 7: Commit**

```bash
git add src/brain/cli.py tests/test_cli.py
git commit -m "Add the reindex command and prove reruns are byte-identical"
```

---

## Fertig-Kriterium der Scheibe

1. **Zwei Läufe auf unverändertem Bestand erzeugen byteweise identische Dateien** — nachgewiesen im Test *und* am echten Vault mit leerem `git status`.
2. **Kennungen überleben den zweiten Lauf.** `_identities.tsv` bleibt unverändert; `revision` steigt nur bei geänderten Inhalten.
3. **Kopien und Werkzeugverzeichnisse sind ausgeschlossen** — `.claude`, `.tools`, `.superpowers`, `.git`, `.obsidian`, `node_modules`, Testfixturen.
4. **Verschachtelte Bereiche fallen aus der Sammlung des Elternbereichs heraus.**
5. **Werkzeugkette sauber:** `pytest`, `ruff`, `mypy --strict`, Coverage 100 %.

## Bekannte Einschränkungen

Zwei Punkte, die diese Scheibe bewusst offen lässt. Sie sind hier festgehalten, nicht gelöst.

**Eine gelöschte oder umbenannte Datei verliert ihre `doc_id` endgültig.** `cli._index_area` baut das Register ausschließlich aus den Dateien, die der aktuelle Lauf findet. Die Zeile einer entfernten Datei verschwindet damit; wird dieselbe Datei später wieder angelegt, bekommt sie eine neue ULID und `revision` beginnt wieder bei 1. `identity.py` verspricht „stabile Identität" — das gilt derzeit nur für Dateien, die sich nie bewegen, und deckt damit stillschweigend auch das **Umbenennen** mit ab, das viel häufiger vorkommt als das Löschen. Ob eine entfernte Zeile einen Grabstein erhält oder wirklich verschwindet, legt den Vertrag des Registers auf der Platte fest. Diese Entscheidung gehört der Wartungsscheibe.

**`graph.json` deklariert nicht, dass `edges` eine Menge von Beziehungen ist.** Kanten werden zwar dedupliziert, aber die Invariante steht nur als Kommentar in `graph.py`, also dort, wo der Erzeuger sie sieht — nicht dort, wo ein Verbraucher sie lesen würde. Die Scheibe, die `graph.json` als erste liest, braucht diese Zusage in der Beschreibung des Formats selbst und besitzt sie damit.

## Rauchtest gegen echten Bestand (2026-08-20)

Der Rauchtest ist gelaufen; der offene Punkt der Abnahme ist damit erledigt.
**Entschieden wurde die zweite Möglichkeit:** der handgeschriebene Abschnitt ist
umgezogen, der Indexer lernt nichts dazu. Prosa und Bestandsregel stehen jetzt in
der `CLAUDE.md` des Vaults, die Zahlen in dessen `98 Messung/externe-bestaende.md`
(Commit `062cc35` in `brain-knowledge`).

**Lauf 1 — der Vault selbst.** Unter seinem eigenen Manifest
(`include = ["10 Rohquellen/**/*.md"]`) ist der Vault leer; der Lauf erzeugt
korrekt null Dokumente.

**Lauf 2 — echtes Material.** Weil der Vault leer ist und die drei externen
Bestände nur gelesen werden dürfen, lief der Indexer gegen eine **Kopie** von
`#GIT/space` im Scratchpad: 38 GB, 1162 Markdown-Dateien. Ergebnis: 144 Dokumente,
34 Kataloge, 19,5 s für den ersten und 15,2 s für den zweiten Lauf. **Die Ausgabe
beider Läufe ist byteweise identisch** — das Fertig-Kriterium der Scheibe hält
damit nicht nur gegen synthetische Bäume.

Die Auswahl stimmte: von 1162 Dateien blieben 795 im `.claude`-Plugin-Cache und
34 unter `.tools` außen vor, dazu die 11 vom Indexer selbst erzeugten `index.md`.
Die Link-Auflösung löste **959 von 979 Links** auf; die 20 übrigen sind zu Recht
offen (externe URLs, Anker, Nicht-Markdown-Ziele, drei tote Links im Bestand).
Weder `%20` noch `..` haben einen einzigen Fehlschlag verursacht.

### Was der Rauchtest zusätzlich zutage gefördert hat

**Ein Graph ohne eine einzige Kante sieht aus wie ein Erfolg.** Mit der
Repo-Wurzel als Bereichswurzel kamen aus denselben 141 Notizen **0 Kanten**
heraus, mit `space/wiki` als Wurzel **959**. Ursache: die Notizen verlinken
wurzelabsolut (`/architecture/ortsmodell.md`), und diese Wurzel ist die des
Obsidian-Vaults, nicht die des Bereichs. Der Indexer verhält sich in beiden
Fällen richtig — er sagt nur nirgends, dass er gerade jeden Link verworfen hat.
Ein Lauf, der 0 von 979 Links auflöst, ist nicht von einem Bestand ohne Links zu
unterscheiden. **Der Indexer braucht eine Auflösungsquote in seiner Ausgabe.**

**`**/.superpowers/**` schließt zu viel aus.** Der Standardausschluss nimmt den
ganzen Ordner heraus, im `space`-Bestand 178 Dateien. Die Spec (§5.3) nennt aber
nur `sdd/` „ignoriert, nicht indexiert" und `specs/` und `plans/` ausdrücklich
Rohquellen — den Vertrag. In der jetzigen Form indiziert der Indexer die Spec
dieses Projekts nicht. Richtig wäre `**/.superpowers/sdd/**`.

**Der erzeugte Wurzelkatalog trägt keine Orientierung.** `render_catalog` listet
nur Verzeichnisse, in denen Dokumente liegen. Die handgeschriebene Einleitung des
Vaults — wozu der Katalog da ist, dass die Suchleiter in `CLAUDE.md` steht, welche
Bereiche leer sind — hat im erzeugten Format keinen Platz. Der Umzug oben löst
das für diesen einen Fall, nicht für den nächsten. Die Scheibe, die den Katalog
zum ersten Mal als Einstieg benutzt, braucht dafür einen Ort.

## Offener Punkt für die Abnahme (erledigt)

Der Vault trägt aus Scheibe 0 eine **handgeschriebene** `index.md` mit einem Abschnitt zu den extern indexierten Beständen. Der Indexer kennt diesen Abschnitt nicht und würde ihn beim Überschreiben verlieren. Das ist beim Rauchtest zu entscheiden: Entweder der Indexer lernt, einen von Hand gepflegten Kopfabschnitt zu erhalten — dann ist das eine eigene Aufgabe —, oder der Abschnitt wandert dorthin, wo er nicht überschrieben wird. **Nicht stillschweigend überschreiben.**

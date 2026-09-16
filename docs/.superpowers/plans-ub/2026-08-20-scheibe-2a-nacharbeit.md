# Scheibe 2a — Nacharbeit nach der Abnahme · Implementierungsplan

> **Für ausführende Agenten:** ERFORDERLICHER SUB-SKILL: `superpowers:subagent-driven-development` (empfohlen) oder `superpowers:executing-plans`, um diesen Plan Aufgabe für Aufgabe abzuarbeiten. Schritte nutzen Checkbox-Syntax (`- [ ]`).

**Ziel:** Die Suchkette an den Vertrag angleichen, wie ihn die Abnahme korrigiert hat — eine zurückgenommene Zusage entfernen, drei Präzisierungen umsetzen und einen Fehler der Vorgabe beheben.

**Vorgehen:** Vier kleine Änderungen und eine Entfernung. Kein neues Teilsystem, keine neue Naht. Die Reihenfolge ist so gewählt, daß jede Aufgabe für sich grün ist: erst die Entfernung (sie macht Code frei, den die übrigen nicht mehr berücksichtigen müssen), dann die Manifest-Angabe, dann die davon abhängigen Ausschlüsse, zuletzt der neue Befehl und die Vorgabe.

**Werkzeuge:** Python ≥ 3.14, `uv`, `pytest`, `ruff`, `mypy`, `coverage`. Extern: `qmd` 2.8.3.

**Spec:** `docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md`, fortgeschrieben in Commit `64c86c0`. Maßgeblich sind §5.3, §5.5, §7.2, §7.4, §16.12, §16.14, §17 und die Entscheidungen 39, 40 sowie 42–45.

**Arbeitsbereich:** `.worktrees/scheibe-2a-suchkette` auf Branch `scheibe-2a-suchkette`. Die Vorarbeit steht dort in 31 Commits; dieser Plan setzt darauf auf.

## Globale Rahmenbedingungen

- **Python ≥ 3.14 überall explizit:** `requires-python = ">=3.14"`, ruff `target-version = "py314"`, mypy `python_version = "3.14"`.
- **`uv` für alles.** Kein `pip`. Aufrufe über `uv run`.
- **TDD.** Erst der fehlschlagende Test, dann die Implementierung.
- **100 % Coverage, gemessen.** Ausschlüsse nur mit begründendem Kommentar: `# pragma: no cover  # <Grund>`, nie nackt.
- **Typisierung vollständig.** Kein `Any`, kein `# type: ignore` ohne begründenden Kommentar. `mypy --strict` läuft ohne Fehler.
- **Imports stehen oben**, auf Modulebene.
- **Sprachen:** Quellcode, Bezeichner, Code-Kommentare, Commit-Nachrichten und Fehlermeldungen **englisch**; Prosa in Dokumenten deutsch. Kommentare in Konfigurationsdateien deutsch.
- **Kommentare erklären das *Warum*.**
- **Keine Suchmaschine, keine Modelle, kein Netz, kein Prozeßstart** in der normalen Testrunde. Die mit `contract` markierten Tests laufen getrennt.
- **Die drei Bestände unter `#GIT\space`, `#GIT\iam_wiki` und `#GIT\#Obsidian\AI` werden nur gelesen.** Keine Aufgabe dieses Plans schreibt dorthin.
- **Die echte `~/.config/qmd/index.yml` wird in Tests nie angefaßt.** Eine `autouse`-Fixture in `tests/conftest.py` biegt `brain.cli.qmd_config_path` und `brain.cli._port` in jedem Test um; sie bleibt bestehen.

### Was diese Nacharbeit nicht anfaßt

Daemon, IPC, MCP (2c). `bench` und jede Messung (2b). Die Frage, ob Specs und Pläne aus dem Punktverzeichnis umziehen sollen — das ist eine Konvention über Projektgrenzen hinweg und ausdrücklich vertagt.

### Die Falle, die man kennen muß

**Aufgabe 1 entfernt eine Zusage, nicht einen Fehler.** Der Code funktioniert; er hält bloß eine Zusage, die die Spec zurückgenommen hat. Die Versuchung ist, ihn „für später" stehenzulassen — das wäre falsch: eine Gruppierung, die nie greift, sieht im Code aus wie eine, die greift, und der nächste Leser baut darauf auf. Was bleibt, ist eine Meldung, und die kostet drei Zeilen.

---

## Dateistruktur

```
src/brain/
  models.py         Manifest um `unsearched` erweitern
  manifest.py       [index] unsearched lesen und prüfen
  privacy.py        matches_never -> matches_globs umbenennen (wird jetzt zweimal gebraucht)
  walk.py           DEFAULT_EXCLUDES in zwei Listen teilen
  core.py           Entdopplung entfernen; unsearched abziehen; Inhaltshash-Befund
  cli.py            also_at aus der Ausgabe; brain embed; --profile Vorgabe fast
  search/port.py    embed() an der Naht
  search/qmd.py     embed() über `qmd embed`
  search/fake.py    embed() in der Attrappe
  search/qmd_config.py  Artefaktnamen und unsearched in die Sammlungsdefinition
tests/
  (die bestehenden Testmodule wachsen mit; keine neue Datei)
```

---

## Aufgabe 1: Die Entdopplung entfernen, den Befund behalten

Die Spec hat die Zusage zurückgenommen (§16.12): Die `doc_id` wird je Pfad vergeben, die Gruppierung konnte nie greifen, und der gemessene Fall liegt außerhalb der registrierten Bereiche. Was bleibt, ist eine Meldung in `status`.

**Files:**
- Modify: `src/brain/core.py`, `src/brain/cli.py`
- Test: `tests/test_core_search.py`, `tests/test_core_tools.py`, `tests/test_cli.py`

**Interfaces:**
- Consumes: `Result`, `SearchAnswer`, `_visible_areas`, `read_identities`
- Produces: `Result` **ohne** das Feld `also_at`; `_assemble(hits, areas, state_dir, n)` unverändert in der Signatur, ohne Gruppierung; `status` meldet zusätzlich geteilte Inhaltshashes

- [ ] **Schritt 1: Die Tests anpassen, die die Zusage prüfen**

In `tests/test_core_search.py` entfallen `test_same_document_under_two_paths_appears_once`, `test_n_is_applied_after_deduplication`, `test_a_repetition_does_not_cost_another_document_its_place`, `test_the_same_doc_id_in_two_areas_is_two_documents` und `test_engine_grouping_that_disagrees_is_reported`. An ihre Stelle tritt einer, der das neue Verhalten festhält:

```python
def test_two_paths_of_one_document_are_two_results(world: Path) -> None:
    """The promise to merge them was retired in spec 16.12 — status reports it now."""
    port = FakePort(results=[[_hit("open", "a.md", score=0.9), _hit("open", "copy/a.md", score=0.7)]])
    answer = _search(port, world)
    assert [r.relative for r in answer.results] == ["a.md", "copy/a.md"]
```

- [ ] **Schritt 2: Den neuen `status`-Test schreiben**

Anhängen an `tests/test_core_tools.py`. Die `world`-Fixture schreibt bereits ein Register, in dem `a.md` und `copy/a.md` denselben `content_hash` (`sha256:1`) tragen:

```python
def test_status_reports_two_paths_that_share_a_content_hash(world: Path) -> None:
    """The retired deduplication leaves a finding behind, not a mechanism."""
    port = FakePort(results=[], indexed={"open": (), "closed": ()})
    lines = status(channel=Channel.LOCAL, port=port, state_dir=world)
    assert any("a.md" in line and "copy/a.md" in line for line in lines)


def test_status_is_silent_when_every_hash_is_unique(tmp_path: Path) -> None:
    """A finding that fires on a healthy corpus is a finding nobody reads."""
    state = _area_with_register(
        tmp_path,
        "doc_id\tpfad\tcontent_hash\trevision\n01AAA\ta.md\tsha256:1\t1\n01BBB\tb.md\tsha256:2\t1\n",
    )
    port = FakePort(results=[], indexed={"solo": ()})
    assert not any("content hash" in line for line in status(
        channel=Channel.LOCAL, port=port, state_dir=state
    ))
```

`_area_with_register(tmp_path, register_text)` gibt es noch nicht — leg ihn in `tests/conftest.py` an. Er baut ein Zustandsverzeichnis mit genau einem sichtbaren, schreibbaren Bereich `solo`: Registrierung, Manifest mit `include = ["**/*.md"]`, das übergebene `_identities.tsv` und ein `graph.json` mit leeren `nodes`, `edges` und `links` (`{"total": 0, "resolved": 0, "dropped": {}}`), damit `status` nicht an der Link-Zeile hängenbleibt. Vorbild ist die vorhandene `world`-Fixture in derselben Datei.

Ebenfalls in `tests/conftest.py` gehört `_search(port, state_dir, **overrides)`, den `tests/test_core_search.py` bereits lokal führt — Aufgabe 1 benutzt ihn, und ein zweites Testmodul soll ihn nicht nachbauen. Verschieb ihn, statt ihn zu kopieren.

- [ ] **Schritt 3: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_core_search.py tests/test_core_tools.py -q`
Expected: FAIL — die Trefferliste ist noch entdoppelt, `status` kennt den Befund nicht

- [ ] **Schritt 4: `_assemble` entkernen**

`by_document` und `keys_seen` entfallen ersatzlos, `Result.also_at` ebenfalls. Die Schleife wird zu:

```python
    for hit in hits:
        manifest = manifests.get(hit.collection)
        if manifest is None or not is_readable(manifest, hit.relative):
            continue
        identity = registers[hit.collection].get(hit.relative)
        if identity is None:
            findings.append(
                f"{hit.collection}/{hit.relative}: hit is not in the register; "
                "reindex to catch up"
            )
        ordered.append(_result(hit, identity))
```

Dazu in den Docstring von `_assemble`, damit niemand die Gruppierung „wiederherstellt":

```python
    """Turn engine hits into results, dropping what this caller may not see.

    There is deliberately no grouping here. The promise to merge several paths
    of one document was retired after the acceptance run of slice 2a (spec
    16.12): `doc_id` is minted per path, so the grouping could never fire, and
    the duplicates that motivated it lived in a mirror the exclude list already
    removes. `status` reports two paths sharing a content hash instead.
    """
```

- [ ] **Schritt 5: Die Übererfragung zurücknehmen**

In `_ask` wird `n * 4` zu `n`. Der Faktor existierte allein, damit nach dem Entdoppeln noch `n` Treffer übrigbleiben; ohne Entdopplung fordert er das Vierfache an und wirft drei Viertel weg. Der Kommentar dazu:

```python
    """One retry on empty, then believe it.

    The engine occasionally returns an empty set without failing — measured at
    four in roughly 120 queries during slice 0. `n` is asked for exactly: the
    fourfold that stood here served the deduplication, which is gone.
    """
```

- [ ] **Schritt 6: Den Befund in `status` bauen**

In `core.py`, aufgerufen aus `status` vor `_unfindable`:

```python
def _shared_hashes(area: Area, state_dir: Path) -> list[str]:
    """Two paths with identical bytes — what is left of the retired deduplication.

    The register knows it anyway, so the line is free. If the case ever turns
    up in a real corpus, it is visible and can be decided on evidence, instead
    of a mechanism kept in reserve for a case nobody has (spec 16.12).
    """
    by_hash: dict[str, list[str]] = {}
    for identity in read_identities(area_artifact_dir(area, state_dir) / "_identities.tsv").values():
        by_hash.setdefault(identity.content_hash, []).append(identity.relative)
    return [
        f"{area.scope}: same content hash under {len(paths)} paths: {', '.join(sorted(paths))}"
        for _, paths in sorted(by_hash.items())
        if len(paths) > 1
    ]
```

- [ ] **Schritt 7: `also_at` aus der Ausgabe entfernen**

In `cli._print_search` fallen die beiden Zeilen mit `result.also_at` weg. In `tests/test_cli.py` entfällt die zugehörige Zusicherung.

- [ ] **Schritt 8: Volle Runde**

Run: `uv run pytest -q && uv run ruff check && uv run ruff format --check && uv run mypy && uv run coverage run -m pytest && uv run coverage report`
Expected: PASS, Coverage 100 %

- [ ] **Schritt 9: Commit**

```bash
git add src/brain/core.py src/brain/cli.py tests/
git commit -m "Retire the deduplication and keep the finding it leaves behind"
```

---

## Aufgabe 2: `unsearched` im Manifest

Die Angabe trennt „indiziert" von „durchsucht" (Entscheidung 42). Sie ist die Voraussetzung für Aufgabe 3 und 4.

**Files:**
- Modify: `src/brain/models.py`, `src/brain/manifest.py`, `src/brain/privacy.py`
- Test: `tests/test_manifest.py`, `tests/test_privacy.py`

**Interfaces:**
- Consumes: `_globs(path, key, value)` aus `manifest.py`
- Produces: `Manifest.unsearched: tuple[str, ...] = ()`; `privacy.matches_globs(patterns, relative) -> bool` (umbenannt aus `matches_never`, Verhalten unverändert)

- [ ] **Schritt 1: Die fehlschlagenden Tests schreiben**

Anhängen an `tests/test_manifest.py`:

```python
def test_reads_unsearched(tmp_path: Path) -> None:
    path = tmp_path / ".brain.toml"
    path.write_text(
        '[area]\nscope = "p"\n\n[index]\ninclude = ["**/*.md"]\n'
        'unsearched = ["docs/.superpowers/**"]\n',
        encoding="utf-8",
    )
    assert read_manifest(path).unsearched == ("docs/.superpowers/**",)


def test_unsearched_defaults_to_nothing(tmp_path: Path) -> None:
    path = tmp_path / ".brain.toml"
    path.write_text('[area]\nscope = "p"\n', encoding="utf-8")
    assert read_manifest(path).unsearched == ()


def test_unsearched_must_be_a_list_of_strings(tmp_path: Path) -> None:
    path = tmp_path / ".brain.toml"
    path.write_text('[area]\nscope = "p"\n\n[index]\nunsearched = "docs/**"\n', encoding="utf-8")
    with pytest.raises(ManifestError, match="unsearched"):
        read_manifest(path)
```

In `tests/test_privacy.py` werden die Aufrufe von `matches_never` auf `matches_globs` umgestellt; die Aussagen bleiben unverändert.

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_manifest.py tests/test_privacy.py -q`
Expected: FAIL — `Manifest` kennt `unsearched` nicht, `matches_globs` existiert nicht

- [ ] **Schritt 3: `models.py` erweitern**

In `Manifest` ergänzen:

```python
    unsearched: tuple[str, ...] = ()
```

- [ ] **Schritt 4: `manifest.py` erweitern**

Im `Manifest(...)`-Aufruf ergänzen:

```python
        unsearched=_globs(path, "unsearched", index.get("unsearched", [])),
```

- [ ] **Schritt 5: `privacy.matches_never` umbenennen**

Der Name war richtig, solange es einen Aufrufer gab; jetzt sind es zwei mit verschiedener Bedeutung. Die Funktion bleibt Zeile für Zeile dieselbe, nur der Name wird allgemein und der Docstring nennt beide Benutzungen:

```python
def matches_globs(patterns: tuple[str, ...], relative: str) -> bool:
    """Match a path against a list of globs, case- and form-insensitively.

    Two callers with different meanings share this: `never` (a path no channel
    may reach) and `unsearched` (a path the search engine is not asked about).
    Both must agree with a filesystem that does not distinguish case, and both
    are compared against the same normal form — a pattern written in NFC and a
    directory stored in NFD are the same directory (spec 5.5).
    """
```

`is_readable` ruft sie weiterhin mit `manifest.never`; `walk.find_files` ebenso.

- [ ] **Schritt 6: Volle Runde**

Run: `uv run pytest -q && uv run ruff check && uv run ruff format --check && uv run mypy && uv run coverage run -m pytest && uv run coverage report`
Expected: PASS, Coverage 100 %

- [ ] **Schritt 7: Commit**

```bash
git add src/brain/models.py src/brain/manifest.py src/brain/privacy.py tests/
git commit -m "Let an area declare which of its sources are not searched"
```

---

## Aufgabe 3: `unsearched` wirkt — Sammlung und Divergenzmeldung

Erst hier bekommt die Angabe ihre drei Wirkungen: nicht in die Sammlung der Suchmaschine, nicht in die Divergenzzählung, kein Rauschen in `status`.

**Files:**
- Modify: `src/brain/core.py`, `src/brain/search/qmd_config.py`, `src/brain/cli.py`
- Test: `tests/test_core_tools.py`, `tests/test_qmd_config.py`

**Interfaces:**
- Consumes: `Manifest.unsearched`, `privacy.matches_globs`
- Produces: `_unfindable` zieht `unsearched` ab; `CollectionSpec.ignore` enthält `manifest.unsearched`

- [ ] **Schritt 1: Die fehlschlagenden Tests schreiben**

Anhängen an `tests/test_core_tools.py`:

```python
def test_unsearched_paths_are_not_counted_as_missing(tmp_path: Path) -> None:
    """They are supposed to be unfindable; reporting them trains the reader to skip."""
    state = _area_with_unsearched(tmp_path, unsearched=["notes/plans/**"])
    port = FakePort(results=[], indexed={"solo": ("notes/open.md",)})
    lines = status(channel=Channel.LOCAL, port=port, state_dir=state)
    assert not any("unknown to the search engine" in line for line in lines)


def test_a_real_backlog_is_still_reported(tmp_path: Path) -> None:
    """Subtracting the declared ones must not silence the line altogether."""
    state = _area_with_unsearched(tmp_path, unsearched=[])
    port = FakePort(results=[], indexed={"solo": ()})
    lines = status(channel=Channel.LOCAL, port=port, state_dir=state)
    assert any("unknown to the search engine" in line for line in lines)
```

`_area_with_unsearched(tmp_path, *, unsearched)` gibt es noch nicht. Bau ihn in `tests/conftest.py` auf `_area_with_register` aus Aufgabe 1 auf: derselbe Bereich `solo`, dessen Register `notes/open.md` und `notes/plans/one.md` führt, mit dem gegebenen `unsearched` im Manifest. Die beiden Tests unterscheiden sich nur in dieser einen Angabe — wenn dein Helfer sie nicht zum einzigen Unterschied macht, prüfen sie zwei Dinge auf einmal.

Anhängen an `tests/test_qmd_config.py`:

```python
def test_unsearched_paths_are_kept_out_of_the_collection(tmp_path: Path) -> None:
    """What we do not want searched should not be indexed by the engine either."""
    config = tmp_path / "index.yml"
    sync_collections(
        config,
        {"p": CollectionSpec(Path("C:/p"), "**/*.md", ("**/.git/**", "docs/.superpowers/**"))},
    )
    entry = yaml.safe_load(config.read_text(encoding="utf-8"))["collections"]["p"]
    assert "docs/.superpowers/**" in entry["ignore"]
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_core_tools.py tests/test_qmd_config.py -q`
Expected: FAIL — `unsearched` wirkt nirgends

- [ ] **Schritt 3: `_unfindable` zieht ab**

In `core._unfindable`, unmittelbar nachdem `ours` gebildet ist:

```python
    ours = [
        relative
        for relative in ours
        if not matches_globs(manifest.unsearched, relative)
    ]
```

und im Docstring der bereits vorhandene Absatz („Paths the manifest keeps out of every answer are left out of the count") wird um den zweiten Grund ergänzt:

```python
    Two kinds are left out of the count: paths the manifest keeps out of every
    answer (`never`), and paths it declares as not searched (`unsearched`).
    Both are supposed to be unfindable. Reporting them would teach the reader
    to skip the line, and a line that gets skipped protects nothing.
```

- [ ] **Schritt 4: Die Sammlungsdefinition schließt sie aus**

In `cli.reindex`, beim Bau der `CollectionSpec`:

```python
            ignore=manifest.exclude + manifest.unsearched + DEFAULT_EXCLUDES,
```

Mehr nicht. `DEFAULT_EXCLUDES` wird in Aufgabe 4 durch `artifact_excludes(area) + ALWAYS_EXCLUDES` ersetzt; diese Aufgabe fügt nur `manifest.unsearched` hinzu und läßt den Rest der Zeile unangetastet. So bleiben beide Aufgaben für sich grün, in welcher Reihenfolge auch immer sie laufen.

- [ ] **Schritt 5: Volle Runde**

Run: `uv run pytest -q && uv run ruff check && uv run ruff format --check && uv run mypy && uv run coverage run -m pytest && uv run coverage report`
Expected: PASS, Coverage 100 %

- [ ] **Schritt 6: Commit**

```bash
git add src/brain/core.py src/brain/cli.py tests/
git commit -m "Subtract declared unsearched sources from the divergence report"
```

---

## Aufgabe 4: Artefaktnamen nur im eigenen Baum ausschließen

Der pauschale Ausschluß kostete in `iam_wiki` 22 handgeschriebene, eingecheckte `index.md` — in einem Wiki die Navigationsebene (Entscheidung 43).

**Files:**
- Modify: `src/brain/walk.py`, `src/brain/cli.py`
- Test: `tests/test_walk.py`, `tests/test_cli.py`

**Interfaces:**
- Consumes: `Area.readonly`
- Produces: `ALWAYS_EXCLUDES: tuple[str, ...]`, `OWN_ARTIFACTS: tuple[str, ...]`, `artifact_excludes(area: Area) -> tuple[str, ...]`; `DEFAULT_EXCLUDES` entfällt

- [ ] **Schritt 1: Die fehlschlagenden Tests schreiben**

Anhängen an `tests/test_walk.py`:

```python
def test_a_readonly_area_keeps_its_own_index_md(tmp_path: Path) -> None:
    """We never write into such a tree, so a file of that name is a human's."""
    (tmp_path / ".brain.toml").write_text(
        '[area]\nscope = "r"\n\n[index]\ninclude = ["**/*.md"]\n', encoding="utf-8"
    )
    (tmp_path / "index.md").write_text("# Navigation\n", encoding="utf-8")
    (tmp_path / "note.md").write_text("x\n", encoding="utf-8")
    manifest = read_manifest(tmp_path / ".brain.toml")
    area = Area(scope="r", path=tmp_path, readonly=True)
    found = [p.name for p in find_files(area, manifest)]
    assert found == ["index.md", "note.md"]


def test_a_writable_area_still_excludes_its_own_artefacts(tmp_path: Path) -> None:
    """Otherwise run 2 reads what run 1 wrote and reruns stop being identical."""
    (tmp_path / ".brain.toml").write_text(
        '[area]\nscope = "w"\n\n[index]\ninclude = ["**/*.md"]\n', encoding="utf-8"
    )
    for name in ("index.md", "index.intro.md", "note.md"):
        (tmp_path / name).write_text("x\n", encoding="utf-8")
    manifest = read_manifest(tmp_path / ".brain.toml")
    area = Area(scope="w", path=tmp_path)
    assert [p.name for p in find_files(area, manifest)] == ["note.md"]


def test_dot_directories_are_excluded_in_both_kinds_of_area(tmp_path: Path) -> None:
    """The split is about our own file names, not about the ballast."""
    (tmp_path / ".brain.toml").write_text(
        '[area]\nscope = "r"\n\n[index]\ninclude = ["**/*.md"]\n', encoding="utf-8"
    )
    (tmp_path / ".git").mkdir()
    (tmp_path / ".git" / "note.md").write_text("x\n", encoding="utf-8")
    (tmp_path / "note.md").write_text("x\n", encoding="utf-8")
    manifest = read_manifest(tmp_path / ".brain.toml")
    for readonly in (True, False):
        area = Area(scope="r", path=tmp_path, readonly=readonly)
        assert [p.name for p in find_files(area, manifest)] == ["note.md"]
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_walk.py -q`
Expected: FAIL — `index.md` wird in beiden Bereichsarten ausgeschlossen

- [ ] **Schritt 3: Die Liste teilen**

`DEFAULT_EXCLUDES` zerfällt. Der Kommentar trägt den Grund für die Teilung, nicht ihre Beschreibung:

```python
ALWAYS_EXCLUDES: tuple[str, ...] = (
    "**/.git/**",
    "**/.obsidian/**",
    # Worktree copies: byte-near twins of real notes, measured in slice 0.
    "**/.claude/**",
    # Licence texts and tool ballast, not knowledge.
    "**/.tools/**",
    # Only the working traces: specs and plans are sources the spec names (5.3).
    "**/.superpowers/sdd/**",
    "**/node_modules/**",
    "**/tests/fixtures/**",
)

# Our own file names. Excluding them keeps run 2 from reading what run 1 wrote,
# which is what byte-identical reruns rest on (spec 14) — but only where run 1
# writes. In a read-only area we write nothing into the tree, so a file of this
# name is a human's: the acceptance run found 22 hand-written index.md in a
# foreign wiki, its whole navigation layer, silently absent from both indexes
# (spec 5.5, decision 43).
OWN_ARTIFACTS: tuple[str, ...] = (
    "**/index.md",
    "**/index.intro.md",
    "**/graph.json",
    "**/_identities.tsv",
    "**/.brain.toml",
)


def artifact_excludes(area: Area) -> tuple[str, ...]:
    """The exclusions that depend on whether this area's tree is ours to write."""
    return () if area.readonly else OWN_ARTIFACTS
```

In `find_files`:

```python
    excludes = manifest.exclude + artifact_excludes(area) + ALWAYS_EXCLUDES
```

**Ein Sonderfall, der einen eigenen Satz braucht:** `**/.brain.toml` bleibt in `OWN_ARTIFACTS`. In einem `readonly`-Bereich liegt unser Manifest im Zustandsverzeichnis, eine `.brain.toml` im fremden Baum gehört also jemand anderem — sie ist trotzdem eine Konfigurationsdatei und kein Wissen. Vermerk das im Kommentar, sonst wirkt sie wie ein Versehen.

- [ ] **Schritt 4: `cli.py` nachziehen**

Jeder Verweis auf `DEFAULT_EXCLUDES` wird zu `artifact_excludes(area) + ALWAYS_EXCLUDES`. Der Import wird entsprechend angepaßt.

- [ ] **Schritt 5: Volle Runde**

Run: `uv run pytest -q && uv run ruff check && uv run ruff format --check && uv run mypy && uv run coverage run -m pytest && uv run coverage report`
Expected: PASS, Coverage 100 %

- [ ] **Schritt 6: Commit**

```bash
git add src/brain/walk.py src/brain/cli.py tests/
git commit -m "Exclude our own file names only where we write them"
```

---

## Aufgabe 5: `brain embed` und die Profil-Vorgabe

Zwei kleine, voneinander unabhängige Punkte, die zusammen einen Commit wert sind, weil beide nur die Kommandozeile betreffen.

**Files:**
- Modify: `src/brain/search/port.py`, `src/brain/search/qmd.py`, `src/brain/search/fake.py`, `src/brain/cli.py`
- Test: `tests/test_search_port.py`, `tests/test_search_qmd.py`, `tests/test_cli.py`, `tests/test_qmd_contract.py`

**Interfaces:**
- Produces: `SearchPort.embed(collections: tuple[str, ...]) -> None`; `brain embed`; `--profile` mit Vorgabe `fast`

- [ ] **Schritt 1: Die fehlschlagenden Tests schreiben**

Anhängen an `tests/test_search_port.py`:

```python
def test_fake_records_an_embed() -> None:
    port = FakePort(results=[])
    port.embed(("space",))
    assert port.embedded == [("space",)]
```

Anhängen an `tests/test_search_qmd.py`:

```python
def test_embed_runs_the_embedding_command() -> None:
    run, recorded = _runner(CompletedProcess([], 0, "", ""))
    QmdPort(runner=run).embed(("space",))
    assert recorded[0][1] == "embed"


def test_a_failed_embed_is_unavailable() -> None:
    run, _ = _runner(CompletedProcess([], 1, "", "model missing"))
    with pytest.raises(SearchUnavailable, match="model missing"):
        QmdPort(runner=run).embed(("space",))
```

Anhängen an `tests/test_cli.py`:

```python
def test_embed_is_its_own_command(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """reindex writes the full-text index; the vectors are a second, slow run."""
    port = FakePort(results=[])
    monkeypatch.setattr("brain.cli._port", lambda: port)
    state = _area_with_register(tmp_path, "doc_id\tpfad\tcontent_hash\trevision\n")
    assert main(["embed", "--state-dir", str(state)]) == 0
    assert port.embedded == [("solo",)]


def test_search_defaults_to_the_fast_profile(
    world: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """Spec 7.3 and 13: `fast` is the CLI default — `full` is where a model waits."""
    port = FakePort(results=[[], []])
    monkeypatch.setattr("brain.cli._port", lambda: port)
    main(["search", "q", "--state-dir", str(world)])
    (_, _, profile, _) = port.calls[0]
    assert profile is Profile.FAST
```

- [ ] **Schritt 2: Tests laufen lassen, Fehlschlag bestätigen**

Run: `uv run pytest tests/test_search_port.py tests/test_search_qmd.py tests/test_cli.py -q`
Expected: FAIL — `embed` existiert nicht, die Vorgabe ist `full`

- [ ] **Schritt 3: Die Naht erweitern**

In `port.py` zum Protokoll:

```python
    def embed(self, collections: tuple[str, ...]) -> None: ...
```

In `qmd.py`:

```python
    def embed(self, collections: tuple[str, ...]) -> None:
        """Generate the vectors the update run does not.

        Measured against qmd 2.8.3: `update` writes the full-text index and
        leaves the embeddings pending. A document without vectors is missing
        from every semantic search, and that is the stage the cross-language
        bridge rests on (spec 16.3) — so this is not an optimisation but the
        difference between a half-current chain and a current one.

        `collections` is not passed on: the embedding command takes a `-c`
        switch, but a partial run leaves the remaining areas silently pending,
        which is the state this command exists to end. The parameter stays in
        the contract because slice 2c needs it.
        """
        self._invoke([*_launcher(self._executable), "embed"])
```

In `fake.py`:

```python
    def embed(self, collections: tuple[str, ...]) -> None:
        self.embedded.append(collections)
```

und `self.embedded: list[tuple[str, ...]] = []` im Konstruktor.

- [ ] **Schritt 4: Die Kommandozeile**

Ein Unterparser `embed` mit `--state-dir`, der `port.embed(...)` über die registrierten Bereiche ruft und die Zahl der Bereiche meldet. Und die Vorgabe:

```python
        "--profile", choices=[p.value for p in Profile], default=Profile.FAST
```

mit Kommentar:

```python
        # `fast` on the command line, `full` over MCP: the difference is who
        # waits. Spec 7.3 and 13 — where *you* wait, the reranker's extra
        # 100-500 ms warm (and its model load cold) is not worth it.
```

- [ ] **Schritt 5: Den Vertragstest erweitern**

In `tests/test_qmd_contract.py`, damit die Annahme über den Befehlsnamen nicht Fiktion bleibt:

```python
def test_the_embedding_command_exists() -> None:
    """If qmd renames it, `brain embed` fails loudly here and not in the field."""
    QmdPort().embed(("space",))
```

- [ ] **Schritt 6: Beide Läufe**

Run: `uv run pytest -q && uv run ruff check && uv run ruff format --check && uv run mypy && uv run coverage run -m pytest && uv run coverage report`
Run: `uv run pytest -m contract -q`
Expected: beide PASS, Coverage 100 %

- [ ] **Schritt 7: Commit**

```bash
git add src/brain/search src/brain/cli.py tests/
git commit -m "Add brain embed and make fast the profile you wait for"
```

---

## Aufgabe 6: Die Abnahmepunkte nachfahren, die sich geändert haben

Kein Code. Drei der sechs Punkte der ersten Abnahme haben eine andere Bedeutung als vorher; sie werden erneut gefahren und das Protokoll bekommt einen Nachtrag.

- [ ] **Schritt 1: `unsearched` für die drei Bestände setzen**

In `C:/Users/micro/AppData/Local/brain/areas/space/.brain.toml` ergänzen:

```toml
[index]
include = ["**/*.md"]
unsearched = ["docs/.superpowers/**"]
```

Für `iam-wiki` und `obsidian-ai` nur, wenn sie ein Punktverzeichnis mit Quellen tragen — sonst nicht.

- [ ] **Schritt 2: Neu indizieren und die drei Punkte prüfen**

```bash
uv run brain reindex
uv run brain status
```

Erwartet und im Nachtrag zu belegen:
1. **`status` schweigt über die 61 Dokumente** — sie sind jetzt erklärt statt gemeldet. Meldet es sie weiter, greift `unsearched` nicht.
2. **`iam-wiki` hat 22 Dokumente mehr** — die handgeschriebenen `index.md`. Prüf mit `wc -l` auf dem Register vorher und nachher.
3. **Die Trefferliste ist nicht mehr entdoppelt** — und `status` meldet stattdessen geteilte Inhaltshashes, falls es welche gibt.

**Und weiterhin gilt:** in keinen der drei Bäume wird geschrieben. Mit `git status --short` in jedem der drei belegen.

- [ ] **Schritt 3: Den Nachtrag schreiben und committen**

Ein Abschnitt „Nachtrag nach der Nacharbeit" ans Ende von `docs/.superpowers/plans/2026-08-20-scheibe-2a-abnahme.md`, mit den Zahlen vorher/nachher und den Ausgaben im Wortlaut. Ein Punkt, der sich **nicht** wie erwartet verhält, ist ein Fund und wird als solcher protokolliert, nicht weggeschliffen.

```bash
git add docs/.superpowers/plans/2026-08-20-scheibe-2a-abnahme.md
git commit -m "Record what the rework changed in the acceptance run"
```

---

## Bekannte Einschränkungen

**Die Ablagefrage bleibt offen.** Specs und Pläne liegen in einem Punktverzeichnis, und die Suchmaschine betritt solche grundsätzlich nicht. `unsearched` erklärt den Zustand, statt ihn zu beheben — das ist richtig, solange der Zustand gewollt ist. Ob er das für **alle** Punktverzeichnisse gilt, ist nicht entschieden: Die Konvention steht in einer globalen CLAUDE.md und gilt über Projektgrenzen hinweg.

**`status` startet einen Prozeß je Bereich.** Die Divergenzmeldung ruft die Suchmaschine einmal je Bereich; bei vier Bereichen ist das spürbar. Ungemessen — 2b mißt es.

**Die Rücknahme der Entdopplung ist keine Aussage über qmds Verhalten.** qmd führt dieselbe Datei weiterhin unter jedem ihrer Pfade auf. Nur ist der Fall in den registrierten Bereichen nicht vorhanden, und unser Register kann ihn nicht ausdrücken. Registriert jemand eine Spiegelung, kommt das Problem zurück — dann aber mit einem Bestand als Beleg.

**Ein skalares `graph.json` bricht `brain status` mit einem Traceback.** `core._graph` prüft `key not in loaded`, bevor feststeht, daß `loaded` ein Container ist; ein Wurzelwert wie `5` oder `null` übersteht `json.loads` und stirbt an einem `TypeError`, den die Fangliste in `main` nicht kennt. Ein `isinstance(loaded, dict)` vor der Schlüsselprüfung schließt es, eine Zeile plus ein Testfall. Bewußt geparkt: Der Fall ist schwer zu erzeugen — ein abgebrochener Schreibvorgang hinterläßt eher ungültiges JSON als ein Skalar —, und das Abschlußreview stuft ihn als nicht merge-blockierend ein. Er gehört in denselben Zug wie die übrige Typprüfung, sobald jemand `_graph` das nächste Mal anfaßt.

**Die Wiederholung bei leerem Ergebnis ist falsch verortet.** Sie ist eine gemessene Eigenart *einer* Suchmaschine (§16.14: vier von rund 120 Abfragen), wohnt aber in `core._ask`, oberhalb einer Naht, die zusagt, daß nichts über ihr die Suchmaschine kennt. Die saubere Form — Wiederholung in der Umsetzung, Meldung „doppelt leer" oben — ist eine Vertragsänderung mit Testfolge: `test_empty_result_is_retried_exactly_once` bindet `len(port.calls) == 2` auf Kernebene, und wandert die Wiederholung nach unten, kann der Kern das nicht mehr wissen. Nach dem Zusammenführen zu entscheiden.

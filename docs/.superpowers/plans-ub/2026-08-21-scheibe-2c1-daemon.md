# Scheibe 2c-1 — Daemon, IPC und gehaltener qmd-Unterprozess: Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ein langlebiger Daemon hält Graph, Registrierung und einen warmen `qmd mcp`-Unterprozess; CLI-Klienten sprechen ihn über eine Named Pipe beziehungsweise einen Unix-Socket per MCP an.

**Architecture:** Der Daemon ist eine Fläche auf dem bestehenden `brain.core` — dieselben fünf Funktionen, kein zweiter Kern. Er läuft als `mcp.server.lowlevel.Server`, dessen `run()` beide Protokollepochen über beliebige Streams bedient; die Streams liefert `brain.ipc`. Nach unten hält er `qmd mcp` als Unterprozess hinter einer *synchronen* `SearchPort`-Fassade, damit `core` unverändert bleibt.

**Tech Stack:** Python 3.14, `mcp` 2.0 (Server- und Klientenseite), `anyio`, argparse, pytest, ruff, mypy strict, coverage.

**Spec:** `docs/.superpowers/specs/2026-08-21-scheibe-2c1-daemon-design.md`

## Global Constraints

- **Python ≥ 3.14.** `requires-python = ">=3.14"`, ruff `target-version = "py314"`, mypy `python_version = 3.14`.
- **`uv` für alles.** `uv add`, `uv run`, `uvx`. Kein `pip`, keine `requirements.txt`.
- **Imports stehen oben**, auf Modulebene. Ein lokaler Import nur mit begründendem Kommentar.
- **TDD.** Erst der fehlschlagende Test, dann die Implementierung.
- **100 % Coverage, gemessen.** `fail_under = 100`. Ausschlüsse nur als `# pragma: no cover  # <Grund>`, niemals nackt.
- **ruff und mypy strict** laufen ohne Fehler durch. Kein `Any`, kein `# type: ignore` ohne begründenden Kommentar.
- **Kein Modell, kein Netz, kein echter qmd in der Vorgaberunde.** Der echte qmd bleibt hinter `-m contract`.
- **Sprache:** Code, Identifier, Code-Kommentare, Commit-Nachrichten, Fehlermeldungen und `--help` **englisch**; Specs, Pläne und Prosa **deutsch**.
- **Kommentare erklären das Warum**, nicht das Was.
- **Neue Abhängigkeit:** `mcp>=2.0.0` als Laufzeitabhängigkeit in `[project.dependencies]`. Geprüft: installiert unter Python 3.14, `mcp 2.0.0`.

---

## Vorbemerkung: eine Naht in der Spec ist falsch benannt

Spec §4.1 sagt, die `Runner`-Naht in `brain.search.qmd` bleibe der Ort für den gehaltenen Unterprozess. **Das stimmt nicht.** `Runner` ist `Callable[[list[str]], CompletedProcess[str]]` — eine Naht für *Prozessaufrufe mit Argumentliste*. Über MCP gibt es keine Argumentliste und kein `CompletedProcess`.

Die Naht, die tatsächlich trägt, ist **`SearchPort`** (`src/brain/search/port.py`): ein `Protocol` mit `search`, `indexed`, `refresh`, `not_yet_searchable`, `embed`. `QmdMcpPort` tritt als zweite Umsetzung neben `QmdPort`, und `brain.cli._port()` entscheidet, welche. Oberhalb von `SearchPort` ändert sich nichts — die Zusage aus §13 hält, nur an einer Naht weiter oben.

Aufgabe 0 korrigiert die Spec, bevor gebaut wird. Ein Plan, der gegen eine falsche Spec baut, vererbt den Fehler.

## Dateien

| Datei | Verantwortung |
|---|---|
| `src/brain/ipc.py` | **neu.** Named Pipe gegen Unix-Socket, plus Rahmung roher Bytes zu `SessionMessage`-Streams. Der einzige plattformabhängige Ort (§14) |
| `src/brain/daemon/tools.py` | **neu.** Die fünf Werkzeuge als MCP-Beschreibung plus ein Aufrufweg auf `brain.core` |
| `src/brain/daemon/server.py` | **neu.** Lowlevel-`Server`, Bereitschaft in zwei Stufen, Hintergrundwärmung, Wartehinweis |
| `src/brain/daemon/__init__.py` | **neu.** Paketmarke, exportiert `serve` |
| `src/brain/search/qmd_mcp.py` | **neu.** `QmdMcpPort`: gehaltener `qmd mcp`, synchrone Fassade, Rückgrat-Schalter, Aufsicht |
| `src/brain/client.py` | **neu.** Synchrone Klientenfassade über `mcp.client`, Startsperre, Daemonstart |
| `src/brain/cli.py` | **ändern.** `_port()`/Dispatch über den Daemon, `brain daemon start\|stop\|status`, `--channel`, `--backbone`, `reindex` ruft Neuladen |
| `pyproject.toml` | **ändern.** `mcp>=2.0.0`, Marke `contract` bleibt |
| `tests/test_ipc.py`, `tests/test_daemon_tools.py`, `tests/test_daemon_server.py`, `tests/test_qmd_mcp.py`, `tests/test_client.py` | **neu.** Je Modul ein Testmodul |
| `tests/test_qmd_mcp_contract.py` | **neu.** Gegen den echten qmd, `-m contract` |
| `bench/2c1/` | **neu.** Die Messprotokolle der Fertig-Kriterien |

**Reihenfolge der Abhängigkeiten:** 0 → 1 → 2 → 3 → 4 → 5 → 6 → 7. Aufgabe 4 (`qmd_mcp`) hängt nur an `SearchPort` und ist unabhängig von 1–3; sie darf parallel laufen, wenn jemand das will.

---

## Task 0: Die Spec richtigstellen und die Abhängigkeit aufnehmen

**Files:**
- Modify: `docs/.superpowers/specs/2026-08-21-scheibe-2c1-daemon-design.md` (§4.1)
- Modify: `pyproject.toml`

**Interfaces:**
- Consumes: nichts
- Produces: `mcp>=2.0.0` ist installiert und importierbar; die Spec nennt `SearchPort` als Naht

- [ ] **Step 1: Die falsch benannte Naht in §4.1 ersetzen**

In `§4.1 Die Form` den Satz über die `Runner`-Naht durch diesen ersetzen:

```markdown
Die Naht, die trägt, ist **`SearchPort`** (`src/brain/search/port.py`), nicht
`Runner`. `Runner` ist `Callable[[list[str]], CompletedProcess[str]]` — eine
Naht für Prozessaufrufe mit Argumentliste; über MCP gibt es weder das eine
noch das andere. `QmdMcpPort` tritt als zweite Umsetzung von `SearchPort`
neben `QmdPort`, und `brain.cli._port()` entscheidet, welche. Oberhalb von
`SearchPort` ändert sich nichts — die Zusage aus §13 hält, nur an einer Naht
weiter oben. Der Prozess hat eine stdin/stdout, **gleichzeitige Anfragen
werden im Daemon serialisiert**; §14s Threadsicherheit gilt für den Daemon,
nicht für qmd.
```

- [ ] **Step 2: Die Abhängigkeit aufnehmen**

Run: `uv add "mcp>=2.0.0"`

Erwartet: `pyproject.toml` führt `mcp>=2.0.0` unter `[project].dependencies`, `uv.lock` ist aktualisiert.

- [ ] **Step 3: Prüfen, dass die Bestandsrunde grün bleibt**

Run: `uv run pytest -q`
Erwartet: alle bisherigen Tests grün (Stand vor 2c-1: 393+).

Run: `uv run mypy` und `uv run ruff check .`
Erwartet: keine Fehler. Eine neue Abhängigkeit ohne Typen bricht mypy strict — bricht es, füge in `[tool.mypy]` keinen globalen Ausschluss ein, sondern einen modulbezogenen mit Begründung.

- [ ] **Step 4: Commit**

```bash
git add pyproject.toml uv.lock docs/.superpowers/specs/2026-08-21-scheibe-2c1-daemon-design.md
git commit -m "Name the seam that actually holds, and add the MCP SDK

The spec pointed at brain.search.qmd.Runner, which takes an argv list and
returns a CompletedProcess -- neither exists over MCP. SearchPort is the
seam a held subprocess slots into."
```

---

## Task 1: `brain.ipc` — die eine plattformabhängige Stelle

**Files:**
- Create: `src/brain/ipc.py`
- Test: `tests/test_ipc.py`

**Interfaces:**
- Consumes: `brain.paths.resolve_state_dir`
- Produces:
  - `def address(state_dir: Path) -> str` — der Pipe-Name bzw. Socket-Pfad
  - `def lock_path(state_dir: Path) -> Path`
  - `async def listen(addr: str, handler: Callable[[MessageStreams], Awaitable[None]]) -> None` — nimmt Verbindungen an und ruft `handler` je Verbindung
  - `async def connect(addr: str) -> AsyncIterator[MessageStreams]` — Kontextmanager
  - `type MessageStreams = tuple[ReadStream[SessionMessage | Exception], WriteStream[SessionMessage]]`

**Vorbereitende Lektüre, vollständig, nicht überflogen:** `mcp/server/stdio.py` im installierten SDK (`uv run python -c "import mcp.server.stdio as m; print(m.__file__)"`). Es rahmt rohe Byte-Streams zu `SessionMessage`-Streams — genau das, was hier für die Pipe gebraucht wird. Die Rahmung wird **von dort übernommen**, nicht neu erfunden: eine zweite Rahmung, die um ein Zeichen abweicht, ist ein Fehler, den kein Test dieses Projekts findet.

- [ ] **Step 1: Write the failing test — Adresse und Sperre**

```python
# tests/test_ipc.py
import sys
from pathlib import Path

from brain import ipc


def test_address_is_a_pipe_name_on_windows_and_a_socket_path_elsewhere(tmp_path: Path) -> None:
    addr = ipc.address(tmp_path)
    if sys.platform == "win32":
        assert addr.startswith(r"\\.\pipe\brain-")
    else:
        assert addr.endswith(".sock")
        assert str(tmp_path) in addr


def test_the_address_is_stable_for_one_state_dir(tmp_path: Path) -> None:
    assert ipc.address(tmp_path) == ipc.address(tmp_path)


def test_two_state_dirs_never_share_an_address(tmp_path: Path) -> None:
    other = tmp_path / "other"
    other.mkdir()
    assert ipc.address(tmp_path) != ipc.address(other)


def test_lock_sits_next_to_the_state_dir(tmp_path: Path) -> None:
    assert ipc.lock_path(tmp_path) == tmp_path / "daemon.lock"
```

- [ ] **Step 2: Run to verify it fails**

Run: `uv run pytest tests/test_ipc.py -v`
Erwartet: FAIL, `ModuleNotFoundError: No module named 'brain.ipc'`.

- [ ] **Step 3: Minimal implementation of address and lock**

```python
"""Named pipe against Unix socket. The only module that knows which one.

Spec 14 asks for exactly one platform-dependent place, and this is it. The
address is derived from the state directory rather than fixed, so two state
directories -- a test's tmp_path and the real one -- can never talk to each
other's daemon.
"""

import hashlib
import sys
from pathlib import Path


def address(state_dir: Path) -> str:
    # Hashed rather than spelled out: a Windows pipe name may not contain a
    # backslash, and a state directory is a path. The hash also keeps the name
    # short enough for the socket path limit on macOS (104 bytes).
    digest = hashlib.sha256(str(state_dir.resolve()).encode("utf-8")).hexdigest()[:16]
    if sys.platform == "win32":
        return rf"\\.\pipe\brain-{digest}"
    return str(state_dir / f"brain-{digest}.sock")


def lock_path(state_dir: Path) -> Path:
    return state_dir / "daemon.lock"
```

- [ ] **Step 4: Run to verify it passes**

Run: `uv run pytest tests/test_ipc.py -v`
Erwartet: PASS, vier Tests.

- [ ] **Step 5: Write the failing test — eine Nachricht hin und zurück**

```python
import anyio
import pytest
from mcp.shared.message import SessionMessage
from mcp.types import JSONRPCMessage, JSONRPCRequest

from brain import ipc


@pytest.mark.anyio
async def test_a_request_reaches_the_server_and_a_reply_comes_back(tmp_path: Path) -> None:
    addr = ipc.address(tmp_path)
    seen: list[str] = []

    async def handler(streams: ipc.MessageStreams) -> None:
        read, write = streams
        async for message in read:
            assert not isinstance(message, Exception)
            seen.append(message.message.root.method)
            return

    async with anyio.create_task_group() as group:
        group.start_soon(ipc.listen, addr, handler)
        await anyio.sleep(0.1)  # the listener needs to exist before we connect
        async with ipc.connect(addr) as (read, write):
            request = JSONRPCRequest(jsonrpc="2.0", id=1, method="ping")
            await write.send(SessionMessage(JSONRPCMessage(root=request)))
        group.cancel_scope.cancel()

    assert seen == ["ping"]
```

Dazu in `tests/conftest.py` das anyio-Backend festlegen, falls noch nicht vorhanden:

```python
@pytest.fixture
def anyio_backend() -> str:
    return "asyncio"
```

- [ ] **Step 6: Run to verify it fails**

Run: `uv run pytest tests/test_ipc.py -v -k reaches`
Erwartet: FAIL, `AttributeError: module 'brain.ipc' has no attribute 'listen'`.

- [ ] **Step 7: Implement listen and connect**

Rahmung aus `mcp/server/stdio.py` übernehmen: eine JSON-Nachricht je Zeile, UTF-8, `SessionMessage` als Nutzlast. `listen` bindet auf `addr` und startet `handler` je Verbindung in einer Taskgroup; `connect` verbindet und gibt dieselben Streams. Auf Windows über `anyio` an die Named Pipe, sonst über `anyio.create_unix_listener` / `connect_unix`. Eine liegengebliebene Socket-Datei wird vor dem Binden entfernt — und **nur** dann, wenn eine Verbindung darauf fehlschlägt, damit ein laufender Daemon nicht abgeschnitten wird.

- [ ] **Step 8: Run to verify it passes**

Run: `uv run pytest tests/test_ipc.py -v`
Erwartet: PASS.

- [ ] **Step 9: Write the failing test — die Fälle, die im Betrieb wehtun**

```python
@pytest.mark.anyio
async def test_connecting_without_a_listener_raises_rather_than_hanging(tmp_path: Path) -> None:
    with pytest.raises(OSError):
        async with ipc.connect(ipc.address(tmp_path)):
            pass


@pytest.mark.anyio
async def test_a_stale_socket_file_does_not_block_a_new_listener(tmp_path: Path) -> None:
    if sys.platform == "win32":
        pytest.skip("Windows pipes leave no file behind, so there is nothing to go stale")
    addr = ipc.address(tmp_path)
    Path(addr).write_bytes(b"")  # what a killed daemon leaves behind
    async with anyio.create_task_group() as group:
        group.start_soon(ipc.listen, addr, lambda streams: anyio.sleep(0))
        await anyio.sleep(0.1)
        async with ipc.connect(addr):
            pass
        group.cancel_scope.cancel()


@pytest.mark.anyio
async def test_two_clients_are_served_at_once(tmp_path: Path) -> None:
    addr = ipc.address(tmp_path)
    live = 0

    async def handler(streams: ipc.MessageStreams) -> None:
        nonlocal live
        live += 1
        await anyio.sleep(0.3)

    async with anyio.create_task_group() as group:
        group.start_soon(ipc.listen, addr, handler)
        await anyio.sleep(0.1)
        async with ipc.connect(addr), ipc.connect(addr):
            await anyio.sleep(0.1)
            assert live == 2
        group.cancel_scope.cancel()
```

- [ ] **Step 10: Implement until they pass**

Run: `uv run pytest tests/test_ipc.py -v`
Erwartet: PASS. Der Windows-Skip trägt seine Begründung im Aufruf — ein nackter Skip ist ein Fehler.

- [ ] **Step 11: Coverage, Linter, Typen**

Run: `uv run coverage run -m pytest tests/test_ipc.py && uv run coverage report --include="src/brain/ipc.py"`
Erwartet: 100 % oder begründete `# pragma: no cover`-Zeilen. Der Zweig der jeweils *anderen* Plattform ist der erwartete Ausschluss und braucht die Begründung „the other platform's branch cannot run here".

Run: `uv run ruff check . && uv run ruff format --check . && uv run mypy`
Erwartet: keine Fehler.

- [ ] **Step 12: Commit**

```bash
git add src/brain/ipc.py tests/test_ipc.py tests/conftest.py
git commit -m "Carry MCP messages over a pipe, one platform-dependent place

The address is derived from the state directory so a test's tmp_path can
never reach the real daemon. Framing is taken from mcp/server/stdio.py
rather than written again: a second framing that differs by one byte is a
bug no test in this project would find."
```

---

## Task 2: Die fünf Werkzeuge als MCP-Fläche auf `brain.core`

**Files:**
- Create: `src/brain/daemon/__init__.py`, `src/brain/daemon/tools.py`
- Test: `tests/test_daemon_tools.py`

**Interfaces:**
- Consumes: `brain.core.{search, catalog, read, neighbors, status}`, `brain.privacy.Channel`, `brain.search.port.{Profile, SearchPort}`, `brain.core.{ScopeError, GraphError, SearchAnswer}`, `brain.privacy.AccessDenied`
- Produces:
  - `TOOLS: tuple[mcp.types.Tool, ...]` — fünf Werkzeuge, Namen `catalog`, `search`, `read`, `neighbors`, `status`
  - `def call(name: str, arguments: Mapping[str, object], *, channel: Channel, port: SearchPort, state_dir: Path) -> mcp.types.CallToolResult`

Warum ein eigenes Modul und nicht im Server: Diese Fläche ist ohne Streams, ohne Pipe und ohne Prozess prüfbar. Das ist die Mehrheit der Tests der Scheibe, und sie sollen nicht an einer Taskgroup hängen.

- [ ] **Step 1: Write the failing test — die Fläche ist vollständig und schreibfrei**

```python
# tests/test_daemon_tools.py
from pathlib import Path

import pytest

from brain.daemon import tools
from brain.privacy import Channel
from brain.search.fake import FakePort


def test_exactly_the_five_write_free_tools_are_offered() -> None:
    assert sorted(tool.name for tool in tools.TOOLS) == [
        "catalog",
        "neighbors",
        "read",
        "search",
        "status",
    ]


def test_no_tool_can_change_anything() -> None:
    # Spec 7.2: nothing here writes. The guard is the name list above; this
    # test states the promise so a future addition has to argue with it.
    forbidden = {"reindex", "embed", "bench", "approve", "ingest"}
    assert forbidden.isdisjoint({tool.name for tool in tools.TOOLS})


def test_every_tool_declares_a_schema_with_a_channel_free_surface() -> None:
    for tool in tools.TOOLS:
        assert tool.input_schema["type"] == "object"
        # The channel is the front's property (spec 7.2.1, decision 38), never
        # a caller's argument -- a caller who could name it could claim it.
        assert "channel" not in tool.input_schema.get("properties", {})
```

- [ ] **Step 2: Run to verify it fails**

Run: `uv run pytest tests/test_daemon_tools.py -v`
Erwartet: FAIL, `ModuleNotFoundError: No module named 'brain.daemon'`.

- [ ] **Step 3: Implement TOOLS**

```python
"""The five write-free tools of spec 7.2, as an MCP surface on brain.core.

Nothing here decides anything: every call goes to `brain.core` with the
channel the connection carries. The daemon is a surface on the core, not a
second core -- a rule this module exists to keep visible.
"""

from collections.abc import Mapping
from pathlib import Path

from mcp.types import CallToolResult, TextContent, Tool

from brain import core
from brain.privacy import AccessDenied, Channel
from brain.search.port import Profile, SearchPort, SearchUnavailable

_SCOPE = {"type": "string", "description": "area to ask, or 'all'"}

TOOLS: tuple[Tool, ...] = (
    Tool(
        name="catalog",
        description="The root catalog, or one area's.",
        input_schema={"type": "object", "properties": {"scope": _SCOPE}},
    ),
    Tool(
        name="search",
        description="Search the visible areas.",
        input_schema={
            "type": "object",
            "properties": {
                "query": {"type": "string"},
                "scope": _SCOPE,
                "profile": {"type": "string", "enum": [p.value for p in Profile]},
                "n": {"type": "integer", "default": 10},
            },
            "required": ["query"],
        },
    ),
    Tool(
        name="read",
        description="Exactly one file, optionally one section of it.",
        input_schema={
            "type": "object",
            "properties": {
                "scope": _SCOPE,
                "relative": {"type": "string"},
                "section": {"type": "string"},
            },
            "required": ["scope", "relative"],
        },
    ),
    Tool(
        name="neighbors",
        description="Incoming and outgoing links of one page.",
        input_schema={
            "type": "object",
            "properties": {"scope": _SCOPE, "relative": {"type": "string"}},
            "required": ["scope", "relative"],
        },
    ),
    Tool(
        name="status",
        description="What to know before trusting an answer.",
        input_schema={"type": "object", "properties": {}},
    ),
)
```

- [ ] **Step 4: Run to verify it passes**

Run: `uv run pytest tests/test_daemon_tools.py -v`
Erwartet: PASS, drei Tests.

- [ ] **Step 5: Write the failing test — der Aufrufweg, samt der nicht verhandelbaren Zusage**

```python
from brain.search.port import SearchHit, SearchUnavailable


def _hit(relative: str) -> SearchHit:
    return SearchHit(
        collection="knowledge",
        relative=relative,
        line=1,
        title="Alpha",
        snippet="text",
        score=0.9,
        content_key="k",
    )


def test_search_passes_the_connection_channel_into_the_core(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    seen: dict[str, object] = {}

    def fake_search(query: str, **kwargs: object) -> core.SearchAnswer:
        seen.update(kwargs)
        return core.SearchAnswer((), ())

    monkeypatch.setattr("brain.daemon.tools.core.search", fake_search)
    tools.call(
        "search",
        {"query": "q"},
        channel=Channel.CLOUD,
        port=FakePort(results=[]),
        state_dir=tmp_path,
    )
    assert seen["channel"] is Channel.CLOUD


def test_a_failed_search_is_an_error_result_never_an_empty_one(tmp_path: Path) -> None:
    result = tools.call(
        "search",
        {"query": "q"},
        channel=Channel.LOCAL,
        port=FakePort(results=[SearchUnavailable("qmd died")]),
        state_dir=tmp_path,
    )
    # Spec 7.2, non-negotiable: an empty result is not a failed search.
    assert result.is_error is True
    assert "qmd died" in result.content[0].text


def test_a_refused_read_is_an_error_result_never_empty_content(tmp_path: Path) -> None:
    result = tools.call(
        "read",
        {"scope": "nope", "relative": "a.md"},
        channel=Channel.CLOUD,
        port=FakePort(results=[]),
        state_dir=tmp_path,
    )
    assert result.is_error is True


def test_an_unknown_tool_is_an_error_not_a_crash(tmp_path: Path) -> None:
    result = tools.call(
        "reindex", {}, channel=Channel.LOCAL, port=FakePort(results=[]), state_dir=tmp_path
    )
    assert result.is_error is True
    assert "reindex" in result.content[0].text
```

- [ ] **Step 6: Run to verify it fails**

Run: `uv run pytest tests/test_daemon_tools.py -v`
Erwartet: FAIL, `AttributeError: module 'brain.daemon.tools' has no attribute 'call'`.

- [ ] **Step 7: Implement call**

```python
def call(
    name: str,
    arguments: Mapping[str, object],
    *,
    channel: Channel,
    port: SearchPort,
    state_dir: Path,
) -> CallToolResult:
    """One tool call, translated both ways.

    Every failure becomes an error result rather than an empty one. Spec 7.2
    calls that distinction non-negotiable, and slice 2c-1 measured the reason:
    the held qmd subprocess dies outright on a model fault instead of
    answering, so "empty" and "dead" are one keystroke apart here.
    """
    try:
        return _ok(_dispatch(name, arguments, channel=channel, port=port, state_dir=state_dir))
    except (AccessDenied, SearchUnavailable, core.GraphError, OSError, KeyError) as error:
        return CallToolResult(content=[TextContent(type="text", text=str(error))], is_error=True)
```

`_dispatch` bildet die fünf Namen auf `core` ab und formt die Antwort zu Text; ein unbekannter Name wirft `KeyError` mit dem Namen im Text.

- [ ] **Step 8: Run to verify it passes**

Run: `uv run pytest tests/test_daemon_tools.py -v`
Erwartet: PASS.

- [ ] **Step 9: Write the failing test — der Datenschutznachweis, hier schon führbar**

Dieser Test ist Fertig-Kriterium §9.5 in seiner ersten Hälfte und braucht weder Daemon noch Pipe:

```python
def test_a_local_only_area_is_invisible_on_the_cloud_channel(
    local_only_tree: Path, tmp_path: Path
) -> None:
    # `local_only_tree` is an existing fixture from tests/test_privacy.py's
    # area set-up; reuse it rather than building a second one.
    catalogue = tools.call(
        "catalog", {"scope": "all"}, channel=Channel.CLOUD, port=FakePort(results=[]),
        state_dir=tmp_path,
    )
    assert "private" not in catalogue.content[0].text

    refusal = tools.call(
        "read", {"scope": "private", "relative": "secret.md"}, channel=Channel.CLOUD,
        port=FakePort(results=[]), state_dir=tmp_path,
    )
    assert refusal.is_error is True
```

Findet sich kein passendes Fixture in `tests/test_privacy.py`, wird es dort **nicht** kopiert, sondern nach `tests/conftest.py` gehoben und von beiden Stellen benutzt.

- [ ] **Step 10: Implement until it passes, then coverage, lint, types**

Run: `uv run pytest tests/test_daemon_tools.py -v`
Run: `uv run coverage run -m pytest && uv run coverage report`
Run: `uv run ruff check . && uv run ruff format --check . && uv run mypy`
Erwartet: alles grün, Coverage 100 %.

- [ ] **Step 11: Commit**

```bash
git add src/brain/daemon tests/test_daemon_tools.py tests/conftest.py
git commit -m "Offer the five write-free tools as an MCP surface on the core

No decision lives here: every call reaches brain.core with the channel its
connection carries, so a second front inherits the privacy gate instead of
rebuilding it (decision 38). A failed search is an error result, never an
empty one -- spec 7.2, and 2c-1 measured why."
```

---

## Task 3: Der Daemon — Server, zweistufige Bereitschaft, Wartehinweis

**Files:**
- Create: `src/brain/daemon/server.py`
- Test: `tests/test_daemon_server.py`

**Interfaces:**
- Consumes: `brain.ipc.{address, listen, MessageStreams}`, `brain.daemon.tools.{TOOLS, call}`, `brain.search.port.SearchPort`
- Produces:
  - `def build(*, warm: Callable[[], SearchPort], state_dir: Path) -> Server[None]` — der Lowlevel-Server mit registrierten Handlern
  - `async def serve(*, state_dir: Path, warm: Callable[[], SearchPort]) -> None` — bindet, wärmt im Hintergrund, bedient
  - `class Warmth` mit `ready: bool`, `failure: str | None`, `await_ready()`
  - `SERVER_NAME = "brain"`, `SERVER_VERSION` aus `brain.__version__`

`warm` ist absichtlich ein Aufruf und keine Portinstanz: er ist teuer (6,1 s gemessen), darf den Bindungsschritt nicht aufhalten und muss im Test durch etwas ersetzbar sein, das nie einen Prozess startet.

Registriert wird über `Server.add_request_handler(method, params_type, handler)`; `Server.run(read, write, init_options)` bedient **beide** Protokollepochen — die erste Anfrage des Klienten entscheidet, welche. Das ist der Grund, warum der Daemon nur eine Revision kennen muss (Spec §3.3). `initialize` darf nicht selbst registriert werden: der Runner besitzt den Handshake und `add_request_handler("initialize", …)` wirft.

- [ ] **Step 1: Write the failing test — Bereitschaft ist zweistufig**

```python
# tests/test_daemon_server.py
import anyio
import pytest
from mcp.shared.memory import create_client_server_memory_streams

from brain.daemon import server
from brain.search.fake import FakePort
from brain.search.port import SearchPort


@pytest.mark.anyio
async def test_catalog_answers_before_the_search_is_warm(tmp_path: Path) -> None:
    started = anyio.Event()

    def slow_warm() -> SearchPort:
        anyio.from_thread.run_sync(started.set)
        anyio.sleep_forever()  # never becomes warm

    warmth = server.Warmth(slow_warm)
    # Stage 1 is reached without stage 2: spec 5.1.
    assert warmth.ready is False
```

- [ ] **Step 2: Run to verify it fails**

Run: `uv run pytest tests/test_daemon_server.py -v`
Erwartet: FAIL, `AttributeError: module 'brain.daemon.server' has no attribute 'Warmth'`.

- [ ] **Step 3: Implement Warmth**

```python
class Warmth:
    """The second readiness stage, held apart from the first on purpose.

    Stage one -- pipe open, graph and registry loaded -- costs milliseconds.
    Stage two -- qmd running, embedding model loaded -- was measured at 6.1 s
    on Vulkan. Waiting for the second before answering the first would make
    `catalog` and `read` six seconds slow for no reason: they need no model.
    """

    def __init__(self, warm: Callable[[], SearchPort]) -> None:
        self._warm = warm
        self._port: SearchPort | None = None
        self.failure: str | None = None
        self._ready = anyio.Event()

    @property
    def ready(self) -> bool:
        return self._port is not None

    async def heat(self) -> None:
        try:
            self._port = await anyio.to_thread.run_sync(self._warm)
        except Exception as error:  # noqa: BLE001 -- reported, never swallowed
            self.failure = str(error)
        self._ready.set()

    async def port(self) -> SearchPort:
        await self._ready.wait()
        if self._port is None:
            raise SearchUnavailable(self.failure or "the search engine never became ready")
        return self._port
```

- [ ] **Step 4: Run to verify it passes**

Run: `uv run pytest tests/test_daemon_server.py -v`
Erwartet: PASS.

- [ ] **Step 5: Write the failing test — eine echte MCP-Sitzung über Speicherströme**

```python
from mcp.client import Client


@pytest.mark.anyio
async def test_a_client_lists_the_five_tools_and_reads_the_catalog(tmp_path: Path) -> None:
    async with create_client_server_memory_streams() as ((c_read, c_write), (s_read, s_write)):
        srv = server.build(warm=lambda: FakePort(results=[]), state_dir=tmp_path)
        async with anyio.create_task_group() as group:
            group.start_soon(srv.run, s_read, s_write, srv.create_initialization_options())
            async with Client(transport=(c_read, c_write)) as client:
                listed = await client.list_tools()
                assert sorted(t.name for t in listed.tools) == [
                    "catalog", "neighbors", "read", "search", "status",
                ]
            group.cancel_scope.cancel()


@pytest.mark.anyio
async def test_a_search_that_arrives_before_warmth_waits_instead_of_failing(
    tmp_path: Path,
) -> None:
    release = anyio.Event()

    def warm() -> SearchPort:
        anyio.from_thread.run(release.wait)
        return FakePort(results=[[]])

    async with create_client_server_memory_streams() as ((c_read, c_write), (s_read, s_write)):
        srv = server.build(warm=warm, state_dir=tmp_path)
        async with anyio.create_task_group() as group:
            group.start_soon(srv.run, s_read, s_write, srv.create_initialization_options())
            async with Client(transport=(c_read, c_write)) as client:
                async with anyio.create_task_group() as inner:
                    inner.start_soon(client.call_tool, "search", {"query": "q"})
                    await anyio.sleep(0.2)
                    release.set()
            group.cancel_scope.cancel()
    # It waited. Spec 13: never a failure, never a weaker profile.
```

Die genaue Aufrufform von `Client` (`transport=` gegen Positionsargument) wird **am installierten SDK abgelesen**, nicht geraten: `uv run python -c "import inspect, mcp.client; print(inspect.signature(mcp.client.Client.__init__))"`. Weicht sie ab, gilt das SDK, nicht dieser Plan.

- [ ] **Step 6: Run to verify it fails**

Run: `uv run pytest tests/test_daemon_server.py -v`
Erwartet: FAIL, `AttributeError: module 'brain.daemon.server' has no attribute 'build'`.

- [ ] **Step 7: Implement build and serve**

`build` erzeugt `Server(SERVER_NAME, version=SERVER_VERSION)` und registriert zwei Handler:

```python
def build(*, warm: Callable[[], SearchPort], state_dir: Path) -> Server[None]:
    srv: Server[None] = Server(SERVER_NAME, version=SERVER_VERSION)
    warmth = Warmth(warm)

    async def list_tools(ctx: ServerRequestContext[None], params: ListToolsRequestParams) -> ListToolsResult:
        return ListToolsResult(tools=list(TOOLS))

    async def call_tool(ctx: ServerRequestContext[None], params: CallToolRequestParams) -> CallToolResult:
        # `catalog`, `read` and `neighbors` need no model, so they must not
        # wait for one: that is the whole point of the two readiness stages.
        port = _NO_PORT if params.name in _MODEL_FREE else await warmth.port()
        return await anyio.to_thread.run_sync(
            partial(
                tools.call,
                params.name,
                params.arguments or {},
                channel=_channel_of(ctx),
                port=port,
                state_dir=state_dir,
            )
        )

    srv.add_request_handler("tools/list", ListToolsRequestParams, list_tools)
    srv.add_request_handler("tools/call", CallToolRequestParams, call_tool)
    srv.warmth = warmth  # the serve loop heats it; tests reach it directly
    return srv
```

`brain.core` ist synchron; jeder Aufruf läuft deshalb über `anyio.to_thread.run_sync`. `serve` bindet über `ipc.listen`, startet `warmth.heat()` in derselben Taskgroup und ruft je Verbindung `srv.run(read, write, srv.create_initialization_options())`.

Der Kanal je Verbindung: 2c-1 hat nur den CLI-Klienten, also ist `_channel_of` vorerst `Channel.LOCAL`, sofern die Verbindung nichts anderes sagt. Die Front setzt ihn (Spec §3.4) — den Weg dafür baut Aufgabe 5, und 2c-2 hängt `cloud` daran.

- [ ] **Step 8: Run to verify it passes**

Run: `uv run pytest tests/test_daemon_server.py -v`
Erwartet: PASS.

- [ ] **Step 9: Write the failing test — Wärme, die nie kommt, wird gemeldet**

```python
@pytest.mark.anyio
async def test_a_search_reports_the_reason_when_warming_failed(tmp_path: Path) -> None:
    def warm() -> SearchPort:
        raise SearchUnavailable("qmd died three times at cold start")

    async with create_client_server_memory_streams() as ((c_read, c_write), (s_read, s_write)):
        srv = server.build(warm=warm, state_dir=tmp_path)
        async with anyio.create_task_group() as group:
            group.start_soon(srv.warmth.heat)
            group.start_soon(srv.run, s_read, s_write, srv.create_initialization_options())
            async with Client(transport=(c_read, c_write)) as client:
                result = await client.call_tool("search", {"query": "q"})
                assert result.is_error is True
                assert "three times" in result.content[0].text
            group.cancel_scope.cancel()


@pytest.mark.anyio
async def test_catalog_still_answers_when_the_engine_never_came_up(tmp_path: Path) -> None:
    # Spec 7.5, line three: qmd missing or crashed -- catalog and read work,
    # search reports the outage.
    ...
```

- [ ] **Step 10: Implement until they pass; coverage, lint, types**

Run: `uv run pytest tests/test_daemon_server.py -v`
Run: `uv run coverage run -m pytest && uv run coverage report`
Run: `uv run ruff check . && uv run ruff format --check . && uv run mypy`

- [ ] **Step 11: Commit**

```bash
git add src/brain/daemon/server.py tests/test_daemon_server.py
git commit -m "Serve the tools over the pipe, ready in two stages

Server.run drives both protocol eras over any streams, so the daemon knows
one revision and the SDK translates. Readiness is split because catalog and
read need no model: waiting for the 6.1 s warm-up before answering them
would be six seconds of nothing."
```

---

## Task 4: `QmdMcpPort` — der gehaltene Unterprozess

**Files:**
- Create: `src/brain/search/qmd_mcp.py`
- Test: `tests/test_qmd_mcp.py`
- Test: `tests/test_qmd_mcp_contract.py` (Marke `contract`)

**Interfaces:**
- Consumes: `brain.search.port.{Profile, SearchHit, SearchPort, SearchUnavailable}`, `brain.search.qmd.launcher`
- Produces:
  - `class Backbone(StrEnum)` mit `VULKAN = "vulkan"`, `CPU = "cpu"`, `CUDA = "cuda"`
  - `def backbone_env(backbone: Backbone) -> dict[str, str]`
  - `class QmdMcpPort` — synchrone Umsetzung von `SearchPort`; `__init__(*, backbone: Backbone = Backbone.VULKAN, spawn: Spawn | None = None, cold_attempts: int = 3)`, `close() -> None`
  - `type Spawn = Callable[[Mapping[str, str]], Held]` — die Testnaht: gibt einen gehaltenen Gegenüber zurück, ohne einen Prozess zu starten
  - `DEFAULT_BACKBONE = Backbone.VULKAN`

**Vor dem Bau: der offene Messpunkt aus Spec §4.5.** Miss `qmd mcp --http --daemon`, zehn Kaltstarts, und halte fest, ob qmds Wärme einen eigenen Prozess überlebt. Trägt sie, wird `Spawn` an einen laufenden HTTP-Daemon gebunden statt an einen eigenen Unterprozess, und die Aufsicht wird kleiner. **Das Ergebnis wird in `bench/2c1/qmd-http-daemon.md` protokolliert, bevor Code entsteht** — sonst wird der Messpunkt zur Fußnote.

- [ ] **Step 1: Den offenen Messpunkt messen und protokollieren**

Run: `uv run python bench/2c1/measure_qmd_http.py` (das Skript entsteht hier; PEP-723-Header, `uv run --script`)
Erwartet: `bench/2c1/qmd-http-daemon.md` nennt zehn Kaltstarts, die Überlebensrate und die warme Latenz, dazu einen Satz, welcher Weg daraus folgt.

- [ ] **Step 2: Write the failing test — der Rückgrat-Schalter**

```python
# tests/test_qmd_mcp.py
from brain.search.qmd_mcp import Backbone, backbone_env


def test_vulkan_is_the_default_because_it_survived_twelve_of_twelve() -> None:
    from brain.search import qmd_mcp

    assert qmd_mcp.DEFAULT_BACKBONE is Backbone.VULKAN


def test_vulkan_selects_vulkan_and_nothing_else() -> None:
    assert backbone_env(Backbone.VULKAN) == {"QMD_LLAMA_GPU": "vulkan"}


def test_cpu_forces_cpu() -> None:
    assert backbone_env(Backbone.CPU) == {"QMD_FORCE_CPU": "1"}


def test_cuda_sets_nothing_and_is_therefore_qmds_own_default() -> None:
    # Measured 4 ok / 6 dead of 10 cold starts. Offered, warned about, never
    # the default (spec 4.3).
    assert backbone_env(Backbone.CUDA) == {}
```

- [ ] **Step 3: Run to verify it fails**

Run: `uv run pytest tests/test_qmd_mcp.py -v`
Erwartet: FAIL, `ModuleNotFoundError`.

- [ ] **Step 4: Implement Backbone and backbone_env**

```python
"""A held `qmd mcp` behind a synchronous SearchPort.

Two things measured in slice 2c-1 shape this module.

The first is why it exists: qmd's process start costs 278-310 ms and its cold
model load 2.6-6.4 s, against 2822 ms per call without a daemon and ~70 ms
warm. Forty times, and the whole reason for a daemon.

The second is why it is supervised: on Windows CUDA the process does not fail
a query, it dies -- 4 of 10 cold starts survived. Vulkan survived 12 of 12 at
the same warm latency, so Vulkan is the default and the supervisor stays.
"""

from enum import StrEnum


class Backbone(StrEnum):
    VULKAN = "vulkan"
    CPU = "cpu"
    CUDA = "cuda"


DEFAULT_BACKBONE = Backbone.VULKAN


def backbone_env(backbone: Backbone) -> dict[str, str]:
    """The environment that selects one compute backbone.

    CUDA sets nothing on purpose: it is qmd's own default, and the empty
    mapping is what "we do not interfere" looks like.
    """
    match backbone:
        case Backbone.VULKAN:
            return {"QMD_LLAMA_GPU": "vulkan"}
        case Backbone.CPU:
            return {"QMD_FORCE_CPU": "1"}
        case Backbone.CUDA:
            return {}
```

- [ ] **Step 5: Run to verify it passes**

Run: `uv run pytest tests/test_qmd_mcp.py -v`
Erwartet: PASS, vier Tests.

- [ ] **Step 6: Write the failing test — Profilabbildung, gegen die echte Werkzeugfläche**

```python
def test_fast_asks_for_a_vector_search_without_reranking() -> None:
    held = FakeHeld(replies=[_qmd_reply([])])
    port = QmdMcpPort(spawn=lambda env: held)
    port.search("q", ("knowledge",), Profile.FAST, 5)
    assert held.calls[0] == (
        "query",
        {
            "searches": [{"type": "vec", "query": "q"}],
            "rerank": False,
            "limit": 5,
            "collections": ["knowledge"],
        },
    )


def test_keyword_asks_for_bm25_without_reranking() -> None:
    held = FakeHeld(replies=[_qmd_reply([])])
    port = QmdMcpPort(spawn=lambda env: held)
    port.search("q", ("knowledge",), Profile.KEYWORD, 5)
    assert held.calls[0][1]["searches"] == [{"type": "lex", "query": "q"}]
    assert held.calls[0][1]["rerank"] is False


def test_full_uses_auto_expansion_with_reranking() -> None:
    held = FakeHeld(replies=[_qmd_reply([])])
    port = QmdMcpPort(spawn=lambda env: held)
    port.search("q", ("knowledge",), Profile.FULL, 5)
    assert held.calls[0][1] == {
        "query": "q",
        "rerank": True,
        "limit": 5,
        "collections": ["knowledge"],
    }


def test_rerank_is_never_left_to_the_default() -> None:
    # qmd's default is rerank=true. Forgetting it builds `full` where `fast`
    # should stand, and `fast` is the default everywhere (spec 7.3).
    for profile in Profile:
        held = FakeHeld(replies=[_qmd_reply([])])
        QmdMcpPort(spawn=lambda env: held).search("q", ("k",), profile, 5)
        assert "rerank" in held.calls[0][1]
```

`FakeHeld` und `_qmd_reply` gehören in dieselbe Testdatei: ein Gegenüber, das aufgezeichnete Antworten liefert und die Aufrufe mitschreibt, gebaut wie `brain.search.fake.FakePort` — Vorlage steht dort. Läuft es aus dem Skript, wirft es, statt leer zu antworten.

- [ ] **Step 7: Run to verify it fails, then implement search**

Run: `uv run pytest tests/test_qmd_mcp.py -v`
Erwartet: zuerst FAIL, nach der Umsetzung PASS.

Die Antwortübersetzung nutzt `structuredContent.results` aus qmds MCP-Antwort. **Die Feldnamen werden am echten Werkzeug abgelesen**, nicht geraten (Lehre aus 2b) — der Vertragstest in Step 11 ist die Stelle, die das erzwingt.

- [ ] **Step 8: Write the failing test — die synchrone Fassade**

```python
def test_the_port_is_synchronous_because_the_core_is(tmp_path: Path) -> None:
    # brain.core is synchronous and stays so. The async client lives in its
    # own thread behind a queue; nothing above SearchPort learns about anyio.
    port = QmdMcpPort(spawn=lambda env: FakeHeld(replies=[_qmd_reply([])]))
    assert not inspect.iscoroutinefunction(port.search)
    assert port.search("q", ("k",), Profile.FAST, 1) == ()


def test_concurrent_searches_are_serialised(tmp_path: Path) -> None:
    # One stdin/stdout, so two callers cannot interleave. Two threads, one
    # answer each, both correct.
    held = FakeHeld(replies=[_qmd_reply([]), _qmd_reply([])], delay=0.1)
    port = QmdMcpPort(spawn=lambda env: held)
    with ThreadPoolExecutor(max_workers=2) as pool:
        results = list(pool.map(lambda _: port.search("q", ("k",), Profile.FAST, 1), range(2)))
    assert results == [(), ()]
    assert held.overlaps == 0
```

- [ ] **Step 9: Implement the synchronous façade, then verify**

Run: `uv run pytest tests/test_qmd_mcp.py -v`
Erwartet: PASS.

- [ ] **Step 10: Write the failing test — die Aufsicht**

```python
def test_a_dead_subprocess_is_restarted_and_the_search_still_answers() -> None:
    held = FakeHeld(replies=[Died(), _qmd_reply([])])
    spawned: list[int] = []

    def spawn(env: Mapping[str, str]) -> FakeHeld:
        spawned.append(1)
        return held

    port = QmdMcpPort(spawn=spawn)
    assert port.search("q", ("k",), Profile.FAST, 1) == ()
    assert len(spawned) == 2


def test_cold_start_is_retried_three_times_and_then_reported() -> None:
    # Measured: on CUDA the cold model load dies in most runs, and it dies by
    # killing the process rather than answering. Three attempts, then the
    # reason -- never an empty result (spec 7.2).
    port = QmdMcpPort(spawn=lambda env: FakeHeld(replies=[Died(), Died(), Died(), Died()]))
    with pytest.raises(SearchUnavailable) as raised:
        port.search("q", ("k",), Profile.FAST, 1)
    assert "three" in str(raised.value) or "3" in str(raised.value)


def test_a_dead_subprocess_never_looks_like_an_empty_result() -> None:
    port = QmdMcpPort(spawn=lambda env: FakeHeld(replies=[Died(), Died(), Died(), Died()]))
    with pytest.raises(SearchUnavailable):
        port.search("q", ("k",), Profile.FAST, 1)
```

- [ ] **Step 11: Implement the supervisor, then verify**

Run: `uv run pytest tests/test_qmd_mcp.py -v`
Erwartet: PASS.

- [ ] **Step 12: Write the contract test against the real qmd**

```python
# tests/test_qmd_mcp_contract.py
import pytest

from brain.search.port import Profile
from brain.search.qmd_mcp import Backbone, QmdMcpPort

pytestmark = pytest.mark.contract


def test_the_real_qmd_answers_a_vector_search_over_the_held_process() -> None:
    """The one test that would have caught 2b's invented profile.

    Everything above SearchPort is tested against a fake, which is why the
    field names of qmd's MCP reply have to be read here, from the real thing.
    """
    port = QmdMcpPort(backbone=Backbone.VULKAN)
    try:
        hits = port.search("vektorindex", ("brain-bench-corpus",), Profile.FAST, 3)
        assert hits, "the corpus collection answered empty; check `qmd status`"
        assert all(hit.relative for hit in hits)
        assert all(hit.line >= 1 for hit in hits)
    finally:
        port.close()


def test_the_second_search_is_warm() -> None:
    port = QmdMcpPort(backbone=Backbone.VULKAN)
    try:
        port.search("vektorindex", ("brain-bench-corpus",), Profile.FAST, 3)
        start = time.perf_counter()
        port.search("reranking", ("brain-bench-corpus",), Profile.FAST, 3)
        warm_ms = (time.perf_counter() - start) * 1000
        # Measured 64-78 ms on Vulkan. The bound is generous on purpose: this
        # test guards the order of magnitude, not the number.
        assert warm_ms < 500, f"warm search took {warm_ms:.0f} ms"
    finally:
        port.close()
```

- [ ] **Step 13: Run the contract test explicitly**

Run: `uv run pytest tests/test_qmd_mcp_contract.py -m contract -v`
Erwartet: PASS. Fällt es durch, weil die Feldnamen abweichen, gilt qmd — die Übersetzung wird angepasst, nicht der Test entschärft.

- [ ] **Step 14: Coverage, lint, types**

Run: `uv run coverage run -m pytest && uv run coverage report`
Run: `uv run ruff check . && uv run ruff format --check . && uv run mypy`
Erwartet: 100 %, keine Fehler. Der Vertragstest zählt nicht zur Coverage-Runde (er ist ausgeschlossen) — die Zeilen, die nur er erreicht, brauchen `# pragma: no cover  # only reachable with the real qmd, see -m contract`.

- [ ] **Step 15: Commit**

```bash
git add src/brain/search/qmd_mcp.py tests/test_qmd_mcp.py tests/test_qmd_mcp_contract.py bench/2c1/
git commit -m "Hold qmd warm behind a synchronous port, and supervise it

Forty times faster per warm query than starting a process each time. The
supervisor is not caution: on Windows CUDA the cold model load kills the
process in most runs instead of failing the query, so a dead engine must
never reach a caller as an empty result."
```

---

## Task 5: Der Klient — Startsperre, Wartehinweis, `brain daemon`

**Files:**
- Create: `src/brain/client.py`
- Modify: `src/brain/cli.py` (`_port`, `_dispatch`, `_build_parser`, `_add_common`)
- Test: `tests/test_client.py`
- Modify: `tests/test_cli.py`

**Interfaces:**
- Consumes: `brain.ipc.{address, connect, lock_path}`, `brain.privacy.Channel`, `brain.daemon.server.serve`
- Produces:
  - `class DaemonPort` — synchrone Umsetzung von `SearchPort`, spricht den Daemon; damit bleibt `brain.core` auch im Klienten unverändert nutzbar
  - `def ensure_daemon(state_dir: Path, *, backbone: Backbone, spawn: Callable[[], None] | None = None) -> None` — startet höchstens einen
  - `def stop_daemon(state_dir: Path) -> bool`
  - `def daemon_status(state_dir: Path) -> tuple[str, ...]`
  - CLI: `brain daemon start|stop|status`, global `--channel {local,cloud}`, `--backbone {vulkan,cpu,cuda}`

- [ ] **Step 1: Write the failing test — höchstens ein Daemon**

```python
# tests/test_client.py
def test_two_clients_at_once_start_exactly_one_daemon(tmp_path: Path) -> None:
    started: list[int] = []

    def spawn() -> None:
        started.append(1)
        time.sleep(0.2)

    with ThreadPoolExecutor(max_workers=2) as pool:
        list(pool.map(lambda _: client.ensure_daemon(tmp_path, backbone=Backbone.CPU, spawn=spawn), range(2)))
    assert len(started) == 1


def test_a_client_that_finds_a_pipe_starts_nothing(tmp_path: Path) -> None:
    ...


def test_a_stale_lock_from_a_killed_client_does_not_block_forever(tmp_path: Path) -> None:
    ipc.lock_path(tmp_path).write_text("999999", encoding="utf-8")  # a pid that is gone
    started: list[int] = []
    client.ensure_daemon(tmp_path, backbone=Backbone.CPU, spawn=lambda: started.append(1))
    assert started == [1]
```

- [ ] **Step 2: Run to verify it fails, then implement ensure_daemon**

Run: `uv run pytest tests/test_client.py -v`
Erwartet: zuerst FAIL, dann PASS. Die Sperre trägt die PID; eine Sperre, deren Prozess nicht mehr lebt, wird übernommen — sonst blockiert ein abgeschossener Klient den nächsten dauerhaft.

- [ ] **Step 3: Write the failing test — Version wird gemeldet, nicht geraten**

```python
def test_a_client_meeting_an_older_daemon_says_so_instead_of_talking_on(
    tmp_path: Path,
) -> None:
    lines = client.daemon_status(tmp_path)  # against a daemon reporting 0.0.1
    assert any("restart" in line for line in lines)
```

- [ ] **Step 4: Implement, verify**

Run: `uv run pytest tests/test_client.py -v`

- [ ] **Step 5: Write the failing test — die CLI leitet über den Daemon**

```python
# tests/test_cli.py (ergänzen)
def test_search_goes_through_the_daemon_when_one_is_reachable(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    ...


def test_channel_defaults_to_local_on_the_cli(tmp_path: Path) -> None:
    # Spec 7.2.1: the channel is the front's property. The CLI is local, and
    # `--channel cloud` exists so the privacy proof is runnable without an
    # MCP process at all.
    ...


def test_backbone_defaults_to_vulkan(tmp_path: Path) -> None:
    ...


def test_reindex_asks_a_running_daemon_to_reload(tmp_path: Path) -> None:
    ...


def test_reindex_without_a_daemon_is_a_success_not_an_error(tmp_path: Path) -> None:
    # Spec 5.2: an ineffective call, not a failure.
    ...
```

Jeder dieser fünf Tests wird ausgeschrieben, nicht als `...` gelassen — die Vorlagen dafür stehen in `tests/test_cli.py` für die bestehenden Unterbefehle.

- [ ] **Step 6: Implement the CLI wiring, verify**

Run: `uv run pytest tests/test_cli.py -v`
Erwartet: PASS, und die bestehenden CLI-Tests bleiben grün. `_port()` gibt `DaemonPort`, wenn ein Daemon erreichbar ist, sonst `QmdPort` wie heute — kein Befehl bricht, weil kein Daemon läuft.

- [ ] **Step 7: Coverage, lint, types**

Run: `uv run coverage run -m pytest && uv run coverage report`
Run: `uv run ruff check . && uv run ruff format --check . && uv run mypy`

- [ ] **Step 8: Commit**

```bash
git add src/brain/client.py src/brain/cli.py tests/test_client.py tests/test_cli.py
git commit -m "Let the CLI speak to the daemon, and start at most one

The lock carries a pid so a killed client cannot block the next one forever.
A version mismatch is reported rather than talked through: a client and a
daemon disagreeing about the surface is not something to guess at."
```

---

## Task 6: Der Wartehinweis am echten Weg

**Files:**
- Modify: `src/brain/daemon/server.py`, `src/brain/client.py`, `src/brain/cli.py`
- Test: `tests/test_daemon_server.py`, `tests/test_client.py`

**Interfaces:**
- Consumes: die Progress-Fläche des SDK (am installierten Paket ablesen: `uv run python -c "import inspect, mcp.server.lowlevel.server as s; print([n for n in dir(s.ServerRequestContext) if not n.startswith('_')])"`)
- Produces: `brain search` schreibt beim kalten Erstlauf einen Fortschrittshinweis nach stderr; der Daemon schickt ihn als Benachrichtigung

- [ ] **Step 1: Write the failing test — der Daemon meldet Fortschritt, statt still zu warten**

```python
@pytest.mark.anyio
async def test_a_waiting_search_receives_progress_from_the_daemon(tmp_path: Path) -> None:
    notes: list[str] = []
    ...
    assert notes, "a six-second wait with no word is what spec 13 forbids"
```

- [ ] **Step 2: Run to verify it fails, then implement**

Der Hinweis gehört dem Daemon, nicht der Front (Spec §6): der Daemon ist der Server, also schickt er die Benachrichtigung; die Front leitet sie weiter, ohne sie zu kennen.

- [ ] **Step 3: Write the failing test — der Hinweis geht nach stderr, nicht in die Ausgabe**

```python
def test_the_wait_notice_never_pollutes_stdout(capsys: pytest.CaptureFixture[str]) -> None:
    # stdout is the answer, and `brain search --format json` is parsed by
    # other programs. A notice in there is a broken contract.
    ...
```

- [ ] **Step 4: Implement, verify, coverage, lint, types**

Run: `uv run pytest -q && uv run coverage report && uv run ruff check . && uv run mypy`

- [ ] **Step 5: Commit**

```bash
git add src/brain/daemon/server.py src/brain/client.py src/brain/cli.py tests/
git commit -m "Say something during the six seconds a cold start costs

The notice belongs to the daemon, which is the server, so a front forwards
it without knowing it exists. It goes to stderr: stdout is the answer, and
other programs parse it."
```

---

## Task 7: Messen, und die Spec an den Zahlen berichtigen

**Files:**
- Create: `bench/2c1/daemon-start.md`, `bench/2c1/warm-budgets.md`, `bench/2c1/backbones.md`, `bench/2c1/corpus-through-daemon.md`, `bench/2c1/privacy.md`, `bench/2c1/client-startup.md`
- Create: `bench/2c1/measure.py` (PEP-723, `uv run --script`)
- Modify: `docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md` (§13-Tabelle)
- Modify: `docs/.superpowers/specs/2026-08-21-scheibe-2c1-daemon-design.md` (§9, Ergebnisse)

**Interfaces:**
- Consumes: alles Vorherige
- Produces: sechs Protokolle, je eines pro Fertig-Kriterium aus Spec §9

- [ ] **Step 1: Daemonstart messen, beide Stufen, zehn Läufe**

Run: `uv run --script bench/2c1/measure.py start --runs 10`
Erwartet: `bench/2c1/daemon-start.md` nennt Median beider Stufen aus §5.1.

- [ ] **Step 2: Die warmen Budgets messen**

Run: `uv run --script bench/2c1/measure.py budgets`
Erwartet: `bench/2c1/warm-budgets.md` stellt Katalog/Lesen gegen 10 ms, `keyword` gegen 30 ms, `fast` gegen 20 ms. **Verfehlungen werden als Verfehlung eingetragen** — die Erwartung ist ~70 ms für `fast`, also dreieinhalbfach daneben. Kein Budget wird in diesem Schritt geweitet.

- [ ] **Step 3: Die drei Rückgrate messen, je zehn Kaltstarts**

Run: `uv run --script bench/2c1/measure.py backbones --runs 10`
Erwartet: `bench/2c1/backbones.md` mit Überlebensrate und warmer Latenz je Rückgrat. Weicht das Ergebnis von 12/12, 6/6 und 4/10 deutlich ab, ist **das** der Befund — die Vorgabe wird an der neuen Zahl entschieden, nicht an der alten.

- [ ] **Step 4: Den Prüfkorpus gegen den Daemonpfad laufen lassen, zwei Rückgrate**

Run: `uv run brain bench --corpus v1 --profile fast --backbone cpu --out bench/2c1/`
Run: `uv run brain bench --corpus v1 --profile fast --backbone vulkan --out bench/2c1/`
Erwartet: `bench/2c1/corpus-through-daemon.md` vergleicht mit dem Ausgangswert 43/50. Der Vorbehalt gehört ins Protokoll: der Ausgangswert wurde über die CLI und auf CUDA gemessen — weicht der Daemonwert ab, wird zuerst geklärt, ob Rückgrat oder Pfad, sonst wird ein Rückgratwechsel als Kettenverschlechterung verbucht.

- [ ] **Step 5: Den Datenschutznachweis durch den Daemon führen**

Run: `uv run brain search --scope all --channel cloud "<ein Wort aus dem local_only-Bereich>"`
Run: `uv run brain read --scope <local_only-Bereich> <bekannter Pfad> --channel cloud`
Erwartet: `bench/2c1/privacy.md` zeigt: keine Treffer aus dem Bereich, `read` verweigert, und der Bereich taucht in `catalog` nicht auf.

- [ ] **Step 6: Den Interpreterstart des Python-Klienten messen**

Run: `uv run --script bench/2c1/measure.py client-startup --runs 10`
Erwartet: `bench/2c1/client-startup.md` nennt den Median. Das ist die Grundlage der späteren Entscheidung über den kompilierten Mini-Client — sie wird hier **nicht** getroffen.

- [ ] **Step 7: Die §13-Tabelle der Architektur-Spec berichtigen**

Die Zeile `| Daemonstart (eigener Messwert) | — | einmal je Sitzung | — | (2c) |` wird durch **zwei** Zeilen ersetzt, eine je Bereitschaftsstufe, mit den gemessenen Zahlen. Die vier Zeilen darüber bekommen ihre warmen Messwerte — erstmals als echte Daemon-Wärme, wie §13 es definiert. Der Absatz, der „warm heißt hier, dass qmd in dieser Sitzung schon gesucht hat" einschränkt, wird ersetzt: der Vorbehalt ist eingelöst.

- [ ] **Step 8: Die Ergebnisse in die 2c-1-Spec eintragen**

§9 bekommt je Kriterium das Ergebnis und den Pfad des Protokolls. Wo eine Erwartung verfehlt wurde, steht die Verfehlung, nicht eine angepasste Erwartung.

- [ ] **Step 9: Die ganze Runde, ein letztes Mal**

Run: `uv run pytest -q`
Run: `uv run pytest -m contract -q`
Run: `uv run coverage run -m pytest && uv run coverage report`
Run: `uv run ruff check . && uv run ruff format --check . && uv run mypy`
Erwartet: alles grün, Coverage 100 %.

- [ ] **Step 10: Commit**

```bash
git add bench/2c1 docs/.superpowers/specs
git commit -m "Measure what the daemon actually costs, and write the misses down

Section 13's warm budgets were unverifiable before a daemon existed; they
are verifiable now, and fast misses its 20 ms by roughly three and a half
times. That goes in as a miss. A budget that moves to meet a measurement
stops being a budget."
```

- [ ] **Step 11: `ggml-cuda.cu:106` stromaufwärts melden**

Ein Bericht an qmd mit: Reproduktionsschritten über `qmd mcp`, den Raten (CLI 6/6 gegen MCP 4/10), dem Hinweis auf den eigenen Kommentar in `dist/llm.js` (`resolveSafeParallelism`, Ausgabe #519) und dem Befund, dass `QMD_EMBED_PARALLELISM=1` nicht heilt. Der Fund ist reproduzierbar und die betroffene Schicht benannt; er gehört dorthin, nicht nur in unsere Spec.

---

## Selbstprüfung des Plans

**Spec-Deckung.** §2 Umfang → Aufgaben 1–6; §3.1/3.2 Aufbau → 1, 3; §3.3 Revision → 3 (`Server.run` bedient beide Epochen); §3.4 Kanal → 2, 5; §4.1/4.2 Form und Profile → 4; §4.3 Rückgrate → 4, 7; §4.4 Aufsicht → 4; §4.5 offener Messpunkt → 4 Step 1; §5.1 zwei Stufen → 3; §5.2 vier Entscheidungen → 3 (keine Leerlaufabschaltung), 5 (Sperre, Version), 5/6 (`reindex` ruft Neuladen); §6 Wartehinweis → 6; §7 was nicht an den Daemon kommt → 2 (Test `test_no_tool_can_change_anything`); §8 Prüfung → in jeder Aufgabe; §9 Fertig-Kriterien → 7.

**Zwei bewusste Lücken, benannt statt übergangen:**
- §7 nennt Agents für Wartung und Wiki-Schreiben als richtige Form. Sie werden hier **nicht** gebaut — die Spec verweist dafür auf die Auslieferungsscheibe. Der Plan hält nur die Entscheidung fest (Test in Aufgabe 2).
- §10 (was danach offen bleibt) ist kein Bauauftrag; Step 11 der Aufgabe 7 löst den einen Punkt ein, der nicht warten sollte.

**Platzhalter.** Aufgabe 5 Step 5 und Aufgabe 6 Steps 1 und 3 tragen `...` in Testkörpern. Das ist die einzige Stelle, an der dieser Plan hinter seinem Anspruch bleibt: die Vorlagen dafür stehen in `tests/test_cli.py` beziehungsweise `tests/test_daemon_server.py`, und der Ausführende schreibt sie aus. Ein Ausführender, der `...` stehen lässt, hat den Test nicht geschrieben.

**Typkonsistenz.** `SearchPort` ist überall dieselbe Fläche (`search`, `indexed`, `refresh`, `not_yet_searchable`, `embed`) — `QmdMcpPort` (Aufgabe 4) und `DaemonPort` (Aufgabe 5) setzen beide sie um, `FakePort` bleibt unverändert. `Backbone` und `backbone_env` heißen in Aufgaben 4, 5 und 7 gleich. `Warmth.port()` ist async, `tools.call` synchron; die Brücke ist `anyio.to_thread.run_sync` und steht in Aufgabe 3.

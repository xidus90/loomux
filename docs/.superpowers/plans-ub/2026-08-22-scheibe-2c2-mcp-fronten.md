# Scheibe 2c-2 — MCP-Fronten: Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `brain mcp` als stdio-Front für MCP-Wirte auf den Daemon aus 2c-1 setzen, und der Kanal wird dabei zur Adresse statt zum Argument.

**Architecture:** Der Daemon lauscht auf zwei Adressen statt auf einer — `…-local` und `…-cloud` —, jede an ihren `Channel` gebunden. Der Kanal steht damit in keiner Nachricht, sondern ist die Pipe, auf der die Verbindung ankam. `brain mcp` ist ein dünner Umleiter: stdio nach oben, `PipeTransport` auf die Cloud-Adresse nach unten, und er beantwortet `initialize` und `tools/list` aus eigener Kraft, damit kein Wirt in seinen Start-Timeout läuft.

**Tech Stack:** Python 3.14, `mcp` 2.0 (`mcp.server.lowlevel.Server`, `mcp.client.Client`), anyio, asyncio-Proactor für Named Pipes, pytest, ruff, mypy, coverage.

**Spec:** [`docs/.superpowers/specs/2026-08-22-scheibe-2c2-mcp-fronten-design.md`](../specs/2026-08-22-scheibe-2c2-mcp-fronten-design.md)

## Global Constraints

- **Python >= 3.14.** `requires-python = ">=3.14"`, ruff `target-version = "py314"`, mypy `python_version = "3.14"`. Immer `uv`, nie `pip`.
- **TDD.** Erst der fehlschlagende Test, dann die Implementierung. Jeder Task endet mit einem Commit.
- **100 % Coverage**, gemessen mit `uv run coverage`. `fail_under = 100`. Jede Ausnahme braucht einen begründenden Kommentar: `# pragma: no cover  # <Grund>`.
- **Typen streng.** `mypy --strict` über `src` und `tests`. Kein `Any`, kein `# type: ignore` ohne begründenden Kommentar.
- **Sprache.** Code, Identifier, Code-Kommentare, Commit-Nachrichten auf Englisch. Prosa in `bench/`-Protokollen auf Deutsch.
- **`brain/core.py`, `brain/privacy.py`, `brain/search/*` bleiben unberührt** (Spec §5). Muss eine davon geändert werden, ist die Naht nicht die, für die wir sie halten — anhalten und die Spec berichtigen, nicht umgehen.
- **Zwei Fehlersorten dürfen nie verschmelzen** (Spec §4.5): Werkzeugfehler sind `CallToolResult(is_error=True)`, Transportfehler sind Protokollfehler.
- **Testkommandos:** `uv run pytest`, `uv run ruff check`, `uv run ruff format`, `uv run mypy`, `uv run coverage run -m pytest && uv run coverage report`.

---

## File Structure

| Datei | Verantwortung | Task |
|---|---|---|
| `src/brain/ipc.py` (ändern) | Adresse trägt den Kanal | 1 |
| `src/brain/daemon/server.py` (ändern) | zwei Endpunkte, je ein Kanal | 2 |
| `src/brain/client.py` (ändern) | Kanal wählt die Adresse | 1, 3 |
| `src/brain/cli.py` (ändern) | `DAEMON_CHANNEL` entfällt; `--no-cloud`/`--no-local`; `status` nennt Endpunkte; `brain mcp` als Befehl | 3, 4, 5 |
| `src/brain/mcp.py` (neu) | die Front: stdio nach oben, Pipe nach unten | 5, 6 |
| `tests/test_ipc.py` (ändern) | Adressbildung je Kanal | 1 |
| `tests/test_daemon_server.py` (ändern) | zwei Endpunkte, Kanaltrennung | 2 |
| `tests/test_client.py` (ändern) | Kanal in Klientenaufrufen | 1, 3 |
| `tests/test_cli.py` (ändern) | Endpunktwahl, Flags, Status | 3, 4 |
| `tests/test_mcp_front.py` (neu) | die Front über echtes stdio | 5, 6 |
| `bench/2c2/*.md` (neu) | die Messprotokolle | 7 |

---

## Task 1: Die Adresse trägt den Kanal

Reine Erweiterung der Adressbildung samt aller Aufrufer. Das Verhalten ändert sich in diesem Task **nicht**: alles läuft weiter auf `Channel.LOCAL`. Der Task ist fertig, wenn zwei Kanäle nachweislich zwei Adressen ergeben und die Suite unverändert grün ist.

**Files:**
- Modify: `src/brain/ipc.py:39-47` (`address`)
- Modify: `src/brain/client.py` (`reachable`, `version`, `call`, `ensure_daemon`, `daemon_status`, `_await_daemon`)
- Test: `tests/test_ipc.py`, `tests/test_client.py`

**Interfaces:**
- Consumes: `brain.privacy.Channel` (`Channel.LOCAL`, `Channel.CLOUD`, jeweils mit `.value` `"local"`/`"cloud"`)
- Produces:
  - `ipc.address(state_dir: Path, channel: Channel) -> str`
  - `client.reachable(state_dir: Path, channel: Channel = Channel.LOCAL) -> bool`
  - `client.version(state_dir: Path, channel: Channel = Channel.LOCAL) -> str`
  - `client.call(state_dir: Path, name: str, arguments: Mapping[str, object], *, channel: Channel = Channel.LOCAL, on_progress: Callable[[str], None] | None = None) -> CallToolResult`
  - `client.ensure_daemon(state_dir: Path, *, spawn: Callable[[], None] | None = None, wait: float = START_TIMEOUT, channels: Sequence[Channel] = (Channel.LOCAL, Channel.CLOUD)) -> None` — erreichbar heißt: **alle** gewünschten Kanäle antworten; ein halb offener Daemon ist kein gestarteter
  - `client.daemon_status(state_dir: Path, channels: Sequence[Channel] = (Channel.LOCAL, Channel.CLOUD)) -> tuple[str, ...]`

**Warum `channels` und nicht `channel` an diesen beiden:** Sie handeln vom Daemon als Ganzem, nicht von einer Verbindung. `reachable`, `version` und `call` sind je Verbindung und nehmen deshalb den Einzahl-Kanal. Diese Grenze wird hier gezogen und nicht in Task 4 nachträglich verschoben.

**Warum ein Vorgabewert:** `Channel.LOCAL` als Default hält Task 1 rein additiv — kein Aufrufer bricht, und der Diff zeigt nur die Adressbildung. Task 3 nimmt der CLI den Default wieder weg, wo er eine Entscheidung verstecken würde.

- [ ] **Step 1: Write the failing tests**

In `tests/test_ipc.py`, unterhalb von `test_two_state_dirs_never_share_an_address`:

```python
def test_the_two_channels_never_share_an_address(tmp_path: Path) -> None:
    assert ipc.address(tmp_path, Channel.LOCAL) != ipc.address(tmp_path, Channel.CLOUD)


def test_the_channel_is_readable_in_the_address(tmp_path: Path) -> None:
    """The digest carries the separation; the plain suffix carries the reading.

    Whoever lists a machine's open pipes should see what is open without
    looking a hash up.
    """
    assert ipc.address(tmp_path, Channel.LOCAL).endswith("-local")
    assert ipc.address(tmp_path, Channel.CLOUD).endswith("-cloud")


def test_the_channel_is_in_the_digest_not_only_in_the_suffix(tmp_path: Path) -> None:
    """Stripping the suffix must not make the two addresses equal.

    A suffix alone would put the separation in a string comparison; in the
    digest it survives a naming slip.
    """
    local = ipc.address(tmp_path, Channel.LOCAL).removesuffix("-local")
    cloud = ipc.address(tmp_path, Channel.CLOUD).removesuffix("-cloud")
    assert local != cloud
```

Den bestehenden Import in `tests/test_ipc.py` ergänzen:

```python
from brain.privacy import Channel
```

Die bestehenden Tests `test_address_is_a_pipe_name_on_windows_and_a_socket_path_elsewhere`, `test_the_address_is_stable_for_one_state_dir` und `test_two_state_dirs_never_share_an_address` rufen `ipc.address(tmp_path)` auf — sie bekommen `Channel.LOCAL` als zweites Argument. Auf der POSIX-Seite prüft der erste Test `addr.endswith(".sock")`; das bleibt so, der Kanal steht dort vor der Endung.

Dazu in `tests/test_client.py` ein Test, der die Trennung auf der Klientenseite festnagelt:

```python
def test_a_client_on_one_channel_does_not_reach_the_other_channels_daemon(state: Path) -> None:
    """No daemon on the cloud address means unreachable, even while the local
    one answers. Two addresses, or the separation is a comment."""
    with _running_daemon(state, channel=Channel.LOCAL):
        assert client.reachable(state, Channel.LOCAL)
        assert not client.reachable(state, Channel.CLOUD)
```

`_running_daemon` gibt es in `tests/test_client.py` noch nicht in dieser Form. Die Datei hält bereits eine Hilfskonstruktion, die einen Daemon in einem Thread startet — sie wird um ein `channel`-Schlüsselwort erweitert, das an `server.serve` durchgereicht wird (dessen Signatur Task 2 liefert). **Bis Task 2 fertig ist, wird dieser eine Test mit `@pytest.mark.skip("waits on task 2: serve takes its endpoints")` markiert** — begründet übersprungen, nicht weggelassen, und Task 2 nimmt die Markierung wieder heraus.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
uv run pytest tests/test_ipc.py -v
```

Erwartet: `TypeError: address() takes 1 positional argument but 2 were given`.

- [ ] **Step 3: Change the address**

`src/brain/ipc.py`, `address` ersetzen:

```python
def address(state_dir: Path, channel: Channel) -> str:
    # Hashed rather than spelled out: a Windows pipe name may not contain a
    # backslash, and a state directory is a path. The hash also keeps the name
    # short enough for the socket path limit on macOS (104 bytes).
    #
    # The channel goes into the digest as well as into the readable suffix. The
    # digest is what actually separates the two: a channel a client can name is
    # a claim, so the gate has to be the address itself (spec 3.2), and a
    # separation that lives only in a suffix would rest on string handling.
    seed = f"{state_dir.resolve()}\0{channel.value}"
    digest = hashlib.sha256(seed.encode("utf-8")).hexdigest()[:16]
    if sys.platform == "win32":
        return rf"\\.\pipe\brain-{digest}-{channel.value}"
    # pragma below: the other platform's branch cannot run on this one
    return str(state_dir / f"brain-{digest}-{channel.value}.sock")  # pragma: no cover
```

Import ergänzen:

```python
from brain.privacy import Channel
```

**Achtung auf einen Zyklus:** `brain.privacy` darf `brain.ipc` nicht importieren. Prüfen mit `uv run python -c "import brain.ipc"`; schlägt es fehl, ist die Richtung falsch herum und der Kanal wird stattdessen als `str` übergeben — dann aber mit einem Kommentar, der den Grund nennt.

- [ ] **Step 4: Thread the channel through the client**

`src/brain/client.py`, die fünf Stellen, an denen `ipc.address(state_dir)` steht:

```python
def reachable(state_dir: Path, channel: Channel = Channel.LOCAL) -> bool:
    """Is a daemon answering on this state directory's pipe for this channel?"""
    try:
        version(state_dir, channel)
    except OSError:
        return False
    return True


def version(state_dir: Path, channel: Channel = Channel.LOCAL) -> str:
    """The running daemon's version, from the handshake it answers with."""

    async def ask() -> str:
        async with Client(PipeTransport(ipc.address(state_dir, channel))) as connected:
            info = connected.session.server_info
            return "" if info is None else info.version

    return _run(ask)
```

`call` bekommt `channel: Channel = Channel.LOCAL` als Schlüsselwortargument und reicht es an `ipc.address` weiter.

`ensure_daemon`, `_await_daemon` und `daemon_status` handeln vom Daemon als Ganzem und bekommen deshalb `channels: Sequence[Channel] = (Channel.LOCAL, Channel.CLOUD)`:

```python
def ensure_daemon(
    state_dir: Path,
    *,
    spawn: Callable[[], None] | None = None,
    wait: float = START_TIMEOUT,
    channels: Sequence[Channel] = (Channel.LOCAL, Channel.CLOUD),
) -> None:
    """Start a daemon unless one is already running or starting.

    Running means every wanted channel answers: a daemon with one of its two
    doors open is not a started daemon, and treating it as one would leave a
    front waiting on a pipe nobody opens.

    The lock carries the starter's pid, and a lock whose process is gone is
    taken over: a client that was killed mid-start would otherwise block every
    later client for ever, which is worse than the double start it prevents.
    """
    if all(reachable(state_dir, channel) for channel in channels):
        return
    lock = ipc.lock_path(state_dir)
    if not _take(lock):
        _await_daemon(state_dir, wait, channels)
        return
    try:
        (spawn or (lambda: _spawn_daemon(state_dir, channels)))()
        _await_daemon(state_dir, wait, channels)
    finally:
        lock.unlink(missing_ok=True)


def daemon_status(
    state_dir: Path, channels: Sequence[Channel] = (Channel.LOCAL, Channel.CLOUD)
) -> tuple[str, ...]:
    """One line per endpoint, plus a version warning when they disagree.

    The endpoint that is missing is the thing you are looking for when
    something does not answer (spec 3.4), so every wanted one gets a line --
    including the ones that are shut.
    """
    lines: list[str] = []
    running = ""
    for channel in channels:
        addr = ipc.address(state_dir, channel)
        if not reachable(state_dir, channel):
            lines.append(f"no daemon on {addr} ({channel.value}); run `brain daemon start`")
            continue
        running = version(state_dir, channel)
        lines.append(f"daemon {running or 'of unknown version'} on {addr} ({channel.value})")
    if running and running != __version__:
        # Reported rather than worked around: a client and a daemon that
        # disagree about the surface is not something to guess at.
        lines.append(
            f"this client is {__version__} and the daemon is {running}; "
            "restart it with `brain daemon stop && brain daemon start`"
        )
    return tuple(lines)
```

`_spawn_daemon` bekommt die Kanäle ebenfalls, damit der gestartete Prozess dieselben Türen öffnet:

```python
def _spawn_daemon(state_dir: Path, channels: Sequence[Channel]) -> None:
    """Start `brain daemon run` detached, so it outlives the shell that asked."""
    off = [f"--no-{channel.value}" for channel in Channel if channel not in channels]
    subprocess.Popen(
        [sys.executable, "-m", "brain", "daemon", "run", "--state-dir", str(state_dir), *off],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        creationflags=getattr(subprocess, "DETACHED_PROCESS", 0),
    )
```

Die Flags `--no-local`/`--no-cloud` gibt es erst in Task 4. Bis dahin ist `off` bei den Vorgabe-Kanälen leer, und ein Test, der `_spawn_daemon` mit nur einem Kanal aufruft, würde ein noch unbekanntes Flag erzeugen. **In Task 1 wird `_spawn_daemon` deshalb nur mit dem vollen Kanalsatz aufgerufen**; der Test dafür kommt in Task 4 zusammen mit den Flags.

Import ergänzen: `from brain.privacy import Channel`.

In `src/brain/daemon/server.py` steht `ipc.address(state_dir)` in `serve` — vorerst `ipc.address(state_dir, Channel.LOCAL)`. Task 2 ersetzt die Zeile ohnehin.

- [ ] **Step 5: Run the full suite**

```bash
uv run pytest
```

Erwartet: alles grün außer dem einen bewusst übersprungenen Test aus Step 1.

- [ ] **Step 6: Lint, types, coverage**

```bash
uv run ruff format && uv run ruff check && uv run mypy && uv run coverage run -m pytest && uv run coverage report
```

- [ ] **Step 7: Commit**

```bash
git add src/brain/ipc.py src/brain/client.py src/brain/daemon/server.py tests/test_ipc.py tests/test_client.py
git commit -m "Put the channel in the pipe name, and in its digest"
```

---

## Task 2: Der Daemon öffnet zwei Endpunkte

**Files:**
- Modify: `src/brain/daemon/server.py:163-175` (`serve`)
- Test: `tests/test_daemon_server.py`, `tests/test_client.py`

**Interfaces:**
- Consumes: `ipc.address(state_dir, channel)` aus Task 1; `build(*, warm, state_dir, channel)` unverändert
- Produces: `server.serve(*, state_dir: Path, warm: Callable[[], SearchPort], channels: Sequence[Channel] = (Channel.LOCAL, Channel.CLOUD)) -> None`

**Der springende Punkt:** ein Prozess, eine `Warmth`, ein gehaltener qmd — aber **je Kanal ein eigener `Daemon`**, weil `build` den Kanal in seine `call_tool`-Closure einschließt. Die beiden teilen sich die `Warmth`, damit nicht zweimal gewärmt wird.

- [ ] **Step 1: Write the failing test**

In `tests/test_daemon_server.py`:

```python
@pytest.mark.anyio
async def test_the_two_endpoints_answer_under_their_own_channels(tmp_path: Path) -> None:
    """The same daemon, asked over two pipes, gates differently.

    This is the whole of slice 2c-2 in one assertion: the channel is the
    address, so a `local_only` area is invisible on the cloud pipe and
    readable on the local one -- without either client naming a channel.
    """
    state = _state_with_a_local_only_area(tmp_path)

    async with anyio.create_task_group() as group:
        group.start_soon(partial(server.serve, state_dir=state, warm=lambda: FakePort(results=[])))
        await anyio.sleep(0.2)  # both listeners have to exist before we connect

        async with Client(PipeTransport(ipc.address(state, Channel.LOCAL))) as local:
            visible = client.text_of(await local.call_tool("catalog", {"scope": "all"}))
        async with Client(PipeTransport(ipc.address(state, Channel.CLOUD))) as cloud:
            hidden = client.text_of(await cloud.call_tool("catalog", {"scope": "all"}))

        group.cancel_scope.cancel()

    assert "geheim" in visible
    assert "geheim" not in hidden


@pytest.mark.anyio
async def test_serving_no_channel_at_all_is_refused(tmp_path: Path) -> None:
    """A daemon without a door holds memory and answers nobody."""
    with pytest.raises(ValueError, match="at least one channel"):
        await server.serve(state_dir=tmp_path, warm=lambda: FakePort(results=[]), channels=())


@pytest.mark.anyio
async def test_both_endpoints_share_one_warm_up(tmp_path: Path) -> None:
    """Two doors, one engine. A second warm-up would mean a second model in
    VRAM and a second cold start (spec 3.1)."""
    warmed: list[int] = []

    def warm() -> SearchPort:
        warmed.append(1)
        return FakePort(results=[])

    async with anyio.create_task_group() as group:
        group.start_soon(partial(server.serve, state_dir=tmp_path, warm=warm))
        await anyio.sleep(0.2)
        async with Client(PipeTransport(ipc.address(tmp_path, Channel.LOCAL))) as local:
            await local.call_tool("status", {})
        async with Client(PipeTransport(ipc.address(tmp_path, Channel.CLOUD))) as cloud:
            await cloud.call_tool("status", {})
        group.cancel_scope.cancel()

    assert warmed == [1]
```

`_state_with_a_local_only_area` ist eine Hilfsfunktion in derselben Datei. Sie baut ein Zustandsverzeichnis mit zwei Bereichen — `offen` mit Vorgabemodus, `geheim` auf `local_only` — im selben Format, das `tests/conftest.py` für die Registrierung schon verwendet. **Sie wird nicht erfunden:** vor dem Schreiben `grep -n "local_only" tests/` laufen lassen und den vorhandenen Aufbau übernehmen; `tests/test_privacy.py` und `tests/test_core_tools.py` halten ihn.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
uv run pytest tests/test_daemon_server.py -k "endpoints or no_channel or warm_up" -v
```

Erwartet: `TypeError: serve() got an unexpected keyword argument 'channels'` und ein Verbindungsfehler auf der Cloud-Adresse.

- [ ] **Step 3: Serve both**

`src/brain/daemon/server.py`, `serve` ersetzen:

```python
async def serve(
    *,
    state_dir: Path,
    warm: Callable[[], SearchPort],
    channels: Sequence[Channel] = (Channel.LOCAL, Channel.CLOUD),
) -> None:
    """Bind first, warm second, answer throughout (spec 5.1).

    One endpoint per channel, all in one process around one warm engine. A
    second daemon would mean a second model in VRAM and a second ten-second
    cold start for a separation the core's gate already makes (spec 3.1).
    """
    if not channels:
        raise ValueError("a daemon needs at least one channel to serve")

    warmth = Warmth(warm)

    def door(channel: Channel) -> Callable[[ipc.MessageStreams], Awaitable[None]]:
        # One `Daemon` per channel: `build` closes over the channel, so the
        # channel a connection carries is fixed when the door is built, not
        # when a message arrives.
        daemon = build(warm=warm, state_dir=state_dir, channel=channel, warmth=warmth)

        async def handle(streams: ipc.MessageStreams) -> None:
            read, write = streams
            await daemon.run(read, write, daemon.create_initialization_options())

        return handle

    async with anyio.create_task_group() as group:
        group.start_soon(warmth.heat)
        for channel in channels:
            group.start_soon(ipc.listen, ipc.address(state_dir, channel), door(channel))
```

`build` bekommt dafür ein optionales `warmth`-Argument, damit die Türen sich eine teilen:

```python
def build(
    *,
    warm: Callable[[], SearchPort],
    state_dir: Path,
    channel: Channel = Channel.LOCAL,
    warmth: Warmth | None = None,
) -> Daemon:
    """The server, bound to one state directory and one channel.

    The channel belongs to the connection's address, never to a caller's
    argument (spec 3.2, 7.2.1, decision 38).

    `warmth` is passed in when several doors share one engine; on its own,
    `build` makes its own, which is what every test of a single door wants.
    """
    warmth = warmth if warmth is not None else Warmth(warm)
```

Der Rest von `build` bleibt unverändert; am Ende steht weiterhin `daemon = Daemon(warmth, ...)`.

Importe in `server.py` ergänzen: `Sequence` aus `collections.abc`.

- [ ] **Step 4: Un-skip the client test from Task 1**

`tests/test_client.py`: die `@pytest.mark.skip`-Markierung an `test_a_client_on_one_channel_does_not_reach_the_other_channels_daemon` entfernen und die Daemon-Hilfskonstruktion so erweitern, dass sie `channels=(channel,)` an `server.serve` weiterreicht.

- [ ] **Step 5: Run the suite**

```bash
uv run pytest
```

Erwartet: grün, keine übersprungenen Tests mehr.

- [ ] **Step 6: Lint, types, coverage**

```bash
uv run ruff format && uv run ruff check && uv run mypy && uv run coverage run -m pytest && uv run coverage report
```

- [ ] **Step 7: Commit**

```bash
git add src/brain/daemon/server.py tests/test_daemon_server.py tests/test_client.py
git commit -m "Open a door per channel on one warm daemon"
```

---

## Task 3: Der Kanal wählt die Adresse, statt den Daemon zu sperren

Hier fällt 2c-1s Verweigerung. `DAEMON_CHANNEL` verschwindet; `brain search --channel cloud` geht ab jetzt über den Daemon — über dessen Cloud-Pipe.

**Files:**
- Modify: `src/brain/cli.py:508-560` (`DAEMON_CHANNEL`, `_through_daemon`)
- Test: `tests/test_cli.py`

**Interfaces:**
- Consumes: `client.reachable(state_dir, channel)`, `client.call(..., channel=…)` aus Task 1
- Produces: keine neuen öffentlichen Namen; `brain.cli.DAEMON_CHANNEL` ist **entfernt**

- [ ] **Step 1: Write the failing tests**

In `tests/test_cli.py`:

```python
def test_a_cloud_call_goes_to_the_cloud_pipe_not_to_the_local_one(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """The channel picks the address. 2c-1 refused the call instead; that kept
    the gate honest but left the cloud front cold (spec 1)."""
    asked: list[Channel] = []

    def reachable(state_dir: Path, channel: Channel = Channel.LOCAL) -> bool:
        asked.append(channel)
        return False

    monkeypatch.setattr("brain.cli.client.reachable", reachable)
    cli.main(["catalog", "--channel", "cloud", "--state-dir", str(tmp_path)])

    assert asked == [Channel.CLOUD]


def test_the_channel_is_never_sent_as_an_argument(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """It travels as the address, or it is a claim (spec 3.2)."""
    sent: list[Mapping[str, object]] = []

    def call(
        state_dir: Path,
        name: str,
        arguments: Mapping[str, object],
        *,
        channel: Channel = Channel.LOCAL,
        on_progress: Callable[[str], None] | None = None,
    ) -> CallToolResult:
        sent.append(arguments)
        return CallToolResult(content=[TextContent(type="text", text="ok")])

    monkeypatch.setattr("brain.cli.client.reachable", lambda state_dir, channel=None: True)
    monkeypatch.setattr("brain.cli.client.call", call)
    cli.main(["catalog", "--channel", "cloud", "--state-dir", str(tmp_path)])

    assert sent == [{"scope": "all"}]
    assert all("channel" not in arguments for arguments in sent)
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
uv run pytest tests/test_cli.py -k "cloud_pipe or never_sent" -v
```

Erwartet: der erste Test schlägt fehl, weil `asked == []` — `_through_daemon` gibt bei `cloud` heute vor jedem `reachable` auf.

- [ ] **Step 3: Remove the refusal**

`src/brain/cli.py`: die Konstante `DAEMON_CHANNEL` samt Kommentar löschen und `_through_daemon` ersetzen:

```python
def _through_daemon(args: argparse.Namespace, state_dir: Path, channel: Channel) -> int | None:
    """Answer from the daemon on this channel's pipe, or None to answer locally.

    The channel picks the address, and the address is what the daemon gates on
    (spec 3.2). Slice 2c-1 refused a call whose channel was not the daemon's,
    because a channel named in a message would be a claim rather than a gate --
    the refusal was right and the fix is a second door, not a second field.

    When no daemon answers, every command still works from this process; spec
    5.2 calls that an ineffective call, not a failure.
    """
    wanted = _TOOL_ARGUMENTS.get(args.command)
    if wanted is None:
        return None
    if not client.reachable(state_dir, channel):
        return None
    arguments = {
        name: getattr(args, "n" if name == "n" else name)
        for name in wanted
        if getattr(args, name, None) is not None
    }
    # The notice goes to stderr: stdout carries the answer, and other programs
    # parse it -- `brain search --format json` above all.
    result = client.call(
        state_dir,
        args.command,
        arguments,
        channel=channel,
        on_progress=lambda message: print(message, file=sys.stderr),
    )
    text = client.text_of(result)
    if result.is_error:
        print(text, file=sys.stderr)
        return 1
    print(text)
    return 0
```

- [ ] **Step 4: Fix the test that guarded the old behaviour**

`tests/test_cli.py` enthält aus 2c-1 einen Test, der prüft, dass ein Cloud-Aufruf **nicht** über den Daemon geht. `grep -n "DAEMON_CHANNEL\|not.*through_daemon" tests/test_cli.py` findet ihn. Er wird nicht gelöscht, sondern umgedreht: Ein Cloud-Aufruf geht über den Daemon, **und** die Antwort ist trotzdem gefiltert. Genau das prüft der Endzu-Ende-Test aus Task 2 bereits auf der Serverseite; hier bleibt die CLI-Seite: die Adresse ist die Cloud-Adresse.

- [ ] **Step 5: Run the suite**

```bash
uv run pytest
```

- [ ] **Step 6: Lint, types, coverage**

```bash
uv run ruff format && uv run ruff check && uv run mypy && uv run coverage run -m pytest && uv run coverage report
```

- [ ] **Step 7: Commit**

```bash
git add src/brain/cli.py tests/test_cli.py
git commit -m "Route a call by its channel instead of refusing it"
```

---

## Task 4: Endpunkte abschalten, und `status` sagt welche

**Files:**
- Modify: `src/brain/cli.py:428-446` (Parser), `:563-578` (`_daemon`), `:585-596` (`_serve_forever`)
- Modify: `src/brain/client.py` (`daemon_status`)
- Test: `tests/test_cli.py`, `tests/test_client.py`

**Interfaces:**
- Consumes: `server.serve(..., channels=…)` aus Task 2
- Produces: `cli._channels_from(args: argparse.Namespace) -> tuple[Channel, ...]`
- `client.daemon_status`, `client.ensure_daemon` und `client._spawn_daemon` nehmen `channels` **bereits seit Task 1** entgegen; dieser Task füllt es nur noch mit etwas anderem als der Vorgabe.

- [ ] **Step 1: Write the failing tests**

In `tests/test_cli.py`:

```python
def test_no_cloud_leaves_the_cloud_endpoint_closed(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    served: list[tuple[Channel, ...]] = []
    monkeypatch.setattr(
        "brain.cli._serve_forever",
        lambda state_dir, backbone, channels: served.append(tuple(channels)),
    )
    cli.main(["daemon", "run", "--no-cloud", "--state-dir", str(tmp_path)])

    assert served == [(Channel.LOCAL,)]


def test_closing_both_endpoints_is_refused(
    tmp_path: Path, capsys: pytest.CaptureFixture[str]
) -> None:
    """A daemon without a door holds memory and answers nobody."""
    code = cli.main(
        ["daemon", "run", "--no-cloud", "--no-local", "--state-dir", str(tmp_path)]
    )

    assert code == 1
    assert "at least one" in capsys.readouterr().err


def test_status_names_the_endpoints_that_are_open(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    """The endpoint that is missing is the thing you are looking for when
    something does not answer (spec 3.4)."""
    monkeypatch.setattr(
        "brain.cli.client.reachable",
        lambda state_dir, channel=Channel.LOCAL: channel is Channel.LOCAL,
    )
    monkeypatch.setattr("brain.cli.client.version", lambda state_dir, channel: __version__)
    cli.main(["daemon", "status", "--state-dir", str(tmp_path)])

    out = capsys.readouterr().out
    assert "local" in out
    assert "cloud" in out
    assert "no daemon" in out  # the cloud line says so
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
uv run pytest tests/test_cli.py -k "no_cloud or both_endpoints or names_the_endpoints" -v
```

Erwartet: `error: unrecognized arguments: --no-cloud`.

- [ ] **Step 3: Add the flags**

`src/brain/cli.py`, in der Parser-Schleife, im `if name in ("start", "run")`-Block hinter `--backbone`:

```python
            # Both endpoints open by default: Claude Code should work without
            # an extra step, and the core's gate already carries the promise.
            # A closed endpoint is a pipe that does not exist -- not a filter
            # rule somebody could forget (spec 3.4).
            one.add_argument("--no-local", action="store_true")
            one.add_argument("--no-cloud", action="store_true")
```

Dazu die Auswertung:

```python
def _channels_from(args: argparse.Namespace) -> tuple[Channel, ...]:
    chosen = tuple(
        channel
        for channel, off in ((Channel.LOCAL, args.no_local), (Channel.CLOUD, args.no_cloud))
        if not off
    )
    if not chosen:
        raise ValueError("a daemon needs at least one open endpoint; drop --no-local or --no-cloud")
    return chosen
```

`_daemon` erweitern:

```python
def _daemon(args: argparse.Namespace, state_dir: Path) -> int:
    match args.daemon_command:
        case "start":
            try:
                channels = _channels_from(args)
            except ValueError as error:
                print(str(error), file=sys.stderr)
                return 1
            client.ensure_daemon(state_dir, channels=channels)
            _print_lines(client.daemon_status(state_dir, channels), file=sys.stderr)
            return 0
        case "stop":
            stopped = client.stop_daemon(state_dir)
            print("daemon stopped" if stopped else "no daemon was running", file=sys.stderr)
            return 0
        case "status":
            _print_lines(client.daemon_status(state_dir), file=sys.stdout)
            return 0
        case _:  # "run", the only one left: serve here rather than detached.
            try:
                channels = _channels_from(args)
            except ValueError as error:
                print(str(error), file=sys.stderr)
                return 1
            _serve_forever(state_dir, args.backbone, channels)
            return 0
```

`_serve_forever` nimmt die Kanäle entgegen und reicht sie durch:

```python
def _serve_forever(state_dir: Path, backbone: str, channels: Sequence[Channel]) -> None:
    """Serve until killed. Its own function so a test can replace it."""
    client.record_pid(state_dir)
    anyio.run(
        partial(
            serve,
            state_dir=state_dir,
            warm=partial(QmdMcpPort, backbone=Backbone(backbone)),
            channels=tuple(channels),
        )
    )
```

`client.ensure_daemon`, `client.daemon_status` und `client._spawn_daemon` nehmen `channels` seit Task 1 entgegen und brauchen hier keine Änderung mehr. Was sich ändert, ist allein, dass die CLI ihnen jetzt etwas anderes als die Vorgabe übergeben kann.

- [ ] **Step 4: Run the suite**

```bash
uv run pytest
```

Bestehende Tests, die `daemon_status(state)` oder `_serve_forever(state, backbone)` aufrufen, müssen mitgezogen werden. `grep -n "daemon_status\|_serve_forever" tests/` findet sie.

- [ ] **Step 5: Lint, types, coverage**

```bash
uv run ruff format && uv run ruff check && uv run mypy && uv run coverage run -m pytest && uv run coverage report
```

- [ ] **Step 6: Commit**

```bash
git add src/brain/cli.py src/brain/client.py tests/test_cli.py tests/test_client.py
git commit -m "Let a door be closed, and say which ones are open"
```

---

## Task 5: `brain mcp` — der Handshake, ohne auf die Wärme zu warten

**Files:**
- Create: `src/brain/mcp.py`
- Modify: `src/brain/cli.py` (Parser: `mcp`-Befehl; `_dispatch`)
- Test: `tests/test_mcp_front.py` (neu)

**Interfaces:**
- Consumes: `brain.daemon.tools.TOOLS`, `client.PipeTransport`, `ipc.address(state_dir, Channel.CLOUD)`, `client.ensure_daemon`
- Produces:
  - `mcp.front(*, state_dir: Path) -> Server[None]` — der Lowlevel-Server der Front
  - `mcp.run(*, state_dir: Path) -> None` — stdio bedienen, bis der Wirt geht

**Die Zusage dieses Tasks:** `initialize` und `tools/list` antworten, ohne dass ein Daemon läuft, und der Daemonstart läuft nebenher. Zehn Sekunden Stufe 2 (2c-1 §9.1) dürfen den Wirt nicht in seinen Timeout laufen lassen.

- [ ] **Step 1: Write the failing tests**

`tests/test_mcp_front.py`:

```python
"""The front, spoken to the way a host speaks to it: over real stdio.

Nothing here mocks the surface. The test that 2c-1 had green while privacy
broke was one that checked an argument's absence instead of a channel's effect
(spec 6) -- a front tested against a stand-in would be the same mistake.
"""

import anyio
import pytest
from mcp.client import Client
from mcp.types import TextContent

from brain import mcp
from brain.daemon.tools import TOOLS


@pytest.mark.anyio
async def test_the_tool_list_answers_without_a_daemon(tmp_path: Path) -> None:
    """Stage two of a daemon start costs ten seconds cold. A host that waits
    that long for `initialize` writes the server off as dead (spec 4.2)."""
    front = mcp.front(state_dir=tmp_path)
    async with _connected(front) as session:
        listed = await session.list_tools()

    assert {tool.name for tool in listed.tools} == {tool.name for tool in TOOLS}


@pytest.mark.anyio
async def test_the_handshake_does_not_start_a_daemon_in_line(tmp_path: Path) -> None:
    """The daemon is nudged in the background, never inside the handshake."""
    started = anyio.Event()

    def spawn() -> None:
        started.set()

    front = mcp.front(state_dir=tmp_path, spawn=spawn)
    with anyio.fail_after(2):
        async with _connected(front) as session:
            await session.list_tools()


@pytest.mark.anyio
async def test_the_front_carries_no_second_tool_list(tmp_path: Path) -> None:
    """The front's list and the daemon's are one object, not two that agree by
    hand. Two lists drift, and the drift shows up as a host calling a tool the
    daemon does not have."""
    assert mcp.front(state_dir=tmp_path) is not None
    assert "TOOLS" in Path("src/brain/mcp.py").read_text(encoding="utf-8")
```

`_connected` ist eine Hilfsfunktion in derselben Datei: ein `Client` über ein Paar Speicherströme auf den Lowlevel-Server, nach dem Muster, das `tests/test_daemon_server.py` für den Daemon bereits verwendet. **Nicht erfinden:** `grep -n "create_memory_object_stream\|Client(" tests/test_daemon_server.py` zeigt den vorhandenen Aufbau; er wird übernommen.

Der dritte Test prüft eine Quelltexteigenschaft und ist damit grob — er steht trotzdem drin, weil die Zusage „eine Liste, nicht zwei" sonst nirgends festgenagelt ist. Findet sich beim Bau ein besserer Weg (etwa `front(...).list_tools()` gegen `TOOLS` identitätsgleich zu prüfen), ersetzt er diesen Test.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
uv run pytest tests/test_mcp_front.py -v
```

Erwartet: `ModuleNotFoundError: No module named 'brain.mcp'`.

- [ ] **Step 3: Write the front**

`src/brain/mcp.py`:

```python
"""The stdio front for MCP hosts: what Claude Code and Gemini CLI talk to.

A redirector, not a second core. What this module writes is the mapping tool
name to tool name and the passing on of arguments and results. If this layer
does more than that, something is wrong (spec 4.1).

It knows no channel -- the cloud pipe it connects to is its channel (spec 3.2)
-- and it knows no protocol revision: the SDK negotiates upward with the host
and speaks 2026-07-28 downward to the daemon (spec 4.3).
"""

from collections.abc import Awaitable, Callable
from pathlib import Path
from typing import Any

import anyio
from mcp.server import ServerRequestContext
from mcp.server.lowlevel import Server
from mcp.server.stdio import stdio_server
from mcp.types import (
    CallToolRequestParams,
    CallToolResult,
    ListToolsResult,
    PaginatedRequestParams,
)

from brain import __version__, client
from brain.daemon.tools import TOOLS
from brain.privacy import Channel

FRONT_NAME = "brain"

# The front's channel. Not an argument, not a setting: the address it dials.
CHANNEL = Channel.CLOUD

type _Context = ServerRequestContext[None, Any]


def front(*, state_dir: Path, spawn: Callable[[], None] | None = None) -> Server[None]:
    """The front's server. Answers the handshake on its own; calls go onward."""

    async def list_tools(ctx: _Context, params: PaginatedRequestParams | None) -> ListToolsResult:
        # Answered here rather than fetched from the daemon: the descriptions
        # are static, and asking would put a cold start inside the handshake.
        # Same object as the daemon's, so the two lists cannot drift.
        return ListToolsResult(tools=list(TOOLS))

    async def call_tool(ctx: _Context, params: CallToolRequestParams) -> CallToolResult:
        return await _forward(ctx, params, state_dir=state_dir, spawn=spawn)

    return Server(
        FRONT_NAME, version=__version__, on_list_tools=list_tools, on_call_tool=call_tool
    )


async def run(*, state_dir: Path, spawn: Callable[[], None] | None = None) -> None:
    """Serve one host over stdio until it goes away."""
    server = front(state_dir=state_dir, spawn=spawn)
    async with anyio.create_task_group() as group:
        # Nudged in the background: `ensure_daemon` blocks for the ten seconds
        # of a cold stage two, and the host must see a ready server long before
        # that (spec 4.2).
        group.start_soon(_nudge, state_dir, spawn)
        async with stdio_server() as (read, write):
            await server.run(read, write, server.create_initialization_options())
        group.cancel_scope.cancel()


async def _nudge(state_dir: Path, spawn: Callable[[], None] | None) -> None:
    # In a thread: `ensure_daemon` is synchronous and spawns a process, and it
    # would otherwise hold the loop that is meant to be answering the host.
    with anyio.CancelScope(shield=False):
        await anyio.to_thread.run_sync(
            lambda: client.ensure_daemon(state_dir, spawn=spawn, channels=(CHANNEL,))
        )
```

`_forward` kommt in Task 6; für diesen Task genügt eine Fassung, die den Aufruf durchreicht und Fehler noch nicht sortiert:

```python
async def _forward(
    ctx: _Context,
    params: CallToolRequestParams,
    *,
    state_dir: Path,
    spawn: Callable[[], None] | None,
) -> CallToolResult:
    return await anyio.to_thread.run_sync(
        lambda: client.call(state_dir, params.name, params.arguments or {}, channel=CHANNEL)
    )
```

- [ ] **Step 4: Wire it into the CLI**

`src/brain/cli.py`, im Parser:

```python
    mcp_parser = sub.add_parser("mcp", help="serve one MCP host over stdio")
    mcp_parser.add_argument("--state-dir", type=Path, default=None)
```

In `_dispatch`, direkt hinter der `daemon`-Weiche:

```python
    if args.command == "mcp":
        anyio.run(partial(mcp_front.run, state_dir=state_dir))
        return 0
```

Import: `from brain import mcp as mcp_front` — umbenannt, weil `mcp` im selben Modul schon das SDK-Paket ist und ein verdeckter Name hier ein stiller Fehler wäre.

- [ ] **Step 5: Run the suite**

```bash
uv run pytest
```

- [ ] **Step 6: Lint, types, coverage**

```bash
uv run ruff format && uv run ruff check && uv run mypy && uv run coverage run -m pytest && uv run coverage report
```

- [ ] **Step 7: Commit**

```bash
git add src/brain/mcp.py src/brain/cli.py tests/test_mcp_front.py
git commit -m "Answer the handshake before the engine is warm"
```

---

## Task 6: Fortschritt nach oben, zwei Fehlersorten getrennt, Daemontod überlebt

**Files:**
- Modify: `src/brain/mcp.py` (`_forward`)
- Test: `tests/test_mcp_front.py`

**Interfaces:**
- Consumes: `client.call(..., on_progress=…)`, `ctx.session.report_progress(progress, total, message)`
- Produces: keine neuen öffentlichen Namen

- [ ] **Step 1: Write the failing tests**

In `tests/test_mcp_front.py`:

```python
@pytest.mark.anyio
async def test_the_waiting_notice_travels_up_as_progress(tmp_path: Path) -> None:
    """stderr is where no host reads. The notice goes up as MCP progress, to
    the token the host sent along (spec 4.4)."""
    seen: list[str] = []

    async def on_progress(progress: float, total: float | None, message: str | None) -> None:
        if message:
            seen.append(message)

    with _a_daemon_that_reports("warming up") as state:
        front = mcp.front(state_dir=state)
        async with _connected(front) as session:
            await session.call_tool("search", {"query": "x"}, progress_callback=on_progress)

    assert seen == ["warming up"]


@pytest.mark.anyio
async def test_a_tool_error_stays_a_tool_error(tmp_path: Path) -> None:
    """`AccessDenied` is content for the model, not a protocol failure."""
    with _a_daemon_that_refuses("area 'geheim' is local only") as state:
        front = mcp.front(state_dir=state)
        async with _connected(front) as session:
            result = await session.call_tool("read", {"scope": "geheim", "relative": "a.md"})

    assert result.is_error
    assert isinstance(result.content[0], TextContent)
    assert "local only" in result.content[0].text


@pytest.mark.anyio
async def test_a_dead_daemon_is_never_an_empty_result(tmp_path: Path) -> None:
    """Give a crashed daemon back as 'nothing found' and the model learns to
    read outage as absence -- the confusion 16.14 was written against."""
    front = mcp.front(state_dir=tmp_path, spawn=lambda: None)  # nothing ever listens
    async with _connected(front) as session:
        result = await session.call_tool("search", {"query": "x"})

    assert result.is_error
    assert isinstance(result.content[0], TextContent)
    assert result.content[0].text != ""


@pytest.mark.anyio
async def test_the_session_survives_a_daemon_that_died(tmp_path: Path) -> None:
    """A Claude Code session is more expensive than a daemon (spec 4.6)."""
    with _a_daemon_that_dies_after_one_call() as state:
        front = mcp.front(state_dir=state)
        async with _connected(front) as session:
            first = await session.call_tool("status", {})
            second = await session.call_tool("status", {})

            # The session still stands: the front outlives the daemon, so the
            # host does not lose its MCP server to a background restart.
            still_there = await session.list_tools()

    assert not first.is_error
    assert second.is_error  # reported, not retried in silence
    assert {tool.name for tool in still_there.tools} == {tool.name for tool in TOOLS}


@pytest.mark.anyio
async def test_a_closed_cloud_endpoint_says_so_rather_than_says_nothing(tmp_path: Path) -> None:
    """`--no-cloud` and "no daemon at all" need different words: one is fixed
    by restarting, the other by changing a flag (spec 3.4)."""
    with _a_daemon_serving_only(Channel.LOCAL) as state:
        front = mcp.front(state_dir=state)
        async with _connected(front) as session:
            result = await session.call_tool("status", {})

    assert result.is_error
    assert isinstance(result.content[0], TextContent)
    assert "does not serve cloud" in result.content[0].text
```

Die vier Hilfen — `_a_daemon_that_reports`, `_a_daemon_that_refuses`, `_a_daemon_that_dies_after_one_call`, `_a_daemon_serving_only` — bauen jeweils einen echten Daemon über `server.serve` mit einem `FakePort` beziehungsweise einem `warm`, das wirft, und geben das Zustandsverzeichnis zurück. Sie werden in dieser Datei geschrieben, nach dem Muster aus `tests/test_client.py`; **kein Mock der Front selbst.**

- [ ] **Step 2: Run the tests to verify they fail**

```bash
uv run pytest tests/test_mcp_front.py -k "progress or tool_error or dead_daemon or survives" -v
```

- [ ] **Step 3: Sort the two error kinds and forward the progress**

`src/brain/mcp.py`, `_forward` ersetzen:

```python
async def _forward(
    ctx: _Context,
    params: CallToolRequestParams,
    *,
    state_dir: Path,
    spawn: Callable[[], None] | None,
) -> CallToolResult:
    """One call onward to the daemon, with the notice travelling back up.

    Two failure kinds, and they must not merge (spec 4.5): what the core
    refuses is content for the model, what the transport loses is an outage.
    Reporting an outage as an empty result would teach the model to read a dead
    daemon as 'nothing found'.
    """
    notices: list[str] = []

    def collect(message: str) -> None:
        notices.append(message)

    try:
        result = await anyio.to_thread.run_sync(
            lambda: client.call(
                state_dir, params.name, params.arguments or {}, channel=CHANNEL, on_progress=collect
            )
        )
    except OSError as error:
        # The pipe is gone: the daemon died, was never there, or is running
        # with this door shut. Said plainly and not retried in silence -- the
        # host decides what happens next (spec 4.6). The session stays up; the
        # next call reconnects.
        return CallToolResult(
            content=[TextContent(type="text", text=_why_no_answer(state_dir, error))],
            is_error=True,
        )
```

Und die Unterscheidung, die §3.4 verlangt:

```python
def _why_no_answer(state_dir: Path, error: OSError) -> str:
    """A shut door and a dead daemon read the same on the wire, and must not
    read the same to the human: one is fixed by a restart, the other by
    dropping `--no-cloud` (spec 3.4)."""
    if client.reachable(state_dir, Channel.LOCAL):
        return (
            "the brain daemon is running but does not serve cloud; "
            "it was started with --no-cloud"
        )
    return f"the brain daemon is not answering: {error}"
    for message in notices:
        # Up as progress, never to stderr: no host reads our stderr. Without a
        # progress token from the host the notice lapses, and that is right --
        # a progress with no recipient is not one (spec 4.4).
        await ctx.session.report_progress(0.0, None, message)
    return result
```

Import ergänzen: `TextContent` aus `mcp.types`.

**Eine Reihenfolge, die zählt:** Die Meldungen werden gesammelt und *nach* dem Aufruf hochgereicht, weil `client.call` synchron in einem Thread läuft und `report_progress` in der Schleife stattfinden muss. Kommt der Hinweis dadurch zu spät, um seinen Zweck zu erfüllen — er soll die zehn Sekunden Stille *erklären*, während sie vergehen —, ist das ein Befund: Dann muss `client.call` den Rückruf über `anyio.from_thread.run` in die Schleife tragen. **Beim Bau prüfen, nicht annehmen**, und das Ergebnis im Commit vermerken.

- [ ] **Step 4: Run the suite**

```bash
uv run pytest
```

- [ ] **Step 5: Lint, types, coverage**

```bash
uv run ruff format && uv run ruff check && uv run mypy && uv run coverage run -m pytest && uv run coverage report
```

- [ ] **Step 6: Commit**

```bash
git add src/brain/mcp.py tests/test_mcp_front.py
git commit -m "Keep an outage from reading as an empty answer"
```

---

## Task 7: Messen, und die Spec berichtigen

Fünf Messläufe, fünf Protokolle unter `bench/2c2/`. **Jede Zahl, die eine Rate ist, nennt ihre Läufe** — sechs Läufe ergaben in 2c-1 einmal 6/6 und hätten einen echten Fund beerdigt, fünfzehn zeigten 3/15 (Spec §8).

**Files:**
- Create: `bench/2c2/privacy-over-mcp.md`, `bench/2c2/two-hosts.md`, `bench/2c2/front-overhead.md`, `bench/2c2/front-startup.md`, `bench/2c2/closed-endpoint.md`
- Modify: `docs/.superpowers/specs/2026-08-22-scheibe-2c2-mcp-fronten-design.md` (§4.3 Revisionstabelle, §7 Ergebnisse)
- Modify: `docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md` §13, wenn die Zahlen es verlangen

**Vor jedem Rückgrat-berührenden Lauf:** `brain daemon stop`. `QmdMcpPort` bekommt sonst das Rückgrat eines laufenden Daemons, nicht das angeforderte (Spec §8).

- [ ] **Step 1: Der Datenschutznachweis über den echten MCP-Kanal (§7.1)**

Ein Bereich auf `local_only` in der echten Registrierung, Claude Code als Wirt an `brain mcp`. Zu prüfen:

1. `catalog` über die Front zeigt den Bereich nicht.
2. `search` über `scope: all` liefert nichts daraus.
3. `read` auf einen bekannten Pfad darin verweigert, als `is_error`.
4. Dieselben drei über die CLI: alles sichtbar.
5. **Die Gegenprobe:** ein Sonden-Klient auf der Cloud-Pipe schickt `{"scope": "geheim", "channel": "local"}` mit. Das Ergebnis ist unverändert verweigert.

Punkt 5 gehört zusätzlich als Test in `tests/test_daemon_server.py` — der Messlauf beweist ihn einmal, der Test hält ihn. Protokoll nach `bench/2c2/privacy-over-mcp.md`, mit den echten Ausgaben, nicht mit Zusammenfassungen.

- [ ] **Step 2: Zwei Wirte, eine Front (§7.2)**

**Gemini CLI ist auf diesem Rechner nicht installiert.** Vor der Installation den Menschen ausdrücklich fragen — es ist eine Installation auf seinem System, keine Projektabhängigkeit.

Dann: beide Wirte auf dieselbe `brain mcp`, alle fünf Werkzeuge je Wirt aufgerufen, die ausgehandelte Revision am Handshake abgelesen. Ablesen ohne Griff in Interna: die Front protokolliert die vom SDK ausgehandelte Revision einmal je Verbindung nach stderr, und der Wirt zeigt sie in seinem eigenen Log. Erfüllt ist es, wenn beide durchkommen **und unser Code keine Revision kennt** — `grep -rn "2025-11-25\|2025-06-18\|2026-07-28" src/` findet außer Kommentaren nichts.

Ergebnis in die Tabelle in §4.3 der Spec eintragen; die Zeile „Gemini CLI: offen" schließen. Protokoll nach `bench/2c2/two-hosts.md`.

- [ ] **Step 3: Was die Front kostet (§7.3)**

Dieselbe Frage, derselbe Daemon, derselbe Lauf — einmal über `brain search`, einmal über die Front. Mindestens 15 warme Aufrufe je Weg, Median und Spanne. Der Unterschied ist der Preis der Umleitung.

**Wird er zweistellig, ist das ein Befund, keine Fußnote:** dann wird er in `bench/2c2/front-overhead.md` benannt und als offener Punkt in §9 der Spec aufgenommen, nicht wegdiskutiert. Protokoll nach `bench/2c2/front-overhead.md`.

- [ ] **Step 4: Der Start läuft in keinen Timeout (§7.4)**

Kein Daemon, kalter Rechner, Claude Code startet `brain mcp`. Gemessen: Zeit bis der Wirt den Server als bereit führt, über mindestens 10 Läufe (Median und Spanne). Dann der erste `search` darauf — kommt er durch, und **kommt der Wartehinweis oben an**? Der zweite Teil prüft die Reihenfolgefrage aus Task 6 Step 3 am echten Wirt.

**Falle:** `Popen.terminate()` beendet auf Windows den Batch-Shim, nicht den Node-Prozess mit dem Modell (Spec §8). Zwischen den Läufen mit `brain daemon stop` aufräumen und danach mit `tasklist` prüfen, dass kein verwaister Daemon steht — 2c-1 hatte nach dem ersten Messlauf vier. Protokoll nach `bench/2c2/front-startup.md`.

- [ ] **Step 5: Der abgeschaltete Endpunkt hält (§7.5)**

`brain daemon start --no-cloud`, dann `brain mcp`: eine ausdrückliche Meldung, kein Hänger, kein Zugriff. Umgekehrt `--no-local` und die CLI. `brain daemon status` nennt in beiden Fällen, was offen ist. Protokoll nach `bench/2c2/closed-endpoint.md`.

- [ ] **Step 6: Die Spec berichtigen**

Die gemessenen Zahlen in §7 der 2c-2-Spec eintragen, die Revisionstabelle in §4.3 schließen. Weicht etwas von der Erwartung ab, wird **die Spec berichtigt**, nicht die Messung gerundet. Sagt eine Zahl etwas über §13 der Architektur-Spec, wird sie auch dort eingetragen.

- [ ] **Step 7: Commit**

```bash
git add bench/2c2 docs/.superpowers/specs
git commit -m "Measure the fronts, and correct the spec where they disagree"
```

---

## Was dieser Plan bewusst nicht enthält

- **Die HTTP-Front für ChatGPT.** Eigene Scheibe, eigene Spec (Spec §2).
- **Den kompilierten Mini-Client.** Die Zahl steht (751 ms), die Entscheidung nicht.
- **Die Untersuchung des `fast`-Budgets** (~78 ms gegen 20 ms aus §13). Eigene Untersuchung, keine stille Anpassung.
- **Den Fehlerbericht an qmd** zu `ggml-cuda.cu`. Eine Veröffentlichung in einem fremden Projekt; liegt beim Menschen.

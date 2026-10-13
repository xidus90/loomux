# GitHub-Auftritt – Implementierungsplan

> **Für ausführende Agenten:** Ausgeführt wird dieser Plan von
> `claude-team.ps1` (Orchestrator, Implementierer mit Modell `sonnet`). Jeder
> Task ist in sich geschlossen; die Schritte haben Checkboxen (`- [ ]`).

**Ziel:** README, Leitfäden und Repo-Metadaten so umbauen, dass ein
Erstbesucher in einer Minute versteht, was loomux kann, und in drei Schritten
installiert.

**Architektur:** Nur Doku. Das README schrumpft auf Pitch, Quickstart, ein
Diagramm, Funktionsliste, Doku-Tabelle, Roadmap, Credits und Lizenz. Was
verschoben wird, landet in zwei neuen Dateien je Sprache, `install.md` und
`releases.md`. Die beiden Dateien, die das Wiki per Hash pinnt
(`docs/de/getting-started.md`, `docs/de/configuration.md`), ändert erst
der letzte Task, mit einer Repin-Pause für den Menschen.

**Werkzeuge:** Markdown, Mermaid, SVG; zum Prüfen ein PEP-723-Skript über
`uv run --script`, `go test`, `sh ci/gate.sh`.

**Spec:** `docs/.superpowers/specs/2026-10-10-github-auftritt-design.md`.
Jeder Implementierer liest sie ganz, bevor er anfängt. Sie enthält die
Begründungen und die belegten Fundstellen.

## Globale Vorgaben

- Label des Pull Requests: `release:none`. Jeder Commit ist `docs:` oder
  `chore:`, nie `fix:` oder `feat:`, auch nicht für die Anker-Reparatur
  (`internal/conventional/conventional.go:84-105`).
- Commit-Nachrichten englisch, ohne Plan- oder Task-Verweise, ohne
  `Co-Authored-By` (AGENTS.md, Abschnitt Rules).
- `README.md`, `docs/en/*`: englisch. `README.de.md`, `docs/de/*`: deutsch.
  Beide Sprachen ändern sich im selben Task und im selben Commit.
- Nicht ändern: `docs/{en,de}/architecture.md`, `cli-reference.md`,
  `hooks.md`, `flows.md`, `benchmarks.md`, `benchmarks/`,
  `open-source-matrix.md`, `.loomux/**`. Gepinnte Dateien werden nie
  umbenannt oder verschoben.
- `docs/de/getting-started.md`, `docs/de/configuration.md`, die Wiki-Seiten
  und `_identities.tsv` ändert nur Task 7.
- Keine Datei schreibt einen Namen aus, auf den die Regex
  `predecessorNames` in `internal/namecheck/references.go:17-19` passt.
  Ausnahme: der Herkunftssatz im README, den `ideaLine`
  (`references.go:101-103`) erlaubt. `go test ./internal/namecheck/` ist der
  Beleg.
- `## Roadmap` bleibt als Überschrift in beiden READMEs wörtlich stehen.
  Andere Doku verlinkt `README*.md#roadmap`.
- In `configuration.md` behalten die bestehenden Überschriften Wortlaut und
  Nummer. Die einzige Ausnahme ist die `[graph]`-Überschrift (Task 7).
- Kein Agent führt `loomux wiki repin`, `loomux init`, `gh repo edit` oder
  `git push` aus.
- Gate: `sh ci/gate.sh` (steht in `.claude/team-gate`). Es braucht rund
  vier Minuten.

## Prüfwerkzeug (für jeden Task)

Jeder Task legt, falls noch nicht vorhanden, dieses Skript im Run-Ordner
`$TEAM_RUN_DIR` ab. `claude-team.ps1` setzt die Variable für jeden Aufruf;
der Ordner ist gitignoriert, und alle Tasks teilen sich eine Kopie. Das
Skript wird nicht committet, denn die Spec schließt einen Linkchecker im
Repo aus. Shell-Variablen überleben keinen Bash-Aufruf, deshalb schreibt
jeder Befehl den Pfad `"$TEAM_RUN_DIR/docs_check.py"` aus. Das
Write-Werkzeug braucht einen absoluten Pfad, also zuerst:

```sh
echo "$TEAM_RUN_DIR"
test -f "$TEAM_RUN_DIR/docs_check.py" && echo present
```

Inhalt von `$TEAM_RUN_DIR/docs_check.py` (mit dem Write-Werkzeug unter dem
ausgegebenen absoluten Pfad anlegen):

```python
# /// script
# requires-python = ">=3.11"
# ///
"""Docs checks: en/de parity, relative links and anchors, forbidden terms.

Usage:
  docs_check.py parity <en.md> <de.md>
  docs_check.py links <file.md>...
  docs_check.py terms <file.md>...
Exit 0 when clean, 1 with one line per finding.
"""
import pathlib
import re
import sys

FENCE_OPEN = re.compile(r"^\s*(`{3,}|~{3,})\s*([\w+-]*)")
LINK = re.compile(r"\]\(([^)\s]+)\)")
TERMS = [
    r"\bG[1-9][a-d]?\b", r"\bW[1-9]\b", r"\bFlow [A-D]\d?\b", r"\b1[a-c]-\d\b",
    r"\b4e\b", r"\b[Ss]tage [A-Z0-9]", r"\bStufe [A-Z0-9]",
    r"(?i)fusion", r"(?i)predecessor", r"Vorgänger", r"Vorläufer",
    r"(?i)reference implementation", r"Referenzimplementierung",
    r"parity-recordings", r"(?i)working papers", r"Arbeitspapiere",
]
# Lines where a listed word means something else (lane order in [verify]).
ALLOWED = ["predecessor is not in the run", "Vorgänger nicht im Lauf steht"]


def outside_fences(path):
    """Yield (lineno, line, block) where block is None outside a fence."""
    fence = None
    for no, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        if fence is None:
            m = FENCE_OPEN.match(line)
            if m:
                fence = m.group(1)
                yield no, line, ("open", m.group(2))
                continue
            yield no, line, None
        elif re.fullmatch(re.escape(fence[0]) + "{%d,}" % len(fence), line.strip()):
            fence = None
            yield no, line, ("close", "")
        else:
            yield no, line, ("in", "")


def skeleton(path):
    levels, blocks, rows, links = [], [], 0, []
    lang, count = "", 0
    for _, line, block in outside_fences(path):
        if block is None:
            h = re.match(r"^(#{1,6})\s", line)
            if h:
                levels.append(len(h.group(1)))
            if re.match(r"^\s*\|", line):
                rows += 1
            links.extend(LINK.findall(line))
        elif block[0] == "open":
            lang, count = block[1], 0
        elif block[0] == "in":
            count += 1
        else:
            blocks.append((lang, None if lang == "mermaid" else count))
    return levels, blocks, rows, links


def norm(target):
    if re.match(r"^[a-z]+:", target):
        return target
    target = target.split("#", 1)[0]
    return target.replace("README.de.md", "README.md").replace("docs/de/", "docs/en/").replace("/de/", "/en/")


def slugs(path):
    out, seen = set(), {}
    for _, line, block in outside_fences(path):
        if block is not None:
            continue
        m = re.match(r"^#{1,6}\s+(.*?)\s*$", line)
        if not m:
            continue
        s = re.sub(r"[^\w\- ]", "", m.group(1).lower()).replace(" ", "-")
        n = seen.get(s, 0)
        seen[s] = n + 1
        out.add(s if n == 0 else f"{s}-{n}")
    return out


def parity(en, de):
    a, b = skeleton(en), skeleton(de)
    found = []
    for name, x, y in (("heading levels", a[0], b[0]), ("code blocks (lang, lines)", a[1], b[1]),
                       ("table rows", a[2], b[2])):
        if x != y:
            found.append(f"{name}: {en}={x} {de}={y}")
    la, lb = sorted(map(norm, a[3])), sorted(map(norm, b[3]))
    if la != lb:
        found.append(f"link targets differ: only {en}: {sorted(set(la) - set(lb))}; only {de}: {sorted(set(lb) - set(la))}")
    return found


def links(path):
    found = []
    for no, line, block in outside_fences(path):
        if block is not None:
            continue
        for target in LINK.findall(line):
            if re.match(r"^[a-z]+:", target):
                continue
            file, _, anchor = target.partition("#")
            dest = (path.parent / file).resolve() if file else path.resolve()
            if not dest.exists():
                found.append(f"{path}:{no}: missing {target}")
            elif anchor and dest.is_file() and anchor not in slugs(dest):
                found.append(f"{path}:{no}: no anchor #{anchor} in {file or path.name}")
    return found


def terms(path):
    found = []
    for no, line, _ in outside_fences(path):
        if any(a in line for a in ALLOWED):
            continue
        for t in TERMS:
            if re.search(t, line):
                found.append(f"{path}:{no}: /{t}/ {line.strip()[:120]}")
    return found


def main(argv):
    sys.stdout.reconfigure(encoding="utf-8")
    cmd, files = argv[1], [pathlib.Path(f) for f in argv[2:]]
    if cmd == "parity":
        found = parity(*files)
    elif cmd == "links":
        found = [f for p in files for f in links(p)]
    elif cmd == "terms":
        found = [f for p in files for f in terms(p)]
    else:
        sys.exit(f"unknown command {cmd}")
    for f in found:
        print(f)
    print(f"{cmd}: {len(found)} finding(s)")
    return 1 if found else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
```

Was die drei Prüfungen bedeuten (Regel aus Spec §2):

- `parity` vergleicht die Folge der Überschriftsebenen außerhalb von
  Codeblöcken, je Codeblock Sprache und Zeilenzahl (bei `mermaid` nur die
  Existenz), die Zahl der Tabellenzeilen und die Linkziele ohne Anker, mit
  `de`→`en` normalisiert. Zusätzlich vergleicht der Implementierer die
  Inhalte der Codeblöcke von Auge. Befehle, Schlüssel und Flags sind gleich;
  abweichen dürfen Kommentare, Zeichenkettenwerte und lokalisierte
  Beispielnamen.
- `links` prüft, dass jedes relative Linkziel existiert und jeder Anker
  nach GitHubs Slug-Regel auflöst.
- `terms` findet Stufenkürzel, Vorgänger-, Fusions- und Archivbezüge.

Belegter Ausgangsstand (2026-10-10):
- `parity` ist heute für alle vier bestehenden Paare sauber (README,
  docs-README, getting-started, configuration).
- `links` meldet genau zwei kaputte Anker, `docs/en/configuration.md:112`
  und `docs/de/configuration.md:114`.

## Review-Fokus

1. **Quickstart in einer frischen PowerShell-Sitzung:** Block 2 und Block 3
   dürfen keine Variable aus Block 1 brauchen. Sie benutzen
   `$env:LOCALAPPDATA`, nie `$bin`. Der Vergleich der Prüfsumme muss ohne
   Rücksicht auf Groß- und Kleinschreibung laufen (`-eq`), denn
   `Get-FileHash` liefert Großbuchstaben, `SHA256SUMS` Kleinbuchstaben.
   Abgedeckt in Task 1 (`install.md`) und Task 3 (README).
2. **Leser unter Linux oder macOS:** Nirgends darf stehen, dass Hooks,
   `init` oder die Selbstaktualisierung dort laufen. Abgedeckt in Task 1,
   Schritt „Windows-Grenzen".
3. **Verlorene Tatsachen:** Jede Tatsache aus den verschobenen
   README-Abschnitten steht danach in `install.md`, in `releases.md` oder
   belegbar schon in `docs/`, und zwar in beiden Sprachen. Abgedeckt in
   Task 1 und Task 2 (Abgleichliste) sowie Task 3 (Streichliste).
4. **Tote Links nach dem Umbau:** Betroffen sind Links aus Dateien, die
   bleiben, auf Abschnitte, die wandern, etwa CONTRIBUTING, die
   PR-Vorlage und AGENTS.md. Abgedeckt in Task 5 per `grep` und `links`.
5. **Namenscheck auf neuen Dateien:** Ein SVG oder ein neuer Leitfaden
   schreibt versehentlich einen Vorgängernamen aus. Abgedeckt in jedem Task
   durch `go test ./internal/namecheck/`.

---

### Task 1: `install.md` in beiden Sprachen

**Domain:** ux

**Files:**
- Create: `docs/en/install.md`
- Create: `docs/de/install.md`

**Interfaces:**
- Consumes: nichts.
- Produces: Die Dateien `docs/en/install.md` und `docs/de/install.md`.
  - Diese Überschriften brauchen spätere Tasks als Anker:
    - en: `## Windows quickstart`, `## Linux and macOS`, `## Building from source`
    - de: `## Schnellstart unter Windows`, `## Linux und macOS`, `## Aus dem Quellcode bauen`
  - Task 3 verlinkt `docs/en/install.md#linux-and-macos` und
    `docs/de/install.md#linux-und-macos`.

Quellen, die verschoben werden:
- `README.md:384-432` bzw. `README.de.md:391-440` (Releases bis vor die
  Labeltabelle, also Download, Prüfsumme, `NOTICE.md`, maschinenweites
  Binary, MCP-Eintrag, Selbstaktualisierung, `init`-Teile `binary` und
  `mcp-json`),
- `README.md:458-467` bzw. `README.de.md:466-476` (Kanäle und
  `loomux upgrade --beta|--stable|--version`),
- `docs/en/getting-started.md:7-38` bzw. `docs/de/getting-started.md:7-38`
  (Voraussetzungen, Installation, `--version`).

Die Quellen bleiben in diesem Task unverändert. Task 3 und Task 7 entfernen
sie.

- [x] **Schritt 1: Prüfwerkzeug anlegen und den roten Ausgangsstand zeigen**

```sh
uv run --script "$TEAM_RUN_DIR/docs_check.py" parity docs/en/install.md docs/de/install.md
```
Erwartet: Abbruch mit `FileNotFoundError` (die Dateien fehlen).

- [x] **Schritt 2: `docs/en/install.md` schreiben**

Gerüst, in dieser Reihenfolge (die Überschriften wörtlich):

```markdown
# Installing loomux

[Overview](README.md) · [First-time setup](getting-started.md) →

<one paragraph: loomux is one binary; Windows is the tested platform; what
this page covers>

## Requirements
## Windows quickstart
## Verifying a download
## The machine-wide binary
## Updates and channels
## Linux and macOS
## Building from source
## Checking the version

[Overview](README.md) · [First-time setup](getting-started.md) →
```

Inhalt je Abschnitt:

- **Requirements**:
  - Windows x64 als getestete Plattform.
  - Die GitHub CLI mit `gh auth login`. Sie übernimmt den Download und
    die Selbstaktualisierung.
  - Git ≥ 2.30.
  - Go ≥ 1.26 nur für den Bau aus dem Quellcode (die Toolchain des Moduls
    ist go1.27.0).
  - Der Satz zu Symlinks aus `getting-started.md:13` kommt **nicht** mit.
- **Windows quickstart**: genau dieser Block (en und de identisch bis auf
  Kommentare):

```powershell
$bin = "$env:LOCALAPPDATA\loomux\bin"
New-Item -ItemType Directory -Force $bin | Out-Null
gh release download --repo xidus90/loomux --pattern 'loomux_*_windows_amd64.exe' --pattern SHA256SUMS --dir $bin --clobber
$exe = Get-ChildItem $bin -Filter 'loomux_*_windows_amd64.exe' | Select-Object -First 1
if (-not $exe) { throw 'download failed: see the gh error above' }
$want = ((Select-String -Path "$bin\SHA256SUMS" -Pattern $exe.Name -SimpleMatch).Line -split '\s+')[0]
if ((Get-FileHash $exe.FullName -Algorithm SHA256).Hash -eq $want) { Move-Item -Force $exe.FullName "$bin\loomux.exe"; Remove-Item "$bin\SHA256SUMS" } else { throw "checksum mismatch: $($exe.Name)" }
```

  Danach zwei Sätze:
  - Ohne Tag lädt `gh release download` das neueste stabile Release.
  - Der Vergleich mit `-eq` ignoriert die Groß- und Kleinschreibung.

  Danach dieser Block mit dem Hinweis, ihn in einer beliebigen Sitzung im
  eigenen Projekt auszuführen:

```powershell
& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" init
& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" status
```

  Dazu ein Satz: Liegt das Binary dort, lädt `init` kein zweites
  (`internal/setup/plan.go:97-98`, Quelle nicht in die Doku schreiben).
  Optional dauerhaft in den Benutzer-`PATH`:

```powershell
[Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ";$env:LOCALAPPDATA\loomux\bin", 'User')
```

  Der Weg wirkt erst in einer neuen Sitzung. Der Einzeiler schreibt den
  Benutzer-`Path` als einfachen String (REG_SZ) und löst dabei
  `%VAR%`-Einträge auf; wer solche Einträge behalten will, ergänzt den
  Ordner lieber über „Umgebungsvariablen bearbeiten". Weiter geht es mit
  `getting-started.md`.
- **Verifying a download**: Die Assetnamen
  `loomux_<version>_<os>_<arch>` für `windows/amd64` (`.exe`),
  `linux/amd64`, `linux/arm64`, `darwin/amd64` und `darwin/arm64`. Dazu
  `sha256sum --check --ignore-missing SHA256SUMS` für Linux/macOS/Git Bash
  und den `NOTICE.md`-Absatz aus `README.md:395-398`, wörtlich übernommen.
- **The machine-wide binary**: der Inhalt von `README.md:400-432`:
  - Pfad, ein Binary je Maschine, nie aus einem Checkout.
  - `claude mcp add loomux -s user -- "$env:LOCALAPPDATA\loomux\bin\loomux.exe" mcp --channel local`.
  - Die Teile `binary` und `mcp-json` von `init` mit dem Link
    `cli-reference.md#11-project-setup-loomux-init`.
  - Den Satz „Its first run by a human is still pending" **weglassen**.
- **Updates and channels**: `serve` aktualisiert eine Minute nach dem Start
  und danach täglich, dazu `update.json`, `loomux upgrade`, die Kanaldatei
  `channel`, `--beta`, `--stable`, `--version <x.y.z>`, „a downgrade does
  not hold", die Regel für Binaries vor 1.0.0 (`README.md:420-427` und
  `458-467`). Die Selbstaktualisierung gibt es nur unter Windows.
- **Linux and macOS**:
  - Release-Binaries gibt es, getestet wird unter Windows.
  - Nur unter Windows gibt es: das maschinenweite Binary und damit den Teil
    `binary` von `init`, die Selbstaktualisierung und die Worktree-Spiegel
    (Junctions).
  - `init` ist dort nicht vorgesehen. Der Teil `binary` scheitert, und
    `.mcp.json`, die Hook-Einträge und die Git-Hooks werden nicht
    geschrieben, sondern als fehlgeschlagen gemeldet, weil sie
    `${LOCALAPPDATA}/loomux/bin/loomux.exe` aufriefen.
  - Wer `init` dort trotzdem nutzt, setzt das Modul `hooks` auf `none` und
    wählt im Modul `base` mit `each` die Teile `binary` und `mcp-json` ab.
    Das geht nur interaktiv, denn für `base` gibt es kein Flag.
  - Was dort läuft: die CLI selbst, also `check`, `graph`, `brain` und
    `serve`.
- **Building from source**: `git clone …` und `cd loomux`. Danach gibt es
  zwei Wege:
  - Unter Windows, für die Arbeit **an** loomux:
    `go run ./cmd/loomux init --yes`. Das baut `bin/loomux.exe` und
    schaltet die Git-Hooks scharf.
  - Unter Linux und macOS: `go build -o loomux ./cmd/loomux`. `init` ist
    dort nicht vorgesehen (siehe „Linux and macOS").
  Dazu ein Satz: `go install …@latest` geht nicht, weil `go.mod` eine
  `replace`-Direktive hat.
- **Checking the version**: der Block aus `getting-started.md:32-37`
  (`loomux --version` mit den drei Ausgabeformen).

- [x] **Schritt 3: `docs/de/install.md` als Zwilling schreiben**

Gleiches Gerüst und gleiche Codeblöcke. Nur die Kommentare und die Prosa
sind deutsch. Überschriften: `# loomux installieren`, `## Voraussetzungen`,
`## Schnellstart unter Windows`, `## Einen Download prüfen`,
`## Das maschinenweite Binary`, `## Updates und Kanäle`,
`## Linux und macOS`, `## Aus dem Quellcode bauen`,
`## Die Version prüfen`. Die Navigation lautet
`[Übersicht](README.md) · [Ersteinrichtung](getting-started.md) →`. Der
Anker auf die CLI-Referenz lautet `cli-reference.md#11-projekt-einrichten-loomux-init`.
Quelle der deutschen Sätze ist `README.de.md:391-440` und `466-476`.

- [x] **Schritt 4: Prüfen**

```sh
uv run --script "$TEAM_RUN_DIR/docs_check.py" parity docs/en/install.md docs/de/install.md
uv run --script "$TEAM_RUN_DIR/docs_check.py" links docs/en/install.md docs/de/install.md
uv run --script "$TEAM_RUN_DIR/docs_check.py" terms docs/en/install.md docs/de/install.md
go test ./internal/namecheck/
```
Erwartet: dreimal `0 finding(s)`, exit 0, und `ok` für namecheck.
`getting-started.md` existiert, also löst der Link auf.

Abgleichliste, von Hand geprüft: Jeder Satz aus `README.md:386-432` und
`458-467` (ohne „first run … pending") findet sich in `docs/en/install.md`
wieder. Für das Deutsche gilt dasselbe mit `README.de.md`.

- [x] **Schritt 5: Gate und Commit**

```sh
sh ci/gate.sh
git add docs/en/install.md docs/de/install.md
git commit -m "docs: add an install guide in English and German"
```

---

### Task 2: `releases.md` in beiden Sprachen

**Domain:** ux

Depends on Task 1. `releases.md` verlinkt `install.md#updates-and-channels`.

**Files:**
- Create: `docs/en/releases.md`
- Create: `docs/de/releases.md`

**Interfaces:**
- Consumes: nichts.
- Produces: Die Dateien `docs/en/releases.md` und `docs/de/releases.md`.
  - Diese Überschriften brauchen spätere Tasks als Anker:
    - en: `## Opening a pull request` → `#opening-a-pull-request`
    - de: `## Einen Pull Request öffnen` → `#einen-pull-request-öffnen`
  - Task 5 verlinkt `docs/en/releases.md#opening-a-pull-request`.

Quellen, die verschoben werden:
- `README.md:434-456` (Labeltabelle, stabil und Pre-Release, CHANGELOG,
  Beta über den Workflow),
- `README.md:469-503` (Opening a pull request),
- `README.md:504-555` (Setting up releases (maintainers), samt Absatz zum
  verlorenen Release),
- deutsch `README.de.md:442-464` und `478-571` entsprechend.

Die Quellen bleiben in diesem Task unverändert; Task 3 entfernt sie.

- [x] **Schritt 1: Den roten Ausgangsstand zeigen**

```sh
uv run --script "$TEAM_RUN_DIR/docs_check.py" parity docs/en/releases.md docs/de/releases.md
```
Erwartet: `FileNotFoundError`.

- [x] **Schritt 2: `docs/en/releases.md` schreiben**

```markdown
# Releases and pull requests

[Overview](README.md)

## How a pull request becomes a release
## Beta releases
## Opening a pull request
## Setting up releases (maintainers)
```

- **How a pull request becomes a release**: der Satz „Every merged pull
  request to `master` is released according to its label", die
  Labeltabelle, stabil und Pre-Release, `CHANGELOG.md` (Link
  `../../CHANGELOG.md`) und „the entries marked `-beta` are the count before
  1.0.0".
- **Beta releases**: der Absatz `README.md:448-456`, also Workflow,
  Parameter, Beispielbefehl und Tag als einzige Spur. Die Kanäle der
  Installation verlinkt er nach `install.md#updates-and-channels` und
  wiederholt sie nicht.
- **Opening a pull request**: `README.md:471-503` wörtlich, mit einer
  Änderung: Aus „Pick one label from the table above" wird „Pick one label
  from the table in
  [How a pull request becomes a release](#how-a-pull-request-becomes-a-release)".
- **Setting up releases (maintainers)**: `README.md:506-555` wörtlich, die
  Schritte 1–4 und der Schlussabsatz. Pfade auf `.github/...` werden zu
  Links relativ zur Datei (`../../.github/rulesets/master.json`), wo sie
  heute Links sind; Code bleibt Code.

- [x] **Schritt 3: `docs/de/releases.md` als Zwilling schreiben**

Überschriften: `# Releases und Pull Requests`,
`## Wie ein Pull Request zum Release wird`, `## Beta-Releases`,
`## Einen Pull Request öffnen`, `## Releases einrichten (Maintainer)`. Der
Kanal-Link lautet `install.md#updates-und-kanäle`. Die deutsche Quelle ist
`README.de.md:442-464` und `478-571`.

- [x] **Schritt 4: Prüfen**

```sh
uv run --script "$TEAM_RUN_DIR/docs_check.py" parity docs/en/releases.md docs/de/releases.md
uv run --script "$TEAM_RUN_DIR/docs_check.py" links docs/en/releases.md docs/de/releases.md
uv run --script "$TEAM_RUN_DIR/docs_check.py" terms docs/en/releases.md docs/de/releases.md
go test ./internal/namecheck/
```
Erwartet: dreimal `0 finding(s)` und `ok`.

Abgleichliste: Jeder Satz aus `README.md:434-456` und `469-555` steht in
`docs/en/releases.md`, deutsch ebenso.

- [x] **Schritt 5: Gate und Commit**

```sh
sh ci/gate.sh
git add docs/en/releases.md docs/de/releases.md
git commit -m "docs: move the release and pull request procedure into the docs"
```

---

### Task 3: README.md und README.de.md umbauen

**Domain:** ux

Depends on Task 1 and Task 2.

**Files:**
- Modify: `README.md` (ganze Datei)
- Modify: `README.de.md` (ganze Datei)

**Interfaces:**
- Consumes: Die Anker `docs/en/install.md#linux-and-macos` und
  `docs/de/install.md#linux-und-macos` sowie die Dateien
  `docs/{en,de}/releases.md`.
- Produces: Die README-Überschrift `## Roadmap` bleibt unverändert.

- [x] **Schritt 1: Den roten Ausgangsstand zeigen**

```sh
uv run --script "$TEAM_RUN_DIR/docs_check.py" terms README.md README.de.md
grep -c '^```mermaid' README.md README.de.md
```
Erwartet:
- `terms` meldet heute rund 70 Befunde (Stufenkürzel der Roadmap,
  Archiv, Fusion, Vorgänger) und endet mit exit 1.
- `mermaid` zählt mehr als 1 je Datei.

- [x] **Schritt 2: `README.md` neu schreiben**

Reihenfolge der Abschnitte:

1. `# loomux` und die Sprach- und Doku-Zeile wie heute (Zeile 3).
2. **Pitch**:
   - Ein Untertitel in Fettschrift, etwa *„One Go binary that guards AI
     coding agents: hooks and a write barrier, a check chain, a code graph
     and a project wiki."*
   - Dazu zwei bis drei Sätze: für Claude Code und Antigravity über Hooks,
     für Cursor als MCP-Client, Codex noch nicht angebunden (aus
     `README.md:7`).
   - Dann diese Liste:
     - one binary, no Python or Node runtime;
     - hook decisions within a 35 ms budget;
     - 100 % test coverage per function (an exemption needs a reason in the
       code);
     - no telemetry; loomux goes online only for updates through `gh` and
       on explicit commands;
     - Windows first: binaries for Linux and macOS exist, testing happens
       on Windows; hook setup and self-update are Windows only.
   - Danach der Namensabsatz („What's in a name?") unverändert.
3. `## Quickstart (Windows)`:
   - Ein Satz zur Voraussetzung: GitHub CLI mit `gh auth login`.
   - Dann drei nummerierte Schritte:
     - Schritt 1 ist der erste Block aus `install.md` § „Windows
       quickstart", unverändert.
     - Schritt 2 ist
       `& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" init` in einer
       beliebigen Sitzung im eigenen Projekt.
     - Schritt 3 ist `… status`.
   - Darunter ein Satz: „On Linux or macOS, see
     [Linux and macOS](docs/en/install.md#linux-and-macos); everything
     else about installing is in [Installing loomux](docs/en/install.md),
     the next step is [First-time setup](docs/en/getting-started.md)."
4. `## How it fits together`: das Flowchart aus `README.md:21-45`, ohne
   Stufen und ohne Vorgänger. Die Knoten heißen wie heute, die Zahlen
   „1."–„5." dürfen bleiben. Der Satz *„A dashed line is specified and not
   built; …"* bleibt darunter. Sonst steht kein Mermaid-Block im README.
5. `## Features`: eine Tabelle `| Feature | What it does | Docs |`, eine
   Zeile je Funktion:
   - Guard and write barrier → `docs/en/hooks.md`
   - Check chain and gate → `docs/en/configuration.md#verify-check-chains--quality-gates`
   - Code graph (Go, Python, GDScript; `ask`, `callers`, `blast`, …) →
     `docs/en/cli-reference.md#6-code-graph-engine-loomux-graph`
   - Wiki and second brain → `docs/en/cli-reference.md#7-second-brain--wiki-loomux-brain`
   - MCP service → `docs/en/cli-reference.md#8-mcp-service--stdio-bridge-loomux-serve--loomux-mcp`
   - Flows → `docs/en/flows.md`
   - Worktree mirrors (Windows only) → `docs/en/cli-reference.md#5-worktree-mirroring-loomux-worktree`
   - Releases and self-update (Windows only) → `docs/en/install.md#updates-and-channels`

   Jede Beschreibung umfasst ein bis zwei Sätze. Die Grundlage ist die
   CLI-Übersicht `README.md:189-329`; dort steht, was gebaut ist. Nur
   Gebautes kommt in die Tabelle.
6. `## Documentation`: die Tabelle aus `README.md:362-371`, oben ergänzt um
   `install.md` und `releases.md`. Die Zeilen stehen in der Reihenfolge
   Install → Getting started → Configuration → CLI reference → Hooks →
   Architecture → Flows → Releases → Benchmarks → Benchmark matrix. Die
   Benchmarks-Zeile heißt „Chronological performance measurements with
   date, baseline and change, cold and warm", ohne „predecessor".
7. `## Roadmap`: der heutige Abschnitt `README.md:141-187` mit diesen
   Änderungen (Spec §1.7):
   - Die Spalte `Stage` fällt in „Coming" weg; die Kopfzeile lautet dann
     `| Feature | What it brings | Depends on | Priority |`.
   - Die Zeile `space on loomux alone` entfällt.
   - In `Brain web app` fällt der Klammerteil „(components of the
     predecessor's web app, kept in the archive release
     `archive/parity-recordings`)" weg.
   - In der Spalte `Depends on` gilt diese Zuordnung:
     - `Flow A ✅` → `Flow core without models ✅`
     - `Flow B, C` und das bloße `Flow` → `Flow runtime`
     - `1b-2 ✅` → `MCP service ✅`
     - `W1` → `Web OS shell`
     - `G4a ✅` → `Graph navigation ✅`
     - `G4b ✅` → `Blast radius of a diff ✅`
     - `G5a ✅` → `Python in the code graph ✅`
     - `G5c ✅` → `GDScript in the code graph ✅`
   - Freitext in der Zeile **Flow runtime**:
     - Aus „(through their CLIs, their APIs or both, which stage B
       decides)" wird „(through their CLIs, their APIs or both, which is
       still to be decided)".
     - Aus „`verify-until-green` as a data flow for the building blocks of
       stage C" wird „`verify-until-green` as a data flow for the building
       blocks it still needs".
     - Aus „Flow B also takes a diagnosis run without a model
       (`run --no-model`)" wird „it will also take a diagnosis run without
       a model (`run --no-model`)".
   - Freitext in der Zeile **TypeScript/TSX**: Aus „that reads Python since
     G5a" wird „that already reads Python".
   - Die Tabelle „Maybe" und der Schlusssatz bleiben.
8. `## Credits`: der Satz aus `README.md:345`, Zeichen für Zeichen.
9. `## Licence` wie heute.

Ersatzlos entfallen: Core Workflows (`51-139`), CLI Reference
(`189-329`), Architectural Decisions (`331-345`, außer dem Satz in
Credits), Architecture Principles (`349-356`), Specifications & Internal
Working Papers (`375-380`), Releases bis Ende (`384-555`). Diese Teile
stehen jetzt in Task 1 und Task 2 oder in der Doku.

Streichliste (fest, vom Planer am 2026-10-10 gegen `docs/en` abgeglichen).
Jede andere Tatsache der entfallenden Abschnitte steht schon in `docs/en/`.
Diese hier fehlen dort und gehen so weiter:

| README (en) | Tatsache | Ziel |
|---|---|---|
| 283-284 | Der Endpunkt des Modells muss auf Loopback bleiben; kein Proxy aus der Umgebung, keine Weiterleitung. | Task 7, `configuration.md` § Machine-wide settings |
| 342-343 | loomux synchronisiert nichts in eine Cloud; alles Wissen, alle Regeln und Graphen bleiben lokal; keine Telemetrie. | Task 7, `configuration.md` § `[privacy]` (ein Satz). Der Pitch nennt „no telemetry" schon. |
| 345 | Die portierten Teile und ihre MIT-Lizenz stehen in `NOTICE.md`. | bleibt im README: Er ist Teil des Herkunftssatzes in den Credits. |
| 352 | Startzeit-Disziplin: Keine Paketvariable und kein `init()` parst eingebettete Daten oder macht Netz-I/O. | Task 4, `docs/{en,de}/README.md` § Core philosophy, als vierter Punkt (ohne die Messzahl). |
| 353 | Hook-Pfade laufen im Prozess und hängen nie von einem laufenden `serve` ab. | Task 4, ebenda, als fünfter Punkt. |

Der Absatz zu stdin-Interpretern (98) steht schon in `hooks.md` §7 und
`configuration.md:66`.

Diese Tatsachen entfallen bewusst:
- `/api/brain` und `/api/graph` (122) und „code symbols link to ADRs"
  (340): Das ist nicht gebaut und steht als Plan in der Roadmap.
- Die Absage an die Node- und C++-Werkzeugkette (341): Sie begründet sich
  aus dem Vorgänger.
- Die Ablage der Arbeitspapiere (375-380): Das steht in AGENTS.md.
- „7.5 ms warm hook" (335): eine veraltete Messung.
- Die Statuszeilen „first run … pending" (246, 432) und „merge-hook not yet
  in use on a host" (238): Sie sind veraltet bzw. ein Status, keine
  Funktion.

Für das Deutsche gilt dieselbe Liste; die Quellsätze stehen an den
entsprechenden Stellen von `README.de.md` (z. B. 349-350, 359-360).

- [x] **Schritt 3: `README.de.md` als Zwilling**

Die Abschnitte heißen `## Schnellstart (Windows)`,
`## Wie es zusammenhängt`, `## Funktionen`, `## Dokumentation`,
`## Roadmap` (wörtlich), `## Credits` und `## Lizenz`. Der Herkunftssatz
kommt Zeichen für Zeichen aus `README.de.md:352`. Die Zuordnungen in der
Roadmap:
- „Flow-Kern ohne Modelle ✅"
- „Flow-Laufzeit"
- „MCP-Dienst ✅"
- „Web-OS-Shell"
- „Graph-Navigation ✅"
- „Blast-Radius eines Diffs ✅"
- „Python im Code-Graph ✅"
- „GDScript im Code-Graph ✅"

Den Freitext so umformulieren:
- „was noch zu entscheiden ist"
- „für die Bausteine, die sie noch braucht"
- „sie wird auch einen Diagnoselauf ohne Modell nehmen (`run --no-model`)"
- „der schon Python liest"

Links zeigen auf `docs/de/…`. Die Anker der deutschen Doku sind in
`docs/de/cli-reference.md` nachzuschlagen, z. B.
`#11-projekt-einrichten-loomux-init`.

- [x] **Schritt 4: Prüfen**

```sh
uv run --script "$TEAM_RUN_DIR/docs_check.py" parity README.md README.de.md
uv run --script "$TEAM_RUN_DIR/docs_check.py" links README.md README.de.md
uv run --script "$TEAM_RUN_DIR/docs_check.py" terms README.md README.de.md
grep -c '^```mermaid' README.md README.de.md
grep -n '^## Roadmap$' README.md README.de.md
go test ./internal/namecheck/
```
Erwartet:
- dreimal `0 finding(s)`,
- `mermaid` je Datei `1`,
- `## Roadmap` in beiden Dateien je einmal,
- `ok` für namecheck.

Den Herkunftssatz meldet `terms` nicht, weil er keinen der Begriffe
enthält.

- [x] **Schritt 5: Gate und Commit**

```sh
sh ci/gate.sh
git add README.md README.de.md
git commit -m "docs: rebuild the README around a pitch, a quickstart and the features"
```

---

### Task 4: Navigationsseiten `docs/en/README.md` und `docs/de/README.md`

**Domain:** ux

Depends on Task 1 and Task 2.

**Files:**
- Modify: `docs/en/README.md` (ganze Datei, heute 26 Zeilen)
- Modify: `docs/de/README.md` (ganze Datei)

**Interfaces:**
- Consumes: die Dateien `install.md` und `releases.md` beider Sprachen.
- Produces: Die Navigationsseite heißt in der Pfad-Navigation von Task 7
  „Overview" bzw. „Übersicht" und liegt unter `README.md` im selben Ordner.

- [x] **Schritt 1: Den roten Ausgangsstand zeigen**

```sh
uv run --script "$TEAM_RUN_DIR/docs_check.py" terms docs/en/README.md docs/de/README.md
```
Erwartet: Die Zeile 17 meldet „predecessor" bzw. „Vorläufer", exit 1.

- [x] **Schritt 2: `docs/en/README.md` neu schreiben**

```markdown
# loomux documentation

<one sentence what loomux is, as in the README pitch>

## Get started

1. [Installing loomux](install.md) — download, verify, the machine-wide binary, updates; Linux and macOS.
2. [First-time setup](getting-started.md) — `loomux init`, reviewing the configuration, wiring your agent.
3. [Configuration](configuration.md) — `.loomux/config.toml`, starting with your first day.

## Reference

| Guide | What it covers |
|---|---|
| [CLI reference](cli-reference.md) | … |
| [Configuration](configuration.md) | … |
| [Hooks](hooks.md) | … |

## Concepts

| Guide | What it covers |
|---|---|
| [Architecture](architecture.md) | … |
| [Flows](flows.md) | … |

## Project

| Guide | What it covers |
|---|---|
| [Releases and pull requests](releases.md) | … |
| [Benchmarks](benchmarks.md) | … |
| [Benchmark matrix](benchmarks/matrix.md) | … |
| [Open-source matrix](open-source-matrix.md) | … |

## Core philosophy

<the three points of today's lines 22-26, unchanged, then two more:>
4. **Start-time discipline**: no package-level variable and no `init()` parses embedded data or does network I/O.
5. **Strict isolation**: hook paths run in-process and never depend on a running `serve`.
```

Die `…` sind Beschreibungen aus der heutigen Tabelle (Zeilen 11-18),
ohne „predecessor". Für Benchmarks gilt dieselbe Formulierung wie in
Task 3.

- [x] **Schritt 3: `docs/de/README.md` als Zwilling**

Die Überschriften lauten `# loomux-Dokumentation`, `## Loslegen`,
`## Referenz`, `## Konzepte`, `## Projekt` und `## Grundsätze` (bzw. der
heutige Titel des Abschnitts zur Philosophie). Die Linktexte sind
„loomux installieren", „Ersteinrichtung" und „Konfiguration".

- [x] **Schritt 4: Prüfen**

```sh
uv run --script "$TEAM_RUN_DIR/docs_check.py" parity docs/en/README.md docs/de/README.md
uv run --script "$TEAM_RUN_DIR/docs_check.py" links docs/en/README.md docs/de/README.md
uv run --script "$TEAM_RUN_DIR/docs_check.py" terms docs/en/README.md docs/de/README.md
diff <(ls docs/en) <(ls docs/de)
go test ./internal/namecheck/
```
Erwartet:
- dreimal `0 finding(s)`,
- keine Ausgabe von `diff` (beide Ordner haben dieselben Namen),
- `ok` für namecheck.

- [x] **Schritt 5: Gate und Commit**

```sh
sh ci/gate.sh
git add docs/en/README.md docs/de/README.md
git commit -m "docs: turn the docs index into a guided path"
```

---

### Task 5: Verweise nachziehen (AGENTS, CONTRIBUTING, PR-Vorlage, Testkommentar)

**Domain:** infra

Depends on Task 1 and Task 2.

**Files:**
- Modify: `AGENTS.md:58-61` (Verweis auf „Opening a pull request")
- Modify: `AGENTS.md:104-107` (Roadmap-Regel)
- Modify: `CONTRIBUTING.md:46-51`
- Modify: `.github/pull_request_template.md:20`
- Modify: `internal/hosts/hostio_internal_test.go:9-10` (nur Kommentar)

**Interfaces:**
- Consumes: `docs/en/releases.md#opening-a-pull-request` (Task 2) und
  `docs/en/install.md` (Task 1).
- Produces: nichts.

- [x] **Schritt 1: Den roten Ausgangsstand zeigen**

```sh
grep -rn 'README.md#opening-a-pull-request\|"Opening a pull request" in `README.md`\|README.md, "Opening a pull request"\|its stage, dependencies\|README.md.s install block' AGENTS.md CONTRIBUTING.md .github/pull_request_template.md internal/hosts/hostio_internal_test.go
```
Erwartet: fünf Treffer.

- [x] **Schritt 2: Ändern**

- `AGENTS.md:60-61`: Aus `follow "Opening a pull request" in `README.md`.`
  wird `follow "Opening a pull request" in `docs/en/releases.md`.`
- `AGENTS.md:106`: Aus `each row with its stage, dependencies and
  priority where it has them` wird `each row with its dependencies and
  priority where it has them`. Den Absatz neu umbrechen, sonst nichts
  ändern.
- `CONTRIBUTING.md:48`: Der Link wird zu
  `[Opening a pull request](docs/en/releases.md#opening-a-pull-request)`,
  und „in the README" wird zu „in the release guide". Der Rest des Satzes
  bleibt.
- `.github/pull_request_template.md:20`: `The full procedure is
  docs/en/releases.md, "Opening a pull request".`
- `internal/hosts/hostio_internal_test.go:10`: Aus „README.md's install
  block names Windows, Linux and macOS" wird „docs/en/install.md names
  Windows, Linux and macOS". Nur der Kommentar ändert sich.

- [x] **Schritt 3: Prüfen**

```sh
grep -rn 'README.md#opening-a-pull-request\|"Opening a pull request" in `README.md`\|README.md, "Opening a pull request"\|its stage, dependencies\|README.md.s install block' AGENTS.md CONTRIBUTING.md .github/pull_request_template.md internal/hosts/hostio_internal_test.go
uv run --script "$TEAM_RUN_DIR/docs_check.py" links CONTRIBUTING.md AGENTS.md
gofmt -l internal/hosts/
```
Erwartet:
- `grep` ohne Treffer (exit 1),
- `links` mit `0 finding(s)`,
- `gofmt` ohne Ausgabe.

- [x] **Schritt 4: Gate und Commit**

```sh
sh ci/gate.sh
git add AGENTS.md CONTRIBUTING.md .github/pull_request_template.md internal/hosts/hostio_internal_test.go
git commit -m "docs: point contributors to the release guide for pull requests"
```

---

### Task 6: Social-Preview-SVG

**Domain:** ux

**Files:**
- Create: `docs/assets/social-preview.svg`

**Interfaces:**
- Consumes: nichts.
- Produces: Das SVG mit 1280×640. Der Abschnitt „Für den Menschen" in
  diesem Plan nennt seinen Pfad.

- [x] **Schritt 1: Den roten Ausgangsstand zeigen**

```sh
test -f docs/assets/social-preview.svg; echo $?
```
Erwartet: `1`.

- [x] **Schritt 2: SVG schreiben**

Vorgaben:
- `<svg xmlns="http://www.w3.org/2000/svg" width="1280" height="640" viewBox="0 0 1280 640">`.
- Ein dunkler Hintergrund in voller Fläche.
- Links mit einem Rand von mindestens 80 px:
  - „loomux" groß, etwa 120 px,
  - darunter der Untertitel aus Task 3 auf höchstens zwei Zeilen
    (`<tspan>`, etwa 36 px),
  - darunter drei Kernaussagen: „One binary · no Python or Node runtime",
    „Hook decisions within 35 ms", „100 % coverage per function".
- `font-family="Segoe UI, Helvetica, Arial, sans-serif"`.
- Keine `<image>`, kein `href` auf externe Ressourcen, kein base64, kein
  `<script>`.
- Rechts darf ein einfaches Motiv aus `<line>` und `<circle>` stehen, das
  ein Gewebe oder einen Graphen andeutet, höchstens 40 Elemente.

- [x] **Schritt 3: Prüfen**

```sh
grep -c '<script\|<image\|base64\|href="http' docs/assets/social-preview.svg
go test ./internal/namecheck/
```
Erwartet:
- `0` (dann endet `grep -c` mit exit 1; das ist hier das gewünschte
  Ergebnis),
- `ok` für namecheck.

Die Maße prüft `svg_ok.py` unten.

Zusätzlich die Datei als XML parsen lassen:

```sh
uv run --script "$TEAM_RUN_DIR/svg_ok.py" docs/assets/social-preview.svg
```

Dafür `svg_ok.py` neben `docs_check.py` anlegen:

```python
# /// script
# requires-python = ">=3.11"
# ///
"""Exit 0 when the file parses as XML with an svg root of 1280x640."""
import sys
import xml.etree.ElementTree as ET

root = ET.parse(sys.argv[1]).getroot()
ok = root.tag.endswith("svg") and root.get("width") == "1280" and root.get("height") == "640"
print("svg ok" if ok else f"bad root: {root.tag} {root.attrib}")
sys.exit(0 if ok else 1)
```
Erwartet: `svg ok`.

- [x] **Schritt 4: Gate und Commit**

```sh
sh ci/gate.sh
git add docs/assets/social-preview.svg
git commit -m "docs: add the source of the repository's social preview"
```

---

### Task 7: Ersteinrichtung und Konfiguration (gepinnte Dateien, Repin-Pause)

**Domain:** ux

Depends on Task 1, Task 2, Task 3, Task 4, Task 5 and Task 6. Er läuft als
letzter, damit nur er eine Pause braucht.

**Files:**
- Modify: `docs/en/getting-started.md`, `docs/de/getting-started.md`
  (ganze Datei)
- Modify: `docs/en/configuration.md`
  - Kopf, Zeilen 1-6
  - 112 (Anker)
  - 703-731 (`[graph]`)
  - 723-726 (Referenzimplementierung)
  - 902 (`version`)
  - der Abschnitt „Machine-wide settings" ab 992
  - `[privacy]` (ab 784), ein Zusatz
- Modify: `docs/de/configuration.md` an denselben Stellen (de-Zeilen:
  114, 725-751, 935, Kopf und „Maschinenweite Einstellungen")
- Modify, nur bei Bedarf: `docs/wiki/topics/datenschutz-und-kanaele.md`,
  `docs/wiki/topics/schreibschranke.md`,
  `docs/wiki/topics/datenmodell-und-bereiche.md` und
  `docs/wiki/entities/qmd.md` (nur Prosa)
- Modify: `docs/wiki/topics/code-graph.md:156-158`,
  `docs/wiki/topics/scheiben-und-abnahme.md:48` (Roadmap-Verweise)
- Modify: `docs/wiki/log.md` (ein Eintrag)
- Modify, durch den Menschen über repin: `_identities.tsv` und die
  `sources:`-Pins der vier Wiki-Seiten

**Interfaces:**
- Consumes: `install.md`, `README.md` (Übersicht) und die Anker von Task 1.
- Produces: nichts.

- [x] **Schritt 1: Den roten Ausgangsstand zeigen**

```sh
uv run --script "$TEAM_RUN_DIR/docs_check.py" links docs/en/configuration.md docs/de/configuration.md
uv run --script "$TEAM_RUN_DIR/docs_check.py" terms docs/en/getting-started.md docs/de/getting-started.md docs/en/configuration.md docs/de/configuration.md
```
Erwartet:
- `links` meldet 2 Befunde (die Anker `…---show--v` in en:112 und de:114).
- `terms` meldet 12 Befunde:
  - en 703/706: G2a, stage
  - en 725: reference implementation
  - en 902: fusion
  - de 725/729: G2a, Stufe
  - de 751: Referenzimplementierung
  - de 935: fusion

- [x] **Schritt 2: `getting-started.md` (en) umbauen**

Neues Gerüst, neu nummeriert:

```markdown
# First-time setup

← [Installing loomux](install.md) · [Overview](README.md) · [Configuration](configuration.md) →

<one paragraph: what this page does; assumes loomux is installed per install.md>

## 1. Initialize your repository
## 2. Review your configuration
## 3. Verify the setup
## 4. Wire your agent
### Claude Code
### Google Antigravity
### Cursor & MCP clients
## 5. Registering an area by hand
### The registry entry
### The manifest
### What goes wrong
## 6. Next steps

← [Installing loomux](install.md) · [Overview](README.md) · [Configuration](configuration.md) →
```

Zuordnung zum heutigen Text:
- §1 ist heute „Step 1" (Zeilen 43-58).
- §2 ist „Step 2" (60-83).
- §3 ist „Step 3" (85-117).
- §4 ist heute §5 (223-285).
- §5 ist heute §4 (119-221).
- §6 ist heute §6, ergänzt um einen Link auf `install.md`; der Rest bleibt
  wörtlich.

Was entfällt:
- §1 „Prerequisites" und §2 „Installation" (7-38) entfallen; sie stehen in
  `install.md`.
- An ihrer Stelle steht im Einleitungsabsatz ein Satz mit Link auf
  `install.md`.
- `loomux` wird in den Beispielen als `loomux` geschrieben, wie heute.

Der Link `configuration.md#4-where-things-live` (heute Zeile 128) bleibt.

- [x] **Schritt 3: `getting-started.md` (de) als Zwilling**

Die Überschriften:
- `# Ersteinrichtung`
- `## 1. Repository initialisieren`
- `## 2. Konfiguration prüfen`
- `## 3. Einrichtung prüfen`
- `## 4. Den Agenten anbinden`
- `## 5. Einen Bereich von Hand einrichten`, darunter `### Der Registry-Eintrag`,
  `### Das Manifest` und `### Was schiefgeht`
- `## 6. Nächste Schritte`

Die Navigation lautet
`← [loomux installieren](install.md) · [Übersicht](README.md) · [Konfiguration](configuration.md) →`.

- [x] **Schritt 4: `configuration.md` (en) ergänzen**

1. Unter dem Titel einfügen:
   - die Navigationszeile
     `← [First-time setup](getting-started.md) · [Overview](README.md)`;
     dieselbe Zeile steht auch am Dateiende;
   - einen Absatz `## Contents` mit gruppierten Links:
     - **Protection**: `[policy.paths]`, `[policy.commands]`, `[guard]`
     - **Checks**: `[verify]`, `.loomux/armed.toml`
     - **Project**: `[project]`, `[modules]`, `[commit]`, `[worktree]`
     - **Areas and wiki**: the area declaration, `[privacy]`
     - **Flows**: `[agent]`, `[flow]`
     - **Where things live** and **the write barrier**

     Jeder Link zeigt auf den bestehenden Anker, den der Implementierer
     mit `links` prüft.
   - einen Abschnitt `## Your first day`:
     - was `loomux init` in `.loomux/config.toml` schreibt (Module,
       Commit-Sprache, Policy-Regeln des Stacks; die Lanes kommen aus den
       Presets);
     - die ersten typischen Änderungen (eine Pfadregel in
       `[policy.paths]`, `loomux check precommit --show`);
     - der Weg einer Änderung:
       `loomux config set <key> <value> --propose` →
       `loomux config proposals` → `loomux config apply`;
     - ein Link auf `cli-reference.md#10-configuration-loomux-config`.

   Beide Überschriften bleiben unnummeriert, damit „1." bis „5." stehen
   bleiben.
2. Zeile 112: Der Anker wird zu `#loomux-check-request---root-path---show--v---arm`.
3. Die Überschrift in Zeile 703 wird zu `### `[graph]` — there is none`.
   Aus dem Absatz verschwinden die Stufen-Bezüge („None of it exists in the
   code stage G2a …" → „None of it exists in the code …"). Die
   Begründungen bleiben.
4. Zeile 725: Aus „— the reference implementation's rule, not a strategy a
   project could turn off" wird „— a fixed rule, not a strategy a project
   could turn off".
5. Zeile 902: `version = "0.1.0-fusion"` wird `version = "1.0.0"`.
6. Unter `### Machine-wide settings` (ab Zeile 992) kommt der Satz aus
   `README.md:283-284` dazu (die Loopback-Pflicht steht dort schon in der
   Tabelle, Proxy und Weiterleitung fehlen): „The endpoint must
   stay on the loopback (`127.0.0.1`, `localhost`, `::1`); no proxy is
   taken from the environment and no redirect is followed."
7. Unter `### [privacy]` kommt ein Satz dazu: „loomux syncs nothing to a
   cloud: knowledge, rules and graphs stay local, and there is no
   telemetry."

- [x] **Schritt 5: `configuration.md` (de) an denselben Stellen**

- Navigation:
  `← [Ersteinrichtung](getting-started.md) · [Übersicht](README.md)`,
  unter dem Titel und am Dateiende.
- `## Inhalt` und `## Der erste Tag`.
- Der Anker in Zeile 114 wird zu `#loomux-check-anfrage---root-pfad---show--v---arm`.
  Den genauen Slug mit `links` gegen `docs/de/cli-reference.md:45` prüfen.
- Die Überschrift in Zeile 725 wird zu `### `[graph]` — den gibt es nicht`.
  In Zeile 729 entfällt „den Stufe G2a ausgeliefert hat".
- In Zeile 751 wird aus „Regel der Referenzimplementierung" bzw. dem
  heutigen Wortlaut „eine feste Regel".
- In Zeile 935 wird die Version `1.0.0`.
- Der Cloud- und Telemetrie-Satz unter `[privacy]` lautet: „loomux synchronisiert nichts in eine Cloud:
  Wissen, Regeln und Graphen bleiben lokal, und es gibt keine Telemetrie."
- Der Satz zu Loopback und Proxy kommt aus `README.de.md` (dem Gegenstück
  zu `README.md:283-284`).

- [x] **Schritt 6: Zitierende Wiki-Seiten prüfen und Log-Eintrag**

1. Die vier Seiten lesen:
   - `docs/wiki/entities/qmd.md`
   - `topics/datenschutz-und-kanaele.md`
   - `topics/schreibschranke.md`
   - `topics/datenmodell-und-bereiche.md`

   Jede Aussage, die sich auf `docs/de/getting-started.md` oder
   `docs/de/configuration.md` stützt (Abschnittsnamen, Nummern, Sätze),
   gegen den neuen Text halten. Was nicht mehr stimmt, wird in der Prosa
   angepasst. Die `sources:`-Blöcke bleiben unangetastet.
   (`entities/qmd.md` nennt seinen Pin nur im Frontmatter.)
2. Zwei Wiki-Seiten verweisen auf Roadmap-Kürzel, die Task 3 entfernt hat.
   Beide Seiten sind nicht zitiert, eine Änderung erzeugt also keinen
   Befund. Ihre Prosa wird so angepasst:
   - `docs/wiki/topics/code-graph.md:156-158` („ordnete die Spec G4 und G5
     zu; die Roadmap im README führt beide heute unter dem Web-OS (W3)")
     → „… die Roadmap im README führt beide heute unter dem
     Graph-Visualizer". Die Stufen der Spec im ersten Halbsatz sind
     Wiki-Historie und bleiben.
   - `docs/wiki/topics/scheiben-und-abnahme.md:48` („Offene Stufen führt die
     Roadmap im README") → „Offene Arbeit führt die Roadmap im README".
     Der Sinn des Satzes bleibt.
3. In `docs/wiki/log.md` einen Eintrag im Format der bisherigen Einträge
   anlegen, mit dem Datum des Tages. Er sagt: Ersteinrichtung und
   Konfiguration wurden neu gegliedert, die Installation ist nach
   `docs/de/install.md` gewandert, und die zitierenden Seiten wurden
   geprüft (welche angepasst wurden).

- [x] **Schritt 7: Prüfen, ohne zu committen**

```sh
for f in getting-started configuration; do uv run --script "$TEAM_RUN_DIR/docs_check.py" parity docs/en/$f.md docs/de/$f.md; done
uv run --script "$TEAM_RUN_DIR/docs_check.py" links docs/en/getting-started.md docs/de/getting-started.md docs/en/configuration.md docs/de/configuration.md README.md README.de.md docs/en/*.md docs/de/*.md
uv run --script "$TEAM_RUN_DIR/docs_check.py" terms docs/en/getting-started.md docs/de/getting-started.md docs/en/configuration.md docs/de/configuration.md
go test ./internal/namecheck/
go run ./cmd/loomux check lint
```
Erwartet:
- `parity`, `links` und `terms` je mit `0 finding(s)`,
- `ok` für namecheck,
- in `check lint` ist **nur** `lint/wiki-sources` rot, und zwar mit genau
  zwei Befunden `[wiki-sources:register-stale]`: einer für
  `docs/de/getting-started.md` (cited by `entities/qmd.md`) und einer für
  `docs/de/configuration.md` (cited by den drei Topic-Seiten).
- Jede andere Lane ist `ok` oder `not-applicable`.

- [x] **Schritt 8: Repin-Pause**

Nicht committen und kein `--no-verify` verwenden. Den Task mit TaskUpdate
auf `pending` setzen; die Beschreibung beginnt mit:

```
WAITING: please run in <absolute path of this worktree>:
  go run ./cmd/loomux wiki repin --root "<absolute path of this worktree>"
Changed cited files: docs/de/getting-started.md, docs/de/configuration.md.
Citing pages checked: entities/qmd.md, topics/datenschutz-und-kanaele.md,
topics/schreibschranke.md, topics/datenmodell-und-bereiche.md (<which were adjusted>).
```

Dieselbe Nachricht geht per SendMessage an den Lead. Er reicht sie an den
Menschen weiter. Warten, bis der Lead meldet, dass repin gelaufen ist.

- [x] **Schritt 9: Nach dem Repin prüfen und committen**

```sh
git status --short
go run ./cmd/loomux check lint
sh ci/gate.sh
```
Erwartet:
- `git status` zeigt zusätzlich `_identities.tsv` und die vier Wiki-Seiten
  (nur ihre `content_hash`- und `revision`-Zeilen),
- `lint/wiki-sources: ok`,
- das Gate endet mit exit 0.

Ein Commit. Doku und Repin bilden eine Änderung, und nur so ist jeder
Commit für sich grün:

```sh
git add docs/en/getting-started.md docs/de/getting-started.md docs/en/configuration.md docs/de/configuration.md docs/wiki _identities.tsv
git commit -m "docs: focus getting started on first-time setup and guide into the configuration"
git status --porcelain
```
Erwartet: Der Commit läuft durch den Pre-Commit-Hook (Gate grün), und
`git status --porcelain` ist leer.

---

## Für den Menschen (nach dem Merge, kein Agent führt das aus)

1. Repo-Metadaten setzen:

```sh
gh repo edit xidus90/loomux \
  --description "One Go binary that guards AI coding agents: hooks and a write barrier, a check chain, a code graph and a project wiki. Local, no Python or Node runtime." \
  --homepage "https://github.com/xidus90/loomux/tree/master/docs/en" \
  --add-topic mcp,developer-tools,cli,coding-agents,guardrails
gh repo view xidus90/loomux --json description,homepageUrl,repositoryTopics
```

Erwartet: 12 Topics (die 7 bisherigen plus 5 neue), die neue
Beschreibung und die Homepage.

2. Social Preview:
   1. `docs/assets/social-preview.svg` im Browser öffnen, bei Zoom 100 % und
      Windows-Anzeigeskalierung 100 %, oder in den DevTools mit
      devicePixelRatio 1. Sonst wird der Screenshot z. B. 2560×1280.
   2. Als PNG mit 1280×640 und unter 1 MB exportieren, per Screenshot des
      Elements oder mit einem SVG-Konverter.
   3. GitHub → Settings → General → Social preview → Edit → „Upload an
      image…".

3. Der Pull Request trägt das Label `release:none`. Im Rumpf steht
   `Release: none — documentation, repository presentation and a test
   comment only` und kein `## Changelog`.

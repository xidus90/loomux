# Stufe 4f PR C: Aufzeichnungen und Tor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die Aufzeichnungen der Vorgänger verlassen den Baum, gesichert als Archiv. Die übersetzten Fälle tragen keine Altnamen mehr, ohne dass sich ändert, was die Wiedergabe vergleicht. Ein Tor-Test hält die Suchliste aus #24 künftig auf ihre benannten Ausnahmen.

**Architecture:** Zwei Blöcke.
- **Block 1 (Tasks 1–5)** läuft auf `refactor/drop-reference-recordings`, gestapelt auf #81 und ohne PR B: Policy, Sichern, Umzug, Löschen, Erwartungen.
- **Block 2 (Tasks 6–7)** folgt, wenn PR B auf `master` liegt und der Zweig darauf gesetzt ist: Doku und der Tor-Test. Der Tor-Test ist der letzte Commit, weil er vorher rot wäre und jeder Commit durch das Tor läuft.

**Tech Stack:** Go, git, Python (Ersetzungsskript, PEP-723 per `uv run --script`), `gh` für das Archiv (nur der Mensch lädt hoch).

**Spec:** `docs/.superpowers/specs/2026-10-05-loomux-stufe-4f-design.md`, Abschnitt „PR C: Aufzeichnungen und Tor“. Grundlage für Task 5 ist die Akte `docs/.superpowers/parity/stufe-4f.md`, Abschnitt 4 („Was die Wiedergabe vergleicht“). Nachträge #24 (e), #32, #33, #34 der Fusions-Spec.

## Global Constraints

- Suchliste (#24), case-insensitive: `ultraloom`, `ultra-brain`, `ulguard`, `ulinit`, `ulflow`, `brain-mcp`, `brain guard`, `ultraLoomOwned`, `ultraloom-wiki-guard`, `.ultraloom`, `.brain.toml`, `.ultra-brain`.
- `.loomux/config.toml` schreibt nie ein Agent (AGENTS.md); Task 1 macht der Mensch.
- PR B (`refactor/drop-predecessor-references`, andere Sitzung) fasst nicht an: `internal/dev/importcases`, `internal/dev/recordcase`, `dev record-case`/`record-mcp-case` samt Tests in `internal/cli/dev_test.go` und `devmcp_test.go`, `testdata/cases` außer den elf `world_after/xdg/qmd/index.yml` von 3a/reindex und 3b/approve, die Orakel-Skripte, den Tor-Test, die Bündel. Umgekehrt fasst PR C nichts an, was in B steht (Code und Doku der Klassen a–d, f, g aus #24, `index.AlwaysExcludes`, `gitfiles.RunsAGate`).
- Was die Wiedergabe vergleicht, steht in der Akte, Abschnitt 4. Eine Datei, die verglichen wird, ändert sich nur so, dass der Vergleich gleich bleibt.
- 100 % Coverage je Funktion; jeder neue Test läuft vor seinem Code rot; eine Mutationsrunde für den Tor-Test (Task 7).
- Kommentare und Commits englisch, ohne Arbeitspapier-Namen; kein `Co-Authored-By`.
- Commit-Ausgaben in eine Datei im Scratchpad, nie nach `/dev/null`. Code mit Backslashes nur per Write/Edit.
- Release-Stufe des PR: `release:major` (`dev import-cases`, `dev record-case`, `dev record-mcp-case` fallen weg).

## Review Focus

1. **Eine Erwartungsdatei, die verglichen wird, ändert ihren Vergleichswert.** Etwa `git.toml` nur in `world/`, aber nicht in `world_after/`, oder eine `stdout` eines `data`-Falls. Erwartet: Die Wiedergabe-Suiten (`go test ./internal/cli -run TestCases`) bleiben grün, und das Prüfskript in Task 5 meldet nach dem Maskieren der Altnamen keinen anderen Unterschied.
2. **Ein Test oder Werkzeug liest noch einen gelöschten Ordner.** Bekannt ist nur `internal/hosts/antigravity_test.go:27`. Erwartet: Nach Task 4 sind `go vet ./...` und `go test ./...` grün, und `git grep` nach `-source/`, `-worlds/`, `-payloads/`, `-map.toml` trifft nur noch Arbeitspapiere.
3. **Der Tor-Test lässt eine Ausnahme zu breit zu.** Beispiel: „ganz `CHANGELOG.md`“ statt fünf Zeilen, oder `docs/.superpowers/` samt einem neuen Unterordner, der kein Arbeitspapier ist. Erwartet: Ein Unit-Test je Ausnahmeart zeigt einen Treffer knapp außerhalb der Ausnahme als Befund.
4. **Ein Changelog-Abschnitt in der künftigen Form `## [7.0.1-beta] - …`** (Neustart-Schritt 3b) bricht den Tor-Test. Erwartet: Ein Unit-Test mit dieser Überschrift ist grün.
5. **Das Archiv ist unvollständig oder weicht ab.** Erwartet:
   - Die Bündel werden in Task 2 neu gebaut.
   - `git bundle verify` ist grün.
   - Ein Wiederherstellen per `git clone --mirror` vergleicht jede Ref.
   - `SHA256SUMS` liegt bei.
   - Die Befehle für den Menschen stehen wörtlich im Plan.

---

### Task 1: Policy freigeben (Mensch)

**Files:**
- Der Mensch ändert: `.loomux/config.toml`.

- [ ] **Step 1: Dem Menschen nennen, was er entfernt.** Die vier Blöcke `[[policy.paths.rules]]` mit `match = ["testdata/cases/1a-source/**"]`, `…/1b-1-source/**`, `…/2a-source/**`, `…/2b-source/**`, jeweils mit der Zeile `reason = "Recordings of the old tools are evidence; …"`. Dazu formuliert er den Kommentar über `[[policy.commands.rules]]` neu, der „(in ultraloom fiel genau so eine Regel jahrelang still aus)“ sagt, etwa „(eine solche Regel fiel früher jahrelang still aus)“. `config unset` erreicht einzelne Einträge eines Tabellen-Arrays nicht, darum geht es von Hand.
- [ ] **Step 2: Wirkung prüfen.** Nach dem Speichern lädt das Tor die Datei (`go run ./cmd/loomux check precommit --show` endet mit Exit 0). Ein Probe-Write auf `testdata/cases/1a-source/x` wird vom Wächter nicht mehr verweigert; dazu dient eine PreToolUse-Nutzlast durch `bin/loomux.exe hook pre-tool-use`, nicht das echte Schreiben.
- [ ] **Step 3: Commit (Mensch oder Agent nach Freigabe):** `chore(config): stop guarding the recordings of the predecessor tools`. Die Datei committet der Mensch. Der Agent nennt nur die Nachricht.

### Task 2: Sichern (Archiv-Tag, Bündel, ulflow-Pläne)

**Files:**
- Create: `docs/.superpowers/plans-ul/2026-09-11-ulflow-welle-0-und-1.md`, `docs/.superpowers/plans-ul/2026-09-11-ulflow-welle-2.md`, `docs/.superpowers/plans-ul/2026-09-14-ulflow-welle-3.md` (aus ultraloom `feature/agent-harness`)
- Modify: `docs/.superpowers/parity/stufe-4f.md` (Abschnitt 5 „Archiv“)
- Scratchpad: `archiv/` (Bündel, Tarballs, Patches, `SHA256SUMS`, `release-notes.md`)

- [ ] **Step 1: Pläne übernehmen**, byte-gleich:

```bash
U="C:/Users/micro/Documents/#GIT/ultraloom"
for f in 2026-09-11-ulflow-welle-0-und-1.md 2026-09-11-ulflow-welle-2.md 2026-09-14-ulflow-welle-3.md; do
  MSYS_NO_PATHCONV=1 git -C "$U" show "feature/agent-harness:docs/.superpowers/plans/$f" > "docs/.superpowers/plans-ul/$f"
done
```

  Danach je Datei ein `git -C "$U" hash-object` gegen `git hash-object` der Kopie vergleichen. Die Hashes müssen gleich sein.

- [ ] **Step 2: Bündel neu bauen.**
  - Für jedes der beiden Repos: `git bundle create … --all`, danach `git bundle verify`.
  - Prüfen per `git clone --mirror` aus dem Bündel: Die Liste von `for-each-ref` (ohne `refs/stash`) ist gleich der des Repos.
  - Dazu je Repo:
    - ein Tar der ungetrackten Dateien (`git ls-files -o --exclude-standard -z`, `tar --force-local -T`),
    - ein Tar von `.superpowers/` und `.claude/settings.local.json`,
    - ein `git diff` als Patch.
  - Am Ende `SHA256SUMS`.
  - Das Verfahren steht in der Akte. Die Werte von 2026-10-05 dienen als Vergleich: Das ultraloom-Bündel trägt 11 Refs plus 4 Worktree-Köpfe, das ultra-brain-Bündel 13 plus 7.

- [ ] **Step 3: Archiv-Commit festlegen und Befehle nennen.** Der Tag zeigt auf den Commit von `origin/master`, der die Aufzeichnungen zuletzt trägt, also den aktuellen Kopf von `origin/master` zum Zeitpunkt dieses Tasks (`git rev-parse origin/master`). Der Agent schreibt `archiv/release-notes.md`, einen Absatz Englisch: was das Archiv enthält und wie man ein Bündel wiederherstellt. Dem Menschen nennt er wörtlich:

```bash
git -C "C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/4f-angehen-e5b020" tag archive/parity-recordings <sha>
git -C "C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/4f-angehen-e5b020" push origin archive/parity-recordings
gh release create archive/parity-recordings --repo xidus90/loomux --title "Archive: parity recordings and reference repositories" --notes-file <scratchpad>/archiv/release-notes.md <scratchpad>/archiv/*.bundle <scratchpad>/archiv/*.tar <scratchpad>/archiv/*.patch <scratchpad>/archiv/SHA256SUMS
```

  Der Tag beginnt nicht mit `v`. Darum übergeht `release.NextVersion` ihn (`tagPattern`), und `selfupdate` liest ihn nicht als Version (`parseVersion` scheitert, `pick` überspringt ihn).

- [ ] **Step 4: Akte, Abschnitt 5 „Archiv“.** Sie hält fest:
  - Tag und Commit,
  - je Datei Größe und SHA-256,
  - das Ergebnis der Wiederherstellung,
  - die beiden verwaisten Worktrees unter `C:/Users/micro/orca/workspaces/` als Schritt für den Menschen beim Löschen der Repos (`git -C <repo> worktree remove <pfad>` oder der Ordner selbst),
  - die zwei Wurzelquellen, die der Mensch nach `brain-knowledge/00 Eingang` legt.
- [ ] **Step 5: Commit** — `docs: keep the ulflow plans and record the archive of the predecessor repositories`. Das Hochladen bestätigt der Mensch. Danach prüft `gh release view archive/parity-recordings --json assets` die Dateiliste gegen `SHA256SUMS`.

### Task 3: Umzug der Antigravity-Nutzlasten

**Files:**
- Move: `testdata/cases/2c-payloads/agy-stop.json`, `agy-inv.json`, `agy-pre.json`, `agy-post.json` → `internal/hosts/testdata/agy-*.json`. Die genauen Namen liest der Implementierer aus `antigravity_test.go:27` und `ls testdata/cases/2c-payloads`.
- Modify: `internal/hosts/antigravity_test.go:27`

- [ ] **Step 1:** `ls testdata/cases/2c-payloads` gegen die Namen halten, die der Test liest. Was der Test nicht liest, bleibt für Task 4 liegen.
- [ ] **Step 2:** Mit `git mv` umziehen und den Pfad im Test auf `filepath.Join("testdata", filename)` setzen. Der Test liegt im Paket `internal/hosts`, also ist `testdata` relativ zum Paket.
- [ ] **Step 3:** `go test ./internal/hosts -count=1` → PASS. Gegenprobe: Mit dem alten Pfad und gelöschtem Altordner (lokal, nicht committet) ist er rot.
- [ ] **Step 4: Commit** — `test(hosts): keep the Antigravity payloads beside the tests that read them`

### Task 4: Aufzeichnungen und Werkzeuge entfernen

**Files:**
- Delete: `testdata/cases/*-source/`, `testdata/cases/*-map.toml`, `testdata/cases/*-worlds/`, `testdata/cases/*-payloads/` (Rest), `testdata/cases/2a-extra-answers.json` (falls vorhanden; `ls`), `internal/dev/importcases/`, `internal/dev/recordcase/`, `docs/.superpowers/parity/stufe-*-orakel/`
- Modify: `internal/cli/dev.go` (Befehle `import-cases`, `record-case`, `record-mcp-case` samt Usage und Hilfefunktionen), `internal/cli/dev_test.go`, `internal/cli/devmcp_test.go` (die Tests dieser Befehle), `.gitattributes` (Zeilen 8, 49, 65, 75–76 laut Akte; vorher `git grep -n -E "source|worlds|payloads|map.toml" .gitattributes`), `testdata/cases/README.md`, `docs/en/cli-reference.md` und `docs/de/cli-reference.md` (Abschnitte und Erwähnungen von `dev import-cases`, `dev record-case`, `dev record-mcp-case`)

- [ ] **Step 1: Aufrufer listen und festhalten.** In den Bericht kommen diese vier Suchen:
  - `git grep -n -E "importcases|recordcase" -- '*.go'`
  - `git grep -n -E "import-cases|record-case|record-mcp-case" -- '*.go' '*.md' ':!docs/.superpowers/**' ':!CHANGELOG.md'`
  - `git grep -n -E -- "-source/|-worlds/|-payloads/|-map\.toml" -- '*.go'`
  - `git grep -n "fakeqmd\|fake-ollama\|record-poppler"`

  Bleiben müssen `internal/dev/fakeqmd`, `dev fake-ollama`, `dev record-poppler` und `internal/cases`, weil die Wiedergabe sie braucht. Sie fallen nur, wenn außer `importcases`/`recordcase` niemand sie ruft. Dann ist das ein Befund: Bericht, kein Löschen.
- [ ] **Step 2: Löschen** per `git rm -r` für die Ordner und Pakete oben. In `internal/cli/dev.go` die drei Fälle aus der Unterbefehlstabelle und ihre Funktionen entfernen, in den beiden Testdateien die Tests dieser Befehle.
- [ ] **Step 3: `testdata/cases/README.md` neu fassen.**
  - Die Tabelle der Quell-, Welt-, Nutzlast- und Map-Ordner fällt.
  - Was bleibt, ist ein kurzer Absatz: Jeder Ordner `<stufe>/` hält die Fälle einer Stufe, je mit `cmd`, `exit`, `stdout`, `world/` und optional `world_after/` und `compare`.
  - Die Erwartungen sind loomux' eigene; woher sie kommen, steht im Archiv `archive/parity-recordings`.
  - Kein Altname.
- [ ] **Step 4: CLI-Referenz** (en/de, gleicher Inhalt): Die Abschnitte der drei Befehle und ihre Erwähnungen in der Übersicht von `dev` fallen. Ein Satz verweist auf das Archiv. PR B ändert dieselben Dateien an anderen Stellen; beim Rebase in Block 2 wird je Zeile entschieden, nie eine Seite ganz genommen.
- [ ] **Step 5:** `go vet ./...` → ok, `go test ./... -count=1` → PASS. Die Coverage je Funktion bleibt bei 100 %: `go test ./internal/cli -coverprofile`, gelesen nur für `dev.go`.
- [ ] **Step 6: Commit** — `refactor(dev)!: drop the recordings of the predecessor tools and the commands that made them`, mit Footer `BREAKING CHANGE: loomux dev import-cases, dev record-case and dev record-mcp-case are gone; the recordings live in the archive release archive/parity-recordings.`

### Task 5: Eigene Erwartungen ohne Altnamen

**Files:**
- Create: `<scratchpad>/neutral.py` (Ersetzungsskript, nicht im Repo)
- Modify: `testdata/cases/**/notes.md` (335), `testdata/cases/**/stderr`, `testdata/cases/2a/**/stdout` und `testdata/cases/3a/area-add/**/stdout`, also die nicht verglichenen Fälle `lanes`/`message`, dazu `testdata/cases/1a/hook-pre-tool-use/barrier-refuses-manifest/stdout`, `testdata/cases/1b-2/brain-*/missing-manifest/result`, `testdata/cases/2a/**/world/README.md`, `testdata/cases/4a2/hook/*/world/git.toml` und `…/world_after/git.toml`
- Delete: `testdata/cases/3a/area-add/*/world_after/repo-new/.mcp.json` (4), `testdata/cases/3a/area-add/known-scope/world_after/repo-new/.ultra-brain/config.toml`
- Modify: `internal/cli/cases_3a_test.go` (die Toleranzen `noMCPJSON` und `missing file in actual: repo-new/.ultra-brain/config.toml` fallen mit den Dateien)
- Prüfen, dann löschen oder umbenennen: `testdata/cases/4d/convert/no-registry/world/vault/.brain.toml`

- [ ] **Step 1: Stand aufnehmen.** Auf derselben Basis wie die Suche aus #24 (ohne `-source` usw., die sind ab Task 4 weg) die Liste der betroffenen Dateien nach `<scratchpad>/c5-vorher.txt` schreiben. Ein schneller Test für den Tor-Test aus Task 7: Sein Läufer meldet hier jede dieser Dateien.

- [ ] **Step 2: Ersetzungstabelle.** Je Zeile ein Muster und ein Ersatz, angewendet in dieser Reihenfolge. Die Muster sind wörtlich, case-sensitiv, Backslashes per Write. Die Liste ist ein Boden: Was nach dem Lauf noch trifft, ordnet der Implementierer einer neuen Zeile zu oder meldet es.

| Muster | Ersatz | Wo |
|---|---|---|
| `ultraloom at tag loomux-1a-source (` | `the reference at tag loomux-1a-source (` | `notes.md` |
| `, brain-mcp mcp over fakeqmd` | `, the reference's MCP front over fakeqmd` | `notes.md` |
| `, brain-mcp over fakeqmd and fake-ollama:` | `, the reference over fakeqmd and fake-ollama:` | `notes.md` |
| `, brain-mcp over fakeqmd:` | `, the reference over fakeqmd:` | `notes.md` |
| `, brain-mcp:` | `, the reference:` | `notes.md` |
| `ultraloom does not enforce it` | `loomux does not enforce it` | `2a/**/stdout` (nur `lanes`) |
| `A project of no language ultraloom knows` | `A project of no language loomux knows` | `2a/**/world/README.md` |
| `\.ultraloom\config.toml` | `\.loomux\config.toml` | `2a/**/stdout`, `notes.md` |
| `\repo-new\.ultra-brain\config.toml` | `\repo-new\.loomux\config.toml` | `3a/area-add/**/stdout` (nur `message`) |
| `as brain-mcp hook install writes it` | `as the reference's hook install writes it` | `4a2/hook/*/world/git.toml` und `…/world_after/git.toml`, beide |
| `\\\\repo-bare\\\\.brain.toml` | `\\\\repo-bare\\\\.loomux\\\\config.toml` | `1b-2/brain-*/missing-manifest/result` (nur `isError` verglichen) |
| `usage: brain-mcp ` | `usage: loomux ` | `**/stderr` |
| `brain-mcp` | `the reference` | `**/stderr`, `notes.md` (Rest) |
| `ultraloom` | `the reference` | `**/stderr`, `notes.md` (Rest) |
| `ultra-brain` | `the reference` | `notes.md` (Rest) |

- [ ] **Step 3: Skript schreiben und fahren.**
  - `neutral.py`, mit PEP-723-Kopf und ohne Abhängigkeiten. Es liest die Tabelle als Liste in sich selbst und wendet je Datei nur die Zeilen an, deren Spalte „Wo“ auf den Pfad passt (Glob).
  - Es schreibt mit `newline="\n"` und erhält die Bytes sonst gleich.
  - Am Ende druckt es je Muster die Zahl der Ersetzungen.
  - Vorher ein Kontrollfall: Ein Fall in einem Scratch-Ordner mit `---`-Zeile, Umlaut im Pfad und einem Muster am Zeilenende wird erwartungsgemäß umgeschrieben (Lernmuster vom 2026-10-05).
- [ ] **Step 4: Dateien mit Altpfad.**
  - `git rm` der vier `.mcp.json` und der `.ultra-brain/config.toml` unter `3a/area-add/*/world_after/repo-new/`.
  - In `internal/cli/cases_3a_test.go` die Konstante `noMCPJSON` und ihre Verwendungen samt der Zeile `"missing file in actual: repo-new/.ultra-brain/config.toml"` entfernen. Die erwartete Abweichung entfällt, weil die Datei fehlt.
  - Für `4d/convert/no-registry/world/vault/.brain.toml` die `notes.md` des Falls lesen. Braucht der Fall nur „eine Datei liegt da“, fällt sie. Trägt sie Bedeutung, etwa als altes Manifest, das ignoriert werden soll, wird sie zu `vault/old-manifest.toml` und `notes.md` sagt es.
- [ ] **Step 5: Prüfen, dass sich nur Namen geändert haben.** Das Prüfskript (`classify.py` aus PR A, Fassung 3, im Scratchpad) bekommt die Altnamen und ihre Ersatztexte als Maske dazu. Für `testdata/cases` darf danach keine Datei als Klasse (c) bleiben, außer den gelöschten Pfaden aus Step 4.
- [ ] **Step 6: Wiedergabe.** `go test ./internal/cli -run 'TestCases' -count=1` → PASS, ebenso `go test ./internal/cases ./internal/hosts -count=1`.
- [ ] **Step 7: Rest.** Die Suchliste über `testdata/cases` darf nichts mehr treffen:

```bash
git grep -l -i -E "ultraloom|ultra-brain|ulguard|ulinit|ulflow|brain-mcp|brain guard|ultraLoomOwned|\.brain\.toml|\.ultra-brain" -- testdata/cases
```

  Erwartet: nur die `world_after/xdg/qmd/index.yml`, die PR B mitzieht. Liegt B schon auf der Basis, trifft auch das nichts.
- [ ] **Step 8: Commit** — `test(cases): state the replay expectations without the predecessor names`

### Task 6: Block 2 vorbereiten — Rebase auf `master` mit PR B, Doku

**Files:**
- Modify: `AGENTS.md` (Zeile „Recorded behaviour of the old tools lives under `testdata/cases/`.“), `README.md`, `README.de.md` (nur, was Aufzeichnungen und `dev import-cases` nennt; `grep -n -i "import-cases\|recorded behaviour\|recording" README*.md`), `docs/en/migration.md`, `docs/de/migration.md` (Zeile 4f, Text „was bleibt“; Status bleibt offen, Priorität 3), `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` (nur falls B dort Nachtrag #35 anlegt: die Ausnahme aus #35 in die Liste „Die benannten Ausnahmen“, falls B das nicht schon getan hat)

- [ ] **Step 1:** Erst wenn PR B gemergt ist: `git fetch` und `git rebase origin/master`. Bei Konflikten in `cli-reference.md`, `dev.go` oder `migration.md` je Zeile entscheiden (Lernmuster Autosquash). Danach müssen `git diff --stat` gegen den alten Kopf und die Dateiliste die eigenen Änderungen zeigen.
- [ ] **Step 2: Doku nachziehen.** Der Satz in `AGENTS.md` wird zu „The replay cases of every stage live under `testdata/cases/`; the recordings they came from are in the archive release `archive/parity-recordings`.“ READMEs und Migrationsplan bekommen je einen Satz: Aufzeichnungen gesichert und entfernt, Tor-Test folgt.
- [ ] **Step 3:** `go test ./internal/plancheck -count=1` → PASS. Commit — `docs: point to the archive of the recordings`

### Task 7: Tor-Test gegen Verweise auf die Vorgänger

**Files:**
- Create: `internal/plancheck/references.go`, `internal/plancheck/references_test.go`
- Modify: `internal/setup/templates/templates_test.go:55` und `:147`. Die Einträge `ulguard`, `.ultraloom`, `brain-mcp`, `ultraloom` fallen aus den Listen, die übrigen (`GEMINI.md`, `.agents/`, `uv`, `shim`, `uv run`) bleiben. Der Tor-Test deckt die Vorlagen jetzt mit ab, weil sie getrackte Dateien sind.

**Interfaces:**
- Produces:
  - `type Exception struct { Path string; Line *regexp.Regexp; Section string; Owner string }`. `Path` ist ein Pfad oder ein Präfix mit `/` am Ende. `Line` (optional) lässt nur passende Zeilen zu. `Section` (optional) lässt nur Treffer unter einer Überschrift `## [<Section>]` oder `## [<Section>-beta]` zu, je höchstens einen. `Owner` ist einer von `history`, `plan-end`, `flow`, `web`, `working-papers`, `self`.
  - `func References(files []string, read func(string) ([]byte, error), exceptions []Exception) ([]string, error)`. Ergebnis: je Fund eine Zeile `<pfad>:<zeile>: <text>`, sortiert.
  - `var exceptions = …`, die Liste des Repos.
  - `TestTheRepositoryNamesNoPredecessor` liest `git ls-files` und den Index.

- [ ] **Step 1: Unit-Tests zuerst** (in `references_test.go`):

```go
func TestReferencesFindsEveryNameOfTheList(t *testing.T) {
	files := map[string]string{
		"a.go":  "x := \"UltraLoom\"\n",
		"b.md":  "see brain guard and .Brain.toml\n",
		"c.txt": "nothing here\n",
	}
	got, err := References(keys(files), reader(files), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.go:1: x := \"UltraLoom\"", "b.md:1: see brain guard and .Brain.toml"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAPathExceptionCoversOnlyItsFileOrFolder(t *testing.T) {
	files := map[string]string{
		"docs/en/benchmarks.md":       "ulguard 12 ms\n",
		"docs/en/benchmarks-notes.md": "ulguard\n",
		"docs/.superpowers/specs-ul/x.md": "ultraloom\n",
		"docs/.superpowers-x/y.md":        "ultraloom\n",
	}
	ex := []Exception{
		{Path: "docs/en/benchmarks.md", Owner: "history"},
		{Path: "docs/.superpowers/specs-ul/", Owner: "flow"},
	}
	got, _ := References(keys(files), reader(files), ex)
	want := []string{"docs/.superpowers-x/y.md:1: ultraloom", "docs/en/benchmarks-notes.md:1: ulguard"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestALineExceptionCoversOnlyMatchingLines(t *testing.T) {
	files := map[string]string{"README.md": "| Flow | ulflow follow-up |\nrun ulflow now\n"}
	ex := []Exception{{Path: "README.md", Line: regexp.MustCompile(`^\| Flow \|`), Owner: "flow"}}
	got, _ := References(keys(files), reader(files), ex)
	if want := []string{"README.md:2: run ulflow now"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestASectionExceptionAllowsOneHitPerNamedEntry(t *testing.T) {
	changelog := "# Changelog\n\n## [7.2.0] - 2026-10-05\n- uses ultraloom\n\n" +
		"## [7.0.1-beta] - 2026-10-04\n- reads .brain.toml\n- and .ultra-brain\n\n" +
		"## [2.3.0] - 2026-09-22\n- flags ulguard\n"
	files := map[string]string{"CHANGELOG.md": changelog}
	ex := []Exception{
		{Path: "CHANGELOG.md", Section: "7.0.1", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "2.3.0", Owner: "history"},
	}
	got, _ := References(keys(files), reader(files), ex)
	want := []string{"CHANGELOG.md:4: - uses ultraloom", "CHANGELOG.md:8: - and .ultra-brain"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestReferencesPassesAReadErrorOn(t *testing.T) {
	broken := func(string) ([]byte, error) { return nil, errors.New("gone") }
	if _, err := References([]string{"a"}, broken, nil); err == nil || !strings.Contains(err.Error(), "a: gone") {
		t.Fatalf("err = %v", err)
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func reader(m map[string]string) func(string) ([]byte, error) {
	return func(p string) ([]byte, error) { return []byte(m[p]), nil }
}
```

- [ ] **Step 2: Rot.** `go test ./internal/plancheck -run 'References|Exception' -count=1` → Build-Fehler. Danach mit einem Stub (`return nil, nil`) `--- FAIL` in jedem der fünf Tests. Befehl und Ausgabe kommen in den Bericht.

- [ ] **Step 3: Umsetzen** (`references.go`):

```go
// The names the fusion spec's addendum #24 searched for when it took stock of
// what loomux still owes its two predecessors; every one that remains has to
// sit in a named exception.
var predecessorNames = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)ultraloom|ultra-brain|ulguard|ulinit|ulflow|brain-mcp|brain guard|ultraloomowned|\.brain\.toml|\.ultra-brain`)
})

var changelogEntry = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^## \[(\d+\.\d+\.\d+)(?:-beta(?:\.\d+)?)?\]`)
})

// An Exception lets a name stand where the spec says it may: a file, or every
// file below a folder when Path ends in "/"; only lines Line matches, when it
// is set; only one line under the changelog entry Section, when that is set.
type Exception struct {
	Path    string
	Line    *regexp.Regexp
	Section string
	Owner   string
}

func (e Exception) covers(path string) bool {
	if strings.HasSuffix(e.Path, "/") {
		return strings.HasPrefix(path, e.Path)
	}
	return path == e.Path
}

// References names every line of files that carries a predecessor's name and
// no exception allows, as "<path>:<line>: <text>".
func References(files []string, read func(string) ([]byte, error), exceptions []Exception) ([]string, error) {
	var found []string
	for _, path := range files {
		data, err := read(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		found = append(found, inFile(path, string(data), exceptions)...)
	}
	slices.Sort(found)
	return found, nil
}

func inFile(path, text string, exceptions []Exception) []string {
	var own []Exception
	for _, e := range exceptions {
		if e.covers(path) {
			own = append(own, e)
		}
	}
	var found []string
	section, used := "", map[string]bool{}
	for i, line := range strings.Split(text, "\n") {
		if m := changelogEntry().FindStringSubmatch(line); m != nil {
			section = m[1]
		}
		if !predecessorNames().MatchString(line) || allowed(own, line, section, used) {
			continue
		}
		found = append(found, fmt.Sprintf("%s:%d: %s", path, i+1, strings.TrimRight(line, "\r")))
	}
	return found
}

// allowed reports whether one of the file's exceptions takes the line; a
// section exception takes the first hit under its entry and no other.
func allowed(own []Exception, line, section string, used map[string]bool) bool {
	for _, e := range own {
		switch {
		case e.Section != "":
			if e.Section == section && !used[section] {
				used[section] = true
				return true
			}
		case e.Line != nil:
			if e.Line.MatchString(line) {
				return true
			}
		default:
			return true
		}
	}
	return false
}
```

`.ultraloom` deckt das Muster `ultraloom` ab, `ultraLoomOwned` und `ultraloom-wiki-guard` ebenso. Der Regex nennt `ultraloomowned` trotzdem einzeln, damit die Liste der Spec wörtlich lesbar bleibt. Ein Mutant, der den Eintrag streicht, ist äquivalent und wird so im Bericht vermerkt.

- [ ] **Step 4: Grün** — `go test ./internal/plancheck -run 'References|Exception' -count=1` → PASS.

- [ ] **Step 5: Liste des Repos und Repository-Test** (in `references.go` bzw. `references_test.go`):

```go
// exceptions are the places the fusion spec lets a predecessor's name stand
// ("Die benannten Ausnahmen", addenda #24, #34, #35). A follow-up project
// that finishes removes its own rows.
var exceptions = []Exception{
	{Path: "docs/en/benchmarks.md", Owner: "history"},
	{Path: "docs/de/benchmarks.md", Owner: "history"},
	{Path: "testdata/bench/1a-hooks.json", Owner: "history"},
	{Path: "testdata/bench/search/v1/baseline/", Owner: "history"},
	{Path: "CHANGELOG.md", Section: "7.0.1", Owner: "history"},
	{Path: "CHANGELOG.md", Section: "7.0.0", Owner: "history"},
	{Path: "CHANGELOG.md", Section: "4.2.2", Owner: "history"},
	{Path: "CHANGELOG.md", Section: "2.5.0", Owner: "history"},
	{Path: "CHANGELOG.md", Section: "2.3.0", Owner: "history"},
	{Path: "docs/en/migration.md", Owner: "plan-end"},
	{Path: "docs/de/migration.md", Owner: "plan-end"},
	{Path: "docs/.superpowers/", Owner: "working-papers"},
	{Path: "README.md", Line: regexp.MustCompile(`ulflow|ultra-brain/web`), Owner: "flow"},
	{Path: "README.de.md", Line: regexp.MustCompile(`ulflow|ultra-brain/web`), Owner: "flow"},
	{Path: "internal/plancheck/references.go", Owner: "self"},
	{Path: "internal/plancheck/references_test.go", Owner: "self"},
}
```

  Hinweise zur Liste:
  - **Archive:** Sie liegen unter `docs/.superpowers/`. Die Spec trennt Papiere und Archive nur beim Abschluss der Folgeprojekte. Bis dahin genügt ein Eintrag, und das Folgeprojekt Flow bzw. Web teilt ihn bei seinem Abschluss auf.
  - **#35 aus PR B:** Der Implementierer liest nach dem Rebase den Nachtrag #35 der Fusions-Spec und ergänzt dessen Ausnahmen wörtlich als Zeilen (`internal/brain/apply/testdata/frontmatter/`, `docs/wiki/log.md` laut Ankündigung von B, Eigentümer laut #35).
  - **Migrationsplan:** Ganze Datei. Die Spec sagt „Herkunftsspalte“, aber die Spalte steht in jeder Zeile neben anderem Text, und eine Zeilenregel trüge nichts.
  - **README:** Die Zeilen-Regex trifft nur die Roadmap-Zeilen der laufenden Folgeprojekte. Bevor sie gilt, prüft der Implementierer per `grep -n -i -E "ulflow|ultra-brain/web" README*.md`, dass dort nur Roadmap-Zeilen stehen.

```go
func TestTheRepositoryNamesNoPredecessor(t *testing.T) {
	out, err := exec.Command("git", "-C", filepath.Join("..", ".."), "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	files := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	root := filepath.Join("..", "..")
	found, err := References(files, func(p string) ([]byte, error) { return os.ReadFile(filepath.Join(root, p)) }, exceptions)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("names of the predecessor tools outside the named exceptions (fusion spec, #24):\n%s", strings.Join(found, "\n"))
	}
}
```

  Binärdateien (Bündel gibt es im Baum keine mehr, aber etwa PNGs) liest der Test mit: Ein Treffer darin ist ein Fund wie jeder andere.

- [ ] **Step 6: Rot an echten Resten.** `go test ./internal/plancheck -run TestTheRepositoryNamesNoPredecessor -count=1`. Erwartet ist ein Fehlschlag, solange `templates_test.go` die Altnamen trägt. Dann die vier Einträge dort streichen. Jeder weitere Fund ist entweder ein Rest aus PR B, den die B-Sitzung oder der Mensch klärt (Bericht, kein Fix in C), oder einer aus C, der hier behoben wird.
- [ ] **Step 7: Grün.** Der volle Lauf `go test ./... -count=1` ist grün, `sh ci/gate.sh` ebenso.
- [ ] **Step 8: Mutationsrunde** für `References`, `inFile`, `allowed` und `covers` per `go test -overlay` mit Windows-Pfaden. Je Teilbedingung einzeln:
  - `HasSuffix` auf `false`,
  - `HasPrefix` auf `==`,
  - die Prüfung `!used` entfernen,
  - `default: return true` auf `false`,
  - `changelogEntry` ohne `-beta`-Teil,
  - Abschnitt nicht zurücksetzen.

  Ausgaben in `t7-mut-<n>.txt`. Überlebende bekommen einen Test oder eine Begründung für Äquivalenz.
- [ ] **Step 9: Commit** — `test(plancheck): hold the names of the predecessor tools to their named exceptions`

---

## Nach dem Plan (nicht Teil der Tasks)

- PR per `release-pr` mit `release:major`. Changelog-Block:
  - Removed: `loomux dev import-cases`, `dev record-case`, `dev record-mcp-case`, the recordings now live in the archive release.
  - Changed: nothing user-visible beyond that.
- Die Zeile 4f wird erst ✅, wenn der Neustart der Schwester-Spec durch ist (#31).

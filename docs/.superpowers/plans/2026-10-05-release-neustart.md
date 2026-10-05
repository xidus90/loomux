# Release-Neustart (Schritt 3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Der PR, dessen Merge v1.0.0 als erstes stabiles Release veröffentlicht. Danach schneidet derselbe Workflow Betas `X.Y.Z-beta.N` von jedem Zweig.

**Architecture:** Drei Teile.
- **Code** (Tasks 1–2): Ein `init` aus einer Beta folgt dem Beta-Kanal. Der stabile Kanal fragt nur stabile Releases ab. Das Release-Skript liest die Tags vom Remote, markiert genau die Versionen mit Suffix als Pre-Release und baut ohne Kanal. Ein zweiter Workflow-Weg schneidet Betas.
- **Doku** (Task 3): Die alten Changelog-Überschriften heißen `## [X.Y.Z-beta]`. `RELEASE_CHANNEL` verschwindet aus der Doku.
- **Menschenschritte** (Task 4): alle `v*`-Tags und -Releases löschen, die Repo-Variable löschen, mergen, lokale Tags aufräumen. Der Agent bereitet die Befehle vor und prüft danach.

**Tech Stack:** Go, Bash (Release-Skripte), GitHub Actions, `gh`.

**Spec:** `docs/.superpowers/specs/2026-10-05-loomux-release-neustart-design.md`, Abschnitte „Versionsschema“, „Release-Skripte“, „Ablauf“ Schritt 3, mit dem Nachtrag aus dem Abschluss-Review der Brücke (Schritt 3b: stabiles Release ohne Kanal, `init` aus einer Beta, `--limit 30`, Beleg aus `update.json`). Fusions-Spec Nachtrag #31.

## Global Constraints

- **Wann gemergt wird:**
  - erst nach 4f PR C und PR D (Nachtrag #36), damit v1.0.0 ohne Altbestände erscheint,
  - erst nachdem jeder bekannte Wirt 7.2.0 oder neuer meldet; der Beleg ist das Feld `running` aus `<Zustandsverzeichnis>/update.json`, das `serve` schreibt,
  - erst nachdem alle `v*`-Tags und -Releases gelöscht sind (Task 4).

  Vorher bleibt der PR offen. Sein Label ist `release:major`, und `NextVersion` gibt ohne `v*`-Tag `1.0.0` zurück.
- `RELEASE_CHANNEL` ist bis zum Merge `beta`. Die Brücke 7.2.0 ist so erschienen. Jede Version der alten Zählung, die vor dem Merge noch erscheint (etwa durch PR C), muss Pre-Release bleiben, sonst bricht die Einordnung in `selfupdate.releaseVersion`.
- Beta-Form `X.Y.Z-beta.N` (N ≥ 1, ohne führende Null). Ein Release mit Suffix ist Pre-Release, eines ohne ist stabil.
- Der Changelog bekommt keine Beta-Einträge. Der Eintrag entsteht mit dem stabilen Release.
- Agenten pushen nie und taggen nie auf dem Remote. Alle Befehle an GitHub (Löschen, Variablen, Merge) führt der Mensch aus.
- 100 % Coverage je Funktion; jeder neue Test läuft vor seinem Code rot; eine Mutationsrunde je geänderter Entscheidungsfunktion.
- Kommentare und Commits englisch, ohne Arbeitspapier-Namen; kein `Co-Authored-By`.

## Review Focus

1. **Ein lokaler Checkout des Runners trägt noch alte Tags.** Erwartet: `release.sh` rechnet die Version allein aus `git ls-remote --tags origin`. Ein Skript-Test mit einer Attrappe für `git ls-remote` zeigt `1.0.0`, obwohl `git tag -l` alte Tags hätte.
2. **v1.0.0 erscheint als Pre-Release oder druckt `(beta)`.** Erwartet: Die Markierung folgt nur dem Suffix der Version. Nach dem Build prüft das Skript `loomux_<v>_linux_amd64 version` == `loomux <v>`. Ein Test sichert beides.
3. **Ein Beta-Lauf schreibt einen `chore(release)`-Commit oder einen Changelog-Eintrag.** Erwartet: Der Beta-Weg erzeugt nur Tag und Pre-Release am Kopf des Zweigs.
4. **Ein `init` aus einer Beta installiert ein älteres stabiles Release und plant trotzdem Antigravity-Einträge.** Erwartet: `Install` aus einem Binary mit `-beta.N` läuft im Beta-Modus und setzt die Markierung. Das installierte Binary ist dann mindestens die Beta.
5. **Die Changelog-Umbenennung trifft eine Zeile, die keine Überschrift ist, oder vergisst eine.** Erwartet:
   - Jede Zeile `^## \[\d+\.\d+\.\d+\] - ` wird zu `## [X.Y.Z-beta] - `.
   - Die Zahl der Überschriften bleibt gleich.
   - Der Tor-Test (`internal/namecheck`, Abschnitt-Ausnahmen) bleibt grün, denn er erkennt `-beta` schon.
   - `InsertChangelog` für `1.0.0` wirft keinen Duplikatfehler.

---

### Task 1: Beta-`init` und stabile Abfrage

**Files:**
- Modify: `internal/selfupdate/install.go` (`Install`), `internal/selfupdate/gh.go` (`latest`)
- Test: `internal/selfupdate/install_test.go`, `internal/selfupdate/gh_test.go`

**Interfaces:**
- Unverändert sind `Install(ctx, o Options) Result` und `latest(ctx, run, t takes)`. Den Typ heißt so, wie die Brücke ihn eingeführt hat: `takes` mit `takesStable`/`takesOld`/`takesAll`. Vor dem Schreiben per `grep -n "takes" internal/selfupdate/*.go` nachlesen.

- [ ] **Step 1: Tests.**
  - `install_test.go`: Ein `init` mit `Version: "1.1.0-beta.1"`, ohne Mode, ohne Markierung und mit der Liste `[v1.0.0 stabil, v1.1.0-beta.2 pre]` installiert `1.1.0-beta.2`. Danach liest `ReadChannel` `true`.
  - Gegenfall: `Version: "1.0.0"` installiert `1.0.0` und setzt keine Markierung.
  - `gh_test.go`: `latest(…, takesStable)` ruft `gh release list` mit `--exclude-pre-releases`; `takesAll` und `takesOld` rufen ohne dieses Flag. Geprüft wird an den Argumenten, die der Fake-Runner sieht, wie im vorhandenen `TestLatestTakesTheHighestReleaseOfTheChannel`.
- [ ] **Step 2: Rot.** `go test ./internal/selfupdate -run 'Install|Latest' -count=1` → `--- FAIL` für beide neuen Fälle.
- [ ] **Step 3: Umsetzen.**
  - In `Install`, vor `installLocked`: Hat `o.Mode` keinen Wert und kein `o.Pin`, und trägt `o.Version` einen Beta-Suffix (`parseVersion(o.Version)` mit `beta > 0`), dann wird `o.Mode = ModeBeta` gesetzt.
  - Kommentar: Ein `init` aus einer Beta bringt das Gegenstück seiner selbst auf die Maschine, nicht ein älteres stabiles Release. Die Einträge, die es schreibt, rufen dann kein älteres Binary.
  - In `latest`: Für `takesStable` kommt `"--exclude-pre-releases"` in die Argumente. Kommentar: Viele Betas können das neueste stabile Release sonst aus den 30 jüngsten drängen.
- [ ] **Step 4: Grün** plus Mutationsrunde: Bedingung `beta > 0` auf `>= 0`, `o.Pin` ignorieren, Flag für jeden Kanal setzen.
- [ ] **Step 5: Commit** — `fix(selfupdate): let an init from a beta install betas and ask only for stable releases on the stable channel`

### Task 2: Release-Skripte und Beta-Weg

**Files:**
- Modify: `internal/release/build.go` (`Build` ohne `channel`), `internal/cli/release.go` (`dev release build` ohne `--channel`, Zeilen ~167–176), Tests dazu
- Modify: `.github/scripts/release.sh`
- Create: `.github/scripts/release-beta.sh`
- Modify: `.github/workflows/release.yml`
- Modify: `AGENTS.md` (Ausnahme zu `chore(release): v*`-Commits und `v*`-Tags: Beta-Tags kommen ohne Commit dazu)
- Test: `internal/release/build_test.go`, `internal/cli/release_test.go`; ein Skript-Test als Go-Test, der `release.sh`-Teile nicht braucht (siehe Step 1)

- [ ] **Step 1: Tests für `Build`.** Die ldflags enthalten `cli.Version=<v>` und kein `cli.Channel=`. `dev release build --channel beta` endet mit Exit 2, weil das Flag unbekannt ist.
- [ ] **Step 2: Rot**, dann umsetzen. `Build(version, out string, run GoBuild)`; die ldflags setzen nur `cli.Version`. `cli.Channel` bleibt in `internal/cli/cli.go` stehen, nur so erkennt sich ein Binary der alten Zählung. Neue Builds setzen ihn nicht mehr.
- [ ] **Step 3: `release.sh`.**
  - Die Tags kommen vom Remote:

```bash
version=$(git ls-remote --tags --refs origin 'v*' | sed 's#.*refs/tags/##' | "$LOOMUX" dev release next-version --bump "$bump")
```

    Der Kommentar dazu sagt: Ein wiederverwendeter Checkout des Runners behält Tags, die auf dem Remote gelöscht sind.
  - `channel` und `CHANNEL` entfallen. Pre-Release genau dann, wenn die Version ein `-` trägt:

```bash
pre=()
case $version in *-*) pre=(--prerelease) ;; esac
```

  - Nach dem Build eine Probe auf das Binary für Linux/amd64. Liegt `$work/dist/loomux_${version}_linux_amd64` vor (der Name aus `release.Targets` per `ls`), dann:

```bash
said=$("$work/dist/loomux_${version}_linux_amd64" version)
[ "$said" = "loomux $version" ] || { echo "the built binary says '$said', want 'loomux $version'" >&2; exit 1; }
```

- [ ] **Step 4: `release-beta.sh`** (neu). Eingaben `REF` und `BUMP` kommen aus dem Workflow, dazu `GH_TOKEN`, `REPO` und `LOOMUX`. Das Skript:
  1. `git fetch --quiet origin "$REF"` und `commit=$(git rev-parse FETCH_HEAD)`.
  2. `version=$(git ls-remote --tags --refs origin 'v*' | sed 's#.*refs/tags/##' | "$LOOMUX" dev release next-beta --bump "$BUMP")`.
  3. `git checkout --quiet --force --detach "$commit"` und `"$LOOMUX" dev release build --version "$version" --out "$work/dist"`, danach dieselbe Probe wie in `release.sh`.
  4. Die Notizen werden eine Zeile: `Beta from $REF at $commit.`
  5. `gh release create "v$version" --target "$commit" --title "v$version" --notes-file … --prerelease "$work"/dist/*`.

  Es gibt keinen Commit und keinen Changelog.
- [ ] **Step 5: `release.yml`.**
  - `workflow_dispatch` bekommt die Eingaben:
    - `mode`: Wahl `pr` | `beta`, Default `pr`,
    - `pr`: nicht mehr Pflicht,
    - `ref`: string,
    - `bump`: Wahl `major` | `minor` | `patch`, Default `minor`.
  - Der Schritt ruft bei `mode == 'beta'` `release-beta.sh` auf, sonst `release.sh`.
  - `CHANNEL` fällt aus der Umgebung.
  - Die Bedingung `if:` des Jobs bleibt.
- [ ] **Step 6: Prüfen.**
  - `go test ./internal/release ./internal/cli -run 'Release|Build' -count=1` → PASS.
  - `bash -n .github/scripts/release.sh .github/scripts/release-beta.sh` ist ohne Fehler.
  - Die YAML wird von `go run ./cmd/loomux check precommit` (Lane `lint`) gelesen, falls es eine YAML-Lane gibt. Sonst `python -c "import yaml…"` in einer Skriptdatei: Die Datei parst.
  - Der Smoke-Test `sh ci/smoke.sh bin/loomux.exe` ist grün.
- [ ] **Step 7: Commit** — `feat(release)!: cut betas from any branch and mark a release as pre-release by its suffix alone`, mit Footer `BREAKING CHANGE: dev release build has no --channel; the release workflow no longer reads RELEASE_CHANNEL.`

### Task 3: Changelog und Doku

**Files:**
- Modify: `CHANGELOG.md` (alle Überschriften `## [X.Y.Z] - …` → `## [X.Y.Z-beta] - …`, dazu ein Absatz unter dem Kopf)
- Modify: `internal/release/changelog.go` (`ChangelogHeader`, falls der neue Absatz zum Kopf einer neuen Datei gehört; sonst nur `CHANGELOG.md`)
- Modify: `README.md`, `README.de.md` (Abschnitte zu `RELEASE_CHANNEL` um Zeile 451/509 ersetzen: stabile Releases aus gemergten PRs, Betas per `workflow_dispatch` mit `mode=beta`, `ref`, `bump`; installieren mit `loomux upgrade --beta` oder `--version`)
- Modify: `docs/en/getting-started.md`, `docs/de/getting-started.md` (Beispielausgabe `loomux 4.0.0 (beta)` → `loomux 1.0.0` bzw. `loomux 1.1.0-beta.1`)
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md` (`dev release build` ohne `--channel`; `next-beta` steht schon)

- [ ] **Step 1: Umbenennen per Skript** (Write in den Scratchpad, `newline="\n"`). Muster `^## \[(\d+\.\d+\.\d+)\] - ` → `## [\1-beta] - `. Vorher und nachher die Zahl der Zeilen `^## \[` zählen; sie bleibt gleich. Unter den ersten Absatz des Kopfs kommt: „Entries marked `-beta` belong to the pre-release count before 1.0.0; their tags were deleted when 1.0.0 was released, the pull requests they link stay.“
- [ ] **Step 2: Prüfen.**
  - `go test ./internal/namecheck -count=1` → PASS. Der Tor-Test erkennt die fünf Ausnahmen jetzt unter `-beta`-Überschriften.
  - `go run ./cmd/loomux dev release changelog-insert --version 1.0.0 --date 2026-10-06 --link x --notes <datei>` auf eine Kopie im Scratchpad endet mit Exit 0 (kein Duplikat).
- [ ] **Step 3: Doku** in beiden Sprachen mit gleichem Inhalt; `grep -n "RELEASE_CHANNEL" README*.md docs/*/*.md` trifft danach nur noch `benchmarks.md`, falls dort.
- [ ] **Step 4: Commit** — `docs: mark the releases before 1.0.0 as betas and describe the beta path`

### Task 4: Menschenschritte (nach Code-Review, vor dem Merge)

Der Agent schreibt die Befehle und die Belege in den PR-Text. Die Akte `parity/stufe-4f.md` liegt nach PR D im Archiv. Er prüft nach jedem Schritt.

- [ ] **Step 1: Beleg Schritt 2.** Je Rechner `running` aus `%LOCALAPPDATA%/loomux/update.json`, eingetragen in den PR-Text. Auf diesem Rechner stand am 2026-10-05 7.2.0 als installiert. `running` muss nach einem Neustart von `serve` ebenfalls 7.2.0 oder neuer sein.
- [ ] **Step 2: Löschen (Mensch).** Erst die Releases, dann die Tags, beides auf dem Remote:

```bash
gh release list --repo xidus90/loomux --limit 200 --json tagName --jq '.[].tagName' | grep '^v' | while read -r t; do gh release delete "$t" --repo xidus90/loomux --yes; done
```

```bash
git ls-remote --tags --refs origin 'v*' | sed 's#.*refs/tags/##' | while read -r t; do git push origin ":refs/tags/$t"; done
```

  Prüfen: `gh release list --repo xidus90/loomux` zeigt nur `archive/parity-recordings`, und `git ls-remote --tags origin 'v*'` ist leer.
- [ ] **Step 3: Variable (Mensch).** `gh variable delete RELEASE_CHANNEL --repo xidus90/loomux`.
- [ ] **Step 4: Merge (Mensch).** Der Release-Workflow veröffentlicht v1.0.0. Der Agent prüft:
  - `gh release view v1.0.0 --json isPrerelease` ist `false`,
  - das heruntergeladene Binary sagt `loomux 1.0.0`,
  - `CHANGELOG.md` hat `## [1.0.0]` oben.
- [ ] **Step 5: Lokale Tags (Mensch, je Klon und Worktree).** `git fetch --prune --prune-tags origin`.
- [ ] **Step 6:** Den Migrationsplan gibt es nach PR D nicht mehr; mit dem veröffentlichten v1.0.0 ist 4f fertig. Danach gehen auch die drei Papiere des Neustarts (diese Datei, `plans/2026-10-05-release-bruecke.md`, `specs/2026-10-05-loomux-release-neustart-design.md`) ins Archiv-Release, in einem kleinen Folge-Commit nach dem Merge, wie die übrigen in PR D.

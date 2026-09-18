# Versionierung, Releases und CI

Stand 2026-09-18. Heute gibt es kein `.github/`, keinen Versions-Tag, und das
Gate läuft nur lokal in `.githooks/pre-commit`. `internal/cli.Version` steht
auf `0.0.0-dev` und ist für `-ldflags -X` vorbereitet. Umgesetzt wird GitHub;
der Aufbau lässt einen GitLab-Ableger zu, ohne die Logik zu kopieren.

## Ziel

- Jeder PR nach `master` erzeugt eine neue Version, sofern er nicht als
  `release:none` markiert ist.
- Das LLM entscheidet die Stufe und schreibt den Changelog; CI vergibt Nummer,
  schreibt `CHANGELOG.md`, setzt Tag und Release.
- Jeder PR und jeder Push auf `master` durchläuft das volle Gate samt
  CLI-Smoke-Test auf Windows und auf Linux `go vet`, Build und Smoke-Test.
- Was Versions- und Changelog-Regeln sind, ist getesteter Go-Code, nicht YAML.

## Aufbau: neutral gegen plattformspezifisch

```
internal/release/         Regeln: Version, PR-Rumpf, CHANGELOG.md, Build
internal/cli              `loomux dev release …` als dünne Hülle darüber
ci/gate.sh                gofmt, vet, Tests, covergate
ci/smoke.sh               CLI-Smoke-Test gegen ein gebautes Binary
.githooks/pre-commit      ruft ci/gate.sh, dazu Index-Prüfung und Binary-Tausch
.github/workflows/        nur GitHub: Auslöser, Rechte, API-Aufrufe
  ci.yml  pr-label.yml  release.yml
```

Die Grenze: Alles, was ohne Netz und ohne Plattform-API entschieden werden
kann, liegt in `internal/release` und `ci/`. In `.github/` steht nur, was
GitHub-spezifisch ist — Auslöser, Rechte, Token, den PR zu einem Commit
finden, den Commit schreiben, Tag und Release anlegen. `.github/workflows/`
ist ein von GitHub fest vorgegebener Pfad.

Ein späterer GitLab-Ableger legt `.gitlab/ci.yml` an (in den
Projekteinstellungen als CI-Datei eingetragen, statt `.gitlab-ci.yml` im
Wurzelverzeichnis), ruft dieselben Skripte und Unterbefehle und ersetzt nur
die API-Aufrufe (Merge-Request-API, Releases-API, Project Access Token statt
App). Dieser Ableger ist nicht Teil dieser Spec.

## Versionsschema

Semantic Versioning `MAJOR.MINOR.PATCH`, Tags `vX.Y.Z`, Start bei `v1.0.0`.

| Label | Wann |
|---|---|
| `release:major` | Breaking Change: ein CLI-Befehl, ein Flag, das Hook-Protokoll, das Format von `.loomux/config.toml` oder ein Exit-Code ändert sich inkompatibel oder entfällt |
| `release:minor` | neue Funktion, abwärtskompatibel |
| `release:patch` | Bugfix oder Abhängigkeitsupdate, ohne Breaking Change |
| `release:none` | nur Doku, CI oder Tests; kein Release |

Die Beta-Phase ist keine Eigenschaft der Nummer. Die Repo-Variable
`RELEASE_CHANNEL` (`beta` | `stable`) steuert sie: Bei `beta` wird jeder
Release als Pre-release markiert, und das Binary nennt sich Beta. Wird die
Variable auf `stable` gesetzt, endet die Beta, ohne dass eine Nummer springt.
Anfangswert: `beta`.

## 1. Version im Binary

`internal/cli` bekommt neben `Version` die Variable `Channel` (Vorgabe leer).
Der Release-Build setzt beide über `-ldflags -X`. Ausgabe von
`loomux version`:

- `loomux 1.1.1 (beta)`, wenn `Channel` nicht leer und nicht `stable` ist
- `loomux 1.1.1`, wenn `Channel` leer oder `stable` ist
- `loomux 0.0.0-dev` bei lokalen Builds, unverändert

Der Kommentar an `Version` wird an den neuen Stand angepasst.

## 2. `internal/release` und `loomux dev release`

Ein Paket, vier Unterbefehle. Jeder liest nur Argumente, stdin und das
Dateisystem, schreibt auf stdout und exitet mit 0 (ok), 1 (Befund, etwa
ungültiger PR-Rumpf) oder 2 (Aufruffehler) — dieselbe Konvention wie die
übrigen `dev`-Befehle.

- **`next-version --bump major|minor|patch [--tags <datei>|-]`** — liest
  Tag-Namen zeilenweise, nimmt den höchsten gültigen `vX.Y.Z` und gibt die
  nächste Version ohne `v` aus. Ohne gültigen Tag: `1.0.0`, unabhängig von
  `--bump`. Tags, die kein `vX.Y.Z` sind, werden ignoriert.
- **`parse-body --labels <l1,l2,…> [--body <datei>|-]`** — prüft einen PR:
  genau ein `release:*`-Label; ohne `release:none` ein `## Changelog`-Block
  mit nur erlaubten Unterüberschriften (`Added`, `Changed`, `Deprecated`,
  `Removed`, `Fixed`, `Security`) und mindestens einem Eintrag. Ausgabe JSON
  `{"bump": "minor", "changelog": "### Added\n- …\n"}`; bei Befund Exit 1 und
  die Gründe auf stderr, einer pro Zeile.
- **`changelog-insert --version <v> --date <jjjj-mm-tt> --link <url>
  [--file CHANGELOG.md] [--notes <datei>|-]`** — fügt den Block unter dem Kopf
  als `## [<v>] - <datum>` mit Link ein; legt die Datei samt Kopf an, wenn sie
  fehlt. Eine Version, die schon in der Datei steht, ist ein Befund (Exit 1),
  damit ein wiederholter Lauf nichts doppelt schreibt.
- **`build --version <v> --channel <c> --out <verz>`** — baut mit
  `CGO_ENABLED=0`, `-trimpath` und den `-ldflags` aus Abschnitt 1 für
  `windows/amd64`, `linux/amd64`, `linux/arm64`, `darwin/amd64`,
  `darwin/arm64`, Namensschema `loomux_<v>_<os>_<arch>[.exe]`, und schreibt
  `SHA256SUMS`. Ruft `go build` als Kindprozess; der Test läuft gegen ein
  vorgetäuschtes `go` über eine Variable für den Befehl.

Das Datum kommt als Argument, nicht aus der Uhr, damit die Ausgabe
deterministisch ist. 100 % Coverage wie überall.

## 3. `ci/`: Gate und Smoke-Test

`ci/gate.sh` enthält die Prüfschritte, die heute in `.githooks/pre-commit`
stehen:

1. `gofmt -l cmd internal` muss leer sein
2. `go vet ./...`
3. `go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/... -coverprofile=coverage.out`
4. `go run ./cmd/loomux dev covergate --profile coverage.out`

`.githooks/pre-commit` behält die Prüfung „Eingaben weichen vom Index ab“,
ruft dann `ci/gate.sh` und tauscht danach das Pilot-Binary; beides hat in CI
keinen Sinn.

`ci/smoke.sh <binary>` prüft ein gebautes Binary:

- `loomux version` exitet mit 0 und beginnt mit `loomux `
- `loomux help` exitet mit 0
- `loomux gibtesnicht` exitet mit 2

## 4. `.github/workflows/ci.yml`

Auslöser: `pull_request` und `push`, beide mit `branches: [master]`; der
`push`-Auslöser zusätzlich mit `paths-ignore: [CHANGELOG.md]`, damit der
Release-Commit (Abschnitt 6) `ci` nicht erneut startet. Go über
`actions/setup-go` mit `go-version-file: go.mod`, `permissions: contents:
read`. Zwei Jobs, beide Pflicht für den Erfolg des Laufs, auf den
`release.yml` wartet:

- `gate-windows` auf `windows-latest`, alle Schritte mit `shell: bash`:
  `ci/gate.sh` (gofmt, vet, Tests mit Coverage, Covergate), Build nach
  `bin/loomux.exe`, `ci/smoke.sh bin/loomux.exe`.
- `build-linux` auf `ubuntu-latest`: `go vet ./...`, Build nach
  `bin/loomux.exe`, `ci/smoke.sh bin/loomux.exe`.

Linux fährt das Test-Gate nicht, weil mehrere Tests und die Coverage von
rund 33 Funktionen an Windows gebunden sind; dort könnte das Gate nicht
grün werden. Der Linux-Job sichert, dass der Baum dort übersetzt, `vet`
besteht und die CLI läuft.

## 5. `.github/workflows/pr-label.yml` und die Regel für das LLM

Auslöser: `pull_request` mit `opened`, `edited`, `labeled`, `unlabeled`,
`synchronize`, `reopened`. Der Workflow gibt Labels und Rumpf an
`loomux dev release parse-body` weiter; dessen Exit-Code ist das Ergebnis.
Der Rumpf geht über eine Datei, nie über eine Shell-Interpolation, weil er von
Fremden stammen kann, sobald das Repo öffentlich ist. Aus demselben Grund
baut der Workflow `loomux` aus `master` (`actions/checkout` mit `ref: master`),
nicht aus dem PR: Ein Fork könnte sonst `internal/release` so ändern, dass
`parse-body` immer 0 liefert. Vom PR kommen nur Labels und Rumpf.

Regel für das LLM, festgehalten in `AGENTS.md`: Wer einen PR erstellt, liest
den Diff gegen `master`, wählt das Label nach der Tabelle oben, setzt es mit
`gh pr create --label release:<stufe>` und schreibt in den PR-Rumpf:

```
Release: minor — <Begründung in einem Satz>

## Changelog
### Added
- …
```

Der Changelog ist englisch und aus Nutzersicht (was neu, geändert, behoben
ist — nicht welche Dateien). Im Zweifel zwischen zwei Stufen gilt die höhere.
Ändert sich der PR später, zieht das LLM Label und Block nach. Der Mensch sieht
beides im Review und korrigiert es gegebenenfalls; die Action schreibt keinen
Text selbst.

## 6. `.github/workflows/release.yml`

Auslöser: `workflow_run` von `ci`, abgeschlossen, Ergebnis `success`, und nur
wenn `workflow_run.event == 'push'` und `head_branch == 'master'` — der
Zweigname allein genügt nicht, weil ein Fork-Zweig ebenfalls `master` heißen
kann. `workflow_run` läuft im Kontext von `master`, der Code stammt also nie
aus einem PR. `permissions: contents: read`; geschrieben wird nur mit dem
App-Token aus Abschnitt 7. Eine `concurrency`-Gruppe `release` ohne Abbruch
laufender Läufe serialisiert schnell aufeinanderfolgende Merges.

1. PR zum Commit ermitteln (`gh api repos/{owner}/{repo}/commits/{sha}/pulls`).
   Kein PR (direkter Push, Release-Commit): Ende ohne Release.
2. `parse-body` mit Labels und Rumpf des PRs. `release:none`: Ende.
3. `git tag -l 'v*' | loomux dev release next-version --bump <stufe>`. Dafür
   checkt der Workflow mit `fetch-depth: 0` und `fetch-tags: true` aus; mit
   der Vorgabe (Tiefe 1, ohne Tags) käme jedes Mal `1.0.0` heraus.
4. `changelog-insert` mit Version, heutigem Datum (UTC) und PR-Link.
5. Commit `Release v<version>` mit der geänderten `CHANGELOG.md` über die
   Git-Data-API (Blob, Tree, Commit, dann `PATCH refs/heads/master` ohne
   `force`); GitHub signiert ihn dabei. Ist `master` inzwischen
   weitergezogen, schlägt das Ref-Update fehl: einmal ab Schritt 3 vom
   aktuellen `master` neu aufsetzen, dann abbrechen.
6. `build` mit Version und `RELEASE_CHANNEL` auf dem Release-Commit.
7. `gh release create v<version> --target <release-commit> --title v<version>`
   mit dem Changelog-Block als Notes, allen Binaries und `SHA256SUMS`,
   `--prerelease`, wenn `RELEASE_CHANNEL` nicht `stable` ist. Das legt den Tag
   auf den Release-Commit, sodass der Tag seine eigene Changelog-Zeile
   enthält.

Pushes mit einem App-Token lösen Workflows aus; weil der Release-Commit nur
`CHANGELOG.md` ändert und `ci` diese Datei ignoriert, startet er weder `ci`
noch `release`. Ohne diesen Ausschluss würde der `release`-Lauf des
Release-Commits einen wartenden Lauf eines echten Merges verdrängen.

Der Release-Commit liegt auf dem jeweils aktuellen `master` und kann darum
einen späteren Merge enthalten, dessen `ci` noch läuft; das ist hingenommen,
dessen eigenes Release folgt.

Die `concurrency`-Gruppe hält höchstens einen Lauf wartend; ein neuer
wartender Lauf bricht den älteren ab. Fallen drei Merges in die Dauer eines
Releases, geht der mittlere verloren. Dafür hat `release.yml` einen
`workflow_dispatch` mit Eingabe `pr` (PR-Nummer), der die Schritte 2 bis 7 für
diesen PR von Hand nachholt.

Schritt 5 darf statt der vier Git-Data-Aufrufe
`PUT /repos/{owner}/{repo}/contents/CHANGELOG.md` mit dem aktuellen `sha` der
Datei und `branch: master` nutzen: GitHub signiert auch diesen Commit, und ein
veralteter `sha` liefert 409, also genau die Bedingung für den Neuversuch.
Die Release-Notes gehen über `--notes-file`, nie über Interpolation.

## 7. Schreibrecht: GitHub App `loomux-release`

Das Repo wird nach der Fusion öffentlich und bekommt Rulesets für `master` und
die Tags `v*`. Das `GITHUB_TOKEN` käme daran nicht vorbei; deshalb schreibt
`release.yml` von Anfang an über eine eigene App, damit der Weg beim
Umschalten nicht bricht.

- App `loomux-release`, installiert nur in diesem Repo. Rechte:
  `contents: write`, `pull-requests: read`, `metadata: read`. Kein Webhook.
- Secrets `RELEASE_APP_ID` und `RELEASE_APP_PRIVATE_KEY`. Jeder Lauf holt mit
  `actions/create-github-app-token` ein Token, das nach einer Stunde verfällt.
- In den Rulesets für `master` und `v*` steht die App als einziger
  Bypass-Akteur. Der Bypass gilt für alles, was die App tut; die Grenze zieht
  `release.yml`, das nur `CHANGELOG.md` schreibt und nur `v*` taggt.
- Solange das Repo privat ist, gibt es keine Rulesets; die App wird trotzdem
  schon genutzt.

Autor und Committer des Release-Commits ist `loomux-release[bot]`.
`AGENTS.md` fordert den Menschen als Autor jedes Commits und verbietet Pushes
außer durch Menschen; beide Regeln bekommen genau eine Ausnahme:
`Release v*`-Commits und `v*`-Tags aus `release.yml`. Für Agenten ändert sich
nichts.

App, Secrets, die vier Labels, die Variable `RELEASE_CHANNEL=beta` und später
die Rulesets legt der Mensch an; ein Agent schlägt die Befehle vor und führt
sie nur nach Rückfrage aus.

## Dokumentation

- `AGENTS.md`: Aufbau aus diesem Dokument (neutral gegen plattformspezifisch),
  Label- und Changelog-Regel aus Abschnitt 5, `ci/gate.sh` als Gate, die
  Ausnahme für die Release-App aus Abschnitt 7.
- `README.md` und `README.de.md`: Abschnitt „Releases“ mit Download,
  Versionsschema, Hinweis auf die Beta, Link auf `CHANGELOG.md`, und die
  Einrichtung von App, Secrets, Labels, Variable und Rulesets.

## Nicht enthalten

GitLab-Ableger, Homebrew/Scoop, Signieren der Binaries, Release-PRs,
Prerelease-Suffixe in der Nummer.

## Prüfung

- `internal/release` und `internal/cli`: Unit-Tests, 100 % Coverage.
- `ci/gate.sh` und `ci/smoke.sh` laufen lokal unter Git Bash grün; der
  Pre-commit-Hook verhält sich wie vorher.
- Workflows vor dem Merge mit `actionlint` geprüft.
- Erster echter Durchlauf: ein PR mit `release:none` (kein Release), danach
  einer mit Label, der `v1.0.0` als Pre-release samt `CHANGELOG.md` erzeugt.

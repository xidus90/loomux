# Weboberfläche Schritt 1: Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux ui` öffnet eine lokale Weboberfläche mit Dashboard, Einstellungsseiten (global und je Projekt), Vorschlägen und einem Assistenten zum Hinzufügen eines Projekts; globale Sperrregeln gelten in jedem Projekt.

**Architecture:** Ein eigener, abgelöster Prozess mit Listener auf `127.0.0.1`, der Seiten mit `html/template` rendert und die Pakete unter `internal/` direkt ruft. Login über Einmalcode auf stdin des Kindes, Sitzung nur im Speicher. Was CLI und Oberfläche beide brauchen (Config schreiben, Vorschläge, Einrichtung), zieht vorher aus `internal/cli` in gemeinsame Pakete.

**Tech Stack:** Go 1.26, `net/http`, `html/template`, `embed`, `net/http/httptest`. Kein JavaScript, keine neue Abhängigkeit.

**Spec:** `docs/.superpowers/specs/2026-11-13-web-oberflaeche-einstellungen-design.md`

## Global Constraints

- 100 % Coverage je Funktion; darunter nur mit `//coverage:exempt <reason>` direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst: Vorlagen beim ersten Gebrauch parsen (`sync.OnceValue`).
- Code, Kommentare, Meldungen, Oberflächentexte englisch. Doku in `docs/en/` und `docs/de/` mit gleichen Dateinamen; beide READMEs.
- Keine neue Abhängigkeit, kein JavaScript in Schritt 1.
- Jede schreibende Route ist `POST`, hinter `http.NewCrossOriginProtection`, Sitzung und Formular-Token.
- Kein Geheimnis auf der Platte. `ui.json` hält nur PID und Port.
- Die Oberfläche schreibt nichts, dessen Diff nicht angezeigt wurde.
- Kein zweiter Schreibweg: Config nur über `internal/config/edit` und das Paket aus Task 3, Einrichtung nur über `internal/setup`.
- Commits: Conventional Commits, kein Arbeitspapier im Text, kein Modell als Mitautor. Niemand außer dem Menschen pusht.
- Vor jedem Commit läuft das Tor (`sh ci/gate.sh`) über den Pre-Commit-Hook; ein roter Lauf wird behoben, nicht umgangen.
- Testwelten setzen `LOOMUX_STATE_DIR` und `XDG_CONFIG_HOME` auf ein Temp-Verzeichnis.

## Review Focus

1. **Projektpfad mit Leerzeichen, `#` oder Umlauten** (dieses Repo liegt unter `#GIT`): Projektseite, Assistent und Umschalter müssen ihn fehlerfrei tragen. → Tests in Task 7 und 10.
2. **Zeilenenden im Rohtext:** Ein Browser schickt ein Textfeld mit CRLF. Eine LF-Datei darf dadurch nicht komplett als geändert erscheinen, BOM und Zeilenende der Datei bleiben. → Test in Task 8.
3. **Veraltete Vorschau** (zweiter Reiter, Agent-Vorschlag inzwischen angewendet): Anwenden schreibt nichts und zeigt die Vorschau neu. → Tests in Task 8 und 9.
4. **Kaputte Config eines Projekts:** Die Einstellungsseite zeigt den Fehler mit Datei und Stelle und bietet den Rohtext zum Reparieren; das Dashboard bleibt stehen. → Tests in Task 7, 8 und 11.
5. **Belegter Port, zweites `loomux ui`, `ui.json` mit toter oder fremder PID:** kein Kill eines fremden Prozesses, sauberer Neustart. → Tests in Task 6.

---

## Dateistruktur

| Pfad | Aufgabe |
|---|---|
| `internal/hooks/statefiles.go` | Wächter: Schreibschutz für `config.toml`, `ui.toml`, `ui.json` im Zustandsverzeichnis |
| `internal/config/policy.go` | `ParsePolicy`, `ReadGlobalPolicy`, `ReadEffectivePolicy` |
| `internal/config/target/` | Ziel einer Config (Projekt oder global): lesen, ändern, prüfen, schreiben, Vorschläge |
| `internal/config/edit/remove.go` | Ein Element einer `TableList` entfernen |
| `internal/setup/execute.go` | Was `init` nach dem Plan tut, für CLI und Assistent |
| `internal/serve/detach.go` | Ablösen eines Kindprozesses, für `serve` und `ui` |
| `internal/web/` | Prozess, Login, Middleware, Seiten, Vorlagen, CSS |
| `internal/web/prefs.go` | `ui.toml`: Projektliste und Kacheln |
| `internal/cli/ui.go` | Befehl `loomux ui` |

---

### Task 1: Der Wächter schützt die Dateien im Zustandsverzeichnis

**Files:**
- Create: `internal/hooks/statefiles.go`, `internal/hooks/statefiles_test.go`
- Modify: `internal/hooks/guardjudge.go` (`judge`, `reasons`), `internal/hooks/pretool.go` (Zustandsverzeichnis an `policyReasons` reichen)

**Interfaces:**
- Produces: `func stateFileReasons(absolute, stateDir string, removes bool) []string`; Konstante `stateFilesReason = "loomux's machine-wide files are a human's to write; an agent proposes a change with `loomux config set|unset … --global --propose`"`; Feld `stateDir string` an `judge`.

- [ ] **Step 1: Failing tests** in `statefiles_test.go`, Tabellentest über `checkTool`/`policyReasons` mit Temp-Zustandsverzeichnis. Jede Zeile erwartet `stateFilesReason`:
  - `Write` und `Edit` auf `<state>/config.toml`, `<state>/ui.toml`, `<state>/ui.json`, `<state>/registry.toml`
  - Bash `echo x > <state>/config.toml`, `rm <state>/config.toml`, `rm -r <state>` (Vorfahr), `mv <state>/config.toml x`
  - PowerShell `Set-Content <state>/config.toml x`
  - unter Windows dieselbe Datei in anderer Groß-/Kleinschreibung
  - Gegenprobe ohne Grund: `<state>/logs/serve.log`, `<state>/config/proposals/x.json`, eine Datei gleichen Namens außerhalb
- [ ] **Step 2:** `go test ./internal/hooks -run StateFile` → FAIL. Geht die `registry.toml`-Zeile schon grün durch, bleibt sie als Festschreibung stehen.
- [ ] **Step 3:** `stateFileReasons` implementieren: Vergleich über aufgelöste, absolute Pfade (wie `guard.ManifestBase` Links auflöst), unter Windows ohne Groß-/Kleinschreibung; bei `removes` auch, wenn der Pfad ein Vorfahr einer geschützten Datei ist. In `judge.reasons` je Ziel aufrufen, vor `rels`.
- [ ] **Step 4:** `go test ./internal/hooks` → PASS.
- [ ] **Step 5:** Commit `fix(guard): refuse agent writes to the machine-wide configuration`.

### Task 2: Globale Sperrregeln

**Files:**
- Modify: `internal/config/policy.go`, `internal/config/policy_test.go`, `internal/config/schema/schema.go` (`GlobalKeys`), `internal/hooks/guard.go` bzw. die Stelle, die `config.ReadPolicy` ruft, `internal/cli/config.go` (`validated`)
- Docs: `docs/{en,de}/configuration.md`, `docs/{en,de}/hooks.md`

**Interfaces:**
- Produces:
  - `func ParsePolicy(path string, data []byte) (Policy, error)` — der Rumpf von `ReadPolicy` ab dem Dekodieren
  - `func GlobalPolicyPath(stateDir string) string` — `<stateDir>/config.toml`
  - `func ReadGlobalPolicy(stateDir string) (Policy, error)` — nur `Paths` und `Commands`; fehlende Datei ist leer; ein `[guard]` in der globalen Datei ist ein Fehler, der die Datei nennt
  - `func ReadEffectivePolicy(root, stateDir string) (Policy, error)` — globale Regeln vor denen des Projekts, `Strict` und `GraphHintOff` vom Projekt
  - `schema.GlobalKeys()` nennt zusätzlich `policy.paths.rules` und `policy.commands.rules` (`TableList`, `Module: Base`, Doc wie in `Keys()` mit dem Zusatz „in every project")

- [ ] **Step 1: Failing tests**
  - `TestReadGlobalPolicy`: fehlende Datei → leer; Regeln gelesen; kaputter Glob, fehlender `reason`, nicht kompilierender Regex, `[guard]` → Fehler mit Pfad
  - `TestEffectivePolicyKeepsGlobalRules`: global `match = ["secrets/**"]`, Projekt ohne Regeln → Regel da; Projekt mit eigener Regel → beide, global zuerst
  - Wächter: Projekt ohne Config, globale Pfadregel `secrets/**` → `Write` auf `secrets/a.txt` verweigert mit dem globalen `reason`; globale Befehlsregel verweigert einen Shell-Befehl
  - Wächter: kaputte globale Datei → Ablehnung mit „cannot read its policy" und dem Pfad; ein Hinweis-Werkzeug (`hintOnly`) läuft durch
  - `validated` global: Text mit kaputter Regel → Fehler
- [ ] **Step 2:** Tests laufen lassen → FAIL.
- [ ] **Step 3:** Implementieren. `ReadPolicy` ruft `ParsePolicy`. Jeder Aufrufer von `ReadPolicy` im Wächterpfad wechselt zu `ReadEffectivePolicy` (alle mit `grep -rn "ReadPolicy(" internal` finden und einzeln entscheiden; `check`-Befehle, die nur das Projekt meinen, bleiben).
- [ ] **Step 4:** `go test ./internal/config/... ./internal/hooks ./internal/cli` → PASS.
- [ ] **Step 5:** Doku: in `configuration.md` ein Abschnitt zur globalen `[policy]` (gilt zusätzlich, nie aufhebbar, nur wo der Hook läuft, sperrt Schreiben, nicht Lesen); in `hooks.md` die Reihenfolge eingebaut → global → Projekt.
- [ ] **Step 6:** Commit `feat(config): apply machine-wide policy rules in every project`.

### Task 3: Config-Logik aus `internal/cli` in ein gemeinsames Paket

Reiner Umzug, kein neues Verhalten. Die vorhandenen Tests in `internal/cli` bleiben grün und sind der Beweis.

**Files:**
- Create: `internal/config/target/target.go`, `proposals.go`, `rules.go`, je mit `_test.go`
- Modify: `internal/cli/config.go`, `config_proposals.go`, `config_ui.go`, `config_llm.go`

**Interfaces:**
- Produces (Paket `target`):
  ```go
  type Target struct { Path string; Keys []schema.Key; Global bool }
  func Project(root string) Target
  func Global(stateDir string) Target
  func (t Target) Read() (string, error)
  func (t Target) Entries(text string) ([]schema.Entry, error)
  func (t Target) Lookup(id string) (schema.Key, bool)
  func (t Target) Change(text, id, input string) (string, error)   // heute proposeChange
  func (t Target) Unset(text, id string) (string, error)           // heute proposeUnset
  func (t Target) Validated(next string) (string, error)
  func (t Target) Write(before, next string) error                 // heute writeConfig
  func (t Target) AppendRule(text, section string, pairs [][2]string) (string, error) // wählt AppendBlock oder AppendInline wie config_ui heute

  type Proposal struct { ID, Op, Key, Input, Created, Target string }
  type ProposalView struct { Proposal; Diff, Error string }
  func (t Target) Proposals() ([]ProposalView, error)
  func (t Target) Store(now time.Time, p Proposal) (string, error)
  func (t Target) Apply(id string) error    // berechnet neu, prüft, schreibt, entfernt die Datei
  func (t Target) Reject(id string) error
  ```
- `internal/cli` behält Flags, Ausgabe, Rückfragen und Exit-Codes und ruft dieses Paket.

- [ ] **Step 1:** Tests für `target` schreiben, die das heutige Verhalten je Funktion festhalten (aus den `cli`-Tests übernommen, nicht dupliziert: was nur Logik prüft, zieht mit um).
- [ ] **Step 2:** Code umziehen, `cli` anpassen.
- [ ] **Step 3:** `go test ./internal/cli ./internal/config/...` → PASS, Coverage 100 % in beiden Paketen.
- [ ] **Step 4:** Commit `refactor(config): move config writes and proposals out of the cli package`.

### Task 4: Ein Element einer Regelliste entfernen

**Files:**
- Create: `internal/config/edit/remove.go`, `remove_test.go`
- Modify: `internal/config/target/rules.go`

**Interfaces:**
- Produces: `func RemoveItem(text, section string, index int) (string, error)` — entfernt das `index`-te Element (ab 0, in der Reihenfolge, in der `schema.CurrentOf` die Blöcke zählt) von `section`, ob als `[[section]]`-Block oder als Element von `key = [ … ]`; `func (t Target) RemoveRule(text, section string, index int) (string, error)` prüft danach mit `Validated`.

- [ ] **Step 1: Failing tests** `TestRemoveItem`:
  - drei `[[policy.paths.rules]]`-Blöcke, Index 1 → Block weg, Kommentar über Block 2 bleibt, Leerzeilen sauber
  - Inline-Liste mit drei Elementen, einzeilig und mehrzeilig, mit und ohne Schlusskomma, Index 0, 1, 2
  - letztes Element: `[[…]]` → kein Rest; Inline → `rules = []`
  - gemischte Datei: Blöcke und Inline für verschiedene Sektionen, nur die genannte ändert sich
  - Index außerhalb → Fehler `no item 3 in policy.paths.rules`
  - BOM und CRLF bleiben
  - Ergebnis parst und hat genau ein Element weniger (über `toml.Unmarshal` geprüft)
- [ ] **Step 2:** `go test ./internal/config/edit -run RemoveItem` → FAIL.
- [ ] **Step 3:** Implementieren auf `scan`, `split`, `join`, `arrayEnds` aus `edit.go`.
- [ ] **Step 4:** PASS. Commit `feat(config): remove one entry of a rule list`.

### Task 5: Was `init` nach dem Plan tut, zieht nach `internal/setup`

Reiner Umzug; `loomux init` verhält sich gleich.

**Files:**
- Create: `internal/setup/execute.go`, `execute_test.go`
- Modify: `internal/cli/init.go`

**Interfaces:**
- Produces:
  ```go
  // Deps are what Execute cannot do itself: run an action and say whether a binary stands.
  type Deps struct { Run Runner; BinaryThere func() bool }
  func Notes(f Facts, c Choice, p Plan) []string   // heute initNotes
  // Execute settles what approved leaves of p (hooks-path follows its hook files,
  // merge-hook falls with a declined area-add), drops unapproved actions into
  // Report.Refused and applies the rest.
  func Execute(root string, f Facts, c Choice, p Plan, approved map[string]bool, d Deps, version string, now time.Time) (Report, error)
  ```
- `internal/cli` behält `initRun` (ruft Unterbefehle der CLI) und stellt bereit: `func setupDeps(root string, f setup.Facts, scope string, stdout, stderr io.Writer) setup.Deps`.

- [ ] **Step 1:** Tests für `Execute`: abgelehntes `area-add` ohne `HookWanted` lässt `merge-hook` in `Refused`; `hooks-path` folgt den Hook-Dateien wie `hooksPathApproved` heute; nicht genehmigte Aktion läuft nicht.
- [ ] **Step 2:** Umziehen, `initCommand` ruft `setup.Notes` und `setup.Execute`.
- [ ] **Step 3:** `go test ./internal/setup ./internal/cli` → PASS.
- [ ] **Step 4:** Commit `refactor(setup): run an approved plan from the setup package`.

### Task 6: Der Prozess `loomux ui`: Ablösen, Login, Stop, Wächter

**Files:**
- Create: `internal/serve/detach.go` (+Test), `internal/web/server.go`, `session.go`, `state.go`, `middleware.go`, `templates.go`, `assets/layout.html`, `assets/login.html`, `assets/style.css` (+Tests), `internal/cli/ui.go` (+Test)
- Modify: `internal/serve/spawn.go` (nutzt `Detach`), `internal/cli/commands.go` (`"ui": uiCommand`), `internal/hooks/guard.go` (`writesConfiguration`, `configurationReason`)
- Docs: `docs/{en,de}/cli-reference.md`, `docs/{en,de}/hooks.md`

**Interfaces:**
- Produces:
  ```go
  // package serve
  type DetachSpec struct { StateDir string; Args []string; LogPath string; Stdin io.Reader }
  func Detach(spec DetachSpec, spawner func(*exec.Cmd) error) (brokeAway bool, err error)

  // package web
  const DefaultPort = 7411
  type Options struct {
      StateDir    string
      Version     string
      Code        string            // the one-time login code
      Listen      func(port int) (net.Listener, error)
      SetupDeps   func(root string, f setup.Facts, scope string, stdout, stderr io.Writer) setup.Deps
      Now         func() time.Time
  }
  func Run(ctx context.Context, o Options) error      // listens, writes ui.json, serves until ctx ends
  type State struct { PID, Port int; Started time.Time }
  func StatePath(stateDir string) string              // <stateDir>/ui.json
  func ReadState(stateDir string) (*State, error)
  func Stop(stateDir string) (stopped bool, err error)
  func (s *server) page(w http.ResponseWriter, r *http.Request, name string, data any) // layout + CSRF token
  ```
- Routen dieser Task: `GET /health` (ohne Sitzung, antwortet `{"pid":N}`), `GET /login?code=`, `GET /assets/style.css`, `GET /` (Platzhalterseite bis Task 11), `POST /logout`.

**Festlegungen:**
- Der Elternprozess würfelt 32 Byte Zufall (`serve.NewToken`), startet `loomux ui --foreground` über `Detach` mit dem Code als eine Zeile auf stdin, wartet bis `/health` die PID aus `ui.json` nennt (höchstens 10 s), öffnet den Browser auf `/login?code=…` und druckt `http://127.0.0.1:<port>/` ohne Code. `--no-browser` druckt stattdessen die Login-Adresse.
- Der Code gilt einmal und zwei Minuten; Vergleich in konstanter Zeit. Danach `303` auf `/` mit Cookie `loomux_ui` (`HttpOnly`, `SameSite=Strict`, `Path=/`). Eine Sitzung lebt, bis der Prozess endet.
- Anfragen ohne Sitzung: `303` auf eine Seite, die sagt „run `loomux ui` in a terminal".
- Middleware-Reihenfolge: Host-Prüfung (`Host` muss `127.0.0.1:<port>` oder `localhost:<port>` sein, sonst `421`), `http.NewCrossOriginProtection`, Sitzung, bei `POST` Formular-Token (`csrf`, je Sitzung, konstante Zeit, sonst `403`).
- Port: `DefaultPort`, bei „address in use" Port 0.
- `Stop`: `ui.json` lesen; `/health` fragen; nur wenn die Antwort dieselbe PID nennt, den Prozess beenden; sonst `ui.json` entfernen und nichts beenden.
- `loomux ui` bei laufender Oberfläche: `Stop`, dann neu starten. `loomux ui stop` beendet nur.
- Browser öffnen ist ein Seam (`var openBrowser`), die echte Fassung `//coverage:exempt it starts the user's browser`.

- [ ] **Step 1: Failing tests**
  - `serve`: `Detach` baut das Kommando mit `Args`, Log, stdin; `Spawn` verhält sich wie zuvor (vorhandene Tests)
  - `web`: Login mit richtigem Code → Cookie und `303`; zweiter Versuch mit demselben Code → `403`; falscher Code → `403`; Code nach 2 min → `403`
  - `GET /` ohne Cookie → Hinweisseite; mit Cookie → `200`
  - `POST` ohne `csrf` → `403`; mit `Origin: http://evil.example` → abgelehnt; `Host: evil.example` → `421`
  - `Run` schreibt `ui.json` mit PID und Port; belegter `DefaultPort` → anderer Port
  - `Stop`: laufend → beendet; `ui.json` mit toter PID → entfernt, kein Kill; `ui.json`, deren Port ein fremder Dienst mit anderer PID beantwortet → kein Kill (Review Focus 5)
  - `cli`: `ui stop` ohne laufenden Prozess → Exit 0 mit „not running"; unbekanntes Argument → Usage, Exit 2
  - Wächter: `loomux ui`, `loomux ui stop`, `./bin/loomux.exe ui` verweigert mit `configurationReason`; auch mit `llm_config` an
- [ ] **Step 2:** FAIL bestätigen.
- [ ] **Step 3:** Implementieren. `configurationReason` nennt `ui` mit („loomux init, config, ui and area add …").
- [ ] **Step 4:** `go test ./internal/serve ./internal/web ./internal/cli ./internal/hooks` → PASS.
- [ ] **Step 5:** Von Hand: `go run ./cmd/loomux ui --no-browser`, Adresse öffnen, Seite erscheint; `go run ./cmd/loomux ui stop`. (Ein Mensch führt das aus; der Wächter verweigert es dem Agenten.)
- [ ] **Step 6:** Doku `cli-reference.md` (Befehl, Flags, Exit-Codes), `hooks.md` (nur ein Mensch, Restrisiko ferngesteuerter Browser).
- [ ] **Step 7:** Commit `feat(ui): serve a local web interface behind a one-time login`.

### Task 7: Projektliste und Einstellungsseite, nur lesend

**Files:**
- Create: `internal/web/prefs.go`, `projects.go`, `settings.go`, `assets/settings.html` (+Tests)

**Interfaces:**
- Produces:
  ```go
  type Prefs struct { Projects []string; Tiles []string; TilesSet bool }
  func ReadPrefs(stateDir string) (Prefs, error)      // <stateDir>/ui.toml; missing → zero value
  func WritePrefs(stateDir string, p Prefs) error     // lock.ReplaceText
  type Project struct { ID, Name, Root, Scope string; Registered, Missing, LLMConfig bool }
  func Projects(stateDir string) ([]Project, error)
  func (s *server) target(id string) (target.Target, Project, bool) // "global" or a Project.ID
  ```
- `ui.toml`: `projects = ["C:/…"]` und `[dashboard] tiles = ["projects", …]`.
- `Project.ID`: die ersten 12 Hex-Zeichen von SHA-256 über den bereinigten Pfad, unter Windows kleingeschrieben. `Name`: letzter Pfadteil.
- Routen: `GET /settings/{id}` mit `?q=` und `?modified=1`; `{id}` ist `global` oder eine Projekt-ID.

**Festlegungen:**
- `Projects` = registrierte Bereiche, an deren `Path` eine `.loomux/config.toml` liegt, vereinigt mit `Prefs.Projects`; doppelte Pfade einmal; sortiert nach `Name`. Ein Pfad aus `Prefs` ohne Config hat `Missing: true`.
- Die Seite gruppiert nach `Key.Module` in der Reihenfolge `base`, `hooks`, `brain`, `graph`. Je Zeile: ID, Doc, Wert (`shownValue`-Regel: bei `Preset` die Vorgabe), Vorgabe, Herkunft. `TableList`: die Elemente als Liste; bei `policy.*` mit Herkunft `built-in`, `global` oder `project` (eingebaute aus `hooks`, dafür exportiert `hooks` zwei Leser `BuiltinPathRules() []config.PathRule` und `BuiltinCommandRules() []config.CommandRule`).
- Die Gruppe eines Moduls, das `[modules]` ausschaltet, trägt den Vermerk „module off — these keys have no effect"; die Schlüssel bleiben sichtbar und bearbeitbar.
- `q` filtert ohne Groß-/Kleinschreibung über ID und Doc; `modified=1` zeigt nur `Origin == Set`.
- An jeder Pfadregel steht: „blocks writes, not reads".
- Kopfzeile: Projektumschalter (Links), „Global", „Add project".

- [ ] **Step 1: Failing tests**
  - `Projects`: Bereich mit Config erscheint, Bereich ohne (Wegweiser) nicht, Pfad aus `ui.toml` erscheint, fehlender Pfad `Missing`
  - Projekt unter `…/#GIT/mein projekt ü/`: ID stabil, Seite `200`, Name richtig (Review Focus 1)
  - `GET /settings/global` listet `model.enabled` und die zwei Regellisten
  - `?q=commit` zeigt nur passende Schlüssel; `?modified=1` nur gesetzte
  - Projektseite zeigt eine globale Regel als `global` und nur lesbar
  - Regel mit `<script>` im `reason` erscheint maskiert
  - kaputte Projekt-Config → `200` mit Fehlertext samt Dateiname und Link zum Rohtext (Review Focus 4)
  - unbekannte ID → `404`
  - `[modules] brain = false` → die Brain-Gruppe trägt den Vermerk
- [ ] **Step 2–4:** FAIL, implementieren, PASS.
- [ ] **Step 5:** Commit `feat(ui): show the settings of a project and of the machine`.

### Task 8: Einstellungen schreiben: Vorschau, Anwenden, Rohtext

**Files:**
- Create: `internal/web/settings_write.go`, `assets/preview.html`, `assets/raw.html` (+Tests)

**Interfaces:**
- Consumes: `target.Target` (`Change`, `Unset`, `Validated`, `Write`), `edit.Diff`.
- Routen:
  - `POST /settings/{id}/preview` — Felder `key`, `value` oder `reset=1`; oder `raw=<ganzer Text>`
  - `POST /settings/{id}/apply` — Felder `before` (SHA-256 hex des gelesenen Textes), `next` (ganzer neuer Text)
  - `GET /settings/{id}/raw`

**Festlegungen:**
- Ein Formular je Schlüssel (kein Sammelspeichern in Schritt 1): Eingabe nach `Kind` — `String`/`Int`/`Float` Textfeld, `Bool` Auswahl `true`/`false`, `Enum` `<select>` über `Choices`, `StringList` Textfeld in der Form von `schema.JoinList`.
- Vorschau: neuen Text berechnen, prüfen, `edit.Diff(before, next)` zeigen. Fehler der Prüfung erscheinen auf der Vorschau mit dem Text der CLI; es gibt dann keinen Anwenden-Knopf. Ein leerer Diff sagt „nothing to change".
- Anwenden: Datei lesen; stimmt ihr Hash nicht mit `before`, nichts schreiben und die Vorschau mit dem neuen Stand neu zeigen samt Satz „the file changed since the preview". Sonst `next` erneut mit `Validated` prüfen und mit `Write(current, next)` schreiben; `303` zurück zur Seite.
- Rohtext: Textfeld mit der ganzen Datei. Vor dem Vergleich werden die Zeilenenden des gesendeten Textes auf das Zeilenende der Datei gebracht (CRLF bleibt CRLF, LF bleibt LF), ein BOM der Datei bleibt. Eine noch nicht vorhandene Datei bekommt LF.

- [ ] **Step 1: Failing tests**
  - `commit.language` auf einen anderen Wert: Vorschau zeigt den Diff, Datei unverändert; Anwenden schreibt; Kommentare in der Datei bleiben
  - Wert gleich Vorgabe → Zeile entfernt (Verhalten von `Change`)
  - `reset=1` entfernt die Zeile
  - ungültiger Wert (`Int` mit „abc", `Enum` außerhalb) → Vorschau mit Fehler, kein Anwenden-Knopf
  - Brain-Schlüssel ohne `[area]` → der Fehlertext von `Change`
  - Datei zwischen Vorschau und Anwenden geändert → nichts geschrieben, neue Vorschau (Review Focus 3)
  - `apply` mit `next`, das nicht prüft → nichts geschrieben
  - Rohtext: LF-Datei, Browser sendet CRLF, eine Zeile geändert → Diff zeigt nur diese Zeile, Datei bleibt LF; dasselbe mit CRLF-Datei und BOM (Review Focus 2)
  - Rohtext auf kaputter Config: Seite lädt, Reparatur lässt sich anwenden (Review Focus 4)
  - globales Ziel: `model.endpoint` auf eine Nicht-Loopback-Adresse → Fehler von `GuardEndpoint`
- [ ] **Step 2–4:** FAIL, implementieren, PASS.
- [ ] **Step 5:** Commit `feat(ui): edit settings with a diff before every write`.

### Task 9: Regeln, Vorschläge und der Schalter `llm_config`

**Files:**
- Create: `internal/web/rules.go`, `proposals.go`, `llmconfig.go`, `assets/confirm.html` (+Tests)

**Interfaces:**
- Consumes: `Target.AppendRule`, `Target.RemoveRule`, `Target.Proposals`, `Target.Apply`, `Target.Reject`, `config.SetLLMConfig`.
- Routen:
  - `POST /settings/{id}/rules/preview` — `section`, und entweder `remove=<index>` oder die Felder der Regel (`match` als Liste und `reason`; `regex` und `reason`; für `commit.allow` dessen Felder laut `config`)
  - Anwenden über `POST /settings/{id}/apply` aus Task 8
  - `POST /settings/{id}/proposals/{pid}/apply`, `POST …/reject`
  - `GET /settings/{id}/llm` (Bestätigungsseite), `POST /settings/{id}/llm` mit `on=true|false`

**Festlegungen:**
- Vorschläge stehen oben auf der Seite ihres Ziels, je mit `Diff` oder `Error` aus `ProposalView`. Ein Vorschlag mit `Error` hat nur „Reject".
- Anwenden eines Vorschlags ist ein Schritt; `Target.Apply` rechnet gegen den jetzigen Stand und schreibt über `Write`. Schlägt es fehl, lädt die Seite mit dem Fehler neu.
- `llm_config` erscheint nur bei `Project.Registered`; sonst der Satz „only a registered area has this switch". Die Bestätigungsseite nennt wörtlich, was der Schalter öffnet (Text aus `docs/en/configuration.md`, Abschnitt zu `llm_config`, ein Satz).

- [ ] **Step 1: Failing tests**
  - Pfadregel anlegen: Vorschau zeigt den neuen Block, Anwenden schreibt, die Seite listet sie als `project`
  - in einer Datei mit Inline-Liste landet die neue Regel in der Liste, nicht als `[[…]]`
  - Regel entfernen über `remove=1`
  - Regel mit kaputtem Glob oder leerem `reason` → Fehler auf der Vorschau
  - auf einer Projektseite hat eine globale oder eingebaute Regel keinen Entfernen-Knopf, und `remove` mit ihrem Index trifft nur Regeln der Projektdatei
  - Vorschlag anwenden → Datei geändert, Vorschlag weg; ablehnen → Datei gleich, Vorschlag weg
  - Vorschlag, der nicht mehr passt → nur „Reject" sichtbar; `apply` darauf → Fehler, nichts geschrieben (Review Focus 3)
  - `llm_config` umlegen → Registry geändert; bei nicht registriertem Projekt `POST` → `409`
- [ ] **Step 2–4:** FAIL, implementieren, PASS.
- [ ] **Step 5:** Commit `feat(ui): manage rules, agent proposals and the llm_config switch`.

### Task 10: Assistent „Add project"

**Files:**
- Create: `internal/web/wizard.go`, `assets/wizard_*.html` (+Tests)
- Docs: `docs/{en,de}/getting-started.md`

**Interfaces:**
- Consumes: `setup.Gather`, `setup.Parts`, `setup.DefaultChoice`, `setup.Build`, `setup.Notes`, `setup.Execute`, `Options.SetupDeps`, `WritePrefs`.
- Routen: `GET /projects/new` (Pfad), `POST /projects/new/facts`, `POST /projects/new/plan`, `POST /projects/new/apply`.

**Festlegungen:**
- Zustandslos: jeder Schritt trägt Pfad und Auswahl als versteckte Felder weiter und rechnet `Gather` und `Build` neu.
- Pfad: absolut, existierendes Verzeichnis; sonst Fehler auf der ersten Seite.
- Auswahl: je Teil aus `setup.Parts(f)` ein Haken, vorbelegt aus `DefaultChoice`; Commit-Sprache als Auswahl; zuletzt `llm_config` (nur wenn der Bereich registriert wird), Vorgabe aus.
- Plan: je `Change` der Diff (`edit.Diff(Before, After)`) mit Haken, vorbelegt an; je `Action` `Describe` mit Haken; `Notes` darunter. Je `Change` ein verstecktes Feld mit dem SHA-256 von `Before`.
- Anwenden: Plan neu bauen; weicht ein `Before`-Hash ab, nichts ausführen und den Plan neu zeigen. Sonst `setup.Execute` mit den Haken als `approved`. Danach den Pfad in `Prefs.Projects` eintragen und den Bericht zeigen (jedes Feld von `setup.Report`, wie `printReport` es heute druckt) mit Link zur Einstellungsseite.
- `setup.Unmergeable`-Fehler erscheinen mit der genannten Datei auf der Befundseite.

- [ ] **Step 1: Failing tests** (Git-Testwelt über die vorhandenen Helfer in `internal/setup`)
  - leerer, relativer, nicht vorhandener Pfad → Fehler
  - frisches Git-Repo: Befund nennt erkannte Hosts; Plan listet `.loomux/config.toml` als neu
  - Anwenden mit allen Haken und einem `Deps`-Fake: Dateien geschrieben, Projekt in `ui.toml`, Bericht zeigt die Aktionen
  - eine Änderung abgewählt → diese Datei bleibt
  - Datei zwischen Plan und Anwenden geändert → nichts geschrieben, Plan neu
  - Pfad `…/#GIT/mein projekt ü` läuft durch alle Schritte (Review Focus 1)
  - `.mcp.json`, die kein Objekt ist → Fehler mit Dateiname
  - schon eingerichtetes Projekt → Plan zeigt nur, was fehlt, oder „nothing to do"
- [ ] **Step 2–4:** FAIL, implementieren, PASS.
- [ ] **Step 5:** `getting-started.md`: der Assistent als zweiter Weg neben `loomux init`.
- [ ] **Step 6:** Commit `feat(ui): set up a project from the browser`.

### Task 11: Dashboard

**Files:**
- Create: `internal/web/dashboard.go`, `tiles.go`, `assets/dashboard.html`, `assets/dashboard_edit.html` (+Tests)

**Interfaces:**
- Produces:
  ```go
  type Tile struct {
      ID, Title string
      Render func(s *server) (template.HTML, error)
  }
  func tiles() []Tile   // fixed order: projects, proposals, flow-runs, review-cases, index, version
  ```
- Routen: `GET /` (ersetzt den Platzhalter), `GET /dashboard/edit`, `POST /dashboard/edit` mit `tile`, `action=show|hide|up|down`, `POST /projects/{id}/forget`.

**Festlegungen:**
- `Prefs.TilesSet == false` → alle Kacheln in fester Reihenfolge. Unbekannte IDs in `ui.toml` werden übergangen. Eine `ui.toml`, die nicht parst, gilt als fehlend; die Seite zeigt oben „ui.toml could not be read: …".
- Eine Kachel, deren `Render` scheitert, zeigt den Fehler in ihrem Rahmen.
- Quellen:
  - `projects`: `Projects()`; je Projekt, ob die Hook-Einträge stehen (`setup.Gather`), `LLMConfig`, `Missing` mit „Forget"-Knopf
  - `proposals`: `Target.Proposals()` je Ziel, Anzahl und Link
  - `flow-runs`: je Projekt die Journale unter `runs.Dir` mit `journal.Pending != nil`, Laufnummer und Projekt (Logik wie `waitingRuns` in `internal/hooks/flowruns.go`; die Auflistung zieht als exportierte Funktion nach `internal/flow/runs`, die beide rufen)
  - `review-cases`: die offenen Fälle, die `loomux brain cases` listet; dessen Leser aus `internal/cli` in das Brain-Paket ziehen, das die Fälle hält, und von beiden rufen
  - `index`: je Projekt `freshness.Probe` für den Code-Graphen; für die Suche der Zustand, den `brain status` nennt
  - `version`: `Options.Version`, `selfupdate.ReadChannel`, `selfupdate.ReadStatus`

- [ ] **Step 1: Failing tests**
  - ohne `ui.toml`: sechs Kacheln in fester Reihenfolge
  - `hide` einer Kachel → weg und in `ui.toml` nicht mehr genannt; `show` → am Ende wieder da
  - `up` an erster Stelle und `down` an letzter ändern nichts
  - unbekannte ID in `ui.toml` → übergangen; kaputte `ui.toml` → alle Kacheln und der Hinweis
  - Projekt mit kaputter Config: seine Zeile nennt den Fehler, die übrigen Kacheln stehen (Review Focus 4)
  - eine Kachel mit Fehler-Fake → Fehler im Rahmen, Status `200`
  - wartender Lauf in einem Projekt erscheint mit Laufnummer
  - offener Vorschlag erscheint mit Link auf `/settings/{id}`
  - `forget` entfernt den Pfad aus `ui.toml`
- [ ] **Step 2–4:** FAIL, implementieren, PASS.
- [ ] **Step 5:** Commit `feat(ui): open on a dashboard whose tiles the user arranges`.

### Task 12: READMEs, Roadmap, Wiki, Abschluss

**Files:**
- Modify: `README.md`, `README.de.md`, `docs/{en,de}/architecture.md` (falls sie die Web-Oberfläche als geplant nennt), betroffene Seiten unter `docs/wiki/`

- [ ] **Step 1:** Funktionen-Tabelle beider READMEs: neue Zeile „Web interface" / „Weboberfläche" mit `loomux ui`, Einstellungen, Assistent, Dashboard, globale Regeln. Das Mermaid-Diagramm: die Kante zum Browser wird durchgezogen und hängt nicht mehr an `Serve`.
- [ ] **Step 2:** Roadmap in beiden READMEs:
  - „Web-OS-Shell" entfällt (gebaut).
  - „Brain-Web-App", „Graph-Visualizer", „Flow-Editor und Kanban": „Hängt ab von" nennt statt „Web-OS-Shell" die Weboberfläche ✅; „Graph-Visualizer" nennt zusätzlich eine 3D-Ansicht.
  - Neu „Lesesperre": Der Wächter verweigert auch das Lesen gesperrter Pfade über `Read` und lesende Shell-Befehle; Priorität 3.
  - Neu „Dashboard mit Drag-and-drop"; Priorität 5.
  - Neu „Sammelspeichern und Ordnerbrowser in der Weboberfläche"; ohne Priorität.
  - Die Prioritäten sind der Vorschlag aus dem Gespräch; der Mensch bestätigt sie im Pull Request.
- [ ] **Step 3:** Wiki: Seiten, die Config, Wächter oder `serve` zitieren, gegen den Code lesen und nachziehen; bei `register-stale` oder `pin-stale` erst lesen, dann `loomux wiki repin`.
- [ ] **Step 4:** `sh ci/gate.sh` → grün.
- [ ] **Step 5:** Commit `docs: describe the web interface and machine-wide rules`.
- [ ] **Step 6:** Pull Request über die Skill `release-pr`: Label `release:minor`, Zuweisung `--assignee @me`, den Push-Befehl nennt der Agent, der Mensch pusht.

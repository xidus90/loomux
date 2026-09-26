# Stufe 4c-1: das lokale Modell — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `reconcile` fragt für einen `local_only`-Bereich ein Ollama auf
Loopback nach einem Vorschlag, prüft ihn mit derselben Belegbindung wie
`approve` und legt ihn als `proposal.md` neben den Fall; ein Ausfall ergibt
einen Fall ohne Vorschlag, nie Verkehr nach außen. Dazu zwei geerbte Fehler
geheilt: `approve --reject` schiebt Quellen und Register vor, und `reindex`
und `approve` teilen eine Sperre je Bereich.

**Architecture:** Die Einstellungen liest `internal/config`
(`ModelSettings`, global aus `<zustand>/config.toml`, eingeengt durch die
Bereichsdeklaration). Das neue Paket `internal/brain/model` hält Tor,
Loopback-Wächter, HTTP-Client und die Rolle `propose` samt eingebettetem
Prompt; es kennt `evidence` und `config`, nicht `maintenance`.
`maintenance.ReconcileContext` baut die Proposer einmal je Lauf und reicht
sie mit dem Kontext bis `landCase`. Die Heilungen liegen in `internal/brain/apply`
und `internal/brain/index`; die Sperre stellt `config.LockArea`.

**Tech Stack:** Go des Moduls, `net/http`, `net/http/httptest`, `embed`,
`third_party/toml`, `internal/lock`, `internal/brain/evidence`,
`internal/cases` (Aufzeichnung und Abspielen).

**Spec:** `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`,
Abschnitte „4c im Einzelnen“ und **„Abweichungen beim Planen von 4c“** (der
zweite gilt, wo er dem ersten widerspricht), dazu „Parität“,
„Fehlerverhalten“, „Selbstnutzung“, „Messen“, „Fertig, wenn“.

**Referenz:** `ultra-brain` Tag `loomux-3-source` = `3cc72d2` = HEAD (am
2026-09-25 geprüft, kein Unterschied unter `src/` und `tests/`). Gelesen:
`src/brain/model/{client,gate,settings,local,prompts}.py`,
`src/brain/maintenance/reconcile.py:119-129,271-310,712-848`,
`src/brain/prompts/vorschlag-v4.md` (3413 Bytes, nur LF, sha256
`05e2ebbbf448a0b5ce8562719b9cd564dfd1f62dac1b8f199fed02aa0c19b1ab`, einziges
`{…}` ist `{paket}` in der letzten Zeile),
`tests/maintenance/test_reconcile_proposal.py` (11 Tests).

## Befunde, gegen den Code gelesen am 2026-09-25

**B1. Der Einhängepunkt ist vorbereitet.** `landCase`
(`internal/brain/maintenance/reconcile.go:729-807`) rendert das Paket vor dem
Schreiben (`:796-799`) und setzt heute `Manual: closed` und
`Note: noteFor(manifest)` (`:786-791`). `Case.PromptVersion` steht in Struct,
Renderer und Leser (`case.go:57-61,259`), wird aber nie gesetzt. `noteFor`
kennt nur den ersten Wortlaut (`:822-832`).

**B2. Vier Aufrufer, kein Hook.** `ReconcileContext` rufen `loomux
reconcile`, der Abgleich vor `reindex` (`cli/index.go:110`), `technicalUpdate`
nach `approve` (`cli/approve.go:211`) und der Upkeep von `serve`
(`serve/upkeep.go:142`, mit Kontext). Kein Hook-Pfad erreicht ihn.

**B3. `[model]` halb da.** `DeclarationKeys` und `ReadDeclaration` prüfen
`model.enabled` und `model.roles` (`config/declaration.go:52,101-106,190-221`),
`Manifest` verwirft sie (`config/manifest.go:94-114`).
`schema.GlobalKeys()` ist `nil` (`config/schema/schema.go:119-121`),
`TestGlobalKeysAreEmptyForNow` und `TestConfigGlobalKnowsNoKeyYet` halten das
fest. `schema.Current` läuft nur über `Keys()` (`current.go:38`), und
`configTarget.entries` filtert danach (`cli/config.go:144-157`): globale
Schlüssel, die das Projekt nicht kennt (`endpoint`, `name`, `temperature`),
kämen nie an. `proposeChange` verlangt für jeden Brain-Schlüssel ein `[area]`
(`cli/config.go:268-276`), auch in der globalen Datei. `validated` prüft die
globale Datei gar nicht (`:309-318`). Es gibt keine Art für eine
Gleitkommazahl (`schema.Kind`, `edit.Render`).

**B4. Die Belegprüfung ist dieselbe.** `evidence.ReadProposal` und
`evidence.CheckEvidence(claims, []evidence.Segment)`
(`brain/evidence/evidence.go:294,368`) sind die Messlatte von `approve`
(`apply/approve.go:184-199`). `maintenance.Segment` hat dieselben vier Felder
wie `evidence.Segment` (`maintenance/package.go:54`); `maintenance` importiert
`evidence` nicht, `evidence` nur `pytext`.

**B5. `--reject` ist ein eigener Pfad** (`apply/reject.go:30-54`) ohne
`guardSources`, ohne Seiten- und Registervorschub.
`TestRejectAdvancesNeitherThePageNorTheRegister` (`reject_test.go:361`) und
der Fall `testdata/cases/3b/approve/reject` halten das Geerbte fest.
`AdvanceFrontmatter` (`apply/frontmatter.go:55-102`) setzt außer den Quellen
`generated.at` und einen `verified`-Eintrag; ein Vorschub nur der Quellen
fehlt.

**B6. Keine gemeinsame Sperre.** `reindex` liest das Register in `collect`
und schreibt es in `publish` (`brain/index/reindex.go:163-190`), `approve`
liest und schreibt es in `advanceRegisters` (`apply/approve.go:415-456`),
beide ohne Sperre. `<zustand>/areas/<flat>` tauscht `lock.ReplaceDir` als
Ganzes; ein schreibbarer Bereich hat das Verzeichnis nicht. `config` importiert
`lock` schon (`config/registrywrite.go:141`). `lock.acquire` legt das
Elternverzeichnis nicht an (`lock/lock.go:81-96`).

**B7. Aufzeichnung.** Weltwurzel ist der Zustand beider Seiten
(`BRAIN_STATE_DIR` in `dev/recordcase/recordcase.go:150`,
`LOOMUX_STATE_DIR` in `cli/cases_3a_test.go:186`); `<welt>/config.toml`
ist also die globale Datei für beide. `fakeqmd` gibt es nur als Handler im
Prozess (`dev/fakeqmd/fakeqmd.go:272`); Python braucht ein Ollama, das als
eigener Prozess lauscht.

**B8. Die Sperrdatei ist neu im Zustand.** Jeder Fall, der `reindex` oder
ein schreibendes `approve` abspielt, bekommt eine Datei
`areas/<flat>.lock`, die die Referenz nicht kennt — wie `registry.lock` in
`expected3a` (`cli/cases_3a_test.go:65`).

## Entscheidungen

Alle aus der Spec, Abschnitt „Abweichungen beim Planen von 4c“, am
2026-09-25 mit dem Nutzer entschieden; hier nur, was der Plan daraus macht.

| # | Frage | Umsetzung |
|---|---|---|
| E1 | Wo die Einstellungen leben | `internal/config/modelsettings.go`: Leser, Vorgaben, Einengung. Die Leser liegen in `config`, das Schema prüft dagegen |
| E2 | Schema der globalen Datei | `schema.GlobalKeys()` mit fünf Schlüsseln, neue Art `Float`; `schema.CurrentOf(keys, text)`; `validated` ruft für die globale Datei `config.ParseModelSettings` |
| E3 | Wann gefragt wird | Einmal je Lauf entstehen die Proposer, nur für `local_only`-Bereiche; ohne solchen Bereich wird `config.toml` nicht gelesen |
| E4 | Abbruch mitten im Aufruf | Ist der Kontext nach einem gescheiterten Aufruf beendet, schreibt `landCase` nichts und gibt den Kontextfehler zurück. Sonst hielte ein beendetes `serve` einen Fall mit „no usable proposal“ fest, und ein verbrauchter Versuch wird nie wieder gefragt (Review Focus 1) |
| E5 | `--reject` ohne Frontmatter | Die Seite bleibt, wie sie ist; das Register wird vorgeschoben. Eine Ablehnung, die an einer Seite ohne Quellenliste scheitert, bliebe für immer offen |
| E6 | Sperre | `<zustand>/areas/<flat>.lock`, blockierend; `reindex` hält sie von `collect` bis `publish` (das Register wird dort gelesen und geschrieben), `approve` je Register um `moveStock`, Lesen und Schreiben |
| E7 | Ollama-Attrappe | `loomux dev fake-ollama --fixture <json> --addr 127.0.0.1:11435`; Antwort aus der Fixture, je Anfrage eine Zeile in ein Log außerhalb der Welt (`--log`), ohne Prompt; gezählt wird im Aufzeichnungsskript und im Abspieltest. Im Abspieltest derselbe Handler auf demselben festen Port |

## Global Constraints

- Coverage 100 % je Funktion; ein Ausschluss nur mit `//coverage:exempt <reason>` direkt über `func` (AGENTS.md).
- Kein `init()` und keine Paketvariable, die eingebettete Daten parst; der Prompt wird als `string` eingebettet und erst beim Aufruf benutzt (Fusions-Spec, „Startzeit-Regel“).
- Loopback heißt: Host, kleingeschrieben, gleich `127.0.0.1`, `localhost` oder `::1`; keine Namensauflösung.
- Aufruf nur `POST <endpoint ohne abschließende />/api/generate` mit `model`, `prompt`, `stream:false`, `think:false`, `options{temperature, num_ctx:8192}`.
- Kein Proxy aus der Umgebung, keine Weiterleitung, 2 s Verbindungsaufbau, 30 s insgesamt.
- Vorgaben: `enabled = false`, `endpoint = "http://127.0.0.1:11434"`, `name = "hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL"`, `temperature = 0.0` (0 bis 2, kein Bool), alle drei Rollen an; ist `roles` gesetzt, ist jede nicht genannte Rolle aus.
- Je Bereich nur `[model] enabled` und `roles`; aus schlägt an, nur in diese Richtung; Rollen werden nur eingeengt.
- `prompt_version = "vorschlag-v4"`; die zweite Notiz wörtlich: `manual review: the local proposer returned no usable proposal (slice-6 spec §3)`.
- Code, Kommentare, Meldungen und Commits englisch; Plan, Akte und Spec deutsch. Commit-Nachrichten nach Conventional Commits, ohne Stufe, Plan oder Task im Text.
- Kein Push durch einen Agenten; nach jedem Subagenten `git log -1 --format='%an <%ae>'` lesen.

## Review Focus

1. **`serve` endet, während Ollama rechnet.** Der Kontext des Upkeeps bricht den Aufruf ab; erwartet wird kein geschriebener Fall und der Kontextfehler, nicht ein Fall mit der zweiten Notiz. Test in Task 5.
2. **`http://LOCALHOST:11434`.** Python liest `hostname` kleingeschrieben und lässt es durch; Go's `Hostname()` tut das nicht. Erwartet wird: erlaubt. Test in Task 3.
3. **Ollama antwortet mit 302 auf einen fremden Host.** Erwartet: kein zweiter Request, keine Antwort, Fall ohne Vorschlag. Test in Task 3.
4. **Ein Bereich schaltet das Modell ab, global ist es an.** Erwartet: kein Request, Fall mit der ersten Notiz (kein Proposer). Test in Task 5.
5. **`approve --reject` auf einer Seite ohne Frontmatter.** Erwartet: Exit 0, Seite unverändert, Register vorgeschoben, Fall entfernt. Test in Task 6.
6. **`approve --reject`, nachdem die Zielseite gelöscht oder umbenannt wurde.** Der alte Pfad las die Seite nie und schloss den Fall; der neue darf ihn nicht für immer offen lassen. Erwartet: Exit 0, Register vorgeschoben, Fall entfernt, keine Seite angelegt. Test in Task 6.
7. **`http://localhost:70000`.** Python wirft beim Lesen des Ports („Port out of range“), der Lauf stoppt als Fehlkonfiguration; Go's `url.Parse` nimmt jede Ziffernfolge an. Erwartet: derselbe Konfigurationsfehler, kein stiller Fall mit der zweiten Notiz. Test in Task 3.

---

### Task 1: `[model]` lesen — global und je Bereich

**Files:**
- Create: `internal/config/modelsettings.go`
- Create: `internal/config/modelsettings_test.go`
- Modify: `internal/config/manifest.go:94-114` (zwei Felder)
- Modify: `internal/config/declaration.go:101-106,134-149` (Felder füllen)

**Interfaces:**
- Produces:
  - `const DefaultModelEndpoint = "http://127.0.0.1:11434"`, `const DefaultModelName = "hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL"`
  - `func ModelRoleNames() []string` → `{"describe", "place", "propose"}`
  - `func GlobalModelKeys() []string` → `{"enabled", "endpoint", "name", "roles", "temperature"}`
  - `type ModelSettings struct { Enabled bool; Endpoint, Name string; Temperature float64; Roles map[string]bool }` (nur eingeschaltete Rollen als Schlüssel)
  - `func ReadModelSettings(stateDir string) (ModelSettings, error)`
  - `func ParseModelSettings(path, text string) (ModelSettings, error)`
  - `func (s ModelSettings) Narrowed(m *Manifest) ModelSettings`
  - `func (s ModelSettings) RoleOn(role string) bool`
  - `Manifest.ModelEnabled *bool` (nil = nicht gesagt), `Manifest.ModelRoles map[string]bool` (nil = nicht gesagt; sonst die eingeschalteten)

- [ ] **Step 1: Failing tests schreiben**

```go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelSettingsDefaultWithoutAFile(t *testing.T) {
	s, err := ReadModelSettings(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if s.Enabled || s.Endpoint != DefaultModelEndpoint || s.Name != DefaultModelName || s.Temperature != 0 {
		t.Fatalf("%+v", s)
	}
	for _, role := range ModelRoleNames() {
		if !s.Roles[role] {
			t.Fatalf("role %s off by default", role)
		}
	}
}

func TestModelSettingsReadTheBlock(t *testing.T) {
	s, err := ParseModelSettings("c.toml", "[model]\nenabled = true\nendpoint = \"http://localhost:1\"\nname = \"m\"\ntemperature = 1\n[model.roles]\npropose = true\nplace = false\n")
	if err != nil {
		t.Fatal(err)
	}
	if !s.Enabled || s.Endpoint != "http://localhost:1" || s.Name != "m" || s.Temperature != 1 {
		t.Fatalf("%+v", s)
	}
	// Once roles is said, an unnamed role is off.
	if !s.Roles["propose"] || s.Roles["place"] || s.Roles["describe"] {
		t.Fatalf("roles %v", s.Roles)
	}
}

func TestModelSettingsIgnoreUnknownKeys(t *testing.T) {
	if _, err := ParseModelSettings("c.toml", "[model]\nkeep_alive = 5\n"); err != nil {
		t.Fatal(err)
	}
}

func TestModelSettingsRefuse(t *testing.T) {
	for text, want := range map[string]string{
		"model = 5":                             "[model] must be a table",
		"[model]\nenabled = \"yes\"":            "[model] enabled must be a boolean",
		"[model]\nendpoint = \"\"":              "[model] endpoint must be a non-empty string",
		"[model]\nname = 3":                     "[model] name must be a non-empty string",
		"[model]\ntemperature = true":           "[model] temperature must be a number",
		"[model]\ntemperature = 2.5":            "[model] temperature must lie between 0 and 2",
		"[model]\ntemperature = -0.1":           "[model] temperature must lie between 0 and 2",
		"[model]\nroles = 1":                    "[model] roles must be a table",
		"[model.roles]\nsummarize = true":       `[model] roles has unknown "summarize"`,
		"[model.roles]\npropose = \"on\"":       "[model] roles.propose must be a boolean",
		"[model":                                "not valid TOML",
	} {
		_, err := ParseModelSettings("c.toml", text)
		if err == nil || !strings.Contains(err.Error(), want) || !strings.HasPrefix(err.Error(), "c.toml: ") {
			t.Errorf("%q: %v, want %q", text, err, want)
		}
	}
}

func TestModelSettingsNameAFileThatCannotBeRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A directory of that name declares nothing, as in ReadAreaManifest.
	if _, err := ReadModelSettings(dir); err != nil {
		t.Fatal(err)
	}
}

func TestNarrowedOffBeatsOnAndRolesOnlyShrink(t *testing.T) {
	base := ModelSettings{Enabled: true, Roles: map[string]bool{"propose": true, "place": true}}
	off, on := false, true
	if base.Narrowed(&Manifest{ModelEnabled: &off}).RoleOn("propose") {
		t.Fatal("an area switched the model off and it stayed on")
	}
	if !base.Narrowed(&Manifest{ModelEnabled: &on}).RoleOn("propose") {
		t.Fatal("an area saying on switched it off")
	}
	globalOff := ModelSettings{Roles: base.Roles}
	if globalOff.Narrowed(&Manifest{ModelEnabled: &on}).RoleOn("propose") {
		t.Fatal("an area switched on what is off globally")
	}
	narrowed := base.Narrowed(&Manifest{ModelRoles: map[string]bool{"place": true, "describe": true}})
	if narrowed.RoleOn("propose") || !narrowed.RoleOn("place") || narrowed.RoleOn("describe") {
		t.Fatalf("roles %v", narrowed.Roles)
	}
	if !base.Narrowed(&Manifest{}).RoleOn("propose") {
		t.Fatal("an area that says nothing narrowed the roles")
	}
}

func TestTheDeclarationCarriesTheModelBlock(t *testing.T) {
	m, err := declaration(map[string]any{
		"area":  map[string]any{"scope": "s"},
		"model": map[string]any{"enabled": false, "roles": map[string]any{"propose": true, "place": false}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.ModelEnabled == nil || *m.ModelEnabled || !m.ModelRoles["propose"] || m.ModelRoles["place"] {
		t.Fatalf("%v %v", m.ModelEnabled, m.ModelRoles)
	}
	m, _ = declaration(map[string]any{"area": map[string]any{"scope": "s"}})
	if m.ModelEnabled != nil || m.ModelRoles != nil {
		t.Fatal("an unsaid block must stay unsaid")
	}
}

func TestTheGlobalKeysAreTheOnesTheReaderReads(t *testing.T) {
	want := []string{"enabled", "endpoint", "name", "roles", "temperature"}
	if got := GlobalModelKeys(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatal(got)
	}
}
```

- [ ] **Step 2: Laufen lassen, rot sehen**

Run: `go test ./internal/config/ -run 'Model|Narrowed|Declaration|GlobalKeys' -count=1`
Expected: FAIL, `undefined: ReadModelSettings` u. a.

- [ ] **Step 3: Implementieren**

`internal/config/modelsettings.go`:

```go
package config

// The local model's settings and their cascade, after
// `src/brain/model/settings.py` of the reference.
//
// Two levels decide: the state (`<state>/config.toml`) and the area's
// declaration. Off beats on, and only in that direction: an area can switch
// off what is on globally, never switch on what is off -- otherwise the
// global switch-off would be none. Unknown keys in [model] pass unjudged, as
// in the reference; an unknown role is refused, because a typo would leave a
// role off that the person believes on.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	DefaultModelEndpoint = "http://127.0.0.1:11434"
	DefaultModelName     = "hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL"
)

// ModelRoleNames are the three things the model may be asked to do.
func ModelRoleNames() []string { return []string{"describe", "place", "propose"} }

// GlobalModelKeys are the keys ParseModelSettings reads; the schema of
// `loomux config --global` is held against them.
func GlobalModelKeys() []string {
	return []string{"enabled", "endpoint", "name", "roles", "temperature"}
}

// ModelSettings is one level of the cascade, or the result of both.
type ModelSettings struct {
	Enabled     bool
	Endpoint    string
	Name        string
	Temperature float64
	Roles       map[string]bool // the roles switched on, and only those
}

// ReadModelSettings reads the global file. Anything but a regular file
// declares nothing and yields the defaults.
func ReadModelSettings(stateDir string) (ModelSettings, error) {
	path := filepath.Join(stateDir, "config.toml")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return ParseModelSettings(path, "")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ModelSettings{}, fmt.Errorf("%s: cannot be read: %w", path, err)
	}
	return ParseModelSettings(path, string(data))
}

// ParseModelSettings reads the [model] block of text; path only names the
// file in a refusal.
func ParseModelSettings(path, text string) (ModelSettings, error) {
	s, err := parseModelSettings(text)
	if err != nil {
		return ModelSettings{}, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

func parseModelSettings(text string) (ModelSettings, error) {
	document := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &document); err != nil {
		return ModelSettings{}, fmt.Errorf("not valid TOML: %w", err)
	}
	block := map[string]any{}
	if value, present := document["model"]; present {
		table, ok := value.(map[string]any)
		if !ok {
			return ModelSettings{}, errors.New("[model] must be a table")
		}
		block = table
	}
	enabled, err := optionalBool(block, "enabled", "[model]", " ")
	if err != nil {
		return ModelSettings{}, err
	}
	endpoint, err := modelString(block, "endpoint", DefaultModelEndpoint)
	if err != nil {
		return ModelSettings{}, err
	}
	name, err := modelString(block, "name", DefaultModelName)
	if err != nil {
		return ModelSettings{}, err
	}
	temperature, err := modelTemperature(block)
	if err != nil {
		return ModelSettings{}, err
	}
	roles, err := declaredRoles(block)
	if err != nil {
		return ModelSettings{}, err
	}
	if roles == nil {
		roles = map[string]bool{}
		for _, role := range ModelRoleNames() {
			roles[role] = true
		}
	}
	return ModelSettings{Enabled: enabled, Endpoint: endpoint, Name: name, Temperature: temperature, Roles: roles}, nil
}

func modelString(block map[string]any, key, fallback string) (string, error) {
	if _, present := block[key]; !present {
		return fallback, nil
	}
	return optionalString(block, key, "[model]", " ")
}

// modelTemperature takes an integer as well, as Python's `int | float` does;
// a boolean is no number although TOML's reader would not confuse them.
func modelTemperature(block map[string]any) (float64, error) {
	value, present := block["temperature"]
	if !present {
		return 0, nil
	}
	var number float64
	switch v := value.(type) {
	case int64:
		number = float64(v)
	case float64:
		number = v
	default:
		return 0, fmt.Errorf("[model] temperature must be a number, found %s", tomlType(value))
	}
	if number < 0 || number > 2 {
		return 0, fmt.Errorf("[model] temperature must lie between 0 and 2, found %v", number)
	}
	return number, nil
}

// declaredRoles is nil where the block names no roles, and otherwise the
// roles switched on.
func declaredRoles(block map[string]any) (map[string]bool, error) {
	if err := checkRoles(block); err != nil {
		return nil, err
	}
	raw, present := block["roles"].(map[string]any)
	if !present {
		return nil, nil
	}
	roles := map[string]bool{}
	for name, value := range raw {
		if value == true {
			roles[name] = true
		}
	}
	return roles, nil
}

// optionalFlag is optionalBool that tells an unsaid key from false.
func optionalFlag(table map[string]any, key, owner string) (*bool, error) {
	if _, present := table[key]; !present {
		return nil, nil
	}
	flag, err := optionalBool(table, key, owner, " ")
	if err != nil {
		return nil, err
	}
	return &flag, nil
}

// Narrowed is s with the area's word on it: enabled only where the area does
// not say false, and the roles cut to those the area names.
func (s ModelSettings) Narrowed(m *Manifest) ModelSettings {
	out := s
	out.Enabled = s.Enabled && (m.ModelEnabled == nil || *m.ModelEnabled)
	if m.ModelRoles != nil {
		roles := map[string]bool{}
		for role := range s.Roles {
			if m.ModelRoles[role] {
				roles[role] = true
			}
		}
		out.Roles = roles
	}
	return out
}

// RoleOn says whether the model may be asked for role at all.
func (s ModelSettings) RoleOn(role string) bool { return s.Enabled && s.Roles[role] }
```

Die Fehlermeldung von `checkRoles` für einen Nicht-Bool lautet heute
`[model] roles.propose must be a boolean` über `optionalBool(roles, name,
"[model]", " roles.")`; der Test oben erwartet genau das. Stimmt es nicht,
den Test an die Meldung anpassen, nicht umgekehrt.

In `manifest.go`, im Struct `Manifest` nach `MergeBranch`:

```go
	// ModelEnabled and ModelRoles are the area's word on the local model;
	// nil where the declaration says nothing. ModelRoles holds the roles
	// switched on. config.ModelSettings.Narrowed reads both.
	ModelEnabled *bool
	ModelRoles   map[string]bool
```

In `declaration.go` ersetzt ihr Füllen die zwei reinen Prüfungen
(`:101-106`):

```go
	modelEnabled, err := optionalFlag(sections["model"], "enabled", "[model]")
	if err != nil {
		return nil, err
	}
	modelRoles, err := declaredRoles(sections["model"])
	if err != nil {
		return nil, err
	}
```

und im Rückgabewert `ModelEnabled: modelEnabled, ModelRoles: modelRoles,`.

- [ ] **Step 4: Grün sehen, ganzes Paket**

Run: `go test ./internal/config/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/modelsettings.go internal/config/modelsettings_test.go internal/config/manifest.go internal/config/declaration.go
git commit -m "feat(config): read the local model's settings globally and per area"
```

---

### Task 2: `loomux config --global` kennt `[model]`

**Files:**
- Modify: `internal/config/schema/schema.go` (Art `Float`, `GlobalKeys`)
- Modify: `internal/config/schema/current.go:32-38` (`CurrentOf`)
- Modify: `internal/config/schema/schema_test.go:73-78` (Test ersetzen)
- Modify: `internal/config/edit/value.go:14-44` (`Float`)
- Modify: `internal/config/edit/value_test.go`
- Modify: `internal/cli/config.go:144-157,268-276,309-318`
- Modify: `internal/cli/config_test.go:425-433` (Test ersetzen)

**Interfaces:**
- Consumes: `config.GlobalModelKeys`, `config.ParseModelSettings`, `config.DefaultModelEndpoint`, `config.DefaultModelName` (Task 1)
- Produces: `schema.Float` (Kind), `schema.CurrentOf(keys []Key, text string) ([]Entry, error)`; `schema.Current(text)` bleibt als `CurrentOf(Keys(), text)`

- [ ] **Step 1: Failing tests schreiben**

In `schema_test.go` ersetzt dieser Test `TestGlobalKeysAreEmptyForNow`:

```go
func TestTheGlobalKeysAreTheModelReadersKeys(t *testing.T) {
	var got []string
	for _, k := range GlobalKeys() {
		if k.Section != "model" || k.Module != Brain || k.Doc == "" {
			t.Errorf("%+v", k)
		}
		got = append(got, k.Name)
	}
	if !slices.Equal(sorted(got), sorted(config.GlobalModelKeys())) {
		t.Fatalf("schema %v, reader %v", got, config.GlobalModelKeys())
	}
	defaults := map[string]string{}
	for _, k := range GlobalKeys() {
		defaults[k.Name] = k.Default
	}
	if defaults["enabled"] != "false" || defaults["temperature"] != "0.0" ||
		defaults["endpoint"] != strconv.Quote(config.DefaultModelEndpoint) ||
		defaults["name"] != strconv.Quote(config.DefaultModelName) ||
		defaults["roles"] != "{ describe = true, place = true, propose = true }" {
		t.Fatalf("%v", defaults)
	}
}

func TestCurrentOfReadsTheKeysItIsGiven(t *testing.T) {
	entries, err := CurrentOf(GlobalKeys(), "[model]\nendpoint = \"http://localhost:1\"\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		switch e.Key.Name {
		case "endpoint":
			if e.Origin != Set || e.Input != "http://localhost:1" {
				t.Errorf("%+v", e)
			}
		case "temperature":
			if e.Origin != Default || e.Value != "0.0" {
				t.Errorf("%+v", e)
			}
		}
	}
}
```

(`strconv` in die Imports.) In `value_test.go`:

```go
func TestRenderAFloat(t *testing.T) {
	for input, want := range map[string]string{"0": "0.0", "0.7": "0.7", " 1.25 ": "1.25", "2": "2.0", "1e-1": "0.1"} {
		if got, err := Render(schema.Float, input); err != nil || got != want {
			t.Errorf("%q: %q %v, want %q", input, got, err, want)
		}
	}
	if _, err := Render(schema.Float, "warm"); err == nil {
		t.Fatal("a word passed as a number")
	}
}
```

In `config_test.go` ersetzt dieser Test `TestConfigGlobalKnowsNoKeyYet`:

```go
func TestConfigGlobalEditsTheModelBlock(t *testing.T) {
	state := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", state)
	if code, out, _ := runConfig(t, "", "list", "--global"); code != 0 || !strings.Contains(out, "model.endpoint") || !strings.Contains(out, "http://127.0.0.1:11434") {
		t.Fatalf("%d %q", code, out)
	}
	// No [area] is asked for: the global file has none.
	if code, _, errOut := runConfig(t, "", "set", "model.enabled", "true", "--yes", "--global"); code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	if code, _, _ := runConfig(t, "", "set", "model.temperature", "0.3", "--yes", "--global"); code != 0 {
		t.Fatal(code)
	}
	data, _ := os.ReadFile(filepath.Join(state, "config.toml"))
	if !strings.Contains(string(data), "enabled = true") || !strings.Contains(string(data), "temperature = 0.3") {
		t.Fatalf("%q", data)
	}
	// The reader in operation judges the value.
	if code, _, errOut := runConfig(t, "", "set", "model.temperature", "3", "--yes", "--global"); code != 1 || !strings.Contains(errOut, "between 0 and 2") {
		t.Fatalf("%d %s", code, errOut)
	}
	// A project key is still no global key.
	if code, _, _ := runConfig(t, "", "set", "commit.threshold", "4", "--yes", "--global"); code != 1 {
		t.Fatal(code)
	}
}
```

`TestConfigProposalsForTheGlobalFile` (`config_proposals_test.go:535`) nutzt
`commit.threshold` als unbekannten globalen Schlüssel und bleibt gültig.

- [ ] **Step 2: Rot sehen**

Run: `go test ./internal/config/... ./internal/cli/ -run 'Global|CurrentOf|Float' -count=1`
Expected: FAIL (`undefined: schema.Float`, `undefined: CurrentOf`).

- [ ] **Step 3: Implementieren**

`schema.go`: `Float` nach `Int` in die `const`-Liste der Arten, und:

```go
// GlobalKeys are the keys of the per-user file `<state>/config.toml`: the
// local model's settings. An area may only narrow enabled and roles.
func GlobalKeys() []Key {
	return []Key{
		{Section: "model", Name: "enabled", Kind: Bool, Default: "false", Module: Brain, Doc: "Let the local model be asked at all; an area can only switch it off."},
		{Section: "model", Name: "endpoint", Kind: String, Default: strconv.Quote(config.DefaultModelEndpoint), Module: Brain, Doc: "Where Ollama listens; it must stay on the loopback."},
		{Section: "model", Name: "name", Kind: String, Default: strconv.Quote(config.DefaultModelName), Module: Brain, Doc: "The Ollama model that is asked."},
		{Section: "model", Name: "roles", Kind: Table, Default: "{ describe = true, place = true, propose = true }", Module: Brain, Doc: "Which roles the model takes; once set, an unnamed role is off."},
		{Section: "model", Name: "temperature", Kind: Float, Default: "0.0", Module: Brain, Doc: "The sampling temperature, between 0 and 2."},
	}
}
```

`strconv.Quote` ergibt für beide Vorgaben dasselbe wie `config.QuoteTOML`,
weil sie weder Backslash noch Steuerzeichen tragen; der Test in Step 1 hält
es fest.

`current.go`: den Körper von `Current` nach `CurrentOf(keys []Key, text
string)` verschieben, `for _, k := range keys`, und

```go
// Current is CurrentOf over the project file's keys.
func Current(text string) ([]Entry, error) { return CurrentOf(Keys(), text) }
```

`edit/value.go`, vor `case schema.Bool`:

```go
	case schema.Float:
		f, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return "", fmt.Errorf("%q is not a number", input)
		}
		// Canonical, and always with a point: TOML reads 2 as an integer,
		// and the key's default is compared by text.
		text := strconv.FormatFloat(f, 'f', -1, 64)
		if !strings.Contains(text, ".") {
			text += ".0"
		}
		return text, nil
```

`cli/config.go`:
- `entries`: `schema.CurrentOf(t.keys, text)` statt `schema.Current(text)`; der Filter über `t.lookup` fällt weg, weil `CurrentOf` nur `t.keys` liest.
- `proposeChange`: die Bedingung wird `if key.Module == schema.Brain && !t.global && id != "area.scope"`.
- `validated`:

```go
// validated hands a new text to the reader that runs in operation: the
// declaration readers for a project file, the model reader for the global
// one.
func (t configTarget) validated(next string) (string, error) {
	if t.global {
		if _, err := config.ParseModelSettings(t.path, next); err != nil {
			return "", err
		}
		return next, nil
	}
	if err := schema.Validate(next); err != nil {
		return "", err
	}
	return next, nil
}
```

- [ ] **Step 4: Grün sehen**

Run: `go test ./internal/config/... ./internal/cli/ -count=1`
Expected: PASS. `TestTheSchemaKnowsEveryKeyTheReadersRead` bleibt grün, weil
`DeclarationKeys()["model"]` weiter `enabled, roles` ist.

- [ ] **Step 5: Commit**

```bash
git add internal/config/schema internal/config/edit internal/cli/config.go internal/cli/config_test.go
git commit -m "feat(config): let config --global list and edit the local model's settings"
```

---

### Task 3: Der Client auf Loopback

**Files:**
- Create: `internal/brain/model/client.go`
- Create: `internal/brain/model/client_test.go`

**Interfaces:**
- Consumes: `config.ModelSettings` (Task 1)
- Produces:
  - `func GuardEndpoint(endpoint string) error`
  - `type Client struct` (unexportierte Felder `settings`, `http *http.Client`, `url string`)
  - `func NewClient(s config.ModelSettings) (*Client, error)` — ruft `GuardEndpoint`
  - `func (c *Client) Ask(ctx context.Context, prompt string) (string, bool)` — `false` für jeden Ausfall

- [ ] **Step 1: Failing tests schreiben**

```go
package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/config"
)

func settingsFor(endpoint string) config.ModelSettings {
	return config.ModelSettings{Enabled: true, Endpoint: endpoint, Name: "m", Temperature: 0.2, Roles: map[string]bool{"propose": true}}
}

func TestGuardEndpointRefusesAnythingOffTheLoopback(t *testing.T) {
	for endpoint, want := range map[string]string{
		" http://localhost:11434":             "spaces or control characters",
		"http://local\thost:1":                "spaces or control characters",
		"http://localhost:1/\x7f":             "spaces or control characters",
		"ftp://localhost/x":                   "must use http or https",
		"//127.0.0.1:11434":                   "must use http or https",
		"http://localhost:notaport":           "port must be a number",
		"http://localhost:70000":              "port must be a number",
		"http://localhost:99999999999999999999": "port must be a number",
		"http://192.0.2.1:11434":              "must stay on the loopback",
		"http://[::1]:11434@evil.example.com": "is not a readable address",
		"http://127.0.0.2:11434":              "must stay on the loopback",
	} {
		err := GuardEndpoint(endpoint)
		if err == nil || !strings.Contains(err.Error(), want) || !strings.HasPrefix(err.Error(), "[model] endpoint") {
			t.Errorf("%q: %v, want %q", endpoint, err, want)
		}
	}
}

// Review Focus 2: Python reads the host lower-cased.
func TestGuardEndpointLetsTheLoopbackThrough(t *testing.T) {
	for _, endpoint := range []string{"http://127.0.0.1:11434", "https://localhost", "http://[::1]:11434/", "http://LOCALHOST:11434"} {
		if err := GuardEndpoint(endpoint); err != nil {
			t.Errorf("%q: %v", endpoint, err)
		}
	}
}

func TestAskPostsTheReferencesPayload(t *testing.T) {
	var got map[string]any
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.Method + " " + r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"response":"answer"}`))
	}))
	defer server.Close()
	client, err := NewClient(settingsFor(server.URL + "//"))
	if err != nil {
		t.Fatal(err)
	}
	text, ok := client.Ask(context.Background(), "frage")
	if !ok || text != "answer" || path != "POST /api/generate" {
		t.Fatalf("%q %v %s", text, ok, path)
	}
	options, _ := got["options"].(map[string]any)
	if got["model"] != "m" || got["prompt"] != "frage" || got["stream"] != false || got["think"] != false ||
		options["temperature"] != 0.2 || options["num_ctx"] != float64(8192) || len(got) != 5 {
		t.Fatalf("%v", got)
	}
}

func TestAskCountsEveryOutageAsNoAnswer(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"500":         func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) },
		"not json":    func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) },
		"not object":  func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`["x"]`)) },
		"no response": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"done":true}`)) },
		"not string":  func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"response":3}`)) },
	} {
		server := httptest.NewServer(handler)
		client, _ := NewClient(settingsFor(server.URL))
		if text, ok := client.Ask(context.Background(), "p"); ok || text != "" {
			t.Errorf("%s: %q %v", name, text, ok)
		}
		server.Close()
	}
}

func TestAnEmptyResponseIsAnAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"response":""}`))
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	if text, ok := client.Ask(context.Background(), "p"); !ok || text != "" {
		t.Fatalf("%q %v", text, ok)
	}
}

// Review Focus 3.
func TestARedirectIsNoAnswerAndNoSecondRequest(t *testing.T) {
	var foreign atomic.Int32
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		foreign.Add(1)
		_, _ = w.Write([]byte(`{"response":"leaked"}`))
	}))
	defer elsewhere.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/api/generate", http.StatusFound)
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	if _, ok := client.Ask(context.Background(), "p"); ok || foreign.Load() != 0 {
		t.Fatalf("ok %v, foreign %d", ok, foreign.Load())
	}
}

// Go's ProxyFromEnvironment never proxies a loopback address anyway, so a
// request through a proxy server would pass with the default transport as
// well; what is held here is the transport itself.
func TestTheClientTakesNoProxyFromTheEnvironment(t *testing.T) {
	client, _ := NewClient(settingsFor("http://127.0.0.1:1"))
	if transport, ok := client.http.Transport.(*http.Transport); !ok || transport.Proxy != nil {
		t.Fatal("the transport may take a proxy")
	}
}

func TestABodyCutShortIsNoAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Raw on the hijacked connection: a body that promises 100 bytes
		// and ends after seven.
		conn, _, _ := w.(http.Hijacker).Hijack()
		_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n{\"respo"))
		_ = conn.Close()
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	if _, ok := client.Ask(context.Background(), "p"); ok {
		t.Fatal("half a body answered")
	}
}

func TestAskGivesUpAfterItsTimeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { <-release }))
	defer server.Close()
	defer close(release)
	client, _ := NewClient(settingsFor(server.URL))
	client.http.Timeout = 50 * time.Millisecond
	if _, ok := client.Ask(context.Background(), "p"); ok {
		t.Fatal("a hanging server answered")
	}
}

func TestNewClientUsesTheReferencesTimeouts(t *testing.T) {
	client, _ := NewClient(settingsFor("http://127.0.0.1:1"))
	if client.http.Timeout != 30*time.Second || connectTimeout != 2*time.Second {
		t.Fatal(client.http.Timeout, connectTimeout)
	}
	if _, err := NewClient(settingsFor("http://192.0.2.1")); err == nil {
		t.Fatal("a client was built for a foreign host")
	}
}

func TestANobodyListeningIsNoAnswer(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	endpoint := server.URL
	server.Close()
	client, _ := NewClient(settingsFor(endpoint))
	if _, ok := client.Ask(context.Background(), "p"); ok {
		t.Fatal("a closed port answered")
	}
}
```

- [ ] **Step 2: Rot sehen**

Run: `go test ./internal/brain/model/ -count=1`
Expected: FAIL (Paket leer).

- [ ] **Step 3: Implementieren**

```go
// Package model is the local model: a client that never leaves the loopback,
// the gate in front of it, and the role `propose`. It follows
// `src/brain/model/` of the reference.
//
// Two kinds of failure are kept strictly apart. A misconfiguration -- an
// address off the loopback -- is an error, because a fallback would hide it
// and the next run would speak outwards again. An outage -- Ollama down, too
// slow, answering nonsense -- is no answer, and no answer means a manual case
// everywhere, never the cloud.
package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// Connect short, read long: 2 s tell "Ollama is not running" from "Ollama is
// computing", 30 s carry a cold start (the reference measured 6.1 s at worst).
const (
	connectTimeout = 2 * time.Second
	totalTimeout   = 30 * time.Second
	numCtx         = 8192
)

// GuardEndpoint refuses every address that is not the loopback, in the
// reference's order (`client.py:59-109`). The host is compared as a string,
// lower-cased as Python's `hostname` reads it; nothing is resolved.
func GuardEndpoint(endpoint string) error {
	for _, r := range endpoint {
		if r <= ' ' || r == 0x7f {
			return fmt.Errorf("[model] endpoint must not contain spaces or control characters, found %s", pytext.Repr(endpoint))
		}
	}
	parts, err := url.Parse(endpoint)
	if err != nil {
		if strings.Contains(err.Error(), "invalid port") {
			return fmt.Errorf("[model] endpoint port must be a number, found %s", pytext.Repr(endpoint))
		}
		return fmt.Errorf("[model] endpoint is not a readable address, found %s", pytext.Repr(endpoint))
	}
	if parts.Scheme != "http" && parts.Scheme != "https" {
		return fmt.Errorf("[model] endpoint must use http or https, found %s", pytext.Repr(endpoint))
	}
	// url.Parse takes any run of digits; Python's `port` raises outside
	// 0..65535, and that is a misconfiguration, not an outage.
	if port := parts.Port(); port != "" {
		if n, err := strconv.Atoi(port); err != nil || n > 65535 {
			return fmt.Errorf("[model] endpoint port must be a number, found %s", pytext.Repr(endpoint))
		}
	}
	switch strings.ToLower(parts.Hostname()) {
	case "127.0.0.1", "localhost", "::1":
		return nil
	}
	return fmt.Errorf("[model] endpoint must stay on the loopback (127.0.0.1, ::1, localhost), found %s", pytext.Repr(endpoint))
}

// Client asks one Ollama on the loopback.
type Client struct {
	settings config.ModelSettings
	http     *http.Client
	url      string
}

// NewClient guards the endpoint and builds the client. No proxy is taken
// from the environment -- the request would go through a foreign machine
// while the guard saw only the entered string -- and no redirect is
// followed: the guard checks the entered address, not one a 302 names.
func NewClient(s config.ModelSettings) (*Client, error) {
	if err := GuardEndpoint(s.Endpoint); err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy:       nil,
		DialContext: (&net.Dialer{Timeout: connectTimeout}).DialContext,
	}
	return &Client{
		settings: s,
		url:      strings.TrimRight(s.Endpoint, "/") + "/api/generate",
		http: &http.Client{
			Transport:     transport,
			Timeout:       totalTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

type generateRequest struct {
	Model   string          `json:"model"`
	Prompt  string          `json:"prompt"`
	Stream  bool            `json:"stream"`
	Think   bool            `json:"think"`
	Options generateOptions `json:"options"`
}

type generateOptions struct {
	Temperature float64 `json:"temperature"`
	NumCtx      int     `json:"num_ctx"`
}

// Ask is one question to the model. false stands for every outage: no
// connection, a timeout, a status that is not 2xx (a redirect included), a
// body that is not a JSON object with a string `response`. An empty string
// is an answer; the role judges it.
func (c *Client) Ask(ctx context.Context, prompt string) (string, bool) {
	body, err := json.Marshal(generateRequest{
		Model: c.settings.Name, Prompt: prompt,
		Options: generateOptions{Temperature: c.settings.Temperature, NumCtx: numCtx},
	})
	if err != nil {
		return "", false
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return "", false
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return "", false
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", false
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return "", false
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return "", false
	}
	object, _ := decoded.(map[string]any)
	text, ok := object["response"].(string)
	return text, ok
}
```

`json.Marshal` eines festen Structs scheitert nie, `NewRequestWithContext`
nie bei einer geprüften URL: Wenn die Coverage die beiden Zweige nicht
erreicht, fallen sie weg (`body, _ := …`) statt einer Ausnahme. Der Zweig
„`io.ReadAll` scheitert“ ist erreichbar und bekommt einen Test (unten).
`http://[::1]:11434@evil.example.com` scheitert in Go wie in Python schon
beim Zerlegen (`net/url: invalid userinfo`, am 2026-09-25 mit `go run`
geprüft) und bekommt dieselbe Meldung.

- [ ] **Step 4: Grün sehen**

Run: `go test ./internal/brain/model/ -count=1 -cover`
Expected: PASS, 100.0 %.

- [ ] **Step 5: Commit**

```bash
git add internal/brain/model
git commit -m "feat(model): ask a local Ollama that never leaves the loopback"
```

---

### Task 4: Tor, Prompt und die Rolle `propose`

**Files:**
- Create: `internal/brain/model/prompts/vorschlag-v4.md` (Byte für Byte aus der Referenz)
- Create: `internal/brain/model/propose.go`
- Create: `internal/brain/model/propose_test.go`
- Modify: `.gitattributes` (Zeile `internal/brain/model/prompts/** -text`)

**Interfaces:**
- Consumes: `NewClient`, `(*Client).Ask` (Task 3); `config.ModelSettings.Narrowed`, `RoleOn` (Task 1); `evidence.ReadProposal`, `evidence.CheckEvidence`, `evidence.Segment`
- Produces:
  - `const ProposeVersion = "vorschlag-v4"`
  - `type Proposer struct` (Feld `client *Client`)
  - `func ProposerFor(s config.ModelSettings, m *config.Manifest, role string) (*Proposer, error)` — `nil, nil`, wenn die Rolle aus ist
  - `func (p *Proposer) Propose(ctx context.Context, pkg string, segments []evidence.Segment) (string, bool)`

- [ ] **Step 1: Prompt holen und prüfen**

```bash
git -C "C:/Users/micro/Documents/#GIT/ultra-brain" show loomux-3-source:src/brain/prompts/vorschlag-v4.md > internal/brain/model/prompts/vorschlag-v4.md
sha256sum internal/brain/model/prompts/vorschlag-v4.md
```

Expected: `05e2ebbbf448a0b5ce8562719b9cd564dfd1f62dac1b8f199fed02aa0c19b1ab`.
Die Zeile in `.gitattributes` kommt vor dem ersten `git add`, sonst
normalisiert `eol=lf` nichts, aber ein Checkout mit `autocrlf` bekäme CRLF.

- [ ] **Step 2: Failing tests schreiben**

```go
package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/evidence"
	"github.com/xidus90/loomux/internal/config"
)

const goodProposal = "## B1 - Die Quelle traegt jetzt einen neuen Stand.\n\nevidence: D1\n\n```\n+neu\n```\n"
const inventedProposal = "## B1 - Die Quelle traegt jetzt einen erfundenen Stand.\n\nevidence: D1\n\n```\n+erfunden\n```\n"

var segments = []evidence.Segment{{Number: "D1", Kind: "D", Label: "diff", Body: "@@ -1 +1 @@\n-alt\n+neu\n"}}

func TestThePromptIsTheReferencesByteForByte(t *testing.T) {
	sum := sha256.Sum256([]byte(proposePrompt))
	if hex.EncodeToString(sum[:]) != "05e2ebbbf448a0b5ce8562719b9cd564dfd1f62dac1b8f199fed02aa0c19b1ab" {
		t.Fatal("vorschlag-v4.md is not the reference's")
	}
	// Replacing {paket} is str.format only while no other brace is there.
	if strings.Count(proposePrompt, "{") != 1 || strings.Count(proposePrompt, "}") != 1 || !strings.Contains(proposePrompt, "{paket}") {
		t.Fatal("the prompt carries braces str.format would read")
	}
}

func answering(t *testing.T, answer string, prompts *[]string) config.ModelSettings {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		if prompts != nil {
			*prompts = append(*prompts, got["prompt"].(string))
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"response": answer})
	}))
	t.Cleanup(server.Close)
	return config.ModelSettings{Enabled: true, Endpoint: server.URL, Name: "m", Roles: map[string]bool{"propose": true}}
}

func TestProposeKeepsAProposalTheEvidencePasses(t *testing.T) {
	var prompts []string
	p, err := ProposerFor(answering(t, goodProposal, &prompts), &config.Manifest{}, "propose")
	if err != nil || p == nil {
		t.Fatal(p, err)
	}
	text, ok := p.Propose(context.Background(), "PAKET", segments)
	if !ok || text != goodProposal {
		t.Fatalf("%q %v", text, ok)
	}
	if len(prompts) != 1 || prompts[0] != strings.ReplaceAll(proposePrompt, "{paket}", "PAKET") {
		t.Fatal("the prompt is not the file with the package in it")
	}
}

func TestProposeDropsWhatTheEvidenceRefuses(t *testing.T) {
	for name, answer := range map[string]string{
		"invented":  inventedProposal,
		"one of two": goodProposal + "\n" + strings.Replace(inventedProposal, "B1", "B2", 1),
		"no claims": "Ich weiss es nicht.",
		"empty":     "",
	} {
		p, _ := ProposerFor(answering(t, answer, nil), &config.Manifest{}, "propose")
		if text, ok := p.Propose(context.Background(), "P", segments); ok || text != "" {
			t.Errorf("%s: %q %v", name, text, ok)
		}
	}
}

func TestProposeWithoutAnAnswerIsNone(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	s := config.ModelSettings{Enabled: true, Endpoint: server.URL, Roles: map[string]bool{"propose": true}}
	server.Close()
	p, _ := ProposerFor(s, &config.Manifest{}, "propose")
	if _, ok := p.Propose(context.Background(), "P", segments); ok {
		t.Fatal("a closed port proposed")
	}
}

func TestTheGateBuildsNoClientForARoleThatIsOff(t *testing.T) {
	off := false
	foreign := config.ModelSettings{Enabled: true, Endpoint: "http://192.0.2.1", Roles: map[string]bool{"propose": true}}
	for name, c := range map[string]struct {
		s config.ModelSettings
		m *config.Manifest
	}{
		"global off":   {config.ModelSettings{Endpoint: "http://192.0.2.1", Roles: foreign.Roles}, &config.Manifest{}},
		"area off":     {foreign, &config.Manifest{ModelEnabled: &off}},
		"role off":     {foreign, &config.Manifest{ModelRoles: map[string]bool{"place": true}}},
	} {
		// A foreign endpoint that passes unnoticed proves no client was built.
		if p, err := ProposerFor(c.s, c.m, "propose"); p != nil || err != nil {
			t.Errorf("%s: %v %v", name, p, err)
		}
	}
	if _, err := ProposerFor(foreign, &config.Manifest{}, "propose"); err == nil {
		t.Fatal("a foreign endpoint was accepted once the role was on")
	}
}
```

Welche Zeilen `evidence.ReadProposal` als unzulässig verwirft, bestimmt
`evidence`; die Überschrift `## B1 - …` ist die aus
`test_reconcile_proposal.py`. Scheitert `TestProposeKeepsAProposalTheEvidencePasses`,
weil `CheckEvidence` den Segmentkörper anders zerlegt, das Segment an
`evidence_test.go` angleichen, nicht die Rolle.

- [ ] **Step 3: Rot sehen**

Run: `go test ./internal/brain/model/ -count=1`
Expected: FAIL (`undefined: ProposerFor`).

- [ ] **Step 4: Implementieren**

```go
package model

import (
	"context"
	_ "embed"
	"strings"

	"github.com/xidus90/loomux/internal/brain/evidence"
	"github.com/xidus90/loomux/internal/config"
)

// ProposeVersion names the prompt in every case it produced: of four measured
// versions two made a tier worse while reading like improvements, so a fallen
// hit rate must be attributable to the prompt or the model later.
const ProposeVersion = "vorschlag-v4"

// proposePrompt is the reference's file byte for byte, its version comment
// included: the whole file goes to the model, as `load().format()` sends it.
//
//go:embed prompts/vorschlag-v4.md
var proposePrompt string

// Proposer is the local model in the role `propose`.
type Proposer struct {
	client *Client
}

// ProposerFor is the gate: a proposer for this area and role, or none. No
// client -- no address set up -- exists unless the role is on after the
// area's word; only then is the endpoint judged. The privacy mode is not
// asked here: it decides what happens to a proposal, and the caller hands
// out proposers only for closed areas.
func ProposerFor(s config.ModelSettings, m *config.Manifest, role string) (*Proposer, error) {
	narrow := s.Narrowed(m)
	if !narrow.RoleOn(role) {
		return nil, nil
	}
	client, err := NewClient(narrow)
	if err != nil {
		return nil, err
	}
	return &Proposer{client: client}, nil
}

// Propose asks for a proposal on pkg and keeps it only if every claim passes
// the evidence binding `approve` holds it to later. No answer, nothing
// readable as a claim, one invented quote among good ones: all three are no
// proposal, and the caller cannot tell them apart -- on purpose.
func (p *Proposer) Propose(ctx context.Context, pkg string, segments []evidence.Segment) (string, bool) {
	answer, ok := p.client.Ask(ctx, strings.ReplaceAll(proposePrompt, "{paket}", pkg))
	if !ok {
		return "", false
	}
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(answer), segments)
	if len(passed) == 0 || len(complaints) > 0 {
		return "", false
	}
	return answer, true
}
```

- [ ] **Step 5: Grün sehen, Startzeit prüfen**

Run: `go test ./internal/brain/model/ -count=1 -cover`
Expected: PASS, 100.0 %.

Run: `go build -o bin/loomux.exe ./cmd/loomux` und danach `GODEBUG=inittrace=1 bin/loomux.exe --version 2>&1 | grep -i model`
Expected: keine Zeile für `internal/brain/model` mit Zuweisungen über 0 B; ein
eingebetteter `string` kostet beim Start nichts.

- [ ] **Step 6: Commit**

```bash
git add .gitattributes internal/brain/model
git commit -m "feat(model): propose a change only when every claim passes the evidence binding"
```

---

### Task 5: `reconcile` fragt den Proposer

**Files:**
- Modify: `internal/brain/maintenance/reconcile.go` (Kopfkommentar `:20-31`, `localOnlyNote`-Block `:61-69`, `ReconcileContext` `:121-170`, `sourceCases` `:357-395`, `mergeCases` und `landMerge`, `landCase` `:729-807`, `noteFor` `:822-832`)
- Create: `internal/brain/maintenance/proposal_test.go`
- Modify: `internal/brain/maintenance/reconcile_test.go:497-520` (Test benennt 3a)

**Interfaces:**
- Consumes: `model.ProposerFor`, `(*model.Proposer).Propose`, `model.ProposeVersion` (Task 4); `config.ReadModelSettings` (Task 1)
- Produces: keine neue API; `landCase` bekommt `ctx context.Context` als ersten und `proposer *model.Proposer` als letzten Parameter

- [ ] **Step 1: Failing tests schreiben**

`proposal_test.go` im externen Testpaket `maintenance_test`, auf den Welten
von `reconcile_test.go` (`sourceArea`, `changedSource`, `caseDirOf`,
`mustReconcile`):

```go
package maintenance_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

const rejectedNote = "manual review: the local proposer returned no usable proposal (slice-6 spec §3)"
const localOnlyNote = "manual review: this area is local_only, so no skill path is offered (spec 5)"

// ollama answers every request with answer and counts the requests.
func ollama(t *testing.T, answer string) (string, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]string{"response": answer})
	}))
	t.Cleanup(server.Close)
	return server.URL, &calls
}

func modelOn(t *testing.T, stateDir, endpoint string) {
	t.Helper()
	text := "[model]\nenabled = true\nendpoint = \"" + endpoint + "\"\n"
	if err := os.WriteFile(filepath.Join(stateDir, "config.toml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// changedClosedSource is the case world of reconcile_test.go in a closed area.
// Its package's first diff segment D1 carries the added line below.
func changedClosedSource(t *testing.T) (*world, string) {
	area := sourceArea("project/a")
	area.PrivacyMode = "local_only"
	return changedSource(t, area), "+func B() int { return 1 }"
}

func proposalQuoting(line string) string {
	return "## B1 - Die Quelle hat eine Funktion B bekommen.\n\nevidence: D1\n\n```\n" + line + "\n```\n"
}

func TestALocalOnlyCaseWithAProposalIsNotManual(t *testing.T) {
	w, added := changedClosedSource(t)
	endpoint, calls := ollama(t, proposalQuoting(added))
	modelOn(t, w.StateDir, endpoint)
	raised := mustReconcile(t, w, w.Now()).Cases[0]
	if raised.Manual || raised.Note != "" || !raised.LocalOnly || raised.PromptVersion != "vorschlag-v4" || calls.Load() != 1 {
		t.Fatalf("%+v, calls %d", raised, calls.Load())
	}
	data, err := os.ReadFile(filepath.Join(caseDirOf(w, raised), "proposal.md"))
	if err != nil || string(data) != proposalQuoting(added) {
		t.Fatalf("%q %v", data, err)
	}
	written, _ := maintenance.ReadCase(filepath.Join(caseDirOf(w, raised), "case.toml"))
	if written.PromptVersion != "vorschlag-v4" || written.Manual {
		t.Fatalf("%+v", written)
	}
}

func TestARefusedProposalMakesAManualCaseWithTheSecondNote(t *testing.T) {
	w, _ := changedClosedSource(t)
	endpoint, calls := ollama(t, proposalQuoting("+erfunden"))
	modelOn(t, w.StateDir, endpoint)
	raised := mustReconcile(t, w, w.Now()).Cases[0]
	if !raised.Manual || raised.Note != rejectedNote || raised.PromptVersion != "" || calls.Load() != 1 {
		t.Fatalf("%+v", raised)
	}
	if _, err := os.Stat(filepath.Join(caseDirOf(w, raised), "proposal.md")); err == nil {
		t.Fatal("a refused proposal was written")
	}
}

func TestAnUnreachableModelIsARefusedProposalToo(t *testing.T) {
	w, _ := changedClosedSource(t)
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	modelOn(t, w.StateDir, server.URL)
	if raised := mustReconcile(t, w, w.Now()).Cases[0]; !raised.Manual || raised.Note != rejectedNote {
		t.Fatalf("%+v", raised)
	}
}

func TestWithTheModelOffTheOldNoteStays(t *testing.T) {
	w, _ := changedClosedSource(t)
	if raised := mustReconcile(t, w, w.Now()).Cases[0]; !raised.Manual || raised.Note != localOnlyNote {
		t.Fatalf("%+v", raised)
	}
}

// Review Focus 4.
func TestAnAreaThatSwitchesTheModelOffIsNotAsked(t *testing.T) {
	area := sourceArea("project/a")
	area.PrivacyMode = "local_only"
	area.Declaration = "[model]\nenabled = false\n"
	w := changedSource(t, area)
	endpoint, calls := ollama(t, "x")
	modelOn(t, w.StateDir, endpoint)
	if raised := mustReconcile(t, w, w.Now()).Cases[0]; raised.Note != localOnlyNote || calls.Load() != 0 {
		t.Fatalf("%+v, calls %d", raised, calls.Load())
	}
}

func TestAnOpenAreaIsNeverAsked(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	endpoint, calls := ollama(t, "x")
	modelOn(t, w.StateDir, endpoint)
	if raised := mustReconcile(t, w, w.Now()).Cases[0]; raised.Manual || raised.Note != "" || calls.Load() != 0 {
		t.Fatalf("%+v, calls %d", raised, calls.Load())
	}
}

// A broken [model] must not stop a run that would not have used it.
func TestAnOpenVaultNeverReadsTheModelBlock(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	if err := os.WriteFile(filepath.Join(w.StateDir, "config.toml"), []byte("model = 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustReconcile(t, w, w.Now())
}

func TestAMisconfiguredModelStopsTheRunBeforeAnyCase(t *testing.T) {
	for name, text := range map[string]string{
		"broken block":     "model = 5\n",
		"off the loopback": "[model]\nenabled = true\nendpoint = \"http://192.0.2.1:11434\"\n",
	} {
		w, _ := changedClosedSource(t)
		if err := os.WriteFile(filepath.Join(w.StateDir, "config.toml"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now()); err == nil || !strings.Contains(err.Error(), "[model]") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Review Focus 1: a pass that ends while the model computes writes nothing.
func TestACancelledPassWritesNoCaseWhileTheModelComputes(t *testing.T) {
	w, _ := changedClosedSource(t)
	ctx, cancel := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		cancel()
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	modelOn(t, w.StateDir, server.URL)
	_, err := maintenance.ReconcileContext(ctx, w.Areas, w.Lookup(), w.Now())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
	root, _ := maintenance.ReviewRoot(w.Areas, w.Lookup())
	if entries, _ := os.ReadDir(filepath.Join(root, "project-a")); len(entries) != 0 {
		t.Fatalf("a case was written: %v", entries)
	}
}

// The same cancellation over a standing case on an older source state: it
// must stay as it was, proposal and all.
func TestACancelledPassKeepsTheStandingCase(t *testing.T) {
	w, _ := changedClosedSource(t)
	standing := mustReconcile(t, w, w.Now()).Cases[0]
	dir := caseDirOf(w, standing)
	if err := os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("kept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc C() int { return 2 }\n")
	ctx, cancel := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		cancel()
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	modelOn(t, w.StateDir, server.URL)
	if _, err := maintenance.ReconcileContext(ctx, w.Areas, w.Lookup(), w.Now().Add(24*time.Hour)); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "proposal.md")); err != nil || string(data) != "kept\n" {
		t.Fatalf("the standing case was touched: %q %v", data, err)
	}
}
```

Hat `areaOptions` kein Feld für zusätzlichen Deklarationstext,
bekommt es eines, `Declaration string`, das `addArea` an die geschriebene
`.loomux/config.toml` anhängt (`world_test.go:147`). Der Ordnername des
Bereichs unter dem Prüfzentrum ist `search.CollectionName("project/a")`; wenn
er nicht `project-a` lautet, den Test mit `search.CollectionName` schreiben.
Die hinzugefügte Zeile `+func B() int { return 1 }` muss in D1 stehen; der
Test `TestALocalOnlyCaseWithAProposalIsNotManual` findet es sonst sofort, und
dann wird der Diff aus `package.md` des Falls gelesen und die Konstante
angepasst.

- [ ] **Step 2: Rot sehen**

Run: `go test ./internal/brain/maintenance/ -run 'Proposal|Model|Asked|Cancelled|Note' -count=1`
Expected: FAIL; `TestALocalOnlyCaseWithAProposalIsNotManual` findet `Manual`
gesetzt, `TestACancelledPassWritesNoCase…` findet einen Fall.

- [ ] **Step 3: Implementieren**

Konstante neben `localOnlyNote`:

```go
	// proposalRefusedNote is `_PROPOSAL_REJECTED_NOTE`, spelt to the letter
	// for the reason localOnlyNote is. It says only what the three ways out
	// of Propose -- no answer, nothing readable, a refused claim -- have in
	// common, because this branch cannot tell them apart.
	proposalRefusedNote = "manual review: the local proposer returned no usable proposal (slice-6 spec §3)"
```

Neue Funktion vor `reviewRootOf`:

```go
// proposers is `_proposers` (reconcile.py:271-310): one proposer per
// local_only area, built once for the whole run. The settings are not read
// at all unless such an area exists: a vault without one never asks the
// model, and a mistyped [model] must not stop a run that would not use it.
// A misconfiguration stops the run before the first case is written.
func proposers(areas []config.Area, manifests map[string]*config.Manifest, stateDir string) (map[string]*model.Proposer, error) {
	var closed []string
	for _, area := range areas {
		if m := manifests[area.Scope]; m != nil && m.PrivacyMode == localOnlyMode && !slices.Contains(closed, area.Scope) {
			closed = append(closed, area.Scope)
		}
	}
	if len(closed) == 0 {
		return nil, nil
	}
	settings, err := config.ReadModelSettings(stateDir)
	if err != nil {
		return nil, err
	}
	found := map[string]*model.Proposer{}
	for _, scope := range closed {
		proposer, err := model.ProposerFor(settings, manifests[scope], "propose")
		if err != nil {
			return nil, err
		}
		found[scope] = proposer
	}
	return found, nil
}
```

In `ReconcileContext` nach `reviewRootOf`:

```go
	asking, err := proposers(areas, manifests, lookup.Primary)
	if err != nil {
		return Report{}, err
	}
```

`sourceCases` und `mergeCases` bekommen `ctx context.Context` als ersten und
`asking map[string]*model.Proposer` als letzten Parameter und reichen
`ctx` sowie `asking[area.Scope]` an `landCase` weiter (`landMerge` ebenso;
ein Map-Zugriff auf `nil` ergibt `nil`). In `landCase`:

- Signatur: `func landCase(ctx context.Context, area config.Area, …, evidence []string, proposer *model.Proposer) (Case, error)`; der Parameter `evidence` heißt schon so und verdeckt das Paket `evidence` — `landCase` braucht das Paket nicht, weil die Umwandlung in `evidenceSegments` liegt.
- Im Literal `landed` fallen `Manual` und `Note` samt Kommentar weg.
- **Reihenfolge.** Heute legt `landCase` erst das Verzeichnis an und räumt
  mit `supersede` den stehenden Fall weg (`:752-759`), dann rendert es das
  Paket. Ein Abbruch während der Frage ließe so ein leeres Fallverzeichnis
  und einen verlorenen alten Fall zurück. `RenderPackage` liest vom Fall nur
  `ID` und `Created` (`package.go:181-182`); darum wandern `segmentsOf`,
  `RenderPackage` und die Frage **vor** `os.MkdirAll` und `supersede`. Dafür
  braucht `segmentsOf` nur `page`, das ebenfalls nach vorn geht; `TargetHash`
  und `SupersededProposal` werden danach ins Literal gesetzt. Die Referenz
  fragt nach `supersede`; die Ausgabe ist dieselbe, nur ein Abbruch verhält
  sich anders — Eintrag in der Akte (Task 9).
- Nach `packageText := RenderPackage(landed, segments)`, noch vor dem
  Anlegen des Verzeichnisses:

```go
	// No mode check before asking: proposers hands one out only for closed
	// areas. Manual still reads the mode, because a closed area whose
	// proposal is missing -- no proposer, or a refused answer -- is a manual
	// case too.
	proposal, proposed := "", false
	if proposer != nil {
		proposal, proposed = proposer.Propose(ctx, packageText, evidenceSegments(segments))
		// A pass that ended while the model computed has spent no attempt:
		// writing the refused note now would keep this case from ever being
		// asked again.
		if err := ctx.Err(); err != nil {
			return Case{}, err
		}
	}
	landed.Manual = closed && !proposed
	landed.Note = noteFor(manifest, proposer != nil, proposed)
	if proposed {
		landed.PromptVersion = model.ProposeVersion
	}
```

- Nach `writeIfChanged(… packageName …)`:

```go
	if proposed {
		if err := writeIfChanged(filepath.Join(directory, proposalName), proposal); err != nil {
			return Case{}, err
		}
	}
```

Und:

```go
// evidenceSegments hands the package's segments to the checker in its own
// type; the two carry the same four fields.
func evidenceSegments(segments []Segment) []evidence.Segment {
	out := make([]evidence.Segment, len(segments))
	for i, s := range segments {
		out[i] = evidence.Segment(s)
	}
	return out
}

// noteFor is `_note_for` (reconcile.py:836-848): the reason a closed area's
// case carries no proposal, in the reference's two versions.
func noteFor(manifest *config.Manifest, hadProposer, proposed bool) string {
	if manifest.PrivacyMode != localOnlyMode || proposed {
		return ""
	}
	if hadProposer {
		return proposalRefusedNote
	}
	return localOnlyNote
}
```

Der Kommentar über `ReconcileContext` („so a pass that ends early has only
read: no case, no stamp“) stimmt danach nicht mehr: Er wird zu „asked after
every area's scan and once more after every question to the local model; a
pass that ends there leaves the cases landed before it and no stamp“.
Der Kopfkommentar `:20-31` verliert beide Punkte; an ihre Stelle kommt:
„The local model is asked for a proposal on every case of a `local_only`
area, source or merge, through the proposer `proposers` built for the run.“
Der Test `TestReconcileMarksALocalOnlyCaseManual` (`reconcile_test.go:499`)
bleibt als Fall „Modell aus“ stehen; seine Kommentare über „stage 3a asks no
model“ werden zu „with the model off, nobody is asked“.

`internal/serve/upkeep.go` reicht seinen Kontext schon hinein; zu ändern ist
dort nichts.

- [ ] **Step 4: Grün sehen, ganzes Paket und Aufrufer**

Run: `go test ./internal/brain/maintenance/ ./internal/cli/ ./internal/serve/ -count=1`
Expected: PASS. `TestCases3a` bleibt grün: kein Fall dort ist `local_only`.

- [ ] **Step 5: Commit**

```bash
git add internal/brain/maintenance
git commit -m "feat(reconcile): ask the local model for a proposal on a closed area's case"
```

---

### Task 6: `approve --reject` schiebt Quellen und Register vor

**Files:**
- Modify: `internal/brain/apply/reject.go` (ganzer Pfad)
- Modify: `internal/brain/apply/approve.go:134-138` (Aufruf)
- Modify: `internal/brain/apply/resolve.go:188-218` (`targetPath` teilen: `targetPlace` ohne die Prüfung „page is gone“)
- Modify: `internal/brain/apply/frontmatter.go` (neue Funktion `AdvanceSources`)
- Modify: `internal/brain/apply/frontmatter_test.go`
- Modify: `internal/brain/apply/reject_test.go:356-372` (Test umdrehen, neue Tests)
- Modify: `internal/cli/cases_3b_test.go:87` (erwartete Abweichungen)

**Interfaces:**
- Consumes: `approval.guardSources`, `approval.updates`, `approval.advanceRegisters`, `targetPath`, `readPage`, `place.write`, `place.staged`, `commit` (alle vorhanden)
- Produces: `func AdvanceSources(page string, updates []SourceUpdate) (string, bool, error)` — `false`, wenn nichts vorzuschieben war; `func (a approval) reject() (Result, error)`; `func targetPlace(r resolved, target string) (string, error)` — alle Prüfungen von `targetPath` außer der, dass die Seite da ist; `targetPath` ruft sie und prüft danach `isFile`

- [ ] **Step 1: Failing tests schreiben**

`frontmatter_test.go`:

```go
func TestAdvanceSourcesTouchesOnlyTheSources(t *testing.T) {
	page := "---\ntitle: T\nsources:\n- doc_id: 01DOC0\n  content_hash: sha256:aa\n  revision: 1\ngenerated:\n  at: '2026-01-01T00:00:00+00:00'\n---\n\nText\n"
	got, changed, err := AdvanceSources(page, []SourceUpdate{{DocID: "01DOC0", ContentHash: "sha256:bb", Revision: 1}})
	if err != nil || !changed {
		t.Fatal(changed, err)
	}
	if !strings.Contains(got, "content_hash: sha256:bb") || !strings.Contains(got, "revision: 2") ||
		!strings.Contains(got, "at: '2026-01-01T00:00:00+00:00'") || strings.Contains(got, "verified") || !strings.HasSuffix(got, "---\n\nText\n") {
		t.Fatalf("%q", got)
	}
}

func TestAdvanceSourcesLeavesAPageWithNothingToAdvance(t *testing.T) {
	for name, page := range map[string]string{
		"no frontmatter":  "Text\n",
		"no sources":      "---\ntitle: T\n---\nText\n",
		"other doc":       "---\nsources:\n- doc_id: 01OTHER\n  revision: 1\n---\nText\n",
	} {
		got, changed, err := AdvanceSources(page, []SourceUpdate{{DocID: "01DOC0", ContentHash: "h", Revision: 1}})
		if err != nil || changed || got != page {
			t.Errorf("%s: %q %v %v", name, got, changed, err)
		}
	}
}
```

`reject_test.go` — der alte Test wird so umgedreht:

```go
// Healed: a rejection advances the page's `sources[]` and the register to
// the state the case was formed over, so the next reconcile does not open it
// again. `generated` and `verified` stay: the page was neither regenerated
// nor confirmed.
func TestRejectAdvancesThePageSourcesAndTheRegister(t *testing.T) {
	j := newRejection(t)
	page := filepath.Join(j.r.wiki, "topics", "thema.md")
	writeFile(t, page, "---\nsources:\n  - doc_id: 01DOC0\n    content_hash: \"sha256:aa\"\n    revision: 1\n---\n\nText\n")
	register := filepath.Join(j.vault, registerName)
	writeFile(t, register, "doc_id\trelative\tcontent_hash\trevision\n01DOC0\tq.md\tsha256:aa\t1\n")
	noRepository(t)
	mustReject(t, j)
	if got := readFile(t, page); !strings.Contains(got, "revision: 2") || strings.Contains(got, "verified") {
		t.Fatalf("page %q", got)
	}
	if got := readFile(t, register); !strings.Contains(got, "01DOC0\tq.md\t") || !strings.HasSuffix(strings.TrimSpace(got), "\t2") {
		t.Fatalf("register %q", got)
	}
}

// Review Focus 5.
func TestRejectOnAPageWithoutFrontmatterStillAdvancesTheRegister(t *testing.T) {
	j := newRejection(t)
	page := filepath.Join(j.r.wiki, "topics", "thema.md")
	writeFile(t, page, "Text\n")
	register := filepath.Join(j.vault, registerName)
	writeFile(t, register, "doc_id\trelative\tcontent_hash\trevision\n01DOC0\tq.md\tsha256:aa\t1\n")
	noRepository(t)
	mustReject(t, j)
	if readFile(t, page) != "Text\n" || !strings.HasSuffix(strings.TrimSpace(readFile(t, register)), "\t2") {
		t.Fatal("page changed or register stood")
	}
	if isFile(filepath.Join(j.r.directory, "case.toml")) {
		t.Fatal("the case stayed")
	}
}

// Review Focus 6: the old path never read the page and closed the case; a
// page deleted or renamed since must not keep the case open for good.
func TestRejectWithTheTargetPageGoneStillClosesTheCase(t *testing.T) {
	j := newRejection(t)
	page := filepath.Join(j.r.wiki, "topics", "thema.md")
	_ = os.Remove(page)
	register := filepath.Join(j.vault, registerName)
	writeFile(t, register, "doc_id\trelative\tcontent_hash\trevision\n01DOC0\tq.md\tsha256:aa\t1\n")
	noRepository(t)
	mustReject(t, j)
	if isFile(page) || !strings.HasSuffix(strings.TrimSpace(readFile(t, register)), "\t2") || isFile(filepath.Join(j.r.directory, "case.toml")) {
		t.Fatal("a page appeared, the register stood, or the case stayed")
	}
}

func TestRejectHaltsOnASourceThatMovedAgain(t *testing.T) {
	j := newRejection(t)
	writeFile(t, filepath.Join(j.vault, "q.md"), "moved again\n")
	register := filepath.Join(j.vault, registerName)
	writeFile(t, register, "doc_id\trelative\tcontent_hash\trevision\n01DOC0\tq.md\tsha256:aa\t1\n")
	noRepository(t)
	_, err := j.run()
	var moved *SourceMoved
	if !errors.As(err, &moved) {
		t.Fatalf("%v", err)
	}
	if !isFile(filepath.Join(j.r.directory, "case.toml")) {
		t.Fatal("the case went although nothing was decided")
	}
}
```

`newRejection`, `mustReject`, `writeFile`, `readFile`, `noRepository` und
`registerName` gibt es in `reject_test.go`; `j.run()` ist die Form, die
`mustReject` ohne `Fatal` ruft. Kennt der Fixture-Fall keine Quelle `q.md`
mit Hash `sha256:aa`, die Werte aus `newRejection` übernehmen: Der Fall
muss die Quelle `01DOC0` mit dem Hash tragen, den das Register trägt, damit
`guardSources` sie findet.

- [ ] **Step 2: Rot sehen**

Run: `go test ./internal/brain/apply/ -run 'Reject|AdvanceSources' -count=1`
Expected: FAIL (`undefined: AdvanceSources`; die Seite bleibt stehen).

- [ ] **Step 3: Implementieren**

`frontmatter.go`, nach `AdvanceFrontmatter`:

```go
// AdvanceSources is the half of AdvanceFrontmatter a rejection needs: the
// `sources[]` entries the updates name get their hash and revision+1, and
// nothing else changes -- no `generated`, no `verified`, since a rejected
// page was neither regenerated nor confirmed. A page without frontmatter or
// without a matching entry is returned as it is, with false: rewriting it
// would only reformat YAML nobody asked to touch.
func AdvanceSources(page string, updates []SourceUpdate) (string, bool, error) {
	block := advanceBlock().FindStringIndex(page)
	match := documentBlock().FindStringSubmatch(page)
	if block == nil || match == nil {
		return page, false, nil
	}
	meta, err := loadFrontmatter(match[1])
	if err != nil {
		return "", false, err
	}
	fresh := make(map[string]SourceUpdate, len(updates))
	for _, update := range updates {
		fresh[update.DocID] = update
	}
	changed := false
	if entries := meta.get("sources"); entries != nil && entries.kind == pyList {
		for _, entry := range entries.items {
			if entry.kind != pyDict {
				continue
			}
			id, ok := pythonStr(entry.get("doc_id"))
			state, found := fresh[id]
			if ok && found {
				entry.set("content_hash", newStr(state.ContentHash))
				entry.set("revision", newInt(state.Revision+1))
				changed = true
			}
		}
	}
	if !changed {
		return page, false, nil
	}
	return "---\n" + dumpYAML(meta) + "---\n" + page[block[1]:], true, nil
}
```

Die Schleife über `sources` teilen sich danach beide Funktionen:
`AdvanceFrontmatter` ruft eine gemeinsame `advanceSourceEntries(meta,
updates) bool`, damit sie nicht zweimal dasteht.

`reject.go` — `reject` wird eine Methode von `approval`:

```go
// reject is `_reject` (apply.py:698-727), healed: a human said no to the
// proposal, and the sources the case was formed over are acknowledged all
// the same -- the page's `sources[]` and the register move on, so the next
// reconcile does not open the same case again. A source that moved once more
// since the case was formed halts it as it halts an approval: that newer
// state was never under review.
//
// The audit block is written even so -- a rejected proposal stays
// traceable -- and the case directory is removed in the same commit.
func (a approval) reject() (Result, error) {
	// targetPlace and not targetPath: a page deleted or renamed since the
	// case was formed must not keep the rejection from closing it, as the
	// unhealed path, which never read the page, did not either.
	page, err := targetPlace(a.r, a.c.Target)
	if err != nil {
		return Result{}, err
	}
	sources, err := a.guardSources()
	if err != nil {
		return Result{}, err
	}
	claims, err := claimHeadings(a.proposal())
	if err != nil {
		return Result{}, err
	}
	advanced, changed := "", false
	if isFile(page) {
		current, err := readPage(page)
		if err != nil {
			return Result{}, err
		}
		advanced, changed, err = AdvanceSources(current, a.updates())
		if err != nil {
			return Result{}, &ApplyError{Msg: a.c.Target + ": " + err.Error()}
		}
	}
	var add []string
	if changed {
		if err := a.p.write(page, advanced); err != nil {
			return Result{}, err
		}
		staged, err := a.p.staged(page)
		if err != nil {
			return Result{}, err
		}
		add = append(add, staged...)
	}
	registers, err := a.advanceRegisters(sources)
	if err != nil {
		return Result{}, err
	}
	audit := filepath.Join(a.r.wiki, "audit.md")
	block := RenderAudit(AuditEntry{
		Now: a.o.Now, Target: a.c.Target, CaseID: a.c.ID, Claims: claims,
		Decided: "abgelehnt durch " + a.o.Reviewer, Changed: "nichts",
	})
	if err := a.p.appendProtocol(audit, block); err != nil {
		return Result{}, err
	}
	if err := a.p.remove(a.r.directory); err != nil {
		return Result{}, err
	}
	staged, err := a.p.staged(audit)
	if err != nil {
		return Result{}, err
	}
	add = append(add, staged...)
	sha, warning := commit(a.p.anchor, a.c.ID, "decision recorded",
		"Reject the proposed change to "+Safe(a.c.Target),
		append(add, registers...), []string{a.p.relative(a.r.directory)}, a.o.Scratch)
	return Result{Case: a.c, Decision: "reject", Written: false, Commit: sha, Warning: warning}, nil
}
```

`approve.go:134-138`:

```go
	a := approval{r: r, p: p, c: c, areas: areas, o: o}
	var result Result
	if o.Decision == "reject" {
		result, err = a.reject()
	} else {
		result, err = a.run()
	}
```

`targetPlace` ist der Körper von `targetPath` bis vor `if !isFile(page)`;
`targetPath` wird `targetPlace` plus diese Prüfung. Ob `resolvePath`
(`guard.ResolvePath`) einen fehlenden Pfad verträgt, sagt der Test von
Review Focus 6; scheitert es dort, löst `targetPlace` das Elternverzeichnis
auf und hängt den Namen an.

`Changed: "nichts"` im Audit bleibt: Am Text der Seite hat sich nichts
geändert; die Quellen stehen im Commit. Ob `readPage` für eine Seite ohne
Frontmatter scheitert, sagt der Test von Review Focus 5; tut es das, liest
`reject` mit `os.ReadFile` und faltet selbst nach LF, wie `readPage` es tut.

- [ ] **Step 4: Grün sehen**

Run: `go test ./internal/brain/apply/ -count=1 -cover`
Expected: PASS, 100.0 %.

- [ ] **Step 5: Den aufgezeichneten Fall abgleichen**

Run: `go test ./internal/cli/ -run 'TestCases3b/approve/reject' -count=1 -v`
Expected: FAIL mit Zeilen zu Seite und Register in `world_after` und zu
`git.after`. Jede Zeile lesen: Sie darf nur die Zielseite, ein
`_identities.tsv` oder den Commit betreffen. Diese Zeilen in
`expected3b["approve/reject"].differ` neben `scratchIndex` eintragen,
`why: "Scratch-Index im Fallsatz; Heilung #1 (--reject schiebt vor)"`.
Eine andere Zeile ist ein Fehler dieses Tasks, kein Eintrag.

Run: `go test ./internal/cli/ -run TestCases3b -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/brain/apply internal/cli/cases_3b_test.go
git commit -m "fix(approve): advance the page's sources and the register on a rejection"
```

---

### Task 7: Eine Sperre je Bereich für `reindex` und `approve`

**Files:**
- Create: `internal/config/arealock.go`
- Create: `internal/config/arealock_test.go`
- Modify: `internal/brain/index/reindex.go:146-172` (`indexArea`)
- Modify: `internal/brain/index/reindex_test.go` (oder die interne Testdatei, die `readDocFn` schon ersetzt)
- Modify: `internal/brain/apply/approve.go:405-456` (Kommentar und `advanceRegisters`)
- Modify: `internal/brain/apply/stock.go:45-51` (Kommentar)
- Modify: `internal/brain/apply/approve_test.go`
- Modify: `internal/cli/cases_3a_test.go`, `internal/cli/cases_3b_test.go` (Sperrdatei in den Erwartungen)

**Interfaces:**
- Produces: `func AreaLockPath(area Area, stateDir string) string`; `func LockArea(area Area, stateDir string) (func() error, error)` — blockiert, legt `<state>/areas` an, gibt die Freigabe zurück

- [ ] **Step 1: Failing tests schreiben**

`arealock_test.go`:

```go
package config

import (
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/lock"
)

func TestTheAreaLockLiesBesideTheSwappedDirectory(t *testing.T) {
	state := t.TempDir()
	area := Area{Scope: "project/a", ReadOnly: true}
	if got, want := AreaLockPath(area, state), ManifestDir(area, state)+".lock"; got != want {
		t.Fatalf("%s, want %s", got, want)
	}
	// A writable area has no directory there and gets the lock all the same.
	if got := AreaLockPath(Area{Scope: "project/a", Path: "/repo"}, state); got != filepath.Join(state, "areas", "project-a.lock") {
		t.Fatal(got)
	}
}

func TestLockAreaHoldsUntilReleased(t *testing.T) {
	state := t.TempDir()
	area := Area{Scope: "project/a"}
	release, err := LockArea(area, state)
	if err != nil {
		t.Fatal(err)
	}
	if _, free, _ := lock.TryAcquire(AreaLockPath(area, state)); free {
		t.Fatal("the lock was free while held")
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	handle, free, err := lock.TryAcquire(AreaLockPath(area, state))
	if err != nil || !free {
		t.Fatal(free, err)
	}
	_ = handle.Release()
}
```

`TryAcquire` gibt bei einer Sperre desselben Prozesses unter Windows
`free = false` (LockFileEx ist je Handle); gilt das auf POSIX nicht
(`flock` je Datei-Beschreibung, `fcntl` je Prozess), hält
`internal/lock/lock_test.go` fest, was gilt, und der Test hier folgt ihm.

Für `reindex`: über die Naht `readDocFn` prüfen, dass die Sperre während
`collect` gehalten wird:

```go
func TestReindexHoldsTheAreaLockWhileItReadsAndWritesTheRegister(t *testing.T) {
	// World as the other reindex tests of this file build it: one writable
	// area with one note.
	w := newIndexWorld(t)
	held := false
	restore := readDocFn
	readDocFn = func(path, root string) (*Document, error) {
		if _, free, _ := lock.TryAcquire(config.AreaLockPath(w.area, w.state)); !free {
			held = true
		}
		return restore(path, root)
	}
	t.Cleanup(func() { readDocFn = restore })
	w.reindex(t)
	if !held {
		t.Fatal("the area lock was free while the stock was read")
	}
}
```

`newIndexWorld` und `w.reindex` stehen für die Welthilfen, die
`reindex_test.go` heute benutzt; ihre echten Namen dort ablesen und
einsetzen. Gibt `TryAcquire` in `true`-Fall einen Handle zurück, ihn im Test
freigeben.

Für `approve`: über die Naht `advanceRegister` (`approve.go:32-34`) dasselbe —
in der ersetzten Funktion muss `TryAcquire(config.AreaLockPath(area, state))`
`free = false` liefern.

- [ ] **Step 2: Rot sehen**

Run: `go test ./internal/config/ ./internal/brain/index/ ./internal/brain/apply/ -run 'Lock' -count=1`
Expected: FAIL (`undefined: AreaLockPath`).

- [ ] **Step 3: Implementieren**

`arealock.go`:

```go
package config

import (
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/lock"
)

// AreaLockPath is the lock `reindex` and `approve` share for one area. It
// lies beside `<state>/areas/<scope>`, never inside it: lock.ReplaceDir
// swaps that directory whole, and an open file in it would hold the rename
// on Windows. A writable area has no such directory and is locked there all
// the same, since its register is read and written by both commands too.
func AreaLockPath(area Area, stateDir string) string {
	return filepath.Join(stateDir, "areas", flat(area.Scope)+".lock")
}

// LockArea blocks until the area's lock is free, takes it and hands back
// its release.
func LockArea(area Area, stateDir string) (func() error, error) {
	path := AreaLockPath(area, stateDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	handle, err := lock.Acquire(path)
	if err != nil {
		return nil, err
	}
	return handle.Release, nil
}
```

`indexArea`, nach der Prüfung `os.Stat(area.Path)`:

```go
	// Held from the register's reading in collect to its writing in publish:
	// an approve advancing it in between would otherwise lose its row.
	release, err := config.LockArea(area, stateDir)
	if err != nil {
		return indexedArea{}, false, err
	}
	defer release()
```

`advanceRegisters`, in der Schleife `for _, register := range order` den
Körper in eine Hilfsfunktion ziehen, die die Sperre des Bereichs hält:

```go
func (a approval) advanceRegisterLocked(register string, rows map[string]identity.Identity) ([]string, error) {
	for _, area := range a.areas {
		if registerWrite(area, a.o.Lookup) != register {
			continue
		}
		release, err := config.LockArea(area, a.o.Lookup.Primary)
		if err != nil {
			return nil, err
		}
		defer release()
		if err := moveStock(area, a.o.Lookup); err != nil {
			return nil, err
		}
	}
	text, err := advanceRegister(register, rows)
	if err != nil {
		return nil, err
	}
	if err := a.p.writeScaffold(register, text); err != nil {
		return nil, err
	}
	return a.p.staged(register)
}
```

Die Kommentare „No lock is taken, as Python takes none“ (`approve.go:412-414`)
und „No lock is shared with `index`“ (`stock.go:45-51`) werden auf die
Sperre umgeschrieben; in `stock.go` bleibt der einfache `os.Rename` samt
Begründung, weil ein alter Checkout ohne Sperre noch laufen kann.

- [ ] **Step 4: Grün sehen**

Run: `go test ./internal/config/ ./internal/brain/... -count=1`
Expected: PASS.

- [ ] **Step 5: Die Sperrdatei in den Fallsätzen**

Run: `go test ./internal/cli/ -run 'TestCases3a|TestCases3b' -count=1`
Expected: FAIL mit `unexpected extra file in actual: areas/<scope>.lock` in
jedem Fall, der `reindex` oder ein schreibendes `approve` abspielt. Eine
Konstante neben `registryLock` anlegen:

```go
// areaLock is the lock reindex and approve share per area, which the
// reference does not take (Heilung #4).
func areaLock(scope string) string {
	return "unexpected extra file in actual: areas/" + scope + ".lock"
}
```

und sie in jede betroffene Erwartung eintragen, mit `why` ergänzt um
„Sperre je Bereich“. Andere neue Zeilen sind ein Fehler dieses Tasks.

Run: `go test ./internal/cli/ -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/config/arealock.go internal/config/arealock_test.go internal/brain/index internal/brain/apply internal/cli/cases_3a_test.go internal/cli/cases_3b_test.go
git commit -m "fix(index): share one lock per area between reindex and approve"
```

---

### Task 8: `loomux dev fake-ollama`

**Files:**
- Create: `internal/dev/fakeollama/fakeollama.go`
- Create: `internal/dev/fakeollama/fakeollama_test.go`
- Modify: `internal/cli/dev.go:43-52` (Eintrag `fake-ollama`), dazu `devFakeOllama`
- Modify: `internal/cli/dev_test.go`

**Interfaces:**
- Produces:
  - `type Fixture struct { Status int \`json:"status"\`; Response *string \`json:"response"\`; Body string \`json:"body"\` }`
  - `func Load(path string) (*Fixture, error)`
  - `func (f *Fixture) Handler(log io.Writer) http.Handler` — antwortet `Status` (Vorgabe 200) mit `{"response": …}`, oder mit `Body` roh, wenn `Response` fehlt; schreibt je Anfrage eine Zeile nach `log`
  - `func Serve(ctx context.Context, addr string, f *Fixture, log io.Writer) error`
  - CLI: `loomux dev fake-ollama --fixture <json> [--addr 127.0.0.1:11435] [--log <datei>]`, Log ohne `--log` auf stderr; läuft bis zum Ende des Prozesses

Das Log liegt nie in der Welt: `record-case` stellt die Welt in einem
Zufallsverzeichnis (`recordcase.go:50`), das die Attrappe nicht kennt, und
eine Datei neben der Quellwelt fände `world_after` nicht. Gezählt wird
darum außerhalb — im Aufzeichnungsskript und im Abspieltest (Task 9). Die
Zeile hat die Form
`POST /api/generate model=<name> temperature=<%g> num_ctx=<n> stream=<bool> think=<bool>`,
ohne Prompt.

- [ ] **Step 1: Failing tests schreiben**

```go
package fakeollama

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, text string) *Fixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ollama-fixture.json")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func post(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(body)))
	return w
}

func TestTheFakeAnswersAndLogsWithoutThePrompt(t *testing.T) {
	var log bytes.Buffer
	w := post(t, fixture(t, `{"response":"hallo"}`).Handler(&log), `{"model":"m","prompt":"geheim","stream":false,"think":false,"options":{"temperature":0,"num_ctx":8192}}`)
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"response":"hallo"}` {
		t.Fatalf("%d %q", w.Code, w.Body.String())
	}
	if log.String() != "POST /api/generate model=m temperature=0 num_ctx=8192 stream=false think=false\n" {
		t.Fatalf("%q", log.String())
	}
}

func TestTheFakeCanAnswerNonsense(t *testing.T) {
	w := post(t, fixture(t, `{"status":500,"body":"<html>"}`).Handler(&bytes.Buffer{}), `{}`)
	if w.Code != 500 || w.Body.String() != "<html>" {
		t.Fatalf("%d %q", w.Code, w.Body.String())
	}
}

func TestLoadRefusesABrokenFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.json")
	_ = os.WriteFile(path, []byte("{"), 0o644)
	if _, err := Load(path); err == nil {
		t.Fatal("a broken fixture loaded")
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("a missing fixture loaded")
	}
}
```

Dazu ein Test für `Serve` mit `127.0.0.1:0` und einem Kontext, der nach
einer Anfrage endet, und ein CLI-Test in `dev_test.go`, dass `dev
fake-ollama` ohne `--fixture` mit Exit 2 endet.

- [ ] **Step 2: Rot sehen**

Run: `go test ./internal/dev/fakeollama/ ./internal/cli/ -run 'Fake|fake' -count=1`
Expected: FAIL.

- [ ] **Step 3: Implementieren**

```go
// Package fakeollama stands in for Ollama where the Python reference and
// loomux are recorded and replayed against the same answers. It runs as a
// process of its own, because the reference speaks HTTP to it, and in the
// replay test as a handler on the same fixed port.
package fakeollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
)

// Fixture is the one answer every request gets.
type Fixture struct {
	Status   int     `json:"status"`
	Response *string `json:"response"`
	Body     string  `json:"body"`
}

// Load reads a fixture.
func Load(path string) (*Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &Fixture{}
	if err := json.Unmarshal(data, f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if f.Status == 0 {
		f.Status = http.StatusOK
	}
	return f, nil
}

type request struct {
	Model   string `json:"model"`
	Stream  bool   `json:"stream"`
	Think   bool   `json:"think"`
	Options struct {
		Temperature float64 `json:"temperature"`
		NumCtx      int     `json:"num_ctx"`
	} `json:"options"`
}

// Handler answers every request from the fixture and writes one line per
// request to log, prompt left out: the prompt carries the case id and the
// day, and the lines must read alike in the recording and the replay.
func (f *Fixture) Handler(log io.Writer) http.Handler {
	var mu sync.Mutex
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got request
		_ = json.NewDecoder(r.Body).Decode(&got)
		mu.Lock()
		fmt.Fprintf(log, "%s %s model=%s temperature=%s num_ctx=%d stream=%t think=%t\n",
			r.Method, r.URL.Path, got.Model, strconv.FormatFloat(got.Options.Temperature, 'g', -1, 64),
			got.Options.NumCtx, got.Stream, got.Think)
		mu.Unlock()
		w.WriteHeader(f.Status)
		if f.Response != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"response": *f.Response})
			return
		}
		_, _ = w.Write([]byte(f.Body))
	})
}

// Serve listens on addr until ctx ends.
func Serve(ctx context.Context, addr string, f *Fixture, log io.Writer) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: f.Handler(log)}
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
```

`devFakeOllama` in `cli/dev.go` nach dem Muster der übrigen `dev`-Befehle:
`flag.NewFlagSet("fake-ollama", flag.ContinueOnError)`, `--fixture`
(Pflicht, sonst Exit 2), `--addr` (Vorgabe `127.0.0.1:11435`), `--log`
(Datei zum Anhängen; ohne es stderr), dann `fakeollama.Load` und
`fakeollama.Serve` unter `signal.NotifyContext(context.Background(),
os.Interrupt)`; ein Fehler geht mit Exit 1 auf stderr.

- [ ] **Step 4: Grün sehen**

Run: `go test ./internal/dev/fakeollama/ ./internal/cli/ -count=1 -cover`
Expected: PASS, 100.0 % für `fakeollama`.

- [ ] **Step 5: Commit**

```bash
git add internal/dev/fakeollama internal/cli/dev.go internal/cli/dev_test.go
git commit -m "feat(dev): serve a fake Ollama for recording and replaying the local model"
```

---

### Task 9: Parität — sieben Fälle gegen die Referenz

**Files:**
- Create: `testdata/cases/4c1-worlds/<welt>/…` (vier Welten, siehe unten)
- Create: `docs/.superpowers/parity/stufe-4c-1-orakel/record.sh`, `record_all.sh`
- Create: `testdata/cases/4c1-map.toml`
- Create: `testdata/cases/4c1-source/reconcile/<name>/…` (aufgezeichnet), `testdata/cases/4c1/reconcile/<name>/…` (importiert)
- Create: `internal/cli/cases_4c1_test.go`
- Create: `docs/.superpowers/parity/stufe-4c-1.md`
- Modify: `.gitattributes` (`testdata/cases/4c1-source/** -text`, `testdata/cases/4c1/** -text`)
- Modify: `docs/.superpowers/parity/stufe-3a.md:53,72`, `docs/.superpowers/parity/stufe-3b.md` (Abschnitt „Geerbt“, Zeilen `:297,304`)

**Interfaces:**
- Consumes: `fakeollama.Load`, `(*Fixture).Handler` (Task 8); `dev record-case`, `dev import-cases` (vorhanden)

- [ ] **Step 1: Welten bauen**

Grundlage ist `testdata/cases/3a-worlds/vault-changed` (die Welt von
`reconcile/changed-source`): eine Kopie je Welt, in deren
`repo-a/.loomux/config.toml` `[privacy] mode = "local_only"` steht, außer in
`open-area`. In der Weltwurzel, die für beide Seiten der Zustand ist (B7),
liegen je Fall `config.toml` und `ollama-fixture.json`:

| Fall | Welt | `config.toml` | Fixture | Erwartet |
|---|---|---|---|---|
| `reconcile/proposal-kept` | closed | `[model]` `enabled = true`, `endpoint = "http://127.0.0.1:11435"` | Antwort, die die hinzugefügte Zeile aus D1 wörtlich zitiert | `proposal.md`, `prompt_version`, kein `manual`; eine Zeile im Log |
| `reconcile/proposal-invented` | closed | wie oben | Zitat `+erfunden` | `manual`, zweite Notiz; eine Zeile im Log |
| `reconcile/model-unreachable` | closed | `endpoint = "http://127.0.0.1:11436"` (niemand lauscht) | — | `manual`, zweite Notiz |
| `reconcile/model-off` | closed | keine | — | `manual`, erste Notiz, kein Log |
| `reconcile/open-area-not-asked` | open-area | wie `proposal-kept` | wie `proposal-kept` | kein Fall-Merkmal, kein Log |
| `reconcile/endpoint-off-loopback` | closed | `endpoint = "http://192.0.2.1:11434"` | — | Exit 1, kein Fall |
| `reconcile/broken-model-block` | closed | `model = 5` | — | Exit 1, kein Fall |

Die Zeile für D1 aus `package.md` einer Probeaufzeichnung von
`model-off` ablesen, nicht raten.

- [ ] **Step 2: Aufzeichnen**

`record.sh` ist `stufe-3a-orakel/record.sh` mit drei Änderungen: `WT` zeigt
auf diesen Checkout, die Welten kommen aus `4c1-worlds`, und um den Aufruf
von `record-case` läuft die Attrappe im Hintergrund, ihr Log außerhalb jeder
Welt. Die Fixture liest sie aus der Quellwelt; sie ändert sich nicht, darum
ist es gleich, dass `record-case` eine Kopie stellt:

```bash
calls="$S/ollama-calls.log"
: > "$calls"
fixture="$WT/testdata/cases/4c1-worlds/$world/ollama-fixture.json"
fake=""
if [ -f "$fixture" ]; then
  "$S/bin/loomux.exe" dev fake-ollama --fixture "$fixture" --addr 127.0.0.1:11435 --log "$calls" &
  fake=$!
  trap '[ -n "$fake" ] && kill $fake 2>/dev/null' EXIT
fi
```

und in `--notes` hinter den Text: `; ollama calls: $(wc -l < "$calls")` —
die Notiz entsteht erst nach dem Lauf, also ruft `record.sh` `record-case`
mit einer Notiz, deren Zahl es danach mit `sed -i` in `$out/notes.md`
einsetzt. Die Zahl je Fall übernimmt der Abspieltest (Step 4).

`record_all.sh` ruft `record.sh` für die sieben Fälle der Tabelle. Danach:

Run: `bash docs/.superpowers/parity/stufe-4c-1-orakel/record_all.sh`
Expected: sieben Verzeichnisse unter `testdata/cases/4c1-source/reconcile/`,
Exit-Codes wie in der Tabelle; `ollama calls: 1` bei `proposal-kept` und
`proposal-invented`, sonst `0`.

- [ ] **Step 3: Importieren**

`4c1-map.toml` ist die Zeile `reconcile` aus `3a-map.toml` samt
`manifests = "verbatim"`.

Run: `bin/loomux.exe dev import-cases --map testdata/cases/4c1-map.toml --from testdata/cases/4c1-source --to testdata/cases/4c1`
Expected: sieben Fälle unter `testdata/cases/4c1/reconcile/`.

- [ ] **Step 4: Abspieltest schreiben und laufen lassen**

`cases_4c1_test.go` nach `cases_3a_test.go`, mit `wantCases4c1 = 7`, der
Zahl der Aufrufe je Fall aus den Notizen der Aufzeichnung

```go
// wantOllamaCalls4c1 are the requests the reference sent in each recording
// (notes.md, "ollama calls"); a case not named sent none.
var wantOllamaCalls4c1 = map[string]int{
	"reconcile/proposal-kept":     1,
	"reconcile/proposal-invented": 1,
}
```

und diesem Zusatz um das Abspielen jedes Falls, dessen Welt eine
`ollama-fixture.json` trägt:

```go
// serveFakeOllama puts the fake on the fixed port the world's config.toml
// names and hands back the lines it logged.
func serveFakeOllama(t *testing.T, world string) *bytes.Buffer {
	t.Helper()
	var calls bytes.Buffer
	path := filepath.Join(world, "ollama-fixture.json")
	if _, err := os.Stat(path); err != nil {
		return &calls
	}
	fixture, err := fakeollama.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:11435")
	if err != nil {
		t.Fatalf("the fixed port of the fake Ollama is taken: %v", err)
	}
	server := &http.Server{Handler: fixture.Handler(&calls)}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return &calls
}
```

Nach dem Abspielen: `strings.Count(calls.String(), "\n")` gegen
`wantOllamaCalls4c1[name]`. Die Fälle laufen nacheinander, nie mit
`t.Parallel()`: Alle teilen den Port, und der Server eines Falls muss
geschlossen sein, bevor der nächste lauscht — `t.Cleanup` im Untertest des
Falls, nicht im äußeren Test.

Run: `go test ./internal/cli/ -run TestCases4c1 -count=1 -v`
Expected: PASS, bis auf erwartete Abweichungen. Jede Abweichung wird gelesen
und in die Akte eingetragen; eine Abweichung außerhalb von Zeitstempel,
Sperrdatei (Task 7) und Meldungstext auf stderr ist ein Fehler aus Task 5
und wird dort behoben.

- [ ] **Step 5: Akte schreiben**

`docs/.superpowers/parity/stufe-4c-1.md` nach dem Muster von `stufe-3b.md`:
Referenz und Tag, die sieben Fälle mit Ergebnis, und die Abweichungsliste:

| Abweichung | Art | Begründung |
|---|---|---|
| `approve --reject` schiebt Quellen und Register vor | freigegeben 2026-09-25 | Heilung #1, Spec „Abweichungen beim Planen von 4c“ |
| Sperre `areas/<scope>.lock` für `reindex` und `approve` | freigegeben 2026-09-25 | Heilung #4; die Datei bleibt liegen wie `registry.lock` |
| Meldungen zu `[model]` in der Form von loomux | Meldungstext | stderr wird nicht verglichen (wie 3b) |
| `temperature` im JSON als `0` statt `0.0` | Format | Ollama liest beides als Zahl |
| Gefragt wird vor dem Anlegen des Fallverzeichnisses, nicht danach | Reihenfolge | Ein abgebrochener Lauf (Ende von `serve`) lässt den stehenden Fall stehen und legt kein leeres Verzeichnis an; die Ausgabe eines vollständigen Laufs ist gleich |

Dazu, wie die Aufrufe gezählt werden (Step 2 und 4). In `stufe-3a.md` wird die Zeile
„`local_only`-Bereich in `reconcile`“ (`:53`) und die zur zweiten Notiz
(`:72`) auf „fällt weg mit 4c-1“ gesetzt; in `stufe-3b.md` bekommen #1
(„Geerbt“) und die Zeilen `:297` und `:304` den Vermerk „geheilt in 4c-1“.

- [ ] **Step 6: Commit**

```bash
git add .gitattributes testdata/cases/4c1-worlds testdata/cases/4c1-source testdata/cases/4c1 testdata/cases/4c1-map.toml internal/cli/cases_4c1_test.go docs/.superpowers/parity
git commit -m "test(reconcile): record the local model's cases against the reference"
```

---

### Task 10: Selbstnutzung, Messung, Mutationen, Doku

**Files:**
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`
- Modify: `docs/en/migration.md`, `docs/de/migration.md` (4c-1 ✅, Fähigkeit „Local Model“)
- Modify: `README.md`, `README.de.md` (`[model]`, `config --global`)
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` (4c-1 ✅, „Offen nach 3b“ #1 und #4 geheilt)
- Modify: `docs/.superpowers/parity/stufe-4c-1.md` (Selbstnutzung, Mutanten)

- [ ] **Step 1: Selbstnutzung (Schritt des Menschen)**

**Befund vom 2026-09-25:** `project/obsidian-ai` ist `local_only` und
schreibgeschützt; sein Wiki liegt laut Registry unter
`brain-knowledge/91 Projekte/obsidian-ai`, und **keine Seite dort trägt
`sources:`**. Ohne eine Seite, die eine Quelle zitiert, öffnet `reconcile`
dort keinen Fall. **Entschieden am 2026-09-25:** Der Mensch legt dort eine
echte Wikiseite an, deren `sources:` eine Notiz unter `IAM-Projects/`
zitiert, lässt `reindex` laufen und ändert danach die Notiz. Quelle und Seite
ändert nur der Mensch.

**Abnahme nach der Referenz** (Scheibe-6-Spec §7 und §9 in `ultra-brain`,
offen als Task 10 in `OFFENE_AUFGABEN.md`, Abschnitt 3): `local_only`
erzeugt einen Vorschlag **ohne ausgehenden Verkehr**; Gegenprobe mit
abgeschaltetem Modell: manueller Fall, ebenfalls kein Verkehr; Zeit je
Vorschlag unter zwei Sekunden. Die Referenz wurde am 2026-09-11 ohne
Paketmitschnitt abgenommen, weil `pktmon` Adminrechte braucht; der Mitschnitt
ist dort als nachholbar vermerkt. Hier holt ihn der Mensch in einer
Admin-Shell nach:

```powershell
pktmon filter remove
pktmon start --capture --pkt-size 0 --file-name "$env:TEMP\4c1.etl"
# in einer zweiten Shell, ohne Adminrechte: bin\loomux.exe reconcile
pktmon stop
pktmon etl2txt "$env:TEMP\4c1.etl" --out "$env:TEMP\4c1.txt"
```

Ausgewertet wird `4c1.txt` auf Pakete, deren Quelle oder Ziel nicht
`127.0.0.1`/`::1` ist, während des Laufs; derselbe Mitschnitt noch einmal
mit `enabled = false`. Die Evidenzquote (120/120 über vier Pakete bei
`temperature: 0`) misst das Modell mit Prompt `vorschlag-v4`, nicht den
Client; der Go-Client sendet dieselbe Nutzlast (Task 3 hält sie fest), darum
wird sie nicht neu gemessen — die Akte sagt es. Ergebnis, Befehle und
Mitschnittauszug kommen in `parity/stufe-4c-1.md`, Abschnitt
„Datenschutznachweis“.

Der Mensch startet Ollama (`ollama serve`) und schaltet das Modell global
ein; ein Agent darf `config` nur mit `--propose`:

```bash
bin/loomux.exe config set model.enabled true --global --propose
```

Der Mensch wendet den Vorschlag mit `loomux config apply <id> --global` an.
Danach eine Quelle in `project/obsidian-ai` (dem `local_only`-Bereich dieses
Rechners) ändern, die eine Wikiseite zitiert, und:

Run: `bin/loomux.exe reconcile`
Expected: ein Fall mit `prompt_version = "vorschlag-v4"` und `proposal.md`,
oder einer mit der zweiten Notiz. Beides ist Selbstnutzung; das Ergebnis,
die Modellversion (`ollama list`) und die Dauer kommen in die Akte. Den Fall
danach mit `loomux approve` entscheiden, als Mensch.

- [ ] **Step 2: Messen**

Die Zeit je Vorschlag (Kriterium der Referenz: unter zwei Sekunden, dort
gemessen 563 bis 1 314 ms) liest sich aus derselben Messung ab: warmer Lauf
mit Modell minus warmer Lauf ohne. `reconcile` mit Modell gegen ohne, je
Fall, kalt und warm (Spec „Messen“):
zweimal dieselbe Welt mit einem geänderten Quelltext, einmal mit
`enabled = false`, einmal mit `true`, je 1 kalter und 5 warme Läufe über
`bin/loomux.exe reconcile`. Eintrag in beide `benchmarks.md` mit Datum,
Uhrzeit, Modell und Ollama-Version, oben (neueste zuerst).

- [ ] **Step 3: Mutationen**

Run: `bin/loomux.exe dev mutants ./internal/brain/model ./internal/config`
dazu `./internal/brain/maintenance` auf `reconcile.go` und
`./internal/brain/apply` auf `reject.go` und `frontmatter.go`, wie die Akte
von 3b es für ihre Dateien tat.
Expected: Überlebende gelesen; jeder bekommt in der Akte einen Test oder
eine Begründung.

- [ ] **Step 4: Doku**

- `README.md` / `README.de.md`: ein Absatz zum lokalen Modell — `[model]` in
  `%LOCALAPPDATA%\loomux\config.toml` über `loomux config --global`, je
  Bereich nur `enabled` und `roles`, nur für `local_only`-Bereiche, nur
  Loopback.
- `migration.md` (en/de): 4c-1 auf ✅ mit Datum; die Fähigkeit „Local Model“
  auf „Implemented (stage 4c-1)“ für `propose`, `describe`/`place` bleiben
  bei 4d; 4d „hängt ab von 4c-1 ✅“.
- Fusions-Spec: Zeile 4c-1 auf ✅, „Offen nach 3b“ #1 und #4 als geheilt.

- [ ] **Step 5: Tor und Commit**

Run: `sh ci/gate.sh`
Expected: alle Lanes `ok`.

```bash
git add docs README.md README.de.md
git commit -m "docs: record the local model's self-use, measurements and state"
```

- [ ] **Step 6: Pull Request vorbereiten**

Skill `release-pr`: Commits nach Thema gruppieren (Korrekturen desselben
Zweigs in ihren Ursprungscommit falten; die zwei `fix`-Commits für #1 und #4
bleiben eigene, weil die Fehler vor dem Zweig auf `master` bestanden),
Label mindestens `release:minor` (`feat` und `fix`), Zeile
`Release: minor — …` und ein `## Changelog`-Block mit `### Added` (lokales
Modell, `config --global` für `[model]`) und `### Fixed` (`--reject`,
Sperre). `loomux dev release parse-body` prüft; den Push-Befehl nennt der
Skill, pushen tut der Mensch.

---

## Selbstprüfung (2026-09-25)

- **Spec-Abdeckung:** Client und Wächter → T3; Tor und Prompt → T4;
  `[model]` global, je Bereich, Einengung → T1, T2; `reconcile` mit
  Vorschlag, zweite Notiz, stehender Fall nicht erneut → T5 (der stehende
  Fall ist unverändert der Weg über `withCurrentMode`); #1 → T6; #4 → T7;
  Attrappe und sieben Fälle → T8, T9; Selbstnutzung, Messen, Mutationen,
  Fertig-Bedingungen → T10. Die Richter gehören zu 4d, `bench` zu 4c-2.
- **Typen:** `config.ModelSettings`, `ProposerFor(s, m, role)`,
  `Propose(ctx, pkg, []evidence.Segment) (string, bool)`,
  `AreaLockPath`/`LockArea`, `AdvanceSources(page, updates) (string, bool,
  error)` stimmen zwischen den Tasks überein.
- **Offen gelassen, mit Regel statt Platzhalter:** die echten Namen der
  Welthilfen in `reindex_test.go` (T7), dort an die Datei gebunden, die sie
  festlegt.

# Verträglichkeitsmatrix als Daten — Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Welche Frameworks loomux fährt und was sie tragen, liegt als Daten im Repo, eine Doku-Seite je Stack lebt davon, und kein Eintrag bleibt ohne Fixture im Tor.

**Architecture:** Eine TOML-Datei neben `presets.toml` hält je Framework eine Zeile; ein Leser im Paket `verify` gibt sie als Typ heraus und weist jede Schemaverletzung zurück. Zwei Tests spiegeln die Daten: einer verlangt zu jeder Zeile ein Fixture-Projekt, in dem ihr Erkennungsmuster wirklich greift, der andere verlangt, dass die Doku-Seite des Stacks genau diese Zeilen nennt. Das Muster ist `internal/verify/linecov/tools.toml` samt `fixtures_test.go`, eine Ebene höher.

**Tech Stack:** Go, `github.com/BurntSushi/toml` (vorhanden), keine neue Abhängigkeit. Kein Versionsvergleich: was eine gefundene Fassung *trägt*, hält die Matrix als Text; die Entscheidung daraus zieht Teilsystem D.

**Spec:** `docs/.superpowers/specs/2026-10-16-godot-versionen-design.md` (Teilsystem B von vier; A ist gelandet, C der Testbericht-Leser, D die Nutznießer)

## Global Constraints

- **Die Matrix ist Daten, nicht Code.** Keine Framework- oder Versionskenntnis in Go-Quellen; der Leser kennt das Schema, nicht die Einträge.
- **Nur vorhandene Frameworks.** Eine Zeile beschreibt, was es gibt; loomux baut nichts nach.
- **Getragen wird, was der Hersteller noch versorgt** — die Zeile sagt, welche Linien eine Fassung trägt, und nennt die Quelle dieser Aussage.
- **Was nicht geprüft ist, sagt das.** Eine Zeile, deren Aussage niemand an einer echten Installation nachgewiesen hat, trägt `proven = false`, und die Doku zeigt es; loomux behauptet nichts, was es nicht geprüft hat.
- **Ein Fixture je Zeile**, nach dem Vorbild von `internal/verify/linecov/fixtures_test.go`.
- **Kein Versionsvergleich in diesem Teilsystem.** Das Projekt hat keine semver-Bibliothek, und die Entscheidung „trägt diese Fassung diese Linie" gehört zu D.
- **Umfang: der Stack `gdscript`.** Der Leser ist stackunabhängig, die Zeilen und die Doku-Seite sind es für diesen Plan nicht: `go`, `python` und die übrigen folgen, wenn sie gebraucht werden.
- Kommentare im Code englisch, Doku zweisprachig (`docs/en`, `docs/de`), Conventional Commits, 100 % Abdeckung mit begründeten Ausschlüssen, kein Modell als Mitautor.

## Review Focus

1. **Ein Erkennungsmuster, das im eigenen Fixture nicht greift** (`found = "plugin.cfg:gdUnit4"`, aber das Fixture schreibt den Namen anders): die Zeile behauptet eine Erkennung, die es nicht gibt. — Test in Task 2.
2. **Eine Doku-Seite, die ein Framework nennt, das die Daten nicht haben** — oder umgekehrt eine Zeile, die in der Doku fehlt: die Seite wird zur zweiten Wahrheit. — Test in Task 3.
3. **Zwei Zeilen mit demselben Stack und Namen:** ein Ladefehler, der beide nennt, kein stilles Gewinnen der letzten. — Test in Task 1.
4. **Ein `kind`, das keine Lane-Art ist** (`lint`, `test`, `coverage`), und ein unbekannter Schlüssel: Ladefehler, nicht stilles Überlesen — ein vertipptes `kidn = "test"` darf nicht konfiguriert aussehen und keine Wirkung haben. — Test in Task 1.
5. **Eine Zeile ohne `proven`:** fehlt das Feld, gilt `false`, und die Doku markiert sie als unerprobt. Ein weggelassenes Feld darf nicht als Zusage lesbar werden. — Tests in Task 1 und 3.

---

## Dateien

- **Neu** `internal/verify/frameworks.toml`: die Matrix. Neben `presets.toml`, weil beide von Stacks sprechen.
- **Neu** `internal/verify/frameworks.go`: Typ `Framework` und `LoadFrameworks`.
- **Neu** `internal/verify/frameworks_test.go`: Schema- und Ladefehlerprüfungen.
- **Neu** `internal/verify/frameworksfixtures_test.go`: die beiden Spiegeltests (Fixture je Zeile, Doku je Zeile).
- **Neu** `internal/verify/testdata/frameworks/gdscript/<name>/…`: je Zeile ein kleines Projekt, in dem das Muster greift.
- **Neu** `docs/en/stacks/gdscript.md`, `docs/de/stacks/gdscript.md`.
- **Ändern** `docs/en/README.md` und `docs/de/README.md` (Verweis auf die neue Seite), `CHANGELOG.md` **nicht** — in diesem Repo schreibt die Release-Automatik den Abschnitt aus dem Rumpf des Pull Requests.

---

### Task 1: Die Matrix und ihr Leser

**Files:**
- Create: `internal/verify/frameworks.toml`, `internal/verify/frameworks.go`, `internal/verify/frameworks_test.go`

**Interfaces:**
- Consumes: —
- Produces:
  ```go
  type Framework struct {
      Stack, Name, Kind, Found, Carries, Source string
      Proven bool
  }
  func LoadFrameworks() ([]Framework, error)
  ```
  `Found` ist ein Muster in der Form von `when_found` (`<datei>:<text>`), `Carries` die Linien, die diese Fassung laut `Source` trägt, als Text für Mensch und Doku.

- [ ] **Step 1: Write the failing tests**

```go
func TestTheMatrixLoads(t *testing.T) {
	fs, err := LoadFrameworks()
	if err != nil {
		t.Fatal(err)
	}
	var gdunit *Framework
	for i := range fs {
		if fs[i].Stack == "gdscript" && fs[i].Name == "gdUnit4" {
			gdunit = &fs[i]
		}
	}
	if gdunit == nil {
		t.Fatal("no row for gdUnit4")
	}
	if gdunit.Kind != "test" || gdunit.Found == "" || gdunit.Carries == "" || gdunit.Source == "" {
		t.Fatalf("%#v", *gdunit)
	}
	if !gdunit.Proven {
		t.Error("gdUnit4 on Godot 4 is the one line this repository has run itself")
	}
}

func TestEveryRowNamesAKindAndASource(t *testing.T) {
	fs, _ := LoadFrameworks()
	for _, f := range fs {
		if !slices.Contains([]string{"lint", "test", "coverage"}, f.Kind) {
			t.Errorf("%s/%s: kind %q", f.Stack, f.Name, f.Kind)
		}
		if f.Source == "" {
			t.Errorf("%s/%s: a claim without a source", f.Stack, f.Name)
		}
	}
}

func TestADuplicateRowIsALoadError(t *testing.T) {
	// Review Focus 3: the last one must not win silently.
	body := "[[framework]]\nstack=\"gdscript\"\nname=\"GUT\"\nkind=\"test\"\nfound=\"plugin.cfg:GUT\"\ncarries=\"3.x\"\nsource=\"https://example.com\"\n"
	_, err := parseFrameworks([]byte(body+body), "frameworks.toml")
	if err == nil || !strings.Contains(err.Error(), "gdscript/GUT") {
		t.Fatalf("%v", err)
	}
}

func TestAnUnknownKeyOrKindIsALoadError(t *testing.T) {
	// Review Focus 4: a misspelt key must not look configured.
	for _, body := range []string{
		"[[framework]]\nstack=\"gdscript\"\nname=\"X\"\nkidn=\"test\"\n",
		"[[framework]]\nstack=\"gdscript\"\nname=\"X\"\nkind=\"typecheck\"\nfound=\"a:b\"\ncarries=\"c\"\nsource=\"d\"\n",
	} {
		if _, err := parseFrameworks([]byte(body), "frameworks.toml"); err == nil {
			t.Errorf("accepted %q", body)
		}
	}
}

func TestAMissingProvenIsFalse(t *testing.T) {
	// Review Focus 5: an omitted field is not a promise.
	body := "[[framework]]\nstack=\"gdscript\"\nname=\"X\"\nkind=\"test\"\nfound=\"a:b\"\ncarries=\"c\"\nsource=\"d\"\n"
	fs, err := parseFrameworks([]byte(body), "frameworks.toml")
	if err != nil || len(fs) != 1 || fs[0].Proven {
		t.Fatalf("%#v %v", fs, err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/verify/ -run 'Matrix|EveryRowNames|DuplicateRow|UnknownKeyOrKind|MissingProven' -v`
Expected: FAIL — `LoadFrameworks` und `parseFrameworks` sind nicht definiert.

- [ ] **Step 3: Implement `parseFrameworks(data []byte, path string) ([]Framework, error)` und `LoadFrameworks()` in `internal/verify/frameworks.go`**

Gelesen wird die eingebettete `frameworks.toml` (`//go:embed`, wie `presets.toml` es hält — prüfe dort die Form und folge ihr). Jeder Fehler nennt die Datei und die Zeile, wie die übrigen Ladefehler dieses Pakets: fehlendes Pflichtfeld, unbekannter Schlüssel (`toml.MetaData.Undecoded`), `kind` außerhalb von `lint`/`test`/`coverage`, zwei Zeilen mit gleichem `stack`/`name`. Die Reihenfolge der Rückgabe ist die der Datei.

- [ ] **Step 4: Write `internal/verify/frameworks.toml`**

Je Framework eine Zeile. Für `gdscript` gehören diese hinein, mit genau diesen Aussagen und Quellen:

| name | kind | found | carries | proven | source |
|---|---|---|---|---|---|
| `gdUnit4` | test | `plugin.cfg:gdUnit4` | Godot 4.3 bis 4.8 | ja | `https://github.com/godot-gdunit-labs/gdUnit4` |
| `NanoCoverage` | coverage | `plugin.cfg:NanoCoverage` | Godot 4 | ja | `https://github.com/IgorBayerl/nano-coverage-godot` |
| `gdtoolkit4` | lint | `project.godot:config_version=5` | Godot 4 | ja | `https://github.com/scony/godot-gdscript-toolkit` |
| `GUT` | test | `plugin.cfg:Gut` | Godot 3.x | nein | `https://github.com/bitwes/Gut` |
| `gdtoolkit3` | lint | `project.godot:config_version=4` | Godot 3 | nein | `https://github.com/scony/godot-gdscript-toolkit` |
| `godot-code-coverage` | coverage | `plugin.cfg:Coverage` | auf Godot 3.5 dokumentiert, 3.6 unbestätigt | nein | `https://github.com/jamie-pate/godot-code-coverage` |

Der Kopf der Datei sagt in zwei Sätzen, was die Matrix entscheidet (ob eine **gefundene** Fassung die Linie des Projekts trägt) und was nicht (ob ein Framework da ist — das sagt `when_found`), und nennt den Spiegeltest, der ein Fixture je Zeile verlangt.

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/verify/ -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/verify/frameworks.go internal/verify/frameworks.toml internal/verify/frameworks_test.go
git commit -m "feat(verify): hold the framework matrix as data"
```

---

### Task 2: Ein Fixture je Zeile

**Files:**
- Create: `internal/verify/frameworksfixtures_test.go`, `internal/verify/testdata/frameworks/gdscript/<name>/…` (sechs Verzeichnisse)

**Interfaces:**
- Consumes: `LoadFrameworks`, `Framework.Found` aus Task 1
- Produces: —

- [ ] **Step 1: Write the failing test**

```go
// Every row's pattern must match inside its own fixture: a row that claims a
// detection nobody can reproduce is worse than no row.
func TestEveryFrameworkRowHasAFixtureItFinds(t *testing.T) {
	fs, err := LoadFrameworks()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		dir := filepath.Join("testdata", "frameworks", f.Stack, f.Name)
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("%s/%s: no fixture at %s", f.Stack, f.Name, dir)
			continue
		}
		if !HasTests(dir, []string{f.Found}) {
			t.Errorf("%s/%s: %q matches nothing under %s", f.Stack, f.Name, f.Found, dir)
		}
	}
}
```

`verify.HasTests(dir string, patterns []string) bool` ist die vorhandene Prüfung hinter `when_found` (`plan.go:454` benutzt sie über `PlanEnv`, verdrahtet in `internal/cli/check.go:219`). Benutze sie, schreibe keine zweite Mustersuche — sie wäre die zweite Wahrheit darüber, was „gefunden" heißt.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/verify/ -run EveryFrameworkRowHasAFixture -v`
Expected: FAIL — sechs Zeilen ohne Fixture.

- [ ] **Step 3: Write the six fixtures**

Je Zeile das Kleinste, worin ihr Muster greift: für ein `plugin.cfg`-Muster ein `addons/<name>/plugin.cfg` mit dem Namensfeld, für ein `project.godot`-Muster eine `project.godot` mit der betreffenden Zeile. Keine Engine, kein Testcode — das Fixture beweist die Erkennung, nicht den Lauf.

- [ ] **Step 4: Run the test**

Run: `go test ./internal/verify/ -run EveryFrameworkRowHasAFixture -v`
Expected: PASS

- [ ] **Step 5: Gegenprobe durch Mutation**

Ändere in einem Fixture den Namen (`gdUnit4` → `gdUnit5`) und lass den Test laufen: er muss rot werden. Nimm die Änderung zurück. Ohne diese Probe steht nicht fest, dass der Test etwas prüft.

- [ ] **Step 6: Commit**

```bash
git add internal/verify/frameworksfixtures_test.go internal/verify/testdata/frameworks
git commit -m "test(verify): demand a fixture that finds each framework row"
```

---

### Task 3: Die Doku-Seite je Stack

**Files:**
- Create: `docs/en/stacks/gdscript.md`, `docs/de/stacks/gdscript.md`
- Modify: `internal/verify/frameworksfixtures_test.go` (zweiter Spiegeltest), `docs/en/README.md`, `docs/de/README.md`

**Interfaces:**
- Consumes: `LoadFrameworks` aus Task 1
- Produces: —

- [ ] **Step 1: Write the failing test**

```go
// The page and the data say the same thing, in both languages: a page that
// names a framework the data lack, or lacks one the data name, is a second
// truth.
func TestEveryFrameworkRowIsDocumented(t *testing.T) {
	fs, err := LoadFrameworks()
	if err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"en", "de"} {
		page := filepath.Join("..", "..", "docs", lang, "stacks", "gdscript.md")
		body, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, f := range fs {
			if f.Stack != "gdscript" {
				continue
			}
			if !strings.Contains(text, f.Name) || !strings.Contains(text, f.Source) {
				t.Errorf("%s: %s is not documented with its source", page, f.Name)
			}
			if !f.Proven && !strings.Contains(text, f.Name+" (unproven)") {
				t.Errorf("%s: %s must be marked unproven", page, f.Name)
			}
		}
		for _, name := range frameworkNamesIn(text) {
			if !slices.ContainsFunc(fs, func(f Framework) bool { return f.Name == name }) {
				t.Errorf("%s names %s, which the matrix does not have", page, name)
			}
		}
	}
}
```

`frameworkNamesIn(text string) []string` liest die Namensspalte der Tabelle auf der Seite; die Form der Tabelle legt Step 3 fest, und der Leser hält sich daran.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/verify/ -run EveryFrameworkRowIsDocumented -v`
Expected: FAIL — die Seiten gibt es nicht.

- [ ] **Step 3: Write both pages**

Eine Tabelle, je Zeile ein Framework: Name, Art der Lane, woran loomux es erkennt, welche Linien es trägt, Quelle. Eine unerprobte Zeile trägt im Namensfeld `(unproven)` und darunter einen Satz, was fehlt, um sie zu erproben. Dazu ein kurzer Kopf: dass die Tabelle aus `internal/verify/frameworks.toml` kommt, dass ein Test beides gegeneinander hält, und dass eine fehlende Zeile kein Verbot ist — loomux fährt, was es findet, und sagt, was es über eine unbekannte Fassung nicht weiß. Die deutsche Seite ist die Übersetzung, nicht eine zweite Aussage.

- [ ] **Step 4: Verweise in beiden README-Dateien**

Eine Zeile je Sprache im Inhaltsverzeichnis, in der Form der Nachbarn.

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/verify/ -count=1`
Expected: PASS

- [ ] **Step 6: Gegenprobe durch Mutation**

Entferne eine Zeile aus der englischen Seite: der Test muss rot werden. Nimm es zurück.

- [ ] **Step 7: Run the whole gate**

Run: `loomux check all`
Expected: alle Lanes grün, 100 % Abdeckung der neuen Dateien, jeder Ausschluss begründet.

- [ ] **Step 8: Commit**

```bash
git add docs internal/verify
git commit -m "docs: say which GDScript frameworks loomux drives and what they carry"
```

---

## Selbstprüfung

**Abdeckung des Entwurfs:** Dieser Plan deckt den Abschnitt „Die Verträglichkeitsmatrix je Stack" — Daten im Repo, Doku-Seite je Stack, Spiegelung in den Tests — und Punkt 3 bis 5 des allgemeinen Vorgehens. Nicht hier: der Testbericht-Leser (C), die Nutznießer der Matrix, also die `unready`-Entscheidung aus einer Framework-Fassung (D), die Korpus-Scheibe (D).

**Offene Entscheidungen, die ich getroffen habe** (im Entwurf dem Bauenden freigestellt): die Matrix liegt als eine Datei je Repo, nicht je Stack; die Doku-Seite wird **geprüft**, nicht erzeugt, weil ein Generator eine dritte Stelle wäre und die Prüfung denselben Dienst tut; `proven` ist ein eigenes Feld, weil der Entwurf verlangt, Unerprobtes als solches zu zeigen.

**Typtreue:** `Framework` mit `Stack`, `Name`, `Kind`, `Found`, `Carries`, `Source`, `Proven`; `LoadFrameworks`, `parseFrameworks`, `frameworkNamesIn` werden in allen drei Tasks gleich benannt.

**Was bewusst fehlt:** jeder Versionsvergleich. Die Matrix sagt in Worten, was eine Fassung trägt; wer daraus `unready` ableitet, braucht einen Vergleicher, und der gehört mit seiner Entscheidung nach D.

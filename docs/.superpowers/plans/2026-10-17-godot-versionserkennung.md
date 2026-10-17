# Godot-Versionserkennung und Binärauswahl — Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `{godot}` löst in jedem Godot-Projekt auf — auch in einem der 3er-Linie, die kein `config/features` hat — und findet auf einer Maschine mit mehreren Engines die zur Linie passende, ohne dass loomux Zustand darüber führt.

**Architecture:** `projectGodot` liefert statt eines Versionsstrings einen Typ `Line`, der die Hauptversion immer und die Nebenversion nur dann trägt, wenn das Projekt sie nennt; `config_version` ist der Rückfall, wenn `config/features` fehlt. Die Auflösung wird von „erstes vorhandenes Binary, dann prüfen" zu „Kandidaten der Reihe nach, erster passender gewinnt": `GODOT_BIN`, die Dateien eines Ordners, den `GODOT_BIN_DIR` nennt, dann `godot`/`godot4` auf dem `PATH`. Nichts wird aufgezeichnet, also kann nichts veralten; die Antwort jedes Binarys wird wie heute je Lauf einmal geholt.

**Tech Stack:** Go, Tests im Paket mit Attrappen-Quellen (`godotSources`), keine Engine, keine neue Abhängigkeit.

**Spec:** `docs/.superpowers/specs/2026-10-16-godot-versionen-design.md` (Teilsystem A von vier; B Matrix, C Testbericht-Leser, D Nutznießer folgen in eigenen Plänen)

## Global Constraints

- Getragen wird, was der Hersteller noch versorgt; **keine einprogrammierte Liste** unterstützter Godot-Versionen.
- Die Version kommt **nie** aus `.loomux/config.toml`, `registry.toml` oder einer eigenen Zustandsdatei — nur aus `project.godot`.
- **Kein neuer Zustand.** Keine Datei, die Installationen aufzeichnet: eine Aufzeichnung veraltet, sobald jemand Godot an derselben Stelle hochzieht, und zöge Schreibpfad, Sperre und einen Pflegebefehl nach sich. Für Sprachen mit Versionsverwalter (mise, asdf, nvm, pyenv, SDKMAN) ist der `PATH` die Antwort; Godot ist der Sonderfall einer heruntergeladenen Datei, und dafür genügt ein Ordner.
- Hauptversionsvergleich **nur** im 3er-Zweig. Ein 4er-Projekt ohne `config/features` bleibt der Fehler, der es heute ist.
- Eine `--version`-Abfrage je **verschiedenem Binary** und Lauf — so hält es die Tabelle `seen` in `GodotFor` schon. Zeitgrenze `probeTimeout` = 30 s, jede gegen das Restbudget (`errBudget`); eine Lane, die nicht mehr fragen kann, endet als `budget`.
- Aufgelöst wird **je Bereich**: `GodotFor` entsteht einmal je Lauf und gibt eine Funktion über `dir` zurück. Mehrere Godot-Projekte verschiedener Linien in einem Lauf sind der Normalfall; `lock` greift je Stack und Bereich und blockiert sie nicht gegeneinander.
- **Ein genanntes Binary wird an die Linie gehalten, nicht übergangen** — `GODOT_BIN` wie `[verify.gdscript] godot`: ein Pfad, der die Linie nicht trägt, lässt die Lane `unready`, und die Meldung nennt `GODOT_BIN_DIR` als Ausweg. Nachträglich festgelegt (siehe Hauptbuch und Entwurf): eine Notiz für eine *laufende* Lane gibt es nicht, ein Übergehen wäre also stillschweigend. Die Reihum-Suche gilt für Ordner und `PATH`.
- Kein neues `detect`-Signal, kein Stackname `godot3`.
- Kommentare im Code englisch, Doku zweisprachig (`docs/en`, `docs/de`).
- Kein `Co-Authored-By` auf ein Modell in den Commits.

## Review Focus

1. **`config/features` und `config_version` widersprechen sich** (`config_version=4` neben `"4.7"`): `config/features` gewinnt, weil Godot es schreibt und `config_version` nur die Dateiform zählt. — Test in Task 1.
2. **`config_version` ist weder 4 noch 5** (Godot 2, oder eine künftige 6): kein Raten, dieselbe `unready`-Meldung wie bei fehlender Angabe, die die Datei nennt. — Test in Task 1.
3. **`GODOT_BIN_DIR` zeigt auf nichts** — fehlender Ordner, leerer Ordner, oder nur Dateien, die keine Engine sind: keine Kandidaten, kein Fehler, die Suche läuft zum `PATH` weiter und der Bericht sagt, was dort gesucht wurde. — Test in Task 3.
4. **Zwei Projekte verschiedener Linien in einem Lauf:** jeder Bereich löst aus seiner eigenen `project.godot` auf und bekommt das Binary seiner Linie, jedes Binary wird einmal gefragt. Ist `GODOT_BIN` gesetzt, bleibt die Lane des Projekts einer anderen Linie `unready` — siehe die nachträgliche Festlegung oben. — Tests in Task 3 (`TestOneRunServesTwoLines`, `TestGodotBinKeepsItsAuthority`).
5. **Ein Kandidat im Ordner ist unter Windows die Fensterfassung** statt `_console.exe`: `console()` greift auch für Ordnerkandidaten, sonst schweigt das Binary zu `--version` und wird fälschlich verworfen. — Test in Task 3.
6. **Das Budget endet mitten in der Kandidatensuche:** die Lane wird `budget` und nennt das Binary, das nicht mehr antworten konnte — sie wird nicht `unready`, als hätte es die falsche Fassung. — Test in Task 3.

---

## Dateien

- **Ändern** `internal/verify/godot.go`: `projectGodot`, `resolve`, `find` — Linie statt Versionsstring, Kandidatenschleife statt „erster Treffer".
- **Neu** `internal/verify/godotline.go`: Typ `Line` samt Vergleich. Eigene Datei, weil `godot.go` mit über 300 Zeilen schon Suche, Abfrage und Zustände trägt.
- **Neu** `internal/verify/godotdir.go`: die Kandidaten eines Ordners.
- **Neu** `internal/verify/godotline_test.go`, `internal/verify/godotdir_test.go`; **ändern** `internal/verify/godot_test.go`.
- **Ändern** `docs/en/configuration.md` und `docs/de/configuration.md` (Zeile `{godot}`), `CHANGELOG.md`.

---

### Task 1: Die Linie aus `project.godot`

**Files:**
- Create: `internal/verify/godotline.go`, `internal/verify/godotline_test.go`
- Modify: `internal/verify/godot.go:211-239` (`featureVersion`, `projectGodot`)
- Test: `internal/verify/godot_test.go` (vorhandene `projectGodot`-Tests auf den neuen Rückgabetyp ziehen)

**Interfaces:**
- Consumes: —
- Produces: `type Line struct { Major, Minor string }`, `func (l Line) String() string`, `func (l Line) Known() bool`, `func projectGodot(path string) (want Line, mono bool, err error)`

- [ ] **Step 1: Write the failing tests**

```go
// godotline_test.go
func TestALineReadsItsFeaturesVersion(t *testing.T) {
	want, _, err := projectGodot(writeProject(t, "config/features=PackedStringArray(\"4.7\")\n"))
	if want != (Line{Major: "4", Minor: "7"}) || err != nil {
		t.Fatalf("%#v %v", want, err)
	}
	if want.String() != "4.7" {
		t.Fatalf("String() = %q", want.String())
	}
}

func TestAThreeProjectFallsBackToConfigVersion(t *testing.T) {
	// Godot 3 writes no config/features at all.
	want, _, err := projectGodot(writeProject(t, "config_version=4\n[application]\n"))
	if want != (Line{Major: "3"}) || err != nil {
		t.Fatalf("%#v %v", want, err)
	}
	if want.Known() || want.String() != "3" {
		t.Fatalf("Known() = %v, String() = %q", want.Known(), want.String())
	}
}

func TestFeaturesBeatConfigVersion(t *testing.T) {
	// Review Focus 1.
	want, _, _ := projectGodot(writeProject(t, "config_version=4\nconfig/features=PackedStringArray(\"4.7\")\n"))
	if want != (Line{Major: "4", Minor: "7"}) {
		t.Fatalf("%#v", want)
	}
}

func TestAnUnknownConfigVersionNamesTheFile(t *testing.T) {
	// Review Focus 2: no guessing from 3 or 6.
	path := writeProject(t, "config_version=3\n")
	if _, _, err := projectGodot(path); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("%v", err)
	}
}
```

`writeProject(t, body)` schreibt `body` in eine `project.godot` unter `t.TempDir()` und gibt den Pfad zurück — das Gegenstück zum vorhandenen `godotWorld`, nur ohne Verzeichnisrückgabe.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/verify/ -run 'Line|ConfigVersion|Features' -v`
Expected: FAIL — `Line` ist nicht definiert, `projectGodot` gibt einen String zurück.

- [ ] **Step 3: Implement `Line` in `internal/verify/godotline.go`**

`Major` ist immer gesetzt, `Minor` nur, wenn das Projekt sie nennt. `Known()` ist `Minor != ""`. `String()` ist `Major` allein oder `Major + "." + Minor`.

- [ ] **Step 4: Implement the fallback in `projectGodot`**

Reihenfolge: `config/features` (vorhandenes `featureVersion`) gewinnt; sonst `config_version` mit `4` → `Line{Major: "3"}`. Jeder andere Wert — `5` ohne `config/features` eingeschlossen — und eine Datei ohne beides geben den Fehler, der heute bei fehlendem `config/features` entsteht, mit dem Pfad darin. Die `[dotnet]`-Erkennung bleibt, wie sie ist.

- [ ] **Step 5: Ziehe die Aufrufer auf den neuen Typ**

`resolve` vergleicht vorläufig weiter auf `want.String()`; der Vergleich selbst ist Task 2.

- [ ] **Step 6: Run the package tests**

Run: `go test ./internal/verify/`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/verify/godotline.go internal/verify/godotline_test.go internal/verify/godot.go internal/verify/godot_test.go
git commit -m "feat(verify): read the Godot line from config_version where features is absent"
```

---

### Task 2: Vergleich nach Linie

**Files:**
- Modify: `internal/verify/godotline.go`, `internal/verify/godot.go:184-190` (die Abweichungsmeldung)
- Test: `internal/verify/godotline_test.go`, `internal/verify/godot_test.go`

**Interfaces:**
- Consumes: `Line` aus Task 1
- Produces: `func (l Line) Matches(reported string) bool`

- [ ] **Step 1: Write the failing tests**

```go
func TestAKnownLineMatchesMajorAndMinor(t *testing.T) {
	l := Line{Major: "4", Minor: "7"}
	if !l.Matches("4.7") || l.Matches("4.8") || l.Matches("3.6") {
		t.Fatal("a named minor must be compared")
	}
}

func TestAThreeLineMatchesTheMajorAlone(t *testing.T) {
	l := Line{Major: "3"}
	if !l.Matches("3.6") || !l.Matches("3.5") || l.Matches("4.7") {
		t.Fatal("config_version names no minor, so only the major can be compared")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/verify/ -run Matches -v`
Expected: FAIL — `Matches` ist nicht definiert.

- [ ] **Step 3: Implement `Matches(reported string) bool`**

`reported` ist, was `parseGodotVersion` liefert (`major.minor`). Bei `Known()` auf beide Stellen vergleichen, sonst nur auf die Hauptversion.

- [ ] **Step 4: Nutze `Matches` in `resolve`**

Der Vergleich `got != want` wird `!want.Matches(got)`. Die Meldung nennt weiter gefundene und verlangte Fassung; bei einer Linie ohne Nebenversion steht dort `Godot 3` statt `Godot 3.6`.

- [ ] **Step 5: Run the package tests**

Run: `go test ./internal/verify/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/verify
git commit -m "feat(verify): compare a Godot binary against the line the project names"
```

---

### Task 3: Kandidaten statt erstem Treffer

**Files:**
- Create: `internal/verify/godotdir.go`, `internal/verify/godotdir_test.go`
- Modify: `internal/verify/godot.go:104-190` (`find` → `candidates`, `resolve`)
- Test: `internal/verify/godot_test.go` (Helfer `sources` um `GODOT_BIN_DIR` und Verzeichniseinträge erweitern)
- Modify: `docs/en/configuration.md:300`, die entsprechende Zeile in `docs/de/configuration.md`, `CHANGELOG.md`

**Interfaces:**
- Consumes: `Line`, `Matches` aus Task 1 und 2
- Produces: `func dirCandidates(dir string, readDir func(string) ([]string, error), console func(string) string) []string`; `godotSources.readDir func(string) ([]string, error)`; `func (s godotSources) candidates() (cands []candidate, looked []string)` mit `type candidate struct { Path, From string }`

- [ ] **Step 1: Write the failing tests**

```go
// godotdir_test.go
func TestADirOffersItsEnginesInAStableOrder(t *testing.T) {
	entries := []string{"Godot_v3.6.3_win64.exe", "Godot_v4.7.2_win64.exe", "readme.txt", "project.godot"}
	got := dirCandidates("C:/engines", listing(entries), func(p string) string { return p })
	want := []string{"C:/engines/Godot_v3.6.3_win64.exe", "C:/engines/Godot_v4.7.2_win64.exe"}
	if !slices.Equal(got, want) {
		t.Fatalf("%#v", got)
	}
}

func TestADirThatIsNotThereOffersNothing(t *testing.T) {
	// Review Focus 3.
	fail := func(string) ([]string, error) { return nil, os.ErrNotExist }
	if got := dirCandidates("C:/gone", fail, nil); got != nil {
		t.Fatalf("%#v", got)
	}
	if got := dirCandidates("C:/empty", listing(nil), nil); got != nil {
		t.Fatalf("%#v", got)
	}
}

func TestADirCandidateGetsTheConsoleBuild(t *testing.T) {
	// Review Focus 5.
	entries := []string{"Godot_v4.7.2_win64.exe", "Godot_v4.7.2_win64_console.exe"}
	got := dirCandidates("C:/e", listing(entries), func(p string) string {
		return strings.Replace(p, "win64.exe", "win64_console.exe", 1)
	})
	if len(got) != 2 || !strings.HasSuffix(got[0], "_console.exe") {
		t.Fatalf("%#v", got)
	}
}

// godot_test.go
func TestTheFirstMatchingCandidateWins(t *testing.T) {
	dir := godotWorld(t, "3.6", false)
	s := sources(dir, "", "linux", map[string]string{"GODOT_BIN_DIR": "/e"}, nil,
		[]string{"/e/godot36", "/e/godot47"},
		map[string]string{"/e/godot36": "3.6.3.stable", "/e/godot47": "4.7.2.stable"})
	s.readDir = listing([]string{"godot36", "godot47"})
	if bin, state, note := s.resolve(dir); bin != "/e/godot36" || state != "" {
		t.Fatalf("%q %v %q", bin, state, note)
	}
}

func TestGodotBinLosesToTheWantedLine(t *testing.T) {
	// Review Focus 4: one variable cannot serve a 3.6 and a 4.7 project at once.
	dir := godotWorld(t, "3.6", false)
	s := sources(dir, "", "linux", map[string]string{"GODOT_BIN": "/e/godot47", "GODOT_BIN_DIR": "/e"}, nil,
		[]string{"/e/godot36", "/e/godot47"},
		map[string]string{"/e/godot36": "3.6.3.stable", "/e/godot47": "4.7.2.stable"})
	s.readDir = listing([]string{"godot36", "godot47"})
	bin, state, note := s.resolve(dir)
	if bin != "/e/godot36" || state != "" {
		t.Fatalf("%q %v %q", bin, state, note)
	}
	if !strings.Contains(note, "GODOT_BIN") {
		t.Fatalf("skipping the variable must be said, not hidden: %q", note)
	}
}

func TestAProjectPinThatContradictsTheProjectStaysUnready(t *testing.T) {
	// [verify.gdscript] godot is the project speaking; a contradiction there is not tidied away.
	dir := godotWorld(t, "3.6", false)
	s := sources(dir, "/e/godot47", "linux", map[string]string{"GODOT_BIN_DIR": "/e"}, nil,
		[]string{"/e/godot36", "/e/godot47"}, map[string]string{"/e/godot47": "4.7.2.stable"})
	s.readDir = listing([]string{"godot36", "godot47"})
	if bin, state, _ := s.resolve(dir); bin != "" || state != StateUnready {
		t.Fatalf("%q %v", bin, state)
	}
}

func TestNoMatchingCandidateNamesWhatWasFound(t *testing.T) {
	dir := godotWorld(t, "4.8", false)
	s := sources(dir, "", "linux", map[string]string{"GODOT_BIN_DIR": "/e"}, nil,
		[]string{"/e/godot47"}, map[string]string{"/e/godot47": "4.7.2.stable"})
	s.readDir = listing([]string{"godot47"})
	if bin, state, note := s.resolve(dir); bin != "" || state != StateUnready ||
		!strings.Contains(note, "4.7") || !strings.Contains(note, "4.8") {
		t.Fatalf("%q %v %q", bin, state, note)
	}
}

func TestTheBudgetEndingMidSearchIsNotAMismatch(t *testing.T) {
	// Review Focus 6: budget, not unready.
	…
}
```

`listing(entries)` ist ein `readDir`, das `entries` für jedes Verzeichnis liefert; `sources` bekommt `readDir` als weiteres Feld, vorbelegt mit einem, das nichts findet.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/verify/ -run 'Dir|Candidate|GodotBin|Budget' -v`
Expected: FAIL — `dirCandidates` und `candidates` sind nicht definiert.

- [ ] **Step 3: Implement `dirCandidates` in `internal/verify/godotdir.go`**

Aus den Einträgen des Ordners die, deren Name mit `godot` beginnt (ohne Rücksicht auf Groß- und Kleinschreibung), Verzeichnisse und `project.godot` ausgenommen, nach Name sortiert für einen festen Lauf. Jeder Pfad geht durch `console`, damit unter Windows die Konsolenfassung genommen wird. Ein Ordner, der nicht lesbar ist, liefert nichts und keinen Fehler — er ist eine Quelle, kein Vertrag.

- [ ] **Step 4: Implement `candidates()` in `internal/verify/godot.go`**

Reihenfolge: `GODOT_BIN` (wenn vorhanden, `From` `"GODOT_BIN"`), dann `dirCandidates` über `GODOT_BIN_DIR` (`From` `"GODOT_BIN_DIR"`), dann `godot`/`godot4` vom `PATH` (`From` `"PATH"`). Der `looked`-Bericht behält seine heutigen Zeilen und bekommt eine über den Ordner: ungesetzt, nicht lesbar, oder wie viele Kandidaten er hatte. `[verify.gdscript] godot` ist **kein** Kandidat — es wird in `resolve` davor behandelt.

- [ ] **Step 5: Baue `resolve` auf die Schleife um**

1. `projectGodot` lesen; scheitert das, ist das die Meldung (ohne Linie ist kein Binary wählbar).
2. Ist `[verify.gdscript] godot` gesetzt und vorhanden: fragen, passt es → nehmen, sonst `unready` wie heute. Hier wird nicht weitergesucht.
3. Sonst `candidates()` der Reihe nach fragen (`s.version`, also je Binary einmal), den ersten nehmen, dessen Antwort `want.Matches` erfüllt. Übersprungene Kandidaten kommen in die Notiz, mit ihrer Fassung und ihrer Quelle — `GODOT_BIN` ausdrücklich, weil das Übergehen einer gesetzten Variablen sichtbar sein muss.
4. Kein Kandidat: gab es keine, `missing-tool` mit `looked`; gab es welche, `unready` mit der ersten gefundenen Fassung, der gewollten Linie und dem Hinweis auf `GODOT_BIN_DIR`.
5. Endet das Budget in der Schleife (`errBudget`), ist das `budget` samt dem Binary, das nicht mehr antworten konnte — nicht `unready`.

- [ ] **Step 6: Run the package tests**

Run: `go test ./internal/verify/`
Expected: PASS

- [ ] **Step 7: Beschreibe die Auflösung in beiden Doku-Fassungen und im CHANGELOG**

Die `{godot}`-Zeile bekommt: die Linie kommt aus `config/features`, sonst aus `config_version` (dann nur die Hauptversion); gesucht wird `GODOT_BIN`, `[verify.gdscript] godot`, die Engines in `GODOT_BIN_DIR`, dann der `PATH`; der erste Kandidat, dessen Fassung zur Linie passt, gewinnt; ein genanntes Binary (`GODOT_BIN` oder `[verify.gdscript] godot`), das die Linie nicht trägt, lässt die Lane `unready` und nennt `GODOT_BIN_DIR` als Ausweg. Im CHANGELOG unter `Added` die neue Variable, unter `Changed` die Auflösung nach Linie.

- [ ] **Step 8: Run the whole gate**

Run: `loomux check`
Expected: alle Lanes grün; Abdeckung der neuen Dateien bei 100 %, jeder Ausschluss mit Begründung.

- [ ] **Step 9: Commit**

```bash
git add internal/verify docs CHANGELOG.md
git commit -m "feat(verify): pick the first Godot binary whose version fits the project's line"
```

---

## Selbstprüfung

**Abdeckung des Entwurfs:** Dieser Plan deckt die Versionsquelle (`config/features` → `config_version`), den Hauptversionsvergleich im 3er-Zweig, die Auswahl aus mehreren Engines und die Asymmetrie zwischen `GODOT_BIN` und dem Projektpfad. Nicht in diesem Plan, sondern in B, C und D: die Matrix samt Doku-Seiten und Fixture-Regel, der Testbericht-Leser, die Godot-3-Lane-Zeilen, die Graph-Sichtbarkeit, die Korpus-Scheibe, die `doctor`-Anzeige der Linie.

**Rücknahme gegenüber dem Entwurf:** Der Entwurf schrieb eine gemeinsame `installs.toml` mit `tool`, `path` und `version` fest. Dieser Plan führt sie nicht ein. Sie hätte nur die `--version`-Abfrage gespart — Millisekunden, keine Sekunden — und dafür Schreibpfad, Sperre, atomaren Tausch, einen `scan`-Befehl und Veraltung mitgebracht. Der Ordner leistet dasselbe ohne Zustand. Der Entwurf ist entsprechend nachgezogen.

**Typtreue:** `Line` mit `Major`/`Minor`, `Known()`, `String()`, `Matches(reported string)` wird in allen drei Tasks gleich benannt; `candidate{Path, From}`, `dirCandidates`, `candidates()`, `s.readDir` ebenso.

**Offene Stelle, die der Umsetzende entscheidet:** ob der Ordner rekursiv gelesen wird. Der Plan verlangt eine Ebene; Godot-Downloads liegen üblicherweise flach, und ein rekursiver Lauf über einen falsch gesetzten Ordner wäre teuer.

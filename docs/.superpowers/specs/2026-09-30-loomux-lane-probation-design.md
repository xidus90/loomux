# loomux: Schonfrist je Lane — Design

Stand 2026-09-30. Entschieden im Gespräch mit dem Nutzer am 2026-09-29 und 2026-09-30.
Zweig `feat/lane-probation`; er steht vorerst auf `docs/stage-4e-spec` und wird nach dessen
Merge auf `master` umgesetzt.

## Anlass

Wer loomux in ein bestehendes Projekt einrichtet, bekommt am selben Tag ein scharfes Tor über
Code, den bisher kein Tor geprüft hat. Zwei Piloten der Umstellung haben das gezeigt:

- `ecoflow` (2026-09-29): Das neue pre-commit-Tor verweigerte den Umstellungs-Commit wegen 83
  alter ruff-Befunde und zweier Preset-Lücken. Der Commit ging nur mit `--no-verify`.
- `space` (2026-09-30): Der Stop-Hook blockierte jede Sitzung (Exit 2), `wiki lint` meldete 790
  Befunde, die das alte Werkzeug nie gezeigt hatte.

Ziel: Mit der Installation muss nicht alles repariert sein. Eine Lane warnt zuerst nur. Scharf,
mit echtem Exit-Code, wird sie, sobald sie einmal grün war.

## Entscheidungen des Nutzers

1. Die Schonfrist gilt **je Lane**, nicht für das ganze Tor.
2. Der Zustand liegt **versioniert im Repo**; er gilt für alle Mitarbeitenden und im CI gleich.
3. **Das Tor schreibt selbst**: Ein grüner Lauf trägt die Lane ein.
4. Scharf stellt **nur der pre-commit-Lauf**, und nur, wenn er **insgesamt grün** endet; der
   Hook legt die Zustandsdatei in denselben Commit.
5. **Ohne Datei ist jede Lane scharf** (heutiges Verhalten). Mit Datei ist jede Lane, die nicht
   darin steht, in Probe, auch eine, die später dazukommt.
6. Keine Frist. Lanes in Probe sind **sichtbar**, und ein **Mensch** kann eine Lane von Hand
   scharf stellen oder zurück in die Probe setzen.
7. `init` legt die Datei an, wenn vor dem Lauf weder `.loomux/config.toml` noch
   `.loomux/armed.toml` noch ein pre-commit-Hook von loomux da war: Nur ein Projekt, das
   loomux noch nicht eingerichtet hatte, bekommt die Schonfrist von selbst. Das `apply.sh` der
   Umstellung legt die Datei in seinem Konfigurationsschritt mit an.
8. Im Stop-Hook werden die Befunde der Probe-Lanes **zum geprüften Stand gemerkt** und bei
   unverändertem Baum von der Platte wiederholt, statt die Kette neu zu fahren.
9. Der Agent erfährt die Lanes in Probe **beim Session-Start**, nicht durch Blockieren.
10. Reihenfolge: erst `docs/stage-4e-spec` mergen und veröffentlichen, dann dieses Feature
    mergen und veröffentlichen, **dann** die Welle der Umstellung.
11. Im Post-Edit-Hook **warnt** eine rote Lane in Probe, statt den Edit zu blockieren.
12. Bei einem **Teilcommit** (`git commit <pfad>`) stellt der Hook nicht scharf; der nächste
    gewöhnliche Commit holt es nach.
13. Ein Stop-Lauf, der nur wegen Probe nicht blockiert, setzt den **Blockzähler auf 0**.
14. Keine Roadmap-Zeile: Plan und Code kommen in einem Pull Request.

## Die Zustandsdatei

`.loomux/armed.toml`, versioniert, geschrieben mit LF.

```toml
# Lanes that are armed: a red run of one of them fails the gate.
# Written by the pre-commit gate; a human edits it through `loomux gate`.
armed = [
  "lint/python@.",
  "test/go@.",
]
```

- **Schlüssel:** immer die volle Form `<art>/<sprache>@<bereich>`, Bereich mit
  Vorwärtsschrägstrichen, `.` für die Wurzel. Der ausgegebene Lane-Name taugt nicht: Er trägt
  `@<bereich>` erst bei mehr als einem Bereich (`internal/verify/plan.go`, Namensbildung bei
  `len(areas) > 1`). Ein zweiter Unterordner mit `project.godot` würde `test/gdscript` in
  `test/gdscript@.` umbenennen, und die Lane fiele still in die Probe zurück. Projektweite Lanes
  (`[verify.project]`, `lint/wiki`) heißen `<art>/project@.` bzw. `lint/wiki@.`; der Plan
  liest vorher nach, wie diese Lanes im Code heute heißen, und bildet den Schlüssel aus Art,
  Sprache und Bereich des Jobs, nicht aus dem Anzeigenamen.
- **Form:** ein Schlüssel je Zeile, sortiert, ohne Duplikate. So ist ein Merge-Konflikt durch
  Vereinigen beider Seiten zu lösen; die Doku sagt das. `merge=union` in `.gitattributes` wird
  **nicht** gesetzt: Es würde auch eine Zeile zurückholen, die ein Mensch mit `gate disarm`
  entfernt hat.
- **Fehlt die Datei**, ist jede Lane scharf, und **jede Ausgabe bleibt Byte für Byte wie
  heute**. Das ist eine Invariante; die aufgezeichneten Fälle unter `testdata/cases/` prüfen sie.
- **Ist die Datei unlesbar** (kein TOML, falscher Typ, unbekannter Schlüssel), ist jede Lane
  scharf, und jeder Lauf meldet das auf stderr. Eine kaputte Datei entschärft nichts.
- **Ein Eintrag ohne Lane** (Sprache entfernt, Bereich umbenannt) schadet nicht; `gate status`
  nennt ihn als verwaist.

## Was „in Probe“ heißt

Eine Lane in Probe läuft wie jede andere und zeigt ihre Befunde. Ihr Zustand bleibt der echte
(`failed`, `timed-out`, `missing-tool`, `unready`), bekommt aber in der Ausgabe den Zusatz
`(probation)` und macht den Lauf nicht rot.

- Das Urteil kennt die Menge der scharfen Lanes: Der Lauf vermerkt an jedem Ergebnis, ob seine
  Lane in Probe ist, und die Stellen, die über Rot entscheiden (`check`, Post-Edit, Stop),
  fragen danach. `verify.Red` selbst sagt weiter nur, ob ein Zustand rot wäre. Rot ist ein
  Ergebnis nur, wenn die Lane scharf ist. `blocked` ist nicht rot, wenn die blockierende Lane in Probe ist (Coverage hinter
  einem Test in Probe); hinter einer scharfen Lane bleibt es rot.
- Das gilt gleich für `loomux check <profil>`, das pre-commit-Tor, den Stop-Hook und das CI:
  Alle lesen dieselbe Datei durch dieselbe Funktion.
- `CheckVerdict`s zweite Regel („nichts zu prüfen für `<art>`“) bleibt unberührt.
- Der Post-Edit-Hook blockiert heute bei einer roten Lane mit Exit 2
  (`internal/hooks/post_edit.go`, `verify.EditReport`). Für eine Lane in Probe blockiert er
  nicht mehr: Er endet mit 0 und gibt den Befund über den Kanal an den Agenten, über den ein
  PostToolUse-Hook bei Exit 0 überhaupt gehört wird (Claude Code:
  `hookSpecificOutput.additionalContext`; stderr sieht bei Exit 0 niemand). Der Plan belegt je
  Wirt, dass dieser Kanal für PostToolUse im Code vorhanden ist, oder baut ihn.
- `loomux check` und das pre-commit-Tor enden bei mindestens einer Lane in Probe immer mit
  einer Zeile `probation: <schlüssel>, <schlüssel> (warn only until a green commit arms them)`.
  Der Stop-Hook schreibt sie nur, wenn eine Lane in Probe rot ist, als Schluss seines Berichts
  (stderr bei Exit 0, den die nächste Sitzungseröffnung weitergibt). Der Post-Edit-Hook schreibt
  keine solche Zeile, sondern markiert jede rote Lane in Probe mit `(probation)`: eine Zeile bei
  jedem Edit wäre Rauschen im Kontext des Agenten. Lanes, die
  nie grün werden können, weil es nichts zu prüfen gibt (`not-applicable`, `unavailable`),
  stehen weder in dieser Zeile noch in `gate status`; `missing-tool` und `unready` stehen dort.

## Scharf stellen

**Nur ein Lauf mit `--arm` schreibt.** `loomux check precommit` läuft auch im CI und von Hand;
schriebe der Befehl selbst, schriebe auch das CI.

- `loomux check precommit --arm`: **Nur wenn der Lauf insgesamt grün endet (Exit 0)**, wird
  jede Lane mit Zustand `ok`, die noch nicht in der Datei steht, eingetragen. Endet der Lauf
  rot, schreibt `--arm` nichts: Was im Repo steht, stammt immer aus einem Commit, der durchging,
  und nach einem verweigerten Commit liegt keine geänderte Datei im Index. Da eine Lane in Probe
  den Lauf nicht rot macht, kostet das nur den Fall, in dem eine schon scharfe Lane rot ist.
  `not-applicable`, `unavailable`, `budget` und jede rote Form einer Lane in Probe tragen
  nichts ein. Gibt es die Datei nicht, schreibt `--arm` nichts (ohne Datei ist ohnehin alles
  scharf). `--arm` ist nur beim Profil `precommit` erlaubt, sonst Aufruffehler (Exit 2).
- **Der Hook von loomux selbst** (`.githooks/pre-commit`, `ci/gate.sh`) bleibt unverändert.
  loomux hat eine Konfiguration und bekommt nach Entscheidung 7 keine `armed.toml`; ein Umbau
  dort wäre Code, der nie schreibt.
- **Die Hook-Vorlage für Projekte** (`internal/setup/gitfiles/gitfiles.go`) ist heute
  `exec "${LOCALAPPDATA}/loomux/bin/loomux.exe" check precommit`. Neu: derselbe Einzeiler mit
  `--arm`. Das Stagen übernimmt der Befehl selbst: Hat `--arm` die Datei geschrieben, legt er
  sie mit `git add -- .loomux/armed.toml` in den Index, den git dem Hook gegeben hat. So gibt
  es genau eine Stelle, die den Teilcommit erkennt (siehe unten) und dann weder schreibt noch
  stagt; ein `git add` im Shell-Hook würde eine noch unversionierte Datei auch in einen
  Teilcommit ziehen. Scheitert das `git add`, meldet der Befehl es auf stderr; der Exit-Code
  des Tors bleibt.
- **Teilcommit:** Bei `git commit <pfad>` gibt git dem Hook einen temporären Index
  (gemessen: `next-index-<pid>.lock`) in `GIT_INDEX_FILE`. Ein `git add` dort brächte die Datei zwar in den
  Commit, im echten Index bliebe aber der alte Stand als gestagte Rücknahme stehen (gemessen
  mit git 2.54.0.vfs.0.4: `MM .loomux/armed.toml`), und ein folgender `--no-verify`-Commit
  würde Lanes still entschärfen. Darum stellt der Hook bei einem Teilcommit **nicht** scharf und
  schreibt nichts. Erkannt wird er am Index, den git übergibt: fehlt `GIT_INDEX_FILE` oder
  zeigt es auf `index` oder `index.lock` des Repos (das nach dem Commit der echte Index wird,
  so bei `git commit` und `git commit -a`), wird scharf gestellt; jeder andere Index ist ein
  Teilcommit. Verglichen wird per `os.SameFile`, nicht nach Schreibweise. Diese Regel ist
  Git-Internes: Der Plan **misst** sie als eigenen Schritt für `git commit`, `-a`, `<pfad>`,
  `--only`, `--include` und `--amend`, bevor Code darauf baut, und hält die Git-Version fest.
- **Bestehende Installationen:** Ein Hook mit der Kennzeile `# loomux pre-commit hook:` und dem
  alten Einzeiler ist der eigene ältere Stand. `init` ersetzt ihn durch den neuen (mit Diff, wie
  jede Änderung von `init`). Ein Hook ohne Kennzeile ist fremd: `init` lässt ihn stehen und
  nennt in den Hinweisen, dass er nicht scharf stellt — aber nur, wenn das Projekt eine
  `armed.toml` hat oder in diesem Lauf bekommt. Ohne die Datei gibt es nichts scharf zu
  stellen, und ein frischer Klon von loomux bekäme sonst bei jedem `init` einen Hinweis auf
  seinen eigenen Hook.
- **Ein Commit mit `--no-verify`** stellt nichts scharf.

**Folge, die die Doku nennt:** Ein Projekt mit eigenem pre-commit-Hook, der loomux nicht ruft
(`space`), stellt nie von selbst scharf. Dort bleibt es bei `gate arm` oder dem Einbau des
Aufrufs in den eigenen Hook.

## Die Befehle des Menschen

Eine eigene Gruppe, nicht unter `check`: `check <name>` ist ein Profilname, und `arm`/`disarm`
dort zu reservieren, wäre eine Änderung am Konfigurationsformat (`[verify.profiles]` kennt
reservierte Namen, `internal/verify/schema.go`).

- `loomux gate status [--root <dir>]`: je Lane `armed`, `probation` oder `orphan`; ohne Datei
  die Zeile `no .loomux/armed.toml: every lane is armed`. Nur lesend, für Agenten erlaubt.
- `loomux gate arm <schlüssel>…`: trägt ein, auch wenn die Lane rot ist („ab jetzt gilt es“).
  Ein Schlüssel, zu dem es keine Lane gibt, ist ein Fehler.
- `loomux gate disarm <schlüssel>…`: entfernt den Eintrag. Gibt es die Datei nicht (alles
  scharf), legt der Befehl sie an, mit jeder anderen Lane als scharf eingetragen.
- `gate arm` ohne Datei schreibt nichts: Ohne Datei ist ohnehin alles scharf. Ein unbekannter
  Schlüssel ist trotzdem zuerst ein Fehler, mit und ohne Datei.
- `loomux gate disarm --all`: legt die Datei leer an oder leert sie. Das ist der Weg, einem
  Bestandsprojekt nachträglich die Schonfrist zu geben.

`arm` und `disarm` führt nur ein Mensch aus. Der Wächter verweigert sie einem Agenten, wie
`init`, `config` und `area add`.

## Der Wächter

`.loomux/armed.toml` bekommt denselben Regelsatz wie `.loomux/config.toml`
(`internal/hooks/guard.go`, `manifestReason`): Write/Edit auf den Pfad, die Shell-Formen
(Umleitung, `cp`, `mv`, `sed -i`, `tee`, `git checkout --`, `git restore`, PowerShell-Schreiber)
und die Schreibweisen über den Eltern-Ordner. Eine entfernte Zeile entschärft, darum reicht die
Write-Sperre allein nicht.

- Ein Commit eines Agenten löst den pre-commit-Hook aus, und der stellt scharf. Das ist
  gewollt: Der Agent kann über diesen Weg nur scharf stellen, nie entschärfen.
- Das Löschen der Datei macht alles scharf, ist also die strengere Richtung. Es wird trotzdem
  verweigert, weil es den Zustand der Kolleginnen und Kollegen ändert.
- Abnahme per Differenzprobe: eine Batterie von Schreibweisen gegen den Stand vor und nach der
  Änderung; keine Zeile, die vorher verweigert war, darf danach durchgehen.
- **Benannte Lücken,** die `armed.toml` mit `config.toml` teilt und die dieses Feature nicht
  schließt: `git checkout <rev> -- .loomux`, `git checkout <rev> -- .`, `git restore -s <rev> .`,
  `git stash`, `git reset --hard` und `git switch` gehen heute durch; jede davon kann Lanes entschärfen. Sie stehen als festgehaltene Durchgänge
  in der Batterie und im Kommentar der Regel, damit niemand sie für geschlossen hält.
- **Eine Folge ohne Datei:** Die Ablehnung von `rm -rf .loomux` nennt einen Grund mehr. Das ist
  neben `init` die zweite Ausnahme von der Byte-gleich-Invariante; sie trifft nur den Text einer
  Ablehnung.

## Der Stop-Hook

Heute (`internal/hooks/stop.go`): Ein grüner Lauf setzt `base` auf `HEAD` und merkt sich den
Baum; beim selben Baum startet kein Werkzeug. Nach drei Blockaden in Folge gibt der Hook für
eine Runde auf.

Neu, für einen Lauf, in dem **keine scharfe Lane rot** ist, aber mindestens eine Lane in Probe
rot:

- Der Hook blockiert nicht (Exit 0) und setzt den Blockzähler auf 0: Die Reihe der Blockaden
  ist unterbrochen, weil das Sitzungsende durchging.
- `base` und der grüne Baum bleiben, wo sie sind: Der Lauf ist **nicht grün**.
- Der Hook merkt sich einen zweiten Stand, „gesehen“: den Baum, `HEAD` (wegen der Graph-Lane),
  die Menge der scharfen Lanes und die Befunde der Probe-Lanes, in seiner Zustandsdatei unter
  `.loomux/state/hooks/`. Die scharfe Menge gehört dazu, weil `armed.toml` in einem Projekt,
  das `.loomux/` ignoriert, nicht im Baum liegt: Ein `gate arm` dort ändert den Baum nicht, muss
  den gemerkten Stand aber entwerten.
- Beim nächsten Sitzungsende mit demselben Baum und demselben `HEAD` startet kein Werkzeug; der
  Hook schreibt die gemerkten Befunde auf stderr und endet mit 0.
- Ändert sich der Baum, läuft die Kette neu.
- Wird eine der Lanes inzwischen scharf (ein Kollege hat gepullt, `armed.toml` hat sich
  geändert), gilt der gemerkte Stand nicht mehr: `armed.toml` gehört zum Baum, also läuft die
  Kette ohnehin neu.

Ohne diese Regel führe ein Projekt in Probe die ganze Kette bei jedem Sitzungsende (gemessen
bei `ultraloom`: 89 s warm).

## Sichtbarkeit

- **Session-Start:** Der Hook gibt dem Agenten eine Zeile je Lane in Probe mit, dazu den
  zuletzt gemerkten Befundstand des Stop-Hooks, wenn es einen gibt. Der Kanal ist der, über den
  der Session-Start heute schon Kontext gibt (Claude Code:
  `hookSpecificOutput.additionalContext`, `internal/hosts/claude.go`; Antigravity:
  `injectSteps`); der Plan liest die Stelle in `internal/hooks/hook_session_start.go` nach und
  hängt dort an, statt einen neuen Kanal zu bauen. Ohne Lane in Probe und ohne Datei bleibt die Ausgabe unverändert.
- **`loomux status` / `doctor`:** dieselbe Liste, dazu verwaiste Einträge und der Hinweis auf
  einen fremden pre-commit-Hook, der nicht scharf stellt.
- **Ignorierte Datei:** Ignoriert das Projekt `.loomux/` oder `.loomux/armed.toml` per
  `.gitignore`, erreicht die Datei nie einen Commit und gilt nur auf diesem Rechner. `gate
  status` und `status` warnen davor.
- **Jeder Torlauf:** die `probation:`-Zeile.

## `init` und das Umstellungsskript

- `init` legt `.loomux/armed.toml` (leer: `armed = []`) genau dann an, wenn vor dem Lauf
  weder `.loomux/config.toml` noch `.loomux/armed.toml` da war **und im Hook-Verzeichnis kein
  pre-commit-Hook von loomux stand** (erkannt an der Kennzeile `# loomux pre-commit hook:`).
  Der Hook ist das Zeichen, dass das Projekt schon eingerichtet war: Ein reines Go-Projekt, das
  loomux seit Monaten ohne Konfiguration nutzt, behält sein scharfes Tor, auch wenn das nächste
  `init` seinen Hook erneuert. Wer dort die Schonfrist will, nimmt `gate disarm --all`.
  Ob `init` in diesem Lauf selbst
  eine Konfiguration schreibt, spielt keine Rolle: In einem reinen Go-Projekt schreibt es
  keine (das tut erst `area add`, `internal/setup/plan.go`), und das Projekt soll die
  Schonfrist trotzdem bekommen. In einem Projekt, das schon eine Konfiguration hat, legt es
  nichts an. Ein frischer Klon von loomux selbst, der sich per `init --yes` einrichtet, hat
  eine Konfiguration und bekommt also keine Datei; ebenso `ecoflow` und `space`, die schon
  umgestellt sind (dort `gate disarm --all` von Hand).
- Ersetzt `init` den eigenen älteren pre-commit-Hook, behält die Datei ihr Ausführungsbit
  (unter POSIX verlöre ein über eine Temp-Datei ersetzter Hook es und liefe still nicht mehr).
- `apply.sh` (`internal/switchover/apply.sh.tmpl`): Schritt 4 („The configuration, never over an
  existing one“) legt neben der `config.toml` die leere `armed.toml` an, wenn er die
  Konfiguration schreibt. Ein zweiter Lauf ändert nichts.

## Fehlerfälle

| Fall | Verhalten |
|---|---|
| Datei fehlt | alles scharf, Ausgabe wie heute |
| Datei unlesbar | alles scharf, Meldung auf stderr bei jedem Lauf |
| `--arm` ohne Datei | schreibt nichts |
| `--arm`, Datei lässt sich nicht schreiben | Meldung auf stderr; der Exit-Code des Tors bleibt, der Commit hängt nicht daran |
| `git add` im Hook scheitert | Meldung; der Commit geht durch, die Datei bleibt als Änderung im Arbeitsbaum |
| Merge-Konflikt in der Datei | Mensch vereinigt beide Seiten; solange Konfliktmarken stehen, gilt „unlesbar“ |
| `gate arm` mit unbekanntem Schlüssel | Fehler, nichts geschrieben |
| Lane in Probe und Werkzeug fehlt | warnt; wird nie von selbst scharf, `gate status` zeigt sie |

## Tests und Abnahme

- **Urteil:** je Zustand mit und ohne Datei; `blocked` hinter einer Lane in Probe gegen
  `blocked` hinter einer scharfen. Die Eingaben trennen die Regel vom nächstliegenden Falschen.
- **Schlüssel:** dieselbe Lane in einem Projekt mit einem und mit zwei Bereichen ergibt
  denselben Schlüssel.
- **Weltentest mit echtem git und echtem Hook:** (a) ein Commit stellt die grünen Lanes scharf,
  und `armed.toml` liegt im selben Commit; (b) ein zweiter Commit ändert die Datei nicht;
  (c) eine rote Lane in Probe lässt den Commit durch, eine scharfe rote hält ihn auf, und in
  diesem verweigerten Lauf wird keine andere grüne Lane eingetragen (Datei und Index unverändert);
  (d) ein Teilcommit (`git commit <pfad>`) stellt nichts scharf: Datei und Index bleiben
  unverändert, `git status` zeigt kein `MM`, und der nächste gewöhnliche Commit stellt scharf;
  (e) `git commit -a` stellt scharf wie (a); (f) `--no-verify` schreibt nichts.
- **Stop-Hook:** Probe-rot bewegt `base` nicht und setzt den Blockzähler auf 0 (zwei Blockaden,
  ein Lauf nur mit Probe-rot, eine Blockade: der Zähler steht auf 1, nicht 3); derselbe Baum
  startet kein Werkzeug und wiederholt die Befunde; ein geänderter Baum läuft neu.
- **Wächter:** die Differenzprobe.
- **Invariante ohne Datei:** die aufgezeichneten Fälle von `check` und den Hooks bleiben Byte
  für Byte gleich. **Ausgenommen ist `init`:** Es schreibt eine Datei mehr
  (`.loomux/armed.toml`) und einen anderen Text für den pre-commit-Hook. Aufgezeichnete
  `init`-Fälle gibt es nicht (`internal/cli/cases_4a2_test.go` spielt nur `merge-hook` nach,
  `parity/stufe-4a-2.md` sagt es); die Änderung trifft die Unit-Tests, die den alten Hook-Text
  festhalten (`internal/setup/plan_test.go`, `internal/setup/gitfiles/gitfiles_test.go`), und
  bekommt einen Abschnitt in `parity/stufe-4a-2.md`. Keine Aufzeichnung der alten Werkzeuge
  wird umgeschrieben.
- **`init`:** legt die Datei an, wenn vor dem Lauf weder Konfiguration noch Datei noch ein
  pre-commit-Hook von loomux stand (ein reines Go-Projekt ohne alles: angelegt; dasselbe mit
  einem loomux-Hook: nicht; eines mit einem fremden Hook: angelegt; ein Projekt mit
  Konfiguration: nicht; eines mit `armed.toml`: unberührt); ersetzt den eigenen älteren Hook, lässt einen fremden stehen und
  nennt ihn. Die Datei gehört zum Teil `config` von `init`: Wer diesen Teil abwählt, bekommt
  auch keine `armed.toml`.
- **`apply.sh`:** Weltentest, dass Schritt 4 die Datei anlegt und ein zweiter Lauf nichts ändert.
- Coverage 100 % je Funktion; je Regel eine Mutationsrunde.

## Doku und Papiere

- `docs/en|de/configuration.md` (die Datei, Probe, Schlüsselform, Merge), `hooks.md`
  (Stop-Hook, Session-Start, pre-commit), `cli-reference.md` (`check precommit --arm`,
  `gate status|arm|disarm`), beide READMEs.
- Nachtrag in der Fusions-Spec (nächste freie Nummer, vor dem Eintragen nachzulesen; zuletzt
  vergeben: #26): das Feature und die Reihenfolge „Welle erst nach Schonfrist“. Danach die
  Zeile 4e in `docs/en|de/migration.md` (Abhängigkeit); `internal/plancheck` muss grün
  bleiben. Eine Roadmap-Zeile gibt es nicht: Plan und Code kommen in einem Pull Request, die
  Zeile entstünde und verschwände im selben; die Fähigkeit steht in der Doku.
- Label `release:minor`: neues Feature, ohne `armed.toml` ändert sich kein Verhalten.

## Nicht Teil dieses Entwurfs

- Eine Frist, nach der Lanes von selbst scharf werden.
- Schonfrist für Wächterregeln (`[policy]`) oder die Commit-Sprache; sie gilt nur für die Lanes
  von `[verify]` und `lint/wiki`.
- Ein nicht blockierender Kanal vom Stop-Hook zum Agenten.
- Die Schonfrist für loomux selbst: `ci/gate.sh` und `.githooks/pre-commit` dieses Repos
  bleiben, wie sie sind. Wer sie später will, legt die Datei mit `gate disarm --all` an und
  baut den Hook dann um.

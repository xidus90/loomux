# Paritätsakte Stufe 4a-2 — `loomux init`

Plan: `plans/2026-09-24-loomux-stufe-4a-2.md`. Specs:
`specs/2026-09-14-loomux-fusion-design.md` und
`specs/2026-09-23-loomux-stufe-4-design.md` (Abschnitt 4a-2). Zweig
`feat/stage-4a-2-init`, gebaut vom 2026-09-24 ab `460a6c13` (Merge-Basis
mit `master`: `3453354e`).

## Referenzen

| Was | Referenz | Wohin in loomux |
|---|---|---|
| Installer `ulinit` | `ultraloom` am Tag `loomux-1a-source` = `9d01a60`, `cmd/init` und seine Pakete | `loomux init`, `internal/setup/...` |
| Anlegen ohne Überschreiben | `internal/write/atomic.go` samt Tests | `internal/setup/write` (Umzug) |
| Zusammenführen der Host-Datei | `internal/settings/merge.go` samt Tests | `internal/setup/hostfile/merge.go` (Umzug, ohne Marke) |
| `AGENTS.md` und `verify-until-green` | `internal/render/templates/` | `internal/setup/templates/files/` |
| Die fünf Brain-Skills | ultra-brain `loomux-3-source` = `3cc72d2`, `.claude/skills/*/SKILL.md` | `internal/setup/templates/files/skills/`, ins Englische übersetzt |
| post-merge-Hook | ultra-brain `loomux-3-source`, `src/brain/maintenance/merge_events.py`, `_hook` in `src/brain/cli.py` | `loomux merge-hook`, `internal/brain/maintenance/mergehook.go` |
| Erstinstallation | keine; `selfupdate.Run` konnte nicht zum ersten Mal installieren (Plan, B5) | `selfupdate.Install` |

Außer dem post-merge-Hook hat kein Teil eine aufgenommene Referenz: Der
größte Teil von `init` ist neu (Plan, B1) und wird über Einheits- und
Golden-Tests gehalten, nicht über eine Fallsuite.

## Entscheidungen, freigegeben am 2026-09-24

Alle fünf wie vorgeschlagen.

| # | Frage | Entscheidung |
|---|---|---|
| E1 | Umfang des Umzugs | Nur `write` und das Zusammenführen aus `settings` ziehen mit ihren Tests um. `interview`/`answers` werden auf `tui` und Schema neu geschrieben; `render` bleibt als zwei Vorlagen. `commit`, `coverage`, `verify`, `tomlstr`, `tooling` und `ulinit check …` fallen weg (siehe „Was wegfällt“) |
| E2 | Was in `.loomux/state/answers.toml` steht | Nur, was kein Schlüssel des Schemas ist: gewählte Hosts und je Modul die gewählten Teile. Alles andere steht in `.loomux/config.toml` |
| E3 | Besitz eines Host-Eintrags | Ein Eintrag gehört `init`, wenn sein Befehl ein loomux-Binary ruft (`loomux`, `loomux.exe`, `bin/loomux.exe`, der kanonische Pfad); keine Marke. Fremde Einträge bleiben stehen, auch `ulguard` und `brain guard`, und werden gemeldet. Vor dem ersten Umschreiben ein `.bak`; eine Datei, die kein JSON ist oder deren `hooks` kein Objekt ist, wird nicht angefasst |
| E4 | Zustände von `merge-hook status` | `installed`, `missing`, `not installed`, `unrecorded`, `orphaned` und `refused` wie in der Referenz; `stale path`, `stale branch` und `shared hook path` entfallen, weil nichts eingebacken ist. `record` prüft Common-Dir und Zweig gegen die einwilligenden Bereiche der Registry |
| E5 | `.mcp.json` | Befehl `${LOCALAPPDATA}/loomux/bin/loomux.exe` mit `mcp --channel local`, statt `loomux` über den `PATH`; nur, wenn der Nutzerbereich keinen Server `loomux` kennt. Die Bestätigung durch die Messung von Task 1 steht aus (siehe „Offen“) |

## Beim Planen gesetzt

Der Mensch sah sie mit dem Plan; gebaut wie gesetzt.

- **pre-push** eines Wirts verweigert einen Push nach `main` oder `master`,
  wie `.githooks/pre-push` dieses Repos (`gitfiles.Hooks`,
  `TestPrePushRefusesTheDefaultBranches`).
- **Sicherungen** liegen unter `.loomux/state/backup/<pfad>.bak`, nicht neben
  der Datei; `.gitignore` braucht nur `/.loomux/state/`
  (`TestAnExistingFileIsBackedUpOnceBeforeItChanges`).
- **Der Hook der Referenz gilt als eigener:** `merge-hook install` ersetzt
  eine Datei mit `# brain post-merge hook`, statt sie zu verweigern
  (Abweichung 3 im Abschnitt zum post-merge-Hook).
- **Der Wächter** verweigert einem Agenten `merge-hook install` und `remove`,
  wie `area add` (`internal/hooks/guard.go`, `wordsWriteConfiguration`).
  `status`, `record` und ein bares `merge-hook` bleiben erlaubt.
- **`[verify]`** schreibt `init` nicht; es fragt nach keiner Abweichung vom
  Preset.
- Ohne Frage gesetzt, weil Spec oder Code es festlegen: Erstinstallation als
  `selfupdate.Install` neben `Run`; Mehrfachauswahl als `tui.Pick`; scheitert
  der Binary-Schritt, entstehen keine Host- und Git-Hook-Einträge, die das
  Binary rufen (`TestAMissingBinaryDropsWhatCallsIt`,
  `TestInitWritesNoHookEntryWhenTheBinaryIsMissing`); die Flags `--root`,
  `--dry-run`, `--detect-only`, `--yes`, `--hooks|--brain|--graph=all|each|none`,
  `--hosts=…`.

## Entscheidungen beim Bau, 2026-09-24

Getroffen vom Orchestrator während der Tasks, jede mit ihrem Preis, falls
sie falsch ist.

1. **Antigravity bekommt noch keine Host-Einträge.** `Entries(HostAntigravity,
   …)` gibt nil zurück, solange die Konstante `antigravityMeasured` in
   `internal/setup/hostfile/table.go` falsch ist; die Messung von Task 1
   steht aus. Preis: Antigravity-Wirte bekommen nichts, bis jemand die
   Konstante umlegt. *Überholt am 2026-09-25, siehe „Antigravity-Einträge“.*
2. **Antigravity bekommt noch keine Skills.** `Skills(…, HostAntigravity)`
   gibt nil zurück, bis Task 1 den Skill-Ort nennt. *Überholt am
   2026-09-25.*
3. **`Answers` und `ReadAnswers` entstanden in Task 11** statt in Task 12,
   weil Task 11 sie zuerst braucht; Task 12 fügte das Schreiben hinzu.
4. **`.mcp.json` nach E5 ohne die Messung.** E5 ist freigegeben, die Messung
   bestätigt nur. Preis: `mcpjson.go` wird neu geschrieben, falls Claude Code
   `${LOCALAPPDATA}` dort nicht auflöst.
5. **Die Doppelung von `run` und `Install` wurde sofort behoben**
   (`installLocked` in `internal/selfupdate/update.go`), nicht aufgeschoben.
6. **Die Zeilenanordnung von `List` und `Pick` teilen einen Helfer**
   (`internal/tui/list.go`), sofort behoben.
7. **Die Übernahme des alten Zustandsverzeichnisses entfällt.** Eine
   `hooks.tsv` von ultra-brain wird nicht gelesen; die Übernahme wird ein
   Punkt der 4e-Liste (`migrate` entfällt). Preis: Ein von `brain-mcp`
   eingerichteter Hook bleibt `unrecorded`, bis ihn jemand neu einrichtet
   (Abweichung 5 unten).
8. **`merge-hook` ist in einem Checkout von loomux aus.** `merge-hook
   install` schriebe eine eingecheckte `.githooks/post-merge` und ließe
   `git status` unsauber. Preis: Ein Mensch schaltet ihn im Dialog an.
9. **`Choice.Modules` erweitert die vereinbarte Schnittstelle**, damit ein
   Checkout nicht `graph = false` geschrieben bekommt, nur weil er den Graphen
   nicht bei jedem `init` baut.
10. **Lebende Hooks in `.git/hooks` behalten ihr Verzeichnis.** Ist
    `core.hooksPath` nicht gesetzt und liegt in `.git/hooks` ein lebender
    `pre-commit`, `pre-push` oder `commit-msg` (kein `*.sample`), plant
    `init` kein `hooks-path`, schreibt nur die fehlenden Hooks dorthin und
    nennt die vorhandenen. „Nie überschreiben, stattdessen melden“ verbietet
    auch das Abschalten. Preis: Solche Projekte bekommen ihre Hooks in
    `.git/hooks` statt in `.githooks`.
11. **Ein leeres `Choice.CommitLanguage` gilt als `en`.**
12. **Git-Hooks eines verknüpften Worktrees oder Submoduls bleiben unberührt.**
    Das Hookverzeichnis ist, was `git rev-parse --path-format=absolute
    --git-path hooks` nennt (`Facts.GitHooksDir`). Liegt es außerhalb der
    Wurzel, plant `init` weder `hooks-path` noch Hooks dort, nur eine Notiz:
    `core.hooksPath` zu setzen schaltete die geteilten Hooks jedes Worktrees
    ab. Preis: Ein Worktree mit lebenden geteilten Hooks bekommt keine
    loomux-Git-Hooks, bis ein Mensch sie einträgt.
13. **`within()` vergleicht über `os.SameFile` aufwärts** statt über
    `EvalSymlinks`; gemessen: `EvalSymlinks` lässt Junction-Pfade unter
    go1.27/Windows stehen.
14. **Eine Handlung wird nur geplant, solange ihr Ergebnis fehlt.**
    `binary-install` nur ohne kanonisches Binary (Updates bleiben bei
    `serve`/`self-update`), `binary-build` nur ohne `<root>/bin/loomux.exe`,
    `merge-hook` nur ohne eigene `post-merge` im Hookverzeichnis,
    `graph-build` nur ohne Graphdatei. Preis: Ein veraltetes Binary oder ein
    veralteter Graph wird von `init` nicht erneuert (das tun `self-update`
    und das pre-commit-Tor).
15. **Die Vorgabe des Interviews je Modul folgt der Wahl**: alle Teile an →
    `all`, alle aus → `none`, sonst `each` mit vorbelegtem `Pick`; ein
    ausdrückliches Flag gewinnt. Der Plan sagte „oder `all`“, was die Teile
    und die Regel der Selbstnutzung unterlaufen hätte.
16. **Abbruch mit `esc`/EOF im Interview oder bei der Bestätigung endet mit 0**
    und „nothing written“: Die Constraints zählen eine verweigerte Bestätigung
    zu 0. Preis: Ein Skript unterscheidet Abbruch und Erfolg nicht am
    Exit-Code.

## Der post-merge-Hook: `brain-mcp hook` → `loomux merge-hook`

Referenz: ultra-brain `loomux-3-source` = `3cc72d2`,
`src/brain/maintenance/merge_events.py` und `_hook` in `src/brain/cli.py`,
aufgenommen mit `brain-mcp.exe` aus dem `.venv` des Checkouts, der auf dem
Tag steht. Aufnahme: `stufe-4a-2-orakel/record_all.sh` über `record.sh`
daneben, in der Umgebung von 3a (`BRAIN_STATE_DIR` ist die gestufte Welt,
`LOCALAPPDATA`, `XDG_STATE_HOME`, `XDG_CACHE_HOME` zeigen in eine Sandbox,
`XDG_CONFIG_HOME` in die Welt, Git über `cases.GitEnv`). Fälle:
`testdata/cases/4a2/`, Suite `internal/cli/cases_4a2_test.go`, 14 Fälle,
elf ohne Unterschied.

Der Name wechselt, weil `hook` in loomux der Namensraum der Host-Hooks ist
(`loomux hook pre-tool-use …`). Die Zeilen der Ausgabe sind dieselben:
`<zustand>: <scope> — <repo>` und ` [<datei>]`, wenn es eine gibt; Exit 1
bei `refused` oder `no repository`, sonst 0. `record` ist neu (siehe unten).

### Abweichungen

1. **Der Hook backt nichts ein (E4).** Die Referenz schrieb das
   Common-Dir des Repos, den Zweig und den Pfad der Ereignisdatei in die
   Hookdatei und prüfte sie dort in Shell. loomux' Hook ruft nur
   `loomux merge-hook record`, das zur Merge-Zeit die Registry fragt; die
   Datei ist überall derselbe Text. Damit entfallen `stale path`,
   `stale branch` und `shared hook path` (nichts kann veralten, und zwei
   Bereiche hinter einem Hookverzeichnis wollen dieselbe Datei) sowie
   `unsafe branch` (kein Zweigname landet mehr in Shell). Grund: Ein
   eingebackener Hook sagt nach jeder Änderung am Manifest oder am
   Zustandsverzeichnis etwas anderes als das Manifest, und die Referenz
   brauchte vier Zustände, um das zu melden.
2. **`hooks.tsv` hat drei Felder statt fünf.** `scope\trepo\thook`; Zweig
   und Ereignispfad der Referenz gibt es nach 1 nicht mehr. Eine Zeile der
   Referenz mit vier oder fünf Feldern wird über ihre ersten drei gelesen.
   Die Suite vergleicht `maintenance/hooks.tsv` deshalb über die ersten drei
   Felder jeder Zeile (`normalize4a2`); alles andere an der Datei zählt.
3. **Die Marke der Referenz gilt als eigene.** Eine Hookdatei mit
   `# brain post-merge hook` ist der Vorgänger desselben Hooks am selben
   Ort: `install` ersetzt sie, `status` nennt sie `unrecorded` statt
   `not installed`, `remove` nimmt sie zurück. Belegt durch
   `hook/install-own-earlier` und `hook/status-own-earlier`, beide ohne
   Unterschied. 4e stellt die Wirte um.
4. **`status` nennt die Hookdatei auf jeder Zeile, die eine kennt.** Die
   Referenz druckt bei `installed`, `missing`, `orphaned` und
   `not installed` keine; loomux druckt ` [<datei>]` auch dort, weil die
   Datei in einem Repo liegt, das der Nutzer nie genannt hat. Fall
   `hook/status-orphaned` (stdout 93 gegen 178 Byte), und in
   `hook/status-installed` zusammen mit 6.
5. **Das Zustandsverzeichnis der Referenz wird nicht gelesen.** Eine
   `hooks.tsv` unter dem alten Verzeichnis von ultra-brain bleibt liegen;
   gelesen und geschrieben wird nur unter `LOOMUX_STATE_DIR`. Ein von
   `brain-mcp` eingerichteter Hook ist für `status` bis zu einer neuen
   Einrichtung `unrecorded` statt `installed`. Entschieden in Task 6 (die
   Übernahme wird ein Punkt der 4e-Liste, `migrate` entfällt).
6. **Prüfstand: der Repo-Pfad als Text.** `hook/status-installed` läuft in
   einer Welt, die die Einrichtung der Referenz nachstellt. Die gestufte
   `hooks.tsv` kann den Pfad nur mit Schrägstrichen schreiben
   (`{{WORLD}}` wird so ersetzt); die Referenz vergleicht ihn als Text mit
   `str(Path)` von Gits Antwort, also mit Rückstrichen, und nennt ihre
   eigene Einrichtung `orphaned`. loomux vergleicht Pfade als Pfade und
   sagt `installed`. Kein Unterschied im Verhalten: Einen Eintrag, den die
   Referenz selbst schreibt, schreibt sie mit Rückstrichen.
7. **Prüfstand: die Schreibweise der Hookdatei beim Entfernen.**
   `hook/remove-installed` gleicher Größe, anderer Inhalt: Die Referenz
   druckt die Hookdatei aus dem Eintrag über `Path`, also mit
   Rückstrichen, loomux druckt den Eintrag, wie er steht, hier die
   gestuften Schrägstriche. Ein Eintrag, den eines der beiden Werkzeuge
   schreibt, trägt schon die Schreibweise der Plattform.
8. **`no repository` nennt den Bereich in der Schreibweise der Plattform.**
   Wie die Referenz (`str(Path)`); die Registry schreibt Schrägstriche.
   Beim Aufnehmen gefunden und in `targetOf` angeglichen, darum ohne
   Unterschied in `hook/install-no-repository`.
9. **`loomux merge-hook record` ist neu.** Die Referenz hatte kein
   Gegenstück; ihr Hook schrieb die Zeile selbst. `record` arbeitet im
   Arbeitsverzeichnis, schreibt nie etwas auf stdout oder stderr und endet
   immer mit 0: keine Registry, eine kaputte Registry, kein Repo, ein
   Zustandsverzeichnis, das sich nicht schreiben lässt, und überzählige
   Argumente eingeschlossen. Ein Merge darf an der Buchführung weder
   scheitern noch etwas ausgeben. Durch Einheitstests belegt
   (`internal/cli/mergehook_test.go`), nicht durch Fälle: die Referenz hat
   nichts, gegen das sich aufnehmen ließe.
10. **Die Meldung ohne Zeile ist englisch.** Die Referenz sagt
    `kein Bereich mit …, kein eingerichteter Hook`, loomux
    `no area consents with [maintenance] on_merge = true, and no hook is
    installed`. Die fünf Fälle, in denen sie steht, vergleichen darum nur
    den Exit (`compare = message`).
11. **Eine fehlende Registry ist eine leere.** Ohne `registry.toml` enden
    `install`, `status` und `remove` mit dem Satz aus 10 und Exit 0; die
    Referenz wirft dort einen `FileNotFoundError` (`read_registry` kennt
    keinen Zweig für eine fehlende Datei) und endet mit 1. Grund: `init`,
    `record` und der Upkeep lesen eine fehlende Registry schon als leer, und
    auf einem Rechner, der nie einen Bereich angelegt hat, ist nichts
    eingerichtet. Eine Registry, die sich nicht lesen lässt, bleibt Exit 1.
    Gefunden am 2026-09-24 von einem unabhängigen Test (Gemini); belegt
    durch `TestMergeHookReadsAMissingRegistryAsEmpty`.

### Was die Fälle nicht zeigen

- `missing` (Eintrag ohne Datei) und `unrecorded` für eine Datei mit
  loomux' Marke sind nur durch die Einheitstests von Task 6 belegt
  (`TestStatusNamesEveryState`).
- `core.hooksPath`: Die Fälle schreiben die Hookdatei nach `.git/hooks`;
  `git.toml` kennt keine Git-Konfiguration. Dass `install` dem
  konfigurierten Pfad folgt, belegt
  `TestInstallHonoursACoreHooksPath` in `internal/brain/maintenance`.
- Eine Registry, die fehlt oder nicht liest: Kein Fall. loomux meldet sie
  auf stderr und endet mit 1 (`TestMergeHookFailsOnARegistryItCannotRead`);
  wie die Referenz dort endet, ist nicht aufgenommen.
- Keine Aufnahme vergleicht eine Hookdatei: Alle liegen unter `.git/`, das
  Aufnahme und Nachspielen überspringen. Überschreiben, Verweigern und
  Entfernen der Datei selbst belegen nur die Einheitstests von Task 6.

## `ulinit` → `loomux init`: Abweichungen

### Die Mängel der Referenz, die nicht mitziehen (Plan, B2)

1. **`--dry-run` konnte Werkzeuge installieren** (`run.go:207-249`).
   `init` installiert nie fremde Software, auch nicht mit `--yes`: Der Teil
   `tools` sucht `git`, `qmd`, `pdftotext`, `yt-dlp` und `ollama` mit
   `exec.LookPath` und nennt für ein fehlendes den Installationsbefehl als
   Notiz (`internal/setup/tools.go`). `--dry-run` endet nach der
   Zusammenfassung, bevor `Apply` läuft (`internal/cli/init.go`,
   `TestInitDryRunShowsEveryChangeAndWritesNothing`).
2. **Installationen liefen über die `git`-Hülle, die Exit 1 ohne Ausgabe als
   Erfolg nahm** (`main.go:185`). Entfällt mit 1. Das einzige, was `init`
   holt, ist das eigene Binary, über `selfupdate.Install` (`gh release
   download`, `SHA256SUMS`, `--version` des geholten Binarys, `swap.Swap`);
   dessen Fehler lässt den Teil `binary` scheitern, und die Einträge, die das
   Binary rufen, entfallen.
3. **`--tool-path` wurde verworfen** (`:190-205`). Es gibt kein solches Flag;
   Werkzeuge werden auf dem `PATH` gesucht.
4. **`core.hooksPath`, `.githooks/commit-msg` und `.gitignore` entstanden
   nach dem Schreiben** und fehlten in `--dry-run` und `installed.toml`. In
   `init` sind sie Teil des Plans: die Handlung `hooks-path`, die Änderungen
   `.githooks/commit-msg` und `.gitignore`. `--dry-run` zeigt sie,
   `installed.toml` nennt sie unter `files` und `actions`
   (`TestApplyWritesInOrderAndStateLast`).
5. **`installed.toml` wurde nie aktualisiert.** `init` schreibt es bei jedem
   Lauf, der eine Datei schreibt oder eine Handlung ausführt, neu und als
   letzte Datei; ein Lauf, der nichts tut, lässt es stehen
   (`TestApplyWritesInOrderAndStateLast`, `TestSkippedFilesAndAQuietRun`).
   Bricht ein Lauf ab, fehlt es, und der nächste Plan zeigt das Offene
   wieder (`TestAnInterruptedRunIsOpenOnTheNextOne`).
6. **`answers.toml` schlug die Flags.** In `init` gewinnen die Flags:
   `DefaultChoice` liest `answers.toml` zuerst, danach setzen `--hosts` und
   `--hooks|--brain|--graph=all|none` die Wahl (`initCommand` in
   `internal/cli/init.go`). `answers.toml` hält nach E2 nur noch Hosts und
   Teile.

### Weitere Abweichungen

7. **Besitz ohne Marke (E3).** ulinit erkannte eigene Blöcke an
   `"ultraLoomOwned": true`, ersetzte sie und formatierte die Datei neu.
   `init` erkennt einen eigenen Block daran, dass sein erster Befehl ein
   loomux-Binary ruft (`hostfile.Owned`), ersetzt ihn nie (ein eigener Block
   auf dem Platz heißt `Kept`) und lässt die Datei byte-gleich, wenn nichts
   dazukommt (`TestMergeLeavesLoomuxsOwnSettingsByteForByte`). Ein fremder
   Block auf demselben Ereignis und Matcher bleibt und wird gemeldet, der
   eigene kommt daneben.
8. **Die Tabelle der Einträge folgt loomux, nicht ulinit** (Plan, B9):
   Matcher mit `MultiEdit`, Timeouts 15/60/300/20/30 statt 10, damit
   `init --dry-run` auf loomux nichts ändert. Die Reihenfolge beim
   Neuschreiben bleibt die von ulinit (`SubagentStart`/`SubagentStop` vor
   `Stop`); loomux' eigene Datei hat `Stop` davor. Eine frische
   Installation ordnet die Ereignisse also anders als loomux' Datei; die
   wird nie umgeschrieben, weil nichts dazukommt.
9. **`Owned` zerlegt den Befehl mit `shellwords`, nachdem jeder Rückstrich
   verdoppelt ist**; sonst würde aus `C:\x\loomux.exe` `C:xloomux.exe`.
   Nebenwirkung: Ein mit Rückstrich maskiertes Leerzeichen trennt jetzt
   Wörter.
10. **`formatRoot` hat keinen Rückfall mehr für einen `json.Indent`-Fehler**:
    Die Eingabe stammt immer vom Decoder oder aus `json.Marshal`.
11. **`AGENTS.md` kennt nur die Commit-Sprache.** Die Vorlage der Referenz
    zählte Commit-Nachrichten zu „immer Englisch“ und hatte eine
    `DocsLanguage`; `templates.Vars` hat nur `CommitLanguage`, und
    Commit-Nachrichten stehen nicht mehr in der Englisch-Liste. Neu ist der
    Abschnitt „Configuration“: Nur ein Mensch schreibt
    `.loomux/config.toml`, ein Agent schlägt mit
    `loomux config set … --propose` vor.
12. **Die fünf Brain-Skills sind englisch** und rufen loomux-Befehle
    (`loomux brain catalog|read|search|check`, `loomux approve`,
    `loomux case|cases`); die Tabelle der Umschreibungen steht im Bericht
    von Task 10. `TestEveryLoomuxCommandInASkillExists` prüft jeden Befehl
    gegen `commands`, `TestNoSkillCallsAnOldCommand` die alten Namen.
    `session-handover` fällt weg (Spec #9).
13. **Ein Checkout von loomux baut sein Binary**, statt es zu installieren:
    Seine Einträge rufen `${CLAUDE_PROJECT_DIR}/bin/loomux.exe`, seine
    Git-Hooks `./bin/loomux.exe` (Git setzt `${CLAUDE_PROJECT_DIR}` nicht).
    `binary-build` in einem Projekt, das kein Checkout ist, endet mit einem
    klaren Fehler statt einem `go build`.
14. **Eine neue `.loomux/config.toml` entsteht über `write.Prepare`/`Commit`**
    (exklusives Anlegen), eine vorhandene über `lock.ReplaceText`; beide
    erst, wenn `schema.Validate` den Text annimmt. Die Constraint „nur über
    `lock.ReplaceText`“ gilt damit für das Umschreiben, nicht für das
    Anlegen.
15. **`--dry-run=…` und `--detect-only=…` sind ein falscher Aufruf** (Exit 2):
    Der Wächter lässt jede Zeile mit dem Wort `--dry-run` durch, ein
    späteres `=false` hätte das zurückgenommen. Aus demselben Grund nimmt
    `--root` kein folgendes `--dry-run` als Wert.
16. **Das Wiki folgt der Vorgabe von `area add`, nicht der Spec-Tabelle**
    (`[layout].wiki = docs/wiki`): `init` übergibt kein `--wiki`, weil das
    Flag nur das Bündel verschiebt und `[layout] wiki` der Konfiguration vom
    Registry-Eintrag trennen würde; die Freigabe der Handlung `area-add`
    nennt, was `area add` nimmt (`docs/wiki` mit `docs/`, sonst `wiki`).

### Was wegfällt (E1)

| Paket von ulinit | Grund |
|---|---|
| `detect`, `gitenv` | gibt es in loomux (`internal/detect`, `internal/gitenv`) |
| `commit` | Englisch-Wortliste; loomux hat `check commit-msg` mit `[commit]` |
| `coverage`, `verify`, `ulinit check …` | loomux hat `check coverage` und `check lint`; `check types` (dmypy) hat keinen Nutzer |
| `tomlstr`, `vendoring`, `brainpath` | nur für die Vorlagen von `brain.toml`, `policy.toml` und `.ultraloom/config.toml`, die wegfallen |
| `answers`, `interview` | neu geschrieben: Schema statt eigener Felder (E2), `tui` statt `bufio` |
| `tooling` | installierte über winget/curl; `init` prüft nur (siehe 1) |
| `render` bis auf zwei Vorlagen | übrig sind `AGENTS.md.tmpl` und `verify-until-green` |

### Die umgezogenen Tests

**`internal/write` → `internal/setup/write`** (Task 4): beide Dateien
unverändert bis auf Namen (`ulinit` → `loomux init`, `.ultraloom/` →
`.loomux/`); kein Test entfiel.

**`internal/settings/merge_test.go` → `internal/setup/hostfile/merge_test.go`**
(Task 8):

| Test der Referenz | In loomux |
|---|---|
| `TestAMissingEventIsAdded` | entfallen; `TestMergeAddsBesideOldEntriesAndNamesThem` deckt es ab |
| `TestOurOwnEntryIsReplacedNotDuplicated` | umgeschrieben: `TestMergeKeepsAnOwnBlockWithAnotherCommand` (Kept statt Ersetzen, keine Marke) |
| `TestAForeignEntryIsLeftAloneAndReported` | umgeschrieben: `TestMergeReportsAForeignHookOnTheSameSlotAndAddsBesideIt` (Foreign statt Skipped, der eigene Eintrag kommt daneben) |
| `TestForeignKeysSurvive` | umgeschrieben: `TestMergeKeepsForeignTopLevelKeysInTheirOrder` |
| `TestBrokenJsonIsRefused` | umgeschrieben: `TestMergeRefusesWhatItCannotRead` (Tabelle, Dateiname im Fehler) |
| `TestAMatcherlessEntryDoesNotClaimEveryEntry` | übernommen; prüft Foreign leer und ein Added |
| `TestOurOwnMatcherlessEntryIsStillReplaced` | entfallen; Ersetzen gibt es nicht mehr |
| `TestHooksOfTheWrongShapeAreRefusedNotOverwritten` | aufgegangen in `TestMergeRefusesWhatItCannotRead` |
| `TestAnEventOfTheWrongShapeIsRefusedNotOverwritten` | aufgegangen in `TestMergeRefusesWhatItCannotRead` |
| `TestAFileWithoutHooksStaysWithoutThem` | entfallen; ohne Added ist die Ausgabe die Eingabe (`TestMergeLeavesLoomuxsOwnSettingsByteForByte`) |
| `TestAnEmptyFileBecomesOneWithJustOurHook` | übernommen |
| `TestATimeoutOfZeroIsLeftOut` | übernommen |
| `TestANullFileIsTreatedAsAnEmptyOne` | übernommen |
| `TestAnEntryThatIsNotAnObjectIsIgnoredForTheLookup` | übernommen |
| `TestOursFirstOnASharedSlotKeepsTheSlot` | entfallen; eigener und fremder Block auf einem Platz prüft der zweite Merge in `TestMergeReportsAForeignHookOnTheSameSlotAndAddsBesideIt` |
| `TestMultipleOurOwnEntriesUnderSameEventAndMatcherAreAddedAndReplaced` | entfallen; die Tabelle hat je Platz einen Eintrag, ein zweiter wäre Kept |
| `TestToolKeyExtraction`, `TestToolKeyReadsUlguardHooks` | entfallen; `toolKey` ist durch `Owned` ersetzt (`TestOwnedKnowsEveryLoomuxBinary`) |
| `TestFirstCommandEdgeCases` | übernommen |
| `TestMergePreservesExactLifecycleOrder` | umbenannt: `TestMergePreservesTheLifecycleOrder`, um ein fremdes Ereignis erweitert |
| `TestHookBlockAndCommandKeyOrder` | übernommen |
| `TestFormatBlockAndCommandFallbacks` | übernommen, mit einem Eintrag (sonst kein Neuschreiben) |
| `TestEmptyHooksObject` | entfallen; `{"hooks":{}}` ohne Added bleibt byte-gleich |
| `TestTopLevelKeyOrderPreserved` | aufgegangen in `TestMergeKeepsForeignTopLevelKeysInTheirOrder` |
| `TestExtractTopEntriesAndFormatRootEdgeCases` | übernommen |
| `TestOrderHooksAndFormatBlockEdgeCases` | übernommen |

## Selbstnutzung

### Agent, 2026-09-24

`bin/loomux.exe init --dry-run` in diesem Worktree
(`.claude/worktrees/beautiful-noether-local-c6dd1a` auf `309f592b`), Exit 0:

```
nothing to change
notes:
  yt-dlp is not on PATH; install it with: winget install --id yt-dlp.yt-dlp -e
  git-hooks: skipped; core.hooksPath C:\Users\micro\Documents\#GIT\loomux\.githooks lies outside the project
```

- **Keine Änderung** an `.claude/settings.json` und `.loomux/config.toml`,
  wie verlangt. `.githooks/` prüft dieser Lauf nicht: Der Teil `git-hooks`
  entfällt hier (nächster Punkt). Dass ein Checkout dort nichts ändert,
  belegen nur `TestSelfUseChangesNothing` und der offene Schritt mit dem
  frischen Klon.
- **Abweichung vom Plan:** Er erwartete `binary-build` als Handlung.
  Nach Entscheidung 14 plant `init` eine Handlung nur, solange ihr Ergebnis
  fehlt; `bin/loomux.exe` liegt da, also keine. `TestSelfUseChangesNothing`
  prüft den Fall ohne Binary (dort `actions == [binary-build]`).
- **Die Hook-Notiz stammt vom Worktree.** Ein verknüpfter Worktree teilt die
  Konfiguration des Hauptcheckouts, und dessen `core.hooksPath` zeigt
  absolut auf `.githooks` des Hauptcheckouts, also außerhalb dieser Wurzel
  (Entscheidung 12). Im Hauptcheckout selbst stünde sie nicht.
- Die Notiz zu `yt-dlp` gilt dieser Maschine.
- Der Wächter verweigerte denselben Aufruf in der Form
  `cd "<worktree>" && bin/loomux.exe init --dry-run`, ohne `$` hinter den
  Flags; der Pfad enthält `#`. Siehe „Offen“.

### Mensch — offen

Noch nicht gelaufen (Plan, Task 15, Schritte 2 bis 4): ein frischer Klon mit
`init` und `--yes`, danach `git config core.hooksPath`, `ls bin/loomux.exe`,
ein zweites `init --dry-run` ohne Änderung und ein leeres
`git status --porcelain`; dann ein Wirt nach Wahl mit `loomux init`
interaktiv, in dem `pre-tool-use` in einer neuen Claude-Code-Sitzung einen
Push verweigert. Ergebnisse mit Datum hier eintragen.

## Mutationsrunde

**Die Runde mit `loomux dev mutants` (2026-09-24).** Gefahren mit einer
Kopie des Binärs von `309f592b` im Scratchpad, `LOOMUX_STATE_DIR` und
`LOOMUX_LEGACY_BRAIN_DIR` im Scratchpad der Sitzung, acht Arbeiter. Umfang
nach dem Plan: `internal/setup`, `internal/setup/hostfile` und
`internal/setup/gitfiles` ganz, `internal/brain/maintenance` nur
`mergehook.go` (`--only mergehook`), `internal/selfupdate` nur `install.go`
(`--only install`). Die erste Runde lief gegen den Baum ohne neue Tests
(zusammen 7:40 min), die zweite mit ihnen (7:57 min), eine dritte nur über
`mergehook.go` nach dem letzten Test dort.

| Paket | erzeugt | nicht übersetzbar | erste Runde: getötet / überlebt | letzte Runde: getötet / stehen |
|---|---:|---:|---:|---:|
| `internal/selfupdate` (`install.go`) | 8 | 3 | 5 / 0 | — |
| `internal/brain/maintenance` (`mergehook.go`) | 168 | 30 | 130 / 8 | 134 / 4 |
| `internal/setup/hostfile` | 180 | 55 | 112 / 13 | 121 / 4 |
| `internal/setup/gitfiles` | 30 | 2 | 24 / 4 | 28 / 0 |
| `internal/setup` | 438 | 75 | 335 / 28 | 358 / 5 |

**Nicht in der Runde**, weil der Plan sie nicht nennt: `internal/setup/write`
(Umzug mit den Tests der Referenz, unverändert bis auf Namen),
`internal/setup/templates` (eingebetteter Text hinter drei dünnen
Funktionen), `internal/tui/pick.go`, `installLocked` in
`internal/selfupdate/update.go` (der geteilte Teil von `Run` und `Install`,
Entscheidung 5 — `install.go` allein hat darum nur acht Mutanten), der
Wächter in `internal/hooks/guard.go` und in `internal/cli` die Dateien
`init.go`, `init_ui.go` und `mergehook.go`. Eine Runde über `cli` hätte nach
der Erfahrung von 4a-1 ein Vielfaches gebraucht.

**Getötet durch neue Tests.**

- `setup` (`internal/setup/mutants_test.go`): ein ausgeschalteter Teil plant
  nichts — `config`, `gitignore`, `tools`, `area` (`plan.go:88`, `:94`,
  `:105`, `:122`; `TestAPartSwitchedOffPlansNothing`); der Plan nennt seinen
  ersten Fehler (`plan.go:158`); ohne Repo und ohne Git-Teil keine Notiz
  (`plan.go:230`); ein `core.hooksPath` innerhalb der Wurzel ohne Notiz
  (`plan.go:291`); unser post-merge in `.git/hooks` zählt als da, auch ohne
  `hooks-path` (`plan.go:272`, zwei Formen, und `facts.go:170`); außerhalb
  eines Repos fragt `Gather` Git nicht (`facts.go:100`) und trägt Gits
  Fehler weiter (`facts.go:141`); ohne Stack ist `graph-build` keine
  Vorgabe (`parts.go:45`); `mcpServers` als erster Schlüssel bleibt einer
  (`mcpjson.go:63`); ein Modul, das an und auf `true` gesetzt ist, behält
  seine Zeile (`configtext.go:79`); ein gescheitertes Edit wird vom
  nächsten nicht überschrieben (`configtext.go:84`); eine andere Regel in
  derselben Tabelle ist nicht die des Katalogs (`configtext.go:129`,
  `:133`); eine verweigerte Konfiguration wird einmal genannt
  (`configtext.go:143`); ein gescheiterter `binary-build` ist ein
  gescheiterter Binary-Schritt, kein Abbruch (`apply.go:52`); nichts wird
  durch einen verlinkten Elternordner geschrieben, weder eine Änderung
  (`apply.go:140`) noch der Zustand (`state.go:66`, `:69`). Mit diesen drei
  Mutanten schreibt `init` durch eine Junction außerhalb des Projekts und
  meldet Erfolg; `TestApplyWritesNothingThroughALink` und
  `TestTheStateIsNotWrittenThroughALink` halten es jetzt fest.
- `hostfile` (`merge_test.go`): `TestFirstCommandOfAnEmptyHookList`
  (`merge.go:140`, sonst ein Index jenseits des Endes),
  `TestAMatcherlessEntryWritesNoMatcher` (`:162`),
  `TestMergeWritesEveryTopLevelKeyOnce` (`:241`, `:247` drei Formen — sonst
  zwei `"hooks"` in einer Datei), `TestOrderHooksWritesOnlyTheEventsThereAreOnce`
  (`:278`, `:285`), `TestFormatBlockIndentsHooksThatAreNoList` (`:388`).
- `gitfiles` (`gitfiles_test.go`): `TestPrePushRefusesTheDefaultBranches`
  prüft jetzt stderr (`gitfiles.go:18`: `>=&2` schrieb die Meldung in eine
  Datei `=` im Paketverzeichnis); `TestHooksTakeOffOnlyAPairOfQuotes`
  (`:31`, drei Formen).
- `maintenance` (`mergehook_test.go`): `TestARepositoryGitCannotNameIsNoRepository`
  (`mergehook.go:126` — ohne die oberste Ebene ging der Hook mit leerem
  Repo hinein); `TestInstallFailsOnAHookItCannotReplace` (`:367`, drei
  Formen — ein schreibgeschützter eigener Hook: `ReplaceText` scheitert,
  das folgende `Chmod` nimmt den Schreibschutz und löschte den Fehler, der
  Hook galt als eingerichtet mit altem Text).

**Was steht, mit Begründung.**

| Paket | Mutant | Entscheidung |
|---|---|---|
| `maintenance` | `RemoveHooks`, `mergehook.go:242`: `target.repo == "" \|\|` entfernt | **Äquivalent.** Ohne Repo ist `target.hook` leer; `isOurs("")` liest nichts und gibt `false`, die Schleife geht weiter. |
| `maintenance` | `writeHook`, `:364`; `writeRecords`, `:419`: Fehler von `MkdirAll` ignoriert | **Äquivalent im Ergebnis.** Fehlt das Verzeichnis, scheitert `lock.ReplaceText` daran (`TestInstallFailsWhereTheHooksDirectoryCannotBeMade`, `TestInstallFailsWhereTheRecordCannotBeWritten`). |
| `maintenance` | `writeHook`, `:367`: `if err == nil` → `false` (kein `Chmod`) | **Unter Windows nicht beobachtbar**, und nur dort läuft die Testkette (`.github/workflows/ci.yml`, Linux baut und prüft nur `vet`). `Chmod 0o755` setzt unter Windows nur den Schreibschutz zurück, den `ReplaceText` gar nicht setzt. |
| `hostfile` | `firstCommand`, `merge.go:140`: `!ok \|\|` entfernt; `:144`: `if !ok` → `false` | **Äquivalent.** Ist `hooks` keine Liste, ist die Variable nil und `len` 0; ist das erste Element keine Map, liest `first["command"]` aus einer nil-Map und gibt `""`. |
| `hostfile` | `extractTopEntries`, `merge.go:176`: `len(data) == 0` → `false` | **Äquivalent.** Der Decoder liefert für leere Daten EOF, und die nächste Zeile gibt nil zurück. |
| `hostfile` | `formatRoot`, `merge.go:203`: `len(existingEntries) == 0 &&` entfernt | **Nicht erreichbar.** `Merge` ruft `formatRoot` nur, wenn es etwas einträgt; `root` hält dann immer den Behälterschlüssel. |
| `setup` | `configText`, `configtext.go:59`: `before != ""` → `true` | **Äquivalent.** `schema.Validate("")` nimmt den leeren Text an. |
| `setup` | `configText`, `configtext.go:68`: nur gesetzte Einträge → alle | **Äquivalent.** Die Vorgaben von `[modules]` sind `true` und die von `commit.language` `en`. Die Fälle darunter schreiben nur bei `false` oder einer anderen Sprache; eine Vorgabe in `current` verhält sich darum wie ein fehlender Schlüssel. |
| `setup` | `configText`, `configtext.go:106`: `text != before` → `true` | **Äquivalent.** Ist nichts geändert, ist `text` der schon geprüfte oder leere Text, und `Validate` nimmt ihn wieder an. |
| `setup` | `DefaultChoice`, `parts.go:92`: `len(answers.Hosts) > 0` → `true`, `>= 0` | **Äquivalent.** Ohne Hosts in den Antworten bleibt `chosen` leer, und die innere Prüfung `len(chosen) > 0` lässt die erkannten Hosts stehen. |

**Nebenwirkungen der Runde.** Zwei getötete Mutanten schrieben in ein
Paketverzeichnis: `internal/brain/maintenance/post-merge` (ein Mutant in
`targetOf` machte den Hookpfad relativ; jetzt wechselt
`TestAHooksDirectoryGitCannotNameIsNoRepository` in ein eigenes
Verzeichnis, die dritte Runde ließ nichts liegen) und
`internal/setup/gitfiles/=` (der `>=&2`-Mutant; der Test lässt das Skript
jetzt in einem eigenen Verzeichnis laufen). Beide nach der Runde von Hand
entfernt.

## Antigravity-Einträge, 2026-09-25

Gemessen am 2026-09-24 mit agy 1.2.8 („Messungen vor dem Bau“ im Plan):

- agy führt einen Hook-Befehl über `cmd.exe` aus: `${LOCALAPPDATA}` bleibt
  wörtlich stehen, `%LOCALAPPDATA%` wird aufgelöst.
- Ein gequoteter Programmpfad zerbricht, weil agy `"…"` als `\"…\"`
  weiterreicht; ungequotet `%LOCALAPPDATA%/loomux/bin/loomux.exe` geht,
  Schrägstriche vorwärts auch.
- Das Arbeitsverzeichnis eines Hooks ist `.agents/` (Messung vom
  2026-09-10), also `--root ..`.
- Ein scheiternder Hook blockiert den Werkzeugaufruf.
- Projekt-Skills liegen unter `.agents/skills/<name>/SKILL.md`.
- Projekt-Hooks lädt agy nur in einem vertrauten Ordner
  (`trustedWorkspaces` in `~/.gemini/antigravity-cli/settings.json`).
- `"timeout"` in Sekunden wird beachtet, Vorgabe 30 s.

Gebaut:

- `antigravityMeasured` ist weg. Die zwei Einträge (`pre-tool-use` mit
  Timeout 15, `post-tool-use` mit 60, Matcher wie zuvor) rufen immer
  `hostfile.AntigravityBinary`, also
  `%LOCALAPPDATA%/loomux/bin/loomux.exe … --host antigravity --root ..`,
  auch in einem Checkout. `Apply` schreibt `.agents/hooks.json` darum nur,
  wenn das installierte Binary steht (wie beim Merge-Hook); die Einträge
  von Claude Code behalten ihr Tor. Schon der Plan lässt die Einträge mit
  einer Notiz weg, wenn das installierte Binary fehlt und im selben Lauf
  kein `binary-install` geplant ist.
- Enthält `LOCALAPPDATA` Leerraum (`Facts.LocalAppDataSpaced`, gelesen in
  `Gather`), plant `init` keine Antigravity-Einträge und sagt es; die
  Hookdatei wird dann nicht gelesen, die Skills kommen trotzdem.
- Die Regeln von Fusions-Spec #21 stehen in `internal/setup/hostfile`: Die
  Gruppe `loomux` gehört uns, jede andere wird Token für Token übernommen
  (`json.Indent` des gelesenen Rohtexts: Schlüsselreihenfolge, Escapes und
  Zahlen bleiben, nur die Einrückung wird zwei Leerzeichen — dieselbe
  Treue, die ultraloom mit seinem Encoder hatte; eine schon so eingerückte
  Gruppe kommt Byte für Byte zurück). Eine fremde Gruppe, die einen der
  gewollten Befehle wörtlich schon ruft, wird als Notiz gemeldet, nie
  repariert. Eine Wurzel `null` wird abgelehnt, mit dem Dateinamen — auch
  in `.claude/settings.json`, wo `null` bisher als leere Datei galt; ein
  Merge, eine Regel.
- **`internal/agenthooks` zieht nicht um.** Der Merge in `hostfile` kannte
  die Gruppe `loomux` schon als eigenen Container; ein zweites Paket für
  dieselbe Datei hätte zwei Schreiber mit zwei Formaten ergeben. Nur seine
  Regeln (Null-Wurzel, Meldung fremder Gruppen) sind übernommen.
- Die Notiz „no entries or skills yet“ ist weg; statt ihrer erinnert der
  Plan an `trustedWorkspaces`.

**Durchstich mit einem laufenden agy, 2026-09-25** (agy 1.2.8, loomux 2.11.1
am kanonischen Ort, ein Wegwerf-Repo unter `C:\Users\micro`, also vertraut).
`init --dry-run --yes --hosts=antigravity` mit dem Binary dieses Zweigs
plante `.agents/hooks.json` genau in der obigen Form; dieser Text wurde in
das Repo gelegt, dann lief `agy -p … --add-dir <repo>
--dangerously-skip-permissions`:

| Auftrag an agy | Ergebnis |
|---|---|
| `.loomux/config.toml` mit dem Schreibwerkzeug anlegen | verweigert: der Hook endete mit Exit 2 und der Begründung des Wächters („the manifest is where the barrier reads its own limits …“), die Datei entstand nicht |
| `hello.txt` mit dem Schreibwerkzeug anlegen | geschrieben, `post-tool-use` lief ohne Einwand |

Damit ist belegt: agy lädt die Gruppe `loomux`, `%LOCALAPPDATA%` löst auf,
`--root ..` trifft die Projektwurzel, und der Wächter von loomux sperrt
unter Antigravity, was er unter Claude Code sperrt.

## Offen

- **Task 1 ist gelaufen, bis auf einen Schritt.** Am 2026-09-24 vom Agenten
  gemessen (Plan, „Messungen vor dem Bau“): Git-Hook mit `${LOCALAPPDATA}`
  läuft, `.mcp.json` mit `${LOCALAPPDATA}` verbindet über
  `claude -p --mcp-config`, der Skill-Ort von Antigravity ist
  `.agents/skills/<name>/SKILL.md`, und agy führt Hooks über `cmd.exe` aus
  (`%LOCALAPPDATA%` ohne Anführungszeichen). Nachbestätigt am 2026-09-25:
  ein post-commit-Hook unter Git for Windows sieht `LOCALAPPDATA` als
  `C:\Users\micro\AppData\Local` und findet dort `loomux.exe`. Offen bleibt
  der Freigabeweg einer Projekt-`.mcp.json` („Pending approval“), der eine
  interaktive Sitzung braucht (Mensch).
- **Task 15, Schritte 2 bis 4 (Mensch):** siehe „Selbstnutzung“.
- **Die aufgeschobenen Kleinigkeiten beider Akten, sortiert am 2026-09-25**
  gegen den Code (die Listen unten und in `stufe-4a-1.md` bleiben als
  Herkunft stehen):
  - **Fehler** — die ersten fünf sind am 2026-09-25 behoben (Auswahl des
    Menschen): der Modus eines Skripts richtet sich nach `#!` oder `.sh`,
    `edit.split` trennt an `\n` und behält in einer gemischten Datei jedes
    `\r`, ein veralteter eigener Block wird gemeldet, `InstallHooks` schreibt
    die Einträge vor dem Fehler, und `status` meldet `moved`:
    - Hooks in einem eigenen `core.hooksPath` oder in `.git/hooks`
      entstehen mit 0644: `writeNew` macht nur `.githooks/…` und `*.sh`
      ausführbar (`internal/setup/write/atomic.go:194`). Auf POSIX überspringt
      git sie still.
    - Gemischte Zeilenenden: Steht ein einziges `\r\n` in der Datei, trennt
      `edit.split` nur daran (`internal/config/edit/edit.go:49`). LF-Zeilen
      kleben dann aneinander, und ein `Set` auf eine solche Zeile verschluckt
      Schlüssel.
    - `Kept` meldet einen veralteten eigenen Block als eingerichtet
      (`internal/setup/hostfile/merge.go:81`).
    - `InstallHooks` kehrt vor `writeRecords` zurück, und schon geschriebene
      Hooks bleiben ohne Eintrag (`internal/brain/maintenance/mergehook.go:157`).
    - `merge-hook status` vergleicht den Hookpfad nicht mit dem heutigen
      Hookverzeichnis (`mergehook.go:188`).
    - An der Kommandozeile:
      - `config --root DIR list` endet mit Exit 2, fremde Flags werden still
        angenommen.
      - `yes` gilt als Ablehnung, nur `y` bestätigt.
      - `Render(StringList)` macht aus `"a,"` ein leeres Element.
      - `init --yes` schreibt einen `Scope` mit Leerzeichen, den die
        interaktive Form verweigert.
      - Ein angebotenes `none` lässt das Modul an.
    - Kosmetisch:
      - `Remove` lässt Kommentare in einem geleerten Abschnitt zurück.
      - `tui.fit` zählt Runen statt Zellen.
      - Die Spalte „Wert“ von `config list` wird zu breit.
      - `samePath` vergleicht auch auf POSIX ohne Groß- und Kleinschreibung.
      - `merge-hook status` zeigt bei einer fremden Datei `[datei]`.
  - **Testlücken:**
    - Vorgaben aller `Keys()` dekodieren
    - `tui.Pick`-Tests prüfen `ok` und `err` nicht
    - Modus in `TestWriteNewExecutablePermissions`
    - `OnMerge` über `.brain.toml`
    - `remove-installed` nur über die Bytezahl
    - Fixture gegen `.claude/settings.json`
    - 8.3 und Junction für das Common-Dir und `within`
    - Vorlagentest prüft nur das erste Wort
    - `approve` wird für entfallene Änderungen nicht gefragt
  - **Fällt weg** (wie die Referenz, nicht erreichbar oder Prozessnotiz):
    - `projectModules` und `FindRoot`
    - `MkdirAll` mit 0o755 (nur Windows)
    - `tui.Pick` bei ungleichen Längen, und Zeilen ohne Notiz
    - `.githooks` am ganzen Pfad
    - `merge_branch`
    - ein verschobenes Repo
    - der Kommentar in `cases_4a2_test.go`
    - `targetOf`-Zusammenlegung
    - `RunsAGate`
    - `!/.loomux/state/`
    - CRLF in `.gitignore`
    - `brain-research`
    - `lifecycleOrder`
    - Modul aus bei ausgeschaltetem `config`
    - `namedConfig`
    - Toolchain-Zeit von `TestBuildPilotBuildsAndSwaps`
  - **Schon erledigt:**
    - Tests von `selfupdate`
    - `DefaultMergeBranch`
    - `registered()` über `SameDir`
    - `SetEscapeHTML(false)`
    - das Wort `default` als Wert
    - `go run` im Wächter
  - Was davon umgesetzt wird, entscheidet der Mensch.
- **Behoben am 2026-09-25** (`plans/2026-09-25-loomux-4a-nachziehen.md`):
  - Die Fehlverweigerung des Wächters bei `cd "C:/…/#GIT/…" &&
    bin/loomux.exe init --dry-run`: `plainLine` ließ in doppelten
    Anführungszeichen kein `#` zu, die ganze Zeile galt als nicht plain.
  - `init` schreibt keine Datei mehr über, die sich seit dem Plan geändert
    hat; ein gescheitertes Schreiben steht in `Failed`; ein Lauf mit einem
    Fehler hinterlässt kein `installed.toml`.
  - `.mcp.json` entfällt mit dem Binary, das sie ruft (`Change.Binary`
    ersetzt die Tabelle `callsBinary`).
  - Eine Hookdatei mit der Wurzel `null` wird abgelehnt statt überschrieben
    (#21; der von ulinit übernommene Test, der sie als leer nahm, ist
    ersetzt).
  - `config` schreibt nicht mehr über eine Datei, die sich seit dem Lesen
    geändert hat (aus der Liste von 4a-1).
- **Aufgeschobene Kleinigkeiten aus dem Bau** (Ledger, `minor (deferred)`):
  - `selfupdate`: `TestInstallReplacesAnOlderBinary` übergeht Fehler der
    Einrichtung und prüft den neuen Inhalt nicht; `NewerOrEqual` wird nur auf
    Gleichheit getestet; der Test zum gescheiterten Download prüft nur
    `Outcome`; `MkdirAll` mit 0o755 statt 0o700 wie anderswo (Task 2).
  - `tui.Pick` bricht bei `len(chosen) != len(rows)` ab; Zeilen ohne Notiz
    enden auf zwei Leerzeichen; einige Tests übergehen `ok`/`err` (Task 3).
  - `write`: `TestWriteNewExecutablePermissions` prüft keinen Modus;
    `atomic.go:194` prüft `.githooks` am ganzen Pfad samt Wurzel (geerbt);
    `writeNew` macht nur `.githooks/…` und `*.sh` ausführbar, Hooks in
    `.git/hooks` oder einem eigenen `core.hooksPath` sind auf POSIX nicht
    ausführbar (Tasks 4, 12).
  - `config`: kein Test hält `OnMerge`/`MergeBranch` über
    `ReadAreaManifestUntilStage4` für `.brain.toml`; `DefaultMergeBranch` ist
    exportiert; `merge_branch` der Referenz wird ignoriert (Task 5).
  - `maintenance`: Einträge nach Bereich, ein verschobenes Repo lässt den
    alten Hook unverfolgt (wie die Referenz); `status` vergleicht den
    Hookpfad des Eintrags nicht mit dem heutigen Hookverzeichnis;
    `InstallHooks` kehrt bei einem Fehler mittendrin vor `writeRecords`
    zurück; `samePath` vergleicht auch auf POSIX ohne Groß-/Kleinschreibung;
    8.3- und Junction-Schreibweisen des Common-Dir ungetestet (Task 6).
  - `merge-hook`: ein Kommentar in `cases_4a2_test.go:517` nennt stdout statt
    `world_after`; `remove-installed` ist nur über die Bytezahl
    festgenagelt; `status` druckt ` [<datei>]` bei `not installed` auch für
    eine fremde Datei; der `targetOf`-Fix in `ffb5ab3b` gehört beim
    Zusammenlegen in `c4445900` (Task 7).
  - `gitfiles`: `RunsAGate` erkennt großzügig über Teilzeichenketten (irrt
    zum Behalten eines Hooks hin); `!/.loomux/state/` zählt nicht als
    vorhanden; gemischte Zeilenenden bekommen CRLF (Task 9).
  - `templates`: der Test auf vorhandene Befehle prüft nur das erste Wort;
    `brain-research` gibt die Konvention „model: opus, effort: low“ an
    Wirte weiter (wie das Original) (Task 10).
  - `hostfile`: `lifecycleOrder` (siehe Abweichung 8); `json.Marshal`
    maskiert `&`, `<`, `>` in umgeschriebenen fremden Befehlen (geerbt); die
    Fixture `loomux-settings.json` kann von `.claude/settings.json`
    abdriften; `Kept` kann ein veralteter eigener Block sein, ohne dass der
    Bericht es sagt (Task 8).
  - `setup` (Plan): Tests nicht zuerst geschrieben (Prozess); ein Modul aus
    bei ausgeschaltetem Teil `config` wird nicht festgehalten (die
    Zusammenfassung nennt es); `namedConfig` prüft über eine
    Teilzeichenkette; `Scope` für Verzeichnisnamen mit Leerzeichen nicht
    bereinigt; `registered()` vergleicht Registry-Pfade nach Schreibweise
    (eine 8.3- oder Junction-Wurzel gilt als nicht registriert); kein
    direkter 8.3-Test für `within` (Task 11).
  - `setup` (Schreiben): ein gescheitertes Schreiben steht nicht in
    `Failed`; `installed.toml` entsteht auch nach einem gescheiterten
    Binary-Schritt; `.mcp.json` ruft das Binary, entfällt aber nicht mit ihm;
    die Sicherung nimmt den Text zur Planzeit, und `ReplaceText` vergleicht
    nicht; kein Test, dass `approve` für entfallene Änderungen nicht gefragt
    wird (Task 12).
  - `init`: ein angebotenes `none` auf einem Modul ohne eingeschalteten Teil
    lässt das Modul an (nur `--graph=none` schaltet es aus);
    (`~` in `core.hooksPath` ist behoben: `git config --type=path` liest es
    als Pfad, und der Merge-Hook wird dort gesucht, wo
    `git rev-parse --git-path hooks` hinzeigt);
    `TestBuildPilotBuildsAndSwaps` legt Toolchain-Zeit auf das Tor
    (Task 13).

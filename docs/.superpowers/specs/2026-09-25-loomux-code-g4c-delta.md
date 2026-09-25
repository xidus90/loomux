# loomux G4c — Nachtrag: der Stop-Hook mit Blast-Logik

**Datum:** 2026-09-25  
**Stand:** entworfen und vom Nutzer freigegeben am 2026-09-25 (Brainstorming, Entscheidungen in §6).  
**Ergänzt:** [`2026-09-23-loomux-code-g4b-delta.md`](2026-09-23-loomux-code-g4b-delta.md), dort E4′
und §4 („Im Profil `stop` … immer `not-applicable`“). Wo beide sich widersprechen, gilt diese Datei.  
**Arbeitsort:** Branch `feat/g4c-stop-blast`, abgezweigt von `master` `0758f9d3` (v2.14.0). Plan
und Umsetzung liegen auf demselben Branch und gehen in einem Pull Request.

## 1. Worum es geht

G4b hat die Art `graph` gebaut (`check graph-fresh`, dann `check blast-audit --cached --threshold 5`)
und sie in die Profilvorgabe `precommit` gelegt. Aus `stop` blieb sie heraus (E4′), weil die Lane den
Index liest und der am Rundenende fast immer leer ist. G4c bringt die Art ans Rundenende: Ein
geänderter Bereich, dessen Seeds mindestens N Aufrufer haben und den kein geänderter Test erreicht,
hält die Runde an, wie er einen Commit anhält.

## 2. Befund: „Arbeitsbaum gegen HEAD“ ist als `git diff HEAD` falsch

Die Fusion-Spec und der Migrationsplan nennen als Form „Arbeitsbaum gegen HEAD“. `blastWith` ohne
`--base` und ohne `--cached` ruft `git diff … HEAD`, und das sieht **unversionierte Dateien nicht**.
Am Rundenende ist das der Normalfall: Ein Agent, der einen Hub ändert und einen neuen `_test.go`
danebenlegt, hat den Test noch nicht mit `git add` aufgenommen.

Nachgemessen am 2026-09-25 auf diesem Repo (709 Dateien, Graph frisch gebaut): `gitenv.Environ`
geändert (Eingangsgrad 56), dazu ein neuer, unversionierter `internal/gitenv/probe_new_test.go`.

| Form | Ergebnis |
|---|---|
| `check blast-audit --threshold 5` (Arbeitsbaum gegen HEAD) | Exit 1, `gitenv.go [stale]` — falsch, der Test ist unsichtbar |
| `--cached`, echter Index | Exit 0, 0 Bereiche — sieht gar nichts |
| `--cached`, `GIT_INDEX_FILE` = Kopie des Index mit `add -A`, im Git-Verzeichnis | Exit 0, 2 Bereiche — richtig |
| dieselbe Kopie, Test gelöscht | Exit 1, `gitenv.go [stale]` — richtig |

Dazu kommt: `blastWith` fällt bei sauberem Arbeitsbaum auf `HEAD~1...HEAD` zurück
(`internal/code/query/blast.go`) und prüfte am Rundenende nach einem Commit den letzten Commit erneut.
Die Form `--cached` nimmt diesen Zweig nicht.

**Folge:** Der Bereich bleibt „Arbeitsbaum gegen HEAD“ im Sinn von „alles, was git nicht ignoriert,
gegen HEAD“. Gebaut wird er über eine Indexkopie, nicht über einen neuen Modus von `blast-audit`.
Gegen HEAD und nicht gegen die Basis der Sitzung, weil jeder Commit schon durch die precommit-Lane
lief.

## 3. Mechanik

**Profilvorgabe.** `stop` wird `lint, types, test, coverage, graph`. Der Kommentar an der Vorgabe in
`internal/verify/schema.go` („without graph: its lane reads the index“) wird ersetzt. Ein Projekt, das
`stop` in `[verify.profiles]` selbst definiert, nimmt die Art erst auf, wenn es sie einträgt;
`.loomux/config.toml` dieses Repos definiert kein Profil.

**Der Index des Rundenendes.** `gitwork.ContentTree` baut heute eine Kopie des Index unter
`os.TempDir()`, nimmt mit `add -A` alles auf, wirft `.loomux/state` wieder hinaus, schreibt den Baum
und löscht die Kopie. Neu ist eine zweite Form, die die Kopie behält und ihren Pfad zurückgibt; der
Aufrufer räumt sie weg. Sie liegt **im eigenen Git-Verzeichnis** (`git rev-parse --absolute-git-dir`,
in einem verknüpften Worktree also `…/.git/worktrees/<name>`), Name `loomux-stop-index-<pid>`, denn
nur dort nimmt `query.indexFileFor` einen geerbten `GIT_INDEX_FILE` an. Die Form ohne Behalten bleibt,
wo sie ist, samt `os.TempDir()`: Ohne Graph ändert sich am Stop-Hook nichts.

**Wann die Kopie entsteht.** `RunStop` behält sie nur, wenn `graph` in den Arten des Profils steht
**und** `store.WiringPath(root)` existiert. Die Arten liest `RunStop` heute erst nach `stopTrees`; die
Reihenfolge wird so gedreht, dass Konfiguration und Arten vor dem Baum feststehen. Ein Fehler beim
Laden der Konfiguration bleibt Exit 1 wie heute. Die Kopie wird nach `verify.Run` gelöscht, auch bei
Rot, Budget und jedem frühen Ausstieg dazwischen (`defer`). Ein Baum, der schon grün war
(`tree == state.Green`), lässt die Kette samt Graph-Lane aus; auch dann wird die Kopie gelöscht.

**Die Prüfung im Plan.** `RunStop` setzt `PlanEnv.GraphReady` und `PlanEnv.GraphEnv` selbst, statt sie
`nil` zu lassen:

- `GraphEnv(root)` liefert `GIT_INDEX_FILE=<kopie>`. Nur der Job der Art `graph` bekommt ihn
  (`plan.go` hängt `GraphEnv` nur dort an); `graph-fresh` liest keinen Index, `blast-audit` liest ihn
  über `runGit`.
- `GraphReady(root)` antwortet in dieser Reihenfolge `false` mit Notiz:
  1. kein `wiring.json` — „no graph at .loomux/state/graph/wiring.json“ (wie `query.GraphReady`);
  2. kein HEAD oder kein Repository — „no HEAD to compare with“; ohne Repository entsteht keine Kopie;
  3. eine laufende Operation (Merge, Rebase, Cherry-Pick, Revert), dieselben Marker und Notizen wie
     `query.GraphReady` (`inProgress`); am Rundenende zeigte die Kopie sonst fremde Änderungen als
     eigene;
  4. der Content-Tree gleicht dem Baum von HEAD — „nothing changed against HEAD“. `stopTrees` kennt
     HEAD und den Content-Tree schon; `gitwork.TreeOf(root, head)` ist ein git-Aufruf mehr, und nur
     wenn `graph` angefragt ist.

  Schritt 1 und 3 teilt sich die Funktion mit `query.GraphReady`; nur Schritt 4 ersetzt dort
  „nothing staged“. Die Aufteilung (gemeinsamer Teil in `query`, Schritt 4 im Hook) legt der Plan fest.

**Unverändert:** der Befehl im Go-Preset, das Überschreiben über `[verify.go.graph] commands` (wer die
Schwelle ändert, ändert sie auch am Rundenende), `blast-audit` und `graph-fresh` selbst, ein Job je
Stack an der Wurzel (G4b §4), das Gedächtnis für grüne Bäume.

**Urteil.** Ein roter `blast-audit` ist eine rote Lane: `stopVerdict` schreibt sie auf stderr und
hält mit Exit 2, der Block zählt, nach `MaxBlocks = 3` gibt der Hook auf und die Basis bleibt, wie bei
jeder Lane. `not-applicable` neben Lanes, die liefen, lässt `CheckVerdict` ohne Notiz; das Urteil
bleibt Exit 0, die Basis rückt vor. `graph-fresh` baut am Rundenende fast immer neu (der Arbeitsbaum
hat Drift); der Neubau schreibt nur unter `.loomux/state/`, das der Content-Tree ausnimmt.

**Handlauf.** `loomux check stop` baut dieselbe Kopie und dieselbe Prüfung wie der Hook, damit ein
Mensch das Urteil des Hooks nachstellen kann; heute liefe dort `query.GraphReady` gegen den echten
Index und meldete „nothing staged“. `internal/cli` importiert `internal/hooks` schon (`check.go`,
`hook.go`); die gemeinsame Hilfe liegt dort. `check precommit` und jede Liste von Arten bleiben beim
echten Index.

**Wirte.** Antigravitys `Stop`-Eintrag läuft durch dasselbe `RunStop` und bekommt die Lane mit; dort
ist nichts eigens zu bauen.

## 4. Kosten

Gemessen am 2026-09-25 auf diesem Repo, fünf Läufe nach je einem Edit an `gitenv.go`, Binary aus
`master` `0758f9d3`, jeder Schritt ein eigener Prozess:

| Schritt | Spanne |
|---|---|
| `check graph-fresh` mit Neubau | 2010–3169 ms |
| `check graph-fresh` ohne Drift | 264–617 ms |
| Indexkopie mit `add -A` | 394–724 ms |
| `check blast-audit --cached` über die Kopie | 421–1316 ms |

Die Lane kostet am Rundenende also 3–4 s und läuft parallel zu `test` und `coverage` im Budget von
270 s. Die Indexkopie fällt nicht zusätzlich an, weil `ContentTree` sie ohnehin baut. Die Messung
des ganzen Stop-Hooks (vier Arten gegen fünf, kalt und warm) folgt in der Umsetzung und geht in
beide `benchmarks.md`.

## 5. Tests und Abnahme

- `gitwork`: Die behaltene Kopie liegt im Git-Verzeichnis, auch in einem verknüpften Worktree, trägt
  unversionierte Dateien und nicht `.loomux/state`; der Baum ist derselbe SHA wie aus der Form ohne
  Behalten; Aufräumen löscht sie.
- Prüfung am Rundenende: kein Graph, kein HEAD, jede laufende Operation und „nothing changed against
  HEAD“ ergeben `not-applicable` mit Notiz; mit einer Änderung ist die Lane bereit.
- `RunStop` in einer Welt mit echtem git und gebautem Graphen:
  - Hub geändert, neuer **unversionierter** Test daneben ⇒ Exit 0;
  - Hub geändert, kein Test ⇒ Exit 2, der Befund auf stderr, ein Block gezählt;
  - kein Graph ⇒ Lane `not-applicable`, Exit 0, die Basis rückt vor;
  - nach dem Lauf liegt keine Kopie mehr im Git-Verzeichnis, auch nach Exit 2.
- Ein Profil `stop` ohne `graph` baut keine Kopie.
- `loomux check stop` gibt in derselben Welt dasselbe Urteil wie der Hook.
- Die Tests, die die Profilvorgabe aufzählen, ziehen nach.
- **Selbstnutzung:** eine Runde in diesem Repo, die einen Hub ohne Test ändert, wird vom echten Hook
  (`bin/loomux.exe`) angehalten; eingetragen in die Paritätsakte `parity/code-g4.md`.

## 6. Getroffene Entscheidungen

**E9 — Ein Befund hält die Runde (Exit 2), wie beim Commit.** Verworfen: nur warnen (§8 der
Säule-3-Spec: „Warnt …“). Dafür bräuchte die Kette einen Zustand „rot, aber nicht haltend“, den es
nicht gibt, und ein übergangener Befund käme beim Commit wieder. Die Rot-Quote bei N = 5 war an
Commits 1 von 50 (G4b, E2′); an halbfertigen Diffs des Rundenendes ist sie nicht gemessen.

**E10 — `graph` in die Profilvorgabe `stop`.** Verworfen: nur möglich machen und einem Menschen das
Eintragen lassen. Die Stufe ist dafür da, dass loomux sich an jedem Rundenende selbst prüft, und ohne
Graph kostet die Lane nur einen `Stat`.

**E11 — Der Bereich über eine Indexkopie, nicht über einen neuen Modus.** Verworfen: `blast-audit`
ohne Flag (sieht keine unversionierten Dateien, fällt auf `HEAD~1` zurück, §2), ein neues Flag oder
ein Platzhalter im Befehl (bräche Projekte, die `[verify.go.graph] commands` schon überschreiben) und
ein Job im Prozess wie `lint/wiki` (überginge das Überschreiben ganz).

**Release:** `release:minor`. Die Profilvorgabe `stop` bekommt eine Art, die ohne Graph
`not-applicable` ist; Vorbild ist G4b bei `precommit`. Kein Exit-Code, kein Flag und kein Format
ändert sich; ein Rundenende kann halten, wo es vorher nicht hielt.

## 7. Doku im selben Pull Request

- Fusion-Spec: Zeile G4c in der Stufentabelle und in der Reihenfolge; nach dem Bau ✅.
- `docs/en/migration.md` und `docs/de/migration.md`: Stufe G4c, die Fähigkeit „Stop-Hook Blast
  Audit“, der Graph im Mermaid-Diagramm.
- `docs/en/configuration.md` und `docs/de/configuration.md`: der Absatz zur Lane `graph`, der sie im
  Profil `stop` `not-applicable` nennt.
- `README.md`, `README.de.md`, `docs/wiki/topics/code-graph.md`, `docs/wiki/topics/scheiben-und-abnahme.md`.
- Beide `benchmarks.md` (§4).

## 8. Nachtrag 2026-09-28: Entscheidungen aus dem Code-Review

Das Code-Review vor dem Pull Request fand drei Stellen, an denen die Mechanik aus §3 mehr braucht,
und eine Grenze, die hier nicht stand. Entschieden vom Nutzer am 2026-09-28:

**E12 — Das Gedächtnis „schon grün“ hängt mit Graph-Lane auch an HEAD.** Der Hook merkt sich nur
den Inhaltsbaum; die Graph-Lane urteilt aber gegen HEAD. Ein Commit innerhalb der Runde (etwa nur der
Test) lässt den Baum gleich, schiebt HEAD weiter und nimmt den Test aus dem, was die Lane neben dem
geänderten Code sieht. Mit Graph-Lane gilt ein Baum deshalb nur unter dem HEAD als grün, unter dem
er grün war (der Basis, die ein grüner Lauf setzt). Ohne Graph-Lane bleibt es beim Baum allein.
Kostet nach jedem Commit innerhalb einer Runde einen Lauf der Kette.

**E13 — Gegen HEAD, nicht gegen die Basis, bleibt; die Grenze wird genannt.** Die Lane prüft, was
seit dem letzten Commit hinzukam, und setzt voraus, dass das pre-commit-Tor jeden Commit geprüft
hat. Ein Commit an ihm vorbei (`--no-verify`, Cherry-Pick, Merge, ein Klon ohne scharfe Hooks) ist
am Rundenende Teil von HEAD und wird nie geprüft. Verworfen: gegen die Basis der Sitzung prüfen —
ein größerer Umbau, und ein Befund träfe dann auch Commits, die das Tor schon abgenommen hat. Die
Grenze steht in `configuration.md`.

**E14 — Eine liegengebliebene Kopie räumt das nächste Rundenende weg.** Ein Prozess, der während der
Kette abgebrochen wird, läuft kein `defer`; seine Kopie `loomux-stop-index-<pid>` (rund 2 MB) bliebe
für immer im Git-Verzeichnis. `gitwork.KeptContentTree` entfernt vor dem Schreiben jede Kopie, deren
Prozess nicht mehr läuft (`child.Alive`), und lässt die eines laufenden stehen: zwei Sitzungen in
einem Repository teilen das Git-Verzeichnis.

**Zur Mechanik.** Die Prüfung am Rundenende (`StopIndex.Ready`) ruft nicht mehr
`query.GraphPrereq`, sondern fragt dasselbe selbst: Die Kopie belegt ein HEAD, und die Marken einer
laufenden Operation liegen im Git-Verzeichnis neben ihr (`gitwork.InProgress`, von beiden Prüfungen
geteilt). So bleibt `internal/hooks` frei von `query`, dessen Extraktoren seit G5a Tree-sitter
tragen. `loomux check stop --show` baut keine Kopie. Und `check stop` fährt dieselben Lanes wie der
Hook, nicht das Tor um sie herum (Marker, schon grüner Baum, Befunde der Subagenten, Zähler) — der
Handlauf aus §3 stellt die Lanes nach, nicht jedes Urteil.

**Nebenbei behoben, schon auf master:** Eine Rebuild-Sperre, deren Prozess nicht mehr läuft, galt
eine Stunde lang; jedes Tor, das den Graphen auffrischt, wartete und scheiterte bis dahin. Mit
`graph` im Profil `stop` hätte das jedes Rundenende getroffen. Eigener Commit `fix(graph)`.

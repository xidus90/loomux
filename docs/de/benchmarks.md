# Benchmarks

Chronologische Leistungsmessungen für loomux, kalt und warm.

## 2026-09-14 16:05 — Basismessung (Vorläufer-Programme)

Gemessen unter Windows x86_64, warm Median über 5 Läufe. Grundlage für die Zielwerte der Loomux-Fusion.

| Fall | Vorläufer-Laufzeit | Zeit | Loomux-Zielwert |
|---|---|---:|---|
| `ulguard --root` (PreToolUse, Edit) | Go (`ulguard.exe` 6,0 MB) | 33 ms | < 35 ms |
| `brain guard` | Go (`brain.exe` 13,7 MB) | 72 ms | vereint in `hook pre-tool-use` |
| Beide Wächter parallel (wie Edit sie sieht) | Go + Go | 73 ms | < 35 ms (vereint im Prozess) |
| Beide Wächter sequenziell (CPU-Zeit) | Go + Go | 114 ms | < 35 ms |
| Go-Startboden (`brain version`, `ulinit --version`) | Go (`ulinit.exe` 8,5 MB) | 32–34 ms | ~32 ms |
| `ulguard hook session-start` | Go | 67 ms | < 100 ms (Stufe 2) |
| `ultraloom hook subagent-start` | Python (mit/ohne `uv run`) | 706–730 ms | < 100 ms (Go-Port) |
| `brain-mcp --help` | Python | 898 ms | < 35 ms (Go-Port) |
| `brain-mcp status` / `brain status` | Python / Go | 3.113 ms / 36 ms | < 35 ms |
| `brain-mcp search` / `brain search` | Python / Go | 7.773 ms / 242 ms | < 50 ms |

### Wichtigste Befunde
1. Die Binärgröße bestimmt die Startzeit nicht: `brain.exe` (13,7 MB) startet in 32 ms, `ulinit.exe` (8,5 MB) in 34 ms, `ulguard.exe` (6,0 MB) in 33 ms.
2. Der Startboden wird ausschließlich durch Paket-`init()`-Funktionen und das Parsen eingebetteter Daten bestimmt, nicht durch die Dateigröße.
3. Python-Hooks und CLI-Brücken verursachten 700–3.000 ms Verzögerung, die durch das reine Go-Binary vollständig entfallen.

## 2026-09-15 01:20 — Der vereinte Wächter gegen die beiden, die er ersetzt

Repo `loomux`, Worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1a`, Zweig
`sdd-1a`, Commit `8ebdc93` plus dem uncommitteten Baum von Task 15 — dieser Task
fügt das Messwerkzeug und diesen Eintrag hinzu, er ändert den Hook-Pfad nicht.

**Ziel.** Zeigen, was `loomux hook pre-tool-use` gegen den Zielwert von 72 ms
kostet (dem Paritätswert beider alter Wächter, wie ein Edit sie traf), und
zerlegen, wofür Startboden und Schreibschranke ihre Zeit ausgeben.

**Methode.** `loomux dev bench-hooks testdata/bench/1a-hooks.json -n 20`: je Fall
ein kalter Lauf, dann 20 warme; die Tabelle nennt kalt, warmen Median, warmes
Minimum, warmes Maximum und die Exit-Codes. Nutzlast: ein `Edit` auf die
`README.md` dieses Worktrees (`testdata/bench/edit-readme.json`). Gelaufen mit
`LOOMUX_STATE_DIR=$TEMP/loomux-bench-state`, darin eine `registry.toml` mit einem
`[[area]]` für diesen Worktree (`workspace = true`); eine `.loomux/config.toml`
gibt es hier nicht (eine fehlende Datei ist eine leere Policy), die echte
Pilot-Konfiguration aus Task 16 kostet also ein TOML-Parse mehr als diese
Messung. Gemessene Binaries: `bin/loomux.exe`, aus diesem Baum mit Go 1.27
gebaut, und die Vorgänger
`C:/Users/micro/AppData/Local/Temp/loomux-old/{ulguard,brain}.exe`, gebaut aus
den Tag-Worktrees `loomux-1a-source` von `ultraloom` und `ultra-brain` — die
getaggten Stände, nicht das, was auf dem PATH liegt.

| Fall | kalt (1. Lauf) | warmer Median | warmes Min | warmes Max | Exit-Codes |
|---|---:|---:|---:|---:|---|
| loomux hook pre-tool-use (Edit auf README.md) | 30,4 ms | 24,5 ms | 23,5 ms | 27,5 ms | [0] |
| ulguard --root (alter Policy-Wächter) | 26,1 ms | 23,7 ms | 23,4 ms | 24,5 ms | [0] |
| brain guard (alte Schreibschranke) | 29,0 ms | 27,3 ms | 26,5 ms | 34,5 ms | [2] |
| beide alten Wächter parallel (wie ein Edit sie trifft) | 37,5 ms | 30,5 ms | 29,5 ms | 32,8 ms | [0 2] |
| beide alten Wächter nacheinander (CPU-Zeit) | 52,6 ms | 51,1 ms | 50,1 ms | 57,1 ms | [0 2] |
| loomux hook session-start | 26,7 ms | 25,0 ms | 24,0 ms | 30,5 ms | [0] |
| loomux version (Startboden) | 28,1 ms | 23,6 ms | 23,0 ms | 24,9 ms | [0] |

### Lesart

1. **Der Zielwert hält mit Abstand.** `hook pre-tool-use` liegt warm bei 24,5 ms
   gegen einen Zielwert von 72 ms und 0,9 ms über seinem eigenen Startboden
   (23,6 ms): der vereinte Wächter entscheidet in unter einer Millisekunde,
   alles andere ist Prozessstart.
2. **Der Vergleich ist nicht der Vergleich vom 2026-09-14, und die Exit-Codes
   sagen warum.** `brain guard` endet hier mit 2, weil dieses Repo nicht in der
   alten Registry steht; es verweigert nach dem Lesen der Registry und tut damit
   weniger als am 2026-09-14, wo dasselbe Binary 72 ms brauchte. Auf einer
   registrierten Area gemessen (`ultra-brain/docs/wiki/index.md`, Exit 0)
   braucht dasselbe getaggte `brain.exe` heute 26,7 ms warm gegen seinen eigenen
   Boden von 24,3 ms (`brain version`). Die 72 / 33 / 73 / 114 ms der Grundlinie
   wurden über 5 Läufe auf einer kälteren Maschine gemessen; gegen die Zahlen
   hier entfällt durch die Fusion weiterhin ein ganzer Prozess: 51,1 ms
   nacheinander und 30,5 ms parallel für zwei Binaries gegen 24,5 ms für eines.
3. **Startzeit: keine `init`-Zeile aus loomux-Code liegt über 1 ms.**
   `GODEBUG=inittrace=1 bin/loomux.exe version` nennt genau zwei Zeilen über der
   Schwelle, und keine davon ist unsere: `github.com/BurntSushi/toml/internal`
   mit 18–22 ms clock (1.673 Allokationen — das Paket löst beim Start die lokale
   Zeitzone auf) und `net` mit 0,5–1,0 ms. Die größte loomux-Zeile ist
   `github.com/xidus90/loomux/internal/config` mit 0,49 ms. Die Startzeit-Regel
   der Spec hält damit. Dagegen gemessen: ein leeres Go-`main` startet auf
   dieser Maschine in 4,5 ms warm; von den ~23,6 ms Boden sind also rund
   4,5 ms Prozessstart und die restlichen ~19 ms das `init` dieser einen
   Abhängigkeit — sie löst die lokale Zeitzone auf (`time.Now().Zone()` in
   `internal/tz.go`) und lädt dafür die Zeitzonendaten des Betriebssystems.
   Dieselben 21 ms stehen im alten `brain.exe`.
4. **Die Schranke selbst, zerlegt.** `LOOMUX_BENCH_REGISTRY=<Kopie>
   LOOMUX_BENCH_TARGET=<Datei in einer registrierten Area> go test
   ./internal/brain/guard/ -run '^$' -bench
   BenchmarkDecideAgainstTheRealRegistry -benchtime 50x -cpuprofile …`
   gegen eine Kopie des echten Zustandsverzeichnisses
   (`%LOCALAPPDATA%\brain`, 10 Areas) misst **916.728 ns/op** für ein `Decide`.
   Das Profil (`go tool pprof -top`) legt alles davon in die Windows-Pfad­auf­lösung:
   `guard.writableRoots` → `finalPath` 66,7 % (ein `CreateFile` plus
   `GetFinalPathNameByHandle` je registrierter Wurzel), das `resolvePath` des
   Ziels selbst 33,3 %, und `runtime.cgocall` trägt 100 % der flachen Samples —
   die TOML-Lesevorgänge von Registry und Manifesten erreichen kein einziges
   Sample. Der vor der Messung notierte Verdacht, ein
   `git rev-parse --git-common-dir` je Ziel in `askGit`, **trat nicht auf**:
   `sameRepository` erreicht ihn nur für ein Ziel unter einem Verzeichnis, das
   ein Manifest und ein eigenes `.git` trägt und nicht die registrierte Wurzel
   selbst ist — was diese Nutzlast nie ist.

Ein Zielwert für Stufe 2 wird hier über die drei benannten Posten hinaus nicht
gesetzt; der Hebel, auf den sie zeigen, ist die Auflösung je Wurzel in
`writableRoots`.

## 2026-09-15 12:00 — Der Exec-Hook gegen einen Function Hook (Claude Mods)

Repo `loomux`, Haupt-Worktree, Commit `311d5b2`, sauberer Baum — diese Messung
ändert keinen Code im Repository; der Probe-Mod, die Probe-Binaries und der
Ersatz-Daemon liegen in einem Scratch-Verzeichnis.

**Ziel.** Entscheiden, ob die in
[anthropics/claude-code#91870](https://github.com/anthropics/claude-code/issues/91870)
vorgeschlagenen Function Hooks („Claude Mods") einen Adapter wert sind. Sie
laufen im Prozess von Claude Code selbst und zahlen deshalb keinen Prozessstart.
Die Frage ist nicht, ob das schneller ist — das ist es bauartbedingt —, sondern
wie viel der heutigen 28 ms der Spawn ist, den wir entfernen würden, was der
Adapter zu einem langlebigen `loomux` stattdessen kostet, und ob dieser Spawn
nicht in Go billig zu machen ist.

**Was das Merkmal ist, am Binary und an den veröffentlichten Deklarationen
nachgerechnet, nicht am Issue.** Claude Code 2.1.272 trägt es hinter
`CLAUDE_CODE_ENABLE_FUNCTION_HOOKS=1` (Rollout-Flag
`tengu_plugin_hooks_modules`). Ein Mod ist ein gewöhnliches Plugin, dessen
`hooks/hooks.json` ein Modul mit `register(on)` nennt; Hooks sind `($, e, next)`
und verschachteln in Registrierungsreihenfolge. Zwei Tatsachen prägen jeden
Adapter, beide aus `anthropics/claude-code:mods/types/claude-code.d.ts`:

- `tool.check` ist genau der Sitz der Schreibschranke: es nimmt `{ tool, input,
  tool_use_id }` und gibt `{ decision: 'allow' | 'ask' | 'deny', reason?,
  rule? }` zurück — dasselbe Urteil, das `hook pre-tool-use` heute schreibt,
  plus `ask`, das der Exec-Hook nicht ausdrücken kann.
- Ein Hook erreicht einen langlebigen Prozess auf zwei Wegen: `$.http.fetch`
  (ein Loopback-Server, den wir betreiben würden) oder `$.mcp.call(server, tool,
  args)` (ein stdio-MCP-Server, den Claude Code startet und besitzt, ohne Port).
  `$.process.run` ist wieder ein Spawn. Gemessen ist unten nur `$.http.fetch`.

**Methode.** Drei Messungen, je ein Glied der Kette.

1. Spawn, wie der Hook ihn heute zahlt, und woraus er besteht:
   `loomux dev bench-hooks -n 20` in einem Durchgang über fünf Fälle — der Hook,
   der Startboden, ein leeres Go-`main`, ein `main`, dessen einziger Import
   `github.com/BurntSushi/toml` v1.6.0 ist, und ein `main`, dessen einzige
   Anweisung `time.Now().Zone()` ist. Alle fünf mit Go 1.27 gebaut, der Hook im
   Haupt-Worktree mit der Pilot-`.loomux/config.toml` an Ort und Stelle und ohne
   `LOOMUX_STATE_DIR`-Überschreibung, Nutzlast ein `Edit` auf die `README.md`
   dieses Repos. Je Fall ein kalter Lauf, dann 20 warme.
2. Dispatch durch die Faltung, in situ: ein Probe-Mod (`register` hängt sich an
   `tool.check`, fragt über `$.http.fetch` und fällt mit `next(e)` durch), unter
   `claude plugin test` gelaufen, 500 Dispatches von `$.tool.check`, der Fetch
   aus dem Speicher beantwortet von einem Hook, den der Test darunter setzt. Der
   Test zählt die Fetches und fordert 501, damit ein still übersprungener Hook
   nicht als schneller durchgeht — der erste Anlauf dieser Messung tat genau
   das, und der Testkasten meldete `no implementation for http.fetch` erst, als
   die Zahl eingefordert wurde.
3. Transport: 200 POSTs von einem Client an einen Go-`net/http`-Server auf
   `127.0.0.1:47613`, der das Urteil so beantwortet wie die Schranke. Gemessen
   von Node 24.14.1 außerhalb von Claude Code, weil der eigene Transportweg
   eines Hooks im Testkasten nicht messbar war (darunter liegt kein echtes
   `http.fetch`). Zwei ungemessene Faktoren liegen zwischen dieser Zahl und der
   echten: Claude Code läuft auf Bun, und `$.http.fetch` geht durch das Noun der
   Engine, nicht durch nacktes `fetch`.

| Fall | kalt (1. Lauf) | warmer Median | warmes Min | warmes Max |
|---|---:|---:|---:|---:|
| `loomux hook pre-tool-use` (Edit auf README.md) | 36,1 ms | 28,0 ms | 27,1 ms | 30,0 ms |
| `loomux version` (Startboden) | 27,5 ms | 27,5 ms | 26,0 ms | 30,0 ms |
| leeres Go-`main` (Spawn-Boden) | 10,6 ms | 8,0 ms | 7,7 ms | 10,0 ms |
| Go-`main`, das nur `BurntSushi/toml` importiert | 93,0 ms | 27,5 ms | 26,0 ms | 32,5 ms |
| Go-`main`, das nur die lokale Zeitzone auflöst | 78,1 ms | 26,7 ms | 25,6 ms | 29,4 ms |
| Mod-`tool.check` durch die Faltung, Fetch aus dem Speicher | — | 0,26 ms | 0,20 ms | 2,14 ms |
| Loopback-Umlauf zu einem Go-Daemon (Node-Client) | 22,4 ms | 0,34 ms | 0,22 ms | 1,53 ms |

Die letzten beiden Zeilen stammen aus Messung 2 und 3 und gehören nicht zum
`bench-hooks`-Durchgang; die ersten fünf sind ein Durchgang, ein Maschinenstand.

### Lesart

1. **Der Function Hook samt Adapter liegt in der Größenordnung 0,6 ms gegen
   28,0 ms.** Das ist die Summe zweier getrennt gemessener Hälften — Dispatch
   0,26 ms in situ, Transport 0,34 ms außerhalb —, also eine Schätzung, keine
   Messung eines Pfades. Je bewachtem Tool-Aufruf sind das ~27 ms. Bevor man das
   dem Mod gutschreibt, Punkt 2 lesen.
2. **19 dieser 28 ms sind Go beim Auflösen der lokalen Zeitzone, und dafür
   braucht es keinen Mod.** Ein `main`, dessen einzige Anweisung
   `time.Now().Zone()` ist, kostet **26,7 ms** gegen **8,0 ms** für ein leeres:
   18,7 ms, auf dem Windows-Weg, der die Zone aus der Registry liest. Der Import
   von `BurntSushi/toml` kostet dieselben 27,5 ms und nichts darüber hinaus, und
   `GODEBUG=inittrace=1 bin/loomux.exe version` nennt
   `github.com/BurntSushi/toml/internal` mit 22 ms Uhrzeit und 1.673 Allocs —
   dieses Paket löst die Zone in seinem `init` auf. **Der Hebel heißt also nicht
   „den TOML-Parser austauschen", sondern „keine lokale Zeit auf dem Hook-Pfad".**
   Jedes Paket und jede Logzeile, die einen lokalen Zeitstempel formatiert, holt
   dieselben 19 ms zurück, einmal je Prozess. Ein Exec-Hook, der die lokale Zone
   nie anfasst, läge bei etwa 9 ms — ~19 der ~27 ms Abstand, in Go geschlossen,
   ohne eine Zeile TypeScript und ohne den portablen Pfad aufzugeben. Nur die
   verbleibenden ~8 ms Windows-Prozessstart brauchen einen In-Process-Hook.
3. **Die Entscheidung selbst liegt weiterhin unter dem Rauschen des Bodens.**
   `hook pre-tool-use` mit 28,0 ms gegen `version` mit 27,5 ms sind 0,5 ms
   auseinander, bei warmen Spannen, die einander fast vollständig überdecken
   (27,1–30,0 gegen 26,0–30,0); in einem zweiten Durchgang lag der Hook
   *unter* dem Boden. Die 24,5 ms vom 2026-09-15 01:20 sind dagegen kein
   Rückschritt: jener Lauf war ein anderer Worktree ohne `.loomux/config.toml`
   (ein TOML-Parse weniger) auf einer ruhigeren Maschine, und `version` wanderte
   mit (23,6 → 27,5 ms). Der Boden ist die ganze Geschichte, in beiden Läufen.
4. **Kalt ist ein Daemon am schlechtesten, und das einmalig.** Der erste
   Loopback-Aufruf kostet 22,4 ms — Client-Aufwärmen und Verbindung. Eine
   Sitzung zahlt ihn einmal; der Exec-Hook zahlt seinen Kaltpreis beim ersten
   Edit und ~28 ms bei jedem weiteren.
5. **Was der Mod kauft, das keine Millisekunden sind.** `ask` als Urteil, eine
   Begründung im Transkript statt einer zurückgegebenen Fehlerzeichenkette, und
   `ui.*`, um den Zustand der Schranke zu zeigen. Dagegen: es gilt nur für
   Claude Code (das Tor `.githooks/pre-commit` und jeder andere Host brauchen
   weiter den Exec-Pfad), die API darf sich zwischen Releases ohne Ankündigung
   ändern, und eine zweite Sprache zieht in einen Baum ein, dessen Entwurf ein
   Go-Binary ist.

**Der Hebel, auf den das zeigt, ist Punkt 2, nicht der Mod.** Die lokale
Zeitzone vom Startpfad fernzuhalten ist eine Go-Änderung in diesem Repository,
messbar mit dem Werkzeug, das schon hier liegt, und sie ist mehr wert als der
Adapter, den sie unattraktiver machen würde.

## 2026-09-15 15:39 — Die Schreibschranke im verknüpften Worktree

Repository `loomux`, Worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1`, Branch
`barrier-worktrees`, Commit `d8bfad2`.

**Ziel.** Zeigen, was das Öffnen verknüpfter Worktrees kostet: ein Write in einem
Worktree ohne eigenen Registry-Eintrag, vor der Änderung (verweigert) und danach
(erlaubt), und ein Write im Hauptcheckout vorher und nachher, den die Änderung
unberührt lassen muss.

**Methode.** `loomux dev bench-hooks testdata/bench/barrier-worktrees.json -n 20`:
je Fall ein kalter Lauf, dann 20 warme. `LOOMUX_STATE_DIR` zeigt auf eine Kopie der
Registry, die nur den Hauptcheckout registriert (`workspace = true`). Binaries:
`before.exe` gebaut aus `b55ae6d`, `after.exe` gebaut aus dem Commit oben, beide
mit Go `go1.27.0 windows/amd64`.

| Fall | kalt (1. Lauf) | warmer Median | warmes Min | warmes Max | Exit-Codes |
|---|---:|---:|---:|---:|---|
| before: Write in linked worktree, no registry entry | 180,4 ms | 66,8 ms | 62,0 ms | 84,6 ms | [2] |
| after: Write in linked worktree, no registry entry | 177,3 ms | 65,3 ms | 62,4 ms | 69,8 ms | [0] |
| before: Edit on README.md in main checkout | 33,2 ms | 31,9 ms | 29,0 ms | 40,4 ms | [0] |
| after: Edit on README.md in main checkout | 34,0 ms | 31,0 ms | 27,1 ms | 32,5 ms | [0] |

### Lesart

1. **Worktree: das Öffnen kostet nichts Messbares.** Nachher gegen vorher sind
   warm 65,3 ms gegen 66,8 ms (1,5 ms weniger) und kalt 177,3 ms gegen 180,4 ms
   (3,1 ms weniger). Die warme Spanne nachher (62,4–69,8) liegt innerhalb der
   Spanne vorher (62,0–84,6). Die beiden Läufe tun verschiedene Arbeit: vorher
   geht die Ablehnung den ganzen Weg bis zur Meldung und sucht dabei das
   Review-Zentrum; nachher versucht die Schranke je Elternverzeichnis einen
   `.git`-Lesezugriff und liest die drei Zeigerdateien (`.git`, `gitdir`,
   `commondir`) nur an der Worktree-Wurzel.
2. **Hauptcheckout: unberührt, wie er sein muss.** Nachher gegen vorher sind warm
   31,0 ms gegen 31,9 ms (0,9 ms weniger) und kalt 34,0 ms gegen 33,2 ms (0,8 ms
   mehr), bei warmen Spannen, die einander überlappen (27,1–32,5 gegen 29,0–40,4).
   Dieser Unterschied ist Rauschen: `Decide` kehrt vor der Worktree-Suche zurück,
   weil das Ziel in einem registrierten Baum liegt.
3. **Gegen den Zielwert von 72 ms, und was die Abstände zwischen den Zeilen
   bedeuten.** Warm bleiben beide Writes unter dem Zielwert: der Worktree-Write mit
   65,3 ms (6,7 ms darunter), der Write im Hauptcheckout mit 31,0 ms (41,0 ms
   darunter). Die Kaltwerte der beiden Gruppen sind nicht vergleichbar. Die
   Worktree-Fälle liefen zuerst, ihr kalter Lauf (180,4 und 177,3 ms) ist also
   zugleich der erste Start des jeweils frisch gebauten Binarys; die kalten Läufe
   im Hauptcheckout (33,2 und 34,0 ms) nutzten die bereits zwischengespeicherten
   Binaries. Der Kaltwert sagt deshalb nichts über den Worktree-Write allein. Der
   warme Abstand von rund 34–35 ms zwischen Worktree- und Hauptcheckout-Zeilen
   (vorher 66,8 gegen 31,9, nachher 65,3 gegen 31,0) besteht in beiden Binaries,
   diese Änderung verursacht ihn also nicht. Nach dem Codepfad ist er vereinbar mit
   den zwei `git rev-parse`-Aufrufen, die `declaredWikiRoot` → `sameRepository` für
   einen Worktree macht, dessen `.loomux/config.toml` einen registrierten Scope
   nennt; der Hauptcheckout überspringt sie, weil sein Pfad dem registrierten
   gleicht (`path.go:468`). Diese Zuordnung hat dieser Durchgang nicht gemessen.
   Der Eintrag vom 2026-09-16 19:32 unten misst sie.

## 2026-09-16 19:10 — Die Brain-Datenbefehle auf der echten Registry

Repo `loomux`, Worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1`, Zweig
`sdd-1b-1`, Commit `9ca6364` plus der uncommittete Baum von Task 15 — dieser Task
fügt die Messfälle, drei Benchmarks und diesen Eintrag hinzu; er ändert keinen Befehl.

**Ziel.** Zeigen, was `loomux brain search` warm und kalt je Profil kostet, gemessen am
Zielwert ≤ 150 ms für `--profile fast` warm Ende zu Ende (Spec: 63–83 ms qmd, 5–17 ms
Probe und Handshake, rund 35 ms Go-Startboden), mit dem Anteil fürs Lesen aller
Identitätsregister eigens ausgewiesen; was `loomux brain status` kostet; und ob der
Einzug der Brain-Pakete Startzeit hinzugefügt hat.

**Methode.** `loomux dev bench-hooks testdata/bench/1b-1-brain.json -n 20` gegen die
echten Zustandsverzeichnisse (`%LOCALAPPDATA%\loomux` mit 11 Bereichen,
`%LOCALAPPDATA%\brain` für die Artefakte schreibgeschützter Bereiche und den
Reconcile-Stempel; weder `LOOMUX_STATE_DIR` noch `LOOMUX_LEGACY_BRAIN_DIR` gesetzt),
Anfrage `latenz`, nach einem Aufwärmaufruf je Profil: Die Kalt-Spalte ist ein kalter
Prozess gegen einen warmen qmd-Daemon. Der kalte Daemon ist getrennt gemessen: Daemon
gestoppt, dann ein zeitgemessenes `brain search` je Profil, dreimal; jeder der neun
Läufe druckte den Aufwärm-Hinweis, hat den Daemon also selbst gestartet. Der
Registeranteil: `go test ./internal/brain/search/ -run '^$' -bench 'OfTheRealRegistry|WithoutTheEngine' -benchtime 50x -benchmem`
mit `LOOMUX_BENCH_REGISTRY`/`LOOMUX_BENCH_LEGACY` auf denselben Verzeichnissen.
Startzeit: `GODEBUG=inittrace=1 loomux --version`, je drei Läufe, am Binary des Commits
vor Task 3 (`aa945cb^`) und an `bin/loomux.exe`. Maschine: AMD Ryzen 7 9800X3D,
Go `windows/amd64`, GOMAXPROCS 16, qmd 2.8.3 auf CUDA (der Vorgabe-Backbone von
loomux; weder `QMD_LLAMA_GPU` noch `QMD_FORCE_CPU` gesetzt).

**Was diese Zahlen unvergleichbar macht.** Dreierlei. (1) `qmd mcp stop` konnte den Daemon
nicht beenden: Ein einziges `qmd status` löscht `~/.cache/qmd/mcp.pid`, während der
Daemon weiterläuft; der Stopp antwortet dann `Not running (no PID file).`, und der
nächste Lauf misst einen warmen Daemon. Die neun kalten Läufe haben darum den Prozess
auf Port 8765 unmittelbar gestoppt, und jeder wurde am Aufwärm-Hinweis geprüft.
(2) Elf weitere qmd-MCP-Daemons früherer Sitzungen lagen die ganze Zeit auf anderen
Ports; ein kalter `fast`-Lauf starb mit einem CUDA-Fehler (`ggml-cuda.cu:106`) und Exit 1
und wurde wiederholt. (3) Die Reihenfolge der Fälle: `version` lief zuerst und war aus
den drei Aufwärmaufrufen schon warm, `status` zuletzt.

| Fall | kalt (1. Lauf) | warmer Median | warmes Min | warmes Max | Exit-Codes |
|---|---:|---:|---:|---:|---|
| loomux version (Startboden) | 45,0 ms | 30,2 ms | 27,7 ms | 44,0 ms | [0] |
| loomux brain search latenz --profile keyword (warmer Daemon) | 79,0 ms | 67,9 ms | 63,9 ms | 106,6 ms | [0] |
| loomux brain search latenz --profile fast (warmer Daemon) | 284,8 ms | 260,7 ms | 236,0 ms | 373,7 ms | [0] |
| loomux brain search latenz --profile full (warmer Daemon) | 805,4 ms | 834,9 ms | 752,8 ms | 940,0 ms | [0] |
| loomux brain status | 2458,3 ms | 2396,7 ms | 2300,9 ms | 2979,8 ms | [0] |

| kalter Daemon (vor jedem Lauf gestoppt) | Lauf 1 | Lauf 2 | Lauf 3 |
|---|---:|---:|---:|
| brain search latenz --profile keyword | 1,15 s | 0,90 s | 0,89 s |
| brain search latenz --profile fast | 11,62 s (Wiederholung) | 4,34 s | 7,71 s |
| brain search latenz --profile full | 4,36 s | 4,41 s | 11,92 s |

| Benchmark (50 Läufe) | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| VisibleAreasOfTheRealRegistry | 1.236.552 | 236.531 | 2.213 |
| RegistersOfTheRealRegistry (11 Register) | 813.614 | 610.488 | 1.121 |
| ExecuteSearchWithoutTheEngine | 1.971.952 | 868.143 | 3.574 |

### Lesart

1. **Der Zielwert hält nicht, und qmd hält ihn offen.** `--profile fast` warm liegt bei
   **260,7 ms** gegen den Zielwert 150 ms — 110,7 ms darüber, mit einer warmen Spanne von
   236,0–373,7 ms, die ihn nie erreicht. Der Posten, der ihn reißt, ist nicht Go: Der
   Startboden (`loomux version`) sind 30,2 ms, alles, was loomux außer Prozessstart und
   qmd tut, sind 1,97 ms (`ExecuteSearchWithoutTheEngine`), und derselbe Aufruf auf dem
   Keyword-Weg kostet 67,9 ms. Der Anteil von qmd an der Antwort sind damit rund 228 ms —
   260,7 abzüglich des Startbodens und der 1,97 ms von oben — gegen die 63–83 ms, die die
   Spec für den ganzen qmd-Aufruf veranschlagt hat; rund 193 ms dieses Anteils sind das,
   was die Vektorsuche gegenüber dem Keyword-Weg hinzufügt. Ein Umbau folgt in diesem
   Task nicht.
2. **Die Register sind 0,3 % einer fast-Antwort.** `RegistersOfTheRealRegistry` liest die
   `_identities.tsv` aller **11** registrierten Bereiche in **0,81 ms**; am warmen
   fast-Median von 260,7 ms sind das 0,31 %. `VisibleAreasOfTheRealRegistry` — die
   Registry plus je Bereich ein Manifest — sind 1,24 ms. Beide zusammen (2,05 ms) machen
   im Rahmen der Streuung schon das ganze `ExecuteSearchWithoutTheEngine` (1,97 ms) aus:
   Registry, Manifeste und Register lesen *ist* das, was loomux außer Prozessstart und
   qmd verbringt.
3. **Kalter Daemon: Das Modell-Laden beherrscht die Zahl und streut.** Gegen die
   23 ms / 5,4 s / 11,9 s des Spikes für einen frisch gestarteten Daemon stehen je Profil
   0,89–1,15 s (keyword), 4,34–11,62 s (fast) und 4,36–11,92 s (full). Die Streuung
   innerhalb eines Profils ist größer als der Abstand zwischen `fast` und `full`; diese
   neun Werte ordnen also nichts: Sie messen einen Daemon-Start samt Modell-Laden, und das
   schwankte um den Faktor drei auf einer Maschine mit elf weiteren liegenden Daemons.
   Ablesbar ist keyword: Es braucht kein Einbettungsmodell und kostet trotzdem rund 1 s,
   weil der Daemon überhaupt hochkommen muss. Jeder Wert bleibt unter seiner geplanten
   Schranke (6,3 s / 22,4 s / 41,9 s). Der warme `full`-Median oben ist keine neue
   Anfrage: Die Messung wiederholt `latenz` 21-mal, was der Spike mit rund 250 ms und
   dieser Lauf mit 834,9 ms gemessen hat, während eine neue Anfrage an einen warmen
   Daemon 4,2–7,9 s brauchte.
4. **`brain status` sind 2,4 s, und nichts davon ist Go.** Warmer Median 2396,7 ms gegen
   einen Startboden von 30,2 ms: Der Befehl fragt die qmd-CLI einmal je indiziertem
   Bereich und einmal nach dem Rückstand, ist also an qmd-Prozessstarts gebunden.
5. **Der Einzug der Brain-Pakete hat keine Startzeit gekostet.** Nur eine `init`-Zeile
   erreicht 1 ms, vor wie nach dem Einzug: `github.com/BurntSushi/toml/internal` mit
   21/20/20 ms clock davor und 18/18/22 ms clock danach (71.264 bytes, und 71.280 im
   letzten Lauf davor; 1.673 allocs durchgehend). **Keine Zeile aus
   `github.com/xidus90/loomux/...` erreicht 1 ms.** Neu nach
   dem Einzug, keine davon über 0 ms clock außer einem Lauf von `internal/brain/search`
   mit 0,50 ms: `internal/brain/catalog` und `internal/brain/search` (die umgezogenen
   `regexp.MustCompile`-Paketvariablen), `internal/dev/mutants` und die fünf
   `golang.org/x/text`-Pakete `unicode/norm`, `internal/language`,
   `internal/language/compact`, `language` und `cases`, die das Binary vor Task 3 gar
   nicht linkt. Step 10 hat darum nicht gegriffen; die sieben Quelldateien mit einer
   `regexp.MustCompile`-Paketvariablen sind genau die, die der Plan nennt.

## 2026-09-16 19:32 — sameRepository ohne git rev-parse

Repository `loomux`, Branch `claude/cranky-kilby-f5a456`, Commit `ee1aadf`;
gemessener Worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1` auf `9ca6364`.

**Ziel.** Die Zuordnung prüfen, die der Eintrag vom 2026-09-15 15:39 offen ließ:
dass der warme Abstand von rund 34 ms zwischen einem Write in einem verknüpften
Worktree und einem im Hauptcheckout die zwei `git rev-parse --git-common-dir`-Aufrufe
sind, die `sameRepository` machte. Diese Änderung liest stattdessen gits
Zeigerdateien.

**Methode.** `loomux dev bench-hooks testdata/bench/barrier-worktrees.json -n 20`:
je Fall ein kalter Lauf, dann 20 warme. `LOOMUX_STATE_DIR` zeigt auf eine Kopie der
Registry, die nur den Hauptcheckout registriert (`workspace = true`). Binaries:
`before.exe` gebaut aus `3855de4` (Code identisch mit `e4e0dc2`), `after.exe`
gebaut aus dem Commit oben, beide mit Go `go1.27.0 windows/amd64`. Beide Binaries
erlauben den Worktree-Write; die Fallnamen sind die vom 2026-09-15. Die Tabelle
zeigt den zweiten von zwei Läufen. Der erste (19:31) wurde zusammen mit dem Bau
von `after.exe` gestartet und hat sich womöglich mit ihm überschnitten, er ist
deshalb nicht gezeigt; seine warmen Mediane liegen höchstens 3,3 ms neben diesen
(Worktree 73,8 vorher / 31,3 nachher, Hauptcheckout 30,3 vorher / 32,0 nachher).
Weil beide Binaries schon einmal gestartet waren, sind alle vier Kaltwerte hier
Starts aus dem Cache und zeilenübergreifend vergleichbar.

| Fall | kalt (1. Lauf) | warmer Median | warmes Min | warmes Max | Exit-Codes |
|---|---:|---:|---:|---:|---|
| before: Write in linked worktree, no registry entry | 79,4 ms | 72,8 ms | 70,6 ms | 94,4 ms | [0] |
| after: Write in linked worktree, no registry entry | 37,9 ms | 34,6 ms | 31,4 ms | 41,6 ms | [0] |
| before: Edit on README.md in main checkout | 34,2 ms | 31,4 ms | 28,2 ms | 34,4 ms | [0] |
| after: Edit on README.md in main checkout | 32,2 ms | 30,5 ms | 28,6 ms | 35,9 ms | [0] |

### Lesart

1. **Worktree: der Write fällt auf weniger als die Hälfte.** Nachher gegen vorher
   sind warm 34,6 ms gegen 72,8 ms (38,2 ms weniger) und kalt 37,9 ms gegen
   79,4 ms (41,5 ms weniger). Die warmen Spannen überlappen nicht (31,4–41,6 gegen
   70,6–94,4). Der Abstand zum Hauptcheckout war vorher 41,4 ms (72,8 gegen 31,4)
   und ist nachher 4,1 ms (34,6 gegen 30,5), bei überlappenden warmen Spannen
   (31,4–41,6 gegen 28,6–35,9). Diese 4,1 ms liegen in der Schwankung zwischen
   Läufen: im ersten Lauf lag der Worktree-Write nachher (31,3 ms) unter dem Write
   im Hauptcheckout nachher (32,0 ms).
2. **Hauptcheckout: Rauschen.** Nachher gegen vorher sind warm 30,5 ms gegen
   31,4 ms (0,9 ms weniger) und kalt 32,2 ms gegen 34,2 ms (2,0 ms weniger), bei
   warmen Spannen, die einander überlappen (28,6–35,9 gegen 28,2–34,4).
   `sameRepository` kehrt dort zurück, bevor es eine von gits Dateien liest, weil
   der Pfad des Checkouts dem registrierten gleicht.
3. **Die Zuordnung: bestätigt.** Der Worktree-Write fällt warm um 38,2 ms, etwa um
   den Abstand, den er zum Hauptcheckout hatte (41,4 ms in diesem Lauf, rund
   34–35 ms am 2026-09-15), und was vom Abstand bleibt, liegt im Rauschen. Der
   Abstand waren die zwei `git rev-parse`-Aufrufe. Gegen den Zielwert von 72 ms:
   der Worktree-Write nachher liegt 37,4 ms darunter. `before.exe` liegt in diesem
   Lauf 0,8 ms darüber (72,8 ms) und 7,5 ms über den 65,3 ms des `after.exe` vom
   2026-09-15, gebaut aus `d8bfad2`, das den Worktree-Write ebenfalls erlaubte.
   Diese Zeile lief dasselbe Urteil und dieselben zwei `git rev-parse`-Aufrufe:
   zwischen `d8bfad2` und `e4e0dc2` ist der einzige Code-Commit `c2e172d`, der
   `linkedCommon` ein `Lstat` und einen Pfadvergleich hinzufügt. Die 7,5 ms sind
   ungeklärt. Die Hauptcheckout-Zeilen zeigen keine allgemeine Verlangsamung: sie
   sind etwas schneller als am 2026-09-15 (vorher 31,4 gegen 31,9, nachher 30,5
   gegen 31,0). Eine andere Sitzung im gemessenen Worktree würde zu einer
   Verlangsamung passen, die nur den Worktree trifft, bleibt aber eine offene
   Vermutung. Die Zuordnung stützt sich auf den Abstand innerhalb dieses Laufs,
   vorher 41,4 ms und nachher 4,1 ms.

## 2026-09-16 22:13 — Registry- und Deklarationsprüfungen an einer Stelle

Repository `loomux`, Worktree
`C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/recursing-bartik-b2d7a1`,
Branch `claude/recursing-bartik-b2d7a1`, Commit `222465b` (Task 5 des Plans
`2026-09-16-loomux-registry-manifest-pruefungen`).

**Ziel.** Zeigen, was die strengen Leser der Registry und der Bereichsdeklarationen
an der Schreibschranke und an `brain catalog` kosten.

**Methode.** `loomux dev bench-hooks testdata/bench/registry-checks.json -n 20`: je
Fall ein kalter Lauf, dann 20 warme, gegen das echte Zustandsverzeichnis
(`%LOCALAPPDATA%\loomux` mit 11 Bereichen, kein `LOOMUX_STATE_DIR`); der Schrankenfall
editiert `C:/Users/micro/Documents/#GIT/loomux/README.md` mit `--root` auf dem
Hauptcheckout. Binaries: `loomux-before.exe` gebaut aus `219ccb1` (dem Plan-Commit vor
Task 2), `bin/loomux.exe` gebaut vom Pre-Commit-Gate auf `222465b`. Go-Benchmarks, je
fünf Läufe, Median der fünf:
`go test ./internal/brain/guard/ -run '^$' -bench DecideAgainstTheRealRegistry -benchtime 50x -benchmem -count 5`
mit `LOOMUX_BENCH_REGISTRY` auf einer Kopie von `%LOCALAPPDATA%\loomux` und
`LOOMUX_BENCH_TARGET=C:/Users/micro/Documents/#GIT/loomux/README.md`, sowie
`go test ./internal/brain/search/ -run '^$' -bench VisibleAreasOfTheRealRegistry -benchtime 50x -benchmem -count 5`
mit `LOOMUX_BENCH_REGISTRY` auf derselben Kopie und `LOOMUX_BENCH_LEGACY` auf
`%LOCALAPPDATA%\brain`. Für die Go-Benchmarks lief „vorher“ in Task 1 auf
`219ccb1` und „nachher“ auf `222465b`, beide auf derselben Zustandskopie; die
bench-hooks-Zeilen vorher/nachher liefen in einem Lauf gegen das echte
Zustandsverzeichnis. Maschine: AMD Ryzen 7 9800X3D, Go
`go1.27.0 windows/amd64`, GOMAXPROCS 16.

| Fall | kalt (1. Lauf) | warmer Median | warmes Min | warmes Max | Exit-Codes |
|---|---:|---:|---:|---:|---|
| before: loomux hook pre-tool-use (Edit on README.md, real registry) | 73,7 ms | 35,5 ms | 33,1 ms | 47,7 ms | [0] |
| after: loomux hook pre-tool-use (Edit on README.md, real registry) | 36,0 ms | 35,7 ms | 32,6 ms | 44,4 ms | [0] |
| before: loomux brain catalog (real registry) | 39,5 ms | 35,6 ms | 33,0 ms | 44,5 ms | [0] |
| after: loomux brain catalog (real registry) | 35,0 ms | 37,0 ms | 31,9 ms | 42,5 ms | [0] |

| Benchmark | ns/op vorher | ns/op nachher | B/op vorher | B/op nachher | allocs/op vorher | allocs/op nachher |
|---|---:|---:|---:|---:|---:|---:|
| DecideAgainstTheRealRegistry | 1.596.244 | 1.384.310 | 137.694 | 140.757 | 1.277 | 1.327 |
| VisibleAreasOfTheRealRegistry | 1.217.696 | 1.432.590 | 237.694 | 226.964 | 2.213 | 2.079 |

### Lesart

1. **Schranke, Ende zu Ende.** Nachher gegen vorher sind 35,7 ms gegen 35,5 ms warm
   (0,2 ms mehr), mit überlappenden warmen Spannen (32,6–44,4 gegen 33,1–47,7).
   Kalt sind es nachher 36,0 ms gegen 73,7 ms (37,7 ms weniger).
2. **`brain catalog`, Ende zu Ende.** Nachher gegen vorher sind 37,0 ms gegen
   35,6 ms warm (1,4 ms mehr), mit überlappenden warmen Spannen (31,9–42,5 gegen
   33,0–44,5). Kalt sind es nachher 35,0 ms gegen 39,5 ms (4,5 ms weniger).
3. **`DecideAgainstTheRealRegistry`.** 1.384.310 gegen 1.596.244 ns/op (211.934 ns
   oder 13,3 % weniger); die fünf Läufe überlappen nicht (1.361.546–1.452.418 gegen
   1.502.812–1.720.166). 140.757 gegen 137.694 B/op (2,2 % mehr), 1.327 gegen
   1.277 allocs/op (50 oder 3,9 % mehr).
4. **`VisibleAreasOfTheRealRegistry`.** 1.432.590 gegen 1.217.696 ns/op
   (214.894 ns oder 17,6 % mehr); die fünf Läufe überlappen nicht
   (1.395.594–1.503.752 gegen 1.146.422–1.388.878). 226.964 gegen 237.694 B/op
   (4,5 % weniger), 2.079 gegen 2.213 allocs/op (134 oder 6,1 % weniger).
5. **Gegen die Grenzen des Plans** (warmer Median der Schranke höchstens 3 ms mehr,
   Go-Benchmarks höchstens 20 % mehr): Schranke +0,2 ms,
   `DecideAgainstTheRealRegistry` −13,3 %, `VisibleAreasOfTheRealRegistry` +17,6 %.

### Ursache

Untersucht am 2026-09-16 um 22:20 nach dem Branch-Review. Ein Worktree von
`219ccb1` und der Branch auf `e01977b` (seit `222465b` ohne Codeänderung) liefen
den obigen `VisibleAreasOfTheRealRegistry`-Befehl abwechselnd, drei Runden mit je
fünf Läufen pro Seite, mit derselben Umgebung. Vorher: Rundenmediane 1.147.168,
1.163.598 und 1.421.438 ns/op; Median aller 15 Läufe 1.178.826
(1.096.538–1.614.158). Nachher: 1.127.922, 1.123.730 und 1.120.828 ns/op; Median
aller 15 Läufe 1.123.730 (1.076.292–1.323.442). Die +17,6 % lassen sich nicht
wiederholen: Die dritte Vorher-Runde springt auf unverändertem Code genauso, also
stammt der Sprung vom Zustand der Maschine während des Laufs, nicht von der
Änderung. CPU-Profile mit 3.000 Iterationen, je Seite zwei Läufe zusammengeführt,
vorher gegen nachher: `ReadAreaManifestUntilStage4` kumuliert 6,41 s gegen 6,42 s,
alle Systemaufrufe 5,13 s gegen 4,99 s, TOML-Dekodierung 1,70 s gegen 1,71 s,
`ReadRegistry` 0,73 s gegen 0,67 s. Die Reihenfolge Stat vor Lesen verschiebt Zeit
von fehlschlagenden Öffnungen der fehlenden Namen (`os.Open` 2,50 s → 1,07 s) zu
Stats (`os.Stat` 1,20 s → 2,55 s) und lässt die Summe gleich. Am Code wurde nichts
geändert.

## 2026-09-17 14:16 — Der Startpfad ohne lokale Zeitzone

Repository `loomux`, Worktree `.claude/worktrees/recursing-bartik-b2d7a1`, Branch
`perf-lazy-local-zone` auf `6cafdd8`. Die Änderung: `go.mod` ersetzt
`github.com/BurntSushi/toml` v1.6.0 durch eine gekürzte Kopie unter
`third_party/toml`, deren `internal/tz.go` die drei lokalen Zonen beim ersten
Gebrauch baut statt in einer Paketvariable.

**Ziel.** Die 18,7 ms, die der Eintrag vom 2026-09-15 12:00 dem Auflösen der
lokalen Zeitzone zuschrieb, vom Hook-Pfad nehmen und prüfen, dass sonst nichts auf
dem Pfad sie zur Laufzeit auflöst.

**Methode.** `loomux dev bench-hooks testdata/bench/lazy-local-zone.json -n 20`,
ein Durchgang, je Fall ein kalter Lauf und 20 warme. `before.exe` ist aus `master`
auf `6cafdd8` gebaut, `after.exe` aus dem Branch, beide mit Go 1.27, kopiert nach
`%TEMP%\loomux-zone-bench`. Der Hook läuft im Hauptcheckout mit dessen
Pilot-`.loomux/config.toml` und der echten Registry, Nutzlast
`testdata/bench/edit-readme-main.json`. Die Böden sind ein leeres `main` und ein
`main`, dessen einzige Anweisung `time.Now().Zone()` ist. Weder
`.loomux/config.toml` noch `registry.toml` enthält einen Datums- oder Zeitwert,
also fragt kein Parse auf diesem Pfad die verzögerten Zonen an.

| Fall | kalt (1. Lauf) | warm Median | warm Min | warm Max | Exit-Codes |
|---|---:|---:|---:|---:|---|
| vorher: loomux hook pre-tool-use (Edit auf README.md) | 51,9 ms | 26,5 ms | 25,5 ms | 32,5 ms | [0] |
| nachher: loomux hook pre-tool-use (Edit auf README.md) | 9,5 ms | 7,5 ms | 7,0 ms | 8,0 ms | [0] |
| vorher: loomux version (Startboden) | 25,5 ms | 24,1 ms | 23,1 ms | 25,0 ms | [0] |
| nachher: loomux version (Startboden) | 7,0 ms | 5,5 ms | 5,5 ms | 6,0 ms | [0] |
| leeres Go-`main` (Spawn-Boden) | 48,0 ms | 4,5 ms | 4,5 ms | 5,7 ms | [0] |
| Go-`main`, das nur die lokale Zone auflöst | 75,6 ms | 23,9 ms | 22,4 ms | 32,1 ms | [0] |

### Lesart

1. **Der Hook, Ende zu Ende.** Nachher gegen vorher sind 7,5 ms gegen 26,5 ms warm
   (19,0 ms weniger), mit getrennten warmen Spannen (7,0–8,0 gegen 25,5–32,5).
   Kalt sind es 9,5 ms gegen 51,9 ms.
2. **Der Startboden hat sich um denselben Betrag bewegt.** `version` braucht
   5,5 ms gegen 24,1 ms warm (18,6 ms weniger), 1,0 ms über dem leeren `main`.
   Das `main` nur mit der Zone kostet in diesem Durchgang 19,4 ms mehr als das
   leere, gegen 18,7 ms am 2026-09-15; die Ersparnis ist genau diese Last.
3. **Nichts auf dem Pfad löst die Zone später auf.** Der Hook liegt nach der
   Änderung 2,0 ms über seinem Boden, vorher 2,4 ms; eine Auflösung zur Laufzeit
   zeigte sich wieder als die 19 ms.
4. **Der Init-Trace, Allokationen statt Uhrzeit.**
   `GODEBUG=inittrace=1 loomux version`: `github.com/BurntSushi/toml/internal`
   fällt von 20 ms Uhrzeit und 1.673 Allokationen auf 0 ms und 2 Allokationen.
   Der Test `TestStartDoesNoWorkInPackageInit` in `cmd/loomux` baut jetzt das
   Binary und schlägt fehl, sobald ein Paket-Init mehr als 500 Allokationen
   macht; eine Abhängigkeit, die die Zone zurückbringt, fängt also das Tor, nicht
   erst die nächste Messung.

## 2026-09-17 20:30 — Der Startboden mit gelinktem MCP-SDK

Repository `loomux`, Worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b-2`,
Branch `sdd-1b-2`, vorher `48e5d38`. Stufe 1b-2, Task 1: `go-sdk` v1.8.0 kommt
herein, `x/sys` steigt auf v0.48.0 und `x/text` auf v0.42.0, die `go`-Direktive
auf 1.26.0.

**Ziel.** Den Startboden festhalten, gegen den der MCP-Dienst gemessen wird, und
sagen, was daran die Abhängigkeitsanhebung ist und was der SDK selbst. Der SDK
ist die erste Abhängigkeit dieser Größe im Binary, und jeder Hook zahlt seinen
Start.

**Methode.** PowerShell, ein `Measure-Command` je Lauf, zwölf Läufe je Binary:
der erste ist der Kaltwert, der Median der übrigen elf der warme. In jedem Fall
`loomux --version`, Go `go1.27.0 windows/amd64`. Drei Binaries: `before` ist
`bin/loomux.exe`, wie `48e5d38` es baute; `bump only` ist ein Bau im
Kratzverzeichnis nach der Anhebung, bei dem nichts den SDK importiert, sodass der
Linker ihn wegwirft; `after` ist `bin/loomux.exe`, wie das Tor es neu baute, mit
`cmd/loomux`, das `internal/mcptools` importiert, und damit gelinktem SDK. Die
Kaltwerte sind erste Läufe in einer schon warmen Shell und zeilenübergreifend
nicht vergleichbar; sie stehen da, gelesen werden sie nicht.

| Fall | Größe | kalt (1. Lauf) | warmer Median | warmes Min | warmes Max |
|---|---:|---:|---:|---:|---:|
| before: `48e5d38` | 13.736.448 B | 18,9 ms | 7,4 ms | 7,3 ms | 7,8 ms |
| bump only: SDK vom Linker verworfen | 13.755.904 B | 46,8 ms | 7,6 ms | 7,3 ms | 8,3 ms |
| after: SDK gelinkt | 16.004.608 B | 45,7 ms | 7,6 ms | 7,4 ms | 8,0 ms |

Paket-Inits über 100 Allokationen, aus `GODEBUG=inittrace=1 loomux version`
auf `after`:

| Paket | Allokationen |
|---|---:|
| `encoding/gob` | 369 |
| `github.com/google/jsonschema-go/jsonschema` | 298 |
| `gopkg.in/yaml.v3` | 274 |
| `github.com/modelcontextprotocol/go-sdk/mcp` | 183 |
| `github.com/xidus90/loomux/internal/brain/wiki` | 131 |
| `github.com/xidus90/loomux/internal/brain/search` | 103 |

### Lesart

1. **Die Startzeit bewegt sich nicht.** Nachher gegen vorher sind warm 7,6 ms
   gegen 7,4 ms (0,2 ms mehr), bei warmen Spannen, die einander fast ganz
   überlappen (7,4–8,0 gegen 7,3–7,8). Das Binary mit bloßer Anhebung liegt bei
   denselben 7,6 ms, also sind auch diese 0,2 ms nicht der SDK. Der Init des SDK
   selbst sind 183 Allokationen und keine messbare Uhrzeit.
2. **Die Größe bewegt sich, und der SDK ist die ganze Bewegung.** Nachher gegen
   vorher sind es 16.004.608 B gegen 13.736.448 B, 2.268.160 B mehr, 16,5 %.
   Davon gehören 19.456 B der Abhängigkeitsanhebung und 2.248.704 B dem gelinkten
   SDK: `mcp`, `jsonschema-go`, `segmentio/encoding`, `uritemplate`, `x/oauth2`
   und `encoding/gob`.
3. **Die Allokationsgrenze hält, mit Luft.** Der größte Init ist `encoding/gob`
   mit 369, unter den 500, die `TestStartDoesNoWorkInPackageInit` zulässt, und
   nahe den 367, die der Plan notiert hat. `gob` ist neu auf diesem Pfad: es kommt
   mit dem SDK herein, ebenso `jsonschema-go` mit 298. Beide liegen unter der
   Grenze, aber unter der Grenze stehen jetzt zwei Pakete, die kein loomux-Code
   unmittelbar verlangt hat, und eine weitere SDK-Abhängigkeit könnte eines
   darüber schieben.
## 2026-09-18 11:10 — Die Graph-Befehle, kalt und warm, und die Schuld aus G1

Repository `loomux`, Worktree `C:/Users/micro/Documents/#GIT/loomux-code-g2`,
Branch `code-g2`, Commit `75c8136`. Diese Aufgabe ergänzt zwei Benchmark-Dateien
und diesen Eintrag; sie ändert keinen Produktionscode.

**Ziel.** Vier Zahlen, die Aufgabe 10 schuldete. Erstens die kalte Zahl zur
Dangling-Masse-Optimierung in `internal/code/pagerank`, die der Eintrag vom
2026-09-16 offen ließ (nur die warme ~9 ms gepoolt gegen ~4,5 s je Dangling-Knoten
existierte). Zweitens und drittens `loomux graph build` auf diesem Repository,
kalt und warm, gegen den ~27–32-ms-Parse-Boden aus §11 der Spezifikation
(254 Dateien; siehe die Korrektur unter „Lesart", Punkt 2, unten — die Zahl,
die ich für diesen Boden zuerst hatte, war über einen verdoppelten Baum
gemessen). Viertens `internal/code/freshness.Probe` allein, gegen die ~3 ms der
Referenzimplementierung für 280 Dateien — die Zahl, von der §7.1 der
Design-Spec seine Entscheidung über die Dateimenge abhängig macht.

**Methode.**

*Dangling-Masse, kalt.* `internal/code/pagerank/dangling_bench_test.go` gab es
vor dieser Aufgabe nicht; sie enthält jetzt `BenchmarkDanglingPooled` (ruft das
produktive `Rank`) und `BenchmarkDanglingPerNode` (eine Kopie von `Rank`s
Schleife mit der einen Zeile, die die Optimierung ersetzt hat: die
Dangling-Masse wird je Dangling-Knoten einzeln zurückgegeben statt gepoolt und
in einem Durchgang verteilt). Beide nutzen dieselbe Fixtur wie
`TestRankBroadSeedsOnMostlyDanglingGraph`: 20.000 Knoten, eine Kette aus 100,
der Rest dangling, jeder Knoten geseedet. Kalt heißt ein Prozess je Messung,
nicht mehrere Runden in einem:

```
$ go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPooled -benchtime 1x -count 1
BenchmarkDanglingPooled-16    	       1	   5875500 ns/op

$ go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPerNode -benchtime 1x -count 1
BenchmarkDanglingPerNode-16    	       1	3299862800 ns/op
```

*`graph build`, kalt und warm.* Ein aus diesem Commit gebautes Binary
(`go build -o /tmp/loomux-bench.exe ./cmd/loomux`), gegen dieses Repository
gefahren. Kalt heißt: Graph und Frischeakte vorher entfernt, sodass die
gemessene Zeit das Schreiben beider auf einen Baum einschließt, der keins von
beiden hatte:

```
$ rm -rf .loomux/state/graph && time /tmp/loomux-bench.exe graph build --root .
254 files, 2801 nodes, 8980 edges (2547 contains, 5081 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 196ms

real	0m0.219s
```

Warm, drei Wiederholungen direkt danach, Graph und Frischeakte unverändert:

```
$ time /tmp/loomux-bench.exe graph build --root .
254 files, 2801 nodes, 8980 edges (2547 contains, 5081 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 191ms   real 0m0.213s
254 files, 2801 nodes, 8980 edges (2547 contains, 5081 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 190ms   real 0m0.214s
254 files, 2801 nodes, 8980 edges (2547 contains, 5081 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 212ms   real 0m0.236s
```

*`graph check`, warm.* Dasselbe Binary, derselbe Baum, der Graph bereits vom
Lauf oben geschrieben:

```
$ time /tmp/loomux-bench.exe graph check --root .
loomux graph check: OK
real	0m0.213s

$ time /tmp/loomux-bench.exe graph check --root .
loomux graph check: OK
real	0m0.223s
```

*Die Sonde allein.* `internal/code/freshness/probe_bench_test.go` gab es vor
dieser Aufgabe nicht. `BenchmarkProbe` schreibt eine Frischeakte für die
echte Dateimenge dieses Repositories — 254 Go-Dateien, einmal außerhalb des
Timers gelesen und gehasht — und misst dann `Probe` allein, wiederholt, gegen
diese Akte:

```
$ go test ./internal/code/freshness/ -bench BenchmarkProbe -benchtime 10x -v
BenchmarkProbe
    probe_bench_test.go:61: probing 254 files
    probe_bench_test.go:61: probing 254 files
BenchmarkProbe-16    	      10	  57477360 ns/op
PASS

$ go test ./internal/code/freshness/ -bench BenchmarkProbe -benchtime 10x
BenchmarkProbe-16    	      10	  59840350 ns/op
```

Zwei getrennte Läufe, als zwei getrennte Blöcke gezeigt statt als eine
zusammengesetzte Zeile: 57,48 ms/op mit `-v` (daher auch die Log-Zeile
„probing 254 files"), 59,84 ms/op bei einer direkten Wiederholung ohne `-v`.
Beide werden unten verwendet; die Zahl, die Tabelle und Lesart dieses Eintrags
tragen, ist 59,84 ms/op — der jüngere der beiden methodisch gleichen Läufe.

Maschine: AMD Ryzen 7 9800X3D, Go `go1.27.0 windows/amd64`, GOMAXPROCS 16 —
dieselbe Maschine, auf der die gepoolten/je-Knoten-Zahlen vom 2026-09-16 und
die Graft-Referenzvergleiche entstanden sind.

| Fall | kalt | warm | Referenz / Boden |
|---|---:|---:|---|
| Pagerank-Dangling-Masse, gepoolt (20k Knoten, 19.900 dangling) | 5,88 ms | — | ~9 ms gepoolt, 2026-09-16 |
| Pagerank-Dangling-Masse, je Knoten (derselbe Graph) | 3,30 s | — | ~4,5 s je Knoten, 2026-09-16 |
| `graph build --root .` (dieses Repository, vor `SkipDir`, überholt) | 219 ms Wanduhrzeit / 196 ms selbstgemeldet | 213–236 ms Wanduhrzeit / 190–212 ms selbstgemeldet | überholt — siehe den Eintrag vom 2026-09-18 11:49 unten für die aktuelle Zahl und ihre eigene Parse-Boden-Bank |
| `graph check --root .` (dieses Repository, vor `SkipDir`, überholt) | — | 213–223 ms | entfällt |
| `freshness.Probe` allein (254 Dateien, dieses Repository) | — | 59,8 ms/op (10 Wdh.) | ~3 ms für 280 Dateien (Graft) |

### Lesart

1. **Die Schuld aus G1 ist beglichen, und die Zahl bestätigt den
   Design-Kommentar fast wörtlich.** Kalt sind gepoolt 5,88 ms gegen 3,30 s je
   Knoten — ein Faktor von rund 561, je auf einem kalten Prozess, nicht als
   Mittel über viele warme. Beide liegen nahe an den warmen Zahlen vom
   2026-09-16 (~9 ms, ~4,5 s) auf derselben Maschine, was für eine Rechnung
   ohne I/O und ohne aufzuwärmenden Cache genau die kleine Lücke zwischen kalt
   und warm ist, die man erwarten würde: die Kosten sind Arithmetik, kein
   Prozesszustand.
2. **`graph build` unterscheidet kalt nicht von warm, und der Code erklärt das,
   bevor die Zahl es tut.** Kalt (219 ms) und warm (213–236 ms) überlappen
   vollständig. `buildGraph`s eigener Kommentar sagt es: der Befehl „reads and
   hashes every file, every time — never the probe's stat fast path", weil ein
   Stat entscheiden darf, ob eine *Abfrage* neu baut, aber nie, wonach der
   Neubau selbst schaut. Für `build` gibt es nichts aufzuwärmen.
   **Korrektur (2026-09-18, später am selben Tag).** Dieser Eintrag lautete
   ursprünglich „gegen den 44–46-ms-Parse-Boden aus §11 sind 190–219 ms das
   4,3- bis 4,8-fache", und eine Korrektur am selben Tag schrieb dann „`build`s
   190–219 ms selbstgemeldet" — auch falsch, denn 219 ms sind in der Tabelle
   dieses Eintrags oben die *kalte Wanduhrzeit*, nicht selbstgemeldet
   (selbstgemeldet kalt sind 196 ms). **Sowohl der Boden als auch die
   `build`-Zahlen dieses Eintrags sind überholt, und keine von beiden gehört
   hier weiter hin**: `SkipDir` kam nach diesem Eintrag hinzu und machte
   `build` selbst schneller (siehe den Eintrag vom 2026-09-18 11:49 unten, der
   seinen eigenen Befehl und seine eigene Rohausgabe trägt, einmal gemessen
   auf dem endgültigen Baum, nach jeder Codeänderung dieser Aufgabe). Dieser
   Absatz bleibt stehen, im Geiste durchgestrichen statt gelöscht, weil das
   Journal chronologisch ist und eine falsche Zahl, die verschwindet, den
   nächsten Leser nichts darüber lehrt, warum sie falsch war.
3. **`graph check`, warm, kostet ungefähr, was `build` kostet, und das ist
   Design, kein Mangel.** 213–223 ms gegen `build`s 190–212 ms warm: `check`
   extrahiert den ganzen Baum neu, um Rumpf-Hashes zu vergleichen, seine Kosten
   sind also ein zweites `buildGraph` plus ein Diff, minus das Schreiben. Dass
   die beiden Zahlen nahe beieinanderliegen, ist die Behauptung der
   CLI-Referenz sichtbar gemacht — `check` liest die Frischeakte nicht, ein
   `touch` ist deshalb kein Befund, aber eben auch kein billiger.
4. **Die Sonde liegt beim 20-Fachen der Referenz, und die Prämisse von §7.1
   hält nicht so sicher, wie sie sich liest.** 59,8 ms/op gegen Grafts ~3 ms
   für 280 Dateien ist weit über dem, was ein Hook-Budget von wenigen zehn
   Millisekunden neben allem anderen auf diesem Pfad noch tragen kann. Es sind
   nicht die Extraktionskosten — `Probe` öffnet nie eine Datei, wenn Größe und
   Änderungszeit übereinstimmen, was hier konstruktionsbedingt der Fall ist.
   Eine separate Messung von `sourceset.Stat` allein gegen die Wurzel dieses
   Repositories reproduzierte dieselben ~57–70 ms, die Kosten liegen also im
   Verzeichnis-Walk von `internal/code/sourceset`, nicht im Vergleich danach.
   Der Baum dieses Repositories hat insgesamt 1.910 Verzeichnisse, davon
   1.826 unterhalb eines der 3 mit `testdata` benannten Verzeichnisse
   (überwiegend aufgezeichnete Testfälle ohne `.go`-Dateien); Grafts
   Referenzbaum mit 280 Dateien hat diese Form nicht. Das Argument aus
   §7.1 — dass ein `git ls-files`-Subprozess je Sondenaufruf mehr kosten würde
   als die Sonde — stimmt weiterhin gegen den Boden eines *Subprozesses*
   (einige zehn Millisekunden), aber der eigene Boden der Sonde auf einem Baum
   mit so vielen Nicht-Quell-Verzeichnissen sind nicht die ~3 ms, von denen der
   Abschnitt ausgeht; er liegt näher an den Subprozess-Kosten, die er vermeiden
   wollte. Dieser Eintrag entscheidet die Design-Frage nicht; er gibt der
   nächsten eine Zahl mit, die sagt, dass die Annahme auf einem
   verzeichnislastigen Baum neu geprüft werden muss, nicht nur auf einem
   dateilastigen.

## 2026-09-18 11:35 — Der fehlende Sperrlisten-Eintrag der Sonde

Repository `loomux`, Worktree `C:/Users/micro/Documents/#GIT/loomux-code-g2`,
Branch `code-g2`. Dieser Eintrag klärt die Frage, die der vorige offen ließ:
ob die Design-Entscheidung aus §7.1 (Verzeichnislauf statt `git ls-files`)
umgekehrt werden soll. Soll sie nicht — die Ursache war ein fehlender Name auf
der Sperrliste, nicht der Lauf selbst.

**Die Prüfung.** `internal/code/sourceset.skipDirs` enthielt `testdata` nicht,
also lief `sourceset.Stat`s `filepath.WalkDir` in jedes Verzeichnis unter
`testdata/` hinein, auf der Suche nach `.go`-Dateien, die dort nie standen —
keine der Fixturen dieses Repositories unter `testdata/` ist `.go` (sie sind
`.go.txt` oder aufgezeichnete Testfall-Korpora). Gos eigenes `go/build`
ignoriert `testdata/` bereits; Grafts eigene Sperrliste kennt den Namen nur
deshalb nicht, weil Graft nicht Go-spezifisch ist. `"testdata"` zur
`skipDirs`-Liste hinzuzufügen, in ihrer bestehenden Reihenfolge, ist die ganze
Änderung (`internal/code/sourceset/sourceset.go`); ein neuer Test,
`TestListSkipsTestdata`, prüft, dass eine `.go`-Datei unter `testdata/` nie
aufgenommen wird.

**Vorher und nachher, derselbe Befehl, derselbe Baum:**

```
$ go test ./internal/code/freshness/ -bench BenchmarkProbe -benchtime 10x
```

| Zeitpunkt | ns/op | sondierte Dateien | Verzeichnisse insgesamt | Verzeichnisse unterhalb einer `testdata`-Wurzel |
|---|---:|---:|---:|---:|
| vorher (wie in der ersten Runde committet) | 59.840.350 | 254 | 1.910 | 1.826 |
| nachher (`testdata` zu `skipDirs` ergänzt) | 4.293.460 und 2.719.400 (zwei Läufe) | 254 | 1.910 (unverändert — die Verzeichnisse selbst sind nicht weg, nur nicht mehr durchlaufen) | 1.826 (unverändert) |

„Verzeichnisse unterhalb einer `testdata`-Wurzel" ist das Prädikat, das trifft,
was der Sperreintrag tatsächlich einspart: 3 Verzeichnisse in diesem Baum
heißen `testdata`, und `filepath.WalkDir` besucht jedes dieser 3 weiterhin
einmal, um zu entscheiden, es zu überspringen — nur die 1.826 Verzeichnisse
darunter werden nicht mehr durchlaufen. (Die Zahl „3 `testdata`-Wurzeln plus
alles darunter" ist 1.829; das ist eine andere, ebenfalls richtige Zahl, aber
nicht die, die der Fix aus dem Lauf entfernt.)

Rohausgabe der beiden „nachher"-Läufe:

```
BenchmarkProbe-16    	      10	   4293460 ns/op
BenchmarkProbe-16    	      10	   2719400 ns/op
```

**`graph build` und `graph check` danach, zur Bestätigung, dass sich nichts
Indizierbares bewegt hat:**

```
$ rm -rf .loomux/state/graph && time /tmp/loomux-bench2.exe graph build --root .
254 files, 2802 nodes, 8983 edges (2548 contains, 5083 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 157ms
real	0m0.217s

$ time /tmp/loomux-bench2.exe graph build --root .
254 files, 2802 nodes, 8983 edges (2548 contains, 5083 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 152ms
real	0m0.176s

$ time /tmp/loomux-bench2.exe graph check --root .
loomux graph check: OK
real	0m0.177s
```

Dateizahl: 254, unverändert gegenüber dem Eintrag vom 2026-09-18 11:10.
Knotenzahl bewegt sich um +1 (2801 → 2802) und Kanten um +3 (8980 → 8983) —
beides erklärt durch den eigenen neuen Testcode dieser Runde innerhalb von
`internal/code` (die ergänzte Funktion `TestListSkipsTestdata` und ihre
Aufrufstellen), nicht durch die `skipDirs`-Änderung: eine `.go`-Datei unter
`testdata/` war nie Teil der Dateimenge, weder vorher noch nachher, also
konnte das Überspringen des Verzeichnisses keinen Knoten entfernen, den es nie
gab. Bau- und Prüfzeiten (152–217 ms) liegen innerhalb des Rauschens des
11:10-Eintrags (190–236 ms).

### Lesart

1. **Die 20-fache Lücke schließt sich auf etwa die Referenz oder darunter, mit
   einer geänderten Zeile.** 59,8 ms/op vorher, 2,7–4,3 ms/op nachher, gegen
   Grafts ~3 ms für 280 Dateien: der zweite Lauf liegt unter der Referenz, der
   erste knapp darüber — beide in der richtigen Größenordnung, wo die Zahl des
   vorigen Eintrags um das 20-Fache daneben lag. Die Dateizahl hat sich nicht
   bewegt (254, beide Male), das ist die Prüfung, dass hier ein Lauf-Kosten-
   Fehler behoben wurde und nicht stillschweigend etwas aus dem Index gefallen
   ist.
2. **Die Bedingung aus §7.1 löst sich in eine dritte Antwort auf, keine der
   beiden vorgesehenen.** Der Abschnitt fragte, ob die Kosten der Sonde einen
   Wechsel auf `git ls-files` erzwingen würden. Taten sie nicht: der
   Verzeichnislauf selbst war nie das Problem, und das Fehlen von Git auch
   nicht. Das Problem war ein fehlender Name auf einer Liste, die genau für
   diesen Zweck schon existierte. §7.1, §13 und die Abweichungstabelle aus
   §3.7 der Spec sind aktualisiert, um das festzuhalten, statt die Bedingung
   so aussehen zu lassen, als wäre sie nie offen gewesen.
3. **`graph build` und `graph check` haben sich nicht aus dem Grund bewegt,
   der zählen würde.** Ihre Datei-, Knoten- und Kantenzahlen sind unverändert,
   abgesehen von den eigenen Testergänzungen dieser Runde — die Bestätigung,
   dass `testdata/` nichts enthielt, was der Graph braucht, und dass die
   Korrektur die Sonde ihre Laufzeit gekostet hat, den Graphen aber nichts.

Die `graph build`/`graph check`-Zahlen oben (152–157 ms selbstgemeldet,
176–217 ms Wanduhrzeit) sind die dieser Runde, ehrlich gemessen und so
belassen; eine spätere Änderung (`goModPaths` teilt sich jetzt
`sourceset.SkipDir`, siehe den Commit nach diesem) machte `build` noch
schneller. Die aktuelle Zahl, einmal gemessen nach jeder Codeänderung dieser
Aufgabe, steht im Eintrag vom 2026-09-18 11:49 unten.

## 2026-09-18 11:49 — Endzahlen, einmal gemessen nach der letzten Codeänderung

Repository `loomux`, Worktree `C:/Users/micro/Documents/#GIT/loomux-code-g2`,
Branch `code-g2`, auf Commit `cc8f7f0` (`goModPaths` teilt sich jetzt
`sourceset.SkipDir`) — diesem Eintrag folgt keine weitere Codeänderung. Jede
Zahl unten wurde in einer Sitzung gemessen, nachdem alle Codeänderungen dieser
Aufgabe gelandet waren, gezielt um zwei Probleme der vorigen beiden Einträge
zu schließen: die `build`-Zahlen des Eintrags vom 2026-09-18 11:10 wurden von
genau dem Commit überholt, den dieser Eintrag beschrieb (`SkipDir` hielt
`goModPaths` davon ab, `testdata/` zu durchlaufen, also wurde `build`
schneller am selben Tag, an dem es langsamer gemessen wurde), und der
„~27–32-ms-Parse-Boden", den beide Benchmark-Dateien behaupteten, hatte keinen
eigenen Befehl und keine eigene Rohausgabe — er verwies auf die Design-Spec,
die wiederum hierher zurückverwies. Jede Zahl unten trägt ihren eigenen Befehl
und ihre eigene Rohausgabe; keine zitiert die Spec, und keine übernimmt eine
Zahl, die diese Aufgabe nicht selbst auf dieser Maschine gemessen hat.

**Der Parse-Boden, hier zum ersten Mal als echte Bank gemessen**
(`internal/code/extract/golang/parsefloor_bench_test.go`, neu in diesem
Commit): `go/parser` allein über die echte Dateimenge dieses Repositories,
mit und ohne `parser.SkipObjectResolution`, spiegelbildlich zu der
Entscheidung, die `extract.go` dokumentiert (`ast.Object` ist abgekündigt,
also führt der Extraktor einen eigenen Gültigkeitsbereich-Stapel, statt
`go/parser` Bezeichner auflösen zu lassen).

```
$ go test ./internal/code/extract/golang/ -bench BenchmarkParse -benchtime 10x -v
goos: windows
goarch: amd64
pkg: github.com/xidus90/loomux/internal/code/extract/golang
cpu: AMD Ryzen 7 9800X3D 8-Core Processor
BenchmarkParseSkipObjectResolution
    parsefloor_bench_test.go:57: parsing 255 files
    parsefloor_bench_test.go:57: parsing 255 files
BenchmarkParseSkipObjectResolution-16    	      10	  18755780 ns/op
BenchmarkParseWithObjectResolution
    parsefloor_bench_test.go:74: parsing 255 files
    parsefloor_bench_test.go:74: parsing 255 files
BenchmarkParseWithObjectResolution-16    	      10	  26208830 ns/op
PASS
ok  	github.com/xidus90/loomux/internal/code/extract/golang	0.735s
```

**Diese Zahlen — 18,8 ms / 26,2 ms warm über 255 Dateien — weichen von den
~27–32 ms / ~38–45 ms ab, die der Koordinator mit einem separaten
Wegwerfwerkzeug am 2026-09-17 gemessen hat.** Meine liegen auf beiden Seiten
niedriger und sind die, die dieses Dokument jetzt trägt, weil sie mit dem
Befehl oben kommen, mit `go test` reproduzierbar sind und auf demselben Baum
entstanden wie jede andere Zahl dieses Eintrags. Die Lücke habe ich nicht
untersucht (anderer Zeitpunkt, anderer Prozess, möglicherweise eine andere
Dateimenge — diese Bank zählt 255, das Werkzeug des Koordinators zählte 254)
über das bloße Feststellen hinaus, statt still die bequemere Zahl zu wählen.

**`graph build` und `graph check`, kalt und warm, auf einem Binary aus diesem
Commit:**

```
$ go build -o /tmp/loomux-final.exe ./cmd/loomux
$ rm -rf .loomux/state/graph && time /tmp/loomux-final.exe graph build --root .
255 files, 2808 nodes, 9003 edges (2553 contains, 5092 calls, 1358 imports)
1128 unresolved import targets, 3 files without a symbol, 94ms
real	0m0.151s

$ time /tmp/loomux-final.exe graph build --root .
255 files, 2808 nodes, 9003 edges (2553 contains, 5092 calls, 1358 imports)
1128 unresolved import targets, 3 files without a symbol, 106ms
real	0m0.131s

$ time /tmp/loomux-final.exe graph build --root .
255 files, 2808 nodes, 9003 edges (2553 contains, 5092 calls, 1358 imports)
1128 unresolved import targets, 3 files without a symbol, 98ms
real	0m0.127s

$ time /tmp/loomux-final.exe graph check --root .
loomux graph check: OK
real	0m0.136s
```

**`freshness.Probe` allein:**

```
$ go test ./internal/code/freshness/ -bench BenchmarkProbe -benchtime 10x -v
BenchmarkProbe
    probe_bench_test.go:61: probing 255 files
    probe_bench_test.go:61: probing 255 files
BenchmarkProbe-16    	      10	   4245920 ns/op
PASS
```

**Das Pagerank-Dangling-Masse-Paar, kalt, je ein Prozess, auf diesem Baum neu
bestätigt** (kein Code in `internal/code/pagerank` hat sich seit dem Eintrag
vom 2026-09-18 11:10 geändert, aber die Regel dieses Eintrags ist „in dieser
Sitzung gemessen, sonst steht es nicht hier"):

```
$ go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPooled -benchtime 1x -count 1
BenchmarkDanglingPooled-16    	       1	   6352900 ns/op

$ go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPerNode -benchtime 1x -count 1
BenchmarkDanglingPerNode-16    	       1	3865115600 ns/op
```

| Fall | Zahl | Befehl |
|---|---:|---|
| Parse-Boden, `SkipObjectResolution` (255 Dateien, warm) | 18,8 ms | `go test ./internal/code/extract/golang/ -bench BenchmarkParseSkipObjectResolution -benchtime 10x` |
| Parse-Boden, mit Objektauflösung (255 Dateien, warm) | 26,2 ms | `go test ./internal/code/extract/golang/ -bench BenchmarkParseWithObjectResolution -benchtime 10x` |
| `graph build --root .`, kalt (255 Dateien) | 151 ms Wanduhrzeit / 94 ms selbstgemeldet | `rm -rf .loomux/state/graph && time /tmp/loomux-final.exe graph build --root .` |
| `graph build --root .`, warm (255 Dateien, zwei Wiederholungen) | 127–131 ms Wanduhrzeit / 98–106 ms selbstgemeldet | `time /tmp/loomux-final.exe graph build --root .` |
| `graph check --root .`, warm | 136 ms Wanduhrzeit, Exit 0 | `time /tmp/loomux-final.exe graph check --root .` |
| `freshness.Probe` allein (255 Dateien, 10 Wdh.) | 4,25 ms/op | `go test ./internal/code/freshness/ -bench BenchmarkProbe -benchtime 10x` |
| Pagerank-Dangling-Masse, gepoolt (20k Knoten, kalt, ein Prozess) | 6,35 ms | `go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPooled -benchtime 1x -count 1` |
| Pagerank-Dangling-Masse, je Knoten (20k Knoten, kalt, ein Prozess) | 3,87 s | `go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPerNode -benchtime 1x -count 1` |

### Lesart

1. **`build` gegen den eigenen Boden: rund das 5- bis 7-Fache, mit Zahlen, die
   je einen eigenen Befehl tragen.** 94–106 ms selbstgemeldet (255 Dateien)
   gegen einen reinen Parse-Boden von 18,8 ms sind das 5,0- bis 5,6-Fache;
   gegen die 26,2 ms mit Objektauflösung das 3,6- bis 4,0-Fache. Beide liegen
   im selben Bereich, in dem der frühere, zufällig doppelt gezählte Vergleich
   landete (4,3- bis 4,8-fache und, korrigiert, 5- bis 7-fache) — das
   Verhältnis hielt über einen falschen Boden, einen korrigierten, aber nicht
   committeten Boden und jetzt einen Boden mit eigener reproduzierbarer Bank
   hinweg stand, was ein Indiz dafür ist, dass die Kosten von Extraktion und
   Auflösung über dem Parse-Boden ein stabiles Vielfaches davon sind, nicht
   ein Artefakt der einen oder anderen fehlerhaften Messung.
2. **`build` wurde schneller, als jeder vorige Eintrag dieser Datei berichtete,
   und der Grund ist eine Änderung dieser Aufgabe, kein Rauschen.** 94–106 ms
   selbstgemeldet hier gegen 190–212 ms im 11:10-Eintrag und 152–157 ms im
   11:35-Eintrag, auf denselben 254–255 Dateien: `goModPaths` durchläuft die
   1.826 `testdata/`-Verzeichnisse nicht mehr auf der Suche nach einem
   `go.mod`, das es dort ohnehin nie findet. Dieser Eintrag ist der einzige
   der drei, in dem Zahl und Code, der sie erzeugt hat, derselbe Commit sind.
3. **Das Dangling-Massen-Verhältnis ist über Neumessungen stabil.** 6,35 ms
   gepoolt gegen 3,87 s je Knoten hier, gegen 5,88 ms / 3,30 s im
   11:10-Eintrag und ~9 ms / ~4,5 s am 2026-09-16 — alle kalt, ein Prozess,
   dieselbe Maschine, drei verschiedene Sitzungen. Die absoluten Zahlen
   bewegen sich mit Lauf-zu-Lauf-Rauschen (ein Faktor von ~1,1–1,2); das
   Verhältnis von rund dem 500- bis 600-Fachen zwischen gepoolt und je Knoten
   nicht.

## 2026-09-18 15:30 — Stufe 1b-2 abgeschlossen: Größe, Hook-Pfad, Handschlag und die zwei Sprünge

Repository `loomux`, Worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b-2`,
Branch `sdd-1b-2`, Commit `0a4786c` — die vier Messungen, gegen die die Stufe
abgenommen wird. Maschine: AMD Ryzen 7 9800X3D, GOMAXPROCS 16, Go
`go1.27.0 windows/amd64`.

**Eine Berichtigung am Auftrag dieser Aufgabe.** Er nannte als Vorwerte
„13,7 MB / 8,1 ms ohne SDK, 16,0 MB / 8,2 ms mit ihm, am 2026-09-17 mit
`Measure-Command` über 20 Läufe gemessen". Der Eintrag vom 2026-09-17 20:30
oben sagt 12 Läufe (einer kalt, Median der elf warmen) und 7,4 / 7,6 ms. Die
hier wiederholte Methode ist die der Datei, und die Zahlen der Datei sind die
Grundlinie.

### 1. Binärgröße und Startboden, vor und nach dem SDK

**Methode.** PowerShell, ein `Measure-Command` je Lauf, zwölf Läufe je Binary:
der erste ist der kalte Wert, der Median der übrigen elf der warme; in jedem
Fall `loomux --version`. `nachher` ist `bin/loomux.exe` auf `0a4786c`.
`vorher` ist `48e5d38` — der Commit vor dem Einzug des SDK, derselbe, den der
Eintrag vom 2026-09-17 gemessen hat —, hier aus einem `git archive`-Export in
ein Kratzverzeichnis neu gebaut; daher 13.746.688 B gegen die damals notierten
13.736.448 B: der Export ist kein Git-Repository, der Bau trägt also keinen
VCS-Stempel. Die kalten Werte sind erste Läufe in einer bereits warmen Shell und
über Zeilen hinweg nicht vergleichbar; sie stehen da, sie werden nicht gelesen.

**Dieser Abschnitt benutzt keine Vorrichtung.** Er misst `loomux --version`
unmittelbar mit `Measure-Command`; die JSON-Vorrichtung gehört allein zu
Abschnitt 2. Die beiden nennen deshalb zwei verschiedene Startböden für dasselbe
Binary — 8,4 ms hier, 6,4 ms dort —, und das ist das Messgerät und nicht das
Binary. Abschnitt 4 sagt, welchen der beiden er abzieht und warum er eine Spanne
angibt.

Das `vorher`-Binary bauen, in PowerShell, weil es im Baum niemand baut:

```powershell
git archive -o "$env:TEMP\loomux-1b-2-src.tar" 48e5d38
New-Item -ItemType Directory -Force "$env:TEMP\loomux-1b-2-src" | Out-Null
tar -xf "$env:TEMP\loomux-1b-2-src.tar" -C "$env:TEMP\loomux-1b-2-src"
Push-Location "$env:TEMP\loomux-1b-2-src"
go build -o "$env:TEMP\loomux-1b-2\before.exe" ./cmd/loomux
Pop-Location
```

| Fall | Größe | kalt (1. Lauf) | warmer Median | warm min | warm max |
|---|---:|---:|---:|---:|---:|
| vorher `48e5d38`: ohne SDK | 13.746.688 B | 53,2 ms | 8,3 ms | 7,9 ms | 10,5 ms |
| nachher `0a4786c`: die ganze Stufe | 17.559.552 B | 46,3 ms | 8,4 ms | 8,1 ms | 10,7 ms |

Paket-Inits über 100 Allokationen, `GODEBUG=inittrace=1 loomux version` auf
`nachher`:

| Paket | Allokationen |
|---|---:|
| `encoding/gob` | 369 |
| `github.com/google/jsonschema-go/jsonschema` | 298 |
| `gopkg.in/yaml.v3` | 274 |
| `github.com/modelcontextprotocol/go-sdk/mcp` | 183 |
| `github.com/xidus90/loomux/internal/brain/wiki` | 131 |
| `github.com/xidus90/loomux/internal/brain/search` | 103 |

**Lesart.** Der Startboden hat sich über die ganze Stufe nicht bewegt: 8,4 ms
gegen 8,3 ms warm, bei warmen Spannen, die einander fast ganz überlappen
(8,1–10,7 gegen 7,9–10,5). Die Größe schon: 17.559.552 B gegen 13.746.688 B,
3.812.864 B mehr, 27,7 %. Davon waren 2.268.160 B der SDK, der in Task 1 kam;
die übrigen ~1.545.000 B sind der Code, den diese Stufe hinzugefügt hat, und die
Teile des SDK, die er erreicht — Streamable-HTTP-Server und -Client,
`x/oauth2`, `uritemplate`. Das ist mehr als die fünf neuen Pakete:
`git diff --stat 48e5d38..0a4786c -- cmd/ internal/` nennt darunter auch den
Fall-Rekorder, den Fall-Importeur und den Korpuslader.
Die Allokationsgrenze ist unbewegt: der größte Init
ist weiterhin `encoding/gob` mit 369, unter den 500, die
`TestStartDoesNoWorkInPackageInit` zulässt, und kein Paket aus
`internal/serve`, `internal/bridge`, `internal/lock`, `internal/mcptools` oder
`internal/brain/answer` steht überhaupt in der Liste.

### 2. Der Hook-Pfad, unverändert und gemessen

**Methode.** `loomux dev bench-hooks testdata/bench/1b-2-hooks.json -n 20`, ein
Durchgang, ein kalter Lauf je Fall und 20 warme. Nutzlast ein `Edit` auf der
`README.md` dieses Worktrees (`testdata/bench/edit-readme-1b-2.json`) gegen das
echte Zustandsverzeichnis — dieser Worktree ist ein verknüpfter Worktree eines
registrierten Workspace, die Schranke geht also den Worktree-Weg. Dieselben zwei
Binaries wie oben, in einem Durchgang mit ihren eigenen Böden, damit der Hook
gegen sie gelesen werden kann.

**Ehe dieser Befehl anderswo läuft, muss die Vorrichtung umgepfadet werden.**
`1b-2-hooks.json` und `edit-readme-1b-2.json` tragen absolute Pfade des
Worktrees, in dem sie aufgezeichnet wurden — jedes `dir`, jedes `stdin` und
zwei `argv`-Einträge in der ersten Datei, drei weitere in der zweiten —, genau
wie `1a-hooks.json` es für Stufe 1a tut. Das `nachher`-Binary ist
`bin/loomux.exe` jenes Worktrees; das `vorher`-Binary baut das PowerShell aus
Abschnitt 1. Die Vorrichtung nennt `%TEMP%\loomux-1b-2\before.exe`, einen
stabilen Ort statt eines Sitzungsverzeichnisses, es sind also nur die
Worktree-Pfade zu ersetzen.

| Fall | kalt (1. Lauf) | warmer Median | warm min | warm max | Exit-Codes |
|---|---:|---:|---:|---:|---|
| vorher: `loomux version` (Startboden, ohne SDK) | 10,2 ms | 7,6 ms | 6,0 ms | 14,1 ms | [0] |
| nachher: `loomux version` (Startboden, SDK gelinkt) | 7,5 ms | 6,4 ms | 5,7 ms | 7,0 ms | [0] |
| vorher: `loomux hook pre-tool-use` (Edit auf README.md) | 12,0 ms | 9,9 ms | 9,0 ms | 12,0 ms | [0] |
| nachher: `loomux hook pre-tool-use` (Edit auf README.md) | 11,5 ms | 9,0 ms | 8,1 ms | 11,0 ms | [0] |

**Lesart.** Der Hook liegt nach der Stufe bei 9,0 ms warm gegen 9,9 ms davor,
kalt bei 11,5 gegen 12,0 ms; die warmen Spannen überlappen (8,1–11,0 gegen
9,0–12,0). Über seinem eigenen Boden kostet der Hook nachher 2,6 ms und vorher
2,3 ms — dieselbe Entscheidung, und der Unterschied liegt im Rauschen, das beide
Böden zeigen. Dass der MCP-Stapel nicht auf diesem Pfad liegt, ist strukturell
und wird von `TestHooksNeverImportServeOrBridge` in
`internal/cli/imports_test.go` gehalten: der Test liest `go list -deps` von
`internal/hooks` und schlägt bei `internal/serve`, `internal/bridge` oder
`.../go-sdk/mcp` fehl. Diese Tabelle ist die gemessene Hälfte derselben
Behauptung. Gegen die 7,5 ms vom 2026-09-17 14:16: jener Lauf war der
Hauptcheckout, wo der Pfadvergleich der Schranke vor der Worktree-Suche endet;
die beiden Zeilen sind nicht vergleichbar, und beide liegen weit unter dem
Zielwert von 72 ms.

#### 2026-09-18 19:40 — dieselben vier Fälle, nachdem die Vorrichtung umgepfadet wurde

Die committete Vorrichtung ist nicht die, aus der die Tabelle oben stammt: das
`vorher`-Binary ist aus einem Sitzungsverzeichnis nach `%TEMP%\loomux-1b-2\`
gezogen und die Vorrichtung darauf umgeschrieben worden. Gleiche Binaries,
gleiche Nutzlast, weniger Läufe — `-n 5` statt `-n 20`, einer kalt und fünf
warm:

| Fall | kalt (1. Lauf) | warmer Median | warm min | warm max | Exit-Codes |
|---|---:|---:|---:|---:|---|
| vorher: `loomux version` (Startboden, ohne SDK) | 39,5 ms | 6,5 ms | 6,0 ms | 6,6 ms | [0] |
| nachher: `loomux version` (Startboden, SDK gelinkt) | 8,0 ms | 6,5 ms | 5,5 ms | 6,6 ms | [0] |
| vorher: `loomux hook pre-tool-use` (Edit auf README.md) | 14,9 ms | 9,2 ms | 9,0 ms | 9,5 ms | [0] |
| nachher: `loomux hook pre-tool-use` (Edit auf README.md) | 10,0 ms | 9,0 ms | 8,5 ms | 9,5 ms | [0] |

**Lesart.** Die Vorrichtung läuft weiterhin, und der Schluss bleibt — aber es
sind nicht dieselben Zahlen, und „reproduziert" wäre dafür zu stark. Der
Abstand vorher/nachher am Hook ist hier **0,2 ms gegen 0,9 ms oben**, und die
beiden Startböden sind hier gleich, wo sie oben um 1,2 ms auseinanderlagen.
Beide Durchgänge sagen dasselbe — der Abstand liegt im Rauschen, der Hook hat
sich nicht bewegt —, und dieser sagt es deutlicher, auf fünf warmen Läufen statt
zwanzig und auf einer beschäftigteren Maschine (die kalte Spalte zeigt es). Wo
die beiden auseinandergehen, ist der Zwanzig-Lauf-Durchgang oben die Messung und
dieser der Beleg, dass die umgepfadete Vorrichtung läuft.

### 3. Der Handschlag der Brücke, mit laufendem und mit kaltem Dienst

**Methode.** Eine Go-Sonde ohne Fremdabhängigkeiten in einem Kratzverzeichnis
startet `loomux mcp --channel local`, schreibt `initialize` als eine Zeile JSON
auf dessen stdin und stempelt den Augenblick, in dem die passende Antwort auf
dessen stdout ankommt; die Uhr läuft vor dem Prozess an, die Zahl enthält also
den Start der Brücke selbst. `LOOMUX_STATE_DIR` zeigt auf ein Kratzverzeichnis
mit einer Kopie der echten `registry.toml` (11 Bereiche), nichts hier rührt also
den echten Dienst an. Warm: 21 Läufe gegen einen bereits laufenden Dienst. Kalt:
`loomux serve stop`, dann ein Lauf, dreimal.

| Fall | erster Lauf | warmer Median | warm min | warm max |
|---|---:|---:|---:|---:|
| `initialize`, Dienst läuft bereits (21 Läufe) | 10,4 ms | 8,5 ms | 7,5 ms | 10,5 ms |
| `initialize`, kein Dienst (3 Läufe: 9,0 / 9,1 / 10,0 ms) | — | 9,1 ms | 9,0 ms | 10,0 ms |

**Lesart.** Beides ist dieselbe Zahl, und das ist der Entwurf und kein Zufall:
`bridge.Run` bietet die fünf Werkzeuge aus `internal/mcptools` selbst an und
stößt den Dienst in einer Goroutine an (`go func() { ensure(...) }()` in
`bridge.go`), also sitzt nie ein Start im Handschlag des Wirts. Ein Wirt sieht
einen antwortenden Server nach etwa 9 ms, ob es einen Dienst gibt oder nicht —
das ist der Startboden der Brücke plus ein Umlauf über ihre Leitungen. Den
kalten Preis zahlt stattdessen der erste `tools/call`, siehe 4.

### 4. Ein `brain_search` über die Brücke gegen die Kommandozeile

**Methode.** Dieselbe Sonde, dasselbe Kratz-Zustandsverzeichnis. Nach dem
Handschlag schickt sie einen `tools/call` für `brain_search` und stempelt die
Antwort; die Kommandozeile ist
`loomux brain search latenz --profile keyword -n 5 --channel local`, mit
`Measure-Command` gemessen. Beide Seiten nageln `profile`, `n` und `channel`
fest, weil die beiden Fronten verschieden zurückfallen (`n` ist über MCP 10 und
auf der Kommandozeile 5) und weil `keyword` das eine Profil ist, dessen warme
Streuung eine Millisekunde nicht begräbt: der Eintrag vom 2026-09-16 19:10 zeigt
`fast` mit 236–373 ms. Je 21 Läufe, der erste gesondert ausgewiesen. Der
qmd-Dämon auf Port 8765 lief durchgehend warm; fünf Verbindungen lagen auf ihm.

| Fall | erster Lauf | warmer Median | warm min | warm max |
|---|---:|---:|---:|---:|
| `brain_search` über die Brücke, Dienst läuft | 55,5 ms | 49,5 ms | 44,5 ms | 61,0 ms |
| `loomux brain search`, direkt | 61,5 ms | 44,5 ms | 37,5 ms | 60,7 ms |
| `brain_search` über die Brücke, kein Dienst (3 Läufe) | — | 310,1 ms | 310,0 ms | 310,2 ms |
| der Dienst meldet sich nach entkoppeltem Start (3 Läufe) | — | 41,9 ms | 41,3 ms | 43,1 ms |

**Lesart.**

1. **Die zwei Sprünge kosten 5 ms von Ende zu Ende und 11–13 ms Arbeit.**
   Der Aufruf über die Brücke liegt bei 49,5 ms gegen 44,5 ms der
   Kommandozeile, 5,0 ms mehr, bei überlappenden warmen Spannen (44,5–61,0 gegen
   37,5–60,7). Die Kommandozeile zahlt aber einen Prozessstart, den der
   Brückenaufruf nicht zahlt; die nackte Antwort sind also 44,5 ms abzüglich des
   Startbodens, und die zwei Sprünge — JSON-RPC über die Leitungen des Wirts,
   dann HTTP mit Bearer-Token zum Dienst — sind die Differenz. **Welcher Boden
   abzuziehen ist, ist nicht frei vom Messgerät.** Mit demselben
   `Measure-Command` wie die 44,5 ms genommen, ist der Boden 8,4 ms (Messung 1)
   und die Sprünge kosten 13,4 ms; mit `dev bench-hooks` genommen, das den
   Prozess selbst startet, ist er 6,4 ms (Messung 2) und die Sprünge kosten
   11,4 ms. Die ehrliche Angabe ist die Spanne: **11–13 ms**, und keine
   Subtraktion über zwei Messgeräte hinweg ist enger. So oder so ist es eine
   Millisekundenzahl gegen eine Antwort, die qmd beherrscht.
2. **Ein kalter Dienst kostet 260 ms, und 250 davon sind ein Abfrageintervall,
   kein Start.** Die drei kalten Läufe sind 310,0, 310,2 und 310,1 ms — eine
   Spanne von 0,2 ms, die Signatur eines gerasterten Wartens und nicht von
   Arbeit. Der Dienst selbst steht und hat `serve.json` 41,3–43,1 ms nach einem
   entkoppelten Start geschrieben. Worauf die Brücke wartet, ist
   `waitForService`, und das fragt im Takt `StartTick = 250 ms`
   (`internal/bridge/connect.go`): ein Dienst, der nach 42 ms bereit ist, wird
   erst nach 250 ms bemerkt. **Das ist der billigste offene Hebel der Stufe:**
   ein kürzerer erster Takt oder ein sich verlangsamender nähme dem ersten
   Aufruf nach einem Kaltstart etwa 200 ms, ohne sonst etwas anzufassen. Es ist
   kein Mangel — die 250 ms sind mit Absicht aus dem qmd-Handschlag übernommen,
   damit es im Projekt einen Warterhythmus gibt — und es wird hier nicht
   geändert.
3. **Die Maschine muss sich erst setzen.** Ein früherer Satz von zehn warmen
   Läufen, unmittelbar nach dem allerersten Kaltstart genommen, ergab
   Handschläge von 20–118 ms und Aufrufe von 101–161 ms. Die Tabellen oben sind
   der gesetzte Zustand, gemessen, nachdem qmd einmal geantwortet hatte. Der
   frühere Satz steht hier, weil er zeigt, wie die erste Minute einer Sitzung
   aussieht, nicht weil er etwas misst.
## 2026-09-18 19:30 - Antwortzeit von graph ask und die Beiakte

Antwortzeit von `loomux graph ask "write barrier refuse" --limit 8 --no-refresh`
auf diesem Repository (267 Dateien, 2.982 Knoten, 9.637 Kanten) im Vergleich:
warmer Pfad mit der Beiakte `ask-index.json` gegen Abfrage ohne Beiakte (nur
Namen und Pfade) sowie Kaltstart.

### Messungen

| Fall | Wert | Befehl |
|---|---:|---|
| `graph ask`, warm mit Beiakte (10 Läufe, Median) | 47,6 ms (Spanne 46,5 - 59,2 ms) | `./bin/loomux.exe graph ask "write barrier refuse" --limit 8 --no-refresh` |
| `graph ask`, warm ohne Beiakte (10 Läufe, Median) | 37,5 ms (Spanne 35,6 - 44,1 ms) | (bei entfernter `ask-index.json`) `./bin/loomux.exe graph ask "write barrier refuse" --limit 8 --no-refresh` |
| `graph ask`, kalt (frischer Prozess, Graph und Beiakte im Cache) | 52,7 ms | `./bin/loomux.exe graph ask "write barrier refuse" --limit 8 --no-refresh` |

### Einordnung

1. **Kosten und Nutzen der Beiakte bei 3.000 Knoten.**
   Warme Anfragen mit Beiakte benötigen ~48 ms; ohne Beiakte (Matching nur auf Namen
   und Pfaden ohne invertierten Rumpfindex) liegen sie bei ~37 ms. Das Deserialisieren
   des 1-MB-JSON-Bestands von `ask-index.json` kostet ~10-12 ms. Bei einem Projekt
   dieser Größe (~3.000 Knoten) ist das Auswerten der Namen im Speicher so schnell,
   dass die Ladezeit der Beiakte die Ersparnis übersteigt.
2. **Warum die Beiakte dennoch existiert.**
   Wie in der Spezifikation und der Graft-Referenz festgehalten, wird das erneute
   Extrahieren aller Funktionsrümpfe ab etwa 30.000 Knoten zu langsam (Graft verzeichnete
   bei dieser Größe ~45 % Ersparnis durch den Index). Die Beiakte skaliert auf große
   Codebasen, bei denen ein voller Rumpf-Scan das interaktive Zeitbudget sprengen würde.
   Dieser Befund wird hier ehrlich als Merkmal der aktuellen Projektgröße festgehalten,
   statt künstlich geschönt zu werden.


## 2026-09-18 21:20 - Eine sortierte Term-Reihenfolge je Abfrage statt dreier je Dokument

### Was gemessen wurde

`ask.Run` über einem synthetischen Graphen aus 3.000 Knoten und 2.999 Kanten mit
einer fünfwortigen Abfrage, vor und nach dem Umzug der Term-Reihenfolge aus dem
Dokument-Scoring (`lexical` rief `overlap` zweimal und `bm25` einmal, jeder
sortierte dieselben Begriffe erneut) in ein `newQuestion` je Abfrage.
Wegwerf-Benchmark, nicht im Baum behalten, `-benchmem -count=5` auf der
Referenzmaschine (AMD Ryzen 7 9800X3D, Windows x86_64).

### Messwerte

| Fall | ns/op (Median aus 5) | B/op (Median aus 5) | allocs/op |
|---|---:|---:|---:|
| Ausgangsstand, Reihenfolge je Dokument | 8.318.985 (Spanne 7,49 - 8,64 ms) | 3.881.243 | 18.167 |
| Änderung, eine Reihenfolge je Abfrage | 8.510.965 (Spanne 7,98 - 9,58 ms) | 3.148.879 | 9.167 |

Der CLI-Weg wurde ebenfalls gemessen, 10 warme Läufe von
`graph ask "how does the write barrier decide what to refuse and why" --limit 5`
auf diesem Repository: 77 ms Median vorher, 79,5 ms nachher. Dort dominiert der
Prozessstart, und der Unterschied ist Rauschen.

### Lesart

1. **Die Zahl der Allokationen halbiert sich, die Zeit bewegt sich nicht.**
   18.167 -> 9.167 Allokationen je Abfrage und 19 % weniger Bytes, während die
   Uhr innerhalb der Lauf-zu-Lauf-Streuung von +-1 ms bleibt. Vier Strings zu
   sortieren ist billig; es 9.000-mal zu tun, ist nur im Allokator messbar,
   nicht auf der Uhr.
2. **Verbucht als entfernte Doppelarbeit, nicht als Beschleunigung.** Bei 3.000
   Knoten ist kein Zeitgewinn zu behaupten. Ob aus der Allokationsersparnis bei
   30.000 Knoten ein Zeitgewinn wird, ist ungeprüft und darf nicht angenommen
   werden.


## 2026-09-19 00:15 - Die Beiaktenprüfung, die die Frischesonde jetzt macht

### Was gemessen wurde

`loomux graph ask` auf einem sauberen Baum, wo die Frischeakte keine Abweichung
meldet und die Sonde ohne Neubau zurückkehrt. Nur diesen Weg berührt die
Änderung: `EnsureFresh` ruft jetzt `lexicon.Usable`, ein `open` und eine
Teillesung der `ask-index.json`, dekodiert bis zum Feld `version`, bevor es
früh zurückkehren darf.

Beide Binaries aus der Quelle an getrennte Pfade gebaut, `86fa192` (v1.1.0) als
Grundstand und `f160d73` samt Arbeitsstand dieses Zweigs als Änderung, gegen
dieselbe Wurzel und denselben Graphenstand gemessen: ein Checkout dieses
Repositories auf `86fa192`, 282 Dateien, 3.073 Knoten, 9.884 Kanten, Beiakte auf
der Platte. Anfrage `"retry backoff" --limit 1`, drei Aufwärmläufe verworfen,
dann 10 Läufe mit `Measure-Command` auf der Referenzmaschine (AMD Ryzen 7
9800X3D, Windows x86_64).

### Messwerte

| Fall | Median | Min | Max |
|---|---:|---:|---:|
| Grundstand, saubere Akte genügt | 51,2 ms | 49,1 ms | 53,6 ms |
| Änderung, saubere Akte plus Beiaktenversion | 49,9 ms | 48,9 ms | 53,1 ms |

### Lesart

1. **Die zusätzliche Lesung zeigt sich nicht.** Die Änderung misst 1,3 ms
   *unter* dem Grundstand, was innerhalb der Streuung beider Reihen liegt (die
   Bereiche überlappen fast vollständig) und damit Rauschen ist, kein Gewinn.
   Die Kosten sind ein `open` und ein paar hundert gepufferte Bytes gegen einen
   50-ms-Prozess, dessen Zeit von Start und Graphenlesung bestimmt wird.
2. **Einen kalten Fall gibt es hier nicht zu messen.** Eine kalte Anfrage hat
   keinen brauchbaren Graphenstand und baut neu; der Bau liest und hasht jede
   Datei und brauchte auf dieser Wurzel 283 ms. Die eine zusätzliche Lesung
   liegt bauartbedingt auf dem warmen Weg und wird auf dem kalten nicht
   erreicht.
3. **`Usable` ist nicht `Read`.** Es dekodiert bis zum Feld `version` und hört
   auf. Das ganzdateiige `Read` zu messen hätte eine andere und größere Zahl
   ergeben; genau dafür existiert die Prüfung als eigene Funktion.


## 2026-09-19 00:31 — `loomux dev bench` an loomux und am Open-Source-Korpus

Repo `loomux`, Zweig `open-source-matrix`, `loomux dev bench --dir . --warm 3`
nach dem Korpuslauf `--corpus docs/de/open-source-matrix.md --languages 25
--warm 3`. Jeder Hook bekommt eine Claude-Code-Nutzlast für einen Edit an
`cmd/loomux/main.go`, post-tool-use fährt also seine echten Lanes.
Referenzmaschine (AMD Ryzen 7 9800X3D, Windows x86_64).

### loomux

| Komponente | Kalt | Warmer Median | Warm Min | Warm Max | Exit-Codes |
|---|---:|---:|---:|---:|---|
| pre-tool-use | 11,1 ms | 9,0 ms | 8,5 ms | 11,0 ms | 0 |
| post-tool-use | 557,6 ms | 509,4 ms | 507,1 ms | 530,0 ms | 0 |
| graph build | 259,8 ms | 252,7 ms | 250,1 ms | 293,4 ms | 0 |

### Korpus

243 Repositorys aus 25 Sprachen, eines übersprungen
(`membraneframework/membrane_core`: ein `|` im Pfad lässt sich unter Windows
nicht auschecken). post-tool-use endete in 137 Repositorys mit 0 und in 95 mit
2; 11 hatten keine Beispieldatei. Details je Repository: [Open-Source
Benchmark-Matrix](benchmarks/matrix.md).

### Lesart

1. **post-tool-use ist der Preis des Edits, und der sind die Lanes.** Rund
   0,5 s bei loomux sind `go vet ./...` auf warmem Build-Cache; der Hook selbst
   liegt im Bereich von pre-tool-use. Eine frühere Pilotmessung dieses
   Werkzeugs nannte 11,5 ms für post-tool-use; jener Lauf schickte keine
   Nutzlast, es lief also keine Lane, und seine Zahlen sind verworfen.
2. **Exit 2 heißt: loomux blockiert, nicht: der Benchmark scheitert.** Von Hand
   geprüft: `go vet` findet Befunde in gorm, `ruff` in celery. Ob ruff dort die
   Konfiguration des Repositorys selbst benutzt, ist nicht geprüft.
3. **Der Baseline-Speedup von 1,1x bei loomux vergleicht loomux mit sich
   selbst**: Die `.claude/settings.json` dieses Repositorys ruft dasselbe
   Binary.

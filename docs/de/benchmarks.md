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

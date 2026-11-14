---
type: Topic
title: Die Schreibschranke
description: Wo ein Agent schreiben darf — Registry, Manifest und Policy, der Schalter llm_config, verknüpfte Worktrees ohne eigenen Eintrag und einzelne Dateien aus open.toml.
open_conflicts: 0
realization: implemented
implemented_in: f650a62f
sources:
  - id: konfiguration
    resource: brain://project/loomux/docs/de/configuration.md
    doc_id: 01M47W91R50RT15H2ES8RV7TJT
    content_hash: "sha256:1e1ce68a6f431871cb168cc2f54e9a69beb4367c0939010c9c473f126d1a7b37"
    revision: 49
  - id: hooks
    resource: brain://project/loomux/docs/de/hooks.md
    doc_id: 01M47W91R5H3G1QSXZV5C17HS9
    content_hash: "sha256:e239ce2f0f5d56bb6b09e1206ebb9e1d13fc8ff48b2e9b072ed36d0d0d7c90a8"
    revision: 20
  - id: benchmarks
    resource: brain://project/loomux/docs/de/benchmarks.md
    doc_id: 01M47W91R5MPENKN7N34H6QTNT
    content_hash: "sha256:d10df343fb61fb76ea53dfdf88a3f2511e6c17849aa11f01be74cdf87ac0d952"
    revision: 22
---

## Wofür

Der Pre-Tool-Wächter entscheidet vor jedem schreibenden Werkzeugaufruf,
**wo** der Write landet. Zwei Stufen, in dieser Reihenfolge: erst die
**Policy** (eingebaut, maschinenweit, Projekt), dann die globale **Schreibschranke**
(`internal/hooks/pretool.go`). Beide verweigern, statt zu raten: **Eine
Schranke, die bei Unklarheit öffnet, ist keine.**

## Die Policy des Projekts

Die Tabelle `[policy]` in `.loomux/config.toml` trägt zwei Regellisten:
Pfadregeln mit Globs gegen schreibende Werkzeuge und Befehlsregeln mit
regulären Ausdrücken gegen `Bash` und `PowerShell`. Eingebaute Regeln gehen
immer mit: Geheimnisse (`.env`, `*.pem`, `*.key` und andere) schreibt kein
Agent; `loomux init`, `config`, `ui` (in jeder Form außer `--help`), `area add`,
`merge-hook install|remove`, `gate arm|disarm`, `convert` und `fetch` führt ein Mensch aus,
ebenso das `--answer` am Tor eines Flows; und einen Interpreter, der sein
Programm von stdin liest (`python -`, ein Heredoc in den Interpreter),
startet kein Agent. Ebenso
`config.toml`, `ui.toml`, `ui.json` und `registry.toml` im Zustandsverzeichnis:
Der Wächter verweigert dort jedes Schreiben, das er erkennt, für die
Schreibwerkzeuge wie für Shell-Zeilen, auch über `cwd` (samt Glob dort),
Home-Ordner und Umgebungsvariablen (auch `${NAME:-x}` und `$($env:NAME)`),
Laufwerks- und Freigabeschreibweisen, Punkt und Leerzeichen am Ende, den
Provider-Präfix `FileSystem::` und einen Link, der in derselben Zeile
entsteht (auch durch `cp -s` oder `cp -l`), auch auf einen Ordner über dem
Zustandsverzeichnis, einen Entpacker ohne Ziel, solange die Shell in diesem
Ordner steht, einen Dekomprimierer, der den Namen ohne Endung schreibt, und
die `--output`-Datei und den Ordner von `-C` oder `--work-tree` bei git; ein Agent
schlägt eine Änderung mit `loomux config set|unset … --global
--propose` vor. Ein Programm, das die Datei selbst öffnet (`python -c`), und ein
Pfad, den die Shell berechnet (`$(cygpath …)`, `($env:X + '…')`), kommen
durch, der strikte Modus hält auch den zweiten nicht auf. Eine
Regel, deren Glob sich nicht lesen lässt, verweigert.

Dieselben zwei Regellisten darf die `config.toml` im Zustandsverzeichnis
tragen. Sie gelten in jedem Projekt zusätzlich zu den eingebauten und den
Regeln des Projekts, kein Projekt hebt eine auf; der Wächter liest sie vor
denen des Projekts (`config.ReadEffectivePolicy`), und die Ablehnung nennt
alle treffenden Gründe. Sie sperren das Schreiben, nicht das Lesen, und nur
dort, wo der Hook läuft. Eine kaputte globale Datei verweigert jeden Aufruf und
nennt ihren Pfad, nur ein reiner Graph-Hinweis läuft durch; ein `[guard]`
darin ist ein Fehler, denn `mode` und `graph_hint` sind Sache des Projekts.

## Wie die Schranke entscheidet

Die Registry im Zustandsverzeichnis (`registry.toml`) führt die Bereiche. Eine
Wurzel wird beschreibbar als Wiki-Pfad eines nicht schreibgeschützten Bereichs
oder als Pfad eines Bereichs mit `workspace = true`. `Decide` prüft danach
in fester Folge:

1. **Das Manifest** `.loomux/config.toml` fasst kein schreibendes Werkzeug an:
   dort liest die Schranke ihre eigenen Grenzen.
2. **Eine Datei mit mehr als einem Link** wird nicht geschrieben (siehe
   [Hardlinks](#hardlinks)).
3. **Offene Ziele** — Memory der Agenten, Scratchpad der Sitzung, die Dateien
   aus `open.toml` — gehen durch, bevor die Registry gelesen wird.
4. **Die Registry**; ist sie unlesbar, verweigert jeder weitere Write.
5. **Schreibgeschützte Zonen** schlagen jede umgebende beschreibbare Wurzel.
6. **Die Allow-List:** was unter keiner Wurzel liegt, wird verweigert, außer
   einem `proposal.md` im Prüfzentrum.

Ein Wiki-Bundle darf sich über ein eingechecktes Manifest selbst als Wurzel
erklären — aber nur, wenn der genannte Bereich bekannt, nicht
schreibgeschützt und **dasselbe Repository** wie das Manifestverzeichnis ist.
Sonst könnte jeder Baum einen registrierten Scope nennen und sich öffnen.

## Hardlinks

Ein Hardlink ist kein Link, dem die Auflösung folgen könnte, sondern ein
zweiter Name derselben Datei: nach `ln ../B/.loomux/config.toml notes.txt`
oder `mklink /H` ändert ein `Edit` auf `notes.txt` das Manifest, und jede
Regel über Namen sieht nur `notes.txt`. Darum verweigern **beide Stufen**
jedes bestehende Schreibziel mit mehr als einem Link (POSIX `st_nlink`,
Windows `nNumberOfLinks`), die Policy auch für die Schreibziele einer
Shell-Zeile (`echo x >`, `cp`). `guard.HardLinked` stellt beiden dieselbe
Frage und dieselbe Begründung. Ein neues Ziel, ein Verzeichnis und das
Entfernen eines Namens (`rm`, die Quelle eines `mv`) bleiben offen.

**Die Zahl entscheidet, nicht der Ort der anderen Namen.** Die feinere
Regel, nur zu verweigern, wenn ein weiterer Name geschützt ist oder außerhalb
der Wurzeln liegt, scheitert an drei Stellen: POSIX nennt die anderen Namen
nicht, man müsste das Dateisystem ablaufen; Windows nennt sie
(`FindFirstFileNameW`), aber jeder Name bräuchte das ganze Urteil beider
Stufen, und die Policy eines fremden Projekts kennt der Wächter nicht — ein
unbeurteilbarer Name würde ohnehin verweigert. Und legitim bearbeitet ein
Agent keine solche Datei: pnpm und uv verlinken ihre Speicher nach
`node_modules` und `.venv`, und ein Write dort träfe jedes Projekt, das den
Speicher teilt. Kosten: im Hook nicht messbar (warm 15,5 gegen 15,0 ms beim
`Edit`; `docs/de/benchmarks.md`, 2026-10-17).

## Verknüpfte Worktrees

Ein verknüpfter Git-Worktree eines `workspace`-Bereichs ist **ohne eigenen
Registry-Eintrag** beschreibbar. Gefragt wird erst, wenn ein Ziel sonst
nirgends hinfällt; ein Write im Hauptcheckout liest keine Git-Datei mehr als
vorher. Die Suche klettert vom Ziel aufwärts und nimmt das erste Verzeichnis,
für das alles gilt:

- `.git` ist eine **reguläre Datei**, kein Symlink, mit `gitdir: `;
- im Verwaltungsverzeichnis stehen `commondir` und ein Rückverweis `gitdir`,
  der auf genau diese `.git`-Datei zeigt;
- das gemeinsame Git-Verzeichnis ist das eines `workspace`-Bereichs, und das
  Verwaltungsverzeichnis liegt **direkt unter `<common>/worktrees`**.

Die letzte Bedingung kam aus dem Abschluss-Review: ohne sie hätte das
Verwaltungsverzeichnis eines verschachtelten fremden Repos auf den Workspace
zeigen und einen Baum außerhalb öffnen können.

**Kein `git`-Prozess, nur Dateien lesen.** `git rev-parse` kostete gemessen
42 ms je Aufruf; mit zwei Aufrufen je Write wäre der Hook über seinem
Zielwert gelandet. Aus demselben Grund fragt seit dem 2026-09-16 auch der
Repository-Vergleich des Manifests die Zeigerdateien statt `git`
(Worktree-Write vorher 65,3 ms warm, danach 34,6 ms; die Messung steht in
`docs/de/benchmarks.md`). Folgen, gewollt: ein verschobener Worktree bleibt bis
`git worktree repair` gesperrt; Submodule und `--separate-git-dir` erkennt
der Dateibefund nicht. Die Abweichung von Pythons `git rev-parse` wirkt in
beide Richtungen — meist verengt sie, in drei Randfällen (`safe.directory`,
Bare-Repository, POSIX-Dateisystemgrenze) öffnet sie einen echten Checkout
desselben Repositorys.

## Einzelne Dateien: `open.toml`

Anlass: die globale Anweisung des Nutzers lässt jeden Agenten
`~/.claude/AGENT_LEARNINGS.md` pflegen, die Memory-Ausnahme öffnet aber nur
`projects/<projekt>/memory/`. **Konfiguration statt Code:** fest im Code
bekäme jeder loomux-Nutzer eine beschreibbare, als Anweisung geladene Stelle
unter `~/.claude`.

`<Zustandsverzeichnis>/open.toml` kennt nur `files = [...]`: absolute Pfade,
keine Verzeichnisse, keine Globs, nichts im Zustandsverzeichnis selbst.
**Ganz oder gar nicht** — ein unbrauchbarer Eintrag, ein fremder Schlüssel
oder kaputtes TOML, und nichts öffnet; die Ablehnung endet dann mit
`<pfad> is ignored: <grund>`. Eine eigene Datei statt einer Tabelle in der
Registry, weil `loomux area add` die Registry neu schreibt und eine Tabelle
daneben still verlöre. Kosten: 0,08 ms ohne Datei, 0,22 ms mit einem Eintrag.

## Der Schalter `llm_config`

Standardmäßig fasst kein Agent das Manifest an. Ein Mensch kann das pro
Projekt aufheben: `loomux config llm on` setzt `llm_config = true` am
Area-Eintrag der Registry (fehlt das Feld, gilt aus; `loomux init` fragt
zuletzt danach, Vorgabe Nein). Nur ein Mensch schaltet; der Wächter verweigert
`config llm on|off` einem Agenten.

**Was er öffnet:** genau die `.loomux/config.toml` direkt unter dem Pfad der
Area, an den drei Sperrstellen (eingebaute Pfadregel, Shell-Zeilenerkennung,
Schritt 1 von `Decide`), und `loomux config set|unset … --yes` als einzelne
einfache Zeile ohne `--root`, `--global`, `--propose`. Ein verknüpfter
Worktree folgt dem Schalter seines Hauptcheckouts.

**Wessen Schalter:** der der Wurzel, für die der Hook installiert ist
(`--root`). Der Schalter einer anderen Area öffnet dort nichts, auch nicht
über einen Link im Projekt; `Decide` öffnet nur das Manifest dieser Wurzel.
Ein Dateiwerkzeug erreicht es nur über einen vollen Pfad, der dorthin
aufgelöst wird, oder auf das Manifest eines verknüpften Worktrees desselben
Repositorys: Claude Code übergibt als `CLAUDE_PROJECT_DIR` den Hauptcheckout,
auch wenn die Sitzung in einem solchen Worktree arbeitet. Eine Shell-Zeile (der relative Pfad und `config set|unset`)
zählt nur, wenn das `cwd` des Hook-Aufrufs genau auf die Wurzel aufgelöst
wird: Die Shell behält ein `cd` über Aufrufe hinweg, und nach einem `cd` in
ein verschachteltes oder benachbartes Projekt landete dieselbe Zeile in
dessen Manifest. Fehlt `cwd` oder weicht es ab, bleibt die Shell zu.

**Was zu bleibt:** jedes andere Manifest, `armed.toml`, `.loomux/state`, die
Registry, `open.toml`, die globale Config, `rm -rf .loomux`. Ist `.loomux`
oder `config.toml` ein Symlink oder eine Junction aus dem Projekt hinaus oder
hat `config.toml` einen zweiten Namen (Hardlink),
bleiben beide Ausnahmen zu. Unlesbare Registry, keine Area an der Wurzel oder
ein Feld ohne bool zählt als aus (Grundsatz 5: Wächter scheitern
geschlossen). Jede Ablehnung, solange der Schalter das Manifest öffnet, endet mit einem Satz, der
das nennt und `loomux config llm off` zeigt.

**Gewollt:** mit dem Schalter an kann das LLM auch Prüfungen lockern
(`guard.mode`, `modules.*`, `[policy]`); das übernimmt der Mensch beim
Einschalten ([Architektur-Grundsätze](architektur-grundsaetze.md)).

## Was bewusst nicht geschieht

- Die Dateiwerkzeuge des Hosts schreiben die Grenzen nicht: das Manifest ist
  gesperrt, auch gegen die Shell-Schreibzugriffe, die der Wächter erkennt, und
  Registry wie `open.toml` liegen außerhalb jedes geöffneten Baums, solange kein
  registrierter Baum das Zustandsverzeichnis umschließt. Shell-Zeilen und die
  Dateiwerkzeuge von MCP-Servern prüft die Schranke nicht gegen das
  Zustandsverzeichnis ([Architektur-Grundsätze](architektur-grundsaetze.md)).
  Das alles gilt innerhalb der Grenzen, die Abschnitt 7 von `docs/de/hooks.md`
  in einer Liste führt.
- Allow-List entfernen oder die Schranke aus dem Hook nehmen: verworfen, weil
  beides fremde Repos, Nutzerdateien und Vault-Quellen öffnet.
- Das Registry-Feld `wiki` geht nicht auf Worktrees über; Bundles dort deckt
  ihr Manifest ab ([Datenmodell und Bereiche](datenmodell-und-bereiche.md)).

## Was offen bleibt

- **`llm_config`, kaputtes Manifest:** Ein Manifest, das ein Agent kaputt
  schreibt, lässt das Lesen der Registry scheitern, weil die Schranke dabei
  jedes Manifest der Bereiche mitliest; sie verweigert
  dann jeden Write, bis ein Mensch es repariert (scheitert geschlossen).
- **Hardlink:** Das Anlegen des Links (`ln`, `mklink /H`,
  `New-Item -ItemType HardLink`) beurteilt keine Stufe, nur das Schreiben
  danach. Ein Link, der zwischen Prüfung und Write entsteht, und ein Skript,
  das über einen Hardlink schreibt, ohne dass die Zeile den Namen als
  Schreibziel nennt, kommen durch.
- **`llm_config`, Mensch von Hand:** Ein Mensch, der `config set` selbst
  ausführt, schreibt weiterhin durch ein verknüpftes `.loomux`.
- **Ein Browser, den ein Agent steuert:** Der Wächter hindert einen Agenten
  daran, `loomux ui` auszuführen, nicht daran, einen Browser zu benutzen, den
  ein Mensch darauf geöffnet hat; die Sitzung gilt, bis der Prozess endet
  (`loomux ui stop` beendet sie, aber nur, wenn `/health` die PID aus
  `ui.json` nennt und der Prozess ein Binary namens `loomux` ausführt).
- **Ein `loomux ui`, das die Zeile außer Sicht buchstabiert:** Der Wächter
  verweigert die berechneten Formen, die er benennen kann (aufgelöstes erstes
  Argument, `xargs`, Aufrufoperator auf `gcm loomux`, `Process.Start`,
  Aufgabenplanung). Einige kommen durch: ein Programmname aus Teilen
  (`& ('loo'+'mux') ui`), eine Skriptdatei, die die Zeile schreibt und
  startet, eine dekodierte Zeile, die an eine Shell geht, und die Binärdatei
  oder ihr Ordner unter einem anderen Namen (Kopie, Link, Build); die Tests
  halten alle als festgehaltene Durchgänge. Ein Ordner namens loomux in einem
  Pfad (`~/src/loomux/docs`) ist dagegen kein Start; `cmd/loomux/…` und ein
  relativer Pfad, der bei `loomux/` beginnt (`./loomux/.` von innerhalb
  `cmd`), zählen weiter als das Paket.
- **`llm_config` unter Antigravity:** Antigravity schickt kein `cwd`; dort
  bleibt die Shell-Ausnahme zu, und nur das Dateiwerkzeug öffnet das
  Manifest.
- **`.git`-Datei außerhalb aller Wurzeln:** angelegt über ein Werkzeug, das
  die Schranke nicht prüft (etwa Bash), kann sie auf ein selbst angelegtes
  Verwaltungsverzeichnis im Workspace zeigen.
- **Worktree-Sitzungen:** Scope und Zustandsverzeichnis einer Sitzung im
  Worktree sowie `loomux worktree link|unlink|remove` lagen außerhalb dieser
  Stufen.

Quellen: Konfigurations-, Hook- und Benchmark-Seite der Nutzerdoku; Policy und Reihenfolge aus
`internal/brain/guard/guard.go` und `internal/hooks/guard.go`.

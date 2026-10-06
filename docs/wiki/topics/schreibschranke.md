---
type: Topic
title: Die Schreibschranke
description: Wo ein Agent schreiben darf — Registry, Manifest und Policy, verknüpfte Worktrees ohne eigenen Eintrag und einzelne Dateien aus open.toml.
open_conflicts: 0
realization: implemented
implemented_in: 68f791cd
sources:
  - id: konfiguration
    resource: brain://project/loomux/docs/de/configuration.md
    doc_id: 01M47W91R50RT15H2ES8RV7TJT
    content_hash: "sha256:bc4b1f80e63e88b2ea5ef40f9aea0e677c27fabcdbd95ef7f37867e4232746a9"
    revision: 1
  - id: hooks
    resource: brain://project/loomux/docs/de/hooks.md
    doc_id: 01M47W91R5H3G1QSXZV5C17HS9
    content_hash: "sha256:ce2e417c948d0642058cee0582b5031167c18a1ca402f3e5f694d80c40184dd1"
    revision: 1
  - id: benchmarks
    resource: brain://project/loomux/docs/de/benchmarks.md
    doc_id: 01M47W91R5MPENKN7N34H6QTNT
    content_hash: "sha256:73cf99840e93d14ff9d4cc7a385fb33fe7f8733867d1ee8523a43faeb943c1d8"
    revision: 1
---

## Wofür

Der Pre-Tool-Wächter entscheidet vor jedem schreibenden Werkzeugaufruf,
**wo** der Write landet. Zwei Stufen, in dieser Reihenfolge: erst die
**Policy** des Projekts, dann die globale **Schreibschranke**
(`internal/hooks/pretool.go`). Beide verweigern, statt zu raten: **Eine
Schranke, die bei Unklarheit öffnet, ist keine.**

## Die Policy des Projekts

Die Tabelle `[policy]` in `.loomux/config.toml` trägt zwei Regellisten:
Pfadregeln mit Globs gegen schreibende Werkzeuge und Befehlsregeln mit
regulären Ausdrücken gegen `Bash` und `PowerShell`. Eingebaute Regeln gehen
immer mit: Geheimnisse (`.env`, `*.pem`, `*.key` und andere) schreibt kein
Agent, und `loomux init`, `config`, `area add`, `merge-hook install|remove`,
`convert` und `fetch` führt ein Mensch aus. Eine
Regel, deren Glob sich nicht lesen lässt, verweigert.

## Wie die Schranke entscheidet

Die Registry im Zustandsverzeichnis (`registry.toml`) führt die Bereiche. Eine
Wurzel wird beschreibbar als Wiki-Pfad eines nicht schreibgeschützten Bereichs
oder als Pfad eines Bereichs mit `workspace = true`. `Decide` prüft danach
in fester Folge:

1. **Das Manifest** `.loomux/config.toml` fasst kein schreibendes Werkzeug an:
   dort liest die Schranke ihre eigenen Grenzen.
2. **Offene Ziele** — Memory der Agenten, Scratchpad der Sitzung, die Dateien
   aus `open.toml` — gehen durch, bevor die Registry gelesen wird.
3. **Die Registry**; ist sie unlesbar, verweigert jeder weitere Write.
4. **Schreibgeschützte Zonen** schlagen jede umgebende beschreibbare Wurzel.
5. **Die Allow-List:** was unter keiner Wurzel liegt, wird verweigert, außer
   einem `proposal.md` im Prüfzentrum.

Ein Wiki-Bundle darf sich über ein eingechecktes Manifest selbst als Wurzel
erklären — aber nur, wenn der genannte Bereich bekannt, nicht
schreibgeschützt und **dasselbe Repository** wie das Manifestverzeichnis ist.
Sonst könnte jeder Baum einen registrierten Scope nennen und sich öffnen.

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

## Was bewusst nicht geschieht

- Kein Agent schreibt die Grenzen: das Manifest ist gesperrt, und
  Registry wie `open.toml` liegen außerhalb jedes geöffneten Baums
  ([Architektur-Grundsätze](architektur-grundsaetze.md)).
- Allow-List entfernen oder die Schranke aus dem Hook nehmen: verworfen, weil
  beides fremde Repos, Nutzerdateien und Vault-Quellen öffnet.
- Das Registry-Feld `wiki` geht nicht auf Worktrees über; Bundles dort deckt
  ihr Manifest ab ([Datenmodell und Bereiche](datenmodell-und-bereiche.md)).

## Was offen bleibt

- **Restrisiko:** eine `.git`-Datei außerhalb aller Wurzeln, angelegt über
  ein Werkzeug, das die Schranke nicht prüft (etwa Bash), kann auf ein selbst
  angelegtes Verwaltungsverzeichnis im Workspace zeigen.
- Scope und Zustandsverzeichnis einer Sitzung im Worktree sowie
  `loomux worktree link|unlink|remove` lagen außerhalb dieser Stufen.

Quellen: Konfigurations-, Hook- und Benchmark-Seite der Nutzerdoku; Policy und Reihenfolge aus
`internal/brain/guard/guard.go` und `internal/hooks/guard.go`.

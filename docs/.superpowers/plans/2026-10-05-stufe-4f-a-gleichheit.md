# Stufe 4f PR A: Gleichheitsprüfung Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Belegen, dass loomux alles trägt, was `ultraloom` und `ultra-brain` auf ihrem letzten Stand können, und dass die Aufnahmen unter `testdata/cases/*-source` diesem Stand entsprechen. Ergebnis ist die Akte `parity/stufe-4f.md`. Nichts wird gelöscht, kein Produktionscode ändert sich.

**Architecture:** Drei Prüfungen, jede mit eigener Tabelle in der Akte:
- Gleichheitsbeleg: jeder Aufnahme-Tag gegen jeden Zweigkopf.
- Stichprobe: Neuaufnahme mit den vorhandenen `record_all.sh`. Verglichen wird per `git diff` gegen die versionierten Aufnahmen, danach wird zurückgesetzt.
- Inventur: die Oberfläche beider Repos über alle Zweige und ungetrackten Dateien, gegen loomux und die Nachträge.

Funde werden Nachträge in der Fusions-Spec und dem Nutzer vorgelegt.

**Tech Stack:** git, bash (Git Bash), `loomux dev record-case`, die Python-Referenz in `C:/Users/micro/Documents/#GIT/ultra-brain/.venv`.

**Spec:** `docs/.superpowers/specs/2026-10-05-loomux-stufe-4f-design.md`, Abschnitt „PR A: Gleichheitsprüfung“; Fusions-Spec „#24 im Einzelnen“, Schritt 1.

## Global Constraints

- Ursprungsrepos: `C:/Users/micro/Documents/#GIT/ultraloom` und `C:/Users/micro/Documents/#GIT/ultra-brain`. Beide werden nur gelesen; kein Checkout, kein Commit, kein `git clean` dort. `git fetch --all` ist erlaubt.
- Bekannte Tags: ultraloom `loomux-1a-source` = `9d01a60`; ultra-brain `loomux-1a-source` = `loomux-3-source` = `3cc72d2`.
- Jede Aufnahme unter `testdata/cases/*-source` stammt von einem dieser Tags (Quellenzeilen der Akten `parity/stufe-*.md`).
- Eine Neuaufnahme schreibt in den Baum dieses Worktrees. Nach jeder Stufe wird mit `git checkout -- testdata/cases/<s>-source` und `git clean -fdq testdata/cases/<s>-source` zurückgesetzt. Danach muss `git status --porcelain testdata/` leer sein, bevor die nächste beginnt.
- Während einer Aufnahme arbeitet niemand sonst im Worktree. Commits laufen erst nach dem Zurücksetzen.
- `$SCRATCH` ist das Scratchpad der ausführenden Sitzung (absoluter Pfad, am Anfang jeder Shell gesetzt). Jede Ausgabe eines langen Laufs geht erst ganz in eine Datei dort, dann wird gefiltert.
- Vor jeder Neuaufnahme kommen die Werkzeugversionen in die Akte: `brain-mcp.exe --version`, die Python-Version der `.venv` (`.venv/Scripts/python.exe --version`), `go version`, `loomux version` des Aufnahme-Binarys, für 4d dazu `pdftotext -v`. Weicht eine von der Version der Originalaufnahme ab (Akte der Stufe), steht das neben dem Befund.
- Akte und Nachträge sind deutsch; Commits englisch, ohne Arbeitspapier-Namen.
- Eine Aussage in der Akte nennt den Befehl, der sie belegt.

## Review Focus

1. **Ein Seitenzweig trägt Verhalten, das in keinem Nachtrag steht.** Beispiel: `feature/agent-harness` mit 61 Commits. Erwartet wird je Commit eine Zuordnung, nicht „Zweig gesichtet“. Eigentümer: Task 3.
2. **Ungetrackte oder geänderte Dateien in den Ursprungs-Checkouts:** ultraloom hat `.gitignore` und `docs/wiki/log.md` geändert und `.githooks/post-merge` ungetrackt. Beides gehört in die Inventur, nicht nur Commits. Eigentümer: Task 3.
3. **Eine Neuaufnahme unterscheidet sich nur in Zeitstempeln oder Pfaden.** Das wird als „nicht reproduzierbar, Ursache X“ erklärt, nicht als Gleichheit verbucht und nicht verschwiegen. Eigentümer: Task 2.
4. **Ein `record_all.sh` scheitert am heutigen loomux,** zum Beispiel weil ein Flag von `dev record-case` sich geändert hat. Dann wird die Ursache festgehalten, ohne das Skript stillschweigend zu reparieren. Eine Reparatur ist eine eigene, benannte Zeile in der Akte. Eigentümer: Task 2.
5. **Der Baum ist nach der Stichprobe nicht sauber.** `git status --porcelain` muss vor jedem Commit leer sein, außer Akte und Spec. Eigentümer: Task 2, Schritt „Zurücksetzen“.

---

### Task 1: Gleichheitsbeleg und Akte anlegen

**Files:**
- Create: `docs/.superpowers/parity/stufe-4f.md`

- [ ] **Step 1: Stände lesen, Ausgabe in den Scratchpad**

```bash
for r in ultraloom ultra-brain; do d="C:/Users/micro/Documents/#GIT/$r"; git -C "$d" fetch --all --quiet; echo "== $r"; git -C "$d" tag -l | while read t; do echo "$t $(git -C "$d" rev-parse --short "$t^{}")"; done; git -C "$d" for-each-ref --format='%(refname:short) %(objectname:short) %(committerdate:short)' refs/heads refs/remotes; git -C "$d" status --short; done > "$SCRATCH/staende.txt" 2>&1
```

- [ ] **Step 2: Akte anlegen**

Kopf: Quelle, Datum, `origin/master`-Commit von loomux. Abschnitt „1. Gleichheitsbeleg“: eine Tabelle je Tag mit diesen Spalten:
- Tag und Commit
- `master` lokal und `origin/master`
- gleich ja/nein, mit dem Befehl, der es belegt

Darunter je Stufe die Zuordnung zum Tag: 1a, 2a, 2c und 4a-2 (`cmd/init`) gehen auf ultraloom; 1b-1, 1b-2, 3a–3c, 4a-2 (Skills, post-merge), 4c-1 und 4d auf ultra-brain. Jede Zuordnung wird aus der Quellenzeile der jeweiligen Akte zitiert. 2b nennt `ultraloom commit-msg` ohne Tag (`stufe-2b.md:3`); ihr Stand wird über `git log loomux-1a-source..master -- src/ultraloom/commit tests/commit` in ultraloom belegt.

Ist ein Tag ungleich `master`: Diese Stufe kommt in Task 2 zur Neuaufnahme, auch ohne Skript. Fehlt das Skript, wird der Fall dem Nutzer vorgelegt.

- [ ] **Step 3: Commit** — `docs: record the equality of the reference tags with both repositories`

### Task 2: Stichprobe der Reproduzierbarkeit

**Files:**
- Modify: `docs/.superpowers/parity/stufe-4f.md`, Abschnitt „2. Stichprobe“
- Liest: `docs/.superpowers/parity/stufe-{3a,3b,3c,4a-2,4c-1,4d}-orakel/record_all.sh`

- [ ] **Step 1: Aufnahmeverzeichnis vorbereiten.** Ein leeres Verzeichnis `$SCRATCH/rec` mit:
  - `bin/loomux.exe` (`go build -o "$SCRATCH/rec/bin/loomux.exe" ./cmd/loomux`)
  - `fakeqmd/qmd.exe` (`go build -o "$SCRATCH/rec/fakeqmd/qmd.exe" ./internal/dev/fakeqmd/_qmd`)

  Was jede Stufe sonst braucht, steht im Kopf ihres `record.sh`. Vor der ersten Aufnahme lesen und in der Akte je Stufe notieren.
- [ ] **Step 2: Je Stufe, nacheinander, nie parallel:**

```bash
RECORD_DIR="$SCRATCH/rec" bash docs/.superpowers/parity/stufe-3b-orakel/record_all.sh > "$SCRATCH/rec-3b.log" 2>&1; echo "exit=$?"
git status --porcelain testdata/cases/3b-source > "$SCRATCH/diff-3b-files.txt"
git diff --stat testdata/cases/3b-source > "$SCRATCH/diff-3b-stat.txt"
git diff testdata/cases/3b-source > "$SCRATCH/diff-3b.patch"
```

- [ ] **Step 3: Unterschiede einordnen.** Jede geänderte, neue oder fehlende Datei fällt in genau eine Klasse:
  - (a) gleich
  - (b) nur Zeit, Pfad oder Reihenfolge, mit der Zeile, die es zeigt
  - (c) Verhalten anders

  Fällt der Lauf selbst (exit ≠ 0), wird Review Focus 4 angewandt. In die Akte kommt je Stufe eine Zeile mit diesen Spalten: Anzahl Fälle, Anzahl (a)/(b)/(c), Exit, Log-Pfad und die Erklärung jeder (b)- und (c)-Datei.
- [ ] **Step 4: Zurücksetzen und prüfen**

```bash
git checkout -- testdata/cases/3b-source && git clean -fdq testdata/cases/3b-source && git status --porcelain testdata/
```
Expected: leere Ausgabe.

- [ ] **Step 5:** Die Schritte 2–4 laufen für 3a, 3c, 4a-2, 4c-1 und 4d. Für 1a bis 2c gibt es kein Skript; die Akte vermerkt: „Beleg allein, Task 1“.
- [ ] **Step 6: Commit** — `docs: record whether the reference recordings reproduce`. Ein (c)-Befund wird dem Nutzer vor dem Commit genannt.

### Task 3: Inventur beider Repos

**Files:**
- Modify: `docs/.superpowers/parity/stufe-4f.md`, Abschnitt „3. Inventur“
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`: neue Nachträge ab #32, je Fund eine Zeile in der Tabelle „Nachgetragen“, Freigabe leer

- [ ] **Step 1: Oberfläche auf `master` beider Repos auflisten,** am Code statt an der Doku. Zu erfassen sind Befehle und Unterbefehle (CLI-Einstieg), Flags, Konfigurationsschlüssel, Hooks, Skills, Vorlagen und MCP-Werkzeuge. Je Stelle die Fundstelle (`datei:zeile`) und die Zuordnung, und zwar genau eine davon:
  - gebaut: der loomux-Befehl oder die Datei, die es trägt
  - Stufe oder Folgeprojekt: die Zeile in der Fusions-Spec
  - freigegeben weggefallen: der Nachtrag mit Freigabe

  Ausgangspunkt ist die Tabelle „Nachgetragen“ der Fusions-Spec samt den Paritätsakten. Was dort schon steht, bekommt nur den Verweis.

  Die Arbeit ist breit, aber gleichförmig. Sie kann je Repo an einen Subagenten gehen; der Controller prüft jede Zuordnung „gebaut“ per `grep` im loomux-Code und jede Zuordnung „Nachtrag“ per Zeilennummer.
- [ ] **Step 2: Seitenzweige Commit für Commit**

```bash
for r in ultraloom ultra-brain; do d="C:/Users/micro/Documents/#GIT/$r"; for b in $(git -C "$d" for-each-ref --format='%(refname:short)' refs/heads refs/remotes); do git -C "$d" log --format="$r $b %h %s" master.."$b"; done; done | sort -u -k3,3 > "$SCRATCH/seitenzweige.txt"
```

  Jeder Commit (dedupliziert über den Hash) bekommt eine Zuordnung wie in Step 1. Bekannte Nachträge sind #20 und #29 (`feature/artefakte-nach-lebensdauer`), #21 (`claude/wiki-stufe-2`) und #22. Ein Commit, der nur Doku oder Plan trägt, wird als solcher vermerkt.
- [ ] **Step 3: Ungetrackte und geänderte Dateien** (`git status --short` aus Task 1) werden genauso eingeordnet. Das geht gegen Nachtrag #22, der schon eine Durchsicht der ungetrackten Dateien festhält.
- [ ] **Step 4: Funde.** Jede Stelle ohne Zuordnung wird ein Nachtrag mit Quelle, Stelle, Vorschlag (Stufe, Folgeprojekt, Roadmap oder Wegfall) und Begründung; die Freigabe bleibt leer. Gibt es keinen Fund, schreibt die Akte das ausdrücklich mit der Zahl der geprüften Stellen und Commits.
- [ ] **Step 5: Commit** — `docs: inventory both reference repositories against loomux`

### Task 4: Plan nachziehen und PR

**Files:**
- Modify: `docs/en/migration.md`, `docs/de/migration.md` (Zeile 4f, Spalte „was bleibt“; Status „offen“, Priorität 3 wie in der Spec)

- [ ] **Step 1:** Zeile 4f in beiden Sprachen: „Die Gleichheitsprüfung ist gelaufen (`parity/stufe-4f.md`): <Ergebnis in einem Satz>; offen sind PR B, PR C und der Neustart (Nachtrag #31).“ Danach `go test ./internal/plancheck/ -count=1` → PASS.
- [ ] **Step 2:** Dem Nutzer vorlegen:
  - jeden neuen Nachtrag, zur Freigabe
  - jeden (c)-Befund der Stichprobe, zur Entscheidung

  PR B beginnt erst, wenn alles entschieden ist.
- [ ] **Step 3:** Commit `docs(migration): note the equality check of the reference repositories`. Dann `release-pr`-Skill mit Label `release:none` und ohne Changelog-Block. Der PR trägt auch die beiden Specs und Pläne dieses Zweigs. Den Push-Befehl nennt der Agent dem Menschen.

# Umzug von Wiki und Doku nach loomux — Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die freigegebenen 25 Wiki-Seiten, 11 Rohquellen und 18 Arbeitspapiere
der beiden alten Repos liegen in loomux, `loomux wiki-gate` meldet keine
Lint-Zeile, und `brain read` und `brain neighbors` beantworten den Bereich
`project/loomux`. (`brain catalog --scope` bleibt außen vor — es liest ein
Artefakt im Repowurzelverzeichnis, das erst `reindex` in Stufe 3 schreibt;
Task 6 misst das nach.)

**Architecture:** Der Umzug ist eine Datenbewegung, keine Codeänderung. Drei
Schichten, in dieser Reihenfolge: erst die Rohquellen (byte-gleich, damit die
`content_hash`-Kette der Wiki-Seiten prüfbar bleibt), dann die Wiki-Seiten (nur
die `resource`-Zeilen und die Typen werden angefasst), zuletzt die inhaltlichen
Korrekturen gegen die freigegebene Paritätsliste der Stufe 1b-1. Jede Schicht
endet mit einem Nachweis: Hashvergleich, `loomux lint`, `loomux wiki-gate`.

**Tech Stack:** Go 1.25 (nur lesend benutzt: `loomux lint`, `loomux wiki-gate`,
`loomux brain …`), Git, PowerShell/Bash für Kopien und Hashvergleiche.

**Spec:** [2026-09-14-loomux-fusion-design.md](../specs/2026-09-14-loomux-fusion-design.md),
Abschnitt „Datenumzug", Punkt 1.
**Freigegebene Prüfliste:** [umzug-wiki-doku.md](../parity/umzug-wiki-doku.md) —
183 Zeilen, alle am 2026-09-16 freigegeben. Sie ist die Autorität darüber,
welche Datei mitzieht; dieser Plan sagt nur, wie.
**Paritätsliste der Stufe 1b-1:** [stufe-1b-1.md](../parity/stufe-1b-1.md) —
die 54 Zeilen, gegen die der Inhalt in Task 5 geprüft wird.

## Global Constraints

- **Quellen:** `C:/Users/micro/Documents/#GIT/loomux-src/ub` (ultra-brain) und
  `.../loomux-src/ul` (ultraloom). Beide werden **nur gelesen**; in den alten
  Repos wird nichts geändert und nichts committet.
- **Ziel:** dieser Worktree. Das Wiki liegt unter `docs/wiki` — so steht es im
  Manifest (`[layout] wiki = "docs/wiki"`) und in der Registry.
- `.loomux/config.toml` und `%LOCALAPPDATA%\loomux\registry.toml` schreibt kein
  Agent. Wer eine Änderung dort braucht, schlägt sie vor.
- Rohquellen ziehen **byte-gleich** um: kein Zeilenende, kein Link, kein
  Leerzeichen wird angefasst. Der Nachweis ist ein Hashvergleich.
- Doku, Prosa und Kommentare deutsch; Code, Bezeichner, Commits und Meldungen
  englisch. Commits tragen den Nutzer als Autor und nennen kein Modell.
- Nichts Persönliches wird committet: aus den alten Repos ziehen nur die
  Dateien um, die die Prüfliste nennt.
- Kein Test und kein Schritt startet qmd, eine GPU, `uv` oder die
  Python-Referenz. `reindex` und `reconcile` gibt es in Go nicht (Stufe 3);
  der Suchindex bleibt darum nach diesem Plan unverändert, und das ist in
  Ordnung.
- Ein Shell-Befehl je Aufruf. Mehrzeilige Commit-Nachrichten über eine Datei
  und `git commit -F`.
- Das Tor läuft bei jedem Commit (`.githooks/pre-commit`); `bin/loomux.exe`
  wird dabei neu gebaut.

## Was heute schon gilt (gemessen, nicht vermutet)

- `loomux lint <datei>` prüft eine Seite, `loomux wiki-gate` das Bündel. Die
  README nennt fälschlich `loomux wiki gate`; Task 7 richtet das.
- Der Torbruch entsteht nur bei `Severity == Error`: `missing-type` (leeres
  `type`), `dead-link`, `conflict-count`. **Unbekannter Typ und Waise sind
  Warnungen** (`internal/brain/wiki/lint.go:62-99`, `gate.go:105`).
- Erlaubte Typen: `concept`, `guide`, `decision`, `reference`, `architecture`,
  `log`, `person`, `domain`, `system`, `component`, `spec`, `topic`. Der Wert
  wird kleingeschrieben verglichen. Das Bündel bringt `Topic` (passt),
  `Source`, `Entity`, `Synthesis` (passen nicht) mit. Das Manifest ist dabei
  nicht die Lücke: loomux liest `[wiki] types` (`internal/config/manifest.go:115`,
  `:221`), und `config.Manifest.KnowsType` kennt die vier Herkunftstypen
  bereits — aber der Wiki-Lint fragt das Manifest nie, er prüft gegen seine
  eigene Liste (`lint.go:14`, `:69`), und `KnowsType` hat außerhalb der Tests
  keinen Aufrufer.
- Gerüstdateien (`_schema.md`, `index.md`, `log.md`, `audit.md`,
  `_identities.tsv`) werden nicht bewertet, zählen aber als Linkquelle.
- **`wiki-drift` kann für ein Wiki im selben Repository nie greifen.**
  `CheckWikiGate` meldet Drift nur, wenn Code geändert wurde und das Wiki nicht
  (`gate.go:93`). Beide Fragen stellt `getGitChangedFiles`
  (`gate.go:48-74`), das nur `cmd.Dir` setzt — und `git status --porcelain`
  meldet aus jedem Unterverzeichnis den Stand des ganzen Repos. Für
  `docs/wiki` ist `wikiChanged` also stets gleich `codeChanged`, und die
  Bedingung `wikiPath != projectRoot` entscheidet nichts mehr. Das Tor liest
  außerdem den Arbeitsbaum, nicht die Commit-Historie. Heute bricht das nichts:
  `wiki-gate` steht weder in `.githooks/pre-commit` noch in
  `.claude/settings.json` noch in `[verify]`; nur der post-edit-Hook lintet die
  gerade bearbeitete Seite. Den Code repariert ein eigener Vorgang, nicht
  dieser Plan.
- Nachgerechnet am 2026-09-16: **kein toter Link** entsteht durch den Umzug —
  keine Inhaltsseite verlinkt einen Katalog. **Keine Waise** entsteht —
  jede der 24 Inhaltsseiten wird von mindestens einer anderen Inhaltsseite
  verlinkt. Von zwölf zitierten Quellen stimmen elf `content_hash`; die der
  Architektur-Spec ist seit deren Bearbeitung veraltet.

## Dateien und Zielorte

| Was | Anzahl | Von | Nach |
|---|---:|---|---|
| Wiki-Inhaltsseiten | 24 | `ub/docs/wiki/{entities,sources,syntheses,topics}/` | `docs/wiki/<gleiches Unterverzeichnis>/` |
| Bündelregelwerk | 1 | `ub/docs/wiki/_schema.md` | `docs/wiki/_schema.md` |
| Zitierte Pläne | 10 | `ub/docs/.superpowers/plans/` | `docs/.superpowers/plans-ub/` |
| Zitierte bench-Quelle | 1 | `ub/bench/2c1/entscheidung-46.md` | `docs/.superpowers/bench-ub/entscheidung-46.md` |
| Papiere mit loomux-Stufe (ub) | 7 | `ub/docs/.superpowers/specs/` | `docs/.superpowers/specs-ub/` |
| Produktentscheidungen (ub) | 1 | `ub/docs/produkt-design.md` | `docs/.superpowers/specs-ub/2026-09-06-produkt-design.md` |
| Papiere mit loomux-Stufe (ul) | 10 | `ul/docs/.superpowers/specs/` | `docs/.superpowers/specs-ul/` |
| Nutzerdoku | 10 | `ub/docs/`, `ul/docs/` | wird **eingearbeitet**, nicht kopiert |
| Kataloge, Log, Audit, Register | — | — | entstehen neu (Task 6) |

Die Architektur-Spec `2026-08-18-ultra-brain-architektur-design.md` steht in
beiden Rollen: zitierte Quelle **und** Papier mit Stufe. Sie zieht einmal um,
nach `docs/.superpowers/specs-ub/`.

---

### Task 1: Rohquellen und Arbeitspapiere byte-gleich übernehmen

**Files:**
- Create: `docs/.superpowers/plans-ub/` (10 Dateien), `docs/.superpowers/specs-ub/`
  (8 Dateien), `docs/.superpowers/specs-ul/` (10 Dateien),
  `docs/.superpowers/bench-ub/entscheidung-46.md`
- Test: `docs/.superpowers/plans-ub/HASHES.txt` (der Nachweis, mitcommittet)

**Interfaces:**
- Produces: die Zielpfade, auf die Task 3 die `resource`-Zeilen der Wiki-Seiten
  umschreibt: `docs/.superpowers/plans-ub/<name>`,
  `docs/.superpowers/specs-ub/<name>`, `docs/.superpowers/bench-ub/entscheidung-46.md`.

- [ ] **Step 1: Prüfsummen der Quellen aufnehmen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-src/ub" && sha256sum \
  docs/.superpowers/plans/2026-08-18-scheibe-0-fundament.md \
  docs/.superpowers/plans/2026-08-19-scheibe-1-indexer.md \
  docs/.superpowers/plans/2026-08-20-scheibe-2a-abnahme.md \
  docs/.superpowers/plans/2026-08-20-scheibe-2a-entscheidungen.md \
  docs/.superpowers/plans/2026-08-20-scheibe-2a-nacharbeit.md \
  docs/.superpowers/plans/2026-08-20-scheibe-2a-suchkette.md \
  docs/.superpowers/plans/2026-08-21-pruefkorpus-v1.md \
  docs/.superpowers/plans/2026-08-21-scheibe-2b-messwerk.md \
  docs/.superpowers/plans/2026-08-21-scheibe-2c1-daemon.md \
  docs/.superpowers/plans/2026-08-22-scheibe-2c2-mcp-fronten.md \
  bench/2c1/entscheidung-46.md > /tmp/quellen-vorher.txt
```

Erwartet: elf Zeilen, keine Fehlermeldung.

- [ ] **Step 2: Kopieren**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && mkdir -p docs/.superpowers/plans-ub docs/.superpowers/specs-ub docs/.superpowers/specs-ul docs/.superpowers/bench-ub && cp "/c/Users/micro/Documents/#GIT/loomux-src/ub/docs/.superpowers/plans/"2026-08-1[89]*.md "/c/Users/micro/Documents/#GIT/loomux-src/ub/docs/.superpowers/plans/"2026-08-2[012]*.md docs/.superpowers/plans-ub/ && cp "/c/Users/micro/Documents/#GIT/loomux-src/ub/bench/2c1/entscheidung-46.md" docs/.superpowers/bench-ub/
```

Erwartet: `ls docs/.superpowers/plans-ub | wc -l` ergibt 10. Sind es mehr,
hat das Glob zu viel gefangen — dann Datei für Datei kopieren, die Namen stehen
in Step 1.

- [ ] **Step 3: Die sieben ub-Specs und `produkt-design.md` kopieren**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && for f in 2026-08-18-ultra-brain-architektur-design 2026-08-21-pruefkorpus-design 2026-08-25-typkatalog-und-migration-design 2026-08-27-scheibe-5-brain-maintenance-design 2026-09-04-python-nach-go-migration-design 2026-09-05-scheibe-6-lokales-modell-design 2026-09-13-schranke-memory-offen-design; do cp "/c/Users/micro/Documents/#GIT/loomux-src/ub/docs/.superpowers/specs/$f.md" docs/.superpowers/specs-ub/; done && cp "/c/Users/micro/Documents/#GIT/loomux-src/ub/docs/produkt-design.md" docs/.superpowers/specs-ub/2026-09-06-produkt-design.md
```

Erwartet: `ls docs/.superpowers/specs-ub | wc -l` ergibt 8.

- [ ] **Step 4: Die zehn ul-Specs kopieren**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && for f in 2026-08-21-teilprojekt-2-backlog 2026-08-22-pruefkette-reihenfolge-design 2026-08-25-commit-sprachpruefung-design 2026-08-25-policy-baukasten-design 2026-08-25-sitzungs-hooks-payloads 2026-08-28-installer-kern-design 2026-09-07-worktree-mirror-design 2026-09-10-antigravity-hook-messung 2026-09-10-go-hooks-drei-hosts-design 2026-09-10-wiki-flottenstandard-design; do cp "/c/Users/micro/Documents/#GIT/loomux-src/ul/docs/.superpowers/specs/$f.md" docs/.superpowers/specs-ul/; done
```

Erwartet: `ls docs/.superpowers/specs-ul | wc -l` ergibt 10.

- [ ] **Step 5: Byte-Gleichheit nachweisen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && sha256sum docs/.superpowers/plans-ub/*.md docs/.superpowers/bench-ub/*.md | awk '{print $1}' | sort > /tmp/quellen-nachher.txt && awk '{print $1}' /tmp/quellen-vorher.txt | sort | diff - /tmp/quellen-nachher.txt && echo IDENTISCH
```

Erwartet: `IDENTISCH`, kein `diff`-Ausgabeblock. Weicht eine Summe ab, hat das
Kopieren Zeilenenden umgeschrieben — dann mit `cp --preserve` bzw. `Copy-Item`
ohne Umwandlung wiederholen, nicht von Hand nachbessern.

- [ ] **Step 6: Die bench-Quelle durchsehen**

`docs/.superpowers/bench-ub/entscheidung-46.md` ist die einzige Datei, die von
außerhalb eines `docs/`-Baums kommt (97 Zeilen). Vor dem Commit einmal ganz
lesen: Sie muss eine Entscheidung über die Suche enthalten und nichts
Persönliches. Ist etwas anderes darin, bleibt sie liegen und der Bericht sagt,
was drinsteht.

- [ ] **Step 7: Den Nachweis ablegen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && sha256sum docs/.superpowers/plans-ub/*.md docs/.superpowers/specs-ub/*.md docs/.superpowers/specs-ul/*.md docs/.superpowers/bench-ub/*.md > docs/.superpowers/plans-ub/HASHES.txt
```

- [ ] **Step 8: Committen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && git add docs/.superpowers/plans-ub docs/.superpowers/specs-ub docs/.superpowers/specs-ul docs/.superpowers/bench-ub && git commit -F C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/b8f89ec2-7b8e-4b7a-8c72-d868c2b9a843/scratchpad/commit-umzug-1.txt
```

Nachrichtendatei `C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/b8f89ec2-7b8e-4b7a-8c72-d868c2b9a843/scratchpad/commit-umzug-1.txt` vorher schreiben, erste Zeile:
`Bring the sources the wiki cites into loomux`.

---

### Task 2: Die 25 Wiki-Seiten kopieren, unverändert

**Files:**
- Create: `docs/wiki/entities/` (3), `docs/wiki/sources/` (11),
  `docs/wiki/syntheses/` (2), `docs/wiki/topics/` (8), `docs/wiki/_schema.md`
- Test: Hashvergleich gegen die Quelle

**Interfaces:**
- Consumes: nichts.
- Produces: `docs/wiki/**` als Ausgangszustand für Task 3 bis 5. Nicht kopiert
  werden `index.md` (fünfmal), `log.md`, `audit.md` und `_identities.tsv` — die
  entstehen in Task 6 neu.

- [ ] **Step 1: Kopieren ohne die Gerüstartefakte**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && mkdir -p docs/wiki && (cd "/c/Users/micro/Documents/#GIT/loomux-src/ub/docs/wiki" && find . -name "*.md" ! -name "index.md" ! -name "log.md" ! -name "audit.md" -print0 | xargs -0 tar cf -) | tar xf - -C docs/wiki
```

- [ ] **Step 2: Zählen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && find docs/wiki -name "*.md" | wc -l
```

Erwartet: `25`. Steht dort 32, sind die fünf Kataloge, `log.md` und `audit.md`
mitgekommen — löschen, nicht behalten.

- [ ] **Step 3: Byte-Gleichheit nachweisen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && (cd docs/wiki && find . -name "*.md" | sort | xargs sha256sum | awk '{print $1}') | diff - <(cd "/c/Users/micro/Documents/#GIT/loomux-src/ub/docs/wiki" && find . -name "*.md" ! -name "index.md" ! -name "log.md" ! -name "audit.md" | sort | xargs sha256sum | awk '{print $1}') && echo IDENTISCH
```

Erwartet: `IDENTISCH`.

- [ ] **Step 4: Das Tor im Ist-Zustand messen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && go run ./cmd/loomux wiki-gate; echo "exit=$?"
```

Erwartet: Exit 0 oder eine `wiki-drift`-Meldung (Code geändert, Wiki nicht) —
aber **keine** `wiki-lint:dead-link`- und keine `wiki-lint:missing-type`-Zeile.
Steht dort doch eine, ist die Rechnung aus dem Abschnitt „Was heute schon gilt"
falsch: die Ausgabe wörtlich in den Bericht, dann Task 4 vorziehen.

- [ ] **Step 5: Committen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && git add docs/wiki && git commit -F C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/b8f89ec2-7b8e-4b7a-8c72-d868c2b9a843/scratchpad/commit-umzug-2.txt
```

Erste Zeile: `Move the wiki pages of ultra-brain into loomux`.

---

### Task 3: Die `resource`-Zeilen auf die neuen Pfade richten

**Files:**
- Modify: alle Seiten unter `docs/wiki/` mit einem `sources:`-Block (24 von 25; nur `_schema.md` hat keinen)

**Interfaces:**
- Consumes: die Zielpfade aus Task 1.
- Produces: Seiten, deren `resource` auf `brain://project/loomux/…` zeigt,
  während `doc_id`, `content_hash` und `revision` unberührt bleiben.

- [ ] **Step 1: Den Ist-Zustand festhalten**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && grep -rn "brain://project/ultra-brain/" docs/wiki | wc -l
```

Erwartet: `27` — 13 Verweise auf die Architektur-Spec, 13 auf Pläne, 1 auf die
bench-Datei, am 2026-09-16 über den ziehenden Bestand gezählt.

- [ ] **Step 2: Umschreiben**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && grep -rl "brain://project/ultra-brain/" docs/wiki | xargs sed -i -e 's|brain://project/ultra-brain/docs/\.superpowers/plans/|brain://project/loomux/docs/.superpowers/plans-ub/|g' -e 's|brain://project/ultra-brain/docs/\.superpowers/specs/|brain://project/loomux/docs/.superpowers/specs-ub/|g' -e 's|brain://project/ultra-brain/bench/2c1/|brain://project/loomux/docs/.superpowers/bench-ub/|g'
```

- [ ] **Step 3: Prüfen, dass nichts übrig blieb**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && grep -rn "ultra-brain/" docs/wiki | grep -v "ultra-brain-architektur-design" ; echo "rest=$?"
```

Erwartet: keine Zeile, `rest=1`. Der Dateiname
`2026-08-18-ultra-brain-architektur-design.md` bleibt, wie er ist — er ist der
Name der Datei, kein Bereichsverweis.

- [ ] **Step 4: Prüfen, dass jeder Verweis jetzt existiert**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && grep -rho "brain://project/loomux/[^ ]*" docs/wiki | sed 's|brain://project/loomux/||' | sort -u | while read -r f; do [ -f "$f" ] || echo "FEHLT $f"; done; echo fertig
```

Erwartet: nur `fertig`, keine `FEHLT`-Zeile.

- [ ] **Step 5: Prüfen, dass nur `resource`-Zeilen abweichen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && git diff --unified=0 docs/wiki | grep -E "^[-+][^-+]" | grep -vc "resource:"
```

Erwartet: `0`.

- [ ] **Step 6: Committen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && git add docs/wiki && git commit -F C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/b8f89ec2-7b8e-4b7a-8c72-d868c2b9a843/scratchpad/commit-umzug-3.txt
```

Erste Zeile: `Point the cited sources at their new home`.

---

### Task 4: Die Seitentypen belassen und den Befund festhalten

**Files:**
- Modify: `docs/wiki/_schema.md` (nur die Bereichszeile und ein Absatz)
- Create: nichts

**Interfaces:**
- Consumes: `internal/brain/wiki/lint.go:14-27` (die zwölf erlaubten Typen) und
  `docs/.superpowers/specs-ub/2026-08-25-typkatalog-und-migration-design.md`,
  §3.4 und §3.5, aus Task 1.
- Produces: nichts, was eine spätere Aufgabe braucht.

**Warum hier nichts umgeschrieben wird.** Der naheliegende Griff wäre, `Source`,
`Entity` und `Synthesis` auf loomux' Kernvokabular abzubilden. Der Typkatalog,
der mit Task 1 umzieht, verbietet das ausdrücklich: §3.5 erklärt die vier zur
**zweiten Achse** — sie sagen, *wie* eine Seite entstanden ist, nicht worüber
sie handelt — und erklärt sie für weiter gültig; §3.4 lehnt `Concept` als
Dopplung zu `Entity` ab. Eine Abbildung würde also eine freigegebene Spec
brechen, um eine Warnung loszuwerden.

Das kostet nichts am Tor: unbekannter Typ ist `Severity == Warning`
(`lint.go:69-75`), und `wiki-gate` bricht nur bei `Error` ab (`gate.go:105`).
Die 16 Seiten melden je eine Warnung, bis der Wiki-Lint das Manifest fragt:
loomux liest `[wiki] types` schon (`internal/config/manifest.go:115`, `:221`),
aber der Lint prüft gegen seine eigene Liste und ruft `KnowsType` nicht
(Paritätszeile 22, freigegeben mit Nachtrag).

- [ ] **Step 1: Den Befund messen und wörtlich festhalten**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && find docs/wiki -name "*.md" ! -name "_schema.md" -print0 | xargs -0 -n1 go run ./cmd/loomux lint 2>&1 | grep "unknown document type" | sort | uniq -c
```

Erwartet: drei Zeilen — `"source"` 11×, `"entity"` 3×, `"synthesis"` 2×; `topic`
taucht nicht auf, weil der Vergleich kleinschreibt. Steht dort etwas anderes,
gehört die Ausgabe wörtlich in den Bericht und die Aufgabe hält an.

- [ ] **Step 2: Das Tor gegenprüfen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && go run ./cmd/loomux wiki-gate 2>&1 | grep -c "wiki-lint:missing-type"
```

Erwartet: `0` — die Warnungen erreichen das Tor nicht.

- [ ] **Step 3: Das Regelwerk richten**

In `docs/wiki/_schema.md` Zeile 42 `project/ultra-brain` durch
`project/loomux` ersetzen. Unter „Die vier Seitentypen" einen Absatz ergänzen,
der den Stand benennt: die vier Herkunftstypen gelten weiter (Typkatalog §3.5);
loomux' Lint kennt daneben zwölf Kerntypen und meldet die vier als unbekannt,
bis es `[wiki] types` liest. Kein Typ wird umgeschrieben.

- [ ] **Step 4: Den Nachtrag notieren**

In `docs/.superpowers/parity/stufe-1b-1.md` bei Zeile 22 (`Weitere Prüfungen
von read_manifest`) den Nachtrag um den Satz ergänzen, dass `[wiki] types` auch
den Lint betrifft: ohne diese Zeile meldet jedes umgezogene Bündel mit eigenen
Typen Warnungen.

- [ ] **Step 5: Committen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && git add docs/wiki docs/.superpowers/parity/stufe-1b-1.md && git commit -F C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/b8f89ec2-7b8e-4b7a-8c72-d868c2b9a843/scratchpad/commit-umzug-4.txt
```

Erste Zeile: `Keep the bundle's own type axis, and say why`.

---

### Task 5: Inhalt gegen die Paritätsliste richten

**Files:**
- Modify: elf Seiten. Die Prüfliste setzt zwölf auf `aktualisieren`; `_schema.md`
  ist in Task 4 erledigt, und `topics/abnahmen-und-echte-umgebung.md` steht auf
  `behalten` und wird nicht angefasst.

**Interfaces:**
- Consumes: `docs/.superpowers/parity/stufe-1b-1.md`, die 54 freigegebenen
  Zeilen.
- Produces: Seiten, deren Aussagen zu loomux stimmen; was nur historisch gilt,
  bleibt stehen und wird als Vergangenheit gekennzeichnet.

Je Seite die Zeile, gegen die zu prüfen ist. Die Seite beschreibt jeweils das
alte System; die Korrektur ist ein Satz oder Absatz, kein Umschreiben:

| Seite | Prüfen gegen | Was heute anders ist |
|---|---|---|
| `entities/qmd.md` | Zeilen 1–2 | loomux ruft den MCP-Daemon mit `searches:[{type:"vec"}]`, `rerank:false`, ohne Erweiterung; Score ist `1/Rang` |
| `entities/brain-daemon.md` | Zeile 49 | loomux fragt nie einen brain-Daemon; die fünf Befehle laufen im Prozess |
| `topics/suche-und-profile.md` | Zeilen 1–2, 5, 51 | Die Profiltabelle stimmt weiter; neu sind der MCP-Weg, der Aufwärm-Hinweis und die Messung vom 2026-09-16 (260,7 ms warm für `fast`, Ziel 150 ms offen) |
| `syntheses/warum-fast-die-vorgabe-bleibt.md` | Zeilen 1–2 | Die Begründung trägt; der Verweis auf `project/ultra-brain` in Zeile 40 wird zu `project/loomux` |
| `topics/datenmodell-und-bereiche.md` | Zeile 3 | Zustand unter `LOOMUX_STATE_DIR`, Artefakte unter `LOOMUX_LEGACY_BRAIN_DIR` bis Stufe 3; Manifest ist `.loomux/config.toml` |
| `topics/datenschutz-und-kanaele.md` | Zeilen 54, 62 | Kanäle und `never`-Globs gelten weiter; ein Treffer aus einer nicht gefragten Sammlung wird der ersten gefragten zugeschlagen |
| `topics/brain-maintenance.md` | — | Die Pflegeschicht gibt es in Go noch nicht; Abschnitt als „ab Stufe 3" kennzeichnen |
| `topics/wiki-schicht.md` | — | Lint und Tor sind Go (`loomux lint`, `loomux wiki-gate`); die vier Herkunftstypen bleiben, loomux meldet sie als unbekannt (Task 4) |
| `topics/scheiben-und-abnahme.md` | — | „Scheiben" heißen in loomux Stufen; die Abnahmeregel („eine Stufe ist fertig, wenn …") aus der Fusions-Spec übernehmen |
| `topics/architektur-grundsaetze.md` | — | Grundsätze gelten weiter; die Arbeitsteilung „zwei Werkzeuge" ist Vergangenheit |
| `topics/abnahmen-und-echte-umgebung.md` | — | Steht auf `behalten`; **nicht anfassen** |
| `sources/architektur-spec.md` | — | Die Quelle hat sich seit `revision: 3` geändert (Hash weicht ab). Als Konflikt kennzeichnen, nicht auflösen |
| `_schema.md` | — | In Task 4 erledigt |

- [ ] **Step 1: Die elf Seiten lesen und die Sätze sammeln**

Je Seite den Satz oder Absatz notieren, der der genannten Paritätszeile
widerspricht — wörtlich, mit Zeilennummer. Das Ergebnis ist eine Liste im
Bericht, bevor eine Datei angefasst wird. Wo eine Seite der Zeile **nicht**
widerspricht, wird sie nicht geändert; das ist ein gültiges Ergebnis und gehört
in den Bericht.

- [ ] **Step 2: Die Sätze richten**

Ein Satz je Stelle, im Ton der Seite (deutsch, verdichtet, keine Aufzählung wo
vorher Prosa stand). Was historisch bleibt, bekommt ein Datum statt einer
Streichung: „Bis zum Umzug am 2026-09-16 …".

- [ ] **Step 3: Den Konflikt der Architektur-Spec eintragen**

In `docs/wiki/sources/architektur-spec.md` einen Kasten in der Form ergänzen,
die `_schema.md` vorschreibt, und `open_conflicts` von `0` auf `1` setzen:

```markdown
> [!conflict] Quelle hat sich seit der Verdichtung geändert
> Diese Seite verdichtet `revision: 3` mit `content_hash`
> `sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff`.
> Die Datei unter `docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md`
> hat am 2026-09-16 den Hash
> `sha256:2440a49696a1f61737a94fe79d27093fbe48f043fa75d8097a6a809a42b99df3`.
> Auflösen kann das erst `reconcile` (Stufe 3). Dieselbe veraltete Summe steht
> in zwölf Seiten; eine dreizehnte nennt die Spec mit
> `sha256:8cb1d728fb0b4a32a3063677dac4631a856212e695bca6da02da09cae5f8458c`.
```

- [ ] **Step 4: Lint über die geänderten Seiten**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && go run ./cmd/loomux lint docs/wiki/sources/architektur-spec.md; echo "exit=$?"
```

Erwartet: Exit 0, keine `conflict-count`-Zeile. Meldet der Lint
`open_conflicts says 0, but 1 conflict box(es) found`, wurde Step 3 nur halb
gemacht.

- [ ] **Step 5: Committen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && git add docs/wiki && git commit -F C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/b8f89ec2-7b8e-4b7a-8c72-d868c2b9a843/scratchpad/commit-umzug-5.txt
```

Erste Zeile: `Say what loomux does on the pages that described the old tools`.

---

### Task 6: Kataloge, Protokoll, Audit und Register neu anlegen

**Files:**
- Create: `docs/wiki/index.md`, `docs/wiki/{entities,sources,syntheses,topics}/index.md`,
  `docs/wiki/log.md`, `docs/wiki/audit.md`, `docs/wiki/_identities.tsv`

**Interfaces:**
- Consumes: die 24 Inhaltsseiten aus Task 2 bis 5.
- Produces: den Bestand, den `loomux brain catalog --scope project/loomux`
  liest.

Die Kataloge werden einmal von Hand geschrieben — in genau der Form, die
`render_catalog` erzeugt. `reindex` schreibt sie auch später nicht neu: die
Referenz überspringt die Kataloge des eigenen Bündels
(`loomux-src/ub/src/brain/cli.py:203-206`, „A bundle owns its catalog"). Wer
sie ab Stufe 3 pflegt, ist eine offene Entscheidung des Nutzers. Diese
Form steht in `loomux-src/ub/src/brain/catalog.py`, Funktion `render_catalog` —
vor dem Schreiben **ganz lesen**, nicht nach der Beschreibung hier arbeiten;
sie nennt auch die Klammer- und Leerzeichenregeln für Linkziele.

- [ ] **Step 1: Die Titel und Beschreibungen einsammeln**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && for f in $(find docs/wiki -name "*.md" ! -name "index.md" ! -name "_schema.md" | sort); do printf '%s|%s|%s\n' "$f" "$(grep -m1 '^title:' "$f" | cut -d' ' -f2-)" "$(grep -m1 '^description:' "$f" | cut -d' ' -f2-)"; done
```

- [ ] **Step 2: Die fünf Kataloge schreiben**

Wurzel (`docs/wiki/index.md`): Überschrift `# docs/wiki`, das Blockzitat
`> Jeder neue Bereich bekommt sofort eine Zeile in diesem Katalog.`, dann die
vier Verzeichniszeilen `* [entities](entities/)` bis `* [topics](topics/)`.
Je Unterverzeichnis ein Katalog mit dessen Seiten, eine Zeile je Seite aus
Step 1.

- [ ] **Step 3: Protokoll und Audit anlegen**

`docs/wiki/log.md`:

```markdown
# Protokoll

> Jede Änderung an einer Seite endet hier mit einer Zeile: Datum, Seite,
> was sich geändert hat und aus welcher Quelle.

- 2026-09-16 — Bündel aus `ultra-brain` übernommen: 24 Inhaltsseiten und das
  Regelwerk; Kataloge, Protokoll, Audit und Register neu angelegt.
```

`docs/wiki/audit.md`:

```markdown
# Wartungsprotokoll

> Hier stehen die Wartungsvorgänge über dem Bündel. Gefüllt wird es ab
> Stufe 3; bis dahin bleibt es leer.
```

- [ ] **Step 4: Das Register anlegen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && printf 'doc_id\trelative\tcontent_hash\trevision\n' > docs/wiki/_identities.tsv
```

Nur die Kopfzeile, und diese Datei füllt `reindex` nicht: das Register des
Bereichs schreibt `reindex` ab Stufe 3 in dessen Artefaktverzeichnis, bei
`project/loomux` die Repo-Wurzel (`docs/wiki/_schema.md`,
`loomux-src/ub/src/brain/cli.py:142,164`). Die Datei hier ist nur der leere
Rahmen. Mit `\t`-Trennern und `\n`-Zeilenende schreiben, nicht mit CRLF.

- [ ] **Step 5: Das Bündel prüfen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && go run ./cmd/loomux wiki-gate; echo "exit=$?"
```

Erwartet: keine `wiki-lint:`-Zeile. Eine `wiki-drift`-Meldung ist in Ordnung
und gehört in den Bericht.

- [ ] **Step 6: Den Bereich lesen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && go run ./cmd/loomux brain catalog --scope project/loomux; echo "exit=$?"
```

Erwartet: **Exit 1 mit einem Lesefehler** auf `index.md` im Repowurzelverzeichnis.
Der Grund ist gemessen, nicht vermutet: `catalog --scope` liest
`AreaArtifactDir(area, stateDir)/index.md`, und das ist für einen schreibbaren
Bereich `area.Path` — also die Wurzel von `loomux`, nicht `docs/wiki`
(`internal/brain/catalog/area.go:18-38`). Weder loomux noch das alte
ultra-brain-Repo hat dort je eine `index.md` gehabt; der Befehl schlug vor dem
Umzug genauso fehl. Die Meldung wörtlich in den Bericht.

Ob die Wurzel eine `index.md` bekommen soll, entscheidet der Nutzer: Es ist ein
Artefakt, das `reindex` ab Stufe 3 selbst schreibt, und die Freigabe vom
2026-09-16 sagt, Artefakte ziehen nicht mit. Dieser Plan legt darum keine an.

- [ ] **Step 6b: Lesen und Nachbarn prüfen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && go run ./cmd/loomux brain read docs/wiki/entities/qmd.md --scope project/loomux | head -5; echo "exit=$?"
```

Erwartet: Exit 0 und die ersten Zeilen der Seite — der Weg über `read` und
`neighbors` braucht kein Artefakt.

- [ ] **Step 7: Committen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && git add docs/wiki && git commit -F C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/b8f89ec2-7b8e-4b7a-8c72-d868c2b9a843/scratchpad/commit-umzug-6.txt
```

Erste Zeile: `Give the moved bundle its catalogs and its log`.

---

### Task 7: Nutzerdoku einarbeiten und die Liste schließen

**Files:**
- Modify: `docs/de/getting-started.md`, `docs/en/getting-started.md`,
  `docs/de/configuration.md`, `docs/en/configuration.md`,
  `docs/de/hooks.md`, `docs/en/hooks.md`, `README.md`, `README.de.md`
- Modify: `docs/.superpowers/parity/umzug-wiki-doku.md`,
  `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`

**Interfaces:**
- Consumes: die zehn Doku-Seiten der alten Repos (nur lesend).
- Produces: nichts, was eine spätere Aufgabe braucht; das ist die Schlussaufgabe.

- [ ] **Step 1: Die vier Abschnitte übernehmen**

Aus `ub/docs/installation.md` die Abschnitte „Wo was liegt", „Einen Bereich
einrichten" und „Was dabei schiefging" — auf loomux umgeschrieben: kein `uv`,
kein `brain init`, sondern `go build -o bin/loomux.exe ./cmd/loomux`, die
Registry unter `%LOCALAPPDATA%\loomux\` und das Manifest `.loomux/config.toml`.
Ziel: `getting-started` (Einrichtung) und `configuration` (Ablage), beide
Sprachen.

- [ ] **Step 2: Die zwei Hook-Abschnitte übernehmen**

Aus `ub/docs/hooks.md` „CRLF ist keine Formatierungsfrage" und „Die
Schreibschranke und das Memory der Agenten"; aus `ul/docs/hooks.md` die Tabelle
der Sprachstacks und Werkzeugketten. Ziel: `hooks` und `configuration`, beide
Sprachen. Was loomux' Seiten schon sagen, wird nicht wiederholt.

- [ ] **Step 3: Den Befehlsnamen richtigstellen**

`README.md` und `README.de.md` nennen `loomux wiki gate`; der Befehl heißt
`loomux wiki-gate` (`internal/cli/commands.go:13`). Beide Stellen richten.

- [ ] **Step 4: Die Spec berichtigen**

In `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`, Abschnitt
„Datenumzug", Punkt 1: die Klammer „(10 und 1)" hinter `_identities.tsv`
streichen. Beide Register trugen nur die Kopfzeile. Die `doc_id`s in der
Frontmatter gehören den zitierten Quellen (`sources[]`), nicht den Seiten;
eigene `doc_id`s der Seiten prägt erst `reindex` in Stufe 3. Ein Satz dazu,
warum die Zahl fiel.

- [ ] **Step 5: Die Prüfliste schließen**

In `docs/.superpowers/parity/umzug-wiki-doku.md` unter **Stand** eine Zeile
ergänzen: umgesetzt am `<Datum>`, mit den Zahlen, die am Ende tatsächlich
stehen (Seiten im Bündel, Dateien unter `plans-ub`, `specs-ub`, `specs-ul`,
`bench-ub`).

- [ ] **Step 6: Alles prüfen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && gofmt -l cmd internal; go vet ./... && go test ./... -count=1 2>&1 | grep -v "^ok\|no test files"; go run ./cmd/loomux wiki-gate; echo "exit=$?"
```

Erwartet: keine Ausgabe von `gofmt`, `go vet` und `go test`; `wiki-gate` ohne
`wiki-lint:`-Zeile.

- [ ] **Step 7: Committen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux-umzug" && git add docs README.md README.de.md && git commit -F C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/b8f89ec2-7b8e-4b7a-8c72-d868c2b9a843/scratchpad/commit-umzug-7.txt
```

Erste Zeile: `Fold the old documentation into the loomux pages`.

---

## Was dieser Plan nicht tut

- **Kein `reindex`, kein `reconcile`.** Beide entstehen in Stufe 3. Bis dahin
  findet `brain search` die neuen Seiten nicht, und `_identities.tsv` bleibt
  leer. Wer die Seiten suchbar braucht, indiziert vorerst mit dem alten
  Werkzeug; dafür muss der Bereich auch in `%LOCALAPPDATA%\brain\registry.toml`
  stehen.
- **Keine Änderung an `.loomux/config.toml` und an der Registry.** Beide
  nennen `docs/wiki` bereits; entsteht das Verzeichnis, lintet der
  post-edit-Hook die bearbeitete Seite von selbst. `loomux wiki-gate` greift
  dagegen nicht von selbst: kein Hook und kein Tor ruft es, es prüft das Bündel
  nur, wenn man es aufruft.
- **Keine Änderung in den alten Repos.** Sie bleiben, wie sie sind; das Archiv
  ist der Beleg.
- **Keine Wiki-Schreibbefehle.** `wiki types`, `retype`, `census` und
  `scaffold` gehören zu Stufe 3.

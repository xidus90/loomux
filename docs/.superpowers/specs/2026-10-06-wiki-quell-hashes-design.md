# Wiki-Quell-Hashes im Tor — Design

Stand: 2026-10-06. Zweig `feat/wiki-source-hash-check`.

## Ziel

Ein Pull Request, der eine zitierte Quelle ändert, soll nicht mehr grün auf
`master` landen, solange Register und Pins hinter ihr liegen. Anlass: Nach
einem reinen Doku-PR standen auf `master` 22 veraltete `content_hash`-Pins in
`docs/wiki`, und weder `lint` noch `brain check` meldeten sie. Ein Skript im
Scratchpad hat sie dann blind nachgezogen.

## Ausgangslage (gemessen)

- Eine Quelle veraltet an zwei Stellen:
  1. **Datei gegen Register:** Die Datei hat einen anderen Hash als ihre Zeile
     in `_identities.tsv`. Das passiert bei jeder Änderung ohne `reindex`.
  2. **Pin gegen Register:** Ein `sources:`-Eintrag einer Seite nennt einen
     anderen `content_hash` oder eine andere `revision` als die Zeile.
- Eine Prüfung nur von Stelle 2 hätte den Anlass nicht gefunden: Ohne
  `reindex` stimmen Pin und Register überein, beide alt.
- Auf `origin/master` (757d600b) sind beide Stellen sauber: 32 Registerzeilen
  passen zu ihren Dateien, 26 Pins zu ihren Zeilen (Skripte `reg-vs-file.py`
  und `t7-sources.py`, Python 3.11, im Scratchpad der Sitzung). Das ist der
  bekannte grüne Ausgangspunkt der neuen Spur.
- Zitiert sind heute 8 Dateien: sechs unter `docs/de/` und zwei Wiki-Seiten
  (`topics/architektur-grundsaetze.md`, `topics/suche-und-profile.md`), die die
  Synthesen zitieren.
- Das Tor hat schon eine Wiki-Spur: `hooks.WikiGateJobs` hängt `lint/wiki`
  (`wiki.LintBundle`, im Prozess, ohne Registry) an den Plan von
  `loomux check` an. `check precommit --show` listet sie nicht, `check lint -v`
  zeigt sie.
- `reindex`, `approve` und `reconcile` arbeiten über die Registry. Sie führt
  `project/loomux` mit dem Pfad des Haupt-Checkouts; aus einem Zweig-Worktree
  schreiben diese Befehle also nicht in den Zweig. Im CI gibt es keine
  Registry.
- `apply.AdvanceSources` schreibt das ganze Frontmatter neu. Auf allen 15
  Seiten mit Pins ändert das die Form: `description` wird bei 80 Zeichen
  umbrochen, die Liste anders eingerückt, die Anführungszeichen um den Hash
  fallen weg (Probe `internal/brain/apply/zzprobe`, nicht committet).
- `identity.RenderIdentities(identity.ReadIdentities(...))` gibt
  `_identities.tsv` byte-gleich zurück (Probe `internal/brain/identity/zzprobe`,
  nicht committet).

## Entscheidungen

1. **Das Tor verweigert beide Stellen.** Ein PR, der eine zitierte Quelle
   ändert, zieht Register und Pins im selben PR nach.
2. **Nur zitierte Dateien.** Stelle 1 prüft nur Registerzeilen von Dateien,
   die ein Pin des eigenen Scopes nennt. Eine nicht zitierte Zeile darf alt
   sein; das nächste `reindex` zieht sie nach, und `reconcile` eröffnet für eine
   Datei ohne abhängige Seite keinen Fall.
3. **Kein automatisches `reindex`** im Hook oder im CI (Gründe unter
   „Verworfen“).
4. **Neuer Befehl `loomux wiki repin`** zieht Register und Pins im Baum nach.
   Wer ihn fährt, erklärt, dass die zitierenden Seiten noch stimmen.
5. **Nur ein Mensch fährt `repin`.** Der Wächter verweigert ihn Agenten, wie
   `config set` ohne `--propose`. Ein Agent prüft die Seiten, nennt den Befehl
   und wartet.
6. **Eigene Funktion, eigene Spur (Ansatz A).** Die Prüfung ist eine reine
   Funktion im Paket `wiki` und läuft als eigener Job neben `lint/wiki`. Das
   Bundle-Lint bleibt unberührt; es ist über `wiki-gate` durch die
   aufgenommenen Fälle in `testdata/cases/1a` an die Referenz gebunden.
7. **An, mit Schonfrist.** Die Spur hat einen eigenen Schlüssel und bekommt so
   die Schonfrist der Armed-Datei. Sie ist einzeln abschaltbar.
8. **Ein Zitierzyklus ist ein eigener Befund.** Zwei Seiten, die einander
   pinnen, können nie beide stimmen; die Prüfung sagt das selbst, statt auf
   einen Befehl zu verweisen, der ablehnt.

## Die Prüfung

### Ort und Spur

- Funktion `wiki.SourceDrift(projectRoot, wikiRoot, scope string)` in
  `internal/brain/wiki`, Rückgabe: die Befunde (Regel, Seite, Text), sortiert
  nach Seite und Regel.
- Eine eigene Funktion `hooks.WikiSourceJobs` liefert den Job: Name
  `lint/wiki-sources`, `Kind: "lint"`, `Stack: "wiki-sources"`, `Area: "."`,
  `Origin: "in-process"`. Bedingung wie bei `lint/wiki` (Art `lint` gewählt,
  ein Wiki vorhanden), dazu die beiden Schalter unten. Sie wird in
  `loomux check` (`internal/cli/check.go`) und in der Spurliste von
  `loomux gate` (`hooks.GateLanes`) angehängt, **nicht im Stop-Gate**
  (Nachtrag unten).
- Der Spurschlüssel ist damit `lint/wiki-sources@.`, verschieden von
  `lint/wiki@.` (`verify.LaneKey`). In einem Projekt mit Armed-Datei warnt die
  Spur bis zum ersten Grün; ohne Armed-Datei (wie loomux) ist sie sofort scharf.
- `wiki-sources` kommt in `verify.StackNames()` (`internal/verify/schema.go`),
  damit `[verify.wiki-sources] lint = false` die Spur abschaltet.
- `[modules] brain = false` schaltet sie ebenfalls ab: `ParseConfig` setzt
  neben `Stacks["wiki"]["lint"]` auch `Stacks["wiki-sources"]["lint"]` auf
  `laneOff()`.
- `hooks/status.go` bleibt, wie es ist: Es beschreibt die Spuren des
  post-edit-Hooks, und die neue Spur läuft nur im Tor.

### Eingaben

Nur Dateien im Baum, keine Registry:

- das Register `<projectRoot>/_identities.tsv` über
  `identity.ReadIdentities`. Das ist der Ort, den `config.ManifestDir` für einen
  beschreibbaren Bereich nennt (`area.Path`). Ein `_identities.tsv` im Bundle
  ist Gerüst von `wiki init` und wird nicht gelesen.
- die Seiten unter `wikiRoot` über `wiki.ReadPage`, Gerüstdateien ausgenommen
  (dieselbe Liste wie in `parse.go`)
- die zitierten Dateien selbst, gehasht wie `identity.ContentHash` (CRLF wird
  zu LF)
- der Scope aus `[area] scope` des Manifests

### Was übersprungen wird (still grün)

- ein Projekt ohne `[area]` oder ohne `<projectRoot>/_identities.tsv`
  (read-only-Bereiche führen ihr Register im Zustandsverzeichnis)
- ein Pin, dessen `resource` nicht mit `brain://<eigener scope>/` beginnt
- ein Eintrag ohne `doc_id`, `content_hash` oder `revision`: Den meldet
  `house/source-incomplete`. Ein Mangel gehört auf eine Achse.
- eine Seite, deren Frontmatter nicht lädt: Die meldet die `okf`-Achse.

### Befunde

Je Pin des eigenen Scopes, in dieser Reihenfolge; ein Pin bekommt höchstens
einen der Befunde 1 bis 3, und dann keinen weiteren:

1. `row-missing`: Der Pfad steht nicht im Register.
2. `source-missing`: Die Datei fehlt im Baum.
3. `doc-id`: `doc_id` des Pins weicht von der Zeile ab.
4. `register-stale` (Stelle 1): Der Hash der Datei weicht von ihrer Zeile ab.
   Einmal je Datei, nicht je Pin; der Text nennt die zitierenden Seiten.
5. `pin-stale` (Stelle 2): `content_hash` oder `revision` des Pins weicht von
   der Zeile ab.
6. `cycle`: Seiten, die einander über Pins des eigenen Scopes zitieren
   (eine starke Zusammenhangskomponente mit mehr als einer Seite, oder eine
   Seite, die sich selbst zitiert). Einmal je Zyklus; der Text nennt die
   Seiten.

### Meldung

Wie `lintBundle` es für `lint/wiki` schreibt: je Befund eine Zeile
`  • [wiki-sources:<regel>] <seite>: <text>`, und als Fehler
`the wiki sources have <n> finding(s)`. Am Ende eine Zeile mit dem Weg:

- bei Befunden 4 und 5: `check the citing pages; then a human runs loomux wiki repin`
- bei Befunden 1 bis 3 und 6: dass `repin` sie nicht behebt und was zu tun ist
  (1: siehe „Grenzen“; 2 und 3: Pin von Hand korrigieren; 6: den Zyklus
  auflösen).

## `loomux wiki repin [--root <pfad>]`

### Ablauf

1. Er liest dieselben Eingaben wie die Prüfung und fährt `SourceDrift`.
2. Gibt es einen Befund 1, 2, 3 oder 6, bricht er mit Exit 1 ab, nennt die
   Befunde und schreibt nichts.
3. Er rechnet im Speicher, bis nichts mehr kippt:
   - Für jede zitierte Datei steht ihr Endhash fest: bei einer Datei außerhalb
     des Wikis ihr Hash auf der Platte, bei einer Wiki-Seite der Hash der Bytes,
     die er für sie schreiben wird.
   - Jeder Pin des eigenen Scopes bekommt `content_hash` = Endhash und
     `revision` = Endrevision seiner Datei.
   - Die Endrevision ist die Revision der Zeile im Register, wie er es gelesen
     hat, plus 1, wenn der Endhash vom Hash dieser Zeile abweicht; sonst bleibt
     sie. Eine Zeile rückt in einem Lauf also höchstens um 1 vor, egal wie oft
     sich eine Seite in der Rechnung ändert.
   - Da Zyklen vorher abgelehnt sind, endet die Rechnung; er verarbeitet die
     Seiten in umgekehrter topologischer Ordnung der Zitate (zitierte Seiten
     zuerst).
4. Er schreibt die geänderten Seiten und, wenn sich eine Zeile geändert hat,
   das Register (`identity.RenderIdentities`). Nicht zitierte Zeilen fasst er
   nicht an.

Die Hashfunktion bekommt eine Variante für Bytes (`identity.ContentHash`
liest heute eine Datei); beide müssen dasselbe Ergebnis geben.

### Zeilen-Edit statt YAML-Neuschreib

`repin` ändert nur den Wert der Zeilen `content_hash:` und `revision:` des
Eintrags im `sources:`-Block, dessen `doc_id` passt. Einrückung, Anführungs­
zeichen (doppelt, einfach, keine), Reihenfolge, die übrigen Zeilen und das
Zeilenende jeder Zeile bleiben. Er lehnt eine Seite ab (Exit 1, nichts
geschrieben), wenn er den Eintrag so nicht sicher findet:

- `sources:` im Flow-Stil (`sources: [...]`)
- ein Eintrag ohne eigene Zeile für `content_hash` oder `revision`
- eine `doc_id`, die in derselben Seite zweimal vorkommt
- ein mehrzeiliger Wert in einer der beiden Zeilen

### Alles oder nichts

Jede Ablehnung fällt, bevor die erste Datei geschrieben ist. Ein Fehler beim
Schreiben selbst (Platte voll, Rechte) bricht ab und nennt, was schon
geschrieben war; ein zweiter Lauf setzt dort fort, weil er nur Abweichungen
nachzieht.

### Ausgabe und Exit-Codes

- `stdout`: erst je geänderter Seite `repinned: <seite>`, dann je
  vorgerückter Registerzeile `register: <pfad> revision <alt> -> <neu>`,
  sonst `nothing to repin`.
- Exit `0` nach einem Lauf, der fertig wurde, auch ohne Änderung; `1` bei einer
  Ablehnung oder einem Fehler (`loomux wiki repin: <grund>` auf `stderr`); `2`
  bei falscher Benutzung.
- Ohne `[area]` oder ohne Register im Baum: `nothing to repin`, Exit 0.

### Wächter

- `hooks.programWrites` bekommt einen Fall `wiki`: `repin` schreibt, außer bei
  einem einzelnen `--help`/`-h`. `wiki init`, `types` und `retype` bleiben frei.
- Der Ablehnungstext in `guard.go` nennt `wiki repin` mit: Er erklärt die
  zitierenden Seiten für richtig, und das entscheidet ein Mensch.

## Grenzen

- **Read-only-Bereiche, fremde Scopes, Projekte ohne Register** werden nicht
  geprüft (siehe oben). Die Doku nennt das.
- **Eine neu zitierte, noch nicht registrierte Datei** ergibt Befund 1, und
  `repin` behebt ihn nicht: Eine Zeile mit neuer `doc_id` legt heute nur
  `reindex` an, und das schreibt in den Checkout der Registry. Weg heute: die
  Datei zuerst ohne Zitat mergen, im Haupt-Checkout `loomux reindex` fahren,
  die neue Zeile in einem Zweig übernehmen und dann zitieren. Kandidat für
  später: `repin` legt die Zeile selbst an (`identity.NewDocID`). Nicht in
  diesem PR.
- **Nach einem Merge-Konflikt** in `_identities.tsv` oder in Pins: `repin`
  erneut fahren; die Prüfung meldet, was dann noch abweicht.

## Verworfen

- **Automatisches `reindex` im pre-commit-Hook oder im CI.** `reindex` läuft
  über alle registrierten Bereiche (`cli-reference.md`, Abschnitt
  `loomux reindex`), schreibt aus einem Worktree in den Haupt-Checkout, und die
  Änderung käme nicht in den Commit (pre-commit verweigert Tor-Eingaben, die vom
  Index abweichen). Im CI gibt es keine Registry, und zurückpushen darf nur ein
  Mensch. Vor allem eröffnet `reindex` erst die Fälle und rückt dann das
  Register vor; automatisch hieße das, jede Änderung still für irrelevant zu
  erklären.
- **Nur Stelle 2 prüfen.** Hätte den Anlass nicht gefunden (siehe oben).
- **Alle Registerzeilen prüfen.** Jede Wiki- oder Spec-Änderung bräuchte
  `repin`, obwohl an 24 der 32 Zeilen kein Pin hängt.
- **Neue Regel in `LintBundle` (Ansatz B).** Neue Signatur und neues Verhalten
  des referenzgebundenen Lints.
- **`loomux check wiki-sources --fix` (Ansatz C).** Neue Preset-Spur, eigener
  Prozess, `--fix` an einer Prüfung wäre ein neues Muster.
- **`repin` über `apply.AdvanceSources`.** Formatiert das Frontmatter jeder
  Seite um (Probe oben).
- **`repin` auch für Agenten.** Der Lauf ist ein `approve --reject` für jeden
  offenen Fall auf einmal; das entscheidet ein Mensch.

## Tests

TDD, 100 % je Funktion, Mutationsrunde je neuer Funktion (jede Teilbedingung
einzeln streichen, jede Fehlerprüfung auf „jeder Fehler“ weiten). Testwelten:

- eine Kette aus drei Gliedern: Datei unter `docs/de` → Topic → Synthese;
  `repin` rückt alle drei Zeilen um genau 1 vor
- ein Zyklus aus zwei Seiten und eine Seite, die sich selbst zitiert
- eine zitierte Datei mit CRLF, deren Zeile mit LF-Hash stimmt (kein Befund)
- ein Pin auf einen fremden Scope (kein Befund, nichts geschrieben)
- ein Projekt ohne Register und eines ohne `[area]`
- „Register aktuell, nur Pin alt“ (der Fall nach `reindex` im Haupt-Checkout)
- eine geänderte nicht zitierte Datei (kein Befund, Zeile unverändert)
- Alles oder nichts: Die zweite Seite ist nicht sicher editierbar, die erste
  bleibt ungeschrieben.
- Zeilen-Edit: doppelte, einfache und keine Anführungszeichen bleiben; CRLF
  einer Seite bleibt; alle Ablehnungsformen
- die Spur: eigener Schlüssel, Schonfrist mit Armed-Datei, scharf ohne,
  `[verify.wiki-sources] lint = false`, `[modules] brain = false`
- der Wächter: `wiki repin` verweigert, `wiki repin --help`, `wiki retype` und
  `wiki types` frei

Testwerte benachbarter Felder sind paarweise verschieden (Revisionen, Hashes),
und das erwartete Element liegt nie dort, wo die Lesereihenfolge es ohnehin
hinlegt.

## Doku

- `docs/en/cli-reference.md` und `docs/de/cli-reference.md`: `wiki repin`, die
  Spur `lint/wiki-sources` bei `loomux check`
- `docs/en/configuration.md` und `docs/de/configuration.md`:
  `[verify.wiki-sources]`, der Brain-Schalter, die Grenzen
- Wiki: `topics/brain-maintenance.md` und `topics/wiki-schicht.md`
- Die Roadmap ändert sich nicht.

## Dieser PR prüft sich selbst

Der PR ändert `docs/de/cli-reference.md` (10 Pins) und
`docs/de/configuration.md` (3 Pins). loomux hat keine Armed-Datei, also ist die
neue Spur ab ihrem ersten Commit scharf. Reihenfolge im Plan:

1. Prüfung und `repin` bauen und committen, solange die Doku unverändert ist
   (die Spur ist dann grün).
2. Die Doku ändern.
3. Ein Mensch fährt `loomux wiki repin` (Entscheidung 5).
4. Doku, Register und Frontmatter zusammen stagen und committen.

Das ist der erste echte Lauf, Ende zu Ende. Der PR-Diff enthält deshalb
`_identities.tsv` und das Frontmatter von rund einem Dutzend Seiten.

## Nachtrag 2026-10-06: nicht im Stop-Gate

Beim Planen gefunden: `hooks.WikiGateJobs` läuft auch im Stop-Gate an jedem
Turn-Ende (`hooks/stop.go`). Eine Spur dort, die nur ein Mensch grün machen
kann (Entscheidung 5), hielte jedes Turn-Ende eines Agenten nach einer
Doku-Änderung an, bis `MaxBlocks` (3) aufgibt. Entscheidung des Nutzers: Die
Spur läuft nur in `loomux check` (pre-commit-Tor und CI) und steht in
`hooks.GateLanes`, damit `loomux gate` sie kennt und scharf schalten kann. Ein
Agent bemerkt veraltete Pins spätestens beim Commit und nennt dann den Befehl.

## Nachtrag 2026-10-07: was `repin` anfasst

Aus dem Abschlussreview: `repin` gab jeden Pin einer Seite an den
Zeilen-Edit, auch aktuelle. Eine Seite, die die Prüfung nie nennt, deren Form
der Zeilen-Edit aber ablehnt (Flow-Liste, Kommentar), ließ so jeden Lauf
scheitern, während die Spur „fahre `repin`“ sagte. Entscheidung:

- `repin` editiert nur Pins, deren Wert sich ändert; eine Seite ohne solche
  Pins bleibt unberührt und wird nicht geprüft.
- Eine `doc_id`, die in einer Seite zweimal vorkommt, ist keine Ablehnung
  mehr: Beide Einträge zitieren dieselbe Datei und bekommen dieselben Werte.
  Die Ablehnungsliste im Abschnitt „Zeilen-Edit statt YAML-Neuschreib“ gilt
  ohne diesen Punkt.

## PR

`release:minor` — neue Prüfung im Tor und neuer Befehl, mit Schonfrist.
Changelog unter `Added`.

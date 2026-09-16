# Produkt-Design — Entscheidungen der Selbsteinrichtung

**Stand:** 2026-08-23
**Bezug:** Architektur-Design §5.1, §5.5, §7.2, §13, §17 (Zeile 2c, 3); Entscheidungen 36, 38, 39
**Anlass:** ultra-brain wurde als Bereich in sein eigenes Repo eingebaut, um es
im Alltag zu benutzen statt nur zu testen. Die Anleitung dazu ist
[docs/installation.md](installation.md); diese Seite hält fest, **warum** die
Einrichtung so aussieht.

---

## 1. Warum diese Seite existiert

Sie ist selbst das Ergebnis eines Tests. Der Auftrag lautete, sie über den
MCP-Server im Wiki abzulegen. Beides ging nicht, und zwar aus einem Grund, der
im Plan steht:

- Der MCP-Front bietet die **fünf schreibfreien Werkzeuge** — `search`,
  `catalog`, `read`, `neighbors`, `status`. Ein Schreibpfad ist Scheibe 3.
- Ein Wiki gibt es noch nicht. Die `CLAUDE.md` des Vaults sagt es wörtlich:
  die Verdichtungsschicht unter `90 Wiki/` existiert derzeit nicht.

Damit ist der Befund des Tests präziser als jede Zusicherung: **das System
liest sein Wissen bereits gut und kann noch nichts davon festhalten.** Diese
Seite ist von Hand entstanden und wird über den MCP nur wiedergefunden. Sobald
Scheibe 3 steht, ist sie der erste Kandidat für einen echten Schreibvorgang.

---

## 2. Die Entscheidungen

### 2.1 ultra-brain ist ein eigener Bereich, kein Anhängsel

Das Repo hätte unter einen bestehenden Bereich gehängt werden können. Es wurde
ein eigener, weil ein Bereich die Einheit ist, an der Datenschutzmodus,
Ausschlussliste und Suchmuster hängen. Ein Code-Repo braucht andere Ausschlüsse
als ein Vault, und ein geteilter Bereich hätte die schwächere von beiden Listen
durchgesetzt.

### 2.2 Beschreibbar, nicht lesend — die Artefakte liegen im Baum

Die drei bestehenden Fremdbestände (`space`, `iam-wiki`, `obsidian-ai`) sind
`readonly`: ihre Kataloge liegen im Zustandsverzeichnis, ihr Baum bleibt
unberührt. Für das eigene Repo fiel die Entscheidung andersherum. `index.md` je
Verzeichnis, `graph.json` und `_identities.tsv` stehen jetzt neben den Quellen.

Der Grund ist Sichtbarkeit: was der Indexer erzeugt, soll man im selben
Verzeichnisbaum sehen, in dem man arbeitet, statt es unter
`%LOCALAPPDATA%` suchen zu müssen. Der Preis sind erzeugte Dateien im
Git-Baum, und den zahlt `.gitignore` — nicht die Versionsverwaltung.

### 2.3 Werkzeug-Caches gehören ins Manifest, nicht in die Standardliste

`.pytest_cache/README.md` stand nach dem ersten Lauf als eigener Bereich im
Wurzelkatalog. Die eingebaute Ausschlussliste kennt `.git`, `.obsidian`,
`node_modules` und einige mehr, aber weder `.venv` noch die Caches von pytest,
ruff und mypy.

Naheliegend wäre gewesen, sie in die eingebaute Liste aufzunehmen. Dagegen
spricht, dass diese Liste für **jeden** Bereich gilt, auch für einen Vault, in
dem ein Verzeichnis dieses Namens etwas völlig anderes sein kann. Ein
Code-Repo deklariert seine Caches selbst; die eingebaute Liste bleibt die
Sammlung dessen, was überall und immer Ballast ist. Derselbe Ausschluss wurde
im Bereich `space` nachgezogen, wo er als stehende Divergenzmeldung sichtbar
war — die Meldung hat den Fehler gefunden, nicht ein Test.

### 2.4 Die Specs bleiben unauffindbar, und das wird deklariert

`docs/.superpowers/specs/` und `plans/` sind das dichteste Material im Repo,
und die Suchmaschine wird sie nie finden: qmd betritt Punktverzeichnisse
grundsätzlich nicht. Zwei Auswege standen offen — die Verzeichnisse umbenennen,
oder die Lücke erklären.

Entschieden wurde die Erklärung: `unsearched = ["docs/.superpowers/**"]`. Über
`catalog` und `read` sind die Dokumente erreichbar, über `search` nicht, und
`status` schweigt darüber, weil es eine Entscheidung ist und kein Versehen. Ein
Umbenennen hätte die Werkzeugkette getroffen, die dort schreibt, und den Gewinn
an Auffindbarkeit mit einem Bruch an anderer Stelle bezahlt.

### 2.5 Der Fremdsammlungs-Schutz wird repariert, nicht gelockert

`brain-mcp reindex` war vor dieser Einrichtung defekt: Der Merkzettel
`qmd-collections.json` kannte drei Sammlungen nicht, die brain selbst angelegt
hatte, und die Regel „fremde Sammlungen fasst du nicht an" brach jeden Lauf ab.

Die Regel bleibt, wie sie ist. Sie steht dort, weil ihr Fehlen einmal gemessen
wurde: eine gleichnamige fremde Sammlung zeigte plötzlich auf unser
Verzeichnis, aus 161 Dateien wurden 0. Repariert wurde der Merkzettel, nachdem
die Einträge in `~/.config/qmd/index.yml` als unsere erkennbar waren — ihre
Ignore-Liste ist wörtlich unsere. Ein Schutz, der einmal falsch anschlägt, wird
nicht abgeschaltet; er wird mit der Wirklichkeit versöhnt.

### 2.6 Die MCP-Front wird über `uv run --directory` angesprochen

`.mcp.json` ruft nicht `.venv/Scripts/brain.exe` auf, sondern
`uv run --directory <repo> brain-mcp mcp`. Der direkte Pfad wäre einen Tick
schneller und bräche, sobald die Umgebung neu gebaut wird oder jemand das Repo
woanders auscheckt. Der Umweg über `uv` löst die Umgebung bei jedem Start neu
auf.

Ebenso bewusst: die Front bekommt **keinen** Kanal-Schalter. Sie wählt die
Cloud-Leitung, und dass sie sie wählt, ist ihre Adresse — keine Einstellung,
die jemand versehentlich umstellt. Wer die Leitung dicht haben will, startet
den Daemon mit `--no-cloud`; dann existiert sie nicht, statt gefiltert zu
werden.

### 2.7 Diese Seite liegt im Repo, nicht im Vault

Der Vault (`brain-knowledge`) indexiert heute nur `10 Rohquellen/**/*.md`. Eine
Design-Entscheidung dort abzulegen hieße, sie als Rohquelle auszugeben, oder
das Suchmuster des Vaults zu erweitern und damit den Ablageort von Scheibe 3
vorwegzunehmen, bevor die Schicht existiert, die ihn tragen soll.

Also liegt sie hier — neben der Spec, über die sie spricht, in einem Bereich,
der sie vollständig indexiert. Der Vault beschreibt Wissen über die Welt;
dieses Repo beschreibt das Produkt.

---

## 3. Was offen bleibt

- **Der Schreibpfad.** Solange Scheibe 3 fehlt, entsteht jede Seite dieser Art
  von Hand. Der Test hat genau das gezeigt und ist damit nicht fehlgeschlagen,
  sondern beantwortet.
- **Wo Produktwissen am Ende lebt.** Die Trennung aus 2.7 ist eine Antwort für
  heute. Sobald das Wiki steht, ist neu zu entscheiden, ob es eine zweite
  Verdichtungsschicht für das Produkt selbst gibt oder ob beides in eine
  gehört.
- **Das Austragen eines Bereichs.** `reindex` gleicht die Sammlungen der
  registrierten Bereiche an, entfernt aber keine. Wer einen Bereich abmeldet,
  räumt qmds Konfiguration von Hand auf. Ob das ein Befehl werden soll, ist
  nicht entschieden.

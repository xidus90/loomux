# Scheibe 5 — Brain Maintenance, `realization` und der Merge-Auslöser

**Stand:** 2026-08-27
**Bezug:** Architektur-Design §5.1, §5.2, §5.4, §5.6, §9.1, §10 vollständig, §11,
§14, §15, §16.12, §16.15, §17 (Zeile 5), Entscheidungen 26, 28, 29, 30
**Vorgänger:** Scheibe 3 (Wiki-Schicht) und Scheibe 4 (Import), beide abgenommen
**Nachfolger:** Scheibe 6 (lokales Modell) — belegt `local_only`; Scheibe 7 (Web-App)

---

## 1. Zweck

Bis hierher kann das System Wissen aufnehmen und verdichten, aber nicht
bemerken, dass es alt geworden ist. Beide Stände — die geänderte Quelle und die
daraus abgeleitete Wiki-Seite — sind auffindbar, und der veraltete sieht besser
aus: verdichtet, verlinkt, mit Quellenangabe (§10.1). Scheibe 5 schließt das,
indem sie die zweite Art von Aktualität überhaupt erst zu einem Zustand macht,
den Code feststellen kann.

Sie tut das ohne neue Vertrauensstellung. Der Code erkennt und bereitet auf, ein
Modell schlägt vor, ein deterministischer Prüfer verwirft Unbelegtes, und
geschrieben wird erst nach deiner Freigabe. Der Vorschlaggeber bekommt dabei
keinen eigenen Ausgang ins Netz: Er ist das Modell der laufenden Sitzung, geführt
von einem Skill, wie schon in Scheibe 3.

## 2. Umfang

**Gebaut wird:**

- `src/brain/maintenance/` — Abgleich, Fallbildung, Paketbau, Evidenzprüfer,
  Freigabe samt Git-Schritt
- die Umbenennungserkennung im Indexlauf (`_index_area`, siehe §9)
- `realization` und `implemented_in` in `wiki/page.py` und im Typkatalog, dazu
  die zwei Lint-Prüfungen aus Entscheidung 29
- der post-merge-Hook und `brain hook install | status | remove`
- `[maintenance]` im Manifest (`on_merge`), `layout.review` im Vault-Manifest
- die CLI-Anbindungen aus dem Vertrag (§9.1): `brain reconcile`, `brain cases`,
  `brain case <id>`, `brain approve <id> [--amend|--reject|--defer]`
- die Skills `brain:review`, `brain:wiki-plan`, `brain:land`
- die Fixtures für Paket, Vorschlag und die zwei Evidenzfälle

**Nicht gebaut wird:**

- **Der Wächter aus §10.3.** Siehe die Abweichungsliste in §12.
- **Ein eigener Modellausgang.** §11 nennt drei Umsetzungen; diese Scheibe baut
  die Schnittstelle und den Sitzungsweg. `local_only` belegt Scheibe 6.
- **Das Prüfzentrum als Oberfläche.** Es ist ein Ordner und vier Befehle;
  Scheibe 7 setzt eine Ansicht darauf.

## 3. Der Ort: das Prüfzentrum

Ein Fall lebt als Verzeichnis im Vault:

```
95 Prüfzentrum/<bereich>/<fall-id>/
    case.toml        Fallzustand
    package.md       Analysepaket mit nummerierten Segmenten
    proposal.md      der Vorschlag, sobald einer geschrieben wurde
```

Das ist keine Entwurfsfreiheit, sondern §5.2: Das Prüfzentrum liegt im Vault
statt im Zustandsverzeichnis, damit offene Fälle in Obsidian lesbar sind, bevor
eine Oberfläche existiert. §5.1 zählt die Prüffälle ausdrücklich zum
versionierten Vault-Inhalt.

**`layout.review` wird gebaut und ist Ausschlussmuster.** Weder Indexer noch
Katalog noch Graph erfassen den Ordner, und auf dem Cloud-Kanal ist er nicht
lesbar. Ohne das gerieten Quellendiffs — auch die aus `local_only`-Bereichen —
über `search` und `read` nach außen und ständen als Treffer neben der echten
Wiki-Seite: genau der Fehler, gegen den §10.1 gebaut ist. Eine
Warteschlangensicht im Zustandsverzeichnis ist erlaubt, aber wegwerfbarer Cache;
Wahrheit sind die Dateien.

## 4. Erkennung

Die Erkennung dieser Scheibe ist der Abgleich, ausgelöst auf zwei Wegen:

- **`brain reconcile`** von Hand, und
- **beim Daemon-Start nachgeholt**, wenn der letzte vollständige Lauf länger als
  24 Stunden zurückliegt; danach alle 24 Stunden, solange der Prozess lebt.
  `status` nennt den Zeitpunkt (§10.3).

Zweistufig wie in §10.3: Zeitstempel und Größe als Vorfilter, Hash nur für
Verdächtige. Der Abgleich erzeugt keine Änderungen, sondern Fälle.

Der Handbefehl ist kein Komfort, sondern der deterministische Auslöser, den die
Abnahme braucht — „Daemon neu starten und hoffen, dass 24 Stunden vergangen
sind" ist kein Testaufbau. Denselben Codepfad bräuchten die Tests ohnehin.

## 5. Der Vorschlaggeber

Der Vorschlag kommt vom Modell der laufenden Sitzung, geführt vom Skill
`brain:review`. Kein API-Schlüssel im Kern, kein ausgehender Verkehr aus dem
Daemon, kein MCP-Prompt: `src/brain/mcp.py` ist Umleiter und bleibt es.

`brain case <id>` zeigt das Paket und ist zugleich das Bestätigungstor (§9.1).
`manual_cloud` und `automatic_cloud` unterscheiden sich allein an diesem Tor,
nicht am Transportweg: Bei `manual_cloud` wartet es auf dich, bei
`automatic_cloud` steht es für regelkonforme Pakete offen.

**`local_only` ist eine Code-Weiche vor dem Skill-Weg**, kein Vermerk im Prompt.
Der Skill arbeitet mit den Datei-Werkzeugen des Wirts und läuft damit außerhalb
des Kanalgatters aus `privacy.py`; ohne die Weiche hinge der Datenschutz an der
Aufmerksamkeit des Modells. Rückfall ist der manuelle Fall, bis Scheibe 6 das
lokale Modell nachliefert.

Der Vorschlag ist `proposal.md` neben dem Fall — kein Zustand im Wiki und kein
neues Schreibwerkzeug am Daemon. Die fünf Daemon-Werkzeuge bleiben lesend.

## 6. Formate

Drei Serialisierungen sind im Projekt etabliert — Frontmatter plus Markdown für
Lesbares, TOML für Deklarationen, TSV für das Register. Es kommt keine vierte
hinzu.

**Der tragende Satz: wörtliche Zitate stehen nie als YAML-, TOML- oder
JSON-Wert, sondern ausschließlich in Fences.** Escaping und Flow-Skalare
normalisieren Anführungszeichen und Zeilenumbrüche still; der
Wortgleich-Abgleich aus §10.6 scheiterte dann an der Ablage statt am Modell.

### 6.1 `case.toml`

```toml
id = "ultra-brain-2026-08-27-a4f2"
area = "project/ultra-brain"
target = "themen/suchkette.md"
target_hash = "sha256:…"       # Stand der Zielseite bei Fallbildung (§10.4)
state = "in_review"            # current | source_changed | due | source_missing | in_review
trigger = "source_change"      # source_change | merge
weight = "change"              # change | time
created = 2026-08-27T09:14:00Z

[[sources]]
doc_id = "…"
revision = 5
content_hash = "sha256:…"
```

`source_missing` gehört zum Vokabular (arch 10.4) und wird von `case.STATES`
angenommen, aber von dieser Scheibe **nirgends erzeugt**: eine verschwundene
Rohquelle beurteilt die Umbenennungserkennung des Indexlaufs, nicht der
Abgleich — `reconcile.py` begründet das an seiner Modul-Kopfzeile. Der Wert
ist zulässig, weil ein späterer Zustandsschreiber ihn braucht; wer ihn heute
von Hand in eine `case.toml` schreibt, bekommt keinen Fehler, aber auch keine
Wirkung.

Bezeichner sind englisch, Prosa deutsch. `tomllib` liest nur — der Fallschreiber
bringt einen kleinen eigenen Emitter für dieses flache Schema mit; eine
Abhängigkeit dafür lohnt nicht.

Das Format trägt die zwei Nebenläufigkeitsfälle aus §10.4 als Felder, nicht als
Prosa: den verworfenen Fall, dessen bereits geschriebener Vorschlag im Fall
vermerkt bleibt, und die Zielseite, die sich unter dem Fall bewegt hat.

### 6.2 `package.md`

Frontmatter, dann je Segment eine Überschrift `## D1 — …` und der Inhalt in
einem Fence. Die Nummern vergibt der Code beim Paketbau, typisiert: `D` für
Diff-Hunks, `W` für Absätze der Wiki-Seite, `Q` für `sources`-Einträge. Damit
ist „Existiert das Segment?" ein Überschriften-Lookup.

Segmentgrenzen sind deterministisch: Diff = ein Hunk, Wiki = ein durch
Leerzeilen getrennter Absatz, Quelle = ein `sources`-Eintrag. Die Nummern sind
nur innerhalb eines Paketlebens stabil — ändert sich die Quelle erneut, wird der
Fall ohnehin verworfen und neu gebildet (§10.4). Paketübergreifende Stabilität
wird ausdrücklich nicht zugesichert.

### 6.3 `proposal.md`

Frontmatter für Fall, Einordnung, Zuversicht und Restunsicherheit; darunter je
Behauptung ein Abschnitt mit `evidence: <Segmentnummer>` und dem Zitat im Fence,
dazu der vorgeschlagene Diff. Der Skill schreibt das mit gewöhnlichen
Datei-Werkzeugen, Obsidian rendert es, der Prüfer parst es über feste
Überschriften und Fence-Extraktion — das Muster dafür steht schon in
`wiki/page.py:18`.

### 6.4 `audit.md` und `log.md`

`audit.md` bekommt je Vorgang einen Block mit den drei Ständen aus §10.7:
vorgeschlagen, entschieden, tatsächlich geändert — nur so bleibt ein abgelehnter
Vorschlag nachvollziehbar. `log.md` bekommt weiterhin nur die Was-Zeile.

## 7. Die Evidenzbindung

Der Prüfer ist reine Textarbeit und kommt ohne Modell aus:

1. Existiert das benannte Segment im Paket?
2. Kommt das Zitat wortgleich darin vor — exakter Substring-Vergleich des
   Zitat-Fences im Segment-Fence, **nach CRLF→LF-Normalisierung**, derselben
   Regel wie `content_hash` (`identity.py:32`). Ohne sie fiele auf einem
   Windows-Checkout jedes Zitat durch.
3. Trägt jede Behauptung mindestens einen Beleg?

Zwei Riegel gehören dazu: Der Paketbauer wählt die Fence-Länge länger als der
längste Backtick-Lauf im Segment — Wiki-Absätze enthalten selbst Codeblöcke —,
und ein leeres Zitat gilt nicht als Beleg, sonst „käme" es in jedem Segment vor.

Fällt eine einzelne Behauptung durch, wird sie entfernt und im Fall vermerkt.
Fallen alle durch oder bleibt die Kernaussage unbelegt, wird der Vorschlag
verworfen und der Fall als manuell zu prüfen geführt — **kein zweiter Versuch**
(§10.6).

Damit sind die beiden Testfälle reine Textfixtures ohne Sprachmodell: ein
erfundenes Zitat fällt durch, ein wörtliches geht durch.

## 8. Freigabe und der isolierte Commit

Nach der Freigabe schreibt der Code: Wiki-Seite, `content_hash` und `revision`
fortschreiben, `generated.at`, der `verified`-Eintrag mit `human:<id>`,
`log.md`, `audit.md`, erneute technische Aktualisierung — und dann genau ein
Commit im Vault-Repo.

**Vor dem Schreiben** prüft der Code den Hash der Zielseite gegen `target_hash`.
Weicht er ab, wird nicht geschrieben; der Fall geht mit Hinweis zurück in die
Warteschlange (§10.4).

**Der Commit läuft über einen Scratch-Index**, nicht über `git add`:
`GIT_INDEX_FILE` auf eine Datei im Zustandsverzeichnis, `read-tree HEAD`, nur
die Pfade dieser Wissensänderung staffeln, dann `write-tree`, `commit-tree`,
`update-ref`. Der Grund ist §16.15: Fremde Prozesse arbeiten im selben Baum —
bei diesem Nutzer Obsidian samt Git-Plugin. Ein geteilter Index kollidiert mit
deren Staging, und ein fremdes `index.lock` ließe `brain approve` scheitern,
nachdem das Wiki schon geschrieben ist. Hinzu kommt: `git commit -- <pfade>`
committet den Arbeitsbaum-Stand dieser Pfade, während der Hash-Riegel aus §10.4
nur die Zielseite schützt.

Der Commit enthält: die Wiki-Seite, `log.md` und `audit.md` des Bereichs,
`_identities.tsv` und die **Löschung des Fallverzeichnisses**. Prüffälle sind
versionierter Vault-Inhalt (§5.1); bliebe ihre Löschung außen vor, wäre ein
zweiter Aufräum-Commit unvermeidlich und „genau ein Commit" nicht haltbar.

Drei Festlegungen dazu:

- **`update-ref` mit Compare-and-Swap.** Zwischen `read-tree` und `update-ref`
  kann ein fremder Prozess committen; ohne CAS ginge dieser Commit still
  verloren. Bei Abweichung wird neu von HEAD aufgesetzt, nie überschrieben.
- **`log.md` und `audit.md` bekommen keinen Hash-Riegel.** Es sind
  Anhänge-Dateien; ein Riegel darauf blockiert oft und schützt wenig. Ihr
  Arbeitsbaum-Stand wird mitcommittet, ausdrücklich.
- **Vault ohne Git-Repo:** Es wird trotzdem geschrieben — das Wiki ist nicht
  Geisel von Git —, der Commit-Schritt entfällt mit protokollierter Warnung.
  Scheitert der Commit nach dem Schreiben, wird gemeldet und erneut versucht;
  **nie** `git add -A`, nie ein Rollback des Geschriebenen.

Prüfbar ist das mechanisch: HEAD unterscheidet sich um genau einen Commit, und
`git show --name-status HEAD` liefert exakt die erwartete Pfadmenge.

## 9. Selbstheilung: Umbenennung einer Rohquelle

§5.6 schreibt das Verfahren wörtlich vor: Verschwindet ein Pfad und taucht
gleichzeitig ein unbekannter Pfad mit demselben Hash auf, gilt das als
Umbenennung — die `doc_id` bleibt, nur der Pfad wird fortgeschrieben, kein
Prüffall entsteht. Gleichzeitiges Umbenennen und Ändern wird nicht geraten,
sondern als „Quelle fehlt" plus „neue Quelle" im Prüfzentrum vorgelegt.

Der Code tut heute das Gegenteil: `cli.py:127–129` schlüsselt über den relativen
Pfad und prägt bei unbekanntem Pfad eine frische `doc_id`; `core.py:131–135`
hält das als Absicht fest. Das ist eine Spec-Code-Lücke, keine offene
Entwurfsfrage.

**Die Erkennung gehört in den Indexlauf, nicht in den Abgleich.** `_index_area`
baut das Register nur aus vorhandenen Dateien und überschreibt es am Ende
(`cli.py:139`); danach ist die Zeile des verschwundenen Pfads weg und die neue
`doc_id` längst geprägt. Ein nachgelagerter Abgleich fände nichts zu
vergleichen.

Festlegungen:

- **Mehrdeutigkeit** — mehrere verschwundene oder mehrere neue Pfade mit
  gleichem Hash — führt zu keiner automatischen Zuordnung, sondern zum Prüffall.
  Stille Neuprägung wäre Raten; §16.12 hat byteidentische Dateien im echten
  Bestand bereits nachgewiesen.
- **`revision` bleibt bei reiner Umbenennung unverändert** — gleicher Hash heißt
  keine Inhaltsänderung.
- **Geltung überall, wo ein `_identities.tsv` liegt**, also auch in Code-Repos.
  Bereichsübergreifende Verschiebungen laufen weiter über den `doc_id`-Lookup
  aus §5.7.3, nicht über Hash-Vergleich.
- Der Kommentar in `core.py:131–135` wird um die Umbenennungsausnahme ergänzt.

## 10. Der zweite Auslöser: Merge nach `main`

**Der Hook zeichnet nur auf.** Er schreibt Repo-Wurzel, Commit-Bereich
(`ORIG_HEAD..HEAD`), Zielzweig und Zeitstempel in eine Ereignisdatei im
Zustandsverzeichnis und endet mit bedingungslosem `exit 0`. Kandidaten, Evidenz
und Fall entstehen beim nächsten `brain reconcile` oder Daemon-Nachholen.

Die Alternativen scheitern belegbar: §15 verbietet den Systemdienst, und der
Daemon-Start unter Windows geht nur über den WMI-Umweg mit Job-Objekt — ein aus
dem Hook gestarteter Hintergrundprozess stürbe mit dem Job-Objekt oder zahlte je
Merge den Kaltstart. Ein synchroner `brain`-Aufruf stellte einen `uv`-Start in
jeden `git merge` und liefe in die Falle aus `hooks/git/ultraloom.sh:6–8`: `uv`
liest das `#` eines absoluten Pfades als URL-Fragment und schneidet dort ab. Der
Hook bleibt deshalb reines POSIX-sh mit eingebranntem Pfad, im Stil von
`hooks/git/pre-commit`.

Der Preis ist Latenz: Ein Merge am Freitagabend wird Montag zum Fall. Das ist
dieselbe Eigenschaft, die §10.3 für den Abgleich schon bewusst gewählt hat —
Aktualität hängt an der Benutzung.

Weiter festgelegt:

- **Idempotenz:** Dedup über den Commit-Bereich; die Ereignisdatei wird erst
  nach erfolgreicher Fallbildung gelöscht.
- **Zielzweig:** Der Hook prüft selbst, ob HEAD auf dem konfigurierten Zweig
  steht, sonst erzeugte jeder Feature-Merge ein Ereignis.
- **Nicht registriertes Repo:** stumm `exit 0`. `brain hook status` erkennt
  verwaiste Hooks, `brain hook remove` entfernt sie.
- **Aktivierung je Repo** über `[maintenance] on_merge = true` in dessen
  `.brain.toml` — nie global (Entscheidung 30). Kein Hook ohne ausdrückliche
  Zustimmung des Repos.
- **Zuordnung Repo → Bereich.** `registry.py` kennt heute `scope`, `path` und
  `workspace`, aber nichts, was einen Bereich als Repo markiert. Diese Scheibe
  legt die Zuordnung fest und baut sie.

Die Evidenz bleibt bei Pfaden und Commit-Nachrichten. **Kein Quelltext** — damit
bleibt §16.1 gewahrt, die Datenschutzprüfung einfach und die Kosten niedrig.

## 11. `realization` und die zwei Lint-Prüfungen

`realization: planned | in_progress | implemented | abandoned` und
`implemented_in` stehen in §5.4 und in Entscheidung 29, existieren aber nirgends
im Code — weder in `wiki/page.py` noch im Typkatalog-Design. Diese Scheibe baut
Feld, Parser und die zwei Prüfungen: Seiten, die seit langem `planned` sind, und
Seiten, die `implemented` melden, ohne einen Commit in `implemented_in` zu
nennen. Das Muster dafür liefern `untouched` und `stale` in `wiki/lint.py`.

`brain:wiki-plan` bleibt in dieser Scheibe. Es läuft zwar vor dem Merge, aber
das Fertig-Kriterium nennt es ausdrücklich, und die Scheibe wurde bewusst
ungeteilt geplant.

## 12. Abweichungen vom Architektur-Vertrag

Fünf Punkte weichen von der Architektur-Spec ab oder greifen über die Scheibe
hinaus. Sie stehen hier, damit sie entschieden sind statt still gebaut:

| # | Abweichung | Begründung |
|---|---|---|
| 1 | **Der Wächter aus §10.3 wird aufgeschoben.** Erkennung ist allein der Abgleich. | Der Wächter ist in §10.3 selbst als „prinzipiell unzuverlässig" geführt und wird vom Abgleich aufgefangen; er steht weder im Entscheidungsprotokoll noch im Fertig-Kriterium. Er fügt Bequemlichkeit hinzu, keine Korrektheit, und ist nachrüstbar. |
| 2 | **Ein fünfter Skill `brain:review`.** Entscheidung 28 legt „vier Skills" fest und benennt sie. | Keiner der vier ist für den Vorschlag bei Quellenänderung zuständig: `wiki-plan` plant vor dem Merge, `land` behandelt den Merge, `ingest` und `research` nehmen auf. Den Anlass in `land` mitzuführen hieße, zwei Auslöser in einen Skill zu zwingen. |
| 3 | **Eingriff in den Indexer aus Scheibe 1.** Die Umbenennungserkennung ändert `_index_area`. | Ohne den Eingriff ist das Fertig-Kriterium dieser Scheibe nicht erreichbar (§9). Die Byteidentitäts-Zusicherung von Scheibe 1 bleibt gültig — unveränderter Bestand hat keine Umbenennung — und wird im Plan erneut nachgewiesen. |
| 4 | **`commit-tree` umgeht die pre-commit-Hooks des Vault-Repos.** | Konstruktionsbedingt, nicht als Nebenwirkung. Der Weg über den Scratch-Index ist die Bedingung dafür, dass ein fremder Prozess im selben Baum den Commit nicht zerreißt (§8). |
| 5 | **Ist der Vorzustand nicht verifizierbar, führt das Analysepaket nur den neuen Stand.** §10.5 verlangt einen Quellendiff, §10.6 „was sich geändert hat, mit Fundstelle". | Die alte Fassung wird über `git show HEAD:<pfad>` im Repo der Quelle geholt und gegen den `content_hash` des Registers geprüft, CRLF→LF normalisiert wie in `identity.py`. Trifft der Hash nicht — Handcommit zwischen zwei Freigaben, Pfad nicht in der Historie, oder gar kein Git-Repo (Nachweis 10) —, dann wäre ein Diff gegen diese Fassung eine erfundene Änderung. Der Rückfall ist deshalb ein Paket ohne Vorzustand, **sichtbar im Paket vermerkt**, damit das Modell nicht so tut, als kenne es einen. Der Vorschlag kann dann nicht belegen, was vorher dort stand. Eine Schattenkopie im Zustandsverzeichnis wäre ein zweites Register und nach §3 verboten. |

## 13. Nachweis

Gegen die echten Bestände zu führen, nicht gegen Attrappen:

1. **Fall aus Quellenänderung.** Eine indizierte Rohquelle ändern,
   `brain reconcile` — es entsteht genau ein Fall für die abhängige Seite, mit
   Paket und nummerierten Segmenten.
2. **Ein Fall je Seite.** Zwei Quellen derselben Seite ändern — ein Fall, dessen
   Paket beide Quellen führt (§10.4).
3. **Evidenzbindung, beide Richtungen.** Ein Vorschlag mit erfundenem Zitat
   fällt durch, einer mit wörtlichem geht durch — als Textfixtures, ohne Modell.
4. **Freigabe.** Nach `brain approve` stimmen Quelle und Wiki, `revision` und
   `content_hash` sind fortgeschrieben, und `git show --name-status HEAD` zeigt
   genau die erwartete Pfadmenge — ein Commit, Fallverzeichnis gelöscht.
5. **Bewegte Zielseite.** Seite zwischen Fallbildung und Freigabe von Hand
   ändern — es wird nicht geschrieben, der Fall kehrt mit Hinweis zurück.
6. **Selbstheilung.** Eine Rohquelle umbenennen — kein Prüffall, das Register
   führt dieselbe `doc_id` mit neuem Pfad.
7. **Mehrdeutige Umbenennung.** Zwei byteidentische Dateien *gleichzeitig*
   umbenannt — keine stille Zuordnung, aber auch kein Prüffall: die alten
   `doc_id` werden nicht weitergereicht, die Dateien erscheinen als neue
   Quellen und die alten Pfade verschwinden aus dem Register. Eine
   verschwundene Rohquelle erzeugt nirgends einen Fall (`source_missing` ist
   mit Begründung nicht gebaut, siehe `reconcile.py`); nur eine *einzeln*
   umbenannte Datei ist eindeutig und behält ihre `doc_id` (Nachweis 6).
8. **Merge-Auslöser.** Zweig mit einer `planned`-Seite nach `main` bringen — der
   Fall nennt genau diese Seite, nach Freigabe steht sie auf `implemented` mit
   richtigem Commit, **und im Analysepaket findet sich kein Quelltext**.
9. **Datenschutz.** Ein Fall zu einem `local_only`-Bereich erzeugt keinen
   Skill-Weg, sondern einen manuellen Fall; das Prüfzentrum liefert auf dem
   Cloud-Kanal weder Treffer noch Inhalte noch Katalogeintrag.
10. **Kein Git-Repo.** Vault ohne Git — geschrieben wird, gewarnt wird,
    abgebrochen wird nicht.

Dazu Messung 4 aus §13 (`brain lint`) mit den zwei neuen
`realization`-Prüfungen.

## 14. Fertig-Kriterium

Das der Architektur-Spec (§17, Zeile 5), unverändert: nachgebauter Fall —
Abgleich erzeugt Fall, Vorschlag besteht die Evidenzbindung, nach Freigabe
stimmen Quelle und Wiki, genau ein Commit. Selbstheilung bei Umbenennung.
Evidenzbindung in beide Richtungen. Und der Merge-Nachweis samt der Zusicherung,
dass im Analysepaket kein Quelltext steht.

## 15. Offene Ränder

- **`automatic_cloud` bleibt unbeaufsichtigt nicht möglich.** Der Modus ist
  belegt — als offenes Tor —, aber ohne laufende Sitzung entsteht kein
  Vorschlag. Ein echt unbeaufsichtigter Lauf wäre eine neue Entscheidung für
  §18, keine Schuld dieser Scheibe.
- **Der Wächter** (Abweichung 1) bleibt nachzuholen, wenn die Latenz des
  Abgleichs im Alltag stört.
- **Die Gewichtung** zeit- gegen änderungsbasierter Fälligkeit ist als Feld
  angelegt, aber die Reihenfolge im Prüfzentrum ist noch nicht gemessen.

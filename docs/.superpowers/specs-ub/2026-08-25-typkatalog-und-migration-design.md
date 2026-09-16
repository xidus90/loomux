# Typkatalog und Typmigration: ein Vokabular, das nicht auseinanderläuft

**Stand:** 2026-08-25
**Bezug:** Architektur §4.2, §5.4, §5.5, §17; OKF v0.2 §4.1; Scheibe 3 (Wiki-Schicht)

---

## 1. Warum

Zwei Projekte, dieselbe Sache, zwei Namen: `space` nennt es `Design Decision`,
`iam_wiki` nennt es `Decision`. Eine Suche über beide Wikis nach Entscheidungen
findet heute die Hälfte. Das ist keine Vorhersage, sondern der gemessene
Zustand — bei gerade zwei Beständen.

Gezählt am 2026-08-25:

| Typ | space | iam_wiki | brain-knowledge |
|---|---|---|---|
| `Architecture` | 39 | 8 | – |
| `Open Question` | 32 | – | – |
| `Balancing Rule` | 32 | – | – |
| `Design Decision` | 30 | – | – |
| `Game System` | 20 | – | – |
| `Reference` | 12 | 2 | – |
| `Page` | – | 36 | – |
| `API Endpoint` | – | 13 | – |
| `Concept` | – | 9 | – |
| `Overview`, `Metric`, `Attested Computation` | – | je 3 | – |
| `Decision`, `Playbook`, `BigQuery Table` | – | je 1 | – |
| `<Type name>` (unausgefüllte Vorlage) | – | 1 | – |
| `Source` / `Topic` / `Entity` / `Synthesis` | 1 | – | 23 |

Drei Befunde stecken darin. Erstens die Drift (`Decision` gegen
`Design Decision`). Zweitens die Unterlassung: 36 Seiten auf `Page` sind kein
Typ, sondern eine nicht getroffene Entscheidung. Drittens ein Kopierfehler, den
niemand bemerkt hat — eine Seite trägt `type: <Type name> # REQUIRED`, also die
unausgefüllte Vorlage aus dem OKF-Dokument selbst.

Dazu kommt ein Widerspruch im eigenen Code: `lint.py` erlaubt genau vier Typen,
während das OKF, auf das sich diese Schicht beruft, ausdrücklich verlangt, dass
Konsumenten unbekannte Typen tolerieren („Type values are **not** registered
centrally … consumers MUST tolerate unknown types gracefully", OKF §4.1). Ein
fremdes, formal einwandfreies OKF-Wiki fiele bei uns durch die Prüfung.

*Anmerkung aus der Abnahme (2026-08-25): Die `space`-Spalte und die
`brain-knowledge`-Zeile kommen an `census()` (der Zählfunktion hinter
`brain types`) unverändert wieder heraus. Die `iam_wiki`-Spalte war per
`grep "^type:"` über das ganze Repository gezählt, ohne zwischen echten
OKF-Seiten und Beispiel-Frontmatter in fremdem Text zu unterscheiden — etwa dem
`type: Page`-Beispiel in einem Codeblock von
`.superpowers/sdd/2026-07-29-repo-doku-aufraeumen/task-2-brief.md`, einer
Planungsdatei ohne eigene Frontmatter. `census()` liest nur, was tatsächlich als
Frontmatter am Dateianfang steht, und findet für `Page` **28** statt 36 echte
Seiten unter `projects/frontend/pages/`; die übrigen Zeilen der Spalte dürften
in ähnlichem Maß mitgezählte Fremdtreffer enthalten. Für die Migrationsplanung
(§6, §9 Nachweis 6, §11) braucht `iam_wiki` deshalb eine saubere Neuzählung mit
`census()`, bevor die Zahlen dieser Spalte als Grundlage dienen.*

## 2. Was recherchiert wurde, und was daraus folgt

| Quelle | Vokabular | Verwertbar? |
|---|---|---|
| **OKF v0.2** (Google Cloud, Juni 2026) | `BigQuery Table`, `BigQuery Dataset`, `API Endpoint`, `Metric`, `Playbook`, `Reference`, `Attested Computation` | teilweise — Datenplattform-Begriffe; `API Endpoint`, `Metric`, `Reference` passen |
| **Karpathy, llm-wiki** | summaries, entity pages, concept pages, comparisons, overview, synthesis | Herkunft unserer vier; beschreibt Wissensarbeit, nicht Software |
| **Diátaxis** | Tutorial, How-to, Reference, Explanation | nein — beschreibt Dokumentation **für Produktnutzer**, nicht das Wissen der Bauenden |
| **arc42** | zwölf Abschnitte: Kontext, Bausteinsicht, Laufzeitsicht, Entscheidungen, Qualität, Glossar … | teilweise — das sind Abschnitte **eines** Dokuments, keine Seitentypen; `Glossary Entry` und der Betriebsgedanke sind entlehnt |
| **ADR / MADR** | Titel, Status, Kontext, Entscheidung, Konsequenzen; `status: proposed \| accepted \| superseded` | ja — als Feldsatz für `Decision` |

Der Ertrag ist schmaler als erhofft: Keine der etablierten Taxonomien
beschreibt, was ein Team über sein eigenes System weiß. Diátaxis zielt auf
Produktnutzer, arc42 auf ein Dokument, OKF auf Datenplattformen. Der Kern
dieses Katalogs stammt deshalb nicht aus der Literatur, sondern aus dem eigenen
Bestand — aus dem, was in zwei unabhängigen Projekten von selbst entstanden
ist.

## 3. Der Katalog

Drei Ränge, mit absteigender Verbindlichkeit.

### 3.1 Rang 1 — der Kern, gilt in jedem Projektwiki

| Typ | Beantwortet | Nicht dafür da |
|---|---|---|
| `Architecture` | Wie ist es gebaut? Aufbau, Zusammenspiel, Datenfluss, Grenzen | nicht das Warum — das ist `Decision` |
| `Decision` | Warum so und nicht anders? Wahl, Alternativen, Folgen | nicht der Zustand danach — der steht in `Architecture` |
| `Open Question` | Was ist ungeklärt, und was hängt daran? | kein Bug, keine Aufgabe — die gehören ins Ticketsystem |
| `Reference` | Worauf stützen wir uns? Fremde Norm, Papier, Bibliothek, externe API | nichts Eigenes — Eigenes ist `Architecture` |

Belegt durch den Bestand: `Architecture` 47 Seiten über zwei Projekte,
`Decision` 31, `Open Question` 32, `Reference` 14.

`Decision` trägt zusätzlich `status: proposed | accepted | superseded` nach
MADR. Der Grund steht im eigenen Verlauf: Entscheidung 46 der Architektur wurde
zweimal gedreht, und ohne Status ist einer Seite nicht anzusehen, ob sie noch
gilt.

### 3.2 Rang 2 — der Katalog, optional aber namensverbindlich

Ein Bereich aktiviert daraus, was er braucht. **Wer den Fall hat, nimmt den
Katalognamen** — das ist die Regel, an der die Drift scheitert.

| Typ | Wann | Herkunft |
|---|---|---|
| `API Endpoint` | eine aufrufbare Schnittstelle je Seite | OKF-Beispielwert |
| `Data Model` | eine Tabelle, ein Schema, eine dauerhafte Struktur | eigener Vorschlag |
| `Metric` | eine Messgröße samt Erhebungsweise und Zahlen | OKF-Beispielwert |
| `Runbook` | eine wiederkehrende Handlung: einrichten, ausrollen, wiederherstellen | OKF nennt `Playbook`; arc42 kennt den Betriebsaspekt |
| `Glossary Entry` | ein Fachbegriff, den das Projekt eigen verwendet | arc42 §12 |

`Metric` ist für ultra-brain sofort einschlägig: Das Messprogramm (§13) hat
mehrfach Entscheidungen umgeworfen — 41/50 gegen 26/50, 95 ms gegen 5351 ms.
Diese Zahlen liegen heute in `bench/` und im Vault unter `98 Messung/`, ohne
Typ und damit ohne Bezug zu den Entscheidungen, die sich auf sie stützen.

### 3.3 Rang 3 — bereichseigene Typen, frei aber deklariert

Was nur ein Projekt kennt, erklärt es im eigenen Manifest:

```toml
[wiki]
types = ["Balancing Rule", "Game System"]
```

Für `space` sind das genau diese zwei (52 Seiten). Der Lint akzeptiert sie
dann — aber weil sie dastehen, nicht weil er alles durchwinkt. Ein Tippfehler
fällt weiterhin auf, und das ist der einzige Grund, warum hier überhaupt eine
Deklaration verlangt wird statt schlichter Freiheit.

### 3.4 Was nicht aufgenommen wird

| Abgelehnt | Grund |
|---|---|
| `Page` (36×) | kein Typ, sondern eine unterlassene Entscheidung; aufzunehmen hieße, sie zu legitimieren |
| `Concept` | Dopplung zu `Entity`, das in OKF, bei Karpathy und im Kern steht |
| `Overview` | Dopplung zu `index.md`, das ohnehin auf jeder Ebene liegt |
| `Tutorial`, `How-to`, `Explanation` | Diátaxis zielt auf Produktnutzer; kommt in den Katalog, sobald ein Projekt veröffentlicht wird |
| `Building Block View`, `Runtime View` | arc42-Abschnitte eines Dokuments; als Typen erzwängen sie eine Gliederung, die kleine Projekte erdrückt |
| `Attested Computation`, `BigQuery Table` | Googles Datenplattform-Welt; für uns Rang 3, wenn überhaupt |
| `Incident`, `Postmortem` | sinnvoll, sobald etwas produktiv betrieben wird — heute betreiben wir nichts |

### 3.5 Die zweite Achse, und ihre Unschärfe

`Source`, `Topic`, `Entity` und `Synthesis` bleiben gültig. Sie sagen etwas
anderes als der Kern: nicht **worüber** eine Seite handelt, sondern **wie sie
entstanden ist** — aus einer Quelle verdichtet, aus mehreren zusammengeführt,
als Steckbrief, als These mit Belegen.

Das ist die schwächste Stelle dieses Entwurfs, und sie wird hier benannt statt
verschwiegen: Das OKF hat nur **ein** `type`-Feld, also müssen beide Achsen
hineinpassen. Eine Seite ist entweder `Architecture` oder `Topic`, nie beides.

In der Praxis dürfte die Trennung nach Urheberschaft genügen: Was
`brain:ingest` schreibt, trägt die Herkunftstypen; was Menschen und
Programmieragenten schreiben, trägt die Kerntypen. Erweist sich das als zu eng,
ist der saubere Ausweg ein zweites Feld `origin:` als produzenteneigene
Erweiterung — OKF §4.1 erlaubt sie ausdrücklich. Diese Erweiterung wird **jetzt
nicht** gebaut; sie ist die vorgesehene Antwort, falls der Fall eintritt.

## 4. Durchsetzung

Der Lint prüft künftig gegen drei Mengen statt gegen eine:

1. Kern und Katalog — immer gültig.
2. Die im Manifest deklarierten Typen des Bereichs.
3. Alles andere — Fehler, mit dem Hinweis, dass der Typ entweder in den
   Katalog oder ins Manifest gehört.

Der Katalog lebt **im Code**, in `src/brain/wiki/types.py`, mit Rang und
Aliastabelle. Die Architektur §5.4 führt dieselbe Tabelle für Menschen, mit dem
ausdrücklichen Hinweis, dass der Code die Wahrheit ist. Eine `types.toml` im
Vault wäre flexibler, erlaubte aber, den Katalog ohne sichtbaren Commit zu
ändern — genau das soll er nicht.

*Korrektur aus der Abnahme (2026-08-25): Dieser Absatz nannte `PAGE_TYPES` und
Architektur §9.1. Gebaut wurde `CORE_TYPES`/`CATALOGUE_TYPES`/`ORIGIN_TYPES` in
`src/brain/wiki/types.py`, und die von Menschen lesbare Tabelle steht in
Architektur §5.4 (OKF-Reichweite) — §9.1 behandelt das Bundle-Schema, trägt
aber keine Typtabelle. Beides oben korrigiert.*

### 4.1 Eine dritte Regel: `outside-area`

Die Regel `dead-link` prüft heute beides, was eine Migration gefährdet — die
Verlinkung zwischen Seiten **und** die Quellpfade im Frontmatter:

```python
targets = (*page.links, *(source.resource for source in page.sources))
```

Sie hat aber eine Lücke, die genau der Umzug aufreißt: Was `_resolve` mit `None`
beantwortet, wird **still übersprungen**. `None` heißt laut eigenem Docstring
„der Link zeigt aus dem Bereich hinaus". Ein Bezug, der den Bereich verlässt,
wird also nicht als tot gemeldet, sondern gar nicht geprüft.

Heute ist das harmlos, solange die Ziele im selben Bereich liegen. Nach einem
Umzug ins Vault-Bundle wäre es gefährlich: Ein Ziel, das die neue
Bereichswurzel mit `..` verlässt oder absolut geschrieben ist, verschwände
lautlos aus der Prüfung — genau dann, wenn man sich nach der Migration am
meisten auf sie verlassen will.

Das ist kein Fehler des ursprünglichen Entwurfs: Die Regel wurde für Bundles
gebaut, deren Quellen im selben Bereich liegen. Erst der Umzug schafft den Fall,
in dem sie danebengreift.

*Korrektur aus der Abnahme (2026-08-25): Hier stand, die 250 Quellbezüge von
`space` auf `docs/.superpowers/…` würden von `_resolve` als „außerhalb"
eingestuft, sodass der Lint dazu schwiege. Am echten Bestand stimmt das nicht:
Diese 250 Bezüge sind relative Pfade ohne führenden Slash und ohne `..` —
`_resolve` liefert für sie kein `None`, sondern einen (falschen)
bereichsrelativen Pfad, und `dead-link` meldet alle 250 schon heute als Fehler,
nicht stillschweigend. Die Lücke, die `outside-area` schließt, betrifft nur
Ziele, für die `_resolve` wirklich `None` liefert: einen absoluten Pfad (seit
Aufgabe 4 eigens als `absolute-link` gemeldet) oder ein `..`, das über die
Bereichswurzel hinausklettert. Für Letzteres liefert `space/wiki` heute 0
Fälle — die Regel ist gebaut, aber am echten Bestand noch ungeprüft; die
Gegenprobe mit einem `brain://`-Verweis steht in §9 Nachweis 8.*

**`outside-area`** schließt sie: Ein Link oder Quellbezug, der den Bereich
verlässt, wird gemeldet statt übergangen — als **Warnung**, nicht als Fehler.
Ein bewusster Verweis über Bereichsgrenzen ist legitim und an seiner Form
erkennbar (`brain://knowledge/…`); ein zurückgebliebener Repo-Pfad ist es nicht.
Die Regel unterscheidet daran: `brain://`-Verweise gehen durch, nackte Pfade ins
Nichts werden gemeldet.

Damit braucht die Linkprüfung **keinen eigenen Skill**. Sie ist rein mechanisch
— es gibt nichts zu urteilen, nur nachzusehen, ob eine Datei existiert — und
gehört nach §4.2 in den Code, wo sie bereits sitzt. `brain lint --scope <bereich>`
läuft jederzeit, auch ohne Migration.

## 5. Zwei Werkzeuge

Die Arbeitsteilung folgt §4.2: Zählen und Schreiben ist Code, Einordnen ist KI,
Entscheiden ist Mensch.

### 5.1 `brain types` — zählt

Ein Befehl ohne Modell. Geht über alle registrierten Bereiche, liest jede
Wiki-Seite und gibt eine Tabelle aus: Typ, Anzahl je Bereich, Rang. Auffälliges
wird markiert — bekannte Aliasse, Nicht-Typen, unausgefüllte Vorlagen.

Das ist mechanisch bestimmbar und gehört deshalb nicht in einen Skill. Es ist
zugleich das Werkzeug, das jedes der vier Projekte sofort brauchen kann, ohne
dass irgendetwas migriert wird.

### 5.2 `brain:migrate-types` — der Skill, der urteilt

Sechs Phasen. Die zwei teuren laufen in Subagenten, die **nur lesen** und einen
Digest zurückgeben; keine Phase außer 4 schreibt, und Phase 4 schreibt nur, was
der Plan sagt.

| Phase | Wer | Was im Kontext bleibt |
|---|---|---|
| 0 Bestand | Code (`brain types`) | eine Tabelle |
| 1 Einordnung | Subagent je unbekanntem Typ | ein Satz je Typ |
| 2 Entscheidung | Mensch, Hauptsitzung | der Migrationsplan als Datei |
| 3 Einzelfälle | Subagent, stapelweise | Vorschlagsliste, keine Dateiinhalte |
| 4 Ausführung | Code, ohne Rückfrage | die Zusammenfassung |
| 5 Abnahme | Code (`brain lint`, `brain status`) | Lint-Ausgabe, Linkzahlen |

**Phase 1** liest je unbekanntem Typ drei bis fünf Stichproben, nie alle Seiten,
und liefert einen Satz: was der Typ beschreibt, ob er in einen Kerntyp passt,
und falls nicht, ob er in den Katalog gehört. Der Maßstab dafür ist messbar,
nicht Geschmack: **Kommt der Typ in mehr als einem Projekt vor, oder beschreibt
er etwas, das jedes Programmierprojekt hat?** Dann Katalog, sonst Rang 3.

**Phase 3** ist der Fall `Page`: 36 Seiten ohne brauchbaren Typ. Ein Subagent
sieht jede kurz an und liefert eine Tabelle aus Dateiname, Vorschlag und
Sicherheit. Was er sicher einordnet, geht als Vorschlag in den Plan; was offen
bleibt, geht als kurze Liste an den Menschen — am Stück, nicht Datei für Datei
über Stunden.

**Phase 4** läuft ohne Rückfragen und ist idempotent: zweiter Lauf, keine
Änderung, null geschriebene Dateien. Ein Abbruch mitten im Lauf ist damit
folgenlos.

**Phase 5** vergleicht, statt nur zu prüfen. `brain lint` und `brain status`
laufen **vor** der Migration und danach; abgenommen ist sie, wenn kein Befund
hinzugekommen ist und die Linkquote nicht gefallen ist. „Ist es sauber" wäre die
schwächere Frage — ein Bundle, das vorher schon Befunde trug, bestünde sie nie.

## 6. Der Migrationsplan

Das Rückgrat: Er entsteht in Phase 2, überlebt jeden Kontextverlust, ist vor der
Ausführung prüfbar, und macht Phase 4 zu reiner Mechanik.

```yaml
bereich: project/space
entscheidungen:
  - typ: Design Decision
    aktion: umbenennen
    ziel: Decision
    seiten: 30
  - typ: Balancing Rule
    aktion: bereichseigen deklarieren
    seiten: 32
  - typ: Page
    aktion: einzeln entscheiden
    seiten: 36
```

Er liegt unter `docs/.superpowers/sdd/<datum>-typmigration-<bereich>.md` — bei
den Arbeitsspuren, nicht im Wiki. Er ist Beleg, nicht Wissen.

## 7. Typen und Umzug sind zwei Migrationen

Ausdrücklich getrennt, weil sie verschieden riskant sind:

**Eine Typumbenennung ändert nur das Frontmatter.** Kein Pfad, kein Link ist
betroffen. Die 30 `Design Decision`-Seiten sind ein rein mechanischer Fall.

**Ein Umzug ändert Pfade**, und dann müssen Links nachgezogen werden. Dabei ist
der Wurzel-Versatz zu beachten: Die **1038** absoluten Verweise im space-Wiki
(1030 Links, 8 Quellbezüge über 166 Seiten) sind ab **Wiki**-Wurzel geschrieben
(`](/architecture/x.md)`), während `graph.py` sie gegen die **Bereichs**-Wurzel
auflöst. Daher lösen sich heute nur 179 von 1494 Kanten auf. Wandert das Wiki
so, dass die Wiki-Wurzel zur Bereichswurzel wird, stimmen alle 1038 ohne eine
einzige Änderung; wandert es anders, müssen sie umgeschrieben werden.

*Korrektur aus der Abnahme (2026-08-25): Hier stand „2415 Links". Die Zahl
stammte aus einer groben `grep`-Zählung über alle 178 Dateien des space-Wikis,
die Gerüstdateien (`index.md`, `log.md`) und Mehrfachvorkommen je Zeile
mitzählte. Die `absolute-link`-Regel (Aufgabe 4) misst am selben Bestand die
1038 oben; `outside-area` meldet dort 0. Der Befund selbst — dass die
Wiki-Wurzel gegen die Bereichswurzel versetzt ist und deshalb nur 179 von 1494
Kanten auflösen — ändert sich dadurch nicht, nur die Zahl der betroffenen
Verweise war zu hoch.*

Die Typmigration läuft deshalb **zuerst**, im Repo, wo alles noch stimmt. Der
Umzug ist eine eigene Arbeit mit eigener Abnahme — und nach der ersten
Migration darf man sich gegen die zweite entscheiden.

## 8. Keine leeren Ordner

`brain wiki init` legt heute stur `sources/`, `topics/`, `entities/` und
`syntheses/` an. Künftig legt es `index.md`, `log.md` und `_schema.md` an; ein
Ordner entsteht, wenn die erste Seite hineingeschrieben wird.

Das löst das Problem an der Wurzel: Ein Bundle für `space` bekommt keine vier
toten Ordner neben seine gewachsene Gliederung, und `brain:ingest` legt an, was
es braucht, wenn es etwas braucht.

## 9. Was diese Arbeit beweisen muss

1. **`brain types` zählt richtig** — die Zahlen aus §1 kommen an den echten
   Beständen wieder heraus, über alle vier Bereiche.
2. **Der Lint akzeptiert Kern und Katalog**, lehnt Unbekanntes ab und
   akzeptiert es, sobald es im Manifest steht.
3. **Ein Tippfehler fällt durch** — `Architecure` wird abgelehnt, obwohl
   `Architecture` gültig ist.
4. **Ein fremdes OKF-Wiki** mit deklarierten Typen besteht die Prüfung, ohne
   dass eine seiner Seiten geändert wird.
5. **Die Typmigration an `space`**: 30 Seiten von `Design Decision` auf
   `Decision`, zweiter Lauf schreibt nichts, kein Link ändert sich, der Lint ist
   danach sauber.
6. **Phase 3 an `iam_wiki`**: Die 36 `Page`-Seiten ergeben eine Vorschlagsliste,
   und die Zahl der Fälle, die der Mensch entscheiden muss, steht im Bericht.
7. **`brain wiki init` legt keinen leeren Ordner mehr an.**
8. **`outside-area` schlägt an:** Eine Seite, deren Quellbezug aus dem Bereich
   hinauszeigt, wird gemeldet — heute schweigt der Lint dazu. Gegenprobe: Ein
   `brain://`-Verweis über Bereichsgrenzen geht durch, ohne Befund.
9. **Phase 5 vergleicht:** Ein Bundle, das vor der Migration Befunde trug, wird
   abgenommen, solange keiner hinzukommt — und abgelehnt, sobald einer dazukommt.

## 10. Reihenfolge

1. `brain types` — Code, klein, sofort für alle vier Projekte nützlich.
2. Katalog und Lint-Regel, Manifest-Feld `[wiki] types`.
3. Die Regel `outside-area` (§4.1) — klein, gehört in dieselbe Datei wie
   Punkt 2, und ohne sie kann Phase 5 nichts beweisen.
4. `brain wiki init` ohne leere Ordner.
5. Der Umbenennungslauf — mechanisch, idempotent.
6. Der Skill `brain:migrate-types` mit seinen sechs Phasen.
7. `space` als erster echter Fall.

Die Punkte 1 bis 5 sind Code und gehören in einen Plan; Punkt 6 ist Prosa und
gehört in einen eigenen. Punkt 7 ist die Abnahme.

## 11. Was offen bleibt

- **Der Kern beruht auf zwei Projekten.** Er wird als Katalog mit Änderungsdatum
  geführt, nicht als Naturgesetz; beim dritten Projekt wird nachgesehen, ob
  etwas fehlt.
- **Die 36 `Page`-Seiten in iam_wiki** werden gemeldet, nicht im ersten Wurf
  umgeschrieben. `iam_wiki` ist `readonly` registriert.
- **Das Feld `origin:`** für die zweite Achse wird nicht gebaut, sondern
  vorgemerkt (§3.5).
- **Der Umzug ins Bundle** ist eine eigene Arbeit mit eigener Spec (§7).

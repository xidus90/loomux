# ultra-brain — Architektur-Design

**Stand:** 2026-08-18 · **Status:** zur Freigabe · **Zielgruppe:** intern (Entscheidungsgrundlage)

Dieses Dokument ist der Vertrag. Taucht beim Bauen eine Streitfrage auf, gilt,
was hier steht — nicht die Erinnerung und nicht die Auslegung durch ein Modell.

---

## 1. Zweck und zentrale Frage

**Wie wird aus gesammeltem Wissen ein System, dem ich vertrauen kann — statt ein
wachsender Müllhaufen?**

Die Frage hat zwei Hälften:

- **Auffindbarkeit** — jede Information wiederfinden, auch ohne das exakte Wort.
- **Vertrauen** — Duplikate, Widersprüche und veraltete Stände fallen auf, statt
  still herumzuliegen.

Die Optik ist ein Fenster. Ein Fenster baut man zuletzt ein.

---

## 2. Fähigkeiten

Fähigkeiten vor Werkzeugen. Jede Fähigkeit bekommt genau ein Bauteil.

| Fähigkeit | Bedeutet | Bauteil |
|---|---|---|
| Finden | Relevantes in Sekunden, auch ohne das exakte Wort | Suchschicht (qmd hinter eigener Schnittstelle) |
| Lesen | Treffer direkt öffnen, ohne App-Wechsel | Obsidian, später Web-App |
| Sauber bleiben | Widersprüche erkennen, Wissen verdichten | Wiki-Schicht (OKF) |
| **Aktuell bleiben** | Merken, wenn eine Quelle sich ändert | **Brain Maintenance** |
| **Abgrenzen** | Einem Projekt nur dessen Wissen zeigen | **Scoping über Manifest** |
| **Wiederverwenden** | Muster über Projekte hinweg teilen | **Geteilte Bereiche + Beförderung** |
| Überblick | Das System auf einen Blick | Indexer + Graph + Web-App |

Die drei fett gesetzten Fähigkeiten gehen über die Referenzarchitektur hinaus
und sind projektspezifisch.

---

## 3. Grundsätze

1. **Markdown ist die einzige Wahrheit.** Index, Graph, Zustandsdatenbank und
   Web-App sind abgeleitete Sichten. Sie dürfen jederzeit gelöscht werden; der
   nächste Lauf baut sie neu, ohne dass Wissen verloren geht.
2. **Deterministisch, wo es geht — und dann auch reproduzierbar.** Ein Modell
   läuft nur dort, wo Verstehen nötig ist. Alles andere ist normaler Code, und
   dieser Code liefert bei gleicher Eingabe byteweise gleiche Ausgabe.
3. **Der Mensch entscheidet, die KI liefert.** Bei jedem Widerspruch, jedem
   Prüffall, jedem Optik-Schritt.
4. **Lokal zuerst.** Wissensbasis, Index, Graph und Suche laufen vollständig auf
   dem eigenen Rechner. An ein Cloud-Modell gehen nur die dafür ausgewählten
   Ausschnitte.
5. **Ausfälle degradieren, sie blockieren nicht.** Fällt eine Schicht aus, wird
   das System unbequemer, nie unbenutzbar.
6. **Rohquellen sind unantastbar.** Die KI liest sie, ändert sie nie. Sie
   schreibt ausschließlich in die Wiki-Schicht.

---

## 4. Architektur

### 4.1 Bausteine

```
   Nutzer                  Claude Code                  Browser
     │                          │                          │
     │  brain search …          │  MCP                     │  liest graph.json
     ▼                          ▼                          ▼
┌────────────────────────────────────────────┐      ┌──────────────┐
│  KERN  (Python 3.14, langlebiger Prozess)  │      │   Web-App    │
│                                            │      │ (Scheibe 7)  │
│  Scope-Auflösung · Katalog · Werkzeuge     │      └──────────────┘
│  ┌──────────────────────────────────────┐  │              ▲
│  │  Suchschnittstelle (austauschbar)    │  │              │
│  │        └── qmd (BM25 + Vektor +      │  │      ┌──────────────┐
│  │             Reranking, lokal)        │  │      │   Indexer    │
│  └──────────────────────────────────────┘  │      │ deterministisch
│  ┌──────────────────────────────────────┐  │      │ graph.json   │
│  │  Vorschlaggeber (3 Umsetzungen)      │  │      │ index.md     │
│  └──────────────────────────────────────┘  │      └──────────────┘
│  ┌──────────────────────────────────────┐  │              ▲
│  │  Brain Maintenance                   │  │              │
│  └──────────────────────────────────────┘  │              │
└────────────────────────────────────────────┘              │
                     │  liest, schreibt nie in Rohquellen   │
                     ▼                                      │
┌─────────────────────────────────────────────────────────────────┐
│  MARKDOWN — die einzige Wahrheit                                │
│   Wissens-Vault:   Rohquellen (frei)                            │
│                    Wikis (OKF) aller Bereiche außer denen, die  │
│                    ihr Wiki im Repo tragen  ·  Prüfzentrum      │
│   Code-Repos:      Rohquellen: README, CHANGELOG, ADRs — und    │
│                    im schreibbaren Bereich ggf. sein Wiki (§5.3)│
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Arbeitsteilung

| Aufgabe | Code | KI | Mensch |
|---|:--:|:--:|:--:|
| Indexieren, Suchen, Scope auflösen | x | | |
| Katalog und Graph erzeugen | x | | |
| Änderungen erkennen, Betroffenheit ermitteln | x | | |
| Eignung und Datenschutz prüfen | x | | |
| Analysepaket schnüren | x | | |
| Evidenzbindung prüfen | x | | |
| Verdichten, Verknüpfen, Widersprüche erkennen | | x | |
| Prüfvorschlag formulieren | | x | |
| Entscheiden (Widerspruch, Freigabe, Optik) | | | x |
| Wiki schreiben, Hash fortschreiben, protokollieren | x | | |

Ein Modell kommt im Alltag an genau zwei Stellen vor: Ingest und Prüfvorschlag.
Suchen, Katalogisieren und Scoping kommen ohne aus. Ist das lokale Modell
eingeschaltet (§11), kommen zwei belanglose Stellen dazu — Ablagevorschlag beim
Import und Ein-Satz-Beschreibungen für Kataloge; beide ohne Wirkung auf den
Wissensbestand, beide durch einen Menschen beziehungsweise durch triviale
Rückfälle abgesichert.

### 4.3 Die Vertrauenskette

Steigender Prüfaufwand, zunehmende Nähe zur Quelle:

**orientieren** (Katalog) → **verdichtet lesen** (Wiki) → **breit suchen**
(Rohquellen) → **am Original prüfen**.

Eine Quellenangabe beweist nur, dass eine Quelle gefunden wurde — nicht, dass
sie aktuell ist, richtig verstanden wurde oder die Aussage stützt.

### 4.4 Die vier Fehlerstellen

| Stelle | Fehler | Gegenmittel |
|---|---|---|
| 1 | Auswahlfehler: Quelle wird nicht gefunden | Hybridsuche, Fragensatz-Messung |
| 2 | Kontextfehler: falsch zugeschnitten übergeben | Chunking-Messung, Abschnittslesen |
| 3 | Generierungsfehler: falsch interpretiert | Quellenangaben, Vertrauenskette |
| 4 | Kompilierungsfehler: falsche Verdichtung im Wiki | Ingest gegen Original, Konflikte markieren, Brain Maintenance |

Stelle 4 existiert nur, weil es eine generierte Wissensschicht gibt. Sie wird
bewusst in Kauf genommen und mit drei Riegeln entschärft: immer gegen die
Originalquelle schreiben, Widersprüche markieren statt auflösen, Prüflauf plus
lesbares Protokoll.

---

## 5. Datenmodell

### 5.1 Drei Orte

**Verdichtetes liegt an dem Ort, den sein Bereich nennt; Rohquellen liegen
dort, wo sie entstehen.** Das ist die tragende Aufteilung. Bis zur
Entscheidung 48 gab es für die linke Hälfte nur eine Antwort — den Vault;
seither kann ein schreibbarer Bereich sein Wiki im eigenen Repo tragen
(§5.3). Was unverändert gilt: **je Bereich genau ein Wiki, an genau einem
Ort, benannt an genau einer Stelle** (§5.5.1).

| Ort | Inhalt | Versioniert |
|---|---|---|
| Wissens-Vault | eigene Notizen **plus die Wikis der vault-platzierten Bereiche** — heute alle außer `project/ultra-brain`, also auch die schreibbaren `project/ecoflow` und `hub` — plus alle Prüffälle | eigenes Git-Repo |
| Code-Repos | Rohquellen: README, CHANGELOG, ADRs, Spec, Plan — plus das Manifest; **ein schreibbarer Bereich darf zusätzlich dessen Wiki tragen** unter `[layout] wiki` (§5.3, Entscheidung 48) | vorhandenes Repo |
| Zustandsverzeichnis | Index, Modelle, Zustandsdatenbank | **nie** |

Begründung für die Zusammenlegung der Wikis — sie trägt weiterhin für jeden
Bereich, der im Vault bleibt, und ist für die schreibbaren durch
Entscheidung 48 punktweise überholt:

- **Ein Ort zum Lesen.** Ein Obsidian-Vault, ein Graph, ein Prüfzentrum, ein
  Lint-Lauf, ein Protokoll — statt je Projekt eines. **Das ist der Posten,
  den Entscheidung 48 wirklich kostet:** ein Wiki im Repo lässt sich im Vault
  nicht mehr mitlesen. Bezahlt wird das mit dem **Hub-Zeiger**: er liegt im
  Vault als `91 Projekte/ultra-brain.md`, dort, wo die Projekte liegen, und
  der **Wegweiser** in `90 Wiki/index.md` verlinkt ihn (2026-08-29 §7). Die
  übrigen Posten verlieren nichts: Prüfzentrum und Lint laufen über die
  Registrierung und damit über alle Bereiche; Graph und Protokoll waren
  ohnehin nie gemeinsam — `graph.json` entsteht je Bereich (§5.8), `log.md`
  und `audit.md` je Bundle.
- **Code-Repos bleiben veröffentlichbar.** Läge das Wiki im Repo, würde es bei
  einer Veröffentlichung mitgehen: Verdichtungen, offene Konflikte,
  Prüfprotokoll. Das betrifft dieses Projekt unmittelbar. **Dieser Grund ist
  nicht widerlegt, sondern durch Entscheidung 48 zum bezahlten Preis
  geworden:** ein schreibbarer Bereich legt sein Wiki ins Repo und
  veröffentlicht es mit; wer ein Repo tatsächlich veröffentlicht, prüft das
  (§5.3, 2026-08-29 §1.3).
- **Keine Wiki-Commits in der Code-Historie.** Gilt weiter für die
  vault-platzierten Bereiche; wo das Wiki im Repo liegt, sind die
  Wiki-Commits gerade der Zweck.
- ~~**Der Versionsvorteil des Repos existiert nicht.**~~ Das war das
  stärkste Gegenargument und stützte sich auf §16.1: Der Abgleich läuft
  ausdrücklich nicht gegen Code-Inhalte, das Projekt-Wiki ist also nicht an
  den Codestand gekoppelt. **Entscheidung 48 hält diesen Punkt für zu eng
  gefasst:** der Vorteil liegt nicht im Abgleich, sondern darin, dass eine
  Seite und die Änderung, die sie beschreibt, im selben Zweig liegen und im
  selben Merge landen. Was ohnehin commit-gebunden ist — ADRs, Spec, Plan,
  README — sind Rohquellen und bleiben im Repo.

Die Verweise über die Repo-Grenze hinweg leisten die `brain://`-Referenzen mit
`doc_id`-Selbstheilung (§5.7.3). Sie mussten dafür nicht erweitert werden.

**Es gibt zwei Platzierungen, und welche gilt, ist eine Eigenschaft des
Bereichs** (Entscheidung 48, §5.3): ein schreibbarer Bereich darf sein Wiki
über `[layout] wiki` im eigenen Repo tragen, ein `readonly`-Bereich nicht.
Die Frage „wo liegt das Wiki von X" hat damit nicht mehr überall dieselbe
Antwort — wohl aber weiterhin **genau eine**, und beantwortet wird sie an
genau einer Stelle: der Registrierung (§5.5.1). Eine zweite Betriebsart des
Programms entsteht dadurch nicht — beide Platzierungen laufen durch denselben
Code. Das ist dieselbe Zurückhaltung, die §5.7.1 den Bereichsfamilien
auferlegt (die Familie steckt im Namen, nicht im Code), hier auf die
Platzierung angewandt (2026-08-29 §3).

Daraus folgt eine Eigenschaft des Datenmodells, die im Code sichtbar bleiben
muss: **Der Ort des Wikis und der Ort der Rohquellen sind zwei unabhängige
Eingaben; der eine wird nirgends aus dem anderen abgeleitet.** Bei
Vault-Platzierung liegen sie ohnehin in verschiedenen Repos; bei
Repo-Platzierung im selben Baum — und gerade dort muss die Unabhängigkeit im
Code sichtbar bleiben, weil sie sich nicht mehr von selbst versteht.

Das Zustandsverzeichnis liegt am plattformüblichen Ort für Anwendungsdaten,
außerhalb jedes versionierten Baums. **Der Ort ist einstellbar**, mit einer
Auflösungsreihenfolge: `--state-dir` auf der Kommandozeile schlägt
`BRAIN_STATE_DIR` in der Umgebung, das schlägt die Plattformvorgabe. Alles
Weitere wird daraus abgeleitet und nirgends hartkodiert — die Registrierung,
die Artefakte der `readonly`-Bereiche, später Modelle und Zustandsdatenbank.

Der Schalter ist keine Bequemlichkeit: Er ist die Voraussetzung dafür, dass ein
Testlauf die Wege über das Zustandsverzeichnis vollständig prüfen kann, ohne je
das echte anzufassen. Es enthält ausschließlich Abgeleitetes — mit der einen
Ausnahme, die `readonly` erzwingt (§5.5).

### 5.2 Wissens-Vault

```
Wissen/
  .brain.toml
  index.md
  log.md
  00 Eingang/                  Eingangsordner; Konverter arbeiten hier
  01 Vorlagen/
  02 Daily Notes/
  10 Rohquellen/               unantastbar; Themencluster darunter
  90 Wiki/                     Wiki des Bereichs knowledge
     _schema.md  index.md  log.md  audit.md  _identities.tsv
     quellen/  themen/  entitaeten/  synthesen/
  91 Projekte/                 eigener Bereich `hub`: die vault-platzierten
                               Projekt-Wikis und die Hub-Zeiger
     ecoflow/                  ein OKF-Bundle je vault-platziertem Projekt
     <weiteres-projekt>/
     ultra-brain.md            Hub-Zeiger: dieses Wiki liegt im Repo (§5.3)
  92 Engineering/              geteilte Bereiche (§5.7)
     python/  frontend/
  95 Prüfzentrum/              offene Fälle aller Bereiche, nach Bereich gruppiert
  99 Archiv/
```

Jedes Bundle unter `91 Projekte/` und `92 Engineering/` hat denselben Aufbau
wie `90 Wiki/` — eigenes Schema, eigene Kataloge, eigene Protokolle, eigenes
Identitätsregister. Es sind vollwertige, voneinander unabhängige Bereiche, die
sich lediglich ein Repo und einen Obsidian-Vault teilen.

Ordnernamen sind **konfiguriert, nicht angenommen** (Abschnitt `[layout]` im
Manifest). Der ausgelieferte Standard verwendet englische Namen; ein
bestehender Vault behält seine.

*Eingeschränkt seit Aufgabe 5 des Typkatalogs (2026-08-25):* Das gilt weiter
für die Vault-weiten Felder `inbox`, `sources` und `review` (§5.5) — nicht mehr
für die Ordner **innerhalb** eines Wiki-Bundles. Die oben gezeigten
`quellen/ themen/ entitaeten/ synthesen/` legt `init_bundle` nicht mehr im
Voraus an; ein solcher Ordner entsteht erst mit seiner ersten Seite, und wer
sie schreibt, bestimmt ihren Ort. Kein Manifest im Bestand hat die
Bundle-Ordner je umbenannt.

**Der oben gezeigte Baum ist der Vault dieses Nutzers, nicht die ausgelieferte
Vorgabe.** Seine deutschen Ordnernamen sind eine Nutzerentscheidung und stehen
hier, weil der Rest der Spec auf genau diese Pfade verweist. Das Werkzeug
liefert englische Namen aus.

Themencluster liegen unterhalb von `10 Rohquellen/`, damit die wichtigste
Grenze des Systems — von dir geschrieben gegen von der KI geschrieben — mit
einer Ordnergrenze zusammenfällt.

**Der Pfad ist das einzige verlässliche Unterscheidungsmerkmal.** Rohquellen
„bleiben frei" (§5.4) heißt *unbeschränkt*, nicht *ohne Frontmatter*: Ein
gewachsener Obsidian-Vault steckt voller handgeschriebener Frontmatter-Blöcke
mit Tags, Daten und Aliassen. Wer Rohquellen am fehlenden Frontmatter oder an
einem fehlenden `type` erkennen wollte, bricht am Altbestand. Die Zugehörigkeit
zur KI-Schicht wird deshalb **ausschließlich** über den Pfad entschieden.

Das Prüfzentrum liegt im Vault statt im Zustandsverzeichnis, damit offene Fälle
in Obsidian lesbar sind, bevor eine Oberfläche existiert. Es nimmt die Fälle
**aller** Bereiche auf, nach Bereich gruppiert — ein Ort für alles, was auf eine
Entscheidung wartet. Nach der Entscheidung verschwindet die Falldatei; der
Vorgang steht im `audit.md` des betroffenen Bereichs.

### 5.3 Projekt-Repo

Ein `readonly`-Code-Repo enthält **ausschließlich Rohquellen** — keine
KI-geschriebene Schicht, kein Prüfzentrum, kein Protokoll. Für ein
schreibbares Repo gilt das seit dem 2026-08-29 nicht mehr: dort darf das
Bundle des Bereichs im Baum selbst liegen („Das Projekt-Wiki zieht ins
Code-Repo", 2026-08-29).

```
<repo>/
  .brain.toml                      Manifest: scope, include/exclude, privacy,
                                   und — im schreibbaren Repo — die Stelle,
                                   an der das Wiki dieses Bereichs liegt
  README.md  CHANGELOG.md          Rohquellen: von Menschen geschrieben
  docs/
    entscheidungen/                Rohquellen: ADRs, Begründungen
    .superpowers/
      specs/  plans/               Rohquellen: der Vertrag
      sdd/                         ignoriert, nicht indexiert
    wiki/                          nur im schreibbaren Repo: das OKF-Bundle,
                                   von der KI geschrieben
  src/  tests/
```

Bei einem `readonly`-Repo liegt das Wiki des Projekts im Vault unter
`91 Projekte/<name>/` (§5.2), und verbunden wird beides allein über die zentrale
Registrierung (§5.5.1) — das Repo deklariert dann nur seinen Scope und was von
ihm indexiert wird. Ein schreibbares Repo nennt zusätzlich im Manifest, wo in
seinem Baum das Bundle liegt:

```toml
[area]
scope = "project/ultra-brain"
wiki  = true

# nur im schreibbaren Repo, repo-relativ:
[layout]
wiki = "docs/wiki"
```

Das sind keine zwei Wahrheiten, weil beide Verschiedenes sagen: **die
Registrierung nennt den Baum, das Manifest die Stelle darin.** Fehlt
`[layout] wiki`, bleibt es beim `wiki`-Eintrag der Registrierung — der nicht
in den Vault zeigen muss, es heute aber überall außer bei diesem Repo tut.

**Die Grenze lautet für `readonly`-Bereiche: Rohquellen im Repo, Verdichtung im
Vault.** Die Rohquellen eines Projekts sind seine menschengeschriebene Prosa —
README, CHANGELOG, ADRs, Spec und Plan. Quellcode gehört standardmäßig nicht
dazu (§5.5), und der Abgleich läuft nicht gegen ihn (§16.1). Im schreibbaren
Repo verläuft dieselbe Grenze eine Ebene tiefer: `docs/` ist deine Prosa,
`docs/wiki/` ist KI-geschrieben — dieselbe Konstruktion, die im Vault
`10 Rohquellen/` gegen `90 Wiki/` trägt.

Wer nur ein `readonly`-Repo hat, bekommt damit genau das, was ein Fremder lesen
soll: die Originale. Die Verdichtung ist persönliches Arbeitsmaterial und
bleibt es. **Bei einem schreibbaren Repo gilt dieser Satz nicht mehr**, und der
Preis ist offen zu benennen: ein Klon bekommt die Verdichtung mit. Bei einem
eigenen, nicht veröffentlichten Repo ist das der Zweck; bei einem
veröffentlichten eine Entscheidung, die getroffen werden muss, denn `brain
init` setzt `docs/wiki` für ein schreibbares Repo, ohne zu fragen (2026-08-29
§1.3).

### 5.4 OKF-Reichweite

**Das Wiki ist ein strenges OKF-v0.2-Bundle. Rohquellen bleiben frei.**

Verwendete OKF-Felder: `type` (Pflicht), `title`, `description`, `resource`,
`tags`, `sources`, `generated`, `verified`, `status`, `stale_after`, dazu
`index.md` und `log.md` auf jeder Ebene.

Seitentypen (Werte englisch, da maschinennah). Drei Ränge mit absteigender
Verbindlichkeit (Typkatalog-Design, 2026-08-25 — ersetzt die vier
Herkunftstypen dieser Fassung durch den vollen Katalog):

| Rang | Typen | Gilt |
|---|---|---|
| Kern | `Architecture`, `Decision`, `Open Question`, `Reference` | in jedem Projektwiki |
| Katalog | `API Endpoint`, `Data Model`, `Metric`, `Runbook`, `Glossary Entry` | optional, aber namensverbindlich — wer den Fall hat, nimmt den Katalognamen |
| Herkunft | `Source`, `Topic`, `Entity`, `Synthesis` | die zweite Achse: nicht *worüber* eine Seite handelt, sondern *wie sie entstanden ist* |

Dazu bereichseigene Typen, die ein Projekt im eigenen Manifest erklärt
(`[wiki] types = [...]`, §5.5) — frei, aber deklariert, damit ein Tippfehler
auffällt statt durchzugehen.

**Die Wahrheit führt der Code, nicht diese Tabelle.** `src/brain/wiki/types.py`
führt Kern, Katalog, Herkunft und die Aliastabelle der bekannten Altnamen; bei
einem Widerspruch gilt, was dort steht.

**Produzenteneigene Erweiterungen** (OKF §4.1 erlaubt sie ausdrücklich):

```yaml
sources:
  - id: nordstern
    resource: /raw/projekte/nordstern-status.md
    doc_id: 01J8F2K9XQ7M       # stabil, überlebt Umbenennen und Verschieben
    content_hash: sha256:9f2a… # Inhalt zum Zeitpunkt der Verdichtung
    revision: 4                # monoton wachsender Zählstand der Quelle
open_conflicts: 1              # Anzahl der Konfliktkästen auf dieser Seite

# nur auf Seiten in Projekt-Bundles:
realization: planned           # planned | in_progress | implemented | abandoned
implemented_in: a1b2c3d        # Commit oder PR; erst beim Landen gesetzt
```

Der Quellbezug `/raw/projekte/nordstern-status.md` ist absichtlich absolut
geschrieben — Rohquellen liegen außerhalb des Wiki-Bundles, das die Regel
liest. Seit Aufgabe 4 des Typkatalogs (2026-08-25) meldet `brain lint` einen
solchen Bezug als `absolute-link`-**Warnung**, nicht als Fehler: Die Regel
urteilt nicht „falsch", sondern „von hier nicht prüfbar" — sie kann nicht
nachsehen, ob die Datei unter `/raw/…` existiert, weil dieser Pfad außerhalb
dessen liegt, was die Regel je zu Gesicht bekommt.

`realization` schließt eine Lücke, die OKF nicht abdeckt: `status` beschreibt
die Reife der **Seite**, nicht den Umsetzungsgrad des **Beschriebenen**. Eine
Projektseite kann „so wollen wir es bauen" oder „so ist es gebaut" sagen — ohne
Unterscheidung liest sich eine Absichtserklärung von vor drei Monaten wie eine
Systembeschreibung. Genau diese fehlende Soll-Ist-Trennung war im
Referenzsystem eine ganze Klasse der markierten Konflikte.

Die maßgebliche Kopie dieses Zustands steht in der Frontmatter, also in Git.
Die Zustandsdatenbank des Daemons ist ein wegwerfbarer Beschleuniger und
jederzeit aus Frontmatter plus Dateisystem neu erzeugbar. Läge der Zustand nur
in einer Datenbank, würde ihr Verlust stillschweigend alle Seiten als aktuell
gelten lassen.

Konvertierte Dateien (§12) tragen einen dreizeiligen Herkunftskopf
(`source_url`, `retrieved`, `converter`). Das ist kein OKF und die einzige
Ausnahme von „Rohquellen bleiben frei"; ohne ihn ginge die Herkunft endgültig
verloren. Handgeschriebene Notizen bleiben unberührt.

### 5.5 Manifest

Eine Datei je Bereich, im Wurzelverzeichnis. Sie beschreibt **nur, was dieser
Bereich mitbringt** — nie, wo etwas anderes liegt.

```toml
[area]
scope = "project/ultra-brain"      # oder "knowledge", "engineering/python"
wiki  = true                       # hat dieser Bereich eine Verdichtungsschicht?
readonly = false                   # darf der Indexer in diesen Baum schreiben?

[layout]                           # `inbox`, `sources`, `review`, `hub`:
inbox = "00 Eingang"               # nur im Vault-Manifest
sources = "10 Rohquellen"
review = "95 Prüfzentrum"
wiki = "docs/wiki"                 # nur im Repo-Manifest eines
                                   # schreibbaren Bereichs (§5.3)

[index]                            # welche Rohquellen dieses Repos zählen
include = ["**/*.md"]              # genau ein Muster erreicht die Suchmaschine
unsearched = ["docs/.superpowers/**"]  # indiziert für catalog und read,
                                       # nicht Gegenstand der Suche (§5.3)
exclude = ["**/node_modules/**", "**/.git/**", "**/.obsidian/**",
           "**/.claude/**",            # Worktree-Kopien: byte-identische Zwillinge
           "**/.tools/**",             # Lizenztexte und Werkzeugballast
           "**/tests/fixtures/**",     # Testdaten, teils Kopien echter Notizen
           "**/.superpowers/sdd/**"]   # Arbeitsspuren; specs/ und plans/ sind Rohquellen

[privacy]
mode  = "local_only"               # local_only | manual_cloud | automatic_cloud
never = ["**/*.env", "**/secrets/**"]

[maintenance]
watch = true
stale_after_days = 90

[llm.local]
enabled = false                    # kann nur einschränken, nie erweitern
```

Standardvorgabe für `include` in einem Projekt-Repo: README, CHANGELOG,
Entscheidungsseiten, Spec und Plan. Quellcode und Sitzungsspuren sind
ausgeschlossen und nur bewusst zuschaltbar.

**`include` trägt genau ein Muster in die Suchmaschine.** qmd kennt pro
Sammlung ein einziges Glob; steht dort eine Liste, erreicht nur ihr erstes
Element die Suche, und alle weiteren zählen ausschließlich für den eigenen
Index — also für `catalog` und `read`. Ein Repo grenzt seinen Bestand deshalb
über `exclude` ein, nicht über eine Aufzählung in `include`: die Ausschlussliste
wird von beiden Indexen gelesen (Entscheidung 39), die Einschlussliste nicht.
Wer trotzdem mehrere Muster einträgt, bekommt keinen Fehler, sondern einen
stehenden Befund: `status` nennt pro Bereich das Muster, das die Suchmaschine
sieht, und die Muster, die sie nicht sieht.

**Spec und Plan sind indizierte Quellen für `catalog` und `read`, nicht für
`search`.** Sie stehen in `unsearched` (§5.5). Der Vertrag eines Projekts wird
gezielt gelesen — man weiß, dass man ihn braucht, und öffnet ihn —, nicht durch
Wiedererkennung gefunden. Die Suchleiter erreicht sie damit über ihre Stufen 1
und 4, nicht über Stufe 3, und das ist Absicht: eine Wissensfrage soll die
verdichtete Antwort finden, nicht das Planungsdokument, das dieselben Wörter
enthält.

**`never` wirkt doppelt.** Die frühere Fassung nannte es „kein Indexfilter,
sondern härter" und versprach zugleich, diese Pfade erreichten unter keinem
Modus ein Modell. Beides zusammen war nicht haltbar: Was im Index liegt, kommt
als Trefferausschnitt zurück. Richtig ist:

1. `never`-Pfade werden **nicht indexiert** — sie können in keinem Suchtreffer
   erscheinen.
2. `read` **verweigert** sie zusätzlich, auch bei direkter Pfadangabe — für den
   Fall, dass ein Pfad erst nachträglich unter `never` fällt und der Index noch
   nicht nachgezogen ist.

Ausschlüsse gewinnen immer gegen Einschlüsse.

**`unsearched` trennt „indiziert" von „durchsucht".** Nicht jede Quelle gehört
in die Bedeutungssuche. Der Vertrag eines Projekts — Spec und Plan — wird
gezielt gelesen, nicht durch Wiedererkennung gefunden; wer nach einem Sachverhalt
fragt, will die Wiki-Seite und nicht den Implementierungsplan vom Juli, der
dieselben Wörter enthält. Diese Pfade stehen deshalb im Katalog und im Graphen,
`read` gibt sie heraus, und die Suche kennt sie nicht.

Die Angabe ist zugleich die Antwort auf einen Befund der Abnahme: qmd betritt
Punktverzeichnisse **grundsätzlich nicht**, unabhängig von jeder Ausschlussliste.
Ohne `unsearched` stünden 61 Dokumente in unserem Index, die die Suche nie
zurückgeben kann, und `status` meldete sie als Rückstand — eine Meldung, die man
wegsieht, und weggesehene Meldungen verderben alle anderen. Mit ihr ist erklärt
statt erraten, warum die Suchmaschine sie nicht kennt, und was `status` dann noch
meldet, ist ein echter Rückstand. **Der Kern erfährt den Grund nie:** er
vergleicht zwei Mengen von Pfaden und zieht ab, was der Bereich selbst als
nicht durchsucht erklärt hat.

**Die eigenen Artefaktnamen werden nur im eigenen Baum ausgeschlossen.**
`index.md`, `index.intro.md`, `graph.json` und `_identities.tsv` fallen aus dem
Index, weil der Indexer sie selbst erzeugt und ein zweiter Lauf sonst läse, was
der erste schrieb. In einem `readonly`-Bereich schreiben wir nicht in den Baum —
dort kann keine Datei dieses Namens von uns stammen, und der Ausschluss ist
gegenstandslos. Er war es nicht nur: Er kostete in `iam_wiki` **22 von Hand
geschriebene, eingecheckte `index.md`** — in einem Wiki genau die
Navigationsebene. Die Standardausschlüsse zerfallen deshalb in zwei Listen: was
immer gilt (Punktverzeichnisse, Werkzeugballast, Testfixturen), und was nur für
einen Baum gilt, in den wir schreiben.

**`readonly` macht die Bestandsregel zu einer Eigenschaft des Bereichs.** Ein
fremder Bestand — indexiert, aber nicht unser — verträgt keine Schreiboperation,
auch keine wohlmeinende. Bisher stand dieser Schutz nur als Prosa in einer
CLAUDE.md, und die liest kein Programm. Mit `readonly = true` schreibt der
Indexer seine Artefakte nicht in den Baum, sondern unter
`<zustandsverzeichnis>/areas/<scope>/` (§5.8). Das gilt dann auch für das
Manifest selbst: `.brain.toml` läge sonst als erste verbotene Schreiboperation
im fremden Baum und wohnt deshalb ebenfalls dort. Die Registrierung verweist
darauf; sie ist ohnehin die einzige Stelle, die Bereiche auf Orte abbildet.

**Die Ausschlussliste hat genau eine Quelle.** `[index] exclude` gilt für beide
Indizes — den eigenen und den von qmd, dessen Sammlung `brain reindex` aus
diesem Abschnitt ableitet (§13). Zwei getrennt gepflegte Listen würden
auseinanderlaufen, und der Kern meldete das von da an als Dauerbefund. Die
Vorgabe schließt `docs/.superpowers/sdd/` aus, **nicht** den ganzen Ordner:
`specs/` und `plans/` sind Rohquellen — der Vertrag —, und ein System, das die
eigene Spec nicht findet, wäre eine Pointe.

### 5.5.1 Registrierung und Auflösung

Die zentrale Registrierung ist die **einzige Stelle, die Bereichsnamen auf
Pfade abbildet**:

```toml
# im Zustandsverzeichnis
[[area]]
scope = "knowledge"
path  = "C:/Users/micro/Documents/Wissen"

[[area]]
scope = "project/ultra-brain"
path  = "C:/Users/micro/Documents/#GIT/ultra-brain"      # Rohquellen
wiki  = "C:/Users/micro/Documents/#GIT/ultra-brain/docs/wiki"   # im Repo

[[area]]
scope = "engineering/python"
path  = "C:/Users/micro/Documents/Wissen/92 Engineering/python"
```

Damit steht der **Baum** des Wikis an genau einem Ort. Bei einem
`readonly`-Bereich ist damit alles gesagt: sein Manifest deklariert seinen
Scope und was von ihm indexiert werden soll; **wo** die Verdichtung liegt,
geht es nichts an.

**Bei einem schreibbaren Bereich nennt das Repo-Manifest zusätzlich die
Stelle im eigenen Baum** (`[layout] wiki`, §5.3, Entscheidung 48). Das sind
keine zwei Wahrheiten, weil beide Verschiedenes sagen: **die Registrierung
nennt den Baum, das Manifest die Stelle darin.** Der Grund für die Teilung
ist ein handfester — die Registrierung hält genau einen absoluten Pfad je
Bereich, und ein verknüpfter Worktree hat einen anderen; ein absoluter
Wiki-Pfad blockierte ausgerechnet das Schreiben im Zweig (2026-08-29 §5.1).

Beides ist die konkrete Form der Regel aus §5.1, dass Wiki-Ort und
Rohquellen-Ort zwei unabhängige Eingaben sind.

**Ein `wiki`-Eintrag steht bei jedem Bereich**, auch wenn Rohquellen und Wiki
im selben Baum liegen. Bei `knowledge` verlaufen sie durch denselben Baum, aber
zwischen `10 Rohquellen/` und `90 Wiki/` — genau dort liegt die wichtigste
Grenze des Systems, und die Schreibschranke sieht sie nur, wenn der Pfad
dasteht. Eine Ausnahme, die nur manchmal gilt, ist teurer als eine Redundanz,
die immer gilt (Verbund-Design §3.1).

**Verschachtelte Bereiche.** Der Vault enthält unter `91 Projekte/` und
`92 Engineering/` fremde Bereiche. Der Indexer schließt jeden registrierten
Bereich automatisch aus der Sammlung seines Elternverzeichnisses aus — ohne
dass jemand ein `exclude` pflegen muss. Die Zusicherung aus §6 („was nicht in
der Sammlung liegt, kann nicht auftauchen") hängt daran.

**Auflösung von `brain://`.** Die Form ist einheitlich
`brain://<scope>/<pfad-im-bereich>`. Da Scopes Schrägstriche enthalten
(`engineering/python`), gilt: **Es gewinnt die längste registrierte Scope-Angabe,
die auf den Anfang passt.** Der Rest ist der Pfad innerhalb des Bereichs. Eine
gesonderte Autorität `vault` gibt es nicht.

Pfadangaben in `sources[].resource` folgen derselben Regel, wenn sie einen
anderen Bereich meinen. Innerhalb desselben Bereichs gilt die OKF-Form
(bundle-relativ mit führendem `/`, oder relativ).

### 5.6 Identitätsregister

Beim Selbstcheck dieser Spec ist eine Lücke aufgefallen, die hier geschlossen
wird: `doc_id` muss Umbenennen und Verschieben überleben — aber Rohquellen
tragen kein Frontmatter (§5.4), können die Kennung also nicht selbst tragen.
Läge sie nur in der Zustandsdatenbank, ginge sie bei deren Neuaufbau verloren,
und jede Umbenennung sähe aus wie „Quelle gelöscht, neue Quelle erschienen" —
womit alle abgeleiteten Wiki-Seiten fälschlich auf `quelle_fehlt` liefen.

Deshalb gibt es je Bereich ein **Identitätsregister** als Textdatei im
versionierten Baum, zum Beispiel `90 Wiki/_identities.tsv`:

```
doc_id          pfad                                  letzter_hash    revision
01J8F2K9XQ7M    10 Rohquellen/projekte/nordstern.md   sha256:9f2a…    4
```

Eigenschaften:

- Es ist **Wahrheit, nicht Ableitung** — die einzige Ausnahme neben dem Markdown
  selbst, und deshalb versioniert, zeilenweise diffbar und nach `doc_id`
  sortiert (stabile Reihenfolge, §14).
- Es wird von Code geschrieben, nie von einem Modell.
- **Umbenennungen** erkennt der Abgleich über den Inhaltshash: Verschwindet ein
  Pfad und taucht gleichzeitig ein unbekannter Pfad mit demselben Hash auf, gilt
  das als Umbenennung — die `doc_id` bleibt, nur der Pfad wird fortgeschrieben,
  und kein Prüffall entsteht.
- **Gleichzeitiges Umbenennen und Ändern** lässt sich so nicht sicher erkennen.
  Dieser Fall wird als „Quelle fehlt" plus „neue Quelle" behandelt und im
  Prüfzentrum vorgelegt, statt geraten zu werden.
- Rohquellen bleiben unangetastet: Die Kennung liegt neben ihnen, nicht in
  ihnen.
- Das Register wird **bereichsübergreifend** gelesen: Läuft ein Verweis ins
  Leere, wird die `doc_id` über alle registrierten Bereiche gesucht (§5.7).

**Lebenszyklus einer `doc_id`:**

- **Vergeben** wird sie ausschließlich vom Code, beim ersten Indexlauf, in dem
  eine Datei auftaucht. Format: ULID — zeitlich sortierbar und ohne
  Absprache eindeutig.
- **Jede indexierte Datei bekommt eine**, Rohquellen wie Wiki-Seiten. Die
  Selbstheilung der Verweise (§5.7.3) setzt sie für Wiki-Seiten voraus.
- **Ein Modell vergibt nie eine.** In §9.2 Schritt 2 *liest* Claude die `doc_id`
  aus dem Ergebnis von `read`, das Frontmatter und Registereintrag mitliefert.
  Existiert für eine neue Datei noch keine, wird sie beim nächsten Indexlauf
  vergeben und der Ingest wartet darauf — eine Wiki-Seite ohne
  Quellenidentität wäre wertlos.
- **`content_hash` wird über auf LF normalisierten Inhalt gebildet**, nicht über
  die rohen Bytes der Datei. Grund: Schreibt Git die Zeilenenden je nach
  Plattform verschieden in den Arbeitsbaum, ändern sich die Bytes ohne
  inhaltliche Änderung — und die Wartungsschicht meldete nach einem Klon auf
  einem zweiten Rechner **jede einzelne Seite** als „Quelle geändert". Ein
  Schwall Prüffälle, von denen kein einziger echt ist, zerstört das Vertrauen in
  die Schicht schneller als jeder übersehene Widerspruch. Eine `.gitattributes`
  im Vault sichert dasselbe zusätzlich auf Git-Ebene ab — aber die Normalisierung
  beim Hashen ist die tragende Maßnahme, weil sie auch ohne Git greift.
- **`revision` wächst um genau 1 je festgestellter Inhaltsänderung**, also je
  Hashwechsel, den Wächter oder Abgleich feststellen — nicht je Ereignis und
  nicht je Lauf. Drei Änderungen zwischen zwei Abgleichen, die der Wächter
  verpasst hat, ergeben zusammen +1: Der Abgleich sieht nur den Unterschied
  zwischen zwei Ständen und behauptet nichts über den Weg dazwischen.

### 5.7 Geteilte Bereiche

#### 5.7.1 Drei Bereichsfamilien

Der Scope-Namensraum hat drei Familien. Der Kern kennt nur „Bereiche"; die
Familie steckt allein im Namen, es gibt keine Sonderbehandlung im Code.

| Familie | Beispiel | Inhalt |
|---|---|---|
| `knowledge` | `knowledge` | Allgemeinwissen, nicht technisch |
| `project/<name>` | `project/ultra-brain` | projektspezifisch, reist mit dem Code |
| `engineering/<name>` | `engineering/python` | technisch, projektübergreifend |

Die dritte Familie schließt eine Lücke: Programmierwissen entsteht in einem
Projekt, gilt aber für alle. Bleibt es im Projekt-Bundle, findet es niemand
wieder, sobald das Projekt ruht.

#### 5.7.2 Ablage

Im Vault unter `92 Engineering/`, ein Bundle je Bereich — jedes ein
vollwertiger Bereich mit eigenem Manifest, eigenem Wiki, eigenem
Identitätsregister.

```
Wissen/92 Engineering/
  python/
    .brain.toml                    scope = "engineering/python"
    _schema.md  index.md  log.md  audit.md  _identities.tsv
    themen/  entitaeten/  synthesen/
  frontend/
    .brain.toml                    scope = "engineering/frontend"
    …
```

Sie liegen damit dort, wo die vault-platzierten Verdichtungen liegen (§5.1)
— ein Vault, ein Prüfzentrum, eine Historie. (Nicht: ein Graph. `graph.json`
entsteht je Bereich, §5.8.) N Bereiche als N Repos hätten N Historien und N
Klonvorgänge zur Folge, und dieser Aufwand fiele sofort an, während der
Nutzen getrennter Repos erst entsteht, wenn ein Bereich tatsächlich
veröffentlicht wird.

**Herauslösen bei Veröffentlichung:** Soll ein geteilter Bereich als eigenes
Repo veröffentlicht werden, wird genau dieses Bundle herausgelöst und im
Register mit neuem Pfad geführt. Die Verweise darauf heilen sich über die
`doc_id` selbst (§5.7.3); es ist eine Verschiebeoperation, keine Reparatur.

**Startgröße: zwei Bereiche, nicht fünf.** Einer für die Sprache, in der
tatsächlich gebaut wird, einer für das übrige Handwerk (Werkzeuge, Prozess,
Architekturmuster). Ein dritter entsteht, wenn eine Seite spürbar in keinen der
beiden passt. Eine Aufteilung auf Vermutung ist teurer als eine spätere
Verschiebung.

#### 5.7.3 Bereichsübergreifende Verweise

Verweise tragen Pfad **und** Kennung — formal eine absolute URI und damit von
OKF §6.2 gedeckt:

```yaml
sources:
  - id: err-pattern
    resource: brain://engineering/python/themen/fehlerbehandlung.md
    doc_id: 01J9R4T2M8QK
    content_hash: sha256:…
```

Läuft der Pfad ins Leere, weil eine Seite den Bereich gewechselt hat, sucht der
Kern die `doc_id` über alle registrierten Bereiche und schreibt den Pfad fort.
**Verweise heilen sich damit selbst**, und ein Umschnitt der Bereichsgrenzen ist
eine Verschiebeoperation statt einer Reparaturaktion. Genau das entschärft das
Risiko, die Grenzen heute noch nicht zu kennen.

**Harte Regel:** Eine Aussage hat genau einen Heimatbereich. Ein Muster, das
zwei Bereiche betrifft, steht an einer Stelle; der andere verlinkt. Sonst
entstehen stille Doppelungen — genau das, wogegen das System gebaut ist.

#### 5.7.4 Beförderung

Adressiert wird über den Scope, nicht über einen Vault-Pfad: seit
Entscheidung 48 kann das Bundle eines Bereichs auch in dessen Repo liegen,
und nur die Registrierung weiß, wo (§5.5.1).

```
brain promote "brain://project/ultra-brain/docs/wiki/topics/fehlerbehandlung.md" \
              --to engineering/python
```

Ablauf: Das Modell schlägt eine projektfreie Fassung vor — Namen, Pfade und
Zahlen des konkreten Falls entfernt —, du gibst frei, die Projektseite verweist
danach auf die geteilte. **Die geteilte Seite ist die Wahrheit; es wird nicht
kopiert.** Eine Kopie driftet, und in einem System, dessen Zweck das
Sichtbarmachen von Widersprüchen ist, wäre das die schlechteste Lösung.

Das Entschärfen ist keine Formsache: Geteilte Bereiche sind die einzigen, die
später einem Team oder der Öffentlichkeit gezeigt werden könnten, und dürfen
nichts Projektinternes tragen.

Ohne `--to` verweigert der Befehl die Arbeit und listet die verfügbaren
Bereiche. Ein Vorschlag für den Zielbereich darf vom Modell kommen, die Wahl
nicht — ein falsch einsortiertes Muster findet später niemand.

#### 5.7.5 Leserichtung

Projekte lesen geteilte Bereiche. **Geteilte Bereiche lesen keine Projekte.**
Sonst wanderte Projektinternes durch die Hintertür in die geteilte Schicht. Der
einzige Weg hinein ist die Beförderung, und die geht durch menschliche Hand.

#### 5.7.6 Geschenkter Effekt

Verweist eine Projektseite auf eine geteilte Seite, steht das als
`sources`-Eintrag mit `content_hash` — womit **Brain Maintenance ohne jede
Erweiterung über Bereichsgrenzen hinweg greift**: Ändert sich das geteilte
Muster, werden alle Projektseiten, die darauf beruhen, zu Prüffällen. Sie
erscheinen im Prüfzentrum unter ihrem jeweiligen Bereich und werden einzeln
freigegeben.

---

### 5.8 Abgeleitete Artefakte je Bereich

Drei Dateien, alle vom Indexer erzeugt, alle byteweise wiederholbar. Sie liegen
im Baum des Bereichs — bei `readonly = true` stattdessen unter
`<zustandsverzeichnis>/areas/<scope>/`. Wer sie liest, findet sie über die
Registrierung, nicht über eine Pfadannahme.

| Datei | Inhalt | Wegwerfbar? |
|---|---|---|
| `index.md` je Ebene | Katalog: Bereiche und Dateien dieser Ebene | ja |
| `graph.json` je Bereich | `nodes`, `edges`, `links` | ja |
| `_identities.tsv` je Bereich | `doc_id`, Pfad, Inhaltshash, Revision | **nein** — trägt Identität |

**`edges` ist eine Menge von Beziehungen.** Zwei Schreibweisen können dieselbe
Kante meinen; sie erscheint einmal. Das steht hier, weil ein Verbraucher es hier
liest — im Erzeuger stünde es dort, wo es niemand sucht.

**`links` beziffert, was die Kantenbildung nicht geschafft hat:** wie viele
Verweise gefunden, wie viele aufgelöst, wie viele je Grund verworfen wurden. Der
Grund ist eine Messung: Fällt die Wurzel des Bereichs nicht mit der Wurzel
zusammen, auf die die Notizen ihre Links beziehen, wird **jeder** Verweis
verworfen — im Rauchtest von Scheibe 1 waren das 0 statt 959 Kanten, und nichts
an der Ausgabe sagte es. Ein Graph ohne Kanten muss von einem Bestand ohne Links
unterscheidbar sein; `status` meldet eine auffällig niedrige Quote.

**`index.intro.md`, optional, je Ebene.** Der Katalog ist die erste Stufe der
Suchleiter und wird bei jeder Wissensfrage gelesen; was dort fehlt, fehlt
überall. Weil `index.md` dem Generator gehört, trägt eine getrennte Datei die
handgeschriebene Einleitung: Der Indexer kopiert sie unverändert zwischen
Überschrift und Bereichsliste und indexiert sie nicht. Fehlt sie, sieht der
Katalog aus wie ohne sie — keine leere Lücke. Vorhanden und leer ist ein Fehler.
Kein Markerformat innerhalb von `index.md`: Der Generator müsste dann seine
eigene Ausgabe zerlegen, und ein zerstörter Marker kostet den Text.

---

## 6. Scoping

Jede Anfrage trifft eine **Liste** von Bereichen und eine Schicht.

| Angabe | Wirkung |
|---|---|
| kein Scope, Aufruf innerhalb eines Bereichs | dieser Bereich plus die im Manifest genannten geteilten Bereiche |
| kein Scope, Aufruf außerhalb | `knowledge` |
| `--scope <name>` (mehrfach angebbar) | genau diese Bereiche |
| `--scope all` | alle registrierten Bereiche |
| `--layer raw` \| `wiki` \| `both` | Rohquellen, Verdichtung, beides |

Der Vorgabe-Scope eines Projekts steht in dessen `.mcp.json` und ist eine Liste:

```json
{ "scope": ["project/ultra-brain", "engineering/python"] }
```

Damit sieht Claude dort Projektwissen plus die geteilten Bereiche, die zu diesem
Projekt passen — und ausdrücklich nicht die übrigen. **Die Körnung der geteilten
Bereiche ist die Körnung der Abgrenzung** (§5.7).

Es bleibt eine Vorgabe, keine harte Grenze: Ein LLM kann bewusst weiten, soll
das Weiten aber benennen (§8).

Technisch: **eine Sammlung je Bereich und Schicht.** Der Scope wählt Sammlungen
aus, statt Treffer nachträglich auszusortieren. Was nicht in der Sammlung liegt,
kann nicht versehentlich in einer Antwort auftauchen.

---

## 7. Der Kern

### 7.1 Prozessmodell

Ein langlebiger Daemon (Python 3.14) hält Index und Modelle im Speicher.
Darauf setzen dünne Klienten auf: ein kompilierter CLI-Client und ein
zustandsloser MCP-Adapter, beide über eine Named Pipe beziehungsweise einen
Unix-Socket.

Der Daemon wird **nicht** als Systemdienst eingerichtet. Der erste Klient, der
keine Pipe vorfindet, startet ihn und wartet auf Bereitschaft.

**Unter Windows trägt „startet ihn" nicht ohne Umweg.** Ein MCP-Wirt legt für
den Front-Prozess ein Job-Objekt mit `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` an
und beendet am Sitzungsende den ganzen Baum — also auch den Daemon und den
qmd-Daemon darunter. Ein Daemon, den die Front als Kind startet, überlebt die
Sitzung nicht, und jede neue Sitzung zahlt den kalten Modellstart erneut: 16 s
statt 0,8 s. `CREATE_BREAKAWAY_FROM_JOB` hilft nicht, das Job-Objekt trägt kein
`BREAKAWAY_OK`. Der Klient startet den Daemon deshalb über WMI
(`Win32_Process.Create`), womit der Prozess vom WMI-Anbieter erzeugt wird und
keinem Job angehört. Der Grundsatz bleibt: **kein Dienst, kein Autostart** —
der Daemon entsteht weiterhin auf Zuruf des ersten Klienten und gehört dem
angemeldeten Benutzer.
→ [`bench/2c3/daemon-lifetime.md`](../../../bench/2c3/daemon-lifetime.md)

Der MCP-Adapter existiert, weil Claude Code je Sitzung und Projekt einen eigenen
MCP-Prozess startet. Ohne Trennung hielte jede Sitzung eigenen Index und eigenes
Modell im Speicher.

### 7.2 Werkzeuge für das LLM — schreibfrei

| Werkzeug | Zweck |
|---|---|
| `catalog` | Wurzel- oder Bereichskatalog |
| `search` | hybride Suche; `scope`, `layer`, `profile`, `n`. Unterscheidet **leeres Ergebnis von fehlgeschlagener Suche** (§16.14). Die frühere Zusage „je Inhaltshash entdoppelt" ist nach der Abnahme von Scheibe 2a zurückgenommen (§16.12) |
| `read` | genau eine Datei, wahlweise nur ein Abschnitt. Meldet **fehlgeschlagenes Lesen ausdrücklich**, nie als leeren Inhalt (§16.14) |
| `neighbors` | ein- und ausgehende Links einer Seite |
| `status` | Indexstand, offene Fälle, abgelaufene Seiten |

**Eine Anforderung an `search` und `read`, die aus Messungen stammt und nicht
verhandelbar ist** — es waren einmal zwei; die Entdopplung ist nach der Abnahme
von Scheibe 2a zurückgenommen (§16.12) und durch eine Meldung in `status`
ersetzt:

1. **Leeres Ergebnis ist nicht dasselbe wie fehlgeschlagene Suche** (§16.14).
   qmd bricht gelegentlich still ab und liefert einen leeren Treffersatz — ein
   Fehler, der sich als Ergebnis tarnt. `search` wiederholt bei leerem Ergebnis
   einmal und meldet, wenn auch die Wiederholung leer bleibt, ausdrücklich einen
   **Fehler** statt einer stillen Null. Für `read` gilt dasselbe: ein
   fehlgeschlagener Lesevorgang wird gemeldet, nicht als leerer Inhalt
   zurückgegeben.

### 7.2.1 Datenschutz gilt auch beim Lesen

Die Gatter aus §10.5 wirkten ursprünglich nur auf dem Wartungspfad. Damit war
`local_only` im Alltag wirkungslos: Claude — ein Cloud-Modell — hätte denselben
Inhalt über `search` und `read` vollständig lesen können. Deshalb setzt der
Kern den Datenschutzmodus **auf jedem ausgehenden Pfad** durch, nicht nur beim
Vorschlaggeber:

| Modus | Kanal `cloud` (MCP; CLI auf Wunsch) | Kanal `local` (CLI, Web-App, lokales Modell) |
|---|---|---|
| `local_only` | **Bereich ist unsichtbar** — keine Treffer, keine Inhalte, kein Katalogeintrag | voller Zugriff |
| `manual_cloud` | voller Zugriff (das Gatter greift beim Vorschlaggeber, §10.5) | voller Zugriff |
| `automatic_cloud` | voller Zugriff | voller Zugriff |

`local_only` bedeutet damit wörtlich, was es sagt: Der Bereich existiert für das
Cloud-Modell nicht. Das ist die strengste und die einzige testbare Auslegung —
und der Preis ist bewusst gewählt: In einem solchen Bereich kann Claude Code
nicht arbeiten. Dort arbeitest du mit CLI, Oberfläche und dem lokalen Modell.

Der Unterschied zu `never` (§5.5): `never` schließt einzelne **Pfade** überall
aus, auch innerhalb eines sonst offenen Bereichs. `local_only` schließt einen
**ganzen Bereich** gegenüber der Cloud.

**Der Kanal ist ein Parameter des Kerns, kein Verhalten des Adapters.** Jede
der fünf Werkzeugfunktionen nimmt ihn entgegen (`local` oder `cloud`), und das
Gatter greift dort — nicht im MCP-Adapter. Sonst könnte ein zweiter Klient es
vergessen, und der Beweis hinge an der Sorgfalt des jeweils äußersten Randes.
So erbt jeder künftige Zugang die Zusage, statt sie nachzubauen. Auf der CLI ist
der Kanal setzbar, damit der Nachweis ohne MCP-Prozess führbar ist.

Prüfbar ist beides ohne Modell: Ein Bereich auf `local_only`, ein `search` über
`all` auf dem Cloud-Kanal — die Treffer dürfen daraus nichts enthalten, und
`read` auf einen bekannten Pfad daraus muss verweigern. Ein Bereich wird dabei
gar nicht erst befragt, statt seine Treffer nachträglich zu filtern.

Es gibt bewusst kein `approve` und kein `ingest` über MCP. Freigeben ist eine
menschliche Entscheidung. Ingest braucht kein Werkzeug: Claude schreibt
Wiki-Seiten als normale Dateien, geführt von `_schema.md`; der Wächter nimmt sie
von dort auf. **Ein LLM kann über diese Schnittstelle nichts am Wissen ändern.**

### 7.3 Drei Suchprofile

| Profil | Kette | Vorgabe für |
|---|---|---|
| `keyword` | BM25, kein Modell | — (auf Wunsch wählbar) |
| `fast` | rein vektoriell: Embedding, **keine** Frageerweiterung, **kein** Reranker | **überall**: CLI, Suchbox der App, MCP |
| `full` | hybrid: Frageerweiterung + Embedding + Reranking | — (auf Wunsch wählbar) |

**Gemessen in Scheibe 2b, und zweimal gedreht.** Hier standen nacheinander zwei
Vorgaben, beide auf zu dünner Grundlage: erst `fast` aus einem Kostenargument,
dann `full` nach der Abnahme von Scheibe 2a — begründet mit zwei Prüffragen.
Gemessen wurde erst an dreißig Fragen, dann am erweiterten Satz aus fünfzig:

| Profil | exakt | Umschreibung | gemischt | sprachübergreifend | gesamt |
|---|---|---|---|---|---|
| `fast`, 30 Fragen (4 Läufe) | 7/8 | 5/8 | 4/6 | **8/8** | **24/30**, dreimal 24 |
| `full`, 30 Fragen (4 Läufe) | 6–7/8 | 5/8 | 4/6 | **8/8** | **22/30**, einmal 23 |
| `fast`, 50 Fragen (3 Läufe) | 11/13 | 8/13 | 8/10 | **13/14** | **39–40/50** |
| `full`, 50 Fragen (1 Lauf) | 10/13 | 7/13 | 8/10 | **14/14** | **39/50** |
| `keyword`, 30 Fragen | 7/8 | 1/8 | 1/6 | **0/8** | 9/30 |

`fast` liegt in jedem vergleichbaren Lauf vorn oder gleichauf: bei dreißig
Fragen um zwei Treffer, bei fünfzig um einen. Der Abstand schrumpft mit dem
größeren Satz — was bleibt, ist, dass der Reranker das Ergebnis **nicht
verbessert**, während er kalt die doppelte Zeit kostet (2792 ms gegen 5852 ms).
Eine Ausnahme ist redlich zu nennen: sprachübergreifend nimmt `full` 14/14
gegen 13/14, dort hilft der Reranker um einen Treffer.

**Der Reranker ist auf diesem Rechner instabil.** Vier von neun `full`-Läufen
sind mitten im Lauf abgebrochen — qmd stirbt beim Reranking mit einem
CUDA-Fehler aus `ggml-cuda.cu`. Kein einziger rein vektorieller Lauf ist
abgestürzt. Das ist kein Argument über Trefferqualität, aber es ist ein
Argument über die Vorgabe: ein Profil, das in knapp der Hälfte der Fälle keine
Antwort liefert, kann nicht der Weg sein, den jede Suche standardmäßig nimmt.
Die Ursache liegt außerhalb dieses Projekts; die Folge nicht.

`full` bleibt wählbar. Es ist die einzige Kette mit Stichwortanteil, und für
Fragen, die auf einem seltenen Bezeichner beruhen, kann das der bessere Weg
sein — belegt ist das nicht, und wer es behauptet, misst es.

**Das frühere `fast` gibt es nicht.** Hier stand „Frageerweiterung + Embedding,
kein Reranker". Diese Kette existiert bei qmd 2.8.3 nicht: die Suchmaschine
kennt `query` (Erweiterung **und** Reranker), `vsearch` (rein vektoriell) und
`search` (BM25) — nichts dazwischen. Die Definition war eine Annahme über ein
fremdes Werkzeug, die nie geprüft wurde. Damit fällt auch das vierte Profil
`vector`: es rief denselben Befehl auf wie `fast`, war also ein zweiter Name
für dieselbe Sache.

### 7.4 CLI

```
brain search "<query>"   --scope … --layer … --profile … -n 5
brain catalog            [area]
brain read <path>        [--section "<heading>"]
brain status
brain cases | brain case <id>
brain approve <id>       [--amend <file>] | --reject | --defer
brain promote <path>     --to <area>
brain hook               install | status | remove
brain lint
brain bench              [--latency]
brain reindex            [--full]
brain embed
brain reconcile
brain init
brain daemon             start | stop | status
```

`brain case` ist zugleich der Bestätigungsschritt für `manual_cloud`: Das
konkrete Paket wird gezeigt, bevor etwas übertragen wird.

**`brain embed` ist ein eigener Befehl, weil `reindex` den Volltextindex
fortschreibt und die Vektoren nicht.** Das ist keine Bequemlichkeit, sondern eine
gemessene Eigenschaft der Suchmaschine: nach einem Aktualisierungslauf sind neue
Dokumente über die Stichwortsuche erreichbar, über die Bedeutungssuche nicht.
Ein Dokument ohne Vektoren ist **sprachübergreifend unauffindbar** — und das ist
die Eigenschaft, für die §16.3 den ganzen Aufwand rechtfertigt. Die beiden
Läufe zu trennen ist die richtige Wahl: Einbetten lädt Modelle und dauert
Minuten, `reindex` ruft man häufig. `status` nennt den Rückstand und verweist auf
den Befehl; wer die Meldung übersieht, sucht sprachübergreifend in einem halben
Index, und dann steht es wenigstens irgendwo.

### 7.5 Verhalten bei Störungen

| Lage | Verhalten |
|---|---|
| Daemon läuft nicht | Klient startet ihn, meldet die Wartezeit |
| Index veraltet | Treffer werden geliefert, sichtbar als veraltet markiert |
| qmd fehlt oder bricht ab | `catalog` und `read` funktionieren; `search` meldet den Ausfall |
| qmd fehlt oder bricht ab, `status` läuft | `status` nennt den Ausfall je Bereich, vergleicht die beiden Indizes nicht, führt seine übrigen Prüfungen zu Ende und endet mit 0 |
| Bereich registriert, Pfad fehlt | Bereich wird übersprungen, `status` nennt ihn |
| Zustandsdatenbank beschädigt | wird verworfen und neu gebaut |

### 7.6 Skills — die Abläufe

Werkzeuge holen Daten, das Regelwerk ordnet die Reihenfolge. **Skills sind die
mehrschrittigen Verfahren dazwischen.**

Ausgeliefert wird alles als **Claude-Code-Plugin**: MCP-Server, Skills,
Git-Hooks und CLI in einem Paket — dasselbe Muster, mit dem qmd ausgeliefert
wird. Skills sind Prosa-Dateien mit Schritten und Prüfpunkten, nach außen
englisch, vom Nutzer änderbar, ohne den Server anzufassen.

**Ein Skill hat keine Sonderrechte.** Er ist eine Arbeitsanweisung für Claude,
kein Umgehungsweg: Die MCP-Werkzeuge bleiben schreibfrei (§7.2), und ein Skill
kann keinen Weg beschreiten, den ein Mensch am selben Ort nicht auch hätte. Ein
Skill macht Abläufe verlässlich, nicht mächtiger.

**Es gibt genau zwei Schreibwege ins Wiki, und sie sind verschieden gesichert:**

| Weg | Wer schreibt | Sicherung |
|---|---|---|
| **Ingest** (§9.2) | Claude, als gewöhnliche Dateien | menschliche Begleitung im Ablauf (Entscheidung 11) plus Git-Historie; **kein** Prüffall |
| **Wartung** (§10.5) | Code, nach Freigabe | Prüffall, Evidenzbindung, Freigabe, Protokoll, isolierter Commit |

Der Unterschied ist beabsichtigt. Ingest ist der Vorgang, bei dem Wissen
überhaupt erst entsteht — ihn über Fälle zu führen hieße, sich jede einzelne
Seite selbst zu genehmigen. Wartung dagegen ändert Bestehendes, oft gegen den
Wortlaut einer bereits geprüften Aussage; dort ist die Freigabe der Kern.

Was in **beiden** Fällen gilt: `10 Rohquellen/` und die Rohquellen der
Code-Repos bleiben unangetastet, und jede Änderung steht in `log.md`.

| Skill | Wann | Was er tut |
|---|---|---|
| `brain:research` | ein Thema durchdringen | Suchleiter systematisch über die Scopes; Belege mit Quellenangabe; **ausdrücklich benennen, was nicht gefunden wurde**; Angebot, das Ergebnis als `Synthesis` (`status: draft`) abzulegen |
| `brain:ingest` | eine Quelle liegt bereit | der Ablauf aus §9.2, in jeder Sitzung gleich |
| `brain:wiki-plan` | eine Arbeit ist fertig | ermittelt, welche Seiten nachzuziehen sind — **schreibt nichts**; Ergebnis wird ein Fall |
| `brain:land` | nach Merge nach `main` | Soll auf Ist: Wortlaut anpassen, `realization: implemented`, `implemented_in` setzen |

Das Ablegen von Rechercheergebnissen ist kein Zusatz, sondern schließt eine
Lücke: Ohne es verschwindet eine gute Antwort im Gesprächsverlauf und wird in
drei Wochen erneut bezahlt.

`wiki-plan` und `land` sind getrennt, weil zwischen „fertig" und „in `main`" ein
Review, ein Umbau oder ein Verwerfen liegen kann. Ein Wiki, das
„implementiert" sagt, während der PR noch offen ist, wäre schlechter als gar
keine Angabe.

---

## 8. Regelwerk (Suchleiter)

Die Reihenfolge steht als Prosa in der CLAUDE.md, nicht im Code.

1. `catalog` lesen — Wurzelkatalog, bei Bedarf genau einen Bereichskatalog.
2. `search` mit `layer=wiki` — liegt das Wissen schon verdichtet vor?
3. `search` mit `layer=raw` — erst jetzt über die Originale; Kandidaten anhand
   der Trefferliste beurteilen, ohne Dateien zu öffnen.
4. `read` — genau EINE Datei, darin nur den relevanten Abschnitt.
5. Antworten, mit Quellenangabe.

Grenzen: höchstens zwei Suchläufe je Frage; ein hoher Relevanzwert ist kein
Beweis; Scope nur bewusst weiten und das Weiten benennen; Widersprüche zitieren,
nicht auflösen.

Die Zwei-Suchlauf-Grenze ist die Antwort auf den Kostentreiber: Jeder weitere
Suchschritt liest den bisherigen Gesprächsverlauf erneut, sodass die Kosten von
Runde zu Runde wachsen. Ein Abbruchkriterium ist billiger als jede Optimierung
an der Suche.

**Stufe 2 heißt vorerst lesen, nicht suchen.** Die tatsächliche Größenordnung
liegt bei rund 290 Markdown-Notizen über drei Bestände (§16.11); der Rohbestand
passt damit nicht ins Kontextfenster, das daraus verdichtete Wiki mit hoher
Wahrscheinlichkeit schon — das Referenzsystem verdichtete knapp 2.000 Notizen
auf rund 40 Seiten unter 100.000 Token, und dieses Wiki wächst zudem nur mit
neuen Quellen (Entscheidung 13).

Solange das Wiki unter der Schwelle bleibt, liest Claude auf Stufe 2 den
Wiki-Katalog und daraus die einschlägigen Seiten. Überschreitet es sie, wird
Stufe 2 zur Suche über die Wiki-Sammlung — der Mechanismus dafür existiert
ohnehin (§6, `layer=wiki`). **Die Schwelle wird gemessen, nicht geschätzt:**
`brain status` weist die Tokenmenge der Wiki-Schicht je Bereich aus und meldet,
wenn sie 100.000 überschreitet.

Der Wurzelkatalog nennt trotzdem Bereiche statt Dateien und jeder Bereich trägt
seinen eigenen `index.md` — nicht wegen der Menge, sondern weil es mehrere
Bereiche gibt und ein Katalog, der Projekte und geteiltes Wissen vermischt,
seinen Zweck verfehlt.

---

## 9. Wiki-Schicht

### 9.1 Schema

`_schema.md` liegt im Wiki-Ordner und wird vor jeder Wiki-Arbeit gelesen.
Sechs Regeln: nur die KI schreibt hier; Rohquellen sind unantastbar; jede Seite
ist vernetzt; Widersprüche werden markiert, nie aufgelöst; jede Änderung endet
mit einem Log-Eintrag; verdichtet wird immer gegen die Originalquelle, nie gegen
eine ältere Zusammenfassung.

### 9.2 Ingest

```
1. _schema.md lesen
2. Quelle lesen; doc_id, content_hash, revision feststellen
3. Betroffene Seiten über den Wiki-Katalog ermitteln
4. Genau diese Seiten öffnen — nie das ganze Wiki
5. Source-Seite schreiben; Topic- und Entity-Seiten aktualisieren,
   fehlende anlegen, Verweise setzen
6. Je Seite prüfen: passt die Aussage → einarbeiten;
                    widerspricht sie  → markieren, nichts überschreiben
7. sources[] mit doc_id, content_hash, revision festschreiben
8. Eintrag in log.md
```

Schritt 3 ist der Grund, warum Ingest bei großem Bestand bezahlbar bleibt.

Betrieb nach Bereich getrennt: `knowledge` Quelle für Quelle mit menschlicher
Begleitung; Projekt-Wikis auf Zuruf beziehungsweise stapelweise nach größeren
Schritten. Der Altbestand wird **nicht** vollständig verdichtet — nur neue
Quellen sowie einzelne Themen auf Zuruf.

### 9.3 Konflikte

Feste, maschinell auffindbare Form:

```markdown
> [!conflict] <Kurztitel>
> [Quelle A](/pfad/a.md) sagt X.
> [Quelle B](/pfad/b.md) sagt Y.
> Beide Stände bleiben stehen. Entscheidung offen.
```

Dazu `open_conflicts: <n>` in der Frontmatter. Der Lint vergleicht die Zahl mit
den tatsächlich vorhandenen Kästen — die Buchführung der KI wird von Code
kontrolliert.

**Auflösung:** Es wird die **Quelle** repariert, nicht der Kasten. Wird nur die
Markierung gelöscht, findet der nächste Durchlauf denselben Konflikt wieder,
weil sich die Dateien weiterhin widersprechen. Vier zulässige Urteile: alter
Stand veraltet; beides gilt in verschiedenem Kontext; neue Information ist
falsch; bewusst offen lassen.

### 9.4 Lint

`brain lint` läuft deterministisch, ohne Modell: fehlendes `type`; Seite ohne
`sources`; verwaiste Seite; toter Link; `open_conflicts` stimmt nicht; lange
unberührte Seite; abgelaufenes `stale_after`.

In Projekt-Bundles kommen zwei Prüfungen dazu, die erst durch `realization`
(§5.4) möglich werden: Seiten, die seit langem `planned` sind, und Seiten, die
`implemented` melden, ohne einen Commit in `implemented_in` zu nennen. Beides
fiel vorher nur auf, wenn jemand zufällig zwei Dateien nebeneinander las.

**Lint prüft die Struktur, Brain Maintenance die fachliche Aktualität.** Eine
gestern geschriebene Seite kann überholt sein, eine hundert Tage alte weiterhin
stimmen; deshalb ist der Inhalts-Hash das starke Signal und das Datum nur ein
Hinweis.

Genau eine Lint-Frage braucht Verstehen und bleibt eine Aufgabe für Claude:
Welche Begriffe tauchen häufig auf, haben aber keine eigene Seite?

---

## 10. Brain Maintenance

### 10.1 Das Problem

Zwei Arten von Aktualität werden verwechselt:

| | Technische Aktualität | Fachliche Aktualität |
|---|---|---|
| Frage | Kennt der Index die aktuelle Datei? | Passt das abgeleitete Wissen noch zur Quelle? |
| Zuständig | Indexlauf | Brain Maintenance |

Beide Stände sind auffindbar, und der veraltete sieht besser aus — verdichtet,
verlinkt, mit Quellenangabe. Ein Bearbeitungsdatum beweist keine inhaltliche
Aktualität; ein Inhalts-Hash macht Änderungen beweisbar sichtbar.

### 10.2 Ableitungskante

Der Rückwärtsindex Quelle → abhängige Seiten wird deterministisch aus den
`sources`-Einträgen der Wiki-Seiten gebaut. Kein zusätzliches Format.

### 10.3 Erkennung

- **Wächter** meldet Änderungen im Betrieb, entprellt bis der Schreibvorgang
  stabil ist. Bequem, aber prinzipiell unzuverlässig.
- **Täglicher Sicherheitsabgleich** ist die Wahrheit und fängt, was der Wächter
  verpasst hat. Zweistufig: Zeitstempel und Größe als Vorfilter, Hash nur für
  Verdächtige.

**Wer löst „täglich" aus?** Es gibt keinen Systemdienst (§15), und der Daemon
startet erst auf Klientenanfrage — ohne Festlegung liefe eine Woche ohne
Klienten auch eine Woche ohne Abgleich. Deshalb: **Der Daemon holt beim Start
nach.** Liegt der letzte vollständige Abgleich länger als 24 Stunden zurück,
läuft er unmittelbar, bevor die erste Anfrage beantwortet wird — und danach
alle 24 Stunden, solange der Prozess lebt. `status` nennt den Zeitpunkt des
letzten Laufs; ist er alt, steht das in jeder Antwort. Damit hängt die
Aktualität an der Benutzung statt an einem Zeitplan, den niemand überwacht.

Der Abgleich erzeugt keine Änderungen, sondern Fälle.

### 10.4 Zustände einer Wiki-Seite

`aktuell` · `quelle_geändert` · `fällig` (`stale_after` erreicht) ·
`quelle_fehlt` · `in_prüfung`

Zeit- und änderungsbasierte Fälligkeit landen in derselben Warteschlange, aber
getrennt gekennzeichnet und unterschiedlich gewichtet.

**Gleichzeitigkeit.** Drei Fälle, die eintreten werden und deshalb festgelegt
sind statt geraten:

- **Die Quelle ändert sich erneut, während ein Fall `in_prüfung` liegt.** Der
  offene Fall wird verworfen und neu gebildet — mit dem aktuellen Stand. Ein
  Vorschlag zu einem überholten Quellstand ist wertlos, und ihn stehen zu
  lassen hieße, dich über etwas entscheiden zu lassen, das es nicht mehr gibt.
  Ein bereits geschriebener Vorschlag bleibt im Fall vermerkt.
- **Die Wiki-Seite wird von Hand geändert, während ein Fall offen ist** (etwa
  in Obsidian). Beim Schreiben nach der Freigabe prüft der Code den Hash der
  Zielseite gegen den Stand, auf dem der Vorschlag beruht. Weicht er ab, wird
  **nicht geschrieben**; der Fall geht zurück in die Warteschlange mit dem
  Hinweis, dass sich die Seite unter ihm bewegt hat.
- **Mehrere Quellen einer Seite ändern sich.** Ein Fall je Seite, nicht je
  Quelle — die Entscheidung betrifft den Text der Seite, und drei Fälle zu
  derselben Seite würden einander widersprechende Vorschläge erzeugen. Das
  Analysepaket führt dann alle geänderten Quellen.

### 10.5 Ablauf

```
Änderung erkannt
   │
   ├─► TECHNISCHER WEG (immer, sofort, ohne KI)
   │   qmd-Index · graph.json · index.md nachführen
   │   → die Suche kennt den aktuellen Dateistand, auch während
   │     eine fachliche Entscheidung offen ist
   │
   ▼
Eignungsprüfung (Code) ── ungeeignet ─► Ende: technisch aktuell, kein Prüffall
   │   Binärdateien, Übergrößen, reine Umformatierungen fallen heraus
   ▼
Datenschutzprüfung (Code)
   │   local_only      → lokales Modell, sonst manueller Fall
   │   manual_cloud    → Paket anzeigen, Bestätigung abwarten
   │   automatic_cloud → nur regelkonforme Pakete automatisch
   ▼
Analysepaket (Code): Quellendiff · betroffene Seite · sources ·
                     verlinkte Nachbarn als Kontext, nicht als Änderungsziel
   ▼
Prüfvorschlag (Modell) — schreibt nichts ins Wiki
   ▼
Evidenzbindung (Code): jede Aussage zeigt auf das Paket, sonst verworfen
   ▼
PRÜFZENTRUM → freigeben │ angepasst freigeben │ ablehnen │ zurückstellen
   ▼ (nur nach Freigabe)
Wiki schreiben · content_hash und revision fortschreiben · generated.at setzen ·
verified-Eintrag mit human:<id> (Trust-Tier „human-reviewed") ·
Wiki-Log · Prüfprotokoll · erneute technische Aktualisierung ·
isolierter Git-Commit im Vault-Repo, mit ausschließlich den Dateien
dieser einen Wissensänderung — das Code-Repo bleibt unberührt
```

### 10.6 Prüfvorschlag

Feste Form: was sich geändert hat (mit Fundstelle); welche Stelle der Wiki-Seite
dem widerspricht; vorgeschlagene Fassung als Diff; Einordnung (veraltet /
beides gilt in verschiedenem Kontext / Neuerung zweifelhaft); Zuversicht und
benannte Restunsicherheit.

Der Vorschlag ist eine Datei neben dem Fall, kein Zustand im Wiki. Wird er
verworfen, bleibt das Wiki unberührt.

**Die Evidenzbindung als prüfbares Verfahren.** „Jede Aussage zeigt auf das
Paket" muss deterministisch entscheidbar sein, sonst ist die Prüfung aus §14
nicht schreibbar. Deshalb:

1. Das Analysepaket wird in **nummerierte Segmente** zerlegt (Diff-Abschnitt,
   Absatz der Wiki-Seite, Eintrag der `sources`-Liste). Die Nummern gehören zum
   Paket, nicht zum Modell.
2. Das erzwungene Ausgabeschema verlangt je Behauptung ein Feld `evidence` mit
   **Segmentnummer und wörtlichem Zitat** daraus.
3. Der Prüfer ist reine Textarbeit: Existiert das Segment? Kommt das Zitat darin
   **wortgleich** vor? Trägt jede Behauptung mindestens einen Beleg?
4. Fällt eine einzelne Behauptung durch, wird sie aus dem Vorschlag entfernt und
   im Fall vermerkt. Fallen alle durch oder bleibt die Kernaussage ohne Beleg,
   wird der Vorschlag verworfen und der Fall als „manuell zu prüfen" geführt —
   **kein zweiter Versuch**, denn ein Modell, das beim ersten Mal unbelegt
   schreibt, tut es beim zweiten meist auch, nur teurer.

Damit ist die Bindung ohne Modell testbar: Ein Vorschlag mit erfundenem Zitat
muss durchfallen, einer mit korrektem Zitat durchgehen. Beides sind Testfälle
mit festen Eingaben.

### 10.7 Protokolle

`log.md` hält fest, **was** sich geändert hat. `audit.md` hält fest, was
**vorgeschlagen**, was **entschieden** und was **tatsächlich geändert** wurde —
nur so bleibt ein abgelehnter Vorschlag nachvollziehbar.

### 10.8 Zweiter Auslöser: Merge nach `main`

Neben der Quellenänderung gibt es ein zweites deterministisch erkennbares
Ereignis: Beschlossenes ist gelandet.

```
post-merge-Hook (Code)  erkennt Merge nach main in einem registrierten Repo
   ▼
Kandidaten (Code): Seiten des Bereichs mit realization ∈ {planned, in_progress}
   ▼
Evidenz (Code): geänderte DATEIPFADE und COMMIT-NACHRICHTEN — kein Codeinhalt
   ▼
Fall im Prüfzentrum: „Realisierung prüfen"
   ▼
brain:land formuliert den Vorschlag · Evidenzbindung · Freigabe · Wiki · Commit
```

**Das Paket enthält Pfade und Commit-Nachrichten, keinen Quelltext.** Damit
bleibt §16.1 gewahrt — es wird weiterhin nicht gegen Codeinhalte abgeglichen —,
die Kosten bleiben niedrig, und die Datenschutzprüfung ist einfach: Dateinamen
und Commit-Texte sind ungleich harmloser als Quelltext. Für die Frage „ist das
Beschriebene jetzt gebaut?" reicht das in aller Regel.

**Die Kandidatenmenge ist klein und deterministisch:** nur Seiten, die noch
nicht `implemented` sind — in einem Projekt typischerweise eine Handvoll. Kein
Suchlauf, keine Heuristik. Das Modell entscheidet allein, welche der Kandidaten
der Merge tatsächlich umsetzt.

Aktiviert wird das je Repo, nie global:

```toml
[maintenance]
on_merge = true          # installiert und nutzt den post-merge-Hook
```

`brain hook install` richtet ihn ein, `brain hook status` zeigt, wo er hängt.
Kein Hook ohne ausdrückliche Zustimmung des Repos — ein Werkzeug, das ungefragt
in fremde Git-Hooks schreibt, wird einmal benutzt.

---

## 11. Vorschlaggeber und lokales Modell

Eine Schnittstelle, drei Umsetzungen, ausgewählt durch den Datenschutzmodus:
lokales Modell · Claude nach Bestätigung · Claude direkt.

Das lokale Modell hat genau drei Rollen: Vorschlaggeber für `local_only`,
Ablagevorschläge beim Import (für alle Bereiche, da Fließbandarbeit),
Ein-Satz-Beschreibungen für die Kataloge.

**Ausdrücklich nicht** für Verdichtung ins Wiki und nicht zum Beantworten von
Wissensfragen. Beides verlangt, mehrere Quellen gleichzeitig zu überblicken;
ein kleines Modell erzeugt dort genau den Kompilierungsfehler, gegen den die
Architektur gebaut ist.

**Umsetzung:** fertiger lokaler Modellserver, angesprochen über HTTP — nicht im
Daemon eingebettet. Ausgabeform wird über ein Schema erzwungen statt erhofft.
Mehrsprachigkeit (Deutsch und Englisch) ist Auswahlkriterium. Eigene
Prompt-Fassung je Umsetzung, getrennt versioniert.

**Opt-out.** Drei Ebenen, die gröbere gewinnt: global, je Bereich, je Rolle.
Vorgabe bei frischer Installation ist `enabled = false`; die Abhängigkeit ist
beim Paketieren optional. Der Schalter ist ein Codegatter vor dem Aufruf, keine
Bitte im Prompt. Rückfälle: manueller Fall; Datei bleibt im Eingang; **keine
Beschreibung**.

> Berichtigt am 2026-09-08. Bis dahin stand hier „erste Zeile als
> Beschreibung". Gebaut wurde sie nicht, und sie wird auch nicht gebaut: Eine
> so gewonnene `description:` stünde im Dateikopf ununterscheidbar neben einer,
> die das Modell geliefert hat. Der Katalog braucht sie nicht — ohne
> Beschreibung entfällt dort nur der Zusatz hinter dem Link. Die Begründung im
> Langen steht in §3 der Scheibe-6-Spec
> ([2026-09-05-scheibe-6-lokales-modell-design.md](2026-09-05-scheibe-6-lokales-modell-design.md)).

**Kritische Regel:** Das lokale Modell abzuschalten darf **niemals** dazu
führen, dass Inhalte stattdessen in die Cloud gehen. Der Datenschutzmodus steht
immer über der Modellverfügbarkeit.

**Messung frei Haus:** Die vier Entscheidungen im Prüfzentrum sind bereits die
Bewertung des Vorschlaggebers. Der Anteil unverändert übernommener Vorschläge
ist eine ehrliche Qualitätszahl; die Rückweisungsquote der Evidenzbindung ist
eine Frühwarnung.

---

## 12. Import und Konverter

`inbox/` ist die einzige Tür.

```
Datei im Eingang → Format erkennen (Code) → konvertieren (Code, idempotent)
→ Herkunftskopf (Code) → Ablageort vorschlagen (Modell) → Mensch entscheidet
```

| Konverter | Umfang | Grenze |
|---|---|---|
| PDF | Textextraktion mit Layouterhalt | keine Texterkennung für Scans; solche Dateien bleiben liegen und werden gemeldet |
| Webseiten | vorrangig Obsidian Web Clipper; eigener Weg nur als Rückfall | Bilder werden lokal abgelegt |
| Transkripte | **mechanisches** Zusammenfügen der Zeitstempelfragmente zu Absätzen, grobe Zeitmarken als Anker | Herkunft aus automatischer Spracherkennung wird vermerkt |

Transkripte sind der wichtigste Konverter: Zeitstempel alle paar Sekunden
zerschneiden jeden Sinnzusammenhang, und das Chunking entscheidet über die
Trefferqualität. Erkennungsfehler werden nicht stillschweigend als Tatsache
übernommen.

**„Zusammenfügen" heißt nicht „zusammenfassen".** Es wird kein Wort umformuliert
und keines weggelassen — die Fragmente werden aneinandergehängt, bis eine
Absatzlänge erreicht ist, und die Zeitmarke des ersten Fragments bleibt als
Anker stehen. Damit ist der Schritt reiner Code (§4.2), idempotent, und er
umgeht keine Datenschutzprüfung. Eine Verbesserung des Wortlauts — etwa das
Korrigieren von Erkennungsfehlern — wäre ein Eingriff in die Quelle durch ein
Modell und ist ausdrücklich **nicht** Teil der Konvertierung (§16.9).

Verstreutes Altmaterial geht denselben Weg, nur stapelweise und begleitet.
Keine Konnektoren zu fremden Systemen.

---

## 13. Messprogramm

| # | Messung | Werkzeug | Aussage |
|---|---|---|---|
| 1 | Fragensatz mit bekannter Zielquelle | `brain bench` | Trefferqualität |
| 2 | Vergleich mit und ohne System | Handarbeit, `/cost` und `/context` | Nutzen gegenüber Nichtstun |
| 3 | Latenz je Operation, kalt und warm | `brain bench --latency` | Budgets, Client- und Build-Entscheidung |
| 4 | Wiki-Gesundheit im Verlauf | `brain lint` | Zustand der Verdichtungsschicht |

Der Fragensatz umfasst **30 echte Fragen** in vier Sorten, verteilt
**8 / 8 / 6 / 8**: exakter Begriff (8), Umschreibung (8), gemischt (6) und
**sprachübergreifend** (8; deutsche Frage, englische Zielquelle, und mindestens
einmal umgekehrt). Die vierte Sorte ist der schärfste Prüfpunkt: Sprachübergreifend
findet allein die Bedeutungssuche — BM25 kann es prinzipiell nicht, und der
Reranker muss ebenfalls mehrsprachig sein, sonst stuft er genau die Treffer
zurück, die das Embedding-Modell gefunden hat.

Erfolgsmaß ist, ob die Zielquelle unter den ersten drei liegt — **nicht** der
Relevanzwert. Ein hoher Wert ist kein Wahrheitsbeweis.

Latenzbudgets. **„Warm" heißt: Der Daemon läuft und hat den Index sowie die
Modelle geladen. „Kalt" heißt: Der Daemon läuft, aber die für diese Operation
nötigen Modelle sind noch nicht geladen.** Der Daemonstart selbst ist
ausdrücklich **kein** Budget, sondern ein eigener Messwert — er fällt einmal je
Sitzung an und wird mit dem Wartehinweis aus §7.5 überbrückt.

| Operation | Budget kalt | Budget warm | **gemessen kalt** | **gemessen warm (Daemon)** |
|---|---|---|---|---|
| Katalog, Scope, Datei lesen | < 10 ms | < 10 ms | 1–7 ms ✅ | 3 ms ✅ |
| Stichwortsuche (`keyword`) | < 30 ms | < 30 ms | 208 ms ❌ | 21 ms ✅ |
| Vektorsuche (`fast`) | ~30 s | < 20 ms | 2792 ms | 78 ms ❌ (3,9-fach) |
| Hybridsuche (`full`) | ~40 s | + 100–500 ms | 5852 ms | (2c-2) |
| Daemonstart, Stufe 1 (eigener Messwert) | — | einmal je Sitzung | — | 1132 ms |
| Daemonstart, Stufe 2 (eigener Messwert) | — | einmal je Sitzung | 9982 ms | 1219 ms |

**Die Spalte „kalt" stammt aus Scheibe 2b, die Spalte „warm" aus Scheibe
2c-1** — erstmals echte Daemon-Wärme im Sinne dieses Abschnitts, zehn
Wiederholungen je Zeile, Median, gemessen *durch* den Daemon und damit
einschließlich Rohr, Rahmung und Klientenschleife
(`bench/2c1/warm-budgets.md`). Der Vorbehalt von 2b — „warm heißt hier nur,
dass qmd schon einmal gesucht hat" — ist damit eingelöst.

Was die Zahlen sagen:

* **Katalog und Datei lesen** halten ihr Budget mit großem Abstand, kalt wie
  warm. Der Daemon kostet sie rund 2 ms Rohr und Rahmung.
* **Die Stichwortsuche** hält ihr Budget jetzt (21 ms gegen 30 ms), nachdem
  sie es ohne Daemon um das Siebenfache verfehlte. Die 208 ms waren
  Prozessstart, nicht Suche.
* **Die Vektorsuche verfehlt ihr Budget um das 3,9-fache** (78 ms gegen
  20 ms). Das steht hier als Verfehlung und nicht als angepasstes Budget: ein
  Budget, das sich der Messung anpasst, ist keines mehr. 20 ms waren von
  Anfang an eine Schätzung ohne Messung darunter; 78 ms sind die Kosten eines
  Modelldurchlaufs plus zweier Prozessgrenzen. Ob das Budget falsch war oder
  der Weg zu teuer ist, entscheidet eine spätere Scheibe an einer Messung, die
  Rohr und Modell trennt.
* **Der Daemonstart** zerfällt in die zwei Bereitschaftsstufen aus §5.1 der
  2c-1-Spec: Stufe 1 (Rohr offen, Graph und Registrierung geladen) kostet
  1132 ms, Stufe 2 (erste Suche beantwortet) 9982 ms mit kaltem Modell und
  1219 ms, wenn qmds eigener Daemon noch läuft. Dass nur der *erste* Start den
  Modellstart zahlt, ist der gemessene Gewinn von 2c-1: qmds Daemon überlebt
  den brain-Daemon, der ihn gestartet hat (`bench/2c1/qmd-http-daemon.md`).
* **Die Hybridsuche** ist offen: `full` lädt zusätzlich das
  Erweiterungsmodell, und in 2c-1 wurde durchgehend die Vorgabe `fast`
  gemessen. Ein Kaltstart mit `full` lag bei **39,7 s**.

**Korrektur nach der ersten Messung (Scheibe 0).** Hier standen ursprünglich
150–400 ms für die kalte Bedeutungssuche. Gemessen wurden auf diesem Rechner
**27,1 s Frageerweiterung, 1,8 s Frage-Embedding und 9,4 s Reranking** — zwei
Größenordnungen daneben. Die Ursache war ein Denkfehler, kein Messfehler: Ich
hatte das Frageerweiterungs-Modell nicht als eigenen Ladeposten gerechnet, und
es ist mit 1,7 Milliarden Parametern das mit Abstand größte der drei.

Für die Architektur folgt daraus nichts Neues, aber ein Argument verschiebt
sich deutlich: **Der Daemon ist keine spätere Bequemlichkeit, sondern die
Voraussetzung dafür, dass die volle Pipeline im Alltag benutzbar ist.** Eine
kalte Bedeutungssuche von einer halben Minute würde niemand zweimal abwarten.
Warm bleiben die Budgets unverändert gültig — das ist der Punkt.

Das `full`-Profil hat bewusst ein weiches Budget: Über MCP wartet ohnehin ein
Sprachmodell auf das Ergebnis. Der Absatz, der hier stand, hat die Vorgabe
zweimal begründet und beide Male falsch — erst `fast` aus einem
Kostenargument, dann `full`, weil es „warm nichts koste und `fast`
Trefferqualität koste". Scheibe 2b hat beides gemessen: `fast` findet **besser**
als `full`, nicht schlechter, und ist kalt halb so teuer (§7.3). `fast` ist
damit die Vorgabe überall.

Das weiche Budget bleibt trotzdem stehen — es gilt jetzt für `full` als
wählbares Profil. Der kalte Erstlauf wird in beiden Fällen mit dem
Wartehinweis aus §7.5 überbrückt, nicht mit einem schwächeren Profil.

> **Nachtrag 2026-08-22: am echten Vault ist `full` deutlich besser, und die
> Vorgabefrage ist wieder offen.** Erster Lauf des 50er-Fragensatzes über alle
> vier registrierten Bereiche (276 indexierte Dokumente), warm über den
> Daemonpfad:
>
> | Profil | exakt | umschreibung | gemischt | sprachübergreifend | gesamt | warm je Frage |
> |---|---|---|---|---|---|---|
> | `keyword` | 9/13 | 1/13 | 1/10 | 0/14 | **11/50** | 15 ms |
> | `fast` | 6/13 | 6/13 | 5/10 | 9/14 | **26/50** | 89 ms |
> | `full` | 11/13 | 9/13 | 7/10 | 14/14 | **41/50** | 5503 ms |
>
> Das steht gegen die Messung aus 2b (24/30 für `fast`, 22/30 für `full`), und
> der Unterschied ist zu groß, um ihn Rauschen zu nennen. Was sich geändert
> hat: der Fragensatz (30 → 50 Fragen), der Bestand (ein Bereich → vier) und
> die Zusammensetzung der Sorten. Am auffälligsten ist die vierte Spalte —
> sprachübergreifend findet `full` **14/14**, `fast` nur 9/14; §16.3 maß dort
> 8/8 für beide.
>
> **Die Vorgabe wird hier nicht gewechselt.** `full` kostet 5,5 s je Frage
> gegen 89 ms — das Sechzigfache, warm gemessen, weil die Frageerweiterung je
> Anfrage ein 1,7-Milliarden-Modell fährt. Eine Vorgabe, die jede Suche um
> fünf Sekunden verzögert, ist eine andere Entscheidung als eine, die zwei
> Treffer mehr holt, und sie gehört nicht in eine Fußnote. Was zu tun ist,
> steht in Entscheidung 46. **Nachgetragen am selben Tag:** der eigene
> Messlauf ist gefahren — je drei Läufe, gleicher Fragensatz, gleicher
> Bestand, dreimal 26/50 gegen dreimal 41/50. `fast` bleibt die Vorgabe
> wegen der Latenz, und der teure Teil von `full` ist nicht der Reranker,
> sondern die Frageerweiterung, die einen Treffer *kostet*.
> Protokoll: [`bench/2c1/entscheidung-46.md`](../../../bench/2c1/entscheidung-46.md).
>
> Protokolle: `98 Messung/bench-2026-08-22-1737-fast.md`,
> `…-1737-keyword.md`, `…-1742-full.md` (im Vault, nicht im Repo).

**`brain reindex` treibt beide Indizes.** Der eigene Lauf und `qmd update`
stehen hinter einem Befehl, und die Sammlungsdefinition bei qmd — Muster und
Ignore-Liste — wird aus dem Manifest abgeleitet (§5.5), statt von Hand gepflegt
zu werden. Damit können die beiden Indizes nicht schon an ihrer Auswahl
auseinanderlaufen. Der Preis ist benannt: `reindex` ist damit nicht mehr
netz- und modellfrei, und das Fertig-Kriterium von Scheibe 1 — zwei Läufe,
byteweise identisch — bezieht sich ausdrücklich auf den **eigenen** Teil der
Ausgabe.

**Wie qmd aufgerufen wird, ist entschieden: gehaltener Unterprozess.** Die
Frage war ein Messpunkt, und die Messung ist eindeutig. Der Prozessstart allein
kostet auf diesem Rechner **278–310 ms** — gemessen mit `qmd --version`, einem
Aufruf, der nichts tut außer zu starten. Das Budget der Stichwortsuche beträgt
30 ms. Der Start verbraucht es also **um das Neunfache**, bevor ein einziges
Zeichen gesucht wurde; die gemessenen 194 ms warm bestehen fast ausschließlich
daraus.

Ein Prozessstart je Anfrage ist damit ausgeschlossen. qmd wird als dauerhafter
Unterprozess gehalten und über seine Schnittstelle angesprochen. Gebaut wird
das in **Scheibe 2c**, wo der Daemon ohnehin entsteht — die Haltung vorher zu
bauen hieße, sie zweimal zu bauen. Die Naht dafür existiert bereits
(`Runner` in `brain.search.qmd`), und nichts oberhalb von ihr ändert sich.

Die Messung läuft **vor** der Entscheidung, nicht danach. Jede Behauptung in
dieser Spec hat einen Prüfpunkt; wo keiner formulierbar ist, wird die Behauptung
gestrichen.

---

## 14. Bau- und Qualitätsregeln

- **TDD.** Erst der fehlschlagende Test. Jede Aufgabe hat Test und
  Fertig-Kriterium, das ohne Codelektüre prüfbar ist.
- **100 % Coverage, gemessen.** Ausschlüsse nur mit begründendem Kommentar.
- **Linting, Formatierung, Typen** laufen sauber durch. Kein `Any`, kein
  `# type: ignore` ohne Begründung.
- **Python ≥ 3.14**, threadsicher gebaut. Ob der Free-Threading-Build zum Einsatz
  kommt, entscheidet ein Verfügbarkeitstest der Abhängigkeiten — nicht die
  Architektur.
- **`uv` für alles.** Kein `pip`, keine `requirements.txt`.
- **Kein Sprachmodell in Tests.** Der Prüfvorschlag wird gegen aufgezeichnete
  Antworten getestet. Hart getestet werden Eignungsprüfung, Datenschutzprüfung
  und Evidenzbindung — die Stellen, an denen ein Fehler wehtut.
- **Suche hinter einer Attrappe** in Tests: keine Suchmaschine, keine Modelle,
  kein Netz.
- **Reproduzierbarkeit.** `graph.json` und `index.md` bei gleicher Eingabe
  byteweise gleich (stabile Sortierung, keine Zeitstempel im Inhalt). Konverter
  idempotent. Abgleich wiederaufsetzbar.
- **Sprache.** Nach außen englisch (README, Handbuch, `--help`, Fehlertexte,
  Vorlagen), intern deutsch (Specs, Pläne, Entscheidungsseiten). CLI-Befehle,
  MCP-Werkzeuge, Manifest-Schlüssel und Scope-Werte englisch.
- **Plattformabhängiges hinter einer Abstraktion:** Anwendungsdatenverzeichnis
  und Named Pipe gegen Unix-Socket. Der Ort des Zustandsverzeichnisses ist
  einstellbar (§5.1); kein Modul bildet ihn selbst.
- **Wo der Index liegt, entscheidet der Bereich.** „Index außerhalb des Baums"
  (Entscheidung 14) gilt uneingeschränkt für alles, was nur die Maschine liest.
  Kataloge im eigenen Vault sind die begründete Ausnahme: Sie sollen in Obsidian
  lesbar sein. Für `readonly`-Bereiche entfällt diese Begründung, und die
  Artefakte wandern ins Zustandsverzeichnis (§5.5, §5.8).

---

## 15. YAGNI — was ausdrücklich nicht gebaut wird

| Nicht gebaut | Stattdessen |
|---|---|
| eigener Suchkern | qmd hinter der Schnittstelle, bis eine Messung dagegen spricht |
| Dienstverwaltung für den Daemon | Selbststart durch den ersten Klienten |
| Texterkennung für gescannte PDFs | Datei bleibt liegen und wird gemeldet |
| Konnektoren zu Notiz-Apps | Eingangsordner |
| Code-Indexierung | nur auf Wunsch im Manifest |
| Abgleich gegen Code-Inhalte | bewusst offener Rand, siehe §16 |
| **ein** Wiki über mehrere Bereiche hinweg | ein Wiki je Bereich; geteilte Bereiche sind eigene Bereiche, verbunden durch Verweise (§5.7) |
| Kopieren beim Befördern | die geteilte Seite ist die Wahrheit, das Projekt verweist |
| Lesen aus Projekten in geteilte Bereiche | Leserichtung einseitig; hinein führt nur die Beförderung |
| ~~Wiki im Code-Repo — auch nicht als Schalter~~ | **durch Entscheidung 48 zurückgenommen:** ein schreibbarer Bereich trägt sein Wiki im eigenen Repo (`[layout] wiki`, §5.3), ein `readonly`-Bereich im Vault. Nicht gebaut bleibt eine zweite Betriebsart oder eine Sonderbehandlung je Platzierung (§5.7.1) |
| Ablaufsteuerung, Zustandsmaschine, Skill-Framework | Skills sind Prosa-Dateien mit Schritten und Prüfpunkten (§7.6) |
| Abgleich gegen Codeinhalte beim Merge | nur Dateipfade und Commit-Nachrichten (§10.8) |
| Mehrbenutzerbetrieb, Anmeldung, Rechte | lokales Einzelplatzwerkzeug |
| Schreibrechte in der Web-App | genau eines: die Freigabe |
| eigene Datenbank als Wahrheit | Markdown; Zustand abgeleitet |

---

## 16. Offene Ränder und Risiken

1. **Projekt-Wikis veralten durch Commits, nicht durch Notizänderungen.** Der
   Abgleich läuft bewusst nicht gegen Code-*Inhalte*. Der häufigste und
   folgenreichste Fall — „das Beschlossene ist inzwischen gebaut" — wird
   inzwischen über das Git-*Ereignis* erkannt (§10.8), ohne dass dafür Quelltext
   gelesen wird. Der Rand ist damit deutlich kleiner, aber nicht geschlossen:
   Eine Codeänderung, die eine bereits als `implemented` markierte Seite
   inhaltlich überholt, fällt weiterhin nur über `stale_after`, den Lint-Hinweis
   auf lange unberührte Seiten oder bewusstes Nachziehen auf. Bewusst
   akzeptiert.
2. **Der Kompilierungsfehler bleibt möglich.** Die drei Riegel senken das
   Risiko, sie beseitigen es nicht. Das Wiki ist eine Leseschicht über den
   Notizen, nie ihr Ersatz.
3. **Mehrsprachigkeit — zweimal gemessen, und beide Vermutungen dieser Spec
   waren falsch.**

   Ursprünglich stand hier: das englischoptimierte Standardmodell werde
   sprachübergreifend abfallen. Der Spike in Scheibe 0 widerlegte das an 30
   Fragen (23/30 für beide Modelle, sprachübergreifend 7/7 gegen 6/7 — nicht
   unterscheidbar). Übrig blieb ein Vorbehalt, der als **tragend** bezeichnet
   wurde: das Standardmodell genüge nur, solange die volle Hybridkette samt
   Reranker aktiv bleibe; die Brücke liege möglicherweise im Reranker, und
   diesen auszutauschen sei deshalb ein Eingriff in eine belegte Eigenschaft.

   **Scheibe 2b hat das gemessen, und der Vorbehalt fällt.**

   | Kette | sprachübergreifend |
   |---|---|
   | `fast` — rein vektoriell, ohne Erweiterung, ohne Reranker | **8/8** |
   | `full` — Erweiterung + Embedding + Reranker | **8/8** |
   | `keyword` — BM25 | **0/8** |

   **Die Sprachbrücke liegt im Einbettungsmodell, und dort allein.** Ohne
   Frageerweiterung und ohne Reranker findet die reine Vektorsuche jede der
   acht sprachübergreifenden Fragen, in beiden Richtungen. Das ist kein
   Ein-Treffer-Unterschied wie in Scheibe 0, sondern 8/8 gegen 0/8 bei der
   einzigen Kette ohne Einbettungsmodell.

   **Was daraus folgt.** Der Reranker verliert seinen Sonderstatus für
   Entscheidung 16 — er trägt die Brücke nicht, und im Gesamtergebnis
   verschlechtert er sie sogar (§7.3). Tragend ist stattdessen das
   **Einbettungsmodell**: es auszutauschen ist der Eingriff, der diese
   Eigenschaft gefährdet, und ein Tausch verlangt dieselbe Messung erneut.

   **Die zwei getrennten Läufe sind einer geworden.** Hier standen zwei
   Läufe, die nicht verwechselt werden dürften: ein rein vektorieller für den
   Anteil des Einbettungsmodells, ein zweiter mit `fast` für die Frage, ob
   `fast` sprachübergreifend trägt. Unter qmd 2.8.3 sind beide **derselbe
   Aufruf**, weil die Kette „Erweiterung ohne Reranker" nicht existiert
   (§7.3). Der eine verbliebene Lauf beantwortet beide Fragen, und das
   Protokoll sagt das ausdrücklich, damit niemand nach dem zweiten sucht.

4. **Chunking ist der wichtigste Qualitätshebel** und liegt zunächst außerhalb
   unserer Kontrolle, weil qmd es bestimmt. Bei rund 290 Notizen ist der Druck
   geringer als ursprünglich angenommen, der Hebel aber derselbe.
5. **Free-Threading-Build** hängt an der Verfügbarkeit passender Pakete. Wird vor
   Implementierungsbeginn geprüft.
6. **Der Scope ist eine Vorgabe, keine Sperre.** Ein LLM kann bewusst weiten.
   Eine harte Grenze war ausdrücklich nicht gewünscht.
7. **Die Grenzen der geteilten Bereiche sind heute nicht bekannt.** Sie zeigen
   sich erst im Gebrauch. Entschärft durch die Selbstheilung der Verweise über
   `doc_id` (§5.7.3) und durch den Start mit nur zwei Bereichen — nicht
   beseitigt. Ein Umschnitt bleibt eine bewusste Aufräumaktion.
8. **Die Beförderung ist die einzige Stelle, an der Projektinternes in eine
   möglicherweise teilbare Schicht gelangen kann.** Das Entschärfen schlägt ein
   Modell vor, die Freigabe erteilt ein Mensch. Ein Restrisiko bleibt und
   verlangt beim Veröffentlichen einen eigenen Durchgang.
9. **Erkennungsfehler aus Transkripten bleiben in der Rohquelle stehen.** Der
   Konverter fügt nur zusammen und korrigiert nichts (§12); ein falsch
   erkanntes Fachwort ist damit dauerhaft im Bestand und für die
   Stichwortsuche unauffindbar. Ein Modell den Wortlaut verbessern zu lassen
   wäre ein Eingriff in die Quelle und wurde bewusst ausgeschlossen. Die
   Verdichtung fängt es teilweise auf, weil die Wiki-Seite den richtigen
   Begriff führt.
10. ~~**qmd unter Windows ist nicht dokumentiert.**~~ **Erledigt (Scheibe 0,
    Aufgabe 1):** qmd 2.8.3 installiert, indexiert, bettet ein und sucht auf
    diesem Rechner ohne Abbruch. Das Risiko ist nicht eingetreten;
    Entscheidung 6 bleibt unverändert.

    Dabei trat ein anderer Fallstrick zutage, der hier festgehalten wird, weil
    er beinahe eine Fehlentscheidung erzeugt hätte: **Der dokumentierte Weg,
    das Embedding-Modell über eine Umgebungsvariable zu wählen, ist
    wirkungslos** — die Konfigurationsdatei hat Vorrang, und das Anlegen einer
    Sammlung schreibt dort die Standardwerte hinein. Wäre der Modellvergleich
    dem README gefolgt, hätte er zweimal dasselbe Modell gemessen, „keinen
    Unterschied" ergeben und zur Ablösung des Suchkerns geführt — wo in
    Wahrheit ein Konfigurationseintrag genügt.
11. **Die Größenordnung wurde zweimal nach unten korrigiert — auf rund 290
    Notizen.** Erst hieß es „über 5.000", dann 870, nach der Sichtung in
    Scheibe 0 sind es etwa **290 echte Notizen**. Die Ursache war jedes Mal
    dieselbe: gezählt wurde, was das Dateisystem hergibt, nicht was Wissen ist.
    Von den 584 Dateien in `space` sind rund 150 Notizen — der Rest sind
    JDK-Lizenztexte unter `.tools/`, Testfixturen und eine vollständige
    Worktree-Kopie unter `.claude/`; bei `iam_wiki` sind 28 von 116 Dateien
    Arbeitsspuren.

    Angepasst wurde daraufhin Stufe 2 der Suchleiter (§8, lesen statt suchen,
    mit gemessener Schwelle) und die Begründung des hierarchischen Katalogs.
    Nicht angepasst: inkrementelles Indexieren — es ist billig zu bauen und
    spart auch hier Zeit.

    **Was das für die Architektur bedeutet, ist ausdrücklich festzuhalten:**
    Ihr Wert liegt nicht in der Menge. Bei 290 Notizen rechtfertigen sich
    Wiki-Schicht, Brain Maintenance und Scoping weiterhin über Vertrauen,
    Abgrenzung und Aktualität — nicht über Skalierung. Wer sie mit „das wird
    sonst zu groß" begründet, begründet sie falsch. Zugleich verschiebt die
    Zahl die Dringlichkeit: Was allein der Menge wegen gebaut würde, wird
    nicht gebaut.

    **Gemessen, nicht vermutet (Scheibe 0, Aufgabe 6):** Sechs Läufe, drei
    Fragen, mit und ohne Regelwerk über denselben Bestand. Alle sechs fanden die
    richtige Datei; der Aufwand lag bei 14 Werkzeugaufrufen mit Suchleiter gegen
    13 ohne. **Bei dieser Bestandsgröße zeigt die Suchleiter keinen messbaren
    Vorteil im Suchaufwand.** Gründe: Sie hat zwei Aufrufe Grundlast (Regelwerk,
    Katalog), 290 Notizen sind für `grep` erschöpfend durchsuchbar, und der
    Bestand trägt bereits handgepflegte Indexdateien, die denselben Zweck
    erfüllen — bei der Frage ohne jeden greppbaren Begriff fand der Lauf **ohne**
    Regelwerk den Einstieg über genau so einen Index.

    Das Maß erfasst Aufrufe, nicht Token; ein breiter `grep` kostet mehr Kontext
    als eine Trefferliste. Diese Einschränkung geht zulasten der Vergleichsseite
    und wird genannt, nicht ausgenutzt.

    **Konsequenz für die Spec:** Die Suchleiter rechtfertigt sich heute **nicht**
    über Effizienz. Wer sie so begründet, begründet sie falsch. Ihr Nutzen liegt
    in der Verlässlichkeit des Vorgehens (genau eine Datei, Quellenangabe,
    Abbruch nach zwei Läufen) und darin, dass sie mit dem Bestand skaliert,
    während die Grundlast konstant bleibt.

    **Einordnung durch den Nutzer (Entscheidung 35):** Der heutige Bestand ist
    ein **Testbestand**; er soll erheblich wachsen. Damit ist der fehlende
    Effizienzvorteil ein Befund über den *jetzigen* Zustand, keine Absage an die
    Konstruktion — die Grundlast bleibt konstant, der Nutzen einer Trefferliste
    wächst mit der Menge. Das entwertet die Messung nicht: Sie bleibt der
    Nullpunkt, gegen den später verglichen wird, und sie verhindert, dass ein
    Vorteil behauptet wird, den es heute nicht gibt.

    **Der Prüfpunkt dafür steht:** Derselbe Fragensatz, dieselben zwei
    Bedingungen, erneut gemessen, sobald der Bestand deutlich gewachsen ist.
    Zeigt sich dann kein Unterschied, ist die Frage nicht vertagt, sondern
    beantwortet.

12. **Duplikate im Bestand verfälschen jede Retrieval-Messung.** In `space`
    liegt unter `.claude/worktrees/` eine byte-identische Spiegelung von
    `space/wiki/`. Ohne Ausschluss konkurriert jede Zielquelle mit ihrem
    eigenen Klon um die vorderen Plätze, und gemessen würde Duplikatrauschen
    statt Trefferqualität. Für den Index gilt daher: Worktree-Kopien,
    Lizenztexte, Testfixturen und Arbeitsspuren sind kein Wissen und werden
    ausgeschlossen. Das betrifft nicht nur die Messung — es ist die sachlich
    richtige Indexkonfiguration und gehört in die Vorgabe für `[index]
    exclude` (§5.5).

    **Duplikate wirken zusätzlich auf einer zweiten Ebene**, die der Ausschluss
    nicht abdeckt und die beim Spike sichtbar wurde: qmd bettet je eindeutigem
    Inhaltshash ein, führt dieselbe Datei aber unter jedem ihrer Pfade in der
    Trefferliste. Bei einer Frage standen Rang 1 und Rang 3 auf **demselben
    Dokument**. Eine Liste mit fünf Plätzen verschenkt so Plätze an
    Wiederholungen — und zwar auch dann, wenn beide Fundorte legitim sind
    (dieselbe Referenzdatei in zwei Bereichen).

    **Die Anforderung, die hier stand, ist nach der Abnahme von Scheibe 2a
    zurückgenommen.** Sie lautete: Trefferlisten werden je Inhaltshash
    entdoppelt, bevor sie den Aufrufer erreichen. Zwei Befunde haben sie
    entkräftet, und beide waren erst am echten Bestand sichtbar.

    **Der Fall existiert — aber nur über Bereichsgrenzen hinweg, und dort ist
    Verschmelzen die falsche Antwort.** Diese Stelle behauptete zunächst, in den
    registrierten Beständen gebe es überhaupt kein Paar von Pfaden mit gleichem
    Inhaltshash. Das war am `space`-Register geprüft und für alle drei behauptet;
    über alle Bereiche zusammen ist es falsch. Gemessen liegen `SPEC.md` und
    `llm-wiki.md` byteweise identisch in **`space` und `iam-wiki`** — genau der
    Fall, den dieser Abschnitt ursprünglich nannte: dieselbe Referenzdatei in
    zwei Bereichen.

    Nur ist er ein **Übergangszustand**, kein Dauerzustand. Zwei Projekte halten
    heute je eine Kopie derselben Referenz; die Architektur hat dafür bereits
    eine Antwort, und sie heißt nicht Entdopplung, sondern **Beförderung in den
    geteilten Bereich** (§5.7, Entscheidungen 22 bis 24). Die geteilte Seite ist
    dann die Wahrheit, und es gibt nichts mehr zu verschmelzen.

    Eine bereichsübergreifende Entdopplung zu bauen hieße, diesen Übergang
    bequem zu machen, statt ihn sichtbar zu lassen — und sie würde in der
    Trefferliste zwei Dateien zusammenführen, die nicht zusammengehören, sondern
    umziehen sollen. Innerhalb eines Bereichs wiederum kommt der Fall nicht vor.

    **Und sie war unerreichbar.** Die Entdopplung stützte sich auf das eigene
    Register (Entscheidung 40), dessen `doc_id` je **Pfad** vergeben wird: zwei
    Pfade mit gleichem Inhalt bekommen verschiedene Kennungen, die Gruppierung
    kann also nie greifen. Der Test, der sie belegte, schrieb ein Register, das
    der Indexer nicht erzeugen kann — er prüfte eine Welt, die es nicht gibt.

    **Was bleibt, ist eine Meldung statt einer Mechanik:** `status` nennt es,
    wenn zwei Pfade denselben Inhaltshash tragen — und zwar **über
    Bereichsgrenzen hinweg** (Entscheidung 47). Bereichsintern kommt der Fall
    gar nicht vor; eine Prüfung je Bereich hätte also nie etwas gesagt, während
    genau die gemessenen Paare zwischen zwei Bereichen liegen. Das Register
    weiß es ohnehin, es kostet nichts, und tritt der Fall je ein, sieht man ihn.
    Eine Entdopplung wieder einzuführen ist dann eine begründete Änderung mit
    einem Bestand als Beleg — nicht eine Vorhaltung für einen Fall, den niemand
    hat.
13. **Der Vault ist Einzelpunkt des Ausfalls für alles, was in ihm liegt.**
   Sein Verlust ist der Verlust der vault-platzierten Wiki-Schichten und des
   Prüfzentrums — während die Rohquellen über die Code-Repos verteilt
   überleben. Das war der Preis der Zusammenlegung (§5.1) und macht das
   entfernte Sicherungsziel aus Entscheidung 14 dringlicher: nicht mehr
   „später", sondern spätestens mit Scheibe 3, sobald das erste Wiki entsteht.

   **Entscheidung 48 hat das Risiko verkleinert, nicht beseitigt.** Das Wiki
   eines schreibbaren Bereichs liegt in dessen Repo und überlebt einen
   Vault-Verlust dort — heute betrifft das genau `project/ultra-brain`. Die
   Wikis von `knowledge`, `engineering/python`, `engineering/craft`, `hub`
   sowie `project/space`, `project/iam-wiki`, `project/obsidian-ai` und
   `project/ecoflow` liegen weiterhin im Vault, das Prüfzentrum ebenso.
   „Der Verlust **aller** Wiki-Schichten" ist damit nicht mehr richtig; die
   Dringlichkeit der Sicherung ändert das nicht.

14. **qmd bricht gelegentlich still ab und liefert einen leeren Treffersatz.**
    Beim Spike in Scheibe 0 trat das bei drei von sechzig Abfragen auf —
    Speicherzugriffsfehler ohne Fehlertext, ohne Rückgabecode, der es verriete;
    im maßgeblichen Wiederholungslauf trat es erneut auf. **Ein Fehler, der sich
    als Ergebnis tarnt**, ist die gefährlichste Sorte: „nichts gefunden" ist eine
    plausible Antwort, und niemand prüft sie nach.

    Für den Kern folgt daraus eine Anforderung, die vorher nicht sichtbar war:
    Die Suchschnittstelle (§7.2) muss **leeren Treffersatz von fehlgeschlagener
    Suche unterscheiden** — durch Wiederholung bei leerem Ergebnis und, wenn auch
    die Wiederholung leer bleibt, durch eine ausdrückliche Meldung an den
    Aufrufer statt einer stillen Null. Andernfalls beantwortet das System
    Wissensfragen gelegentlich mit „dazu habe ich nichts", obwohl die Quelle
    vorliegt — und genau dieses Vertrauen soll das System herstellen.

    **Nachtrag aus der Abnahme von Scheibe 2a: der Zustand „leer" ist auf einer
    Bedeutungssuche gar nicht erreichbar.** Eine Vektorsuche liefert immer die
    nächsten Nachbarn — es gibt keinen Abstand, ab dem sie schweigt. Gemessen:
    eine Anfrage nach `zzqqxwvk-nichtvorhanden-42` lieferte auf dem Profil `full`
    fünf Treffer, den ersten mit **75 %**. Die Frageerweiterung dichtet aus der
    Zeichenfolge eine plausible Frage, der Reranker sortiert das Ergebnis und
    gibt ihm eine Zahl.

    Daraus folgt **keine** Änderung an der Anforderung oben — die Unterscheidung
    leer/fehlgeschlagen bleibt richtig und billig. Es folgt eine Klarstellung,
    die §8 bisher nur als Faustregel führte („ein hoher Relevanzwert ist kein
    Beweis"): **Der Wert ist ein Abstandsmaß, keine Aussage über Vorhandensein.**
    Ein leeres Ergebnis entsteht praktisch nur auf der Stichwortsuche. Wer auf
    einer Bedeutungssuche prüfen will, ob etwas überhaupt existiert, muss bis zur
    Rohquelle durchgehen; die Trefferliste kann es nicht sagen. Eine Schwelle,
    unterhalb derer unterdrückt wird, wurde erwogen und verworfen: sie schwankt
    mit Bestand und Frageart, und ein unterdrückter Treffer ist unsichtbar — also
    genau die Sorte stiller Verlust, gegen die §16.14 gebaut wurde.

15. **Ein Dateizähler taugt nicht als Unverändert-Nachweis**, wenn andere
    Prozesse im selben Baum arbeiten. Während der Messung lief eine
    Godot-Sitzung und schrieb in ihre eigenen Cache-Verzeichnisse; die
    Gesamtdateizahl änderte sich, ohne dass eine Notiz berührt wurde. Tragfähig
    ist die Kombination aus `git status`, Prüfung der Änderungszeiten der
    Notizen selbst und dem Ausschluss von Werkzeugverzeichnissen.

---

## 17. Zerlegung in Scheiben

| # | Scheibe | Fertig-Kriterium |
|---|---|---|
| 0 | Fundament ohne Code | Suchleiter funktioniert im Alltag; Vergleich mit/ohne einmal gelaufen; **Spike: trägt qmd sprachübergreifend?** |
| 1 | Indexer | zwei Läufe auf unverändertem Bestand erzeugen byteweise identische Dateien |
| 2a | Suchkette ohne Daemon | die fünf schreibfreien Werkzeuge laufen als direkte CLI über die drei externen Bestände. **Datenschutznachweis:** ein Bereich auf `local_only` liefert auf dem Cloud-Kanal weder Treffer noch Inhalte noch Katalogeintrag, auf dem lokalen Kanal alles; ein `never`-Pfad ist auf keinem Kanal erreichbar. **Leer ≠ Fehler:** eine erzwungen fehlschlagende Suche liefert eine ausdrückliche Fehlermeldung, eine echte Suche ohne Treffer ein leeres Ergebnis — beide sind an der Ausgabe zu unterscheiden (§16.14). Eine bekannte Zielquelle wird gefunden |
| 2b | Messwerk | **erledigt.** `brain bench` in allen Profilen und `brain bench --latency` gelaufen, Protokolle im Vault unter `98 Messung/`. Der qmd-Aufruf ist entschieden: **gehaltener Unterprozess** — der Prozessstart allein kostet ~280 ms gegen ein Budget von 30 ms (§13). Mehrsprachigkeit: die zwei getrennten Läufe aus §16.3 sind unter qmd 2.8.3 **ein** Lauf, weil die Kette dazwischen nicht existiert; er beziffert beides und ergibt 8/8 (§16.3) |
| 2c | Daemon, IPC und MCP | Daemon hält Index und Modelle, Klient startet ihn und meldet die Wartezeit; der MCP-Adapter setzt auf denselben Kern. Der Datenschutznachweis aus 2a wird über den echten MCP-Kanal wiederholt |
| 3 | Wiki-Schicht **plus `brain:research` und `brain:ingest`** | präparierte Notiz: drei Widersprüche markiert, harmlose Ergänzung eingearbeitet, nichts überschrieben, `open_conflicts` stimmt; eine Recherche landet auf Wunsch als `Synthesis` mit `status: draft` im Wiki |
| 4 | Import und Konverter | **erledigt.** Alle neun Nachweise aus der Spec (§11) gegen die echten Dateien im Wurzelverzeichnis geführt, Protokoll unter `docs/.superpowers/sdd/2026-08-25-scheibe-4-import/abnahme.md`. Die beiden Transkripte ergeben zweimal hintereinander dieselbe Datei (MD5 unverändert); 1799 Wörter vor und nach der Konvertierung identisch. Am `Bauanleitung_Second-Brain.pdf`: 11 594 Zeichen, 95 Absätze, null Ersatzzeichen. Am echten Abruf zu `mHSOsy_usAg`: 1289 Fragmente wurden 40 Absätze, `asr: true`, `source_url` die tatsächlich abgerufene URL |
| 5 | Brain Maintenance **plus `realization`, `brain:wiki-plan`, `brain:land`, Merge-Hook** | nachgebauter Fall: Abgleich erzeugt Fall, Vorschlag besteht Evidenzbindung, nach Freigabe stimmen Quelle und Wiki, genau ein Commit. **Selbstheilung:** eine Rohquelle umbenennen → kein Prüffall, Register führt dieselbe `doc_id` mit neuem Pfad. **Evidenzbindung:** ein Vorschlag mit erfundenem Zitat fällt durch, einer mit wörtlichem Zitat geht durch. Zweiter Nachweis: Zweig mit einer `planned`-Seite nach `main` gebracht → Fall im Prüfzentrum nennt genau diese Seite, nach Freigabe steht sie auf `implemented` mit richtigem Commit — **und im Analysepaket findet sich kein Quelltext** |
| 6 | Lokales Modell (opt-in) | `local_only` erzeugt Vorschlag ohne ausgehenden Verkehr; Gegenprobe: abgeschaltet → manueller Fall, ebenfalls kein Verkehr |
| 7 | Web-App | Freigabe in der App erzeugt dieselben Wirkungen wie über die CLI |

```
0 ──► 1 ──► 2a ──► 2b ──► 2c ──┬─► 3 ──┐
                               │       ├─► 5 ──► 6 ──┐
                               └─► 4 ──┘             │
                                                     ├─► 7
   (alle vorherigen Scheiben) ───────────────────────┘
```

Scheibe 2 ist in drei Teile zerlegt, weil ihr Fertig-Kriterium vier
Teilsysteme in einem Satz bündelte — und weil die Messung, die über das
Prozessmodell entscheiden soll (§13), sonst mitten in der Scheibe läge, deren
Bau sie bestimmt. 2a baut die Kette, 2b misst sie, 2c setzt den Daemon
darunter. Der Datenschutznachweis steht in 2a, weil der Kanal dort zum Begriff
wird; 2c wiederholt ihn nur am echten Adapter.

Scheibe 3 und 4 hängen nur am Kern, nicht aneinander. Scheibe 5 braucht beide.
Nach jeder Scheibe läuft das zutreffende Messprogramm; eine Scheibe ohne Zahlen
gilt nicht als fertig.

Die Web-App entsteht nach dem Muster „Zielbilder zuerst": Mockups von einer
Bild-KI erzeugen, dann Schritt für Schritt nachbauen mit Abnahme im Browser.
Keine Beschreibungen als Auftrag.

**Dokumentation:** diese Architektur-Spec als Vertrag; je Scheibe ein eigener
Plan, wenn sie ansteht. Scheibe 5 und 7 bekommen zusätzlich eine Teil-Spec, weil
dort noch echte Designfragen offen sind (Formate für Fall, Paket und Protokoll
beziehungsweise die Optik).

---

## 18. Entscheidungsprotokoll

| # | Frage | Entscheidung |
|---|---|---|
| 1 | Ausgangsbestand | Mischung: kleiner Vault plus viel verstreutes Material |
| 2 | Trennung | ein System, **drei Bereichsfamilien** (`knowledge`, `project/<name>`, `engineering/<name>`, §5.7.1); ursprünglich zwei Bereiche — durch Entscheidung 22 um die Familie `engineering/<name>` erweitert |
| 3 | Umfang | eigener CLI+MCP-Kern, Indexer und Graph, Web-App als letzte Scheibe |
| 4 | Latenz | nach der Messung in Scheibe 0 getrennt (§13): Katalog, Scope und Dateilesen unter 50 ms; **Suche je nach Profil darüber** — `full` warm +100–500 ms, kalt rund 40 s. Modellladen als eigener, gemessener Posten |
| 5 | Kern-Stack | Python 3.14 als Daemon, threadsicher; kompilierter Mini-Client; MCP am Daemon |
| 6 | Suchkern | qmd hinter eigener Schnittstelle, austauschbar |
| 7 | OKF-Reichweite | Wiki streng OKF, Rohquellen frei |
| 8 | Scoping | Scope-Parameter je Aufruf, Vorgabe-Scope in `.mcp.json`; die Ortsauflösung liegt in der zentralen Registrierung (§5.5.1). Die frühere Vorrangregel „Repo gewinnt" ist durch Entscheidung 32 überholt — es gibt sie nicht mehr |
| 9 | Repo-Vorgabe | ausschließlich Rohquellen (README/CHANGELOG/ADRs); Code und Sitzungsspuren nur auf Wunsch. Das frühere „Bundle im Repo" war durch Entscheidung 25 überholt und ist durch Entscheidung 48 für **schreibbare** Bereiche wieder zugelassen; für `readonly`-Repos gilt die Vorgabe unverändert |
| 10 | Wiki-Zuschnitt | je Bereich ein Wiki, auch für Projekte, mit `stale_after`-Pflicht |
| 11 | Ingest | `knowledge` Quelle für Quelle begleitet; Projekt-Wikis auf Zuruf |
| 12 | Import | Eingangsordner plus Konverter für PDF, Web, Transkripte |
| 13 | Erstbefüllung | nur ab jetzt; Altbestand auf Zuruf |
| 14 | Ablage | Git lokal; Index außerhalb des Baums. Das „Remote später" ist durch Entscheidung 27 überholt — das entfernte Ziel wird ab Scheibe 3 verbindlich (§16.13) |
| 15 | Messung | alle vier Messungen |
| 16 | Sprachen im Bestand | gemischt deutsch/englisch, sprachübergreifend finden |
| 17 | Abgleich in Projekten | **nicht** gegen Code-Inhalte; das Landeereignis wird sehr wohl ausgewertet (§10.8) |
| 18 | Zustandsort | Frontmatter ist Wahrheit, Datenbank ist wegwerfbar |
| 19 | Schreibrecht Web-App | genau eines: die Freigabe |
| 20 | Oberflächensprache | Befehle und Schlüssel englisch; Doku nach außen englisch, intern deutsch |
| 21 | Lokales Modell | drei Rollen, nie Verdichtung; opt-out auf drei Ebenen; Vorgabe aus |
| 22 | Geteiltes Programmierwissen | dritte Bereichsfamilie `engineering/<name>`, mehrere geteilte Bereiche statt einem |
| 23 | Ablage der geteilten Bereiche | im Vault unter `92 Engineering/`; Herauslösen erst bei Veröffentlichung; Start mit zwei Bereichen |
| 24 | Fluss zwischen Bereichen | Beförderung statt Kopie; geteilte Seite ist die Wahrheit; Leserichtung einseitig |
| 25 | ~~**Ablage aller Wikis**~~ | sämtliche Wikis im Vault; Code-Repos tragen nur Rohquellen; ein Wiki im Repo gibt es auch nicht als Schalter. **Durch Entscheidung 48 zurückgenommen**, soweit ein Bereich schreibbar ist |
| 26 | Prüfzentrum | ein Prüfzentrum im Vault für alle Bereiche, nach Bereich gruppiert |
| 27 | Sicherung | entferntes Ziel wird verbindlich, sobald das erste Wiki entsteht (Scheibe 3) — der Vault ist Einzelpunkt |
| 28 | Abläufe als Skills | vier Skills im Claude-Code-Plugin, Prosa mit Prüfpunkten; keine Sonderrechte, kein Skill-Framework |
| 29 | Soll-Ist-Trennung | `realization` und `implemented_in` auf Projektseiten; zwei zusätzliche Lint-Prüfungen |
| 30 | Merge-Auslöser | post-merge-Hook erzeugt Fall, Skill schlägt vor, Mensch gibt frei; Evidenz nur Pfade und Commit-Nachrichten; je Repo aktiviert |
| 31 | Datenschutz auf dem Lesepfad | `local_only` macht einen Bereich für das Cloud-Modell unsichtbar; `never` wirkt als Indexausschluss **und** Leseverweigerung |
| 32 | Ortsauflösung | die zentrale Registrierung bildet Bereich auf Pfad ab; Manifeste nennen nur den eigenen Scope; `brain://<scope>/<pfad>`, längste passende Registrierung gewinnt |
| 33 | Zwei Schreibwege | Ingest schreibt begleitet und ohne Fall; Wartung schreibt ausschließlich nach Freigabe |
| 34 | Größenordnung korrigiert | rund 290 Markdown-Notizen über drei Bestände; die Zahl wurde **zweimal** nach unten korrigiert — erst „über 5.000", dann 870, nach der Sichtung in Scheibe 0 rund 290 (§16.11); Stufe 2 der Suchleiter liest das Wiki, statt es zu durchsuchen — mit gemessener Schwelle bei 100.000 Token |
| 35 | Testbestand, Wachstum erwartet | der heutige Bestand ist ein Testbestand; er soll erheblich wachsen. Der Indexer (Scheibe 1) bleibt im Plan — begründet über das erwartete Wachstum, nicht über den heutigen Nutzen |
| 36 | **Fremde Bestände** | `readonly = true` im Manifest; Artefakte und Manifest liegen dann im Zustandsverzeichnis. Die Bestandsregel wird damit eine Eigenschaft des Bereichs statt Prosa, die kein Programm liest |
| 37 | Ort des Zustandsverzeichnisses | einstellbar: `--state-dir` schlägt `BRAIN_STATE_DIR` schlägt Plattformvorgabe. Voraussetzung dafür, dass Tests die Wege dorthin prüfen können |
| 38 | Kanal | `local` gegen `cloud` ist Parameter jeder Werkzeugfunktion des Kerns, nicht Verhalten des MCP-Adapters — jeder künftige Zugang erbt das Gatter, statt es nachzubauen |
| 39 | Ein Index-Befehl, eine Ausschlussliste | `brain reindex` treibt eigenen Lauf und `qmd update`; qmds Sammlung wird aus `[index]` abgeleitet. Preis: `reindex` ist nicht mehr netz- und modellfrei. **Nach der Abnahme von 2a eingekürzt:** die Liste hat weiterhin eine Quelle, aber sie ist nicht der Hebel, für den sie gehalten wurde — die Reichweite der Suchmaschine ist enger als jede Liste (Punktverzeichnisse werden grundsätzlich übersprungen). Was dort herausfällt, ist keine Divergenz, sondern eine Eigenschaft, und wird über `unsearched` erklärt statt gemeldet |
| 40 | ~~Entdopplung~~ Divergenz der Indizes | **Die Entdopplung ist nach der Abnahme von 2a zurückgenommen** (§16.12): ihre `doc_id` wird je Pfad vergeben, die Gruppierung konnte nie greifen, und der gemessene Fall liegt außerhalb der registrierten Bereiche. Was bleibt: die Divergenz der beiden Indizes ist ein Befund für `status` statt eines Fehlers der Suche — **in beiden Richtungen**, auch wenn unser Index mehr kennt als die Suchmaschine |
| 41 | Katalog-Einleitung | optionale Datei `index.intro.md` je Ebene, unverändert eingefügt — kein Markerformat in der erzeugten Datei |
| 42 | **Indiziert ist nicht durchsucht** | `[index] unsearched` erklärt Quellen, die in Katalog und Graph stehen und `read` beantwortet, die die Bedeutungssuche aber nicht kennt. Spec und Plan eines Projekts fallen darunter (§5.3). Der Kern erfährt nie, *warum* die Suchmaschine etwas nicht kennt — er zieht ab, was der Bereich selbst erklärt hat |
| 43 | Artefaktnamen bereichsabhängig | `index.md`, `index.intro.md`, `graph.json` und `_identities.tsv` fallen nur in schreibbaren Bereichen aus dem Index; in einem `readonly`-Bereich kann keine davon von uns stammen. Der pauschale Ausschluss kostete 22 handgeschriebene Wiki-Seiten |
| 44 | Einbetten ist ein eigener Lauf | `brain embed` neben `reindex`, weil `qmd update` nur den Volltextindex fortschreibt. Ein Dokument ohne Vektoren ist sprachübergreifend unauffindbar; `status` nennt den Rückstand |
| 45 | Der Relevanzwert ist ein Abstandsmaß | Auf einer Bedeutungssuche ist „nichts gefunden" nicht erreichbar — gemessen: 75 % auf eine Zeichenfolge, die im Bestand nicht vorkommt (§16.14). Keine Schwelle: sie schwankt mit Bestand und Frageart, und ein unterdrückter Treffer wäre ein stiller Verlust |
| 46 | **`fast` ist die Vorgabe überall** | zweimal gedreht, jetzt gemessen. Nach Scheibe 2a stand hier `full`, gestützt auf zwei Fragen. Scheibe 2b hat an dreißig gemessen, viermal `full` und viermal `fast`: `full` liefert 22/30 (einmal 23), `fast` dreimal 24/30 — `fast` ist in **jedem** Lauf besser, kalt halb so teuer (2792 ms gegen 5852 ms) und warm nicht langsamer. Der Reranker kostet zwei Treffer und die doppelte kalte Zeit (§7.3, §13). **Nachgemessen am 2026-08-22, gleicher Fragensatz, gleicher Bestand, je drei Läufe:** am echten Vault (50 Fragen, vier Bereiche) findet `full` **41/50** und `fast` **26/50** — dreimal dieselbe Zahl auf beiden Seiten, also kein Rauschen. Die 2b-Zahl galt für dreißig Fragen über *einen* Bereich und ist damit überholt, nicht widerlegt. **`fast` bleibt die Vorgabe** — aus dem verbliebenen Grund: 95 ms gegen 5351 ms je Frage, das 57-fache. Aufgeteilt zeigt sich, dass der Reranker die Treffer holt (`vec`+Reranking: 40/50 in 3952 ms) und die Frageerweiterung nur Zeit kostet (`full`: 39/50 in 5364 ms). **Geprüft und verworfen:** `full` auf `vec`+Reranking umzudefinieren sah roh gemessen frei aus, kostete durch die volle Kette aber fünf Fragen (36/50 gegen 41/50, auf beiden Rückgraten). Die Rohmessung verglich Dateinamen, die Kette vergleicht Pfade. `full` bleibt die Hybridkette (`bench/2c1/entscheidung-46.md`) |
| 47 | Dubletten über Bereiche sind ein Umzugsfall | dieselbe Referenzdatei in zwei Projekten (gemessen: `SPEC.md`, `llm-wiki.md` in `space` und `iam-wiki`) wird nicht in der Trefferliste verschmolzen, sondern gehört in den geteilten Bereich (§5.7, Entscheidungen 22–24). `status` nennt solche Paare, damit der Umzug sichtbar wird — bereichsübergreifend, weil der Fall es ist |
| 48 | **Wiki im Repo, aber nur wo geschrieben wird** | Entscheidung 25 ist zurückgenommen, und zwar nicht umgekehrt: ein **schreibbarer** Bereich legt sein Bundle über `[layout] wiki` in sein eigenes Repo (`docs/wiki`), ein `readonly`-Bereich kann das nicht und behält die Vault-Platzierung. Ausschlaggebend war der Lebenszyklus: eine Seite entsteht im Feature-Zweig, wird mit ihm gereviewt, gemergt und zurückgerollt — im Vault gibt es keine dazu passenden Zweige. Beide Platzierungen bestehen dauerhaft nebeneinander; die Platzierung ist eine Eigenschaft des Bereichs, keine Klasse von Repos (2026-08-29 „Das Projekt-Wiki zieht ins Code-Repo") |

Entscheidung 25 ersetzt die frühere Festlegung, das Projekt-Wiki liege im
Code-Repo. Ausschlaggebend waren zwei Befunde: Ein Repo mitzuveröffentlichen
hieße, sein Wiki mitzuveröffentlichen — und der vermeintliche Versionsvorteil
des Repos existiert nicht, weil der Abgleich ausdrücklich nicht gegen
Code-Inhalte läuft (§16.1). Das Landeereignis eines Merges auszuwerten (§10.8)
widerspricht dem nicht: Es prüft, **dass** etwas gebaut wurde, nicht **was**.

**Entscheidung 48 nimmt das für schreibbare Bereiche zurück.** Von den zwei
Befunden trägt der erste weiter und ist zum offen benannten Preis geworden: wer
ein Repo veröffentlicht, veröffentlicht sein Wiki mit, und `brain init` setzt
`docs/wiki` ohne zu fragen (2026-08-29 §1.3). Der zweite Befund war zu eng
gefasst — der Versionsvorteil des Repos liegt nicht im Abgleich gegen
Code-Inhalte, sondern darin, dass eine Seite und die Änderung, die sie
beschreibt, im selben Zweig liegen und im selben Merge landen.

---

## 19. Quellen

| Quelle | Rolle | Was übernommen wurde |
|---|---|---|
| qmd (github.com/tobi/qmd) | Baustein | hybride Lokalsuche, vollständig lokal |
| Karpathy, LLM Wiki | Muster | drei Schichten, Ingest/Query/Lint, Widersprüche markieren |
| Open Knowledge Format v0.2 | Muster | Frontmatter-Familien, index.md, log.md, Actor-Konvention, Trust-Tiers |
| Bauanleitung Second Brain (PDF) | Muster | Fähigkeiten zuerst, Suchleiter, Wiki-Schema, Widerspruchs-Workflow, sechs Prozessschritte, sieben Lektionen |
| Video 1 — Bauanleitung (Transkript) | Muster | zwei Suchprofile, Ingest über Seitenverzeichnis, Kosten entstehen beim Suchen, Katalog grob statt fein, lokale Randbedingung |
| Video 2 — RAG/Hybrid/Wiki (Transkript) | Muster | vier Wege ins Kontextfenster, sieben Entscheidungskriterien, vier Fehlerstellen, Vertrauenskette |
| Video 3 — Brain Maintenance (Transkript) | Muster | zwei Aktualitätsbegriffe, doc_id/hash/revision, Wächter plus Abgleich, zwei Tore, Analysepaket, Prüfzentrum, Audit, isolierter Commit |
| Superpowers | Baustein | Prozess Brainstorming → Spec → Plan → Bauen |
| Mockups (später zu erzeugen) | Messlatte | visuelles Soll für Scheibe 7 |

---
type: Topic
title: Die Wiki-Schicht
description: Schema, Ingest, Konfliktform, Lint — und die zwei Schreibwege ins Wiki.
open_conflicts: 0
realization: implemented
implemented_in: db780a0
sources:
  - id: architektur-spec
    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
    doc_id: 01M39G4J14CK311B66GRSAK7HQ
    content_hash: "sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff"
    revision: 3
  - id: stufe-3-spec
    resource: brain://project/loomux/docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md
    doc_id: 01M39G4J140NNQD5Q2HRVGKBG2
    content_hash: "sha256:06779f84695b1540346b2f2f73e9f5a0aa48f15ce835048b1efd3e209ea7faa9"
    revision: 1
  - id: stufe-4-spec
    resource: brain://project/loomux/docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md
    doc_id: 01M39NDHWJ80C90J6YB9B9AAQ9
    content_hash: "sha256:a9c28b567436dca62d44c1eab1fb4d39749e9193c1f3bb70709f1dd69e349cec"
    revision: 1
---

## Das Bundle

Das Wiki ist ein strenges [OKF](../entities/okf.md)-v0.2-Bundle; Rohquellen
bleiben frei. Vier Seitentypen: `Source` (eine Seite je Rohquelle), `Topic`
(mehrere Quellen zu einem Thema), `Entity` (Steckbrief), `Synthesis` (These,
Belege, offene Fragen — startet als Entwurf).

Zwei produzenteneigene Erweiterungen tragen die Schicht:

- **`sources[]` mit `doc_id`, `content_hash` und `revision`** je Beleg. Aus
  diesen Einträgen wird der Rückwärtsindex Quelle → abhängige Seiten
  deterministisch gebaut — kein zusätzliches Format.
- **`realization`** (`planned` · `in_progress` · `implemented` · `abandoned`)
  plus `implemented_in`, nur auf Projektseiten. Sie schließen eine Lücke, die
  OKF nicht abdeckt: `status` beschreibt die Reife der **Seite**, nicht den
  Umsetzungsgrad des **Beschriebenen**. Ohne diese Trennung liest sich eine
  Absichtserklärung von vor drei Monaten wie eine Systembeschreibung — genau
  diese Verwechslung war im Referenzsystem eine ganze Klasse der markierten
  Konflikte.

Die maßgebliche Kopie dieses Zustands steht in der Frontmatter, also in Git. Die
Zustandsdatenbank ist ein wegwerfbarer Beschleuniger; läge der Zustand nur dort,
würde ihr Verlust stillschweigend alle Seiten als aktuell gelten lassen.

## Das Schema — sechs Regeln

`_schema.md` liegt im Bundle und wird vor jeder Wiki-Arbeit gelesen: nur die KI
schreibt hier; Rohquellen sind unantastbar; jede Seite ist vernetzt;
Widersprüche werden markiert, nie aufgelöst; jede Änderung endet mit einem
Log-Eintrag; **verdichtet wird immer gegen die Originalquelle, nie gegen eine
ältere Zusammenfassung**. Es ist je Bundle editierbar und gilt vor allem
Allgemeinen.

## Ingest in acht Schritten

Schema lesen → Quelle lesen und Identität feststellen → **betroffene Seiten über
den Katalog ermitteln** → genau diese Seiten öffnen → Source-Seite schreiben und
Topics und Entities nachziehen → je Seite prüfen → `sources[]` festschreiben →
Log-Eintrag.

Der dritte Schritt ist der Grund, warum Ingest bei großem Bestand bezahlbar
bleibt: Es wird der Katalog gelesen, nicht das Bundle durchsucht.

Der Altbestand wird **nicht** vollständig verdichtet — nur neue Quellen sowie
einzelne Themen auf Zuruf.

## Konflikte

Feste, maschinell auffindbare Form mit einem Kasten `[!conflict]`, zwei belegten
Ständen und dem Schlusssatz, dass beide Stände stehen bleiben und die
Entscheidung offen ist. Dazu `open_conflicts` in der Frontmatter — **die
Buchführung der KI wird von Code kontrolliert.**

Bei der Auflösung sind vier Urteile zulässig: alter Stand veraltet; beides gilt
in verschiedenem Kontext; neue Information ist falsch; bewusst offen lassen.
Repariert wird dabei die **Quelle**, nicht der Kasten — wer nur die Markierung
löscht, findet denselben Widerspruch im nächsten Durchlauf wieder, weil sich die
Dateien weiterhin widersprechen. Und bei Unklarheit wird markiert, nicht
entschieden.

## Lint

Der Lint läuft deterministisch, ohne Modell: fehlendes `type`, Seite ohne
`sources`, verwaiste Seite, toter Link, falsch gezählte Konflikte, lange
unberührte Seite, abgelaufenes `stale_after`, in geteilten Bereichen zusätzlich
die verletzte Leserichtung. In Projekt-Bundles kommen die beiden
`realization`-Prüfungen dazu.

In loomux gibt es dafür seit Stufe 3c **zwei Regelsätze nebeneinander**, weil
Texte, Regelnamen und Auslöser der beiden Referenzen voneinander abweichen:

- **Die Go-Form aus Stufe 1a** hinter `loomux lint <datei>` (eine Seite),
  `loomux wiki-gate` (das Bündel im Tor, zusammen mit der Driftprüfung) und
  der Lane `lint/wiki`: fünf Regeln, `missing-type`, `conflict-count`,
  `dead-link`, `orphan` und `outside-area` für ein Ziel außerhalb des
  Bündels. Dabei bleibt es, entschieden am 2026-09-23.
- **Die zwölf Regeln von `lint.py`** als eigener Regelsatz hinter
  `loomux lint --scope <bereich>|all`, über jedes registrierte Bündel,
  darunter `no-sources`, `untouched`, `stale`, `wrong-direction` und die
  beiden `realization`-Prüfungen `implemented-without-commit` und
  `long-planned`.

Daneben prüft `loomux brain check file|bundle|all` Seiten, Bündel und
Föderation auf drei Achsen: OKF-Form, Hausregeln, Föderation. Es steht unter
`brain`, weil `loomux check all` die Prüfkette ist; `check code` ist
entfallen. Die vier Seitentypen bleiben, aber der Einzelseiten-Lint fragt eine
eigene Liste statt das Manifest: `Topic` geht als `topic` durch, `Source`,
`Entity` und `Synthesis` melden je eine Warnung `unknown document type`, die
nichts aufhält, weil das Tor nur bei `Error` abbricht (`_schema.md`).

**Lint prüft die Struktur, [Brain Maintenance](brain-maintenance.md) die
fachliche Aktualität.** Eine gestern geschriebene Seite kann überholt sein, eine
hundert Tage alte weiterhin stimmen; deshalb ist der Inhalts-Hash das starke
Signal und das Datum nur ein Hinweis. Genau eine Lint-Frage braucht Verstehen
und bleibt Aufgabe eines Modells: Welche Begriffe tauchen häufig auf, haben aber
keine eigene Seite?

## Zwei Schreibwege, verschieden gesichert

| Weg | Wer schreibt | Sicherung |
|---|---|---|
| **Ingest** | Claude, als gewöhnliche Dateien | menschliche Begleitung plus Git-Historie; **kein** Prüffall |
| **Wartung** | Code, nach Freigabe | Prüffall, Evidenzbindung, Freigabe, Protokoll, isolierter Commit |

Der Unterschied ist beabsichtigt: Ingest ist der Vorgang, bei dem Wissen
überhaupt erst entsteht — ihn über Fälle zu führen hieße, sich jede einzelne
Seite selbst zu genehmigen. Wartung dagegen ändert Bestehendes, oft gegen den
Wortlaut einer bereits geprüften Aussage.

Der isolierte Commit der Wartung ist in loomux `loomux approve`: ein Commit
genau der berührten Pfade, über einen eigenen Index auf den aktuellen Ref des
Tresors, ohne eigenen Zweig und ohne den Index des Nutzers anzufassen.

## Die Wiki-Werkzeuge

`loomux wiki init|types|retype` bringt die drei Wiki-Werkzeuge der Referenz:
`init` legt ein Bündel an, `types` zählt die Seitentypen über alle Bereiche,
`retype` benennt einen Typ in einem Bündel um. Zählen und Anlegen sind keine
eigenen Befehle. Einen Bereich samt Wiki-Gerüst meldet `loomux area add` an;
seine Hook-Hälfte bleibt `loomux init` aus Stufe 4 vorbehalten.

## Skills

Vier Prosa-Skills ohne Sonderrechte: `brain:research` (Suchleiter über die
Scopes, Belege mit Quellenangabe, **ausdrücklich benennen, was nicht gefunden
wurde**, Angebot einer Synthese als Entwurf), `brain:ingest` (der Ablauf oben),
`brain:wiki-plan` (ermittelt Nachzuziehendes, **schreibt nichts**),
`brain:land` (Soll auf Ist nach dem Merge). Ein Skill kann keinen Weg
beschreiten, den ein Mensch am selben Ort nicht auch hätte — er macht Abläufe
verlässlich, nicht mächtiger.

Stufe 4 (`docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`) zählt
fünf Brain-Skills. Seit 4a-2 schreibt `loomux init` sie in einen Wirt, auf
loomux-Befehle umgeschrieben und ins Englische übersetzt, sonst inhaltlich
gleich; in einem Wirt in Gebrauch ist noch keiner.

Quellen: Architektur-Design; Design von Stufe 3 in loomux.

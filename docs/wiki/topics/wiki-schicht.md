---
type: Topic
title: Die Wiki-Schicht
description: Schema, Ingest, Konfliktform, Lint — und die zwei Schreibwege ins Wiki.
open_conflicts: 0
realization: implemented
implemented_in: e26dc63
sources:
  - id: architektur-spec
    resource: brain://project/ultra-brain/docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md
    doc_id: 01M0QGS594F2KCWTWK9XV07M05
    content_hash: "sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff"
    revision: 3
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

`brain lint` läuft deterministisch, ohne Modell: fehlendes `type`, Seite ohne
`sources`, verwaiste Seite, toter Link, falsch gezählte Konflikte, lange
unberührte Seite, abgelaufenes `stale_after`, in geteilten Bereichen zusätzlich
die verletzte Leserichtung. In Projekt-Bundles kommen die beiden
`realization`-Prüfungen dazu.

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

## Skills

Vier Prosa-Skills ohne Sonderrechte: `brain:research` (Suchleiter über die
Scopes, Belege mit Quellenangabe, **ausdrücklich benennen, was nicht gefunden
wurde**, Angebot einer Synthese als Entwurf), `brain:ingest` (der Ablauf oben),
`brain:wiki-plan` (ermittelt Nachzuziehendes, **schreibt nichts**),
`brain:land` (Soll auf Ist nach dem Merge). Ein Skill kann keinen Weg
beschreiten, den ein Mensch am selben Ort nicht auch hätte — er macht Abläufe
verlässlich, nicht mächtiger.

Quelle: [Architektur-Design ultra-brain](../sources/architektur-spec.md).

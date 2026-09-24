---
type: Topic
title: Grundsätze und Vertrauenskette
description: Die sechs Grundsätze, die Arbeitsteilung zwischen Code, KI und Mensch, und die vier Fehlerstellen.
open_conflicts: 0
realization: implemented
implemented_in: 859baed
sources:
  - id: architektur-spec
    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
    doc_id: 01M39G4J14CK311B66GRSAK7HQ
    content_hash: "sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff"
    revision: 3
---

## Sechs Grundsätze

1. **Markdown ist die einzige Wahrheit.** Index, Graph, Zustandsdatenbank und
   Web-App sind abgeleitete Sichten und jederzeit löschbar.
2. **Deterministisch, wo es geht — und dann reproduzierbar.** Ein Modell läuft
   nur dort, wo Verstehen nötig ist; alles andere liefert bei gleicher Eingabe
   byteweise gleiche Ausgabe.
3. **Der Mensch entscheidet, die KI liefert.**
4. **Lokal zuerst.** An ein Cloud-Modell gehen nur ausgewählte Ausschnitte.
5. **Ausfälle degradieren, sie blockieren nicht.**
6. **Rohquellen sind unantastbar.** Die KI liest sie und schreibt ausschließlich
   in die Wiki-Schicht.

## Wer was tut

Ein Modell kommt im Alltag an genau **zwei** Stellen vor: beim Ingest und beim
Prüfvorschlag. Suchen, Katalogisieren und Scoping kommen ohne aus; Schreiben,
Hashfortschreiben und Protokollieren sind Code; Entscheiden ist Mensch. Ist das
lokale Modell eingeschaltet, kommen zwei folgenlose Stellen dazu —
Ablagevorschlag beim Import und Ein-Satz-Beschreibungen für Kataloge.

Für den Entwurf selbst gilt dieselbe Trennung noch einmal enger:
[Gegenprüfung vor jeder Designempfehlung](../syntheses/gegenpruefung-vor-jeder-designempfehlung.md)
hält fest, dass eine Empfehlung erst gegengeprüft wird, bevor sie den Menschen
erreicht.

## Die Vertrauenskette

Steigender Prüfaufwand, zunehmende Nähe zur Quelle:

**orientieren** (Katalog) → **verdichtet lesen** (Wiki) → **breit suchen**
(Rohquellen) → **am Original prüfen**.

Der Satz, der die Kette begründet: *Eine Quellenangabe beweist nur, dass eine
Quelle gefunden wurde* — nicht, dass sie aktuell ist, richtig verstanden wurde
oder die Aussage stützt.

## Die vier Fehlerstellen

| # | Fehler | Gegenmittel |
|---|---|---|
| 1 | Auswahlfehler: Quelle wird nicht gefunden | Hybridsuche, Fragensatz-Messung |
| 2 | Kontextfehler: falsch zugeschnitten übergeben | Chunking-Messung, Abschnittslesen |
| 3 | Generierungsfehler: falsch interpretiert | Quellenangaben, Vertrauenskette |
| 4 | Kompilierungsfehler: falsche Verdichtung im Wiki | Ingest gegen Original, Konflikte markieren, Wartung |

Stelle 4 existiert **nur**, weil es eine generierte Wissensschicht gibt. Sie
wird bewusst in Kauf genommen und mit drei Riegeln entschärft: immer gegen die
Originalquelle schreiben, Widersprüche markieren statt auflösen, Prüflauf plus
lesbares Protokoll. Beseitigt ist sie damit nicht — das Wiki ist eine
Leseschicht über den Notizen, nie ihr Ersatz.

Wie diese Riegel konkret aussehen, steht unter [Die Wiki-Schicht](wiki-schicht.md)
und [Brain Maintenance](brain-maintenance.md); die Quelle ist
[Architektur-Design ultra-brain](../sources/architektur-spec.md).

---
type: Topic
title: Grundsätze und Vertrauenskette
description: Die sechs Grundsätze aus ultra-brain, die Grundsätze der Fusion, die Arbeitsteilung zwischen Code, KI und Mensch und die vier Fehlerstellen.
open_conflicts: 0
realization: in_progress
sources:
  - id: architektur-spec
    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
    doc_id: 01M39G4J14CK311B66GRSAK7HQ
    content_hash: "sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff"
    revision: 3
  - id: fusion-spec
    resource: brain://project/loomux/docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md
    doc_id: 01M39G4J1486TZBN5SM5489MZS
    content_hash: "sha256:6fb778dd75759468370010fea63889b110dde93d03b7f0986894f5c371c65e00"
    revision: 3
---

## Sechs Grundsätze

Aus dem Architektur-Design ultra-brain übernommen.

1. **Markdown ist die einzige Wahrheit.** Index, Graph, Zustandsdatenbank und
   Web-App sind abgeleitete Sichten und jederzeit löschbar.
2. **Deterministisch, wo es geht — und dann reproduzierbar.** Ein Modell läuft
   nur dort, wo Verstehen nötig ist; alles andere liefert bei gleicher Eingabe
   byteweise gleiche Ausgabe.
3. **Der Mensch entscheidet, die KI liefert.**
4. **Lokal zuerst.** An ein Cloud-Modell gehen nur ausgewählte Ausschnitte.
5. **Ausfälle degradieren, sie blockieren nicht** — in der Wissensschicht.
   Fällt qmd, der Index oder das Modell aus, wird loomux unbequemer und sagt
   das laut, nie unbenutzbar und nie stumm leer. **Wächter und Tore sind
   ausgenommen, sie scheitern geschlossen:** Ein Wächter, der bei kaputter
   Konfiguration durchlässt, vergäbe Schreibrechte genau dann, wenn niemand
   sie prüfen kann, und ein grünes Tor ohne Arbeit täuscht Sicherheit vor.
   Entschieden am 2026-09-24 (Fusions-Spec, „Fehlerverhalten“).
6. **Rohquellen sind unantastbar.** Die KI liest sie und schreibt ausschließlich
   in die Wiki-Schicht.

## Grundsätze der Fusion

Die Fusions-Spec setzt für loomux als Ganzes eigene Regeln:

- **Ein Repo, ein Go-Modul, ein Binary `loomux`**; alles sind Unterbefehle.
  Kein Python bleibt im Produkt — weder als Laufzeit noch als Hook noch als
  Werkzeugskript.
- **Nichts wird ersatzlos gestrichen.** Was eine der beiden Altseiten kann,
  kann loomux am Ende auch, oder die Abweichung steht mit Begründung und
  Freigabe in einer Liste.
- **Harter Schnitt:** neues Konfigformat, keine Rückwärtskompatibilität zur
  Laufzeit.
- **Windows zuerst.** POSIX baut; Worktree-Spiegel und Job Objects sind
  Windows-only und per Build-Tag getrennt.
- **Startzeit-Regel:** Kein `init()` und keine Paketvariable parst
  eingebettete Daten; geladen wird beim ersten Gebrauch, weil jedes
  importierte Paket bei jedem Hook-Aufruf mitläuft.
- **`hooks` importiert nie `serve`.** Der Pfad an jedem Edit hängt nicht an
  einem laufenden Dienst.
- **`.loomux/config.toml` ist für Agenten nie beschreibbar.** Sie bestimmt
  die Schreibrechte und trägt die Policy; Agenten schlagen Änderungen vor,
  der Mensch schreibt sie.
- **Kein grünes Tor ohne Arbeit.** Eine Lane, die nicht laufen kann, ist rot
  und nennt den Installationsbefehl.

## Wer was tut

Ein Modell kommt im Alltag an genau **zwei** Stellen vor: beim Ingest und beim
Prüfvorschlag. Suchen, Katalogisieren und Scoping kommen ohne aus; Schreiben,
Hashfortschreiben und Protokollieren sind Code; Entscheiden ist Mensch. Ist das
lokale Modell eingeschaltet, kommen zwei folgenlose Stellen dazu —
Ablagevorschlag beim Import und Ein-Satz-Beschreibungen für Kataloge.

**In loomux ist das lokale Modell seit 4c-1 gebaut, die Selbstnutzung steht
aus.** Vorschläge für `local_only`-Fälle in `reconcile` kommen über Ollama auf
Loopback, seit 4d auch der Kopfsatz (`describe`) und der Ablagevorschlag
(`place`) in `convert`; ohne eingeschaltetes Modell öffnet ein `local_only`-Bereich seinen
Fall ohne Vorschlag. Ein Ausfall des Modells soll
zu einem Fall ohne Vorschlag führen, nie in die Cloud.

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
und [Brain Maintenance](brain-maintenance.md), die Stufen unter
[Die Stufen und ihre Abnahme](scheiben-und-abnahme.md); Quellen sind das
Architektur-Design ultra-brain und die Fusions-Spec.

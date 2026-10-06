---
type: Topic
title: Grundsätze und Vertrauenskette
description: Die sechs Grundsätze des Wissenssystems, die Grundsätze der Fusion, die Arbeitsteilung zwischen Code, KI und Mensch und die vier Fehlerstellen.
open_conflicts: 0
realization: in_progress
sources:
  - id: architektur
    resource: brain://project/loomux/docs/de/architecture.md
    doc_id: 01M47W91R5WKHDNYTXW8ZXTCSW
    content_hash: "sha256:7e34bcbe26fc6c0977631769395ca6f97d8bea9be643fbe4641809bb35266fbf"
    revision: 2
---

## Sechs Grundsätze

Aus dem Architektur-Design des Vorgängers übernommen; es liegt heute in den
Arbeitspapieren des Archiv-Release `archive/parity-recordings`. Keine
Nutzerdoku unter `docs/de/` belegt diese Grundsätze bisher; `architecture.md`
steht als formale Quelle da, weil sie die Säulen des Systems beschreibt.

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

Die Fusions-Spec (`docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`
in den Arbeitspapieren des Archiv-Release `archive/parity-recordings`) setzt
für loomux als Ganzes eigene Regeln; auch sie belegt keine Nutzerdoku:

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

**In loomux ist das lokale Modell seit 4c-1 gebaut und selbst genutzt
(2026-09-29).** Vorschläge für `local_only`-Fälle in `reconcile` kommen über Ollama auf
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
[Die Stufen und ihre Abnahme](scheiben-und-abnahme.md). Die Grundsätze, die
Vertrauenskette und die vier Fehlerstellen sind aus dem Architektur-Design und
der Fusions-Spec des Vorgängers verdichtet (heute in den Arbeitspapieren des
Archiv-Release `archive/parity-recordings`); `docs/de/architecture.md` beschreibt
nur die Säulen des Systems und belegt sie nicht.

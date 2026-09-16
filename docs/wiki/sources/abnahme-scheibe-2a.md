---
type: Source
title: Abnahme der Scheibe 2a
description: Der erste Lauf außerhalb des Arbeitsbereichs — dreizehn Befunde, von denen kein einziger im Testlauf sichtbar war.
open_conflicts: 0
sources:
  - id: abnahme-scheibe-2a
    resource: brain://project/ultra-brain/docs/.superpowers/plans/2026-08-20-scheibe-2a-abnahme.md
    doc_id: 01M0QGS5931BKD9QS6B8C2TNCT
    content_hash: "sha256:71c8d0e5c0196b026a5b96125297b8f7662829501f8e1f2e8efebc36065a5027"
    revision: 1
---

Protokoll des ersten Laufs gegen die echten Bestände, mit einer Regel, die das
Protokoll erst brauchbar macht: **Kein Produktivcode wurde angefasst — auch
dort nicht, wo dieser Lauf einen Fehler gefunden hat.**

## Die schwersten Befunde

- **B1 — die Suche stürzte auf Windows an einem Gedankenstrich ab.** Der
  allererste echte Suchaufruf endete mit einem Traceback: Ein Seitentitel
  enthielt ein `→`, die Standardausgabe war cp1252, und das Programm brach
  **mitten in der Trefferliste** ab — drei Treffer waren gedruckt, der vierte
  riss alles mit. Die Scheibe hatte rund 250 grüne Tests und einen Commit namens
  „pin non-ASCII output"; keiner davon berührte die Kodierung der Ausgabe im
  echten Prozess.
- **B4 — der Abgleich der Ausschlussliste bewirkte nichts.** Erwartet wurden 178
  Dateien mehr; eingetreten ist: nichts. Die Suchmaschine betritt
  Punktverzeichnisse gar nicht. Die eigentliche Überraschung stand daneben:
  **61 Dateien standen im eigenen Index und konnten von der Suche nie gefunden
  werden** — Specs und Pläne, genau die Dokumente, die der Vertrag zu Quellen
  erklärt. Und die Divergenzmeldung schwieg dazu, weil sie nur die eine Richtung
  kannte.
- **B5 — der Ausschluss der eigenen Artefaktnamen kostete 22 fremde Wiki-Seiten**
  — in einem Wiki genau die Navigationsebene, in beiden Indizes unsichtbar.
- **B6 — der Entdopplungspfad war im Betrieb unerreichbar**, und die Lücke lag
  **unter einem grünen Test**: Die Fixture schrieb ein Register, das der Indexer
  gar nicht erzeugen kann. *Der Test ist nicht falsch — er ist unverankert.*
- **B3 — auf dem Vorgabeprofil gibt es kein „nichts gefunden".** 75 % auf eine
  Zeichenfolge, die im Bestand nicht vorkommt.

## Was der Nachtrag nach der Nacharbeit noch fand

- **B9 — die neue Profilvorgabe kostete den Nutzenpunkt die Hälfte.** Die Seite,
  die die Frage im Titel trägt, verschwand aus den ersten drei; der Reranker war
  das, was sie nach oben gezogen hatte. Und die Begründung im Code („die
  100–500 ms sind es nicht wert") ließ sich warm gar nicht messen: 3,20 s gegen
  3,08 s sind Rauschen. Gespart wird das **kalte** Laden des Modells — eine
  Aussage über den ersten Aufruf, bezahlt mit jeder Anfrage.
- **B10 — geteilte Inhaltshashes gibt es, nur nicht dort, wo hingesehen wird.**
  Zwei byteweise identische Dateipaare, aber **über Bereichsgrenzen hinweg**,
  während die Ersatzmeldung je Bereich nachsah. Der Satz dazu: *Die bekannte
  Einschränkung sagt, das Problem käme zurück, wenn jemand eine Spiegelung
  registriert — dann aber mit einem Bestand als Beleg. Der Beleg lag schon vor,
  ungesehen.*
- **B11 — dieselbe Korrektur vervielfachte nebenbei die Verweisauflösung**, von
  7 auf 157 aufgelöste Links: Die Navigationsseiten sind nicht nur Quellen,
  sondern **Ziele**. Ein als Sichtbarkeitsfrage protokollierter Befund war
  zugleich, unbemerkt, die zweitgrößte Ursache des schwersten offenen Befunds.
- **B12 — zwei Profile waren derselbe Aufruf.** Im ersten Protokoll standen sie
  als getrennte Messungen mit unterschiedlichen Zeiten; dieser Unterschied kann
  nichts als Rauschen gewesen sein.
- **B13 — die befürchtete Langsamkeit trat nicht ein:** 1,3 s. *Die
  Einschränkung beschreibt eine Sorge, die auf dieser Maschine nicht eintritt.
  Sie bleibt stehen — aber als Zahl, nicht als Vermutung.*
- **B8 — der Bestand steht nicht still.** Eine parallele Sitzung arbeitete im
  selben Baum. Jede Aussage der Form „zwei Läufe sind identisch" ist damit an
  einen Zeitpunkt gebunden.

## Der Beleg der Bestandsregel

Kein Schreibzugriff in einen der drei Bäume, belegt über `git status` vor und
nach dem Lauf plus die Suche nach eigenen Artefaktnamen. Der schärfste Beleg
ist ein anderer: Seit der Nacharbeit **liest** der Indexer die 22 verfolgten
Navigationsseiten des fremden Wikis — und `git status` bleibt leer. Er liest sie
und schreibt sie nicht.

Siehe [Was erst die echte Umgebung zeigt](../topics/abnahmen-und-echte-umgebung.md).

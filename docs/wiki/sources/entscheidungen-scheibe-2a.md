---
type: Source
title: Entscheidungen der Ausführung, Scheibe 2a
description: Das Ledger der Rulings — was während der Ausführung entschieden wurde, warum, und was es kostet, wenn es falsch war.
open_conflicts: 0
sources:
  - id: entscheidungen-scheibe-2a
    resource: brain://project/ultra-brain/docs/.superpowers/plans/2026-08-20-scheibe-2a-entscheidungen.md
    doc_id: 01M0QGS593QJAQMMK5N520KBND
    content_hash: "sha256:0c7bfd21611ad0bb511986c9e98735c0d54fa3f1c259308f856978faa913606c"
    revision: 1
---

Eine eigene Datei bewahrt, was während der Ausführung entschieden wurde, ohne
in die Spec zu gehören: Auflösungen von Widersprüchen im Plan, Abweichungen vom
Plantext und die Gründe. Der Zweck ist ausdrücklich, dass es die
Arbeitsverzeichnisse überlebt.

**Jede Zeile nennt drei Dinge: die Entscheidung, den Grund, und was sie kostet,
wenn sie falsch war.** Diese dritte Spalte ist das Ungewöhnliche — sie zwingt
dazu, den Preis eines Irrtums zu beziffern, bevor man weiß, ob man irrt.

## Wiederkehrende Muster

- **Der Plan wird während der Ausführung nicht umgeschrieben.** Zweimal
  ausdrücklich so entschieden, mit derselben Begründung: Es hatte in einer
  früheren Aufgabe bereits Schaden angerichtet. Der Plan ist die Aufzeichnung
  dessen, was geplant *war*; die Abweichung steht im Ledger und in der Historie.
- **Gegen den Auftragsbrief entscheiden, wenn der Brief die Zusage bricht.** Ein
  wörtlich vorgeschriebener Wächter wurde durch einen ersetzt, der unter
  Optimierung nicht verschwindet — *die Scheibe existiert, um stille Fehlschläge
  in laute zu verwandeln; ein Wächter, der wegoptimiert wird, ist derselbe
  Fehler in klein.*
- **Messen statt vermuten, auch beim Verwerfen.** Das Escapen von
  Shell-Metazeichen wurde **gemessen** verworfen, nicht vermutet — die nötige
  Zahl der Escape-Ebenen hing davon ab, ob ein Argument zufällig ein Leerzeichen
  enthielt.
- **Eine Hilfe-Ausgabe, die für einen erfundenen Unterbefehl grün wäre, belegt
  nichts.** Gegen die Suchmaschine gemessen und deshalb ein anderer Test.
- **Halb geschlossene Lücken sind schlechter als offene** — *sie sehen erledigt
  aus, und der nächste Leser prüft sie nicht noch einmal.* Deshalb wurde eine
  Sicherheitslücke eine Ebene tiefer in derselben Runde geschlossen statt
  vertagt.
- **Ununterscheidbarkeit als Teil des Gatters:** Ein unbekannter Scope und ein
  bekannter-aber-unsichtbarer Scope erzeugen dieselbe Meldung — eine
  unterscheidbare Fehlermeldung verriete dem Cloud-Modell die Existenz genau des
  Bereichs, den der strengste Modus verbergen soll
  ([Datenschutz und Kanäle](../topics/datenschutz-und-kanaele.md)).
- **Ein Datenschutzmuster darf nicht vom Dateisystem des Lesers abhängen.** Die
  Groß- und Kleinschreibung wurde plattformunabhängig behandelt, weil ein
  eingechecktes Manifest geteilt wird und kein Test auf einem Rechner etwas über
  den anderen sagt. Übersperren ist die sichere Richtung.
- **Vertagen ist erlaubt, aber nur schriftlich.** Zwei Punkte wurden ausdrücklich
  parkiert — mit Begründung, Erreichbarkeitsabschätzung und der Auflage, sie als
  bekannte Einschränkung einzuchecken.

Siehe [Plan Scheibe 2a — Nacharbeit](plan-scheibe-2a-nacharbeit.md) und
[Was erst die echte Umgebung zeigt](../topics/abnahmen-und-echte-umgebung.md).

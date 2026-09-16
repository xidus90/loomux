---
type: Source
title: Plan Scheibe 0 — Fundament ohne Code
description: Zwei Tore, der Fragensatz, der Spike zur Mehrsprachigkeit, der leere Vault und der Praxisvergleich.
open_conflicts: 0
realization: implemented
implemented_in: 7d3e267
sources:
  - id: plan-scheibe-0
    resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-18-scheibe-0-fundament.md
    doc_id: 01M0QGS5938AFTQ36ECK6FQ4VM
    content_hash: "sha256:f927f47a11773eae3c57d9a8416882a735e6dcf1f990ef69c222c77a5bd797eb"
    revision: 1
---

Sechs Aufgaben ohne eine Zeile Anwendungscode. Die Reihenfolge ist bewusst:
zuerst die zwei Tore, die das Projekt kippen könnten, dann der Vault, dann die
Messung gegen den Zustand ohne System.

## Die Bestandsregel

**Der neue Vault startet leer. Die bestehenden Bestände werden weder verschoben
noch zusammengeführt noch umsortiert.** Erlaubt ist ausschließlich lesendes
Indexieren — ohne das würde der Spike an erfundenen Testdateien messen statt an
echtem Wissen. Der Satz, der die Regel gegen Ausreden schützt: *Wenn eine
Aufgabe das nahelegt, ist die Aufgabe falsch, nicht die Regel.*

Ausdrücklich getrennt gehalten werden dabei zwei Begriffe, die sich leicht
verwechseln: eine **qmd-Sammlung** ist ein Ordner, den die Suchmaschine liest;
ein **Bereich** ist eine Einheit des Systems mit Manifest, Scope und Wiki. Die
drei Altbestände wurden Sammlungen, nicht Bereiche.

## Der Fragensatz entsteht vor der Messung

Sonst wird er unbewusst an die Ergebnisse angepasst. 30 Fragen in vier Sorten
(8 exakt, 8 Umschreibung, 6 gemischt, 8 sprachübergreifend), jede mit der Datei,
die getroffen werden müsste — und **jede vom Menschen bestätigt**: Eine Frage
taugt nur, wenn vorher bekannt ist, welche Datei die Antwort enthält, und wenn
man sie im Alltag tatsächlich so stellen würde.

## Was der Spike ergab

23/30 für **beide** Modelle, sprachübergreifend 7/7 gegen 6/7 — kein
unterscheidbarer Abstand. Die Erwartung, das englischoptimierte Standardmodell
falle sprachübergreifend ab, war damit widerlegt.

Zwei methodische Lehren stehen im Plan festgehalten:

- **Der Entscheidungstabelle fehlte die Zeile „kein Unterschied" — und genau
  dieser Fall trat ein.** Eine Tabelle ohne diese Zeile zwingt dazu, das
  nächstliegende Feld anzukreuzen, und erzeugt eine Begründung, die zum Ergebnis
  nicht passt. Nachgetragen wurde sie erst nach der Messung.
- **Was diese Messung grundsätzlich nicht kann:** Sie vergleicht zwei Pipelines,
  die sich in einem von drei Modellen unterscheiden. Ein gleiches Ergebnis
  beweist deshalb nicht, dass das Einbettungsmodell gleichgültig ist — es kann
  ebenso heißen, dass die Sprachbrücke gar nicht dort liegt. Das wurde später
  gemessen ([Suche, Profile und Messwerte](../topics/suche-und-profile.md)).

## Drei Fallstricke, die der Lauf zutage förderte

- **Das Einbettungsmodell wird über die Konfigurationsdatei gesetzt, nicht über
  die dokumentierte Umgebungsvariable** ([qmd](../entities/qmd.md)). Vor jedem
  Durchgang wird es ausdrücklich gesetzt und die geladene Fassung ins Protokoll
  übernommen — sonst misst „Durchgang A" still das Modell aus Durchgang B.
- **Trefferpfade werden über Dateiname plus Elternordner verglichen**, nicht
  über den vollen Pfad: Das Ausgabeformat kippt je nach Arbeitsverzeichnis.
- **Duplikate mussten vor der Messung ausgeschlossen werden.** In `space` liegt
  unter `.claude/worktrees/` eine byte-identische Spiegelung des Wikis; ohne
  Ausschluss konkurriert jede Zielquelle mit ihrem eigenen Klon um die ersten
  drei Plätze. Geprüft wurde das nicht nach Gefühl, sondern mit einem eigenen
  Schritt: jede Zielquelle muss **genau einmal** im Index stehen, sonst wird
  nicht weitergemessen.

## Der Unverändert-Nachweis

Belegt wurde die Bestandsregel mit `git status` plus den Änderungszeiten der
Notizen selbst, vorher und nachher. **Ein Dateizähler taugt dafür nicht** — eine
parallel laufende fremde Sitzung änderte die Gesamtdateizahl, ohne eine Notiz zu
berühren.

## Der Praxisvergleich

Fünf echte Fragen, je einmal mit und ohne System, Token und Zeit erfasst. Die im
Plan festgehaltene Erwartung ist ungewöhnlich ehrlich: *Zeigt die Messung keinen
Unterschied, ist das ein Ergebnis — kein Grund, sie zu wiederholen, bis sie
gefällt.* Genau so kam es
([Suche, Profile und Messwerte](../topics/suche-und-profile.md)).

## Was ausdrücklich nicht passierte

Kein Zusammenführen von Beständen, keine Aufnahme der Altbestände als Bereiche,
kein Wiki, kein Daemon, keine MCP-Werkzeuge, kein Python, keine Konverter, keine
Web-App.

Siehe [Die Scheiben und ihre Abnahmen](../topics/scheiben-und-abnahme.md).

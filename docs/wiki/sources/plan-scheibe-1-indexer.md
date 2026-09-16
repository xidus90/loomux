---
type: Source
title: Plan Scheibe 1 — Indexer
description: Der deterministische Indexer, seine Reproduzierbarkeitsfalle und was der Rauchtest am echten Bestand zutage förderte.
open_conflicts: 0
realization: implemented
implemented_in: e1b9542
sources:
  - id: plan-scheibe-1
    resource: brain://project/ultra-brain/docs/.superpowers/plans/2026-08-19-scheibe-1-indexer.md
    doc_id: 01M0QGS593D5A8B9E3P0SV6GD9
    content_hash: "sha256:1c38fd1e30390ffb31ab13083d09006e6f6a49996776312c3f781da0eb09d90e"
    revision: 1
---

Ein Programm, das durch die registrierten Bereiche läuft und daraus `index.md`
je Ebene, `graph.json` und das Identitätsregister erzeugt. Kein Modell, kein
Daemon, kein Netz — reine Textverarbeitung. Es wird aufgerufen und ist danach
beendet.

**Reproduzierbarkeit ist die Hauptanforderung**, nicht eine Eigenschaft neben
anderen: gleiche Eingabe → byteweise gleiche Ausgabe. Daraus folgen stabile
Sortierung, keine Zeitstempel im Inhalt, LF beim Schreiben unabhängig von der
Plattform und Vorwärts-Schrägstriche relativ zur Bereichswurzel.

## Die Falle, die man kennen muss

Eine `doc_id` ist eine ULID und **enthält einen Zeitstempel** — zwei Vergaben
derselben Datei ergäben verschiedene Kennungen. Das Fertig-Kriterium hält nur
deshalb, weil der zweite Lauf das Register *liest* und Kennungen ausschließlich
für **neue** Dateien vergibt. Wer die Kennung bei jedem Lauf neu erzeugt, bricht
die Hauptanforderung der Scheibe.

## Der Rauchtest am echten Bestand

Gegen eine Kopie eines echten Bestands (1162 Markdown-Dateien): 144 Dokumente,
34 Kataloge, 19,5 s im ersten und 15,2 s im zweiten Lauf, **byteweise identische
Ausgabe**. Von 979 Links wurden 959 aufgelöst; die übrigen 20 sind zu Recht
offen (externe URLs, Anker, Nicht-Markdown-Ziele, drei tote Links im Bestand).

Drei Befunde kamen dabei heraus, die keine Testsuite gefunden hätte:

- **Ein Graph ohne eine einzige Kante sieht aus wie ein Erfolg.** Mit der
  Repo-Wurzel als Bereichswurzel kamen aus denselben Notizen **0 Kanten**
  heraus, mit der Vault-Wurzel **959** — die Notizen verlinken wurzelabsolut auf
  die Wurzel des Obsidian-Vaults. Der Indexer verhielt sich beide Male richtig;
  er sagte nur nirgends, dass er gerade jeden Link verworfen hatte. Daraus wurde
  die Anforderung, eine **Auflösungsquote** auszugeben
  ([Datenmodell und Bereiche](../topics/datenmodell-und-bereiche.md)).
- **Der pauschale Ausschluss des Arbeitsordners nahm zu viel heraus** — in einem
  Bestand 178 Dateien, darunter Spec und Plan. In dieser Form indizierte der
  Indexer die eigene Spec des Projekts nicht.
- **Der erzeugte Wurzelkatalog trägt keine Orientierung.** Die handgeschriebene
  Einleitung des Vaults hatte im erzeugten Format keinen Platz. Entschieden
  wurde, den Abschnitt umziehen zu lassen statt dem Indexer etwas beizubringen —
  und ausdrücklich festgehalten, dass das den *nächsten* Fall nicht löst.

Bemerkenswert an dieser Stelle ist der Umgang mit dem offenen Punkt: Der Vault
trug eine handgeschriebene `index.md`, die der Indexer überschrieben hätte. Der
Plan legte fest: **nicht stillschweigend überschreiben** — entscheiden.

## Bewusst offen gelassen

- **Eine gelöschte oder umbenannte Datei verliert ihre `doc_id` endgültig**, weil
  das Register ausschließlich aus den Dateien des aktuellen Laufs gebaut wird.
  Das Versprechen „stabile Identität" gilt damit nur für Dateien, die sich nie
  bewegen — und deckt stillschweigend das Umbenennen mit ab, das viel häufiger
  vorkommt als das Löschen. Ob eine entfernte Zeile einen Grabstein erhält,
  gehört der Wartungsscheibe ([Brain Maintenance](../topics/brain-maintenance.md)).
- **`graph.json` deklariert nicht, dass die Kantenliste eine Menge ist.** Die
  Zusage stand nur als Kommentar beim Erzeuger, nicht dort, wo ein Verbraucher
  sie liest.

Siehe [Die Scheiben und ihre Abnahmen](../topics/scheiben-und-abnahme.md).

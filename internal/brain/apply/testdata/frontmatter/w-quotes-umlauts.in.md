---
type: Source
title: Plan Scheibe 2b — Messwerk
description: "`brain bench` misst Trefferqualität und Latenz — vier Module, die die Suchmaschine nicht kennen."
open_conflicts: 0
realization: implemented
implemented_in: 6f65c70
sources:
  - id: plan-scheibe-2b
    resource: brain://project/loomux/docs/de/configuration.md
    doc_id: 01M0QGS593NNV4TQ4XFBRQWD2K
    content_hash: "sha256:e82b5213f0a5b0e3570941b3f6ae5b19d908c10846bc2d8067f5fde92ce65586"
    revision: 1
---

Ein Messwerk, das Trefferqualität an einem geprüften Fragensatz und Latenz je
Operation misst und beides protokolliert — **damit die offenen Entscheidungen
an Zahlen fallen statt an Vermutungen.**

## Die Bauform

Vier Module, jedes mit einer Aufgabe und **ohne Kenntnis der Suchmaschine**. Das
Messwerk bekommt Funktionen hereingereicht — eine Frage-Funktion, eine Uhr —,
**nie einen Prozess**. Erst der CLI-Unterbefehl setzt beides zusammen, holt den
Umgebungskopf und schreibt Protokoll und JSON.

Der Protokollkopf nennt die drei Modelle der Kette in der Reihenfolge, die ein
Leser erwartet: was einbettet, was die Frage erweitert, was neu sortiert. Ohne
diese Angabe ist eine Messreihe später nicht mehr einzuordnen.

## Die Abnahme ist Handarbeit

Der letzte Task enthält keinen Code: vier Qualitätsläufe, ein Latenzlauf, und
dann trägt die Spec ein, was gemessen wurde. Zwei Regeln stehen dabei
ausdrücklich fest:

- **Bricht ein Lauf am Fragensatz ab, ist der Fragensatz zu reparieren — nicht
  die Prüfung.** Abgewanderte Belege sind nach so langer Zeit zu erwarten.
- **Den Aufrufaufschlag der Suchmaschine als Zahl notieren, nicht als Eindruck:**
  gemessene Stichwortzeit gegen die Zeit, die die Suchmaschine selbst meldet;
  die Differenz ist der Prozessstart. Aus dieser Zahl fiel die Entscheidung für
  den gehaltenen Unterprozess ([qmd](../entities/qmd.md)).

Was dieser Lauf ergab, steht unter
[Suche, Profile und Messwerte](../topics/suche-und-profile.md); der eingecheckte
Prüfbestand daneben unter [Plan Prüfkorpus v1](plan-pruefkorpus-v1.md).

---
type: Source
title: Plan Scheibe 2a — Suchkette ohne Daemon
description: Die fünf schreibfreien Werkzeuge, die Naht zur Suchmaschine und die Einschränkungen, die 2a bewusst offen ließ.
open_conflicts: 0
realization: implemented
implemented_in: f6315d3
sources:
  - id: plan-scheibe-2a-suchkette
    resource: brain://project/ultra-brain/docs/.superpowers/plans/2026-08-20-scheibe-2a-suchkette.md
    doc_id: 01M0QGS593GGJ1X2Q2P0XJFVKW
    content_hash: "sha256:5dbd29fd127681cc63433ce94492a6f6a029e3d555339811c33c00dd70436507"
    revision: 1
---

Die fünf schreibfreien Werkzeuge laufen als direkte CLI über die drei externen
Bestände. **qmd bleibt die Suchmaschine, wird aber nie direkt angesprochen:**
Der Kern redet mit einer engen Schnittstelle, hinter der eine Umsetzung den
Prozess aufruft und eine zweite die Attrappe für Tests ist. Die Zusagen der
Spec liegen **oberhalb** dieser Naht, damit sie über jeder Umsetzung gelten.

## Die Falle, die man kennen muss

**Es gibt jetzt zwei Indizes über denselben Bestand** — unseren und den von
[qmd](../entities/qmd.md). Sie können auseinanderlaufen, und die Spec verlangt,
dass das **sichtbar** wird statt still zu bleiben. Wer beim Entdoppeln einen
unbekannten Pfad einfach verwirft, baut genau den stillen Fehler ein, den die
Scheibe verhindern soll: Der Treffer verschwände aus der Liste, ohne dass
irgendwo stünde, warum.

## Was von der Suchmaschine gemessen feststand

Der Plan hält ausdrücklich fest, was gegen qmd 2.8.3 geprüft und nicht geraten
wurde: JSON auf stdout, Diagnosezeilen auf stderr, ein leeres Ergebnis als
leeres Array **mit Rückgabewert 0** — und damit von einem Fehlschlag nur über
Rückgabewert und stderr zu unterscheiden. Genau das ist der Grund für die
Anforderung, leer von fehlgeschlagen zu trennen.

## Bewusst offen gelassen

- **`reindex` ist nicht mehr netz- und modellfrei.** Das Fertig-Kriterium der
  Scheibe 1 („zwei Läufe, byteweise identisch") gilt ab hier für den **eigenen**
  Teil der Ausgabe.
- **Die Wiederholung bei leerem Ergebnis kostet im Regelfall doppelte Zeit.**
- **Der Faktor, mit dem mehr Treffer angefordert werden als gebraucht, ist
  geraten** — und gehört an eine Zahl gebunden statt an eine Vermutung.
- **`brain` schreibt die Konfigurationsdatei eines fremden Werkzeugs.** Der
  Preis ist echt; drei Riegel dagegen (Sicherungskopie, Vertragstest,
  Beschränkung auf registrierte Bereiche) beseitigen das Risiko nicht.
- **qmd kennt je Sammlung genau ein Einschlussmuster.** Ein Manifest mit
  mehreren lässt sich nicht mitteilen; für die drei Altbestände folgenlos, für
  ein Code-Repo mit der Vorgabe aus der Spec nicht.
- **`status` startet einen Prozess je Bereich** — spürbar langsamer, aber
  ungemessen. Die Abnahme hat es später gemessen: 1,3 s
  ([Abnahme der Scheibe 2a](abnahme-scheibe-2a.md)).

Siehe [Datenschutz und Kanäle](../topics/datenschutz-und-kanaele.md) und
[Suche, Profile und Messwerte](../topics/suche-und-profile.md).

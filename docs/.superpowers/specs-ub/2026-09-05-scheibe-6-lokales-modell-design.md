# Scheibe 6 — Das lokale Modell: Entwurf

**Stand:** 2026-09-06 (Läufe 4 bis 10 eingearbeitet; Fassung vom 2026-09-05
stand auf Zahlen ohne Rohdaten); am 2026-09-11 in §3 um die zweite Quelle der
Datenschutzweiche ergänzt, das aktuelle Manifest des Bereichs
**Bezug:** Architektur-Spec 2026-08-18 §4.2, §6, §11, §12, §17, §18
(Entscheidungen 21, 31); Scheibe 5 (Brain Maintenance) §6, §7;
Python-nach-Go-Migration 2026-09-04 §3
**Messung:** `bench/scheibe-6-ollama/protokoll.md` §7–§16 (Läufe 4 bis 10:
4 850 Aufrufe, drei Stufen, fünf Pakete, zwei Temperaturen, vier
Prompt-Fassungen); §1–§6 halten die Vorläufe fest, deren Rohdaten teils fehlen
**Status:** Entwurf zur Freigabe

---

## 1. Warum

Ein Bereich auf `local_only` bekommt heute **keinen Vorschlag**:
[reconcile.py:670](../../../src/brain/maintenance/reconcile.py) legt den Fall
mit `manual = true` an und vermerkt „this area is local_only, so no skill path
is offered". Das ist keine Lücke, sondern der bewusst offene Platz für diese
Scheibe. Dasselbe gilt für zwei kleinere Rollen: Beim Import schlägt niemand
einen Ablageort vor, und `catalog.py` liest zwar `doc.description`, aber
niemand schreibt sie.

Die Architektur (§11) gibt dem lokalen Modell **genau drei Rollen** —
Vorschlaggeber für `local_only`, Ablagevorschläge beim Import,
Ein-Satz-Beschreibungen für die Kataloge — und verbietet ihm ausdrücklich das
Verdichten ins Wiki und das Beantworten von Wissensfragen.

## 2. Was der Spike entschieden hat

Gemessen wurde gegen den **echten Prüfer**: Ein Vorschlag gilt als bestanden,
wenn `read_proposal` + `check_evidence` ihn ohne Beanstandung annehmen — nicht,
wenn das Modell sich selbst für gut hält.

**Ein Modell, nicht drei.** Alle drei Rollen laufen auf derselben Stufe: die
schnellen unter einer Sekunde. Als Mediane über die drei Stufen, Lauf 4
(je 30 Läufe je Stufe und Temperatur): Beschreibung 431 ms (E2B) bis 776 ms
(12B), Ablage 546 ms (E2B) bis 927 ms (12B). Der Vorschlag braucht auch am
längsten Paket höchstens 1 314 ms (12B, Lauf 6). Die frühere Annahme, der
Vorschlaggeber brauche ein größeres Modell als die schnellen Rollen, stammte
aus einer Messung, die versehentlich das Denken mitmaß.

**`think: false` ist Pflicht, nicht Feinschliff.** Gemma 4 denkt per Vorgabe;
Ollama nennt die Fähigkeit in `ollama show` nicht einmal. Gemessen an einem
Vorschlag: 1 573–2 079 Denk-Token und 24–28 Sekunden gegen 56–66 Token und
1,1–1,3 Sekunden — **das Zwanzigfache an Zeit, kein Treffer mehr**. Der
Schalter steht im Aufruf, nicht in der Konfiguration; ein Modell, das ihn
ignoriert, ist für diese Rollen ungeeignet.

**Die Leiter, vier Hardwareklassen.** Belegung aus `ollama ps` über die
HTTP-Schnittstelle, also in **GiB** — die Fassung vom 2026-09-05 las die CLI,
die dezimale GB ausweist, daher stehen dort 1,8 / 3,1 / 7,8 für dieselbe
Belegung. Die Evidenzspalte ist die Vorschlagsrolle über **120 Läufe je
Stufe** (vier Pakete à 30) bei `temperature: 0`, daneben dieselbe Zahl bei
einer Temperatur von 0,8. Beide Spalten gelten für die Prompt-Fassung
`vorschlag-v4` (§5).

| Klasse | Modell (QAT, `UD-Q4_K_XL`) | Belegt (GiB) | Vorschlag | T 0 | T 0,8 |
|---|---|---|---|---|---|
| 4 GB | `gemma-4-E2B-it-qat` | 1,6 | 614–1 049 ms | **120/120** | 971/1 000 |
| 8 GB | `gemma-4-E4B-it-qat` | 2,9 | 563–782 ms | **120/120** | **988/1 000** |
| 16 GB | `gemma-4-12B-it-qat` | 7,2 | 1 038–1 314 ms | **120/120** | **150/150** |
| 24 GB | `gemma-4-12B-it-qat` (nicht 31B) | 7,2 | 1 038–1 314 ms | **120/120** | **150/150** |

**Die beiden Quotenspalten sind nicht spaltenweise vergleichbar.** In der
Spalte „T 0,8“ stehen E2B und E4B mit **1 000** Läufen (Protokoll §16), das
12B mit **150**. Als Wilson-Intervall heißt 150/150 nur **mindestens 97,5 %**
— nicht mehr, als die 971/1 000 des E2B (95,87 bis 97,97 %) ohnehin abdecken.
Dasselbe gilt für „T 0“: 120/120 heißt mindestens 96,9 %. Wer eine
12B-Zeile als Abnahmewert braucht, muss sie über 1 000 Läufe nachmessen.

**Warum das E2B nicht auch die 8-GB-Klasse bekommt, obwohl es schneller ist.**
Unter der ersten Prompt-Fassung kippte es am langen Paket auf **25/30**,
während E4B und 12B 30/30 hielten — und zwar in der schwereren Fehlerklasse:
alle fünf Fehlschläge waren `quote not found`, einer davon um genau ein
Leerzeichen, weil das Modell die Diff-Zeile lesbar gemacht statt kopiert hat.

**Die Fassung `vorschlag-v4` hat das geheilt** (Protokoll §13): am langen
Paket 30/30 statt 25/30. Damit war der Grund für die 8-GB-Zeile wieder
fraglich — 148/150 gegen 150/150 ist kein Unterschied, den man von Rauschen
trennen kann (Fisher, p = 0,50).

**Ein Lauf über je 1 000 Vorschläge hat es entschieden** (Protokoll §16):

| Stufe | Quote | 95-Prozent-Intervall |
|---|---|---|
| `gemma-4-E2B` | 971/1 000 | 95,87 bis 97,97 % |
| `gemma-4-E4B` | **988/1 000** | **97,91 bis 99,31 %** |

**p = 0,0107 — der Unterschied ist echt.** Wer 8 GB hat, kauft für 1,3 GiB
mehr Belegung rund 1,7 Prozentpunkte Trefferquote. Und er kauft sie an der
Stelle, die zählt: Beide Stufen scheitern **ausschließlich** am Zitieren, nie
an der Form; die 4-GB-Zeile behält das E2B, weil dort nichts anderes
hineinpasst, mit diesem Preis als benanntem.

**Verworfen, mit Grund:** `gemma-4-26B-A4B` (Mischexperte, 15 GB) besteht nur
**5 von 10** — wörtliches Zitieren profitiert nicht von aktiven Parametern.
`llama3.1:8b` schwankte zwischen 5/10 und 7/10. `glm-4.7-flash` lieferte
zerhacktes Deutsch bei 116 Sekunden. (Diese drei stammen aus den Vorläufen
und sind nicht nachgemessen; sie fielen deutlich genug durch.)

`gemma-4-31B` (18 GB dezimal, also rund 16,4 GiB) trifft 10/10, ist aber
dreimal langsamer als das 12B bei gleicher Quote. **Der zweite Grund in der
Fassung vom 2026-09-05 — „die freien 16 GB gehören qmd" — ist widerlegt:
qmd hält warm rund 3,0 GB.** An seine Stelle tritt ein gemessener: Das 31B
lädt neben qmd zwar mit
`100% GPU`, danach steht die Karte aber bei **24 056 von 24 576 MiB** —
**520 MiB frei**. Diese Zahl ist der Stand der **ganzen Karte**, nicht die
Summe aus Modell und qmd; Browser, Editoren und der Fenstermanager liegen
darin. Genau deshalb ist sie das brauchbare Maß: Auf einem Arbeitsrechner
bestimmt der Kartenstand die Reserve, nicht die Modellgröße.

**Vorgabe je Klasse ist eine Empfehlung, keine Erkennung.** Die Anwendung
schlägt beim Einrichten die Stufe vor, die zur gefundenen Karte passt; sie
wählt nichts hinter dem Rücken des Menschen.

## 3. Die Schnittstelle

Eine Schnittstelle, drei Umsetzungen, ausgewählt durch den Datenschutzmodus
(§11): **lokales Modell · Claude nach Bestätigung · Claude direkt.**

```
Proposer.propose(package)      -> Vorschlag oder None
Proposer.place(document)       -> scope-Vorschlag oder None
Proposer.describe(document)    -> ein Satz oder None
```

`None` heißt immer dasselbe: **Rückfall, kein Fehler** — manueller Fall, Datei
bleibt im Eingang, keine Beschreibung.

**Für `describe` ist der Rückfall kein Satz, sondern keiner.** Die Fassung vom
2026-09-05 nannte hier und in §5 die erste Zeile des Dokuments; gebaut ist sie
nicht, und sie wird auch nicht gebaut. Eine so gewonnene `description:` stünde
im Kopf ununterscheidbar neben einer, die das Modell geliefert und der Prüfer
angenommen hat — wer die Datei später liest, könnte die erfundene nicht mehr
als solche erkennen. Gebraucht wird sie ohnehin nicht: `catalog.py` setzt
nichts an die Stelle einer fehlenden Beschreibung, es lässt den Zusatz
` - {description}` hinter dem Link weg
([catalog.py:58](../../../src/brain/catalog.py)), und der Titel bleibt der
Linktext. Die Kopfzeile bleibt also weg, und der Katalog zeigt den Titel
allein.

Der Einbau in Scheibe 5 ist eine Zeile: Wo `reconcile` heute `manual = true`
setzt, fragt es künftig den Vorschlaggeber und setzt `manual` nur, wenn der
`None` liefert. Der Fall trägt weiterhin den Grund — jetzt in zwei Fassungen:
„kein Vorschlaggeber eingeschaltet" oder „Vorschlag verworfen".

**`manual` trägt die Datenschutzweiche nicht und hat sie nie getragen.** Die
Marke heißt „für diesen Fall steht kein Skill-Pfad offen", gleich aus welchem
Grund: geschlossener Bereich ohne lokalen Vorschlag — oder ein Vorschlag, den
die Evidenzbindung abweist, und den setzt `apply._refuse` in **jedem**
Datenschutzmodus, auch in `manual_cloud`. Eine Äquivalenz `manual ⇔ Bereich
ist local_only` galt also zu keinem Zeitpunkt.

Mit dem Zweig oben wird die Marke zusätzlich löchrig: Ein `local_only`-Fall
mit lokalem Vorschlag trägt `manual = false` und trüge sonst gar keine Marke
mehr, an der ein Wolkenablauf anhielte — er bekäme den Quelldiff eines
geschlossenen Bereichs zu sehen (§6). Der Fall führt deshalb den Modus des
Bereichs als eigenes Feld `local_only`, gesetzt unabhängig davon, ob eine
Antwort kam.

**Das Feld allein trägt die Weiche nicht, denn es ist eine Momentaufnahme.**
Drei Wege lassen es veralten: ein Fall, der geschrieben wurde, bevor es das
Feld gab; ein Bereich, der erst nach der Fallbildung auf `local_only`
umgestellt wurde und dessen Fälle erst ein späterer Lauf wieder besucht; und
ein Rückschreiben von `case.toml`, das das Feld verliert. `brain case` fragt
deshalb zusätzlich das **aktuelle Manifest** des Bereichs, gefunden über das
Register, wie es beim Aufruf steht. Die Halt-Zeile der Fallanzeige steht,
sobald eine der beiden Quellen `local_only` sagt — das Feld in `case.toml`
oder das Manifest —, und nicht an `manual`.

**Lässt sich der Modus nicht feststellen, gilt der Bereich als `local_only`.**
Unbestimmt heißt: Das Register nennt den Bereich nicht, der Bereich hat kein
Manifest, oder sein Manifest ist nicht lesbar. Eine Weiche, die öffnet, weil
ihr die Auskunft fehlt, ist keine. Es steht dann dieselbe Halt-Zeile, und
direkt unter ihr eine feste Zeile:

```
Datenschutzmodus unbekannt: Der Bereich ist nicht registriert oder hat kein lesbares Manifest und gilt deshalb als local_only
```

Sie sagt dem Leser, warum die Halt-Zeile an einem Bereich steht, den er als
offen kennen mag. Sie nennt weder Ursache noch Pfad, weil Go und Python ihre
Register- und Manifestfehler verschieden formulieren und die Ausgabe beider
Seiten gleich sein soll. Die Zusatzzeile hängt allein am unbestimmten Modus:
Sagt das Manifest `local_only`, fehlt sie; ist der Modus unbestimmt, steht sie
auch dann, wenn das Feld schon `local_only` sagt. Die Halt-Zeile selbst ist in
allen drei Fällen dieselbe,
weil der Prüfskill genau an ihr hält. Ein Manifest, das nicht als TOML liest,
erreicht die Anzeige heute gar nicht: Das Prüfzentrum wird über dieselben
Manifeste gefunden, und diese Suche bricht vorher mit einem Fehler ab. Die
Weiche behandelt den Fall trotzdem als unbestimmt, statt sich auf diese
Reihenfolge zu verlassen.

**Jede weitere Fallansicht schuldet dieselbe Weiche aus beiden Quellen**, auch
die geplante `GET /api/case/<id>` der Web-Front (Scheibe 7). Eine Ansicht, die
nur das Feld liest, zeigt genau die Fälle, die das Manifest schließt.

**Die Halt-Zeile allein reicht nicht.** Sie steht über dem Paket, und wer
`brain case <id>` in einer Wolkensitzung ruft, hätte den Quelldiff im Kontext,
bevor er die Marke gelesen hat. Für einen zurückgehaltenen Fall — ob Feld,
Manifest oder unbestimmter Modus ihn schließt — druckt der Befehl deshalb
weder `package.md` noch `proposal.md` noch die abgelöste Fassung; er nennt nur
ihren Ort. Der Vorschlag steht mit unter der Schranke, weil die
Evidenzbindung wortgleiche Zitate aus den Paketsegmenten verlangt und er den
Quelldiff damit bauartbedingt mitführt. Die ausdrückliche Flagge
`brain case --package <id>` hebt die Schranke auf: Ein Mensch am Terminal
tippt sie, ein Ablauf tut es nicht versehentlich — und tut er es doch, ist es
eine bewusste Handlung, die der Prüfskill verbietet. Sie hebt nur den
Dateiblock auf; Halt-Zeile und Zusatzzeile bleiben stehen.

**Der Vorschlag geht durch dieselbe Evidenzbindung wie jeder andere.** Ein
lokal erzeugter Vorschlag bekommt keinen Rabatt; er ist genau deshalb
brauchbar, weil derselbe Prüfer ihn abweist, wenn er das Zitat erfindet.

## 4. Konfiguration und Opt-out

**Vorgabe bei frischer Installation: `enabled = false`.** Die Abhängigkeit ist
beim Paketieren optional (§11).

Drei Ebenen — global, je Bereich, je Rolle. **Aus schlägt An, und nur in
dieser Richtung:** Ein `enabled = false` auf einer gröberen Ebene gilt für
alles darunter und lässt sich feiner **nicht** zurücknehmen; ein `true` gilt
nur, solange keine gröbere Ebene abgeschaltet hat. Wer global abschaltet,
schaltet damit jeden Bereich und jede Rolle ab — die feinere Ebene kann
einschränken, nie erweitern.

```toml
# <zustand>/config.toml  (neu; das Register liegt daneben)
[model]
enabled     = false
endpoint    = "http://127.0.0.1:11434"
name        = "hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL"
temperature = 0.0
roles       = { propose = true, place = true, describe = true }
```

**Die Temperatur ist eine Vorgabe, kein Zwang** — anders als `think: false`,
das im Aufruf steht und nicht verhandelbar ist. Gemessen über die drei Stufen
zusammen (je 360 Vorschläge, Lauf 8 mit `vorschlag-v4`): bei 0,0 **360 von
360**, bei 0,8 **359 von 360** — unter der ersten Prompt-Fassung waren es 352
(Läufe 4 und 6). Das ist eine kleine Stichprobe: 359/360 heißt 98,44 bis
99,95 % (Wilson), 352/360 heißt 95,68 bis 98,87 %. Der belastbare Wert steht
in §2 — über 1 000 Läufe je Stufe fällt die empfohlene Stufe bei 0,8 auf
988/1 000.

Wer die Temperatur hochsetzt, verliert Treffer, aber nichts Grundsätzliches
— die Evidenzbindung fängt jeden Ausrutscher ab. Deshalb
steht sie hier und nicht in §2. Die Quote in §9 gilt für den Vorgabewert.

```toml
# .ultra-brain/config.toml eines Bereichs
[model]
enabled = false        # schaltet das Modell für diesen Bereich ab
```

**Der Schalter ist ein Codegatter vor dem Aufruf, keine Bitte im Prompt**
(§11). Wer `enabled = false` setzt, bei dem wird die HTTP-Verbindung gar nicht
erst geöffnet.

## 5. Die drei Rollen im Einzelnen

| Rolle | Ausgabeform erzwungen durch | Rückfall |
|---|---|---|
| `propose` | das Vorschlagsformat aus Scheibe 5 §6.3, geprüft von `check_evidence` | manueller Fall |
| `place` | JSON-Schema am Endpunkt (`format`), Scope **muss** aus dem Register stammen | Datei bleibt im Eingang |
| `describe` | ein Satz, Länge begrenzt, Sprache geprüft, als YAML zurücklesbar | **keine Beschreibung** — der Katalog zeigt den Titel allein (§3) |

**Ausgabeform wird erzwungen, nicht erhofft** (§11): `place` läuft über die
Schema-Zwangsform des Endpunkts — im Spike 5/5 gültiges JSON mit gültigem
Scope über alle Modelle. Ein Scope, den das Register nicht kennt, wird
verworfen, auch wenn er wohlgeformt ist.

**Je Umsetzung eine eigene Prompt-Fassung, getrennt versioniert** (§11). Die
Prompts liegen als Dateien neben dem Code, nicht als Zeichenketten darin, und
tragen eine Versionsnummer, die im Fall protokolliert wird — sonst ist eine
Änderung der Trefferquote später nicht zuzuordnen. **Ausgangsfassung für `propose`
ist `vorschlag-v4`**, gemessen unter `bench/scheibe-6-ollama/prompts/`; im
Betrieb zieht sie neben den Code.

**Und die Versionierung ist kein Formalismus, sondern die Lehre aus vier
gemessenen Fassungen** (Protokoll §13): Zwei von ihnen haben eine Stufe
verschlechtert, eine davon um 57 Punkte — und **beide sahen beim Lesen wie
Verbesserungen aus**. Der Grund war jedes Mal derselbe: In einem Prompt
konkurrieren Anweisungen um Aufmerksamkeit, und ein Zusatz kann eine
bestehende Regel aushebeln, ohne sie zu erwähnen. `v2` erklärte das Kopieren
der Diff-Spalte so ausführlich, dass die Überschriftenform weiter oben
verdrängt wurde; die Zitate wurden richtig, aber kein Anspruch mehr erkannt.
**Eine Prompt-Änderung ohne Lauf über alle Stufen und alle Pakete ist eine
Wette.**

**`describe` prüft fünf Regeln, nicht vier.** Vier davon sind am Messwerk
geeicht: genau ein Satz, höchstens 22 Wörter, deutsch, keine zerhackten
Wörter. Die fünfte kommt nicht vom Messwerk, sondern vom
Ziel, in das der Satz zieht: **Der Satz muss als YAML zurücklesbar sein.** Ein
Satz mit einem Doppelpunkt bricht die Zuordnung auf, ein Doppelkreuz macht den
Rest zum Kommentar, eckige Klammern am Satzanfang lesen sich als Folge — und
ein Satz, der den Dateikopf zerbricht, macht die Datei für jeden Leser
unbrauchbar, nicht nur für den Katalog. Geprüft wird am Ziel selbst statt an
einer Liste verbotener Zeichen: Eine Liste vergisst den vierten Fall.

Die Regel wirkt ausschließlich verschärfend — sie verwirft, sie nimmt nichts
an, was die vier anderen verworfen hätten. Sie ist deshalb **keine
Prompt-Änderung und damit keine Wette** im Sinne des Absatzes darüber; die
Prompt-Fassung `beschreibung-v1` bleibt unangetastet.

**Die gemessene Quote gilt für vier Regeln, die fünfte ist ungemessen.** Was
Protokoll §8 ausweist — 450/450 bei `temperature: 0`; bei 0,8 E2B 30/30, E4B
26/30, 12B 30/30 —, ist an vier Regeln gemessen. Dazu kommen drei
Einschränkungen, die eine spätere Nachmessung mitnehmen muss:

- Gemessen wurde an **1 800 Zeichen einer Wikiseite**. Im Betrieb steht dort
  der Rumpf eines Transkripts oder eines PDF — anderer Satzbau, andere
  Eigennamen, andere Zeichen.
- **Vier der sieben Fehlschläge lagen bei 23 Wörtern** (Protokoll §8), also
  direkt an der Grenze, die `_DESCRIBE_MAX_WORDS` mit 22 zieht. Die Quote
  hängt an dieser einen Zahl stärker als an allem anderen.
- Wie oft die fünfte Regel im Betrieb zuschlägt, ist **nicht gemessen**. Sie
  kann die Quote nur senken.

**Sprachprüfung an Funktionswörtern, nicht an Umlauten, und die Liste führt
jedes Wort in beiden Schreibweisen.** Der Spike hat sich daran eine Messung
verdorben („Der Bau ist in acht Scheiben zerlegt." hat keinen Umlaut), der
Neubau des Messwerks sich eine zweite — spiegelverkehrt, weil die Liste `fuer`
kannte, aber nicht `für`. Dieselbe Regel, zweimal falsch: Ein Prüfer, der eine
von zwei zulässigen Schreibweisen nicht kennt, verwirft richtiges Deutsch.

**Zerhackte Wörter erkennt man daran, dass der Bindestrich ein Wort
zerschneidet** — schiebt man die Teile zusammen und es entsteht ein geläufiges
Wort (`Pro-jekt` → `Projekt`), ist es zerhackt. Das ist das Merkmal, an dem
`glm-4.7-flash` scheiterte, während es formal fehlerfrei aussah. Der Weg über
die Worthäufigkeit der Teile allein taugt **nicht**: Deutsche Komposita sind
von Natur aus selten (`Teilsystem`, `Suchkette`) und sehen dabei aus wie
Bruchstücke.

## 6. Die kritische Regel

**Das lokale Modell abzuschalten darf niemals dazu führen, dass Inhalte
stattdessen in die Cloud gehen** (§11). Der Datenschutzmodus steht über der
Modellverfügbarkeit. Als Tabelle, damit kein Fall unbedacht bleibt:

| Modus | Modell an | Modell aus oder nicht erreichbar |
|---|---|---|
| `local_only` | lokaler Vorschlag | **manueller Fall** — nie Cloud |
| `manual_cloud` | lokaler Vorschlag (spart die Bestätigung) | Paket anzeigen, Bestätigung abwarten |
| `automatic_cloud` | lokaler Vorschlag | regelkonformes Paket automatisch |

Die Zeile, auf die es ankommt, ist die erste rechts: Ausfall des Modells
erzeugt **Arbeit für den Menschen**, keinen ausgehenden Verkehr.

**Scheibe 6 liefert von dieser Tabelle nur die Zeile `local_only`.** Der
Vorschlaggeber wird ausschließlich für geschlossene Bereiche gebaut
(`reconcile._proposers` filtert darauf); `manual_cloud` und `automatic_cloud`
bekommen in dieser Scheibe **keinen** lokalen Vorschlag, sondern verhalten
sich wie vor ihr. Die beiden Zellen „lokaler Vorschlag" in der mittleren und
unteren Zeile sind damit Absicht der Zielgestalt, nicht Soll dieser Scheibe —
sie sparen eine Bestätigung beziehungsweise einen Aufruf und lösen kein
Datenschutzproblem. Eine spätere Scheibe holt sie nach; das Fertig-Kriterium
in §9 verlangt sie ausdrücklich nicht.

## 7. Datenschutznachweis

Zwei Nachweise, beide Teil des Fertig-Kriteriums:

1. **Im Code:** Der Klient lehnt jede Adresse ab, die nicht auf `127.0.0.1`
   oder `localhost` zeigt — geprüft gegen einen untergeschobenen Endpunkt, der
   auf eine fremde Adresse zeigt. Ein Fehlschlag ist ein Fehler, kein
   stillschweigender Rückfall.
2. **Am echten Lauf:** Ein Vorschlag für einen `local_only`-Bereich wird unter
   Mitschnitt erzeugt (`pktmon` bzw. `netsh trace`); im Mitschnitt steht kein
   Paket an eine Adresse außerhalb der Schleife. Gegenprobe mit
   abgeschaltetem Modell: ebenfalls keines, und ein manueller Fall entsteht.

   > **Abgenommen am 2026-09-11 ohne Paketmitschnitt.** `pktmon` und
   > `netsh trace` brauchen Adminrechte, und der Nutzer hat entschieden, keine
   > Shell dafür zu starten. An ihre Stelle traten zwei Belege ohne
   > Adminrechte: ein Audit-Hook im Lauf-Prozess, der jedes `connect`,
   > `sendto`, `sendmsg` und jede Namensauflösung lückenlos mitschreibt, und
   > eine Abtastung der Verbindungstabelle für Ollama und den Lauf-Prozess im
   > Abstand von rund 30 ms, deren Filter eine Positivkontrolle bestanden hat.
   > **Die Grenzen:** Für den Lauf-Prozess ist der Hook dichter als ein
   > Mitschnitt, auch für UDP. Für den Ollama-Server bleibt zweierlei
   > unbemerkt: eine Verbindung, die kürzer lebt als ein Abtastschritt, und
   > UDP über einen Socket auf `0.0.0.0` oder `[::]`, weil `verbindungen.ps1`
   > eine Wildcard-Bindung als Schleife wertet. Ein Mitschnitt nach dem Wortlaut
   > oben bleibt nachholbar, sobald eine Admin-Shell verfügbar ist. Befehle
   > und Ergebnisse stehen in
   > [bench/scheibe-6-ollama/abnahme.md](../../../bench/scheibe-6-ollama/abnahme.md).

## 8. Was dieser Entwurf nicht entscheidet

- **In welcher Sprache der Vorschlaggeber entsteht.** Er hängt an
  `maintenance/`, das die Migration in ihrer **Migrationsscheibe 9** nach Go
  holt — die Nummer gehört der Migrationsreihe, nicht der Scheibenreihe
  dieses Entwurfs; beide Reihen zählen ab eins und kollidieren in mehreren
  Nummern. Auslöser wie bei Scheibe 7 §4: die **Messscheibe 2** der Migration.

  **Stand 2026-09-07:** Die Messscheibe 2 ist gebaut und auf master. Der
  Auslöser ist damit gezogen, die Frage aber nicht beantwortet — vor „Go oder
  Python“ steht die Frage, **ob der Daemon bleibt**; die Grenze zwischen
  beiden Sprachen hängt daran und ist nicht entschieden. Bis sie es ist, gilt
  die Schnittstelle aus §3 sprachneutral, und dieser Entwurf schreibt keine
  Sprache vor.
- **Wann die beiden Cloud-Modi den lokalen Vorschlag bekommen.** Scheibe 6
  baut nur den `local_only`-Zweig (§6). Für `manual_cloud` und
  `automatic_cloud` ist der lokale Vorschlag eine Ersparnis und keine
  Schranke, und was er dort kostet — ein Modellaufruf je Fall in jedem
  offenen Bereich — ist nicht gemessen. Die Scheibe, die das nachholt, steht
  noch nicht fest.
- **Ob eine eigene Prompt-Fassung je Stufe sich lohnt.** Gemessen wurde eine
  Fassung für alle drei — und die Läufe 7 und 8 zeigen, dass eine Änderung
  Stufen ungleich trifft: `v2` hob die kleinste und ließ die mittlere um 57
  Punkte fallen. Ob getrennte Fassungen mehr holen als die eine gemeinsame,
  ist offen; §5 hält die Möglichkeit ohnehin offen.

**Erledigt seit der Fassung vom 2026-09-05** (Protokoll §7–§13):

- ~~Wie viele Läufe die Abnahme verlangt.~~ 30 je Stufe, Paket und Temperatur;
  120 je Stufe und Temperatur insgesamt. Zehn waren zu wenig, weil
  `llama3.1:8b` zwischen 5/10 und 7/10 schwankte.
- ~~Mehr als ein Paket.~~ Vier, von 682 bis 9 281 Zeichen — und die Länge war
  es, die die Leiter überhaupt erst geordnet hat (§2).
- ~~VRAM neben gehaltenem qmd.~~ qmd hält warm 3,0 GB; alle drei Stufen laufen
  daneben mit `100% GPU` und unveränderten Zeiten, in beide Richtungen ohne
  Verdrängung.
- ~~Ob ein getunter Prompt die 4-GB-Stufe hebt.~~ Ja: `vorschlag-v4` hebt
  jede Stufe, die kleinste am deutlichsten, und am langen Paket von 25/30 auf
  30/30. Vier Fassungen, 1 440 Läufe (Protokoll §13), gegengeprüft an einem
  Paket, das beim Feinschliff nie aufgeschlagen war (§14) — dort ist der
  Gewinn größer, nicht kleiner.
- ~~Ob die 8-GB-Zeile nach dem Feinschliff noch trägt.~~ Ja, mit je 1 000
  Läufen entschieden: 988/1 000 gegen 971/1 000, Fisher p = 0,0107
  (Protokoll §16).

## 9. Fertig-Kriterium

Aus §17 der Architektur, ergänzt um das, was der Spike messbar gemacht hat:

> `local_only` erzeugt einen Vorschlag **ohne ausgehenden Verkehr**;
> Gegenprobe: abgeschaltet → manueller Fall, ebenfalls kein Verkehr.

Dazu, jede Zahl mit ihrer Bedingung, weil eine Quote ohne Läufe, Pakete und
Temperatur nicht lesbar ist:

| Kriterium | Maß |
|---|---|
| Evidenzbindung auf der empfohlenen Stufe | **120/120** über vier Pakete, je 30 Läufe, bei `temperature: 0` und Prompt `vorschlag-v4`. Bei 0,8 und 1 000 Läufen: 98,8 % (E4B) |
| erfundenes Zitat | fällt durch — der Prüfer vergleicht jedes Zitat Zeichen für Zeichen gegen die Quelle. Die 4 850 Aufrufe des Spikes belegen, dass er auf jedem Weg lief; dass er dicht ist, folgt aus dem Vergleich, nicht aus der Zahl |
| Ausgabeform `place` | ausschließlich Scopes aus dem Register |
| Zeit je Vorschlag | unter **zwei Sekunden**; gemessen 563 bis 1 314 ms über alle Stufen und Pakete |

Die 4-GB-Stufe erreicht die Quote bei `temperature: 0` ebenfalls, bei 0,8
97,1 Prozent (§2). Das Kriterium gilt für den Vorgabewert aus §4.

**Eine Quote ohne ihr Intervall ist keine Abnahme.** „150 von 150" heißt
nicht „fehlerfrei", sondern „mindestens 97,5 Prozent"; dieselbe Stufe kam
über 1 000 Läufe auf 98,8. Kleine Stichproben fielen hier durchweg zu gut
aus (Protokoll §16).

<!-- version: vorschlag-v4
Aenderung gegenueber v3: Regel 2 sagt jetzt, woher der Text zu nehmen ist,
statt nur zu verlangen, dass die Nummer stimmt.

v3 hat die 8-GB- und die 16-GB-Stufe auf 120/120 gebracht und die 4-GB-Stufe
am langen Paket von 25/30 auf 30/30. Ihre restlichen sieben Fehlschlaege sind
aber eine andere Klasse: Sie zitiert Text aus `W1` unter `evidence: D1`, oder
laesst den Codeblock ganz weg. Kein Kopierfehler mehr, sondern ein
Zuordnungsfehler.

Aenderung gegenueber v2: die Ueberschriftenform wird nach dem Beispiel noch
einmal genannt.

v2 hat gemessen, was sie sollte -- das Kopieren der ersten Spalte -- und
dabei etwas anderes zerstoert: Die 8-GB-Stufe fiel von 118/120 auf 61/120,
und 58 der 59 Fehlschlaege waren `no segment cited`, weil das Modell
`## Widerspruch zwischen D1 und W1` schrieb statt `## B1 — ...`. Die Zitate
darin waren richtig kopiert. Der lange Beispielblock hat die Formvorgabe
verdraengt, die weit oben steht.

Aenderung gegenueber v1: ein Abschnitt ueber die erste Spalte einer
Diff-Zeile, plus ein Beispielpaar richtig/falsch.

Der Grund steht in Protokoll Paragraf 12. Alle fuenf Fehlschlaege der
4-GB-Stufe am langen Paket waren `quote not found`, und alle fuenf sassen im
ersten Zeichen der Zeile: dreimal ein eingefuegtes Leerzeichen nach dem
Minus, zweimal zusaetzlich ein umgedrehtes Vorzeichen. Das Modell hat die
Zeile gelesen, wie ein Mensch sie liest -- als Aufzaehlung mit einem
Gedankenstrich davor -- statt sie als Text zu nehmen.

Der Rest des Prompts ist woertlich v1, damit ein Unterschied in der Quote
diesem einen Abschnitt zuzuordnen ist.
-->
Du bekommst ein Pruefpaket. Es besteht aus nummerierten Abschnitten: `D` sind
Aenderungen an einer Quelle, `W` sind Absaetze der Wiki-Seite, `Q` sind
Quellenangaben.

Deine Aufgabe: Finde **einen** Widerspruch zwischen einer Aenderung (`D`) und
einem Wiki-Absatz (`W`) und schreibe genau einen Anspruch.

Antworte in genau dieser Form, ohne Vorrede und ohne Nachwort:

## B1 — <ein Satz, was der Widerspruch ist>

evidence: <die Nummer des Abschnitts, z.B. D1>

```
<hier den Text des genannten Abschnitts hineinkopieren>
```

Die drei Regeln, an denen der Vorschlag geprueft wird:

1. Der Text im Codeblock muss **Zeichen fuer Zeichen** aus dem genannten
   Abschnitt stammen. Kopiere ihn, formuliere ihn nicht um. Kuerzen ist
   erlaubt, aendern nicht.
2. Geh so vor: **Waehle zuerst den Abschnitt**, schreib seine Nummer hinter
   `evidence:`, und kopiere dann aus **genau diesem** Abschnitt. Nicht aus
   einem anderen, und nicht aus dem Gedaechtnis. Der Codeblock darf nie leer
   bleiben.
3. Schreibe die Ueberschrift auf Deutsch.

**Achtung beim ersten Zeichen einer Zeile.** In einem `D`-Abschnitt beginnt
jede Zeile mit `-`, `+` oder einem Leerzeichen. Dieses Zeichen ist **Teil des
Textes**, nicht Formatierung. Setze kein Leerzeichen dahinter, das nicht da
steht, und tausche `-` nicht gegen `+` oder umgekehrt.

Steht im Abschnitt:

    -Die Kette nutzt zwei Rueckgrate.
    +Die Kette nutzt drei Rueckgrate.

dann ist **richtig**:

    -Die Kette nutzt zwei Rueckgrate.

und **falsch** sind:

    - Die Kette nutzt zwei Rueckgrate.
    +Die Kette nutzt zwei Rueckgrate.

**Zur Erinnerung, weil das Beispiel lang war:** Deine Ueberschrift beginnt
mit `## B1 — ` und dann dem Satz. Nicht `## Widerspruch ...`, nicht `## D1
gegen W1`. Genau `## B1 — `.

Das Pruefpaket:

{paket}

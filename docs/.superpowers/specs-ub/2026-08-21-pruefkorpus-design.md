# Prüfkorpus — ein versionierter Testbestand

**Stand:** 2026-08-21
**Bezug:** Architektur-Design §13 (Messprogramm), §16.3 (Mehrsprachigkeit)
**Einordnung:** eigene Scheibe **nach 2b**. 2b baut das Messwerk und misst den
echten Bestand; erst danach entsteht das Prüfstück, das dieses Werkzeug
wiederholbar bedienen kann.

---

## 1. Zweck, und die Grenze davon

Der Prüfkorpus ist ein **Regressionsnetz**, kein Messinstrument.

| | echter Fragensatz (Vault) | Prüfkorpus (Repo) |
|---|---|---|
| Beantwortet | *Findet das System **meinen** Bestand?* | *Ist die Kette schlechter geworden als beim letzten Stand?* |
| Wiederholbar | nein — der Bestand wächst, Belege verschwinden | ja, byteweise, auf jedem Rechner |
| Versionierbar | nein, er verrät den Bestand | ja, gehört ins Repo |
| Trägt Entscheidungen | ja — Profilwahl, Modellwahl, Budgets | **nein** |

**Die letzte Zeile ist tragend.** Ein zusammengestellter Korpus hat
zusammengestellte Eigenschaften: Notizlängen, Sprachmischung, Dubletten,
Chunk-Grenzen. Liegt `fast` dort gleichauf mit `full`, sagt das über den echten
Vault nichts. Keine Architekturentscheidung wird je auf Korpuszahlen gestützt;
sie beantworten ausschließlich die Frage nach der Verschlechterung.

Der Nutzen wächst ab 2c, wenn sich die Kette laufend ändert — Daemon,
gehaltener Unterprozess, später ein anderer Reranker. Ohne wiederholbaren
Korpus fällt eine Verschlechterung erst im Alltag auf, und dann ist nicht mehr
zu sagen, welche Änderung sie verursacht hat.

**Zweiter Nutzen:** Heute läuft jeder Test gegen `FakeSearchPort`, der
zurückgibt, was der Test vorgibt. Der Korpus erlaubt einen Integrationstest
gegen echtes qmd — langsam, deshalb nicht in jedem Lauf, aber vorhanden.

## 2. Der Satz — was ein Stand ist

Ein Stand ist immer **genau derselbe Satz**:

- **100 Notizen**
- **50 Fragen**
- **10 Themen**

Keine Ausnahme. Die feste Zahl macht zwei Stände vergleichbar und macht
sichtbar, wenn einer unvollständig ist.

### 2.1 Verteilung der 50 Fragen

| Sorte | Anzahl |
|---|---|
| exakt | 13 |
| Umschreibung | 13 |
| gemischt | 10 |
| **sprachübergreifend** | **14** |

Das hält das Verhältnis 8 / 8 / 6 / 8 aus §13 und legt den Rest auf die
schärfste Sorte, statt ihn gleichmäßig zu verteilen. Von den 14
sprachübergreifenden laufen **mindestens 5 in der Gegenrichtung** — englische
Frage, deutsche Quelle.

### 2.2 Zehn Themen, die sich überschneiden

Zehn Themen zu je zehn Notizen. Die Überschneidung ist der eigentliche Zweck:
Zu jeder Frage gibt es Notizen, die zum selben Thema gehören und die Antwort
trotzdem nicht enthalten. Ohne solche **Störer** prüft der Korpus nur, ob die
Suche das einzige passende Dokument findet — das kann grep auch.

Jedes Thema bekommt deshalb ein **Nachbarthema**, mit dem es Vokabular teilt
(etwa Netzwerktechnik und Netzwerksicherheit). Die Störer liegen dann nicht
bloß daneben, sondern glaubhaft daneben.

## 3. Herkunft der Notizen

Öffentlich lizenzierter Text, **nach Thema ausgewählt**, nicht zufällig
zusammengetragen. Jede Notiz dient einer geprüften Eigenschaft:

- **Sprachbrücke:** dasselbe Thema existiert als deutscher und als englischer
  Artikel; in den Korpus kommt nur **eine** Sprache, die Frage stellt die
  andere. Beide Richtungen.
- **Exakte Begriffe:** Fachthemen mit eindeutigen Bezeichnern, Normnummern,
  Fachwörtern.
- **Umschreibung:** Alltagsthemen, bei denen Frage und Text kein Stichwort
  teilen.
- **Gemischt:** deutsche Prosa mit englischen Fachbegriffen — technische
  Themen liefern das von selbst.

**Lizenz wird ernst genommen.** Zu jeder Notiz stehen Quelle, Abrufdatum und
Lizenz fest. Ein Repository mit fremdem Text ohne Herkunftsangabe ist ein
Problem, das man erst bemerkt, wenn es eines ist.

## 4. Ablage und Versionierung

```
bench/corpus/v1/
  notes/           die 100 Notizdateien, .md
  questions.yaml   dieselbe Form wie der echte Satz
  themes.yaml      die 10 Themen, je mit Nachbarthema und zugehörigen Notizen
  manifest.json    Prüfsumme je Datei, Quelle, Lizenz
  HERKUNFT.md      Quelle und Lizenz je Notiz, für Menschen lesbar
```

Der Korpus liegt im **Projekt-Repo** und ist eingecheckt — das erste
Messartefakt, das öffentlich sein darf. Der echte Fragensatz bleibt im Vault.

**Nummerierte Stände, additiv.** „Erweitern" heißt immer ein neuer Stand mit
demselben Satz: `v2` bringt 100 weitere Notizen, 50 weitere Fragen, 10 weitere
Themen. **Aus `v1` wird nichts geändert und nichts entfernt.** Nach `v2` liegen
200 Notizen und 100 Fragen im Repo; gemessen werden kann wahlweise gegen `v1`,
gegen `v2` oder gegen alle Stände, und die alte Zahl bleibt nachrechenbar.

**Ein neuer Stand bringt neue Themen.** Wiederholte `v2` die Themen aus `v1`,
entstünden Notizen, die einander über Standgrenzen hinweg stören — ein Lauf
über beide Stände ergäbe dann etwas anderes als die Summe der Einzelläufe, und
genau die Vergleichbarkeit, für die die Nummerierung da ist, wäre dahin. Die
Überschneidung bleibt **innerhalb** eines Standes.

Jede Messung nennt den Stand im Protokollkopf. Eine Zahl ohne Standangabe ist
keine Zahl.

## 5. Das Prüfskript

Die Regeln aus §2 bis §4 gelten durch ein Skript, nicht durch Sorgfalt beim
Anlegen. Es prüft je Stand:

- genau 100 Notizen, 50 Fragen, 10 Themen
- die Verteilung 13 / 13 / 10 / 14, davon mindestens 5 in der Gegenrichtung
- jedes Thema hat zehn Notizen und ein benanntes Nachbarthema
- jede `expect`-Datei existiert, jeder `beleg` steht wörtlich darin
- die Prüfsummen im `manifest.json` stimmen — eine unbemerkte Änderung an
  einem alten Stand fällt damit auf, statt still alte Zahlen zu entwerten
- zu jeder Notiz stehen Quelle und Lizenz im `manifest.json`
- kein Thema kehrt aus einem früheren Stand wieder

**Ein Stand, der das nicht besteht, existiert nicht** — das Skript läuft in der
Prüfstrecke, nicht auf Zuruf.

## 6. Anschluss an das Messwerk

Der Korpus braucht kein eigenes Werkzeug. `brain bench` aus Scheibe 2b liest
einen Fragensatz und misst gegen einen `SearchPort`; für den Korpus zeigen
beide woandershin:

```
brain bench --corpus bench/corpus/v1 [--profile ...]
```

Das indexiert `notes/` in eine eigene qmd-Sammlung, fährt `questions.yaml` und
schreibt das Protokoll — mit dem Stand im Kopf und dem ausdrücklichen Vermerk,
dass diese Zahlen Regression messen und keine Entscheidung tragen.

Ergänzung an 2b: `bench` muss den Fragensatzpfad und die Sammlung von außen
annehmen können. Das ist eine kleine Erweiterung, keine zweite Kette.

## 7. Fertig-Kriterium für `v1` — erfüllt

- [x] 100 Notizen, 10 Themen mit Nachbarthemen, Quellen und Lizenzen vollständig
- [x] 50 Fragen in der Verteilung 13 / 13 / 10 / 14, 6 in der Gegenrichtung
- [x] Prüfskript läuft grün und hängt in der Prüfstrecke (`tests/test_corpus_v1.py`)
- [x] `brain bench --corpus bench/corpus/v1` gelaufen, Protokoll abgelegt
- [x] die erste Zahl ist als **Ausgangswert** festgehalten, nicht als Bewertung

## 8. Der Ausgangswert

**Stand `v1`, gemessen am 21. August 2026** unter qmd 2.8.3 (facd35e),
Einbettungsmodell `embeddinggemma-300M-Q8_0`, Frageerweiterung
`qmd-query-expansion-1.7B-q4_k_m`, Reranker `Qwen3-Reranker-0.6B-Q8_0`,
Windows 11:

| Sorte | Profil `fast` |
|---|---|
| exakt | 13/13 |
| Umschreibung | 11/13 |
| gemischt | 8/10 |
| sprachübergreifend | 11/14 |
| **gesamt** | **43/50** |

**Das ist ein Ausgangswert, keine Note.** Ob 43 von 50 gut sind, sagt diese
Zahl nicht und soll sie nicht sagen — sie ist der Wert, gegen den jede spätere
Messung gehalten wird. Fällt eine spätere Messung unter 43, hat sich etwas
verschlechtert, und der Vergleich der Fehllisten sagt, was.

**Das Protokoll dieses Laufs liegt im Repo**, unter
`bench/corpus/v1/baseline/` (`bench-2026-08-21-1437-fast.md` und `.json`).
Dort steht die **Fehlliste** — welche sieben Fragen fehlschlugen und wie —,
und die ist der eigentliche Inhalt des Ausgangswerts: Ohne sie ließe sich beim
nächsten Lauf sagen „schlechter“, aber nicht „diese sieben waren es damals,
diese neun sind es jetzt“.

Zwei Beobachtungen zum ersten Lauf, die für den Bau des Standes sprechen:

**Exakt 13/13.** Jeder Bezeichner wurde vor der Aufnahme gegen alle hundert
Notizen auf Eindeutigkeit geprüft. Dass die Stichwortsuche sie danach
vollständig findet, ist die Gegenprobe auf diese Prüfung — nicht die Leistung
der Suchmaschine.

**Sprachübergreifend 11/14 gegen 13/14 am echten Bestand.** Der Korpus ist in
dieser Sorte schwerer, und das ist Absicht: Seine Fragen wurden ausdrücklich so
geschrieben, dass sie mit ihrer Zielnotiz kein Inhaltswort teilen, während der
echte Satz gewachsene Fragen enthält. Ein künstlicher Bestand, der leichter
wäre als der echte, würde eine Verschlechterung erst bemerken, wenn sie im
Alltag längst weh tut.

**`full` ist auf diesem Rechner nicht messbar.** Drei Versuche, den Stand mit
dem Hybridprofil zu messen, sind alle drei mit einem CUDA-Fehler im Reranking
abgestürzt (`ggml-cuda.cu:106`). Kein einziger rein vektorieller Lauf ist
abgestürzt. Damit bestätigt der Korpus, was schon die Messung am echten
Bestand zeigte (Architektur-Spec §7.3) — an einem völlig anderen Bestand, was
die Beobachtung von der Eigenart einer bestimmten Notizsammlung löst.

# Sprachprüfung für Commit-Nachrichten — Entwurf

Stand: 2026-08-25. Status: zur Durchsicht.

Nachfolger: `2026-08-27-sprachen-erweitern-design.md`. Zur Sprachabdeckung ist
dieses Papier überholt — es kennt nur Deutsch gegen Englisch und umgekehrt,
während der Nachfolger die verschmolzene romanische Gruppe und den Test auf
nichtlateinische Schriften ergänzt. Alles Übrige hier gilt unverändert.

## Warum

Eine Sprachregel, die nur in Dateien steht, die niemand automatisch lädt, hält
nicht. space hat das gemessen: Die Regel „Commits sind englisch" trug rund
hundert Commits und brach dann sechzehn Mal hintereinander — weil die
konkurrierende Regel in einer Datei stand, die bei jedem Zug geladen wird, und
die eigene nicht.

Am 2026-08-25 ist derselbe Fall noch einmal eingetreten, diesmal von außen: Ein
Agent mit einer eigenen Regel („Prosa deutsch") schrieb eine deutsche
Commit-Nachricht in ein Projekt, dessen Regel er nicht geladen hatte. space'
Hook hat sie abgewiesen. Genau dafür ist er da — und genau deshalb gehört das
Werkzeug in ultraloom, statt in jedem Projekt neu zu entstehen.

## Was gebaut wird

Ein Unterkommando:

    ultraloom commit-msg <datei>

Es liest die Nachrichtendatei, die git ihm gibt, und beendet mit einem
Exit-Code. Ein `.githooks/commit-msg` ruft es auf; git ruft den Hook.

### Warum ein eigenes Kommando und keine Policy-Regelart

Die Policy prüft **Werkzeugaufrufe, bevor sie geschehen**, und liest dafür
Claude Codes Payload. Eine Commit-Nachricht kommt aus einem git-Hook, lange
nachdem der Agent fertig ist — und oft aus einer Datei, weil mehrzeilige
Nachrichten über `git commit -F` geschrieben werden. Eine Policy-Regel über
Kommandos sähe genau die Nachrichten nicht, um die es geht.

Übernommen wird die **Machart** der Policy, nicht ihre Struktur: eingebaute
Vorgaben plus projektspezifische Ergänzung, jede Ablehnung mit Begründung,
Exit 1 für Innenfehler und Exit 2 für den Befund.

## Was es ausdrücklich nicht tut

**Es entscheidet nicht, ob ein Text die richtige Sprache hat.** Das kann kein
Werkzeug zuverlässig. Es entscheidet, ob ein Text **offensichtlich die falsche**
Sprache hat — eine viel kleinere Frage, und die einzige, die der beobachtete
Fehler stellt.

Der Unterschied ist nicht akademisch. Ein Gate mit Falschmeldungen wird mit
`--no-verify` umgangen und schützt danach nichts mehr.

## Die Heuristik

**Stoppwörter ohne Homograph in der Zielsprache.** Für `language = "en"` sind
das deutsche Funktionswörter, die im Englischen nichts bedeuten: `der`, `das`,
`dem`, `des`, `und`, `oder`, `nicht`, `ein`, `eine`, `einen`, `einem`, `eines`
und weitere.

Ausdrücklich **nicht** in der Liste, obwohl deutsch: `die`, `war`, `man`,
`den`, `hat`, `in`, `so`, `an`. Alle sind gewöhnliche englische Wörter. „Let
the process die in the war room" darf kein Befund sein.

**Die Schwelle zählt Treffer je Zeile, nicht je Nachricht.** Vorgabe: zwei.

Das ist gemessen, nicht gewählt. In space' Historie liegen die drei echten
Falschmeldungsformen — ein deutscher Seitentitel, ein deutsches Zitat, ein
deutscher Dateiname — bei **genau einem** Treffer je Zeile; deutsche Prosa
deutlich darüber. Eine Schwelle von eins hätte drei korrekte Commits abgelehnt,
zwei lehnt keinen ab.

Je Zeile und nicht je Nachricht aus demselben Grund: Ein Text, der zwei
deutsche Seitentitel aufzählt, sind zwei Zeilen mit je einem Treffer — nicht
eine Zeile mit zweien.

**Der Diff wird abgeschnitten.** `git commit --verbose` hängt den vollständigen
Diff unter eine Scherenmarke (`# ------------------------ >8 ---`). Er ist
unkommentiert und enthält, was immer die Änderung berührt. Ohne den Schnitt
lehnte das Gate jeden Commit ab, der deutschsprachige Dateien anfasst.

**Kommentarzeilen zählen nicht.** git schreibt seine Hinweise mit `#` in die
Datei; sie sind nicht die Nachricht.

**Zweite Messung, an dieser Historie.** `ultraloom commit-msg --calibrate 100
--language en --root .` über ultraloom selbst, am 2026-08-26: von hundert
Nachrichten lehnt die Heuristik **genau eine** ab — bei Schwelle 1 und bei
Schwelle 2 dieselbe, ab Schwelle 3 keine mehr.

Die abgelehnte ist `8e9c13d` „Make the umlaut test actually depend on folding".
Sie handelt von der Stoppwortliste und zitiert `das, und` blank in einer
Klammer, ohne Backticks. Das ist der ehrliche Grenzfall: ein englischer Commit,
der über deutsche Wörter spricht. Genau dafür gibt es Code-Spans und
`[[commit.allow]]`.

Der Befund stützt die Vorgabe nur halb. Schwelle 2 kostet hier dasselbe wie
Schwelle 1 — der Abstand, den space' Historie zeigt, taucht in dieser nicht
auf, weil ultraloom keine deutschen Seitentitel in Commits zitiert. Die Vorgabe
bleibt bei zwei, weil space' Messung sie trägt; diese hier widerspricht ihr
nicht, bestätigt sie aber auch nicht.

## Konfiguration

    [commit]
    language  = "en"   # oder "de"; ohne den Abschnitt prüft ultraloom nichts
    threshold = 2      # Treffer je Zeile, ab denen eine Zeile als falsch gilt

    [[commit.allow]]
    regex  = "^Co-Authored-By:"
    reason = "Trailer, keine Prosa."

**Ohne `[commit]` passiert nichts.** Die Prüfung ist ein Opt-in: Ein Projekt
ohne Sprachregel bekommt keine, und ein `commit-msg`-Hook, der nichts prüft,
beendet mit 0.

**`language` wählt das Sprachpaar.** `"en"` sucht deutsche Stoppwörter, `"de"`
englische. Beide Listen liegen als Konstante im Modul — sie hängen am
Sprachpaar, nicht am Projekt, und ein Projekt, das seine eigene pflegen müsste,
bekäme eine Prüfung ohne Kalibrierdaten.

**Die beiden Richtungen sind nicht gleich gut belegt.** Die deutsche Liste ist
an space' Historie kalibriert — hundert englische Commits gegen sechzehn
deutsche, und die drei Falschmeldungsformen sind namentlich bekannt. Für
`language = "de"` gibt es diese Daten nicht: Englische Funktionswörter ohne
deutschen Homographen (`the`, `and`, `with`, `this`, `that`, `from`, `which`)
sind leicht zu benennen, aber ihre Schwelle ist geraten, bis jemand sie an
einem deutschsprachigen Repository misst. Die README sagt das dazu, und
`--calibrate` ist der Weg, es nachzuholen. Eine geratene Schwelle als gemessene
auszugeben, wäre genau der Fehler, den dieses Werkzeug verhindern soll.

**`threshold` ist konfigurierbar**, weil die Vorgabe an *space'* Historie
kalibriert ist und ein anderes Projekt anders schreibt. Wer sie ändert, soll
das an seinen eigenen Commits messen können — dazu unten der Kalibriermodus.

**`[[commit.allow]]`** nimmt **ganze Zeilen** aus der Wertung, deren Muster
passt: Trailer, zitierte Pfade, Ausschnitte fremder Ausgaben. Anders als die
Regeln der Policy nur **`regex`**, kein `match`: Ein Glob beschreibt einen Pfad
oder Dateinamen, und auf eine Textzeile angewandt ist seine Bedeutung unklar —
soll `WIP*` die ganze Zeile treffen oder irgendwo darin vorkommen? Genau diese
Unklarheit hätte `match = "WIP*"` still als Regex gelesen und "WIP" gefolgt von
null oder mehr "P" bedeutet, ohne Fehler und ohne Hinweis. `regex` bleibt
Pflichtfeld, `reason` ebenso.

Die Zeile fällt vollständig weg, nicht nur das getroffene Wort. Das ist die
gröbere und die richtige Wahl: Wer eine Zeile ausnimmt, meint sie als Ganzes —
eine Zeile, aus der einzelne Wörter gestrichen werden, ergäbe eine Bewertung
über einen Text, den niemand geschrieben hat.

## Der Kalibriermodus

    ultraloom commit-msg --calibrate <n>

Liest die letzten `n` Commit-Nachrichten des Repositories, wendet die Heuristik
an und zeigt, welche Nachrichten bei welcher Schwelle abgelehnt würden. Wer die
Schwelle verstellt, sieht damit vorher, was das kostet.

Das ist kein Beiwerk: Die Vorgabe „zwei" ist genau so entstanden, und ohne
diesen Weg müsste jedes Projekt die Kalibrierung von Hand nachbauen — oder die
Schwelle raten.

## Exit-Protokoll

    0  in Ordnung, oder kein [commit]-Abschnitt konfiguriert
    1  interner Fehler — Datei unlesbar, Konfiguration kaputt
    2  die Nachricht ist offensichtlich in der falschen Sprache

Anders als bei der Policy ist eine kaputte Konfiguration hier **Exit 1**, nicht
Exit 2. Der Grund ist der Unterschied im Schaden: Eine Policy, die stillschweigend
durchlässt, gibt eine Datei preis; ein Sprachgate, das durchlässt, kostet eine
Commit-Nachricht in der falschen Sprache. Ein Gate, das wegen eines Tippfehlers
in der Konfiguration jeden Commit blockiert, ist an dieser Stelle das größere
Ärgernis — und der Fehler fällt beim nächsten `ultraloom check` ohnehin auf.

Bei Exit 2 nennt die Meldung **jede** beanstandete Zeile mit ihren Treffern und
sagt, wie man es umgeht (`--no-verify`) und dass das die Regel nicht aufhebt.

## Der git-Hook

ultraloom liefert das Kommando, nicht den Hook. Die README zeigt die drei
Zeilen, die ein Projekt braucht:

    #!/usr/bin/env sh
    exec ultraloom commit-msg "$1"

Dazu der Hinweis auf `core.hooksPath`, weil ein Hook unter `.git/hooks/` nicht
versioniert ist und in einem frischen Klon fehlt.

**Kein `install`-Unterkommando.** Ein Werkzeug, das ungefragt in `.git/`
schreibt oder eine git-Konfiguration verstellt, ist genau die Sorte
Nebenwirkung, die dieses Projekt bei anderen kritisiert. Drei Zeilen in der
README sind ehrlicher.

## Tests

- `tests/commit/test_language.py` — die Heuristik: eine eindeutig deutsche
  Nachricht, eine eindeutig englische, die drei Falschmeldungsformen aus space'
  Historie (Seitentitel, Zitat, Dateiname) als Gegenprobe, die Schwelle je
  Zeile statt je Nachricht, beide Sprachrichtungen.
- `tests/commit/test_config.py` — `[commit]` fehlt, `language` unbekannt,
  `threshold` keine Zahl oder null, `allow`-Regel ohne `reason`.
- `tests/commit/test_cli.py` — Datei fehlt, Datei unlesbar, Exit-Codes.
- Der Scherenschnitt und die Kommentarzeilen bekommen je einen eigenen Test:
  beide sind der Grund, warum das Gate in einem echten Repository überhaupt
  benutzbar ist.

100 % Coverage wie im übrigen Repo.

## Grenzen

- **Ein Sprachpaar je Richtung.** `de`/`en` sind gebaut; eine dritte Sprache
  braucht eine Wortliste, die jemand kalibriert hat.
- **`--no-verify` bleibt offen.** Das ist Absicht und dokumentiert: Der nächste
  Commit prüft wieder.
- **Die Heuristik kennt keinen Kontext.** Ein englischer Satz mit vier
  deutschen Zitaten in einer Zeile wird abgelehnt. Dafür gibt es
  `[[commit.allow]]`.

## Ausdrücklich nicht in diesem Vorhaben

- **Kommentare und Docstrings.** Dieselbe Regel gilt dort, aber die
  Falschmeldungsfläche ist eine andere: Fachbegriffe, Namen und zitierte
  Ausgaben stehen im Quelltext dichter als in Commit-Nachrichten. Eigenes
  Vorhaben, wenn überhaupt.
- **Eine Policy-Regelart `[policy.commit]`.** Verworfen; die Begründung steht
  oben.

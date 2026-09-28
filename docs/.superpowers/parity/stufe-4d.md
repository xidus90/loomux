# Paritätsakte Stufe 4d

**Quelle:** ultra-brain `loomux-3-source` (`3cc72d2`), wie in Stufe 3a bis 4c-1.
**Spec:** `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`,
Abschnitte „4d im Einzelnen“ und „Abweichungen beim Planen von 4d“
**Plan:** `docs/.superpowers/plans/2026-09-26-loomux-stufe-4d.md`

`loomux convert` wandelt PDFs und Transkripte aus dem Eingang eines Bereichs
in Markdown, `loomux fetch` legt die Untertitel eines Videos über `yt-dlp` in
einen Eingang. Die Referenz ist `src/brain/convert/` in ultra-brain.

## Messungen

Gemessen am 2026-09-26 unter Windows 11 Pro 10.0.26200, aus Git Bash (GNU
bash 5.3.15, x86_64-pc-cygwin): `pdftotext` und `yt-dlp` zwischen 21:28 und
21:36 UTC, `pdftotext` an den verbreiterten Test-PDFs noch einmal um 21:57
UTC.

### Die Test-PDFs

`testdata/convert/pdf/make_pdfs.py` (PEP 723, `pypdf==6.16.2`) baut die
Fixtures von `tests/convert/test_pdf.py:12-127` der Referenz nach: `text.pdf`,
`paragraphs.pdf` (zwei Kopien des Textes, getrennt durch `\n\n\n` als
PDF-Escape im Textoperator, also die Bytes `5c 6e`, nicht Zeilenumbrüche),
`blank.pdf`, `pageless.pdf`, `mixed.pdf` (drei Textseiten, sieben leere),
`allscan.pdf`, `corrupt.pdf` (nur `%PDF-1.4\n`), dazu `Bericht März.pdf`
(gleicher Inhalt wie `text.pdf`) und `encrypted.pdf` (pypdf, RC4-128,
Benutzer- und Eigentümerkennwort `geheim`). `encrypted.pdf` trägt keine
Zufallskennung: pypdf bildet die `/ID` aus einer Prüfsumme des Inhalts, und
ein neuer Lauf des Skripts erzeugt die Datei Byte für Byte gleich.

**Eine Abweichung von der Referenz: die Seite ist 5000 pt breit**
(`/MediaBox [0 0 5000 842]` statt `[0 0 595 842]`). Grund ist die erste
Messung unten: `pdftotext` schneidet Text am Seitenrand ab, pypdf nicht. Auf
der A4-Breite der Referenz gaben xpdf und Poppler von der Zeile nur 96
Zeichen aus (`…von Hand geschriebenen P`), in jedem Modus (`-layout`, ohne
Schalter, `-raw`); gegen die Scan-Schwelle 100 wäre jede Textseite ein Scan
gewesen. 5000 pt fassen die Zeilen der Referenz (390 und 783 Zeichen) für
beide Leser. Für pypdf ändert die Breite nichts.

Was die Referenz selbst aus den Dateien liest (`page.extract_text(
extraction_mode="layout")`, `len(page.strip())` je Seite, pypdf der
Referenz-venv), vor und nach der Verbreiterung gleich: `text.pdf` [390],
`paragraphs.pdf` [783], `mixed.pdf` [390, 390, 390, 0, 0, 0, 0, 0, 0, 0],
`Bericht März.pdf` [390], `blank.pdf` [0], `allscan.pdf` [0, 0],
`pageless.pdf` []. In `paragraphs.pdf` macht pypdf aus den drei Escapes drei
Zeilenumbrüche (`kann.\n\n\nHallo`), also zwei Absätze.

### `pdftotext`: xpdf 4.06 und Poppler 25.07.0

Aufruf wie E6, je Datei im Verzeichnis `testdata/convert/pdf` mit dem Namen
relativ: `pdftotext -layout -enc UTF-8 -eol unix <name> -`. xpdf ist
`/mingw64/bin/pdftotext` (Git Bash findet es zuerst), Poppler
`%LOCALAPPDATA%\Microsoft\WinGet\Packages\oschwartz10612.Poppler_Microsoft.Winget.Source_8wekyb3d8bbwe\poppler-25.07.0\Library\bin\pdftotext.exe`,
über den vollen Pfad gerufen.

**`-v`:**

| Build | Exit | stdout | stderr |
|---|---|---|---|
| xpdf 4.06 | **99** | 83 Bytes, CRLF: `pdftotext version 4.06 [www.xpdfreader.com]` / `Copyright 1996-2025 Glyph & Cog, LLC` | leer |
| Poppler 25.07.0 | 0 | leer | 148 Bytes, CRLF: `pdftotext version 25.07.0` / `Copyright 2005-2025 The Poppler Developers - http://poppler.freedesktop.org` / `Copyright 1996-2011, 2022 Glyph & Cog, LLC` |

Die Erkennung aus E6 hält: Nur Popplers Ausgabe enthält `Poppler` (in
„The Poppler Developers“), und zwar auf stderr. xpdf schreibt `-v` auf stdout
und endet mit **99**; wer `-v` bei Exit ≠ 0 als „fehlt“ liest, meldet für xpdf
ein fehlendes Programm statt eines falschen. Beide Builds schreiben `-v` mit
CRLF.

**Je Datei, an den verbreiterten PDFs.** stdout ist bei beiden Builds für
jede Datei Byte für Byte gleich (`cmp`); verschieden sind nur Exit-Code und
stderr. `\r` gezählt mit `tr -cd '\r' | wc -c`.

| Datei | Exit xpdf | Exit Poppler | stdout | Ende (hex) | `\f` | `\r` |
|---|---|---|---|---|---|---|
| `text.pdf` | 0 | 0 | 392 Bytes: eine Zeile, 390 Zeichen, `\n`, `\f` | `… 6b 61 6e 6e 2e 0a 0c` | 1 | 0 |
| `paragraphs.pdf` | 0 | 0 | 782 Bytes: **eine** Zeile, 780 Zeichen, zweimal `Pruefbestand`, `\n`, `\f` | `… 6b 61 6e 6e 2e 0a 0c` | 1 | 0 |
| `Bericht März.pdf` | 0 | 0 | 392 Bytes, gleich `text.pdf` (`cmp`): gelesen | wie `text.pdf` | 1 | 0 |
| `blank.pdf` | 0 | 0 | 1 Byte: `\f` | `0c` | 1 | 0 |
| `allscan.pdf` | 0 | 0 | 2 Bytes: `\f\f` | `0c 0c` | 2 | 0 |
| `mixed.pdf` | 0 | 0 | 1183 Bytes: dreimal die Seite von `text.pdf` (Zeile, `\n`, `\f`), dann siebenmal `\f` | `… 2e 0a` + achtmal `0c` | 10 | 0 |
| `pageless.pdf` | 0 | **99** | 0 Bytes | — | 0 | 0 |
| `corrupt.pdf` | 1 | 1 | 0 Bytes | — | 0 | 0 |
| `encrypted.pdf` | 1 | 1 | 0 Bytes | — | 0 | 0 |

- Jede Seite endet auf `\f`, auch die letzte. Eine Textseite ist
  `<zeile>\n\f`, eine leere Seite nur `\f`, ohne Zeilenumbruch.
- **`paragraphs.pdf`:** Die drei Escapes ergeben bei beiden Builds nichts,
  weder Zeilenumbruch noch Leerzeichen: `…pruefen kann.Hallo aus dem…`. Wo
  pypdf zwei Absätze liefert, liefert `pdftotext` einen. Beide Kopien des
  Textes sind da.
- Gegen die Scan-Schwelle 100: `text.pdf`, `Bericht März.pdf` und
  `paragraphs.pdf` sind Text, `mixed.pdf` hat drei Textseiten und sieben
  Scanseiten, `blank.pdf` und `allscan.pdf` sind ganz Scan.
- `-eol unix` hält: kein `\r` in stdout. stderr bleibt CRLF, bei beiden Builds.
- stderr von `pageless.pdf` unter Poppler: `Syntax Error: Invalid page count 0`
  / `Command Line Error: Wrong page range given: the first page (1) can not be
  after the last page (0).`; unter xpdf leer.
- stderr von `corrupt.pdf`: xpdf `Syntax Error: Couldn't read xref table` /
  `Syntax Warning: PDF file is damaged - attempting to reconstruct xref
  table...` / `Syntax Error: Couldn't find trailer dictionary` /
  `Syntax Error: Couldn't read xref table`; Poppler zweimal `Syntax Error:
  Couldn't find trailer dictionary`, dann `Syntax Error: Couldn't read xref
  table`.
- stderr von `encrypted.pdf`, beide: `Command Line Error: Incorrect password`.
- Exit-Codes und stderr sind an den PDFs der Referenzbreite dieselben wie an
  den verbreiterten; dort war stdout von `text.pdf` 98 Bytes (96 Zeichen),
  `paragraphs.pdf` gleich `text.pdf`, `mixed.pdf` 301 Bytes.

**Warum hier nicht `grep -c $'\r'` zählt,** wie der Brief es vorsah: Es zählt
in diesem Git Bash aus zwei gemessenen Gründen nicht verlässlich.

1. In einem Bash-Skript verliert `$'\r'` innerhalb einer Befehlsersetzung
   `$(…)` sein CR: `c=$(printf '%s' $'\r' | od -An -tx1)` ergibt leer, auf
   oberster Ebene und über eine Variable (`x=$'\r'`; `$(… "$x" …)`) ergibt
   dasselbe `0d`. Das Muster wird leer und trifft jede Zeile. Daher die
   zuerst eingetragenen Zahlen 1, 2 und 4, die genau die Zeilenzahlen waren;
   sie sind verworfen. Auf oberster Ebene ergibt `printf 'a\nb\n' | grep -c
   $'\r'` richtig 0.
2. GNU grep 3.0 aus Git Bash entfernt ohne `-U` ein CR vor dem LF: Eine Datei
   mit den Bytes `61 0d 0a 62 0a` ergibt mit `grep -c $'\r'` 0, auch über
   stdin, mit `grep -U -c $'\r'` 1; ein CR mitten in der Zeile (`61 0d 62 0a`)
   findet es auch ohne `-U`. Ein CRLF, das die Prüfung finden soll, sieht
   `grep -c $'\r'` also auch in der richtigen Form nicht.

### Messungen: Poppler, die Aufnahme des Menschen

`testdata/convert/poppler/faketool.json` (3993 Bytes), vom Menschen mit
`loomux dev record-poppler` gegen Poppler 25.07.0 aufgenommen (`-v`: `pdftotext
version 25.07.0` / `Copyright 2005-2025 The Poppler Developers - …` /
`Copyright 1996-2011, 2022 Glyph & Cog, LLC`; der Aufzeichner legt stderr von
`-v` in `stdout` der Antwort, und zwar mit LF). Die Aufnahme deckt sich mit
der Messung oben: dieselben Exit-Codes (`corrupt.pdf` und `encrypted.pdf` 1,
`pageless.pdf` 99, sonst 0), `text.pdf` und `Bericht März.pdf` je eine Zeile
von 390 Zeichen, `paragraphs.pdf` eine Zeile von 780 Zeichen ohne etwas an der
Naht (`…pruefen kann.Hallo aus dem…`), `mixed.pdf` drei Textseiten und
siebenmal `\f`, `blank.pdf` `\f`, `allscan.pdf` `\f\f`. Die Version ist
dieselbe; der Plan musste nicht anhalten.

`TestPopplersOwnOutputConvertsAsTheSeamSays` (`internal/brain/convert/poppler_test.go`)
spielt sie durch `extract` ab: `text.pdf` und `Bericht März.pdf` eine Seite,
390 Zeichen; `paragraphs.pdf` eine Seite, ein Absatz, 780 Zeichen, zweimal
`Pruefbestand` (die Abweichung zu pypdf aus „Widersprüche zum Plan“, 2);
`mixed.pdf` zehn Seiten, sieben übersprungen, drei Absätze; `blank.pdf` eine,
`allscan.pdf` zwei Seiten, alle übersprungen, Text leer; `corrupt.pdf`,
`encrypted.pdf` und `pageless.pdf` ein Fehler.

Die Aufnahme hat kein Feld für stderr (`faketool.Answer` kennt keins). Beim
Abspielen lautet der Fehler darum `pdftotext exited 1 and said nothing` bzw.
`exited 99 and said nothing`, nicht `exited 99: Syntax Error: Invalid page
count 0` wie unter echtem Poppler (Zeile „Eine PDF ohne Seiten unter echtem
Poppler“ unter „Abweichungen“). Der Zweig mit stderr-Zeile hält darum
`TestAPagelessPDFUnderPopplerIsUnreadable`, nicht dieser Golden.

### `yt-dlp` 2026.08.19

`uvx yt-dlp --version` ergab `2026.08.19` (uvx 0.12.16). Die venv der
Referenz hat dasselbe yt-dlp 2026.08.19. Alle Läufe in `%LOCALAPPDATA%\Temp\ytm`
bzw. `…\ytd`, Video `mHSOsy_usAg` („Der echte Weg zum Second Brain: Ohne
Programmieren. Vergiss Obsidian.“, `language: de-DE`).

**Die Befehlszeile von E7 ohne `--ignore-errors`, zweimal (21:33 und 21:34
UTC, zehn Sekunden auseinander):**

```
yt-dlp --ignore-config --no-playlist --no-progress --skip-download --write-subs --write-auto-subs --sub-langs de,en --sub-format json3 --write-info-json -o v https://www.youtube.com/watch?v=mHSOsy_usAg
```

- Exit **1**, beide Male.
- Geschrieben: nur `v.de.json3`. **Kein `v.info.json`, kein `v.en.json3`**,
  keine `.part`-Reste.
- stdout (487 Bytes), beide Läufe gleich:

  ```
  [youtube] Extracting URL: https://www.youtube.com/watch?v=mHSOsy_usAg
  [youtube] mHSOsy_usAg: Downloading webpage
  [youtube] mHSOsy_usAg: Downloading visionos player API JSON
  [youtube] mHSOsy_usAg: Downloading m3u8 information
  [info] mHSOsy_usAg: Downloading subtitles: de, en
  [info] mHSOsy_usAg: Downloading 1 format(s): 616+251-1
  [info] Writing video subtitles to: v.de.json3
  [download] Destination: v.de.json3
  [download] Download completed
  [info] Writing video subtitles to: v.en.json3
  ```

- stderr (944 Bytes, LF), beide Läufe gleich: zwei Arten Warnung (kein
  JavaScript-Laufzeitsystem, „YouTube extraction without a JS runtime has been
  deprecated, and some formats may be missing“; zweimal keine
  Impersonation), dann `ERROR: Unable to download video subtitles for 'en':
  HTTP Error 429: Too Many Requests`.

**Die 429-Antwort ist aufgetreten, nicht bestellt:** Die englische Spur ist
eine automatische Übersetzung und antwortete in beiden Läufen mit 429. yt-dlp
schreibt die Info-JSON erst nach den Untertiteln; bricht eine Spur ab, endet
der Lauf, bevor die Info-JSON geschrieben ist. Übrig bleibt die deutsche Spur,
ohne Titel und ohne die Angabe, ob sie manuell oder automatisch ist. Darum
trägt die Befehlszeile jetzt `--ignore-errors` (Entscheidung „fetch“ der
Spec).

**Mit `--ignore-errors` (21:35 UTC), die Aufnahme:** dieselbe Befehlszeile mit
`--ignore-errors` vor `-o v`.

- Exit **0**. Geschrieben: `v.de.json3` (865 555 Bytes, LF, sha256
  `987229cea302aae6ec649b0c31b65f4830498d70bcd071e7978913a0bac73e42`,
  2578 Ereignisse, Byte für Byte gleich der `v.de.json3` der beiden Läufe
  ohne den Schalter) und `v.info.json` (1 123 314 Bytes); kein `v.en.json3`.
- stdout wie oben, dazu als letzte Zeile
  `[info] Writing video metadata as JSON to: v.info.json`.
- stderr (946 Bytes, LF): dieselben drei Warnungen, dann statt des Fehlers
  `WARNING: Unable to download video subtitles for 'en': HTTP Error 429: Too
  Many Requests`.
- Aus der ungekürzten Info-JSON: `subtitles` ist leer (keine manuelle Spur).
  `automatic_captions` nennt 159 Sprachen; `de` („German“) und `en`
  („English“) tragen je 14 Einträge, zweimal die Folge `json3, srv1, srv2,
  srv3, ttml, srt, vtt`; `de-orig` („German (Original)“) trägt die Folge
  einmal; `en-orig` fehlt.
- **Die zwei `json3` unter `de` sind verschiedene Spuren** (Adressen ohne
  Signatur- und Ablaufparameter gelesen): Eintrag 0 ist
  `kind=asr&lang=en-US&variant=timing-optimized&tlang=de`, eine Übersetzung
  nach Deutsch; Eintrag 7 ist `kind=asr&lang=de`, die deutsche
  Spracherkennung selbst, dieselbe Adresse wie `de-orig`. Die Befehlszeile
  von yt-dlp nimmt je Sprache die **letzte** passende Spur
  (`YoutubeDL.process_subtitles`: `matches[-1]`, gelesen in der Quelle von
  2026.08.19); die aufgenommene `v.de.json3` trägt `acAsrConf` je Wort, ist
  also die Spracherkennung. Die Referenz nimmt die **erste** `json3`
  (Eintrag 0). loomux behält die Wahl von yt-dlp; die Spec führt das als
  freigegebene Abweichung (Zeile „fetch: Spur einer Sprache“).

**Rückfall ohne json3 (21:34 UTC):**
`yt-dlp --ignore-config --no-playlist --no-progress --skip-download
--write-subs --write-auto-subs --sub-langs de --sub-format xyz -o v <url>`
endet mit Exit 0 und schreibt `v.de.vtt` (426 657 Bytes), mit der Warnung
`WARNING: No subtitle format found matching "xyz" for language de, using
vtt. Use --list-subs for a list of available subtitles`. Fehlt das verlangte
Format, schreibt yt-dlp also eine andere Endung; loomux prüft die Endung
`.json3`, bevor es liest.

**Die Referenz fragt anders** (`src/brain/convert/fetch.py:92-125`,
`_run_yt_dlp`): `YoutubeDL({"skip_download": True, "quiet": True})
.extract_info(url, download=False)`, dann die erste `json3`-Spur von `de`,
sonst `en`, zuerst unter `subtitles`, dann unter `automatic_captions`, und
lädt nur diese eine Spur über `urllib`. Sie holt nie beide Sprachen, eine
429-Antwort der englischen Übersetzung trifft sie nur, wenn es kein `de`
gibt.

### Die Aufnahme

`testdata/convert/ytdlp/mHSOsy_usAg/` hält den Lauf mit `--ignore-errors`, so
wie er ablief: `argv` (die Befehlszeile ohne `uvx`, eine Zeile, Argumente
durch Leerzeichen getrennt, ohne Shell-Anführung), `exit` (`0`), `stderr`
(wörtlich) und unter `files/` die geschriebenen Dateien: `v.de.json3`
ungekürzt und `v.info.json` gekürzt (989 Bytes, LF, ohne Zeilenumbruch am
Ende). Gekürzt mit dem Schnipsel aus dem Brief auf `id`, `title`,
`subtitles` und `automatic_captions`, dort nur `de` und `en` und je Spur nur
`ext`: Die übrigen Schlüssel liest loomux nicht, und die Adressen der Spuren
tragen Signaturen, die nach Stunden verfallen. Inhalt:
`{"id": "mHSOsy_usAg", "title": "Der echte Weg zum Second Brain: Ohne
Programmieren. Vergiss Obsidian.", "subtitles": {}, "automatic_captions":
{"en": [14 × {"ext": …}], "de": [14 × {"ext": …}]}}`, die `ext` in der
Folge oben.

### Widersprüche zum Plan und wie sie entschieden sind

1. **Keine Info-JSON nach einer scheiternden Spur** — entschieden vom Nutzer:
   Die Befehlszeile von E7 bekommt `--ignore-errors`; die Aufnahme ist der
   Lauf mit dem Schalter. Die Spec ist nachgeführt (Entscheidung „fetch“,
   Messliste).
2. **Text jenseits des Seitenrands** — entschieden vom Controller: Die
   Test-PDFs bekommen eine 5000 pt breite Seite (siehe „Die Test-PDFs“). Übrig
   bleibt `paragraphs.pdf`: `pdftotext` liefert einen Absatz, pypdf zwei.
3. **`pageless.pdf`:** Poppler endet mit 99 und einer Fehlermeldung, xpdf mit
   0 ohne Ausgabe; die Referenz sieht „keine Seiten“. Unter Poppler wird der
   Fall eine PDF, die sich nicht lesen lässt, nicht eine ohne Seiten.
   Entschieden: freigegebene Abweichung, nur stderr lautet anders (Zeile
   „Eine PDF ohne Seiten unter echtem Poppler“ unter „Abweichungen“).
4. **`-v` von xpdf endet mit 99** und schreibt auf stdout. Die Erkennung aus
   E6 hält, wenn `-v` unabhängig vom Exit-Code gelesen wird. Entschieden so:
   `pdf.go` (`resolve`) sucht `Poppler` in dem, was `-v` auf stdout und
   stderr schreibt, gleich welcher Exit-Code; ein `-v` ohne Ausgabe nennt
   den Exit-Code nur in seiner Meldung.
5. **Eine andere Spur als die Referenz** — entschieden vom Nutzer: loomux
   behält die Wahl von yt-dlp (letzte `json3` einer Sprache); freigegebene
   Abweichung in der Spec. Das Orakel von Task 9 verdeckt den Unterschied,
   weil es die aufgenommene Datei an die Stelle des Downloads setzt.

## Die Zipf-Tabelle: `internal/brain/model/zipf.go`

**Referenz:** wordfreq 3.1.1 (`zipf_frequency(wort, "de")`, `tokenize`,
`wordfreq/tokens.py`, `wordfreq/numbers.py`), mit `regex` 2026.9.10 unter
Python 3.14.7 (unidata 16.0.0). **loomux:** Go 1.27.0, Unicode 17.0.0.
Gemessen am 2026-09-27; die Batterie ist
`internal/brain/model/testdata/zipf-battery.json` (205 Wörter), die Tests
`TestTheTableAnswersWhatWordfreqAnswers` und
`TestTheSplitIsWordfreqsTokenizer`.

Der Richter der Referenz schneidet seine Teile mit Pythons `re`
(`[\wÄÖÜäöüß]+`, `judge.py`); dort ist `\w` `str.isalnum()` oder `_`:
Buchstaben, alle Zahlzeichen (Nd, No, Nl) und der Unterstrich, keine Marken,
keine Satzzeichen. wordfreq liest einen solchen Teil mit seinem eigenen
Tokenizer: `\w` des Moduls `regex` (Alphabetic, Mark, Nd, Pc, Join_Control),
Wortgrenzen nach UAX #29, dazu Fall 1 von `TOKEN_RE` für Schriften ohne
Leerzeichen. `zipfTokens` bildet ihn nach, soweit ein Teil ihn erreichen
kann: `casefold(NFC(wort))`, geschnitten an jeder Rune außerhalb von
`[\p{L}\p{Nl}\p{M}\p{Nd}\p{Pc}\x{200C}\x{200D}]`. Hoch- und tiefgestellte
Ziffern und Brüche (No) fallen dabei weg, wie bei wordfreq: `CO₂` → `co`
(4,93), `m²` → `m` (5,6), `1½` → `1` (6,18), `H₂O` → `h`, `o` (4,96). Vorher
gab loomux all diesen Band 0, und der Richter hätte `CO₂-Bilanz` gemeldet.

**Probe über jedes Zeichen, das in einem Teil stehen kann** (jedes, das
Pythons `re \w` nimmt), je in den Umgebungen `a·b`, `1·2`, allein, `ab·`,
`·ab`: `zipfTokens` gegen `tokenize(…, "de")`. Abweichungen gibt es nur in
den Schriften ohne Leerzeichen (Zeile 4); in Latein, Griechisch und
Kyrillisch keine, und jedes No-Zeichen außer elf aus Khmer und New Tai Lue
liest der Nachbau in diesen Umgebungen wie wordfreq. Das gilt nur dort: Ein
Buchstabe der Klasse Prepend (UAX #29) davor, etwa U+0D4E oder U+111C2, zieht
ein No-Zeichen in wordfreqs Token (`tokenize` behält U+0D4E mit `²` als ein
Token), loomux lässt es weg. Das Skript lag im Scratchpad (`split_check.py`,
`split_scripts.py`), nicht im Repo.

1. **Mehrere Token** (`several tokens`). wordfreq verbindet die Token eines
   Wortes über `1/f = 1/f1 + 1/f2 + …` und rundet dann; fehlt ein Token,
   antwortet es mit dem Minimum (Zipf 0). loomux nimmt das niedrigste Band
   der Token; fehlt eines, Band 0 — das ist exakt. Die kombinierte Frequenz
   liegt unter jeder einzelnen, das wahre Band also höchstens beim
   niedrigsten; genau wäre es nur mit den rohen Frequenzen, die die Tabelle
   für Wörter ohne Ziffernfolge nicht trägt (nur das Band). Es unterscheidet
   sich, wenn alle Token im selben Band nahe an dessen Untergrenze liegen:
   zwei Token mit je Zipf 3,1 ergeben etwa 2,8 (Band 1), loomux sagt Band 2.
   Das kommt nur bei Teilen mit No-Zeichen vor, denn nur dort zerfällt ein
   Teil in mehrere Token. In der Batterie stimmen `H₂O`, `CO₂qmd` (0,0) und
   `m²Adonis` (2,96). Entschieden vom Controller (T2-R1).
2. **Alphabetic als L, Nl und M.** Das `\w` des Moduls `regex` enthält
   Alphabetic (L, Nl und Other_Alphabetic); loomux nimmt L, Nl und M. Über
   alle Code Points gegen `regex` verglichen: Außerhalb von M liegen in
   Other_Alphabetic 130 Zeichen der Kategorie So, die eingekreisten und
   eingerahmten Buchstaben (ab U+24B6 und U+1F130). Sie sind kein `re \w` und
   stehen in keinem Teil. Dazu kommen 4 699 Zeichen, die Python 3.14
   (Unicode 16) noch nicht kennt, `regex` und Go (Unicode 17) schon; dort
   folgt loomux `regex`. Ohne Wirkung auf den Richter.
3. **Satzzeichen zwischen Ziffern oder Buchstaben.** UAX #29 hält `,` `.`
   `;` zwischen Ziffern in einem Token (WB11/12: `3,5`, `1.000`, geglättet
   `0,0`, `0.000`) und `:` `·` `'` `.` zwischen Buchstaben (WB6/7:
   `won't`, `z.b`). `zipfTokens` schneidet dort. Ein Teil des Richters
   enthält nie ein Satzzeichen. In der Batterie stimmen `3,5` und `0,0,0` im
   Band zufällig, `1.000` nicht (wordfreq 0,58, loomux Band 2, weil `1` und
   `000` je Band 2 haben); alle drei stehen in `splitExceptions`, `1.000`
   auch in `bandExceptions`. Ohne Wirkung auf den Richter.
4. **Schriften ohne Leerzeichen.** Fall 1 von `TOKEN_RE` macht Folgen aus
   Ideogrammen und den Schriften in `SPACELESS_SCRIPTS` (Hiragana, Katakana,
   Thai, Khmer, Lao, Myanmar, Tai Le, Tai Lü, Lanna) zu eigenen Token; Tai
   Viet und die Kana-Wiederholungszeichen U+3031 bis U+3035 trennt UAX #29
   ab. loomux hält eine Folge von Wortzeichen zusammen. Es unterscheidet
   sich, sobald ein Teil Latein mit einer dieser Schriften mischt, auch bei
   den elf No-Zeichen U+17F0 bis U+17F9 (Khmer) und U+19DA (New Tai Lue), die
   wordfreq als Token behält und loomux weglässt. Die Probe fand 107 713 solche Zeichen, davon CJK
   98 682, ohne Namen in unidata 16 6 145, Tangut 768, Khitan 471, Nushu 396,
   Hentaigana 285, Myanmar und Tai je 170, Katakana 131, Hiragana 95, New Tai
   Lue 81, Khmer 74, Thai 67, Lao 66 und kleinere Gruppen. Kein deutscher
   Text; ohne Wirkung auf den Richter.
5. **Ziffern nur ASCII** (`digits only ASCII`). `MULTI_DIGIT_RE` und
   `DIGIT_RE` nehmen mit `\d` die Ziffern jeder Schrift, glätten sie zu `0`,
   und `digit_freq` liest sie mit `int()`. loomux glättet nur `[0-9]`. Steht
   eine Ziffernfolge einer anderen Schrift (etwa arabisch-indisch) in einem
   Teil, rechnet wordfreq mit dem geglätteten Schlüssel und `digit_freq`,
   loomux schlägt den Token ungeglättet nach und findet ihn nicht (Band 0).
   U+0E51 (Thai-Eins) ist ein Schlüssel von `large_de` mit Zipf 1,09, eine
   einzelne Ziffer ohne Folge, die beide ungeglättet nachschlagen. Die
   eingebettete Tabelle trägt ihn nicht, sie hält nur Schlüssel ab Zipf 2,5;
   für den Richter liegt er so bei beiden unter 2,5.

## Die Richter: `internal/brain/model/judge.go`

**Referenz:** `src/brain/model/judge.py` und `local._reads_back`, gerufen mit
dem Python der Referenz-venv (Python 3.14.7, wordfreq 3.1.1, PyYAML 6.0.3,
regex 2026.9.3). **loomux:** Go 1.27.0, `gopkg.in/yaml.v3` v3.0.1. Gemessen
am 2026-09-27; die Batterie ist `internal/brain/model/testdata/judge-battery.json`
(72 Sätze), geschrieben von
`docs/.superpowers/parity/stufe-4d-orakel/judge_battery.py`, der Test
`TestTheJudgesAgreeWithTheReference`. Die Batterie fragt die Richter der
Referenz selbst: `is_german`, `chopped_words`, `is_one_sentence`,
`word_count`, `_reads_back`.

Die 72 Funktionswörter stehen in `judge.go`, und nur dort: 4c-2 ist gemergt
und erkennt die Richtung einer Frage an eigenen Listen
(`internal/dev/benchsearch/corpus.go`), braucht diese also nicht.
`FunctionWords()` gibt eine Kopie heraus.

**Teile, die wordfreq an einer hoch- oder tiefgestellten Ziffer teilt.** Vier
Sätze prüfen den Weg von `zipfTokens` durch den Richter; alle vier urteilen
wie die Referenz.

| Satz | Referenz | loomux | Weg |
|---|---|---|---|
| `Die CO₂-Bilanz ist in der Liste.` | nicht zerhackt | nicht zerhackt | Signal 2: `CO₂` → Token `co` (4,93), kein Fragment; `Bilanz` ist länger als fünf Zeichen. Bevor `band` wie wordfreqs Tokenizer schnitt, hätte loomux `CO₂` Band 0 gegeben und den Satz gemeldet |
| `Das Werk ist CO₂-neutral und billig.` | **zerhackt** | **zerhackt** | Signal 1: nur der erste Teil groß, `CO₂neutral` → Token `co`, `neutral`, zusammen über 3,0; in loomux ist das niedrigste Band beider Token 2 |
| `Der Euro-m²-Preis ist in der Liste.` | nicht zerhackt | nicht zerhackt | Signal 2: `m²` → Token `m` (5,6), kein Fragment |
| `Der Preis pro m² ist hoch.` | — | — | kein Bindestrich; nur `WordCount` (`²` ist No, also `re \w`, `m²` ein Wort) und `IsGerman` |

`CO₂-neutral` ist ein Fehlalarm der Referenz: Signal 1 hält das Wort für
zerschnitten, weil `co` und `neutral` zusammen häufig genug sind. loomux
übernimmt ihn (Parität). Die Näherung „niedrigstes Band“ (Zipf-Abschnitt,
Zeile 1) wirkt je Signal in eine andere Richtung: Bei Signal 2 (Fragment,
Band 0) kann loomux nur nachsichtiger sein als die Referenz, bei Signal 1
(`commonEnough` des zusammengeschobenen Wortes) nur strenger. Die Batterie
trifft keinen dieser Fälle.

**Probe über jedes Zeichen.** Für jede Rune, die Pythons `[\wÄÖÜäöüß]`
nimmt, `c.isupper()` und `c.lower()` gegen `startsUpper` und
`strings.ToLower`, dazu `re \w` gegen `[\p{L}\p{N}_]` über alle Code Points.
Die Skripte lagen im Scratchpad (`judge_probe.py`, `judgeprobe/main.go`,
`judge_probe_check.py`), nicht im Repo.

1. **`re \w` gegen `[\p{L}\p{N}_]`:** 4 657 Unterschiede, alle an Zeichen,
   die Python 3.14 (Unicode 16) nicht kennt und Go (Unicode 17) schon. Sonst
   gleich, auch an Marken: `naïve-Idee` zerfällt in beiden in `nai` und
   `ve-Idee`.
2. **`isupper()` gegen `unicode.IsUpper`:** Python liest die Eigenschaft
   Uppercase (Lu und Other_Uppercase), `unicode.IsUpper` nur Lu. Unter den
   Wortzeichen trennt das die römischen Zahlzeichen U+2160 bis U+216F (Nl).
   **Behoben:** `startsUpper` liest Lu und Other_Uppercase. Vorher meldete
   loomux `Ⅻ-jekt` nicht, die Referenz schon (Batteriesatz
   `Das Ⅻ-jekt steht bereit.`). Dazu 28 Zeichen aus Unicode 17.
3. **`lower()` gegen `strings.ToLower`:** Python bildet klein mit der vollen
   Abbildung, Go mit der einfachen; unter den Zeichen, die Python kennt,
   trennt das nur U+0130 (`İ`): Python `i` mit U+0307, Go `i`. **Behoben:**
   `pyLower`. Vorher zählte `İN` als Funktionswort `in` (Batteriesatz
   `İN DER Stadt.`: Referenz nicht deutsch, loomux deutsch). Dazu 28 Zeichen
   aus Unicode 17. Pythons Schluss-Sigma hängt vom Kontext ab und steht in
   keiner Einzelzeichenprobe; `pyLower` bildet es nicht nach, weil kein
   Funktionswort ein Sigma enthält.

**`readsBack` und der Tab.** Die Batterie fand einen Satz, an dem yaml.v3
und PyYAML auseinandergehen: `Der Tabulator\tist im Satz.` PyYAML trennt
Token nur mit Leerzeichen und weist einen Tab im Plain Scalar ab („found
character '\t' that cannot start any token“, gemessen am Anfang, mitten im
Satz, vor einem Leerzeichen und am Ende); yaml.v3 nimmt ihn mitten im Satz
als Text, und `index.ParseFrontmatter` gäbe den Satz unverändert zurück. Der
PyYAML-Nachbau in `apply` (`pyyaml_tabs.go`) weist einen solchen Kopf aber ab.
Entschieden vom Controller (2026-09-27): `readsBack` weist jeden Satz mit Tab
ab, bevor yaml.v3 ihn liest (`TestASentenceWithATabNeverReadsBack`). Damit
urteilt es auf allen Sätzen der Batterie wie PyYAML, und loomux schreibt nie
einen Kopf mit Tab, den der Nachbau abweist. Wo die beiden sonst noch
auseinandergehen, steht unter „Abweichungen“.

**Die Prämisse von E4 ist geprüft.** `TestEverySentenceTheJudgeReadsBackComesBackOutOfAHead`
schreibt jeden der 53 Sätze der Batterie, die `readsBack` durchlässt, als
`description:` in einen Kopf und verlangt ihn unverändert aus
`index.ParseFrontmatter(…, true)` zurück; alle 53 kommen zurück. Gegenprobe:
Mit umgedrehtem Vergleich meldet der Test 63 Zeilen (die 53 Sätze und die zehn
nackten Skalare unten, die loomux durchlässt).

## Erkennen, Absätze und Kopf: `internal/brain/convert`

**Referenz:** `src/brain/convert/{detect,transcript,header}.py`, gerufen mit
dem Python der Referenz-venv (Python 3.14.7) über
`docs/.superpowers/parity/stufe-4d-orakel/convert_detect.py`. **loomux:** Go
1.27.0. Gemessen am 2026-09-27.

**Zwei Befunde gegen den Plan, beide nach der Messung umgesetzt:**

1. **`Path.suffix` von Python 3.14** (`pathlib/__init__.py:461`) streift
   zuerst die führenden Punkte ab und schneidet dann ab dem letzten Punkt:
   `a.` → `.`, `..x` → leer, `a..` → `.`, `x.txt.` → `.`, `a. ` → `. `,
   `.txt`, `.` und `..` → leer. Der Plan rechnete mit der Regel älterer
   Versionen („der letzte Punkt, nicht am Anfang, nicht am Ende“: `a.` →
   leer, `..x` → `.x`). Folge: `..txt` mit einer Zeitmarke und `..pdf` sind
   in der Referenz `unsupported`, nach dem Plan wären sie Transkript und PDF
   gewesen, und das `.md`-Kriterium des Laufs hätte `..md` getroffen.
   `pySuffix` folgt 3.14. Tests: `TestPySuffixIsPathSuffix`, die Zeilen
   `..txt` und `..pdf` in `TestDetectReadsTheEndingAndThenTheHead`.
2. **`\s` und `\S` der Referenz sind Pythons Leerraum** (die 29 Zeichen von
   `str.isspace()`), Gos `\s` ist `[\t\n\f\r ]`. Gemessen: Eine Bereichszeile
   mit U+00A0, `\v`, `\x1c`, U+0085 oder U+3000 vor dem Zeilenende ist in der
   Referenz `transcript-range` und wird `[00:00] Hallo.`;
   `description:` + U+00A0 + `Satz.`, `description: Satz.` + U+00A0 und
   `description: Satz.` + U+3000 lesen sich als `Satz.`,
   `converter:` + U+3000 + `brain-pdf/1` und `converter: brain-pdf/1` +
   U+00A0 als `brain-pdf/1`; `\s*` überquert in beiden Sprachen einen
   Zeilenumbruch (`converter:\nfoo` → `foo`). Mit den ASCII-Klassen des Plans
   wäre die Bereichszeile `unsupported` gewesen und das U+00A0 Teil des
   Werts. `rangeMark`, `converterLine` und `descriptionLine` bauen auf
   `pytext.SpaceClass` und `pytext.NonSpaceClass`, die neu exportiert sind;
   `internal/brain/evidence` hält noch eine eigene Kopie (`pySpace`). Tests:
   die Zeilen `nbsp.txt` bis `ideo.txt` in
   `TestDetectReadsTheEndingAndThenTheHead`, `TestARangeLineEndsInPythonsSpace`,
   `TestTheHeadLinesReadPythonsSpace`, `TestTheSpaceClassesHoldWhatIsSpaceHolds`.

**Die Prüfung gegen `DescriptionOf`,** aus der Richter-Prüfung hierher
vertagt: `TestAnAcceptedSentenceComesBackOutOfTheHead` schreibt jeden der 53
Sätze der Batterie mit `reads_back: true` über `Head{…}.String()` in einen
Kopf und verlangt ihn unverändert aus `index.ParseFrontmatter(…, true)` und
aus `DescriptionOf` zurück. Alle 53 kommen aus beiden zurück; `readsBack`
bleibt, wie es ist. `description_of` der Referenz gibt dieselben 53 ebenfalls
unverändert zurück.

**Der Kopf Byte für Byte:** `header()` der Referenz schreibt mit leerer URL,
`retrieved` 2026-08-24, `brain-pdf/2`, `asr` falsch und ohne Satz
`---\nsource_url:\nretrieved: 2026-08-24\nconverter: brain-pdf/2\nasr: false\n---\n\n`,
mit dem Satz `Der Bericht beschreibt die Abnahme.` (2026-09-07, `pdf`) die
fünfte Zeile `description: Der Bericht beschreibt die Abnahme.` vor dem
Schlusszaun; `TestTheHeadHasFourLinesAndAFifthForADescription` vergleicht
beide ganz. Ein leerer String als Satz ergibt in der Referenz eine Zeile
`description: ` (nur `None` lässt sie weg), in loomux keine; kein Aufrufer
reicht einen: `describe` und `description_of` geben einen nicht leeren Satz
oder `None`.

**CRLF ist im Kopf ein Zeichen:** 3000 Zeilen `a\r\n` und dann `[00:00]
Hallo.\r\n` sind in der Referenz `transcript-bracket` (die Marke steht beim
Zeichen 6000); zählte `\r\n` als zwei, stünde sie bei 9000, jenseits der 8192.
Zeile `crlf-far.txt` in `TestDetectReadsTheEndingAndThenTheHead`.

## Fallsatz 4d, aufgezeichnet am 2026-09-27

Neunundzwanzig Fälle `convert/…`, aufgezeichnet am 2026-09-27 zwischen 18:33
und 18:34 UTC mit `stufe-4d-orakel/record_all.sh` gegen `brain-mcp.exe` am
Tag `loomux-3-source` (vorher geprüft: `HEAD` von ultra-brain ist `3cc72d2`),
übersetzt mit `testdata/cases/4d-map.toml` (`manifests = "verbatim"`),
abgespielt von `TestCases4d` (`internal/cli/cases_4d_test.go`). Aufgenommen
aus Git Bash; das Arbeitsverzeichnis der Aufnahme war `%LOCALAPPDATA%\Temp\r4d`
mit `bin/loomux.exe` und `fakeqmd/qmd.exe`, gebaut aus diesem Baum (`21f556c4`).
Go 1.27.0, uv 0.12.16.

**Welten.** `stufe-4d-orakel/make_worlds.py` legt je Fall eine Welt unter
`testdata/cases/4d-worlds/` an: `registry.toml` in der Weltwurzel (für beide
Seiten der Zustand), ein Bereich `knowledge` unter `vault/` mit `.brain.toml`
(`[layout] inbox = "00 Eingang"`), die Dateien des Falls im Eingang, PDFs aus
`testdata/convert/pdf/`. Die Transkripte sind die der Referenztests
(`tests/convert/test_cli_convert.py`), die Sätze der Fixtures die von
`test_local_describe.py` und `test_local_place.py`. `readonly-area` legt die
Deklaration nach `areas/knowledge/.brain.toml` des Zustands, wo beide Seiten
sie für einen `readonly`-Bereich lesen, und keine unter `vault/`: Ohne sie
hätte der Bereich keinen Eingang, und der Fall bestünde aus diesem Grund. Der
Import legt jede `.brain.toml` Byte für Byte nach `.loomux/config.toml`, auch
die unter `areas/knowledge/`. `.gitattributes` hält `4d-worlds`, `4d-source`
und `4d` als `-text`; `git ls-files --eol` zeigt für die fünf Kopien des
CRLF-Transkripts `i/crlf`. Das leere `00 Eingang` von `no-inbox` hält git
nicht fest; die Welt nennt keinen Eingang, es wirkt also nicht.

**Die Naht für `pdftotext`.** Die Referenz liest PDFs selbst mit pypdf, loomux
ruft `pdftotext`. `stufe-4d-orakel/pypdf_pages.py` (pypdf 6.16.2, die Version
der Referenz-venv) schreibt in jede der sieben Welten mit PDF eine
`faketool.json`: je PDF die Antwort auf `pdftotext -layout -enc UTF-8 -eol unix
<name> -` mit dem Text, den pypdf der Referenz liest, Seite für Seite mit `\f`
danach, und auf `pdftotext -v` eine Zeile mit `Poppler`. `corrupt.pdf`, an dem
pypdf mit `PyPdfError` scheitert, antwortet mit Exit 1 ohne Ausgabe,
`pageless.pdf` (keine Seite) mit Exit 0 ohne Ausgabe. Beim Abspielen zeigt
`convertTools` auf diese Antworten (`useFakePdftotext`); eine Welt ohne PDF hat
keine, und `Look` meldet `pdftotext` als fehlend. Damit stimmt der Kommentar an
`convertTools` in `internal/cli/convert.go`. `faketool.json` liegt auf beiden
Seiten in der Welt; die Referenz liest sie nicht.

**Attrappe.** Wie in 4c-1: `loomux dev fake-ollama` auf `127.0.0.1:11435` mit
der `ollama-fixture.json` der Welt, das Log außerhalb der Welt, beim Abspielen
`serveFakeOllama`. Die Zahl der Anfragen muss `wantOllamaCalls4d` treffen, die
aus den Notizen der Aufnahme stammt.

**Normalisierung.** `normalize4d` hält beide Seiten an zwei Stellen gleich:
`retrieved: JJJJ-MM-TT` wird `retrieved: {{DAY}}` (die gestellte Welt behält
keine Änderungszeit, `retrieved:` ist auf beiden Seiten der Tag des Laufs; B8,
E11), und `converter: brain-pdf/1` wird `brain-pdf/2` (Vorschlag 8). stderr
vergleicht der Fallsatz nicht (wie 3b und 4c-1), stdout und die ganze Welt
schon. „Gleich“ heißt unten: null Abweichungen außerhalb von stderr nach
`normalize4d`; `expected4d` ist leer.

| Fall | Eingang | Exit | Aufrufe | Ergebnis der Referenz | Abspielen |
|---|---|---|---|---|---|
| `transcript-bracket` | `video (mHSOsy_usAg).txt`, Klammermarken | 0 | 0 | `video (mHSOsy_usAg).txt.md`, `source_url: https://www.youtube.com/watch?v=mHSOsy_usAg`, ein Absatz `[00:00] …` | gleich |
| `transcript-range-lead-in` | `export.txt`: Titelzeile, Bereichszeilen | 0 | 0 | Vorspann `Titel des Videos` als eigener Absatz, dann `[01:05] Hallo zusammen. Spät im Video.`; die Marke `01:02:03` fällt heraus und wird kein Anker, weil der Absatz unter der Schwelle weiterläuft (die Umrechnung zu `[62:03]` hält `TestMarksBecomeMinutes`) | gleich |
| `crlf-transcript` | `video.txt` mit CRLF | 0 | 0 | `[00:00] Hallo zusammen. Und weiter.`, kein `\r` im Ziel | gleich |
| `mixed-case-order` | `B.txt`, `a.txt`, `_z.txt` | 0 | 0 | stdout in der Folge `_z.txt.md`, `a.txt.md`, `B.txt.md` (B4) | gleich |
| `same-stem` | `doku.pdf` (`text.pdf`), `doku.txt` | 0 | 0 | `doku.pdf.md` und `doku.txt.md` | gleich |
| `unsupported-and-good` | `notiz.txt` (Prosa), `video.txt` | 1 | 0 | `video.txt.md`; `skipped: notiz.txt: no converter knows this format` | gleich |
| `hand-written-target` | `video.txt`, `video.txt.md` von Hand | 1 | 0 | nichts geschrieben; `skipped: video.txt.md: not written by us, left untouched` | gleich |
| `broken-utf8-source` | `kaputt.txt` (8 328 saubere Bytes, dann `\xff\xfe`), `video.txt` | 1 | 0 | `video.txt.md`; `skipped: kaputt.txt: cannot be read as UTF-8 text` — erkannt als Transkript, erst das volle Lesen scheitert | gleich |
| `broken-utf8-target` | `video.txt`, `video.txt.md` aus `\xff\xfe` | 1 | 0 | nichts geschrieben; `skipped: video.txt.md: cannot be read as UTF-8 text` | gleich |
| `pdf-text` | `buch.pdf` (`text.pdf`) | 0 | 0 | `buch.pdf.md`, `converter: brain-pdf/1` | gleich |
| `pdf-umlaut-name` | `Bericht März.pdf` | 0 | 0 | `Bericht März.pdf.md` | gleich |
| `pdf-scan` | `scan.pdf` (`blank.pdf`), `video.txt` | 1 | 0 | `video.txt.md`; `skipped: scan.pdf: no extractable text, looks like a scan` | gleich |
| `pdf-pageless` | `leer.pdf` (`pageless.pdf`) | 1 | 0 | nichts; `skipped: leer.pdf: no pages to extract` | gleich über die Naht; unter echtem Poppler eine andere Meldung, siehe „Abweichungen“ |
| `pdf-partial` | `buch.pdf` (`mixed.pdf`) | 1 | 0 | `buch.pdf.md` mit den drei Textseiten; `skipped: buch.pdf: 7 page(s) skipped as scanned` | gleich |
| `pdf-corrupt` | `buch.pdf` (`corrupt.pdf`), `video.txt` | 1 | 0 | `video.txt.md`; `skipped: buch.pdf: cannot be read as a PDF (Stream has ended unexpectedly)`, davor pypdfs Warnung `EOF marker not found` | gleich; loomux sagt `… (pdftotext exited 1 and said nothing)` |
| `no-inbox` | — (Bereich ohne `[layout] inbox`) | 0 | 0 | nichts, keine Zeile | gleich |
| `readonly-area` | `video.txt`, `readonly = true` | 0 | 0 | nichts, obwohl die Deklaration einen Eingang nennt | gleich |
| `single-file` | `video.txt`, Befehl mit dem Pfad | 0 | 0 | `video.txt.md` | gleich |
| `no-registry` | `video.txt`, keine `registry.toml` | 1 | 0 | nichts; `error: [Errno 2] No such file or directory: '…\\registry.toml'` | gleich; loomux sagt `error: open …\registry.toml: Das System kann die angegebene Datei nicht finden.` |
| `broken-manifest` | `video.txt`, zweiter Bereich mit `mode = "cloud"` | 1 | 0 | nichts geschrieben; `error: …\x\.brain.toml: [privacy] mode must be one of automatic_cloud, local_only, manual_cloud, found 'cloud'` | gleich; loomux nennt `…\x\.loomux\config.toml` und `"cloud"` |
| `describe-kept` | `video.txt`, `describe` an | 0 | 1 | Kopf mit `description: Der Bericht beschreibt die Abnahme der zweiten Scheibe.` | gleich |
| `describe-refused` | wie oben, englischer Satz | 0 | 1 | Satz verworfen, Kopf mit vier Zeilen | gleich |
| `second-run` | `video.txt`, `video.txt.md` eines ersten Laufs | 0 | 0 | Ziel neu geschrieben und auf stdout, `retrieved:` der Tag des Laufs, der stehende Satz bleibt; die Attrappe wird nicht gefragt | gleich |
| `place-suggested` | `video.txt`, `place` an, zweiter Bereich `project/x` | 0 | 1 | `video.txt.md` und `suggested: video.txt.md: belongs in project/x, left in the inbox` auf stdout; nichts bewegt | gleich |
| `place-unknown-scope` | wie oben, Fixture `project/erfunden` | 0 | 1 | `video.txt.md`, kein Vorschlag | gleich |
| `model-unreachable` | `video.txt`, Endpunkt `:11436` | 0 | 0 | Kopf mit vier Zeilen | gleich |
| `model-off-in-area` | `video.txt`, im Bereich `[model] enabled = false` | 0 | 0 | Kopf mit vier Zeilen; die Attrappe lauscht und wird nicht gefragt | gleich |
| `broken-model-block` | `video.txt`, `[model] enabled = 5` | 1 | 0 | nichts; `error: …\config.toml: [model] enabled must be a boolean, found 5` | gleich; loomux sagt `found integer` |
| `endpoint-off-loopback` | `video.txt`, Endpunkt `http://192.0.2.1:11434` | 1 | 0 | nichts; `error: [model] endpoint must stay on the loopback (127.0.0.1, ::1, localhost), found 'http://192.0.2.1:11434'` | gleich, auch stderr Wort für Wort |

**`second-run`** ist der „zweite Lauf“ aus Vorschlag 10, so weit eine gestellte
Welt ihn trägt. Sie behält keine Änderungszeit (B8), `retrieved:` ist auf
beiden Seiten der Tag des Laufs, und der feste Tag `2020-01-01` im Ziel lässt
beide den Kopf neu schreiben; stdout nennt auf beiden Seiten die Zieldatei.
Ein vergangener Tag, nicht der Tag der Aufnahme: Sonst schriebe die Referenz
nichts und ein späteres Abspielen doch. Was der Fall an der Referenz misst:
Das Ziel gilt als eigenes, der stehende Satz bleibt, und das Modell wird nicht
gefragt (`ollama calls: 0`), obwohl `describe` an ist und die Fixture einen
anderen Satz hätte. Dass ein zweiter Lauf gar nichts schreibt, halten die
Go-Tests (`TestTheSecondRunWritesNothing`, `TestAStandingSentenceIsNeverAskedFor`,
`TestConvertNamesWhatItWroteAndExitsZero`).

**Keine Welt macht ein Ziel unschreibbar.** Eine Zeile `cannot be written`
trüge einen `*os.LinkError` mit dem zufälligen Namen der Zwischendatei
(`<ziel>.<ziffern>.tmp`); kein Fall trifft sie, und eine Normalisierung dafür
braucht der Fallsatz nicht.

**Was der Fallsatz unterscheidet.** Je Regel ein Mutant über `go test
-overlay`, der Baum blieb unberührt; die Skripte lagen im Scratchpad
(`mut4d/make_mutants.py`, `run_mutants.sh`), nicht im Repo.

| Mutant | rot |
|---|---|
| `sortedOn` nach Bytes statt klein geschrieben | `mixed-case-order` |
| `Look` der Naht meldet immer „fehlt“ | `pdf-partial`, `pdf-text`, `pdf-umlaut-name`, `same-stem` (die Naht wird benutzt) |
| `normalize4d` ohne Angleichung von `brain-pdf/1` | dieselben vier (die Angleichung wirkt) |
| `normalize4d` ohne Angleichung von `retrieved:` | keiner — Aufnahme und Abspielen am selben Tag |
| jede Datei der gestellten Welt auf den 2001-02-03 datiert, Angleichung von `retrieved:` bleibt | keiner (ein anderer Tag besteht) |
| dasselbe ohne Angleichung von `retrieved:` | die 20 Fälle, die eine Datei schreiben |
| Quelle roh gelesen (`os.ReadFile` statt `pytext.ReadText`) | `broken-utf8-source`; `crlf-transcript` bleibt grün, weil `split` jedes `\r` als Leerraum schluckt |
| Quelle roh gelesen und `split` trennt nur an Leerzeichen und `\n` | `broken-utf8-source`, `crlf-transcript` (ein `\r` im Ziel fällt auf) |
| `readonly` beim Eingang nicht geprüft | `readonly-area` |
| stehender Satz nicht übernommen | `second-run` |
| `Place` nimmt jeden nicht leeren Bereich | `place-unknown-scope` |
| Exit 0 trotz `skipped:` | die acht Fälle mit Exit 1 aus `skipped:` |

Die Batterien der Richter und der Zipf-Tabelle stehen oben („Die
Zipf-Tabelle“, „Die Richter“), mit ihren Ausnahmen.

## Abweichungen

Auf allen 72 Sätzen der Batterie urteilen `IsGerman`, `ChoppedWords`,
`IsOneSentence`, `WordCount` und `readsBack` wie die Referenz.

| Abweichung | Art | Begründung |
|---|---|---|
| `readsBack` auf nackten Skalaren, die YAML 1.1 anders auflöst als yaml.v3 | yaml.v3 statt PyYAML (E4), unerreichbar | Gemessen am 2026-09-27 mit `_reads_back` der Referenz (PyYAML 6.0.3), Fund des Reviews. Die Liste ist nicht vollständig, sie nennt Beispiele. PyYAML `false`, loomux `true`: z. B. `yes`, `on`, `no`, `off`, `Yes`, `NO` (PyYAML: bool), `1:20` (PyYAML: int 80), `1:20.` (PyYAML: float 80.0), `=` und `<<` (PyYAML: ConstructorError), außerhalb des Tests auch `2001-12-14 21:59:43.10 -5` (PyYAML: datetime). PyYAML `true`, loomux `false`: z. B. `0o17`, `1e3`, `-.5` (yaml.v3: Zahlen), außerhalb des Tests auch `1.5e3`. Jedes dieser Muster füllt einen ganzen Skalar ohne zwei getrennte Wörter; `IsGerman` verlangt zwei verschiedene Funktionswörter und weist die 13 des Tests ab wie die beiden übrigen (die Referenz ebenso; gemessen am 2026-09-27), und `describe` nimmt einen Satz nur, wenn alle Richter ihn durchlassen. Die Abweichung erreicht `describe` also nie. Test: `TestABareScalarPartsFromPyYAMLOnlyWhereIsGermanRefuses` (hält beide Urteile, `IsGerman` falsch, und liest die zehn, die loomux durchlässt, über `index.ParseFrontmatter` zurück) |
| Ziffern nur ASCII: Zeitmarken und Zipf-Glättung | ASCII-Ziffern statt Pythons `\d`, entschieden im Plan | Dieselbe Regel an zwei Stellen. In der Zipf-Tabelle glättet loomux nur `[0-9]`, wordfreq jede Dezimalziffer (Abschnitt „Die Zipf-Tabelle“, Zeile 5, `digits only ASCII`); eine Ziffernfolge einer anderen Schrift in einem Teil eines Satzes von `describe` ist kein Fall, den ein Eingang trifft. In den Zeitmarken der Transkripte, gemessen am 2026-09-27 mit `convert_detect.py`: `[٠٠:٠٥] Hallo.` (arabisch-indische Ziffern) ist in der Referenz `transcript-bracket` und wird `[00:05] Hallo.`, weil `int()` jede Dezimalziffer liest; loomux nennt die Datei `unsupported`. Steht eine solche Marke in einem Transkript mit ASCII-Marken, schneidet die Referenz sie als Marke heraus: Ihr Text entfällt, und Anker wird sie nur, wo an ihr ein Absatz beginnt. loomux lässt sie als Text im vorigen Fragment stehen, und Absatzgrenzen können sich verschieben. Gemessen an `[00:00] Eins.\n[٠٠:٠٥] Zwei.\n[00:09] Drei.\n`: Referenz `[00:00] Eins. Zwei. Drei.`, mit Schwelle 1 `[00:00] Eins.\n\n[00:05] Zwei.\n\n[00:09] Drei.`; loomux `[00:00] Eins. [٠٠:٠٥] Zwei. Drei.`, mit Schwelle 1 `[00:00] Eins. [٠٠:٠٥] Zwei.\n\n[00:09] Drei.`. Tests: `TestDetectTakesASCIIDigitsOnly`, `TestAMarkInOtherDigitsStaysText` |
| Ein kaputtes Byte kurz hinter den ersten 8192 Zeichen | Implementierungsdetail von CPython, nicht nachgebaut | `TextIOWrapper.read(8192)` dekodiert ganze Blöcke (der erste 8192 Bytes, jeder weitere so groß wie die noch fehlenden Zeichen mal Bytes je Zeichen des vorigen Blocks, mindestens 8192) und scheitert an jedem kaputten Byte darin, auch hinter dem 8192. Zeichen; loomux liest genau 8192 Zeichen. Gemessen am 2026-09-27 mit `convert_detect.py`: `[00:00] ` + 4092 × `ä` (zusammen 8192 Bytes) + 4102 × `x` + `\xff` ist in der Referenz `unsupported`, in loomux `transcript-bracket`. Die Blockgröße ist mitgemessen: Nach `[00:00] ` + 2046 × U+1F600 (8192 Bytes, 2054 Zeichen) liest der zweite Block int(8192 / 2054 × 6138) = 24 480 Bytes; ein kaputtes Byte 10 000 oder 24 479 Bytes nach dem ersten Block macht die Datei `unsupported` (feste Blöcke von 8192 Bytes hätten es nie gelesen), eines 24 480 Bytes danach nicht mehr. Die Grenze kann nur fallen, wenn die ersten 8192 Zeichen mehr als 8192 Bytes brauchen (Umlaute, CRLF). Der Ausgang bleibt derselbe: Nichts wird geschrieben, und der Lauf meldet die Datei; nur der Grund lautet anders (Referenz „no converter knows this format“, loomux der Fehler des vollen Lesens oder, wenn die Zieldatei schon steht, deren Meldung). Test: `TestDetectReadsNoFurtherThanTheHead` |
| PDF über `pdftotext` statt pypdf | freigegeben 2026-09-26 | Entscheidung „PDF“ der Spec. Die Fälle messen über die Naht bei den Seiten (`faketool.json` aus `pypdf_pages.py`, „Fallsatz 4d“), die Poppler-Goldens (Task 13) das echte Werkzeug. Ein Unterschied des Extraktors ist gemessen: In `paragraphs.pdf` liefert pypdf zwei Absätze, `pdftotext` einen („Widersprüche zum Plan“, 2) |
| `converter: brain-pdf/2` | freigegeben 2026-09-26 | Vorschlag 8; in den Fällen angeglichen (`normalize4d`). Tests: `TestTheHeadHasFourLinesAndAFifthForADescription`, `TestTwoSourcesWithTheSameStemBothSurvive` |
| Meldung für eine unlesbare PDF | Meldungstext | Die Referenz nennt die Ausnahme von pypdf (an `corrupt.pdf`: `buch.pdf: cannot be read as a PDF (Stream has ended unexpectedly)`, davor pypdfs Warnung `EOF marker not found` auf stderr), loomux `pdftotext exited <n>: <erste Zeile von stderr>`, über die Naht `pdftotext exited 1 and said nothing`. Der Ausgang ist gleich (übersprungen, nichts geschrieben, Exit 1); stderr wird nicht verglichen. Fall `convert/pdf-corrupt`, Test `TestAPDFPdftotextRefusesIsUnreadable` |
| Eine PDF ohne Seiten unter echtem Poppler | Folge der Entscheidung „PDF“, gemessen am 2026-09-26 | Poppler 25.07.0 endet an `pageless.pdf` mit 99 und `Syntax Error: Invalid page count 0` („Messungen“); loomux meldet `leer.pdf: cannot be read as a PDF (pdftotext exited 99: Syntax Error: Invalid page count 0)`, die Referenz `leer.pdf: no pages to extract`. Der Ausgang ist gleich: übersprungen, nichts geschrieben, Exit 1; nur stderr lautet anders. Über die Naht (pypdf findet keine Seite, also Exit 0 ohne Ausgabe) sagen beide „no pages“, darum hält der Fall `convert/pdf-pageless` den Unterschied nicht fest. xpdf endet dort mit 0 ohne Ausgabe, wird aber vorher abgewiesen („Nur Poppler“). Test: `TestAPagelessPDFUnderPopplerIsUnreadable` |
| Nur Poppler | freigegeben 2026-09-26 | Vorschlag 12; xpdf wird wie ein fehlendes Programm behandelt (Vorschlag 9), jede PDF des Laufs eine Zeile `skipped:`. Tests: `TestOnlyPopplerIsTaken`, `TestXpdfLeavesThePDFsAndTheTranscriptGoes` |
| `place`: Antworten, die nur Pythons `json` liest | `encoding/json` statt `json.loads`; erreichbar nur, wenn ein Endpunkt das Schema nicht einhält | Gemessen am 2026-09-27 mit dem Python der Referenz-venv (3.14.7; das Skript `place_json_probe.py` lag im Scratchpad), `json.loads` wie in `LocalProposer.place` (`local.py:121-145`, `json.loads` in Zeile 137), Antwort `{"scope": "project/x", "grund": …}`: Mit `NaN`, `Infinity`, `-Infinity` oder `1e400` (Python: `inf`) liest die Referenz das Objekt und legt nach `project/x` ab; `encoding/json` weist alle vier ab, loomux legt nichts ab. Verschachtelung: `encoding/json` nimmt höchstens 10 000 Ebenen, das Objekt und 9 999 Listen darin gehen, 10 000 nicht; Python liest tiefer und legt ab, bis `json.loads` an der Stapeltiefe mit `RecursionError` scheitert — gemessen ab 11 324 Listen im Objekt, aus einer flachen Aufrufkette (im Lauf von `convert` steht der Aufruf tiefer im Stapel, die Grenze also eher darunter; sie hängt am Stapel, nicht an einer Zahl). `local.py:138` fängt nur `JSONDecodeError`: Der `RecursionError` beendet den ganzen Lauf der Referenz, loomux läuft ohne Ablage weiter. Das Schema in `format` verlangt für `grund` eine Zeichenkette, ein Ollama, das es einhält, schickt keine dieser Antworten. Test: `TestPlaceRefusesWhatOnlyPythonsJSONReads` (mit der Gegenprobe bei 10 000 Ebenen) |
| Leerer Satz im Kopf | unerreichbar | `header(description="")` der Referenz schreibt eine Zeile `description: ` (nur `None` lässt sie weg), loomux schreibt für `""` keine („Der Kopf Byte für Byte“). Kein Aufrufer reicht einen leeren Satz: `describe` gibt einen nicht leeren Satz oder `None`, `description_of` ebenso, auf beiden Seiten. Test: `TestTheHeadHasFourLinesAndAFifthForADescription` (der Kopf ohne Satz hat vier Zeilen) |
| `--channel` und `--state-dir` an `convert` und `fetch` | Flags | Die Referenz nimmt beide über `_add_common` (`cli.py:450-452`) an beiden Befehlen an; `--channel` (`local` oder `cloud`) wirkt dort bei keinem der beiden, denn `_convert` und `_fetch` bekommen es nicht. loomux kennt keines und endet mit Exit 2 (`flag provided but not defined`), bevor etwas gesucht oder gestartet ist. `--state-dir` folgt der in 1b-1 freigegebenen Abweichung: Der Zustand kommt aus `LOOMUX_STATE_DIR` und `LOOMUX_LEGACY_BRAIN_DIR`. Test: `TestConvertAndFetchTakeNoChannelAndNoStateDir` |
| Fehlermeldungen in der Form von loomux | Meldungstext | stderr wird nicht verglichen. Gemessen in den Fällen: `[model] enabled` nennt die Referenz mit dem Wert (`found 5`), loomux mit dem Typ (`found integer`); `[privacy] mode` setzt die Referenz in einfache, loomux in doppelte Anführungszeichen, und loomux nennt die übersetzte Datei `.loomux\config.toml`; eine fehlende Registry meldet die Referenz als `[Errno 2] No such file or directory: '…'`, loomux mit der Meldung des Systems. `endpoint-off-loopback` ist Wort für Wort gleich |
| `fetch` über das Programm `yt-dlp`, das die Datei schreibt | freigegeben 2026-09-26 | Entscheidung „fetch“; Randfall einer manuellen Spur ohne json3 |
| `fetch`: innerhalb einer Sprache die letzte json3-Spur (Wahl von yt-dlp), die Referenz nahm die erste | freigegeben 2026-09-26 | Zeile „fetch: Spur einer Sprache“ der Spec; für `mHSOsy_usAg` die deutsche Spracherkennung statt der Übersetzung aus `en-US`. Das Orakel von Task 9 verdeckt es, weil es die aufgenommene Datei an die Stelle des Downloads setzt |
| `fetch` mit `--no-playlist`, `--ignore-config` und `--ignore-errors` | Entscheidung „fetch“, E7 | Eine Adresse mit `&list=` holt nur das Video; eine scheiternde Spur (429) beendet yt-dlp nicht vor der Info-JSON. Test: `TestYtdlpIsCalledOnceInAFreshDirectory` |
| `fetch` verweigert eine URL, die nur eine Playlist nennt | Entscheidung des Nutzers 2026-09-28 | `--no-playlist` grenzt nur eine Watch-URL mit `&list=` auf ihr Video ein; die Playlist-Seite (`/playlist` auf `youtube.com`, `www.`, `m.`, `music.youtube.com`) grenzt es nicht ein, yt-dlp liefe jeden Eintrag in dieselben Namen `v.*`. Dasselbe gilt für `/watch` mit `list=` ohne (oder mit leerem) `v=`: yt-dlp 2026.08.19 leitet sie in `YoutubeTabIE._real_extract` (`extractor/youtube/_tab.py`, „Common mistake: https://www.youtube.com/watch?list=playlist_id“) auf `/playlist?list=…` um, bevor `_yes_playlist` `--no-playlist` fragt. loomux endet mit Exit 2, bevor ein Werkzeug gesucht ist; Watch-URL mit `v=` und `list=` und `youtu.be/<id>?list=` gehen weiter; `v` und `list` zählen wie in yt-dlps `parse_qs` erst mit einem nicht leeren Wert (`watch?v=&v=<id>&list=…` geht weiter, `watch?list=&list=…` nicht), und ein Schema zählt nur am Anfang (`youtube.com/watch?list=…&next=https://x` wird als https gelesen). Nicht verweigert, obwohl yt-dlp sie ebenfalls als Playlist liest: eine nackte Playlist-ID (`YoutubePlaylistIE`, `_tab.py:2445` ff., `_VALID_URL` mit optionalem Host-Teil bis `:2455`), ein anderer `youtube.com`-Pfad mit `list=` ohne `v=` wie `/embed/videoseries` (dieselbe `_VALID_URL`, `/.*?\?.*?\blist=`), eine andere Subdomain von `youtube.com` und `youtubekids.com` (`YoutubeTabIE._VALID_URL`, `(?:\w+\.)?youtube(?:kids)?\.com`, `_tab.py:1071` ff., `playlist|watch` in `:1082`). Entscheidung 2026-09-28: Diese Formen werden nicht verweigert, nur in der CLI-Referenz benannt. Die Referenz ruft `extract_info(url, download=False)` ohne `noplaylist` (`fetch.py:107-108`, aus dem Code gelesen, nicht gemessen): Sie löst die Einträge auf, das Playlist-Wörterbuch trägt oben keine `subtitles`, und sie endet mit `<url>: no subtitle track to fetch, and this system does no ASR`. Tests: `TestFetchRefusesAURLThatNamesOnlyAPlaylist`, `TestFetchTakesAVideoURLThatAlsoNamesAPlaylist` |
| `fetch` in einen `readonly`-Bereich verweigert | freigegeben 2026-09-26 | Vorschlag 2, geheilte Lücke. Test: `TestFetchRefusesAnAreaItCannotWriteInto` |
| `convert` und `fetch` verweigern bei `[modules] brain = false` | Spec, Vorschlag 3, E10 | Modul Brain. Tests: `TestConvertRefusesWhereTheProjectSwitchedTheBrainOff`, `TestFetchRefusesWhereTheProjectSwitchedTheBrainOff` |
| Frist für `pdftotext` (2 min) und `yt-dlp` (10 min) | Plan E6, E7 | Die Referenz hatte keine. Tests: `TestPdftotextGetsTheNameAndTheInbox`, `TestYtdlpIsCalledOnceInAFreshDirectory` |

`readsBack` über yaml.v3 (Plan E4) hat keine eigene Zeile: Wo es von PyYAML
abweicht, steht in der ersten Zeile oben (nackte Skalare, unerreichbar über
`IsGerman`); der Tab ist behoben („Die Richter“). Die Stichprobe der Erkennung
ist die Zeile „Ein kaputtes Byte kurz hinter den ersten 8192 Zeichen“.

## Mutanten

`bin/loomux.exe dev mutants ./internal/brain/convert ./internal/brain/model ./internal/programs`
am 2026-09-27, 21:05 bis 21:08, auf c5477e58. Erster Lauf: `convert` 356 Mutanten, davon 56
nicht übersetzbar, 19 von 300 überlebten; `model` 236, 55, 12 von 181; `programs` 4, 0, 0 von 4.
`pull.go` in `model` stammt von `master` (vor 4d), seine fünf Überlebenden sind trotzdem hier
behandelt, weil der Lauf das ganze Paket nimmt.

Getötet durch neue oder geschärfte Tests (19):

| Stelle | Mutante | Test |
|---|---|---|
| `convert/detect.go:96` | `count < headChars` → `<=` | `TestTheHeadEndsAtItsLastCharacter` (Marke endet auf Zeichen 8192 bzw. 8193) |
| `convert/detect.go:98` | `if false`; `size <= 1` → `size < 1` | `TestABrokenByteInTheHeadIsNoTextButAReplacementCharacterIs` (kaputtes Byte hinter einer Marke) |
| `convert/detect.go:98` | `r == utf8.RuneError && size <= 1` → `r == utf8.RuneError` | ebenda (ein ausgeschriebenes U+FFFD ist ein Zeichen) |
| `convert/fetch.go:38` | `[<>…]` → `[<=>…]` | `TestTheTitleBecomesAUsableName`, Fall `a=b <c>` |
| `convert/fetch.go:162` | `i >= 0` → `i > 0` | `TestAChosenTrackYtdlpDidNotWriteNamesWhy`, Fall mit der Sprachzeile als erster Zeile |
| `convert/header.go:73` | `if !HasPrefix(text, "---\n")` → `if false` | `TestConvertedByKnowsOurOwnFileOnly`, Fall `converter: …\n---\n` ohne Kopf |
| `convert/run.go:49` | `if rel != ""` → `if true` | `TestAreasWithoutAWritableInboxAreLeftAlone` (ein Transkript im Bereich ohne `[layout] inbox`) |
| `convert/run.go:111` | `if err != nil` → `if false` | `TestAnEndpointOffTheLoopbackStopsTheRunWhenOnlyDescribeIsOn` |
| `convert/run.go:229` | `e.skipped > 0` → `true`, `>= 0` | `TestTwoSourcesWithTheSameStemBothSurvive` (eine PDF ohne Scanseite hinterlässt keine Zeile) |
| `convert/transcript.go:36` | `size(current) > limit` → `>=` | `TestAParagraphAtTheThresholdStaysOpen` |
| `model/pull.go:30`, `:90` | `StatusCode > 299` → `>= 299` | `TestOnlyA2xxStatusIsAnAnswer` (299 ist eine Antwort) |
| `model/pull.go:30`, `:90` | `StatusCode < 200 \|\| …` → nur `> 299` | ebenda (101 über einen gekaperten Server, der einzige Status unter 200, den Gos Client als Antwort zurückgibt) |
| `model/pull.go:93` | `… == nil && answer.Error != ""` → `… == nil` | `TestAPullRefusalWithAnEmptyErrorNamesTheStatusOnly` |
| `model/zipf.go:163`, `:165` | `zipf >= 3.0` → `>`, `zipf >= 2.5` → `>` | `TestBandOfTakesEachThresholdIntoTheBandAbove` |

Zweiter Lauf: `convert` 7, `model` 5, `programs` 0 überleben, alle äquivalent:

| Stelle | Mutante | Warum äquivalent |
|---|---|---|
| `convert/detect.go:85` | `if err != nil` (nach `os.Open`) → `if false` | `os.Open` gibt bei einem Fehler ein `nil`-`*os.File`; `Read` darauf gibt `os.ErrInvalid`, `Close` ebenso ohne Panik. `readHead` endet dann in Zeile 91 mit `false`, wie die Abkürzung. |
| `convert/detect.go:91` | `if err != nil && …` → `if false` | Einen anderen Lesefehler als `EOF` liefert eine lokale Datei nur, bevor sie ein Byte gab (ein Verzeichnis): Der Kopf ist dann leer, trägt keine Marke, und `Detect`, der einzige Aufrufer, sagt `Unsupported` wie bei `false`. |
| `convert/detect.go:91` | `err != io.EOF` → `err == io.EOF` | `io.ReadFull` gibt `io.EOF` nur, wenn es kein Byte las (leere Datei); der leere Kopf trägt keine Marke, `Detect` sagt `Unsupported` so oder so. Einen anderen Lesefehler, den die Mutante nun durchlässt, liefert eine lokale Datei nur, bevor sie ein Byte gab (ein Verzeichnis), wie in der Zeile darüber: Auch dann ist der Kopf leer, und `Detect` sagt `Unsupported`. |
| `convert/header.go:77` | `if end == -1` → `if false` | Der Kopf wäre `text[:2]`, also `--`; weder `^converter:` noch `^description:` trifft darin, `headLine` gibt `false` wie die Abkürzung. |
| `convert/pdf.go:150` | `if out == ""` → `if false` | `strings.Split("", "\f")` ist `[""]`, die leere letzte Seite fällt weg: eine leere Liste statt `nil`, gleich für `len` und `range`. |
| `convert/run.go:108` | `if settings.Enabled` → `if true` | `ProposerFor` fragt `RoleOn`, und `Narrowed` hält `Enabled` nur, wo es die Maschine setzt: Ohne es gibt jede Rolle `nil, nil`, die Karten bleiben leer. |
| `convert/run.go:138` | `if pl != nil` → `if true` | Die Liste `scopes` erreicht nur `Place`, und `one` ruft es nur mit einem `placer`; ohne ihn endet `one` vorher. |
| `model/judge.go:99` | `if tightened == ""` → `if false` | Für `""` ist die Zahl der Satzenden 0, der erste Teil des `&&` falsch, und der Index `len-1` wird nie gelesen. |
| `model/zipf.go:223` | `r >= '0' && r <= '9'` → `r >= '0'` | `multiDigit` lässt nur Ziffern, `.` und `,` in einen Lauf; `.` (0x2E) und `,` (0x2C) liegen unter `0`. |
| `model/zipf.go:223` | `r >= '0'` → `r > '0'` | `0` wird zu `0`, mit oder ohne Ersetzung. |
| `model/zipf.go:257` | `year <= referenceYear` → `<` | Bei 2019 gibt der erste Zweig `yearLogPeak − 0,0083 · 0`, der zweite `yearLogPeak`: dieselbe Zahl. |
| `model/zipf.go:259` | `year <= referenceYear+plateauWidth` → `<` | Bei 2039 gibt der Zweig danach `yearLogPeak − 0,2 · 0`, also wieder `yearLogPeak`. |

## Selbstnutzung

Am 2026-09-27 durch den Menschen, in PowerShell mit `bin\loomux.exe convert`,
am echten Eingang `brain-knowledge/00 Eingang` (Bereich `knowledge`,
`manual_cloud`), Poppler 25.07.0, Modell aus. Die Namen der Dateien sind
privat und stehen hier nicht; `<scan>` und `<teilweise>` vertreten sie.

**Der Eingang.** Zwei eigene PDFs (Angebote), eine ganz gescannt, eine
teilweise; die Transkripte kamen am 2026-09-28 dazu (Abschnitt
„Transkripte“ unten).

| PDF | Seiten | Zeichen je Seite nach `pytext.Strip` | Ergebnis |
|---|---:|---|---|
| `<scan>.pdf` | 5 | jede 0 | nichts geschrieben: `skipped: <scan>.pdf: no extractable text, looks like a scan` |
| `<teilweise>.pdf` | 41 | 7 Scanseiten je 0; 34 Textseiten, kleinste 142, größte 18 528 | `<teilweise>.pdf.md` geschrieben, dazu `skipped: <teilweise>.pdf: 7 page(s) skipped as scanned` |

**Die Scan-Schwelle 100 trägt an echtem Poppler.** Keine Textseite liegt
darunter (die kleinste hat 142 Zeichen), keine Scanseite darüber (alle 0).
Der Plan musste nicht anhalten; die Schwelle bleibt (Entscheidung „PDF“).

**Der Kopf** von `<teilweise>.pdf.md`: `converter: brain-pdf/2`,
`retrieved: 2025-09-11` (das Änderungsdatum der Quelle), `asr: false`, keine
Zeile `description` (Modell aus). Die Zieldatei behält oder löscht der
Mensch; kein Agent hat im Eingang geschrieben.

**Der erste Lauf** gab das Ziel auf stdout und die beiden `skipped:`-Zeilen
auf stderr aus, Exit 1. **Jeder weitere Lauf** wiederholt die beiden
`skipped:`-Zeilen mit Exit 1 und schreibt nichts neu. Der Plan erwartete für
den zweiten Lauf keine Ausgabe und Exit 0; das gilt nur für einen Eingang
ohne Scan. Eine Scanseite bleibt im Eingang liegen und wird bei jedem Lauf
gemeldet, in der Referenz ebenso (`convert/pdf-scan` und `convert/pdf-partial`
unter „Fallsatz 4d“ zeigen die Zeilen und Exit 1 auf beiden Seiten). Keine
Abweichung.

**Die Zeit** von `convert` über diesen Eingang (zwei PDFs und ein Ziel,
nichts Neues) steht in `docs/de/benchmarks.md`, Eintrag vom 2026-09-27:
kalt 556,8 ms, warm im Median 437,1 ms.

**Transkripte** (2026-09-28, durch den Menschen, in Git Bash mit
`bin/loomux.exe convert`, Modell aus). In den Eingang kopiert: die beiden
eingecheckten Transkripte von ultra-brain,
`Transkript_Video1_Second-Brain-Bauanleitung (mHSOsy_usAg).txt` (50 KB) und
`Transkript_Video3_Brain-Maintenance (uI1Z-KJI1Tg).txt` (13 KB), beide in
Klammerform. Der erste Lauf schrieb beide Ziele (`….txt.md`); ihr Kopf:
`source_url: https://www.youtube.com/watch?v=<id>` aus der Id im Namen,
`retrieved: 2026-09-28` (die Kopie setzte das Änderungsdatum), `converter:
brain-transcript/1`, `asr: true`, keine `description`; der Rumpf beginnt mit
`[00:00] …`-Absätzen. Der zweite Lauf schrieb nichts neu.

Git Bash findet `/mingw64/bin/pdftotext` (xpdf 4.06) vor Poppler. Beide
Läufe meldeten darum je PDF `skipped: <name>.pdf: C:\Program
Files\Git\mingw64\bin\pdftotext.exe is not Poppler's pdftotext (pdftotext
version 4.06 [www.xpdfreader.com]); install Poppler with: winget install --id
oschwartz10612.Poppler -e` und wandelten die Transkripte daneben trotzdem —
der Prüfpunkt „Git Bash findet xpdf“ am echten Eingang, wie
`TestXpdfLeavesThePDFsAndTheTranscriptGoes` ihn hält. Die Ziele der PDFs aus
dem Lauf vom 2026-09-27 blieben unberührt.

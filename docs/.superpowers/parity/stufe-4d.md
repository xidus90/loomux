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
Benutzer- und Eigentümerkennwort `geheim`). `encrypted.pdf` trägt eine
Zufallskennung von pypdf; sie ist einmal erzeugt und eingecheckt, ein neuer
Lauf des Skripts ändert sie.

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
4. **`-v` von xpdf endet mit 99** und schreibt auf stdout. Die Erkennung aus
   E6 hält, wenn `-v` unabhängig vom Exit-Code gelesen wird.
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
liest der Nachbau wie wordfreq. Das Skript lag im Scratchpad (`split_check.py`,
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
   Der einzige Schlüssel der Tabelle mit einer Nicht-ASCII-Ziffer ist U+0E51
   (Thai-Eins), eine einzelne Ziffer ohne Folge, die beide ungeglättet
   nachschlagen.

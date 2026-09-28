# Stufe 4d: `convert` und `fetch` — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux convert` wandelt PDFs und Transkripte im Eingang jedes
schreibbaren Bereichs in Markdown mit Herkunftskopf, fragt das lokale Modell
nach einem Satz für den Kopf (`describe`) und einem Zielbereich (`place`),
und `loomux fetch` legt die Untertitel eines Videos über `yt-dlp` in einen
Eingang — beide mit dem Verhalten der Python-Referenz, wo es sie gibt.

**Architecture:** Das neue Paket `internal/brain/convert` hält Erkennung,
Transkript, Kopf, PDF über `pdftotext`, den Lauf über die Eingänge und
`fetch`; externe Programme laufen über eine Naht `convert.Tools`
(`child.Run`, `exec.LookPath`). `internal/brain/model` bekommt die Richter
samt eingebetteter Zipf-Tabelle, die Rollen `describe` und `place` und ein
optionales Ausgabeschema im Client aus 4c-1. `internal/programs` hält die
Installationsbefehle, die `init` und `convert` nennen. `internal/cli` bekommt
die zwei Befehle, `internal/hooks` die Wächterregel; `internal/notices` trägt
`NOTICE.md` mit den Lizenzen aller fremden Teile im Binary, die
`internal/release` neben die Binaries legt.

**Tech Stack:** Go des Moduls, `regexp`, `compress/gzip`, `embed`,
`encoding/json`, `gopkg.in/yaml.v3`, `internal/child`, `internal/lock`,
`internal/brain/pytext`, `internal/cases` (Aufzeichnung und Abspielen),
`internal/dev/faketool`, `internal/dev/fakeollama`; für Orakel und
Generatoren `uv run --script` mit `wordfreq==3.1.1`, `pypdf==6.16.2`,
`pyyaml==6.0.3` (die Versionen aus `ultra-brain/uv.lock` am Tag
`loomux-3-source`); `uvx yt-dlp` für die Messung.

**Spec:** `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`,
Abschnitte „4d im Einzelnen“ und **„Abweichungen beim Planen von 4d“** (der
zweite gilt, wo er dem ersten widerspricht), dazu „Parität“,
„Fehlerverhalten“, „Selbstnutzung“, „Fertig, wenn“ und „Offen und vor dem
Bau zu messen“.

**Referenz:** `ultra-brain` Tag `loomux-3-source` = `3cc72d2` = HEAD (am
2026-09-26 geprüft: kein Unterschied unter `src/brain/convert`,
`src/brain/model`, `src/brain/prompts`, `src/brain/cli.py`, `tests/convert`,
`tests/model`). Gelesen: `src/brain/convert/{detect,transcript,header,pdf,run,fetch}.py`,
`src/brain/model/{local,judge,prompts,proposer,gate}.py`,
`src/brain/registry.py:16-139`, `src/brain/writer.py`, `cli.py:450-560,
1519-1559, 2400-2436`, `tests/convert/*.py` (91 Tests),
`tests/model/test_{judge,local_describe,local_place,prompts}.py`.
Prompts: `ablage-v1.md` 474 Bytes, sha256
`efb2ed013e30e1d38ae3a682d53288c4c48a6b3a236937a9d9d7f83f13ec5148`, trägt
`{scopes}` und `{text}`; `beschreibung-v1.md` 204 Bytes, sha256
`a2f11bf4b41592a0ee798a3ebb0335740aa90b5d3c0a1d7642f3e298ca919125`, trägt
`{text}`; beide nur LF.

## Befunde, gegen den Code gelesen am 2026-09-26

**B1. Die Registry kennt keinen Eingang.** `config.Area`
(`config/registry.go:28-47`) hat kein Feld dafür; die Referenz liest ihn beim
Lesen der Registry aus der Deklaration (`registry._inbox_of`), und zwar für
**jeden** Bereich, sodass eine kaputte Deklaration den ganzen Lauf beendet,
bevor etwas geschrieben ist (`test_a_broken_manifest_stops_the_run_…`). In
loomux liest man die Deklaration über
`config.ReadAreaManifestUntilStage4(config.ResolvedAreaDir(area, state,
fallback))`; eine fehlende ist `config.ErrNoManifest` und heißt „kein
Eingang, Modus `manual_cloud`“ (`run._mode`). Der Eingang liegt unter
`area.Path`, nicht unter dem Artefaktverzeichnis (`registry.py:109`);
`Manifest.InboxLayout()` weist einen absoluten Wert ab.

**B2. Das Modell aus 4c-1 passt.** `config.ReadModelSettings(stateDir)`,
`ModelSettings.Narrowed(m)`, `RoleOn(role)` und `model.ProposerFor(s, m,
role) (*Proposer, error)` (`model/propose.go:32-43`) sind da; `Proposer`
kennt nur `Propose`. `Client.Ask(ctx, prompt)` (`model/client.go:124`)
sendet kein `format`. Die Referenz liest die globale Datei bei **jedem**
`convert`-Durchgang über alle Eingänge, auch ohne Eingang
(`run._proposers`), und baut die Proposer vor der Schleife, sodass ein
Endpoint außerhalb von Loopback den Lauf beendet, bevor eine Datei gewandelt
ist.

**B3. Die Referenz liest Text mit universellen Zeilenenden.**
`path.read_text(encoding="utf-8")` und `path.open(encoding="utf-8")`
übersetzen `\r\n` und `\r` in `\n` (`detect`, `_read_text`);
`write_if_changed` vergleicht dagegen roh (`newline=""`). `pytext.ReadText`
tut das Erste, `lock.ReplaceText` schreibt ohne Übersetzung.

**B4. Sortiert wird wie unter Windows.** `sorted(inbox.iterdir())` ordnet
`Path`-Objekte; Python 3.14 vergleicht unter Windows klein geschrieben.
Gemessen am 2026-09-26 mit dem Python der Referenz:
`['_z.txt', 'a.txt', 'b.pdf', 'B.txt', 'Ä.txt']`. `os.ReadDir` ordnet nach
Bytes.

**B5. Python-Klassen sind Unicode, RE2-Klassen ASCII.** `\w`, `\b`, `\d`,
`\s`, `str.split()`, `str.strip()`, `str.splitlines()` und `len()` (Code
Points) haben in Go andere Gegenstücke; `pytext.IsSpace`, `pytext.Strip`,
`pytext.SplitLines` bilden Pythons Leerraum und Zeilengrenzen nach.

**B6. wordfreq faltet und zerlegt.** `zipf_frequency` bildet
`casefold(NFC(wort))` (`Straße` → `strasse`), glättet Ziffernfolgen
(`\d[\d.,]+` → Nullen) und multipliziert dann mit `digit_freq` (Benford,
Jahreszahlen), rundet die Frequenz auf drei geltende Stellen und die Zipf-Zahl
auf zwei Nachkommastellen (`wordfreq/__init__.py:238-282`,
`numbers.py`). Im Wörterbuch `large_de` stehen 634 502 Schlüssel, davon
3 297 mit Ziffern; ab Zipf 2,5 sind es 85 034 Wörter (285 KB gzip), davon
39 858 ab 3,0. Die Beschränkung auf Wörter bis fünf Zeichen, mit der die
Spec rechnete (46 505 Wörter, 135 KB), trägt nicht: Der Richter zählt die
Länge eines Teils, wie er geschrieben steht, die Tabelle ihren gefalteten
Schlüssel — `Grüße` hat fünf Zeichen, `grüsse` sechs, und das Wort fehlte.

**B7. Ein PyYAML-Nachbau steckt in `apply`** (`apply/pyyaml*.go`), aber
`apply` importiert `maintenance`, und `maintenance` importiert `model`: ein
Import aus `model` wäre ein Zyklus. Die Köpfe der gewandelten Dateien liest
loomux selbst über `yaml.v3` (`index.ParseFrontmatter`) und die Regex von
`DescriptionOf`.

**B8. Die gestellte Welt behält keine Änderungszeit** (`cases.StageWorld`,
`internal/cases/runner.go:66-92`); `retrieved:` ist also in Aufzeichnung und
Abspielen der Tag des Laufs. `RunCaseWith` normalisiert beide Seiten mit
demselben `Normalizer` (`runner.go:206-277`).

**B9. Fälle laufen nur unter Windows** (`.github/workflows/ci.yml`: die
Linux-Spur fährt `vet`, `child`-Tests, Bau und Rauchtest). Aufgezeichnete
Pfade in stdout haben darum Rückstriche.

**B10. `pdftotext -v` spricht auf dem Rechner xpdf.** Git Bash findet
`/mingw64/bin/pdftotext` („pdftotext version 4.06 [www.xpdfreader.com]“,
„Copyright 1996-2025 Glyph & Cog, LLC“). Poppler 25.07.0 ist seit dem
2026-09-26 installiert (`winget install --id oschwartz10612.Poppler -e`),
unter `%LOCALAPPDATA%\Microsoft\WinGet\Packages\oschwartz10612.Poppler_Microsoft.Winget.Source_8wekyb3d8bbwe\poppler-25.07.0\Library\bin\pdftotext.exe`,
das Verzeichnis im `PATH` des Nutzers. `-v` schreibt nach stderr, Exit 0:
„pdftotext version 25.07.0“, „Copyright 2005-2025 The Poppler Developers -
http://poppler.freedesktop.org“, „Copyright 1996-2011, 2022 Glyph & Cog,
LLC“ — die Erkennung aus E6 hält. `yt-dlp` ist nicht installiert
(`uvx 0.12.16` ist da).

**B11. Der Wächter sperrt loomux-Befehle an einer Stelle.**
`wordsWriteConfiguration` (`hooks/guard.go:424-487`) entscheidet je
Unterbefehl; die Begründung steht in `guard.go:268`, die Tabelle der
Beispiele in `TestTheGuardRefusesCommandsThatWriteTheConfiguration`
(`hooks/guard_test.go:364-740`).

**B12. Das Release lädt alles aus `dist/` hoch** (`.github/scripts/release.sh:81-85`);
`release.Build` (`release/build.go:35-66`) schreibt Binaries und
`SHA256SUMS` und gibt die Namen zurück.

## Entscheidungen

Die Fragen der Spec sind entschieden (Abschnitt „Abweichungen beim Planen
von 4d“, 2026-09-26); hier steht, was der Plan daraus macht, und was er
selbst festlegt.

| # | Frage | Umsetzung |
|---|---|---|
| E1 | Wo der Eingang gelesen wird | `convert.Areas(areas, state, fallback)` liest jede Deklaration vorab, gibt je Bereich Deklaration, Eingang und Modus zurück und bricht beim ersten Fehler ab (B1) |
| E2 | Wo die Richter liegen | `internal/brain/model/judge.go` und `zipf.go`; die Tabelle unter `internal/brain/model/zipf/`, eingebettet, erst beim ersten Gebrauch entpackt |
| E3 | Aufbau der Zipf-Tabelle | Drei Abschnitte: `[common]` (Zipf ≥ 3,0), `[mid]` (2,5 bis unter 3,0, jede Länge, B6) und `[digits]` mit der rohen Frequenz der 3 297 Schlüssel mit Ziffernfolge; Go glättet und rechnet `digit_freq` wie wordfreq. Die Spec ist nachgeführt (85 034 Wörter, rund 285 KB gzip); Task 2 trägt die gemessene Gesamtgröße samt `[digits]` ein |
| E4 | Die fünfte Regel von `describe` | `readsBack` prüft mit `yaml.v3`, dem Leser, der den Kopf in loomux liest (B7); eine Batterie zeichnet PyYAMLs Urteil auf, jede Abweichung wird gelesen und in der Akte eingetragen |
| E5 | Ausgabeschema | `Client.AskFormat(ctx, prompt, format)`; `Ask` ruft es mit `nil`, `format` steht mit `omitempty` hinter `options` |
| E6 | Aufruf von `pdftotext` | `pdftotext -layout -enc UTF-8 -eol unix <name> -` mit `Dir` = Verzeichnis der Datei, damit die Befehlszeile ohne Weltpfad auskommt; `-eol unix`, weil Poppler unter Windows sonst CRLF schreiben kann und die Scan-Schwelle und die Naht mit LF rechnen; `pdftotext -v` einmal je Lauf, sobald eine PDF ansteht; Poppler heißt: die Ausgabe von `-v` (stdout und stderr) enthält `Poppler`, gleich welcher Exit-Code — gemessen in Task 1 endet xpdf 4.06 mit 99 und schreibt auf stdout, Poppler 25.07.0 mit 0 auf stderr, beide mit CRLF; xpdf ist also ein falsches Programm, kein fehlendes. Die erste Zeile von `-v` und von stderr geht ohne `\r` in eine Meldung. Frist 120 s je Aufruf |
| E7 | Aufruf von `yt-dlp` | `yt-dlp --ignore-config --no-playlist --no-progress --skip-download --write-subs --write-auto-subs --sub-langs de,en --sub-format json3 --write-info-json --ignore-errors -o v <url>` mit `Dir` = ein frisches Verzeichnis unter `os.TempDir()`, Frist 600 s — Schalter für Schalter und in dieser Reihenfolge die Befehlszeile der Aufnahme (`testdata/convert/ytdlp/mHSOsy_usAg/argv`). Fester Name `v` statt `%(id)s`: Die Id eines fremden Extraktors ist beliebig lang. `--no-playlist`: eine Adresse mit `&list=` holt nur das Video; `--ignore-config`: eine Nutzerkonfiguration ändert weder Namen noch Ort; `--ignore-errors` (Entscheidung des Nutzers, 2026-09-26): ohne ihn beendet eine scheiternde Spur — gemessen 429 auf der automatisch übersetzten `en` — yt-dlp mit Exit 1, bevor die Info-JSON geschrieben ist; mit ihm wird der Fehler eine Warnung, Exit 0, die Info-JSON steht da. Innerhalb einer Sprache nimmt yt-dlp die letzte json3-Spur, die Referenz nahm die erste (freigegeben, Spec-Zeile „fetch: Spur einer Sprache“) |
| E8 | Reihenfolge im Eingang | Unter Windows nach `strings.ToLower(name)`, sonst nach Bytes (B4) |
| E9 | Installationsbefehle | Neues Paket `internal/programs` mit der Liste aus `setup/tools.go`; `setup` und `convert` lesen sie dort |
| E10 | Das Modul Brain | `brainModuleOff(name, stderr)` in `internal/cli`: das Projekt nach oben vom Arbeitsverzeichnis (`hosts.FindRoot`), dessen `[modules] brain = false` beide Befehle mit Exit 1 verweigert; ohne Projekt laufen beide |
| E11 | Der Tag in `retrieved:` | Die Fälle normalisieren `retrieved: JJJJ-MM-TT` auf beiden Seiten; die Regel „Änderungszeit, UTC“ halten Go-Tests mit `os.Chtimes` fest (B8) |

## Global Constraints

- Coverage 100 % je Funktion; ein Ausschluss nur mit `//coverage:exempt <reason>` direkt über `func` (AGENTS.md).
- Kein `init()` und keine Paketvariable, die eingebettete Daten parst oder eine Regex übersetzt; Regexe über `sync.OnceValue`, die Tabelle beim ersten Gebrauch (Fusions-Spec, „Startzeit-Regel“).
- Wandlerkennungen: `brain-transcript/1`, `brain-pdf/2`.
- Schwellen: Transkript 1200 Zeichen, Scan 100 Zeichen je Seite, `describe` 1800 Zeichen und höchstens 22 Wörter, `place` 1800 Zeichen; gezählt in Code Points.
- Prompts `beschreibung-v1` und `ablage-v1` Byte für Byte wie die Referenz (sha256 oben).
- Schema von `place` mit den Schlüsseln und Werten der Referenz: `{"type": "object", "properties": {"scope": {"type": "string"}, "grund": {"type": "string"}}, "required": ["scope", "grund"]}`. Gemeint ist der Inhalt, nicht die Bytes: Reihenfolge der Schlüssel und Leerraum dürfen abweichen (`encoding/json` schreibt Map-Schlüssel sortiert), kein Fall vergleicht Anfragerümpfe, und Ollama liest JSON.
- Modell nur auf Loopback, über den Client aus 4c-1; ein Ausfall heißt „kein Satz“ bzw. „keine Ablage“, nie Verkehr nach außen. loomux selbst spricht nie ins Netz; das Netz ist Sache von `yt-dlp`.
- Meldungen englisch, im Wortlaut der Referenz, wo sie einen hat.
- Code, Kommentare, Meldungen und Commits englisch; Plan, Akte und Spec deutsch. Commit-Nachrichten nach Conventional Commits, ohne Stufe, Plan oder Task im Text, ohne Mitautor.
- Kein Push durch einen Agenten; nach jedem Subagenten `git log -1 --format='%an <%ae>'` lesen.

## Review Focus

1. **Ein Eingang mit `B.txt` und `a.txt`.** Erwartet: `a.txt` zuerst, wie die Referenz unter Windows; stdout und `skipped:` in dieser Reihenfolge. Test in Task 7, dazu der Fall `convert/mixed-case-order` in Task 12.
2. **Ein Transkript mit CRLF** (unter Windows gespeichert). Erwartet: erkannt und gewandelt wie mit LF, ohne `\r` im Ergebnis. Test in Task 7 (`TestACRLFTranscriptConvertsAsLF`; die Erkennung allein in Task 5), Fall `convert/crlf-transcript` in Task 12.
3. **Git Bash findet xpdf, PowerShell Poppler.** Erwartet unter xpdf (dessen `-v` mit 99 endet): je PDF eine Zeile `skipped:` mit dem Installationsbefehl für Poppler, die Zieldatei unangetastet, Exit 1; ein Transkript daneben wird trotzdem gewandelt. Test in Task 7 (`TestXpdfLeavesThePDFsAndTheTranscriptGoes`; die Erkennung allein in Task 6), Exit 1 bei einer `skipped:`-Zeile in Task 8.
4. **`yt-dlp` endet mit Exit 1, nachdem die deutsche Spur geschrieben ist.** Mit `--ignore-errors` (E7) wurde die gemessene 429 der englischen Spur eine Warnung mit Exit 0; ein Fehler, den yt-dlp nach dem Schreiben meldet, endet aber weiter mit 1. Erwartet: die Datei landet im Eingang, Exit 0. Test in Task 9.
5. **Eine PDF namens `Bericht März.pdf`.** Erwartet: `pdftotext` bekommt den Namen relativ, mit dem Eingang als Arbeitsverzeichnis, und die Datei wird gewandelt. Test in Task 6 (Befehlszeile und `Dir`), Messung in Task 1.

---

### Task 1: Messen, bevor gebaut wird

Erledigt am 2026-09-26 in zwei Commits (`8c9c56d4`, dann `b4ac7d3a` nach
den Entscheidungen zu C1 bis C5); die Zahlen stehen in der Akte, Abschnitte
„Messungen“ und „Widersprüche zum Plan“.

**Files:**
- Create: `testdata/convert/pdf/make_pdfs.py` und die PDFs, die es schreibt
- Create: `testdata/convert/ytdlp/mHSOsy_usAg/` (`argv`, `exit`, `stderr`, `files/…`)
- Create: `docs/.superpowers/parity/stufe-4d.md` (Abschnitte „Messungen“, „Widersprüche zum Plan“)
- Modify: `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md` („Vorgeschlagen, mit der Spec freizugeben“ → „Vorgeschlagen und mit der Spec freigegeben (2026-09-26)“; Messliste: `-o v` statt `-o %(id)s`, Grund aus E7, und das Ergebnis mit `--ignore-errors`; Entscheidung „fetch“ mit `--ignore-errors`; neue Zeile „fetch: Spur einer Sprache“)
- Modify: `.gitattributes`

**Interfaces:**
- Produces: die Test-PDFs `text.pdf`, `paragraphs.pdf`, `blank.pdf`, `pageless.pdf`, `mixed.pdf`, `allscan.pdf`, `corrupt.pdf`, `Bericht März.pdf`, `encrypted.pdf` unter `testdata/convert/pdf/`, 5000 pt breit; eine echte Aufnahme von `yt-dlp` mit `--ignore-errors` unter `testdata/convert/ytdlp/mHSOsy_usAg/` (`exit` 0, `files/v.de.json3`, `files/v.info.json`), die Task 9 abspielt.

- [x] **Step 1: Test-PDFs erzeugen**

`testdata/convert/pdf/make_pdfs.py` baut die Fixtures der Referenz
(`tests/convert/test_pdf.py:12-127`) nach, dazu eine Datei mit Umlaut im
Namen und eine verschlüsselte:

```python
# /// script
# requires-python = ">=3.12"
# dependencies = ["pypdf==6.16.2"]
# ///
"""The PDFs stage 4d measures and tests with, built by hand as the reference's tests build them.

Run: uv run --script testdata/convert/pdf/make_pdfs.py
"""

from pathlib import Path

from pypdf import PdfReader, PdfWriter

LONG_TEXT = (
    "Hallo aus dem Pruefbestand. Diese Zeile testet die Extraktion aus einer "
    "von Hand geschriebenen PDF-Datei, ohne Texterkennung und ohne fremde "
    "Erzeugerbibliothek im Testbestand. Der Inhalt bleibt frei erfunden und "
    "dient allein dazu, die Schwelle von hundert Zeichen je Seite sicher zu "
    "ueberschreiten, damit die Datei nicht faelschlich als Scan gilt und der "
    "Test etwas Sinnvolles pruefen kann."
)


def assemble(objects: list[bytes]) -> bytes:
    out = bytearray(b"%PDF-1.4\n")
    offsets: list[int] = []
    for number, body in enumerate(objects, start=1):
        offsets.append(len(out))
        out += f"{number} 0 obj\n".encode() + body + b"\nendobj\n"
    table = len(out)
    out += f"xref\n0 {len(objects) + 1}\n".encode() + b"0000000000 65535 f \n"
    for offset in offsets:
        out += f"{offset:010d} 00000 n \n".encode()
    out += (
        f"trailer\n<< /Size {len(objects) + 1} /Root 1 0 R >>\nstartxref\n{table}\n%%EOF\n"
    ).encode()
    return bytes(out)


def pages(texts: list[str]) -> bytes:
    n = len(texts)
    font = 3 + 2 * n
    kids = " ".join(f"{3 + 2 * i} 0 R" for i in range(n))
    objects: list[bytes] = [
        b"<< /Type /Catalog /Pages 2 0 R >>",
        f"<< /Type /Pages /Kids [{kids}] /Count {n} >>".encode(),
    ]
    for text in texts:
        stream = f"BT /F1 12 Tf 72 720 Td ({text}) Tj ET".encode("latin-1")
        # pdftotext clips at the page edge, pypdf does not: 5000 pt holds the 390/783-char lines for both.
        objects.append(
            f"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 5000 842] "
            f"/Resources << /Font << /F1 {font} 0 R >> >> "
            f"/Contents {len(objects) + 2} 0 R >>".encode()
        )
        objects.append(
            b"<< /Length " + str(len(stream)).encode() + b" >>\nstream\n" + stream + b"\nendstream"
        )
    objects.append(b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
    return assemble(objects)


def main() -> None:
    here = Path(__file__).parent
    files = {
        "text.pdf": pages([LONG_TEXT]),
        "paragraphs.pdf": pages([LONG_TEXT + "\\n\\n\\n" + LONG_TEXT]),
        "blank.pdf": pages([""]),
        "pageless.pdf": assemble(
            [b"<< /Type /Catalog /Pages 2 0 R >>", b"<< /Type /Pages /Kids [] /Count 0 >>"]
        ),
        "mixed.pdf": pages([LONG_TEXT, LONG_TEXT, LONG_TEXT, *([""] * 7)]),
        "allscan.pdf": pages(["", ""]),
        "corrupt.pdf": b"%PDF-1.4\n",
        "Bericht März.pdf": pages([LONG_TEXT]),
    }
    for name, data in files.items():
        (here / name).write_bytes(data)
    writer = PdfWriter(clone_from=PdfReader(here / "text.pdf"))
    # RC4 because pypdf writes it without an extra package; AES needs one.
    writer.encrypt(user_password="geheim", owner_password="geheim", algorithm="RC4-128")
    with (here / "encrypted.pdf").open("wb") as out:
        writer.write(out)


if __name__ == "__main__":
    main()
```

Run: `uv run --script testdata/convert/pdf/make_pdfs.py`
Expected: neun Dateien unter `testdata/convert/pdf/`. `encrypted.pdf` trägt
eine Zufallskennung von pypdf und wird darum einmal erzeugt und eingecheckt,
nicht bei jedem Lauf neu.

Die Seite ist 5000 pt breit, nicht A4 wie in der Referenz: Auf 595 pt gaben
xpdf und Poppler von der Zeile nur 96 Zeichen aus (Step 2), jede Textseite
wäre gegen die Scan-Schwelle 100 ein Scan gewesen. pypdf liest vor und nach
der Verbreiterung dasselbe (`text.pdf` 390 Zeichen, `paragraphs.pdf` 783).

In `.gitattributes` dazu:

```
# The stage-4d fixtures are evidence: PDFs built by hand, a recorded yt-dlp
# run, recordings of the reference and of Poppler. No fold of any kind.
testdata/convert/** -text
testdata/cases/4d-worlds/** -text
testdata/cases/4d-source/** -text
testdata/cases/4d/** -text
internal/brain/model/zipf/** -text
```

- [x] **Step 2: xpdf messen**

Für jede Datei aus Step 1, im Verzeichnis `testdata/convert/pdf` (der Name
relativ, wie E6 es vorsieht):

```bash
cd testdata/convert/pdf && for f in *.pdf; do printf '== %s\n' "$f"; pdftotext -layout -enc UTF-8 -eol unix "$f" - | od -c | tail -3; echo "exit=${PIPESTATUS[0]}"; done
```

und einmal `pdftotext -v; echo "exit=$?"` (stdout und stderr getrennt:
`pdftotext -v 2>/dev/null` und `pdftotext -v 1>/dev/null`).

Dieselbe Schleife läuft danach mit Poppler, über den vollen Pfad aus B10,
weil Git Bash xpdf zuerst findet:
`P="$LOCALAPPDATA/Microsoft/WinGet/Packages/oschwartz10612.Poppler_Microsoft.Winget.Source_8wekyb3d8bbwe/poppler-25.07.0/Library/bin/pdftotext.exe"`,
dann `"$P"` statt `pdftotext`. Fehlt die Datei (eine neuere Poppler-Version
liegt unter einem anderen Verzeichnis), findet `ls "$LOCALAPPDATA"/Microsoft/WinGet/Packages/oschwartz10612.Poppler*/*/Library/bin/pdftotext.exe`
sie. Für beide Builds wird zusätzlich festgehalten, ob die
Ausgabe mit `-eol unix` ein `\r` trägt (gezählt mit `tr -cd '\r' | wc -c`;
`grep -c $'\r'` zählt in Git Bash nicht verlässlich, die Akte sagt warum)
und was `-v` genau sagt; enthält Popplers `-v` das Wort `Poppler` nicht,
wird E6 an das Gemessene angepasst, bevor Task 6 beginnt.

Gemessen: `-v` endet bei xpdf 4.06 mit 99 auf stdout, bei Poppler 25.07.0
mit 0 auf stderr, beide mit CRLF; stdout mit `-eol unix` trägt kein `\r`,
stderr bleibt CRLF; `pageless.pdf` endet unter Poppler mit 99, unter xpdf
mit 0 ohne Ausgabe; `paragraphs.pdf` ist bei beiden Builds eine Zeile.

Expected: je Datei Exit-Code, ob die Ausgabe auf `\f` endet, wie eine leere
Seite aussieht, was `pageless.pdf`, `corrupt.pdf` und `encrypted.pdf`
ergeben und ob `Bericht März.pdf` gelesen wird. Eintragen in
`parity/stufe-4d.md`, Abschnitt „Messungen“, mit der Version aus `-v` und
dem Strom, auf den `-v` schreibt. Weicht etwas von E6 ab (etwa: der Name mit
Umlaut wird nicht gelesen), hält der Plan hier an und fragt den Menschen.

- [x] **Step 3: yt-dlp messen**

```bash
uvx yt-dlp --version
```

In einem kurzen Wegwerfverzeichnis (MAX_PATH):

```bash
D="$LOCALAPPDATA/Temp/ytm" && rm -rf "$D" && mkdir -p "$D" && cd "$D" && uvx yt-dlp --ignore-config --no-playlist --no-progress --skip-download --write-subs --write-auto-subs --sub-langs de,en --sub-format json3 --write-info-json -o v "https://www.youtube.com/watch?v=mHSOsy_usAg" > stdout 2> stderr; echo "exit=$?"; ls -la
```

Expected: `v.info.json` und je Sprache mit Spur eine Datei `v.<lang>.json3`.
Festhalten: Dateinamen, Exit-Code, stderr, welche Sprachen in `subtitles`
und `automatic_captions` stehen und mit welchen `ext`.

Gemessen, zweimal: Exit 1, nur `v.de.json3`, **kein** `v.info.json` — die
automatisch übersetzte Spur `en` antwortete mit 429, und yt-dlp endet, bevor
es die Info-JSON schreibt. Darum derselbe Aufruf mit `--ignore-errors` vor
`-o v` (Entscheidung des Nutzers, 2026-09-26; seitdem Teil von E7):

```bash
D="$LOCALAPPDATA/Temp/ytm" && rm -rf "$D" && mkdir -p "$D" && cd "$D" && uvx yt-dlp --ignore-config --no-playlist --no-progress --skip-download --write-subs --write-auto-subs --sub-langs de,en --sub-format json3 --write-info-json --ignore-errors -o v "https://www.youtube.com/watch?v=mHSOsy_usAg" > stdout 2> stderr; echo "exit=$?"; ls -la
```

Gemessen: Exit 0, `v.de.json3` (Byte für Byte wie ohne den Schalter) und
`v.info.json`, die 429 als `WARNING`. Unter `automatic_captions.de` stehen
zwei json3-Spuren (eine Übersetzung aus `en-US`, dann die deutsche
Spracherkennung); yt-dlp nimmt die letzte, die Referenz nahm die erste — vom
Nutzer als Abweichung freigegeben (Spec, Zeile „fetch: Spur einer Sprache“).

Dann der Rückfall ohne json3:

```bash
cd "$D" && rm -f v.* && uvx yt-dlp --ignore-config --no-playlist --no-progress --skip-download --write-subs --write-auto-subs --sub-langs de --sub-format xyz -o v "https://www.youtube.com/watch?v=mHSOsy_usAg" 2>&1 | tail -5; ls
```

Expected: welche Endung geschrieben wird, wenn das verlangte Format fehlt.
Das belegt, dass loomux die Endung `.json3` prüft, bevor es liest.

Eine 429-Antwort einer Spur lässt sich nicht bestellen. Tritt sie in einem
der Läufe auf, wird sie mit Exit-Code und übrig gebliebenen Dateien
festgehalten; sonst steht in der Akte „nicht provoziert“, und Task 9 prüft
den Fall mit der Attrappe. Sie trat auf (beide Läufe ohne den Schalter).

- [x] **Step 4: Die Aufnahme ablegen**

Unter `testdata/convert/ytdlp/mHSOsy_usAg/`: `argv` (die Befehlszeile mit
`--ignore-errors` aus Step 3 ohne `uvx`), `exit` (`0`), `stderr`, und unter
`files/` die geschriebenen Dateien. `v.info.json` wird gekürzt auf die
Schlüssel, die loomux liest:

```bash
"/c/Users/micro/Documents/#GIT/ultra-brain/.venv/Scripts/python.exe" -c "
import json, sys
info = json.load(open(sys.argv[1], encoding='utf-8'))
keep = {k: info.get(k) for k in ('id', 'title')}
for kind in ('subtitles', 'automatic_captions'):
    keep[kind] = {lang: [{'ext': t.get('ext')} for t in tracks] for lang, tracks in (info.get(kind) or {}).items() if lang in ('de', 'en')}
json.dump(keep, open(sys.argv[2], 'w', encoding='utf-8', newline='\n'), ensure_ascii=False, indent=1)
" "$D/v.info.json" testdata/convert/ytdlp/mHSOsy_usAg/files/v.info.json
```

Die `.json3`-Dateien bleiben ganz. Die Akte sagt, was gekürzt ist und warum:
Die übrigen Schlüssel liest loomux nicht, und die Adressen der Spuren tragen
Signaturen, die nach Stunden verfallen.

- [x] **Step 5: Spec nachführen und committen**

In der Stufe-4-Spec die Überschrift „Vorgeschlagen, mit der Spec
freizugeben“ zu „Vorgeschlagen und mit der Spec freigegeben (2026-09-26)“,
in der Messliste `-o %(id)s` zu `-o v` mit dem Grund aus E7. Nach den
Entscheidungen zu C1 und C5 außerdem: die Entscheidung „fetch“ mit
`--ignore-errors` und seinem Grund, das Ergebnis in der Messliste und die
Zeile „fetch: Spur einer Sprache“.

```bash
git add .gitattributes testdata/convert docs/.superpowers/parity/stufe-4d.md docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md
git commit -m "test(convert): record what pdftotext and yt-dlp do on this machine"
```

Die Korrektur nach den Entscheidungen (breitere PDFs, Aufnahme mit
`--ignore-errors`, Spec-Zeilen) steht im zweiten Commit:

```bash
git commit -m "test(convert): widen the PDF fixtures and record yt-dlp with --ignore-errors"
```

---

### Task 2: Die Zipf-Tabelle

**Files:**
- Create: `internal/brain/model/zipf/generate.py`
- Create: `internal/brain/model/zipf/de.txt.gz` (erzeugt)
- Create: `internal/brain/model/zipf/NOTICE.md`
- Create: `internal/brain/model/zipf/battery.py`, `internal/brain/model/testdata/zipf-battery.json` (erzeugt)
- Create: `internal/brain/model/zipf.go`, `internal/brain/model/zipf_test.go`
- Modify: `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md` (gemessene Größe, E3)

**Interfaces:**
- Produces: `func commonEnough(word string) bool` (Zipf ≥ 3,0), `func fragment(part string) bool` (Zipf < 2,5; nur für Teile bis fünf Zeichen gefragt), `func ZipfNotice() string` (der Text von `zipf/NOTICE.md`).

- [ ] **Step 1: Generator schreiben und laufen lassen**

`internal/brain/model/zipf/generate.py`:

```python
# /// script
# requires-python = ">=3.12"
# dependencies = ["wordfreq==3.1.1"]
# ///
"""Writes de.txt.gz, the German Zipf frequencies chopped_words asks about, and NOTICE.md.

Three sections. [common]: every key without a digit run whose Zipf
frequency is at least 3.0. [mid]: every such key at 2.5 or more and below
3.0, of any length -- the judge counts a part's length as written, and
casefolding makes `Grüße` (five) the key `grüsse` (six). [digits]: every key
with a digit run and its raw frequency, because wordfreq multiplies that by
digit_freq of the token the caller asked about, which no table of Zipf
numbers can hold.

NOTICE.md carries the license and wordfreq's own account of its sources,
copied from the installed package's README so that no word is lost.

Run: uv run --script internal/brain/model/zipf/generate.py
"""

import gzip
from importlib.metadata import distribution
from pathlib import Path

from wordfreq import get_frequency_dict, zipf_frequency
from wordfreq.numbers import MULTI_DIGIT_RE

HEAD = """# Third-party notice: German word frequencies

loomux embeds `internal/brain/model/zipf/de.txt.gz`, a table derived from the
German word frequencies of wordfreq 3.1.1 by Robyn Speer
(<https://github.com/rspeer/wordfreq>). It holds the Zipf frequency class of
every word the chopped-word judge asks about, and the raw frequency of the
keys that carry digits.

The table is an adaptation of wordfreq's data and is licensed under the
Creative Commons Attribution-ShareAlike 4.0 International license
(<https://creativecommons.org/licenses/by-sa/4.0/>), as the data it derives
from. It is not covered by loomux's own license. `generate.py` in the same
directory rebuilds it.

What follows is quoted from wordfreq's README, version 3.1.1.

"""


def section(readme: str, title: str) -> str:
    """One `## ` section of the README, heading included, up to the next one."""
    start = readme.index(f"\n## {title}\n") + 1
    end = readme.find("\n## ", start + 1)
    return readme[start:] if end == -1 else readme[start : end + 1]


def notice() -> str:
    metadata = distribution("wordfreq").read_text("METADATA") or ""
    readme = metadata.split("\n\n", 1)[1]
    return HEAD + section(readme, "License") + "\n" + section(readme, "Citations to work that wordfreq is built on")


def main() -> None:
    here = Path(__file__).parent
    here.joinpath("NOTICE.md").write_text(notice(), encoding="utf-8", newline="\n")
    common: list[str] = []
    mid: list[str] = []
    digits: list[str] = []
    for key, freq in get_frequency_dict("de", wordlist="best").items():
        if MULTI_DIGIT_RE.search(key):
            digits.append(f"{key}\t{freq!r}")
            continue
        zipf = zipf_frequency(key, "de")
        if zipf >= 3.0:
            common.append(key)
        elif zipf >= 2.5:
            mid.append(key)
    lines = [
        "# wordfreq 3.1.1, de, wordlist best; written by generate.py, do not edit",
        "[common]",
        *sorted(common),
        "[mid]",
        *sorted(mid),
        "[digits]",
        *sorted(digits),
    ]
    data = ("\n".join(lines) + "\n").encode("utf-8")
    here.joinpath("de.txt.gz").write_bytes(gzip.compress(data, compresslevel=9, mtime=0))


if __name__ == "__main__":
    main()
```

Run: `uv run --script internal/brain/model/zipf/generate.py`
Expected: `de.txt.gz` um 300 KB (gemessen 2026-09-26: 85 034 Wörter ab Zipf
2,5, 285 KB gzip, dazu `[digits]`) und `NOTICE.md`. Zweimal laufen lassen:
beide Male dieselbe sha256 beider Dateien (`mtime=0`). Die Größe, die Zahl
der Zeilen je Abschnitt und die sha256 der **entpackten** Tabelle notieren.

- [ ] **Step 2: Den Lizenzhinweis lesen**

`NOTICE.md` trägt nach dem Kopf aus `HEAD` die Abschnitte „License“ und
„Citations to work that wordfreq is built on“ aus dem README von wordfreq
3.1.1, wörtlich: Google Books Ngrams samt Nutzungsbedingungen, Leeds
Internet Corpus, Wikipedia, ParaCrawl, OPUS OpenSubtitles 2018, die
SUBTLEX-Listen mit den Bedingungen von Marc Brysbaert („Wordfreq and code
derived from it must credit the SUBTLEX authors“ — die Zitate, darunter
SUBTLEX-DE, Brysbaert et al. 2011, stehen im zweiten Abschnitt) und die
Twitter-Daten. Lesen, ob beide Abschnitte ganz da sind; fehlt etwas, stimmt
`section` nicht mit dem README überein.

- [ ] **Step 3: Batterie der Referenz aufzeichnen**

`internal/brain/model/zipf/battery.py`:

```python
# /// script
# requires-python = ">=3.12"
# dependencies = ["wordfreq==3.1.1"]
# ///
"""Records wordfreq's answer for words a Go port could read differently.

Run: uv run --script internal/brain/model/zipf/battery.py
"""

import json
from pathlib import Path

from wordfreq import get_frequency_dict, tokenize, zipf_frequency
from wordfreq.numbers import digit_freq

DIGIT_RUNS = ["20", "100", "007", "1990", "2026", "2045", "3,5", "1.000", "12345"]

HAND = [
    # the judge's own cases (tests/model/test_judge.py)
    "Projekt", "fortzusetzen", "Fertistellung", "Fertigkriterium", "Ferti", "jekt",
    "Mess", "Know", "EMail", "Teilsystem", "Suchkette", "Pro", "Stellung", "fort",
    "zu", "set", "zen", "Mail", "qmd", "Profil", "Python", "nach", "Go", "Migration",
    "how", "Fertig", "Kriterium", "Ende", "Zustand", "Verschlüsselung", "Scheiben", "Wiki",
    # case folding, umlauts, sharp s (a short part that casefolds longer), NFC against NFD
    "Straße", "STRASSE", "Äpfel", "ÄRGER", "über", "Über", "naïve", "naïve", "ẞ",
    "Maß", "Fuß", "Grüße", "Buße", "Soße", "ẞoße", "Füße",
    # digits: single, runs, years, decimals, mixed with letters
    "3", "7", "G4", "x86", "2026", "1990", "1990er", "100", "3,5", "1.000", "H2O", "Win11", "0815",
    # underscores, ligatures and other word characters
    "E_Mail", "_", "a_b", "ſ", "ǅ", "Ⅻ", "ﬁnden",
]


def main() -> None:
    table = get_frequency_dict("de", wordlist="best")
    near = sorted(
        key for key in table if 2.4 <= zipf_frequency(key, "de") <= 3.1 and 4 <= len(key) <= 6
    )[::200]
    words = HAND + near + [word.capitalize() for word in near]
    rows = [
        {"word": word, "zipf": zipf_frequency(word, "de"), "tokens": tokenize(word, "de")}
        for word in words
    ]
    battery = {"words": rows, "digit_freq": {run: digit_freq(run) for run in DIGIT_RUNS}}
    out = Path(__file__).parents[1] / "testdata" / "zipf-battery.json"
    out.write_text(json.dumps(battery, ensure_ascii=False, indent=1) + "\n", encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()
```

Run: `uv run --script internal/brain/model/zipf/battery.py`
Expected: `testdata/zipf-battery.json` mit einigen hundert Wörtern und neun
Werten von `digit_freq`. Jedes Wort, dessen `tokens` mehr als ein Element
hat, wird notiert: Für es gilt die Annahme „ein Wort, ein Token“ nicht, und
es kommt in `multiToken` (Step 4).

- [ ] **Step 4: Den failing test schreiben**

`internal/brain/model/zipf_test.go`:

```go
package model

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

// zipfSum is the sha256 of the unpacked table generate.py wrote; a changed
// table is a changed judge, and the battery below would not know.
const zipfSum = "<sha256 der entpackten Datei aus Step 1>"

func TestTheZipfTableIsTheGeneratedOne(t *testing.T) {
	reader, err := gzip.NewReader(bytes.NewReader(zipfData))
	if err != nil {
		t.Fatal(err)
	}
	text, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(text)
	if hex.EncodeToString(sum[:]) != zipfSum {
		t.Fatal("de.txt.gz is not the table generate.py writes")
	}
}

type battery struct {
	Words []struct {
		Word   string   `json:"word"`
		Zipf   float64  `json:"zipf"`
		Tokens []string `json:"tokens"`
	} `json:"words"`
	DigitFreq map[string]float64 `json:"digit_freq"`
}

func readBattery(t *testing.T) battery {
	t.Helper()
	data, err := os.ReadFile("testdata/zipf-battery.json")
	if err != nil {
		t.Fatal(err)
	}
	var b battery
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	return b
}

// TestTheTableAnswersWhatWordfreqAnswers holds both questions the judge asks
// against wordfreq's own number for every word of the battery. A word
// wordfreq reads as several tokens must be listed in multiToken: the table
// answers for one token only.
func TestTheTableAnswersWhatWordfreqAnswers(t *testing.T) {
	for _, row := range readBattery(t).Words {
		if len(row.Tokens) > 1 {
			if !multiToken[row.Word] {
				t.Errorf("wordfreq reads %q as %q; list it in multiToken", row.Word, row.Tokens)
			}
			continue
		}
		if got, want := commonEnough(row.Word), row.Zipf >= 3.0; got != want {
			t.Errorf("commonEnough(%q) = %v, wordfreq says %.2f", row.Word, got, row.Zipf)
		}
		if utf8.RuneCountInString(row.Word) <= partMax {
			if got, want := fragment(row.Word), row.Zipf < 2.5; got != want {
				t.Errorf("fragment(%q) = %v, wordfreq says %.2f", row.Word, got, row.Zipf)
			}
		}
	}
}

// multiToken are the battery's words wordfreq reads as several tokens. The
// judge never hands it one: a part is a run of word characters. Filled from
// the battery in step 3; each entry is a line of the Akte.
var multiToken = map[string]bool{}

// TestDigitFreqIsWordfreqs holds the Benford and year estimate against
// wordfreq's own values. math.Pow and C's pow may part in the last bit, so
// the comparison allows one part in 1e15.
func TestDigitFreqIsWordfreqs(t *testing.T) {
	for run, want := range readBattery(t).DigitFreq {
		if got := digitFreq(run); math.Abs(got-want) > want*1e-15 {
			t.Errorf("digitFreq(%q) = %v, wordfreq says %v", run, got, want)
		}
	}
}

func TestZipfNoticeNamesTheLicenseAndTheSource(t *testing.T) {
	notice := ZipfNotice()
	for _, want := range []string{"CC", "4.0", "wordfreq", "Google Books Ngrams"} {
		if !strings.Contains(notice, want) {
			t.Errorf("the notice does not name %q", want)
		}
	}
}

func TestParseZipfRefusesWhatGenerateNeverWrites(t *testing.T) {
	for _, text := range []string{
		"word before any section\n",
		"[digits]\nno-tab\n",
		"[digits]\n00\tnot-a-number\n",
		"[unknown]\nx\n",
		// longer than bufio.Scanner's 64 KiB: the scanner stops with an error
		"[common]\n" + strings.Repeat("x", 70000) + "\n",
	} {
		if _, err := parseZipf([]byte(text)); err == nil {
			t.Errorf("parseZipf(%q) took it", text)
		}
	}
}

func TestUnpackRefusesWhatIsNoGzip(t *testing.T) {
	if _, err := unpackZipf([]byte("plain")); err == nil {
		t.Fatal("plain bytes unpacked")
	}
}

func TestPyRoundRoundsLikePython(t *testing.T) {
	for _, c := range []struct {
		x    float64
		n    int
		want float64
	}{
		{2.675, 2, 2.67}, // the binary value lies below the half
		{0.125, 2, 0.12}, // an exact half goes to even
		{0.375, 2, 0.38},
		{1.23456e-7, 9, 1.23e-7},
	} {
		if got := pyRound(c.x, c.n); got != c.want {
			t.Errorf("pyRound(%v, %d) = %v, want %v", c.x, c.n, got, c.want)
		}
	}
}

func TestSmashNumbersZeroesRunsAndLeavesSingleDigits(t *testing.T) {
	for in, want := range map[string]string{"2026": "0000", "g4": "g4", "3,5": "0,0", "x86": "x00", "a1b": "a1b"} {
		if got := smashNumbers(in); got != want {
			t.Errorf("smashNumbers(%q) = %q, want %q", in, got, want)
		}
	}
}
```

Dazu `"math"` in den Imports.

Run: `go test ./internal/brain/model/ -run 'Zipf|PyRound|DigitFreq|Unpack|Smash' -count=1`
Expected: FAIL, `zipfData`, `commonEnough`, `fragment`, `partMax`,
`ZipfNotice`, `parseZipf`, `unpackZipf`, `pyRound`, `digitFreq` und
`smashNumbers` sind nicht definiert.

- [ ] **Step 5: Implementieren**

`internal/brain/model/zipf.go`:

```go
package model

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// zipfData is generate.py's table, packed. It is unpacked on first use, never
// at start: the hook path must not pay for a judge it never calls.
//
//go:embed zipf/de.txt.gz
var zipfData []byte

//go:embed zipf/NOTICE.md
var zipfNotice string

// ZipfNotice is the license notice of the embedded table; NOTICE.md, which
// the release ships beside the binaries, quotes it.
func ZipfNotice() string { return zipfNotice }

// partMax is the longest part the fragment signal looks at (judge.py:56).
const partMax = 5

type zipfTable struct {
	common map[string]bool    // Zipf >= 3.0, no digit run
	mid    map[string]bool    // 2.5 <= Zipf < 3.0, no digit run
	digits map[string]float64 // keys with a digit run, raw frequency
}

var loadedZipf = sync.OnceValue(func() *zipfTable {
	return mustZipf(zipfData)
})

// mustZipf unpacks and reads the embedded table.
//
//coverage:exempt the panic arm needs a broken embedded table, which TestTheZipfTableIsTheGeneratedOne rules out at build time
func mustZipf(data []byte) *zipfTable {
	text, err := unpackZipf(data)
	if err != nil {
		panic(err)
	}
	table, err := parseZipf(text)
	if err != nil {
		panic(err)
	}
	return table
}

func unpackZipf(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}

func parseZipf(text []byte) (*zipfTable, error) {
	table := &zipfTable{common: map[string]bool{}, mid: map[string]bool{}, digits: map[string]float64{}}
	section := ""
	lines := bufio.NewScanner(bytes.NewReader(text))
	for lines.Scan() {
		line := lines.Text()
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case line == "[common]" || line == "[mid]" || line == "[digits]":
			section = line
		case section == "[common]":
			table.common[line] = true
		case section == "[mid]":
			table.mid[line] = true
		case section == "[digits]":
			key, raw, ok := strings.Cut(line, "\t")
			freq, err := strconv.ParseFloat(raw, 64)
			if !ok || err != nil {
				return nil, fmt.Errorf("zipf table: bad digits line %q", line)
			}
			table.digits[key] = freq
		default:
			return nil, fmt.Errorf("zipf table: %q stands outside a section", line)
		}
	}
	// A line past the scanner's limit ends the loop like the end of the
	// text; without this check the table would be cut short silently.
	if err := lines.Err(); err != nil {
		return nil, fmt.Errorf("zipf table: %w", err)
	}
	return table, nil
}

// band is what the table knows about one token: 2 at Zipf 3.0 or more, 1 at
// 2.5 or more, 0 below.
func band(word string) int {
	token := pytext.CaseFold(pytext.NFC(word))
	table := loadedZipf()
	smashed := smashNumbers(token)
	if raw, ok := table.digits[smashed]; ok {
		freq := raw
		if smashed != token {
			freq *= digitFreq(token)
		}
		return bandOf(zipfOf(freq))
	}
	switch {
	case smashed != token:
		return 0
	case table.common[token]:
		return 2
	case table.mid[token]:
		return 1
	}
	return 0
}

func bandOf(zipf float64) int {
	switch {
	case zipf >= 3.0:
		return 2
	case zipf >= 2.5:
		return 1
	}
	return 0
}

// commonEnough says whether a word pushed together is a real word
// (judge.py:45-48).
func commonEnough(word string) bool { return band(word) == 2 }

// fragment says whether a short part is rarer than a word part would be
// (judge.py:49-51). The judge asks it for parts of at most partMax
// characters, counted as written.
func fragment(part string) bool { return band(part) == 0 }

// log10 is Python's math.log(x, 10), which divides two natural logarithms
// and can differ from math.Log10 in the last bit.
func log10(x float64) float64 { return math.Log(x) / math.Log(10) }

// zipfOf is word_frequency's rounding and freq_to_zipf's, as
// zipf_frequency(word, lang, min_zipf=0) calls them.
func zipfOf(freq float64) float64 {
	freq = math.Max(freq, 1e-9)
	lead := math.Floor(-log10(freq))
	freq = pyRound(freq, int(lead)+3)
	return pyRound(log10(freq)+9, 2)
}

// pyRound is Python's round(x, n) for n >= 0: the exact binary value rounded
// to n decimals, an exact half to even.
func pyRound(x float64, n int) float64 {
	rounded, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', n, 64), 64)
	return rounded
}

// The constants of wordfreq/numbers.py.
var digitFreqs = [10]float64{0.009, 0.300, 0.175, 0.124, 0.096, 0.078, 0.066, 0.057, 0.050, 0.045}

const (
	yearLogPeak   = -1.9185
	notYearProb   = 0.1
	referenceYear = 2019
	plateauWidth  = 20
)

func pow10(x float64) float64 { return math.Pow(10, x) }

// Digits are ASCII here, where wordfreq's `\d` takes every script's: a digit
// of another script in a German hyphenated word is a row of the Akte, not a
// case the judge meets.
var multiDigit = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`[0-9][0-9.,]+`) })
var pureDigit = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`[0-9]+`) })

// smashNumbers is wordfreq's: every digit of a run of two or more
// characters becomes 0.
func smashNumbers(text string) string {
	return multiDigit().ReplaceAllStringFunc(text, func(run string) string {
		return strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return '0'
			}
			return r
		}, run)
	})
}

// digitFreq is wordfreq's digit_freq.
func digitFreq(text string) float64 {
	freq := 1.0
	for _, run := range multiDigit().FindAllString(text, -1) {
		for _, digits := range pureDigit().FindAllString(run, -1) {
			if len(digits) == 4 {
				freq *= yearFreq(digits)
			} else {
				freq *= benfordFreq(digits)
			}
		}
	}
	return freq
}

func benfordFreq(digits string) float64 {
	return digitFreqs[digits[0]-'0'] / pow10(float64(len(digits)-1))
}

func yearFreq(digits string) float64 {
	year, _ := strconv.Atoi(digits)
	var logFreq float64
	switch {
	case year <= referenceYear:
		logFreq = yearLogPeak - 0.0083*float64(referenceYear-year)
	case year <= referenceYear+plateauWidth:
		logFreq = yearLogPeak
	default:
		logFreq = yearLogPeak - 0.2*float64(year-(referenceYear+plateauWidth))
	}
	return pow10(logFreq) + notYearProb*benfordFreq(digits)
}
```

Weicht ein Wort der Batterie ab, wird die Ursache gesucht, nicht das Wort in
`multiToken` verschoben: Dort stehen nur Wörter mit mehr als einem Token.
Liegt die Ursache in der Normalform (zeigt `ﬁnden`, dass wordfreq für
Deutsch NFKC statt NFC nimmt), wird `band` umgestellt und die Batterie
bestätigt es.

Run: `go test ./internal/brain/model/ -run 'Zipf|PyRound|DigitFreq|Unpack|Smash' -count=1`
Expected: PASS.

- [ ] **Step 6: Coverage und Commit**

Run: `go test ./internal/brain/model/ -coverprofile=cover.out -count=1 && go tool cover -func=cover.out | grep -v 100.0%`
Expected: nur `mustZipf` (ausgenommen) und nichts sonst unter 100 %.

In der Stufe-4-Spec, Entscheidung „Zipf-Tabelle“: die gemessene Größe
samt Abschnitt `[digits]` nachtragen (ein Satz mit Verweis auf E3 dieses
Plans).

```bash
git add internal/brain/model/zipf internal/brain/model/zipf.go internal/brain/model/zipf_test.go internal/brain/model/testdata/zipf-battery.json docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md
git commit -m "feat(model): embed the German word frequencies the chopped-word judge reads"
```

---

### Task 3: Die Richter

**Files:**
- Create: `internal/brain/model/judge.go`, `internal/brain/model/judge_test.go`
- Create: `docs/.superpowers/parity/stufe-4d-orakel/judge_battery.py`, `internal/brain/model/testdata/judge-battery.json` (erzeugt)

**Interfaces:**
- Consumes: `commonEnough`, `fragment`, `partMax` (Task 2)
- Produces: `func FunctionWords() []string`, `func IsGerman(text string) bool`, `func ChoppedWords(text string) []string`, `func IsOneSentence(text string) bool`, `func WordCount(text string) int`, `func readsBack(sentence string) bool`

- [ ] **Step 1: Batterie der Referenz aufzeichnen**

`docs/.superpowers/parity/stufe-4d-orakel/judge_battery.py` läuft mit dem
Python der Referenz, weil es deren Richter selbst fragt:

```python
"""Records the reference judges' verdicts for sentences a Go port could read differently.

Run with the reference's interpreter, on the tag loomux-3-source:
  "C:/Users/micro/Documents/#GIT/ultra-brain/.venv/Scripts/python.exe" \
    docs/.superpowers/parity/stufe-4d-orakel/judge_battery.py
"""

import json
from pathlib import Path

from brain.model.judge import chopped_words, is_german, is_one_sentence, word_count
from brain.model.local import _reads_back

SENTENCES = [
    # tests/model/test_judge.py
    "Der Bau ist in acht Scheiben zerlegt.",
    "Die Scheiben definieren Fertigkriterien für verschiedene Bauabschnitte.",
    "Die Scheiben definieren Fertigkriterien fuer verschiedene Bauabschnitte.",
    "Die Scheiben erfordern durch Prüfkriterien die Ferti-Stellung um das Pro-jekt fort-zu-set-zen",
    "The build is split into eight slices.",
    "De bouw is in acht plakken verdeeld.",
    "Data in tables",
    *(f"Ein Satz mit {w} darin." for w in [
        "Pro-jekt", "fort-zu-set-zen", "Ferti-Stellung", "E-Mail", "qmd-Profil",
        "Python-nach-Go-Migration", "Know-how", "Fertig-Kriterium", "Ende-Zustand",
        "Ende-zu-Ende-Verschlüsselung",
    ]),
    "Das Teilsystem-Scheiben steht bereit.",
    "Das Wiki-Suchkette steht bereit.",
    "Ein Satz.", "Ein Satz. Noch einer.", "Ein Satz. Und Text danach", "Kein Ende", "   ",
    "Für größere Bereiche gilt das auch.",
    # tests/model/test_local_describe.py
    "Der Bericht beschreibt die Abnahme der zweiten Scheibe.",
    'Der Bericht nennt die Regel "Aus schlägt An" und ihre Grenzen.',
    "[Der Bericht] beschreibt die Abnahme der zweiten Scheibe.",
    "Der Bericht: die Abnahme der zweiten Scheibe.",
    "Der Bericht ist in #1 der Reihe.",
    "Der Bericht beschreibt\ndie Abnahme der Scheibe.",
    # umlauts and sharp s at the edges of hyphenated words
    "Die Über-Prüfung ist für das Pro-jekt nötig.",
    "Der Ärger-Faktor ist bei der Straßen-Planung hoch.",
    "Die E-Mail-Adresse ist in der Liste.",
    "Die Python-3-Migration ist in der Stufe G4-Plan.",
    "Das Win-11-Update ist für die x86-Rechner.",
    "Die nai\u0308ve-Idee ist in der Mappe.",
    "Das _-Zeichen ist in der Regel ohne Be-deu-tung.",
    # what the head of a file makes of a sentence
    "Ja.", "yes.", "on.", "Der Wert ist 1.", "- Die Liste ist hier.", "* Der Stern ist hier.",
    "'Das Zitat' steht am Anfang.", '"Das Zitat" steht am Anfang.', "Der Wert ist %x.",
    "Der Bericht & der Anhang sind da.", "! Der Ausruf ist da.", "? Die Frage ist da.",
    "Der Bericht ist @heute da.", "{Die Klammer} ist im Satz.", "Der Bericht | die Pipe.",
    "> Das Zitat ist da.", "Der Tabulator\tist im Satz.", "Der Satz endet mit Doppelpunkt:",
    "Der Satz hat: einen Doppelpunkt.", "Der Satz hat:einen Doppelpunkt.", "Der Satz hat ein # im Text.",
    "Der Satz hat ein#im Text.", "Der Satz ist ~ da.", "Der Satz ist null.", "Der Satz ist 2026-09-26.",
    "&Anker ist im Satz.", "!Tag ist im Satz.", "Der Satz trägt ein … am Ende.",
]


def main() -> None:
    rows = [
        {
            "text": text,
            "is_german": is_german(text),
            "chopped": list(chopped_words(text)),
            "one_sentence": is_one_sentence(text),
            "word_count": word_count(text),
            "reads_back": _reads_back(text),
        }
        for text in SENTENCES
    ]
    out = Path(__file__).parents[4] / "internal" / "brain" / "model" / "testdata" / "judge-battery.json"
    out.write_text(json.dumps(rows, ensure_ascii=False, indent=1) + "\n", encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()
```

Run: `"C:/Users/micro/Documents/#GIT/ultra-brain/.venv/Scripts/python.exe" docs/.superpowers/parity/stufe-4d-orakel/judge_battery.py`
Expected: `internal/brain/model/testdata/judge-battery.json` mit einer Zeile
je Satz. Vorher `git -C "C:/Users/micro/Documents/#GIT/ultra-brain" rev-parse HEAD`
gegen `3cc72d2` lesen.

- [ ] **Step 2: Den failing test schreiben**

`internal/brain/model/judge_test.go`:

```go
package model

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

const glm = "Die Scheiben erfordern durch Prüfkriterien die Ferti-Stellung um das Pro-jekt fort-zu-set-zen"

func TestGermanIsMeasuredAtFunctionWordsNeverAtUmlauts(t *testing.T) {
	for _, text := range []string{
		"Der Bau ist in acht Scheiben zerlegt.",
		"Die Scheiben definieren Fertigkriterien für verschiedene Bauabschnitte.",
		"Die Scheiben definieren Fertigkriterien fuer verschiedene Bauabschnitte.",
		glm,
	} {
		if !IsGerman(text) {
			t.Errorf("IsGerman(%q) = false", text)
		}
	}
	for _, text := range []string{"The build is split into eight slices.", "De bouw is in acht plakken verdeeld.", "Data in tables"} {
		if IsGerman(text) {
			t.Errorf("IsGerman(%q) = true", text)
		}
	}
}

func TestAChoppedWordIsFound(t *testing.T) {
	for _, word := range []string{"Pro-jekt", "fort-zu-set-zen", "Ferti-Stellung"} {
		if got := ChoppedWords("Ein Satz mit " + word + " darin."); !slices.Equal(got, []string{word}) {
			t.Errorf("ChoppedWords(%q) = %q", word, got)
		}
	}
	if got := ChoppedWords(glm); !slices.Equal(got, []string{"Ferti-Stellung", "Pro-jekt", "fort-zu-set-zen"}) {
		t.Errorf("the glm sentence: %q", got)
	}
}

func TestAnOrdinaryHyphenatedWordIsNotChopped(t *testing.T) {
	for _, word := range []string{"E-Mail", "qmd-Profil", "Python-nach-Go-Migration", "Know-how", "Fertig-Kriterium", "Ende-Zustand", "Ende-zu-Ende-Verschlüsselung"} {
		if got := ChoppedWords("Ein Satz mit " + word + " darin."); len(got) != 0 {
			t.Errorf("ChoppedWords(%q) = %q", word, got)
		}
	}
	for _, word := range []string{"Teilsystem-Scheiben", "Wiki-Suchkette"} {
		if got := ChoppedWords("Das " + word + " steht bereit."); len(got) != 0 {
			t.Errorf("a rare compound %q counted as chopped: %q", word, got)
		}
	}
}

func TestOneSentenceEndsOnceAndAtTheEnd(t *testing.T) {
	for text, want := range map[string]bool{
		"Ein Satz.": true, "Ein Satz. Noch einer.": false, "Ein Satz. Und Text danach": false, "Kein Ende": false, "   ": false,
	} {
		if got := IsOneSentence(text); got != want {
			t.Errorf("IsOneSentence(%q) = %v", text, got)
		}
	}
}

func TestWordCountCountsAWordWithUmlautsAsOne(t *testing.T) {
	if n := WordCount("Für größere Bereiche gilt das auch."); n != 6 {
		t.Errorf("got %d", n)
	}
	if n := WordCount("Der Bau ist in acht Scheiben zerlegt."); n != 7 {
		t.Errorf("got %d", n)
	}
}

func TestTheFunctionWordsAreTheReferencesSeventyTwo(t *testing.T) {
	words := FunctionWords()
	if len(words) != 72 || !slices.Contains(words, "für") || !slices.Contains(words, "fuer") || !slices.Contains(words, "mittels") {
		t.Fatalf("%d words: %q", len(words), words)
	}
	words[0] = "changed"
	if FunctionWords()[0] == "changed" {
		t.Fatal("FunctionWords hands out its own list")
	}
}

// readsBackDeviations are the battery's sentences whose verdict yaml.v3 and
// PyYAML reach differently; each is a row of the Akte. Filled in step 4.
var readsBackDeviations = map[string]string{}

func TestTheJudgesAgreeWithTheReference(t *testing.T) {
	data, err := os.ReadFile("testdata/judge-battery.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Text        string   `json:"text"`
		IsGerman    bool     `json:"is_german"`
		Chopped     []string `json:"chopped"`
		OneSentence bool     `json:"one_sentence"`
		WordCount   int      `json:"word_count"`
		ReadsBack   bool     `json:"reads_back"`
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if got := IsGerman(row.Text); got != row.IsGerman {
			t.Errorf("IsGerman(%q) = %v", row.Text, got)
		}
		if got := ChoppedWords(row.Text); !slices.Equal(got, row.Chopped) && !(len(got) == 0 && len(row.Chopped) == 0) {
			t.Errorf("ChoppedWords(%q) = %q, reference %q", row.Text, got, row.Chopped)
		}
		if got := IsOneSentence(row.Text); got != row.OneSentence {
			t.Errorf("IsOneSentence(%q) = %v", row.Text, got)
		}
		if got := WordCount(row.Text); got != row.WordCount {
			t.Errorf("WordCount(%q) = %d, reference %d", row.Text, got, row.WordCount)
		}
		_, deviates := readsBackDeviations[row.Text]
		if got := readsBack(row.Text); (got != row.ReadsBack) != deviates {
			t.Errorf("readsBack(%q) = %v, PyYAML %v, listed as deviation: %v", row.Text, got, row.ReadsBack, deviates)
		}
	}
}
```

Run: `go test ./internal/brain/model/ -run 'German|Chopped|Sentence|WordCount|FunctionWords|Judges' -count=1`
Expected: FAIL, die Richter sind nicht definiert.

- [ ] **Step 3: Implementieren**

`internal/brain/model/judge.go`:

```go
package model

import (
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// functionWords is the reference's closed list (judge.py:34-40): common
// German function words, every one with an umlaut twice, with it and
// transcribed. It tells German from English; it identifies no language.
var functionWords = []string{
	"der", "die", "das", "den", "dem", "des", "ein", "eine", "einer", "einem", "einen", "und", "oder", "aber", "nicht",
	"ist", "sind", "war", "waren", "wird", "werden", "wurde", "hat", "haben", "in", "im", "an", "auf", "aus", "bei", "mit",
	"von", "zu", "zur", "zum", "fuer", "für", "ueber", "über", "unter", "durch", "gegen", "ohne", "nach", "vor", "seit",
	"sich", "als", "wenn", "weil", "dass", "es", "sie", "er", "wir", "man", "jede", "jeder", "jedes", "kein", "keine", "nur",
	"schon", "noch", "auch", "dann", "damit", "deshalb", "wobei", "welche", "welcher", "mittels",
}

// FunctionWords is a copy of the list, for the search bench as well.
func FunctionWords() []string { return slices.Clone(functionWords) }

// wordRun is Python's `[\wÄÖÜäöüß]+`: RE2's \w is ASCII, so the letters,
// digits and the underscore of every script are spelled out.
var wordRun = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`[\p{L}\p{N}_]+`) })

// hyphenated is `\b[\wÄÖÜäöüß]+(?:-[\wÄÖÜäöüß]+)+\b` without the
// boundaries: a leftmost, greedy run of word characters starts and ends
// where Python's \b stands.
var hyphenated = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`[\p{L}\p{N}_]+(?:-[\p{L}\p{N}_]+)+`)
})

// IsGerman needs two different function words; the umlaut never counts.
// `in`, `die` and `man` occur in English and Dutch sentences too, two
// different ones practically never.
func IsGerman(text string) bool {
	seen := map[string]bool{}
	for _, w := range wordRun().FindAllString(text, -1) {
		seen[strings.ToLower(w)] = true
	}
	hits := 0
	for _, w := range functionWords {
		if seen[w] {
			hits++
		}
	}
	return hits >= 2
}

// ChoppedWords are hyphenated words a model assembled from fragments, by two
// signals (judge.py:69-117): the parts push together into a common word
// while no part but the first is capitalised, or a capitalised word holds a
// short part rarer than a word part would be.
func ChoppedWords(text string) []string {
	var hits []string
	for _, w := range hyphenated().FindAllString(text, -1) {
		parts := strings.Split(w, "-")
		onlyFirstCapital := !slices.ContainsFunc(parts[1:], startsUpper)
		cutApart := onlyFirstCapital && commonEnough(strings.Join(parts, ""))
		fragmented := startsUpper(w) && slices.ContainsFunc(parts, func(part string) bool {
			return utf8.RuneCountInString(part) <= partMax && fragment(part)
		})
		if cutApart || fragmented {
			hits = append(hits, w)
		}
	}
	return hits
}

// startsUpper is Python's `s[:1].isupper()`.
func startsUpper(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

// IsOneSentence is exactly one sentence end, standing at the very end.
func IsOneSentence(text string) bool {
	tightened := pytext.Strip(text)
	if tightened == "" {
		return false
	}
	return strings.Count(tightened, ".")+strings.Count(tightened, "!")+strings.Count(tightened, "?") == 1 &&
		strings.ContainsAny(tightened[len(tightened)-1:], ".!?")
}

// WordCount is the number of word runs.
func WordCount(text string) int { return len(wordRun().FindAllString(text, -1)) }

// readsBack says whether the sentence comes back unchanged out of a file
// head, read by yaml.v3 as loomux reads every head (`index.ParseFrontmatter`).
// The reference asks PyYAML; the battery names where the two part.
func readsBack(sentence string) bool {
	var loaded map[string]any
	if err := yaml.Unmarshal([]byte("description: "+sentence+"\n"), &loaded); err != nil {
		return false
	}
	value, ok := loaded["description"].(string)
	return ok && len(loaded) == 1 && value == sentence
}
```

Run: `go test ./internal/brain/model/ -run 'German|Chopped|Sentence|WordCount|FunctionWords|Judges' -count=1`
Expected: PASS bis auf Zeilen der Batterie.

- [ ] **Step 4: Jede Abweichung der Batterie lesen**

Jede rote Zeile von `TestTheJudgesAgreeWithTheReference` bekommt eine
Ursache. Bei `IsGerman`, `ChoppedWords`, `IsOneSentence` und `WordCount` ist
eine Abweichung ein Fehler im Port und wird im Code behoben (etwa eine
Zeichenklasse, die ein Kombinationszeichen anders zählt). Bei `readsBack`
ist sie ein Unterschied zwischen PyYAML und yaml.v3: Der Satz kommt mit
Begründung in `readsBackDeviations` und als Zeile in die Akte
(`parity/stufe-4d.md`, Abschnitt „Abweichungen“). Ein Satz, den yaml.v3
**annimmt** und PyYAML **nicht**, wird dabei gegen `index.ParseFrontmatter`
geprüft: Kommt er als Wert von `description:` unverändert zurück, ist die
Abweichung hier harmlos; sonst wird `readsBack` enger. Die Probe gegen
`DescriptionOf` folgt in Task 5, wo es entsteht.

Run: `go test ./internal/brain/model/ -count=1`
Expected: PASS.

- [ ] **Step 5: Coverage und Commit**

Run: `go test ./internal/brain/model/ -coverprofile=cover.out -count=1 && go tool cover -func=cover.out | grep -v 100.0%`
Expected: nur `mustZipf`.

```bash
git add internal/brain/model/judge.go internal/brain/model/judge_test.go internal/brain/model/testdata/judge-battery.json docs/.superpowers/parity/stufe-4d-orakel/judge_battery.py
git commit -m "feat(model): judge a local sentence by function words, chopped words and its read-back"
```

---

### Task 4: Die Rollen `describe` und `place`

**Files:**
- Create: `internal/brain/model/prompts/beschreibung-v1.md`, `internal/brain/model/prompts/ablage-v1.md`
- Create: `internal/brain/model/describe.go`, `internal/brain/model/describe_test.go`
- Create: `internal/brain/model/place.go`, `internal/brain/model/place_test.go`
- Modify: `internal/brain/model/client.go:103-148` (`format`), `internal/brain/model/client_test.go` (ein Test)
- Modify: `internal/brain/pytext/text.go`, `internal/brain/pytext/text_test.go` (`FirstRunes`)

**Interfaces:**
- Consumes: `IsGerman`, `ChoppedWords`, `IsOneSentence`, `WordCount`, `readsBack` (Task 3); `ProposerFor`, `Client` (4c-1)
- Produces: `const DescribeVersion = "beschreibung-v1"`, `const PlaceVersion = "ablage-v1"`, `func (p *Proposer) Describe(ctx context.Context, text string) (string, bool)`, `func (p *Proposer) Place(ctx context.Context, text string, scopes []string) (string, bool)`, `func (c *Client) AskFormat(ctx context.Context, prompt string, format any) (string, bool)`, `func pytext.FirstRunes(s string, n int) string` (Pythons `s[:n]`; `describe` und `place` rufen es hier, `ToInbox` in Task 9)

- [ ] **Step 1: Prompts übernehmen**

```bash
git -C "C:/Users/micro/Documents/#GIT/ultra-brain" show loomux-3-source:src/brain/prompts/beschreibung-v1.md > internal/brain/model/prompts/beschreibung-v1.md
git -C "C:/Users/micro/Documents/#GIT/ultra-brain" show loomux-3-source:src/brain/prompts/ablage-v1.md > internal/brain/model/prompts/ablage-v1.md
sha256sum internal/brain/model/prompts/beschreibung-v1.md internal/brain/model/prompts/ablage-v1.md
```

Expected: `a2f11bf4…9125` und `efb2ed01…5148` wie oben. `.gitattributes`
hält `internal/brain/model/prompts/** -text` schon (4c-1).

- [ ] **Step 2: Den failing test schreiben**

In `internal/brain/pytext/text_test.go` dazu:

```go
func TestFirstRunesIsPythonsSliceOfCharacters(t *testing.T) {
	for _, c := range []struct {
		s    string
		n    int
		want string
	}{
		{"äöü", 2, "äö"},
		{"ab", 5, "ab"},
		{"ab", 0, ""},
		{"", 3, ""},
	} {
		if got := FirstRunes(c.s, c.n); got != c.want {
			t.Errorf("FirstRunes(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
	}
}
```

In `client_test.go` dazu:

```go
func TestAskFormatSendsTheSchemaAndAskSendsNone(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		bodies = append(bodies, got)
		_ = json.NewEncoder(w).Encode(map[string]string{"response": "x"})
	}))
	defer server.Close()
	client, err := NewClient(settingsFor(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	client.Ask(context.Background(), "p")
	client.AskFormat(context.Background(), "p", map[string]any{"type": "object"})
	if _, present := bodies[0]["format"]; present {
		t.Fatal("Ask sent a format")
	}
	if format, _ := bodies[1]["format"].(map[string]any); format["type"] != "object" {
		t.Fatalf("AskFormat sent %v", bodies[1]["format"])
	}
}
```

`describe_test.go`:

```go
package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

const goodSentence = "Der Bericht beschreibt die Abnahme der zweiten Scheibe."

// allRoles is answering with every role on, and every request body kept.
func allRoles(t *testing.T, answer string, bodies *[]map[string]any) *Proposer {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		if bodies != nil {
			*bodies = append(*bodies, got)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"response": answer})
	}))
	t.Cleanup(server.Close)
	settings := config.ModelSettings{Enabled: true, Endpoint: server.URL, Name: "m",
		Roles: map[string]bool{"propose": true, "place": true, "describe": true}}
	p, err := ProposerFor(settings, &config.Manifest{}, "describe")
	if err != nil || p == nil {
		t.Fatal(p, err)
	}
	return p
}

func describe(t *testing.T, answer, text string) (string, bool) {
	t.Helper()
	return allRoles(t, answer, nil).Describe(context.Background(), text)
}

func TestTheDescribePromptIsTheReferencesByteForByte(t *testing.T) {
	sum := sha256.Sum256([]byte(describePrompt))
	if hex.EncodeToString(sum[:]) != "a2f11bf4b41592a0ee798a3ebb0335740aa90b5d3c0a1d7642f3e298ca919125" {
		t.Fatal("beschreibung-v1.md is not the reference's")
	}
	if strings.Count(describePrompt, "{") != 1 || !strings.Contains(describePrompt, "{text}") {
		t.Fatal("the prompt carries braces str.format would read")
	}
}

func TestAGoodSentenceComesBackTightened(t *testing.T) {
	for _, answer := range []string{goodSentence, "  " + goodSentence + "\n\n"} {
		if got, ok := describe(t, answer, "langer text"); !ok || got != goodSentence {
			t.Errorf("%q: %q %v", answer, got, ok)
		}
	}
}

func TestDescribeRefusesWhatBreaksARule(t *testing.T) {
	for _, answer := range []string{
		"Ein Satz. Und noch einer dazu.",
		"Der Bericht " + strings.Repeat("und der Anhang ", 10) + "sind da.",
		"The report describes the second slice.",
		"Das Pro-jekt ist in acht Scheiben zerlegt.",
		"Der Bericht: die Abnahme der zweiten Scheibe.",
		"Der Bericht beschreibt\ndie Abnahme der Scheibe.",
		"Der Bericht ist in #1 der Reihe.",
		"[Der Bericht] beschreibt die Abnahme der zweiten Scheibe.",
	} {
		if got, ok := describe(t, answer, "t"); ok {
			t.Errorf("%q passed as %q", answer, got)
		}
	}
}

func TestAQuoteInsideTheSentenceStands(t *testing.T) {
	answer := `Der Bericht nennt die Regel "Aus schlägt An" und ihre Grenzen.`
	if got, ok := describe(t, answer, "t"); !ok || got != answer {
		t.Fatalf("%q %v", got, ok)
	}
}

func TestOnlyTheMeasuredHeadReachesTheDescribePrompt(t *testing.T) {
	var bodies []map[string]any
	allRoles(t, goodSentence, &bodies).Describe(context.Background(), strings.Repeat("Ä", 1799)+"B"+strings.Repeat("C", 500))
	prompt := bodies[0]["prompt"].(string)
	if !strings.Contains(prompt, strings.Repeat("Ä", 1799)+"B") || strings.Contains(prompt, "C") {
		t.Fatal("the cut is not the first 1800 characters")
	}
	if _, present := bodies[0]["format"]; present {
		t.Fatal("describe sent a schema")
	}
}

func TestDescribeWithoutAnAnswerIsNone(t *testing.T) {
	settings := config.ModelSettings{Enabled: true, Endpoint: "http://127.0.0.1:1", Name: "m", Roles: map[string]bool{"describe": true}}
	p, _ := ProposerFor(settings, &config.Manifest{}, "describe")
	if got, ok := p.Describe(context.Background(), "t"); ok {
		t.Fatalf("%q", got)
	}
}
```

Die Umlaute im Schnitt-Test sind Absicht: 1800 **Zeichen**, nicht Bytes.

`place_test.go`:

```go
package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

const placeAnswer = `{"scope": "project/ultra-brain", "grund": "Es geht um den Indexer."}`

func place(t *testing.T, answer string, scopes ...string) (string, bool) {
	t.Helper()
	return allRoles(t, answer, nil).Place(context.Background(), "text", scopes)
}

func TestThePlacePromptIsTheReferencesByteForByte(t *testing.T) {
	sum := sha256.Sum256([]byte(placePrompt))
	if hex.EncodeToString(sum[:]) != "efb2ed013e30e1d38ae3a682d53288c4c48a6b3a236937a9d9d7f83f13ec5148" {
		t.Fatal("ablage-v1.md is not the reference's")
	}
	if strings.Count(placePrompt, "{") != 2 || !strings.Contains(placePrompt, "{scopes}") || !strings.Contains(placePrompt, "{text}") {
		t.Fatal("the prompt carries braces str.format would read")
	}
}

func TestAScopeFromTheRegisterComesBack(t *testing.T) {
	if got, ok := place(t, placeAnswer, "project/ultra-brain", "space"); !ok || got != "project/ultra-brain" {
		t.Fatalf("%q %v", got, ok)
	}
}

func TestPlaceDropsWhatIsNoKnownScope(t *testing.T) {
	for _, answer := range []string{
		`{"scope": "project/erfunden", "grund": "..."}`,
		"project/ultra-brain",
		`{"grund": "weiss nicht"}`,
		`["project/ultra-brain"]`,
		`{"scope": 5, "grund": "x"}`,
	} {
		if got, ok := place(t, answer, "project/ultra-brain"); ok {
			t.Errorf("%q passed as %q", answer, got)
		}
	}
}

func TestTheSchemaGoesOutAsTheFormat(t *testing.T) {
	var bodies []map[string]any
	allRoles(t, placeAnswer, &bodies).Place(context.Background(), "text", []string{"project/ultra-brain"})
	want := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scope": map[string]any{"type": "string"},
			"grund": map[string]any{"type": "string"},
		},
		"required": []any{"scope", "grund"},
	}
	if !reflect.DeepEqual(bodies[0]["format"], want) {
		t.Fatalf("format %v", bodies[0]["format"])
	}
}

func TestTheScopesAndTheHeadReachThePlacePrompt(t *testing.T) {
	var bodies []map[string]any
	allRoles(t, placeAnswer, &bodies).Place(context.Background(), strings.Repeat("A", 1799)+"B"+strings.Repeat("C", 500), []string{"project/ultra-brain", "space"})
	prompt := bodies[0]["prompt"].(string)
	if !strings.Contains(prompt, "\n- project/ultra-brain\n- space\n") {
		t.Fatal("the scopes are not in the rig's list form")
	}
	if !strings.Contains(prompt, strings.Repeat("A", 1799)+"B") || strings.Contains(prompt, "C") {
		t.Fatal("the cut is not the first 1800 characters")
	}
}

// A text that names a placeholder is not read as one: str.format fills each
// field once, and so does the replacer.
func TestATextNamingAPlaceholderStaysText(t *testing.T) {
	var bodies []map[string]any
	allRoles(t, placeAnswer, &bodies).Place(context.Background(), "über {scopes} und {text}", []string{"a"})
	if !strings.Contains(bodies[0]["prompt"].(string), "über {scopes} und {text}") {
		t.Fatal("the text was filled in again")
	}
}

func TestPlaceWithoutAnAnswerIsNone(t *testing.T) {
	settings := config.ModelSettings{Enabled: true, Endpoint: "http://127.0.0.1:1", Name: "m", Roles: map[string]bool{"place": true}}
	p, _ := ProposerFor(settings, &config.Manifest{}, "place")
	if got, ok := p.Place(context.Background(), "t", []string{"a"}); ok {
		t.Fatalf("%q", got)
	}
}
```

Run: `go test ./internal/brain/model/ ./internal/brain/pytext/ -run 'Describe|Place|Scope|Schema|Sentence|Quote|AskFormat|Placeholder|FirstRunes' -count=1`
Expected: FAIL, `describePrompt`, `placePrompt`, `Describe`, `Place`,
`AskFormat`, `FirstRunes` fehlen.

- [ ] **Step 3: Implementieren**

In `internal/brain/pytext/text.go`, hinter `RStrip`:

```go
// FirstRunes is `s[:n]` for n >= 0: the first n characters, not bytes.
func FirstRunes(s string, n int) string {
	for i := range s {
		if n == 0 {
			return s[:i]
		}
		n--
	}
	return s
}
```

Eine Funktion für alle Stellen, die Pythons Zeichenschnitt nachbauen:
`describe` und `place` hier, `ToInbox` in Task 9.

In `client.go` bekommt `generateRequest` ein Feld und `Ask` einen Bruder:

```go
type generateRequest struct {
	Model   string          `json:"model"`
	Prompt  string          `json:"prompt"`
	Stream  bool            `json:"stream"`
	Think   bool            `json:"think"`
	Options generateOptions `json:"options"`
	// Format is the output schema the endpoint enforces (`place` only); it
	// stands behind options, where the reference's payload appends it.
	Format any `json:"format,omitempty"`
}
```

`Ask` wird zu `AskFormat(ctx, prompt, nil)`, und der Rumpf zieht nach
`AskFormat`, das `Format: format` setzt. Der Kommentar „A struct of strings,
bools and numbers always marshals“ wird zu „A struct of strings, bools,
numbers and a schema of maps and slices always marshals“.

```go
// Ask is AskFormat without a schema.
func (c *Client) Ask(ctx context.Context, prompt string) (string, bool) {
	return c.AskFormat(ctx, prompt, nil)
}
```

`internal/brain/model/describe.go`:

```go
package model

import (
	"context"
	_ "embed"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// DescribeVersion names the prompt that asked for a file head's sentence.
const DescribeVersion = "beschreibung-v1"

//go:embed prompts/beschreibung-v1.md
var describePrompt string

// The cut and the length the rig measured beschreibung-v1 on (local.py:24-36).
const (
	describeMaxChars = 1800
	describeMaxWords = 22
)

// Describe is the one sentence for a file head, or none, and the head keeps
// four lines. Five rules, each enough to refuse; there is no fallback to the
// document's first line, which would stand in the head as if measured.
func (p *Proposer) Describe(ctx context.Context, text string) (string, bool) {
	answer, ok := p.client.Ask(ctx, strings.ReplaceAll(describePrompt, "{text}", pytext.FirstRunes(text, describeMaxChars)))
	if !ok {
		return "", false
	}
	sentence := pytext.Strip(answer)
	if !IsOneSentence(sentence) || WordCount(sentence) > describeMaxWords || !IsGerman(sentence) ||
		len(ChoppedWords(sentence)) > 0 || !readsBack(sentence) {
		return "", false
	}
	return sentence, true
}
```

`internal/brain/model/place.go`:

```go
package model

import (
	"context"
	_ "embed"
	"encoding/json"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// PlaceVersion names the prompt that asked for a file's target area.
const PlaceVersion = "ablage-v1"

//go:embed prompts/ablage-v1.md
var placePrompt string

// placeMaxChars is the head the rig measured ablage-v1 on (local.py:48-51).
const placeMaxChars = 1800

// placeSchema is the rig's _ABLAGE_SCHEMA; the endpoint enforces it, the
// prompt only names it.
func placeSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scope": map[string]any{"type": "string"},
			"grund": map[string]any{"type": "string"},
		},
		"required": []string{"scope", "grund"},
	}
}

// Place is one scope from scopes, or none, and the file stays in its inbox.
// A well-formed scope the register does not know is no answer.
func (p *Proposer) Place(ctx context.Context, text string, scopes []string) (string, bool) {
	listed := make([]string, len(scopes))
	for i, scope := range scopes {
		listed[i] = "- " + scope
	}
	prompt := strings.NewReplacer("{scopes}", strings.Join(listed, "\n"), "{text}", pytext.FirstRunes(text, placeMaxChars)).Replace(placePrompt)
	answer, ok := p.client.AskFormat(ctx, prompt, placeSchema())
	if !ok {
		return "", false
	}
	var read any
	if err := json.Unmarshal([]byte(answer), &read); err != nil {
		return "", false
	}
	object, _ := read.(map[string]any)
	scope, _ := object["scope"].(string)
	if !slices.Contains(scopes, scope) {
		return "", false
	}
	return scope, true
}
```

Ein leerer `scope` ist nie in der Liste, darum braucht `slices.Contains`
keinen eigenen Zweig dafür.

Run: `go test ./internal/brain/model/ ./internal/brain/pytext/ -count=1`
Expected: PASS.

- [ ] **Step 4: Coverage und Commit**

Run: `go test ./internal/brain/model/ ./internal/brain/pytext/ -coverprofile=cover.out -count=1 && go tool cover -func=cover.out | grep -v 100.0%`
Expected: nur `mustZipf`.

```bash
git add internal/brain/model internal/brain/pytext
git commit -m "feat(model): ask the local model for a file head's sentence and a target area"
```

---

### Task 5: Erkennen, Absätze fügen, der Kopf

**Files:**
- Create: `internal/brain/convert/detect.go`, `detect_test.go`
- Create: `internal/brain/convert/transcript.go`, `transcript_test.go`
- Create: `internal/brain/convert/header.go`, `header_test.go`
- Modify, nur wenn ein Satz aus `readsBackDeviations` es verlangt (Step 1): `internal/brain/model/judge.go`, `judge_test.go`

**Interfaces:**
- Produces: `type Format int` mit `Unsupported`, `TranscriptBracket`, `TranscriptRange`, `PDF`; `func Detect(path string) Format`; `func pySuffix(name string) string`; `func ToParagraphs(text string, f Format) string`; `const TranscriptConverter = "brain-transcript/1"`, `const PDFConverter = "brain-pdf/2"`; `type Head struct{ SourceURL string; Retrieved time.Time; Converter string; ASR bool; Description string }` mit `func (h Head) String() string`; `func SourceURLFrom(name string) string`; `func ConvertedBy(text string) (string, bool)`; `func DescriptionOf(text string) (string, bool)`. Pythons `text[:n]` ist `pytext.FirstRunes` aus Task 4; dieses Paket hat keine eigene Kopie.

- [ ] **Step 1: Den failing test schreiben**

`internal/brain/convert/detect_test.go`:

```go
package convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func file(t *testing.T, name string, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDetectReadsTheEndingAndThenTheHead(t *testing.T) {
	for _, c := range []struct {
		name, data string
		want       Format
	}{
		{"video.txt", "[00:00] Hallo zusammen.\n\n[00:36] Und weiter.\n", TranscriptBracket},
		{"export.txt", "00:00:00 - 00:00:57\nHallo zusammen.\n", TranscriptRange},
		{"notiz.txt", "Nur Text, keine Zeitmarken.\n", Unsupported},
		{"buch.pdf", "%PDF-1.7\n", PDF},
		{"BUCH.PDF", "irrelevant", PDF},
		{"video.md", "[00:00] Hallo zusammen.\n", Unsupported},
		{"gross.txt", strings.Repeat("Prosa.\n", 20000) + "[00:00] spät.\n", Unsupported},
		{"kaputt.txt", "\xff\xfe\x00\x00rubbish", Unsupported},
		// Universal newlines, as Python's text mode reads: CRLF and a lone
		// CR both end a line.
		{"crlf.txt", "00:00:00 - 00:00:57\r\nHallo.\r\n", TranscriptRange},
		{"cr.txt", "Titel\r[00:00] Hallo.\r", TranscriptBracket},
		// Path.suffix of a dotfile is empty.
		{".txt", "[00:00] Hallo.\n", Unsupported},
	} {
		if got := Detect(file(t, c.name, c.data)); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDetectCallsWhatItCannotReadUnsupported(t *testing.T) {
	if got := Detect(filepath.Join(t.TempDir(), "gone.txt")); got != Unsupported {
		t.Fatal(got)
	}
	// A directory opens but does not read.
	dir := filepath.Join(t.TempDir(), "ordner.txt")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Detect(dir); got != Unsupported {
		t.Fatal(got)
	}
}

// Broken bytes past the sample pass detection; the full read fails later.
func TestDetectLooksAtTheHeadOnly(t *testing.T) {
	data := "[00:00] Anfang ist sauber.\n" + strings.Repeat("x", 45000) + "\n" + strings.Repeat("\xff\xfe", 50)
	if got := Detect(file(t, "spaet.txt", data)); got != TranscriptBracket {
		t.Fatal(got)
	}
}

func TestPySuffixIsPathSuffix(t *testing.T) {
	for name, want := range map[string]string{"a.txt": ".txt", ".txt": "", "a.": "", "a": "", "a.b.PDF": ".PDF", "..x": ".x"} {
		if got := pySuffix(name); got != want {
			t.Errorf("pySuffix(%q) = %q, want %q", name, got, want)
		}
	}
}
```

`internal/brain/convert/transcript_test.go`:

```go
package convert

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func bracket(count, chars int) string {
	var parts []string
	for i := range count {
		parts = append(parts, fmt.Sprintf("[%02d:%02d] ", i/60, i%60)+strings.Repeat("wort ", chars/5))
	}
	return strings.Join(parts, "\n\n")
}

func TestTheThresholdDecidesWhereAParagraphEnds(t *testing.T) {
	if got := toParagraphs(bracket(3, 100), TranscriptBracket, 1200); strings.Contains(got, "\n\n") {
		t.Fatal("three short fragments made more than one paragraph")
	}
	got := toParagraphs(bracket(20, 100), TranscriptBracket, 1200)
	paragraphs := strings.Split(got, "\n\n")
	if len(paragraphs) != 2 {
		t.Fatalf("%d paragraphs", len(paragraphs))
	}
	for _, p := range paragraphs {
		if strings.Count(p, "[") != 1 || !strings.HasPrefix(p, "[") {
			t.Fatalf("a paragraph keeps more than the first mark: %.40q", p)
		}
	}
}

func TestNotOneWordIsLostOrAdded(t *testing.T) {
	text := "[00:00] Hallo zusammen, das hier ist mein\n\n[00:04] Second Brain mit 2000 Notizen.\n"
	words := func(s string) []string {
		return slices.DeleteFunc(strings.Fields(s), func(w string) bool { return strings.HasPrefix(w, "[") })
	}
	if got := ToParagraphs(text, TranscriptBracket); !slices.Equal(words(got), words(text)) {
		t.Fatalf("%q", got)
	}
}

func TestMarksBecomeMinutes(t *testing.T) {
	for _, c := range []struct {
		text string
		f    Format
		want string
	}{
		{"00:01:05 - 00:01:57\nHallo zusammen.\n\n00:02:10 - 00:02:44\nUnd weiter.\n", TranscriptRange, "[01:05] "},
		{"01:02:03 - 01:02:44\nSpät im Video.\n", TranscriptRange, "[62:03] "},
		{"[02:00:00] Zwei Stunden rein.\n", TranscriptBracket, "[120:00] "},
	} {
		if got := ToParagraphs(c.text, c.f); !strings.HasPrefix(got, c.want) {
			t.Errorf("%q: %q", c.text, got)
		}
	}
	if got := ToParagraphs("00:01:05 - 00:01:57\nHallo.\n", TranscriptRange); strings.Contains(got, "00:01:05") {
		t.Fatal("the range line survived")
	}
}

func TestTheLeadInIsKeptAsItsOwnParagraph(t *testing.T) {
	got := strings.Split(ToParagraphs("Titel des Videos\n\n[00:00] Hallo zusammen.\n", TranscriptBracket), "\n\n")
	if len(got) != 2 || got[0] != "Titel des Videos" || !strings.HasPrefix(got[1], "[00:00]") {
		t.Fatalf("%q", got)
	}
}

func TestEmptyPiecesFallAway(t *testing.T) {
	if got := ToParagraphs("", TranscriptBracket); got != "" {
		t.Fatalf("%q", got)
	}
	if got := ToParagraphs("[00:00]\n[00:01] Hallo.\n", TranscriptBracket); got != "[00:01] Hallo." {
		t.Fatalf("%q", got)
	}
}

// Python's str.split breaks on \x1c..\x1f; strings.Fields does not.
func TestWhitespaceIsPythons(t *testing.T) {
	if got := ToParagraphs("[00:00] a\x1fb c\n", TranscriptBracket); got != "[00:00] a b c" {
		t.Fatalf("%q", got)
	}
}

// The threshold counts characters, not bytes.
func TestTheThresholdCountsCharacters(t *testing.T) {
	text := "[00:00] " + strings.Repeat("ä", 700) + "\n[00:01] " + strings.Repeat("ö", 400) + "\n[00:02] x\n"
	if got := strings.Count(ToParagraphs(text, TranscriptBracket), "\n\n"); got != 0 {
		t.Fatalf("1103 characters were cut into %d+1 paragraphs", got)
	}
}
```

`internal/brain/convert/header_test.go`:

```go
package convert

import (
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/index"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestSourceURLFromTheName(t *testing.T) {
	for name, want := range map[string]string{
		"Transkript_Video1_Second-Brain-Bauanleitung (mHSOsy_usAg).txt": "https://www.youtube.com/watch?v=mHSOsy_usAg",
		"NoteGPT_Transcript_RAG, Hybrid-Suche oder Wiki.txt":           "",
		"Notiz (Entwurf).txt":                                           "",
		"Notiz (Draft-Notes).txt":                                       "https://www.youtube.com/watch?v=Draft-Notes",
	} {
		if got := SourceURLFrom(name); got != want {
			t.Errorf("%q: %q", name, got)
		}
	}
}

func TestTheHeadHasFourLinesAndAFifthForADescription(t *testing.T) {
	four := Head{SourceURL: "https://www.youtube.com/watch?v=mHSOsy_usAg", Retrieved: day(2026, 8, 24), Converter: TranscriptConverter, ASR: true}.String()
	want := "---\nsource_url: https://www.youtube.com/watch?v=mHSOsy_usAg\nretrieved: 2026-08-24\nconverter: brain-transcript/1\nasr: true\n---\n\n"
	if four != want {
		t.Fatalf("%q", four)
	}
	empty := Head{Retrieved: day(2026, 8, 24), Converter: PDFConverter}.String()
	if !strings.Contains(empty, "source_url:\n") || !strings.Contains(empty, "asr: false\n") || strings.Contains(empty, "description") {
		t.Fatalf("%q", empty)
	}
	five := Head{Retrieved: day(2026, 9, 7), Converter: "pdf", Description: "Der Bericht beschreibt die Abnahme."}.String()
	if !strings.Contains(five, "asr: false\ndescription: Der Bericht beschreibt die Abnahme.\n---\n\n") {
		t.Fatalf("%q", five)
	}
}

func TestConvertedByKnowsOurOwnFileOnly(t *testing.T) {
	ours := "---\nsource_url:\nretrieved: 2026-08-24\nconverter: brain-pdf/1\nasr: false\n---\n\nText.\n"
	if got, ok := ConvertedBy(ours); !ok || got != "brain-pdf/1" {
		t.Fatalf("%q %v", got, ok)
	}
	for _, text := range []string{
		"---\ntitle: Meine Notiz\n---\n\nText.\n",
		"Einfach Text.\n",
		"---\nconverter: brain-pdf/1\nasr: false\n",
	} {
		if _, ok := ConvertedBy(text); ok {
			t.Errorf("%q counted as ours", text)
		}
	}
}

func TestDescriptionOfReadsTheHeadOnly(t *testing.T) {
	head := Head{Retrieved: day(2026, 9, 7), Converter: "pdf", Description: "Der Bericht beschreibt die Abnahme."}.String()
	if got, ok := DescriptionOf(head); !ok || got != "Der Bericht beschreibt die Abnahme." {
		t.Fatalf("%q %v", got, ok)
	}
	for _, text := range []string{
		Head{Retrieved: day(2026, 9, 7), Converter: "pdf"}.String(),
		"Einfach Text.\n",
		"---\nsource_url:\nretrieved: 2026-09-07\nconverter: brain-pdf/1\nasr: false\n---\n\ndescription: Das ist ein Satz aus dem Text.\n",
	} {
		if _, ok := DescriptionOf(text); ok {
			t.Errorf("%q yielded a description", text)
		}
	}
}

// What describe accepts comes out of the head unchanged, for both readers
// loomux has: yaml.v3 through index.ParseFrontmatter, and DescriptionOf.
func TestAnAcceptedSentenceComesBackOutOfTheHead(t *testing.T) {
	for _, sentence := range []string{
		"Der Bericht beschreibt die Abnahme der zweiten Scheibe.",
		`Der Bericht nennt die Regel "Aus schlägt An" und ihre Grenzen.`,
	} {
		text := Head{Retrieved: day(2026, 9, 7), Converter: PDFConverter, Description: sentence}.String()
		meta, err := index.ParseFrontmatter(text, true)
		if err != nil || meta["description"] != sentence {
			t.Errorf("yaml.v3 reads %v (%v)", meta["description"], err)
		}
		if got, _ := DescriptionOf(text); got != sentence {
			t.Errorf("DescriptionOf reads %q", got)
		}
	}
}
```

In die Liste von `TestAnAcceptedSentenceComesBackOutOfTheHead` kommt
außerdem jeder Satz aus `readsBackDeviations` (Task 3), den yaml.v3
**annimmt** und PyYAML **nicht**; der Implementierer kopiert ihn aus
`judge_test.go` (die Variable ist eine Testvariable des Pakets `model` und
von hier nicht erreichbar). Kommt er aus `DescriptionOf` nicht unverändert
zurück, ist die Abweichung nicht harmlos, und `readsBack` in
`internal/brain/model/judge.go` wird enger, mit einem Test dort, im Commit
dieses Tasks.

Run: `go test ./internal/brain/convert/ -count=1`
Expected: FAIL, das Paket hat keine Quelldatei.

- [ ] **Step 2: Implementieren**

`internal/brain/convert/detect.go`:

```go
// Package convert turns what lands in an area's inbox into Markdown with a
// provenance head: transcripts joined into paragraphs, PDFs through
// Poppler's pdftotext, and a video's subtitles fetched through yt-dlp. It
// follows src/brain/convert/ of the reference.
package convert

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"
)

// Format is what a file in an inbox carries.
type Format int

const (
	Unsupported Format = iota
	TranscriptBracket
	TranscriptRange
	PDF
)

// headChars is how far detection reads: a timestamp that only shows up
// after that does not make the file a transcript (detect.py:17-20).
const headChars = 8192

// Both marks stand at the start of a line (detect.py:11-15). Digits are
// ASCII, where Python's \d takes every script's; a transcript in other
// digits is a row of the Akte.
var bracketMark = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?m)^\[(\d{2}:\d{2}(?::\d{2})?)\]`)
})

var rangeMark = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?m)^(\d{2}:\d{2}:\d{2}) - \d{2}:\d{2}:\d{2}\s*$`)
})

// Detect reads the ending first and the head of a .txt second: the
// transcripts on hand all end in .txt and still carry two kinds of mark.
// Unsupported is an answer, not an error.
func Detect(path string) Format {
	switch strings.ToLower(pySuffix(filepath.Base(path))) {
	case ".pdf":
		return PDF
	case ".txt":
	default:
		return Unsupported
	}
	head, ok := readHead(path)
	switch {
	case !ok:
		return Unsupported
	case rangeMark().MatchString(head):
		return TranscriptRange
	case bracketMark().MatchString(head):
		return TranscriptBracket
	}
	return Unsupported
}

// pySuffix is Path.suffix: from the last dot, unless the dot starts or ends
// the name.
func pySuffix(name string) string {
	i := strings.LastIndex(name, ".")
	if i <= 0 || i == len(name)-1 {
		return ""
	}
	return name[i:]
}

// readHead is the first headChars characters in Python's text mode:
// universal newlines, and a byte that is not UTF-8 among them fails the
// read. Four bytes a character hold headChars characters in full, so no
// character the head needs is ever cut off by the buffer.
func readHead(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	buf := make([]byte, 4*headChars)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", false
	}
	data := buf[:n]
	var out strings.Builder
	for count := 0; len(data) > 0 && count < headChars; count++ {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size <= 1 {
			return "", false
		}
		if r == '\r' {
			if len(data) > 1 && data[1] == '\n' {
				size = 2
			}
			r = '\n'
		}
		out.WriteRune(r)
		data = data[size:]
	}
	return out.String(), true
}
```

`internal/brain/convert/transcript.go`:

```go
package convert

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// threshold is the one number that carries both kinds of transcript: 1289
// fragments at 37 characters and 67 at 737 (transcript.py:12-15).
const threshold = 1200

type piece struct{ mark, text string }

// ToParagraphs joins the fragments of a transcript into paragraphs, each
// carrying the mark of its first fragment, without changing a word.
func ToParagraphs(text string, f Format) string { return toParagraphs(text, f, threshold) }

func toParagraphs(text string, f Format, limit int) string {
	var paragraphs, current []string
	mark := ""
	for _, p := range split(text, f) {
		if len(current) == 0 {
			mark = p.mark
		} else if p.mark != "" && mark == "" {
			// The paragraph in progress is the lead-in without a mark; the
			// first marked fragment starts its own.
			paragraphs = append(paragraphs, joined(mark, current))
			current = nil
			mark = p.mark
		}
		current = append(current, p.text)
		if size(current) > limit {
			paragraphs = append(paragraphs, joined(mark, current))
			current = nil
		}
	}
	if len(current) > 0 {
		paragraphs = append(paragraphs, joined(mark, current))
	}
	return strings.Join(paragraphs, "\n\n")
}

// split is re.split over the marks: the lead-in, then each mark with the
// speech up to the next one, the speech's whitespace pulled together.
func split(text string, f Format) []piece {
	pattern := bracketMark()
	if f == TranscriptRange {
		pattern = rangeMark()
	}
	matches := pattern.FindAllStringSubmatchIndex(text, -1)
	var pieces []piece
	lead := text
	if len(matches) > 0 {
		lead = text[:matches[0][0]]
	}
	if l := pytext.Strip(lead); l != "" {
		pieces = append(pieces, piece{text: l})
	}
	for i, m := range matches {
		end := len(text)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		if body := strings.Join(strings.FieldsFunc(text[m[1]:end], pytext.IsSpace), " "); body != "" {
			pieces = append(pieces, piece{mark: minutes(text[m[2]:m[3]]), text: body})
		}
	}
	return pieces
}

// minutes writes hh:mm:ss or mm:ss as mm:ss, hours turned into minutes: an
// anchor names the place in the video, not a time of day.
func minutes(raw string) string {
	parts := strings.Split(raw, ":")
	n := make([]int, len(parts))
	for i, part := range parts {
		n[i], _ = strconv.Atoi(part)
	}
	if len(n) == 3 {
		return fmt.Sprintf("%02d:%02d", n[0]*60+n[1], n[2])
	}
	return fmt.Sprintf("%02d:%02d", n[0], n[1])
}

// size is the length the reference measures: each part plus one.
func size(parts []string) int {
	total := 0
	for _, part := range parts {
		total += utf8.RuneCountInString(part) + 1
	}
	return total
}

func joined(mark string, parts []string) string {
	body := strings.Join(parts, " ")
	if mark == "" {
		return body
	}
	return "[" + mark + "] " + body
}
```

`internal/brain/convert/header.go`:

```go
package convert

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// The converters' own counts, raised by hand when the output changes
// (header.py:11-17). The PDF count is 2: pdftotext writes another text than
// the reference's pypdf.
const (
	TranscriptConverter = "brain-transcript/1"
	PDFConverter        = "brain-pdf/2"
)

var youtubeID = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`\(([0-9A-Za-z_-]{11})\)`) })
var converterLine = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?m)^converter:\s*(\S+)\s*$`) })
var descriptionLine = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?m)^description:\s*(\S.*?)\s*$`) })

// SourceURLFrom is the source's URL as far as a file name yields one: an
// eleven-character parenthesised run is read as a YouTube id, even where it
// is none -- the length is all a name offers.
func SourceURLFrom(name string) string {
	if m := youtubeID().FindStringSubmatch(name); m != nil {
		return "https://www.youtube.com/watch?v=" + m[1]
	}
	return ""
}

// Head is the provenance head: four lines, five with a sentence from the
// local model. `asr` says whether speech recognition made the text, so that
// it never counts as a verbatim quote later.
type Head struct {
	SourceURL   string
	Retrieved   time.Time
	Converter   string
	ASR         bool
	Description string
}

// String is the head as YAML frontmatter with a blank line after. The
// description line is left out rather than written empty: an empty line
// would rewrite every file converted before the model existed.
func (h Head) String() string {
	var b strings.Builder
	b.WriteString("---\n")
	if h.SourceURL == "" {
		b.WriteString("source_url:\n")
	} else {
		fmt.Fprintf(&b, "source_url: %s\n", h.SourceURL)
	}
	fmt.Fprintf(&b, "retrieved: %s\nconverter: %s\nasr: %t\n", h.Retrieved.Format("2006-01-02"), h.Converter, h.ASR)
	if h.Description != "" {
		fmt.Fprintf(&b, "description: %s\n", h.Description)
	}
	b.WriteString("---\n\n")
	return b.String()
}

// frontmatter is the head block, or false where there is none to read.
func frontmatter(text string) (string, bool) {
	if !strings.HasPrefix(text, "---\n") {
		return "", false
	}
	end := strings.Index(text[3:], "\n---")
	if end == -1 {
		return "", false
	}
	return text[:3+end], true
}

// ConvertedBy names the converter that wrote a file. What names none was
// written by a person and is never overwritten.
func ConvertedBy(text string) (string, bool) { return headLine(text, converterLine()) }

// DescriptionOf is the sentence a head already carries; a second run reads
// it back rather than ask for another.
func DescriptionOf(text string) (string, bool) { return headLine(text, descriptionLine()) }

func headLine(text string, line *regexp.Regexp) (string, bool) {
	head, ok := frontmatter(text)
	if !ok {
		return "", false
	}
	m := line.FindStringSubmatch(head)
	if m == nil {
		return "", false
	}
	return m[1], true
}
```

`text.find("\n---", 3)` in der Referenz sucht ab Index 3; `strings.Index(text[3:], …)` plus 3 ist dasselbe.

Run: `go test ./internal/brain/convert/ -count=1`
Expected: PASS.

- [ ] **Step 3: Coverage und Commit**

Run: `go test ./internal/brain/convert/ -coverprofile=cover.out -count=1 && go tool cover -func=cover.out | grep -v 100.0%`
Expected: nichts unter 100 %. Den `io.ReadFull`-Fehlerzweig deckt das
Verzeichnis `ordner.txt` (`os.Open` gelingt, `Read` scheitert). Tut es das
unter Windows nicht, bekommt `readHead` ein `//coverage:exempt` mit dem
gemessenen Grund.

`internal/brain/model` steht mit im `git add`: Es ändert sich nur, wenn ein
Satz aus `readsBackDeviations` `readsBack` enger macht (Step 1), und bleibt
sonst ohne Wirkung.

```bash
git add internal/brain/convert internal/brain/model
git commit -m "feat(convert): recognise a transcript, join its fragments and write the provenance head"
```

---

### Task 6: PDF über Poppler

**Files:**
- Create: `internal/programs/programs.go`, `programs_test.go`
- Modify: `internal/setup/tools.go` (Liste aus `programs`)
- Create: `internal/brain/convert/pdf.go`, `pdf_test.go`

**Interfaces:**
- Produces: `programs.All() []programs.Program` (`Name`, `Install`), `programs.Install(name string) string`; `type Tools struct{ Run func(child.Spec) child.Result; Look func(string) (string, error) }`, `func SystemTools() Tools`; `type pdftotext struct` mit `func newPDFToText(t Tools) *pdftotext`, `func (p *pdftotext) extract(path string) (extraction, error)`; `type extraction struct{ text string; pages, skipped int }`; `type toolError struct{ msg string }`; `func wash(page string) string`; `func splitPages(out string) []string`; `func firstLine(said string) string`.

- [ ] **Step 1: `programs` mit Test**

`internal/programs/programs_test.go`:

```go
package programs

import "testing"

func TestEveryProgramNamesItsInstaller(t *testing.T) {
	names := map[string]bool{}
	for _, p := range All() {
		if p.Name == "" || p.Install == "" || names[p.Name] {
			t.Fatalf("%+v", p)
		}
		names[p.Name] = true
	}
	for _, name := range []string{"git", "qmd", "pdftotext", "yt-dlp", "ollama"} {
		if !names[name] {
			t.Errorf("%s is missing", name)
		}
	}
}

func TestInstallFindsTheCommandByName(t *testing.T) {
	if got := Install("pdftotext"); got != "winget install --id oschwartz10612.Poppler -e" {
		t.Fatal(got)
	}
	if got := Install("nothing"); got != "" {
		t.Fatal(got)
	}
}
```

`internal/programs/programs.go`:

```go
// Package programs names the external programs loomux calls and the command
// that installs each. loomux never runs the command; init and convert name
// it.
package programs

// Program is one external program and its installer.
type Program struct{ Name, Install string }

// All are the programs in the order init checks them. The winget ids were
// each confirmed with `winget search` on 2026-09-24; pdftotext comes with
// Poppler.
func All() []Program {
	return []Program{
		{"git", "winget install --id Git.Git -e"},
		// winget has no qmd; the command is the one of the project's README,
		// https://github.com/tobi/qmd (read 2026-09-24).
		{"qmd", "npm install -g @tobilu/qmd"},
		{"pdftotext", "winget install --id oschwartz10612.Poppler -e"},
		{"yt-dlp", "winget install --id yt-dlp.yt-dlp -e"},
		{"ollama", "winget install --id Ollama.Ollama -e"},
	}
}

// Install is the command that installs name, or "" for a program loomux
// does not name.
func Install(name string) string {
	for _, p := range All() {
		if p.Name == name {
			return p.Install
		}
	}
	return ""
}
```

In `internal/setup/tools.go` wird `tools()` zu:

```go
// tools are checked on every run; the list lives in internal/programs,
// which convert reads as well.
func tools() []tool {
	var out []tool
	for _, p := range programs.All() {
		out = append(out, tool{p.Name, p.Install})
	}
	return out
}
```

Run: `go test ./internal/programs/ ./internal/setup/ -count=1`
Expected: PASS (die Tests von `setup` sehen dieselbe Liste).

- [ ] **Step 2: Den failing test für `pdf.go` schreiben**

`internal/brain/convert/pdf_test.go`:

```go
package convert

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

const popplerV = "pdftotext version 25.07.0\nCopyright 2005-2025 The Poppler Developers - http://poppler.freedesktop.org\n"

const longText = "Hallo aus dem Pruefbestand. Diese Zeile testet die Extraktion aus einer " +
	"von Hand geschriebenen PDF-Datei, ohne Texterkennung und ohne fremde Erzeugerbibliothek."

// fakeTools answers pdftotext from a table keyed by the argument after the
// program, and counts the calls.
type fakeTools struct {
	missing  bool
	version  child.Result
	pages    map[string]child.Result
	calls    []child.Spec
	versions int
}

func (f *fakeTools) tools() Tools {
	return Tools{
		Look: func(name string) (string, error) {
			if f.missing {
				return "", exec.ErrNotFound
			}
			return `C:\bin\` + name + ".exe", nil
		},
		Run: func(spec child.Spec) child.Result {
			f.calls = append(f.calls, spec)
			if spec.Argv[1] == "-v" {
				f.versions++
				return f.version
			}
			return f.pages[spec.Argv[6]]
		},
	}
}

func poppler(pages map[string]child.Result) *fakeTools {
	return &fakeTools{version: child.Result{Stderr: popplerV}, pages: pages}
}

func ok(out string) child.Result { return child.Result{Stdout: out} }

func TestATextPDFYieldsItsWashedText(t *testing.T) {
	f := poppler(map[string]child.Result{"text.pdf": ok("   " + longText + "      Ende.\f")})
	e, err := newPDFToText(f.tools()).extract(filepath.Join(`C:\inbox`, "text.pdf"))
	if err != nil || e.pages != 1 || e.skipped != 0 || !strings.Contains(e.text, "Pruefbestand") || strings.Contains(e.text, "  ") {
		t.Fatalf("%+v %v", e, err)
	}
}

// Review Focus 5: the name goes relative, the inbox is the directory.
func TestPdftotextGetsTheNameAndTheInbox(t *testing.T) {
	f := poppler(map[string]child.Result{"Bericht März.pdf": ok(longText + "\f")})
	path := filepath.Join(`C:\Vault\00 Eingang`, "Bericht März.pdf")
	if _, err := newPDFToText(f.tools()).extract(path); err != nil {
		t.Fatal(err)
	}
	call := f.calls[len(f.calls)-1]
	want := []string{`C:\bin\pdftotext.exe`, "-layout", "-enc", "UTF-8", "-eol", "unix", "Bericht März.pdf", "-"}
	if strings.Join(call.Argv, "|") != strings.Join(want, "|") || call.Dir != `C:\Vault\00 Eingang` || call.Timeout != 2*time.Minute {
		t.Fatalf("%+v", call)
	}
}

func TestScanPagesAreCountedPerPage(t *testing.T) {
	for _, c := range []struct {
		name, out            string
		text                 bool
		pages, skipped       int
		containsPruefbestand int
	}{
		{"blank", "\f", false, 1, 1, 0},
		{"pageless", "", false, 0, 0, 0},
		{"mixed", strings.Repeat(longText+"\f", 3) + strings.Repeat("\f", 7), true, 10, 7, 3},
		{"allscan", "\f\f", false, 2, 2, 0},
		{"short", "zu kurz\f", false, 1, 1, 0},
	} {
		f := poppler(map[string]child.Result{c.name + ".pdf": ok(c.out)})
		e, err := newPDFToText(f.tools()).extract(c.name + ".pdf")
		if err != nil || (e.text != "") != c.text || e.pages != c.pages || e.skipped != c.skipped ||
			strings.Count(e.text, "Pruefbestand") != c.containsPruefbestand {
			t.Errorf("%s: %+v %v", c.name, e, err)
		}
	}
}

func TestBlankLinesSeparateParagraphs(t *testing.T) {
	if got := wash("erste  Zeile\nzweite\t\tZeile\n\n\ndritte\n"); got != "erste Zeile zweite Zeile\n\ndritte" {
		t.Fatalf("%q", got)
	}
	if got := wash("erster Absatz\n\n"); got != "erster Absatz" {
		t.Fatalf("%q", got)
	}
}

func TestSplitPagesDropsTheLastFormFeed(t *testing.T) {
	if got := splitPages("a\fb\f"); len(got) != 2 || got[1] != "b" {
		t.Fatalf("%q", got)
	}
	if got := splitPages("a\fb"); len(got) != 2 {
		t.Fatalf("%q", got)
	}
	if got := splitPages(""); len(got) != 0 {
		t.Fatalf("%q", got)
	}
}

func TestAPDFPdftotextRefusesIsUnreadable(t *testing.T) {
	for name, res := range map[string]child.Result{
		"corrupt":  {Code: 1, Stderr: "Syntax Error: Couldn't find trailer dictionary\n"},
		"notstart": {Code: -1, Err: errors.New("boom")},
		"slow":     {Code: -1, TimedOut: true},
		"latin1":   {Stdout: "\xe4\f"},
	} {
		f := poppler(map[string]child.Result{name + ".pdf": res})
		_, err := newPDFToText(f.tools()).extract(name + ".pdf")
		var tool *toolError
		if err == nil || errors.As(err, &tool) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Review Focus 3: Git Bash finds xpdf. Its -v ends with 99 and writes to
// stdout with CRLF (measured): the exit code decides nothing, the text does,
// and no CR lands in the message.
func TestOnlyPopplerIsTaken(t *testing.T) {
	f := &fakeTools{version: child.Result{Code: 99, Stdout: "pdftotext version 4.06 [www.xpdfreader.com]\r\nCopyright 1996-2025 Glyph & Cog, LLC\r\n"}}
	_, err := newPDFToText(f.tools()).extract("text.pdf")
	var tool *toolError
	if !errors.As(err, &tool) || !strings.Contains(err.Error(), "(pdftotext version 4.06 [www.xpdfreader.com]);") ||
		!strings.Contains(err.Error(), "winget install --id oschwartz10612.Poppler -e") {
		t.Fatal(err)
	}
}

func TestAMissingPdftotextNamesItsInstaller(t *testing.T) {
	f := &fakeTools{missing: true}
	_, err := newPDFToText(f.tools()).extract("text.pdf")
	var tool *toolError
	if !errors.As(err, &tool) || !strings.Contains(err.Error(), "pdftotext is not on PATH; install it with: winget install --id oschwartz10612.Poppler -e") {
		t.Fatal(err)
	}
}

func TestTheVersionIsAskedOncePerRun(t *testing.T) {
	f := poppler(map[string]child.Result{"a.pdf": ok(longText + "\f"), "b.pdf": ok(longText + "\f")})
	p := newPDFToText(f.tools())
	p.extract("a.pdf")
	p.extract("b.pdf")
	if f.versions != 1 {
		t.Fatalf("asked %d times", f.versions)
	}
}

func TestSystemToolsAreChildAndLookPath(t *testing.T) {
	tools := SystemTools()
	if tools.Run == nil || tools.Look == nil {
		t.Fatal(tools)
	}
}
```

Run: `go test ./internal/brain/convert/ -run 'PDF|Pdftotext|Scan|Paragraphs|SplitPages|Poppler|Version|SystemTools' -count=1`
Expected: FAIL, `Tools`, `newPDFToText`, `extract`, `wash`, `splitPages`,
`toolError`, `SystemTools` fehlen.

- [ ] **Step 3: Implementieren**

`internal/brain/convert/pdf.go`:

```go
package convert

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/programs"
)

// scanThreshold is the least a page must hold to count as text: measured
// text pages held 1931 to 2593 characters (pdf.py:15-18). Measured on
// pypdf's output; the self-use measures it again on Poppler's.
const scanThreshold = 100

// pdfTimeout bounds one pdftotext call; the reference had none, and a
// hanging PDF must not hold the whole batch.
const pdfTimeout = 2 * time.Minute

// Tools is the seam to the programs convert and fetch start.
type Tools struct {
	Run  func(child.Spec) child.Result
	Look func(string) (string, error)
}

// SystemTools starts the real programs from the PATH.
func SystemTools() Tools { return Tools{Run: child.Run, Look: exec.LookPath} }

// toolError is a program that is missing or the wrong one: every PDF of the
// run is left for a person, with the command that fixes it.
type toolError struct{ msg string }

func (e *toolError) Error() string { return e.msg }

// extraction is what one PDF yields: the washed text of the pages kept, ""
// where none was, and how many pages there were and were skipped as scans.
type extraction struct {
	text           string
	pages, skipped int
}

// pdftotext is Poppler's program, found and checked once per run.
type pdftotext struct {
	tools Tools
	once  sync.Once
	exe   string
	err   error
}

func newPDFToText(t Tools) *pdftotext { return &pdftotext{tools: t} }

// resolve finds pdftotext and takes it only when `-v` names Poppler: xpdf
// writes another text under the same name, and two builds on one machine
// would rewrite every PDF target from the other shell. The exit code of -v
// decides nothing: xpdf 4.06 ends it with 99 and writes to stdout, Poppler
// 25.07.0 with 0 to stderr (measured 2026-09-26), so xpdf is the wrong
// program, not a missing one.
func (p *pdftotext) resolve() (string, error) {
	p.once.Do(func() {
		install := programs.Install("pdftotext")
		exe, err := p.tools.Look("pdftotext")
		if err != nil {
			p.err = &toolError{"pdftotext is not on PATH; install it with: " + install}
			return
		}
		res := p.tools.Run(child.Spec{Argv: []string{exe, "-v"}, Timeout: pdfTimeout})
		said := res.Stdout + res.Stderr
		if !strings.Contains(said, "Poppler") {
			p.err = &toolError{fmt.Sprintf("%s is not Poppler's pdftotext (%s); install Poppler with: %s", exe, firstLine(said), install)}
			return
		}
		p.exe = exe
	})
	return p.exe, p.err
}

// firstLine is the first line a program said. pdftotext writes -v and its
// errors with CRLF under Windows, xpdf and Poppler alike (measured
// 2026-09-26); the CR must not end up inside a skipped: line.
func firstLine(said string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(said), "\n")
	return strings.TrimSpace(first)
}

// extract is this PDF's text, each page held to the scan threshold on its
// own: three text pages among seven scans keep their three.
func (p *pdftotext) extract(path string) (extraction, error) {
	exe, err := p.resolve()
	if err != nil {
		return extraction{}, err
	}
	res := p.tools.Run(child.Spec{
		Argv:    []string{exe, "-layout", "-enc", "UTF-8", "-eol", "unix", filepath.Base(path), "-"},
		Dir:     filepath.Dir(path),
		Timeout: pdfTimeout,
	})
	switch {
	case res.Err != nil:
		return extraction{}, fmt.Errorf("pdftotext did not start: %v", res.Err)
	case res.TimedOut:
		return extraction{}, fmt.Errorf("pdftotext took longer than %s", pdfTimeout)
	case res.Code != 0:
		return extraction{}, fmt.Errorf("pdftotext exited %d: %s", res.Code, firstLine(res.Stderr))
	case !utf8.ValidString(res.Stdout):
		return extraction{}, fmt.Errorf("pdftotext wrote no UTF-8")
	}
	pages := splitPages(res.Stdout)
	var kept []string
	for _, page := range pages {
		if utf8.RuneCountInString(pytext.Strip(page)) >= scanThreshold {
			kept = append(kept, wash(page))
		}
	}
	return extraction{text: strings.Join(kept, "\n\n"), pages: len(pages), skipped: len(pages) - len(kept)}, nil
}

// splitPages cuts pdftotext's output at its form feeds; the one after the
// last page ends no page.
func splitPages(out string) []string {
	if out == "" {
		return nil
	}
	pages := strings.Split(out, "\f")
	if pages[len(pages)-1] == "" {
		pages = pages[:len(pages)-1]
	}
	return pages
}

var spaceRuns = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`[ \t]{2,}`) })

// wash pulls the layout's floods of whitespace into one character and
// blank-separated lines into paragraphs (pdf.py:53-66).
func wash(page string) string {
	var paragraphs, current []string
	for _, line := range pytext.SplitLines(page) {
		line = pytext.Strip(spaceRuns().ReplaceAllString(line, " "))
		switch {
		case line != "":
			current = append(current, line)
		case len(current) > 0:
			paragraphs = append(paragraphs, strings.Join(current, " "))
			current = nil
		}
	}
	if len(current) > 0 {
		paragraphs = append(paragraphs, strings.Join(current, " "))
	}
	return strings.Join(paragraphs, "\n\n")
}
```

Die Referenz verbindet `_wash(page) for page in kept if page.strip()`; eine
behaltene Seite trägt mindestens 100 Zeichen, die Bedingung ist also immer
wahr und fehlt hier.

Run: `go test ./internal/brain/convert/ -count=1`
Expected: PASS. `TestSystemToolsAreChildAndLookPath` deckt `SystemTools`.

- [ ] **Step 4: Coverage und Commit**

Run: `go test ./internal/brain/convert/ ./internal/programs/ ./internal/setup/ -coverprofile=cover.out -count=1 && go tool cover -func=cover.out | grep -v 100.0%`
Expected: nichts unter 100 %.

Run: `go test ./internal/cli/ -run 'TestHooksNever|TestTheCommandLineDoesReach|TestSelfupdateStaysBelow' -count=1`
Expected: PASS. Das sind die Grenztests in `internal/cli/imports_test.go`,
die `go list -deps` fragen (`TestInSetupTreeTakesTheTreeAndNoSibling` prüft
nur die Hilfsfunktion und regelt keinen Import). `setup` importiert jetzt
`programs`, das außerhalb des Installer-Baums liegt und das `hooks` nicht
erreicht; schlägt einer an, ist es eine echte Kante auf den Pro-Edit-Pfad,
und der Task hält an, statt eine Liste zu erweitern.

```bash
git add internal/programs internal/setup/tools.go internal/brain/convert/pdf.go internal/brain/convert/pdf_test.go
git commit -m "feat(convert): read a PDF's text through Poppler's pdftotext, page by page"
```

---

### Task 7: Der Lauf über die Eingänge

**Files:**
- Create: `internal/brain/convert/run.go`, `run_test.go`, `model_test.go`

**Interfaces:**
- Consumes: `Detect`, `ToParagraphs`, `Head`, `ConvertedBy`, `DescriptionOf`, `SourceURLFrom` (Task 5); `newPDFToText`, `toolError`, `Tools` (Task 6); `model.ProposerFor`, `(*model.Proposer).Describe`, `Place` (Task 4); `config.ReadModelSettings`, `config.ReadAreaManifestUntilStage4`, `config.ResolvedAreaDir`, `config.ErrNoManifest`, `lock.ReplaceText`, `pytext.ReadText`
- Produces: `type Area struct{ config.Area; Manifest *config.Manifest; Inbox, Mode string }`, `func Areas(areas []config.Area, stateDir, fallbackDir string) ([]Area, error)`, `type Outcome struct{ Written, Skipped, Suggested []string }`, `func ConvertFile(ctx context.Context, tools Tools, path string) (target, message string)`, `func ConvertAll(ctx context.Context, tools Tools, areas []Area, stateDir string) (Outcome, error)`, `func writeIfChanged(path, text string) (bool, error)`

- [ ] **Step 1: Den failing test schreiben — ohne Modell**

`internal/brain/convert/run_test.go`:

```go
package convert

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/config"
)

// world is one machine: a state directory with a registry, and the areas
// below root.
type world struct {
	t        *testing.T
	root     string
	state    string
	registry strings.Builder
	tools    Tools
}

func newWorld(t *testing.T) *world {
	t.Helper()
	root := t.TempDir()
	state := filepath.Join(root, "state")
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	w := &world{t: t, root: root, state: state, tools: poppler(nil).tools()}
	w.flush()
	return w
}

func (w *world) flush() {
	if err := os.WriteFile(filepath.Join(w.state, "registry.toml"), []byte(w.registry.String()), 0o644); err != nil {
		w.t.Fatal(err)
	}
}

// area registers scope; a declaration of "" writes none. It answers the
// area's directory.
func (w *world) area(scope, declaration string, readonly bool) string {
	dir := filepath.Join(w.root, strings.ReplaceAll(scope, "/", "-"))
	if err := os.MkdirAll(filepath.Join(dir, ".loomux"), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if declaration != "" {
		if err := os.WriteFile(filepath.Join(dir, ".loomux", "config.toml"), []byte(declaration), 0o644); err != nil {
			w.t.Fatal(err)
		}
	}
	fmt.Fprintf(&w.registry, "[[area]]\nscope = %q\npath = %q\n", scope, filepath.ToSlash(dir))
	if readonly {
		w.registry.WriteString("readonly = true\n")
	}
	w.registry.WriteString("\n")
	w.flush()
	return dir
}

// inbox registers an area whose declaration names `00 Eingang`, with extra
// tables after it, and answers the inbox.
func (w *world) inbox(scope, extra string) string {
	dir := w.area(scope, fmt.Sprintf("[area]\nscope = %q\n\n[layout]\ninbox = \"00 Eingang\"\n\n%s", scope, extra), false)
	inbox := filepath.Join(dir, "00 Eingang")
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		w.t.Fatal(err)
	}
	return inbox
}

func (w *world) run() (Outcome, error) {
	w.t.Helper()
	areas, err := config.ReadRegistry(w.state)
	if err != nil {
		return Outcome{}, err
	}
	entries, err := Areas(areas, w.state, filepath.Join(w.root, "legacy"))
	if err != nil {
		return Outcome{}, err
	}
	return ConvertAll(context.Background(), w.tools, entries, w.state)
}

func put(t *testing.T, dir, name, data string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestATranscriptBecomesAMarkdownFile(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "video (mHSOsy_usAg).txt", "[00:00] Hallo zusammen.\n")
	out, err := w.run()
	target := filepath.Join(inbox, "video (mHSOsy_usAg).txt.md")
	if err != nil || !slices.Equal(out.Written, []string{target}) || len(out.Skipped) != 0 {
		t.Fatalf("%+v %v", out, err)
	}
	text := read(t, target)
	for _, want := range []string{"converter: brain-transcript/1\n", "source_url: https://www.youtube.com/watch?v=mHSOsy_usAg\n", "asr: true\n", "---\n\n[00:00] Hallo zusammen.\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %q", want, text)
		}
	}
}

func TestTheSecondRunWritesNothing(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n")
	w.run()
	first := read(t, filepath.Join(inbox, "video.txt.md"))
	out, err := w.run()
	if err != nil || len(out.Written) != 0 || len(out.Skipped) != 0 || read(t, filepath.Join(inbox, "video.txt.md")) != first {
		t.Fatalf("%+v %v", out, err)
	}
}

// retrieved is the source's modification day in UTC, never "now".
func TestRetrievedIsTheSourcesDayInUTC(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	source := put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	late := time.Date(2026, 1, 2, 23, 30, 0, 0, time.FixedZone("EST", -5*3600))
	if err := os.Chtimes(source, late, late); err != nil {
		t.Fatal(err)
	}
	w.run()
	if !strings.Contains(read(t, source+".md"), "retrieved: 2026-01-03\n") {
		t.Fatal(read(t, source+".md"))
	}
}

func TestTwoSourcesWithTheSameStemBothSurvive(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	w.tools = poppler(map[string]child.Result{"doku.pdf": ok(longText + "\f")}).tools()
	put(t, inbox, "doku.pdf", "%PDF-1.4\n")
	put(t, inbox, "doku.txt", "[00:00] Aus dem Transkript.\n")
	if out, err := w.run(); err != nil || len(out.Written) != 2 {
		t.Fatalf("%+v %v", out, err)
	}
	pdf := read(t, filepath.Join(inbox, "doku.pdf.md"))
	if !strings.Contains(pdf, "Pruefbestand") || !strings.Contains(pdf, "converter: brain-pdf/2\n") || !strings.Contains(pdf, "asr: false\n") {
		t.Fatal(pdf)
	}
	if !strings.Contains(read(t, filepath.Join(inbox, "doku.txt.md")), "Aus dem Transkript.") {
		t.Fatal("the transcript's target")
	}
}

func TestWhatIsLeftBehindIsReportedAndTheRunGoesOn(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	w.tools = poppler(map[string]child.Result{
		"scan.pdf":    ok("\f"),
		"leer.pdf":    ok(""),
		"teil.pdf":    ok(strings.Repeat(longText+"\f", 3) + strings.Repeat("\f", 7)),
		// pdftotext writes its errors with CRLF under Windows (measured).
		"kaputt.pdf":  {Code: 1, Stderr: "Syntax Error\r\nSyntax Error: Couldn't read xref table\r\n"},
	}).tools()
	put(t, inbox, "notiz.txt", "Nur Prosa.\n")
	put(t, inbox, "kaputt.txt", "[00:00] Anfang ist sauber.\n"+strings.Repeat("x", 8300)+"\n"+strings.Repeat("\xff\xfe", 50))
	put(t, inbox, "hand.txt", "[00:00] Hallo.\n")
	put(t, inbox, "hand.txt.md", "---\ntitle: Meine Notiz\n---\n\nHandarbeit.\n")
	put(t, inbox, "bytes.txt", "[00:00] Hallo.\n")
	put(t, inbox, "bytes.txt.md", strings.Repeat("\xff\xfe", 8192))
	for _, name := range []string{"scan.pdf", "leer.pdf", "teil.pdf", "kaputt.pdf"} {
		put(t, inbox, name, "%PDF-1.4\n")
	}
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	out, err := w.run()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"notiz.txt: no converter knows this format",
		"kaputt.txt: cannot be read as UTF-8 text",
		"hand.txt.md: not written by us, left untouched",
		"bytes.txt.md: cannot be read as UTF-8 text",
		"scan.pdf: no extractable text, looks like a scan",
		"leer.pdf: no pages to extract",
		"teil.pdf: 7 page(s) skipped as scanned",
		"kaputt.pdf: cannot be read as a PDF (pdftotext exited 1: Syntax Error)",
	} {
		if !slices.Contains(out.Skipped, want) {
			t.Errorf("no %q in %q", want, out.Skipped)
		}
	}
	if !slices.Contains(out.Written, filepath.Join(inbox, "video.txt.md")) || !slices.Contains(out.Written, filepath.Join(inbox, "teil.pdf.md")) {
		t.Fatalf("%q", out.Written)
	}
	if read(t, filepath.Join(inbox, "hand.txt.md")) != "---\ntitle: Meine Notiz\n---\n\nHandarbeit.\n" {
		t.Fatal("a hand-written file was touched")
	}
}

// Review Focus 3: xpdf leaves every PDF for a person; the transcript still goes.
func TestXpdfLeavesThePDFsAndTheTranscriptGoes(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	w.tools = (&fakeTools{version: child.Result{Code: 99, Stdout: "pdftotext version 4.06 [www.xpdfreader.com]\r\n"}}).tools()
	put(t, inbox, "buch.pdf", "%PDF-1.4\n")
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	out, _ := w.run()
	if len(out.Skipped) != 1 || !strings.HasPrefix(out.Skipped[0], "buch.pdf: ") || !strings.Contains(out.Skipped[0], "install Poppler with: ") {
		t.Fatalf("%q", out.Skipped)
	}
	if _, err := os.Stat(filepath.Join(inbox, "buch.pdf.md")); err == nil {
		t.Fatal("a PDF target was written")
	}
	if !slices.Equal(out.Written, []string{filepath.Join(inbox, "video.txt.md")}) {
		t.Fatalf("%q", out.Written)
	}
}

func TestAnUnwritableTargetIsReported(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "a.txt", "[00:00] Hallo.\n")
	target := put(t, inbox, "a.txt.md", "---\nsource_url:\nretrieved: 2020-01-01\nconverter: brain-transcript/1\nasr: true\n---\n\nAlt.\n")
	if err := os.Chmod(target, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(target, 0o644) })
	put(t, inbox, "b.txt", "[00:00] Welt.\n")
	out, _ := w.run()
	if len(out.Skipped) != 1 || !strings.HasPrefix(out.Skipped[0], "a.txt.md: cannot be written (") {
		t.Fatalf("%q", out.Skipped)
	}
	if !slices.Equal(out.Written, []string{filepath.Join(inbox, "b.txt.md")}) {
		t.Fatalf("%q", out.Written)
	}
}

func TestAnUnreadableTimestampIsReported(t *testing.T) {
	saved := statSource
	t.Cleanup(func() { statSource = saved })
	statSource = func(string) (os.FileInfo, error) { return nil, os.ErrPermission }
	source := put(t, t.TempDir(), "video.txt", "[00:00] Hallo.\n")
	target, message := ConvertFile(context.Background(), poppler(nil).tools(), source)
	if target != "" || !strings.HasPrefix(message, "video.txt: cannot be read (") {
		t.Fatalf("%q %q", target, message)
	}
}

func TestAreasWithoutAWritableInboxAreLeftAlone(t *testing.T) {
	w := newWorld(t)
	w.area("project/x", "", false)
	w.area("project/y", "[area]\nscope = \"project/y\"\n", false)
	// A readonly area's declaration lies where ResolvedAreaDir reads it,
	// under the state directory; in the area it would leave corpus without
	// an inbox, and the test would pass for that reason.
	corpus := w.area("corpus", "", true)
	declared := filepath.Join(w.state, "areas", "corpus", ".loomux")
	os.MkdirAll(declared, 0o755)
	put(t, declared, "config.toml", "[area]\nscope = \"corpus\"\n\n[layout]\ninbox = \"00 Eingang\"\n")
	os.MkdirAll(filepath.Join(corpus, "00 Eingang"), 0o755)
	put(t, filepath.Join(corpus, "00 Eingang"), "video.txt", "[00:00] Hallo.\n")
	gone := w.area("gone", "[area]\nscope = \"gone\"\n\n[layout]\ninbox = \"00 Eingang\"\n", false)
	registered, err := config.ReadRegistry(w.state)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := Areas(registered, w.state, filepath.Join(w.root, "legacy"))
	if i := slices.IndexFunc(entries, func(a Area) bool { return a.Scope == "corpus" }); err != nil || i < 0 || entries[i].Inbox == "" {
		t.Fatalf("corpus has no inbox, so ReadOnly is not what leaves it alone: %+v %v", entries, err)
	}
	out, err := w.run()
	if err != nil || len(out.Written)+len(out.Skipped) != 0 {
		t.Fatalf("%+v %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(corpus, "00 Eingang", "video.txt.md")); err == nil {
		t.Fatal("a readonly area was written into")
	}
	if _, err := os.Stat(filepath.Join(gone, "00 Eingang")); err == nil {
		t.Fatal("an inbox was made")
	}
}

func TestMarkdownAndDirectoriesInTheInboxAreNoSources(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "fertig.MD", "[00:00] Hallo.\n")
	os.Mkdir(filepath.Join(inbox, "ordner.txt"), 0o755)
	if out, err := w.run(); err != nil || len(out.Written)+len(out.Skipped) != 0 {
		t.Fatalf("%+v %v", out, err)
	}
}

// Review Focus 1: the order is the reference's under Windows.
func TestTheInboxIsWalkedInThePlatformsOrder(t *testing.T) {
	names := []string{"B.txt", "a.txt", "_z.txt", "b.pdf"}
	if got := sortedOn("windows", slices.Clone(names)); !slices.Equal(got, []string{"_z.txt", "a.txt", "b.pdf", "B.txt"}) {
		t.Fatalf("windows: %q", got)
	}
	if got := sortedOn("linux", slices.Clone(names)); !slices.Equal(got, []string{"B.txt", "_z.txt", "a.txt", "b.pdf"}) {
		t.Fatalf("linux: %q", got)
	}
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "B.txt", "[00:00] Zwei.\n")
	put(t, inbox, "a.txt", "[00:00] Eins.\n")
	out, _ := w.run()
	if !slices.Equal(out.Written, []string{filepath.Join(inbox, "a.txt.md"), filepath.Join(inbox, "B.txt.md")}) {
		t.Fatalf("%q", out.Written)
	}
}

// Review Focus 2.
func TestACRLFTranscriptConvertsAsLF(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "video.txt", "00:00:00 - 00:00:05\r\nHallo zusammen.\r\n\r\n00:00:06 - 00:00:09\r\nUnd weiter.\r\n")
	w.run()
	text := read(t, filepath.Join(inbox, "video.txt.md"))
	if strings.Contains(text, "\r") || !strings.Contains(text, "[00:00] Hallo zusammen. Und weiter.\n") {
		t.Fatalf("%q", text)
	}
}

func TestABrokenDeclarationStopsTheRunBeforeAnythingIsWritten(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	w.inbox("project/x", "[privacy]\nmode = \"cloud\"\n")
	_, err := w.run()
	if err == nil || !strings.Contains(err.Error(), "mode") {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(inbox, "video.txt.md")); err == nil {
		t.Fatal("a file was written before the refusal")
	}
}

// An absolute inbox would leave the area tree; the declaration is refused
// like any broken one, before a file is touched.
func TestAnAbsoluteInboxInTheDeclarationStopsTheRun(t *testing.T) {
	w := newWorld(t)
	elsewhere := filepath.ToSlash(filepath.Join(w.root, "elsewhere"))
	w.area("knowledge", fmt.Sprintf("[area]\nscope = \"knowledge\"\n\n[layout]\ninbox = %q\n", elsewhere), false)
	if _, err := w.run(); err == nil || !strings.Contains(err.Error(), "[layout] inbox must be relative to the area") {
		t.Fatal(err)
	}
}

func TestOneNamedFileConvertsOnItsOwn(t *testing.T) {
	source := put(t, t.TempDir(), "video.txt", "[00:00] Hallo.\n")
	target, message := ConvertFile(context.Background(), poppler(nil).tools(), source)
	if target != source+".md" || message != "" || strings.Contains(read(t, target), "description") {
		t.Fatalf("%q %q", target, message)
	}
}

func TestWriteIfChangedLeavesAnEqualFile(t *testing.T) {
	path := put(t, t.TempDir(), "x.md", "same\n")
	if written, err := writeIfChanged(path, "same\n"); written || err != nil {
		t.Fatal(written, err)
	}
	if written, err := writeIfChanged(path, "other\n"); !written || err != nil || read(t, path) != "other\n" {
		t.Fatal(written, err)
	}
}
```

Run: `go test ./internal/brain/convert/ -run 'Transcript|SecondRun|Retrieved|SameStem|LeftBehind|Xpdf|Unwritable|Timestamp|Writable|Markdown|Order|CRLF|Declaration|NamedFile|WriteIfChanged' -count=1`
Expected: FAIL, `Areas`, `ConvertAll`, `ConvertFile`, `Outcome`,
`statSource`, `sortedOn`, `writeIfChanged` fehlen.

- [ ] **Step 2: Den failing test schreiben — mit Modell**

`internal/brain/convert/model_test.go` stellt das lokale Modell der Referenz
nach (`tests/convert/test_cli_convert.py:531-573`): ein Server auf Loopback,
der die Rolle am Versionskommentar des Prompts erkennt, nie am Schema, und
das Schema gegen die Rolle prüft.

```go
package convert

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

const sentence = "Der Bericht beschreibt die Abnahme der zweiten Scheibe."

type fakeModel struct {
	mu       sync.Mutex
	scope    string
	sentence string
	describe []string
	place    []string
}

func (m *fakeModel) prompts() (describe, place []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.describe), slices.Clone(m.place)
}

// serveModel switches the model on for the machine and answers from a
// loopback server; scope is what place answers.
func serveModel(t *testing.T, w *world, scope string) *fakeModel {
	t.Helper()
	m := &fakeModel{scope: scope, sentence: sentence}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		prompt, _ := body["prompt"].(string)
		_, format := body["format"]
		m.mu.Lock()
		answer := ""
		switch {
		case strings.HasPrefix(prompt, "<!-- version: ablage-v1\n") && format:
			m.place = append(m.place, prompt)
			answer = fmt.Sprintf(`{"scope": %q, "grund": "Es passt."}`, m.scope)
		case strings.HasPrefix(prompt, "<!-- version: beschreibung-v1 -->\n") && !format:
			m.describe = append(m.describe, prompt)
			answer = m.sentence
		default:
			t.Errorf("a request of no known role (format %v): %.40q", format, prompt)
		}
		m.mu.Unlock()
		_ = json.NewEncoder(rw).Encode(map[string]string{"response": answer})
	}))
	t.Cleanup(server.Close)
	put(t, w.state, "config.toml", fmt.Sprintf("[model]\nenabled = true\nendpoint = %q\n", server.URL))
	return m
}

func roles(describe, place bool) string {
	return fmt.Sprintf("[model]\nroles = { propose = true, place = %t, describe = %t }\n", place, describe)
}

func TestTheDescriptionLandsInTheHeadAskedWithTheBody(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(true, false))
	m := serveModel(t, w, "knowledge")
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n[00:02] Und dann weiter.\n")
	if out, err := w.run(); err != nil || len(out.Skipped) != 0 {
		t.Fatalf("%+v %v", out, err)
	}
	if !strings.Contains(read(t, filepath.Join(inbox, "video.txt.md")), "description: "+sentence+"\n") {
		t.Fatal("no description")
	}
	describe, place := m.prompts()
	if len(describe) != 1 || !strings.HasSuffix(describe[0], "[00:00] Hallo zusammen. Und dann weiter.\n") || len(place) != 0 {
		t.Fatalf("%q %q", describe, place)
	}
}

func TestARefusedDescriptionLeavesTheHeadAndIsAskedAgain(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(true, false))
	m := serveModel(t, w, "knowledge")
	m.sentence = "The report describes the second slice."
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n")
	w.run()
	if strings.Contains(read(t, filepath.Join(inbox, "video.txt.md")), "description") {
		t.Fatal("a refused sentence landed")
	}
	m.mu.Lock()
	m.sentence = sentence
	m.mu.Unlock()
	w.run()
	if describe, _ := m.prompts(); len(describe) != 2 || !strings.Contains(read(t, filepath.Join(inbox, "video.txt.md")), "description: "+sentence) {
		t.Fatalf("asked %d times", len(describe))
	}
}

func TestAStandingSentenceIsNeverAskedFor(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(true, true))
	m := serveModel(t, w, "knowledge")
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n")
	w.run()
	first := read(t, filepath.Join(inbox, "video.txt.md"))
	out, _ := w.run()
	describe, place := m.prompts()
	if len(out.Written)+len(out.Suggested) != 0 || len(describe) != 1 || len(place) != 1 || read(t, filepath.Join(inbox, "video.txt.md")) != first {
		t.Fatalf("%+v %d %d", out, len(describe), len(place))
	}
}

func TestTheModelIsLeftAloneWhileTheMachineSwitchedItOff(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	m := serveModel(t, w, "knowledge")
	settings := filepath.Join(w.state, "config.toml")
	put(t, w.state, "config.toml", strings.Replace(read(t, settings), "enabled = true", "enabled = false", 1))
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	w.run()
	if describe, place := m.prompts(); len(describe)+len(place) != 0 {
		t.Fatal("asked while off")
	}
}

// Off beats on, from below as well.
func TestAnAreaThatSwitchedTheModelOffIsNeverAsked(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "[model]\nenabled = false\n")
	m := serveModel(t, w, "knowledge")
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	w.run()
	if describe, place := m.prompts(); len(describe)+len(place) != 0 {
		t.Fatal("asked although the area said no")
	}
}

func TestAMisconfiguredModelStopsTheRunBeforeAnythingIsWritten(t *testing.T) {
	for block, want := range map[string]string{
		"[model]\nenabled = 5\n": "enabled",
		"[model]\nenabled = true\nendpoint = \"http://192.168.0.10:11434\"\n": "must stay on the loopback",
	} {
		w := newWorld(t)
		inbox := w.inbox("knowledge", "")
		put(t, w.state, "config.toml", block)
		put(t, inbox, "video.txt", "[00:00] Hallo.\n")
		if _, err := w.run(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v", block, err)
		}
		if entries, _ := os.ReadDir(inbox); len(entries) != 1 {
			t.Errorf("%q: something was written", block)
		}
	}
}

// With describe off, ProposerFor never judges the endpoint for it; place is
// where the run stops.
func TestAnEndpointOffTheLoopbackStopsTheRunWhenOnlyPlaceIsOn(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(false, true))
	put(t, w.state, "config.toml", "[model]\nenabled = true\nendpoint = \"http://192.168.0.10:11434\"\n")
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	if _, err := w.run(); err == nil || !strings.Contains(err.Error(), "must stay on the loopback") {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(inbox); len(entries) != 1 {
		t.Fatal("something was written")
	}
}

func TestTheSuggestionNamesTheScopeAndMovesNothing(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(false, true))
	elsewhere := w.area("project/ultra-brain", "", false)
	m := serveModel(t, w, "project/ultra-brain")
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n")
	out, _ := w.run()
	if !slices.Equal(out.Suggested, []string{"video.txt.md: belongs in project/ultra-brain, left in the inbox"}) {
		t.Fatalf("%q", out.Suggested)
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 1 { // only .loomux
		t.Fatal("something moved")
	}
	_, place := m.prompts()
	if len(place) != 1 || !strings.Contains(place[0], "\n- knowledge\n- project/ultra-brain\n") || !strings.Contains(place[0], "[00:00] Hallo zusammen.") {
		t.Fatalf("%q", place)
	}
}

func TestAScopeTheRegisterDoesNotKnowIsNoSuggestion(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(false, true))
	serveModel(t, w, "project/erfunden")
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n")
	if out, _ := w.run(); len(out.Suggested) != 0 || len(out.Written) != 1 {
		t.Fatalf("%+v", out)
	}
}

func TestTargetsAreNeverMoreOpenThanTheInbox(t *testing.T) {
	w := newWorld(t)
	open := w.inbox("knowledge", roles(false, true))
	closed := w.inbox("project/x", "[privacy]\nmode = \"local_only\"\n\n"+roles(false, true))
	w.area("project/bare", "", false)
	w.area("project/auto", "[area]\nscope = \"project/auto\"\n\n[privacy]\nmode = \"automatic_cloud\"\n", false)
	w.area("corpus", "", true)
	m := serveModel(t, w, "knowledge")
	put(t, open, "video.txt", "[00:00] Offen aus dem Wissensbereich.\n")
	put(t, closed, "video.txt", "[00:00] Vertraulich aus dem Projekt.\n")
	w.run()
	_, place := m.prompts()
	for _, prompt := range place {
		switch {
		case strings.Contains(prompt, "Vertraulich"):
			if !strings.Contains(prompt, "\n- project/x\n") || strings.Contains(prompt, "- knowledge\n") || strings.Contains(prompt, "- project/bare\n") {
				t.Errorf("the closed inbox was offered %q", prompt)
			}
		default:
			if !strings.Contains(prompt, "- knowledge\n- project/x\n- project/bare\n") || strings.Contains(prompt, "project/auto") || strings.Contains(prompt, "corpus") {
				t.Errorf("the open inbox was offered %q", prompt)
			}
		}
	}
	if len(place) != 2 {
		t.Fatalf("%d placements", len(place))
	}
}

// A file is asked about under the switches of its own area.
func TestEachFileIsAskedUnderItsOwnAreasSwitches(t *testing.T) {
	for _, c := range []struct {
		block    string
		withheld []string
	}{
		{"[model]\nenabled = false\n", []string{"describe", "place"}},
		{roles(false, true), []string{"describe"}},
		{roles(true, false), []string{"place"}},
	} {
		w := newWorld(t)
		open := w.inbox("knowledge", "")
		closed := w.inbox("project/x", c.block)
		m := serveModel(t, w, "knowledge")
		put(t, open, "video.txt", "[00:00] Offen aus dem Wissensbereich.\n")
		put(t, closed, "video.txt", "[00:00] Vertraulich aus dem Projekt.\n")
		w.run()
		describe, place := m.prompts()
		for role, prompts := range map[string][]string{"describe": describe, "place": place} {
			count := func(body string) int {
				return len(slices.DeleteFunc(slices.Clone(prompts), func(p string) bool { return !strings.Contains(p, body) }))
			}
			want := 1
			if slices.Contains(c.withheld, role) {
				want = 0
			}
			if count("Offen") != 1 || count("Vertraulich") != want {
				t.Errorf("%q %s: open %d, closed %d", c.block, role, count("Offen"), count("Vertraulich"))
			}
		}
	}
}
```

Run: `go test ./internal/brain/convert/ -count=1`
Expected: FAIL wie in Step 1.

- [ ] **Step 3: Implementieren**

`internal/brain/convert/run.go`:

```go
package convert

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/model"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// openness orders the privacy modes from closed to open (run.py:34-36).
var openness = map[string]int{"local_only": 0, "manual_cloud": 1, "automatic_cloud": 2}

// Area is a registered area as convert sees it: its declaration, its inbox
// and the privacy mode it declares.
type Area struct {
	config.Area
	Manifest *config.Manifest // nil where the area declares nothing
	Inbox    string           // "" where the declaration names none
	Mode     string
}

// Areas reads every area's declaration before a file is touched: one broken
// declaration stops the run, as reading the registry stops the reference's.
// An area without a declaration has no inbox and counts as manual_cloud.
func Areas(areas []config.Area, stateDir, fallbackDir string) ([]Area, error) {
	out := make([]Area, 0, len(areas))
	for _, a := range areas {
		entry := Area{Area: a, Mode: "manual_cloud"}
		manifest, err := config.ReadAreaManifestUntilStage4(config.ResolvedAreaDir(a, stateDir, fallbackDir))
		switch {
		case errors.Is(err, config.ErrNoManifest):
		case err != nil:
			return nil, err
		default:
			rel, err := manifest.InboxLayout()
			if err != nil {
				return nil, err
			}
			entry.Manifest, entry.Mode = manifest, manifest.PrivacyMode
			if rel != "" {
				entry.Inbox = filepath.Join(a.Path, rel)
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

// Outcome is what a run wrote, what it left for a person, and where it
// would file what it wrote; a suggestion is a line, not a move.
type Outcome struct {
	Written, Skipped, Suggested []string
}

type describer interface {
	Describe(context.Context, string) (string, bool)
}

type placer interface {
	Place(context.Context, string, []string) (string, bool)
}

// statSource is the seam a test replaces to see a failing stat.
var statSource = os.Stat

type run struct {
	ctx context.Context
	pdf *pdftotext
}

// ConvertFile converts one named path without asking the model: it belongs
// to no area, and the next run over the inboxes fills the sentence in.
func ConvertFile(ctx context.Context, tools Tools, path string) (string, string) {
	r := &run{ctx: ctx, pdf: newPDFToText(tools)}
	target, message, _ := r.one(path, nil, nil, nil)
	return target, message
}

// ConvertAll goes through every writable inbox in registry order. The model
// settings are read and the proposers built before the first file, so a
// misconfiguration stops the run with nothing converted and nothing left
// unlisted. No error in a single file stops it.
func ConvertAll(ctx context.Context, tools Tools, areas []Area, stateDir string) (Outcome, error) {
	settings, err := config.ReadModelSettings(stateDir)
	if err != nil {
		return Outcome{}, err
	}
	var usable []Area
	for _, a := range areas {
		if a.Inbox != "" && !a.ReadOnly && isDir(a.Inbox) {
			usable = append(usable, a)
		}
	}
	// A nil *model.Proposer in an interface would not be nil, so only a
	// proposer that exists goes into the maps.
	describers, placers := map[string]describer{}, map[string]placer{}
	if settings.Enabled {
		for _, a := range usable {
			d, err := model.ProposerFor(settings, a.Manifest, "describe")
			if err != nil {
				return Outcome{}, err
			}
			if d != nil {
				describers[a.Scope] = d
			}
			p, err := model.ProposerFor(settings, a.Manifest, "place")
			if err != nil {
				return Outcome{}, err
			}
			if p != nil {
				placers[a.Scope] = p
			}
		}
	}
	// The register, less what nobody may file into, in registry order.
	var register []Area
	for _, a := range areas {
		if !a.ReadOnly {
			register = append(register, a)
		}
	}
	r := &run{ctx: ctx, pdf: newPDFToText(tools)}
	var out Outcome
	for _, a := range usable {
		pl := placers[a.Scope]
		var scopes []string
		if pl != nil {
			for _, target := range register {
				if openness[target.Mode] <= openness[a.Mode] {
					scopes = append(scopes, target.Scope)
				}
			}
		}
		entries, err := os.ReadDir(a.Inbox)
		if err != nil {
			return out, err
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		for _, name := range sortedOn(runtime.GOOS, names) {
			path := filepath.Join(a.Inbox, name)
			if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() || strings.EqualFold(pySuffix(name), ".md") {
				continue
			}
			target, message, suggestion := r.one(path, describers[a.Scope], pl, scopes)
			if target != "" {
				out.Written = append(out.Written, target)
			}
			if message != "" {
				out.Skipped = append(out.Skipped, message)
			}
			if suggestion != "" {
				out.Suggested = append(out.Suggested, suggestion)
			}
		}
	}
	return out, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// sortedOn is sorted(inbox.iterdir()): Python compares paths in lower case
// under Windows, by code point elsewhere.
func sortedOn(goos string, names []string) []string {
	key := func(s string) string { return s }
	if goos == "windows" {
		key = strings.ToLower
	}
	slices.SortStableFunc(names, func(a, b string) int { return strings.Compare(key(a), key(b)) })
	return names
}

// one converts path and answers the target it wrote ("" where nothing was
// written), what it left behind, and where the written file belongs.
func (r *run) one(path string, d describer, p placer, scopes []string) (string, string, string) {
	name := filepath.Base(path)
	format := Detect(path)
	if format == Unsupported {
		return "", name + ": no converter knows this format", ""
	}
	// The source's extension rides along: doku.pdf and doku.txt must not
	// overwrite each other.
	targetName := name + ".md"
	target := filepath.Join(filepath.Dir(path), targetName)
	kept := ""
	if _, err := os.Stat(target); err == nil {
		existing, err := pytext.ReadText(target)
		if err != nil {
			return "", targetName + ": cannot be read as UTF-8 text", ""
		}
		if _, ours := ConvertedBy(existing); !ours {
			return "", targetName + ": not written by us, left untouched", ""
		}
		// A standing sentence is never traded for another; a head without
		// one is asked again on every run.
		kept, _ = DescriptionOf(existing)
	}
	var body, converter, skip string
	asr := false
	if format == PDF {
		e, err := r.pdf.extract(path)
		var tool *toolError
		switch {
		case errors.As(err, &tool):
			return "", name + ": " + tool.Error(), ""
		case err != nil:
			return "", fmt.Sprintf("%s: cannot be read as a PDF (%v)", name, err), ""
		case e.text == "" && e.pages == 0:
			return "", name + ": no pages to extract", ""
		case e.text == "":
			return "", name + ": no extractable text, looks like a scan", ""
		}
		if e.skipped > 0 {
			skip = fmt.Sprintf("%s: %d page(s) skipped as scanned", name, e.skipped)
		}
		body, converter = e.text, PDFConverter
	} else {
		source, err := pytext.ReadText(path)
		if err != nil {
			return "", name + ": cannot be read as UTF-8 text", ""
		}
		body, converter, asr = ToParagraphs(source, format), TranscriptConverter, true
	}
	info, err := statSource(path)
	if err != nil {
		return "", fmt.Sprintf("%s: cannot be read (%v)", name, err), ""
	}
	description := kept
	if description == "" && d != nil {
		description, _ = d.Describe(r.ctx, body)
	}
	head := Head{SourceURL: SourceURLFrom(name), Retrieved: info.ModTime().UTC(), Converter: converter, ASR: asr, Description: description}
	written, err := writeIfChanged(target, head.String()+body+"\n")
	if err != nil {
		return "", fmt.Sprintf("%s: cannot be written (%v)", targetName, err), ""
	}
	if !written {
		return "", skip, ""
	}
	// Only a file this run wrote is placed: every file stays in its inbox,
	// and asking on every run would repeat the line for every file waiting.
	if p == nil {
		return target, skip, ""
	}
	scope, ok := p.Place(r.ctx, body, scopes)
	if !ok {
		return target, skip, ""
	}
	return target, skip, targetName + ": belongs in " + scope + ", left in the inbox"
}

// writeIfChanged writes only on change, so an untouched file keeps its
// time; the comparison is byte for byte, as `newline=""` reads.
func writeIfChanged(path, text string) (bool, error) {
	if existing, err := os.ReadFile(path); err == nil && string(existing) == text {
		return false, nil
	}
	return true, lock.ReplaceText(path, text)
}
```

`TestOneNamedFileConvertsOnItsOwn` erwartet `source+".md"`; `filepath.Join`
ergibt dasselbe für einen Pfad, den `t.TempDir()` schon sauber liefert.

Run: `go test ./internal/brain/convert/ -count=1`
Expected: PASS. Scheitert `TestTargetsAreNeverMoreOpenThanTheInbox` an der
Reihenfolge der Liste, ist `register` falsch geordnet: Die Referenz nimmt die
Reihenfolge der Registry (`rank` ist ein Dict in Einfügeordnung).

- [ ] **Step 4: Coverage und Commit**

Run: `go test ./internal/brain/convert/ -coverprofile=cover.out -count=1 && go tool cover -func=cover.out | grep -v 100.0%`
Expected: nichts unter 100 %. Den Fehlerzweig von `InboxLayout` in `Areas`
deckt `TestAnAbsoluteInboxInTheDeclarationStopsTheRun`, den des zweiten
`ProposerFor` (`place`) in `ConvertAll`
`TestAnEndpointOffTheLoopbackStopsTheRunWhenOnlyPlaceIsOn`. Offen bleiben
kann der Fehlerzweig von
`os.ReadDir` in `ConvertAll` (ein Eingang, der `isDir` besteht und sich dann
nicht lesen lässt); ihn deckt ein Test, der zwischen `Areas` und `ConvertAll`
dem Eingang die Leserechte nimmt — unter Windows über `icacls <inbox> /deny
%USERNAME%:(RX)` in `t.Cleanup` zurückgenommen. Gelingt das nicht, bekommt
`ConvertAll` kein Exempt, sondern der Zweig zieht in eine eigene Funktion
`readInbox` mit Exempt und gemessenem Grund.

```bash
git add internal/brain/convert
git commit -m "feat(convert): go through every writable inbox, describing and placing what it converts"
```

---

### Task 8: `loomux convert`

**Files:**
- Create: `internal/cli/convert.go`, `internal/cli/convert_test.go`
- Modify: `internal/cli/commands.go` (Zeile `convert`; `fetch` kommt mit Task 9)

**Interfaces:**
- Consumes: `convert.Areas`, `convert.ConvertAll`, `convert.ConvertFile` (Task 7), `convert.SystemTools`, `convert.Tools` (Task 6); `parseInterspersed`, `reportReconcileError` (vorhanden); `hosts.FindRoot`, `config.ReadModules`, `config.ManifestPath`; in den Tests `run` (`cli_test.go`), `writeFile(t, path, body) string` (`wiki_test.go:13`, legt auch das Elternverzeichnis an) und `serveFakeOllama` (`cases_4c1_test.go`)
- Produces: `func convertCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int`, `var convertTools = convert.SystemTools()`, `func brainModuleOff(name string, stderr io.Writer) (int, bool)`

- [ ] **Step 1: Den failing test schreiben**

`internal/cli/convert_test.go`:

```go
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/convert"
	"github.com/xidus90/loomux/internal/child"
)

// convertWorld is a state directory with one area `knowledge` whose inbox is
// `00 Eingang`, the process pointed at it and at an empty working directory.
func convertWorld(t *testing.T) (state, inbox string) {
	t.Helper()
	root := t.TempDir()
	state = filepath.Join(root, "state")
	area := filepath.Join(root, "vault")
	inbox = filepath.Join(area, "00 Eingang")
	for _, dir := range []string{state, inbox, filepath.Join(area, ".loomux")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(area, ".loomux", "config.toml"), "[area]\nscope = \"knowledge\"\n\n[layout]\ninbox = \"00 Eingang\"\n")
	writeFile(t, filepath.Join(state, "registry.toml"), "[[area]]\nscope = \"knowledge\"\npath = \""+filepath.ToSlash(area)+"\"\n")
	t.Setenv("LOOMUX_STATE_DIR", state)
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", filepath.Join(root, "legacy"))
	t.Chdir(t.TempDir())
	return state, inbox
}

// noPDFTools stands for a machine whose pdftotext is never asked.
func noPDFTools(t *testing.T) {
	t.Helper()
	saved := convertTools
	t.Cleanup(func() { convertTools = saved })
	convertTools = convert.Tools{
		Look: func(string) (string, error) { t.Fatal("pdftotext was looked up"); return "", nil },
		Run:  func(child.Spec) child.Result { t.Fatal("a program was started"); return child.Result{} },
	}
}

func TestConvertNamesWhatItWroteAndExitsZero(t *testing.T) {
	_, inbox := convertWorld(t)
	noPDFTools(t)
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	code, out, errOut := run("convert")
	if code != 0 || out != filepath.Join(inbox, "video.txt.md")+"\n" || errOut != "" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
	if code, out, _ := run("convert"); code != 0 || out != "" {
		t.Fatalf("the second run: %d %q", code, out)
	}
}

func TestConvertListsWhatIsLeftOnStderrAndExitsOne(t *testing.T) {
	_, inbox := convertWorld(t)
	noPDFTools(t)
	writeFile(t, filepath.Join(inbox, "notiz.txt"), "Nur Prosa.\n")
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	code, out, errOut := run("convert")
	if code != 1 || !strings.HasSuffix(out, "video.txt.md\n") || errOut != "skipped: notiz.txt: no converter knows this format\n" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestConvertOfOneFileNeedsNoRegistry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", filepath.Join(dir, "nowhere"))
	t.Chdir(dir)
	noPDFTools(t)
	writeFile(t, filepath.Join(dir, "video.txt"), "[00:00] Hallo.\n")
	if code, out, errOut := run("convert", "video.txt"); code != 0 || out != "video.txt.md\n" || errOut != "" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestConvertWithoutARegistryIsOneErrorLine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", dir)
	t.Chdir(dir)
	if code, _, errOut := run("convert"); code != 1 || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestConvertStopsAtABrokenModelBlock(t *testing.T) {
	state, inbox := convertWorld(t)
	writeFile(t, filepath.Join(state, "config.toml"), "[model]\nenabled = 5\n")
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	if code, out, errOut := run("convert"); code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestConvertStopsAtABrokenDeclaration(t *testing.T) {
	_, inbox := convertWorld(t)
	writeFile(t, filepath.Join(filepath.Dir(inbox), ".loomux", "config.toml"), "[area]\nscope = \"knowledge\"\n\n[privacy]\nmode = \"cloud\"\n")
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	if code, out, errOut := run("convert"); code != 1 || out != "" || !strings.Contains(errOut, "mode") {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestConvertTakesOnePathAtMost(t *testing.T) {
	convertWorld(t)
	if code, _, errOut := run("convert", "a", "b"); code != 2 || !strings.Contains(errOut, "unrecognized arguments: b") {
		t.Fatalf("%d %q", code, errOut)
	}
	if code, _, _ := run("convert", "--nope"); code != 2 {
		t.Fatal(code)
	}
}

func TestConvertRefusesWhereTheProjectSwitchedTheBrainOff(t *testing.T) {
	convertWorld(t)
	project := t.TempDir()
	os.MkdirAll(filepath.Join(project, ".loomux"), 0o755)
	writeFile(t, filepath.Join(project, ".loomux", "config.toml"), "[modules]\nbrain = false\n")
	t.Chdir(project)
	code, _, errOut := run("convert")
	if code != 1 || !strings.Contains(errOut, "[modules] brain = false") || !strings.Contains(errOut, filepath.Join(project, ".loomux", "config.toml")) {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestConvertReportsAnUnreadableModulesTable(t *testing.T) {
	convertWorld(t)
	project := t.TempDir()
	os.MkdirAll(filepath.Join(project, ".loomux"), 0o755)
	writeFile(t, filepath.Join(project, ".loomux", "config.toml"), "[modules\n")
	t.Chdir(project)
	if code, _, errOut := run("convert"); code != 1 || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestConvertRunsWhereTheProjectKeepsTheBrainOn(t *testing.T) {
	_, inbox := convertWorld(t)
	noPDFTools(t)
	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".loomux", "config.toml"), "[modules]\nbrain = true\n")
	t.Chdir(project)
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	if code, out, errOut := run("convert"); code != 0 || out != filepath.Join(inbox, "video.txt.md")+"\n" || errOut != "" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestConvertOfOneFileListsWhatIsLeft(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", filepath.Join(dir, "nowhere"))
	t.Chdir(dir)
	noPDFTools(t)
	writeFile(t, filepath.Join(dir, "notiz.txt"), "Nur Prosa.\n")
	if code, out, errOut := run("convert", "notiz.txt"); code != 1 || out != "" || errOut != "skipped: notiz.txt: no converter knows this format\n" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

// A suggestion stands on stdout after the written paths, and it is no work
// left for a person: the exit code stays 0.
func TestConvertNamesWhereAWrittenFileBelongs(t *testing.T) {
	state, inbox := convertWorld(t)
	noPDFTools(t)
	writeFile(t, filepath.Join(state, "config.toml"), "[model]\nenabled = true\nendpoint = \"http://127.0.0.1:11435\"\nroles = { place = true }\n")
	writeFile(t, filepath.Join(state, "ollama-fixture.json"), `{"response": "{\"scope\": \"knowledge\", \"grund\": \"Es passt.\"}"}`+"\n")
	serveFakeOllama(t, state)
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	code, out, errOut := run("convert")
	want := filepath.Join(inbox, "video.txt.md") + "\nsuggested: video.txt.md: belongs in knowledge, left in the inbox\n"
	if code != 0 || out != want || errOut != "" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}
```

Die letzten drei Tests decken, was das Tor beim Commit dieses Tasks sonst
ungedeckt fände: den Arm `modules.Brain` in `brainModuleOff`, die
`skipped:`-Zeile einer einzelnen Datei und die Schleife über `suggested:`
(die Fälle aus Task 12 kommen erst später). `writeFile` ist die Hilfe aus
`wiki_test.go`; eine zweite Definition im selben Paket kompilierte nicht.

Run: `go test ./internal/cli/ -run 'TestConvert' -count=1`
Expected: FAIL, `convertTools` und der Befehl fehlen.

- [ ] **Step 2: Implementieren**

`internal/cli/convert.go`:

```go
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/xidus90/loomux/internal/brain/convert"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/hosts"
)

// convertTools is the seam to pdftotext and yt-dlp; the replay of the
// recorded cases answers from the world's fixture instead.
var convertTools = convert.SystemTools()

// convertCommand turns the inboxes, or one named file, into Markdown. It
// writes into an area, so it stands at the top level beside reindex; the
// exit code is 1 as soon as anything is left for a person.
func convertCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux convert", flag.ContinueOnError)
	flags.SetOutput(stderr)
	free, err := parseInterspersed(flags, args)
	if err != nil {
		return 2
	}
	if len(free) > 1 {
		fmt.Fprintf(stderr, "loomux convert: unrecognized arguments: %s\n", strings.Join(free[1:], " "))
		return 2
	}
	if code, off := brainModuleOff("convert", stderr); off {
		return code
	}
	ctx := context.Background()
	var outcome convert.Outcome
	if len(free) == 1 {
		target, message := convert.ConvertFile(ctx, convertTools, free[0])
		if target != "" {
			outcome.Written = []string{target}
		}
		if message != "" {
			outcome.Skipped = []string{message}
		}
	} else {
		lookup := config.NewArtifactLookup()
		areas, err := config.ReadRegistry(lookup.Primary)
		if err != nil {
			return reportReconcileError(stderr, err)
		}
		entries, err := convert.Areas(areas, lookup.Primary, lookup.Fallback)
		if err != nil {
			return reportReconcileError(stderr, err)
		}
		if outcome, err = convert.ConvertAll(ctx, convertTools, entries, lookup.Primary); err != nil {
			return reportReconcileError(stderr, err)
		}
	}
	for _, written := range outcome.Written {
		fmt.Fprintln(stdout, written)
	}
	// Beside the written paths, not among the leftovers: a suggestion is no
	// work waiting.
	for _, suggestion := range outcome.Suggested {
		fmt.Fprintf(stdout, "suggested: %s\n", suggestion)
	}
	for _, message := range outcome.Skipped {
		fmt.Fprintf(stderr, "skipped: %s\n", message)
	}
	if len(outcome.Skipped) > 0 {
		return 1
	}
	return 0
}

// brainModuleOff refuses a command of the brain module where the project
// found above the working directory switched the module off. Outside a
// project nothing is switched off.
func brainModuleOff(name string, stderr io.Writer) (int, bool) {
	root, err := hosts.FindRoot(".")
	if err != nil {
		return 0, false
	}
	modules, err := config.ReadModules(root)
	if err != nil {
		return reportReconcileError(stderr, err), true
	}
	if modules.Brain {
		return 0, false
	}
	fmt.Fprintf(stderr, "loomux %s: the brain module is off in %s ([modules] brain = false)\n", name, config.ManifestPath(root))
	return 1, true
}
```

In `commands.go` die Zeile `"convert": convertCommand,` in alphabetischer
Stelle.

Run: `go test ./internal/cli/ -run 'TestConvert|TestHelp' -count=1`
Expected: PASS.

- [ ] **Step 3: Coverage und Commit**

Run: `go test ./internal/cli/ -coverprofile=cover-cli.out -count=1 && go tool cover -func=cover-cli.out | grep 'internal/cli/convert.go' | grep -v 100.0%`
Expected: keine Zeile: `convertCommand` und `brainModuleOff` zu 100 %.

```bash
git add internal/cli/convert.go internal/cli/convert_test.go internal/cli/commands.go
git commit -m "feat(cli): add loomux convert"
```

---

### Task 9: `loomux fetch`

**Files:**
- Create: `internal/brain/convert/fetch.go`, `internal/brain/convert/fetch_test.go`
- Create: `docs/.superpowers/parity/stufe-4d-orakel/fetch_expected.py`, `testdata/convert/ytdlp/mHSOsy_usAg/expected/…` (erzeugt)
- Modify: `internal/cli/convert.go` (`fetchCommand`), `internal/cli/convert_test.go`, `internal/cli/commands.go`

**Interfaces:**
- Consumes: `pytext.FirstRunes` (Task 4); `Tools`, `writeIfChanged`, `Areas` (Tasks 6–7); die Aufnahme aus Task 1 (Lauf mit `--ignore-errors`: `argv`, `exit` 0, `stderr`, `files/v.de.json3`, `files/v.info.json`)
- Produces: `type Subtitles struct{ Title string; Fragments []Fragment }`, `type Fragment struct{ Mark, Text string }`, `func Fetch(tools Tools, url string) (Subtitles, error)`, `func ToInbox(subs Subtitles, url, inbox string) (string, error)`, `func fetchCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int`

- [ ] **Step 1: Das Ergebnis der Referenz für die Aufnahme erzeugen**

`docs/.superpowers/parity/stufe-4d-orakel/fetch_expected.py` läuft mit dem
Python der Referenz und legt ab, was deren `fetch` und `to_inbox` aus der
Aufnahme aus Task 1 machten:

```python
"""What the reference's fetch writes for the yt-dlp recording of Task 1.

The track is chosen as `_run_yt_dlp` chooses it (fetch.py:118-124); the
download itself is replaced by the recorded file, which is the one network
seam the reference's tests do not reach either. Within a language the
recorded file is yt-dlp's choice, the last json3 track listed, where the
reference downloaded the first; the spec releases that difference (row
"fetch: Spur einer Sprache"), and this oracle does not see it.

Run with the reference's interpreter, on the tag loomux-3-source:
  "C:/Users/micro/Documents/#GIT/ultra-brain/.venv/Scripts/python.exe" \
    docs/.superpowers/parity/stufe-4d-orakel/fetch_expected.py
"""

import json
import shutil
from pathlib import Path

from brain.convert.fetch import fetch, to_inbox

RECORDING = Path(__file__).parents[4] / "testdata" / "convert" / "ytdlp" / "mHSOsy_usAg"
URL = "https://www.youtube.com/watch?v=mHSOsy_usAg"


def runner(url: str) -> tuple[dict[str, str], str]:
    files = RECORDING / "files"
    info = json.loads((files / "v.info.json").read_text(encoding="utf-8"))
    meta = {"title": info.get("title") or "", "upload_date": "", "channel": ""}
    for key in ("subtitles", "automatic_captions"):
        by_lang = info.get(key) or {}
        lang = "de" if by_lang.get("de") else "en"
        tracks = by_lang.get("de") or by_lang.get("en") or []
        if any(track.get("ext") == "json3" for track in tracks):
            return meta, (files / f"v.{lang}.json3").read_text(encoding="utf-8")
    return meta, ""


def main() -> None:
    out = RECORDING / "expected"
    shutil.rmtree(out, ignore_errors=True)
    out.mkdir()
    written = to_inbox(fetch(URL, runner=runner), URL, out)
    (out / "name").write_text(written.name + "\n", encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()
```

Run: `"C:/Users/micro/Documents/#GIT/ultra-brain/.venv/Scripts/python.exe" docs/.superpowers/parity/stufe-4d-orakel/fetch_expected.py`
Expected: unter `expected/` die Datei, die die Referenz geschrieben hätte,
und `name` mit ihrem Namen.

- [ ] **Step 2: Den failing test schreiben**

`internal/brain/convert/fetch_test.go`:

```go
package convert

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

const json3 = `{"events": [
 {"tStartMs": 0, "segs": [{"utf8": "Hallo zusammen, das hier"}]},
 {"tStartMs": 3000, "segs": [{"utf8": "\n"}]},
 {"tStartMs": 4200, "segs": [{"utf8": " ist mein Second Brain."}]}
]}`

// ytdlp stands for yt-dlp: it leaves files in its working directory and
// exits as told, and records the call.
type ytdlp struct {
	files map[string]string
	exit  int
	calls []child.Spec
}

func (y *ytdlp) tools() Tools {
	return Tools{
		Look: func(name string) (string, error) { return `C:\bin\` + name + ".exe", nil },
		Run: func(spec child.Spec) child.Result {
			y.calls = append(y.calls, spec)
			for name, text := range y.files {
				os.WriteFile(filepath.Join(spec.Dir, name), []byte(text), 0o644)
			}
			return child.Result{Code: y.exit, Stderr: "ERROR: something\n"}
		},
	}
}

func info(title string) string {
	return `{"id": "mHSOsy_usAg", "title": "` + title + `", "subtitles": {}, "automatic_captions": {"de": [{"ext": "json3"}]}}`
}

func TestTheFragmentsComeBackWithTheirMarks(t *testing.T) {
	y := &ytdlp{files: map[string]string{"v.info.json": info("Der echte Weg"), "v.de.json3": json3}}
	subs, err := Fetch(y.tools(), "https://www.youtube.com/watch?v=mHSOsy_usAg")
	if err != nil || subs.Title != "Der echte Weg" || !slices.Equal(subs.Fragments, []Fragment{{"00:00:00", "Hallo zusammen, das hier"}, {"00:00:04", "ist mein Second Brain."}}) {
		t.Fatalf("%+v %v", subs, err)
	}
}

func TestYtdlpIsCalledOnceInAFreshDirectory(t *testing.T) {
	y := &ytdlp{files: map[string]string{"v.info.json": info("x"), "v.de.json3": json3}}
	Fetch(y.tools(), "https://www.youtube.com/watch?v=mHSOsy_usAg&list=PL1")
	call := y.calls[0]
	want := []string{`C:\bin\yt-dlp.exe`, "--ignore-config", "--no-playlist", "--no-progress", "--skip-download",
		"--write-subs", "--write-auto-subs", "--sub-langs", "de,en", "--sub-format", "json3", "--write-info-json",
		"--ignore-errors", "-o", "v", "https://www.youtube.com/watch?v=mHSOsy_usAg&list=PL1"}
	if !slices.Equal(call.Argv, want) || call.Timeout != 10*time.Minute {
		t.Fatalf("%q", call.Argv)
	}
	if _, err := os.Stat(call.Dir); err == nil {
		t.Fatal("the working directory outlived the fetch")
	}
}

func TestAMarkPastAHundredMinutesKeepsTheHour(t *testing.T) {
	late := `{"events": [{"tStartMs": 6300000, "segs": [{"utf8": "Spät im Video."}]}]}`
	y := &ytdlp{files: map[string]string{"v.info.json": info("x"), "v.de.json3": late}}
	subs, _ := Fetch(y.tools(), "https://www.youtube.com/watch?v=aaaaaaaaaaa")
	if subs.Fragments[0].Mark != "01:45:00" {
		t.Fatal(subs.Fragments)
	}
}

// Review Focus 4: the English track failed, the German one landed. Under
// --ignore-errors a failed track is a warning and yt-dlp ends with 0
// (measured); an error it still reports after writing ends it with 1, and
// what it wrote decides.
func TestAFailedSecondTrackDoesNotLoseTheFirst(t *testing.T) {
	y := &ytdlp{exit: 1, files: map[string]string{"v.info.json": info("x"), "v.de.json3": json3}}
	if subs, err := Fetch(y.tools(), "https://youtu.be/mHSOsy_usAg"); err != nil || len(subs.Fragments) != 2 {
		t.Fatalf("%+v %v", subs, err)
	}
}

func TestTheTrackIsChosenAsTheReferenceChoosesIt(t *testing.T) {
	json3s := func(langs ...string) map[string][]track {
		out := map[string][]track{}
		for _, lang := range langs {
			out[lang] = []track{{Ext: "json3"}}
		}
		return out
	}
	for _, c := range []struct {
		name    string
		meta    infoJSON
		lang    string
		chosen  bool
	}{
		{"manual de", infoJSON{Subtitles: json3s("de", "en"), Automatic: json3s("de")}, "de", true},
		{"manual en before automatic de", infoJSON{Subtitles: json3s("en"), Automatic: json3s("de")}, "en", true},
		{"automatic de", infoJSON{Automatic: json3s("en", "de")}, "de", true},
		{"automatic en", infoJSON{Automatic: json3s("en")}, "en", true},
		// A language that has tracks and none in json3 ends its kind's search.
		{"manual de without json3", infoJSON{Subtitles: map[string][]track{"de": {{Ext: "vtt"}}, "en": {{Ext: "json3"}}}, Automatic: json3s("en")}, "en", true},
		{"nothing", infoJSON{}, "", false},
	} {
		if lang, ok := chooseTrack(c.meta); lang != c.lang || ok != c.chosen {
			t.Errorf("%s: %q %v", c.name, lang, ok)
		}
	}
}

func TestWithoutASubtitleTrackThereIsNothingToFetch(t *testing.T) {
	for _, files := range []map[string]string{
		{"v.info.json": `{"title": "x", "subtitles": {}, "automatic_captions": {}}`},
		{"v.info.json": info("x"), "v.de.vtt": "WEBVTT"},
		{"v.info.json": info("x"), "v.de.json3": "  \n"},
	} {
		_, err := Fetch((&ytdlp{files: files}).tools(), "https://www.youtube.com/watch?v=aaaaaaaaaaa")
		if err == nil || !strings.Contains(err.Error(), "no subtitle track to fetch, and this system does no ASR") {
			t.Errorf("%v: %v", files, err)
		}
	}
}

func TestWhatYtdlpLeavesUnreadableIsAnError(t *testing.T) {
	for _, files := range []map[string]string{
		{},
		{"v.info.json": "not json"},
		{"v.info.json": info("x"), "v.de.json3": "[1, 2]"},
	} {
		if _, err := Fetch((&ytdlp{exit: 1, files: files}).tools(), "https://x"); err == nil {
			t.Errorf("%v passed", files)
		}
	}
}

// A chosen track that is there and does not read is an error of its own,
// not "no subtitle track": that answer is kept for a track yt-dlp never
// wrote.
func TestAChosenTrackThatDoesNotReadIsAnError(t *testing.T) {
	tools := Tools{
		Look: func(name string) (string, error) { return name, nil },
		Run: func(spec child.Spec) child.Result {
			os.WriteFile(filepath.Join(spec.Dir, "v.info.json"), []byte(info("x")), 0o644)
			os.Mkdir(filepath.Join(spec.Dir, "v.de.json3"), 0o755)
			return child.Result{}
		},
	}
	_, err := Fetch(tools, "https://www.youtube.com/watch?v=aaaaaaaaaaa")
	if err == nil || !strings.Contains(err.Error(), "v.de.json3 cannot be read") {
		t.Fatal(err)
	}
}

func TestAMissingYtdlpNamesItsInstaller(t *testing.T) {
	tools := Tools{Look: func(string) (string, error) { return "", exec.ErrNotFound }}
	_, err := Fetch(tools, "https://x")
	if err == nil || !strings.Contains(err.Error(), "yt-dlp is not on PATH; install it with: winget install --id yt-dlp.yt-dlp -e") {
		t.Fatal(err)
	}
}

func TestTheFileLandsInTheInboxInBracketForm(t *testing.T) {
	inbox := filepath.Join(t.TempDir(), "00 Eingang")
	subs := Subtitles{Title: "Der echte Weg", Fragments: []Fragment{{"00:00:00", "Hallo."}, {"00:00:04", "Weiter."}}}
	path, err := ToInbox(subs, "https://www.youtube.com/watch?v=mHSOsy_usAg", inbox)
	if err != nil || filepath.Base(path) != "Der echte Weg (mHSOsy_usAg).txt" || read(t, path) != "[00:00:00] Hallo.\n\n[00:00:04] Weiter.\n" {
		t.Fatalf("%q %v", path, err)
	}
	again, _ := ToInbox(subs, "https://www.youtube.com/watch?v=mHSOsy_usAg", inbox)
	if entries, _ := os.ReadDir(inbox); again != path || len(entries) != 1 {
		t.Fatal("the second fetch did not reuse the name")
	}
}

func TestTheTitleBecomesAUsableName(t *testing.T) {
	inbox := t.TempDir()
	one := []Fragment{{"00:00", "Text"}}
	for _, c := range []struct{ title, url, want string }{
		{"Was: RAG / Wiki?", "https://www.youtube.com/watch?v=mHSOsy_usAg", "Was RAG  Wiki (mHSOsy_usAg).txt"},
		{"???:::", "https://www.youtube.com/watch?v=mHSOsy_usAg", "video (mHSOsy_usAg).txt"},
		{"x", "https://www.youtube.com/shorts/mHSOsy_usAg", "x (mHSOsy_usAg).txt"},
		{"x", "https://www.youtube.com/live/mHSOsy_usAg", "x (mHSOsy_usAg).txt"},
		{"x", "https://vimeo.com/123", "x.txt"},
		{strings.Repeat("ä", 300), "https://vimeo.com/1", strings.Repeat("ä", 150) + ".txt"},
	} {
		path, err := ToInbox(Subtitles{Title: c.title, Fragments: one}, c.url, inbox)
		if err != nil || filepath.Base(path) != c.want {
			t.Errorf("%q: %q %v", c.title, filepath.Base(path), err)
		}
	}
}

// The recording of Task 1, fed through Fetch and ToInbox, writes what the
// reference's fetch and to_inbox wrote for it. The recording was made with
// the command line Fetch runs, --ignore-errors included.
func TestTheRecordingLandsAsTheReferenceWroteIt(t *testing.T) {
	recording := filepath.Join("..", "..", "..", "testdata", "convert", "ytdlp", "mHSOsy_usAg")
	argv := strings.TrimSpace(read(t, filepath.Join(recording, "argv")))
	if want := "yt-dlp " + strings.Join(ytdlpArgs("https://www.youtube.com/watch?v=mHSOsy_usAg"), " "); argv != want {
		t.Fatalf("the recording ran %q, Fetch runs %q", argv, want)
	}
	files := map[string]string{}
	entries, err := os.ReadDir(filepath.Join(recording, "files"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		files[e.Name()] = read(t, filepath.Join(recording, "files", e.Name()))
	}
	subs, err := Fetch((&ytdlp{files: files}).tools(), "https://www.youtube.com/watch?v=mHSOsy_usAg")
	if err != nil {
		t.Fatal(err)
	}
	path, err := ToInbox(subs, "https://www.youtube.com/watch?v=mHSOsy_usAg", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimSpace(read(t, filepath.Join(recording, "expected", "name")))
	if filepath.Base(path) != name || read(t, path) != read(t, filepath.Join(recording, "expected", name)) {
		t.Fatalf("%q differs from the reference's %q", filepath.Base(path), name)
	}
}
```

`"Was: RAG / Wiki?"` wird ohne `:`, `/` und `?` zu `"Was RAG  Wiki"` — zwei
Leerzeichen, wie die Referenz sie stehen lässt (sie entfernt die Zeichen,
sie ersetzt sie nicht).

Run: `go test ./internal/brain/convert/ -run 'Fragments|Ytdlp|Mark|Track|Subtitle|Unreadable|Inbox|Title|Recording' -count=1`
Expected: FAIL, `Fetch`, `ToInbox`, `Subtitles`, `Fragment`, `track`,
`infoJSON`, `chooseTrack`, `ytdlpArgs` fehlen.

- [ ] **Step 3: Implementieren**

`internal/brain/convert/fetch.go`:

```go
package convert

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/programs"
)

// fetchTimeout bounds one yt-dlp call: extraction and two subtitle
// downloads on a slow line.
const fetchTimeout = 10 * time.Minute

// maxStem keeps a title from an arbitrary portal below Windows' 255
// characters, with room for the id and the extension (fetch.py:74-78).
const maxStem = 150

var watchID = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?:v=|youtu\.be/|shorts/|live/)([0-9A-Za-z_-]{11})`)
})

// unsafeName is what Windows will not have in a file name, and every
// control character.
var unsafeName = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`) })

// Subtitles is what a fetch brings back: the title names the file, the
// fragments fill it.
type Subtitles struct {
	Title     string
	Fragments []Fragment
}

// Fragment is one piece of speech and the mark it began at, hh:mm:ss.
type Fragment struct{ Mark, Text string }

type track struct {
	Ext string `json:"ext"`
}

type infoJSON struct {
	Title     string             `json:"title"`
	Subtitles map[string][]track `json:"subtitles"`
	Automatic map[string][]track `json:"automatic_captions"`
}

// Fetch has yt-dlp write a video's subtitles into a fresh directory and
// reads the language the reference would have chosen. Within a language
// yt-dlp picks the track, the last json3 one listed, where the reference
// took the first -- a released deviation. loomux itself never speaks to the
// network: what the portal asks of a client, yt-dlp keeps up with.
func Fetch(tools Tools, url string) (Subtitles, error) {
	exe, err := tools.Look("yt-dlp")
	if err != nil {
		return Subtitles{}, fmt.Errorf("yt-dlp is not on PATH; install it with: %s", programs.Install("yt-dlp"))
	}
	dir, err := os.MkdirTemp("", "loomux-fetch-")
	if err != nil {
		return Subtitles{}, err
	}
	defer os.RemoveAll(dir)
	res := tools.Run(child.Spec{Argv: append([]string{exe}, ytdlpArgs(url)...), Dir: dir, Timeout: fetchTimeout})
	// The exit code decides nothing. Under --ignore-errors a failed track
	// (429 on an auto-translated one, measured) is a warning and the info
	// JSON is still written; an error yt-dlp reports after writing ends it
	// with 1, and what it wrote may be the one to take.
	data, err := os.ReadFile(filepath.Join(dir, "v.info.json"))
	if err != nil {
		last := strings.TrimSpace(res.Stderr)
		if i := strings.LastIndex(last, "\n"); i >= 0 {
			last = last[i+1:]
		}
		return Subtitles{}, fmt.Errorf("%s: yt-dlp found no video (exit %d): %s", url, res.Code, last)
	}
	var meta infoJSON
	if err := json.Unmarshal(data, &meta); err != nil {
		return Subtitles{}, fmt.Errorf("%s: yt-dlp wrote an unreadable v.info.json: %v", url, err)
	}
	// A track yt-dlp did not write (another format came back, or none) is no
	// track; one that is there and does not read is an error of its own.
	var raw []byte
	if lang, ok := chooseTrack(meta); ok {
		name := "v." + lang + ".json3"
		raw, err = os.ReadFile(filepath.Join(dir, name))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return Subtitles{}, fmt.Errorf("%s: yt-dlp's %s cannot be read: %v", url, name, err)
		}
	}
	if pytext.Strip(string(raw)) == "" {
		return Subtitles{}, fmt.Errorf("%s: no subtitle track to fetch, and this system does no ASR", url)
	}
	fragments, err := parseJSON3(raw)
	if err != nil {
		return Subtitles{}, fmt.Errorf("%s: the subtitle track is no json3: %v", url, err)
	}
	return Subtitles{Title: meta.Title, Fragments: fragments}, nil
}

// ytdlpArgs is the command line the recording under testdata/convert/ytdlp
// ran, flag for flag and in its order.
func ytdlpArgs(url string) []string {
	return []string{"--ignore-config", "--no-playlist", "--no-progress", "--skip-download",
		"--write-subs", "--write-auto-subs", "--sub-langs", "de,en", "--sub-format", "json3", "--write-info-json",
		"--ignore-errors", "-o", "v", url}
}

// chooseTrack is the reference's order (fetch.py:118-124): manual tracks
// before automatic ones, de before en within each -- and a language that
// has tracks but none in json3 ends its kind's search. It picks a language;
// which of its json3 tracks the file holds, yt-dlp decided.
func chooseTrack(meta infoJSON) (string, bool) {
	for _, byLang := range []map[string][]track{meta.Subtitles, meta.Automatic} {
		lang, tracks := "de", byLang["de"]
		if len(tracks) == 0 {
			lang, tracks = "en", byLang["en"]
		}
		if slices.ContainsFunc(tracks, func(t track) bool { return t.Ext == "json3" }) {
			return lang, true
		}
	}
	return "", false
}

// parseJSON3 reads json3, not vtt: the vtt track scrolls and repeats every
// line. An event of whitespace alone is no fragment.
func parseJSON3(raw []byte) ([]Fragment, error) {
	var doc struct {
		Events []struct {
			TStartMs float64 `json:"tStartMs"`
			Segs     []struct {
				UTF8 string `json:"utf8"`
			} `json:"segs"`
		} `json:"events"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	var out []Fragment
	for _, event := range doc.Events {
		var text strings.Builder
		for _, seg := range event.Segs {
			text.WriteString(seg.UTF8)
		}
		if t := pytext.Strip(text.String()); t != "" {
			out = append(out, Fragment{Mark: mark(int64(event.TStartMs)), Text: t})
		}
	}
	return out, nil
}

// mark is hh:mm:ss, not mm:ss: past minute 100 the bracket mark would stop
// matching, and the rest of the video would melt into one paragraph.
func mark(milliseconds int64) string {
	total := milliseconds / 1000
	return fmt.Sprintf("%02d:%02d:%02d", total/3600, total%3600/60, total%60)
}

// ToInbox files the fragments in bracket form, the form a person writes by
// hand, under the video's title.
func ToInbox(subs Subtitles, url, inbox string) (string, error) {
	stem := pytext.Strip(pytext.FirstRunes(pytext.Strip(unsafeName().ReplaceAllString(subs.Title, "")), maxStem))
	if stem == "" {
		stem = "video"
	}
	name := stem + ".txt"
	if m := watchID().FindStringSubmatch(url); m != nil {
		name = fmt.Sprintf("%s (%s).txt", stem, m[1])
	}
	lines := make([]string, len(subs.Fragments))
	for i, f := range subs.Fragments {
		lines[i] = "[" + f.Mark + "] " + f.Text
	}
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(inbox, name)
	if _, err := writeIfChanged(target, strings.Join(lines, "\n\n")+"\n"); err != nil {
		return "", err
	}
	return target, nil
}
```

In `internal/cli/convert.go`:

```go
// fetchCommand puts a video's subtitles into an area's inbox. The area must
// be writable: a read-only area takes no file, a fetched one included.
func fetchCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux fetch", flag.ContinueOnError)
	flags.SetOutput(stderr)
	scope := flags.String("scope", "knowledge", "which area's inbox receives the file")
	free, err := parseInterspersed(flags, args)
	if err != nil {
		return 2
	}
	if len(free) != 1 {
		fmt.Fprintln(stderr, "loomux fetch: expected one URL")
		return 2
	}
	if code, off := brainModuleOff("fetch", stderr); off {
		return code
	}
	lookup := config.NewArtifactLookup()
	areas, err := config.ReadRegistry(lookup.Primary)
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	entries, err := convert.Areas(areas, lookup.Primary, lookup.Fallback)
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	i := slices.IndexFunc(entries, func(a convert.Area) bool { return a.Scope == *scope })
	switch {
	case i < 0:
		return reportReconcileError(stderr, fmt.Errorf("no area named %s in the registry", pytext.Repr(*scope)))
	case entries[i].Inbox == "":
		return reportReconcileError(stderr, fmt.Errorf("area %s declares no inbox; add `[layout] inbox = ...` to its manifest", pytext.Repr(*scope)))
	case entries[i].ReadOnly:
		return reportReconcileError(stderr, fmt.Errorf("area %s is read-only; fetch writes into no read-only area", pytext.Repr(*scope)))
	}
	subs, err := convert.Fetch(convertTools, free[0])
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	target, err := convert.ToInbox(subs, free[0], entries[i].Inbox)
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	fmt.Fprintln(stdout, target)
	return 0
}
```

Dazu die Imports `slices` und `internal/brain/pytext`, und in `commands.go`
`"fetch": fetchCommand,`.

In `convert_test.go` dazu:

```go
func fakeYtdlp(t *testing.T, files map[string]string) {
	t.Helper()
	saved := convertTools
	t.Cleanup(func() { convertTools = saved })
	convertTools = convert.Tools{
		Look: func(name string) (string, error) { return name, nil },
		Run: func(spec child.Spec) child.Result {
			for name, text := range files {
				writeFile(t, filepath.Join(spec.Dir, name), text)
			}
			return child.Result{}
		},
	}
}

func TestFetchWritesIntoTheScopedInbox(t *testing.T) {
	_, inbox := convertWorld(t)
	fakeYtdlp(t, map[string]string{
		"v.info.json": `{"title": "Der echte Weg", "subtitles": {"de": [{"ext": "json3"}]}}`,
		"v.de.json3":  `{"events": [{"tStartMs": 0, "segs": [{"utf8": "Hallo."}]}]}`,
	})
	code, out, errOut := run("fetch", "https://www.youtube.com/watch?v=mHSOsy_usAg", "--scope", "knowledge")
	want := filepath.Join(inbox, "Der echte Weg (mHSOsy_usAg).txt")
	if code != 0 || out != want+"\n" || errOut != "" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestFetchRefusesAnAreaItCannotWriteInto(t *testing.T) {
	state, _ := convertWorld(t)
	bare := t.TempDir()
	corpus := t.TempDir()
	os.MkdirAll(filepath.Join(state, "areas", "corpus", ".loomux"), 0o755)
	writeFile(t, filepath.Join(state, "areas", "corpus", ".loomux", "config.toml"), "[area]\nscope = \"corpus\"\n\n[layout]\ninbox = \"00 Eingang\"\n")
	registry := filepath.Join(state, "registry.toml")
	data, _ := os.ReadFile(registry)
	writeFile(t, registry, string(data)+"\n[[area]]\nscope = \"project/x\"\npath = \""+filepath.ToSlash(bare)+"\"\n\n[[area]]\nscope = \"corpus\"\npath = \""+filepath.ToSlash(corpus)+"\"\nreadonly = true\n")
	fakeYtdlp(t, nil)
	for scope, want := range map[string]string{
		"knowledge2": "no area named 'knowledge2' in the registry",
		"project/x":  "area 'project/x' declares no inbox",
		"corpus":     "area 'corpus' is read-only",
	} {
		if code, _, errOut := run("fetch", "https://x", "--scope", scope); code != 1 || !strings.Contains(errOut, want) {
			t.Errorf("%s: %d %q", scope, code, errOut)
		}
	}
}

func TestFetchNeedsExactlyOneURL(t *testing.T) {
	convertWorld(t)
	for _, args := range [][]string{{"fetch"}, {"fetch", "a", "b"}, {"fetch", "--nope", "a"}} {
		if code, _, _ := run(args...); code != 2 {
			t.Errorf("%q: %d", args, code)
		}
	}
}

func TestFetchReportsWhatYtdlpCouldNotDo(t *testing.T) {
	convertWorld(t)
	fakeYtdlp(t, map[string]string{})
	if code, _, errOut := run("fetch", "https://x"); code != 1 || !strings.Contains(errOut, "yt-dlp found no video") {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestFetchRefusesWhereTheProjectSwitchedTheBrainOff(t *testing.T) {
	convertWorld(t)
	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".loomux", "config.toml"), "[modules]\nbrain = false\n")
	t.Chdir(project)
	fakeYtdlp(t, nil)
	if code, _, errOut := run("fetch", "https://x"); code != 1 || !strings.Contains(errOut, "loomux fetch: the brain module is off") {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestFetchWithoutARegistryIsOneErrorLine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", dir)
	t.Chdir(dir)
	fakeYtdlp(t, nil)
	if code, _, errOut := run("fetch", "https://x"); code != 1 || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestFetchStopsAtABrokenDeclaration(t *testing.T) {
	_, inbox := convertWorld(t)
	writeFile(t, filepath.Join(filepath.Dir(inbox), ".loomux", "config.toml"), "[area]\nscope = \"knowledge\"\n\n[privacy]\nmode = \"cloud\"\n")
	fakeYtdlp(t, nil)
	if code, _, errOut := run("fetch", "https://x"); code != 1 || !strings.Contains(errOut, "mode") {
		t.Fatalf("%d %q", code, errOut)
	}
}

// A file where the inbox should be: ToInbox cannot make the directory.
func TestFetchReportsAnInboxItCannotWriteInto(t *testing.T) {
	_, inbox := convertWorld(t)
	if err := os.RemoveAll(inbox); err != nil {
		t.Fatal(err)
	}
	writeFile(t, inbox, "not a directory\n")
	fakeYtdlp(t, map[string]string{
		"v.info.json": `{"title": "Der echte Weg", "subtitles": {"de": [{"ext": "json3"}]}}`,
		"v.de.json3":  `{"events": [{"tStartMs": 0, "segs": [{"utf8": "Hallo."}]}]}`,
	})
	if code, out, errOut := run("fetch", "https://www.youtube.com/watch?v=mHSOsy_usAg"); code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}
```

Die vier letzten Tests decken die Zweige von `fetchCommand`, die sonst erst
das Tor fände: Modul Brain aus, keine Registry, eine kaputte Deklaration und
ein Eingang, der sich nicht anlegen lässt.

Die Welt für `corpus` legt die Deklaration dorthin, wo ein
schreibgeschützter Bereich sie hat (`<zustand>/areas/<scope>`); sonst hätte
er keinen Eingang, und der Test prüfte die falsche Weigerung.

Run: `go test ./internal/brain/convert/ ./internal/cli/ -run 'Fetch|Fragments|Ytdlp|Mark|Track|Subtitle|Unreadable|Inbox|Title|Recording' -count=1`
Expected: PASS.

- [ ] **Step 4: Coverage und Commit**

Run: `go test ./internal/brain/convert/ -coverprofile=cover.out -count=1 && go tool cover -func=cover.out | grep -v 100.0%`
Expected: nichts unter 100 %. Offen bleiben können die Fehlerzweige von
`os.MkdirTemp` in `Fetch` und `os.MkdirAll` in `ToInbox`; `ToInbox` deckt
ein Eingang unter einer Datei (`inbox = <datei>/x`), `Fetch` ein `TMP`, das
auf eine Datei zeigt (`t.Setenv("TMP", datei)`, unter Windows liest
`os.TempDir` `TMP`).

Run: `go test ./internal/cli/ -coverprofile=cover-cli.out -count=1 && go tool cover -func=cover-cli.out | grep 'internal/cli/convert.go' | grep -v 100.0%`
Expected: keine Zeile. Das Tor misst 100 % je Funktion schon beim Commit
dieses Tasks, `fetchCommand` eingeschlossen.

```bash
git add internal/brain/convert/fetch.go internal/brain/convert/fetch_test.go internal/cli/convert.go internal/cli/convert_test.go internal/cli/commands.go docs/.superpowers/parity/stufe-4d-orakel/fetch_expected.py testdata/convert/ytdlp
git commit -m "feat(fetch): put a video's subtitles into an inbox through yt-dlp"
```

---

### Task 10: Der Wächter verweigert einem Agenten `convert` und `fetch`

**Files:**
- Modify: `internal/hooks/guard.go:268` (Begründung), `guard.go:455-484` (`wordsWriteConfiguration`, zwei Fälle)
- Modify: `internal/hooks/guard_test.go` (`TestTheGuardRefusesCommandsThatWriteTheConfiguration`, `TestCheckToolNamesTheConfigurationReason`)
- Modify: `docs/en/cli-reference.md:220-235`, `docs/de/cli-reference.md:226-240` (die Liste der verweigerten Befehle)
- Modify: `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md` („Die Wächterregel“: ein Satz mit Verweis auf Vorschlag 4)

**Interfaces:**
- Consumes: B11

- [ ] **Step 1: Den failing test schreiben**

In `TestTheGuardRefusesCommandsThatWriteTheConfiguration` in die Liste
`refused`:

```go
		"loomux convert",
		"loomux convert x.pdf",
		`loomux convert "C:\vault\00 Eingang\x.pdf"`,
		"loomux fetch https://www.youtube.com/watch?v=mHSOsy_usAg --scope knowledge",
		"go run ./cmd/loomux convert",
		`& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" fetch https://x`,
		"loomux convert --help x",
```

und in `allowed`:

```go
		"loomux convert --help",
		"loomux convert -h",
		"loomux fetch --help",
		"loomux fetch -h",
```

In `TestCheckToolNamesTheConfigurationReason` dazu:

```go
	got = checkTool(t.TempDir(), "Bash", map[string]any{"command": "loomux convert"}, config.Policy{})
	if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "convert and fetch write into an area's inbox") }) {
		t.Fatalf("convert reasons %v", got)
	}
```

Run: `go test ./internal/hooks/ -run 'TestTheGuardRefusesCommandsThatWriteTheConfiguration|TestCheckToolNamesTheConfigurationReason' -count=1`
Expected: FAIL, `convert` und `fetch` werden durchgelassen.

- [ ] **Step 2: Implementieren**

In `wordsWriteConfiguration` vor `case "area":`:

```go
	case "convert", "fetch":
		// Both write into an area's inbox, which the write barrier keeps
		// from agents; a lone --help or -h only reads.
		return len(args) != 2 || (args[1] != "--help" && args[1] != "-h")
```

Die Begründung in `guard.go:268` wird:

```go
reasons = append(reasons, "loomux init, config and area add write the configuration the guard reads, merge-hook install and remove write executable hooks into repositories, and convert and fetch write into an area's inbox, which the write barrier keeps from agents; a human runs them. An agent proposes a change with `loomux config set|unset … --propose`, which a human applies")
```

Der Kommentar über `writesConfiguration` („a loomux command that writes
.loomux/config.toml or the global config in-process“) bekommt „or an
area's inbox“.

Run: `go test ./internal/hooks/ -count=1`
Expected: PASS.

- [ ] **Step 3: Doku und Commit**

In beiden `cli-reference.md` in der Aufzählung der verweigerten Befehle
(`hook pre-tool-use`) `loomux convert` und `loomux fetch` (außer allein mit
`--help`/`-h`) ergänzen, dazu den neuen Wortlaut der Begründung. In der
Stufe-4-Spec unter „Die Wächterregel“: „Seit 4d auch `convert` und `fetch`
(Vorschlag 4 in „Abweichungen beim Planen von 4d“).“

```bash
git add internal/hooks docs/en/cli-reference.md docs/de/cli-reference.md docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md
git commit -m "feat(guard): refuse convert and fetch to an agent"
```

---

### Task 11: Die Hinweise aller fremden Teile gehen mit dem Release

Entschieden am 2026-09-26: `NOTICE.md` trägt nicht nur die Zipf-Tabelle,
sondern jedes fremde Stück, das im Binary landet. Gemessen am selben Tag
(`go list -deps ./cmd/loomux`, `go tool nm`): 14 fremde Module —
`BurntSushi/toml` (über `third_party/toml`, `COPYING`), `google/jsonschema-go`,
`modelcontextprotocol/go-sdk`, `odvcencio/gotreesitter`, `segmentio/asm`,
`segmentio/encoding`, `yosida95/uritemplate/v3`, `golang.org/x/{oauth2,sync,sys,term,text,time}`
(mit `PATENTS`), `gopkg.in/yaml.v3` (mit `NOTICE`) —, dazu die
Standardbibliothek von Go (`LICENSE`, `PATENTS` unter `GOROOT`). Von den 206
Grammatiken, die `gotreesitter/grammars/grammar_blobs` einbettet, bleibt nach
dem Linken nur `python` (tree-sitter-python, MIT, Max Brunsfeld) im Binary;
die übrigen entfernt der Linker, darunter eine unter GPL-3.0. Welche
Grammatik im Binary steht, entscheidet das Paket
`gotreesitter/grammars/<name>`, das loomux importiert.

**Files:**
- Create: `internal/dev/notices/notices.go`, `notices_test.go` (der Erzeuger)
- Create: `internal/notices/notices.go`, `notices_test.go`, `NOTICE.md` (erzeugt, eingebettet)
- Modify: `internal/cli/dev.go`, `internal/cli/dev_test.go` (`dev notices`)
- Modify: `internal/release/build.go:35-66`, `internal/release/build_test.go`

Die Spec trägt die Entscheidung schon (Zeile `NOTICE.md` unter „Mit dem
Nutzer entschieden“, Vorschlag 13); dieser Task ändert sie nicht.

**Interfaces:**
- Consumes: `model.ZipfNotice()` (Task 2)
- Produces: `notices.Text() string`; im Erzeuger (importiert als `devnotices`) `Render(root string, run Runner) (string, error)`, `GoCommand(dir string, args ...string) ([]byte, error)` und `type Runner func(dir string, args ...string) ([]byte, error)`; `loomux dev notices [--out <datei>]`

- [ ] **Step 1: Den failing test für den Erzeuger schreiben**

`internal/dev/notices/notices_test.go`:

```go
package notices

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGo answers go list and go env from a scripted module world.
type fakeGo struct {
	root    string
	goroot  string
	listErr error
	envErr  error
	listOut string
}

func (f *fakeGo) run(dir string, args ...string) ([]byte, error) {
	switch strings.Join(args, " ") {
	case "env GOROOT":
		return []byte(f.goroot + "\n"), f.envErr
	case "list -deps -json ./cmd/loomux":
		if f.listErr != nil || f.listOut != "" {
			return []byte(f.listOut), f.listErr
		}
		mod := filepath.ToSlash(filepath.Join(f.root, "mod"))
		ts := filepath.ToSlash(filepath.Join(f.root, "ts"))
		return []byte(`{"ImportPath": "fmt", "Standard": true}
{"ImportPath": "example.com/a", "Module": {"Path": "example.com/a", "Version": "v1.2.3", "Dir": "` + mod + `"}}
{"ImportPath": "example.com/a/sub", "Module": {"Path": "example.com/a", "Version": "v1.2.3", "Dir": "` + mod + `"}}
{"ImportPath": "github.com/odvcencio/gotreesitter/grammars/python", "Module": {"Path": "github.com/odvcencio/gotreesitter", "Version": "v0.55.0", "Dir": "` + ts + `"}}
{"ImportPath": "github.com/odvcencio/gotreesitter/grammars/runtime", "Module": {"Path": "github.com/odvcencio/gotreesitter", "Version": "v0.55.0", "Dir": "` + ts + `"}}
{"ImportPath": "github.com/xidus90/loomux/cmd/loomux", "Module": {"Path": "github.com/xidus90/loomux", "Main": true}}
`), nil
	}
	return nil, errors.New("unexpected " + strings.Join(args, " "))
}

func world(t *testing.T) *fakeGo {
	t.Helper()
	root := t.TempDir()
	write := func(rel, text string) {
		path := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go/LICENSE", "Go license\r\n")
	write("go/PATENTS", "Go patents\n")
	write("mod/LICENSE", "A license\n")
	write("mod/README.md", "not a license\n")
	write("ts/LICENSE", "TS license\n")
	write("ts/licenses/texts/MIT.txt", "MIT text\n")
	write("ts/licenses/notices/python-NOTICE.txt", "Python grammar notice\n")
	write("ts/licenses/grammars.json", `{"entries": [
 {"name": "python", "repo": "https://github.com/tree-sitter/tree-sitter-python", "ref": "abc", "spdx": "MIT", "copyright_holders": ["Copyright (c) 2016 Max Brunsfeld"], "notice_file": "notices/python-NOTICE.txt", "copyleft": false},
 {"name": "gpl", "repo": "r", "ref": "x", "spdx": "GPL-3.0", "copyright_holders": [], "copyleft": true}
]}`)
	return &fakeGo{root: root, goroot: filepath.Join(root, "go")}
}

func TestRenderNamesEveryLinkedPieceOnce(t *testing.T) {
	text, err := Render("repo", world(t).run)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"## Go standard library and runtime\n", "Go license\n", "Go patents\n",
		"## example.com/a v1.2.3\n", "A license\n",
		"## github.com/odvcencio/gotreesitter v0.55.0\n", "TS license\n",
		"## tree-sitter grammar python\n", "https://github.com/tree-sitter/tree-sitter-python@abc", "Copyright (c) 2016 Max Brunsfeld", "MIT text\n", "Python grammar notice\n",
		"# Third-party notice: German word frequencies",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Count(text, "## example.com/a ") != 1 || strings.Contains(text, "not a license") || strings.Contains(text, "\r") || strings.Contains(text, "grammar gpl") {
		t.Fatal(text)
	}
}

func TestRenderRefusesALinkedCopyleftGrammar(t *testing.T) {
	f := world(t)
	os.WriteFile(filepath.Join(f.root, "ts", "licenses", "grammars.json"), []byte(`{"entries": [{"name": "python", "spdx": "GPL-3.0", "copyleft": true}]}`), 0o644)
	if _, err := Render("repo", f.run); err == nil || !strings.Contains(err.Error(), "copyleft") {
		t.Fatal(err)
	}
}

func TestRenderRefusesWhatItCannotRead(t *testing.T) {
	for name, spoil := range map[string]func(f *fakeGo){
		"go list fails":     func(f *fakeGo) { f.listErr = errors.New("no go") },
		"go list garbles":   func(f *fakeGo) { f.listOut = "{" },
		"go env fails":      func(f *fakeGo) { f.envErr = errors.New("no env") },
		"no Go license":     func(f *fakeGo) { os.Remove(filepath.Join(f.goroot, "LICENSE")) },
		"no module license": func(f *fakeGo) { os.Remove(filepath.Join(f.root, "mod", "LICENSE")) },
		"no grammar entry":  func(f *fakeGo) { os.WriteFile(filepath.Join(f.root, "ts", "licenses", "grammars.json"), []byte(`{"entries": []}`), 0o644) },
		"broken grammars":   func(f *fakeGo) { os.WriteFile(filepath.Join(f.root, "ts", "licenses", "grammars.json"), []byte(`{`), 0o644) },
		"no license text":   func(f *fakeGo) { os.Remove(filepath.Join(f.root, "ts", "licenses", "texts", "MIT.txt")) },
		"no notice file":    func(f *fakeGo) { os.Remove(filepath.Join(f.root, "ts", "licenses", "notices", "python-NOTICE.txt")) },
		"no grammar audit":  func(f *fakeGo) { os.Remove(filepath.Join(f.root, "ts", "licenses", "grammars.json")) },
		"no module dir":     func(f *fakeGo) { os.RemoveAll(filepath.Join(f.root, "mod")) },
	} {
		f := world(t)
		spoil(f)
		if _, err := Render("repo", f.run); err == nil {
			t.Errorf("%s: rendered", name)
		}
	}
}

// A license file that is listed but cannot be read stops the notice rather
// than leaving its section short.
func TestRenderRefusesALicenseFileItCannotRead(t *testing.T) {
	saved := readFile
	t.Cleanup(func() { readFile = saved })
	readFile = func(path string) ([]byte, error) {
		if filepath.Base(path) == "PATENTS" {
			return nil, os.ErrPermission
		}
		return saved(path)
	}
	if _, err := Render("repo", world(t).run); !errors.Is(err, os.ErrPermission) {
		t.Fatal(err)
	}
}
```

Run: `go test ./internal/dev/notices/ -count=1`
Expected: FAIL, `Render` fehlt.

- [ ] **Step 2: Den Erzeuger implementieren**

`internal/dev/notices/notices.go`:

```go
// Package notices writes NOTICE.md: the license of every third-party piece
// the loomux binary links -- Go's standard library, each module, each
// tree-sitter grammar whose package is imported -- and the notice of the
// embedded word frequencies. A test holds the committed file to what this
// renders, so a new dependency cannot ship without its notice.
package notices

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/brain/model"
)

// Runner runs the go command in dir.
type Runner func(dir string, args ...string) ([]byte, error)

// GoCommand runs the real go command.
func GoCommand(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	return cmd.Output()
}

const gotreesitter = "github.com/odvcencio/gotreesitter"

// noticeFile is every file a notice quotes; licenseFile is the one a
// section cannot do without.
var noticeFile = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)^(licen[cs]e|copying|notice|patents)(\.[a-z]+)?$`)
})

var licenseFile = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?i)^(licen[cs]e|copying)`) })

// readFile is the seam a test replaces to see a listed license file that
// does not read; neither chmod nor a missing file makes one under Windows.
var readFile = os.ReadFile

type pkg struct {
	ImportPath string
	Standard   bool
	Module     *struct {
		Path, Version, Dir string
		Main               bool
	}
}

type grammar struct {
	Name             string   `json:"name"`
	Repo             string   `json:"repo"`
	Ref              string   `json:"ref"`
	SPDX             string   `json:"spdx"`
	CopyrightHolders []string `json:"copyright_holders"`
	NoticeFile       string   `json:"notice_file"`
	Copyleft         bool     `json:"copyleft"`
}

// Render is NOTICE.md for the binary built from root.
func Render(root string, run Runner) (string, error) {
	out, err := run(root, "list", "-deps", "-json", "./cmd/loomux")
	if err != nil {
		return "", fmt.Errorf("go list: %w", err)
	}
	modules := map[string]pkg{}
	var grammars []string
	standard := false
	for decoder := json.NewDecoder(bytes.NewReader(out)); ; {
		var p pkg
		if err := decoder.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return "", fmt.Errorf("go list: %w", err)
		}
		switch {
		case p.Standard:
			standard = true
		case p.Module == nil || p.Module.Main:
		default:
			modules[p.Module.Path] = p
			if name, ok := strings.CutPrefix(p.ImportPath, gotreesitter+"/grammars/"); ok && !strings.Contains(name, "/") &&
				name != "runtime" && name != "grammar_blobs" {
				grammars = append(grammars, name)
			}
		}
	}
	var b strings.Builder
	b.WriteString("# Third-party notices\n\nThe loomux binary links the third-party software and data below. `loomux dev notices` writes this file; do not edit it.\n")
	if standard {
		goroot, err := run(root, "env", "GOROOT")
		if err != nil {
			return "", fmt.Errorf("go env: %w", err)
		}
		if err := section(&b, "Go standard library and runtime", strings.TrimSpace(string(goroot))); err != nil {
			return "", err
		}
	}
	paths := make([]string, 0, len(modules))
	for path := range modules {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		m := modules[path].Module
		if err := section(&b, m.Path+" "+m.Version, m.Dir); err != nil {
			return "", err
		}
	}
	if len(grammars) > 0 {
		if err := grammarSections(&b, modules[gotreesitter].Module.Dir, grammars); err != nil {
			return "", err
		}
	}
	b.WriteString("\n" + model.ZipfNotice())
	return b.String(), nil
}

// section is one heading and every license file in dir, verbatim.
func section(b *strings.Builder, heading, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	fmt.Fprintf(b, "\n## %s\n", heading)
	found := false
	for _, e := range entries {
		if e.IsDir() || !noticeFile().MatchString(e.Name()) {
			continue
		}
		text, err := readFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "\n### %s\n\n```\n%s\n```\n", e.Name(), clean(text))
		found = found || licenseFile().MatchString(e.Name())
	}
	if !found {
		return fmt.Errorf("%s: no license file in %s", heading, dir)
	}
	return nil
}

// grammarSections is each linked grammar from gotreesitter's own license
// audit, refusing a copyleft one: its terms would reach the whole binary.
func grammarSections(b *strings.Builder, dir string, names []string) error {
	data, err := os.ReadFile(filepath.Join(dir, "licenses", "grammars.json"))
	if err != nil {
		return err
	}
	var audit struct {
		Entries []grammar `json:"entries"`
	}
	if err := json.Unmarshal(data, &audit); err != nil {
		return fmt.Errorf("grammars.json: %w", err)
	}
	slices.Sort(names)
	for _, name := range slices.Compact(names) {
		i := slices.IndexFunc(audit.Entries, func(g grammar) bool { return g.Name == name })
		if i < 0 {
			return fmt.Errorf("grammar %s: not in gotreesitter's license audit", name)
		}
		g := audit.Entries[i]
		if g.Copyleft {
			return fmt.Errorf("grammar %s: %s is copyleft and would bind the whole binary", name, g.SPDX)
		}
		text, err := os.ReadFile(filepath.Join(dir, "licenses", "texts", g.SPDX+".txt"))
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "\n## tree-sitter grammar %s\n\n%s@%s, %s\n\n%s\n\n```\n%s\n```\n", name, g.Repo, g.Ref, g.SPDX,
			strings.Join(g.CopyrightHolders, "\n"), clean(text))
		if g.NoticeFile != "" {
			notice, err := os.ReadFile(filepath.Join(dir, "licenses", filepath.FromSlash(g.NoticeFile)))
			if err != nil {
				return err
			}
			fmt.Fprintf(b, "\n```\n%s\n```\n", clean(notice))
		}
	}
	return nil
}

func clean(text []byte) string {
	return strings.TrimRight(strings.ReplaceAll(string(text), "\r\n", "\n"), "\n")
}
```

`notice_file` in `grammars.json` ist ein Pfad; ob relativ zu `licenses/`,
zeigt ein Blick in einen Eintrag, der ihn trägt (`elixir`, `pkl`). Stimmt die
Annahme nicht, wird die Verknüpfung in `grammarSections` angepasst. So oder
so bekommt der Test einen Eintrag mit `notice_file` (die Datei unter
`ts/licenses/notices/`), damit der Zweig gedeckt ist. `GoCommand` deckt ein
Test, der `GoCommand(".", "version")` ruft und `go version` in der Ausgabe
erwartet.

Run: `go test ./internal/dev/notices/ -count=1`
Expected: PASS.

- [ ] **Step 3: Befehl, eingebettete Datei und Wache**

In `internal/cli/dev.go` der Eintrag `"notices": devNotices` und der Import
`devnotices "github.com/xidus90/loomux/internal/dev/notices"` (zwei Pakete
heißen `notices`; der Erzeuger bekommt überall diesen Namen):

```go
// devNotices writes NOTICE.md from the modules and grammars the binary
// links. Whether the committed file is current, a test says.
func devNotices(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev notices", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", filepath.Join("internal", "notices", "NOTICE.md"), "the file to write")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	text, err := devnotices.Render(".", devnotices.GoCommand)
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	if err := os.WriteFile(*out, []byte(text), 0o644); err != nil {
		return reportReconcileError(stderr, err)
	}
	fmt.Fprintln(stdout, *out)
	return 0
}
```

`internal/notices/notices.go`:

```go
// Package notices carries NOTICE.md, the licenses of every third-party piece
// in the binary, for the release to ship beside it.
package notices

import _ "embed"

//go:embed NOTICE.md
var text string

// Text is NOTICE.md.
func Text() string { return text }
```

`internal/notices/notices_test.go`:

```go
package notices

import (
	"testing"

	devnotices "github.com/xidus90/loomux/internal/dev/notices"
)

// TestTheNoticeIsCurrent fails when a dependency, a grammar or the Zipf
// notice changed and NOTICE.md was not written again: run loomux dev notices.
func TestTheNoticeIsCurrent(t *testing.T) {
	want, err := devnotices.Render("../..", devnotices.GoCommand)
	if err != nil {
		t.Fatal(err)
	}
	if Text() != want {
		t.Fatal("internal/notices/NOTICE.md is stale; run loomux dev notices")
	}
}
```

Run: `go run ./cmd/loomux dev notices && go test ./internal/notices/ ./internal/dev/notices/ ./internal/cli/ -run 'Notice|Notices' -count=1`
Expected: `internal/notices/NOTICE.md` geschrieben, PASS. Die Datei lesen:
14 Module, Go, `tree-sitter grammar python`, der Zipf-Hinweis.

Für `devNotices` in `dev_test.go`, aus der Wurzel des Repos
(`t.Chdir("../..")`): ein Schreiben nach `--out` in einem temporären
Verzeichnis (Exit 0, Pfad auf stdout, Inhalt gleich `notices.Text()`), ein
unschreibbares `--out` (Exit 1) und ein unbekanntes Flag (Exit 2); ein
Fehler von `Render` über ein Arbeitsverzeichnis ohne `go.mod`
(`t.Chdir(t.TempDir())`), Exit 1.

- [ ] **Step 4: Das Release legt `NOTICE.md` hinaus**

Test in `build_test.go`:

```go
func TestBuildShipsTheNoticeBesideTheBinaries(t *testing.T) {
	out := t.TempDir()
	names, err := Build("1.2.3", "", out, func(env []string, args ...string) error {
		return os.WriteFile(args[len(args)-2], []byte("binary"), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if names[len(names)-2] != "NOTICE.md" || names[len(names)-1] != "SHA256SUMS" {
		t.Fatalf("%q", names)
	}
	notice, err := os.ReadFile(filepath.Join(out, "NOTICE.md"))
	if err != nil || string(notice) != notices.Text() {
		t.Fatal("NOTICE.md is not the embedded notice")
	}
	sums, _ := os.ReadFile(filepath.Join(out, "SHA256SUMS"))
	sum := sha256.Sum256(notice)
	if !strings.Contains(string(sums), hex.EncodeToString(sum[:])+"  NOTICE.md\n") {
		t.Fatalf("SHA256SUMS: %s", sums)
	}
}

func TestBuildStopsWhenTheNoticeCannotBeWritten(t *testing.T) {
	out := t.TempDir()
	_, err := Build("1.2.3", "", out, func(env []string, args ...string) error {
		os.MkdirAll(filepath.Join(out, "NOTICE.md"), 0o755)
		return os.WriteFile(args[len(args)-2], []byte("binary"), 0o644)
	})
	if err == nil {
		t.Fatal("built without a notice")
	}
}
```

Imports: `crypto/sha256`, `encoding/hex`, `os`, `path/filepath`, `strings`,
`internal/notices`. Der vorhandene Test, der die Namensliste vergleicht,
bekommt `NOTICE.md` vor `SHA256SUMS`.

In `Build` nach der Schleife über die Ziele, vor `SHA256SUMS`:

```go
	// Every third-party license the binary carries goes out beside it; a
	// notice in the source tree never reaches whoever downloads a release.
	notice := []byte(notices.Text())
	if err := os.WriteFile(filepath.Join(out, "NOTICE.md"), notice, 0o644); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(notice)
	fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(sum[:]), "NOTICE.md")
	names = append(names, "NOTICE.md")
```

Run: `go test ./internal/release/ ./internal/selfupdate/ ./internal/cli/ -count=1`
Expected: PASS. `selfupdate` liest aus `SHA256SUMS` nur die Zeile seines
Binarys; `TestSelfupdateStaysBelowItsCallers` bleibt grün.

- [ ] **Step 5: Commit**

```bash
git add internal/dev/notices internal/notices internal/cli/dev.go internal/cli/dev_test.go internal/release
git commit -m "build(release): ship every third-party license the binary links"
```

---

### Task 12: Parität — die Fälle gegen die Referenz

**Files:**
- Create: `docs/.superpowers/parity/stufe-4d-orakel/record.sh`, `record_all.sh`, `pypdf_pages.py`, `make_worlds.py`
- Create: `testdata/cases/4d-worlds/<welt>/…`, `testdata/cases/4d-map.toml`
- Create: `testdata/cases/4d-source/convert/<name>/…` (aufgezeichnet), `testdata/cases/4d/convert/<name>/…` (importiert)
- Create: `internal/cli/cases_4d_test.go`
- Modify: `docs/.superpowers/parity/stufe-4d.md`

**Interfaces:**
- Consumes: die PDFs aus Task 1; `dev record-case`, `dev import-cases`, `dev fake-ollama`, `serveFakeOllama` (`cases_4c1_test.go`), `faketool.Load`, `(*faketool.Fixture).Match`

- [ ] **Step 1: Welten bauen**

`make_worlds.py` (PEP 723 ohne Abhängigkeiten) legt je Fall eine Welt unter
`testdata/cases/4d-worlds/` an: `registry.toml` in der Weltwurzel (für beide
Seiten der Zustand, 4c-1 B7), ein Bereich `knowledge` unter `vault/` mit
`.brain.toml` (`[area] scope = "knowledge"`, `[layout] inbox = "00 Eingang"`)
und die Dateien des Falls im Eingang; PDFs werden aus
`testdata/convert/pdf/` kopiert. Die Tabelle ist die Fallliste:

| Fall | Eingang | Besonderes | Exit |
|---|---|---|---|
| `transcript-bracket` | `video (mHSOsy_usAg).txt` mit Klammermarken | — | 0 |
| `transcript-range-lead-in` | `export.txt`: Titelzeile, dann Bereichszeilen, eine über eine Stunde | — | 0 |
| `crlf-transcript` | `video.txt` mit CRLF | — | 0 |
| `mixed-case-order` | `B.txt`, `a.txt`, `_z.txt` | — | 0 |
| `same-stem` | `doku.pdf` (`text.pdf`), `doku.txt` | — | 0 |
| `unsupported-and-good` | `notiz.txt` (Prosa), `video.txt` | — | 1 |
| `hand-written-target` | `video.txt`, `video.txt.md` von Hand | — | 1 |
| `broken-utf8-source` | `kaputt.txt` (saubere 8 300 Bytes, dann `\xff\xfe`), `video.txt` | — | 1 |
| `broken-utf8-target` | `video.txt`, `video.txt.md` aus `\xff\xfe` | — | 1 |
| `pdf-text` | `buch.pdf` (`text.pdf`) | — | 0 |
| `pdf-umlaut-name` | `Bericht März.pdf` | — | 0 |
| `pdf-scan` | `scan.pdf` (`blank.pdf`), `video.txt` | — | 1 |
| `pdf-pageless` | `leer.pdf` (`pageless.pdf`) | — | 1 |
| `pdf-partial` | `buch.pdf` (`mixed.pdf`) | — | 1 |
| `pdf-corrupt` | `buch.pdf` (`corrupt.pdf`), `video.txt` | — | 1 |
| `no-inbox` | — | Bereich ohne `[layout] inbox` | 0 |
| `readonly-area` | `video.txt` | `readonly = true`; die Deklaration unter `areas/knowledge/.brain.toml` der Welt (dem Zustand), wo beide Seiten sie für einen `readonly`-Bereich lesen, keine unter `vault/` | 0 |
| `single-file` | `video.txt` | Befehl `brain-mcp convert "{{WORLD}}/vault/00 Eingang/video.txt"` | 0 |
| `no-registry` | — | keine `registry.toml` | 1 |
| `broken-manifest` | `video.txt` | zweiter Bereich mit `[privacy] mode = "cloud"` | 1 |
| `describe-kept` | `video.txt` | `config.toml`: Modell an, Endpoint `127.0.0.1:11435`, `roles = { describe = true }`; Fixture: guter Satz | 0 |
| `describe-refused` | `video.txt` | wie oben; Fixture: englischer Satz | 0 |
| `second-run` | `video.txt`, `video.txt.md`, wie ein erster Lauf mit Modell ihn schrieb (Satz, `retrieved: 2020-01-01`) | wie `describe-kept`; Fixture: ein anderer guter Satz | 0 |
| `place-suggested` | `video.txt` | `roles = { place = true }`; zweiter Bereich `project/x` ohne Eingang; Fixture `{"scope": "project/x", "grund": "Es passt."}` | 0 |
| `place-unknown-scope` | `video.txt` | wie oben; Fixture mit `project/erfunden` | 0 |
| `model-unreachable` | `video.txt` | Endpoint `127.0.0.1:11436` | 0 |
| `model-off-in-area` | `video.txt` | global an, im Bereich `[model] enabled = false`; Fixture da | 0 |
| `broken-model-block` | `video.txt` | `[model] enabled = 5` | 1 |
| `endpoint-off-loopback` | `video.txt` | Endpoint `http://192.0.2.1:11434` | 1 |

`second-run` ist der „zweite Lauf“ aus Vorschlag 10 der Spec, so weit eine
gestellte Welt ihn trägt: Sie behält keine Änderungszeit (B8), `retrieved:`
ist auf beiden Seiten der Tag des Laufs, und der feste Tag im Ziel lässt
beide den Kopf neu schreiben — stdout nennt auf beiden Seiten die Zieldatei.
Ein vergangener Tag, nicht der Tag der Aufnahme: Sonst schriebe die Referenz
nichts und ein späteres Abspielen doch. Was der Fall an der Referenz misst:
Das Ziel gilt als eigenes, der stehende Satz bleibt, und das Modell wird
nicht gefragt (`ollama calls: 0`), obwohl `describe` an ist und die Fixture
einen anderen Satz hätte. Dass ein zweiter Lauf gar nichts schreibt, halten
die Go-Tests (`TestTheSecondRunWritesNothing`,
`TestAStandingSentenceIsNeverAskedFor`, `TestConvertNamesWhatItWroteAndExitsZero`).

`pypdf_pages.py` (PEP 723, `pypdf==6.16.2`) schreibt in jede Welt mit PDF
eine `faketool.json` für die Go-Seite: je PDF die Antwort auf
`pdftotext -layout -enc UTF-8 -eol unix <name> -` mit dem Text, den `pypdf` der
Referenz liefert, Seite für Seite mit `\f` danach, und die Antwort auf
`pdftotext -v`:

```python
# /// script
# requires-python = ">=3.12"
# dependencies = ["pypdf==6.16.2"]
# ///
"""The pdftotext seam: what the reference's pypdf reads from each PDF, in the form pdftotext prints.

Run: uv run --script docs/.superpowers/parity/stufe-4d-orakel/pypdf_pages.py
"""

import json
from pathlib import Path

from pypdf import PdfReader
from pypdf.errors import PyPdfError

WORLDS = Path(__file__).parents[4] / "testdata" / "cases" / "4d-worlds"
VERSION = "pdftotext version 25.07.0 (seam: the reference reads with pypdf 6.16.2)\nThe Poppler Developers\n"


def answer(pdf: Path) -> dict[str, object]:
    prefix = f"pdftotext -layout -enc UTF-8 -eol unix {pdf.name} -"
    try:
        pages = [page.extract_text(extraction_mode="layout") or "" for page in PdfReader(pdf).pages]
    except PyPdfError:
        return {"prefix": prefix, "exit": 1, "stdout": ""}
    return {"prefix": prefix, "exit": 0, "stdout": "".join(page + "\f" for page in pages)}


def main() -> None:
    for world in sorted(WORLDS.iterdir()):
        pdfs = sorted(world.rglob("*.pdf"))
        if not pdfs:
            continue
        answers = [{"prefix": "pdftotext -v", "exit": 0, "stdout": VERSION}, *(answer(pdf) for pdf in pdfs)]
        (world / "faketool.json").write_text(json.dumps({"answers": answers}, ensure_ascii=False, indent=1) + "\n", encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()
```

`make_worlds.py` schreibt die Welten der Tabelle; die Transkripte sind die
der Referenztests (`tests/convert/test_cli_convert.py`), die Sätze der
Fixtures die von `test_local_describe.py` und `test_local_place.py`:

```python
# /// script
# requires-python = ">=3.12"
# ///
"""Builds testdata/cases/4d-worlds: one world per recorded convert case.

Run: uv run --script docs/.superpowers/parity/stufe-4d-orakel/make_worlds.py
"""

import json
import shutil
from pathlib import Path

ROOT = Path(__file__).parents[4]
WORLDS = ROOT / "testdata" / "cases" / "4d-worlds"
PDFS = ROOT / "testdata" / "convert" / "pdf"

REGISTRY = '[[area]]\nscope = "knowledge"\npath = "{{WORLD}}/vault"\n'
SECOND = '\n[[area]]\nscope = "project/x"\npath = "{{WORLD}}/x"\n'
MANIFEST = '[area]\nscope = "knowledge"\n\n[layout]\ninbox = "00 Eingang"\n'
MODEL = '[model]\nenabled = true\nendpoint = "http://127.0.0.1:11435"\n'
HALLO = "[00:00] Hallo zusammen.\n"
GOOD = "Der Bericht beschreibt die Abnahme der zweiten Scheibe."
# What a first run with describe on wrote for HALLO, on a day long past: the
# staged world keeps no modification time, so retrieved: is the run's day and
# both sides write the head again -- with the standing sentence, unasked.
FIRST_RUN = (
    "---\nsource_url:\nretrieved: 2020-01-01\nconverter: brain-transcript/1\nasr: true\n"
    f"description: {GOOD}\n---\n\n{HALLO}"
)


def text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8", newline="\n")


def world(name, files, *, registry=REGISTRY, manifest=MANIFEST, state=None, fixture=None, extra=None):
    """files maps an inbox name to text, to bytes, or to a PDF under testdata/convert/pdf."""
    root = WORLDS / name
    shutil.rmtree(root, ignore_errors=True)
    inbox = root / "vault" / "00 Eingang"
    inbox.mkdir(parents=True)
    if registry is not None:
        text(root / "registry.toml", registry)
    if manifest is not None:
        text(root / "vault" / ".brain.toml", manifest)
    for file, content in files.items():
        if isinstance(content, Path):
            shutil.copyfile(content, inbox / file)
        elif isinstance(content, bytes):
            (inbox / file).write_bytes(content)
        else:
            text(inbox / file, content)
    if state is not None:
        text(root / "config.toml", state)
    if fixture is not None:
        text(root / "ollama-fixture.json", json.dumps(fixture, ensure_ascii=False) + "\n")
    for rel, content in (extra or {}).items():
        text(root / rel, content)


def main() -> None:
    world("transcript-bracket", {"video (mHSOsy_usAg).txt": "[00:00] Hallo zusammen, das hier ist mein\n\n[00:04] Second Brain mit 2000 Notizen.\n"})
    world("transcript-range-lead-in", {"export.txt": "Titel des Videos\n\n00:01:05 - 00:01:57\nHallo zusammen.\n\n01:02:03 - 01:02:44\nSpät im Video.\n"})
    world("crlf-transcript", {"video.txt": b"00:00:00 - 00:00:05\r\nHallo zusammen.\r\n\r\n00:00:06 - 00:00:09\r\nUnd weiter.\r\n"})
    world("mixed-case-order", {"B.txt": "[00:00] Zwei.\n", "a.txt": "[00:00] Eins.\n", "_z.txt": "[00:00] Null.\n"})
    world("same-stem", {"doku.pdf": PDFS / "text.pdf", "doku.txt": "[00:00] Aus dem Transkript.\n"})
    world("unsupported-and-good", {"notiz.txt": "Nur Prosa.\n", "video.txt": HALLO})
    world("hand-written-target", {"video.txt": HALLO, "video.txt.md": "---\ntitle: Meine Notiz\n---\n\nHandarbeit.\n"})
    world("broken-utf8-source", {"kaputt.txt": b"[00:00] Anfang ist sauber.\n" + b"x" * 8300 + b"\n" + b"\xff\xfe" * 50, "video.txt": HALLO})
    world("broken-utf8-target", {"video.txt": HALLO, "video.txt.md": b"\xff\xfe" * 8192})
    world("pdf-text", {"buch.pdf": PDFS / "text.pdf"})
    world("pdf-umlaut-name", {"Bericht März.pdf": PDFS / "Bericht März.pdf"})
    world("pdf-scan", {"scan.pdf": PDFS / "blank.pdf", "video.txt": HALLO})
    world("pdf-pageless", {"leer.pdf": PDFS / "pageless.pdf"})
    world("pdf-partial", {"buch.pdf": PDFS / "mixed.pdf"})
    world("pdf-corrupt", {"buch.pdf": PDFS / "corrupt.pdf", "video.txt": HALLO})
    world("no-inbox", {}, manifest='[area]\nscope = "knowledge"\n')
    # A readonly area's declaration lies in the state directory, where both
    # sides look for it (registry.manifest_path); under vault/ it would leave
    # the area without an inbox, and the case would pass for that reason.
    world("readonly-area", {"video.txt": HALLO}, registry=REGISTRY + "readonly = true\n",
          manifest=None, extra={"areas/knowledge/.brain.toml": MANIFEST})
    world("single-file", {"video.txt": HALLO})
    world("no-registry", {"video.txt": HALLO}, registry=None)
    world("broken-manifest", {"video.txt": HALLO}, registry=REGISTRY + SECOND,
          extra={"x/.brain.toml": '[area]\nscope = "project/x"\n\n[privacy]\nmode = "cloud"\n'})
    describe = MODEL + "roles = { describe = true }\n"
    world("describe-kept", {"video.txt": HALLO}, state=describe, fixture={"response": GOOD})
    world("describe-refused", {"video.txt": HALLO}, state=describe, fixture={"response": "The report describes the second slice."})
    world("second-run", {"video.txt": HALLO, "video.txt.md": FIRST_RUN}, state=describe,
          fixture={"response": "Das Video begrüßt die Zuschauer zum Auftakt der Reihe."})
    place = MODEL + "roles = { place = true }\n"
    world("place-suggested", {"video.txt": HALLO}, registry=REGISTRY + SECOND, state=place,
          fixture={"response": '{"scope": "project/x", "grund": "Es passt."}'}, extra={"x/.keep": ""})
    world("place-unknown-scope", {"video.txt": HALLO}, registry=REGISTRY + SECOND, state=place,
          fixture={"response": '{"scope": "project/erfunden", "grund": "Es passt."}'}, extra={"x/.keep": ""})
    world("model-unreachable", {"video.txt": HALLO}, state=MODEL.replace("11435", "11436"))
    world("model-off-in-area", {"video.txt": HALLO}, manifest=MANIFEST + "\n[model]\nenabled = false\n",
          state=MODEL, fixture={"response": GOOD})
    world("broken-model-block", {"video.txt": HALLO}, state="[model]\nenabled = 5\n")
    world("endpoint-off-loopback", {"video.txt": HALLO}, state='[model]\nenabled = true\nendpoint = "http://192.0.2.1:11434"\n')


if __name__ == "__main__":
    main()
```

Run: `uv run --script docs/.superpowers/parity/stufe-4d-orakel/make_worlds.py && uv run --script docs/.superpowers/parity/stufe-4d-orakel/pypdf_pages.py`
Expected: 29 Welten; eine `faketool.json` in jeder Welt mit PDF.

- [ ] **Step 2: Aufzeichnen**

`record.sh` entsteht aus dem von 4c-1, mit Welten aus `4d-worlds` und
Ausgabe nach `4d-source`; der Kopfkommentar nennt danach Stufe 4d:

```bash
sed 's/4c1-worlds/4d-worlds/g; s/4c1-source/4d-source/g' docs/.superpowers/parity/stufe-4c-1-orakel/record.sh > docs/.superpowers/parity/stufe-4d-orakel/record.sh
```

`record_all.sh`:

```bash
#!/usr/bin/env bash
# Zeichnet alle Fälle von 4d auf, einen nach dem anderen; record.sh beendet
# die Ollama-Attrappe vor dem nächsten.
set -uo pipefail
S="$(cd "$(dirname "$0")" && pwd)"
R() { bash "$S/record.sh" "$@"; }
C="brain-mcp convert"

R convert/transcript-bracket transcript-bracket "" "a bracket transcript with a YouTube id in its name" "$C"
R convert/transcript-range-lead-in transcript-range-lead-in "" "a range transcript with a lead-in and a mark past the hour" "$C"
R convert/crlf-transcript crlf-transcript "" "a transcript saved with CRLF" "$C"
R convert/mixed-case-order mixed-case-order "" "B.txt, a.txt and _z.txt: the order of the written paths" "$C"
R convert/same-stem same-stem "" "doku.pdf and doku.txt both survive" "$C"
R convert/unsupported-and-good unsupported-and-good "" "prose is left behind, the transcript beside it converts" "$C"
R convert/hand-written-target hand-written-target "" "a target without a converter line is never overwritten" "$C"
R convert/broken-utf8-source broken-utf8-source "" "broken bytes past the sample fail the read, not the run" "$C"
R convert/broken-utf8-target broken-utf8-target "" "a target that is no UTF-8 is left untouched" "$C"
R convert/pdf-text pdf-text "" "a text PDF converts" "$C"
R convert/pdf-umlaut-name pdf-umlaut-name "" "a PDF with an umlaut and a space in its name" "$C"
R convert/pdf-scan pdf-scan "" "a scan is left behind, the transcript converts" "$C"
R convert/pdf-pageless pdf-pageless "" "a PDF without pages gets its own message" "$C"
R convert/pdf-partial pdf-partial "" "three text pages convert, seven scans are reported" "$C"
R convert/pdf-corrupt pdf-corrupt "" "a PDF that does not parse is left behind" "$C"
R convert/no-inbox no-inbox "" "an area without an inbox is skipped silently" "$C"
R convert/readonly-area readonly-area "" "a readonly area is never written into" "$C"
R convert/single-file single-file "" "one named file, no sweep over the registry" "$C \"{{WORLD}}/vault/00 Eingang/video.txt\""
R convert/no-registry no-registry "" "no registry is one error line" "$C"
R convert/broken-manifest broken-manifest "" "a broken declaration stops the run before a file is written" "$C"
R convert/describe-kept describe-kept "" "the model's sentence lands in the head" "$C"
R convert/describe-refused describe-refused "" "an English sentence is refused, the head keeps four lines" "$C"
R convert/second-run second-run "" "a second run: the target is ours, its sentence stands and the model is not asked" "$C"
R convert/place-suggested place-suggested "" "the suggestion names project/x and moves nothing" "$C"
R convert/place-unknown-scope place-unknown-scope "" "a scope the register does not know is no suggestion" "$C"
R convert/model-unreachable model-unreachable "" "nobody listens on the endpoint: no sentence, exit 0" "$C"
R convert/model-off-in-area model-off-in-area "" "the area switched the model off: nobody is asked" "$C"
R convert/broken-model-block broken-model-block "" "a [model] enabled that is no boolean stops the run" "$C"
R convert/endpoint-off-loopback endpoint-off-loopback "" "an endpoint off the loopback stops the run before a file is written" "$C"
```

Vorher `git -C "C:/Users/micro/Documents/#GIT/ultra-brain" rev-parse HEAD`
gegen `3cc72d2` lesen, dann das Arbeitsverzeichnis der Aufzeichnung stellen
(ein kurzer Pfad, MAX_PATH):

```bash
R4="$LOCALAPPDATA/Temp/r4d" && rm -rf "$R4" && mkdir -p "$R4/bin" "$R4/fakeqmd"
go build -o "$R4/bin/loomux.exe" ./cmd/loomux
go build -o "$R4/fakeqmd/qmd.exe" ./internal/dev/fakeqmd/_qmd
```

Run: `RECORD_DIR="$LOCALAPPDATA/Temp/r4d" bash docs/.superpowers/parity/stufe-4d-orakel/record_all.sh`
Expected: 29 Verzeichnisse unter `testdata/cases/4d-source/convert/`,
Exit-Codes wie in der Tabelle; `ollama calls: 1` bei `describe-kept`,
`describe-refused`, `place-suggested`, `place-unknown-scope`, sonst `0`
(auch bei `second-run`).

- [ ] **Step 3: Importieren**

`testdata/cases/4d-map.toml`:

```toml
# convert's cases; a recorded manifest is moved, not decoded and encoded again.
manifests = "verbatim"

[[command]]
from = "brain-mcp convert"
to   = "loomux convert"
```

Run: `bin/loomux.exe dev import-cases --map testdata/cases/4d-map.toml --from testdata/cases/4d-source --to testdata/cases/4d`
Expected: 29 Fälle unter `testdata/cases/4d/convert/`.

- [ ] **Step 4: Abspieltest schreiben und laufen lassen**

`internal/cli/cases_4d_test.go`:

```go
package cli

import (
	"bytes"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/xidus90/loomux/internal/brain/convert"
	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/dev/faketool"
)

// wantCases4d is pinned, not merely non-zero: a partial import must not pass
// as parity.
const wantCases4d = 29

// wantOllamaCalls4d are the requests the reference sent in each recording.
var wantOllamaCalls4d = map[string]int{
	"convert/describe-kept":       1,
	"convert/describe-refused":    1,
	"convert/place-suggested":     1,
	"convert/place-unknown-scope": 1,
}

// expected4d are the mismatches a replay has to report, exactly, per case.
var expected4d = map[string][]string{}

// retrievedDay is compiled on first use, as every regexp in loomux is.
var retrievedDay = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?m)^retrieved: \d{4}-\d{2}-\d{2}$`)
})

// normalize4d holds both sides to what two runs on two days cannot share:
// the day a staged world's files were written, and the PDF converter's
// count, which pdftotext raised from 1 to 2 (a released deviation).
func normalize4d(_, tree map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(tree))
	for name, data := range tree {
		data = retrievedDay().ReplaceAll(data, []byte("retrieved: {{DAY}}"))
		out[name] = bytes.ReplaceAll(data, []byte("converter: brain-pdf/1\n"), []byte("converter: brain-pdf/2\n"))
	}
	return out
}

// useFakePdftotext answers pdftotext from the world's faketool.json, which
// carries what the reference's pypdf read; a world without one has none.
func useFakePdftotext(t *testing.T, world string) {
	t.Helper()
	fixture, err := faketool.Load(filepath.Join(world, faketool.FixtureName))
	if err != nil {
		t.Fatal(err)
	}
	saved := convertTools
	t.Cleanup(func() { convertTools = saved })
	convertTools = convert.Tools{
		Look: func(name string) (string, error) {
			if len(fixture.Answers) == 0 {
				return "", exec.ErrNotFound
			}
			return name, nil
		},
		Run: func(spec child.Spec) child.Result {
			answer, ok := fixture.Match(spec.Argv)
			if !ok {
				return child.Result{Code: 127, Stderr: "faketool: no answer\n"}
			}
			return child.Result{Code: answer.Exit, Stdout: answer.Stdout}
		},
	}
}

func TestCases4d(t *testing.T) {
	corpus, err := filepath.Abs(filepath.Join("..", "..", "testdata", "cases", "4d"))
	if err != nil {
		t.Fatal(err)
	}
	all, err := cases.DiscoverCases(corpus, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases4d {
		t.Fatalf("found %d cases, want %d", len(all), wantCases4d)
	}
	for _, c := range all {
		name := c.Verb + "/" + c.Name
		t.Run(name, func(t *testing.T) {
			var calls *callLog
			outcome, err := cases.RunCaseWith(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", dir)
				t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
				useFakePdftotext(t, dir)
				calls = serveFakeOllama(t, dir)
				return Run(args, stdin, stdout, stderr)
			}, normalize4d)
			if err != nil {
				t.Fatal(err)
			}
			got := slices.DeleteFunc(slices.Clone(outcome.Mismatches), func(m string) bool { return strings.HasPrefix(m, "stderr:") })
			wanted := slices.Clone(expected4d[name])
			slices.Sort(got)
			slices.Sort(wanted)
			if !slices.Equal(got, wanted) {
				t.Fatalf("mismatches differ from the expected ones\ngot:\n%s\nwant:\n%s\nstdout:\n%s",
					strings.Join(got, "\n"), strings.Join(wanted, "\n"), outcome.ActualStdout)
			}
			if n := strings.Count(calls.String(), "\n"); n != wantOllamaCalls4d[name] {
				t.Fatalf("the fake Ollama got %d requests, the reference sent %d:\n%s", n, wantOllamaCalls4d[name], calls.String())
			}
		})
	}
}
```

`t.Chdir(dir)` stellt die Welt als Arbeitsverzeichnis; eine Welt trägt keine
`.loomux/config.toml` in ihrer Wurzel, `brainModuleOff` findet also kein
Projekt — es sei denn, eine Welt oberhalb von `%TEMP%` hätte eines; dann
zeigt es der erste Lauf.

Run: `go test ./internal/cli/ -run TestCases4d -count=1 -v`
Expected: PASS bis auf erwartete Abweichungen. Jede wird gelesen und in die
Akte eingetragen; eine Abweichung außerhalb von stderr, `retrieved:` und
`converter: brain-pdf/…` ist ein Fehler aus Task 5 bis 8 und wird dort
behoben.

- [ ] **Step 5: Akte schreiben**

`docs/.superpowers/parity/stufe-4d.md` nach dem Muster von
`stufe-4c-1.md`: Referenz und Tag, Messungen (Task 1; die Abschnitte
„Messungen“ und „Widersprüche zum Plan“ stehen schon und bleiben), die 29
Fälle mit Ergebnis, die Batterien (Task 2 und 3) mit ihren Ausnahmen, und die
Abweichungsliste:

| Abweichung | Art | Begründung |
|---|---|---|
| PDF über `pdftotext` statt `pypdf` | freigegeben 2026-09-26 | Entscheidung „PDF“; die Fälle messen über die Naht bei den Seiten, die Poppler-Goldens (Task 13) das echte Werkzeug |
| `converter: brain-pdf/2` | freigegeben 2026-09-26 | Vorschlag 8; in den Fällen angeglichen |
| Meldung für eine unlesbare PDF | Meldungstext | `pdftotext exited …` statt der Ausnahme von `pypdf`; stderr wird nicht verglichen |
| Nur Poppler | freigegeben 2026-09-26 | Vorschlag 12 |
| `fetch` über das Programm `yt-dlp`, das die Datei schreibt | freigegeben 2026-09-26 | Entscheidung „fetch“; Randfall einer manuellen Spur ohne json3 |
| `fetch`: innerhalb einer Sprache die letzte json3-Spur (Wahl von yt-dlp), die Referenz nahm die erste | freigegeben 2026-09-26 | Zeile „fetch: Spur einer Sprache“; für `mHSOsy_usAg` die deutsche Spracherkennung statt der Übersetzung aus `en-US`; das Orakel von Task 9 verdeckt es, weil es die aufgenommene Datei an die Stelle des Downloads setzt |
| `fetch` mit `--no-playlist`, `--ignore-config` und `--ignore-errors` | Entscheidung „fetch“, E7 | eine Adresse mit `&list=` holt nur das Video; eine scheiternde Spur (429) beendet yt-dlp nicht vor der Info-JSON |
| `fetch` in einen `readonly`-Bereich verweigert | freigegeben 2026-09-26 | Vorschlag 2, geheilte Lücke |
| `convert`/`fetch` verweigern bei `[modules] brain = false` | Spec | Modul Brain |
| `readsBack` über yaml.v3 | Plan E4 | die Sätze aus `readsBackDeviations`, je mit Grund |
| Ziffern nur ASCII | Plan | Transkriptmarken und `digit_freq`; eine Ziffer einer anderen Schrift ist kein Fall, den ein Eingang trifft |
| Stichprobe der Erkennung | Plan | Python dekodiert in Blöcken zu 8192 Bytes und kann an einem kaputten Byte hinter dem 8192. Zeichen scheitern, Go liest genau die 8192 Zeichen |
| Frist für `pdftotext` (2 min) und `yt-dlp` (10 min) | Plan E6, E7 | die Referenz hatte keine |

- [ ] **Step 6: Commit**

```bash
git add testdata/cases/4d-worlds testdata/cases/4d-source testdata/cases/4d testdata/cases/4d-map.toml internal/cli/cases_4d_test.go docs/.superpowers/parity
git commit -m "test(convert): record convert's cases against the reference"
```

---

### Task 13: Poppler-Goldens (Schritt des Menschen, dann des Agenten)

**Files:**
- Modify: `internal/cli/dev.go`, `internal/cli/dev_test.go` (`dev record-poppler`)
- Create: `testdata/convert/poppler/faketool.json` (vom Menschen aufgezeichnet)
- Create: `internal/brain/convert/poppler_test.go`
- Modify: `docs/.superpowers/parity/stufe-4d.md` (Messungen: Poppler)

**Interfaces:**
- Consumes: die PDFs aus Task 1; `newPDFToText` (Task 6); `faketool.Fixture`, `faketool.Answer`
- Produces: `loomux dev record-poppler --exe <pdftotext> --dir <pdfs> --out <fixture>`

- [ ] **Step 1: Der Aufzeichner**

Ein Entwicklerbefehl ruft das echte Poppler für jede Test-PDF, mit dem
Verzeichnis der PDFs als Arbeitsverzeichnis wie E6, und schreibt stdout und
Exit-Code je Aufruf verlustfrei in die Form, die die Attrappe abspielt; eine
Bash-Zeile mit `jq` verlöre dabei Bytes.

`internal/cli/dev.go`, Eintrag `"record-poppler": devRecordPoppler`, und:

```go
// devRecordPoppler runs pdftotext -v and pdftotext -layout -enc UTF-8 -eol unix <name> -
// for every PDF in --dir and writes the answers as a faketool fixture. A
// human runs it once, with Poppler installed, to record the real tool.
func devRecordPoppler(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev record-poppler", flag.ContinueOnError)
	fs.SetOutput(stderr)
	exe := fs.String("exe", "", "Poppler's pdftotext")
	dir := fs.String("dir", "", "the directory of the PDFs")
	out := fs.String("out", "", "the fixture to write")
	if err := fs.Parse(args); err != nil || *exe == "" || *dir == "" || *out == "" {
		return 2
	}
	entries, err := os.ReadDir(*dir)
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	version := popplerRun(child.Spec{Argv: []string{*exe, "-v"}})
	answers := []faketool.Answer{{Prefix: "pdftotext -v", Exit: version.Code, Stdout: version.Stdout + version.Stderr}}
	for _, e := range entries {
		if !strings.EqualFold(filepath.Ext(e.Name()), ".pdf") {
			continue
		}
		res := popplerRun(child.Spec{Argv: []string{*exe, "-layout", "-enc", "UTF-8", "-eol", "unix", e.Name(), "-"}, Dir: *dir})
		answers = append(answers, faketool.Answer{Prefix: "pdftotext -layout -enc UTF-8 -eol unix " + e.Name() + " -", Exit: res.Code, Stdout: res.Stdout})
		fmt.Fprintf(stdout, "%s: exit %d, %d bytes, stderr %q\n", e.Name(), res.Code, len(res.Stdout), strings.TrimSpace(res.Stderr))
	}
	data, _ := json.MarshalIndent(faketool.Fixture{Answers: answers}, "", " ")
	if err := os.WriteFile(*out, append(data, '\n'), 0o644); err != nil {
		return reportReconcileError(stderr, err)
	}
	return 0
}

// popplerRun is the seam a test replaces to record without Poppler.
var popplerRun = child.Run
```

Dazu in `dev_test.go`:

```go
func TestDevRecordPopplerWritesOneAnswerPerPDF(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.pdf"), "%PDF")
	writeFile(t, filepath.Join(dir, "b.txt"), "x")
	saved := popplerRun
	t.Cleanup(func() { popplerRun = saved })
	popplerRun = func(spec child.Spec) child.Result {
		if spec.Argv[1] == "-v" {
			return child.Result{Stderr: "pdftotext version 25.07.0\n"}
		}
		return child.Result{Stdout: "Text\f"}
	}
	out := filepath.Join(t.TempDir(), "faketool.json")
	code, stdout, _ := run("dev", "record-poppler", "--exe", "pdftotext", "--dir", dir, "--out", out)
	fixture, err := faketool.Load(out)
	if code != 0 || err != nil || len(fixture.Answers) != 2 || fixture.Answers[0].Stdout != "pdftotext version 25.07.0\n" ||
		fixture.Answers[1].Prefix != "pdftotext -layout -enc UTF-8 -eol unix a.pdf -" || fixture.Answers[1].Stdout != "Text\f" || strings.Count(stdout, "\n") != 1 {
		t.Fatalf("%d %v %+v %q", code, err, fixture, stdout)
	}
	for _, args := range [][]string{
		{"dev", "record-poppler", "--dir", dir, "--out", out},
		{"dev", "record-poppler", "--exe", "x", "--out", out},
		{"dev", "record-poppler", "--exe", "x", "--dir", dir},
	} {
		if code, _, _ := run(args...); code != 2 {
			t.Errorf("%q: %d", args, code)
		}
	}
	if code, _, _ := run("dev", "record-poppler", "--exe", "x", "--dir", filepath.Join(dir, "gone"), "--out", out); code != 1 {
		t.Errorf("an unreadable --dir: %d", code)
	}
	if code, _, _ := run("dev", "record-poppler", "--exe", "x", "--dir", dir, "--out", filepath.Join(dir, "gone", "f.json")); code != 1 {
		t.Errorf("an unwritable --out: %d", code)
	}
}
```

Run: `go test ./internal/cli/ -run RecordPoppler -count=1`
Expected: PASS.

Commit, bevor der Mensch aufzeichnet: Bis zu seinem Schritt laufen andere
Tasks weiter, und ein Commit von Task 14 träfe sonst auf ein ungestagtes
`dev.go` („inputs differ from the index“ im Pre-Commit).

```bash
git add internal/cli/dev.go internal/cli/dev_test.go
git commit -m "feat(dev): record what Poppler's pdftotext prints for each test PDF"
```

- [ ] **Step 2: Schritt des Menschen**

Der Mensch installiert Poppler (`winget install --id oschwartz10612.Poppler -e`),
baut `bin/loomux.exe` und ruft in PowerShell, deren `PATH` Poppler findet
(Git Bash fände xpdf):

```powershell
New-Item -ItemType Directory -Force testdata\convert\poppler
bin\loomux.exe dev record-poppler --exe (Get-Command pdftotext).Source --dir testdata\convert\pdf --out testdata\convert\poppler\faketool.json
```

Expected: `testdata/convert/poppler/faketool.json` und je PDF eine Zeile auf
stdout. Die Zeilen kommen in die Akte, Abschnitt „Messungen: Poppler“,
samt Version aus `-v`.

- [ ] **Step 3: Den Golden-Test schreiben**

`internal/brain/convert/poppler_test.go`:

```go
package convert

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/dev/faketool"
)

// recordedPoppler answers from what the real Poppler printed for the test
// PDFs (recorded by a human with loomux dev record-poppler); the fusion spec
// wants every external program replayed from a real run.
func recordedPoppler(t *testing.T) Tools {
	t.Helper()
	fixture, err := faketool.Load(filepath.Join("..", "..", "..", "testdata", "convert", "poppler", faketool.FixtureName))
	if err != nil || len(fixture.Answers) == 0 {
		t.Fatalf("no Poppler recording: %v", err)
	}
	return Tools{
		Look: func(name string) (string, error) { return name, nil },
		Run: func(spec child.Spec) child.Result {
			answer, ok := fixture.Match(spec.Argv)
			if !ok {
				t.Fatalf("Poppler was not recorded for %q", spec.Argv)
			}
			return child.Result{Code: answer.Exit, Stdout: answer.Stdout}
		},
	}
}

func TestPopplersOwnOutputConvertsAsTheSeamSays(t *testing.T) {
	p := newPDFToText(recordedPoppler(t))
	for name, check := range map[string]func(extraction, error) bool{
		"text.pdf":         func(e extraction, err error) bool { return err == nil && e.pages == 1 && strings.Contains(e.text, "Pruefbestand") },
		"Bericht März.pdf": func(e extraction, err error) bool { return err == nil && strings.Contains(e.text, "Pruefbestand") },
		// One line: the escaped line breaks between the two copies yield
		// nothing in pdftotext, where pypdf makes two paragraphs of them.
		"paragraphs.pdf": func(e extraction, err error) bool {
			return err == nil && strings.Count(e.text, "Pruefbestand") == 2 && !strings.Contains(e.text, "\n") &&
				strings.Contains(e.text, "pruefen kann.Hallo aus dem")
		},
		"blank.pdf":     func(e extraction, err error) bool { return err == nil && e.text == "" && e.pages == 1 },
		"mixed.pdf":     func(e extraction, err error) bool { return err == nil && e.pages == 10 && e.skipped == 7 },
		"allscan.pdf":   func(e extraction, err error) bool { return err == nil && e.text == "" && e.pages == 2 },
		"corrupt.pdf":   func(e extraction, err error) bool { return err != nil },
		"encrypted.pdf": func(e extraction, err error) bool { return err != nil },
		// Poppler ends with 99 on a PDF without pages; the reference saw none.
		"pageless.pdf": func(e extraction, err error) bool { return err != nil && strings.Contains(err.Error(), "exited 99") },
	} {
		if e, err := p.extract(name); !check(e, err) {
			t.Errorf("%s: %+v %v", name, e, err)
		}
	}
}
```

Die Erwartungen sind die Messung aus Task 1 an den 5000 pt breiten
Test-PDFs mit Poppler 25.07.0 (Akte, „Messungen“): `text.pdf` und
`Bericht März.pdf` je eine Zeile von 390 Zeichen; `paragraphs.pdf` **eine**
Zeile von 780 Zeichen mit beiden Kopien des Textes und nichts an der Naht
(`…pruefen kann.Hallo aus dem…`), wo pypdf zwei Absätze liefert;
`pageless.pdf` Exit 99 („cannot be read as a PDF“), wo die Referenz „keine
Seiten“ sieht. Beide Abweichungen zur Naht kommen in die Akte, Abschnitt
„Messungen: Poppler“. Weicht die Aufnahme des Menschen von der Messung aus
Task 1 ab (etwa eine andere Poppler-Version), hält der Plan an und fragt den
Menschen, statt die Erwartung nachzuziehen.

Run: `go test ./internal/brain/convert/ -run Poppler -count=1`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add docs/.superpowers/parity testdata/convert/poppler internal/brain/convert/poppler_test.go
git commit -m "test(convert): replay what the real Poppler prints for the test PDFs"
```

---

### Task 14: Selbstnutzung, Messung, Mutationen, Doku, Pull Request

**Files:**
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`
- Modify: `docs/en/migration.md`, `docs/de/migration.md` (4d, Fähigkeit „Eingang“)
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` (Zeile 4d)
- Modify: `README.md`, `README.de.md`
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md` (Abschnitt „Intake: `loomux convert`, `loomux fetch`“)
- Modify: `docs/.superpowers/parity/stufe-4d.md` (Selbstnutzung, Mutanten)

- [ ] **Step 1: Selbstnutzung (Schritt des Menschen)**

Entschieden am 2026-09-26: am echten Eingang `brain-knowledge/00 Eingang`
(Bereich `knowledge`, `manual_cloud`). Der Mensch

1. installiert Poppler, falls Task 13 es nicht schon tat;
2. legt eine eigene PDF und ein Transkript (Klammer- oder Bereichsform) in
   `00 Eingang`;
3. ruft in PowerShell `bin\loomux.exe convert` — mit oder ohne Modell, wie
   er will; mit Modell nach `loomux config set model.enabled true --global`;
4. ruft es ein zweites Mal: Erwartet ist keine Ausgabe und Exit 0.

Festgehalten in der Akte, Abschnitt „Selbstnutzung“: die Ausgabe beider
Läufe, der Kopf der beiden Zieldateien, und je Seite der PDF die Zahl der
Zeichen nach `pytext.Strip` — die Nachmessung der Scan-Schwelle 100 an echtem
Poppler (Entscheidung „PDF“). Liegt eine Textseite unter 100 oder eine
Scanseite darüber, hält der Plan an, und der Mensch entscheidet über die
Schwelle. Die Zieldateien behält oder löscht der Mensch; ein Agent schreibt
dort nichts (Vorschlag 4).

- [ ] **Step 2: Messen (Schritt des Menschen, dann des Agenten)**

`loomux convert` misst der Mensch: Seit Task 10 verweigert der Wächter es
dem Agenten, und es schriebe in den Tresor. Über den Eingang aus Step 1, der
nach dem zweiten Lauf nichts Neues trägt, in PowerShell 1 kalt und 5 warm:

```powershell
1..6 | ForEach-Object { (Measure-Command { bin\loomux.exe convert }).TotalMilliseconds }
```

Der Agent trägt die Zahlen in beide `benchmarks.md` ein, mit Datum,
Uhrzeit, Befehl, Zahl der Dateien im Eingang, kalt und warm, oben (neueste
zuerst). Dazu misst der Agent selbst `GODEBUG=inittrace=1 bin/loomux.exe --version`
gegen ein Build von `origin/master` (`git worktree add` nach `$SCRATCH/base`,
dort `go build -o $SCRATCH/base.exe ./cmd/loomux`): Die eingebettete Tabelle darf
keinen Startaufwand zeigen (Fusions-Spec, „Startzeit-Regel“); der Test im
Tor, der Paket-Inits über 500 Allokationen scheitern lässt, prüft dasselbe
laufend.

- [ ] **Step 3: Mutationen**

Run: `bin/loomux.exe dev mutants ./internal/brain/convert ./internal/brain/model ./internal/programs`
Expected: Überlebende gelesen; jeder bekommt in der Akte einen Test oder
eine Begründung.

- [ ] **Step 4: Doku**

- `cli-reference.md` (en/de): ein Abschnitt „Intake: `loomux convert`,
  `loomux fetch`“ hinter „Upkeep“ — Aufruf, was wohin geschrieben wird,
  Herkunftskopf, stdout/stderr/Exit, `pdftotext` nur Poppler, `yt-dlp` samt
  Schaltern, die Rollen `describe` und `place`, `[modules] brain`, der
  Wächter.
- `README.md` / `README.de.md`:
  - ein Absatz zum Eingang, mit den zwei Programmen und dem Hinweis auf
    `NOTICE.md`;
  - im Befehlsblock (`README.md:190-234`, `README.de.md` ebenso, „one line
    each“) je eine Zeile für `loomux convert [datei]` und
    `loomux fetch <url> [--scope …]`, mit dem Hinweis, dass es Befehle des
    Menschen sind (der Wächter verweigert sie einem Agenten);
  - im Absatz „The local model“ / „Das lokale Modell“ den Satz „No other
    area is ever sent to the model“ (`README.md:240`, `README.de.md:241-242`)
    berichtigen: `reconcile` fragt weiter nur für `local_only`-Bereiche, aber
    `convert` schickt die ersten 1800 Zeichen jeder gewandelten Datei eines
    Eingangs an das lokale Modell (`describe`, `place`), gleich welcher
    Modus, sobald Modell und Rolle an sind;
  - im Abschnitt „Releases“ neben `SHA256SUMS` die Datei `NOTICE.md` des
    Releases nennen (die Lizenzen aller fremden Teile im Binary);
  - den Abschnitt „Roadmap“ nach der Regel in `AGENTS.md` gegenlesen und
    nachführen: mindestens die Einleitung („the open migration stages 4c-1
    to 4e hold priority 3“, `README.md:155-156`) nach dem Stand von
    `migration.md` und die Zeile „Skill suites and review“ („migration stage
    4“); eine Zeile ändert sich nur, wenn 4d Roadmap-Arbeit beginnt,
    beendet, hinzufügt oder streicht.
- `migration.md` (en/de): 4d auf „🚧 gebaut; Schritte des Menschen offen
  (Poppler-Goldens, Selbstnutzung)“ oder ✅, je nachdem, ob Task 13 Step 2
  und Task 14 Step 1 gelaufen sind; die Fähigkeit „Eingang“ entsprechend;
  „Lokales Modell“ nennt `describe` und `place` als gebaut; 4e „hängt ab von
  4d“ nachführen.
- Fusions-Spec: Zeile 4d im selben Stand.

- [ ] **Step 5: Tor und Commit**

Run: `sh ci/gate.sh > "$SCRATCH/gate.log" 2>&1; echo $?` (`$SCRATCH` ist der Scratchpad der Sitzung)
Expected: 0, alle Lanes `ok`; bei Rot erst `grep -- '--- FAIL' "$SCRATCH/gate.log"`.

```bash
git add docs README.md README.de.md
git commit -m "docs: document convert and fetch, their self-use and their state"
```

- [ ] **Step 6: Pull Request vorbereiten**

Skill `release-pr`: Commits nach Thema gruppieren (die `docs:`-Commits, die
nur Spec, Plan und Akte schreiben — bis zur Nachführung am 2026-09-27 sechs —
in einen, Korrekturen desselben Zweigs in ihren Ursprungscommit), Label
mindestens `release:minor` (`feat`), Zeile `Release: minor — …` und ein
`## Changelog`-Block mit `### Added` (`loomux convert`, `loomux fetch` — ein
Agent darf beide nicht aufrufen —, die Rollen `describe` und `place`,
`NOTICE.md` im Release). Die Wächterregel gehört zu den neuen Befehlen und
steht darum unter `Added`, nicht unter `Changed`. `loomux dev
release parse-body` prüft; den Push-Befehl nennt der Skill, pushen tut der
Mensch.

---

## Selbstprüfung (2026-09-26)

- **Spec-Abdeckung:** Erkennung, Transkript, Kopf → T5; PDF über
  `pdftotext`, nur Poppler, fehlendes Programm → T6 (Vorschläge 9, 12); der
  Lauf ohne Abbruch, zweiter Lauf ohne Änderung, `readonly`, Rollen
  `describe` und `place` mit den Rückfällen der Referenz, Rangfolge der
  Ablage → T7; `loomux convert`, Modul Brain → T8 (Vorschlag 3); `fetch`
  samt `readonly` → T9 (Entscheidung „fetch“, Vorschlag 2); Richter und
  Zipf-Tabelle mit Lizenz → T2, T3 (Entscheidung „Zipf-Tabelle“); Weg des
  Hinweises und die Hinweise aller fremden Teile → T11 (Vorschlag 13,
  Entscheidung `NOTICE.md`); Wächter → T10 (Vorschlag 4); Ausgabeschema
  → T4 (Vorschlag 5); Funktionswortliste in `internal/brain/model` → T3
  (Vorschlag 6); `convert <datei>` ohne Modell → T7, T8 (Vorschlag 7);
  `brain-pdf/2` → T5, T12 (Vorschlag 8); Parität → T12, T13 (Vorschlag 10);
  Messen vor dem Bau → T1 (Vorschlag 11); Selbstnutzung, Messen, Mutationen,
  „Fertig, wenn“ → T14.
- **Typen:** `convert.Tools`, `convert.Area`, `Areas(areas, state, fallback)`,
  `ConvertAll(ctx, tools, []Area, state) (Outcome, error)`,
  `ConvertFile(ctx, tools, path) (string, string)`, `Fetch(tools, url)`,
  `ToInbox(subs, url, inbox)`, `(*model.Proposer).Describe(ctx, text)
  (string, bool)`, `Place(ctx, text, []string) (string, bool)`,
  `(*Client).AskFormat(ctx, prompt, format)` stimmen zwischen den Tasks
  überein. Pythons `text[:n]` gibt es einmal, als `pytext.FirstRunes` (T4),
  das `describe`, `place` und `ToInbox` rufen.
- **Schritte des Menschen:** Poppler ist installiert (B10); Task 13 Step 2
  (Poppler aufzeichnen); Task 14 Step 1
  (Selbstnutzung am echten Eingang) und Step 2 (Messung von `convert`).
  Ohne Halt laufen Task 1 bis 12 und Task 13 Step 1; Task 13 Step 3 wartet
  auf die Aufnahme. Der Pull Request kann mit offenen Schritten des Menschen
  geöffnet werden wie bei 4c-1; gemergt wird er erst nach Task 13 Step 2,
  weil sonst nichts das echte Poppler gemessen hat.
- **Offen gelassen, mit Regel statt Platzhalter:** die sha256 der Tabelle
  (T2, erst nach dem Generator bekannt), `multiToken` und
  `readsBackDeviations` (T2, T3, aus den Batterien), der Name der Test-Hilfe
  in `build_test.go` (T11, an die Datei gebunden). Das Verhalten von Poppler
  bei `pageless.pdf` (Exit 99) und `paragraphs.pdf` (eine Zeile) ist seit
  Task 1 gemessen; T13 erwartet es, die Aufnahme des Menschen bestätigt es.
- **Nachgeführt am 2026-09-27** nach Task 1 und dem Pre-flight-Scan: die
  Messungen (`--ignore-errors` in E7, 5000 pt breite Test-PDFs, `-v` ohne
  Blick auf den Exit-Code, CRLF in stderr und `-v`, die Spur einer Sprache
  nach yt-dlp) und die Entscheidungen S1 bis S13 stehen im Text der Tasks,
  die sie betreffen: `pytext.FirstRunes` statt `head`/`firstRunes` (T4, T5,
  T9), die Tests für 100 % je Funktion (T2, T7, T8, T9, T11), die
  Deklaration eines `readonly`-Bereichs unter `<zustand>/areas/<scope>` (T7,
  T12), der Fall `second-run` (T12, 29 Fälle), zwei Commits in T13,
  README-Roadmap, Befehlsblock und `NOTICE.md` in T14; die Spec nennt in
  Vorschlag 9 `internal/programs` und in der Entscheidung „fetch“ alle
  Schalter von E7.


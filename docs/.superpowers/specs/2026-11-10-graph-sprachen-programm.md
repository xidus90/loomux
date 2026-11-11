# Programm: C#, C++, Java, PHP und der Web-Stack im Code-Graphen und in der Prüfkette

**Stand:** 2026-11-10 · **Zweig:** `feat/graph-languages` (auf `feat/graph-bridges`, PR #149)

## Ziel

C#, C++, Java, PHP, JavaScript/JSX, TypeScript/TSX, HTML, CSS und SCSS sind
so eingebaut wie Go, Python und GDScript:

- **Graph:** ein Extraktor je Sprache auf dem gemeinsamen Tree-sitter-Kern,
  ein Resolver für Imports, Vererbung und Aufrufe, Test-Erkennung im Blast.
- **Prüfkette:** Presets für lint, types, test, coverage, graph und rules,
  Erkennung des Stacks, Zeile in `support.md`.
- **Kompatibilität:** jede vom Upstream gepflegte Version in `compat.toml`,
  ein Syntax-Test je Version, Quellen im Scout, Einrichtung in CI.
- **Brücken:** C# und C++ bieten Godot-Namen an (`godot.class`,
  `godot.method`, `godot.signal`); HTML, CSS und JS verbinden sich über
  `css.class`, `css.id`, `html.id` und `js.function`.

Alles in einem Worktree, ein Zweig, Commits je Sprachfamilie gruppiert.

## Reihenfolge und Teilprojekte

Jedes Teilprojekt hat eine eigene Spec und einen eigenen Plan unter
`docs/.superpowers/`:

1. **C# + Godot**
2. **C++ + Godot**
3. **Java**
4. **PHP**
5. **JS/JSX/TS/TSX** als eine Familie
6. **HTML/CSS/SCSS** mit den Web-Brücken; hängt an 5

Das Gemeinsame wird beim ersten Teilprojekt gebaut, das es braucht: die
Lockerung des Versionsformats in `compat` bei C#, die Sprachfamilien im
Resolver bei JS/TS.

## Parse-Prüfung (Recall gegen die C-Laufzeit)

Zwölf öffentliche Repos, flach geklont, je Datei die ERROR- und MISSING-Knoten
gezählt: einmal mit gotreesitter v0.57.0 und einmal mit der C-Laufzeit
(`tree-sitter` 0.26.1 aus PyPI, Grammatiken aus PyPI).

| Endung | Repo | gotreesitter: Dateien mit Fehlern | C-Laufzeit |
|---|---|---|---|
| `.cs` | jellyfin, godot-demo-projects | 0,2 % / 0 % | — |
| `.java` | guava, spring-petclinic | 0,5 % / 0 % | — |
| `.php` | laravel, composer | 0,1 % / 0,3 % | — |
| `.js` `.ts` `.tsx` | vite, express, bootstrap | ≤ 0,4 % | — |
| `.jsx` | vite-Vorlagen | 33,3 % | 33,3 % |
| `.cpp` | nlohmann/json, godot-cpp, fmt | 34 % / 43 % / 45 % | 31 % / 43 % / 40 % |
| `.hpp` `.h` | dieselben | 66–81 % | 75–80 % |
| `.css` | bootstrap, vite | 18 % / 4 % | 18 % / 4 % |
| `.scss` | bootstrap, vite | 84 % / 21 % | 84 % / 21 % |

**Befund:** gotreesitter steht der C-Laufzeit nicht nach; die Vorbedingung
der Roadmap für C++ ist erfüllt. Die Fehler liegen in den Grammatiken selbst:
- C++: Makros ohne Semikolon in Klassenkörpern (`GDCLASS(Ref, RefCounted)`),
  Namespace-Makros
- SCSS: `@use "x" as *;`
- CSS: Range-Media-Queries `(width >= 576px)`
- JSX mit der JavaScript-Grammatik

Die Extraktoren dieser Sprachen müssen darum Definitionen außerhalb eines
ERROR-Knotens auch dann lesen, wenn die Datei Fehler hat, und für jede
Grammatik-Lücke einen Test mitbringen, der zeigt, was trotzdem ankommt.

**C++: Wie viele Definitionen ankommen.** Gemessen werden Funktions-,
Klassen- und Struct-Definitionen mit Rumpf, und wie viele davon ein
Extraktor erreicht:

| Repo | Ohne ERROR-Vorfahr | Eigener Teilbaum ohne ERROR |
|---|---|---|
| godot-cpp | 38 % | 99 % |
| nlohmann/json | 35 % | 99 % |
| fmt | 16 % | 99 % |

- **Python-Regel scheidet aus.** Dort zählt keine Definition unter einem
  ERROR-Vorfahren. Für C++ verlöre sie zwei Drittel der Definitionen.
- **Regel für C++:** Eine Definition zählt, wenn ihr eigener Teilbaum keinen
  ERROR enthält, egal was darüber liegt.
- **Makrozeilen ausblenden.** Eine Zeile, die nur aus einem Makroaufruf ohne
  Semikolon besteht (`GDCLASS(X, Base)`, `Q_OBJECT`), wird vor dem Parsen
  durch Leerzeichen gleicher Länge ersetzt. So zerbricht `GDCLASS` die Klasse
  nicht mehr. Sonst erkennt die Grammatik die Klasse gar nicht, und keine der
  beiden Zählungen sieht sie. Die Byte-Offsets bleiben dabei gleich, und
  Spans und Rumpftexte werden aus dem Original geschnitten.

## Versionsachsen (Regel aus AGENTS.md: jede gepflegte Version)

Recherchiert am 2026-11-10 an den Quellen (endoflife.date, Herstellerseiten,
GitHub-Releases):

| Stack | `runtime` | Versionen | Vorab im Scout | EOL-Quelle |
|---|---|---|---|---|
| csharp | `dotnet` | 10.0, 11.0 (8 und 9 enden heute) | 12.0-Previews | endoflife.date `dotnet` |
| cpp | `cpp` | 17, 20, 23 (ISO-Standards) | 26 | keine; Liste von Hand |
| java | `java` | 8, 11, 17, 21, 25, 27 (Eclipse Temurin) | 28-EA | endoflife.date `eclipse-temurin` |
| php | `php` | 8.2, 8.3, 8.4, 8.5 | 8.6-RC (GA am 2026-11-19) | endoflife.date `php` |
| typescript | `node` | 22, 24, 26 | ungerade Linien | endoflife.date `nodejs` |
| css, html | — | lebende Standards, keine Versionsachse | — | — |

Entscheidungen dazu:

- **C++ hat keine Laufzeit mit gepflegten Versionen.** Die Achse sind die
  ISO-Standards, die die gepflegten Compiler vollständig umsetzen. Die
  Syntax-Einträge laufen nur den Graphen (wie `go-syntax-*`); die Lanes
  laufen auf einer CI-Werkzeugkette mit `-std=c++NN`. Der Scout liest für
  `cpp` keine Releases; der Standard 26 steht als Vorab-Eintrag in der Liste,
  bis ISO ihn veröffentlicht. *Kosten, falls falsch:* eine weitere Achse
  (GCC-Majors) später, ohne Umbau der Einträge.
- **Versionsformat:** `compat` verlangt heute `\d+\.\d+`. Java (`21`),
  Node (`22`) und C++ (`20`) haben keine Nebenversion. Das Format wird zu
  `\d+(\.\d+)?`. *Kosten, falls falsch:* eine Regex.
- **JS und TS teilen die Node-Achse.** Die Sprache hat Editionen (ES2026),
  aber gepflegt werden Laufzeiten. Je Node-Linie ein Syntax-Fixture mit dem,
  was diese Linie ohne Flag neu kann.
- **Werkzeug-Pins aus der Recherche:**
  - typescript-eslint 8.72 schließt TypeScript 7 aus. Das Fixture pinnt
    TypeScript 6.0.
  - stylelint-config-standard-scss 17 verlangt stylelint 17. stylelint wird
    auf 17 gepinnt.
- **Godot und .NET:** Godot 4.5 verlangt .NET 8 oder neuer; 8 endet heute.
  Die Godot-C#-Fixtures laufen darum auf .NET 10.
- **HTML-Lint:** `htmlhint` wird durch `html-validate` (11.21, gepflegt)
  ersetzt, wie im Brainstorming zu den Web-Stacks entschieden.

## Berührungsliste je Sprache

Jeder Plan geht sie vollständig durch. Die bestehenden Spiegel-Tests sind das
Tor: `all_test`, `sourceset`-Gleichheit, `support_test`, `layer1`,
`TestEveryFixtureIsUsed`, `notices_test`, `docs_test`. Neue Meta-Tests gibt es
nicht.

- **Graph:**
  - `extract/all/all.go` und die Pins in `all_test.go`
  - `sourceset.extensions`; die eingefrorene Kopie in `legacy_test.go`
  - `blast.IsTestPath`, `ask.testNames`, `repomap.langFromPath`,
    `blast.isFileQuery`
  - `corpus.pruneDirs`
  - `resolve.edgesOf` mit dem Resolver der Sprache
- **Prüfkette:**
  - `verify/presets.toml` (Stacks und Endungen) und `StackNames`
  - `carrierRank`, `IsSource`, `rules/files.go contentExts`, `rules/tests.toml`
  - `detect/signals.go`
  - Werkzeug-Zeilen im Config-Schema
- **Kompatibilität:**
  - `compat.toml` mit Fixtures unter `compat/testdata/`, `runtimeOf`
  - die Vorbereitung in `run.go`
  - Scout-Quellen und Parser
  - `ci/compat-setup.sh` und die Setup-Schritte in `compat.yml`
- **Doku:**
  - `docs/{en,de}/support.md`, `cli-reference.md`, `configuration.md`,
    `architecture.md`
  - Wiki `topics/code-graph.md`
  - beide READMEs (Roadmap: C#- und C++-Zeile fallen weg, wenn fertig; Java,
    PHP und Web kommen nicht dazu, weil sie in diesem Zweig fertig werden)
  - `CHANGELOG` über den PR
- **Hinweise:** `NOTICE.md` neu erzeugen (alle Grammatiken MIT).

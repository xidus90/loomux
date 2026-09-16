# loomux: Registry- und Manifestprüfungen an einer Stelle

**Datum:** 2026-09-16
**Stand:** entworfen, nicht umgesetzt
**Bezug:** [Stufe 1b-1](2026-09-15-loomux-stufe-1b-1-design.md), Paritätsliste
`docs/.superpowers/parity/stufe-1b-1.md` Zeilen 24, 27, 29, 31; Schreibschranke aus Stufe 1a
(`internal/brain/guard`). Basis ist der Zweig `sdd-1b-1` (`d7d0b2b`), nicht `master`.
**Messgrundlage:** 32 Fehlerfälle vom 2026-09-16 gegen die Python-Referenz
(`loomux-src/ub`, `3cc72d2`, `brain-mcp catalog`), gegen `loomux brain catalog` und gegen die
Schranke (`loomux hook pre-tool-use`) am Stand `d7d0b2b`; Tabelle im Anhang.

## Ziel

`loomux brain` verweigert dieselben kaputten Registries und Manifeste wie die Referenz und wie die
Schranke. Die Regeln stehen einmal, in `internal/config`; Schranke und `brain/*` rufen sie. Die
Meldungen sind loomux-eigen — die Referenz bestimmt, **was** verweigert wird, nicht **wie** es
formuliert ist.

## Ausgangslage

- **Zwei Leser derselben Dateien.** `config.ReadRegistry` dekodiert typisiert, überspringt Einträge
  ohne `scope`/`path` still und prüft sonst nichts; `readManifestAmong` prüft nur `[area] scope`
  und `[privacy] mode`. Die Schranke liest Registry und Manifeste **nicht** über `config`, sondern
  mit eigenen Lesern (`guard/registry.go`, `guard/manifest.go`) über `map[string]any` und bildet
  Pythons Prüfungen samt `repr`-Wortlaut nach (`guard/python.go`).
- **Gemessen, 32 Fälle:** Die Schranke verweigert 27 davon, 22 mit Pythons Wortlaut; die fünf
  `[model]`-Fälle lässt sie durch. `loomux brain` trifft keinen wörtlich (`mode` am nächsten, mit
  `"cloud"` statt `'cloud'`), lässt 9 still durch (`untouched_days = 0`, `on_merge`, `branch`,
  alle `[model]`, absolute `inbox`), lässt 3 Einträge ohne gültiges `scope`/`path` still weg,
  beendet 15 mit einem TOML-Typfehler und liest in 4 Fällen (doppelter Scope, Scope ohne brauchbare Zeichen,
  geteiltes Zustandsverzeichnis, zwei `signpost`) weiter, bis es später an etwas anderem scheitert.
  Bei doppeltem Scope liefert `privacy.VisibleAreas` beide Einträge, `privacy.Single` den ersten.
- **Echte Daten bestehen alles:** beide Registries (`%LOCALAPPDATA%\loomux`, 11 Bereiche;
  `%LOCALAPPDATA%\brain`, 10), alle Altmanifeste und das einzige `.loomux/config.toml` mit `[area]`
  (`project/loomux`) bestehen die Referenz einschließlich `[model]`. Jede Flagge der Registries ist
  ein Wahrheitswert, jeder `scope`, `path` und `wiki` eine nichtleere Zeichenkette, jeder gelesene
  `[layout]`-Wert eine Zeichenkette. Die strengeren Regeln unten verweigern heute also nichts.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Wo die Regeln stehen | `internal/config`. Die Leser der Schranke ziehen dorthin um; Schranke und `brain/*` rufen dieselben Funktionen (Variante b, vom Nutzer gewählt) |
| Kaputter Eintrag | **verweigert den ganzen Aufruf**, kein Überspringen mit Warnung (Begründung unten) |
| Wortlaut | loomux-eigen: Zeichenketten mit `%q`, Typen mit TOML-Namen, Einträge nach Position oder Scope. Kein `repr`-Nachbau, kein `truthy` |
| Typen | streng: ein Schlüssel, den loomux liest oder die Referenz prüft, muss den erwarteten TOML-Typ haben. `readonly = "yes"` wird verweigert, nicht als wahr gelesen |
| Unbekannte Schlüssel | nicht geprüft. `project/ultraloom` führt `[maintenance] merge_branch`, `knowledge` führt `[llm]` und `[maintenance] watch` — eine Allowlist würde echte Bereiche sperren |
| `config.ReadManifest` (Hook `post_edit`, `wiki.Root`) | bleibt typisiert und ungeprüft. Beide Aufrufer lesen einen Fehler als „kein Manifest“; ein strengerer Leser ließe dort die Wiki-Lane **still** ausfallen. Die sichtbare Verweigerung liefert die Schranke |
| Schranke | wird mit angepasst: neue Wortlaute, fünf neue `[model]`-Verweigerungen, strenge Typen statt `truthy` |
| Reihenfolge bei mehreren Defekten | eigene: erst alle Registry-Einträge in Dateireihenfolge, dann je Bereich das Manifest. Der erste Defekt beendet den Aufruf |

### Warum verweigern statt überspringen

1. Die Schranke verweigert bei genau diesen Defekten heute schon **jeden** Schreibzugriff außerhalb
   von Memory und Scratchpad (gemessen). Die Datenbefehle lesen nur; eine Verweigerung dort sperrt
   niemanden aus einem Baum aus, der nicht ohnehin gesperrt ist.
2. Bei doppeltem Scope und geteiltem Zustandsverzeichnis gibt es keinen sicheren Eintrag zum
   Überspringen — welcher der beiden fällt weg, und wessen `privacy` gilt?
3. Ein still übersprungener Eintrag ließe `brain` eine andere Registry sehen als die Schranke.
4. Die Registry liegt außerhalb jedes Bereichs; ein Mensch repariert sie im Editor, und die Meldung
   nennt Datei, Eintrag und Grund.

Die einzige Verschärfung beim Schreiben sind die neuen Typ- und `[model]`-Regeln der Schranke; laut
Messung trifft keine davon einen registrierten Bereich.

## Regeln und Wortlaute

Jede Meldung beginnt mit `<datei>: `, dem Pfad in Betriebssystemschreibweise
(`filepath.Join`). `<typ>` ist der TOML-Name des gefundenen Werts: `string`, `integer`, `float`,
`boolean`, `datetime`, `array`, `table`. Die CLI schreibt wie bisher `error: <meldung>` und endet
mit Exit 1; die Schranke schreibt wie bisher `loomux cannot read the registry, so it refuses:
<meldung>`.

### Registry (`config.ReadRegistry`)

Einträge heißen `[[area]] #N` (ab 1, wie jedes `#N` in dieser Spec), sobald ein gültiger Scope feststeht `[[area]] "<scope>"`.
Geprüft wird je Eintrag in dieser Reihenfolge; der erste Treffer beendet den Aufruf.

| # | Regel | Meldung |
|---|---|---|
| G1 | Datei nicht lesbar | `open <pfad>: <Systemtext>` (Go-Fehler unverändert, **ohne** vorangestelltes `<pfad>: `) |
| G2 | kein gültiges TOML | `<pfad>: not valid TOML: <Decoder>` (unverändert) |
| G3 | `area` ist kein Array | `area must be an array of [[area]] tables, found <typ>` |
| G4 | Eintrag ist keine Tabelle | `[[area]] #N must be a table, found <typ>` |
| G5 | `scope` fehlt | `[[area]] #N is missing "scope"` |
| G6 | `scope` falscher Typ oder leer | `[[area]] #N: scope must be a non-empty string, found <typ>` bzw. `found ""` |
| G7 | Scope doppelt | `[[area]] #N: duplicate scope "<scope>" (first at #M)` |
| G8 | Scope ohne brauchbare Zeichen | `[[area]] "<scope>": scope has no letter, digit, "_", "." or "-" and cannot name a state directory` |
| G9 | zwei Scopes, ein Zustandsverzeichnis | `scopes "<a>" and "<b>" share the state directory "<name>"` |
| G10 | `path` fehlt | `[[area]] "<scope>" is missing "path"` |
| G11 | `path` falscher Typ oder leer | `[[area]] "<scope>": path must be a non-empty string, found <typ>` bzw. `found ""` |
| G12 | `wiki` vorhanden, falscher Typ oder leer | `[[area]] "<scope>": wiki must be a non-empty string, found <typ>` bzw. `found ""` |
| G13 | `readonly`, `signpost`, `shared`, `workspace` vorhanden und kein Wahrheitswert | `[[area]] "<scope>": <schlüssel> must be a boolean, found <typ>` |
| G14 | zweites `signpost = true` | `scopes "<a>" and "<b>" both declare signpost; only one area may` |

G8 und G9 fragen `config.ManifestDir` mit `ReadOnly: true` nach dem Verzeichnis, wie
`guard.areaStateDir` heute; die Regel für brauchbare Zeichen bleibt `flat`.

### Manifest (`config.ReadDeclaration`)

`ReadDeclaration(path) (*Manifest, error)` liest **eine** Datei. Fehlt `[area]`, antwortet es
`ErrNoArea` vor jeder anderen Prüfung (Stufe 1a, R7a). Sonst in dieser Reihenfolge:

| # | Regel | Meldung |
|---|---|---|
| M1 | Datei nicht lesbar / kein TOML | wie heute: `<pfad>: cannot be read: <Go-Fehler>` bzw. `<pfad>: not valid TOML: <Decoder>` |
| M2 | `area`, `privacy`, `wiki`, `maintenance`, `model`, `layout`, `index` vorhanden und keine Tabelle | `[<abschnitt>] must be a table, found <typ>` |
| M3 | `[area] scope` fehlt | `[area] is missing "scope"` |
| M4 | `[area] scope` falscher Typ oder leer | `[area] scope must be a non-empty string, found <typ>` bzw. `found ""` |
| M5 | `[privacy] mode` kein bekannter Wert | `[privacy] mode must be one of automatic_cloud, local_only, manual_cloud, found "<wert>"` bzw. `found <typ>` |
| M6 | `[wiki] types` kein Array | `[wiki] types must be an array of strings, found <typ>` |
| M7 | Element von `[wiki] types` keine Zeichenkette | `[wiki] types #N must be a string, found <typ>` |
| M8 | `[maintenance] on_merge` kein Wahrheitswert | `[maintenance] on_merge must be a boolean, found <typ>` |
| M9 | `[maintenance] branch` falscher Typ oder leer | `[maintenance] branch must be a non-empty string, found <typ>` bzw. `found ""` |
| M10 | `[model] enabled` kein Wahrheitswert | `[model] enabled must be a boolean, found <typ>` |
| M11 | `[model] roles` keine Tabelle | `[model] roles must be a table, found <typ>` |
| M12 | unbekannte Rolle | `[model] roles has unknown "<a>", "<b>"; known are describe, place, propose` (sortiert) |
| M13 | Rolle kein Wahrheitswert | `[model] roles.<name> must be a boolean, found <typ>` |
| M14 | `[layout] wiki`, `hub`, `review`, `inbox` vorhanden und keine Zeichenkette | `[layout] <schlüssel> must be a string, found <typ>` |
| M15 | `[index] include`/`exclude`/`unsearched`, `[privacy] never` kein Array | `[<abschnitt>] <schlüssel> must be an array of strings, found <typ>` |
| M16 | Element einer Glob-Liste keine Zeichenkette | `[<abschnitt>] <schlüssel> #N must be a string, found <typ>` |
| M17 | `[wiki] untouched_days` keine ganze Zahl ≥ 1 | `[wiki] untouched_days must be an integer >= 1, found <typ>` bzw. `found <zahl>` |

`[check] lanes` wird gelesen wie heute (Liste von Namen oder Tabelle Name → Befehl) und bleibt
ungeprüft. Ein leerer `[layout]`-Wert bedeutet wie heute „nicht gesagt“.

`Manifest` bekommt zwei Felder: `Path` (die gelesene Datei, für Meldungen) und `LayoutInbox`.

### `inbox` (`(*Manifest).InboxLayout`)

Eine Methode neben `WikiLayout` und `HubLayout`, weil die Referenz den Wert dort prüft, wo er
gebraucht wird (`_inbox_of`), nicht beim Lesen. Absolut heißt `filepath.IsAbs` — die heutige
Semantik der Schranke, unter Windows ist `/in` also nicht absolut. Meldung:
`<pfad>: [layout] inbox must be relative to the area, found "<wert>"`.

`WikiLayout` und `HubLayout` behalten ihre heutigen `%q`-Meldungen ohne Pfadpräfix.

## Zuschnitt

### `internal/config`

- `registry.go`: `ReadRegistry(stateDir)` dekodiert über `toml.Decode` in `map[string]any` und
  prüft G1–G14. `registryEntry`/`registryFile` entfallen. Der Kommentar über das Überspringen
  entfällt mit dem Verhalten.
- `manifest.go`: `ReadDeclaration` (M1–M17) baut `*Manifest` aus der geprüften Map, samt Lanes.
  `readManifestAmong` bleibt für `ReadManifest` mit dem typisierten Dekodierer; für den
  Bereichsleser übernimmt `ReadDeclaration`. Damit stehen in `manifest.go` zwei Dekodierer — die
  bewusste Grenze aus der Entscheidung zu `ReadManifest`.
- `legacy.go`: `ReadAreaManifestUntilStage4(dir)` fragt die drei Namen wie heute (regulär und
  lesbar, sonst nächster Name; regulär und unlesbar ist ein Fehler), ruft `ReadDeclaration` und
  überspringt `.loomux/config.toml` bei `ErrNoArea`. Ein Altname ohne `[area]` antwortet
  `<pfad>: [area] is missing "scope"`, wie die Referenz ihn verweigert.
- Ein kleiner Helfer `tomlType(any) string` für `<typ>`; welche Go-Typen BurntSushi v1.6.0 für
  Datum und Uhrzeit liefert, wird im Plan gemessen, nicht angenommen.

### `internal/brain/guard`

- `registry.go`: `readRegistry` ruft `config.ReadRegistry` und projiziert auf `area`; der zweite
  Durchlauf liest je Bereich `manifestPath` (ein Name, loomux-Zustandsverzeichnis für
  schreibgeschützte Bereiche — unverändert), ruft `config.ReadDeclaration` und
  `InboxLayout`. `areaEntries`, `required`, `checkInbox`, `isAbsolutePath` entfallen.
- `manifest.go`: `readManifest`, `manifest`, `table`, `globs`, `isStringList`, `contains`,
  `sorted`, `privacyModes`, `wikiLayout`, `escapes`, `namesAPart` entfallen. `declaredWikiRoot`
  und `reviewCentre` (`guard.go`) rufen `config.ReadDeclaration`, lesen `Scope`, `WikiLayout()`
  und `LayoutReview`. `declarationIn`, `isRegularFile`, `bundleDir`, `manifestName` bleiben.
- `python.go`: `truthy`, `pyRepr`, `pyReprString`, `pyReprFloat` entfallen.
  `pythonJSONString` und `pythonTypeName` bleiben — sie formen die Antwort an den Host und das
  Payload-Protokoll, nicht Registry-Meldungen.
- Die Verweigerungsgründe (`loomux cannot read the registry, so it refuses: …`) bleiben; nur der
  Text nach dem Doppelpunkt ändert sich.

### `internal/brain/privacy`

- `VisibleAreas`: nach `VisibleManifest` für **jeden** Bereich, sichtbar oder nicht,
  `manifest.InboxLayout()`; der erste Fehler beendet den Aufruf. So prüft die Referenz alle Bereiche
  in `read_registry`, bevor sie Sichtbarkeit fragt.
- `Single` bleibt; der Kommentar „den ersten“ bekommt den Hinweis, dass die Registry keine zwei
  gleichen Scopes mehr durchlässt.

### Unberührt

`config.ReadManifest`, `wiki.Root`, `hooks/post_edit.go`, `importcases.registeredDirs` (duldet
einen Registry-Fehler weiter als „nichts zu falten“), `internal/brain/pytext`.

## Paritätsliste

Neue Liste `docs/.superpowers/parity/registry-manifest-pruefungen.md`, jede Zeile `offen` bis zur
Freigabe:

1. **Wortlaut** aller Registry- und Manifestmeldungen in loomux-Form (Tabellen oben) statt
   Pythons `repr`-Form — betrifft `brain` und Schranke; Messtabelle als Beleg.
2. **Strenge Typen:** `readonly`/`signpost`/`shared`/`workspace` außer `true`/`false`, `wiki = ""`,
   `[layout]`-Werte außer Zeichenketten — Python liest per `bool()` bzw. als „nicht gesagt“,
   loomux verweigert.
3. **Traceback-Fälle:** `wiki = 1` in der Registry (`TypeError`), `privacy = 5` und `wiki = 5` im
   Manifest (`AttributeError`) — Python bricht ab, loomux meldet G12 bzw. M2.
4. **Reihenfolge bei zwei Defekten:** Python prüft `inbox` eines Bereichs, bevor es den nächsten
   Registry-Eintrag liest; loomux prüft erst alle Einträge. Das Urteil ist gleich, der genannte
   Grund kann abweichen.
5. **Registry nicht lesbar:** `open <pfad>: …` statt bisher `<pfad>: open <pfad>: …`
   (ersetzt den Registry-Teil von Zeile 24 der Liste 1b-1).
6. **Schranke verweigert mehr:** `[model]`-Defekte (M2 für `model`, M10–M13) und `[layout]`
   `review`/`hub`/`wiki` als Nicht-Zeichenkette (M14) in irgendeinem registrierten Bereich. Bisher
   ließ sie `[model]` durch; ein nicht-textuelles `review` machte nur die Vorschlagsausnahme
   unwirksam (`reviewCentre` antwortet `""`), ein nicht-textuelles `wiki` verweigerte nur auf dem
   Weg über `declaredWikiRoot`. Jetzt liest der zweite Registry-Durchlauf jedes Manifest mit
   `ReadDeclaration` und verweigert jeden Write.

In `stufe-1b-1.md` werden die Zeilen 27, 29 und 31 auf „abgelöst durch
`registry-manifest-pruefungen.md`“ gesetzt und Zeile 24 im Registry-Teil angepasst. Keine
bestehende Zeile in `stufe-1a.md` oder `schranke-worktrees.md` hält einen Schrankenwortlaut fest;
dort ändert sich nichts.

## Nachweis

- **Tests in `internal/config`:** ein Tabellentest je Leser mit allen 32 Eingaben des Anhangs und
  je einem Fall für jede Regel G3–G14 und M2–M17, die der Anhang nicht abdeckt (u. a. G14 mit
  `signpost = "yes"`, M7, M13, M14, M16, `datetime`). Sollwerte sind die Meldungen dieser Spec.
  Kein Test startet Python.
- **Tests in der Schranke:** die bestehenden Erwartungen in `decide_test.go`, `internal_test.go`,
  `mutation_test.go` wechseln auf die neuen Wortlaute; ein Fall je `[model]`-Regel verweigert.
- **Tests in `privacy`:** doppelter Scope verweigert; absolute `inbox` eines `local_only`-Bereichs
  verweigert auch auf dem Cloud-Kanal.
- **Fallkorpus:** `go test ./internal/cli/` mit allen 71 Fällen von 1b-1 und den Fällen von 1a
  läuft grün. Die Fälle vergleichen Exit und stdout, nicht stderr.
- **Abdeckung:** 100 % je Funktion über `loomux dev covergate`.
- **Echte Daten:** `loomux brain catalog` gegen `%LOCALAPPDATA%\loomux` antwortet wie vorher (11
  Bereiche); ein Write in dieses Repo wird von der neu gebauten Schranke erlaubt.
- **Messung:** `BenchmarkDecideAgainstTheRealRegistry` (`internal/brain/guard`) und
  `VisibleAreasOfTheRealRegistry` (`internal/brain/search`) vor und nach der Änderung, kalt und
  warm, dazu die Schranke als Prozess, eingetragen in `docs/en/benchmarks.md` und
  `docs/de/benchmarks.md`. Der Dekodierweg der Schranke ändert sich nicht grundsätzlich
  (`Unmarshal` in eine Map wird `Decode` in eine Map); ein Rückschritt über das Rauschen hinaus
  wird untersucht, bevor gemergt wird.
- **READMEs:** Die Zeilen „Unified Pre-Tool Guard“ und „Brain Data Commands“ in `README.md` und
  `README.de.md` nennen, dass kaputte Registries und Manifeste in beiden verweigert werden.

## Anhang: Messung 2026-09-16

Welt je Fall: Registry mit Bereich `x` (`path` und `wiki` auf `<W>/a`), Manifest identisch als
`a/.brain.toml` (Referenz) und `a/.loomux/config.toml` (loomux). Schranke: `Write` auf `<W>/a/x.md`,
Welten außerhalb des Sitzungs-Scratchpads (darin gibt die Schranke vor der Registry frei).
„=Py“ heißt Pythons Wortlaut. „still“ heißt: kein Fehler. „TOML“ heißt:
`not valid TOML: … incompatible types …`.

| Fall | Referenz | Schranke heute | `brain` heute | loomux neu |
|---|---|---|---|---|
| doppelter Scope | `duplicate scope 'x'` | =Py | später: `no manifest found` | G7 |
| `scope` fehlt | `entry {'path': '…'} is missing the 'scope' key` | `entry map[path:…] …` | still, Eintrag fehlt | G5 |
| `path` fehlt | `entry 'x' is missing the 'path' key` | =Py | still, Eintrag fehlt | G10 |
| `scope = 3` | `scope must be a non-empty string, found 3` | =Py | TOML | G6 |
| `path = ""` | `path must be a non-empty string, found ''` | =Py | still, Eintrag fehlt | G11 |
| `scope = "///"` | `scope '///' has no usable characters; …` | =Py | später: `no manifest found` | G8 |
| `a/b` und `a-b` | `scopes 'a/b' and 'a-b' share the state directory 'a-b' …` | =Py | später: `no manifest found` | G9 |
| zwei `signpost` | `scopes 'x' and 'y' both declare signpost; …` | =Py | später: `no manifest found` | G14 |
| `[area]` statt `[[area]]` | `areas must be declared as [[area]] tables, not [area]` | =Py | TOML | G3 |
| `area = [1]` | `expected an [[area]] table, found 1` | =Py | TOML | G4 |
| `readonly = "yes"` | schreibgeschützt, dann fehlendes Manifest | schreibgeschützt, verweigert als read-only | TOML | G13 |
| `wiki = 1` (Registry) | `TypeError` (Traceback) | `wiki must be a string, found 1` | TOML | G12 |
| `types = "Topic"` | `[wiki] types must be a list of strings` | =Py | TOML | M6 |
| `types = [1]` | `[wiki] types must be a list of strings` | =Py | TOML | M7 |
| `untouched_days = 0` | `[wiki] untouched_days must be an integer >= 1` | =Py | still | M17 |
| `untouched_days = true` | wie oben | =Py | TOML | M17 |
| `untouched_days = "5"` | wie oben | =Py | TOML | M17 |
| `on_merge = "yes"` | `on_merge must be a boolean, found 'yes'` | =Py | still | M8 |
| `branch = ""` | `branch must be a non-empty string` | =Py | still | M9 |
| `model = 5` | `[model] must be a table` | still | still | M2 |
| `[model] enabled = "yes"` | `enabled must be a boolean, found 'yes'` | still | still | M10 |
| `[model] roles = 5` | `[model] roles must be a table` | still | still | M11 |
| `[model.roles] guess = true` | `roles knows only describe, place, propose, found guess` | still | still | M12 |
| `[model.roles] place = "yes"` | `roles.place must be a boolean` | still | still | M13 |
| `include = "*.md"` | `[index] include must be an array of strings` | =Py | TOML | M15 |
| `include = [3]` | `[index] include contains a non-string: 3` | =Py | TOML | M16 |
| `never = "x"` | `[privacy] never must be an array of strings` | =Py | TOML | M15 |
| `[area] scope = 3` | `[area] scope is required and must be a non-empty string` | =Py | TOML | M4 |
| `inbox = "C:/in"` | `[layout] inbox must be relative to the area, found 'C:/in'` | =Py | still | InboxLayout |
| `privacy = 5` | `AttributeError` (Traceback) | `[privacy] must be a table` | TOML | M2 |
| `wiki = 5` (Manifest) | `AttributeError` (Traceback) | `[wiki] must be a table` | TOML | M2 |
| `mode = "cloud"` | `mode must be one of …, found 'cloud'` | =Py | `found "cloud"` | M5 |

Kontrollfall ohne Defekt: Referenz und `brain` drucken `* [x](brain://x/)`, die Schranke erlaubt.

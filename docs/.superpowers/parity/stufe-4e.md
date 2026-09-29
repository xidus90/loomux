# Paritätsakte Stufe 4e

**Spec:** `docs/.superpowers/specs/2026-09-28-loomux-stufe-4e-design.md`

Stufe 4e stellt die vier Wirte auf loomux um. Diese Akte hält fest, was die
Umstellung bestimmt: die Messungen unten, danach die Checkliste.

## Messungen

Gemessen am 2026-09-29 unter Windows 11 Pro 10.0.26200, im Arbeitsbaum
`next-level-4-c1494e` auf `docs/stage-4e-spec` (HEAD `12728d0c`). Der Befehl
`bin/loomux.exe --version` lieferte `loomux 0.0.0-dev`: ein Entwicklungsbau aus
diesem Baum, kein Release. Außer der Wegwerfkopie aus Schritt 1 wurde nichts
geschrieben; kein Manifest, keine Registry, keine `.loomux/config.toml`
wurde geschrieben.

### 1. `init --dry-run` gegen ein von Hand geschriebenes `[area]`

**Ergebnis: der `[area]`-Block bleibt erhalten. `init` ersetzt die Datei nie,
es hängt Blöcke an oder ändert einzelne Zeilen.**

Aufbau: `git worktree add --detach <scratchpad>/probe HEAD`. Die Wegwerfkopie
trägt schon ein von Hand geschriebenes `[area] scope = "project/loomux"` samt
`[layout]` und `[index]` (kein `init` hat sie erzeugt).

**Abweichung vom Auftrag:** den Scope auf `probe/x` zu ändern ging nicht. Der
Wächter verweigert jedes Schreibwerkzeug auf eine `.loomux/config.toml`, auch in
der Wegwerfkopie („the manifest is where the barrier reads its own limits, so no
writing tool may touch it“). Ich habe ihn nicht per Shell umgangen. Der Befund
trägt trotzdem, weil die vorhandene Datei schon ein `[area]` hat.

Zwei Läufe, je eine schlichte Zeile aus dem Arbeitsbaum, gegen die Kopie:

```
go run ./cmd/loomux init --dry-run --yes --root <scratchpad>/probe
go run ./cmd/loomux init --dry-run --yes --graph none --root <scratchpad>/probe
```

- Der erste Lauf meldet keine Änderung an `.loomux/config.toml` (nur die Aktion
  `binary-build` und Hinweise): die Datei bleibt, wie sie ist.
- Der zweite Lauf verlangt eine Änderung. Die Ausgabe ist ein Anhängen ans Ende
  der Datei: `+ [modules]` und `+ graph = false`, hinter der letzten
  `[[policy.paths.rules]]`. Der `[area]`-Block kommt im Diff nicht vor.

Fundstelle: `internal/setup/configtext.go`, `configText` ändert den Text nur über
`edit.Set`, `edit.Remove` und `edit.AppendBlock`, bricht vorher bei einer Datei
ab, die `schema.Validate` nicht besteht (`namedConfig`), und gibt `before`
unverändert zurück, wenn nichts zu tun ist. Ein Ersetzen der Datei gibt es im
Code nicht.

Folgerung für die Reihenfolge der Blöcke 4 und 5 der Checkliste: `init` ist
gegen ein vorhandenes `[area]` sicher; Block 4 muss nicht auf Block 5 warten,
damit ein handgeschriebenes `[area]` heil bleibt. **Nicht gemessen:** ein `init`
ohne `[area]` in der Datei (das die Bereichserklärung erst anlegt), und ein
echter Lauf ohne `--dry-run`.

### 2. Die vier Manifeste, die die Probe vom 2026-09-28 nicht deckte

Die zehn Bereiche stehen in `%LOCALAPPDATA%\brain\registry.toml` (die alte
Registry). **Berichtigt am 2026-09-29:** die neue Registry
`%LOCALAPPDATA%\loomux\registry.toml` (Datei vom 17.09.) führte schon zur
Messzeit dieselben zehn Bereiche plus `project/loomux`, siehe Block 3. Die
frühere Aussage an dieser Stelle, sie halte „nur die Vorlage mit einem
Eintrag“, war falsch, nicht später überholt. Gelesen wurden vier Dateien, die
der Vorlauf offen ließ:

| Scope | Manifest | Abschnitte und Schlüssel |
|---|---|---|
| `engineering/python` | `C:/Users/micro/Documents/#GIT/brain-knowledge/92 Engineering/python/.brain.toml` | `[area] scope`, `[index] include` |
| `engineering/craft` | `…/brain-knowledge/92 Engineering/craft/.brain.toml` | `[area] scope`, `[index] include` |
| `project/obsidian-ai` | `C:/Users/micro/Documents/#GIT/#Obsidian/AI/.brain.toml` | `[area] scope`, `[layout] wiki`, `[index] include`, `exclude`, `unsearched`, `[privacy] mode` |
| `hub` | `…/brain-knowledge/91 Projekte/.brain.toml` | `[area] scope`, `[index] include` |

Alle Schlüssel dieser vier Dateien (`scope`, `wiki`, `include`, `exclude`,
`unsearched`, `mode`) stehen in `DeclarationKeys()`; **kein** unbekannter
Schlüssel. Damit sind alle zehn Bereiche gelesen: `brain-knowledge`,
`ecoflow`, `iam_wiki`, `space`, `ultra-brain`, `ultraloom` waren es schon.

Die Dateien, die es gibt (Fundstelle: Verzeichnislisting `ls -a`):

- `.brain.toml` liegt in `brain-knowledge`, `ultra-brain`, `space`, `iam_wiki`,
  `#Obsidian/AI`, `ecoflow` und in den drei Ordnern unter `brain-knowledge`
  (`92 Engineering/python`, `92 Engineering/craft`, `91 Projekte`).
- `ultraloom` hat `.ultra-brain/config.toml` und **kein** `.brain.toml`.
- `ultra-brain` hat neben `.brain.toml` einen **leeren** Ordner `.ultra-brain/`.

Aus der Gegenprobe, die sich bei Schritt 4a ergab: `ultraloom/.ultra-brain/config.toml`
trägt drei Schlüssel, die kein Leser liest, `[area] wiki = true`,
`[layout] sources = "docs"` und `[maintenance] merge_branch = "master"`
(siehe 4a). Task 3 misst die Klassen je Schlüssel selbst.

### 3. Die Leser des Bestands, die an `ResolvedAreaDir` vorbeigehen

Befehle:

```
git grep -n "config.ManifestDir(" -- 'internal/**/*.go' ':!*_test.go'
git grep -n "ResolvedAreaDir(" -- 'internal/**/*.go' ':!*_test.go'
```

Die Regel: `ResolvedAreaDir` (`internal/config/artifacts.go:65`) gibt für einen
schreibbaren Bereich `area.Path`, für einen read-only Bereich
`Resolve("areas/<flat scope>")`. `ManifestDir` (`internal/config/manifest.go:481`)
kennt nur den neuen Ort, ohne Rückfall.

**Direkt an `ResolvedAreaDir` vorbei (`ManifestDir`, sechs Fundstellen, alle in
der Erwartung der Spec):**

| Fundstelle | Was sie tut | Lesen (braucht das Aside) oder Schreiben/Tausch (braucht es nicht) |
|---|---|---|
| `internal/brain/apply/resolve.go:276` `registerWrite` | Pfad des Registers, wie es **geschrieben** wird, immer unter dem neuen Ort | Schreiben; braucht es nicht |
| `internal/brain/apply/stock.go:61` `moveStock`, `target` | Ziel des Tauschs | Tausch; braucht es nicht |
| `internal/brain/guard/registry.go:51` `areaStateDir`, gerufen von `manifestPath` (`:59`) | Pfad der **Deklaration** eines read-only Bereichs; `checkDeclaration` (`:70`) und `reviewCentre` (`guard.go:208`) lesen sie mit `ReadDeclaration` | **Lesen; braucht das Aside.** Ohne es überspringt `reviewCentre` den Bereich, solange das Ziel fehlt (`isRegularFile` ist falsch), und `checkDeclaration` meldet keinen Fehler; der Rückfallordner fehlt hier ganz, dieser Leser kennt nur den neuen Ort |
| `internal/brain/index/staging.go:38` `recoverStock` | `lock.Recover(ManifestDir)`: heilt ein erschlagenes `ReplaceDir` | Heilen; braucht es **nicht** (es ist der Heiler; gäbe man ihm das Aside, hätte er nichts zu tun) |
| `internal/brain/index/staging.go:62` `publish`, `target` | Ziel, in das `ReplaceDir` tauscht | Tausch; braucht es nicht |
| `internal/dev/benchsearch/corpusrun.go:70` `write` | legt Deklaration und Register eines frischen Testbestands an | Schreiben; braucht es nicht |

`internal/config/registry.go:187` `stateDirOf` ist der siebte Aufruf von
`ManifestDir` (kein `config.`-Präfix, darum nicht im Grep der Spec): sie **öffnet
nichts**, sondern benennt das Zustandsverzeichnis, um zwei Bereiche mit demselben
Verzeichnis zu erkennen (`registry.go:142-149`). Braucht es nicht.

**Über `ResolvedAreaDir` (alle Leser, nehmen das Aside mit, sobald die
Auflösung es liefert):** `apply/resolve.go:268` `registerRead`, `apply/stock.go:65`
(die Quelle der Kopie), `catalog/area.go:15`, `check/house/federation.go:373`,
`check/run/run.go:350` und `:504`, `convert/run.go:38`, `graph/read.go:33`,
`index/reindex.go:166`, `maintenance/reconcile.go:271`, `maintenance/scan.go:77`,
`privacy/areas.go:42`, `search/search.go:132`, `status/status.go:60` und `:123`,
`cli/lintsweep.go:93`, `cli/wikicmd.go:181`. Sie öffnen den Ordner alle zum
**Lesen**; keiner schreibt hindurch. (`index/reindex.go:166` liest die
Ausgangsdateien für den Lauf und schreibt danach über `publish`.)

**Einzige Fundstelle, die ein Leser ist und am Aside vorbeigeht:**
`guard/registry.go`. Die anderen fünf Fundstellen schreiben oder tauschen und
dürfen das Aside nicht bekommen.

**Beobachtung außerhalb des Auftrags, für den Plan:** `moveStock` beruft
`lock.Recover(target)` vor dem Schreiben (siehe 4). Der Schreibpfad von
`approve` (`apply/approve.go:452-460`) hängt daran.

### 4. `Manifest.Lanes` und `moveStock`

```
git grep -n "\.Lanes\b" -- 'internal/**/*.go' ':!*_test.go'
```

**`Manifest.Lanes`:** außer `ReadManifest` liest sie **kein** Produktcode. Die
Treffer in `internal/config/manifest.go:217,223` sind `file.Check.Lanes`, der
Draht (`toml.Primitive`), aus dem `readManifestAmong` das Feld füllt
(`manifest.go:104` deklariert es, `:215-230` befüllt es). Die Treffer in
`internal/verify/*` sind `Stack.Lanes`, `preset.Lanes`, `variant.Lanes`: ein
anderer Typ. Ein weiterer Leser existiert nur im Test
(`internal/config/manifest_test.go:465`, `TestReadManifestCheckLanes`); einen
Konstruktor `LaneConfig` außerhalb von `manifest.go` gibt es nicht
(`git grep LaneConfig`). **Gleicher Befund wie am 2026-09-28.**

**`moveStock`** (`internal/brain/apply/stock.go`, einziger Aufrufer
`approve.go:460`, unter `config.LockArea`): tut außer der Kopie noch drei Dinge.

1. `recoverDir(target)` (= `lock.Recover`, `stock.go:62`) heilt ein erschlagenes
   `ReplaceDir`, **bevor** es entscheidet. Es ruft `registerWrite` **nicht**;
   das Wort steht nur im Kommentar (`stock.go:30`), `registerWrite` liegt in
   `resolve.go:275` und wird von `approve.go:452` und `resolve.go:286,334`
   gerufen.
2. `samePath(source, target)` (`:66`): liegt der Bestand schon am neuen Ort,
   kehrt die Funktion zurück.
3. Staging und Tausch: `stagingDir`, `defer os.RemoveAll(staging)`,
   `copyStock`, `beforeSwap` (nur ein Testhaken, in Produktion `nil`), dann ein
   glattes `swapDir` (= `os.Rename`); ein Fehlschlag, während das Ziel steht,
   wird verschluckt.

Nur ein read-only Bereich ist betroffen (`if !area.ReadOnly { return nil }`).

**Was bei Wegfall des Rückfalls (Ordner leer) von `moveStock` bleibt:**
`ArtifactLookup.Resolve` gibt bei `Fallback == ""` den neuen Ort zurück
(`artifacts.go:30`), also ist `source == target`, und `moveStock` kehrt nach dem
`recoverDir` zurück. Von ihm bleibt allein der Aufruf von `lock.Recover`.
**Bedenken (gelesen, nicht gelaufen):** fiele `moveStock` ganz weg, schriebe
`approve` das Register eines read-only Bereichs über `registerWrite` in ein
fehlendes Ziel, neben dem ein Aside einer erschlagenen `ReplaceDir` liegt; ein
späterer `Recover` sähe dann „Ziel da“ und **löschte das Aside**, also den alten
Bestand. Der Plan muss das `lock.Recover` im Schreibpfad von `approve` also
behalten, oder 1′ ist vorher gebaut. Die Lücke würde erst bei C entstehen, nicht
vorher: solange der Rückfall gilt, macht `moveStock` genau diesen Schritt.

### 4a. Deckt `DeclarationKeys ∪ schema.Keys` jeden Leser?

```
git grep -n 'toml:"\|toml.Unmarshal\|toml.Decode\|PrimitiveDecode' -- 'internal/**/*.go' ':!*_test.go'
```

Gelesen sind die Manifest-Leser: `config/manifest.go` (`manifestFile`),
`declaration.go`, `policy.go`, `agent.go`, `modules.go`, `flowsettings.go`,
`guardsettings.go`, `modelsettings.go`, `searchsettings.go`,
`worktree/mirror/mirrorcfg.go`, `verify/commit/policy.go`, `verify/schema.go`.
Nicht gelesen wurden Treffer, die keine Bereichs- oder Projektmanifeste lesen
(`cases`, `dev/importcases`, `flow/load`, `setup/state.go`, `serve`, `sessions`).

`DeclarationKeys()`: `area{scope}`, `privacy{mode,never}`, `wiki{types,untouched_days}`,
`maintenance{on_merge,branch}`, `model{enabled,roles}`, `layout{wiki,hub,review,inbox}`,
`index{include,exclude,unsearched}`.

**Differenz: gelesen, aber weder in `DeclarationKeys` noch in `schema.Keys`:**

| Leser | Gelesen | Wo es schon steht |
|---|---|---|
| `verify/schema.go` `parseVerify`, `parseStack` | `[verify.<stack>]` (Stackname aus `StackNames()`) mit den Arten `lint`, `types`, `test`, `coverage`, `graph`, dazu `[verify.gdscript] import_check` | `schema.Keys` führt `verify` nur mit `max_parallel`, `timeout`, `profiles`; die Stack-Tabellen fehlen. `TopKeys()` deckt die drei |
| `config/modelsettings.go` `parseModelSettings` | `[model] endpoint`, `name`, `temperature` | nur die **globale** `config.toml` (`stateDir/config.toml`, `GlobalModelKeys`), kein Projektmanifest |
| `config/searchsettings.go` | `[search] backbone` | nur die globale Datei (`GlobalSearchKeys`) |

Die beiden globalen Schlüsselgruppen sind kein Befund für `area check`, weil kein
Bereichsmanifest sie liest; die `[verify.<stack>…]`-Tabellen sind es: ein Leser
liest sie, `schema.Keys` kennt sie nicht, und eine Klassifizierung nach
`DeclarationKeys ∪ schema.Keys` hielte sie für unbekannt. Sie brauchen eine
**dritte Tabelle** in `classifyKeys` (Leser liest sie wirklich).

Alle übrigen Abschnitte sind gedeckt: `[commit]` (`language`, `threshold`,
`conventional`, `allow`) und `[policy.*]` stehen in `schema.Keys` und werden von
`verify/commit/policy.go` und `policy.go` gelesen; `[guard] mode`, `[worktree] mirror`,
`[modules]`, `[flow]`, `[agent]` (`default`, `mcp_servers`, `models.*`, `roles.*`)
stehen in `schema.Keys`.

**Differenz: geschrieben oder in Manifesten vorhanden, aber von keinem Leser gelesen:**

| Schlüssel | Woher | Beleg |
|---|---|---|
| `[check] lanes` | `manifestFile.Check.Lanes` füllt `Manifest.Lanes`, das kein Produktcode liest (Schritt 4) | am 2026-09-28 gemessen, heute bestätigt |
| `[layout] sources` | schreibt `area add` (`internal/cli/area.go:283`); steht in `ultraloom/.ultra-brain/config.toml` | kein Leser: `manifestFile.Layout` und `DeclarationKeys["layout"]` kennen `sources` nicht |
| **`[area] wiki = true`** (neu) | schreibt `area add` (`area.go:280`, `"wiki = true\n"`); steht in `ultraloom/.ultra-brain/config.toml` | `manifestFile.Area` und `declaration` lesen nur `scope` |
| **`[maintenance] merge_branch`** (neu) | steht in `ultraloom/.ultra-brain/config.toml` | kein Leser; der Kommentar in `area.go:270-272` nennt es schon („the reference writes `merge_branch`, which its own reader never reads“). Gelesen wird `branch` |

Zwei der vier sind gegenüber dem Auftrag neu (`[area] wiki`, `merge_branch`).
Alle vier sind für den Menschen wirkungslos und gehören als Einträge in
`legacyHints` (Task 2); `[layout] sources` und `[area] wiki` schreibt `area add`
selbst weiter, das die Umstellung dann mit ändert.

### 5. Die sechs offenen Entscheidungen aus `parity/artefakte-nach-lebensdauer.md`

Gelesen ist die Akte (Nr. 1, 2, 3, 6, 8, 9). Die Frage ist, ob eine Entscheidung
**Zustandsdateien unter `%LOCALAPPDATA%\loomux`** (Ort oder Form dessen, was dort
liegt) oder **das Manifest** berührt.

| # | Entscheidung | Zustandsdateien oder Manifest | Steht vor der Checkliste (Block 3 oder 4)? |
|---|---|---|---|
| 1 | schreibbare Bereiche schreiben nur den Wurzelkatalog, `graph.json` und Wurzelkatalog wandern ins Zustandsverzeichnis | **ja**: Ort der Artefakte; `staging.go`, `graph/read.go`, `status/status.go` | ja |
| 2 | Revision zählt nur noch Prüfungen | nein: ändert nur, wann `reindex` in `_identities.tsv` eine Zeile fortschreibt, nicht Ort oder Form; das Register read-only Bereiche liegt allerdings unter `areas/<scope>` (mittelbar) | nein |
| 3 | Register mit Aliasen, Grabsteinen, Union-Merge | **ja**: Form von `_identities.tsv` (auch der Datei unter `areas/<scope>`), dazu `.gitattributes` | ja |
| 6 | `reindex` löst einen verknüpften Worktree über das gemeinsame Git-Verzeichnis auf, schreibt nur das Register des Zweigs | nein: schreibt in den Baum des Bereichs, nicht ins Zustandsverzeichnis, und ändert kein Manifest. **Hängt an #3** (das Register des Zweigs) | nein |
| 8 | frühere Indexausgabe (alte Kataloge, `graph.json`) aus dem Baum räumen | **ja**: nur nach #1 und zieht dessen Verlegung nach; entfällt, wenn #1 entfällt | ja, mit #1 |
| 9 | Schranke verweigert `_identities.tsv` in registrierten Bereichen; Registercommit im Tresor | nein: Regeln der Schranke (`internal/brain/guard`) und ein Commit über `internal/brain/vcs`; weder Zustandsdatei noch Manifest | nein |

**Folgerung:** #1, #3 und #8 berühren Zustandsdateien; keine berührt das
Manifest. Steht die Entscheidung dafür aus, gehören sie **vor** die Checkliste
(Block 3 oder 4), damit die Umstellung nicht ein zweites Mal an dieselben Dateien
muss; #2, #6 und #9 nicht. Die Wertung „#2 mittelbar“ ist mein Urteil, keine
Messung: die Spec ordnet #2 nicht ein.

### 6. `loomux area check` an den zehn echten Bereichen (2026-09-29)

Binary aus dem Arbeitsbaum (`go build -o bin/loomux.exe ./cmd/loomux`), je
Bereich eine schlichte Zeile `./bin/loomux.exe area check "<pfad>"`. Der
loomux-Wächter hat keinen der Aufrufe verweigert. Nur lesend.

| Bereich | Gewählt | Exit | Klassen |
|---|---|---|---|
| `brain-knowledge` | `.brain.toml` | 1 | read: `area.scope`, `index.exclude`, `index.include`, `layout.hub`, `layout.inbox`, `layout.review`, `privacy.mode`, `privacy.never`. **ignored:** `area.wiki` (Hinweis), `layout.sources` (Hinweis), `llm.local` (Hinweis `[model]`), `maintenance.stale_after_days` (kein Hinweis), `maintenance.watch` (kein Hinweis) |
| `ecoflow` | `.brain.toml` | 0 | alle read (`area.scope`, `index.exclude/include/unsearched`, `layout.wiki`) |
| `iam_wiki` | `.brain.toml` | 0 | alle read (dazu `privacy.mode`) |
| `space` | `.brain.toml` | 0 | alle read (dazu `privacy.mode`) |
| `ultra-brain` | `.brain.toml` | 1 | read: sechs Schlüssel; **ignored:** `check.lanes` (Hinweis `[verify]`). Der leere Ordner `.ultra-brain/` wird übergangen |
| `ultraloom` | `.ultra-brain\config.toml` | 1 | read: `area.scope`, `index.include`, `index.unsearched`, `layout.wiki`, `maintenance.on_merge`, `privacy.mode`; **ignored:** `area.wiki`, `layout.sources`, `maintenance.merge_branch` (alle mit Hinweis) |
| `brain-knowledge/92 Engineering/python` | `.brain.toml` | 0 | `area.scope`, `index.include`: read |
| `brain-knowledge/92 Engineering/craft` | `.brain.toml` | 0 | `area.scope`, `index.include`: read |
| `#Obsidian/AI` | `.brain.toml` | 0 | alle read (`layout.wiki`, `index.exclude/include/unsearched`, `privacy.mode`) |
| `brain-knowledge/91 Projekte` | `.brain.toml` | 0 | `area.scope`, `index.include`: read |

Kein Bereich meldet `shadowed`, `refused` oder eine Datei ohne `[area]`.

**Schlüssel, die weder in `DeclarationKeys` noch im Schema noch in `legacyHints`
stehen und für die Leser wirkungslos sind** (Vorschlag für `legacyHints`, nicht
eingetragen): `maintenance.watch` und `maintenance.stale_after_days` in
`brain-knowledge/.brain.toml`. Ein Leser in `internal/` liest sie nicht (`grep`
nach `stale_after_days` und `watch` findet nur die Frontmatter-Größe
`stale_after` der Seiten), und im alten `ultra-brain` stehen sie nur in Spec
und Plan (`2026-08-18-ultra-brain-architektur-design.md:459-460`), in keinem
Leser. Ein Gegenstück in loomux gibt es nicht; ein Hinweis „in der alten Spec
genannt, von keinem Leser gelesen“ wäre die ehrliche Fassung. Ob er
aufgenommen wird, entscheidet der Controller.

### 7. Die offenen Messungen der Umstellung (2026-09-29)

Binary `bin/loomux.exe` aus dem Arbeitsbaum (Stand `90a0e7a4`), uv 0.12.16.
Jeder Aufruf als schlichte Einzelzeile. Mit Umleitung in eine Datei hat der
Wächter `init --dry-run` verweigert, ohne Umleitung ließ er es durch.

**`init --dry-run` zeigt die `settings.json` als Diff, nicht als ganze
Datei.** Die Ausgabe ist ein Zeilendiff je Datei (`--- <pfad>`, `-`/`+`/`  `).
Bei `iam_backend` stehen die alten Befehle (`ulguard`, `ulguard post-edit`,
`uv run … ultraloom hook subagent-start|stop`, `brain wiki-gate`) als `-` und
dann wieder als `+`, jeweils neben dem neuen `loomux.exe hook …`. Der Diff
richtet die Zeilen neu aus, entfernt wird nichts. Die Hinweise sagen es
ausdrücklich: „SessionStart/ keeps a hook of the project beside ours“, dasselbe
für SubagentStart und SubagentStop. `init` lässt die alten Einträge also
stehen, und Schritt 6 des Skripts (`dev switchover prune-hooks`) wird
gebraucht. `ecoflow` hat keine `settings.json`; der Trockenlauf legt dort nur
`.loomux/config.toml` (eine `pip`-Regel), `.gitignore` und `AGENTS.md` an.

**Module und Guard-Modus von `init --yes`.** Laut Trockenlauf von `iam_backend`
laufen die Aktionen `area-add` (`--scope project/iam_backend`, Wiki
`docs/wiki`), `merge-hook` und `graph-build`, dazu die Hooks für Claude Code
und Antigravity. Ein Abschnitt `[guard]` wird nicht geschrieben. Nach
`internal/config/guardsettings.go` (`GuardModes`, der Default steht zuerst)
gilt damit `default`, wie vom Nutzer vorgegeben. Nebenbefunde: Antigravity
bekommt aus einem Dev-Build (`0.0.0-dev`) keine Einträge („run a released
loomux init“); ollama lief nicht, `yt-dlp` fehlt im PATH.

**`area add` ändert keinen vorhandenen Bereich.** `config.AddArea`
(`internal/config/registrywrite.go`) lehnt einen schon registrierten Scope mit
„scope … is already registered“ ab, und `planArea` setzt `Workspace: true`
fest. `readonly`, `wiki` oder `workspace` eines registrierten Bereichs ändert
es nicht. Für `space`, `iam_wiki`, `ecoflow`, `ultra-brain` und `ultraloom`
braucht es also die Registry-Datei aus Schritt 2 des Skripts.

**Session-Hooks der `iam_*`.** `uv run --project <projekt>/.ultraloom/vendor/ultraloom ultraloom --help`:

| Projekt | Ergebnis |
|---|---|
| `iam_backend` | **nicht messbar:** `.ultraloom/vendor` fehlt, uv meldet „Project directory … does not exist“ (Exit 2). Die `settings.json` ruft den Pfad trotzdem auf, die Session-Hooks laufen dort heute also ins Leere |
| `iam_frontend` | läuft. Das vendorte Submodul hat schon vorher Änderungen in sieben Dateien (`cmd/guard/post_edit.go`, `src/ultraloom/cli.py`, `commit/*.py`, `process.py`); gemessen wird also dieser Stand, nicht ein Release |
| `iam_workers` | läuft. Der erste Aufruf hat `.venv` im Vendor-Ordner angelegt (45 Pakete). Die ist ignoriert, `git status` bleibt leer. **Der Schritt war nicht rein lesend** |

**Wikis: Inhalt und Überschneidung** (Python-Skript, `filecmp` byteweise; `.git` ausgenommen):

| Ziel | Vault `91 Projekte/…` | Zustandsverzeichnis | Projekt `docs/wiki` | Projekt `wiki` | gleiche Namen, abweichend |
|---|---|---|---|---|---|
| `ecoflow` | 5 (Gerüst) | fehlt | 4, versioniert seit `ee1616d` (2026-09-06) | fehlt | Vault∩Projekt 4 gleich benannt, abweichend: `index.md` |
| `space` | 5 (Gerüst) | 72 (68 md) | 217 (204 md) | fehlt | Vault∩Projekt: `index.md`, `log.md`; Vault∩Zustand: `_identities.tsv`, `index.md`; Zustand∩Projekt: `index.md` |
| `iam_wiki` | 5 (Gerüst) | 31 (27 md), **nur erzeugte Ordner-`index.md`** (`architecture/`, `projects/backend/apis/` …) ohne eigene Seite | 4, versioniert seit `5a116cc` (2026-09-06) | fehlt | Vault∩Projekt: `index.md`; Vault∩Zustand: `_identities.tsv`, `index.md` |
| `ultra-brain` | fehlt | fehlt | 33 (32 md) | fehlt | — |
| `ultraloom` | 5 (Gerüst) | fehlt | 10 (4 md), versioniert seit `9106da8` | fehlt | Vault∩Projekt 5 gleich benannt, 4 abweichend: `_identities.tsv`, `_schema.md`, `audit.md`, `index.md` |

Nachmessung am selben Tag, weil das Skript Pfade ab der Wurzel jedes Orts
verglich und die Unterbäume `wiki/` und `docs/wiki/` des Zustandsverzeichnisses
so nie namentlich gegen das Projekt hielt:

- `space`: `project-space/wiki` (11 Dateien) und `project-space/docs/wiki`
  (11) tragen nur `index.md`-Kataloge je Ordner, dieselben 11 Ordner, die
  `space/docs/wiki` hat. Die beiden Kataloge unterscheiden sich untereinander
  (`cmp`, Zeile 1). Es ist keine eigene Seite darunter.
- `iam_wiki`: Die 27 Markdown-Dateien im Zustandsverzeichnis sind erzeugte
  Kataloge (`## Dateien` mit Links, z. B. `projects/backend/decisions/index.md`,
  422 Bytes). Die Seiten, auf die sie zeigen, liegen **im Wurzelverzeichnis des
  Projekts** (`iam_wiki/projects/backend/decisions/api-permission-discrepancies.md`):
  106 Markdown-Dateien außerhalb von `docs/`, davon 80 versioniert; die übrigen
  26 sind ignorierte Arbeitspapiere unter `.superpowers/`. Das Wiki von
  `iam_wiki` ist also das Projekt selbst, nicht `docs/wiki`.

Befund gegen die Spec: Die Spec-Tabelle sagt „`iam_wiki` … **nein**“ (kein Wiki
im Projekt) und für `ecoflow` „ein Ordner `wiki` fehlt“. Beide Projekte haben
aber seit dem 2026-09-06 ein versioniertes `docs/wiki`-Gerüst. In **keinem**
Vault-Ordner und in keinem Zustandsverzeichnis liegt eine Seite, die im Projekt
fehlt und kein Gerüst ist: Die einzige echte Wikisammlung ist `space/docs/wiki`,
und die liegt schon im Projekt; bei `iam_wiki` liegen die Seiten im
Wurzelverzeichnis des Projekts. `WikiDst` ist `<projekt>/docs/wiki` für
`ecoflow`, `space`, `ultra-brain` und `ultraloom`; für `iam_wiki` entscheidet
der Nutzer zwischen Wurzelverzeichnis und `docs/wiki`. `WikiSrcs` trägt je Ziel höchstens das Vault-Gerüst.
Dessen einzige Datei, die im Projekt fehlt, ist `_identities.tsv` (bei
`ecoflow`, `space` und `iam_wiki`). Die abweichenden Gerüstdateien (`index.md`,
`log.md`, bei `ultraloom` auch `_schema.md` und `audit.md`) entscheidet der
Nutzer je Datei, sonst bricht Schritt 3 des Skripts mit `differs:` ab.
Nicht gemessen: ob sich `_identities.tsv` des Vaults und die des
Zustandsverzeichnisses nur in `doc_id` unterscheiden oder in mehr.

### 8. Pilot `ecoflow` (2026-09-29)

Umgestellt mit dem gerenderten `apply.sh` (ecoflow-Commit `7cd2597`),
gemessen mit `dev bench hooks -n 5` gegen die Baseline aus Messung 7. Der
Vergleich steht in `bench-4e-ecoflow.md`.

**Ablauf der Umstellung.**

- `apply.sh` gab in der PowerShell des Nutzers keine Ausgabe aus. Die
  Wirkungen sind trotzdem nachgeprüft: neue `.claude/settings.json` mit den
  sechs loomux-Hooks, `.loomux/config.toml`, geänderter `wiki`-Pfad in der
  Registry.
- Das pre-commit-Tor von ecoflow verweigerte den Umstellungs-Commit aus drei
  Gründen: ruff meldet 83 Befunde, die schon vorher im Code standen; das
  mypy-Preset lief ohne Ziel; `coverage` war keine Abhängigkeit des Projekts.
  Die letzten beiden behebt `b111e14d` auf diesem Zweig. Der Nutzer hat mit
  `--no-verify` committet.
- Das zweite `apply.sh --check` nach der Umstellung ist noch nicht gelaufen.

**Messung.** Die neuen Fälle entstehen aus der neuen `settings.json` mit
derselben `--file` (`README.md`) und denselben Namen der Zusatzfälle
(`status`, `search (fast)`, `wiki lint`), jetzt über `loomux brain …`; neu
dazu `graph stats` und `graph check`. `dev bench cases` setzt
`${CLAUDE_PROJECT_DIR}` ein, `${LOCALAPPDATA}` aber nicht; der erste Lauf
brach mit `fork/exec ${LOCALAPPDATA}/loomux/bin/loomux.exe` ab. Die Variable
ist in `cases.json` von Hand durch `C:/Users/micro/AppData/Local` ersetzt.
Gemessen wurde die installierte Binary `loomux 5.3.0 (beta)`.

- Der Stop-Hook endet mit Exit 2: er fährt das Tor von ecoflow, und das fällt
  an den 83 ruff-Befunden. Das ist erwartet und kein Befund der Umstellung.
- `status` braucht warm 5,2 s gegen 43 ms bei `brain status` des Altstands,
  weil `loomux brain status` qmd abfragt. `wiki lint` braucht 158 statt
  49 ms, `search (fast)` warm 122 statt 236 ms, kalt 2,7 s, weil der erste
  Aufruf den Suchdienst anläuft.
- `git status --porcelain --untracked-files=all` von ecoflow war vor und nach
  der Messung gleich. Die uncommitteten Dateien des Nutzers (`auth.py`,
  `endpoints.py`, `_identities.tsv`, `docs/index.md`, `graph.json`,
  `layout.json`) blieben unberührt; kein Hook schrieb eine Datei, die git
  sieht.

**Was der Pilot beweist.** Das Skript stellt ein Projekt ohne alte Hooks
um, Registry und Deklaration lesen sich danach, und alle sechs Hooks laufen
im Projekt mit Exit 0 bis auf den Stop-Hook, der am Vorbestand fällt.

**Was er nicht beweist.** ecoflow hatte keine `settings.json` und keine alten
Hooks. Nicht geprüft sind darum das Entfernen alter Hook-Einträge, das
Bereinigen einer bestehenden `settings.json` und ein Vergleich alter gegen
neue Hooks desselben Ereignisses; im Vergleich erscheinen alle Hooks als
„neu“. Das leisten erst die Ziele mit alten Hooks.

## Checkliste

**Stand 2026-09-29:** Block 4 (Deklarationen von Hand) und Block 5 (`init`, alte Einträge) sind durch den Ablauf der Spec `2026-09-29-loomux-stufe-4e-umstellung-vorbereitet-design.md` abgelöst: das LLM bereitet vor, ein `apply.sh` schreibt. Block 1 bis 3 und der Rauchtest (Block 6) gelten weiter.


Der Mensch hakt ab, kein Agent. Die Blöcke gelten in dieser Reihenfolge.
Befehle sind für den `!`-Präfix in Git Bash geschrieben (Vorwärtsschrägstriche,
`/c/…`-Pfade); eine nötige Antwort steht per Pipe davor (`echo n | …`). Ein
`!`-Befehl bekommt keine Eingabe und keine Konsole: was eine echte Konsole
braucht (TUI, Raw-Modus, zum Beispiel `loomux init` interaktiv), geht **nicht**
mit `!`; das öffnest du im eigenen Terminal. Die Befehle laufen aus dem
Wurzelverzeichnis von loomux mit `bin/loomux.exe`; ein Pfad mit `#` steht in
Anführungszeichen.

### Block 1: Voraussetzungen

- [ ] `brain-knowledge` hat einen Remote und ist committet
      (`git -C "/c/Users/micro/Documents/#GIT/brain-knowledge" remote -v` und
      `git -C "/c/Users/micro/Documents/#GIT/brain-knowledge" status --short`
      zeigen einen Remote und nichts Offenes).
- [ ] Die Menschenschritte von 4a-2 (Zeile 4a-2 in `docs/en/migration.md`) sind
      gelaufen: `init --yes` auf einem frischen Klon, danach ein leerer
      `git status`; ein Wirt interaktiv eingerichtet (im eigenen Terminal, nicht
      mit `!`); ein Projekt-`.mcp.json` einmal in Claude Code freigegeben.

### Block 2: Vorher entscheiden

- [ ] `[index]` für `project/loomux` eintragen. Vorschlag aus
      `parity/stufe-3a.md`, Abschnitt „Umstieg: `project/loomux` braucht ein
      `[index]`“: `include = ["docs/wiki/**/*.md"]` (32 Dokumente, keine
      versionierte Datei geändert, vier neue Dateien). **Du** trägst es in
      `.loomux/config.toml` ein; ein Agent schreibt diese Datei nie
      (`AGENTS.md`).
- [ ] Festlegen, welche der vier erzeugten Dateien versioniert werden:
      `_identities.tsv`, `graph.json`, `index.md`, `docs/index.md`. Die
      Abwägung steht am selben Ort in `stufe-3a.md` (`_identities.tsv` spricht
      fürs Versionieren, die drei anderen sind ableitbar).
- [ ] Die Entscheidungen aus `parity/artefakte-nach-lebensdauer.md`, die
      Zustandsdateien berühren, zuerst treffen, weil sie den Maschinenzustand
      ändern (Messung Nr. 5): **#1** (Ort von `graph.json` und Wurzelkatalog),
      **#3** (Form von `_identities.tsv`, mit Aliasen, Grabsteinen, Union-Merge)
      und **#8** (frühere Indexausgabe räumen; hängt an #1).
      #2, #6 und #9 berühren weder Zustandsdateien noch Manifest und müssen
      nicht davor fallen.

### Block 3: Maschinenzustand

Quelle ist das alte Verzeichnis `/c/Users/micro/AppData/Local/brain`, Ziel das
neue `/c/Users/micro/AppData/Local/loomux` (beide Verzeichnisse liegen am
2026-09-29 so vor). Das alte bleibt als Sicherung liegen; nichts darin wird
gelöscht, und geschrieben wird nur ins neue. Verglichen wird von Hand oder mit
`diff`; ein Unterschied heißt: die jüngere Datei bleibt, die ältere wird
nachgetragen, wenn ihr Inhalt fehlt. Im Zweifel nichts überschreiben und den
Fund notieren.

- [ ] `registry.toml`: `! diff /c/Users/micro/AppData/Local/brain/registry.toml
      /c/Users/micro/AppData/Local/loomux/registry.toml`. Am 2026-09-29 trägt
      die neue Registry schon dieselben zehn Bereiche wie die alte, dazu
      `project/loomux` (Vergleich der Einträge ohne Kommentare: der einzige
      Unterschied ist dieser Eintrag). Messung Nr. 2 nannte die neue Registry
      zuerst „Vorlage mit einem Eintrag“; das war falsch (Berichtigung dort).
      Erwartet: nichts
      nachzutragen. Ein weiterer Unterschied ist zu klären, bevor es
      weitergeht.
- [ ] Identitätsregister der drei read-only-Bereiche:
      `! diff -rq /c/Users/micro/AppData/Local/brain/areas /c/Users/micro/AppData/Local/loomux/areas`.
      Die alte Seite hat `project-iam-wiki`, `project-obsidian-ai` und
      `project-space`, die neue dieselben drei plus die `*.lock`-Dateien
      (`_identities.tsv` liegt je Bereich darin). Die neuen Ordner stammen vom
      26.09. und weichen ab (etwa `_identities.tsv` von `project-space`: 68823
      Bytes alt, 69144 neu); ein alter Stand wird nicht zurückkopiert, wenn der
      neue jünger ist.
- [ ] `maintenance/last-run.txt`: beide Verzeichnisse haben sie; ansehen und
      vergleichen (`! cat` je Datei).
- [ ] `maintenance/merge-events.done.tsv` und, solange ein Python-Hook noch
      schreibt, `maintenance/merge-events.tsv`: ein `find` am 2026-09-29 fand
      in keinem der beiden Verzeichnisse eine Datei `merge-events*`.
      Nachsehen, ob sich das geändert hat; wenn ja, aus dem alten ins neue
      kopieren, was im neuen fehlt.
- [ ] `qmd-collections.json` liegt im Wurzelverzeichnis des Zustands
      (`/c/Users/micro/AppData/Local/brain/qmd-collections.json`, neu:
      `…/loomux/qmd-collections.json`). Nur dann kopieren, wenn im neuen **keine**
      liegt. Am 2026-09-29 liegt dort eine (26.09.), also **nicht** kopieren:
      eine dort geschriebene ist die jüngere. Ohne sie beginnt die Liste leer,
      und der erste `reindex` verweigert jede Sammlung, die qmd schon führt.

### Block 4: Deklarationen der zehn Bereiche

**Welche Bereiche read-only sind.** Die alte Registry
(`/c/Users/micro/AppData/Local/brain/registry.toml`) setzt `readonly = true` bei
genau drei der zehn: `project/space`, `project/iam-wiki` und
`project/obsidian-ai`. Die anderen sieben sind schreibbar. (Die neue Registry
hat für diese drei dieselben Einträge, Vergleich in Block 3.)

**Wo die Deklaration liegt, und wer sie liest.** Die Regel ist
`config.ManifestDir` (`internal/config/manifest.go`) und, mit Rückfall,
`config.ResolvedAreaDir` (`internal/config/artifacts.go`):

- Ein **schreibbarer** Bereich hat seine Deklaration in seinem eigenen Baum.
  Ziel: `<bereich>/.loomux/config.toml`.
- Ein **read-only** Bereich hat sie **nicht in seinem Baum**, sondern im
  Zustandsverzeichnis: `/c/Users/micro/AppData/Local/loomux/areas/<flacher Scope>/`
  (flacher Scope: jede Folge von Zeichen außerhalb von `A-Za-z0-9_.-` wird ein
  Bindestrich, also `project-space`, `project-iam-wiki`, `project-obsidian-ai`).
  Ziel: `…/areas/<flacher Scope>/.loomux/config.toml`. Die Ordner gibt es am
  2026-09-29 schon (Stand vom 26.09.), und sie tragen heute ein wörtlich
  kopiertes `.brain.toml`.
- `guard` (`internal/brain/guard/registry.go`, `manifestPath`, `checkDeclaration`)
  liest für einen read-only Bereich **nur** diese Stelle und **nur** den Namen
  `.loomux/config.toml`; für einen schreibbaren `<bereich>/.loomux/config.toml`.
  Eine Deklaration, die fehlt, ist kein Fehler: der Bereich wird übersprungen,
  die Schranke bleibt blind für ihn (`parity/stufe-3a.md`, „Stufe 4: `guard` muss
  die Deklaration eines read-only-Bereichs finden“).
- Die brain-Leser (`config.ResolvedAreaDir` mit
  `config.ReadAreaManifestUntilStage4`) lesen dieselbe Stelle und nehmen die
  Namen in der Reihenfolge `.loomux/config.toml`, `.ultra-brain/config.toml`,
  `.brain.toml`. Der erste Name, der eine Datei ist und ein `[area]` hat,
  gewinnt. Liegt `.loomux/config.toml` dort, wird ein daneben liegendes
  `.brain.toml` nicht mehr gelesen.
- `reindex` schreibt für einen read-only Bereich den ganzen Ordner neu und
  tauscht ihn ein (`internal/brain/index/staging.go`, `publish`): es kopiert
  den vorhandenen Ordner samt allen Unterordnern erst nach `staging`, schreibt
  darin und tauscht. Eine dort abgelegte `.loomux/config.toml` bleibt dabei
  erhalten. Der Baum des Bereichs selbst wird für einen read-only Bereich nicht
  beschrieben.
- Was danach im Baum des read-only Bereichs (etwa `space/.brain.toml`) liegt,
  liest für die Deklaration keiner mehr, sobald der Ordner unter `areas/`
  existiert.

Je Bereich bis zu vier Schritte, die **schreibbaren** und die **read-only** haben
verschiedene Ziele (unten je Bereich genannt):

1. `! bin/loomux.exe area check "<verzeichnis>"` (nur lesend; es nimmt jedes
   Verzeichnis, gemessen für das Zustandsverzeichnis von `project-space`,
   `project-iam-wiki` und `project-obsidian-ai` am 2026-09-29).
2. Die Deklaration **von Hand** ins Ziel legen (`.loomux/config.toml`). Kein
   Agent schreibt diese Datei; lege sie nicht mit `loomux area add` an, das
   schreibt in die `.loomux/config.toml` des Projekts, nicht in ein
   Zustandsverzeichnis.
3. `! bin/loomux.exe area check "<verzeichnis>"` erneut: die Zeile `chosen:`
   nennt `.loomux/config.toml` (unter Windows druckt der Befehl
   `.loomux\config.toml`, mit Rückwärtsschrägstrich). Damit findet `guard` sie
   und nimmt sie an, sofern `[layout] inbox` relativ ist (`checkDeclaration` in
   `internal/brain/guard/registry.go` ruft zusätzlich `InboxLayout()`, das einen
   absoluten Wert ablehnt; keines der zehn Manifeste trägt `inbox`), denn
   `area check` wählt mit derselben Namensreihenfolge wie die brain-Leser, und
   `guard` liest denselben Ort. Ein danebenliegendes `.brain.toml` erscheint als
   `shadowed`, wenn es selbst ein `[area]` trägt (`internal/cli/areacheck.go`).
   **Exit 0 gibt es erst, wenn die alte Datei weg ist.** `area check` liest
   jede vorhandene Manifestdatei und zählt einen ignorierten Schlüssel auch in
   der überdeckten Datei (etwa `readonly = true` im `.brain.toml`). Solange sie
   liegt, ist Exit 1 mit Zeilen ausschließlich dieser Datei (`.brain.toml: …`
   und `shadowed`) erwartet und **kein Fehler**, wenn `chosen:` stimmt. Danach
   Schritt 4.
4. **Alte Datei entfernen oder umbenennen**, nur bei den Bereichen, wo Schritt 3
   Exit 1 wegen der überdeckten Datei meldet (die sechs unten mit „Exit 1“):
   `.brain.toml` bzw. `.ultra-brain/config.toml` löschen oder umbenennen, dann
   `area check` ein drittes Mal: Exit 0. Für die drei read-only-Bereiche ist es
   das kopierte `.brain.toml` im Zustandsordner (Block 5 räumt dort nichts,
   auch nicht im Host `space`, dessen Baum-`.brain.toml` ein anderes ist). Das
   ist **Handarbeit des Menschen**: dieser Agent schreibt in keinem dieser Bäume.
   Nur bei `brain-knowledge` räumt Block 5 (einer der vier Wirte) die Datei
   ohnehin; dort ist der dritte Lauf erst nach Block 5 grün, vorher ist Exit 1
   erwartet. Für `ultra-brain` und `ultraloom` (keine Wirte) gibt es keinen
   anderen Schritt, der es tut; `ecoflow`, `engineering/*` und `hub` melden
   schon vorher Exit 0 und brauchen ihn nicht.

Schlüssel, die `area check` als „ignored“ meldet, gehen beim Umlegen nicht mit,
weil kein Leser sie liest (Messung Nr. 6, am 2026-09-29). Du entscheidest je
Schlüssel: weglassen oder ein Gegenstück eintragen. **Zuerst die drei
read-only-Bereiche**, weil `guard` sie sonst nicht sieht: er liest für sie nur
`areas/<flacher Scope>/.loomux/config.toml`, und dort liegt heute keine. Die
schreibbaren folgen. Die Bereiche und ihr Befund:

**Read-only (Ziel im Zustandsverzeichnis).** Prüfpfad und Ziel:
`/c/Users/micro/AppData/Local/loomux/areas/<flacher Scope>`. Was `area check`
dort am 2026-09-29 sagt (Exit 1 in allen dreien): der Ordner trägt das
wörtlich kopierte `.brain.toml`, und darin steht `[area] readonly = true`, ein
Schlüssel, den `area check` als „ignored“ meldet; die Read-only-Eigenschaft
gilt aus der Registry, nicht aus dem Manifest. Beim Umlegen weglassen (deine
Entscheidung). Inhalt sonst aus dem `.brain.toml` daneben, dessen Schlüssel alle
„read“ sind. **Nach Schritt 3 bleibt Exit 1**, solange das kopierte `.brain.toml`
im Zustandsordner liegt (`readonly` bleibt dort ignoriert, dazu die Zeile
`shadowed`); das ist erwartet. Erst Schritt 4 (das kopierte `.brain.toml`
entfernen oder umbenennen) bringt Exit 0. Sicherung ist die Registry und die
Datei im Baum daneben.

- [ ] `project/space` (Host `space`): Ziel
      `/c/Users/micro/AppData/Local/loomux/areas/project-space/.loomux/config.toml`.
- [ ] `project/iam-wiki` (Baum `iam_wiki`): Ziel
      `/c/Users/micro/AppData/Local/loomux/areas/project-iam-wiki/.loomux/config.toml`.
- [ ] `project/obsidian-ai` (Baum `#Obsidian/AI`): Ziel
      `/c/Users/micro/AppData/Local/loomux/areas/project-obsidian-ai/.loomux/config.toml`.

**Schreibbar (Ziel im Baum, `<bereich>/.loomux/config.toml`).** Der Befund
ist der von Messung Nr. 6 am Baum, mit der `.brain.toml` bzw. der
`.ultra-brain/config.toml` im Baum als Vorlage:

- [ ] `knowledge` = `brain-knowledge` (`C:/Users/micro/Documents/#GIT/brain-knowledge`,
      `.brain.toml`): Exit 1. Ignoriert: `area.wiki`, `layout.sources`,
      `llm.local` (Hinweis auf `[model]`), `maintenance.stale_after_days`,
      `maintenance.watch`. Nach Schritt 3 bleibt Exit 1 (nur Zeilen des
      überdeckten `.brain.toml`, plus `shadowed`); Exit 0 erst nach Schritt 4,
      hier durch Block 5.
- [ ] `project/ultra-brain` (`.brain.toml`, Pfad
      `C:/Users/micro/Documents/#GIT/ultra-brain` laut alter Registry):
      Exit 1. Ignoriert: `check.lanes` (Hinweis auf `[verify]`). Der leere
      Ordner `.ultra-brain/` wird übergangen. Nach Schritt 3 bleibt Exit 1
      (`.brain.toml`, `shadowed`); Exit 0 erst, wenn du das `.brain.toml` von
      Hand entfernst oder umbenennst (Schritt 4, Bereich ohne Wirt).
- [ ] `project/ultraloom` (`.ultra-brain/config.toml`, es gibt kein
      `.brain.toml`; Pfad `C:/Users/micro/Documents/#GIT/ultraloom` laut alter
      Registry): Exit 1. Ignoriert: `area.wiki`, `layout.sources`,
      `maintenance.merge_branch`. Nach Schritt 3 bleibt Exit 1
      (`.ultra-brain/config.toml`, `shadowed`); Exit 0 erst, wenn du diese Datei
      von Hand entfernst oder umbenennst (Schritt 4, Bereich ohne Wirt).
- [ ] `project/ecoflow` (`.brain.toml`, Pfad
      `C:/Users/micro/Documents/#GIT/ecoflow` laut alter Registry): Exit 0,
      alle Schlüssel gelesen.
- [ ] `engineering/python`
      (`C:/Users/micro/Documents/#GIT/brain-knowledge/92 Engineering/python`):
      Exit 0.
- [ ] `engineering/craft`
      (`C:/Users/micro/Documents/#GIT/brain-knowledge/92 Engineering/craft`):
      Exit 0.
- [ ] `hub` (`C:/Users/micro/Documents/#GIT/brain-knowledge/91 Projekte`):
      Exit 0.

(Am Baum melden `iam_wiki`, `space` und `#Obsidian/AI` Exit 0; das ist für die
read-only-Bereiche nicht die Stelle, die `guard` liest.)

**Hinweis zu den Bereichen unter `brain-knowledge`** (`92 Engineering/python`,
`92 Engineering/craft`, `91 Projekte`): bekommen sie eine eigene
`.loomux/config.toml`, nimmt `hosts.FindRoot(".")` (`internal/hosts/hostio.go`,
läuft vom Arbeitsverzeichnis aufwärts bis zur ersten `.loomux/config.toml`) aus
einem ihrer Unterordner diese als Projektwurzel statt der von `brain-knowledge`.
Das betrifft `config`, `check`, `convert` und `mcp` ohne `--root`, wenn du sie
aus einem solchen Unterordner rufst; die Claude-Hooks übergeben `--root` und
sind nicht betroffen.

Kein Bereich meldete am 2026-09-29 `shadowed`, `refused` oder eine Datei ohne
`[area]`. Die Ausgabe von heute ist der Ausgangsstand; melde jede Abweichung.

**Offene Fragen an den Menschen (aus dem Code nicht zu klären):**

- [ ] Braucht der read-only Host `space` zusätzlich ein `[area]` in seiner
      eigenen `.loomux/config.toml` im Baum (Block 5 legt sie an), oder ist es
      dort verkehrt? `manifestPath` nennt für einen read-only Bereich den
      Baum ausdrücklich „the tree the barrier does not own“. Ob `init` in einem
      read-only Baum ein `[area]` wünscht oder nur die Policy anlegt, ist
      nicht gemessen.
- [ ] Bleibt eine im Zustandsverzeichnis abgelegte `.loomux/config.toml` über
      den nächsten `reindex` des Bereichs erhalten? Aus dem Code folgt ja
      (`copyTree` kopiert alle Dateien nach `staging`, `writeStock` schreibt nur
      Kataloge, `graph.json` und `_identities.tsv`); gelaufen ist es nicht.
      Nach dem ersten `reindex` je read-only Bereich: Schritt 3 wiederholen.

**Warum Block 4 vor Block 5:** `init` ersetzt `.loomux/config.toml` nie; es
hängt Blöcke an oder ändert einzelne Zeilen, ein von Hand eingetragenes `[area]`
bleibt erhalten (Messung Nr. 1, an einer Wegwerfkopie mit `init --dry-run`).
**Lücke:** nicht gemessen sind `init` ohne `[area]` in der Datei und ein Lauf
ohne `--dry-run`. Der erste echte Lauf je Wirt ist die Probe; im Irrtumsfall ist
der Schaden ein von Hand nachzutragender `[area]`-Block.

### Block 5: Je Wirt

Die Wirte sind `space`, `iam_backend`, `ecoflow` und `brain-knowledge`. Je Wirt:

`iam_frontend`, `iam_workers` und `ultraloom` gehören **nicht** zu den vier
Wirten (die Stufe-4e-Spec, Abschnitt „Ziel“, nennt genau `space`,
`iam_backend`, `ecoflow`, `brain-knowledge`).

Je Wirt:

- [ ] `loomux init` im **eigenen Terminal**, nicht mit `!` (der interaktive
      Lauf braucht eine Konsole, ein `!`-Befehl hat keine Eingabe): erst
      `cd "<wirt>"`, dann `loomux init` (mit dem Binary aus
      `/c/Users/micro/Documents/#GIT/loomux/bin/loomux.exe`, wenn `loomux` nicht
      auf dem `PATH` liegt). Die Pfade der vier Wirte stehen in der Akte nicht
      alle; `brain-knowledge` ist
      `C:/Users/micro/Documents/#GIT/brain-knowledge`, `space` und `ecoflow`
      stehen in der alten Registry unter `C:/Users/micro/Documents/#GIT/space`
      und `…/ecoflow`; `iam_backend` ist dort **nicht** eingetragen, sein Pfad
      ist dir bekannt.
- [ ] Alte Einträge von Hand entfernen: Marke `ultraLoomOwned`, Gruppe
      `ultraloom-wiki-guard`, Einträge von `brain guard` und `ulguard`.
- [ ] Danach die alten Ordner und Dateien im Baum entfernen: `.ultraloom/`,
      `.brain.toml`, `.ultra-brain/`. Ein schreibbarer Wirt braucht dafür schon
      `<wirt>/.loomux/config.toml` mit `[area]` (Block 4); der read-only Host
      `space` hat seine Deklaration im Zustandsverzeichnis, sein Baum-`.brain.toml`
      liest keiner mehr.

Danach **einmal**, nach Block 3 und nach dem letzten Wirt (nicht je Wirt):

- [ ] `! bin/loomux.exe merge-hook install`: er wirkt rechnerweit auf die
      Registry, kein `--root`. Er setzt den post-merge-Hook in jedem Repository
      jedes Bereichs, dessen Manifest `[maintenance] on_merge = true` sagt
      (`README.md`); die Registry muss die Bereiche dafür kennen (Block 3), und
      die Deklarationen der Wirte sollen dann stehen. Ein von
      `brain-mcp` gesetzter Hook gilt als `unrecorded`, weil die alte
      `hooks.tsv` nicht gelesen wird; `! bin/loomux.exe merge-hook status` zeigt
      es danach.

### Block 6: Rauchtest je Wirt

Je Wirt (`space`, `iam_backend`, `ecoflow`, `brain-knowledge`):

- [ ] Ein erlaubter Edit gelingt: in Claude Code, im Wirt gestartet, eine
      gewöhnliche Quelldatei des Wirts ändern lassen (eine Datei, die nicht
      `.loomux/config.toml` ist).
- [ ] Ein verweigerter Edit wird verweigert: dieselbe Sitzung soll
      `<wirt>/.loomux/config.toml` schreiben. Der Wächter verweigert jedes
      Schreibwerkzeug auf diese Datei (`AGENTS.md`, Regeln; gemessen in
      Messung Nr. 1). Hinweis: ein Wirt, dessen `.loomux/config.toml` noch
      fehlt, hat keine Regel, die das belegt.
- [ ] Ein Commit läuft durch commit-msg und pre-commit: im Wirt einen Commit
      mit einer Nachricht machen, die gegen die Regeln des Wirts verstößt
      (die Regeln des Wirts stehen in seiner `[commit]`-Tabelle; welche
      Regel welcher Wirt hat, steht in der Akte nicht), und einen gültigen.
      Erwartet: der erste wird abgelehnt, der zweite läuft durch das
      pre-commit-Tor. **Offen:** die Akte belegt weder die Hooks der Wirte
      noch ihre Regeln; die Erwartung ist aus dem Aufbau von `loomux init`
      gelesen, nicht gemessen.
- [ ] Eine Suche über MCP liefert Treffer: in Claude Code, im Wirt gestartet,
      das Tool `brain_search` mit einer Frage aufrufen, deren Antwort im
      Bereich des Wirts steht. Als Gegenprobe im Terminal
      `! bin/loomux.exe brain search "<frage>"` (durchsucht die sichtbaren
      Bereiche über den qmd-Daemon, `README.md`). Voraussetzung ist ein
      freigegebenes Projekt-`.mcp.json` (Block 1).

## Überlebende Mutanten

**Die Runde für `loomux area check` (2026-09-29).** Sie deckt nur Stück A,
`internal/cli/areacheck.go` (`areaCheck`, `fileDeclaresArea`,
`checkOneManifest`, `flattenKeys`, `classifyKeys`, `schemaKnows` und die
Tabellen `manifestNames` und `legacyHints`). Die Runde für den Aufräum-PR
(Stück C) steht noch aus und kommt in denselben Abschnitt, wenn dessen Code
steht.

**Warum nicht `loomux dev mutants`.** Das Werkzeug gibt jedem Mutanten eine
feste Grenze (`goTimeout = "60s"`, `internal/dev/mutants/mutants.go:61`), lässt
sich nicht auf einzelne Tests eingrenzen (Schalter nur `-family`, `-only`,
`-workers`) und zählt einen Zeitüberlauf als getötet. Die volle Suite
`go test ./internal/cli -count=1` brauchte am 2026-09-29 **87 s** (gemessen,
Ausgabe in eine Datei), also mehr als die Grenze: jeder unbemerkte Mutant wäre
in die Zeitgrenze gelaufen und als getötet gezählt worden, und ein Lauf des
Werkzeugs über das Paket hätte fast keine Überlebenden gemeldet. Das wäre ein
Tor, das grün meldet, ohne etwas gemessen zu haben.

**Methode.** Eine eigene Runde von Hand: je Mutant eine Kopie von
`areacheck.go` mit genau einer Änderung, eingespielt per
`go test -overlay` (der Baum bleibt unberührt), gegen die gezielten Tests
`-run 'TestAreaCheck|TestClassifyKeys|TestFlattenKeys|TestLegacyHints'
-count=1 -failfast` (2 s statt 87 s, weit unter der Grenze; kein Mutant lief in
einen Zeitüberlauf). Ein Mutant gilt als getötet, wenn ein Test mit
`--- FAIL` rot wird; ein Mutant, der nicht übersetzt, wurde umgeschrieben, bis
er übersetzt (`_ = x`, `&& false`), sonst hätte er nichts gemessen. Die
Mutationen: Vergleichsoperatoren gedreht, `&&` und `||` vertauscht, jede
Teilbedingung einzeln gestrichen, jeder Rückgabewert und jeder Exit-Code
(0, 1, 2) geändert, `continue` zu `break` oder gestrichen, Anfangswerte
gekippt, Ausgabetexte geändert, das Sortieren gestrichen (mit `-count=30`,
weil die Reihenfolge einer Map zufällig ist), die Einträge von
`manifestNames` vertauscht oder gestrichen und jeder der sieben Einträge von
`legacyHints` gestrichen.

| Funktion oder Tabelle | Mutanten |
|---|---:|
| `areaCheck` | 46 |
| `checkOneManifest` | 23 |
| `schemaKnows` | 14 |
| `flattenKeys` | 13 |
| `classifyKeys` | 10 |
| `legacyHints` | 7 |
| `fileDeclaresArea` | 5 |
| `manifestNames` | 4 |
| **zusammen** | **122** |

**Erste Runde: 122 Mutanten, 11 davon nicht übersetzbar (danach
umgeschrieben, sodass sie übersetzen, und nochmals gefahren), einer wegen eines
falschen Suchtexts im Skript zunächst übersprungen und nachgeholt, 8
überlebten.** Sieben davon zeigten echte Lücken; zu jeder ist
ein Test nachgelegt, der gegen den Mutanten rot läuft (Beleg per Overlay,
zweite Runde):

| Mutant | Lücke | Test, der ihn tötet |
|---|---|---|
| `fileDeclaresArea` gibt `err == nil` statt `!errors.Is(err, ErrNoArea)` zurück | Eine Datei mit kaputtem `[area]` (Fehler, aber nicht `ErrNoArea`) deklariert trotzdem einen Bereich und wird als überdeckt gemeldet; kein Test hatte so eine Datei | `TestAreaCheckCallsAShadowedDeclarationThatIsBrokenShadowed` |
| `checkOneManifest`: `errors.Is(err, ErrNoArea) && policyOnlyIsFine` zu `policyOnlyIsFine` | Ein `.loomux/config.toml` mit kaputtem `[area]` hieße „policy only“ statt „refused“ | `TestAreaCheckRefusesALoomuxConfigWhoseAreaIsBroken` |
| Schlüsselzeile trägt `"x"` statt des Dateinamens | Kein Test las den Namen in der Schlüsselzeile | `TestAreaCheckAcceptsAFullyReadManifest` (prüft jetzt `<name>: area.scope  read`) |
| `schemaKnows`: `Contains(id, "verify.")` statt `HasPrefix` | `xverify.a` würde als gelesen gelten | `TestClassifyKeysDoesNotTakeAnotherTableForVerify` (zweiter Fall) |
| `schemaKnows`: `HasPrefix(key.ID(), id)` ohne den Punkt | `agent.mod` gälte als bekannt, weil `agent.models` so anfängt | `TestClassifyKeysKnowsOnlyWholeSegmentsOfASchemaKey` |
| `schemaKnows`: `Contains(key.ID(), id+".")` statt `HasPrefix` | ein Schlüssel `models` gälte als bekannt, weil `agent.models.…` ihn in der Mitte trägt | `TestClassifyKeysKnowsOnlyWholeSegmentsOfASchemaKey` |
| `manifestNames`: `.ultra-brain` und `.brain.toml` vertauscht | die Ausgabe folgt nicht der Reihenfolge, in der der Leser die Namen probiert | `TestAreaCheckLetsUltraBrainShadowBrainToml` (prüft die Reihenfolge der Zeilen) |

**Nach dem Nachlegen: 122 Mutanten, 121 getötet, 1 überlebt.** Alle Funktionen
von `areacheck.go` stehen weiter bei 100 % Coverage.

**Bleibt stehen (äquivalent).**

| Mutant | Warum er dasselbe tut |
|---|---|
| `flattenKeys`: das `continue` nach `ids = append(ids, name)` gestrichen | Fällt die Schleife durch, ist `table` bei einem Nicht-Table-Wert `nil` und bei einer leeren Tabelle leer; `for key := range table` läuft dann null Mal, es entsteht keine weitere ID. Das `continue` spart nur die Schleife, es ändert nichts an der Ausgabe |

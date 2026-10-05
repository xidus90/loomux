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

### 9. Pilot `space` (2026-09-30)

Umgestellt mit dem gerenderten `apply.sh` (space-Commit `46e04b9f`), gemessen
am 2026-09-30 mit `dev bench hooks -n 5` gegen die Baseline vom 2026-09-29
(Messung 7). Der Vergleich steht in `bench-4e-space.md`.

**Ablauf der Umstellung.**

- Die Ausgabe von `apply.sh` war im PowerShell-Fenster des Nutzers
  abgeschnitten. Die Wirkungen hat der Controller nachgeprüft, nicht die
  Ausgabe gelesen.
- space behält sein eigenes `.githooks/pre-commit`; es hat den
  Umstellungs-Commit `46e04b9f` angenommen, ohne `--no-verify`.
- Das zweite `apply.sh --check` zeigte nur `already` und `kept`, dazu für
  `init` und `prune-hooks` „would be run“.
- `graph build` fand 0 Dateien, weil GDScript nicht indiziert wird;
  `graph stats` meldet 0 Dateien, 0 Symbole, 0 Kanten.

**Was der Pilot über ecoflow hinaus beweist.**

- Die alten Hooks sind entfernt: `prune-hooks` nahm 7 Gruppen aus der
  `settings.json` (`ulguard`, `brain guard`, `brain wiki-gate`,
  `ultraloom hook …`). Übrig ist neben den sechs loomux-Hooks nur space'
  eigener `run.sh session_start.py`.
- Eine bestehende `settings.json` ist bereinigt, nicht neu geschrieben.
- Alte und neue Hooks desselben Ereignisses stehen nebeneinander (unten).
- Das Bereichsverzeichnis im Zustand heißt jetzt `project-space.alt`.
- `readonly` ist aus dem Registry-Eintrag gefallen, `workspace` blieb.

**Messung.** Die neuen Fälle entstehen aus der neuen `settings.json` mit
derselben `--file` (`AGENTS.md`) und denselben Namen der Zusatzfälle
(`commit-msg`, `status`, `search (fast)`, `wiki lint`), die drei letzten
jetzt über `loomux brain …`; neu dazu `graph stats` und `graph check`;
`python entry (ultraloom --help)` fällt weg. `dev bench cases` aus HEAD
(`2c637567`) setzt `${LOCALAPPDATA}` jetzt selbst ein, `cases.json` ist
unverändert gemessen. Gemessen wurde die installierte Binary
`loomux 5.3.1 (beta)` (ecoflow: 5.3.0), seriell, ein Lauf von 41 s
(08:08:58–08:09:39 Ortszeit).

Die Alt-Seite des Vergleichs ist eine zusammengesetzte Datei
(`before-merged.json`): die erste Baseline (`bench-2026-09-29-1901`), darin
die drei Fälle `SubagentStart`, `SubagentStop` und `Stop` durch die Einträge
der Nachmessung ersetzt (`baseline-2`, `bench-2026-09-29-1931`), weil die
erste dort nur den Ablehnungspfad maß. Kein Wert ist gerechnet oder
geschätzt, jeder Eintrag ist ganz aus einer der beiden Dateien übernommen.

Umgebung, anders als am Vortag: Der qmd-Dienst auf Port 8765 lief nicht
(Baseline: lief die ganze Zeit); stattdessen lief ein
`loomux serve --foreground` und fünf `loomux mcp` offener Sitzungen.

| Fall | Median alt | Median neu | Exit alt | Exit neu |
|---|---:|---:|---|---|
| SessionStart | 456,8 ms | 251,4 ms | [0] | [0 0] |
| PreToolUse | 48,4 ms | 9,6 ms | [0 0] | [0] |
| PostToolUse | 52,2 ms | 13,2 ms | [0] | [0] |
| SubagentStart | 1765,4 ms | 1112,6 ms | [0] | [0] |
| SubagentStop | 1820,6 ms | 8,0 ms | [0] | [0] |
| Stop | 608,8 ms | 408,3 ms | [1 1] | [2] |
| commit-msg | 488,1 ms | 262,1 ms | [0] | [0] |
| status | 208,1 ms | 2113,9 ms | [0] | [0] |
| search (fast) | 197,6 ms | 76,1 ms | [0] | [0] |
| wiki lint | 47,8 ms | 169,0 ms | [0] | [1] |

- **`commit-msg` ist die Kontrolle.** Der Hook ist auf beiden Seiten
  derselbe (`.githooks/commit-msg` von space, `commit_language.py`) und
  läuft doch 1,9-mal so schnell. Der Rechner war am zweiten Tag also
  schneller oder ruhiger; ein Faktor bis etwa 2 belegt für sich nichts.
  `SessionStart` (2,2-fach) liegt in diesem Rauschen, zumal dort jetzt zwei
  Befehle nebeneinander laufen (`run.sh session_start.py` und
  `loomux hook session-start`) statt einem.
- **`PreToolUse` 48 → 10 ms, `PostToolUse` 52 → 13 ms** (Faktor 5 und 4,
  über dem Rauschen): ein loomux-Prozess statt `ulguard` plus `brain guard`.
- **`SubagentStop` 1821 → 8 ms warm, kalt 1089 ms.** Nur der erste Lauf
  arbeitet sichtbar; ob die warmen Läufe abkürzen, weil der Schnappschuss
  schon verbraucht ist, ist nicht nachgestellt. Der warme Wert ist darum
  kein Maß für die Arbeit des Hooks; der kalte ist es eher.
- **`Stop` Exit 2**, einmal von Hand nachgestellt: `lint/gdscript` fällt,
  weil `uvx gdlint .` kein Paket `gdlint` findet („gdlint was not found in
  the package registry“; das Werkzeug liegt im Paket `gdtoolkit`);
  `test/gdscript` meldet `missing-tool`, weil `godot` nicht auf dem PATH
  steht (dort liegt nur `Godot_v4.7.1-stable_mono_win64.exe`); `lint/wiki`
  meldet denselben einen Befund wie das alte `brain wiki-gate`
  (`open-questions/renderbudget-ungemessen.md` ohne `type`). Zwei der drei
  Ursachen sind also Befunde der Umstellung (das Preset), die dritte
  Vorbestand. Beide Befunde des Presets sind auf diesem Zweig behoben, siehe
  Punkt 2 unten. Der alte Stop fiel mit `[1 1]` an fehlenden `[verify]`-Spuren
  und demselben Wiki-Befund; keine Seite hat eine Prüfkette wirklich
  gefahren, die Zeiten vergleichen zwei Arten zu scheitern.
  Einer der fünf warmen Läufe dauerte 10 ms: nach drei Blockaden in Folge
  (`MaxBlocks`) gibt der Hook einmal auf. Der Bericht hält nur `[2]` fest.
- **`wiki lint` Exit 1 statt 0:** `loomux brain check bundle --scope
  project/space` meldet 790 Fehler (370 `house/source-incomplete`, 323
  `house/dead-link`, 92 `house/unknown-type`, 3 `house/no-sources`, je 1
  `okf/catalog-malformed` und `okf/frontmatter-unparsable`); das alte
  `brain check bundle` gab am selben Stand still Exit 0. Viele tote Links
  zeigen auf `docs/.superpowers/…` und werden relativ zur Seite aufgelöst.
- **`status` 208 → 2114 ms**, wie bei ecoflow. Es meldet außerdem
  „project/space: never indexed; run `brain reindex`“.
- **`search (fast)`** warm 76 statt 198 ms, kalt 11,5 s.
- **Was die Hooks schrieben:** `git status --porcelain
  --untracked-files=all` von space war vor und nach der Messung gleich (nur
  `?? export_presets.cfg` des Nutzers, unberührt). Neu und ignoriert:
  `.loomux/state/hooks/loomux-bench.json` (Basis `46e04b9f`, `blocks: 3`).
  Unter `%LOCALAPPDATA%\loomux` und in `brain-knowledge` änderte sich nichts.

**Was space gelehrt hat** (vor der Welle zu ändern, je als eigener Task mit
Test):

1. `apply.sh --check` kann nicht sagen, ob `init` und `prune-hooks` etwas
   ändern würden; es sagt nur „would be run“. Ein zweiter Lauf ist damit
   nicht als leer zu erkennen.
2. Das GDScript-Preset lief auf diesem Rechner in keiner Spur: `uvx gdlint`
   findet kein Paket dieses Namens (das Werkzeug liegt im Paket `gdtoolkit`),
   und `godot` heißt auf dem PATH anders. Der Stop-Hook blockierte damit jede
   Sitzung in space bis zur dritten Blockade. Beides ist auf diesem Zweig
   behoben: `4b5854f8` lässt die Lint-Spur `uvx --from gdtoolkit gdlint`
   fahren, und `35509182` streicht das Test-Preset für GDScript; die Spur
   `test/gdscript` gilt als nicht anwendbar, bis ein Projekt
   `[verify.gdscript.test]` selbst setzt. Für space bleibt: das installierte
   loomux 5.3.1 hat beide Fehler noch, bis ein Release mit den Korrekturen
   installiert ist. Bis dahin schweigt die Stop-Sperre nur mit der Marke
   `.loomux/no-verify`, die der Mensch setzt.
3. Vier Stücke der alten Konfiguration haben keinen Platz im Schema:
   `[relevance]` (Markdown-Änderungen liefen leer durch), die
   Coverage-Schwelle, `[gates]` (`tests_in_stop`, `types_in_stop`,
   `[gates.wiki]`) und `docs_language`. Das Skript verliert sie still.
4. `wiki lint` wechselt mit der Umstellung von Exit 0 auf 790 Fehler. Vor
   der Welle ist zu klären, welche Hausregeln für ein Projekt-Wiki gelten
   und wie die Auflösung relativer Links gemeint ist; sonst sieht jedes
   umgestellte Wiki kaputt aus.
5. Nach der Umstellung ist der Bereich „never indexed“; das Skript fährt
   kein `reindex`. Ob der `reindex` von `hub` gelaufen ist, hat diese
   Messung nicht geprüft.
6. `graph build` indiziert GDScript nicht; `init` schaltet `graph` trotzdem
   ein und der Graph bleibt leer. `init` sollte das sagen oder das Modul
   auslassen.
7. Die Ausgabe von `apply.sh` ging im PowerShell-Fenster zum zweiten Mal
   verloren (ecoflow: gar keine, space: abgeschnitten). Das Skript sollte
   sein Protokoll zusätzlich in eine Datei schreiben.
8. Der Bericht von `dev bench hooks` hält je Fall eine Liste von Exit-Codes,
   nicht je Lauf; ein Lauf, der anders endet (das Aufgeben des Stop-Hooks),
   ist nur an der Zeit zu erkennen. Und ein Stop-Fall mit fünf Läufen läuft
   in die Schleifensperre, misst also gemischt.
9. `compare` paart einen unveränderten Fall (`commit-msg`) mit Faktor 1,9:
   zwei Messtage sind ohne Kontrollfall nicht vergleichbar. Die Welle sollte
   je Ziel einen unveränderten Kontrollfall mitführen oder Vorher und
   Nachher am selben Tag messen.
10. `.gitignore` von space behält die toten Zeilen `.ultraloom/hooks/` und
    `.ultraloom/vendor/`; das Skript räumt sie nicht auf.

### 10. Die Welle (2026-10-01/02)

**Ziele und Abweichungen vom Plan.**

- **`ultra-brain` und `ultraloom` sind nicht umgestellt.** Entscheidung des
  Nutzers vom 2026-10-01: beide Repos werden nach der Migration gelöscht. Ihre
  Baselines vom 2026-09-29 bleiben unter `.superpowers/switchover/` liegen,
  ohne Nachmessung. Die Welle umfasste damit `iam_backend`, `iam_frontend`,
  `iam_workers`, `iam_wiki` und `brain-knowledge`.
- **`--brain=none` registriert keinen Bereich und installiert keinen
  Merge-Hook.** Der Plan (Task 11) ging davon aus, dass `init` die `iam_*`
  registriert; unter `--brain=none` gehören Bereich und Merge-Hook zum
  Brain-Modul und entfallen. Folge, gemessen am 2026-10-02: Die Schreibschranke
  verweigert jeden Edit in `iam_backend`, `iam_frontend` und `iam_workers`
  („lies outside every writable tree“, Exit 2), siehe unten.
- **Vendorte ultraloom-Formen.** Die eigenen `.githooks` der `iam_*` riefen
  `uv run ultraloom check all` und `ulinit`. `iam_backend` und `iam_workers`:
  Variante A, `apply.sh` entfernt die beiden alten Git-Hooks, ein zweites
  `init` (vom Menschen, der Wächter verweigerte es dem Agenten) legt die von
  loomux an. `iam_frontend`: husky wich `.githooks` (H2): `core.hooksPath`
  `.githooks`, `scripts.prepare` und husky entfernte der Mensch per `npm`.
  Das vendorte Submodul trug sieben ungesicherte Änderungen; sie sind als
  Patch gesichert und verworfen.
- **`iam_wiki` ist das gemeinsame Wiki der drei `iam_*`-Code-Projekte**, kein
  eigenes Projekt. Umgestellt nur als Wiki-Bereich mit eigenem Skript
  `apply-wiki.sh` (die Vorlage kann `init` nicht auslassen): Registry-Eintrag
  `wiki` auf die Wurzel, `readonly` entfällt (sonst Sperrzone), minimale
  `.loomux/config.toml`, `_identities.tsv` aus dem Zustand übernommen
  (byte-gleich, 94 `doc_id`), `.brain.toml` entfernt, Zustandsbereich `.alt`;
  kein `init`, keine Hooks, keine Lanes. `[layout] wiki` lehnt die Wurzel ab,
  darum gibt es dort **keine Wiki-Lane** (Folgepunkt).
- **Der Vault** (`brain-knowledge`): vier Deklarationen, `init`,
  `on_merge = true`, eigenes Skript `apply-vault.sh`. Der erste Commit fiel am
  neuen pre-commit: die Arten `types`, `test`, `coverage`, `graph` des Profils
  `precommit` haben im Vault „nothing to check“ und enden mit Exit 1. Der
  Mensch setzte `[verify.profiles] precommit = ["lint"]` von Hand, weil
  `config set verify.profiles.precommit` den Schlüssel nicht kennt. Danach
  entfernte ein zweiter PR die Gerüste
  `91 Projekte/{ecoflow,space,iam-wiki,ultraloom}`.
- Die Piloten `ecoflow` und `space` bekamen `gate disarm --all` und einen
  pre-commit-Hook mit `--arm` (vom Menschen).

**Nachmessung** am 2026-10-02, 12:35–12:40, seriell, mit der installierten
Binary `loomux 6.1.0 (beta)` für Hooks und `dev bench`. Fälle aus der neuen
`settings.json` mit derselben `--file` wie in der Baseline (`README.md` bei
`iam_backend`, sonst `AGENTS.md`), Zusatzfälle mit denselben Namen.
Umgebung: qmd-Dienst auf 8765 lief (wie bei der Baseline), dazu
`loomux serve` und zehn `loomux mcp`; ein `strata.exe --serve` hielt rund
45 GB, CPU-Last 24 %. Berichte: `bench-4e-<ziel>.md`. Die Alt-Seite der
`iam_*` ist wie bei `space` zusammengesetzt (`before-merged.json`), die
übernommenen Fälle nennt der jeweilige Bericht.

| Ziel | Fall | Median alt | Median neu | Exit alt → neu |
|---|---|---:|---:|---|
| `iam_backend` | PreToolUse | 41,5 ms | 13,0 ms | [0] → [2] |
| `iam_backend` | PostToolUse | 64,5 ms | 30,0 ms | [0] → [0] |
| `iam_backend` | commit-msg | 98,9 ms | 24,1 ms | [0] → [0] |
| `iam_backend` | status | 105,1 ms | 3 191,8 ms | [0] → [0] |
| `iam_frontend` | SessionStart | 221,4 ms | 81,0 ms | [0] → [0] |
| `iam_frontend` | SubagentStop | 1 898,3 ms | 9,6 ms | [0] → [0] |
| `iam_frontend` | Stop | 232,8 ms | 178,7 ms | [1] → [0] |
| `iam_workers` | PreToolUse | 43,0 ms | 12,0 ms | [0] → [2] |
| `iam_workers` | search (fast) | 206,6 ms | 70,7 ms | [0] → [0] |
| `iam_wiki` | wiki lint | 47,5 ms | 81,1 ms | [0] → [1] |
| `brain-knowledge` | wiki lint | 47,0 ms | 41,4 ms | [0] → [1] |

- **`PreToolUse` Exit 2 in allen drei `iam_*`-Code-Projekten:** der Wächter
  verweigert den Edit, weil `--brain=none` keinen Bereich registriert und die
  Wurzel damit in keinem beschreibbaren Baum liegt. Einzeln nachgestellt
  (`hook pre-tool-use` mit der Nutzlast). Das ist ein Befund der Umstellung,
  kein Rauschen: Ein Agent kann in diesen Projekten heute nicht schreiben.
- **`PreToolUse` Exit 2 im Vault** auf `AGENTS.md`: Die Wurzel des Vaults ist
  kein beschreibbarer Baum, nur die vier Bereichsordner; `AGENTS.md` liegt
  daneben. Erwartet.
- **`Stop` Exit 1 im Vault:** dieselbe Ursache wie beim Commit („nothing to
  check for `types`, `test`, `coverage`, `graph`“), diesmal im Stop-Profil,
  das keine Überschreibung hat.
- **`Stop` von `iam_frontend`:** kalt 27,6 s (Lint und Typen über das Projekt),
  warm 179 ms: die Warmläufe treffen eine unveränderte Basis.
- **`wiki lint` Exit 1:** `iam_wiki` meldet Hausregeln (`unknown-type`,
  `source-incomplete`, `no-sources`), der Vault `house/unlisted-area` für die
  Bereiche, auf die `knowledge/index.md` nicht verweist. Die alte Prüfung gab
  an beiden Ständen Exit 0.
- **`status` ist überall um den Faktor 30 bis 70 langsamer** (rund 3,1–3,3 s
  warm), wie bei den Piloten: es fragt qmd ab.
- **Kein Kontrollfall.** `commit-msg` ist in den `iam_*` nicht mehr derselbe
  Hook (loomux statt des alten Skripts); zwischen Baseline und Nachmessung
  liegen drei Tage. Faktoren bis etwa 2 belegen für sich nichts (Messung 9,
  Punkt 9).
- **Nicht gemessen:** das pre-commit-Tor (Ruling 4, auf keiner Seite);
  `ultra-brain` und `ultraloom` (nicht umgestellt).
- **Was die Messung in den Projekten schrieb:** `git status --porcelain
  --untracked-files=all` war in allen fünf Zielen vor und nach der Messung
  gleich. Vorbestand, unberührt: `iam_frontend` `?? .eslintcache`,
  `?? .ultraloom/` (die Bench-Datei der Baseline), `iam_wiki`
  ` M _identities.tsv`, ` M graph.json`, `brain-knowledge`
  ` M .loomux/config.toml` (nur das Zeilenende hinter
  `precommit = ["lint"]`). Neu und ignoriert:
  `.loomux/state/hooks/loomux-bench.json` in `iam_backend`, `iam_frontend`,
  `iam_workers` und `brain-knowledge`.

**Zuordnung für den anonymisierten Bericht** (`docs/*/benchmarks.md`; nur
hier): Beispielprojekt 1 `ecoflow`, 2 `space`, 3 `iam_backend`,
4 `iam_frontend`, 5 `iam_workers`, 6 `iam_wiki`, 7 `brain-knowledge`.

**Befunde für loomux (Folgepunkte, ohne Stufe in der Fusions-Spec):**

1. Eine Art eines Profils, für die es nichts zu prüfen gibt, lässt
   `check precommit` und den Stop-Hook mit Exit 1 fallen; ein Projekt ohne
   Code (der Vault) kann so nie committen und keine Sitzung beenden, ohne das
   Profil von Hand zu kürzen. **Erledigt am 2026-10-05:** Eine Art aus `all`
   oder einem eingebauten Profil ohne Lane wird ausgelassen, solange eine
   Lane einer anderen angefragten Art lief (Fusions-Spec #28, Status). Das
   von Hand gekürzte Profil des Vaults (`precommit = ["lint"]`) bleibt
   gültig; ab hier ist es nicht mehr nötig.
2. `verify.profiles` ist über `config set` nicht setzbar (unbekannter
   Schlüssel); der Vorschlagsweg aus AGENTS.md greift dafür nicht.
3. `init --brain=none` lässt ein Projekt ohne Bereich; die Schreibschranke
   verweigert dann jeden Edit darin. Für die `iam_*` fehlt ein Bereich
   (oder ein `workspace`-Eintrag ohne Bündel).
4. `[layout] wiki` lehnt die Wurzel eines Repos ab; ein Wiki-Repo wie
   `iam_wiki` hat damit keine Wiki-Lane.
5. `dev bench cases` lehnt eine `settings.json` ohne Hook ab, statt nur die
   Zusatzfälle zu schreiben.

**Stand der Mutationsrunden:** die der neuen Pakete und CLI-Funktionen der
Welle lief am 2026-10-02, die des Aufräum-PR am 2026-10-04; beide stehen im
Abschnitt „Überlebende Mutanten“ unten (siehe auch Abschnitt 11).

### 11. Der Aufräum-PR (2026-10-04)

Gezählt am 2026-10-03 in zwei Scratch-Worktrees auf `714d4b27`, vor dem
Schreiben des Plans:

- **Altmanifeste in den Lauf-Bäumen:** in allen zwölf Fallsuiten genau
  **zwei** (`3a/area-add/known-scope/world_after/repo-new/.ultra-brain/config.toml`,
  eine erwartete Datei, die loomux nicht schreibt; und
  `4d/convert/no-registry/world/vault/.brain.toml`, ein Fall, der vor dem Lesen
  eines Manifests endet). Die `*-worlds`-Bäume sind Aufnahmequellen, nicht
  Lauf-Bäume.
- **Übersetzer beim Bereitstellen:** 0 Fälle. **Freigegebene Abweichungen:**
  0 Fälle. Mit nur `.loomux/config.toml` und ohne Altverzeichnis laufen alle
  Fallsuiten (`TestRecordedCasesOfStage1a`, `…1b1`,
  `TestRecordedMCPCasesOfStage1b2`, `TestCases2a` bis `4d`) grün.
- **Ratschlagsfälle:** **12**, nicht zehn: neben den zehn der Spec trägt
  `brain-status/backlog` in 1b-1 und 1b-2 den Ratschlag `brain embed`. Nach dem
  Re-Import mit den neuen Karten ändern sich genau 6 + 6 Dateien.
- **Ausgang der Auflagen aus `stufe-3a.md`:** Ratschläge getragen; Asides und
  read-only-Deklarationen entfallen; `merge-events.done.tsv` und
  `qmd-collections.json` fallengelassen (Nutzer, 2026-10-03).

Die Zeiten der Leser ohne das Altverzeichnis stehen im Eintrag
„Reading Without the Old State Directory“ von `docs/en/benchmarks.md` und
`docs/de/benchmarks.md`.

### 12. Rauchtest Block 6, lesend geprobt (2026-10-04)

Mit `loomux 7.0.1 (beta)` aus `%LOCALAPPDATA%\loomux\bin` gegen die vier Wirte
`space`, `iam_backend`, `ecoflow` und `brain-knowledge`. Die Wächterproben
schicken nur eine Write-Nutzlast an `loomux hook pre-tool-use --host claude
--root <wirt>`, sie schreiben nichts; die `commit-msg`-Proben rufen
`.githooks/commit-msg` jedes Wirts mit einer Nachrichtendatei auf, ohne zu
committen.

| Probe | space | iam_backend | ecoflow | brain-knowledge |
|---|---|---|---|---|
| Write auf eine gewöhnliche Datei | Exit 0 | Exit 0 | Exit 0 | Exit 0 unter `91 Projekte`; `README.md` an der Wurzel Exit 2, „outside every writable tree“, wie die Registry es will |
| Write auf `.loomux/config.toml` | Exit 2 | Exit 2 | Exit 2 | Exit 2 |
| `commit-msg` „docs: Änderung der Übersicht für die Prüfung“ | abgelehnt | abgelehnt | abgelehnt | abgelehnt |
| `commit-msg` „kaputte nachricht ohne typ“ | **angenommen** | abgelehnt | abgelehnt | abgelehnt |
| `commit-msg` „chore: smoke test“ | angenommen | angenommen | angenommen | angenommen |

`loomux brain search` von der Befehlszeile lieferte Treffer aus `project/space`
und `project/loomux`. Nicht lesend probbar und offen für den Menschen: die
Suche über MCP (`brain_search`) in einer Claude-Code-Sitzung im Wirt und das
pre-commit-Tor bei einem echten Commit (das Tor ist inzwischen gefahren,
siehe „Echte Commits“ unten).

**Befund `space`.** Der Wirt läuft beim Commit-Text nicht über loomux:
`.githooks/commit-msg` ruft das projekteigene
`.claude/hooks/commit_language.py` über `run.sh`, nicht `loomux check
commit-msg`. Es lehnt nur eine deutlich deutsche Nachricht ab; die anderen
drei Wirte lehnen auch „kaputte nachricht ohne typ“ ab. Der Umstellungscommit
`46e04b9f` (2026-09-29) hat `.githooks/commit-msg` nicht angefasst, `init`
lässt einen eigenen Hook stehen. Ebenso startet `.claude/settings.json` von
`space` neben `loomux hook session-start` weiter
`bash .claude/hooks/run.sh session_start.py`. Das `pre-commit` von `space`
fährt die eigenen Python-Prüfungen und danach
`loomux check precommit --arm`, läuft also über loomux. Ob `commit-msg` und
`session_start.py` auf loomux umgestellt werden oder als Projekthooks bleiben,
entscheidet der Nutzer.

**Entschieden am selben Tag.** `commit-msg` ist umgestellt: `space`
`4cbfed6a` auf `main` ruft `loomux check commit-msg`; `commit_language.py`
und seine Tests sind entfernt. `session_start.py` bleibt, weil loomux keine
seiner vier Aufgaben trägt; daraus wird eine Roadmap-Zeile (Fusions-Spec
Nachtrag #30).

**Suche über MCP (am selben Tag).** `brain_search` mit „commit message
language gate“ aus einer Claude-Code-Sitzung im loomux-Worktree lieferte zehn
Treffer, neun davon aus `project/space` (oben `decisions/sprachregel.md`).
Damit antwortet der MCP-Dienst mit Treffern aus einem umgestellten Wirt;
gefahren ist es nicht aus einer Sitzung *im* Wirt, wie Block 6 es verlangt.
Der Index zeigte dabei noch den Stand vor `4cbfed6a` (ohne den Nachtrag in
`sprachregel.md`).

**Suche über MCP aus den Wirten (am selben Tag).** Je Wirt eine
Claude-Code-Sitzung (`claude -p`, Modell haiku), im Wirt gestartet, die nur
`brain_search` rufen darf. Der Dienst kommt aus der Nutzerkonfiguration
(`~/.claude.json`, `loomux mcp --channel local`), steht also in jeder Sitzung.

| Wirt | Frage | Ergebnis |
|---|---|---|
| `space` | „Godot user dir override per worktree“ | drei Treffer aus `project/space` (`dev-setup.md`, `parallele-arbeitsbereiche.md`, `log.md`) |
| `ecoflow` | „EcoFlow client authentication“ | drei Treffer aus `project/ecoflow` |
| `brain-knowledge` | „Python engineering conventions“ | drei Treffer, aus `project/ecoflow` und `project/space`, keiner aus dem Tresor selbst |
| `iam_backend` | „Django task status view“ | kein `brain_search` in der Sitzung, nur die `graph_*`-Werkzeuge: `[modules] brain = false`, wie bei der Welle mit `init --brain=none` gewollt |

**Befund `space`, `.mcp.json`.** Die versionierte `.mcp.json` von `space`
trägt weiter einen Server `brain`, der `uv run … --directory
C:/Users/micro/Documents/#GIT/ultra-brain brain mcp` startet. Er stammt aus
der Zeit vor der Umstellung; sobald `ultra-brain` gelöscht ist (Entscheidung
vom 2026-10-01), startet er nicht mehr. `iam_backend` trägt in seiner
`.mcp.json` nur fremde Server (`postgres`, `context7`), `ecoflow` und
`brain-knowledge` haben keine.

**Echte Commits durchs pre-commit-Tor (am selben Tag).** Je Wirt ein leerer
Commit `chore: smoke test` auf einem Wegwerfzweig `smoke-test`, danach zurück
auf den Ausgangszweig und den Wegwerfzweig gelöscht.

| Wirt | Ergebnis |
|---|---|
| `ecoflow` | Commit `cb43685` entstand (vom Nutzer gefahren); die Ausgabe des Tors ist nicht festgehalten |
| `space` | Commit `ba107d7c` entstand nach dem vollen Tor: Python-Gate übersprungen (nichts gestaged), GDScript-Suite, dann `loomux check precommit --arm`. Ein zweiter, gleichzeitiger Versuch fiel an der Sperre der Suite (`.suite-lock`) ab, ohne etwas zu ändern |
| `iam_backend` | Commit `97215d0` entstand; `lint/python` ok, `types/python` ok (14,9 s), `test/python` rot nach 1113,4 s, `coverage/python` blockiert — beide nur Warnung, weil sie in der Schonfrist stehen |
| `brain-knowledge` | Commit `deab81e` entstand; `lint/wiki` ok |

**Befund `iam_backend`.** Alle 3936 Tests von `test/python` scheitern beim
Aufsetzen an `django.db.utils.OperationalError: connection timeout expired`:
auf `localhost:5432` lauscht kein PostgreSQL. Der Hook arbeitet richtig; der
Rechner hat keine Testdatenbank laufen. Solange das so bleibt, wird die Lane
nie grün, die Schonfrist endet nie, und jeder Commit kostet rund 19 Minuten.

### 13. Selbstnutzungsprobe nach dem Aufräum-PR (2026-10-05)

Die letzte offene Fertig-Bedingung der Aufräum-Spec: `brain catalog` und
`brain status` an der echten Registry, mit `loomux 7.0.1 (beta)` aus
`%LOCALAPPDATA%\loomux\bin`, am 2026-10-05 um 11:41.

| Befehl | Exit | Zeit | Ergebnis |
|---|---|---|---|
| `loomux brain catalog` | 0 | 0,15 s | zehn Bereiche; die `iam_*`-Arbeitsbereiche ohne `[area]` fehlen, wie es sein soll |
| `loomux brain status` | 0 | 5,2 s | letzter Abgleich 2026-10-05T08:47Z; nur Inhaltshinweise (Links, nicht eingebettete Seiten, gleiche Inhalte unter mehreren Pfaden), kein Fehler |

Kein Bereich meldet ein Altmanifest oder „no manifest found“. In der Registry
steht noch der Probebereich `project/loomux-area-probe` unter
`AppData/Local/Temp`; ihn zu entfernen ist ein Schritt für den Menschen.
Damit sind alle Fertig-Bedingungen erfüllt, und 4e ist ✅. Die Folgezeilen
der Roadmap hängen nicht an 4e (Aufräum-Spec, „Nicht Teil davon“).
Ob die Lane eine laufende Datenbank voraussetzt, eine eigene startet oder
ohne Datenbank laufen soll, entscheidet der Nutzer.

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

**Abgelöst (Stand 2026-10-04, ohne Häkchen).** Die Umstellung hat den neuen
Zustand unter `…/Local/loomux` selbst angelegt und fortgeschrieben; das alte
Verzeichnis `…/Local/brain` bleibt als Sicherung. Ein `diff -rq` beider am
2026-10-04 zeigte, was danach zu erwarten ist: `project-space` und
`project-iam-wiki` liegen im neuen nur noch als `.alt`, `project-obsidian-ai`
weicht ab, `last-run.txt` ist im neuen jünger (2026-10-03), und die
`merge-events*`-Dateien gibt es nur im neuen. Nichts davon wird
zurückkopiert. Die Kästchen unten bleiben leer; der Mensch hakt sie ab oder
streicht sie.

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

**Abgelöst (Stand 2026-10-04, ohne Häkchen).** Seit 2026-09-29 durch den
Ablauf mit `apply.sh` (siehe den Hinweis über Block 1); die Welle vom
2026-10-01/02 hat ihn für alle Ziele gefahren (Messung 10), `ultra-brain` und
`ultraloom` werden nicht umgestellt.

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

**Abgelöst (Stand 2026-10-04, ohne Häkchen).** Wie Block 4: `apply.sh` fährt
`init --yes` und `dev switchover prune-hooks` und löscht die alten Dateien;
`merge-hook install` lief nach dem letzten Ziel am 2026-10-02.

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
Tabellen `manifestNames` und `legacyHints`). Die Runde des Aufräum-PR (Stück
C, 2026-10-03/04) steht unten im Unterabschnitt „Die Runde des Aufräum-PR“.

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

**Die Runde für Vergleich, Fallbau und Umstellung (2026-10-02).** Sie deckt
`internal/dev/benchcompare`, `internal/dev/benchcases`, `internal/switchover`
(`prune.go`, `render.go`, `apply.sh.tmpl`) und die CLI-Funktionen von
`dev bench compare`, `dev bench cases` und `dev switchover render|prune-hooks`
in `internal/cli/benchcompare.go` und `internal/cli/switchover.go`. Gemessen
gegen **12726817**; die Tests, die sie nachlegt, stehen in 3c176b97,
d85956ba, cc5a02ce und 2452afdc. Binary: `loomux 0.0.0-dev`, gebaut aus
12726817 (`git describe`: v6.1.0-3-g12726817; ein Build aus dem Baum trägt
keine Versionsnummer).

**Frühere Runden.** Das Ledger nennt Runden der Implementierer (Task 2: 80
Mutanten, Task 3: 101, Task 4: 87, Task 5: 88, Task 6: 119 und nach Fixrunde 1
151), gefahren gegen die Commits vor dem Rebase (738798d4, 8b97ee74,
1e2bd82d/b89646b2, f0321c5b, ee498614/b85d4ff8). Keine davon steht in dieser
Akte, und `apply.sh.tmpl` hat seither Schritt 4 (armed.toml) bekommen; die
Runde ist darum ganz wiederholt, nicht nur nachgeprüft.

**Zeiten der unveränderten Suiten** (Ausgabe je in eine Datei):
`benchcompare` 0,2 s, `benchcases` 0,2 s, `switchover` 68 s (davon
`TestApply` 64 s, `TestPruneHooks` 0,2 s, `TestRender|TestTheScript` 0,3 s),
`internal/cli` gezielt (`TestDevBench|TestBench|TestWriteAndClose|TestWriteNewFile|TestDevSwitchover`)
0,4 s.

**Methode.** `loomux dev mutants` setzt heute die Grenze je Mutant auf das
Dreifache der unveränderten Suite, mindestens 60 s (`boundFor`, `minBound` in
`internal/dev/mutants/round.go`), weist Zeitüberläufe gesondert aus und zählt
einen Mutanten, der nicht übersetzt, als „no mutant“. Es erzeugt aber nur die
Familien a1–a4 (Bedingung eines `if` gestrichen oder verneint, Operanden von
`&&`/`||`, Vergleichsoperatoren). Darum: `dev mutants` für a1–a4 bei den
beiden kurzen Paketen (8 Arbeiter × 0,2 s weit unter der Grenze; kein
Zeitüberlauf), eine Handrunde für alle übrigen Arten (Rückgabewerte,
Exit-Codes 0/1/2, `continue`/`break`, Anfangswerte, Ausgabetexte,
Tabelleneinträge, jede Teilbedingung einzeln) und für `switchover` und die
CLI-Funktionen ganz von Hand: `switchover` braucht 68 s, also lief jeder
Mutant als `go test -overlay` gegen die gezielten Tests (Overlay-Pfade in der
Form `C:/…`). Die Mutanten der Vorlage liefen über `LOOMUX_SWITCHOVER_TEMPLATE`
auf Kopien im Scratchpad gegen `TestApply`; eine unveränderte Kopie überlebte
wie erwartet, 53 getötete Mutanten belegen, dass die Kopie gelesen wird.
Überlebte ein `render.go`-Mutant die gezielten Tests, lief er noch gegen
`TestApply`, also gegen die ganze Suite des Pakets. Ein Mutant, der nicht
übersetzte, wurde umgeschrieben, bis er übersetzte (`_ = x`, `&& false`,
`true ||`), und nie als getötet gezählt. Nie liefen zwei Runden zugleich.

| Paket oder Datei | Mutanten | getötet | äquivalent oder begründet | umgeschrieben, damit sie übersetzen |
|---|---:|---:|---:|---:|
| `benchcompare` (`dev mutants` 37 übersetzbar + Hand 64) | 101 | 100 | 1 | 9 |
| `benchcases` (`dev mutants` 79 übersetzbar + Hand 89) | 168 | 168 | 0 | 11 |
| `switchover/prune.go` | 85 | 81 | 4 | 3 |
| `switchover/render.go` | 151 | 146 | 5 | 4 |
| `switchover/apply.sh.tmpl` | 55 | 53 | 2 | 0 |
| `cli/benchcompare.go` | 91 | 91 | 0 | 1 |
| `cli/switchover.go` | 49 | 47 | 2 | 1 |
| **zusammen** | **700** | **686** | **14** | **29** |

Drei `render.go`-Mutanten (`VaultOld` immer bereinigt, `INIT_ARGS` mit `|`
oder leer) überlebten die Render-Tests und fielen erst in `TestApply`; sie
zählen als getötet. Zwei Blöcke der Mutantenliste mit leerem Ersatztext hatte
das Skript zunächst verschmolzen; sie sind mit `// dropped` als Ersatz neu
gefahren, das Skript prüft seither, dass jeder Kopf geparst wird.

**Echte Lücken, je mit einem Test, der gegen den Mutanten rot läuft (Beleg
per Overlay) und auf dem Code grün ist:**

| Mutant | Lücke | Test, der ihn tötet | Commit |
|---|---|---|---|
| `benchcases.Build`: Fehler von `split` übergangen | Der Fall fiel trotzdem, nur als „a hook names no command“; der Test fragte nur nach dem Ereignis | `TestBuildRefusesWhatItCannotMeasure` (verlangt `Stop: unterminated`) | 3c176b97 |
| `format`: `lineStart >= 2` zu `> 2` | Eine Datei, deren erste Zeile nur ihr CRLF ist, bekäme LF | `TestPruneHooksSeesTheCRLFOfTheVeryFirstLine` | d85956ba |
| `check`: `{64}` zu `{63,64}` | Eine Summe mit 63 Stellen ging durch | `TestRenderWantsAPlainSHA256` (Fall `testSum[1:]`) | d85956ba |
| `check`: `IndexFunc(…) >= 0` zu `> 0` | Leerraum nur am Anfang ging durch | `TestRenderRefusesAnOldFileOutsideTheProject` (Fall `" a"`) | d85956ba |
| `check`: `unicode.IsSpace` zu Leerzeichen und Tab | Ein geschütztes Leerzeichen ging durch | dieselbe (Fall `a\u00a0b`) | d85956ba |
| `vaultSource`: Abfrage `VaultOld == ""` gestrichen | Ein Tresor ohne `vault_old` nannte eine Quelle, die der Tresor selbst ist | `TestRenderNamesTheSourceThatIsTheVaultFolder` | d85956ba |
| `vaultSource`: Vergleich ohne Groß-/Kleinschreibung | Eine Quelle in anderer Schreibung als der Ordner galt als dieselbe; das Skript vergleicht als Text | dieselbe | d85956ba |
| `benchLoadReport`, `benchReadExtras`, `switchoverReadParams`, `devSwitchoverPruneHooks`: Lesefehler übergangen (4 Mutanten) | Die Datei fiel trotzdem, aber mit einer späteren Klage über leere Daten; die Tests fragten nur nach Exit-Code und Pfad | `TestDevBenchCompareReportsUnreadableReports`, `TestDevBenchCasesReportsUnreadableInputs`, `TestDevSwitchoverRenderRefusesParametersItCannotUse`, `TestDevSwitchoverPruneHooksReportsWhatFails` (verlangen den Text von `os.ReadFile`) | cc5a02ce |
| `benchLoadReport`: JSON-Fehler übergangen | Ein kaputter Bericht fiel nur am Schema | `TestDevBenchCompareReportsUnreadableReports` | cc5a02ce |
| `benchLoadReport`: `!=` zu `>` beim Schema | Ein älteres Schema ging durch | `TestDevBenchCompareRefusesAReportOfAnotherSchema` (Schema 0) | cc5a02ce |
| `switchoverReadParams`: Pfad fehlt in der Meldung | Eine kaputte Parameterdatei hieß nicht beim Namen | `TestDevSwitchoverRenderRefusesParametersItCannotUse` | cc5a02ce |
| `benchCaseFiles`: `os.LookupEnv` durch eine leere Umgebung ersetzt | Kein CLI-Test las eine Variable | `TestDevBenchCasesReadsTheEnvironmentForAHook` | cc5a02ce |
| `apply.sh.tmpl` Schritt 1: Ausnahme „config.toml steht“ gestrichen | Ohne `config_new`-Datei neben einer stehenden Konfiguration bräche das Skript ab | `TestApplyNeedsNoNewConfigurationWhereOneStands` | 2452afdc |
| `apply.sh.tmpl` Schritt 6: `removed=1` zu `removed=0` | Ein erster Lauf meldete zusätzlich „files: already removed“ | `TestApplyMovesTheWikiIntoTheProject` | 2452afdc |

**Bleibt stehen.**

| Mutant | Warum |
|---|---|
| `benchcompare.Factor`: `before <= 0` zu `before < 0` | Bei `before == 0` und `after > 0` ergibt `before / after` ebenfalls 0, die Rückgabe für „keine Zeit“ |
| `readObject`: Fehler des Schlüssel-`Token` übergangen | Scheitert `Token` an einem Schlüssel, steht der Decoder in `tokenObjectKey`, und `Decode` lehnt den Wert dort selbst ab (`tokenValueAllowed`); mit 17 kaputten Eingaben gegengeprobt |
| `readObject`: Fehler von `Decode` übergangen | `readValue` setzt den Fehler fest (`dec.err`) oder die Eingabe ist zu Ende, und das schließende `Token` meldet ihn; das Ergebnis wird bei einem Fehler verworfen; mit derselben Probe gegengeprüft |
| `event.prune`: Fehler von `json.Unmarshal` in `[]RawMessage` übergangen | Für jeden Wert, der keine Liste ist, bleibt `groups` leer, `left` und `groups` sind gleich lang, die Funktion gibt nichts zurück wie zuvor |
| `event.prune`: Gruppen mit `", "` statt `","` verbunden | `format` schickt jeden Block durch `json.Indent` oder `json.Compact`, die Leerraum zwischen Werten verwerfen |
| `written`: `.claude` aus `keep` gestrichen | `.claude` fällt weiter über `HasPrefix(".claude/settings.json", ".claude/")` |
| `written`: `p.WikiDst != ""` gestrichen, `&&` zu `||` | Bei leerem `WikiDst` ist `wiki` gleich `.`, das nie mit `<projekt>/` beginnt; beide Bedingungen sind dann falsch |
| `written`: `HasPrefix(wiki, project+"/")` gestrichen oder ohne `/` | Ein Wiki außerhalb des Projekts bleibt nach `TrimPrefix` ein absoluter Pfad, und eine alte Datei ist immer relativ (`inside`), sie gleicht ihm nie und beginnt nie mit ihm |
| `matchFlags.String` gibt `""` | `flag` ruft `String` nur für die Hilfe, um einen Vorgabewert zu erkennen; beide Werte zeigen keinen |
| `devSwitchoverPruneHooks`: `err == nil` vor dem Schreiben gestrichen | Bei einem Fehler ist `removed` immer `nil` (Lesefehler: nie gesetzt; `PruneHooks`: gibt bei einem Fehler `nil` zurück) |
| `apply.sh.tmpl` Schritt 3: Abbruch bei `missed` gestrichen | Nicht erreichbar ohne ein `cp`, das falsch kopiert und Erfolg meldet: abweichende Dateien hält schon die Staging-Prüfung auf, und `PATH` beginnt mit `/usr/bin`, ein Test kann `cp` nicht ersetzen. Eine Wache gegen das Werkzeug, kein Verhalten des Skripts |
| `apply.sh.tmpl` Schritt 7: `MOVED = 1` zu `true` | Hält git noch Dateien von `VAULT_OLD`, ist die Quelle gefüllt (`filled`), also `WIKI=merge` und `MOVED=1`; ein `VAULT_OLD` ohne Quelle in `WIKI_SRCS` lehnt `Render` ab |

**Code-Befunde.** Kein Fehler im Code. Die beiden Punkte, die das Ledger für
Task 4 offen hielt, sind behoben: `writeAndClose` ist die Naht für Schreib-
und Schließfehler (die beiden Mutanten dazu fielen), und `dev bench cases`
macht `--out` absolut (`benchAbs`). Beim Fahren der Runde schrieb ein
CLI-Mutant `cases.json` und drei Payloads ins Paketverzeichnis
`internal/cli`; sie gingen versehentlich in einen Commit und sind vor dem
Weitermachen wieder herausgenommen.

Alle Funktionen der Pakete stehen weiter bei 100 % Coverage.

### Die Runde des Aufräum-PR (2026-10-04)

**Methode.** Handrunde per `go test -overlay` (Windows-Pfade `C:/…`), je
Mutant eine Kopie der Datei mit genau einer Änderung, gegen die gezielten
`-run`-Tests und, wo das Urteil an einer Fallsuite hängt, gegen diese. Je Task
lief zuerst ein unveränderter Kontrollmutant, der überleben musste. Ein
Mutant, der nicht übersetzt, hätte als BADMUTANT gezählt und wurde so
geschrieben, dass er baut; BADMUTANT am Ende: 0. Task 4 (nur Entfernen von
`Manifest.Lanes`) hat keine Runde, weil dort kein neuer Regelcode steht.

| Funktion | Mutanten | getötet | überlebt | BADMUTANT |
|---|---:|---:|---:|---:|
| `ReadAreaDeclaration`, `IsUndeclared`, Hinweistext (`internal/config`) | 8 | 8 | 0 | 0 |
| Aufrufer von `IsUndeclared` in `apply`, `check/house`, `check/run`, `convert`, `maintenance`, `wiki`, `cli/lintsweep`, `privacy` | 8 | 8 (6 erst nach einem neuen Test) | 0 | 0 |
| `areaCheck`, Wahl des Manifests (`chosenManifest`) | 4 | 4 | 0 | 0 |
| `recoverStock`, `registerOf`, `ArtifactLookup`, `ManifestDir` (Entfernen des Rückfalls) | 8 | 8 | 0 | 0 |
| `ImportMCP` und `[[result]]` (`internal/dev/importcases`) | 3 | 2 | 1 | 0 |
| Ratschlagszeilen in `graph/read.go`, `status/status.go`, `search/stamp.go` | 18 | 18 | 0 | 0 |
| **zusammen** | **49** | **48** | **1** | **0** |

Die erste Runde der Aufrufer von `IsUndeclared` ließ sechs Mutanten leben (die
Stellen in `check/house/federation.go`, `check/run/run.go`, `convert/run.go`,
`maintenance/reconcile.go`, `wiki/census.go`, `cli/lintsweep.go`): ihre Tests
trugen keine Policy-Datei ohne `[area]`. Je ein neuer Test mit einer solchen
Datei tötet sie (`undeclared_test.go` in `house`, `run`, `convert`,
`maintenance`, `wiki`, `internal/cli/lintsweep_undeclared_test.go`); in der
Tabelle zählen sie als getötet.

**Überlebender Mutant.**

| Funktion | Mutant | Entscheidung |
|---|---|---|
| `ImportMCP`, `internal/dev/importcases` | `"result": true` aus der Skip-Liste von `copyTree` gestrichen (m2) | **Äquivalent, stehengelassen.** Ohne Skip kopiert `copyTree` die Datei `result` zuerst, der eigene Schreiber überschreibt sie danach mit demselben Ziel. Dass genau ein Schreiber je Datei schreibt, ist eine Konvention, kein beobachtbares Verhalten. |

Alle Funktionen der berührten Pakete stehen weiter bei 100 % Coverage (je
Funktion; `copyTree` und `foldHookState` tragen ihre bestehende
`//coverage:exempt`).

### Nachprüfung der drei Runden (2026-10-04)

Jede stehengelassene Begründung der drei Runden oben ist am Code von
`origin/master` `3934fb5b` (v7.0.1) nachgerechnet, mit dem Gegenfall, den sie
nicht nennt. Die Tabellen oben bleiben, wie sie geschrieben wurden; was sich
geändert hat, steht hier.

**Halten am Code:** `flattenKeys` (`continue`); `benchcompare.Factor` (auch
bei `before == 0, after == 0` greift `after <= 0`, und `-0` zeigt
`formatFactor` wie `0`); `event.prune` (Unmarshal-Fehler; `", "`, weil beide
Zweige von `format` durch `json.Compact` oder `json.Indent` gehen);
`written` (`.claude`; `p.WikiDst != ""` und `&&` zu `||`: `check` verlangt
für `wiki_dst` einen absoluten Pfad, `inside` lässt in einer alten Datei weder
`:` noch ein führendes `/` zu, also gleicht `f` nie einem Wiki außerhalb des
Projekts); `matchFlags.String` (Vorgabe und Nullwert sind beide `[]` im
Original und beide `""` im Mutanten); `devSwitchoverPruneHooks` (jeder
Fehlerweg von `PruneHooks` und `os.ReadFile` lässt `removed` leer);
`apply.sh.tmpl` Schritt 3 (die Staging-Prüfung in Zeile 143–148 hält
abweichende Dateien vor der Registry auf) und Schritt 7 (`VAULT_OLD` mit
Dateien in git heißt `filled "$VAULT_SRC"`, also `WIKI=merge` und
`MOVED=1`).

**`readObject`, neuer Beleg.** Die 17 Eingaben der ersten Probe liegen
nirgends. Neu gefahren: beide Mutanten neben dem Original als Kopie in einem
Fuzz-Test, 29 Saateingaben (kaputte Schlüssel `1`, `null`, `true`, `[]`,
`{}`, fehlende Werte, abgeschnittene Objekte, Rest nach dem Objekt,
doppelte Schlüssel) und 60 s Fuzzing (3,7 Mio. Eingaben): kein Unterschied
in Fehler und Besuchen, keine Panik. Ein Kontrollmutant ohne die Prüfung auf
Rest nach dem Objekt fiel an zwei Saaten auf; die Probe misst also.

**`ImportMCP` (m2), Begründung zu scharf.** Beobachtbar ist der Unterschied
doch, aber nur nach einem Abbruch: Scheitert `readRecorded`, liegt beim
Mutanten eine rohe Kopie von `result` im Ziel, beim Original keine. Ein
vollständiger Import endet gleich. Der Mutant bleibt stehen, weil ein
abgebrochener Import das Ziel ohnehin halb geschrieben hinterlässt.

**`chosenManifest`, zwei Lücken.** Der Aufräum-PR hat an `areacheck.go` nur
`manifestNames`, `chosenManifest`, dessen Aufruf und einen Text in
`legacyHints` geändert; für `checkOneManifest`, `fileDeclaresArea`,
`flattenKeys`, `classifyKeys` und `schemaKnows` gilt die Runde vom
2026-09-29 weiter. `chosenManifest` ist neu und hatte oben vier Mutanten.
Neu gefahren: neun und ein Kontrollmutant (überlebt), gegen
`-run 'TestAreaCheck|TestClassifyKeys|TestFlattenKeys|TestLegacyHints'`;
einer musste umgeschrieben werden, damit er baut. Sieben getötet, zwei
überlebten, beide echte Lücken:

| Mutant | Lücke | Test, der ihn tötet |
|---|---|---|
| `!info.Mode().IsRegular()` gestrichen | Ein Ordner `.loomux/config.toml` neben einem `.brain.toml` mit `[area]` hieße `chosen: none` statt `chosen: .brain.toml` | `TestAreaCheckChoosesPastADirectoryUnderTheLoomuxName` |
| `errors.Is(err, config.ErrNoArea)` zu `err != nil` | Ein kaputtes `[area]` in `.loomux/config.toml` ließe die Wahl zum `.brain.toml` dahinter weitergehen und meldete die kaputte Datei als überdeckt | `TestAreaCheckChoosesNoneBehindALoomuxConfigWhoseAreaIsBroken` |

Beide Tests sind am Code grün und laufen je gegen ihren Mutanten rot (per
Overlay belegt); gegen den Kontrollmutanten bleiben beide grün.

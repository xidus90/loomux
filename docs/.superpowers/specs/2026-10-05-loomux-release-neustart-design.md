# loomux Release-Neustart und Beta-Kanal — Design

**Stand:** Entwurf 2026-10-05, gegen `origin/master` 80db6bb0.
**Bezug:** Fusions-Spec, Nachtrag #31; Schwester-Spec
`2026-10-05-loomux-stufe-4f-design.md` (der Neustart schließt 4f ab).

## Ziel

Mit dem Abschluss von 4f endet die Beta. Die Zählung beginnt neu, v1.0.0 ist das
erste stabile Release, und die alten Tags und Releases v1.0.0 bis v7.x
verschwinden. Künftig gibt es Betas als eigene Versionen, von jedem Zweig
geschnitten. Installiert werden sie mit `loomux upgrade --beta` oder
`loomux upgrade --version x.y.z`. Eine Beta-Installation bekommt die nächste
Beta automatisch.

## Entscheidungen des Nutzers (2026-10-05)

1. Echter Neustart: alte Tags und Releases löschen, v1.0.0 stabil.
2. Im Changelog bekommen die alten Einträge eine Beta-Kennung.
3. Betas sind Releases von einem Zweig, installierbar per `--beta` und per
   `--version x.y.z`.
4. Eine Beta-Installation bekommt automatisch die nächste Beta.

## Gemessene Randbedingungen

- **Tags:** Es gibt 60 Tags, darunter v1.0.0 bis v1.3.0 vom 2026-09-18/19.
  Alle Releases sind Pre-Releases (`release.sh`: `channel=${CHANNEL:-beta}`).
- **Kanal im Binary:** Heute steht er in `cli.Channel` (ldflags).
  `loomux version` druckt `loomux 7.1.0 (beta)`.
- **Self-Update:**
  - `selfupdate.parseVersion` nimmt keinen Suffix an.
  - `pick` nimmt im Kanal `beta` Pre-Releases und stabile Releases.
  - `Newer` vergleicht nur Nummern.
  - Ein installiertes 7.x hält v1.0.0 deshalb für älter und holt es nie. Die
    neue Regel wirkt nur in Binaries, die sie schon enthalten.
- **`NextVersion`:** übergeht jeden Tag, der nicht `vX.Y.Z` ist
  (`tagPattern`). Ohne solchen Tag liefert es 1.0.0.
- **`InsertChangelog`:** lehnt eine Version ab, deren `\n## [x.y.z]` schon in
  der Datei steht. Ohne Umbenennung der alten Einträge scheitern v1.0.0 bis
  v1.3.0 und später jede Version bis 7.1.0 an `ErrDuplicate`.
- **`internal/setup/plan.go`:** Die Antigravity-Prüfung ruft
  `selfupdate.AtLeast(installiert, init)` auf.

## Versionsschema

- **Stabil:** `X.Y.Z`, nur aus einem gemergten PR auf `master`, wie heute
  (Label → Stufe, `chore(release)`-Commit, Changelog-Eintrag).
- **Beta:** `X.Y.Z-beta.N`.
  - Geschnitten per `workflow_dispatch` mit den Eingaben `ref` (Zweig) und
    `bump` (major/minor/patch).
  - `X.Y.Z` ist `NextVersion(stabile Tags, bump)`.
  - `N` ist 1 + das höchste N unter den Tags derselben Basis.
  - Eine Beta erzeugt nur Tag und Pre-Release am Kopf von `ref`: kein Commit
    auf dem Zweig, kein Changelog-Eintrag. Der Eintrag kommt mit dem stabilen
    Release.
  - Der Rumpf des Releases nennt Zweig und Commit.
- **Alte Zählung:** jede Version ohne Suffix, deren Binary `(beta)` hinter
  der Nummer druckt. Neue Builds drucken nie einen Kanal. Ein Beta-Build
  trägt `-beta.N` in der Nummer und druckt `loomux 1.1.0-beta.1`.

## Vergleich und Kanal (`internal/selfupdate`)

- **Lesen der Version:** `parseVersion` liest `X.Y.Z` und `X.Y.Z-beta.N`. Die
  Rangfolge folgt SemVer: `1.1.0-beta.2` < `1.1.0-beta.10` < `1.1.0`.
- **Alte Zählung:** Eine Version der alten Zählung rangiert unter jeder
  Version der neuen. Erkannt wird sie am Kanal `beta` des laufenden Binarys
  (`cli.Channel`) bzw. an `(beta)` in der `--version`-Ausgabe eines
  installierten. `Newer` und `AtLeast` nutzen denselben Vergleich. Damit gilt
  ein installiertes 7.2.0 vor einem `init` 1.0.0 nicht als neu genug.
- **Kanal des laufenden Binarys:**
  - Trägt seine Version `-beta.N` oder gehört sie zur alten Zählung, nimmt
    das automatische Update stabile Releases und Betas.
  - Sonst nimmt es nur stabile.
  - Ein Beta-Binary, das von einem neueren stabilen Release überholt wird,
    installiert dieses und ist danach stabil. Nach einem stabilen Release
    gibt es bis zur nächsten Beta keine neuere Beta, und wer weiter Betas
    will, ruft `--beta`.
- **Einordnung eines Releases:** Ein Tag allein sagt nicht, welcher Zählung er
  angehört (`v8.0.0` alt gegen `v1.0.0` neu). Darum vergleichen `Newer` und
  `pick` ganze `Release`s (Tag und `isPrerelease`):
  - Suffix `-beta.N`: neue Beta.
  - Ohne Suffix, `isPrerelease=true`: alte Zählung, rangiert unter jeder neuen
    Version.
  - Ohne Suffix, `isPrerelease=false`: neues stabiles Release.

  Folge: Die Brücke nimmt v1.0.0 auch dann, wenn ein altes Pre-Release stehen
  blieb, und ein neues Binary nimmt ein altes Pre-Release nie. Das Löschen in
  Schritt 3a ist damit Hygiene, keine Vorbedingung von 3c.
- **`cli.Channel`:** Es bleibt nur, damit ein Binary der alten Zählung sich
  als solches erkennt. Neue Builds setzen es nicht, `dev release build`
  verliert `--channel`.

## `loomux upgrade`

| Aufruf | Wählt |
|---|---|
| `loomux upgrade` | das neueste Release im Kanal des laufenden Binarys (wie `serve`) |
| `loomux upgrade --beta` | das neueste Release überhaupt, Betas eingeschlossen; danach ist das Binary im Beta-Kanal |
| `loomux upgrade --version x.y.z` | genau diesen Tag, auch eine Beta, auch ein Downgrade; fehlt er, Exit 1 mit „no release x.y.z“ |

- `--beta` und `--version` schließen sich aus (Exit 2).
- **Downgrade per `--version`:** hält nicht. `serve` hebt binnen 24 Stunden
  wieder auf das neueste Release seines Kanals. Das steht in der
  CLI-Referenz.
- **Auswahl der Releases:** `gh release list --limit 30` bleibt für die
  Auswahl des neuesten. `--version` holt den Tag direkt (`gh release view
  <tag>`).
- **Unverändert:** Prüfsumme, `--version`-Probe, Tausch und `update.json`.

## Release-Skripte

- **Pre-Release-Markierung:** `release.sh` setzt `--prerelease` genau dann,
  wenn die Version ein `-` trägt. `CHANNEL` und die Repo-Variable
  `RELEASE_CHANNEL` entfallen.
- **Quelle der Tags:** Heute liest `release.sh` die Tags mit `git tag -l 'v*'`
  aus dem Checkout des self-hosted Runners, nach `git fetch --force --tags`.
  Dieser Abruf löscht keinen Tag, der auf dem Remote gelöscht wurde. Ein
  wiederverwendeter Checkout sähe nach Schritt 3a noch die alten Tags, und 3c
  veröffentlichte `8.0.0` als stabil. Darum liest `release.sh` die Tags künftig
  mit `git ls-remote --tags origin 'v*'`, für stabile Releases und für Betas.
- **Workflow:** `release.yml` bekommt neben dem PR-Weg den Beta-Weg. Ein
  Eingang `mode` (`pr` | `beta`) trennt die beiden, weil `workflow_dispatch`
  heute „PR N nachveröffentlichen“ heißt. Der Beta-Weg nimmt `ref` und
  `bump`. Den Beta-Tag schreibt die
  `loomux-release`-App wie heute die stabilen. Die Ausnahme in `AGENTS.md`
  nennt künftig `v*`-Tags beider Arten.
- **Neuer Befehl:** `loomux dev release next-beta --bump <stufe>` liest die
  Tags und gibt `X.Y.Z-beta.N` aus. Getestet wie `NextVersion`.

## Ablauf: drei Schritte in fester Reihenfolge

1. **Brücke** (alte Zählung, noch Beta, ein PR `release:minor`).
   - Die Brücke ist die nächste Version der alten Zählung, heute 7.2.0.
   - Inhalt: alles aus „Vergleich und Kanal“, `upgrade --beta`/`--version`
     und `next-beta`.
   - Nicht dabei sind die Änderungen an `release.sh` und `release.yml`. Die
     Brücke selbst muss noch als Pre-Release mit Kanal `beta` erscheinen, sonst
     gälte 7.2.0 nach der neuen Regel als stabil. Diese Änderungen kommen mit
     Schritt 3b.
   - Jede Installation zieht die Brücke automatisch.
   - Wartet auf nichts aus 4f.
2. **Warten,** bis jeder bekannte Wirt 7.2.0 oder neuer meldet: eigener
   Rechner, `ecoflow`, `space`, `iam_backend`, `iam_frontend`, `iam_workers`,
   Vault. Belegt wird das je Maschine aus `<Zustandsverzeichnis>/update.json`
   oder per `loomux version`, festgehalten in `parity/stufe-4f.md`.
3. **Neustart.** Er läuft erst, wenn 4f PR C gemergt und Schritt 2 belegt ist.
   Die Schritte führt der Mensch aus; der Agent nennt die Befehle.
   - **a.** Alle GitHub-Releases `v*` und alle Tags `v*` werden gelöscht,
     remote und lokal. Geprüft wird das mit `gh release list` (leer) und
     `git ls-remote --tags origin 'v*'` (leer). Zwischen 3a und 3c findet
     `serve` kein Release und schreibt `result: failed` („no release in
     channel“), und der Sitzungsstart warnt. Das ist erwartet, und das Fenster
     soll kurz bleiben.
   - **b.** Ein PR `release:major` benennt die alten Changelog-Überschriften
     auf `## [X.Y.Z-beta] - <datum>` um. Die Form unterscheidet sich von
     `-beta.N` und von `X.Y.Z`, sodass `InsertChangelog` keinen Duplikatfehler
     wirft. Ein Satz im Kopf sagt, dass die Einträge mit `-beta` einer
     abgeschlossenen Beta-Zählung angehören und ihre Tags gelöscht sind. Die
     README-Abschnitte zu `RELEASE_CHANNEL` und die Beispielausgabe in
     `docs/*/getting-started.md` ziehen mit. Derselbe PR bringt die
     Release-Skripte (Abschnitt oben) und entfernt `--channel`. Die
     Repo-Variable `RELEASE_CHANNEL` löscht der Mensch vor dem Merge.
   - **c.** Der Merge dieses PR veröffentlicht durch `NextVersion` ohne `v*`-Tag
     v1.0.0 als stabiles Release. Die Brücken-Installationen holen es selbst.
   - **d.** Jeder lokale Klon und jedes Worktree fährt
     `git fetch --prune --prune-tags`, sonst rechnet ein lokaler
     `dev release` mit alten Tags.

## Tests

- **`parseVersion` und Rangfolge:** Tabellen gegen SemVer, mit `beta.2`
  gegen `beta.10`, Beta gegen stabil derselben Basis und einem Fall der alten
  Zählung gegen jede neue Version. `AtLeast` und `Newer` mit denselben Fällen.
- **`pick`:** Kanal stabil, Beta und alte Zählung, jeweils mit einer Liste, in
  der das erwartete Release nicht an erster Stelle steht.
- **`upgrade`:** `--beta`, `--version` vorhanden/fehlend/Beta, beide
  zusammen (Exit 2). Gefahren über den vorhandenen Runner-Fake von
  `internal/selfupdate`.
- **`next-beta`:** ohne Tags, mit stabilen Tags, mit vorhandenen Betas
  derselben und einer anderen Basis.
- **`release.sh`:** Die Pre-Release-Entscheidung als Funktion von `version`,
  über den Smoke-Test von `ci/`.

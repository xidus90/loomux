# Releases und Pull Requests

[Übersicht](README.md)

Diese Seite behandelt, wie ein gemergter Pull Request zum Release wird, wie eine Beta von Hand entsteht, wie man einen Pull Request öffnet und wie Maintainer die Release-Mechanik einrichten.

## Wie ein Pull Request zum Release wird

Jeder gemergte Pull Request nach `master` wird nach seinem Label veröffentlicht:

| Label | Bedeutung | Version |
|---|---|---|
| `release:major` | Inkompatible Änderung an Befehl, Flag, Hook-Protokoll, Konfigurationsformat oder Exit-Code | `X+1.0.0` |
| `release:minor` | Neue Funktion, kompatibel | `X.Y+1.0` |
| `release:patch` | Fehlerbehebung oder Abhängigkeits-Update, kompatibel | `X.Y.Z+1` |
| `release:none` | Nur Doku, CI oder Tests | kein Release |

Ein Release ohne Suffix ist stabil; eines mit Suffix (`1.1.0-beta.1`) ist ein
Pre-Release. Was sich geändert hat, steht in [`CHANGELOG.md`](../../CHANGELOG.md);
ein Eintrag entsteht nur mit einem stabilen Release, und die mit `-beta`
markierten Einträge sind die Zählung vor 1.0.0.

Die Kompatibilitätsmatrix hält ein Major-Release auf: Ein Pull Request mit dem Label `release:major` fährt jeden Eintrag der [Kompatibilitätsliste](support.md#geprüfte-kombinationen) auf Linux und Windows (`loomux dev compat matrix --full`); einer, der die Liste oder ein Fixture berührt, fährt die Einträge, die er ändert. Ebenso eine Änderung am Code, der die Kombinationen prüft: Die `[[watch]]`-Tabellen der Liste nennen solche Dateien (die Preset- und Framework-Tabellen, die Compat-Skripte und -Code, die Suites von gdUnit4 und GUT) und die Einträge, die eine Änderung daran fährt, alle oder die bestimmter Stacks oder Addons; eine Testdatei wählt nichts. Eine neue Laufzeit- oder Framework-Version zu unterstützen heißt: ein Eintrag mehr in `internal/compat/compat.toml` und die passende Zeile auf der Support-Seite; ein wöchentlicher Job (`compat-scout`) eröffnet für jede Upstream-Version, die der Liste fehlt, ein Issue mit dem Label `compat-scout`. Siehe [`loomux dev compat`](cli-reference.md#loomux-dev-compat-matrixrunscout-flags).

## Beta-Releases

Eine Beta entsteht von Hand aus jedem Branch, Tag oder Commit, ohne Pull
Request: den Workflow `release` mit `mode=beta`, `ref` (was gebaut wird) und
`bump` (`major`, `minor` oder `patch`, Standard `minor`) starten, etwa
`gh workflow run release.yml -f mode=beta -f ref=mein-branch -f bump=minor`.
Er veröffentlicht `vX.Y.Z-beta.N` als Pre-Release für die Version, die das
nächste stabile Release bekäme, `N` eins über der höchsten Beta davon. Er
committet nichts und lässt das Changelog unberührt; der Tag ist seine einzige
Spur. Installieren mit `loomux upgrade --beta` (das neueste Release samt Betas)
oder `loomux upgrade --version <x.y.z-beta.n>`.

**Eine Beta nur aus einem Branch, Tag oder Commit starten, dessen Code
geprüft ist.** Der Workflow baut diesen Stand und führt das gebaute Binary aus,
während der Schritt den Token der App `loomux-release` hält, die beide
Rulesets durchlassen. Code in einem Stand, den niemand geprüft hat, könnte mit
diesem Token auf `master` pushen oder `v*`-Tags anlegen.

Wie eine Maschine dem Beta-Kanal folgt, steht unter
[Updates und Kanäle](install.md#updates-und-kanäle).

## Einen Pull Request öffnen

Auf `master` wird nie direkt committet; jede Änderung läuft über einen Pull
Request. Die Hooks in `.githooks` lehnen einen Commit auf `master` und einen
Push dorthin ab, sobald `go run ./cmd/loomux init --yes` einen frischen Klon
eingerichtet hat (von Hand: `git config core.hooksPath .githooks`). Mit
einem LLM erledigt der Skill `release-pr` die Schritte unten; von Hand:

1. Commits thematisch gruppieren, ein Commit pro Änderung. Eine spätere
   Korrektur an etwas, das dieser Branch eingeführt hat, gehört in den
   Commit, der es eingeführt hat; nur die Behebung eines Fehlers, der schon
   auf `master` war, behält einen eigenen Commit. Einen bereits gepushten
   Branch neu zu schreiben braucht `git push --force-with-lease`. Jede
   Nachricht ist ein Conventional Commit (`feat: …`, `fix: …`, `docs: …`;
   `!` für eine inkompatible Änderung). Der Hook `commit-msg` prüft das
   lokal, und der Check `pr-label` lehnt einen Pull Request ab, sobald ein
   Commit-Titel die Form verletzt.
2. Ein Label aus der Tabelle in
   [Wie ein Pull Request zum Release wird](#wie-ein-pull-request-zum-release-wird)
   wählen; zwischen zwei Stufen die höhere.
   Es liegt nie unter den Commits: `!` verlangt major, `feat` minor, `fix`
   patch.
3. Den Rumpf schreiben:
   ```
   Release: <level> — <one-sentence reason>

   ## Summary
   - <what changes for a user>

   ## Changelog
   ### Added
   - <entry>
   ```
   Changelog-Überschriften sind nur `Added`, `Changed`, `Deprecated`,
   `Removed`, `Fixed`, `Security`; bei `release:none` entfällt der Abschnitt.
4. Prüfen: `go run ./cmd/loomux dev release parse-body --labels release:<level> --body <file>`.
5. Branch pushen, dann `gh pr create --base master --label release:<level> --body-file <file>`.

## Releases einrichten (Maintainer)

1. Labels:
   ```sh
   gh label create release:major --color B60205 --description "Breaking change"
   gh label create release:minor --color 0E8A16 --description "New feature, compatible"
   gh label create release:patch --color 1D76DB --description "Bug fix or dependency update"
   gh label create release:none --color CCCCCC --description "No release"
   ```
2. GitHub App (Settings → Developer settings → GitHub Apps → New): Name
   `loomux-release`, Webhook aus, Repository-Rechte `Contents: Read and
   write`, `Pull requests: Read-only`, `Metadata: Read-only`,
   `Administration: Read-only` (listet die Runner, siehe Schritt 4), „Only on
   this account“. Private Key erzeugen, App nur in `xidus90/loomux`
   installieren, dann:
   ```sh
   gh secret set RELEASE_APP_CLIENT_ID --body <client-id>
   gh secret set RELEASE_APP_PRIVATE_KEY < loomux-release.private-key.pem
   ```
3. Rulesets: `.github/rulesets/master.json` für `master` und
   `.github/rulesets/tags.json` für Tags `v*`. Beide lassen die App
   `loomux-release` (ihre App ID, nicht die Client-ID, in `actor_id`) und
   Repository-Admins durch: die App committet und taggt jedes Release, und
   ein Admin kann ein kaputtes Release von Hand reparieren. Alle anderen
   kommen nur über einen Pull Request nach `master`, mit grünen Statuschecks
   `gate-windows` und `build-linux` (Workflow `ci`) sowie `check` (Workflow
   `pr-label`) und der Freigabe seines letzten Pushs durch den Code-Owner
   (`.github/CODEOWNERS`). Die Checks fahren die Workflows und das Tor des
   Pull Requests selbst, er kann sie also grün machen; was hält, ist die
   Freigabe, und ein Push danach braucht eine neue. Das gilt nur, solange
   außer dem Code-Owner und den Repository-Admins niemand Schreibzugriff hat
   (heute nennt `.github/CODEOWNERS` nur `@xidus90`): Der Key aus Schritt 2 ist
   ein einfaches Repository-Secret, und `ci` und `pr-label` reichen es an die
   Workflows jedes Pull Requests aus einem Branch dieses Repositorys weiter. Wer
   Schreibzugriff hat, kann es dort vor jeder Freigabe lesen und einen Token
   erzeugen, den beide Rulesets durchlassen. Beide anlegen, dann
   prüfen, dass `master` Regeln listet:
   ```sh
   gh api -X POST repos/xidus90/loomux/rulesets --input .github/rulesets/master.json
   gh api -X POST repos/xidus90/loomux/rulesets --input .github/rulesets/tags.json
   gh api repos/xidus90/loomux/rules/branches/master
   ```
   Eine geänderte Datei geht an `gh api -X PUT repos/xidus90/loomux/rulesets/<id>`.
4. Runner: Jeder Workflow fährt zuerst `pick-runner` auf einem
   GitHub-Runner, der jeden Job an den self-hosted Runner seines
   Betriebssystems schickt, solange einer online ist, sonst an GitHubs. Er
   listet die Runner mit dem Token der App, ohne `Administration: Read-only`
   läuft also alles bei GitHub; ein Pull Request aus einem Fork bekommt keine
   Secrets und läuft ebenfalls dort. Seine eigenen Workflow-Dateien können
   die self-hosted Runner trotzdem nennen, also wartet jeder Lauf von außen
   auf Freigabe:
   ```sh
   gh api -X PUT repos/xidus90/loomux/actions/permissions/fork-pr-contributor-approval -f approval_policy=all_external_contributors
   ```

Ist ein Release ausgefallen, den Workflow `release` von Hand starten
(`gh workflow run release.yml -f pr=<nummer>`). Er verweigert einen Pull
Request, der nicht nach `master` gemergt ist, und ein erneuter Lauf nach dem
`chore(release): v*`-Commit verwendet diesen Commit wieder. Gibt es für den Pull
Request schon einen Tag oder ein Release, nicht erneut starten, sondern das
vorhandene Release von Hand reparieren.

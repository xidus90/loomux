# Schalter: das LLM darf die Projekt-Config bearbeiten

Stand 2026-10-16, Basis `origin/master` 18566e83, Zweig `feat/llm-config-switch`.

## Anlass

Entwickler pflegen die loomux-Config eines Projekts heute nur über
`loomux init` oder von Hand. Ein Agent kann eine Änderung nur vorschlagen
(`loomux config set|unset … --propose`), und ein Mensch übernimmt sie mit
`loomux config apply`. Gewünscht ist, dass ein Entwickler pro Projekt
bewusst einschalten kann, dass sein LLM die Config dieses Projekts
bearbeitet, etwa beim Einrichten oder zum Debuggen, und das wieder
ausschalten kann.

## Ziel und Erfolg

- Pro Area gibt es einen Schalter, den nur ein Mensch umlegt.
- Ist er an, ist die `.loomux/config.toml` dieses Projekts für Agenten
  bearbeitbar wie jede andere Datei im Projekt: mit Datei-Werkzeugen
  (`Write`, `Edit`, …) und mit `loomux config set|unset` ohne `--propose`.
  Alle Schlüssel, ohne Freigabe- oder Sperrliste.
- Ist er aus (Standard), verhält sich alles wie heute.
- Erfolg heißt: Mit dem Schalter an bearbeitet ein Agent die Config seines
  Projekts in einem Schritt. Mit dem Schalter aus verweigert der Wächter wie
  heute. Kein Weg, den der Wächter erkennt, lässt den Agenten den Schalter
  selbst umlegen oder eine Datei außerhalb der Projekt-Config öffnen.

## Nicht-Ziele

- Keine Nutzer, Konten oder Rollen. Gemeint ist der Entwickler am Rechner.
- Keine Freigabe- oder Sperrliste einzelner Schlüssel. Mit offenem Schalter
  darf das LLM auch Prüfungen lockern (`guard.mode`, `modules.*`,
  `rules.layers.*`, `privacy.*`, `[policy]` usw.). Das nimmt der Entwickler
  mit dem Einschalten bewusst in Kauf. (Entschieden mit dem Nutzer, gegen
  die Empfehlung einer Freigabeliste.)
- Nicht freigegeben werden: die globale Config (`loomux config --global`),
  `registry.toml`, `open.toml`, `armed.toml`, die Hook-Zustände unter
  `.loomux/state/`, die Config anderer Projekte oder Worktrees, und die
  Befehle `init`, `area add`, `merge-hook`, `gate`, `config llm`,
  `config apply`.
- Kein MCP-Tool für die Config.

## Anti-Ziele

Eine Lösung ist gescheitert, auch wenn sie technisch läuft, wenn

- der Agent den Schalter selbst setzen oder offen halten kann,
- ein offener Schalter in Projekt A die Config von Projekt B oder eine
  andere loomux-Datei schreibbar macht,
- ein kaputter oder unlesbarer Schalter Schreibrechte vergibt statt sie zu
  verweigern (Grundsatz 5: Wächter scheitern geschlossen).

## Entscheidungen

1. **Ort des Schalters: Feld am Area-Eintrag in `registry.toml`.** Name
   `llm_config`, Typ bool. Fehlt das Feld, gilt `false`; die Registry
   schreibt wie bisher nur wahre Flags (`RenderRegistry`,
   `internal/config/registrywrite.go`). Gelesen in
   `internal/config/registry.go` neben `readonly`, `workspace` usw.
   *Verworfen: `.env` im Projekt.* Git-Befehle (`checkout`, `stash`,
   `restore`) kann der Wächter nicht abfangen, eine `.env` reiste mit einem
   Branch mit, Worktrees hätten verschiedene Werte, und der Hook bräuchte
   einen neuen Parser. *Verworfen: Feld in `config.toml`*, weil der Agent es
   bei offenem Schalter selbst schreiben könnte.
   (Entschieden mit dem Nutzer: pro Projekt einzeln, Registry statt `.env`.)
2. **Umschalten: `loomux config llm on|off`**, ohne Argument zeigt
   `loomux config llm` den Stand. Wirkt auf die Area der Projektwurzel. Der
   Befehl schreibt die Registry über einen neuen Update-Weg in
   `registrywrite.go` (unter der vorhandenen Registry-Sperre, mit derselben
   Validierung vor dem Tausch). `on|off` gehört zu den Befehlen, die nur ein
   Mensch ausführt (`writesConfiguration` in `internal/hooks/guard.go`
   deckt `config` schon ab; ein Test hält es fest). Die Form ohne Argument
   liest nur und ist auch Agenten erlaubt. Ein Projekt ohne
   Registry-Eintrag kann den Schalter nicht tragen: `config llm on` bricht
   dort ab und nennt `loomux init`.
3. **Was der Schalter öffnet: genau eine Datei.** Die Manifest-Sperre gibt
   es heute an drei Stellen, und jede bekommt dieselbe Ausnahme:
   - die eingebaute Pfadregel `**/.loomux/config.toml`
     (`internal/hooks/guard.go`, `builtinPaths`),
   - die Erkennung von Shell-Zeilen, die das Manifest schreiben
     (`internal/hooks/guard.go`, `manifestReason`),
   - Schritt 1 in `Decide` (`internal/brain/guard/guard.go`, `isManifest`).

   Die Ausnahme greift nur für die `.loomux/config.toml` direkt unter dem
   Pfad einer Area mit `llm_config = true`. Die Config eines verknüpften
   Worktrees gehört zur Area des Hauptcheckouts und folgt deren Schalter.
   Jede andere `.loomux/config.toml` (Geschwister-Worktree eines anderen
   Projekts, fremdes Repo) bleibt gesperrt. Liegt die Datei unter keiner
   beschreibbaren Wurzel, öffnet der Schalter sie nicht: er hebt nur die
   Manifest-Sperre auf, nicht die Allow-List.
4. **`loomux config set|unset` ohne `--propose`.** Mit offenem Schalter
   lässt der Wächter diese Zeilen durch, wenn sie `--yes` tragen (sonst
   hinge der Agent in der Rückfrage) und weder `--global` noch `--root`
   nennen. Werte prüft wie heute `configTarget.validated`
   (`internal/cli/config.go`). Tabellen lehnt `config set` weiter ab; die
   bearbeitet der Agent dann mit Edit.
5. **Wurzeln bleiben im Projekt.** Bearbeitbare Schlüssel wie
   `layout.review` oder eine Wiki-Erklärung im Manifest öffnen über die
   vorhandenen Prüfungen nur Pfade innerhalb derselben Area bzw. desselben
   Repositorys (`reviewCentre`, Manifest-Erklärung in
   `schreibschranke.md`). Der Builder bestätigt das mit Tests, weil es mit
   offenem Schalter die einzige Grenze ist, die die Config noch hat.
6. **Setup:** Als letzte Frage des interaktiven `loomux init` fragt loomux,
   ob das LLM die Config dieses Projekts bearbeiten darf. Vorgabe ist Nein.
   Bei Ja setzt `init` das Feld, auch für eine schon registrierte Area (über
   den Update-Weg aus Entscheidung 2). Nicht-interaktiv wird nichts
   gesetzt. (Vom Nutzer so gewünscht: „Nach fertigem Setup soll gefragt
   werden, ob es auf true gestellt werden soll.“)
7. **Sichtbarkeit:** Ist der Schalter für das Projekt an, sagen das
   `loomux config list` und `loomux config llm`, und jede Ablehnung des
   Wächters in diesem Projekt endet mit einem Satz dazu.
8. **Fehler schließen:** Ein Feld, das kein bool ist, macht wie jedes
   kaputte Flag die ganze Registry unlesbar (`registry.go`), und die
   Schranke verweigert dann ohnehin. Die Ausnahmen aus 3 und 4 greifen nur
   bei gelesener Registry und `true`.
9. **Grundsatz ändern.** `docs/wiki/topics/architektur-grundsaetze.md` sagt
   heute: „`.loomux/config.toml` ist für Agenten nie beschreibbar … Agenten
   schlagen Änderungen vor, der Mensch schreibt sie.“ Neu: Standardmäßig
   gilt das weiter; ein Mensch kann es pro Projekt mit
   `loomux config llm on` aufheben und übernimmt damit, dass das LLM auch
   die Grenzen in dieser Config ändern kann. Diese Änderung des Grundsatzes
   gibt der Nutzer mit dem Design frei.

## Meine Entscheidungen (zur Prüfung)

- Name `llm_config` und Befehl `loomux config llm on|off`.
- Kein Ablaufdatum für den Schalter.
- `--yes` Pflicht, `--global` und `--root` verboten in der freigegebenen
  `config set`-Zeile.
- Projekt ohne Registry-Eintrag: `config llm on` bricht ab, statt die
  Area anzulegen.
- Der Schalter hebt nur die Manifest-Sperre auf; liegt das Projekt unter
  keiner beschreibbaren Wurzel, bleibt die Datei zu.
- `config apply` bleibt dem Menschen vorbehalten; mit offenem Schalter
  braucht der Agent es nicht.

## Doku und Wiki im selben Task

- `docs/de/configuration.md` und `docs/en/configuration.md`: Feld,
  Befehl, Setup-Frage.
- `docs/de/hooks.md` und `docs/en/hooks.md`: die Ausnahme von der
  Manifest-Sperre und vom Mensch-Befehl.
- Wiki: `topics/schreibschranke.md` (neuer Abschnitt),
  `topics/architektur-grundsaetze.md` (Entscheidung 9), Index.
- Welche weiteren Seitentypen nachgezogen werden, steht in `AGENTS.md`.

## Dem Builder überlassen

- Wie die drei Sperrstellen an die Area und ihren Schalter kommen
  (`checkTool` hat heute keinen Zugriff auf das Zustandsverzeichnis).
- Texte der Meldungen.
- Testaufteilung; 100 % Coverage nach Projektregel, Differenzbatterie des
  Wächters muss halten.

## Restrisiko

- Mit offenem Schalter kann das LLM jede Prüfung abschalten, die in der
  Projekt-Config steht. Das ist gewollt und der Grund, warum nur ein Mensch
  umschaltet.
- Shell-Zeilen, die der Wächter nicht als Schreibzugriff erkennt, können
  `registry.toml` erreichen. Das gilt heute schon für die Registry
  insgesamt (`schreibschranke.md`, „Was bewusst nicht geschieht“); das neue
  Feld macht es wertvoller, nicht größer.

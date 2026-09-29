# loomux Stufe 4e: Umstellung der Wirte — Design

**Stand:** Entwurf 2026-09-28, gegen `origin/master` a588ac9f; vom Nutzer
freigegeben am 2026-09-30.
**Bezug:** Stufe-4-Spec (`2026-09-23-loomux-stufe-4-design.md`, Abschnitt „4e“),
Fusions-Spec (Zeile „4e Umstellung“, Nachträge #19, #20, #24),
`parity/stufe-3a.md` („Auflagen an spätere Stufen“).

## Ziel

Die vier Wirte (`space`, `iam_backend`, `ecoflow`, `brain-knowledge`) laufen auf
loomux, und die Lese-Rückfälle auf die Altprojekte entfallen, ohne dass dabei
eine Deklaration, ein Stempel oder ein Schutz verloren geht. Es gibt keinen
Übersetzer (`migrate` fällt weg, #19); der Mensch legt von Hand um, und ein
lesender Befehl zeigt vorher, was die heutigen Leser am alten Manifest tun.

## Schnitt: drei Stücke in fester Reihenfolge

| Stück | Art | Wer | Wartet auf |
|---|---|---|---|
| **A** `loomux area check <pfad>` | Code, ein PR | Agent, Mensch merged | nichts |
| **B** Checkliste `parity/stufe-4e.md` | ohne Code | Mensch hakt ab | A; für die Wirte: Remote für `brain-knowledge`, die offenen Menschenschritte von 4a-2 (`init --yes` auf frischem Klon mit leerem `git status`) |
| **C** Aufräum-PR | Code, ein PR | Agent, Mensch merged | B, Punkte Maschinenzustand und Deklarationen abgehakt |

Die Zeile 4e in `migration.md` sagt heute „one clean-up pull request“; mit A
sind es zwei. Die Zeile folgt dieser Spec (`internal/plancheck` hält beide
Dateien, das Diagramm und die Spec-Tabelle gegeneinander); Priorität bleibt 3.

Abhängigkeiten der Stufe im Ganzen (die Zeile nennt sie): 4a-2 (Menschenschritte
offen), 4c-1 (Selbstnutzung offen), Remote für `brain-knowledge`, und
`parity/artefakte-nach-lebensdauer.md` (sechs Entscheidungen offen). Für A gilt
keine davon. 4c-1 und die sechs Entscheidungen blockieren C nur, wenn eine sie
berührt: der Plan liest sie vor Task 1 gegen Zustandsdateien und Manifest und
hält das Ergebnis im Ledger fest. Zu ✅ wird die Stufe erst, wenn alle vier
Abhängigkeiten erledigt sind.

## A: `loomux area check <pfad>`

Nur lesend, keine Konfiguration wird angefasst, keine Datei geschrieben.

**Gemessen am 2026-09-28 (Wegwerftest in `internal/config`, danach gelöscht).**
Die Spec #24 spricht von „annehmen, umbenennen (`[llm.local]`), ablehnen“. Im
Code steht davon nichts: `ReadAreaManifestUntilStage4` probiert die drei Namen
der Reihe nach und ruft für alle denselben Leser `ReadDeclaration`, ohne
Umbenennung, und nichts ruft `Undecoded()`. Die Probe: `ReadDeclaration` auf ein
Manifest mit `[area] scope`, `[llm.local]`, einer unbekannten Tabelle und einem
unbekannten Top-Level-Schlüssel **gibt ein Manifest und keinen Fehler zurück**.
Dasselbe gilt für sechs echte Manifeste (`brain-knowledge`, `ecoflow`,
`iam_wiki`, `space`, `ultra-brain` je `.brain.toml`, `ultraloom`
`.ultra-brain/config.toml`), von denen mindestens `brain-knowledge` ein
`[llm.local]` trägt. Ein Schlüssel, den der Leser nicht kennt, wird also nicht
abgelehnt, sondern **still ignoriert**; beim Umlegen von Hand ginge er verloren,
ohne dass etwas meldet. Das ist die eigentliche Gefahr, die der Befehl sichtbar
macht. Drei Schlüsselklassen und ein Befund der Datei, gemessen und nicht aus der Spec übernommen (die Probe
deckt sechs der zehn Bereiche; der Plan misst die übrigen vier, alle
schreibgeschützten Bereiche dabei):

**Keine Änderung am Leser.** `ReadDeclaration` liest in eine
`map[string]any` und benennt seine Schlüssel in einer Tabelle,
`config.DeclarationKeys()`; das Schema aus 4a-1 (`schema.Keys()`) nennt alle
Schlüssel, die ein Leser von `.loomux/config.toml` annimmt. `area check`
zerlegt das Manifest in Schlüssel und legt sie gegen diese zwei Tabellen: ein
Schlüssel in `DeclarationKeys` ist **angenommen**, einer nur im Schema wird von
einem anderen Leser derselben Datei gelesen (**anderswo gelesen**, kein
Verlust), keiner von beiden ist **ignoriert**. Damit gibt es keine zweite
Auslegung, und `ReadDeclaration` bleibt unverändert.

- **angenommen:** ein Schlüssel, den `ReadDeclaration` liest (`DeclarationKeys`);
- **anderswo gelesen:** ein Schlüssel des Schemas, den ein anderer Leser der
  Datei liest (`[verify]`, `[commit]`); ohne Verlust, Exit 0;
- **ignoriert:** ein Schlüssel, den kein Leser liest, mit dem Hinweis, wohin er
  im neuen Schema gehört, falls es ein Gegenstück gibt (eine kleine feste
  Tabelle, z. B. `llm.local` → `model`; nur ein Hinweis, keine Übersetzung).
  Auch ein Schlüssel, den `ReadManifest` in ein ungenutztes Feld liest
  (`check.lanes`), ist wirkungslos und damit ignoriert.

Dazu kommt **abgelehnt** als Befund der **Datei**, nicht eines Schlüssels: der
Leser gibt einen Fehler zurück (fehlendes `scope`, unbekannte Rolle unter
`[model]`, ungültiges `[inbox]`, kein gültiges TOML).

**Schnittstelle.** Der Befehl liest je Manifestname (`.loomux/config.toml`,
`.ultra-brain/config.toml`, `.brain.toml`) den Bestand und meldet **je Schlüssel**
eine Zeile: Name, Klasse, Grund. Er benutzt `ReadDeclaration`; eine eigene
zweite Auslegung ist ausgeschlossen. Die Ausgabe nennt auch, welchen Namen
`ReadAreaManifestUntilStage4` heute wählen würde.

**Exit-Code:** 0, wenn alles angenommen ist; 1, wenn Handarbeit nötig ist
(ignorierte Schlüssel, abgelehntes Manifest); 2 bei Aufruffehlern. Nach dem Umlegen
bestätigt ein zweiter Lauf: `.loomux/config.toml` ist ganz angenommen, und der
alte Name liegt nicht mehr im Weg (Exit 0).

Er entfällt in 4f mit dem Rückfall.

## B: die Checkliste

`parity/stufe-4e.md`, vom Menschen abgehakt, in Blöcken:

1. **Voraussetzungen:** `brain-knowledge` hat einen Remote und ist committet;
   der Menschenschritt von 4a-2 (`init --yes` auf frischem Klon) ist gelaufen.
2. **Vorher entscheiden (aus `stufe-3a.md`, „Umstieg“):** `[index]` für
   `project/loomux` eintragen und festlegen, welche der vier erzeugten Dateien
   (`_identities.tsv`, `graph.json`, `index.md`, `docs/index.md`) versioniert
   werden. Der erste echte `reindex` ändert sonst 345 Dateien.
3. **Maschinenzustand:** `%LOCALAPPDATA%\brain` gegen `%LOCALAPPDATA%\loomux`
   vergleichen: `registry.toml`, die Identitätsregister unter `areas/`, und in
   `maintenance/` `last-run.txt`, `merge-events.done.tsv` (und, solange ein
   Python-Hook schreibt, `merge-events.tsv`). `qmd-collections.json` steht bisher
   in keiner Checkliste und wird **nur kopiert, wenn im neuen Verzeichnis keine
   liegt** (eine dort geschriebene ist die jüngere); sonst beginnt die Liste leer
   und der erste `reindex` verweigert jede Sammlung, die qmd schon führt. Das alte
   Verzeichnis bleibt als Sicherung.
4. **Deklarationen der zehn Bereiche:** je Bereich `loomux area check`, die
   Deklaration von Hand nach `.loomux/config.toml`, `area check` erneut (Exit 0).
   Gilt für die read-only-Bereiche zuerst, weil `guard` nur `.loomux/config.toml`
   liest (Auflage aus `stufe-3a.md`). **Reihenfolge gegen `init`:** gemessen
   (Akte, Messung 1): `init` schreibt `.loomux/config.toml` ganz mit dem
   bearbeiteten Text, der Inhalt bleibt, und ein von Hand eingetragenes `[area]`
   bleibt erhalten; die Checkliste nennt das Ergebnis.
5. **Je Wirt:** `loomux init`; alte Einträge von Hand entfernen (Marke
   `ultraLoomOwned`, Gruppe `ultraloom-wiki-guard`, `brain guard`, `ulguard`),
   danach `.ultraloom/`, `.brain.toml`, `.ultra-brain/`;
   `loomux merge-hook install`, weil ein von `brain-mcp` gesetzter Hook als
   `unrecorded` gilt (die alte `hooks.tsv` wird nicht gelesen).
6. **Rauchtest je Wirt:** ein erlaubter Edit, ein verweigerter Edit, ein Commit
   durch commit-msg und pre-commit, eine Suche über MCP.

Befehle für den Menschen in Git-Bash-Syntax (`!`-Präfix), Antworten per Pipe.

## C: der Aufräum-PR

Umfang wird im Plan gegen den Code nachgerechnet; die Zahl „14 Dateien“ aus #24
ist eine Inventur der Namen, kein Umfang. Gelesen am 2026-09-28:
`ReadAreaManifestUntilStage4` hat 12 Aufrufer außerhalb von Tests,
`LegacyBrainDirUntilStage3` vier (`internal/cli/brain.go`, `dev.go`, `serve.go`,
`internal/config/artifacts.go`). `Manifest.Lanes` ist das Feld, das aus dem alten
`[check] lanes` gefüllt wird (`manifest.go`, `ReadManifest`); wer es entfernt,
prüft, ob `verify` es liest. `internal/brain/apply/stock.go` (`moveStock`,
einer der zwei Aufrufer von `lock.Recover`) kopiert den Bestand aus dem Altverzeichnis
und entfällt mit dem Rückfall — der Plan liest, ob es außer der Kopie noch etwas
tut, bevor die Spec es zusagt.

**Was C tut:**

1. Die Rückfälle entfallen samt Umgebungsvariable `LOOMUX_LEGACY_BRAIN_DIR`, den
   alten Manifestnamen und den Kommentaren, die `loomux migrate` als Ende nennen.
2. **Erschlagener Tausch (Entscheidung 2026-09-28, Variante 1′).**
   `config.ResolvedAreaDir` liefert `<ziel>.loomux-aside`, wenn das Ziel fehlt
   und das Aside da ist. Das gilt nur für den Zweig des Zustandsverzeichnisses
   (read-only-Bereiche, `ManifestDir`); ein schreibbarer Bereich tauscht nie.
   Der Plan greppt, welche Leser den Bestand über `config.ManifestDir(` oder den
   qmd-Pfad direkt öffnen und an `ResolvedAreaDir` vorbeigehen (heute:
   `apply/resolve.go`, `apply/stock.go`, `guard/registry.go`, `index/staging.go`,
   `config/registry.go`, `dev/benchsearch/corpusrun.go`); für jeden steht im
   Plan, ob er 1′ braucht. Der Leser schreibt nichts: ein `lock.Recover` im
   Lesepfad hätte einem lebenden `ReplaceDir` das Aside unter den Füßen
   weggenommen, weil `ReplaceDir` kein Lock nimmt. Geheilt (zurückgeschoben)
   wird weiter vom Indexlauf.
   **Restfenster, ehrlich:** Ein Leser, der den Aside-Pfad schon aufgelöst hat,
   kann ins Leere öffnen, wenn der Tausch fertig wird und das Aside entfernt.
   Er trifft dann `ErrNoManifest`, und `privacy.VisibleAreas` verweigert weiter
   alle Bereiche: derselbe Ausfall wie heute, nur im engeren Fenster (Zeit
   zwischen Auflösen und Öffnen statt Zeit bis zur Heilung). Es ist eine
   Verkleinerung, keine Beseitigung.
   Nach Vorschrift von `stufe-3a.md` (Abschnitt „der abwesende Bereichsordner“):
   der Absatz dort wird gestrichen und die Entscheidung in die Abweichungstabelle
   der Stufe eingetragen; der Kommentar an `ReplaceDir` („That fallback ends with
   stage 4“) wird nachgezogen.
3. Die Ratschläge nennen `loomux reconcile` und `loomux reindex`
   (`ReconcileAdvice` in `internal/brain/search/stamp.go`, `status.go`, zwölf
   Meldungen in `internal/brain/graph/read.go`); die berührten Fälle aus 1b-1 und
   1b-2 bekommen die neue Erwartung.
4. `migration.md` (en/de), Roadmap, beide READMEs und die Fusions-Spec folgen; 4e
   wird ✅ erst mit der Checkliste.

## Tests

- **A:** ein Fall je Klasse gegen ein Manifest mit angenommenem, ignoriertem und
  abgelehntem Schlüssel; Exit 0/1/2; der Befehl schreibt nichts (Verzeichnis vorher
  gegen nachher); und ein Lauf gegen die zehn echten Bereiche als Selbstnutzung,
  ohne dass dabei ein Bereich verändert wird. Ein Test, der `area check` gegen
  `ReadDeclaration` hält: liest der Leser einen Schlüssel neu, ändert sich die
  Klasse mit.
- **C, Aside:** diskriminierend sind drei Fälle — Aside und kein Ziel liefert das
  Aside; Aside und Ziel liefert das Ziel; keins von beiden verhält sich wie heute.
  Ein Fall stellt ein erschlagenes `ReplaceDir` nach (Aside ohne Ziel) und prüft,
  dass `VisibleAreas` weiter antwortet. Ein Test „der Leser schreibt nichts“
  entfällt: die Lösung hat keinen Schreibaufruf, keine Mutation würde ihn rot machen.
- **C, Rückfälle:** die bestehenden Fälle, die den Rückfall prüften, werden
  entfernt oder auf den neuen Zustand umgestellt; kein Test bleibt, der das
  Altverzeichnis noch erwartet.
- Wie bei jeder Stufe: 100 % Coverage je Funktion, Mutationsrunde mit
  dokumentierten Überlebenden (Bedingung 3 der Fusions-Spec, für A und C).

## Messen (Bedingung 4)

`area check` liegt auf keinem heißen Pfad und wird nicht gemessen. Gemessen
wird 1′: `loomux brain search` und `privacy.VisibleAreas` warm, vor und nach der
Änderung, mit vorhandenem Ziel (der Regelfall; hier kostet 1′ nichts, was die
Messung zeigen muss) und mit fehlendem Ziel und vorhandenem Aside. Eingetragen
in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`.

## Nachzutragen in der Fusions-Spec, bevor der Plan sie kennt

- Zeile „4e Umstellung“: die Aufteilung A/B/C, `area check` mit seinen Klassen
  aus Messung statt aus der Vermutung `[llm.local]`, die Entscheidung 1′ für den
  Aside-Fall und die zwei Punkte der Checkliste, die bisher fehlten
  (`qmd-collections.json`, `[index]` für `project/loomux`).
- Nachtrag #24, Klasse (a): „umbenennen (`[llm.local]`)“ ist im Code ohne
  Gegenstück; die Messung vom 2026-09-28 bestätigt es und ersetzt „umbenennen“
  durch „ignoriert“. Das korrigiert die Entscheidung vom 2026-09-28 im Wortlaut
  (`area check` bleibt, seine Klassen ändern sich), der Nutzer liest es bei der
  Durchsicht dieser Spec.

## Offene Punkte (im Plan zu messen)

- Welche der zehn Manifeste tragen Schlüssel, die der Leser ignoriert (sechs
  gemessen, vier offen).
- Ob `init` ein von Hand eingetragenes `[area]` in `.loomux/config.toml` erhält
  (`init --dry-run`, bestimmt die Reihenfolge der Blöcke 4 und 5).
- Ob `Manifest.Lanes` außer `ReadManifest` einen Leser hat.
- Ob `moveStock` außer der Kopie noch etwas tut.
- Ob die sechs Entscheidungen aus `parity/artefakte-nach-lebensdauer.md` Zustands-
  dateien oder das Manifest berühren.

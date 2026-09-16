# Scheibe 0 — Fundament ohne Code · Implementierungsplan

> **Für ausführende Agenten:** ERFORDERLICHER SUB-SKILL: `superpowers:subagent-driven-development` (empfohlen) oder `superpowers:executing-plans`, um diesen Plan Aufgabe für Aufgabe abzuarbeiten. Schritte nutzen Checkbox-Syntax (`- [ ]`).

**Ziel:** Ein neuer, leerer Wissens-Vault steht mit Manifest, Katalog und Regelwerk; qmd findet in den bestehenden Beständen; und die Architekturfrage „trägt qmd sprachübergreifend?" ist mit Zahlen beantwortet.

**Vorgehen:** Kein Anwendungscode. Zuerst zwei Tore, die das Projekt kippen könnten (läuft qmd unter Windows? findet es sprachübergreifend?), dann der neue Vault, Regelwerk und Katalog, zum Schluss die Messung gegen den Zustand ohne System.

**Werkzeuge:** qmd (Node ≥ 22, vorhanden: v24.14.1), Claude Code, Obsidian, git. Keine Python-Abhängigkeiten in dieser Scheibe.

**Spec:** `docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md`

## Globale Rahmenbedingungen

Gelten für jede Aufgabe, aus der Spec wörtlich übernommen:

- **Markdown ist die einzige Wahrheit.** Alles andere ist abgeleitet und wegwerfbar (§3.1).
- **Rohquellen sind unantastbar** (§3.6). In dieser Scheibe verschärft: siehe die Bestandsregel unten.
- **Der Mensch entscheidet** (§3.3). Jede Modellwahl, jeder Ort, jede Aufnahme in den Bestand wird vorgeschlagen, nie ausgeführt.
- **Deterministisch, wo es geht** (§3.2): Zählen, Auflisten und Messen per Shell, nicht per Modell.
- **Sprachen:** Prosa deutsch; Manifest-Schlüssel, Scope-Werte und alles Maschinennahe englisch (§14). Die deutschen Ordnernamen des Vaults sind eine Entscheidung des Nutzers (§5.2) — die ausgelieferte Vorgabe des Werkzeugs bleibt englisch.
- **Messung vor Entscheidung**, nie danach (§13).

### Die Bestandsregel dieser Scheibe

**Der neue Wissens-Vault startet leer. Die bestehenden Vaults werden weder verschoben noch zusammengeführt noch umsortiert.**

Erlaubt ist ausschließlich **lesendes Indexieren**: qmd legt seinen Index im Zustandsverzeichnis ab und ändert keine einzige Datei im Quellordner. Ohne das würde der Spike an erfundenen Testdateien messen statt an echtem Wissen — und wäre wertlos.

Nicht erlaubt, auch nicht „nur kurz": Dateien verschieben, umbenennen, Frontmatter ergänzen, Ordner umbauen, `.obsidian`-Einstellungen ändern. Wenn eine Aufgabe das nahelegt, ist die Aufgabe falsch, nicht die Regel.

Betroffene Bestände (Stand heute, rund 290 Markdown-Notizen über drei Bestände, §16.11):

| Pfad | Dateien | davon Notizen |
|---|---:|---:|
| `C:\Users\micro\Documents\#GIT\#Obsidian\AI` | 23 | 23 |
| `C:\Users\micro\Documents\#GIT\iam_wiki` | 116 | 88 (28 sind Arbeitsspuren) |
| `C:\Users\micro\Documents\#GIT\space` (enthält `space\wiki` mit 147) | 584 | ~150 (Rest: JDK-Lizenztexte unter `.tools/`, Testfixturen, Worktree-Kopie unter `.claude/`) |

Die Dateizahl ist **nicht** die Notizzahl. Genau diese Verwechslung war laut
§16.11 die Ursache der zweimal zu hoch angesetzten Größenordnung.

`Documents\GIT_COPY\` ist eine Spiegelung und bleibt vollständig außen vor.

### qmd-Sammlung ist nicht gleich Bereich

Zwei Begriffe, die in dieser Scheibe leicht verwechselt werden:

| | qmd-Sammlung | Bereich (§5.5) |
|---|---|---|
| Was | Ein Ordner, den qmd indexiert | Eine Einheit des Second Brain mit Manifest, Scope, Wiki |
| Wirkung | rein lesend, Index im Zustandsverzeichnis | Teil des Systems, wird verdichtet und gewartet |
| In Scheibe 0 | die drei bestehenden Vaults **als Messgrundlage** | ausschließlich der neue, leere Vault |

Die bestehenden Vaults werden **nicht** zu Bereichen. Ob und wann sie es werden, ist eine spätere Entscheidung.

---

## Dateistruktur

**Im `ultra-brain`-Repo** (wird veröffentlicht — nichts Privates):

| Datei | Zweck |
|---|---|
| `docs/.superpowers/plans/2026-08-18-scheibe-0-fundament.md` | dieser Plan |
| `docs/.superpowers/sdd/spike-mehrsprachigkeit.md` | Protokoll des Modellvergleichs — **gitignored**, zieht in Aufgabe 4 Schritt 6 in den Vault um |
| `docs/.superpowers/sdd/vergleich-mit-ohne.md` | Protokoll der Token- und Zeitmessung — ebenso |
| `bench/questions.example.yaml` | anonymisiertes Beispiel des Fragensatzes für fremde Nutzer |

Die Protokolle liegen zunächst im gitignorierten Arbeitsspur-Ordner, weil sie Pfade und Inhalte privater Notizen nennen und das Repo veröffentlicht wird (§5.1). Sie ziehen in den Vault um, sobald es ihn gibt.

**Im neuen Wissens-Vault** (privat, eigenes Repo, entsteht in Aufgabe 4):

| Datei | Zweck |
|---|---|
| `.brain.toml` | Manifest des Bereichs `knowledge` |
| `.gitignore` | schließt Obsidian-Cache und Abgeleitetes aus |
| `CLAUDE.md` | Brain-First-Regelwerk |
| `index.md` | Wurzelkatalog |
| `98 Messung/questions.yaml` | Fragensatz mit erwarteten Zielquellen |
| `98 Messung/*.md` | die umgezogenen Protokolle |

**Im Zustandsverzeichnis** (nie versioniert): qmd-Index und Modelle.

---

## Aufgabe 1: Tor — läuft qmd auf diesem Rechner?

Das erste Tor, weil ein Scheitern den Suchkern der Architektur sofort ändert (§16.10). Windows wird von qmd nicht dokumentiert.

**Dateien:**
- Anlegen: `docs/.superpowers/sdd/spike-mehrsprachigkeit.md`
- Wegwerfmaterial im Temp-Verzeichnis

**Liefert an spätere Aufgaben:** ob qmd nutzbar ist, und den exakten Befehl zur Modellwahl.

- [ ] **Schritt 1: qmd installieren**

```bash
npm install -g @tobilu/qmd
qmd --version
```

Erwartung: eine Versionsnummer. Bricht die Installation ab, notiere die vollständige Fehlermeldung und gehe zu Schritt 6.

- [ ] **Schritt 2: Wegwerf-Sammlung mit drei Notizen anlegen**

```bash
mkdir -p /c/Users/micro/AppData/Local/Temp/qmd-probe
cd /c/Users/micro/AppData/Local/Temp/qmd-probe
printf 'Unser Stil für Untertitel: kurze Zeilen, keine Satzzeichen am Ende.\n' > untertitel.md
printf 'Error handling policy: fail fast, never swallow exceptions silently.\n' > errors.md
printf 'Einkaufsliste für Samstag: Brot, Milch, Kaffee.\n' > einkauf.md
```

Bewusst gewählt: eine deutsche Fachnotiz, eine englische Fachnotiz, eine unbeteiligte Ablenkung.

- [ ] **Schritt 3: Indexieren**

```bash
qmd collection add /c/Users/micro/AppData/Local/Temp/qmd-probe --name probe
qmd embed
```

Erwartung: Die Modelle werden geladen (rund 2 GB) und der Index entsteht ohne Abbruch.

- [ ] **Schritt 4: Suche prüfen**

```bash
qmd query "Wie formatieren wir Untertitel?" -n 3
```

Erwartung: `untertitel.md` steht oben. Bleibt sie aus, ist das noch kein Ausschlusskriterium — notiere das Ergebnis.

- [ ] **Schritt 5: Befehl für die Modellwahl herausfinden**

```bash
qmd --help
qmd embed --help
```

Notiere den exakten Weg, auf dem sich das Embedding-Modell auf die mehrsprachige Qwen3-Variante umstellen lässt. **Nicht dem README folgen:** Der dort beschriebene Weg über die Umgebungsvariable `QMD_EMBED_MODEL` ist **wirkungslos**, weil die Konfigurationsdatei Vorrang hat (§16.10). Wirksam ist ausschließlich der Eintrag in `~/.config/qmd/index.yml`:

```bash
sed -i 's|^  embed: .*|  embed: <modell-uri>|' /c/Users/micro/.config/qmd/index.yml
qmd embed -f
```

Notiere die geladene Fassung aus der Ausgabe (`Model: …`). **Dieser Weg wird in Aufgabe 3 gebraucht** — ohne ihn misst der Modellvergleich zweimal dasselbe Modell.

- [ ] **Schritt 6: Protokoll schreiben und entscheiden**

In `docs/.superpowers/sdd/spike-mehrsprachigkeit.md`: Version, Installationsverlauf, Ergebnis der Testsuche, Modellwahl-Befehl.

**Abbruchkriterium:** Läuft qmd nicht oder bricht das Indexieren reproduzierbar ab, endet Scheibe 0 hier. Der Befund geht in die Spec (§16.10 wird von „Risiko" zu „eingetreten"), und Entscheidung 6 zum Suchkern wird neu getroffen. **Nicht weiterarbeiten, als wäre nichts gewesen.**

- [ ] **Schritt 7: Aufräumen**

```bash
qmd collection remove probe
rm -rf /c/Users/micro/AppData/Local/Temp/qmd-probe
```

---

## Aufgabe 2: Der Fragensatz — vor jeder Messung

Muss **vor** dem Spike entstehen, sonst wird er unbewusst an die Ergebnisse angepasst (§13).

Da in dieser Scheibe nichts verschoben wird, bleiben die Pfade dauerhaft gültig — der Fragensatz muss später nicht nachgezogen werden.

**Dateien:**
- Anlegen: `docs/.superpowers/sdd/questions.yaml` (zieht in Aufgabe 4 in den Vault um)
- Anlegen: `bench/questions.example.yaml`

**Liefert an spätere Aufgaben:** den festen Satz für Aufgabe 3 und Aufgabe 7.

- [ ] **Schritt 1: Kandidaten aus dem Bestand sammeln**

Claude durchsucht die drei Vaults **lesend** und schlägt je Sorte Fragen in der Zielverteilung 8 / 8 / 6 / 8 vor, jeweils mit der Datei, die getroffen werden müsste:

| Sorte | Prüft |
|---|---|
| `exakt` | Stichwortsuche |
| `umschreibung` | Bedeutungssuche |
| `gemischt` | ob die Hybridpipeline trägt |
| `sprachuebergreifend` | das mehrsprachige Modell |

Die vierte Sorte ist die wichtigste: **deutsche Frage, englische Zielquelle** — und mindestens einmal umgekehrt.

- [ ] **Schritt 2: Du bestätigst oder ersetzt jede Frage**

Eine Frage taugt nur, wenn du **vorher** weißt, welche Datei die Antwort enthält, und wenn du sie im Alltag tatsächlich so stellen würdest. Erfundene Fragen messen nichts.

- [ ] **Schritt 3: Fragensatz festschreiben**

```yaml
# Erfolgsmaß: Liegt die erwartete Datei unter den ersten drei Treffern?
# NICHT der Relevanzwert — ein hoher Wert ist kein Wahrheitsbeweis (Spec §13).
# Pfade sind absolut, weil in dieser Scheibe nichts verschoben wird.

- id: q01
  sort: exakt
  query: "<deine Frage>"
  expect: "C:/Users/micro/Documents/#GIT/space/<datei>.md"
- id: q02
  sort: umschreibung
  query: "<deine Frage>"
  expect: "C:/Users/micro/Documents/#GIT/iam_wiki/<datei>.md"
- id: q03
  sort: gemischt
  query: "<deine Frage>"
  expect: "C:/Users/micro/Documents/#GIT/space/<datei>.md"
- id: q04
  sort: sprachuebergreifend
  query: "<deutsche Frage>"
  expect: "C:/Users/micro/Documents/#GIT/space/<englische-datei>.md"
```

30 Einträge in der Verteilung 8 / 8 / 6 / 8 (exakt / umschreibung / gemischt / sprachuebergreifend), wie in §13 festgelegt.

- [ ] **Schritt 4: Prüfen, dass jede Zielquelle existiert**

```bash
grep 'expect:' docs/.superpowers/sdd/questions.yaml \
  | sed 's/.*expect: *"\(.*\)"/\1/' \
  | while read -r p; do [ -f "$p" ] || echo "FEHLT: $p"; done
```

Erwartung: keine Ausgabe.

- [ ] **Schritt 5: Anonymisiertes Beispiel fürs Repo**

`bench/questions.example.yaml` bekommt dieselbe Struktur mit erfundenen Fragen und Pfaden — die Form für fremde Nutzer, ohne deinen Bestand preiszugeben.

- [ ] **Schritt 6: Commit**

```bash
git add bench/questions.example.yaml
git commit -m "Add example benchmark question set"
```

Der echte Fragensatz bleibt ungetrackt und zieht in Aufgabe 4 in den Vault.

---

## Aufgabe 3: Spike — trägt qmd sprachübergreifend?

Die architekturkritische Messung. Zwei Modelle gegeneinander, nicht „qmd ja/nein" (§16.3).

**Dieser Schritt ist rein lesend.** qmd legt seinen Index im Zustandsverzeichnis ab; in den Quellordnern wird keine Datei erzeugt, geändert oder gelöscht.

**Dateien:**
- Fortschreiben: `docs/.superpowers/sdd/spike-mehrsprachigkeit.md`

**Verbraucht:** den Fragensatz aus Aufgabe 2, den Modellwahl-Befehl aus Aufgabe 1 Schritt 5.

### Was Aufgabe 1 für diese Aufgabe geklärt hat

Drei Befunde aus dem Tor, ohne die diese Messung falsche Zahlen erzeugen würde:

**1 · Das Modell wird über die Konfigurationsdatei gesetzt, nicht über eine
Umgebungsvariable.** Der im README beschriebene Weg über `QMD_EMBED_MODEL` ist
wirkungslos — die Konfiguration hat Vorrang. Wirksam ist:

```bash
sed -i 's|^  embed: .*|  embed: <modell-uri>|' /c/Users/micro/.config/qmd/index.yml
qmd embed -f
```

**Vor jedem Durchgang wird das Modell ausdrücklich gesetzt und die geladene
Fassung aus der Ausgabe (`Model: …`) ins Protokoll übernommen.** Nicht darauf
verlassen, dass ein früherer Lauf aufgeräumt hat — sonst misst „Durchgang A"
womöglich still das Modell aus Durchgang B.

**2 · Trefferpfade werden über Dateiname plus übergeordneten Ordner
verglichen**, nicht über den vollen Pfad. Die Standardausgabe ist ein
sammlungsrelativer URI der Form `qmd://sammlung/pfad/datei.md #hash`; die
Variante `--format files --full-path` liefert absolute Pfade, kippt aber auf
`./`-relative, sobald das Arbeitsverzeichnis über der Sammlung liegt, und
ändert zusätzlich die Spaltenzahl. Der Vergleich über Name plus Elternordner
übersteht alle drei Formate. Sind zwei Kandidaten darin gleich, entscheidet
der volle Pfad.

**3 · `PATH` muss das npm-Verzeichnis enthalten:**

```bash
export PATH="$PATH:/c/Users/micro/AppData/Roaming/npm"
```

**Zeitbedarf:** Kalt kostet eine einzelne Abfrage rund 30 s (Modell-Ladezeit,
davon 27 s Frageerweiterung). Zwei Durchgänge über 30 Fragen
plus zweimal Einbetten von rund 290 Notizen sind keine Nebenbei-Aufgabe — plane die
Zeit ein, statt mittendrin abzubrechen.

- [ ] **Schritt 1: Ausgangsstand der Bestände festhalten**

Damit am Ende belegbar ist, dass nichts angefasst wurde. **Ein Dateizähler
taugt dafür nicht** — §16.15 hält gemessen fest, dass fremde Prozesse im selben
Baum (dort eine laufende Godot-Sitzung) die Gesamtdateizahl ändern, ohne dass
eine Notiz berührt wurde. Tragfähig ist `git status` plus die Änderungszeiten
der **Notizen selbst**:

```bash
for v in "#Obsidian/AI" "iam_wiki" "space"; do
  d="/c/Users/micro/Documents/#GIT/$v"
  [ -d "$d/.git" ] && git -C "$d" status --short | sed "s|^|$v |"
  find "$d" -type f -name '*.md' \
    -not -path '*/.git/*' -not -path '*/.claude/*' \
    -not -path '*/.tools/*' -not -path '*/.superpowers/*' \
    -printf '%T@ %p\n' | sort
done | tee docs/.superpowers/sdd/bestand-vorher.txt
```

**Zusatzhinweis, kein Nachweis:** die reine Dateizahl je Bestand, danebengelegt,
um grobe Ausreißer zu sehen.

- [ ] **Schritt 2: Bestände als Sammlungen registrieren — mit verbindlichem Ausschluss**

```bash
qmd collection add "/c/Users/micro/Documents/#GIT/#Obsidian/AI" --name obsidian-ai
qmd collection add "/c/Users/micro/Documents/#GIT/iam_wiki"     --name iam-wiki
qmd collection add "/c/Users/micro/Documents/#GIT/space"        --name space
```

`space/wiki` liegt innerhalb von `space` und wird nicht gesondert registriert — sonst stünde derselbe Inhalt doppelt im Index. `GIT_COPY` bleibt außen vor.

**Diese Pfade müssen ausgeschlossen werden, sonst ist die Messung wertlos:**

| Auszuschließen | Warum |
|---|---|
| `**/.claude/**` | `space/.claude/worktrees/schiebereglerseite/` spiegelt `space/wiki/` **byte-identisch**. Ein erheblicher Teil der 30 Zielquellen hätte dort einen Zwilling — die richtige Datei konkurriert mit ihrem eigenen Klon um die ersten drei Plätze, und gemessen würde Duplikatrauschen statt Trefferqualität. |
| `**/.tools/**` | JDK-Lizenztexte, kein Wissen |
| `**/.superpowers/**` | Arbeitsspuren, kein Wissen |
| `**/tests/fixtures/**` | Testdaten, teils leicht abweichende Kopien echter Notizen |
| `**/.git/**`, `**/.obsidian/**` | Werkzeugdaten |

Der Ausschluss ist keine Optimierung, sondern die sachlich richtige Indexkonfiguration: Worktree-Kopien, Lizenztexte und Testfixturen sind kein Wissen. Findet der Ausschluss über qmd keinen Weg, ist die Alternative, die betroffenen Fragen auf zwillingsfreie Zieldateien umzustellen — nicht, ohne Ausschluss zu messen.

- [ ] **Schritt 2b: Ausschluss überprüfen, bevor gemessen wird**

Für jede Zielquelle aus dem Fragensatz prüfen, dass sie **genau einmal** im Index steht:

```bash
grep 'expect:' "<pfad>/questions.yaml" | sed 's/.*expect: *"\(.*\)"/\1/' \
  | while read -r p; do
      n=$(basename "$p")
      c=$(qmd query "$n" --format files 2>/dev/null | grep -c "$n")
      echo "$c  $n"
    done
```

Erwartung: überall `1`. Steht dort eine Zahl größer eins, greift der Ausschluss nicht — **dann nicht weitermessen**, sondern erst die Konfiguration korrigieren.

- [ ] **Schritt 3: Durchgang A — Standardmodell**

```bash
qmd embed
```

Dann jede Frage aus dem Fragensatz einzeln:

```bash
qmd query "<query>" -n 5
```

Je Frage notieren: Liegt `expect` unter den ersten drei? Ja/Nein.

- [ ] **Schritt 4: Durchgang A auswerten**

```
Sorte                 | Treffer in Top-3
exakt                 | x/8
umschreibung          | x/8
gemischt              | x/6
sprachuebergreifend   | x/8
gesamt                | x/30
```

**Nachträglich korrigiert (nach der Messung):** Hier stand die Erwartung, die
vierte Sorte falle ab, weil das Standardmodell englischoptimiert sei. Das ist
**widerlegt** (§16.3). Gemessen wurden 23/30 für beide Modelle, bei den
belastbaren sprachübergreifenden Fragen 7/7 für das Standardmodell gegen 6/7
für das mehrsprachige. Die Erwartung steht hier nur noch als das, was sie war —
eine Vermutung, die die Messung nicht bestätigt hat.

- [ ] **Schritt 5: Durchgang B — mehrsprachiges Modell**

Modell mit dem in Aufgabe 1 notierten Befehl umstellen, neu einbetten, denselben Fragensatz laufen lassen, gleiche Auswertung.

- [ ] **Schritt 6: Zeiten und Größen mitschreiben**

Dauer des Einbettens je Durchgang, Plattenbedarf des Index, Antwortzeit einer Einzelabfrage. Diese Zahlen sind die Ausgangsbasis für die Latenzmessung in Scheibe 2 (§13).

- [ ] **Schritt 7: Belegen, dass nichts verändert wurde**

Denselben Befehl wie in Schritt 1 erneut laufen lassen und vergleichen:

```bash
for v in "#Obsidian/AI" "iam_wiki" "space"; do
  d="/c/Users/micro/Documents/#GIT/$v"
  [ -d "$d/.git" ] && git -C "$d" status --short | sed "s|^|$v |"
  find "$d" -type f -name '*.md' \
    -not -path '*/.git/*' -not -path '*/.claude/*' \
    -not -path '*/.tools/*' -not -path '*/.superpowers/*' \
    -printf '%T@ %p\n' | sort
done > docs/.superpowers/sdd/bestand-nachher.txt
diff docs/.superpowers/sdd/bestand-vorher.txt docs/.superpowers/sdd/bestand-nachher.txt && echo "unverändert"
```

Erwartung: keine Unterschiede — kein Eintrag aus `git status`, keine geänderte
Änderungszeit einer Notiz. Steht dort etwas, wurde die Bestandsregel verletzt —
Ursache klären, bevor es weitergeht. Eine abweichende **Gesamtdateizahl** allein
ist kein Verstoß (§16.15), sondern ein Anlass nachzusehen, welche Datei es
betrifft.

- [ ] **Schritt 8: Entscheiden und in die Spec zurückspielen**

| Ergebnis | Folge |
|---|---|
| **Beide Modelle tragen, kein messbarer Unterschied** | **Entscheidung 6 bestätigt — aber begründet durch die Leistung der *Suchkette*, nicht durch ein Modell. Die Modellwahl fällt dann nach Kosten (Einbettungszeit, Indexgröße), nicht nach Trefferqualität, und der Vorbehalt lautet: die Aussage gilt nur, solange die volle Hybridkette samt Reranker aktiv ist.** |
| Modell B trägt sprachübergreifend, A fällt ab | Modell B wird gesetzt; Entscheidung 6 bestätigt |
| Beide Modelle scheitern sprachübergreifend | Entscheidung 6 wird geöffnet: eigener Suchkern rückt vor — genau dafür wurde die Suche gekapselt |
| Modell B trägt, verliert aber deutlich bei exakten Begriffen | beide behalten und über Profile trennen — Befund für Scheibe 2 |

**Die erste Zeile fehlte ursprünglich — und genau dieser Fall trat ein.** Eine
Entscheidungstabelle ohne die Zeile „kein Unterschied" zwingt dazu, das
nächstliegende Feld anzukreuzen, und erzeugt so eine Begründung, die zum
Ergebnis nicht passt. Nachgetragen nach Aufgabe 3.

**Was diese Messung grundsätzlich nicht kann:** Sie vergleicht zwei
*Pipelines*, die sich in einem von drei Modellen unterscheiden.
Frageerweiterung und Reranker sind in beiden Durchgängen dieselben und beide
mehrsprachig. Ein gleiches Ergebnis beweist deshalb **nicht**, dass das
Einbettungsmodell für die Sprachkreuzung gleichgültig ist — es kann ebenso
bedeuten, dass die Brücke gar nicht dort liegt.

Daraus folgen **zwei verschiedene Nachfolgeläufe** (beide Scheibe 2, §16.3):
Wer den Anteil des *Einbettungsmodells* wissen will, braucht einen rein
vektoriellen Lauf ohne Frageerweiterung und ohne Reranking — dafür der
Unterbefehl `qmd vsearch` statt des sonst durchgängig verwendeten `qmd query`;
`query` fährt die volle Kette, `vsearch` bewusst nur den Vektorteil. Wer wissen
will, ob das **`fast`-Profil** sprachübergreifend trägt, braucht einen eigenen
Lauf mit `fast` — denn `fast` ist nicht rein vektoriell, sondern
Frageerweiterung plus Embedding ohne Reranker (§7.3). Der `vsearch`-Lauf
beantwortet diese zweite Frage **nicht**.

Ergebnis und Entscheidung als Absatz in die Spec (§16.3 aktualisieren).

---

## Aufgabe 4: Den neuen Wissens-Vault anlegen

Er startet **leer**. Was hineinkommt, entscheidest du später Stück für Stück über den Eingangsordner.

**Dateien:**
- Anlegen: `<vault>/.brain.toml`, `.gitignore`, Ordnerskelett
- Verschieben: die Protokolle aus `docs/.superpowers/sdd/` nach `<vault>/98 Messung/`

- [ ] **Schritt 1: Ort festlegen**

Ein neues Verzeichnis, das mit keinem bestehenden Vault verschachtelt ist — sonst indexiert qmd später denselben Inhalt doppelt und Obsidian streitet sich um Einstellungen. Vorschlag: `C:\Users\micro\Documents\#GIT\brain-knowledge`. Du entscheidest.

- [ ] **Schritt 2: Verzeichnis und Repo anlegen**

```bash
mkdir -p "<vault>" && cd "<vault>" && git init -q
```

- [ ] **Schritt 3: Ordnerskelett**

Nur was Scheibe 0 braucht. `90 Wiki/`, `91 Projekte/`, `92 Engineering/` und `95 Prüfzentrum/` entstehen erst in Scheibe 3 und 5 — Ordner, die monatelang leer bleiben, verwirren nur.

```bash
mkdir -p "00 Eingang" "01 Vorlagen" "10 Rohquellen" "98 Messung" "99 Archiv"
```

- [ ] **Schritt 4: `.gitignore`**

```gitignore
# Obsidian-Arbeitsdateien: Cache und Fensterzustand, nicht der Wissensstand.
.obsidian/workspace.json
.obsidian/cache/

# Abgeleitetes: qmd-Index und Modelle gehören nie in den versionierten Baum.
.brain/
```

- [ ] **Schritt 5: Manifest**

```toml
[area]
scope = "knowledge"
wiki  = true

[layout]
inbox   = "00 Eingang"
sources = "10 Rohquellen"
review  = "95 Prüfzentrum"

[index]
include = ["10 Rohquellen/**/*.md"]
exclude = ["99 Archiv/**", ".obsidian/**"]

[privacy]
mode  = "manual_cloud"
never = ["**/*.env", "**/secrets/**"]

[maintenance]
watch = true
stale_after_days = 90

[llm.local]
enabled = false
```

Die `never`-Liste ist ernst gemeint: Trage ein, was ein Cloud-Modell auch dann nicht sehen soll, wenn es versehentlich unter `include` fällt (§5.5). `manual_cloud` heißt, dass du jedes Paket vor der Übertragung siehst (§7.2.1).

- [ ] **Schritt 6: Protokolle und Fragensatz umziehen**

```bash
mv docs/.superpowers/sdd/spike-mehrsprachigkeit.md "<vault>/98 Messung/"
mv docs/.superpowers/sdd/questions.yaml            "<vault>/98 Messung/"
```

Ab hier ist der Vault der Ort für Messartefakte.

- [ ] **Schritt 7: Commit**

```bash
cd "<vault>" && git add -A
git commit -m "Set up knowledge vault: manifest, gitignore, folder skeleton"
```

---

## Aufgabe 5: Katalog und Regelwerk

**Dateien:**
- Anlegen: `<vault>/index.md`, `<vault>/CLAUDE.md`

- [ ] **Schritt 1: Wurzelkatalog**

Der Vault ist leer — der Katalog beschreibt deshalb die Struktur und verweist auf die extern indexierten Bestände:

```markdown
# Katalog

## In diesem Vault

* [Eingang](00 Eingang/) - Neues, noch nicht eingeordnet
* [Rohquellen](10 Rohquellen/) - eingeordnetes Wissen (noch leer)
* [Messung](98 Messung/) - Fragensatz und Messprotokolle

## Extern indexiert (nur lesend, nicht Teil dieses Vaults)

* `#GIT/space` - größter Altbestand, 584 Dateien / davon rund 150 Notizen
* `#GIT/iam_wiki` - 116 Dateien / davon 88 Notizen
* `#GIT/#Obsidian/AI` - 23 Dateien / davon 23 Notizen

> Jeder neue Bereich bekommt sofort eine Zeile in diesem Katalog.
```

Der zweite Abschnitt ist wichtig: Ohne ihn weiß eine frische Sitzung nicht, dass es außerhalb des Vaults durchsuchbares Wissen gibt.

- [ ] **Schritt 2: Brain-First-Regelwerk**

`<vault>/CLAUDE.md`, Wortlaut aus §8, angepasst daran, dass es noch keine MCP-Werkzeuge gibt — Claude ruft `qmd` über die Shell:

```markdown
# Brain-First-Protokoll

Bei jeder Wissensfrage gilt diese Reihenfolge. Nie blind suchen,
nie ganze Ordner einlesen.

1. `index.md` lesen — den Wurzelkatalog. Steht die Quelle dort,
   öffne direkt diese.
2. Wiki lesen, sobald es existiert (ab Scheibe 3).
3. `qmd query "<frage>" -n 5` — Kandidaten anhand der Trefferliste
   beurteilen, ohne Dateien zu öffnen.
4. Genau EINE Datei öffnen, darin nur den relevanten Abschnitt.
5. Antworten, mit Quellenangabe.

Grenzen:
- Höchstens zwei Suchläufe je Frage. Bringt der zweite nichts,
  sag das, statt weiterzusuchen.
- Ein hoher Relevanzwert ist kein Beweis. Bei wichtigen Aussagen
  bis zur Rohquelle durchgehen.
- Widersprüche werden zitiert, nicht aufgelöst.

Bestandsregel (Scheibe 0):
- Die extern indexierten Vaults werden NUR GELESEN. Dort wird nichts
  verschoben, umbenannt, ergänzt oder gelöscht — auch nicht auf Zuruf,
  ohne dass die Bestandsregel ausdrücklich aufgehoben wurde.
- Geschrieben wird ausschließlich in diesem Vault.
```

Die Zwei-Suchlauf-Grenze ist kein Sparzwang: Jeder weitere Suchschritt liest den bisherigen Gesprächsverlauf erneut, sodass die Kosten von Runde zu Runde wachsen (§8).

- [ ] **Schritt 3: In einer frischen Sitzung prüfen**

Neue Claude-Code-Sitzung im Vault, eine Frage stellen, deren Antwort in einer tief liegenden Notiz eines externen Bestands steht. Beobachten: Wird zuerst `index.md` gelesen? Bleibt es bei höchstens zwei Suchläufen? Wird genau eine Datei geöffnet?

Hält sich die Sitzung nicht daran, liegt es am Wortlaut, nicht am Modell — schärfen und wiederholen.

- [ ] **Schritt 4: Commit**

```bash
git add index.md CLAUDE.md
git commit -m "Add root catalog and brain-first search ladder rules"
```

---

## Aufgabe 6: Der Praxisvergleich

Der Prüfpunkt, ohne den das Fundament nur eine Behauptung wäre (§13, Messung 2).

**Dateien:**
- Anlegen: `<vault>/98 Messung/vergleich-mit-ohne.md`

- [ ] **Schritt 1: Fünf echte Fragen wählen**

Aus dem Fragensatz die fünf, die du im Alltag wirklich stellst. Mindestens zwei sollten Wissen aus **mehreren** Dateien verbinden — dort entsteht der Unterschied, nicht bei Einzelfakten.

- [ ] **Schritt 2: Durchgang ohne System**

Frische Claude-Code-Sitzung in einem neutralen, leeren Ordner. Jede Frage stellen, danach `/cost`. Token, Zeit und Richtigkeit notieren.

- [ ] **Schritt 3: Durchgang mit System**

Frische Sitzung im Vault, dieselben fünf Fragen, gleiche Erfassung, zusätzlich `/context` für den Füllstand.

- [ ] **Schritt 4: Auswerten**

```markdown
| Frage | ohne: Token / Zeit / richtig | mit: Token / Zeit / richtig |
```

**Ehrliche Erwartung:** Bei einfachen Fragen ist der Unterschied klein oder null. Das System gewinnt bei tief vergrabenem Wissen und bei Fragen über mehrere Dateien. Zeigt die Messung keinen Unterschied, ist das ein Ergebnis — kein Grund, sie zu wiederholen, bis sie gefällt.

- [ ] **Schritt 5: Commit**

```bash
git add "98 Messung"
git commit -m "Add measurement protocol: with and without brain"
```

---

## Fertig-Kriterium der Scheibe

Alle fünf Punkte müssen erfüllt sein:

1. **qmd läuft** auf diesem Rechner — oder der Gegenbefund steht in der Spec und Entscheidung 6 ist neu getroffen.
2. **Die sprachübergreifende Frage ist beantwortet**, mit Zahlen aus zwei Modelldurchgängen.
3. **Die bestehenden Bestände sind nachweislich unverändert** — `git status` in jedem Bestandsrepo leer und die Änderungszeiten der Notizen unverändert (§16.15). Die Dateizahl ist dabei nur ein Zusatzhinweis, kein Nachweis: Fremde Prozesse im selben Baum ändern sie, ohne eine Notiz zu berühren.
4. **Die Suchleiter funktioniert im Alltag**: Katalog, Suche, genau eine Datei, Antwort mit Quelle.
5. **Der Vergleich mit und ohne System ist gelaufen** und protokolliert — auch wenn das Ergebnis ernüchtert.

Erst dann beginnt Scheibe 1, der Indexer.

---

## Was in dieser Scheibe ausdrücklich nicht passiert

- **Kein Zusammenführen, Verschieben oder Umsortieren bestehender Vaults.** Sie werden gelesen, sonst nichts.
- **Keine Aufnahme bestehender Bestände als Bereiche.** Sie sind Messgrundlage, nicht Teil des Systems.
- Kein Wiki. Es entsteht in Scheibe 3, und der Altbestand wird ohnehin nicht verdichtet (Entscheidung 13).
- Kein Daemon, keine MCP-Werkzeuge, kein Python. Claude ruft `qmd` über die Shell.
- Keine Konverter. Der Eingangsordner entsteht, wird aber noch nicht bedient.
- Keine Web-App, kein Graph.

# Die Schreibschranke lässt das Memory der Agenten immer offen

**Datum:** 2026-09-13
**Stand:** entworfen, nicht umgesetzt
**Betrifft:** `pkg/guard` (`Decide` und ein neues Modul für die Memory-Bäume),
`docs/hooks.md`

## Anlass

Am 2026-09-11 und erneut am 2026-09-13 verweigerte `brain guard` einer
Claude-Code-Sitzung das Schreiben eines Memory-Eintrags nach
`C:\Users\micro\.claude\projects\C--Users-micro-Documents--GIT-ultra-brain\memory\`:

```
… lies outside every writable tree; writing is allowed only below: …
```

Das ist die Regel, wie sie heute steht, und kein Rückschritt: `87f6e2a` und
`b256f3b` antworteten am 2026-09-11 auf denselben Aufruf gleich, mit Exit 2.
Schreibbar ist nur, was `registry.toml` als Bereich trägt — ein Wiki eines nicht
schreibgeschützten Bereichs oder der Pfad eines `workspace`-Bereichs
(`writableRoots`, `pkg/guard/guard.go:138-157`). Das Memory eines Agenten ist
beides nicht. Eine eigene Freigabeliste kennt die Schranke nicht; die einzige
Ausnahme außerhalb der Bäume ist `proposal.md` im Prüfzentrum
(`isProposal`, `guard.go:261-264`).

Die Folge: Kein Agent unter dieser Schranke kann sich etwas merken, in keinem
Projekt, für keinen Nutzer. Die Registrierung als Bereich wäre ein Ausweg pro
Nutzer und pro Projekt, und sie nähme das Memory als Wissen in den Index auf.

## Ziel

Das Memory von Claude Code und Antigravity ist für jeden Nutzer immer
schreibbar, ohne Konfiguration und ohne Registrierung — und nichts sonst, was
bisher gesperrt war, wird dadurch offen.

## Entscheidungen

Am 2026-09-13 mit dem Nutzer getroffen:

1. **Umfang: Claude Code und Antigravity.** Nicht ganz `~/.claude`: dort liegen
   `settings.json`, Hooks und Plugins, und ein Agent, der sie umschreibt, schaltet
   seine eigenen Schranken ab.
2. **Bei Antigravity `knowledge/` und `brain/<gesprächs-id>/`.** `knowledge/` ist
   das Gedächtnis über Gespräche hinweg; `brain/<id>/` trägt Pläne,
   Aufgabenlisten und `scratch/` einer Unterhaltung.
3. **Feste Ausnahme in `brain guard`** (Weg A), nicht die Registrierung als
   Bereich (B) und kein neuer Registry-Schlüssel (C). B muss jeder Nutzer pro
   Projekt pflegen und indexiert das Memory; C wäre eine zweite Freigabeliste,
   die niemand braucht.

## Messungen

Am 2026-09-13 auf diesem Rechner (Windows 11):

- **Claude Code** legt das Auto-Memory unter
  `~/.claude/projects/<kodierter-arbeitspfad>/memory/` ab, Einträge als
  `*.md` plus `MEMORY.md`. Der kodierte Pfad ist ein einziges Segment
  (`C--Users-micro-Documents--GIT-ultra-brain`). `CLAUDE_CONFIG_DIR` ist in dieser
  Sitzung nicht gesetzt; dass Claude Code damit das Konfigurationsverzeichnis
  und so auch `projects/` verlegt, ist nicht gemessen, sondern die dokumentierte
  Bedeutung der Variable.
- **Antigravity** trägt drei Wurzeln unter `~/.gemini`: `antigravity`,
  `antigravity-cli`, `antigravity-ide`, dazu `antigravity-backup`. In jeder der
  drei liegt `knowledge/` (heute nur `knowledge.lock`) und `brain/` mit einem
  Verzeichnis je Gesprächs-ID; eines davon enthält `.system_generated`,
  `.user_uploaded` und `scratch`. Ob Antigravity diese Dateien über seine
  Schreibwerkzeuge (`write_to_file`, `replace_file_content`,
  `multi_replace_file_content`, `guard.go:60-62`) anlegt oder am Hook vorbei,
  ist nicht gemessen. Die Ausnahme schadet im zweiten Fall nicht.
- **Die Antigravity-Migration** legt Claude-Memory als Regeln unter
  `<repo>/.agents/rules/` ab (`antigravity-for-claude-code` 0.24.0,
  `docs/MIGRATION.md`). Das sind Anweisungen an den Agenten, kein Gedächtnis.

## Entwurf

### Die Memory-Bäume

Ein neues Modul in `pkg/guard` beantwortet eine Frage: liegt dieser aufgelöste
Pfad in einem Memory-Baum? Es liest dafür das Home-Verzeichnis und
`CLAUDE_CONFIG_DIR` über austauschbare Funktionen, so wie
`config.defaultStateDir` Umgebung und Home als Argumente nimmt
(`pkg/config/registry.go:164`) — damit beide Zweige aus einem Test erreichbar
sind, ohne die echte Umgebung zu verändern.

| Host | Wurzel | Offen ist |
|---|---|---|
| Claude Code | `$CLAUDE_CONFIG_DIR`, sonst `<home>/.claude` | `<wurzel>/projects/<genau ein Segment>/memory`: alles unterhalb davon, nicht das Verzeichnis selbst |
| Antigravity | `<home>/.gemini/antigravity`, `…/antigravity-cli`, `…/antigravity-ide` | `<wurzel>/knowledge`: alles unterhalb davon, nicht das Verzeichnis selbst; `<wurzel>/brain/<genau ein Segment>`: alles unterhalb davon, nicht das Verzeichnis selbst |

Nicht offen, und die Tests halten das fest:

- alles andere unter `~/.claude`: `settings.json`, `CLAUDE.md`, `plugins/`,
  `projects/<x>/<transkript>.jsonl`, `projects/memory`, `projects/<x>/<y>/memory`;
- unter `~/.gemini`: `antigravity-backup`, `settings.json`, `config/`, das
  Verzeichnis `brain/` selbst und Dateien direkt darin.

Die Ausnahme öffnet außerdem `<repo>/.agents/rules/` nicht; ob der Pfad sonst
schreibbar ist, entscheidet wie bisher die Registry.

**Auflösen.** Die Wurzeln laufen durch `resolvePath` wie jeder Zielpfad, damit
der Vergleich auf beiden Seiten dieselbe Schreibweise hat (Groß- und
Kleinschreibung, Junctions). Eine Wurzel, die sich nicht auflösen lässt, entfällt
still, und ein nicht ermittelbares Home ebenso: die Ausnahme darf nur öffnen,
nie etwas schließen, also ist ihr Ausfall kein Grund für eine Ablehnung — dann
entscheidet die Schranke wie heute. Ein Home oder `CLAUDE_CONFIG_DIR`, das
nicht absolut ist, gilt als nicht gesetzt: ein relatives würde am
Arbeitsverzeichnis des Hooks verankert, also in irgendeinem Repo. Ein
ungesetztes `CLAUDE_CONFIG_DIR` fällt auf `<home>/.claude` zurück, ein relatives
also auch. Unter Windows trifft dieselbe Prüfung einen Pfad mit Wurzel, aber
ohne Laufwerk (`\Users\x\.claude`); der landete auf dem Laufwerk des
Arbeitsverzeichnisses und bleibt so geschlossen, nie geöffnet.

**Segmente zählen, nicht Präfixe.** Das Zielpfad-Segment nach `projects/` muss
genau eines sein, bevor `memory` folgt. Ein Präfixvergleich auf
`<wurzel>/projects/` ließe `projects/a/b/memory` durch; ein Vergleich auf
`…/memory` ohne Segmentgrenze ließe `memory-alt` durch.

### Die Stelle in `Decide`

Die Reihenfolge in `Decide` (`guard.go:378-488`) ist die Konstruktion. Die
Ausnahme kommt an eine feste Stelle:

1. Werkzeug und Ziele lesen, Ziele auflösen — unverändert. Symlinks und
   Junctions werden dabei verfolgt; ein Link im Memory-Verzeichnis, der ins Repo
   zeigt, landet aufgelöst im Repo und ist nicht ausgenommen.
2. **Manifest-Sperre — unverändert und weiter zuerst.** Eine `.brain.toml` oder
   `.ultra-brain/config.toml` ist auch im Memory-Verzeichnis kein Schreibziel.
3. **Neu: Liegen alle Ziele des Aufrufs in Memory-Bäumen, lässt die Schranke
   durch** — vor dem Lesen der Registry. So bleibt das Memory offen, wenn die
   Registry kaputt ist oder keinen schreibbaren Bereich nennt.
4. Registry, schreibbare Bäume, Sperrzonen, deklarierte Wikis — unverändert.
5. **Neu: Memory-Ziele zählen beim Vergleich mit den Bäumen als drinnen.** Ein
   Aufruf mit mehreren Zielen (`file_path` neben `notebook_path`), von denen nur
   ein Teil im Memory liegt, wird für den Rest wie heute entschieden.
6. `proposal.md` — unverändert.
7. **Die Ablehnung nennt die Memory-Wurzeln mit**, hinter den erlaubten Bäumen:
   `…, plus the agents' memory below: <wurzeln>`. Wer abgewiesen wird, erfährt
   so, wo er schreiben darf.

Bewusst in Kauf genommen: Weil Schritt 3 vor den Sperrzonen steht, wäre eine
schreibgeschützte Wiki-Zone, die jemand in ein Memory-Verzeichnis legt, dort
nicht gesperrt. Kein Bereich liegt heute dort, und ein Wiki im Memory eines
Agenten ergibt keinen Sinn.

### Was sich nicht ändert

- Die Registry, `writableRoots`, `forbiddenRoots`, `declaredWikiRoot` und
  `reviewCentre`.
- Die Hooks (`hooks/guard.sh`, `.agents/hooks.json`) und ihr Exit-Code-Vertrag.
- `ulguard` aus ultraloom hat eine eigene Richtlinie und ist nicht Teil davon.

## Tests

In `pkg/guard`, testgetrieben, im Stil von `decide_test.go`:

1. Claude-Code-Memory offen: `<home>/.claude/projects/<x>/memory/a.md` und
   `…/memory/MEMORY.md` — mit gültiger Registry, mit einer Registry ohne
   schreibbaren Bereich und mit einer unlesbaren Registry.
2. Antigravity offen: `<home>/.gemini/antigravity-cli/knowledge/k.md` und
   `<home>/.gemini/antigravity-ide/brain/<id>/task.md`.
3. Gesperrt bleiben: `<home>/.claude/settings.json`,
   `<home>/.claude/projects/<x>/t.jsonl`, `<home>/.claude/projects/memory/a.md`,
   `<home>/.claude/projects/<x>/<y>/memory/a.md`, `…/<x>/memory-alt/a.md`,
   `<home>/.gemini/antigravity-backup/knowledge/k.md`,
   `<home>/.gemini/antigravity/brain/t.md`.
4. Manifest-Sperre gewinnt: `<home>/.claude/projects/<x>/memory/.brain.toml`.
5. Junction aus dem Memory-Verzeichnis in einen gesperrten Baum wird verweigert.
   Nur unter Windows, wo `pkg/guard` Junctions schon misst; sonst übersprungen
   mit Grund.
6. `CLAUDE_CONFIG_DIR` gesetzt: `<config>/projects/<x>/memory/a.md` offen,
   `<home>/.claude/projects/<x>/memory/a.md` nicht mehr.
7. Home nicht ermittelbar: die Ausnahme entfällt, ein Memory-Pfad wird wie heute
   verweigert.
8. Gemischter Aufruf: Memory-Ziel neben einem Ziel außerhalb aller Bäume wird
   verweigert, und die Meldung nennt nur das zweite Ziel.
9. Die Ablehnung nennt die Memory-Wurzeln.

**Kein neuer Fall im Fallkorpus.** Der Läufer (`pkg/cases/runner.go:172-200`)
gibt jedem Fall die Umgebung des Rechners plus `BRAIN_STATE_DIR` und eine feste
stdin-Datei; ein Fall kann sein Home nicht setzen. Eine Nutzlast unter einem
Memory-Pfad müsste also das echte Home des Rechners wörtlich tragen, und die
Ablehnungsmeldung schriebe es in die Erwartungsdatei. Die Tests in `pkg/guard`
setzen das Home über die austauschbare Funktion und decken dasselbe ab.

## Doku

- `docs/hooks.md`: ein Absatz zur Schreibschranke, welche Memory-Bäume immer
  offen sind und warum `~/.claude` im Übrigen gesperrt bleibt.

## Außerhalb des Schnitts

- **Der Scratchpad von Claude Code** (`%TEMP%\claude\…\scratchpad`). Die
  Schranke verweigert ihn heute ebenfalls; der Nutzer hat nur nach dem Memory
  gefragt.
- **`<repo>/.agents/rules/`.** Regeln an den Agenten; ein Agent, der sie
  umschreibt, könnte die Schranke aushebeln.
- **Weitere Hosts** (Codex, Gemini CLI ohne Antigravity). `pkg/guard` kennt ihre
  Schreibwerkzeuge, ihr Memory ist nicht gemessen.
- **Eine Umgebungsvariable für `~/.gemini`.** Keine gemessen; die Wurzeln hängen
  am Home.

## Fertig, wenn

1. Die Tests oben laufen grün, der Junction-Test auf diesem Rechner.
2. Nach dem Merge und der Installation schreibt eine Claude-Code-Sitzung einen
   Memory-Eintrag nach `~/.claude/projects/<projekt>/memory/`, ohne dass
   `brain guard` ablehnt.
3. Ein Schreibversuch auf `~/.claude/settings.json` wird weiter verweigert, und
   die Meldung nennt die Memory-Wurzeln.

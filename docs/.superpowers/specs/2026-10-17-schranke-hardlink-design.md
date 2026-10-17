# Hardlinks an Wächter und Schreibschranke

## Problem

Ein Agent legt mit `ln ../B/.loomux/config.toml notes.txt` oder `mklink /H`
einen Hardlink auf eine geschützte Datei an und ändert sie danach mit `Edit`
oder `Write` über den neuen Namen. Beide Stufen urteilen über den Pfad:
`guard.ResolvePath` folgt Symlinks und Junctions, ein Hardlink ist aber kein
Link, sondern ein zweiter Name derselben Datei. Nachgewiesen auf `18566e83`.

## Entscheidung

Ein bestehendes reguläres Ziel mit mehr als einem Link (POSIX `st_nlink`,
Windows `nNumberOfLinks` aus `GetFileInformationByHandle`) wird in beiden
Stufen verweigert, gleich wo die anderen Namen liegen. Eine Regel, keine
Ausnahme.

Die feinere Regel „nur verweigern, wenn ein anderer Name geschützt ist oder
außerhalb der beschreibbaren Wurzeln liegt“ wurde verworfen:

1. Unter POSIX gibt es keinen Aufruf, der die anderen Namen nennt; man müsste
   das Dateisystem ablaufen. Das geht nicht in 72 ms.
2. Unter Windows nennt `FindFirstFileNameW` sie, aber jeder Name müsste durch
   das ganze Urteil beider Stufen, und die Policy eines fremden Projekts kennt
   der Wächter nicht. Ein Name in einem anderen Projekt wäre damit unbeurteilbar
   und müsste ohnehin verweigert werden.
3. Was ein Agent legitim mit mehreren Links bearbeitet, ist nichts: pnpm- und
   uv-Speicher verlinken in `node_modules` und `.venv`, und eine Änderung dort
   schriebe in jedes Projekt, das denselben Speicher teilt. Git legt im
   Arbeitsbaum keine Hardlinks an.

Ein neues Ziel hat keinen Link und geht durch, ebenso ein Verzeichnis (unter
POSIX hat jedes mehr als einen Link). Ein Ziel, dessen Linkzahl der Wächter
nicht lesen kann, obwohl es da ist, wird verweigert; „nicht da“ liest jede
Stufe so, wie ihr Resolver es liest (`stopsResolving`).

## Ort

- `guard.LinkCount` (Windows: `CreateFile` ohne Zugriffsrechte, wie
  `finalName`; sonst `os.Stat`).
- `Decide`: direkt nach der Manifestprüfung, vor den offenen Zielen, damit
  auch ein Hardlink im Memory oder im Scratchpad verweigert wird.
- Pre-Tool-Wächter: in `judge.reasons` für jedes Ziel, das nicht entfernt
  wird, also für schreibende Werkzeuge und für Schreibziele einer Shell-Zeile
  (`echo x >`, `cp`). Ein `rm` oder die Quelle eines `mv` entfernt nur den
  Namen und bleibt offen.
- `guard.ManifestUnlinked`: ein Manifest mit mehr als einem Link gilt als
  verknüpft, also bleibt der Schalter `llm_config` dafür zu. Sonst schriebe
  `loomux config set … --yes`, das die Datei an Ort und Stelle schreibt, in
  das Manifest des anderen Projekts.

## Grenzen

- Das Anlegen des Links (`ln`, `mklink /H`, `New-Item -ItemType HardLink`)
  bleibt unbeurteilt; verweigert wird das Schreiben danach.
- Zwischen Prüfung und Schreiben kann ein Link entstehen (TOCTOU), wie bei
  einer Junction.
- Ein Programm, das über einen Hardlink schreibt, ohne dass die Zeile den
  Namen als Schreibziel nennt, kommt durch wie jedes Skript.

## Erfolg

Ein vorher roter Test je Stufe, die Differenzbatterie unverändert, Coverage
100 %, Gate grün, Hook-Latenz gemessen und in `docs/*/benchmarks.md`.

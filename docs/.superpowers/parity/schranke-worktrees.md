# Paritätsliste Schreibschranke: verknüpfte Worktrees

**Quelle:** ultra-brain `loomux-1a-source` (`3cc72d2`), `_writable_roots`.
**Spec:** [2026-09-15-loomux-schranke-worktrees-design.md](../specs/2026-09-15-loomux-schranke-worktrees-design.md)
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor die Stufe als fertig gilt.

| Fall / Bereich | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|
| Write in einem verknüpften Worktree eines `workspace`-Bereichs ohne eigenen Registry-Eintrag | brain guard: verweigert (`lies outside every writable tree`) | erlaubt; Erkennung ohne `git`-Prozess über `.git` als reguläre Datei (kein Symlink), Rückverweis `gitdir`, `commondir` und die Lage des Verwaltungsverzeichnisses direkt unter `<common>/worktrees` | Ein Worktree ist dasselbe Repository; `git rev-parse` kostet 42 ms gegen 24,5 ms Hook (Spec). Restrisiko: ein außerhalb per ungeprüftem Werkzeug (etwa Bash) angelegte `.git`-Datei; ein gelöschtes Verwaltungsverzeichnis eines noch vorhandenen früheren Worktrees kann im Workspace neu angelegt werden und öffnet diesen Worktree wieder, keinen fremden Baum | freigegeben 2026-09-15 |

# Pilot-Binary außerhalb des Checkouts – Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die Claude-Hooks des loomux-Checkouts starten ein Binary unter
`${LOCALAPPDATA}/loomux/dev/<Ordnername>/`, damit ein Low-Label auf dem
Checkout sie nicht trifft. loomux nennt das Label, wenn es doch als Low
läuft.

**Architecture:** `swap.Install` kopiert den getauschten Build als neue
Datei in den Dev-Ordner und tauscht dort mit derselben Rotation.
`hostfile.Checkout` zeigt auf diesen Ort. `BinaryOf` erkennt die alte und
die neue Form. `verify.writable` fragt eine Seam nach der
Integritätsstufe.

**Tech Stack:** Go, `golang.org/x/sys/windows` (schon im Modul).

**Spec:** `docs/.superpowers/specs/2026-11-09-dev-binary-ausserhalb-design.md`

## Global Constraints

- Code, Kommentare und Meldungen auf Englisch, Doku auf Deutsch und Englisch (`docs/de`, `docs/en`).
- 100 % Coverage, TDD. Keine Arbeit beim Paket-Init: `TestStartDoesNoWorkInPackageInit` erlaubt höchstens 500 Allokationen.
- Hook-Zeile exakt: `"${LOCALAPPDATA}/loomux/dev/$(basename "${CLAUDE_PROJECT_DIR}")/loomux.exe"`.
- Dev-Ordner exakt: `filepath.Join(os.Getenv("LOCALAPPDATA"), "loomux", "dev", filepath.Base(absRoot))`.
- Der Build wird in den Dev-Ordner **kopiert** (neue Datei), nie mit `os.Rename` verschoben.
- Kein Commit mit `--no-verify`. Kein Model als Co-Autor.

## Review Focus

1. `LOCALAPPDATA` ist leer (Linux-CI): `swap-binary` darf nicht scheitern, sondern tauscht nur in `--dir` und meldet nichts. Test in Task 1.
2. Der Dev-Ordner existiert noch nicht (erster Lauf): `Install` legt ihn an. Test in Task 1.
3. Eine bestehende `settings.json` mit der alten Form `${CLAUDE_PROJECT_DIR}/bin/loomux.exe`: `BinaryOf` gibt `Checkout` zurück, `init` hebt die Zeilen auf die neue Form. Test in Task 2.
4. Das Binary am neuen Ort fehlt, etwa weil seit dem Wechsel noch kein Build lief: `init` im Checkout baut und tauscht an beide Orte. Test in Task 2 über die Seam `buildCheckout`.
5. Medium-Prozess: Die Meldung von `writable` bleibt Wort für Wort wie heute. Test in Task 3.
6. `Install`, gestartet von einem Low-Prozess (`bin\loomux.exe init` bei gesetztem Label), kann in `%LOCALAPPDATA%` nichts anlegen. Ruling: Der Tausch in `--dir` bleibt bestehen. Der Befehl endet mit `1` und nennt den Dev-Ordner. Test in Task 1 mit injiziertem Create-Fehler.
7. Die Settings zeigen auf den Dev-Ordner, das Binary dort fehlt, z. B. direkt nach einem Pull, bevor `pre-commit` oder `init` lief. Ein Hook mit Exit 127 blockiert in Claude Code nicht. Guard und Schreibschranke wären dann still aus. Gegenmittel: Das PR-`## Changelog` sagt „nach dem Pull einmal `loomux init` oder einen Commit“. Ob die Hook-Zeile zusätzlich auf `bin/loomux.exe` zurückfällt, entscheidet der Mensch.

---

### Task 1: `swap.Install` und `dev swap-binary --root`

**Files:**
- Modify: `internal/swap/swap.go`
- Modify: `internal/cli/dev.go:356-368` (`devSwapBinary`)
- Modify: `internal/cli/init.go:84-96` (`buildPilot`)
- Test: `internal/swap/swap_test.go`, `internal/cli/dev_test.go` (oder die Datei, in der `devSwapBinary` heute getestet wird)

**Interfaces:**
- Produces: `func DevDir(root string) (string, error)` gibt `"", nil` zurück, wenn `LOCALAPPDATA` leer ist.
- Produces: `func Install(src, dir string) error` legt `dir` an, kopiert `src` als neue Datei nach `dir/loomux.new.exe` (`os.Create` + `io.Copy`) und ruft `Swap(dir)` auf.
- Produces: `devSwapBinary` nimmt `--root` (Standard `.`). Nach `Swap(*dir)` holt er `DevDir(root)`. Ist der Ordner nicht leer, folgt `Install(filepath.Join(*dir, "loomux.exe"), devDir)`.
- Produces: `buildPilot(root)` macht nach `swap.Swap(bin)` dasselbe.

- [ ] **Step 1: Failing tests**
  - `TestDevDirIsTheCheckoutsFolderUnderLocalAppData`: `t.Setenv("LOCALAPPDATA", tmp)`; `DevDir("/x/y/compat-version-breadth")` ergibt `tmp/loomux/dev/compat-version-breadth`.
  - `TestDevDirIsEmptyWithoutLocalAppData`: `LOCALAPPDATA=""` ergibt `"", nil`.
  - `TestInstallCopiesAndKeepsTheSource`: Nach `Install(src, dir)` existiert `src` weiter mit gleichem Inhalt. `dir/loomux.exe` hat den Inhalt von `src`. `dir` wurde angelegt.
  - `TestInstallRotatesThePreviousBinary`: Liegt in `dir` schon ein `loomux.exe` mit Inhalt `old`, steht `old` danach in `dir/loomux.old.exe`.
  - `TestInstallReportsAMissingSource`: Fehlt `src`, kommt ein Fehler, der den Pfad nennt.
  - CLI: `dev swap-binary --dir <bin> --root <r>` mit gesetztem `LOCALAPPDATA` tauscht in `<bin>` **und** in `DevDir(r)`. Ohne `LOCALAPPDATA` nur in `<bin>`. Exit-Code jeweils `0`.
  - `TestInstallFailureLeavesTheFirstSwap`: Das Anlegen im Dev-Ordner scheitert über eine Seam `var create = os.Create`. Dann bleibt `<bin>/loomux.exe` der neue Build, der Exit-Code ist `1`, und `stderr` nennt den Dev-Ordner.
- [ ] **Step 2:** `go test ./internal/swap/ ./internal/cli/ -run 'DevDir|Install|SwapBinary'`. Erwartet: FAIL (undefined: DevDir, Install, flag `root`).
- [ ] **Step 3:** Implementieren wie unter Interfaces. Die Fehler von `Install` nennen den Pfad, in der Form der vorhandenen `Swap`-Meldungen.
- [ ] **Step 4:** Tests erneut. Erwartet: PASS. `go test ./internal/swap/ ./internal/cli/` grün.
- [ ] **Step 5:** `.githooks/pre-commit` bleibt unverändert. Es ruft `dev swap-binary --dir bin` aus der Wurzel, und `--root .` ist der Standard.
- [ ] **Step 6: Commit** `feat(swap): install the pilot binary outside the checkout too`

### Task 2: Hook-Zeile auf den Dev-Ordner

**Files:**
- Modify: `internal/setup/hostfile/table.go:24` (`Checkout`)
- Modify: `internal/setup/hostfile/merge.go:212` (`BinaryOf`: Suffix `/loomux.exe` statt `/bin/loomux.exe`)
- Modify: `internal/setup/facts.go:113-122` (`BinaryPath`)
- Modify: `internal/cli/init.go:430` (`notCheckout`-Text)
- Modify: `.claude/settings.json`: alle acht Zeilen mit `${CLAUDE_PROJECT_DIR}/bin/loomux.exe`, auch `worktree link` und `worktree unlink`
- Modify: `internal/setup/hostfile/testdata/loomux-settings.json`
- Test: `internal/setup/hostfile/merge_test.go`, `internal/setup/facts_test.go`

**Interfaces:**
- Consumes: Der Dev-Ordner aus Task 1, in derselben Form `LOCALAPPDATA/loomux/dev/<Base(root)>/loomux.exe`.
- Produces: `Checkout = "\"${LOCALAPPDATA}/loomux/dev/$(basename \"${CLAUDE_PROJECT_DIR}\")/loomux.exe\""`.

- [ ] **Step 1: Failing tests**
  - `TestBinaryOfKnowsTheOldCheckoutForm`: Eine Datei mit der alten Zeile `"${CLAUDE_PROJECT_DIR}/bin/loomux.exe" hook stop …` ergibt `Checkout`.
  - `TestBinaryOfKnowsTheNewCheckoutForm`: Die neue Zeile ergibt `Checkout`.
  - `TestMergeLiftsTheOldCheckoutLines`: `Merge` mit `Entries(claude, Checkout)` über die alte Datei ersetzt jede alte Zeile. Keine doppelte Zeile, keine Notiz.
  - `TestBinaryPathOfTheCheckoutIsTheDevFolder`: Mit `LOCALAPPDATA=tmp` und `root=/a/loomux` ist das Ergebnis `tmp/loomux/dev/loomux/loomux.exe`.
  - Der bestehende `TestTheRepositorysOwnSettingsNeedNoChange` wird rot, sobald `Checkout` geändert ist. Das zeigt, dass `.claude/settings.json` nachziehen muss.
- [ ] **Step 2:** `go test ./internal/setup/...`. Erwartet: die neuen Tests FAIL.
- [ ] **Step 3:** Konstante, Suffix-Prüfung und `BinaryPath` ändern. `.claude/settings.json` und `testdata/loomux-settings.json` auf die neue Form heben. `notCheckout` nennt den neuen Pfad.
- [ ] **Step 4:** `go test ./internal/setup/... ./internal/cli/`. Erwartet: PASS, auch `TestTheRepositorysOwnSettingsNeedNoChange`.
- [ ] **Step 4b: `staleBinary` kennt den Dev-Ordner.** `internal/hooks/hook_session_start.go:140ff` warnt heute nur, wenn das laufende Binary unter `root` liegt. Neu zählt auch `swap.DevDir(root)` als eigenes Binary des Projekts, und der Rebuild-Hinweis nennt `dev swap-binary`. Test `TestStaleBinaryNamesAnOldBinaryInTheDevFolder`: Ein Executable in `DevDir(root)`, älter als `internal/verify/plan.go`, wird als veraltet gemeldet. Andere Prüfungen von `os.Executable()` auf „im Checkout“ gibt es nicht. `serve/state.go` liest nur die Identität.
- [ ] **Step 5: Von Hand** `go build -o bin/loomux.new.exe ./cmd/loomux && go run ./cmd/loomux dev swap-binary --dir bin`. Danach existiert `%LOCALAPPDATA%\loomux\dev\dev-binary-outside-checkout\loomux.exe`. Die Hooks dieser Session stammen aus dem Hauptcheckout und bleiben davon unberührt.
- [ ] **Step 6: Commit** `fix(setup): run the checkout's hooks from a binary outside it`

### Task 3: Low-Integrität benennen

**Files:**
- Create: `internal/verify/integrity_windows.go`, `internal/verify/integrity_other.go`
- Modify: `internal/verify/gowork.go:21-52` (`goWork`, `writable`)
- Test: `internal/verify/gowork_test.go`, `internal/verify/integrity_windows_test.go`

**Interfaces:**
- Produces: `var lowIntegrity = isLowIntegrity`, eine Seam. `func isLowIntegrity() bool` liest unter Windows die Integritätsstufe des eigenen Tokens (`TokenIntegrityLevel`, Struktur `Tokenmandatorylabel`) und gibt für den RID der Label-SID `rid < 0x2000` zurück (`SECURITY_MANDATORY_MEDIUM_RID`). Den genauen Aufruf in der gepinnten Version von `golang.org/x/sys/windows` vorher nachsehen. Die Build-Tags und die Coverage macht die Funktion wie `internal/child/child_windows.go`. Unter anderen Systemen gibt sie `false` zurück. Sie läuft erst beim Aufruf, nie beim Init.
- Produces: Ist `lowIntegrity()` wahr, lautet die Meldung von `goWork` bei einem Schreibfehler exakt:
  `<KEY> <dir> is not writable: loomux runs at Low integrity, typically because the folder of its binary <exe> carries a Low label; run a copy from outside that folder (<err>)`.
  `<exe>` stammt aus `os.Executable()`.

- [ ] **Step 1: Failing tests**
  - `TestGoWorkNamesTheLowLabel`: `lowIntegrity` auf `true` gestubbt, nicht beschreibbares Verzeichnis. Die Note enthält `runs at Low integrity` und den Pfad des Test-Binarys.
  - `TestGoWorkKeepsTheSandboxHintAtMedium`: `lowIntegrity` auf `false` gestubbt. Die Note ist genau die heutige mit `grant it in the sandbox or set`.
  - `TestIsLowIntegrityIsFalseForThisProcess` (windows): Der Testprozess läuft Medium, das Ergebnis ist `false`.
- [ ] **Step 2:** `go test ./internal/verify/ -run 'GoWork|Integrity'`. Erwartet: FAIL.
- [ ] **Step 3:** Implementieren. Der Fehlerzweig von `GetTokenInformation` gibt `false` zurück. Sein `coverage:exempt` braucht eine Begründung: Ein Test kann dem eigenen Prozess das Token nicht entziehen.
- [ ] **Step 4:** `go test ./internal/verify/ ./cmd/loomux/`. Erwartet: PASS, auch `TestStartDoesNoWorkInPackageInit`.
- [ ] **Step 5: Commit** `feat(verify): name a Low integrity label as the reason a Go lane is unready`

### Task 4: Doku

**Files:**
- Modify: `docs/de/cli-reference.md`, `docs/en/cli-reference.md`: `dev swap-binary` (neues `--root`, zweiter Ort, Kopie statt Umbenennen)
- Modify: `docs/de/configuration.md`, `docs/en/configuration.md`: Absatz zu den Go-Lanes um das Low-Label ergänzen. Kernaussage: Die Hooks des Checkouts laufen aus `${LOCALAPPDATA}/loomux/dev/<Ordner>/`. Zwei Checkouts mit gleichem Ordnernamen teilen sich diesen Platz.

- [ ] **Step 1:** Texte schreiben, en und de gleichen Inhalts.
- [ ] **Step 2:** `go test ./...` bis auf die Docs-Tests (`docs_test.go`) sowie `loomux check lint`. Erwartet: grün.
- [ ] **Step 3: Commit** `docs: describe the pilot binary outside the checkout`

### Task 5: Abnahme von Hand

- [ ] Mit dem Menschen: Das Label auf dem Repo wird wieder gesetzt (`icacls <repo> /setintegritylevel (OI)(CI)L`). Den Befehl führt der Mensch aus. Danach, aus dem Worktree:
  - `"%LOCALAPPDATA%\loomux\dev\dev-binary-outside-checkout\loomux.exe" check lint` ergibt `lint/go: ok`.
  - `bin\loomux.exe check lint` ergibt `lint/go: unready` mit `runs at Low integrity` aus Task 3.

  Danach entfernt der Mensch das Label wieder. Ohne seine Zustimmung wird nichts davon ausgeführt.
- [ ] Das PR-`## Changelog` nennt den einmaligen Schritt nach dem Pull (Review Focus 7).

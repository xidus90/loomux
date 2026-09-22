package wiki

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/lock"
)

// The five texts are German, like the bundle they open: they are copied into
// a foreign repository and read there by the people and the models who work
// on it. They are `_SCHEMA`, `_INDEX`, `_LOG`, `_AUDIT` and `_IDENTITIES` of
// src/brain/wiki/scaffold.py, byte for byte.
const bundleSchema = `# Schema dieses Bundles

Vor jeder Wiki-Arbeit gelesen. Sechs Regeln (Architektur §9.1):

1. Nur die KI schreibt hier.
2. Rohquellen sind unantastbar.
3. Jede Seite ist vernetzt.
4. Widersprüche werden markiert, nie aufgelöst.
5. Jede Änderung endet mit einem Log-Eintrag.
6. Verdichtet wird immer gegen die Originalquelle, nie gegen eine ältere
   Zusammenfassung.

## Die vier Seitentypen

- ` + "`Source`" + ` — eine Quelle, verdichtet; trägt ` + "`sources[]`" + ` mit ` + "`doc_id`" + `,
  ` + "`content_hash`" + ` und ` + "`revision`" + `.
- ` + "`Topic`" + ` — ein Thema über mehrere Quellen hinweg.
- ` + "`Entity`" + ` — eine Person, ein Werkzeug, ein Ort, ein Begriff.
- ` + "`Synthesis`" + ` — eine eigene Ableitung aus mehreren Seiten.

## Die Form eines Konflikts

Fest und maschinell auffindbar (Architektur §9.3); ` + "`open_conflicts: <n>`" + ` in
der Frontmatter zählt die Kästen, und der Lint hält die Zahl dagegen.

` + "```markdown" + `
> [!conflict] <Kurztitel>
> [Quelle A](/pfad/a.md) sagt X.
> [Quelle B](/pfad/b.md) sagt Y.
> Beide Stände bleiben stehen. Entscheidung offen.
` + "```" + `

Aufgelöst wird die Quelle, nie der Kasten: wer nur die Markierung löscht,
findet denselben Widerspruch im nächsten Durchlauf wieder.
`

const bundleIndex = `# Katalog

> Jede neue Seite bekommt sofort eine Zeile in diesem Katalog.
`

const bundleLog = `# Protokoll

> Jede Änderung an einer Seite endet hier mit einer Zeile: Datum, Seite,
> was sich geändert hat und aus welcher Quelle.
`

const bundleAudit = `# Wartungsprotokoll

> Hier stehen die Wartungsvorgänge über dem Bundle. Gefüllt wird es ab
> Scheibe 5; bis dahin bleibt es leer.
`

// bundleRegister carries the four columns of an area's identity register, so
// the bundle's own register is read by the code that already reads one.
const bundleRegister = "doc_id\trelative\tcontent_hash\trevision\n"

// bundleFrame is the frame in the order the reference writes it.
func bundleFrame() [][2]string {
	return [][2]string{
		{"_schema.md", bundleSchema},
		{"index.md", bundleIndex},
		{"log.md", bundleLog},
		{"audit.md", bundleAudit},
		{"_identities.tsv", bundleRegister},
	}
}

// InitBundle creates what is missing of a bundle's frame at root and returns
// the files it wrote, in the order it wrote them. It is `init_bundle`
// (src/brain/wiki/scaffold.py).
//
// An existing file is never rewritten, not even back to the shipped text:
// `_schema.md` is meant to be edited, and a run that restored the default
// would undo that edit without a word. No directory for pages is made
// either; one appears when the first page is written into it.
//
// One step stricter than the reference, which asks `exists()`: only a regular
// file counts as present. A directory named `log.md` is no log, and passing
// over it would leave a bundle without one and say nothing -- so the write is
// attempted, fails, and the error names the file.
func InitBundle(root string) ([]string, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	var written []string
	for _, file := range bundleFrame() {
		target := filepath.Join(root, file[0])
		if info, err := os.Stat(target); err == nil && info.Mode().IsRegular() {
			continue
		}
		if err := lock.ReplaceText(target, file[1]); err != nil {
			return written, fmt.Errorf("%s: cannot be written: %w", target, err)
		}
		written = append(written, target)
	}
	return written, nil
}

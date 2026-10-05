package plancheck

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const specOK = `# Spec

| Stufe | Stand | Inhalt |
|---|---|---|
| **1a** | ✅ 2026-09-14 | pilot |
| **3** | ➗ in drei Teilstufen zerfallen | split |
| **9z** |

| Teilstufe | Stand | Inhalt |
|---|---|---|
| **4a** Schema und ` + "`config`" + ` | 🚧 gebaut; offen | ` + "`config list\\|get`" + ` |
| **4d** | offen | convert |
| **4f** Altverweise | vorgeschlagen | refs |

| Prio | Stufe | Hängt ab von | Warum hier |
|---|---|---|---|
| — | **1a** Pilot | — | fertig |
| 3 | **4** ` + "`init`" + `, Modell | 1a | offen |
| 5 |
`

func planDoc(lang string, rows []string, nodes []string) string {
	header := "| Stage | Status | Origin | What it delivered | Depends on | Priority |"
	if lang == "de" {
		header = "| Stufe | Stand | Herkunft | Was sie gebracht hat | Hängt ab von | Priorität |"
	}
	return "# Plan\n\n" + header + "\n|---|---|---|---|---|---|\n" + strings.Join(rows, "\n") +
		"\n\nText.\n\n```mermaid\nflowchart TD\n" + strings.Join(nodes, "\n") + "\n    s1a --> s4a\n```\n"
}

var (
	rowsOK = []string{
		"| **1a** | ✅ | alpha | pilot | — | — |",
		"| **4a** | 🚧 built; human steps open | new | `config list\\|get` | 1a ✅ | 3 |",
		"| **4d** | open | beta | convert | 4a | 3 |",
		"| **4f** | proposed (awaiting sign-off) | — | refs | 4d | — (proposed) |",
	}
	nodesOK = []string{
		`        s1a["1a pilot"]:::done`,
		`        s4a["4a init · P3"]:::partial`,
		`        s4d["4d convert · P3"]:::planned`,
		`        s4f["4f refs · proposed"]:::planned`,
	}
)

func replaced(list []string, i int, with string) []string {
	out := append([]string(nil), list...)
	out[i] = with
	return out
}

func TestCheck(t *testing.T) {
	en, de := planDoc("en", rowsOK, nodesOK), planDoc("de", rowsOK, nodesOK)
	cases := []struct {
		name         string
		en, de, spec string
		want         []string
	}{
		{"agree", en, de, specOK, nil},
		{"another table comes first", "| Capability | Status |\n|---|---|\n| **Guard** | ✅ |\n\n" + en, de, specOK, nil},
		{"no English table", "# Plan\n", de, specOK, []string{"en: no stage table"}},
		{"no German table", en, "# Plan\n", specOK, []string{"de: no stage table"}},
		{"row without a bold stage", planDoc("en", replaced(rowsOK, 0, "| 1a | ✅ | x | y | — | — |"), nodesOK), de, specOK,
			[]string{`en: row "1a" names no stage in bold`}},
		{"German lacks a row", en, planDoc("de", rowsOK[:3], nodesOK[:3]), specOK,
			[]string{"en lists 4 stages, de 3"}},
		{"languages disagree", en, planDoc("de", replaced(rowsOK, 2, "| **4d** | open | x | y | 4a | 2 |"), nodesOK), specOK,
			[]string{`row 3: en {4d open 3}, de {4d open 2}`, "de: 4d is drawn with P3, the table says 2"}},
		{"stage the spec lacks", planDoc("en", append(rowsOK, "| **7x** | open | x | y | — | 3 |"), append(nodesOK, `    s7x["7x new"]:::planned`)),
			planDoc("de", append(rowsOK, "| **7x** | open | x | y | — | 3 |"), append(nodesOK, `    s7x["7x new"]:::planned`)), specOK,
			[]string{"7x: the spec lists no such stage", "7x: open, but the spec gives it no priority"}},
		{"state differs", en, de, strings.Replace(specOK, "| **4d** | offen |", "| **4d** | ✅ 2026-09-27 |", 1),
			[]string{"4d: the plan says open, the spec done"}},
		{"group ranked with a dash", en, de, strings.Replace(specOK, "| 3 | **4**", "| — | **4**", 1),
			[]string{"4a: open, but the spec gives it no priority", "4d: open, but the spec gives it no priority"}},
		{"priority differs", en, de, strings.Replace(specOK, "| 3 | **4**", "| 2 | **4**", 1),
			[]string{`4a: priority "3", the spec says "2"`, `4d: priority "3", the spec says "2"`}},
		{"done stage ranked", planDoc("en", replaced(rowsOK, 0, "| **1a** | ✅ | x | y | — | 6 |"), nodesOK),
			planDoc("de", replaced(rowsOK, 0, "| **1a** | ✅ | x | y | — | 6 |"), nodesOK), specOK,
			[]string{`1a: priority "6", the spec says "—"`}},
		{"empty priority cell", planDoc("en", replaced(rowsOK, 3, "| **4f** | proposed | x | y | 4d | |"), nodesOK),
			planDoc("de", replaced(rowsOK, 3, "| **4f** | proposed | x | y | 4d | |"), nodesOK), specOK,
			[]string{`4f: priority "", the spec says "—"`}},
		{"unknown status", planDoc("en", replaced(rowsOK, 2, "| **4d** | parked | x | y | 4a | 3 |"), nodesOK),
			planDoc("de", replaced(rowsOK, 2, "| **4d** | parked | x | y | 4a | 3 |"), nodesOK), specOK,
			[]string{"4d: the plan says unknown, the spec open", "en: 4d is drawn planned, the table says unknown", "de: 4d is drawn planned, the table says unknown"}},
		{"node the table lacks", planDoc("en", rowsOK, append(nodesOK, `    s5x["5x ghost"]:::planned`)), de, specOK,
			[]string{"en: the diagram draws 5x, which the table lacks"}},
		{"node class differs", en, planDoc("de", rowsOK, replaced(nodesOK, 1, `        s4a["4a init · P3"]:::done`)), specOK,
			[]string{"de: 4a is drawn done, the table says built"}},
		{"node rank differs", planDoc("en", rowsOK, replaced(nodesOK, 2, `        s4d["4d convert · P4"]:::planned`)), de, specOK,
			[]string{"en: 4d is drawn with P4, the table says 3"}},
		{"stage drawn twice or not at all", planDoc("en", rowsOK, append(append([]string(nil), nodesOK[:3]...), `        s4d2["4d again"]:::planned`)), de, specOK,
			[]string{"en: 4d is drawn 2 times", "en: 4f is drawn 0 times"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Check(c.en, c.de, c.spec); !reflect.DeepEqual(got, c.want) {
				t.Errorf("Check =\n%q\nwant\n%q", got, c.want)
			}
		})
	}
}

func TestATableAtTheEndOfTheTextIsRead(t *testing.T) {
	got := tables("text\n| a | b\\|c |\n|---|---|\n| **1a** | ✅ |")
	want := [][][]string{{{"a", "b\\|c"}, {"**1a**", "✅"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tables = %q, want %q", got, want)
	}
}

// TestTheRepositoryPlanFollowsTheSpec holds docs/en/migration.md and
// docs/de/migration.md to each other and to the fusion spec.
func TestTheRepositoryPlanFollowsTheSpec(t *testing.T) {
	read := func(parts ...string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, parts...)...))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	findings := Check(
		read("docs", "en", "migration.md"),
		read("docs", "de", "migration.md"),
		read("docs", ".superpowers", "specs", "2026-09-14-loomux-fusion-design.md"),
	)
	for _, f := range findings {
		t.Error(f)
	}
}

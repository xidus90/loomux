// Package benchcompare sets two runs of the same cases side by side: what
// both measured with the factor between them, what only the later run has
// (new) and what only the earlier one had (dropped). Cases are paired by
// name, so the two case files give the same thing the same name.
package benchcompare

import (
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// Row is one case both runs measured.
type Row struct {
	Name          string
	Before, After benchreport.Timing
}

// Result sorts the cases of two runs by who measured them.
type Result struct {
	Compared []Row
	Added    []benchreport.Timing
	Dropped  []benchreport.Timing
}

func applies(t benchreport.Timing) bool { return t.Applicable == nil || *t.Applicable }

// index keys a run by case name and refuses a name that occurs twice: two
// lines of one name make the pairing a guess.
func index(run []benchreport.Timing) (map[string]benchreport.Timing, error) {
	byName := map[string]benchreport.Timing{}
	for _, t := range run {
		if _, dup := byName[t.Name]; dup {
			return nil, fmt.Errorf("case %q occurs twice in one run", t.Name)
		}
		byName[t.Name] = t
	}
	return byName, nil
}

// Compare pairs the cases of two runs by name. A case that does not apply
// on a side is left out of that side.
func Compare(before, after []benchreport.Timing) (Result, error) {
	if _, err := index(before); err != nil {
		return Result{}, err
	}
	later, err := index(after)
	if err != nil {
		return Result{}, err
	}
	var r Result
	paired := map[string]bool{}
	for _, b := range before {
		if !applies(b) {
			continue
		}
		a, ok := later[b.Name]
		if !ok || !applies(a) {
			r.Dropped = append(r.Dropped, b)
			continue
		}
		paired[b.Name] = true
		r.Compared = append(r.Compared, Row{Name: b.Name, Before: b, After: a})
	}
	for _, a := range after {
		if applies(a) && !paired[a.Name] {
			r.Added = append(r.Added, a)
		}
	}
	return r, nil
}

// Factor is how many times faster the later value is: 2 is twice as fast,
// 0.5 half as fast. A side with no time gives none (0), not a division.
func Factor(before, after float64) float64 {
	if before <= 0 || after <= 0 {
		return 0
	}
	return before / after
}

func formatFactor(f float64) string {
	if f == 0 {
		return "–"
	}
	return fmt.Sprintf("%.2f×", f)
}

type words struct {
	compared, added, dropped, name, before, after, cold, warmMean, median, exit, factor, none string
	// summary takes six counts, in this order: the cases compared, those
	// faster, slower and unclear among them, the new and the dropped ones.
	summary string
}

var languages = map[string]words{
	"de": {compared: "Verglichen", added: "Neu", dropped: "Weggefallen", name: "Fall", before: "alt", after: "neu",
		cold: "kalt", warmMean: "warm Ø", median: "Median", exit: "Exit", factor: "×", none: "keine",
		summary: "%d verglichen, %d schneller, %d langsamer, %d unklar, %d neu, %d weggefallen"},
	"en": {compared: "Compared", added: "New", dropped: "Dropped", name: "case", before: "before", after: "after",
		cold: "cold", warmMean: "warm mean", median: "median", exit: "exit", factor: "×", none: "none",
		summary: "%d compared, %d faster, %d slower, %d unclear, %d new, %d dropped"},
}

// Markdown is the comparison as a report: a heading, one summary line and
// the three lists. title names the project, or the anonymised example.
func Markdown(r Result, title, lang string) (string, error) {
	w, ok := languages[lang]
	if !ok {
		return "", fmt.Errorf("unknown language %q; known are de and en", lang)
	}
	faster, slower, unclear := 0, 0, 0
	for _, row := range r.Compared {
		switch f := Factor(benchreport.Mean(row.Before.WarmMS), benchreport.Mean(row.After.WarmMS)); {
		case f > 1:
			faster++
		case f > 0 && f < 1:
			slower++
		default:
			unclear++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, w.summary+"\n\n", len(r.Compared), faster, slower, unclear, len(r.Added), len(r.Dropped))
	fmt.Fprintf(&b, "## %s\n\n", w.compared)
	fmt.Fprintf(&b, "| %s | %s %s | %s %s | %s | %s %s | %s %s | %s | %s %s | %s %s | %s %s | %s %s |\n", w.name,
		w.cold, w.before, w.cold, w.after, w.factor, w.warmMean, w.before, w.warmMean, w.after, w.factor,
		w.median, w.before, w.median, w.after, w.exit, w.before, w.exit, w.after)
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|\n")
	for _, row := range r.Compared {
		mb, ma := benchreport.Mean(row.Before.WarmMS), benchreport.Mean(row.After.WarmMS)
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %v | %v |\n", row.Name,
			benchreport.FormatMS(row.Before.ColdMS), benchreport.FormatMS(row.After.ColdMS),
			formatFactor(Factor(row.Before.ColdMS, row.After.ColdMS)),
			benchreport.FormatMS(mb), benchreport.FormatMS(ma), formatFactor(Factor(mb, ma)),
			benchreport.FormatMS(row.Before.MedianMS), benchreport.FormatMS(row.After.MedianMS),
			row.Before.ExitCodes, row.After.ExitCodes)
	}
	for _, list := range []struct {
		title string
		rows  []benchreport.Timing
	}{{w.added, r.Added}, {w.dropped, r.Dropped}} {
		fmt.Fprintf(&b, "\n## %s\n\n", list.title)
		if len(list.rows) == 0 {
			fmt.Fprintf(&b, "%s\n", w.none)
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n|---|---:|---:|---:|\n", w.name, w.cold, w.warmMean, w.median)
		for _, t := range list.rows {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", t.Name, benchreport.FormatMS(t.ColdMS),
				benchreport.FormatMS(benchreport.Mean(t.WarmMS)), benchreport.FormatMS(t.MedianMS))
		}
	}
	return b.String(), nil
}

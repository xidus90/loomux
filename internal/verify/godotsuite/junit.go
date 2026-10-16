package godotsuite

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Shard is one engine process: what it was asked to run and what came back.
type Shard struct {
	Index  int
	Suites []string // res:// paths planned for this shard
	Dir    string   // <shard>
	Code   int      // engine exit code
	Output string   // engine stdout+stderr
}

type junitProblem struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

type junitSuites struct {
	Suites []struct {
		Name    string  `xml:"name,attr"`
		Package string  `xml:"package,attr"`
		Time    float64 `xml:"time,attr"`
		Cases   []struct {
			Name     string         `xml:"name,attr"`
			Failures []junitProblem `xml:"failure"`
			Errors   []junitProblem `xml:"error"`
		} `xml:"testcase"`
	} `xml:"testsuite"`
}

func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return ""
}

// text prefers the character data: the message attribute only names the
// location of the failure, the data carries what was wrong.
func (p junitProblem) text() string {
	if l := firstLine(p.Text); l != "" {
		return l
	}
	return firstLine(p.Message)
}

// newestReport is the results.xml of the report with the newest mtime, the
// higher report_<n> on a tie; "" when there is none.
func newestReport(dir string) string {
	paths, _ := filepath.Glob(filepath.Join(dir, "reports", "report_*", "results.xml"))
	type cand struct {
		path string
		mod  int64
		n    int
	}
	var cs []cand
	for _, p := range paths {
		var mod int64 // a report that vanished since the glob sorts oldest
		if fi, err := os.Stat(p); err == nil {
			mod = fi.ModTime().UnixNano()
		}
		n, _ := strconv.Atoi(strings.TrimPrefix(filepath.Base(filepath.Dir(p)), "report_"))
		cs = append(cs, cand{p, mod, n})
	}
	if len(cs) == 0 {
		return ""
	}
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].mod != cs[j].mod {
			return cs[i].mod > cs[j].mod
		}
		return cs[i].n > cs[j].n
	})
	return cs[0].path
}

// stem is a suite's file name without directory and extension, which is how
// gdUnit4 names a testsuite in its report.
func stem(name string) string {
	return strings.TrimSuffix(filepath.Base(filepath.ToSlash(name)), ".gd")
}

// Verdict judges one shard by its newest report and returns the findings
// (E7) and the seconds each suite took, keyed by res:// path.
func Verdict(s Shard) (findings []string, times map[string]float64) {
	report := newestReport(s.Dir)
	if report == "" {
		return []string{fmt.Sprintf("shard %d wrote no test report (exit %d); it ran %d suites:\n%s",
			s.Index, s.Code, len(s.Suites), s.Output)}, nil
	}
	data, err := os.ReadFile(report)
	var doc junitSuites
	if err == nil {
		err = xml.Unmarshal(bytes.TrimPrefix(data, []byte(bom)), &doc)
	}
	if err != nil {
		return []string{fmt.Sprintf("shard %d wrote an unreadable test report %s: %v", s.Index, report, err)}, nil
	}
	cases := 0
	for _, su := range doc.Suites {
		cases += len(su.Cases)
	}
	if cases == 0 {
		return []string{fmt.Sprintf("shard %d ran no test cases; none of its %d suites loaded", s.Index, len(s.Suites))}, nil
	}

	byKey, byStem := map[string]string{}, map[string]string{}
	for _, p := range s.Suites {
		byKey[planKey(p)] = p
		byStem[stem(p)] = p
	}
	times = map[string]float64{}
	ran := map[string]bool{}
	for _, su := range doc.Suites {
		path, planned := byKey[suiteKey(su.Package, su.Name)]
		if !planned {
			path, planned = byStem[stem(su.Name)]
		}
		if planned {
			ran[path] = true
			times[path] = su.Time
		} else {
			path = su.Name
		}
		for _, c := range su.Cases {
			for _, p := range append(append([]junitProblem{}, c.Failures...), c.Errors...) {
				findings = append(findings, fmt.Sprintf("%s %s: %s", path, c.Name, p.text()))
			}
		}
	}
	var missing []string
	for _, p := range s.Suites {
		if !ran[p] {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		findings = append(findings, fmt.Sprintf("shard %d did not run: %s", s.Index, strings.Join(missing, ", ")))
	}
	if len(findings) == 0 && s.Code != 0 {
		findings = append(findings, fmt.Sprintf("shard %d exited with %d while its report shows nothing wrong:\n%s",
			s.Index, s.Code, s.Output))
	}
	return findings, times
}

// planKey is a planned path without scheme and extension, the form a
// report's package and name give together.
func planKey(resPath string) string {
	return strings.TrimSuffix(strings.TrimPrefix(resPath, "res://"), ".gd")
}

// suiteKey joins the package (the suite's directory) and name of a report
// suite; a suite at the project root has none.
func suiteKey(pkg, name string) string {
	name = strings.TrimSuffix(name, ".gd")
	if pkg == "" || pkg == "." {
		return name
	}
	return pkg + "/" + name
}

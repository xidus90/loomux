package linecov

import (
	"encoding/xml"
	"fmt"
	"path"
	"strconv"
)

// jacocoFormat reads JaCoCo XML, which Kover writes too. A source file is
// named by package and file name only, so its record asks for a suffix search.
type jacocoFormat struct{}

func (jacocoFormat) Name() string { return "jacoco" }
func (jacocoFormat) Detect(h []byte) bool {
	root, _ := xmlRoot(h)
	return root.Name.Local == "report"
}

type jacocoDoc struct {
	Packages []struct {
		Name  string `xml:"name,attr"`
		Files []struct {
			Name  string `xml:"name,attr"`
			Lines []struct {
				Nr string `xml:"nr,attr"`
				CI string `xml:"ci,attr"`
			} `xml:"line"`
		} `xml:"sourcefile"`
	} `xml:"package"`
}

// Read counts a line as hit when one of its instructions ran (ci > 0).
func (jacocoFormat) Read(content []byte, _ ReadContext) ([]Record, []string) {
	var doc jacocoDoc
	if err := xml.Unmarshal([]byte(text(content)), &doc); err != nil {
		return nil, []string{"unreadable XML: " + err.Error()}
	}
	var recs []Record
	var problems []string
	for _, p := range doc.Packages {
		for _, f := range p.Files {
			rec := Record{Path: path.Join(p.Name, f.Name), Suffix: true, Lines: map[int]uint64{}}
			for _, l := range f.Lines {
				n, err1 := strconv.Atoi(l.Nr)
				ci, err2 := strconv.ParseUint(l.CI, 10, 64)
				if err1 != nil || err2 != nil || n < 1 {
					problems = append(problems, fmt.Sprintf("%s: unreadable line nr=%q ci=%q", rec.Path, l.Nr, l.CI))
					continue
				}
				rec.Lines[n] = addSat(rec.Lines[n], ci)
			}
			recs = append(recs, rec)
		}
	}
	return recs, problems
}

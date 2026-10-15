package linecov

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// xmlRoot returns the first element of an XML document and the decoder
// standing behind it, or a zero element for anything else. DOCTYPE, comments
// and the declaration are passed over; nothing is fetched.
func xmlRoot(h []byte) (xml.StartElement, *xml.Decoder) {
	if len(h) == 0 || h[0] != '<' {
		return xml.StartElement{}, nil
	}
	d := xml.NewDecoder(bytes.NewReader(h))
	for {
		tok, err := d.Token()
		if err != nil {
			return xml.StartElement{}, nil
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se, d
		}
	}
}

type coberturaFormat struct{}

func (coberturaFormat) Name() string { return "cobertura" }

// Detect asks for Cobertura's line-rate or one of its own children, because
// Clover's root element is coverage as well.
func (coberturaFormat) Detect(h []byte) bool {
	root, d := xmlRoot(h)
	if root.Name.Local != "coverage" {
		return false
	}
	if slices.ContainsFunc(root.Attr, func(a xml.Attr) bool { return a.Name.Local == "line-rate" }) {
		return true
	}
	for {
		tok, err := d.Token()
		if err != nil {
			return false
		}
		switch x := tok.(type) {
		case xml.StartElement:
			if x.Name.Local == "packages" || x.Name.Local == "sources" {
				return true
			}
			// A failed skip fails the next Token as well.
			_ = d.Skip()
		case xml.EndElement:
			return false
		}
	}
}

type coberturaDoc struct {
	Sources []string `xml:"sources>source"`
	Classes []struct {
		Filename string `xml:"filename,attr"`
		Lines    []struct {
			Number string `xml:"number,attr"`
			Hits   string `xml:"hits,attr"`
		} `xml:"lines>line"`
	} `xml:"packages>package>classes>class"`
}

// Read keeps one record per class; a file split over classes is summed when
// the records are merged.
func (coberturaFormat) Read(content []byte, _ ReadContext) ([]Record, []string) {
	var doc coberturaDoc
	if err := xml.Unmarshal([]byte(text(content)), &doc); err != nil {
		return nil, []string{"unreadable XML: " + err.Error()}
	}
	var bases []string
	for _, s := range doc.Sources {
		bases = append(bases, strings.TrimSpace(s))
	}
	var recs []Record
	var problems []string
	for _, c := range doc.Classes {
		rec := Record{Path: c.Filename, Bases: bases, Lines: map[int]uint64{}}
		for _, l := range c.Lines {
			n, err1 := strconv.Atoi(l.Number)
			hits, err2 := strconv.ParseUint(l.Hits, 10, 64)
			if err1 != nil || err2 != nil || n < 1 {
				problems = append(problems, fmt.Sprintf("%s: unreadable line number=%q hits=%q", c.Filename, l.Number, l.Hits))
				continue
			}
			rec.Lines[n] = addSat(rec.Lines[n], hits)
		}
		recs = append(recs, rec)
	}
	return recs, problems
}

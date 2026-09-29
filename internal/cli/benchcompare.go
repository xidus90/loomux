package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/dev/benchcases"
	"github.com/xidus90/loomux/internal/dev/benchcompare"
	"github.com/xidus90/loomux/internal/dev/benchhooks"
	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// benchWriteNew is the seam of the file writes of `dev bench cases`: a test
// makes a write fail halfway through a run.
var benchWriteNew = writeNewFile

// benchAbs is the seam of resolving --out of `dev bench cases`: a test takes
// the working directory away.
var benchAbs = filepath.Abs

// writeAndCloseFile is the seam of the write inside writeNewFile: a test makes
// it break off halfway.
var writeAndCloseFile = writeAndClose

// writeAndClose writes data to w and closes it whether or not the write
// worked, and reports both: a close that fails may be the only sign that the
// data did not reach the disk.
func writeAndClose(w io.WriteCloser, data []byte) error {
	_, werr := w.Write(data)
	return errors.Join(werr, w.Close())
}

// writeNewFile writes a file that must not exist yet, whole or not at all: a
// file cut short under its final name would pass for the whole one and keep
// the next run from writing it. Only a file this call created is taken back;
// one that stood in the way stays.
func writeNewFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if err := writeAndCloseFile(f, data); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

// benchJSONString is s as it stands inside a JSON string, without the quotes:
// a path with a backslash put into a JSON text this way still decodes to the
// path.
func benchJSONString(s string) string {
	quoted, _ := json.Marshal(s) // a string always encodes
	return string(quoted[1 : len(quoted)-1])
}

// benchLoadReport reads the timings of one report file and refuses a report
// of another schema: its fields may mean something else.
func benchLoadReport(path string) ([]benchreport.Timing, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var report benchreport.Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if report.Schema != benchreport.Schema {
		return nil, fmt.Errorf("%s: schema %d, this loomux reads schema %d", path, report.Schema, benchreport.Schema)
	}
	return report.Timings, nil
}

// benchCompareRun reads both reports and pairs their cases.
func benchCompareRun(beforePath, afterPath string) (benchcompare.Result, error) {
	before, err := benchLoadReport(beforePath)
	if err != nil {
		return benchcompare.Result{}, err
	}
	after, err := benchLoadReport(afterPath)
	if err != nil {
		return benchcompare.Result{}, err
	}
	return benchcompare.Compare(before, after)
}

// devBenchCompare sets two `dev bench hooks` reports side by side. Like the
// measuring commands it judges the report files before it does the work.
func devBenchCompare(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev bench compare", flag.ContinueOnError)
	fs.SetOutput(stderr)
	before := fs.String("before", "", "JSON report of the earlier run")
	after := fs.String("after", "", "JSON report of the later run")
	title := fs.String("title", "loomux dev bench compare", "heading of the comparison")
	lang := fs.String("lang", "de", "language of the comparison: de or en")
	out := fs.String("out", "", "directory to write the markdown and JSON report to")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "loomux dev bench compare: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	if *before == "" || *after == "" {
		fmt.Fprintln(stderr, "loomux dev bench compare: --before and --after are required")
		return 2
	}
	// The language is a matter of the call, asked before a file is read.
	if _, err := benchcompare.Markdown(benchcompare.Result{}, "", *lang); err != nil {
		fmt.Fprintf(stderr, "loomux dev bench compare: %v\n", err)
		return 2
	}
	stamp := benchreport.Stamp(benchClock())
	md, js, err := benchTargets(*out, "compare", stamp)
	var result benchcompare.Result
	if err == nil {
		result, err = benchCompareRun(*before, *after)
	}
	if err == nil {
		// The language was checked above, so the comparison always renders.
		text, _ := benchcompare.Markdown(result, *title, *lang)
		fmt.Fprint(stdout, text)
		if md != "" {
			// The times came out of JSON, so they are finite and the result always encodes.
			payload, _ := json.MarshalIndent(result, "", "  ")
			err = benchWriteBoth(md, js, []byte(text), append(payload, '\n'))
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev bench compare: %v\n", err)
		return 1
	}
	return 0
}

// benchFile is a file a run of `dev bench cases` is going to write.
type benchFile struct {
	path string
	data []byte
}

// benchReadExtras reads the extra cases of a project that has no settings
// file to derive them from (or more than it has). {{ROOT}} and {{OUT}} are
// replaced in the text of the file, before it is parsed, by the values as
// they stand inside a JSON string.
func benchReadExtras(path, root, dir string) ([]benchhooks.Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.NewReplacer("{{ROOT}}", benchJSONString(root), "{{OUT}}", benchJSONString(dir)).Replace(string(data))
	var cases []benchhooks.Case
	if err := json.Unmarshal([]byte(text), &cases); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cases, nil
}

// benchCaseFiles builds the case file of a project and the payload files its
// cases read: the cases of its settings first, then the extra ones.
func benchCaseFiles(settings, extras, root, file, dir string) ([]benchFile, error) {
	cases := []benchhooks.Case{}
	var payloads []benchcases.Payload
	if settings != "" {
		data, err := os.ReadFile(settings)
		if err != nil {
			return nil, err
		}
		if cases, payloads, err = benchcases.Build(data, root, file, dir, os.LookupEnv); err != nil {
			return nil, err
		}
	}
	if extras != "" {
		more, err := benchReadExtras(extras, root, dir)
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, c := range cases {
			seen[c.Name] = true
		}
		for _, c := range more {
			if seen[c.Name] {
				return nil, fmt.Errorf("case %q is defined twice, by the settings or the extras", c.Name)
			}
			seen[c.Name] = true
			cases = append(cases, c)
		}
	}
	text, _ := json.MarshalIndent(cases, "", "  ") // strings only, so it always encodes
	files := []benchFile{{filepath.Join(dir, "cases.json"), append(text, '\n')}}
	for _, p := range payloads {
		files = append(files, benchFile{filepath.Join(dir, p.Name), p.Data})
	}
	return files, nil
}

// benchWriteAll writes the files or none of them: it refuses before the
// first write when the directory is missing or a file is there, and takes
// back what it wrote when a later write fails.
func benchWriteAll(dir string, files []benchFile) error {
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Errorf("no directory at %s; name one with --out", dir)
	}
	for _, f := range files {
		if _, err := os.Stat(f.path); err == nil {
			return fmt.Errorf("%s already exists", f.path)
		}
	}
	var written []string
	for _, f := range files {
		if err := benchWriteNew(f.path, f.data); err != nil {
			for _, path := range written {
				_ = os.Remove(path)
			}
			return err
		}
		written = append(written, f.path)
	}
	return nil
}

// devBenchCases writes the case file of a project's hooks, so a measurement
// starts from what the project runs and not from a list written by hand.
func devBenchCases(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev bench cases", flag.ContinueOnError)
	fs.SetOutput(stderr)
	settings := fs.String("settings", "", "Claude settings.json holding the hooks")
	root := fs.String("root", "", "project directory the hooks run in")
	file := fs.String("file", "", "markdown file the sample edit touches")
	out := fs.String("out", "", "existing directory to write cases.json and the payloads to")
	extras := fs.String("extras", "", "JSON file with more cases; {{ROOT}} and {{OUT}} are replaced")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "loomux dev bench cases: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	if *root == "" || *file == "" || *out == "" {
		fmt.Fprintln(stderr, "loomux dev bench cases: --root, --file and --out are required")
		return 2
	}
	if *settings == "" && *extras == "" {
		fmt.Fprintln(stderr, "loomux dev bench cases: --settings or --extras is required")
		return 2
	}
	// The cases name their payloads by this path, and a trailing slash of
	// --out would double the one the payload name is joined with. The
	// measuring run opens them from its own working directory, so a relative
	// --out is resolved here; Abs fails only without a working directory, and
	// then the path stays as given.
	dir := filepath.ToSlash(filepath.Clean(*out))
	if abs, err := benchAbs(*out); err == nil {
		dir = filepath.ToSlash(abs)
	}
	files, err := benchCaseFiles(*settings, *extras, *root, *file, dir)
	if err == nil {
		err = benchWriteAll(dir, files)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev bench cases: %v\n", err)
		return 1
	}
	return 0
}

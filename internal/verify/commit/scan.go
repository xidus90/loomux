package commit

import (
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Finding records one commit message line that reached the refusal threshold.
type Finding struct {
	LineNumber int
	Line       string
	Hits       []string
}

var (
	scissorsRE   = regexp.MustCompile(`^#\s*-+\s*>8\s*-+`)
	trailerRE    = regexp.MustCompile(`^(?:[A-Z][A-Za-z]*(?:-[A-Za-z]+)+|Fixes|Closes|Refs|Ref|Cc|Link|Bug|BREAKING CHANGE):\s`)
	codeSpanRE   = regexp.MustCompile("`[^`]*`")
	openSpanRE   = regexp.MustCompile("`.*$")
	quotedSpanRE = regexp.MustCompile(`"[^"]*"`)
	// Python's \b, \s and \w are Unicode, Go's are ASCII. The boundary is the
	// captured character in front of the particle, put back on replacement.
	nameParticleRE = regexp.MustCompile(`(^|[^\p{L}\p{N}_])(?:von|van|de|du|della|di)[\s\v\x{1c}-\x{1f}\x{85}\p{Z}]+[A-Z][\p{L}\p{N}_]+`)
	// Python's [^\W\d_]: letters plus the numbers that are not decimal digits.
	wordRE   = regexp.MustCompile(`[\p{L}\p{Nl}\p{No}]+`)
	umlautRE = regexp.MustCompile(`[äöüÄÖÜßẞ]`)
)

// Scan checks text line by line for words of the other language and foreign script runs.
func Scan(text string, lang string, threshold int, allow []*regexp.Regexp) []Finding {
	stopwords := Stopwords(lang)
	var findings []Finding
	inCode := false

	lines := splitLines(text)
	for idx, line := range lines {
		number := idx + 1
		if scissorsRE.MatchString(line) {
			break
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.TrimSpace(line) == "" {
			inCode = false
			continue
		}
		scored, nextInCode := spans(line, inCode)
		inCode = nextInCode
		if number == 1 {
			inCode = false
		}
		allowed := false
		for _, re := range allow {
			if re != nil && re.MatchString(line) {
				allowed = true
				break
			}
		}
		if allowed {
			continue
		}
		hits := lineHits(scored, stopwords, lang, number == 1)
		if len(hits) >= threshold {
			findings = append(findings, Finding{
				LineNumber: number,
				Line:       line,
				Hits:       hits,
			})
		}
	}
	return findings
}

// splitLines splits like Python's str.splitlines: at every boundary it knows,
// \r\n as one, and without an empty line after a final boundary.
func splitLines(text string) []string {
	var lines []string
	start := 0
	for i, r := range text {
		switch r {
		case '\r':
			if strings.HasPrefix(text[i+1:], "\n") {
				continue
			}
		case '\n', '\v', '\f', '\x1c', '\x1d', '\x1e', 0x85, 0x2028, 0x2029:
		default:
			continue
		}
		end := i
		if r == '\n' && i > 0 && text[i-1] == '\r' {
			end--
		}
		lines = append(lines, text[start:end])
		start = i + utf8.RuneLen(r)
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}

func spans(line string, inCode bool) (string, bool) {
	text := line
	if inCode {
		idx := strings.IndexByte(text, '`')
		if idx == -1 {
			return " ", true
		}
		text = " " + text[idx+1:]
	}
	text = codeSpanRE.ReplaceAllString(text, " ")
	inCode = strings.ContainsRune(text, '`')
	if inCode {
		text = openSpanRE.ReplaceAllString(text, " ")
	}
	return quotedSpanRE.ReplaceAllString(text, " "), inCode
}

func lineHits(line string, stopwords map[string]struct{}, lang string, isSubject bool) []string {
	if !isSubject && trailerRE.MatchString(line) {
		return nil
	}
	stripped := nameParticleRE.ReplaceAllString(line, "${1} ")
	stripped = stripPathTokens(stripped)

	var words []string
	matches := wordRE.FindAllStringIndex(stripped, -1)
	for _, loc := range matches {
		orig := stripped[loc[0]:loc[1]]
		if isJoined(stripped, loc[0], loc[1]) {
			continue
		}
		folded := foldGerman(orig)
		if _, ok := stopwords[folded]; ok {
			words = append(words, folded)
		} else if lang == "en" && umlautRE.MatchString(orig) {
			words = append(words, orig)
		}
	}

	runs := scriptRuns(stripped)
	return append(words, runs...)
}

func isJoined(line string, start, end int) bool {
	if start > 0 {
		b := line[start-1]
		if b == '-' || b == '_' {
			return true
		}
	}
	if end < len(line) {
		b := line[end]
		if b == '-' || b == '_' {
			return true
		}
	}
	return false
}

func foldGerman(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 'ä', 'Ä':
			b.WriteString("ae")
		case 'ö', 'Ö':
			b.WriteString("oe")
		case 'ü', 'Ü':
			b.WriteString("ue")
		case 'ß', 'ẞ':
			b.WriteString("ss")
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func stripPathTokens(line string) string {
	var b strings.Builder
	b.Grow(len(line))
	i := 0
	for i < len(line) {
		r, size := utf8.DecodeRuneInString(line[i:])
		if unicode.IsSpace(r) {
			b.WriteRune(r)
			i += size
			continue
		}
		start := i
		for i < len(line) {
			r, size = utf8.DecodeRuneInString(line[i:])
			if unicode.IsSpace(r) {
				break
			}
			i += size
		}
		token := line[start:i]
		if isPathToken(token) {
			b.WriteByte(' ')
		} else {
			b.WriteString(token)
		}
	}
	return b.String()
}

func isPathToken(token string) bool {
	if strings.ContainsAny(token, `/\`) {
		return true
	}
	dot := strings.LastIndexByte(token, '.')
	if dot != -1 {
		suffix := token[dot+1:]
		if len(suffix) >= 1 && len(suffix) <= 5 && isAsciiAlphanumeric(suffix) {
			return true
		}
	}
	return false
}

func isAsciiAlphanumeric(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

var scriptNames []string
var scriptNamesOnce sync.Once

func initScriptNames() {
	for name := range unicode.Scripts {
		if name == "Latin" || name == "Common" || name == "Inherited" || name == "Han" || name == "Hiragana" || name == "Katakana" {
			continue
		}
		scriptNames = append(scriptNames, name)
	}
	sort.Strings(scriptNames)
}

func scriptOf(r rune) string {
	scriptNamesOnce.Do(initScriptNames)
	if r == '\u30FC' {
		return "CJK"
	}
	if !unicode.Is(unicode.L, r) {
		return ""
	}
	if unicode.Is(unicode.Latin, r) {
		return ""
	}
	if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
		return "CJK"
	}
	for _, name := range scriptNames {
		if unicode.Is(unicode.Scripts[name], r) {
			return name
		}
	}
	return ""
}

func scriptRuns(line string) []string {
	var runs []string
	var current []rune
	var currentScript string

	for _, r := range norm.NFKD.String(line) {
		if len(current) > 0 && unicode.Is(unicode.M, r) {
			current = append(current, r)
			continue
		}
		s := scriptOf(r)
		if s != "" && s == currentScript {
			current = append(current, r)
			continue
		}
		if len(current) > 0 {
			runs = append(runs, formatRun(current))
		}
		if s != "" {
			current = []rune{r}
		} else {
			current = nil
		}
		currentScript = s
	}
	if len(current) > 0 {
		runs = append(runs, formatRun(current))
	}
	return runs
}

func formatRun(run []rune) string {
	nfc := norm.NFC.String(string(run))
	runes := []rune(nfc)
	if len(runes) > 12 {
		runes = runes[:12]
	}
	return string(runes)
}

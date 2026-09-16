package pytext

import (
	"fmt"
	"strings"
	"time"
)

// IsoFormat is `datetime.isoformat()` of an aware value:
// `YYYY-MM-DDTHH:MM:SS`, `.ffffff` only when the microseconds are not zero,
// and the offset as `±HH:MM`, with `:SS` only when it has seconds. Go keeps
// nanoseconds and Python does not; the digits below the microsecond are
// truncated here. Python truncates them only in `fromisoformat`; its own
// conversions, `fromtimestamp` and `now`, round to the nearest microsecond
// and an exact tie to the even one (measured). No stamp reaches this
// difference: ParseAwareIsoFormat never yields sub-microsecond digits.
func IsoFormat(t time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%04d-%02d-%02dT%02d:%02d:%02d",
		t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second())
	if micro := t.Nanosecond() / 1000; micro != 0 {
		fmt.Fprintf(&b, ".%06d", micro)
	}
	_, offset := t.Zone()
	sign := '+'
	if offset < 0 {
		sign, offset = '-', -offset
	}
	fmt.Fprintf(&b, "%c%02d:%02d", sign, offset/3600, offset/60%60)
	if seconds := offset % 60; seconds != 0 {
		fmt.Fprintf(&b, ":%02d", seconds)
	}
	return b.String()
}

// ParseAwareIsoFormat is `datetime.fromisoformat(s)` for the one form this
// stage reads, the reconcile stamp, and answers false where Python raises
// or answers a naive value. The form accepted is
//
//	YYYY-MM-DD (T or space) HH:MM:SS [(. or ,) digits] (Z | ±HH:MM[:SS])
//
// with a valid calendar date, hour <= 23, minute and second <= 59, and
// digits past the sixth cut as Python cuts them. `_write_last_run` writes
// `now.isoformat() + "\n"` and `read_last_run` strips before parsing; the
// caller hands in `Strip(text)`, and no other form fits the writer.
//
// Python accepts more, each measured and each a line of the parity list:
// any other separator character, shortened and basic times and offsets,
// week dates, 24:00, a fraction in the offset, an offset minute or second
// of 60, and a blank before the offset. A stamp in such a form reads as
// never reconciled here.
func ParseAwareIsoFormat(s string) (time.Time, bool) {
	if len(s) < 20 || !shape(s[:19], "dddd-dd-dd?dd:dd:dd") || (s[10] != 'T' && s[10] != ' ') {
		return time.Time{}, false
	}
	year, month, day := number(s[0:4]), number(s[5:7]), number(s[8:10])
	hour, minute, second := number(s[11:13]), number(s[14:16]), number(s[17:19])
	rest := s[19:]
	micro := 0
	if rest[0] == '.' || rest[0] == ',' {
		end := 1
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		if end == 1 {
			return time.Time{}, false
		}
		fraction := (rest[1:end] + "00000")[:6]
		micro, rest = number(fraction), rest[end:]
	}
	offset := 0
	switch {
	case rest == "Z":
	case (shape(rest, "?dd:dd") || shape(rest, "?dd:dd:dd")) && (rest[0] == '+' || rest[0] == '-'):
		hours, minutes, seconds := number(rest[1:3]), number(rest[4:6]), 0
		if len(rest) == 9 {
			seconds = number(rest[7:9])
		}
		if hours > 23 || minutes > 59 || seconds > 59 {
			return time.Time{}, false
		}
		offset = hours*3600 + minutes*60 + seconds
		if rest[0] == '-' {
			offset = -offset
		}
	default:
		return time.Time{}, false
	}
	lastDay := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if year < 1 || month < 1 || month > 12 || day < 1 || day > lastDay || hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(month), day, hour, minute, second, micro*1000, time.FixedZone("", offset)), true
}

// shape says whether s has the length of pattern and, position by position,
// an ASCII digit under `d`, anything under `?` and the same byte elsewhere.
func shape(s, pattern string) bool {
	if len(s) != len(pattern) {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch pattern[i] {
		case 'd':
			if s[i] < '0' || s[i] > '9' {
				return false
			}
		case '?':
		default:
			if s[i] != pattern[i] {
				return false
			}
		}
	}
	return true
}

// number is the value of a run of ASCII digits the caller has checked.
func number(digits string) int {
	n := 0
	for i := 0; i < len(digits); i++ {
		n = n*10 + int(digits[i]-'0')
	}
	return n
}

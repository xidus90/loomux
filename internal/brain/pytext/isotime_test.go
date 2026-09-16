package pytext

import (
	"testing"
	"time"
)

func TestIsoFormatLikePython(t *testing.T) {
	// d.isoformat() of the same values on Python 3.14.7.
	zone := func(seconds int) *time.Location { return time.FixedZone("", seconds) }
	cases := []struct {
		in   time.Time
		want string
	}{
		{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), "2000-01-01T00:00:00+00:00"},
		{time.Date(2000, 1, 1, 12, 30, 45, 5000, time.UTC), "2000-01-01T12:30:45.000005+00:00"},
		{time.Date(2000, 1, 1, 12, 30, 45, 123000000, time.UTC), "2000-01-01T12:30:45.123000+00:00"},
		{time.Date(2026, 9, 15, 8, 0, 0, 0, zone(7200)), "2026-09-15T08:00:00+02:00"},
		{time.Date(2026, 9, 15, 8, 0, 0, 0, zone(-19800)), "2026-09-15T08:00:00-05:30"},
		{time.Date(2026, 9, 15, 8, 0, 0, 0, zone(30)), "2026-09-15T08:00:00+00:00:30"},
		{time.Date(2026, 9, 15, 8, 0, 0, 0, zone(-3630)), "2026-09-15T08:00:00-01:00:30"},
		{time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), "0001-01-01T00:00:00+00:00"},
		{time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC), "9999-12-31T23:59:59.999999+00:00"},
		// Go only: Python holds no nanoseconds. IsoFormat truncates below the
		// microsecond; Python's fromtimestamp would round .123456789 to
		// .123457 (measured), fromisoformat truncates to .123456.
		{time.Date(2000, 1, 1, 0, 0, 0, 123456789, time.UTC), "2000-01-01T00:00:00.123456+00:00"},
		{time.Date(2000, 1, 1, 0, 0, 0, 999, time.UTC), "2000-01-01T00:00:00+00:00"},
	}
	for _, c := range cases {
		if got := IsoFormat(c.in); got != c.want {
			t.Errorf("IsoFormat(%v) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestParseAwareIsoFormatAcceptsWhatPythonReadsAsAware(t *testing.T) {
	// Left: the input; right: datetime.fromisoformat(input).isoformat() on
	// Python 3.14.7, which is also IsoFormat of the parsed value here.
	cases := []struct{ in, want string }{
		{"2000-01-01T00:00:00+00:00", "2000-01-01T00:00:00+00:00"},
		{"2000-01-01T00:00:00Z", "2000-01-01T00:00:00+00:00"},
		{"2000-01-01T00:00:00.1+00:00", "2000-01-01T00:00:00.100000+00:00"},
		{"2000-01-01T00:00:00.12+00:00", "2000-01-01T00:00:00.120000+00:00"},
		{"2000-01-01T00:00:00.123+00:00", "2000-01-01T00:00:00.123000+00:00"},
		{"2000-01-01T00:00:00.1234+00:00", "2000-01-01T00:00:00.123400+00:00"},
		{"2000-01-01T00:00:00.12345+00:00", "2000-01-01T00:00:00.123450+00:00"},
		{"2000-01-01T00:00:00.123456+00:00", "2000-01-01T00:00:00.123456+00:00"},
		{"2000-01-01T00:00:00.1234567+00:00", "2000-01-01T00:00:00.123456+00:00"},
		{"2000-01-01T00:00:00.123456789+00:00", "2000-01-01T00:00:00.123456+00:00"},
		{"2000-01-01T00:00:00,5+00:00", "2000-01-01T00:00:00.500000+00:00"},
		{"2000-01-01T00:00:00.5Z", "2000-01-01T00:00:00.500000+00:00"},
		{"2026-09-15T06:12:03.123456+02:00", "2026-09-15T06:12:03.123456+02:00"},
		{"2026-09-15T06:12:03-05:30", "2026-09-15T06:12:03-05:30"},
		{"2000-01-01 00:00:00+00:00", "2000-01-01T00:00:00+00:00"},
		{"2000-01-01T00:00:00+02:00:30", "2000-01-01T00:00:00+02:00:30"},
		{"2000-01-01T00:00:00-01:00:30", "2000-01-01T00:00:00-01:00:30"},
		{"2000-01-01T00:00:00-00:00", "2000-01-01T00:00:00+00:00"},
		{"2000-01-01T00:00:00+23:59", "2000-01-01T00:00:00+23:59"},
		{"2000-01-01T23:59:59-23:59", "2000-01-01T23:59:59-23:59"},
		{"2000-02-29T00:00:00+00:00", "2000-02-29T00:00:00+00:00"},
		{"2999-01-01T00:00:00+00:00", "2999-01-01T00:00:00+00:00"},
	}
	for _, c := range cases {
		got, ok := ParseAwareIsoFormat(c.in)
		if !ok || IsoFormat(got) != c.want {
			t.Errorf("ParseAwareIsoFormat(%+q) = %s, %v; want %s", c.in, IsoFormat(got), ok, c.want)
		}
	}
}

func TestParseAwareIsoFormatAnswersTheInstant(t *testing.T) {
	got, ok := ParseAwareIsoFormat("2026-09-15T06:12:03.5+02:00")
	if want := time.Date(2026, 9, 15, 4, 12, 3, 500000000, time.UTC); !ok || !got.Equal(want) {
		t.Errorf("ParseAwareIsoFormat = %v, %v; want %v", got, ok, want)
	}
}

func TestParseAwareIsoFormatRefusesWhatPythonRefusesOrReadsAsNaive(t *testing.T) {
	// Python 3.14.7: "naive" for the first three, ValueError for the rest.
	for _, in := range []string{
		"2000-01-01T00:00:00",
		"2000-01-01",
		"2000-01-01T00:00:00.5",
		"garbage",
		"",
		"2000-01-01T00:00:00z",
		"2000-01-01T00:00:00.+00:00",
		"2000-01-01T00:00:00+24:00",
		" 2000-01-01T00:00:00+00:00",
		"2000-01-01T00:00:00+00:00\n",
		"0000-01-01T00:00:00+00:00",
		"2000-1-01T00:00:00+00:00",
		"\U0000ff12\U0000ff10\U0000ff10\U0000ff10-01-01T00:00:00+00:00",
		"2000-01-01T00:00:00+00:00Z",
		"2000-01-01T00:00:00UTC",
		"2000-01-01T00:00:00+5:00",
		"2000-01-01T1:00:00+00:00",
		"2000-02-30T00:00:00+00:00",
		"1900-02-29T00:00:00+00:00",
		"2000-13-01T00:00:00+00:00",
		"2000-00-01T00:00:00+00:00",
		"2000-01-00T00:00:00+00:00",
		"2000-01-01T00:60:00+00:00",
		"2000-01-01T00:00:60+00:00",
		"2000-01-01T00:00:00+00:00.5",
		"2000-01-01T00:00:00+02:00:30Z",
	} {
		if got, ok := ParseAwareIsoFormat(in); ok {
			t.Errorf("ParseAwareIsoFormat(%+q) = %v, want false", in, got)
		}
	}
}

func TestParseAwareIsoFormatRefusesFormsOnlyPythonReads(t *testing.T) {
	// Python 3.14.7 reads each of these as aware; each is a parity line.
	for _, in := range []string{
		"2000-01-01t00:00:00+00:00",
		"2000-01-01X00:00:00+00:00",
		"2000-01-01\U000000e900:00:00+00:00",
		"2000-01-01T00:00+00:00",
		"2000-01-01T00+00:00",
		"20000101T000000+0000",
		"2000-01-01T00:00:00+0200",
		"2000-01-01T00:00:00+02",
		"2000-01-01T00:00:00+02:00:30.5",
		"2000-W01-1T00:00:00+00:00",
		"2000-01-01T24:00:00+00:00",
		"2000-01-01T00:00:00+00:60",
		"2000-01-01T00:00:00+00:00:60",
		"2000-01-01T00:00:00 +00:00",
	} {
		if got, ok := ParseAwareIsoFormat(in); ok {
			t.Errorf("ParseAwareIsoFormat(%+q) = %v, want false", in, got)
		}
	}
}

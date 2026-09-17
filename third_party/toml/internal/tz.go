package internal

import (
	"sync"
	"time"
)

// Timezones used for local datetime, date, and time TOML types.
//
// The exact way times and dates without a timezone should be interpreted is not
// well-defined in the TOML specification and left to the implementation. These
// defaults to current local timezone offset of the computer.
//
// loomux: upstream v1.6.0 resolves the offset in a package variable, so every
// process that imports the parser pays for loading the local time zone at
// start -- ~19 ms on Windows, where it is read from the registry, and a hook
// runs once per tool call. Here the three zones are built on first use, which
// only a TOML date or time without an offset ever asks for. The pointers stay
// stable across calls, so comparing a Location against them still works.
// Note that this behaviour is valid according to the TOML spec as the exact
// behaviour is left up to implementations.
var localZones = sync.OnceValue(func() [3]*time.Location {
	_, offset := time.Now().Zone()
	return [3]*time.Location{
		time.FixedZone("datetime-local", offset),
		time.FixedZone("date-local", offset),
		time.FixedZone("time-local", offset),
	}
})

// LocalDatetime is the zone of a TOML local date-time.
func LocalDatetime() *time.Location { return localZones()[0] }

// LocalDate is the zone of a TOML local date.
func LocalDate() *time.Location { return localZones()[1] }

// LocalTime is the zone of a TOML local time.
func LocalTime() *time.Location { return localZones()[2] }

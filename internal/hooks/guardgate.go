package hooks

import (
	"slices"
	"strings"
)

// gateReason refuses an agent the two commands that write the armed lanes.
const gateReason = "loomux gate arm and disarm decide which lanes fail the gate; a human runs them, and " +
	"an agent arms a lane only through a green `loomux check precommit --arm`. `loomux gate status` shows the lanes"

// armsOrDisarms says whether a shell line runs `loomux gate` with anything
// but status. An agent arms a lane only by a commit that goes through; the
// command would let it disarm one as well. The line is read the way
// writesConfiguration reads it, and has the same holes; with anyProgram, in
// strict mode, every program knownProgram does not name counts as loomux.
//
// What reads is named and everything else is refused, so a subcommand added
// later is refused until it is listed here; so is a flag in front of the
// subcommand, which gate answers with its usage. A bare `loomux gate` prints
// the usage and passes. A Start-Process of loomux is none of this rule's: the
// configuration rule refuses it already, whatever it runs, and a second
// reason would change what that refusal says.
func armsOrDisarms(line string, anyProgram bool) bool {
	return anyRunLine(line, func(l string, _ bool) bool { return lineArms(l, anyProgram) })
}

// lineArms is armsOrDisarms for one line, without the lines it runs from a
// string; no flag exempts here, so a nested line reads alike.
func lineArms(line string, anyProgram bool) bool {
	for _, variant := range lineVariants(line) {
		for _, segment := range segments(variant) {
			all := readings(segment)
			for k, words := range all {
				// The field reading, the last, splits quoted strings
				// (writesConfiguration).
				if readingArms(words, anyProgram, k < len(all)-1) {
					return true
				}
			}
		}
	}
	return false
}

// readingArms judges one reading from its head and from every word after a
// lone { or }, for readingWrites' reason: a block opens a command.
func readingArms(words []string, anyProgram, scan bool) bool {
	for i, w := range words {
		if (w == "{" || w == "}") && wordsArm(words[i+1:], anyProgram, scan) {
			return true
		}
	}
	return wordsArm(words, anyProgram, scan)
}

// wordsArm says whether one reading, past what runs in front of the program,
// is loomux gate with a subcommand other than status: as the program, as a
// wrapper named by a path in strict mode, or behind a program the guard does
// not know (behindUnknown).
func wordsArm(words []string, anyProgram, scan bool) bool {
	read := readPrefixes(words)
	if anyProgram && slices.ContainsFunc(read.named, func(call []string) bool { return programArms(call, true) }) {
		return true
	}
	if len(read.program) == 0 {
		return false
	}
	return programArms(read.program, anyProgram) ||
		scan && behindUnknown(read.program, func(rest []string) bool { return programArms(rest, false) })
}

// programArms is wordsArm for words that start with the program. A block's
// closing brace glued to the subcommand (status}) is no part of it, as in
// programWrites.
func programArms(words []string, anyProgram bool) bool {
	args, ok := programArgs(words, anyProgram)
	return ok && len(args) > 1 && args[0] == "gate" && strings.TrimRight(args[1], "})") != "status"
}

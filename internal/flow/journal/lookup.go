package journal

// Lookup is journal.py's lookup: the latest entry for this node and this input.
//
// An empty outcome matches any. A non-empty one skips entries that ended
// otherwise instead of stopping at the first match, because a visit limit and a
// gate's pause both write non-ok entries under the key of an entry that
// succeeded.
func Lookup(entries []Entry, node, inputHash, outcome string) (Entry, bool) {
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if entry.Node != node || entry.InputHash != inputHash {
			continue
		}
		if outcome == "" || entry.Outcome == outcome {
			return entry, true
		}
	}
	return Entry{}, false
}

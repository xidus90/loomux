package journal

// PendingGate is gate.py's PendingGate: a gate that stopped a run and is
// waiting for an answer.
//
// InputHash identifies the visit and not the node. A gate on a cycle pauses
// once per pass, and an answer addressed only to the node name would be spent
// on the first pass -- which the journal has already answered -- instead of on
// the pause that is actually open.
type PendingGate struct {
	Node      string
	Question  string
	InputHash string
}

// Pending is gate.py's `pending_gate`: the open question of this run, or nil.
//
// A damaged journal comes back as an error and never as "nothing waiting": a
// run whose journal cannot be read is not a run that has been answered.
func Pending(path string) (*PendingGate, error) {
	entries, err := Entries(path)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	last := entries[len(entries)-1]
	if last.Outcome != "paused" || last.Detail == nil {
		return nil, nil
	}
	return &PendingGate{Node: last.Node, Question: *last.Detail, InputHash: last.InputHash}, nil
}

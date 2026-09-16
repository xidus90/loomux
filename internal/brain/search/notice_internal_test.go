package search

import "testing"

func TestNewQmdMcpPortHandsItsNoticeToTheDefaultConnect(t *testing.T) {
	gotPort := 0
	var gotNotice func(string)
	connectDefault = func(port int, notice func(string)) ConnectFunc {
		gotPort, gotNotice = port, notice
		return nil
	}
	defer func() { connectDefault = DefaultConnect }()

	var heard []string
	NewQmdMcpPort(WithPort(9001), WithNotice(func(message string) { heard = append(heard, message) }))
	if gotPort != 9001 || gotNotice == nil {
		t.Fatalf("default connect got port %d and notice %v", gotPort, gotNotice != nil)
	}
	gotNotice("heard")
	if len(heard) != 1 || heard[0] != "heard" {
		t.Fatalf("the notice handed on is not the one given: %q", heard)
	}
}

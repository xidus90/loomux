package search

import (
	"errors"
	"fmt"
)

// ScriptedSearch is a scripted search outcome.
type ScriptedSearch struct {
	Hits []SearchHit
	Err  error
}

// ScriptedIndexed is a scripted listing outcome.
type ScriptedIndexed struct {
	Paths []string
	Err   error
}

// ScriptedPending is a scripted pending count outcome.
type ScriptedPending struct {
	Count int
	Err   error
}

// FakePort hands back scripted results and records what it was asked.
type FakePort struct {
	Results   []ScriptedSearch
	Refreshes []error
	Listings  map[string]ScriptedIndexed
	Pending   []ScriptedPending

	Calls            []SearchCall
	Refreshed        [][]string
	Listed           []string
	Embedded         [][]string
	SearchableCounts int
}

// SearchCall records a call to Search.
type SearchCall struct {
	Query       string
	Collections []string
	Profile     Profile
	N           int
}

// NewFakePort creates a FakePort.
func NewFakePort() *FakePort {
	return &FakePort{
		Listings: make(map[string]ScriptedIndexed),
	}
}

// Search executes a scripted search.
func (f *FakePort) Search(query string, collections []string, profile Profile, n int) ([]SearchHit, error) {
	f.Calls = append(f.Calls, SearchCall{
		Query:       query,
		Collections: collections,
		Profile:     profile,
		N:           n,
	})
	if len(f.Results) == 0 {
		return nil, errors.New("FakePort ran out of scripted results")
	}
	scripted := f.Results[0]
	f.Results = f.Results[1:]
	if scripted.Err != nil {
		return nil, scripted.Err
	}
	return scripted.Hits, nil
}

// Indexed returns scripted indexed paths.
func (f *FakePort) Indexed(collection string) ([]string, error) {
	f.Listed = append(f.Listed, collection)
	if item, ok := f.Listings[collection]; ok {
		if item.Err != nil {
			return nil, item.Err
		}
		return item.Paths, nil
	}
	return nil, fmt.Errorf("FakePort has no listing scripted for %q", collection)
}

// Refresh records refresh.
func (f *FakePort) Refresh(collections []string) error {
	f.Refreshed = append(f.Refreshed, collections)
	if len(f.Refreshes) > 0 {
		err := f.Refreshes[0]
		f.Refreshes = f.Refreshes[1:]
		if err != nil {
			return err
		}
	}
	return nil
}

// NotYetSearchable returns scripted pending count.
func (f *FakePort) NotYetSearchable() (int, error) {
	f.SearchableCounts++
	if len(f.Pending) > 0 {
		item := f.Pending[0]
		f.Pending = f.Pending[1:]
		if item.Err != nil {
			return 0, item.Err
		}
		return item.Count, nil
	}
	return 0, nil
}

// Embed records embed.
func (f *FakePort) Embed(collections []string) error {
	f.Embedded = append(f.Embedded, collections)
	return nil
}

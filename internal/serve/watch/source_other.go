//go:build !windows

package watch

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// fsSource watches through fsnotify, which cannot watch a tree: every
// directory below a root gets a watch of its own, and one made later is added
// when its creation is reported.
type fsSource struct {
	w      *fsnotify.Watcher
	events chan Event
	mu     sync.Mutex
	// owners maps each watched directory to the roots it serves, and whether
	// it is part of the root's tree or only the directory of an extra file.
	// Nested areas share directories, so a watch goes only with its last root.
	owners map[string]map[string]bool
}

// NewSource returns the platform's change source.
//
//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func NewSource() (Source, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	s := &fsSource{w: w, events: make(chan Event), owners: map[string]map[string]bool{}}
	go s.pump()
	return s, nil
}

//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (s *fsSource) Events() <-chan Event { return s.events }

// Add watches every directory of root's tree outside dot directories, and the
// directories of extra flatly. A missing directory of extra is skipped: git
// creates info/ only when it needs it. A watch that fails, an exhausted
// inotify limit above all, fails the whole root.
//
//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (s *fsSource) Add(root string, extra []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.addTree(root, root); err != nil {
		s.removeLocked(root)
		return err
	}
	for _, p := range extra {
		s.watch(filepath.Dir(p), root, false)
	}
	return nil
}

// addTree watches dir and the directories below it. Only dir itself must be
// readable; a subdirectory that vanishes or refuses is left out.
//
//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (s *fsSource) addTree(dir, root string) error {
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil && p == dir:
			return err
		case err != nil || !d.IsDir():
			return nil
		case p != dir && strings.HasPrefix(d.Name(), "."):
			return filepath.SkipDir
		}
		// A subdirectory gone since the walk read it costs nothing; the
		// others below dir still need their watches.
		if err := s.watch(p, root, true); err != nil && (p == dir || !errors.Is(err, fs.ErrNotExist)) {
			return err
		}
		return nil
	})
}

// watch adds dir every time, even when owners knows it: a watch dies with its
// directory, and one made again under the same name needs a new watch.
//
// ponytail: owners keeps the entries of directories that never come back;
// prune them on Remove/Rename events if long-lived services show it.
//
//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (s *fsSource) watch(dir, root string, tree bool) error {
	if err := s.w.Add(dir); err != nil {
		return err
	}
	if s.owners[dir] == nil {
		s.owners[dir] = map[string]bool{}
	}
	s.owners[dir][root] = s.owners[dir][root] || tree
	return nil
}

//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (s *fsSource) Remove(root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeLocked(root)
}

//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (s *fsSource) removeLocked(root string) {
	for dir, roots := range s.owners {
		delete(roots, root)
		if len(roots) == 0 {
			s.w.Remove(dir)
			delete(s.owners, dir)
		}
	}
}

// Close ends the watches and returns once the pump is gone. Nobody reads
// Events any more, so Close drains it meanwhile.
//
//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (s *fsSource) Close() error {
	drained := make(chan struct{})
	go func() {
		for range s.events {
		}
		close(drained)
	}()
	err := s.w.Close()
	<-drained
	return err
}

// pump turns fsnotify's events into this package's until the watcher closes.
// An error fsnotify cannot tie to a directory ends the watch of every root, as
// changes may be lost from then on.
//
//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (s *fsSource) pump() {
	defer close(s.events)
	for {
		select {
		case e, ok := <-s.w.Events:
			if !ok {
				return
			}
			for _, ev := range s.route(e) {
				s.events <- ev
			}
		case err, ok := <-s.w.Errors:
			if !ok {
				return
			}
			overflow := errors.Is(err, fsnotify.ErrEventOverflow)
			for _, root := range s.allRoots() {
				if overflow {
					s.events <- Event{Root: root, Overflow: true}
				} else {
					s.events <- Event{Root: root, Err: err}
				}
			}
		}
	}
}

// route turns an event into one for each root it belongs to, and watches a new
// directory for the roots whose tree it joins. A root whose new directory
// cannot be watched gets an error instead, as it is watched only in part.
//
//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (s *fsSource) route(e fsnotify.Event) []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	var evs []Event
	for root, tree := range s.owners[filepath.Dir(e.Name)] {
		ev := Event{Root: root, Path: e.Name}
		if tree && e.Has(fsnotify.Create) && !strings.HasPrefix(filepath.Base(e.Name), ".") {
			if info, err := os.Lstat(e.Name); err == nil && info.IsDir() {
				// A directory gone again by now has nothing left to watch.
				if err := s.addTree(e.Name, root); err != nil && !errors.Is(err, fs.ErrNotExist) {
					ev = Event{Root: root, Err: err}
				}
			}
		}
		evs = append(evs, ev)
	}
	return evs
}

//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (s *fsSource) allRoots() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	set := map[string]bool{}
	for _, roots := range s.owners {
		for root := range roots {
			set[root] = true
		}
	}
	var out []string
	for root := range set {
		out = append(out, root)
	}
	return out
}

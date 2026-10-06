package apply

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/lock"
)

// registerName is `_REGISTER` (apply.py:194): the one scaffold file a
// registered area may keep outside the vault.
const registerName = "_identities.tsv"

// place writes only below its anchors and records every file it touched,
// so an abort can name them. It is `_Place` (apply.py:280-295) reduced to
// what the barrier reads; where a case lies and what it names are the
// caller's to resolve.
type place struct {
	anchor string // the vault
	wiki   string
	// registers are the `_identities.tsv` of every registered area, which
	// `_is_external_register` (apply.py:532-547) reads from the registry on
	// each call. The caller hands them over once, so the barrier reads no
	// registry of its own.
	registers []string
	touched   []string // relative to anchor, in write order
}

// The barrier's three questions to the file system, as variables so that a
// test can model an answer the machine it runs on cannot give -- an 8.3
// alias on a volume that keeps none, a link where symlinks need a
// privilege, a deletion that fails half-way. Python's tests monkeypatch
// `_resolved`, `_is_link` and `rmtree` for the same reasons.
var (
	// resolvePath is `_resolved`, `Path.resolve()`. The guard's resolver is
	// that port already: it follows junctions, which `is_symlink` misses,
	// and folds an 8.3 alias to the long name, which is how the alias of a
	// scaffold file is caught.
	resolvePath = guard.ResolvePath
	isLink      = linkAt
	removeAll   = os.RemoveAll
	replaceText = lock.ReplaceText
)

// linkAt is `Path.is_symlink`: true for a symlink only. A junction is not
// one here any more than in Python; it is caught where it matters, by the
// containment check, because it resolves out of the vault.
func linkAt(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

// gateError is a refusal of the barrier, as opposed to a failure of the
// disk: the caller reports the one as its own kind of error and passes the
// other on as it is.
type gateError struct{ msg string }

func (e *gateError) Error() string { return e.msg }

func refuse(format string, args ...any) error {
	return &gateError{msg: fmt.Sprintf(format, args...)}
}

// preflight is `_preflight` (apply.py:467-484): the fixed write targets are
// asked before a single byte is written, because a link found at the
// append would leave the page patched and the protocols half-kept.
func (p *place) preflight(caseDir string) error {
	for _, path := range []string{
		filepath.Join(p.wiki, "log.md"),
		filepath.Join(p.wiki, "audit.md"),
		filepath.Join(p.anchor, registerName),
	} {
		if err := p.gate(path, true); err != nil {
			return err
		}
	}
	return p.gate(caseDir, false)
}

// write is `_write` for a page: a scaffold name anywhere on the path is
// refused.
func (p *place) write(path, text string) error {
	if err := p.gate(path, false); err != nil {
		return err
	}
	return p.touch(path, func() (bool, error) { return replaceIfChanged(path, text) })
}

// writeScaffold is `_write(..., scaffold=True)`: the named way to `log.md`,
// `audit.md` and `_identities.tsv`, the one write whose last component must
// be a scaffold file.
func (p *place) writeScaffold(path, text string) error {
	if err := p.gate(path, true); err != nil {
		return err
	}
	return p.touch(path, func() (bool, error) { return replaceIfChanged(path, text) })
}

// recordCase is `_record_case`: the only way `case.toml` is written back.
func (p *place) recordCase(path string, c maintenance.Case) error {
	if err := p.gate(path, false); err != nil {
		return err
	}
	return p.touch(path, func() (bool, error) { return maintenance.WriteCase(path, c) })
}

// remove is `_remove`: recorded once the directory is known to be there,
// before the deletion runs, and never withdrawn, because a deletion that
// stops half-way has still changed the directory.
func (p *place) remove(dir string) error {
	if err := p.gate(dir, false); err != nil {
		return err
	}
	// `rmtree` raises on a directory that is not there, and os.RemoveAll
	// answers nil: without this a missing directory would pass as deleted.
	if _, err := os.Lstat(dir); err != nil {
		return err
	}
	p.touched = append(p.touched, p.relative(dir))
	return removeAll(dir)
}

// touch is `_touch` (apply.py:511-529): the record stands before the write
// runs, so a write that fails half-way is still named, and it is withdrawn
// when the write changed no byte, since git would show nothing to look at.
func (p *place) touch(path string, operation func() (bool, error)) error {
	p.touched = append(p.touched, p.relative(path))
	changed, err := operation()
	if err != nil {
		return err
	}
	if !changed {
		p.touched = p.touched[:len(p.touched)-1]
	}
	return nil
}

// replaceIfChanged is `write_if_changed`: the bytes are compared as they
// stand, so a file that differs in its line endings alone is rewritten, and
// the write swaps a whole file in.
func replaceIfChanged(path, text string) (bool, error) {
	standing, err := os.ReadFile(path)
	if err == nil && string(standing) == text {
		return false, nil
	}
	// `path.exists()` is false only for a missing file; anything else that
	// keeps the file from being read is a failure the write must not hide.
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := replaceText(path, text); err != nil {
		return false, err
	}
	return true, nil
}

// gate is `_gate` (apply.py:556-632): everything a path must survive before
// the disk is touched, in one walk from the anchor down.
//
// No component may be a link, since the write would follow it while the
// record names the path as spelt. Every component must resolve inside the
// anchor, which is what catches a junction out of the vault. And no
// component may carry a scaffold name, neither as spelt nor as the file
// system resolves it -- the second is what an 8.3 alias gives away.
// `scaffold` exempts the last component alone and then demands that it
// really is a scaffold file, so a miswired call site fails loudly.
func (p *place) gate(path string, scaffold bool) error {
	if scaffold && p.isExternalRegister(path) {
		if isLink(path) {
			return refuse("%s: is a link, not a regular register file", path)
		}
		return nil
	}
	anchor, root, err := p.anchorFor(path)
	if err != nil {
		return err
	}
	parts, ok := lexicalParts(path, anchor)
	if !ok {
		return refuse("%s: not inside the vault %s", path, p.anchor)
	}
	if len(parts) == 0 {
		return refuse("%s: the vault root is not a write target", path)
	}
	last := len(parts) - 1
	if scaffold && !isScaffoldName(parts[last]) {
		return refuse("%s: declared a scaffold write, but %s is not a scaffold file", path, parts[last])
	}
	walked := strings.TrimRight(anchor, `\/`)
	for index, part := range parts {
		walked += string(filepath.Separator) + part
		if isLink(walked) {
			return refuse("%s: %s is a link, not a directory", walked, filepath.Base(walked))
		}
		resolved, err := resolvePath(walked)
		if err != nil {
			return err
		}
		exempt := scaffold && index == last
		if !exempt && (isScaffoldName(part) || isScaffoldName(filepath.Base(resolved))) {
			return refuse("%s: a scaffold file, not a page a case may change", path)
		}
		if !guard.IsRelativeTo(resolved, root) {
			return refuse("%s: resolves to a path not inside the vault", walked)
		}
	}
	return nil
}

// isExternalRegister is `_is_external_register`: the path names one of the
// registered registers, as spelt or as resolved. The caller has already
// asked for a scaffold write; the name must still be the register's. A
// register that cannot be resolved matches nothing, and the path then takes
// the walk, which refuses it outside the vault.
func (p *place) isExternalRegister(path string) bool {
	if !strings.EqualFold(normalisedName(filepath.Base(path)), registerName) {
		return false
	}
	resolved, err := resolvePath(path)
	if err != nil {
		return false
	}
	for _, valid := range p.registers {
		if samePath(path, valid) {
			return true
		}
		if other, err := resolvePath(valid); err == nil && samePath(resolved, other) {
			return true
		}
	}
	return false
}

// anchorFor is `_anchor` (apply.py:449-464): the vault, except for a write
// under a wiki that lies outside it -- `_resolve` admits such a wiki only as
// a real bundle, and it is the one tree such a write can belong to. The
// anchor comes back as spelt, for the lexical walk, and as resolved, for
// the containment check.
func (p *place) anchorFor(path string) (anchor, root string, err error) {
	resolved, err := resolvePath(path)
	if err != nil {
		return "", "", err
	}
	wikiDir, err := resolvePath(p.wiki)
	if err != nil {
		return "", "", err
	}
	vault, err := resolvePath(p.anchor)
	if err != nil {
		return "", "", err
	}
	if guard.IsRelativeTo(resolved, wikiDir) && !guard.IsRelativeTo(wikiDir, vault) {
		return p.wiki, wikiDir, nil
	}
	return p.anchor, vault, nil
}

// relative is `_relative`: the vault-relative path in slashes, the path
// itself for one outside the vault.
func (p *place) relative(path string) string {
	if parts, ok := lexicalParts(path, p.anchor); ok {
		return strings.Join(parts, "/")
	}
	return filepath.ToSlash(path)
}

// samePath is `Path.__eq__`: equal components under the platform's case
// rule.
func samePath(left, right string) bool {
	return guard.IsRelativeTo(left, right) && guard.IsRelativeTo(right, left)
}

// lexicalParts is `path.relative_to(base).parts`: decided on the spelling
// alone, with `.` dropped as pathlib drops it and `..` kept, so that the
// walk resolves it rather than a cleaning that would hide it.
func lexicalParts(path, base string) ([]string, bool) {
	if !guard.IsRelativeTo(path, base) {
		return nil, false
	}
	var parts []string
	for _, part := range components(path)[len(components(base)):] {
		if part != "." {
			parts = append(parts, part)
		}
	}
	return parts, true
}

// components splits a path as the guard's IsRelativeTo counts it: the
// volume with its separator first, then every name.
func components(path string) []string {
	volume := filepath.VolumeName(path)
	rest := path[len(volume):]
	var parts []string
	if rest != "" && os.IsPathSeparator(rest[0]) {
		volume += rest[:1]
		rest = rest[1:]
	}
	if volume != "" {
		parts = append(parts, volume)
	}
	return append(parts, strings.FieldsFunc(rest, func(r rune) bool {
		return r < 0x80 && os.IsPathSeparator(byte(r))
	})...)
}

// normalisedName is `_normalised` (apply.py:635-653) short of the fold: one
// component as the file system opens it. Everything from a `:` names a
// stream of the file in front of it, and a trailing dot or blank is dropped
// when a name is opened.
func normalisedName(name string) string {
	if cut := strings.IndexByte(name, ':'); cut >= 0 {
		name = name[:cut]
	}
	return strings.TrimRight(name, ". ")
}

// isScaffoldName asks the normalised name of the scaffold set. EqualFold
// stands in for `casefold`: it folds the long s onto `s` as casefold does,
// and the five names are ASCII, so no fold that changes a length can reach
// one of them.
func isScaffoldName(name string) bool {
	name = normalisedName(name)
	for scaffold := range wiki.ScaffoldFiles {
		if strings.EqualFold(name, scaffold) {
			return true
		}
	}
	return false
}

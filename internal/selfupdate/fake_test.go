package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// fakeGH answers the four calls an update makes: the release list, the
// installed binary's --version (key "installed"), the download into --dir,
// and the downloaded binary's --version (key "--version").
type fakeGH struct {
	list      string
	files     map[string]string
	installed string
	version   string
	fail      map[string]error
	calls     []string
}

func (f *fakeGH) run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := "--version"
	switch {
	case name == "gh":
		key = args[1]
	case filepath.Base(name) == "loomux.exe":
		key = "installed"
	}
	f.calls = append(f.calls, key)
	if err := f.fail[key]; err != nil {
		return nil, err
	}
	switch key {
	case "list":
		return []byte(f.list), nil
	case "download":
		dir := args[len(args)-1]
		for n, body := range f.files {
			if err := os.WriteFile(filepath.Join(dir, n), []byte(body), 0o755); err != nil {
				return nil, err
			}
		}
		return nil, nil
	case "installed":
		return []byte(f.installed), nil
	}
	return []byte(f.version), nil
}

// release is a fake whose only release is a well-formed v<ver> beta.
func release(ver string) *fakeGH {
	asset := AssetName(ver, "windows", "amd64")
	body := "binary " + ver
	sum := sha256.Sum256([]byte(body))
	return &fakeGH{
		list:    fmt.Sprintf(`[{"tagName":"v%s","isPrerelease":true}]`, ver),
		files:   map[string]string{asset: body, "SHA256SUMS": hex.EncodeToString(sum[:]) + "  " + asset + "\n"},
		version: "loomux " + ver + " (beta)\n",
		fail:    map[string]error{},
	}
}

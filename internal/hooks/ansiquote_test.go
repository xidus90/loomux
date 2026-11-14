package hooks

import (
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

func TestDollarQuotedSpellsTheWordTheShellReads(t *testing.T) {
	for in, want := range map[string]string{
		`echo x > $'a/config.toml'`: `echo x > 'a/config.toml'`,
		`echo x > $"a/config.toml"`: `echo x > "a/config.toml"`,
		`rm $'a'/b`:                 `rm 'a'/b`,
		`echo $'\x63onfig'`:         `echo 'config'`,
		`echo $'\143onfig'`:         `echo 'config'`,
		`echo $'\u0063onfig'`:       `echo 'config'`,
		`echo $'\U00000063onfig'`:   `echo 'config'`,
		`echo $'\777'`:              "echo '\xff'",
		`echo $'\UFFFFFFFF'`:        "echo '�'",
		`echo $'\U00110000'`:        "echo '�'",
		`echo $'a\tb\n\e\\\'\"\?'`:  "echo 'a\tb\n\x1b\\'\\''\"?'",
		`echo $'a\qb'`:              `echo 'a\qb'`,
		`echo $'a\xzz'`:             `echo 'a\xzz'`,
		`echo $'open`:               `echo 'open'`,
		`echo $'end\`:               `echo 'end\'`,
		`echo '$'"x" "$'a'" \$'b'`:  `echo '$'"x" "$'a'" \$'b'`,
		`echo "say \"$'a'\"" $'b'`:  `echo "say \"$'a'\"" 'b'`,
		`plain line $ and $`:        `plain line $ and $`,
	} {
		if got := dollarQuoted(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}

// A manifest named in ANSI-C quoting is still the manifest.
func TestAnAnsiQuotedManifestPathIsRefused(t *testing.T) {
	root := t.TempDir()
	for _, line := range []string{
		`echo x > $'.loomux/config.toml'`,
		`echo x > $'.loomux/\x63onfig.toml'`,
		`rm $'.loomux'/config.toml`,
	} {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Equal(got, []string{manifestReason}) {
			t.Errorf("%q: reasons %q, want the manifest's", line, got)
		}
	}
}

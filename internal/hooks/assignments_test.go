package hooks

import (
	"maps"
	"slices"
	"testing"
)

// Every form of a variable or alias a line sets is read, with its value
// as the shell hands it on.
func TestAssignmentsReadsEveryForm(t *testing.T) {
	for line, want := range map[string]assigned{
		"D=.loomux; echo $D":                   {vars: map[string][]string{"d": {".loomux"}}},
		"export D=.loomux X=1":                 {vars: map[string][]string{"d": {".loomux"}, "x": {"1"}}},
		"declare -x D='a b'":                   {vars: map[string][]string{"d": {"a b"}}},
		"local D=x; readonly E=y; typeset F=z": {vars: map[string][]string{"d": {"x"}, "e": {"y"}, "f": {"z"}}},
		"$D='.loomux'":                         {vars: map[string][]string{"d": {".loomux"}}},
		"$D = '.loomux'":                       {vars: map[string][]string{"d": {".loomux"}}},
		"${D}=x":                               {vars: map[string][]string{"d": {"x"}}},
		"$env:D = 'x'":                         {vars: map[string][]string{"d": {"x"}}},
		"Set-Variable D .loomux":               {vars: map[string][]string{"d": {".loomux"}}},
		"sv -Name D -Value x":                  {vars: map[string][]string{"d": {"x"}}},
		"New-Variable -Value x -Name D":        {vars: map[string][]string{"d": {"x"}}},
		"set D=.loomux& echo %D%":              {vars: map[string][]string{"d": {".loomux"}}},
		"alias l=loomux":                       {aliases: map[string][]string{"l": {"loomux"}}},
		"alias ll='ls -l' m=n":                 {aliases: map[string][]string{"ll": {"ls -l"}, "m": {"n"}}},
		"Set-Alias l loomux":                   {aliases: map[string][]string{"l": {"loomux"}}},
		"sal -Name l -Value loomux":            {aliases: map[string][]string{"l": {"loomux"}}},
		"D=a; D=b":                             {vars: map[string][]string{"d": {"a", "b"}}},
		"echo a=b; x --flag=y; 1x=2":           {},
		"$D == 'x'; Set-Variable; sal l":       {},
		"$D xy; $D=; =x":                       {},
		"_D=x":                                 {vars: map[string][]string{"_d": {"x"}}},
	} {
		got := assignments(line)
		if !maps.EqualFunc(got.vars, want.vars, slices.Equal) || !maps.EqualFunc(got.aliases, want.aliases, slices.Equal) {
			t.Errorf("%q: %+v, want %+v", line, got, want)
		}
	}
}

// A variable is put in at $NAME, ${NAME}, $env:NAME and %NAME%, in any
// case, but not where it is set; an alias where it stands as a word.
func TestSubstitutedPutsEveryValueInPlace(t *testing.T) {
	for line, want := range map[string][]string{
		"D=x; echo $D/a ${D}b $env:D %D% $d $DX": {"D=x; echo x/a xb x x x $DX"},
		"$D = 'x'; rm $D/a":                      {"$D = 'x'; rm x/a"},
		"D=a; D=b; rm $D":                        {"D=a; D=b; rm a", "D=a; D=b; rm b"},
		"alias l=loomux; l init; ls; (l x)":      {"alias l=loomux; loomux init; ls; (loomux x)"},
		"echo $D":                                nil,
		"D=x; if ($D == 'y') {}":                 {"D=x; if (x == 'y') {}"},
		"alias l=loomux; ls -l x":                {"alias l=loomux; ls -l x"},
	} {
		if got := substituted(line); !slices.Equal(got, want) {
			t.Errorf("%q: %q, want %q", line, got, want)
		}
	}
	many := "A=1; A=2; B=1; B=2; C=1; C=2; D=1; D=2; E=1; E=2; echo $A$B$C$D$E"
	if got := substituted(many); len(got) != maxSubstituted {
		t.Errorf("%d variants, want the cap %d", len(got), maxSubstituted)
	}
}

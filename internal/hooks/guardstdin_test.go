package hooks

import (
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// stdinLines are command lines about interpreters and what the stdin rule
// says of each: true for a refusal.
func stdinLines() map[string]bool {
	return map[string]bool{
		"python -":  true,
		"python3 -": true,
		"python":    true,
		"cd x && python3 - <<'EOF'\nprint(1)\nEOF": true,
		"python <<EOF\nprint(1)\nEOF":              true,
		"PYTHONIOENCODING=utf8 python -":           true,
		"C:/Python314/python.exe -":                true,
		`C:\Python314\python.exe -`:                true,
		"node":                                     true,
		"uv run python -":                          true,
		"python -X utf8 -":                         true,
		"python - > out.txt 2>&1":                  true,
		"uv run -":                                 true,
		"uv run --with rich -":                     true,
		"uvx --from x python -":                    true,
		"perl -c":                                  true,
		"python -E -":                              true,
		"py -3.12 -":                               true,
		"python3.12 -":                             true,
		"ruby -E utf-8 -":                          true,
		"node --input-type=module -":               true,
		"node -r fs -":                             true,
		"python -W ignore -":                       true,
		"python -- -":                              true,
		"python -u":                                true,
		"cat <<'EOF' | python -\nprint(1)\nEOF":    true,
		"@'\nprint(1)\n'@ | python -":              true,
		"$code = @\"\nprint(1)\n\"@ | python -":    true,
		`sh -c "python -"`:                         true,
		"a || python -":                            true,
		"python <<<'print(1)'":                     true,
		"echo x | python - <<'EOF'\nprint(1)\nEOF": true,
		"cat <<-EOF > a.txt\n\tx\n\tEOF\npython -": true,
		"cat <<A <<B\nx\nA\ny\nB\nnode -":          true,
		"nohup python -":                           true,
		"env -u X python -":                        true,
		"time python -":                            true,
		"exec python -":                            true,
		"echo '@'\npython -":                       true,
		"x ;python -":                              true,
		"python -S \"C:/Users/micro/.claude/scripts/x.py\"": false,
		"python -c 'print(1)'":                              false,
		"python -m pytest -q":                               false,
		"python script.py arg":                              false,
		"echo 'print(1)' | python -":                        false,
		"python - < script.py":                              false,
		"python - 0<script.py":                              false,
		"go test ./... && python3 tools/x.py":               false,
		"node -e 'console.log(1)'":                          false,
		"cat <<'EOF' > file.txt\npython -\nEOF":             false,
		"git log --format=%B":                               false,
		"echo python":                                       false,
		"uv run python script.py":                           false,
		"uv run --script tool.py":                           false,
		"uv run":                                            false,
		"uv run pytest -q":                                  false,
		"python --version":                                  false,
		"python -V":                                         false,
		"node -v":                                           false,
		"perl -v":                                           false,
		"ruby -v":                                           false,
		"perl -ne 'print' f.txt":                            false,
		"perl -e'print 1'":                                  false,
		"python -uc 'print(1)'":                             false,
		"python -W ignore script.py":                        false,
		"python -Wignore script.py":                         false,
		"python script.py -":                                false,
		"node --eval=1":                                     false,
		"node --print 1":                                    false,
		"py -V:3.12 x.py":                                   false,
		"git commit -m \"fix; python -\"":                   false,
		"Get-Content x.py | python -":                       false,
		"cat <<'EOF' > a.md\nuv run -\nEOF":                 false,
		"cat <<\"END\" > a.md\nnode\nEND":                   false,
		"cat <<\\END > a.md\nperl\nEND":                     false,
		"$t = @'\npython -\n'@":                             false,
		"pythonic -":                                        false,
		"cat x | grep y | python script.py":                 false,
		"bash <<< 'echo hi'":                                false,
		"ruby -e 'p 1'":                                     false,
		"uv build -":                                        false,
		"python -- -u":                                      false,
		"sh -c \"cat <<EOF > f\npython -\nEOF\"":            false,
		"cat <<'EOF' > f\nx\nEOF\npython -":                 true,
		"perl -E'say 1'":                                    false,
		"node --input-type=module x.js":                     false,
		"python > out.txt":                                  true,
		"node --input-type module -":                        true,
		"python -c'print(1)'":                               false,
		"perl -I lib -":                                     true,
		"command -v python":                                 false,
		"if command -v python3 >/dev/null 2>&1; then echo y; fi": false,
		"X=1 command -v python":                                  false,
		"command -V node":                                        false,
		"command -p python -":                                    true,
		"command python -":                                       true,
		"node --test":                                            false,
		"node --test tests/":                                     false,
		"node --run build":                                       false,
		"py --list":                                              false,
		"py --list-paths":                                        false,
		"py -0":                                                  false,
		"py -0p":                                                 false,
		"perl --version":                                         false,
		"perl --help":                                            false,
		"python3.14t -":                                          true,
		"echo $(date) | python -":                                false,
		"cat $(ls) | python -":                                   false,
		"(echo x) | python -":                                    false,
		"(cat <<'EOF') | python -\nprint(1)\nEOF":                true,
		"x |& python -":                                          true,
		"uv run --script -":                                      true,
		"uv run -- python -":                                     true,
		"python -i":                                              true,
		"python -v -":                                            true,
		"command uv run -":                                       true,
		"| python -":                                             true,
		" | python -":                                            true,
		"(cat <<'EOF'; echo x) | python -\nprint(1)\nEOF":           true,
		"cat <<'EOF' $(date) | python -\nprint(1)\nEOF":             true,
		"{ cat <<'EOF'; } | python -\nprint(1)\nEOF":                true,
		"cat <<'EOF' | grep p | python -\nprint(1)\nEOF":            true,
		"{ cat <<'EOF'\nprint(1)\nEOF\n} | python -":                true,
		"cat <<'EOF' > f.txt\nx\nEOF\necho 'print(1)' | python -":   false,
		"cat <<'EOF' > f.txt; echo 'print(1)' | python -\nx\nEOF":   false,
		"cat <<'EOF' > f.txt && echo 'print(1)' | python -\nx\nEOF": false,
		"cat <<'EOF' > f.txt & echo 'print(1)' | python -\nx\nEOF":  false,
		"cat <<'EOF' > f.txt || echo 'print(1)' | python -\nx\nEOF": false,
		"(echo 'print(1)'; true) | python -":                        false,
		"{ echo 'print(1)'; } | python -":                           false,
		"echo ${X} | python -":                                      false,
		"echo $(python -)":                                          true,
		"{ cat <<'EOF'; }; echo 'print(1)' | python -\nx\nEOF":      false,
		"(cat <<'EOF' > f); echo 'print(1)' | python -\nx\nEOF":     false,
		"echo a); (cat <<'EOF'; echo b) | python -\nx\nEOF":         true,
		"(a || python -)":           true,
		"{ a || python -; }":        true,
		"echo $(a || python -)":     true,
		"a && (b || python -)":      true,
		"if ($x) { a || python - }": true,
		"echo ' { '; a || python -": true,
		"Get-ChildItem | ForEach-Object { $_.Name}\nSet-Content f.txt @'\nx\n'@\nGet-Content x.py | python -": false,
		"echo '{'\ncat <<'EOF' > f\nx\nEOF\necho 'print(1)' | python -":                                       false,
		"echo ' { '\ncat <<'EOF' > f\nx\nEOF\necho 'print(1)' | python -":                                     false,
		"}\n{\ncat <<'EOF' > f\nx\nEOF\necho 'print(1)' | python -":                                           false,
		// Named limits, held as they are: a REPL after a script, and a
		// line ending in @" that is no PowerShell here-string.
		"python -i script.py":               false,
		"git commit -m \"see @\"\npython -": false,
	}
}

// The rule refuses exactly the lines it means, in both modes and from every
// tool that carries a shell line.
func TestTheGuardRefusesAnInterpreterThatReadsStdin(t *testing.T) {
	root := t.TempDir()
	for line, want := range stdinLines() {
		for _, strict := range []bool{false, true} {
			got := slices.Contains(checkTool(root, "Bash", map[string]any{"command": line}, config.Policy{Strict: strict}), stdinReason)
			if got != want {
				t.Errorf("strict %v, %q: refused %v, want %v", strict, line, got, want)
			}
		}
	}
	for tool, input := range map[string]map[string]any{
		"PowerShell":  {"command": "python -"},
		"run_command": {"CommandLine": "python -"},
		"manage_task": {"Action": "send_input", "Input": "python -\n"},
	} {
		if !slices.Contains(checkTool(root, tool, input, config.Policy{}), stdinReason) {
			t.Errorf("%s: python - passes", tool)
		}
	}
}

// A heredoc's body is cut up to its end word, and a here-string becomes <<@.
func TestWithoutHereBodiesKeepsTheCommands(t *testing.T) {
	for line, want := range map[string]string{
		"cat <<EOF > f\nx\nEOF\npython -": "cat <<EOF > f\npython -",
		"cat <<-EOF\n\tx\n\tEOF\nls":      "cat <<-EOF\nls",
		"cat << 'E O'\nx":                 "cat << 'E O'",
		"cat <<EOF\r\nx\r\nEOF\r\nls":     "cat <<EOF\r\nls",
		"echo '<<EOF'\nls":                "echo '<<EOF'\nls",
		"bash <<< x\nls":                  "bash <<< x\nls",
		"@'\nx\n'@ | python -":            "<<@ | python -",
		"$a = @\"\nx\n\"@":                "$a = <<@",
		"(@'\nx":                          "(<<@",
		"echo '@'\nls":                    "echo '@'\nls",
		"cat <<A <<B\n1\nA\n2\nB\nls":     "cat <<A <<B\nls",
		"cat <<\nls":                      "cat <<\nls",
		"cat <<'E'\nx\nE\nls":             "cat <<'E'\nls",
		"cat <<\"E\"\nx\nE\nls":           "cat <<\"E\"\nls",
		"cat <<\\E\nx\nE\nls":             "cat <<\\E\nls",
		"bash <<<x\nls":                   "bash <<<x\nls",
	} {
		if got := withoutHereBodies(line); got != want {
			t.Errorf("%q: %q, want %q", line, got, want)
		}
	}
}

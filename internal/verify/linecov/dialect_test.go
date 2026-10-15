package linecov

import "testing"

func TestDialectsMatchTheWayTheirToolsDo(t *testing.T) {
	const root = "/w/app"
	for _, row := range []struct {
		dialect, pattern, rel string
		want                  bool
	}{
		{"nano", "**/*_test.gd", "test/a_test.gd", true},
		{"nano", "**/*_test.gd", "a_test.gd", true},
		{"nano", "res://harness/**", "harness/p.gd", true},
		{"nano", "res://harness/**", "core/harness/p.gd", false},
		{"nano", "addons/**", "addons/x/plugin.gd", false}, // Nano matches full res:// paths
		{"nano", "res://addons/**", "addons/x/plugin.gd", true},
		{"lcov", "*/third_party/*", "src/third_party/x.c", true},
		{"lcov", "third", "src/third_party/x.c", true},
		{"lcov", "/w/app/src/*", "src/a/b.c", true},
		{"lcov", "/other/*", "src/a.c", false},
		{"lcov", "src/*.c", "src/a/b.c", true}, // * crosses slashes in lcov
		{"lcov", "a?b", "a/b.c", true},
		{"coveragepy", "skip_*.py", "src/skip_a.py", true},
		{"coveragepy", "src/gen/*", "src/gen/a.py", true},
		{"coveragepy", "src/gen/*", "src/gen/b/c.py", true}, // a trailing /* is anything below
		{"coveragepy", "src/*.py", "src/b/c.py", false},     // any other * stays in its directory
		{"coveragepy", "*/migrations/*", "app/x/migrations/0001.py", true},
		{"coveragepy", "*/migrations/*", "migrations/0001.py", true}, // a leading */ is any prefix, also none
		{"coveragepy", "*/migrations/*", "app/migrationsx/a.py", false},
		{"coveragepy", "**/gen/*", "a/gen/b/c.py", true},
		{"coveragepy", "/w/app/src/*", "src/a.py", true},
		{"coveragepy", "src\\gen\\*", "src/gen/a.py", true},
		{"minimatch", "src/**/*.js", "src/a.js", true},
		{"minimatch", "src/**/*.js", "src/x/y.js", true},
		{"minimatch", "./src/*.js", "src/a.js", true},
		{"minimatch", "src/*.js", "src/x/a.js", false},
		{"minimatch", "a?.js", "ab.js", true},
		{"regex", "src/", "src/a.cpp", true},
		{"regex", "^/w/app/lib/", "lib/a.cpp", true},
		{"regex", "^lib/", "src/lib/a.cpp", false},
		{"regex", "src/", "lib/src/b.cpp", false}, // re.match: anchored at the start
		{"regex", "build/", "src/build/gen.cpp", false},
		{"regex", "a|lib/", "src/lib/a.cpp", false}, // the anchor holds for every alternative
		{"regex", "/w/app/lib/", "lib/a.cpp", true},
		{"dir", "src", "src/a.php", true},
		{"dir", "src", "src", true},
		{"dir", "./src/", "src/a.php", true},
		{"dir", "src", "srcx/a.php", false},
		{"dir", ".", "a.py", true}, // coverage.py's source = . names the whole root
	} {
		m, err := dialects()[row.dialect](row.pattern, root)
		if err != nil {
			t.Errorf("%s %q: %v", row.dialect, row.pattern, err)
			continue
		}
		if got := m(row.rel); got != row.want {
			t.Errorf("%s %q on %q: %v", row.dialect, row.pattern, row.rel, got)
		}
	}
}

func TestDialectsReadGlobSyntaxAndRootsLikeTheirTools(t *testing.T) {
	for _, row := range []struct {
		dialect, pattern, rel, root string
		want                        bool
	}{
		{"minimatch", "**/*.{spec,test}.js", "a/b.spec.js", "/w", true},
		{"minimatch", "**/*.{spec,test}.js", "a/b.test.js", "/w", true},
		{"minimatch", "**/*.{spec,test}.js", "a/b.other.js", "/w", false},
		{"minimatch", "{a,{b,c}}/x.js", "c/x.js", "/w", true},
		{"coveragepy", "[ab]*.py", "src/a1.py", "/w", true},
		{"coveragepy", "[ab]*.py", "src/c1.py", "/w", false},
		{"coveragepy", "[!ab]*.py", "src/c1.py", "/w", true},
		{"coveragepy", "[!ab]*.py", "src/a1.py", "/w", false},
		{"coveragepy", "[]a]*.py", "]x.py", "/w", true},
		{"coveragepy", "[[a]x.py", "[x.py", "/w", true},
		{"coveragepy", "[!]a]x.py", "bx.py", "/w", true},
		{"coveragepy", "[!]a]x.py", "]x.py", "/w", false},
		{"coveragepy", "[^]a]x.py", "bx.py", "/w", true},
		{"coveragepy", "[^ab]*.py", "c1.py", "/w", true},
		{"coveragepy", "[^ab]*.py", "a1.py", "/w", false},
		{"coveragepy", "[]]x.py", "]x.py", "/w", true},
		{"coveragepy", "c:/src/*", "src/a.py", "C:/", true},
		{"coveragepy", "skip_*.py", "src/skip_a.py", "C:/w", true},
		{"coveragepy", "a*", "ab", "C:/wxyz", true},
		{"coveragepy", "x.py", "x.py", "/", true},
		{"coveragepy", "/w/a/*", "a/x.py", "/W", false},        // only drive letters compare without case
		{"coveragepy", "src[!x]a.py", "src/a.py", "/w", false}, // a class never matches a slash
		{"coveragepy", "{a,b}.py", "{a,b}.py", "/w", true},     // braces are literal outside minimatch
		{"minimatch", "[ab].js", "a.js", "/w", true},
		{"nano", "res://[ab].gd", "a.gd", "/w", true},
		{"lcov", "[ab].c", "[ab].c", "/w", true}, // lcov escapes brackets
		{"lcov", "[ab].c", "a.c", "/w", false},
		{"nano", "res://src/**/x", "src/x", "/w", false}, // ** is a plain .* there
		{"nano", "res://src/**/x", "src/a/x", "/w", true},
		{"minimatch", "src/**/x", "src/x", "/w", true},
		{"lcov", "src/**/x", "src/x", "/w", false},
		{"lcov", "C:/src/*", "src/a.c", "C:/", true},
		{"minimatch", "C:/src/*.js", "src/a.js", "C:/", true},
		{"coveragepy", "c:/w/src/*", "src/a.py", "C:/w", true},
		{"dir", "/w/app", "a.py", "/w/app", true},
		{"dir", "c:\\w", "a.py", "C:/w", true},
		{"lcov", "src\\*.c", "src/a.c", "/w", true},
		{"regex", "x\\.c$", "x.c", "/w", true}, // a backslash stays a regexp escape
		{"regex", "x\\.c$", "xyc", "/w", false},
	} {
		m, err := dialects()[row.dialect](row.pattern, row.root)
		if err != nil {
			t.Errorf("%s %q: %v", row.dialect, row.pattern, err)
			continue
		}
		if got := m(row.rel); got != row.want {
			t.Errorf("%s %q on %q under %q: %v", row.dialect, row.pattern, row.rel, row.root, got)
		}
	}
}

func TestDialectsRefuseWhatTheyCannotRead(t *testing.T) {
	for _, row := range []struct{ dialect, pattern string }{
		{"regex", "("},
		{"minimatch", "!src/**"},
		{"coveragepy", "src/["},
		{"coveragepy", "src/[!"},
		{"minimatch", "src/{a,b"},
		{"minimatch", "src/[ab"},
		{"minimatch", "src/{a,[b}"},
		{"coveragepy", "src/[ab"},
		{"nano", "res://[ab"},
		{"coveragepy", "[z-a]"},
	} {
		if _, err := dialects()[row.dialect](row.pattern, "/w"); err == nil {
			t.Errorf("%s %q: no error", row.dialect, row.pattern)
		}
	}
}

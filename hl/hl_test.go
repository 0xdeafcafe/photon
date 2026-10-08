package hl

import (
	"testing"
)

type run struct {
	text string
	c    Class
}

// lines lexes src line by line in lang, carrying state, and returns each
// line's runs. It fails the test if a line's runs do not tile it exactly.
func lines(t *testing.T, lang string, src ...string) [][]run {
	t.Helper()
	l := For(lang)
	if l == nil {
		t.Fatalf("For(%q) = nil", lang)
	}
	var st State
	var all [][]run
	for _, s := range src {
		var rs []run
		pos := 0
		l.Line(&st, s, func(i, j int, c Class) {
			if i != pos || j <= i || j > len(s) {
				t.Fatalf("%q: run [%d,%d) after %d", s, i, j, pos)
			}
			rs = append(rs, run{s[i:j], c})
			pos = j
		})
		if pos != len(s) {
			t.Fatalf("%q: runs end at %d of %d", s, pos, len(s))
		}
		all = append(all, rs)
	}
	return all
}

func has(rs []run, text string, c Class) bool {
	for _, r := range rs {
		if r.text == text && r.c == c {
			return true
		}
	}
	return false
}

func TestClasses(t *testing.T) {
	tests := []struct {
		name string
		lang string
		src  string
		text string
		want Class
	}{
		{"go keyword", "go", `func main() {`, "func", Keyword},
		{"go call", "go", `func main() {`, "main", Func},
		{"go call site", "go", `	fmt.Println("hi")`, "Println", Func},
		{"go string", "go", `x := "hello"`, `"hello"`, String},
		{"go number", "go", `x := 42`, "42", Number},
		{"go constant", "go", `return nil`, "nil", Number},
		{"go type", "go", `var x int`, "int", Type},
		{"go comment", "go", `x := 1 // note`, "// note", Comment},
		{"go inline block", "go", `a /* b */ c`, "/* b */", Comment},
		{"py comment", "py", `x = 1  # note`, "# note", Comment},
		{"py keyword", "py", `def f(x):`, "def", Keyword},
		{"py hash in word", "py", `a#b`, "a#b", Plain},
		{"json key", "json", `  "name": "loafer"`, `"name"`, Func},
		{"json value", "json", `  "name": "loafer"`, `"loafer"`, String},
		{"json constant", "json", `  "ok": true`, "true", Number},
		{"sh variable", "sh", `echo "$HOME" $USER`, "$USER", Func},
		{"sh comment", "sh", `ls # list`, "# list", Comment},
		{"sh keyword", "sh", `if true; then`, "if", Keyword},
		{"yaml key", "yaml", `name: loafer`, "name", Func},
		{"md heading", "md", `## Title`, "## Title", Keyword},
		{"md code", "md", "use `go test` now", "`go test`", String},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := lines(t, tt.lang, tt.src)[0]
			if !has(rs, tt.text, tt.want) {
				t.Errorf("want %q as %d in %v", tt.text, tt.want, rs)
			}
		})
	}
}

func TestBlockCommentSpansLines(t *testing.T) {
	got := lines(t, "go", "x := 1 /* start", "still comment", "end */ y := 2")
	if !has(got[0], "/* start", Comment) {
		t.Errorf("line 1: %v", got[0])
	}
	if len(got[1]) != 1 || got[1][0] != (run{"still comment", Comment}) {
		t.Errorf("line 2: %v", got[1])
	}
	if !has(got[2], "end */", Comment) || !has(got[2], "2", Number) {
		t.Errorf("line 3: %v", got[2])
	}
}

func TestMultiLineString(t *testing.T) {
	got := lines(t, "go", "s := `abc", "def`; x := 1")
	if !has(got[1], "def`", String) || !has(got[1], "1", Number) {
		t.Errorf("line 2: %v", got[1])
	}
	got = lines(t, "py", `s = """abc`, "def", `"""`, "x = 1")
	if !has(got[1], "def", String) || !has(got[3], "1", Number) {
		t.Errorf("python triple: %v", got)
	}
}

func TestMarkdownFence(t *testing.T) {
	got := lines(t, "md", "text", "```go", "func f() {}", "```", "func")
	if !has(got[2], "func", Keyword) {
		t.Errorf("fenced go: %v", got[2])
	}
	if !has(got[1], "```go", Comment) {
		t.Errorf("fence line: %v", got[1])
	}
	if has(got[4], "func", Keyword) {
		t.Errorf("after fence: %v", got[4])
	}
}

func TestFor(t *testing.T) {
	same := map[string][]string{
		"go":   {"golang", "go", "main.go", "GO", " go ", "go.mod", "/a/b/c.go"},
		"py":   {"py", "python", "python3", "x/y.py"},
		"sh":   {"bash", "sh", "Makefile", "script.zsh"},
		"js":   {"ts", "tsx", "javascript", "a.mjs"},
		"json": {"jsonc", "x.json"},
	}
	for want, names := range same {
		w := For(want)
		if w == nil {
			t.Fatalf("For(%q) = nil", want)
		}
		for _, n := range names {
			if got := For(n); got != w {
				t.Errorf("For(%q) = %p, want For(%q) = %p", n, got, want, w)
			}
		}
	}
	for _, n := range []string{"unknownlang", "", "README", "x.unknown"} {
		if For(n) != nil {
			t.Errorf("For(%q) != nil", n)
		}
	}
}

// TestTiles checks that the runs cover each line exactly, in order, in every
// language, for awkward lines as well as ordinary ones.
func TestTiles(t *testing.T) {
	samples := []string{
		"", " ", "\t", "func main() {", `x := "unterminated`, `x := 'a'`, "s := `raw",
		`"esc \" ape" \`, "/* open", "close */ x", "// c", "# c", "-- c", "$", "${", "${x}",
		"1.5e10 0x1F 3.", "héllo wörld → 世界", "a-b-c", "key: value", "- item: 1",
		"`tick", "```", "## h", "> q", "| a | b |", "|---|---|", "[t](u) **b** `c`", "[", "[x]", "**",
		"1. one", "\"\"\"", "'''x", `'it's`, "\\", `"\`,
	}
	for name, l := range langByName {
		var st State
		for _, s := range samples {
			pos := 0
			l.Line(&st, s, func(i, j int, c Class) {
				if i != pos || j <= i || j > len(s) {
					t.Fatalf("%s %q: run [%d,%d) after %d", name, s, i, j, pos)
				}
				pos = j
			})
			if pos != len(s) {
				t.Fatalf("%s %q: runs end at %d", name, s, pos)
			}
		}
	}
	var st State
	var nilLang *Lang
	nilLang.Line(&st, "abc", func(i, j int, c Class) {
		if i != 0 || j != 3 || c != Plain {
			t.Errorf("nil lang: [%d,%d) %d", i, j, c)
		}
	})
}

func BenchmarkLine(b *testing.B) {
	const s = `	if err := srv.Handle(ctx, "message", 42); err != nil { // report it upstream`
	l := For("go")
	var st State
	n := 0
	out := func(i, j int, c Class) { n += j - i }
	b.ReportAllocs()
	for b.Loop() {
		st = State{}
		l.Line(&st, s, out)
	}
	if n == 0 {
		b.Fatal("no runs")
	}
}

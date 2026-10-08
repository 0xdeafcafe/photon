// Package hl is a small syntax highlighter, ported from rush's
// (convo/highlight.go) without the drawing: one pass over a line's bytes, no
// regular expressions, and a word's class looked up without allocating. It
// reports classes, not colours, so the caller decides how each is drawn.
package hl

import (
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Class is what a run of source is.
type Class uint8

// The classes a run can have.
const (
	Plain   Class = iota // anything not below
	Keyword              // a keyword
	String               // a string, or inline code in Markdown
	Number               // a number or a constant such as true or nil
	Func                 // a call, a key, a variable
	Type                 // a type
	Comment              // a comment
)

// Lang is what the highlighter knows of a language.
type Lang struct {
	words      map[string]byte // 'k' keyword, 't' type, 'c' constant
	line       string          // a line comment's start: //, #, --
	line2      string          // a second one, as # in PHP
	open, shut string          // a block comment's ends
	quotes     string          // characters that quote a string on one line
	multi      byte            // a quote that may run over lines: ` in Go and JS
	triple     bool            // """ and ''' strings, over lines
	caps       bool            // a capitalised word is a type
	vars       bool            // $name is a variable
	keys       bool            // a key before : (JSON, YAML, TOML's =)
	md         bool            // Markdown, drawn by its own rules
}

func set(class byte, ws string, into map[string]byte) map[string]byte {
	if into == nil {
		into = map[string]byte{}
	}
	for _, w := range strings.Fields(ws) {
		into[w] = class
	}
	return into
}

func words(kw, types, consts string) map[string]byte {
	m := set('k', kw, nil)
	set('t', types, m)
	return set('c', consts, m)
}

var (
	langGo = &Lang{words: words(
		"break case chan const continue default defer else fallthrough for func go goto if import interface map package range return select struct switch type var",
		"bool byte complex64 complex128 error float32 float64 int int8 int16 int32 int64 rune string uint uint8 uint16 uint32 uint64 uintptr any comparable",
		"true false nil iota"),
		line: "//", open: "/*", shut: "*/", quotes: `"'`, multi: '`', caps: true}
	langJS = &Lang{words: words(
		"async await break case catch class const continue debugger default delete do else export extends finally for from function if import in instanceof let new of return static super switch this throw try typeof var void while with yield as type interface enum implements declare readonly keyof satisfies namespace abstract private protected public",
		"string number boolean any unknown never object void bigint symbol Promise Array Record Map Set",
		"true false null undefined NaN Infinity"),
		line: "//", open: "/*", shut: "*/", quotes: `"'`, multi: '`', caps: true}
	langPy = &Lang{words: words(
		"and as assert async await break class continue def del elif else except finally for from global if import in is lambda nonlocal not or pass raise return try while with yield match case",
		"int str float bool list dict set tuple bytes object type self cls",
		"True False None"),
		line: "#", quotes: `"'`, triple: true}
	langRust = &Lang{words: words(
		"as async await break const continue crate dyn else enum extern fn for if impl in let loop match mod move mut pub ref return static struct super trait type unsafe use where while",
		"i8 i16 i32 i64 i128 isize u8 u16 u32 u64 u128 usize f32 f64 bool char str String Vec Option Result Box Self self",
		"true false None Some Ok Err"),
		line: "//", open: "/*", shut: "*/", quotes: `"`, caps: true}
	langSh = &Lang{words: words(
		"if then else elif fi for in do done while until case esac function return local export readonly unset set shift exit break continue source eval exec trap",
		"echo printf cd test read declare",
		"true false"),
		line: "#", quotes: `"'`, vars: true}
	langJSON = &Lang{words: words("", "", "true false null"), quotes: `"`, keys: true, line: "//"}
	langYAML = &Lang{words: words("", "", "true false null yes no on off ~"), line: "#", quotes: `"'`, keys: true}
	langTOML = &Lang{words: words("", "", "true false"), line: "#", quotes: `"'`, keys: true, triple: true}
	langCSS  = &Lang{words: words("important media import supports keyframes font-face", "", "inherit initial unset none auto"), open: "/*", shut: "*/", quotes: `"'`, line: "//"}
	langSQL  = &Lang{words: words(
		"select from where and or not in is null as join left right inner outer full on group by order having limit offset insert into values update set delete create table index view drop alter add column primary key foreign references unique default distinct union all case when then else end with returning exists between like asc desc begin commit rollback if SELECT FROM WHERE AND OR NOT IN IS NULL AS JOIN LEFT RIGHT INNER OUTER ON GROUP BY ORDER HAVING LIMIT OFFSET INSERT INTO VALUES UPDATE SET DELETE CREATE TABLE INDEX VIEW DROP ALTER ADD COLUMN PRIMARY KEY FOREIGN REFERENCES UNIQUE DEFAULT DISTINCT UNION ALL CASE WHEN THEN ELSE END WITH RETURNING EXISTS BETWEEN LIKE ASC DESC BEGIN COMMIT ROLLBACK IF",
		"int integer bigint text varchar char boolean timestamp timestamptz date jsonb json uuid serial numeric real INT INTEGER BIGINT TEXT VARCHAR BOOLEAN TIMESTAMP DATE JSONB UUID SERIAL NUMERIC",
		"true false TRUE FALSE"),
		line: "--", open: "/*", shut: "*/", quotes: `'"`}
	langRuby = &Lang{words: words(
		"alias and begin break case class def defined do else elsif end ensure for if in module next not or redo rescue retry return self super then undef unless until when while yield require require_relative attr_accessor attr_reader",
		"", "true false nil"),
		line: "#", quotes: `"'`, caps: true}
	// langC covers the C family: C, C++, C#, Java, Kotlin, Swift, Dart.
	langC = &Lang{words: words(
		"auto break case catch class const continue default delete do else enum extern final finally for fun func goto if implements import in include interface let namespace new override package private protected public return sizeof static struct switch template this throw throws try typedef typename union using val var virtual void volatile when while guard defer extension protocol init self super import object data sealed suspend",
		"int long short char float double bool boolean byte string String unsigned signed size_t Int Long Double Float Bool Boolean",
		"true false null nullptr nil NULL"),
		line: "//", open: "/*", shut: "*/", quotes: `"'`, caps: true}
	langLua = &Lang{words: words("and break do else elseif end for function goto if in local not or repeat return then until while", "", "true false nil"), line: "--", quotes: `"'`}
	langPHP = &Lang{words: words(
		"abstract and array as break case catch class clone const continue declare default do echo else elseif empty endif extends final finally fn for foreach function global if implements include interface isset list match namespace new or print private protected public readonly require return static switch throw trait try unset use var while yield",
		"int string bool float array object mixed void", "true false null TRUE FALSE NULL"),
		line: "//", line2: "#", open: "/*", shut: "*/", quotes: `"'`, vars: true, caps: true}
	langMD = &Lang{md: true}
)

// langByName is a language by a fence's tag or a file's extension.
var langByName = map[string]*Lang{
	"go": langGo, "golang": langGo,
	"js": langJS, "javascript": langJS, "jsx": langJS, "mjs": langJS, "cjs": langJS,
	"ts": langJS, "typescript": langJS, "tsx": langJS, "mts": langJS, "cts": langJS,
	"py": langPy, "python": langPy, "python3": langPy, "pyi": langPy,
	"rs": langRust, "rust": langRust,
	"sh": langSh, "bash": langSh, "zsh": langSh, "shell": langSh, "console": langSh, "fish": langSh,
	"json": langJSON, "jsonc": langJSON, "json5": langJSON, "jsonl": langJSON,
	"yaml": langYAML, "yml": langYAML,
	"toml": langTOML, "ini": langTOML, "env": langTOML,
	"css": langCSS, "scss": langCSS, "less": langCSS,
	"sql": langSQL, "psql": langSQL,
	"rb": langRuby, "ruby": langRuby,
	"c": langC, "h": langC, "cpp": langC, "cc": langC, "hpp": langC, "c++": langC, "cs": langC, "csharp": langC,
	"java": langC, "kt": langC, "kts": langC, "kotlin": langC, "swift": langC, "dart": langC, "scala": langC, "zig": langC,
	"lua": langLua, "php": langPHP,
	"md": langMD, "markdown": langMD, "mdx": langMD,
}

// For is the language of a fence tag, a file name, an extension or an
// interpreter; nil when there's none to highlight.
func For(name string) *Lang {
	name = strings.ToLower(strings.TrimSpace(name))
	if l, ok := langByName[name]; ok {
		return l
	}
	switch filepath.Base(name) {
	case "makefile", "dockerfile", ".bashrc", ".zshrc", ".profile":
		return langSh
	case "go.mod", "go.sum":
		return langGo
	}
	if ext := filepath.Ext(name); ext != "" {
		return langByName[ext[1:]]
	}
	return nil
}

// State carries a block comment, a multi-line string or a Markdown fence from
// one line to the next. The zero value is the start of a file.
type State struct {
	block bool
	str   string // what closes the string still open
	fence *Lang  // a Markdown fence's language, while in it
	in    inner  // what the fenced code left open
}

// Fence is the language of the Markdown fence st is in: nil outside one, or
// in one with no language.
func (st State) Fence() *Lang { return st.fence }

// inner is what a fenced block's code carries from line to line: a value,
// so copies of a state never share it.
type inner struct {
	block bool
	str   string
}

// Line lexes one line of source in l, going on from st and leaving st for the
// next line. It calls out for each run [i,j) of s, as byte offsets, in order
// and without gaps, so the runs cover all of s. Adjacent runs of one class are
// merged. A nil l is all Plain.
func (l *Lang) Line(st *State, s string, out func(i, j int, c Class)) {
	// Merge adjacent runs of a class, holding the last back until the next
	// differs.
	var pi, pj int
	pc := Plain
	emit := func(i, j int, c Class) {
		if i >= j {
			return
		}
		if pj > pi && c == pc && i == pj {
			pj = j
			return
		}
		if pj > pi {
			out(pi, pj, pc)
		}
		pi, pj, pc = i, j, c
	}
	if l == nil {
		emit(0, len(s), Plain)
	} else {
		lex(l, st, s, emit)
	}
	if pj > pi {
		out(pi, pj, pc)
	}
}

// lex lexes a line of code in l through out, going on from st.
func lex(l *Lang, st *State, s string, out func(i, j int, c Class)) {
	if l.md {
		markdown(st, s, out)
		return
	}
	i := 0
	// What the last line left open.
	if st.block {
		if k := strings.Index(s, l.shut); k >= 0 {
			out(0, k+len(l.shut), Comment)
			i, st.block = k+len(l.shut), false
		} else {
			out(0, len(s), Comment)
			return
		}
	}
	if st.str != "" {
		if k := strings.Index(s, st.str); k >= 0 {
			out(0, k+len(st.str), String)
			i, st.str = k+len(st.str), ""
		} else {
			out(0, len(s), String)
			return
		}
	}
	for i < len(s) {
		c := s[i]
		switch {
		case l.line != "" && strings.HasPrefix(s[i:], l.line) && (l.line != "#" || i == 0 || s[i-1] == ' ' || s[i-1] == '\t'),
			l.line2 != "" && strings.HasPrefix(s[i:], l.line2):
			out(i, len(s), Comment)
			i = len(s)
		case l.open != "" && strings.HasPrefix(s[i:], l.open):
			if k := strings.Index(s[i+len(l.open):], l.shut); k >= 0 {
				end := i + len(l.open) + k + len(l.shut)
				out(i, end, Comment)
				i = end
			} else {
				out(i, len(s), Comment)
				st.block, i = true, len(s)
			}
		case l.triple && (strings.HasPrefix(s[i:], `"""`) || strings.HasPrefix(s[i:], `'''`)):
			q := s[i : i+3]
			if k := strings.Index(s[i+3:], q); k >= 0 {
				out(i, i+3+k+3, String)
				i += 3 + k + 3
			} else {
				out(i, len(s), String)
				st.str, i = q, len(s)
			}
		case l.multi != 0 && c == l.multi:
			if k := strings.IndexByte(s[i+1:], c); k >= 0 {
				out(i, i+k+2, String)
				i += k + 2
			} else {
				out(i, len(s), String)
				st.str, i = string(c), len(s)
			}
		case strings.IndexByte(l.quotes, c) >= 0 && !(c == '\'' && i > 0 && isWord(s[i-1]) && l != langSh):
			j := i + 1
			for j < len(s) && s[j] != c {
				if s[j] == '\\' {
					j++
				}
				j++
			}
			j = min(j+1, len(s))
			col := String
			if l.keys && keyAfter(s, j) {
				col = Func
			}
			out(i, j, col)
			i = j
		case l.vars && c == '$' && i+1 < len(s) && (isWord(s[i+1]) || s[i+1] == '{'):
			j := i + 1
			if s[j] == '{' {
				if k := strings.IndexByte(s[j:], '}'); k >= 0 {
					j += k + 1
				}
			} else {
				for j < len(s) && isWord(s[j]) {
					j++
				}
			}
			out(i, j, Func)
			i = j
		case c >= '0' && c <= '9' && (i == 0 || !isWord(s[i-1])):
			j := i + 1
			for j < len(s) && (isWord(s[j]) || s[j] == '.' && j+1 < len(s) && s[j+1] >= '0' && s[j+1] <= '9') {
				j++
			}
			out(i, j, Number)
			i = j
		case isWord(c) || c >= utf8.RuneSelf:
			j := i + 1
			for j < len(s) && (isWord(s[j]) || s[j] >= utf8.RuneSelf || l == langCSS && s[j] == '-') {
				j++
			}
			w := s[i:j]
			col := Plain
			switch l.words[w] {
			case 'k':
				col = Keyword
			case 't':
				col = Type
			case 'c':
				col = Number
			default:
				switch {
				case l.keys && keyAfter(s, j) && lineStartOrIndent(s, i), next(s, j) == '(':
					col = Func
				case l.caps && c >= 'A' && c <= 'Z' && len(w) > 1:
					col = Type
				}
			}
			out(i, j, col)
			i = j
		default:
			j := i + 1
			for j < len(s) && !isWord(s[j]) && s[j] < utf8.RuneSelf && s[j] != '"' && s[j] != '\'' && s[j] != '`' && s[j] != '/' && s[j] != '#' && s[j] != '$' && s[j] != '-' {
				j++
			}
			out(i, j, Plain)
			i = j
		}
	}
}

// markdown lexes a line of Markdown: a heading, a quote, a rule and a fenced
// block whole; a list's mark and a table's pipes quiet; inline code, bold and
// links in the line. st carries a fence from line to line.
func markdown(st *State, s string, out func(i, j int, c Class)) {
	t := strings.TrimLeft(s, " \t")
	ind := len(s) - len(t)
	if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
		switch fence := t[:3]; st.str {
		case "":
			st.str, st.fence, st.in = fence, nil, inner{}
			if tag := strings.Fields(strings.Trim(t, "`~")); len(tag) > 0 {
				st.fence = For(tag[0])
			}
		case fence:
			st.str, st.fence = "", nil
		}
		out(0, len(s), Comment)
		return
	}
	if st.str != "" && st.fence != nil {
		// A fenced block in its own language, from where its last line left it.
		in := State{block: st.in.block, str: st.in.str}
		lex(st.fence, &in, s, out)
		st.in = inner{in.block, in.str}
		return
	}
	if st.str != "" {
		out(0, len(s), String)
		return
	}
	h := 0
	for h < len(t) && t[h] == '#' {
		h++
	}
	switch {
	case t == "":
		out(0, len(s), Plain)
		return
	case h > 0 && h <= 6 && (h == len(t) || t[h] == ' '):
		out(0, len(s), Keyword)
		return
	case t[0] == '>':
		out(0, len(s), Comment)
		return
	case len(t) >= 3 && strings.Trim(t, string(t[0])+" ") == "" && strings.IndexByte("-*_=", t[0]) >= 0,
		t[0] == '|' && strings.Trim(t, "|-: ") == "":
		out(0, len(s), Comment) // a rule, a heading's underline, a table's
		return
	}
	i := 0
	if m := listMark(t); m > 0 {
		out(0, ind+m, Number)
		i = ind + m
	}
	table := t[0] == '|'
	for i < len(s) {
		c := s[i]
		switch {
		case c == '`':
			n := 1
			for i+n < len(s) && s[i+n] == '`' {
				n++
			}
			j := i + n
			if k := strings.Index(s[j:], s[i:j]); k >= 0 {
				j += k + n
			}
			out(i, j, String)
			i = j
		case c == '*' && i+1 < len(s) && s[i+1] == '*' && strings.Contains(s[i+2:], "**"):
			j := i + 2 + strings.Index(s[i+2:], "**") + 2
			out(i, j, Type)
			i = j
		case c == '[' && mdLink(s[i:]) > 0:
			k := strings.IndexByte(s[i:], ']')
			j := i + mdLink(s[i:])
			out(i, i+1, Comment)
			out(i+1, i+k, Func)
			out(i+k, j, Comment)
			i = j
		case c == '|' && table:
			out(i, i+1, Comment)
			i++
		default:
			j := i + 1
			for j < len(s) && strings.IndexByte("`*[|", s[j]) < 0 {
				j++
			}
			out(i, j, Plain)
			i = j
		}
	}
}

// listMark is how long a list item's mark is with its space: "- ", "* ",
// "1. ", "2) "; 0 when t isn't one.
func listMark(t string) int {
	if len(t) >= 2 && strings.IndexByte("-*+", t[0]) >= 0 && t[1] == ' ' {
		return 2
	}
	n := 0
	for n < len(t) && n < 9 && t[n] >= '0' && t[n] <= '9' {
		n++
	}
	if n > 0 && n+1 < len(t) && (t[n] == '.' || t[n] == ')') && t[n+1] == ' ' {
		return n + 2
	}
	return 0
}

// mdLink is how long the [text](url) that s starts with is, 0 when it
// doesn't start one.
func mdLink(s string) int {
	k := strings.IndexByte(s, ']')
	if k < 1 || !strings.HasPrefix(s[k:], "](") {
		return 0
	}
	e := strings.IndexByte(s[k:], ')')
	if e < 0 {
		return 0
	}
	return k + e + 1
}

func isWord(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// next is the first byte after s[j:]'s spaces, or 0.
func next(s string, j int) byte {
	for j < len(s) && s[j] == ' ' {
		j++
	}
	if j < len(s) {
		return s[j]
	}
	return 0
}

// keyAfter is whether what ends at j is a key: a : or = follows.
func keyAfter(s string, j int) bool {
	n := next(s, j)
	return n == ':' || n == '='
}

// lineStartOrIndent is whether s[:i] is only indentation or a list's dash.
func lineStartOrIndent(s string, i int) bool {
	for k := 0; k < i; k++ {
		if s[k] != ' ' && s[k] != '\t' && s[k] != '-' {
			return false
		}
	}
	return true
}

package canvas

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/0xdeafcafe/photon/cellw"
)

// Wrap breaks r into rows at most w cells wide, at spaces where it can
// and through a word only when the word alone is wider than w. Spaces a
// row ends on are dropped.
func Wrap(r Row, w int) []Row {
	if w < 1 {
		w = 1
	}
	if r.Width() <= w {
		return []Row{r}
	}
	var out []Row
	var line Row
	have := 0
	put := func(text string, st Style, tw int) {
		if n := len(line); n > 0 && line[n-1].St == st {
			line[n-1].Text += text
			line[n-1].W += tw
		} else {
			line = append(line, Seg{text, st, tw})
		}
		have += tw
	}
	brk := func() {
		// Drop the spaces the row ends on.
		for n := len(line); n > 0; n = len(line) {
			t := strings.TrimRight(line[n-1].Text, " ")
			have -= line[n-1].W - cellw.String(t)
			if t != "" {
				line[n-1].Text, line[n-1].W = t, cellw.String(t)
				break
			}
			line = line[:n-1]
		}
		out = append(out, line)
		line, have = nil, 0
	}
	for _, s := range r {
		for word := range words(s.Text) {
			ww := cellw.String(word)
			if have+ww <= w {
				put(word, s.St, ww)
				continue
			}
			if strings.TrimSpace(word) == "" {
				brk() // a space where the row is full: break there, drop it
				continue
			}
			if have > 0 {
				brk()
			}
			for ww > w { // longer than a row: cut it through
				head := ansi.Truncate(word, w-have, "")
				put(head, s.St, cellw.String(head))
				brk()
				word = ansi.TruncateLeft(word, cellw.String(head), "")
				ww = cellw.String(word)
			}
			put(word, s.St, ww)
		}
	}
	if len(line) > 0 || len(out) == 0 {
		out = append(out, line)
	}
	return out
}

// words yields s as words and runs of spaces, in order.
func words(s string) func(func(string) bool) {
	return func(yield func(string) bool) {
		for s != "" {
			i := strings.IndexByte(s, ' ')
			switch {
			case i < 0:
				yield(s)
				return
			case i == 0:
				j := len(s) - len(strings.TrimLeft(s, " "))
				if !yield(s[:j]) {
					return
				}
				s = s[j:]
			default:
				if !yield(s[:i]) {
					return
				}
				s = s[i:]
			}
		}
	}
}

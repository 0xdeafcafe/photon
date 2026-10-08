// Package canvas keeps what's on screen as rows of styled pieces until the
// very end, when Emit writes ANSI. Panes are rows of an exact width put
// side by side; an overlay is spliced in by column, and what's behind it
// dimmed by changing style, so nothing is ever parsed back out of ANSI.
package canvas

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/0xdeafcafe/photon/cellw"
	"github.com/0xdeafcafe/photon/theme"
)

// Attr is a style's flags.
type Attr uint8

const (
	HasFG Attr = 1 << iota
	HasBG
	Bold
	Faint
	Italic
	Underline
)

// Style is how a piece is drawn. The zero Style is the terminal's own.
type Style struct {
	FG, BG theme.RGB
	A      Attr
	Link   string // OSC 8 target, if any
}

func (s Style) Fg(c theme.RGB) Style { s.FG, s.A = c, s.A|HasFG; return s }
func (s Style) Bg(c theme.RGB) Style { s.BG, s.A = c, s.A|HasBG; return s }
func (s Style) With(a Attr) Style    { s.A |= a; return s }

// Seg is a run of text in one style, and its width in cells.
type Seg struct {
	Text string
	St   Style
	W    int
}

// T makes a Seg, measuring it once.
func T(text string, st Style) Seg { return Seg{text, st, cellw.String(text)} }

// Row is one line of the screen.
type Row []Seg

func (r Row) Width() int {
	w := 0
	for _, s := range r {
		w += s.W
	}
	return w
}

// Fit cuts r to w cells (ending in "…" when it cuts) or pads it with
// spaces in fill, so it's exactly w wide.
func Fit(r Row, w int, fill Style) Row {
	have := r.Width()
	if have > w {
		r = Cut(r, w-1)
		r = append(r, Seg{"…", r.last(fill), 1})
		have = w
	}
	if have < w {
		r = append(r, Seg{strings.Repeat(" ", w-have), fill, w - have})
	}
	return r
}

func (r Row) last(def Style) Style {
	if len(r) == 0 {
		return def
	}
	return r[len(r)-1].St
}

// Cut keeps the first w cells of r. A wide character that would straddle
// the cut is dropped, and the row may come out a cell short.
func Cut(r Row, w int) Row {
	out := make(Row, 0, len(r))
	for _, s := range r {
		if w <= 0 {
			break
		}
		if s.W > w {
			t := ansi.Truncate(s.Text, w, "")
			out = append(out, Seg{t, s.St, cellw.String(t)})
			break
		}
		out = append(out, s)
		w -= s.W
	}
	return out
}

// Drop is r without its first w cells, padded with a space in its style
// where a wide character straddled the cut.
func Drop(r Row, w int) Row {
	out := make(Row, 0, len(r))
	for _, s := range r {
		switch {
		case w >= s.W:
			w -= s.W
		case w > 0:
			t := ansi.TruncateLeft(s.Text, w, "")
			tw := cellw.String(t)
			if tw > s.W-w { // a wide character straddles the cut: drop it too
				t = ansi.TruncateLeft(s.Text, w+1, "")
				tw = cellw.String(t)
			}
			if gap := s.W - w - tw; gap > 0 {
				t, tw = strings.Repeat(" ", gap)+t, tw+gap
			}
			out = append(out, Seg{t, s.St, tw})
			w = 0
		default:
			out = append(out, s)
		}
	}
	return out
}

// Splice puts over onto base starting at cell col.
func Splice(base Row, col int, over Row) Row {
	left := Fit(Cut(base, col), col, Style{})
	out := append(left[:len(left):len(left)], over...)
	return append(out, Drop(base, col+over.Width())...)
}

// Restyle draws every piece of r in st, as what's behind an overlay is.
func Restyle(r Row, st Style) Row {
	out := make(Row, len(r))
	for i, s := range r {
		s.St = st
		out[i] = s
	}
	return out
}

// Join puts rows of panes side by side; each pane's rows must already be
// its exact width.
func Join(panes ...[]Row) []Row {
	h := 0
	for _, p := range panes {
		h = max(h, len(p))
	}
	out := make([]Row, h)
	for i := range out {
		for _, p := range panes {
			if i < len(p) {
				out[i] = append(out[i], p[i]...)
			}
		}
	}
	return out
}

// Emit writes rows as ANSI into b, one line each, changing style only
// where it changes and resetting at each line's end.
func Emit(b *strings.Builder, rows []Row) {
	for i, r := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		cur, link := Style{}, ""
		for _, s := range r {
			if s.St.Link != link {
				if link != "" {
					b.WriteString("\x1b]8;;\x1b\\")
				}
				if link = s.St.Link; link != "" {
					b.WriteString("\x1b]8;;")
					b.WriteString(link)
					b.WriteString("\x1b\\")
				}
			}
			if s.St != cur {
				sgr(b, s.St)
				cur = s.St
			}
			b.WriteString(s.Text)
		}
		if link != "" {
			b.WriteString("\x1b]8;;\x1b\\")
		}
		if cur != (Style{}) {
			b.WriteString("\x1b[m")
		}
	}
}

func sgr(b *strings.Builder, s Style) {
	b.WriteString("\x1b[0")
	for _, f := range []struct {
		a    Attr
		code string
	}{{Bold, ";1"}, {Faint, ";2"}, {Italic, ";3"}, {Underline, ";4"}} {
		if s.A&f.a != 0 {
			b.WriteString(f.code)
		}
	}
	if s.A&HasFG != 0 {
		rgb(b, ";38;2;", s.FG)
	}
	if s.A&HasBG != 0 {
		rgb(b, ";48;2;", s.BG)
	}
	b.WriteByte('m')
}

func rgb(b *strings.Builder, lead string, c theme.RGB) {
	var n [12]byte
	b.WriteString(lead)
	b.Write(strconv.AppendUint(n[:0], uint64(c.R), 10))
	b.WriteByte(';')
	b.Write(strconv.AppendUint(n[:0], uint64(c.G), 10))
	b.WriteByte(';')
	b.Write(strconv.AppendUint(n[:0], uint64(c.B), 10))
}

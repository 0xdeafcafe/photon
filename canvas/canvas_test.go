package canvas

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/0xdeafcafe/photon/theme"
)

func plain(r Row) string {
	var b strings.Builder
	Emit(&b, []Row{r})
	return ansi.Strip(b.String())
}

func TestFitCutDrop(t *testing.T) {
	r := Row{T("hello ", Style{}), T("wörld", Style{}.With(Bold))}
	for w, want := range map[int]string{5: "hell…", 11: "hello wörld", 14: "hello wörld   "} {
		if got := plain(Fit(r, w, Style{})); got != want || Fit(r, w, Style{}).Width() != w {
			t.Errorf("Fit %d = %q, want %q", w, got, want)
		}
	}
	wide := Row{T("a中b", Style{})} // 中 is two cells
	if got := plain(Cut(wide, 2)); got != "a" {
		t.Errorf("Cut through a wide char = %q", got)
	}
	if got := plain(Drop(wide, 2)); got != " b" || Drop(wide, 2).Width() != 2 {
		t.Errorf("Drop through a wide char = %q", got)
	}
}

func TestSplice(t *testing.T) {
	base := Row{T("0123456789", Style{})}
	got := Splice(base, 3, Row{T("XY", Style{})})
	if plain(got) != "012XY56789" || got.Width() != 10 {
		t.Fatalf("splice = %q", plain(got))
	}
	if plain(Splice(Row{T("ab", Style{})}, 4, Row{T("Z", Style{})})) != "ab  Z" {
		t.Fatal("splice past the end should pad")
	}
}

func TestEmitStylesOnlyOnChange(t *testing.T) {
	red := Style{}.Fg(theme.RGB{R: 255})
	var b strings.Builder
	Emit(&b, []Row{{T("a", red), T("b", red), T("c", Style{})}})
	if got := b.String(); got != "\x1b[0;38;2;255;0;0mab\x1b[0mc" {
		t.Fatalf("emit = %q", got)
	}
}

func BenchmarkEmit(b *testing.B) {
	st := Style{}.Fg(theme.RGB{R: 200, G: 190, B: 180}).Bg(theme.RGB{R: 17, G: 16, B: 14})
	row := Row{T("  # prod-alerts-page ", st), T("│", Style{}), T(strings.Repeat("message text ", 12), st.With(Bold))}
	rows := make([]Row, 60)
	for i := range rows {
		rows[i] = row
	}
	var sb strings.Builder
	for b.Loop() {
		sb.Reset()
		Emit(&sb, rows)
	}
}

func TestWrap(t *testing.T) {
	bold := Style{}.With(Bold)
	r := Row{T("the quick ", Style{}), T("brown fox", bold), T(" jumps over", Style{})}
	var got []string
	for _, l := range Wrap(r, 10) {
		got = append(got, plain(l))
		if l.Width() > 10 {
			t.Errorf("row %q is %d wide", plain(l), l.Width())
		}
	}
	if strings.Join(got, "|") != "the quick|brown fox|jumps over" {
		t.Fatalf("wrap = %q", got)
	}
	long := Wrap(Row{T("see https://example.com/a/very/long/path ok", Style{})}, 12)
	var parts []string
	for _, l := range long {
		parts = append(parts, plain(l))
	}
	if strings.Join(parts, "|") != "see|https://exam|ple.com/a/ve|ry/long/path|ok" {
		t.Fatalf("long word = %q", parts)
	}
}

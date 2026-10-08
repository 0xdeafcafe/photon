// Package fuzzy is rush's command bar matcher: each word of a query must
// appear in the text, its letters in order, a whole run and the start of
// a word scoring more and a gap less.
package fuzzy

import (
	"slices"
	"strings"
	"unicode"
)

// Match scores how well q matches s; at are the runes of s it matched.
func Match(q, s string) (score int, at []int, ok bool) {
	hay := []rune(strings.ToLower(s))
	orig := []rune(s)
	if len(hay) != len(orig) { // lowering changed the length: match as is
		hay = orig
	}
	for _, w := range strings.Fields(strings.ToLower(q)) {
		ws, wat, ok := fuzzyWord([]rune(w), hay, orig)
		if !ok {
			return 0, nil, false
		}
		score += ws
		at = append(at, wat...)
	}
	return score, at, true
}

func fuzzyWord(w, hay, orig []rune) (int, []int, bool) {
	if i := bestSub(w, hay, orig); i >= 0 {
		at := make([]int, len(w))
		for k := range w {
			at[k] = i + k
		}
		score := 100 + 10*len(w) - min(i, 30)
		if wordStart(hay, orig, i) {
			score += 40
		}
		return score, at, true
	}
	var at []int
	score, k, last := 0, 0, -2
	for i := 0; i < len(hay) && k < len(w); i++ {
		if hay[i] != w[k] {
			continue
		}
		switch {
		case i == last+1:
			score += 6
		case wordStart(hay, orig, i):
			score += 8
		default:
			score -= min(i-last, 6)
		}
		at = append(at, i)
		last = i
		k++
	}
	if k < len(w) {
		return 0, nil, false
	}
	return score, at, true
}

func bestSub(w, hay, orig []rune) int {
	first := -1
	for i := 0; i+len(w) <= len(hay); i++ {
		if !slices.Equal(hay[i:i+len(w)], w) {
			continue
		}
		if wordStart(hay, orig, i) {
			return i
		}
		if first < 0 {
			first = i
		}
	}
	return first
}

func wordStart(hay, orig []rune, i int) bool {
	if i == 0 {
		return true
	}
	p := hay[i-1]
	if !unicode.IsLetter(p) && !unicode.IsDigit(p) {
		return true
	}
	return unicode.IsLower(orig[i-1]) && unicode.IsUpper(orig[i])
}

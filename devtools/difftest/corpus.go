// Package difftest compares the collations of glibcollate with the real
// strcoll_l of the image their tables were taken from.
package difftest

import (
	"math/rand/v2"
	"strings"
	"unicode/utf8"
)

// Pair is one input to strcoll.
type Pair struct {
	A, B string
}

// Corpus returns the pairs to compare, generated from a fixed seed so that a
// failure can be reproduced. No string contains NUL, since strcoll cannot see
// past it.
func Corpus(seed uint64) []Pair {
	g := &gen{r: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))} //nolint:gosec // a fixed seed keeps failures reproducible
	var ps []Pair

	// Random ASCII up to length 8.
	ascii := g.alphabet(0x01, 0x7f)
	for range 150000 {
		ps = append(ps, Pair{g.str(ascii, 0, 8), g.str(ascii, 0, 8)})
	}

	// Punctuation and spaces in every position of short words.
	var punct []rune
	for c := rune(0x20); c < 0x7f; c++ {
		if !isAlnum(c) {
			punct = append(punct, c)
		}
	}
	var withPunct []string
	for _, w := range []string{"a", "ab", "aB", "Ab", "a1", "1a", "00a", "00A", "abc"} {
		for _, p := range punct {
			for i := 0; i <= len(w); i++ {
				withPunct = append(withPunct, w[:i]+string(p)+w[i:])
			}
		}
		withPunct = append(withPunct, w)
	}
	ps = append(ps, g.pairsWithin(withPunct, 100000)...)
	twoPunct := g.alphabetOf("ab -_.,'$" + string(punct))
	for range 30000 {
		ps = append(ps, Pair{g.str(twoPunct, 0, 6), g.str(twoPunct, 0, 6)})
	}

	// Digits mixed with letters, and mixed case.
	digits := g.alphabetOf("0123456789aAbBzZ")
	for range 30000 {
		ps = append(ps, Pair{g.str(digits, 1, 6), g.str(digits, 1, 6)})
	}
	words := []string{"apple", "Apple", "APPLE", "aPPle", "banana", "Banana", "resume", "Resume", "zebra", "Zebra"}
	for range 10000 {
		a := g.mixCase(words[g.r.IntN(len(words))])
		ps = append(ps, Pair{a, g.mixCase(a)})
	}

	// Latin-1, accented letters precomposed and combining.
	latin := append(g.alphabet(0xa0, 0x100), g.alphabetOf("aeiouAEIOUcnsyCNSY")...)
	combining := g.alphabet(0x300, 0x370)
	accented := func() string {
		var b strings.Builder
		for range 1 + g.r.IntN(6) {
			b.WriteRune(latin[g.r.IntN(len(latin))])
			if g.r.IntN(3) == 0 {
				b.WriteRune(combining[g.r.IntN(len(combining))])
			}
		}
		return b.String()
	}
	for range 50000 {
		ps = append(ps, Pair{accented(), accented()})
	}
	for _, p := range [][2]string{{"é", "é"}, {"Å", "Å"}, {"ñ", "ñ"}, {"résumé", "resume"}, {"Straße", "Strasse"}, {"cote", "côte"}, {"coté", "côte"}} {
		ps = append(ps, Pair{p[0], p[1]}, Pair{p[1], p[0]})
	}

	// Other scripts and emoji, alone and mixed with ASCII.
	scripts := [][]rune{
		g.alphabet(0x370, 0x400),   // Greek
		g.alphabet(0x400, 0x530),   // Cyrillic
		g.alphabet(0x4e00, 0x4f00), // CJK
		g.alphabet(0x3040, 0x3100), // Hiragana and Katakana
		g.alphabet(0xac00, 0xad00), // Hangul syllables
		g.alphabet(0x1100, 0x1200), // Hangul jamo
		g.alphabet(0x1f300, 0x1f700),
		g.alphabet(0x1f900, 0x1fa00),
		g.alphabet(0x0600, 0x0700), // Arabic
		g.alphabet(0x0900, 0x0980), // Devanagari
		g.alphabet(0x0e00, 0x0e80), // Thai
	}
	for _, s := range scripts {
		mixed := append(append([]rune{}, s...), g.alphabetOf("aA1 -")...)
		for range 15000 {
			ps = append(ps, Pair{g.str(s, 1, 5), g.str(s, 1, 5)})
			ps = append(ps, Pair{g.str(mixed, 1, 6), g.str(mixed, 1, 6)})
		}
	}
	every := g.alphabet(0x01, 0x30000)
	for range 50000 {
		ps = append(ps, Pair{g.str(every, 1, 4), g.str(every, 1, 4)})
	}

	// The empty string and long strings.
	for _, s := range []string{"", " ", "a", "-", "$", "é", "日"} {
		ps = append(ps, Pair{"", s}, Pair{s, ""})
	}
	for range 2000 {
		long := g.str(ascii, 100, 2000)
		ps = append(ps, Pair{long, g.mutate(long)})
		ps = append(ps, Pair{long, long + g.str(ascii, 1, 3)})
		ps = append(ps, Pair{long, long})
		l2 := g.str(latin, 100, 500)
		ps = append(ps, Pair{l2, g.mutate(l2)})
	}

	// Strings that are not valid UTF-8.
	for range 50000 {
		ps = append(ps, Pair{g.invalid(), g.invalid()})
		ps = append(ps, Pair{g.invalid(), g.str(ascii, 0, 4)})
	}

	// Near misses, pairs that differ in one place.
	n := len(ps)
	for i := 0; i < n; i += 7 {
		ps = append(ps, Pair{ps[i].A, g.mutate(ps[i].A)})
	}
	return ps
}

type gen struct {
	r *rand.Rand
}

func (g *gen) alphabet(lo, hi rune) []rune {
	var rs []rune
	for c := lo; c < hi; c++ {
		if utf8.ValidRune(c) {
			rs = append(rs, c)
		}
	}
	return rs
}

func (g *gen) alphabetOf(s string) []rune { return []rune(s) }

func (g *gen) str(alphabet []rune, minLen, maxLen int) string {
	var b strings.Builder
	for range minLen + g.r.IntN(maxLen-minLen+1) {
		b.WriteRune(alphabet[g.r.IntN(len(alphabet))])
	}
	return b.String()
}

func (g *gen) pairsWithin(ss []string, n int) []Pair {
	ps := make([]Pair, 0, n)
	for range n {
		ps = append(ps, Pair{ss[g.r.IntN(len(ss))], ss[g.r.IntN(len(ss))]})
	}
	return ps
}

func (g *gen) mixCase(s string) string {
	b := []byte(s)
	for i, c := range b {
		if g.r.IntN(2) == 0 {
			continue
		}
		switch {
		case 'a' <= c && c <= 'z':
			b[i] = c - 'a' + 'A'
		case 'A' <= c && c <= 'Z':
			b[i] = c - 'A' + 'a'
		}
	}
	return string(b)
}

// mutate changes, inserts or deletes one byte, never producing NUL.
func (g *gen) mutate(s string) string {
	b := []byte(s)
	c := byte(1 + g.r.IntN(255)) //nolint:gosec // 1..255
	switch {
	case len(b) == 0:
		return string(c)
	case g.r.IntN(3) == 0:
		i := g.r.IntN(len(b))
		return string(b[:i]) + string(b[i+1:])
	case g.r.IntN(2) == 0:
		i := g.r.IntN(len(b) + 1)
		return string(b[:i]) + string(c) + string(b[i:])
	default:
		b[g.r.IntN(len(b))] = c
		return string(b)
	}
}

// invalid returns a string with malformed UTF-8. It mixes stray bytes,
// truncated sequences at the end and in the middle, overlong forms and
// surrogates.
func (g *gen) invalid() string {
	pieces := []string{
		"a", "Z", "1", "-", " ", "é", "日", "😀",
		"\x80", "\xbf", "\xc0", "\xc1\xbf", "\xc3", "\xe6", "\xe6\x97", "\xf0", "\xf0\x9f", "\xf0\x9f\x98",
		"\xc0\xaf", "\xe0\x80\xaf", "\xed\xa0\x80", "\xf4\x90\x80\x80", "\xf8\x88\x80\x80\x80", "\xfe", "\xff",
	}
	var b strings.Builder
	for range 1 + g.r.IntN(5) {
		if g.r.IntN(4) == 0 {
			b.WriteByte(byte(1 + g.r.IntN(255))) //nolint:gosec // 1..255
			continue
		}
		b.WriteString(pieces[g.r.IntN(len(pieces))])
	}
	return b.String()
}

func isAlnum(c rune) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
}

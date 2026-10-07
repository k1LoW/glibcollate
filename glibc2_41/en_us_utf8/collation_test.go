package en_us_utf8_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/k1LoW/glibcollate"
	"github.com/k1LoW/glibcollate/glibc2_41/en_us_utf8"
	"github.com/k1LoW/glibcollate/internal/strcoll"
	"github.com/k1LoW/glibcollate/internal/tables/lccollate_f062bf818ec7"
)

// The expected values below were measured with strcoll on Debian glibc
// 2.41-12+deb13u4. They hold for this collation only.

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

func TestCompareGlibc2_41(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"00A", "0_0a", -1},
		{"_00a", "00A", -1},
		{"a-b", "ab", -1},
		{"", "", 0},
		{"", "a", -1},
		{"a", "", 1},
		{"a", "a", 0},
	}
	for _, tt := range tests {
		if got := sign(en_us_utf8.Collation.Compare(tt.a, tt.b)); got != tt.want {
			t.Errorf("Compare(%q, %q) sign = %d, want %d", tt.a, tt.b, got, tt.want)
		}
		if got := sign(en_us_utf8.Collation.Compare(tt.b, tt.a)); got != -tt.want {
			t.Errorf("Compare(%q, %q) sign = %d, want %d", tt.b, tt.a, got, -tt.want)
		}
	}
}

func TestOrderByGlibc2_41(t *testing.T) {
	in := []string{"a", "B", "_c", "b", "A-1", "a1", "-a", " a", "ab", "a b", "a-b", "a_b", "Ab", "aB"}
	want := []string{" a", "-a", "a", "a1", "A-1", "a b", "a-b", "a_b", "ab", "aB", "Ab", "b", "B", "_c"}
	got := slices.Clone(in)
	// The byte-order tie-break belongs to the caller.
	slices.SortFunc(got, func(a, b string) int {
		if c := en_us_utf8.Collation.Compare(a, b); c != 0 {
			return c
		}
		return strings.Compare(a, b)
	})
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestASCIIPunctuationGlibc2_41(t *testing.T) {
	// A character that is ignorable at the first three levels cannot decide
	// the order before the case difference between b and B does.
	for c := byte(0x20); c < 0x7f; c++ {
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
			continue
		}
		p := string(c)
		lower := en_us_utf8.Collation.Compare("a"+p+"b", "aB")
		upper := en_us_utf8.Collation.Compare("a"+p+"B", "ab")
		if c == '$' {
			if lower >= 0 || upper >= 0 {
				t.Errorf("%q should carry a primary weight that sorts before letters, got %d and %d", p, lower, upper)
			}
			continue
		}
		if lower >= 0 || upper <= 0 {
			t.Errorf("%q should be ignorable at the first three levels, got %d and %d", p, lower, upper)
		}
	}
}

func TestEqualIsZero(t *testing.T) {
	for _, s := range []string{"a", "abc", "Ünïcödé", "日本語", "a-b c_d"} {
		if got := en_us_utf8.Collation.Compare(s, strings.Clone(s)); got != 0 {
			t.Errorf("Compare(%q, %q) = %d, want 0", s, s, got)
		}
	}
}

func TestLookup(t *testing.T) {
	for _, name := range []string{"en_US.UTF-8", "en_US.utf8", "en_US.Utf-8"} {
		c, ok := glibcollate.Lookup(name, "2.41")
		if !ok || c != en_us_utf8.Collation {
			t.Errorf("Lookup(%q, 2.41) = %v, %v", name, c, ok)
		}
	}
	if _, ok := glibcollate.Lookup("en_US.UTF-8", "2.36"); ok {
		t.Error("Lookup found a version that is not imported")
	}
}

func TestCompareDoesNotAllocate(t *testing.T) {
	a, b := "Straße naïve 日本語 a-b", "Strasse naive 日本語 ab"
	if n := testing.AllocsPerRun(100, func() { en_us_utf8.Collation.Compare(a, b) }); n != 0 {
		t.Errorf("Compare allocates %v times per call", n)
	}
}

func BenchmarkSort(b *testing.B) {
	words := make([]string, 50000)
	seed := uint32(1)
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 -_.,'$"
	for i := range words {
		var sb strings.Builder
		for range 4 + i%8 {
			seed = seed*1664525 + 1013904223
			sb.WriteByte(alphabet[seed>>24%uint32(len(alphabet))])
		}
		words[i] = sb.String()
	}
	b.ResetTimer()
	for range b.N {
		s := slices.Clone(words)
		slices.SortFunc(s, en_us_utf8.Collation.Compare)
	}
}

func TestNeverReadsPastNUL(t *testing.T) {
	// With this property, strings ending in a truncated multibyte sequence are
	// ordered as glibc orders them, not only those in the differential corpus.
	if err := strcoll.CheckBounded(lccollate_f062bf818ec7.Table); err != nil {
		t.Error(err)
	}
}

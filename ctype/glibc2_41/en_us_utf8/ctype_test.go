package en_us_utf8_test

import (
	"testing"
	"unicode/utf8"

	"github.com/k1LoW/glibctext/ctype"
	"github.com/k1LoW/glibctext/ctype/glibc2_41/en_us_utf8"
)

// The expected values below were measured with towlower_l and towupper_l on
// Debian glibc 2.41-12+deb13u4. They hold for this ctype only.

func TestToLowerToUpperGlibc2_41(t *testing.T) {
	tests := []struct {
		r, lower, upper rune
	}{
		{'A', 'a', 'A'},
		{'a', 'a', 'A'},
		{'0', '0', '0'},
		{0, 0, 0},
		{'Ä', 'ä', 'Ä'},
		{'ÿ', 'ÿ', 'Ÿ'},
		{'İ', 'i', 'İ'},           // U+0130 lowers to plain i, not as in tr_TR
		{'ı', 'ı', 'I'},           // U+0131
		{'ß', 'ß', 'ß'},           // no single character upper case
		{'ẞ', 'ß', 'ẞ'},           // U+1E9E
		{'ſ', 'ſ', 'S'},           // U+017F
		{'ς', 'ς', 'Σ'},           // U+03C2
		{'ǅ', 'ǆ', 'Ǆ'},           // U+01C5, title case
		{'\u212A', 'k', '\u212A'}, // KELVIN SIGN
		{'ა', 'ა', 'Ა'},           // U+10D0 upper cases to Mtavruli
		{'日', '日', '日'},
		// Unicode 16 additions, which Go's unicode package (Unicode 15)
		// does not map.
		{0xA7CB, 0x0264, 0xA7CB},
		{0x0264, 0x0264, 0xA7CB},
		{0x1C89, 0x1C8A, 0x1C89},
		{0x10D50, 0x10D70, 0x10D50},
		// Past Unicode and negative values are returned as they are.
		{0x110000, 0x110000, 0x110000},
		{-1, -1, -1},
	}
	for _, tt := range tests {
		if got := en_us_utf8.Ctype.ToLower(tt.r); got != tt.lower {
			t.Errorf("ToLower(%U) = %U, want %U", tt.r, got, tt.lower)
		}
		if got := en_us_utf8.Ctype.ToUpper(tt.r); got != tt.upper {
			t.Errorf("ToUpper(%U) = %U, want %U", tt.r, got, tt.upper)
		}
	}
}

func TestLowerUpper(t *testing.T) {
	tests := []struct {
		s, lower, upper string
	}{
		{"", "", ""},
		{"Straße İstanbul", "straße istanbul", "STRAßE İSTANBUL"},
		{"ΟΔΥΣΣΕΥΣ", "οδυσσευσ", "ΟΔΥΣΣΕΥΣ"},
		{"a\x00B", "a\x00b", "A\x00B"},
		// Bytes that are not valid UTF-8 are kept, not replaced by U+FFFD.
		{"A\xffb\xc3", "a\xffb\xc3", "A\xffB\xc3"},
		{"�A", "�a", "�A"},
	}
	for _, tt := range tests {
		if got := en_us_utf8.Ctype.Lower(tt.s); got != tt.lower {
			t.Errorf("Lower(%+q) = %+q, want %+q", tt.s, got, tt.lower)
		}
		if got := en_us_utf8.Ctype.Upper(tt.s); got != tt.upper {
			t.Errorf("Upper(%+q) = %+q, want %+q", tt.s, got, tt.upper)
		}
	}
}

func TestMapsToValidRunes(t *testing.T) {
	// Lower and Upper would write U+FFFD for a result that is not a valid
	// rune, where wcstombs fails, so the table must never give one.
	for r := range rune(utf8.MaxRune + 1) {
		if !utf8.ValidRune(r) {
			continue
		}
		if l, u := en_us_utf8.Ctype.ToLower(r), en_us_utf8.Ctype.ToUpper(r); !utf8.ValidRune(l) || !utf8.ValidRune(u) {
			t.Errorf("%U maps to %U and %U", r, l, u)
		}
	}
}

func TestLookup(t *testing.T) {
	for _, name := range []string{"en_US.UTF-8", "en_US.utf8", "en_US.Utf-8"} {
		c, ok := ctype.Lookup(name, "2.41")
		if !ok || c != en_us_utf8.Ctype {
			t.Errorf("Lookup(%q, 2.41) = %v, %v", name, c, ok)
		}
	}
	if _, ok := ctype.Lookup("en_US.UTF-8", "2.36"); ok {
		t.Error("Lookup found a version that is not imported")
	}
}

func TestToLowerDoesNotAllocate(t *testing.T) {
	if n := testing.AllocsPerRun(100, func() {
		en_us_utf8.Ctype.ToLower('Ä')
		en_us_utf8.Ctype.ToUpper(0x10D50)
	}); n != 0 {
		t.Errorf("ToLower and ToUpper allocate %v times per call", n)
	}
}

package wctype

import (
	"strings"
	"unicode/utf8"
)

// ToLower returns what towlower_l returns for r under t. A negative r is
// passed as the wint_t of the same bits.
func (t *Table) ToLower(r rune) rune {
	return rune(towlowerL(uint32(r), t))
}

// ToUpper returns what towupper_l returns for r under t.
func (t *Table) ToUpper(r rune) rune {
	return rune(towupperL(uint32(r), t))
}

// Lower maps each character of s with ToLower.
func (t *Table) Lower(s string) string {
	return mapString(s, t.ToLower)
}

// Upper maps each character of s with ToUpper.
func (t *Table) Upper(s string) string {
	return mapString(s, t.ToUpper)
}

// mapString differs from strings.Map in keeping bytes that are not valid
// UTF-8 as they are, since replacing them with U+FFFD would turn input glibc
// rejects into a valid string that looks like a mapped one.
func mapString(s string, f func(rune) rune) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && n == 1 {
			b.WriteByte(s[i])
		} else {
			b.WriteRune(f(r))
		}
		i += n
	}
	return b.String()
}

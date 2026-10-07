// Package localename normalizes locale names the way glibc does.
package localename

import "strings"

// Normalize rewrites the codeset of a name of the form
// language[_territory][.codeset][@modifier] as glibc's _nl_normalize_codeset
// does, so that "en_US.UTF-8" and "en_US.utf8" give the same key.
// The rest of the name is kept as is, since glibc matches it case-sensitively.
func Normalize(name string) string {
	dot := strings.IndexByte(name, '.')
	if dot < 0 {
		return name
	}
	end := len(name)
	if at := strings.IndexByte(name[dot+1:], '@'); at >= 0 {
		end = dot + 1 + at
	}
	return name[:dot+1] + normalizeCodeset(name[dot+1:end]) + name[end:]
}

func normalizeCodeset(codeset string) string {
	onlyDigit := true
	var b strings.Builder
	for i := 0; i < len(codeset); i++ {
		c := codeset[i]
		switch {
		case 'a' <= c && c <= 'z':
			onlyDigit = false
			b.WriteByte(c)
		case 'A' <= c && c <= 'Z':
			onlyDigit = false
			b.WriteByte(c + 'a' - 'A')
		case '0' <= c && c <= '9':
			b.WriteByte(c)
		}
	}
	if onlyDigit {
		return "iso" + b.String()
	}
	return b.String()
}

// PackageName returns the directory and package name used for a locale,
// such as en_us_utf8 for en_US.UTF-8.
func PackageName(name string) string {
	n := strings.ToLower(Normalize(name))
	return strings.NewReplacer(".", "_", "@", "_", "-", "_").Replace(n)
}

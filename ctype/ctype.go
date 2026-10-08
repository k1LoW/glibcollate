// Package ctype maps the case of characters exactly as glibc's towlower and
// towupper do for a given locale and glibc version, in pure Go without cgo or
// the host's libc.
//
// Each ctype lives in its own package named after the glibc version and the
// locale, such as
//
//	github.com/k1LoW/glibctext/ctype/glibc2_41/en_us_utf8
//
// Its tables are glibc's compiled LC_CTYPE data, taken from the
// locale-archive of the image recorded in that package's documentation, and
// its mapping is a port of glibc's wctype/wcfuncs_l.c. Only the packages a
// program imports are linked into it.
//
// Importing a ctype package also registers it, so that it can be found with
// [Lookup] as below.
//
//	import _ "github.com/k1LoW/glibctext/ctype/glibc2_41/en_us_utf8"
//
//	c, ok := ctype.Lookup("en_US.UTF-8", "2.41")
package ctype

import (
	"fmt"
	"sync"

	"github.com/k1LoW/glibctext/internal/localename"
)

// Ctype maps the case of characters as glibc's LC_CTYPE does for a locale.
type Ctype interface {
	// ToLower returns what towlower_l returns for r. A negative r is passed
	// as the wint_t of the same bits.
	ToLower(r rune) rune
	// ToUpper returns what towupper_l returns for r.
	ToUpper(r rune) rune
	// Lower maps each UTF-8 encoded character of s with ToLower, as
	// mbstowcs, towlower_l on each wide character and wcstombs do. Bytes
	// that are not valid UTF-8, which mbstowcs rejects, are kept as they are.
	Lower(s string) string
	// Upper maps each UTF-8 encoded character of s with ToUpper, as Lower
	// does with ToLower.
	Upper(s string) string
}

type key struct {
	locale  string
	version string
}

var (
	mu       sync.RWMutex
	registry = map[key]Ctype{}
)

// Register makes c available to [Lookup] under the locale and glibc version.
// Ctype packages call it from init. It panics if the pair is registered twice
// or c is nil.
func Register(locale, version string, c Ctype) {
	if c == nil {
		panic("ctype: Register ctype is nil")
	}
	k := key{localename.Normalize(locale), version}
	mu.Lock()
	defer mu.Unlock()
	if _, dup := registry[k]; dup {
		panic(fmt.Sprintf("ctype: Register called twice for %s (glibc %s)", locale, version))
	}
	registry[k] = c
}

// Lookup returns the registered ctype for the locale and glibc version, such
// as ("en_US.UTF-8", "2.41"). The codeset part of the locale name is
// normalized as glibc does, so "en_US.utf8" finds the same ctype. Only ctypes
// whose packages are imported are registered.
func Lookup(locale, version string) (Ctype, bool) {
	mu.RLock()
	defer mu.RUnlock()
	c, ok := registry[key{localename.Normalize(locale), version}]
	return c, ok
}

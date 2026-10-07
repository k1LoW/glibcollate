// Package collate orders strings exactly as glibc's strcoll does for a
// given locale and glibc version, in pure Go without cgo or the host's libc.
//
// Each collation lives in its own package named after the glibc version and
// the locale, such as
//
//	github.com/k1LoW/glibctext/collate/glibc2_41/en_us_utf8
//
// Its tables are glibc's compiled LC_COLLATE data, taken from the
// locale-archive of the image recorded in that package's documentation, and
// its comparison is a port of glibc's string/strcoll_l.c. Only the packages a
// program imports are linked into it.
//
// Strings are read as strcoll reads C strings. Their bytes are looked up in
// the compiled tables as they are, so invalid UTF-8 is ordered as glibc orders
// it, and an embedded NUL ends the string. Each collation package tests that
// glibc never reads past the end of a string with its tables, which would make
// the result depend on memory beyond the string.
//
// Importing a collation package also registers it, so that it can be found
// with [Lookup] as below.
//
//	import _ "github.com/k1LoW/glibctext/collate/glibc2_41/en_us_utf8"
//
//	c, ok := collate.Lookup("en_US.UTF-8", "2.41")
package collate

import (
	"fmt"
	"sync"

	"github.com/k1LoW/glibctext/internal/localename"
)

// Collation orders strings as glibc's strcoll does for a locale.
type Collation interface {
	// Compare returns what strcoll_l returns for a and b, sign for sign.
	// It returns 0 when strcoll returns 0 and never breaks ties by byte order.
	Compare(a, b string) int
}

type key struct {
	locale  string
	version string
}

var (
	mu       sync.RWMutex
	registry = map[key]Collation{}
)

// Register makes c available to [Lookup] under the locale and glibc version.
// Collation packages call it from init. It panics if the pair is registered
// twice or c is nil.
func Register(locale, version string, c Collation) {
	if c == nil {
		panic("collate: Register collation is nil")
	}
	k := key{localename.Normalize(locale), version}
	mu.Lock()
	defer mu.Unlock()
	if _, dup := registry[k]; dup {
		panic(fmt.Sprintf("collate: Register called twice for %s (glibc %s)", locale, version))
	}
	registry[k] = c
}

// Lookup returns the registered collation for the locale and glibc version,
// such as ("en_US.UTF-8", "2.41"). The codeset part of the locale name is
// normalized as glibc does, so "en_US.utf8" finds the same collation.
// Only collations whose packages are imported are registered.
func Lookup(locale, version string) (Collation, bool) {
	mu.RLock()
	defer mu.RUnlock()
	c, ok := registry[key{localename.Normalize(locale), version}]
	return c, ok
}

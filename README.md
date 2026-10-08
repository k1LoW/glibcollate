# glibctext

glibctext reproduces the locale-dependent string handling of glibc exactly, in pure Go with no cgo and no dependency on the host's libc. It gives the same results on macOS, Windows and Linux as glibc gives on Linux.

It is meant for programs that have to reproduce the behavior of software running on glibc. Its scope is the locale categories that act on strings. Collation (`LC_COLLATE`) is in the `collate` package, which orders strings exactly as glibc's `strcoll` does for a given locale and glibc version. Case mapping (`LC_CTYPE`) is in the `ctype` package, which maps characters exactly as glibc's `towlower` and `towupper` do.

## Usage

Each collation is a package named after the glibc version and the locale.

```go
import "github.com/k1LoW/glibctext/collate/glibc2_41/en_us_utf8"

slices.SortFunc(names, func(a, b string) int {
	if c := en_us_utf8.Collation.Compare(a, b); c != 0 {
		return c
	}
	return strings.Compare(a, b) // tie-break by byte order
})
```

`Compare` returns what `strcoll_l` returns for the same inputs, including 0 for strings that `strcoll` treats as equal. It does not break ties by byte order. Callers that need a total order add the tie-break themselves.

A collation can also be looked up by name after its package is imported. The codeset part of the locale name is normalized as glibc does, so `en_US.utf8` and `en_US.UTF-8` are the same.

```go
import (
	"github.com/k1LoW/glibctext/collate"
	_ "github.com/k1LoW/glibctext/collate/glibc2_41/en_us_utf8"
)

c, ok := collate.Lookup("en_US.UTF-8", "2.41")
```

Only the collation packages a program imports are linked into it. Each one adds roughly 0.7 MB of tables.

### Case mapping

```go
import "github.com/k1LoW/glibctext/ctype/glibc2_41/en_us_utf8"

en_us_utf8.Ctype.ToLower('Σ')        // 'σ', what towlower_l returns
en_us_utf8.Ctype.Upper("straße")     // "STRAßE"
```

`ToLower` and `ToUpper` return what `towlower_l` and `towupper_l` return for one character. `Lower` and `Upper` map each character of a UTF-8 string with them, as converting with `mbstowcs`, mapping each wide character and converting back with `wcstombs` does. Bytes that are not valid UTF-8 are kept as they are. A ctype can be looked up by name with `ctype.Lookup` in the same way as a collation.

The mapping is glibc's, not Go's `unicode` package. The two follow different versions of Unicode, and for `en_US.UTF-8` of glibc 2.41 they differ in 54 characters, such as U+A7CB, which glibc lowers to U+0264 and Go leaves as it is.

A collation and a ctype of the same locale are separate packages with the same name, so a program that needs both imports one under another name.

## Collations and ctypes

| Package | Locale | glibc | Source image |
| --- | --- | --- | --- |
| `collate/glibc2_41/en_us_utf8` | `en_US.UTF-8` | 2.41 (Debian 2.41-12+deb13u4) | `postgres:18@sha256:74935e72241653ca55e0414067e6d8763aceb8a810eb51b452253ec3dcfc4336` |
| `ctype/glibc2_41/en_us_utf8` | `en_US.UTF-8` | 2.41 (Debian 2.41-12+deb13u4) | `postgres:18@sha256:74935e72241653ca55e0414067e6d8763aceb8a810eb51b452253ec3dcfc4336` |

The source image is used only to pin glibc. Pinned by digest, it fixes both the glibc build and its compiled `locale-archive`, so regenerating gives the same tables.

A collation or a ctype is never changed to follow a newer glibc. A newer glibc gets packages of its own.

## How it works

- **Tables.** The tables are glibc's own compiled `LC_COLLATE` and `LC_CTYPE` data, read from the `locale-archive` of the source image. The locale sources are not reinterpreted. `devtools/gen` copies the archive out of the image, extracts the locale and writes the tables as Go source under `internal/tables`.
- **Comparison.** `internal/strcoll` is a port of glibc's `string/strcoll_l.c` and `locale/weight.h`. It follows the C source closely so that the two can be compared line by line. `internal/wctype` is likewise a port of `towlower_l` and `towupper_l` in `wctype/wcfuncs_l.c` and of the table lookup in `wctype/wchar-lookup.h`.
- **Input.** `Compare` reads strings as C strings, as `strcoll` reads them. Bytes are looked up in the compiled tables as they are, so invalid UTF-8 is ordered exactly as glibc orders it. An embedded NUL ends the string, as it would for `strcoll`. glibc would read memory past the end of a string if a table let a truncated multibyte sequence match a range of characters. Each collation package checks that its tables cannot do that, so the result never depends on what lies beyond the string. `Lower` and `Upper` of a ctype map NUL like any other character and keep invalid UTF-8 as it is.

## Verification

`devtools/difftest` builds a small C program that calls `strcoll_l`, runs it on the unmodified source image with testcontainers, and checks that `Compare` returns the same value for about a million pairs of strings. The pairs cover random ASCII, punctuation and spaces in every position, digits and letters, mixed case, Latin-1 and accented letters both precomposed and combining, Greek, Cyrillic, CJK, Hangul, emoji, empty and long strings, and invalid UTF-8. Any difference fails the test.

For each ctype, it calls `towlower_l` and `towupper_l` for every wide character from 0 to U+10FFFF and a few values past it, and checks that `ToLower` and `ToUpper` return the same. The comparison is exhaustive.

```console
$ make test       # unit and regression tests, no Docker needed
$ make difftest   # differential test against the real strcoll_l, needs Docker
```

## Adding a glibc version or a locale

1. Pick a Debian or Ubuntu based image that ships the glibc version and has the locale in `/usr/lib/locale/locale-archive`, and pin it by digest. The generator reads the glibc version from the dpkg status file and runs nothing inside the image.
2. Run the generator.

   ```console
   $ cd devtools
   $ go run ./gen -image debian:bookworm@sha256:... -locale en_US.UTF-8
   ```

   It writes `internal/tables/lccollate_<hash>`, `internal/tables/lcctype_<hash>`, `collate/glibc<version>/<locale>` and `ctype/glibc<version>/<locale>`. Tables with the same content share one package.
3. Add the image to `Makefile`, to `targets` in `devtools/difftest/strcoll_test.go` and to `ctypeTargets` in `devtools/difftest/towctrans_test.go`, and run `make difftest`.

Regression tests with expected values belong to each collation and ctype package, since they hold for one glibc version only.

## License

glibctext is licensed under the GNU Lesser General Public License version 2.1 or later. See [LICENSE](LICENSE).

`internal/strcoll` and `internal/wctype` are ports of files from the GNU C Library and keep their notices.

```
Copyright (C) 1995-2025 Free Software Foundation, Inc.
This file is part of the GNU C Library.
```

The tables under `internal/tables` are compiled from the locale data of the GNU C Library, which carries this notice.

```
This file is part of the GNU C Library and contains locale data.
The Free Software Foundation does not claim any copyright interest
in the locale data contained in this file.  The foregoing does not
affect the license of the GNU C Library as a whole.  It does not
exempt you from the conditions of the license if your use would
otherwise be governed by that license.
```

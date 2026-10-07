# glibcollate

glibcollate orders strings exactly as glibc's `strcoll` does for a given locale and glibc version. It is pure Go, with no cgo and no dependency on the host's libc, so it gives the same order on macOS, Windows and Linux.

It is meant for programs that have to reproduce the order of software running on glibc.

## Usage

Each collation is a package named after the glibc version and the locale.

```go
import "github.com/k1LoW/glibcollate/glibc2_41/en_us_utf8"

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
	"github.com/k1LoW/glibcollate"
	_ "github.com/k1LoW/glibcollate/glibc2_41/en_us_utf8"
)

c, ok := glibcollate.Lookup("en_US.UTF-8", "2.41")
```

Only the collation packages a program imports are linked into it. Each one adds roughly 0.7 MB of tables.

## Collations

| Package | Locale | glibc | Source image |
| --- | --- | --- | --- |
| `glibc2_41/en_us_utf8` | `en_US.UTF-8` | 2.41 (Debian 2.41-12+deb13u4) | `postgres:18@sha256:74935e72241653ca55e0414067e6d8763aceb8a810eb51b452253ec3dcfc4336` |

The source image is used only to pin glibc. Pinned by digest, it fixes both the glibc build and its compiled `locale-archive`, so regenerating gives the same tables.

A collation is never changed to follow a newer glibc. When another glibc orders strings differently, it gets its own package.

## How it works

- **Tables.** The tables are glibc's own compiled `LC_COLLATE` data, read from the `locale-archive` of the source image. The locale sources are not reinterpreted. `devtools/gen` copies the archive out of the image, extracts the locale and writes the tables as Go source under `internal/tables`.
- **Comparison.** `internal/strcoll` is a port of glibc's `string/strcoll_l.c` and `locale/weight.h`. It follows the C source closely so that the two can be compared line by line.
- **Input.** Strings are read as C strings, as `strcoll` reads them. Bytes are looked up in the compiled tables as they are, so invalid UTF-8 is ordered exactly as glibc orders it. An embedded NUL ends the string, as it would for `strcoll`. glibc would read memory past the end of a string if a table let a truncated multibyte sequence match a range of characters. Each collation package checks that its tables cannot do that, so the result never depends on what lies beyond the string.

## Verification

`devtools/difftest` builds a small C program that calls `strcoll_l`, runs it on the unmodified source image with testcontainers, and checks that `Compare` returns the same value for about a million pairs of strings. The pairs cover random ASCII, punctuation and spaces in every position, digits and letters, mixed case, Latin-1 and accented letters both precomposed and combining, Greek, Cyrillic, CJK, Hangul, emoji, empty and long strings, and invalid UTF-8. Any difference fails the test.

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

   It writes `internal/tables/lccollate_<hash>` and `glibc<version>/<locale>`. Tables with the same content share one package.
3. Add the image to `Makefile` and to `targets` in `devtools/difftest/strcoll_test.go`, and run `make difftest`.

Regression tests with expected values belong to each collation package, since they hold for one glibc version only.

## License

glibcollate is licensed under the GNU Lesser General Public License version 2.1 or later. See [LICENSE](LICENSE).

`internal/strcoll` is a port of files from the GNU C Library and keeps their notices.

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

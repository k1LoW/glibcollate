# AGENTS.md

## What glibctext is for

glibctext reproduces the locale-dependent string handling of glibc exactly, in pure Go. Its scope is the locale categories that act on strings (`LC_COLLATE` and `LC_CTYPE`), not formatting categories such as `LC_NUMERIC` or `LC_TIME`. The `collate` package orders strings exactly as glibc's `strcoll` does for a given locale and glibc version, and the `ctype` package maps case exactly as `towlower` and `towupper` do. Exactness is the whole point. A result that is usually right is worse than none, because a wrong order or a wrong case looks like any other and nobody notices it.

This sets the rules for every change.

- Never reconstruct collation rules from observation, and never use `golang.org/x/text/collate` or ICU. Neither matches glibc.
- Never map case with Go's `unicode` package. It follows another Unicode version than glibc (15.0 against 16.0 for glibc 2.41), so 54 characters already differ for en_US.UTF-8.
- Never interpret the locale sources (`iso14651_t1_common` and the like) in Go. The tables are glibc's own compiled `LC_COLLATE`, produced by glibc's `localedef`. Its handling of `script`, `reorder-after` and per-section directions is too easy to get subtly wrong.
- The comparison is a port of glibc's C source, not a reimplementation.
- Any input where `Compare` and the real `strcoll_l` disagree, or `ToLower` and `ToUpper` and the real `towlower_l` and `towupper_l` disagree, is a bug. There is no list of known differences.

## Layout

```
collate/                       Collation, Register, Lookup. No dependencies.
  glibc<ver>/<locale>/         One package per glibc version and locale (generated collation.go, hand-written tests)
ctype/                         Ctype, Register, Lookup. No dependencies.
  glibc<ver>/<locale>/         One package per glibc version and locale (generated ctype.go, hand-written tests)
internal/strcoll/              Port of string/strcoll_l.c and locale/weight.h, plus CheckBounded
internal/wctype/               Port of wctrans_table_lookup in wctype/wchar-lookup.h and towlower_l/towupper_l in wctype/wcfuncs_l.c
internal/tables/lccollate_<h>/ Generated tables, named by the first 12 hex digits of the LC_COLLATE sha256
internal/tables/lcctype_<h>/   Generated tables, named by the first 12 hex digits of the LC_CTYPE sha256
internal/localename/           Locale name normalization, same as glibc's _nl_normalize_codeset
devtools/                      Separate module (testcontainers etc.), replace => ../
  gen/                         Generator. Copies locale-archive and dpkg status out of an image
  difftest/                    Differential test. C harness on the unmodified image via testcontainers
```

## Rules that must hold

- **`Compare` returns exactly what `strcoll_l` returns.** The difftest checks the exact value, not only the sign. `Compare` never breaks ties by byte order. Callers do that themselves.
- **A published collation or ctype never changes meaning.** There is no unversioned name such as `EnUS`. A new glibc gets a new package (`collate/glibc2_42/en_us_utf8`) even when its data is identical, in which case it imports the same `internal/tables` package.
- **Only imported collations and ctypes are linked.** The `collate` and `ctype` packages never import the per-locale packages. Tables are static data (string constants and `int32` composite literals), and a per-locale package's `init` only calls `Register`. Do not build tables in `init`. Verify with `go tool nm` on a binary that imports only the `collate` or `ctype` package.
- **The root module has zero dependencies and no cgo.** Anything that needs Docker, testcontainers or C lives in `devtools`. `CGO_ENABLED=0 go build ./...` must keep working for every GOOS/GOARCH.
- **`Compare`, `ToLower` and `ToUpper` do not allocate.** `TestCompareDoesNotAllocate` and `TestToLowerDoesNotAllocate` guard them.
- **`Lower` and `Upper` keep bytes that are not valid UTF-8.** They compose mbstowcs, towlower_l and wcstombs, which have no single glibc counterpart, so invalid input is passed through rather than replaced with U+FFFD. NUL is mapped like any other character (`towlower_l(0)` is 0) and does not end the string. `TestMapsToValidRunes` checks that a table never maps a valid rune to an invalid one.
- **`Compare` reads its input as C strings.** Bytes are looked up as they are, so invalid UTF-8 is ordered as glibc orders it. An embedded NUL ends the string. `at()` returns NUL past the end of the Go string.

## The port

- Keep `internal/strcoll` and `internal/wctype` structured like the C source so that it can be checked line by line. Keep its quirks, for example `seq1.idxmax` is not reset between passes and `rule` is saved only from the first sequence of pass 0.
- `size_t` is `uint64` and `int32_t` is `int32`, so arithmetic wraps as it does in C on 64-bit Linux, also on 32-bit Go targets. The integer conversions are intentional, and gosec G115 is excluded for `internal/strcoll` and `internal/wctype` in `.golangci.yml` for that reason.
- License (LGPL-2.1-or-later). Keep the FSF copyright block and the LGPL notice of the original file verbatim. LGPL 2.1 section 2(b) also requires a notice that the file was modified and the date, which is the `Modified by k1LoW on <date>` line. Update it when the port changes, and update the years in `_EXTRA_CREDITS` when a port comes from a newer glibc.
- towlower and towupper read only the two map tables at `_NL_CTYPE_MAP_OFFSET` + `__TOW_toupper` (0) and + `__TOW_tolower` (1). Unlike the `isw*` functions, they have no ASCII fast path.
- strcoll reads only the multibyte items (NRULES, RULESETS, TABLEMB, WEIGHTMB, EXTRAMB, INDIRECTMB). The WC tables, the symbol hash and COLLSEQ serve wcscoll, regex and fnmatch.

## Reading the compiled data

- The `LC_COLLATE` item index follows the `_NL_COLLATE_*` enum in `locale/langinfo.h`, which has `GAP1` to `GAP3` after INDIRECTMB. `locale/categories.def` omits the gaps, so do not use its order. CODESET is item 18.
- The magic is `LIMAGIC(LC_COLLATE)`, which is `0x20051014 ^ 3`. The data is little endian on the images used so far.
- The `LC_CTYPE` item index follows the `_NL_CTYPE_*` enum in `locale/langinfo.h`, which has `GAP1` to `GAP6` among the first items. `CODESET` is an alias of `_NL_CTYPE_CODESET_NAME` (14) and takes no index of its own, so `_NL_CTYPE_MAP_NAMES` is 11 and `_NL_CTYPE_MAP_OFFSET` is 18. The magic is `LIMAGIC(LC_CTYPE)`, which is `0x20090720 ^ 0`. The generator checks that `MAP_NAMES` starts with `toupper` and `tolower`.
- In `locale-archive`, names are normalized (`en_US.utf8`), and `LC_COLLATE` is record index 3 and `LC_CTYPE` record index 0 of `locrecent`. The generator scans the name hash table linearly instead of reimplementing glibc's hash.
- For glibc 2.41, the LC_COLLATE extracted from the archive is byte-identical to `localedef --no-archive -i en_US -c -f UTF-8`, and the amd64 and arm64 images give the same sha256.

## Reading past the end of a string

`findidx` has a branch for range entries that can step past the terminating NUL when a string ends in a truncated multibyte sequence. glibc then reads whatever memory follows, which the port cannot reproduce. `strcoll.CheckBounded` proves this cannot happen for a table. It requires that no sequence contains NUL and that every range differs from its start only in its last byte. The 2.41 en_US.UTF-8 table has 1,320 range entries and passes. Every collation package must have a `TestNeverReadsPastNUL` for its table, and the generator logs a warning when the check fails. If a new table fails it, stop and discuss before publishing the collation.

## Adding a glibc version or a locale

1. **Pick the image.** It must be Debian or Ubuntu based (the generator reads the libc6 version from `/var/lib/dpkg/status`), and it must have the locale in `/usr/lib/locale/locale-archive`. Pin it by digest. The image is used only to pin glibc and its compiled archive. When the locale is missing, the generator lists the locales the archive has.
2. **Check the C source.** Diff `string/strcoll_l.c`, `locale/weight.h`, `wctype/wchar-lookup.h` and `wctype/wcfuncs_l.c` between the version the port came from (2.41) and the new one. If they differ, the new version needs its own port. Do not change the existing port in a way that alters the result for published collations. Install `glibc-source` in the image to get the exact tarball and `debian/patches`.
3. **Check the distro patches.** Grep `debian/patches` for `string/strcoll`, `locale/weight`, `locale/loadlocale`, `categories.def`, `ld-collate`, `wctype/`, `ld-ctype`, `C-ctype`, `localeinfo.h`, `localedata/locales/`. For 2.41-12+deb13u4, only `locale/check-unknown-symbols.diff` touches `ld-collate.c`, and it only adds a warning. `git-updates.diff` touches `locale/lc-ctype.c` and `locale/localeinfo.h`, but only how the current locale is kept in thread-local storage. No patch touches `wctype/`, `ld-ctype.c` or the `en_US` and `i18n_ctype` sources.
4. **Generate.** Add `IMAGE_GLIBC<ver>` to the `Makefile` and run the generator, for example `cd devtools && go run ./gen -image <image@digest> -locale en_US.UTF-8`. It writes `internal/tables/lccollate_<h>/table.go`, `internal/tables/lcctype_<h>/table.go`, `collate/glibc<ver>/<pkg>/collation.go` and `ctype/glibc<ver>/<pkg>/ctype.go`. Running it twice must give identical files.
5. **Write the regression tests.** Add `collation_test.go` and `ctype_test.go` to the new packages with values measured on the real glibc of that image, never copied from another version. Include `TestNeverReadsPastNUL` and `TestMapsToValidRunes` for their tables.
6. **Add the difftest targets** to `targets` in `devtools/difftest/strcoll_test.go` and `ctypeTargets` in `devtools/difftest/towctrans_test.go`, and run `make difftest`.
7. **Update the docs.** Add the collation and the ctype to the tables in `README.md`.

## Tests

- `make test` runs unit and regression tests without Docker.
- `make difftest` builds `devtools/difftest/harness/strcoll.c` in a `debian:trixie` stage, copies the binary onto the unmodified target image, and compares about a million pairs. The harness prints `gnu_get_libc_version()`, and the test fails when it is not the expected version. The corpus has a fixed seed and no NUL bytes. When changing the harness or corpus, run a negative control once (compare against `C.UTF-8`) to confirm that the test can still fail.
- The difftest also builds `devtools/difftest/harness/towctrans.c` into the same image. It calls `towupper_l` and `towlower_l` for every wide character from 0 to 0x10FFFF and a few past it, so the ctype comparison is exhaustive, not a sample. Its negative control is the `C` locale, which maps ASCII letters only. `C.UTF-8` maps non-ASCII letters as well, so it may not make the test fail.
- `make lint` lints both modules with the same `.golangci.yml`.
- CI (`.github/workflows/ci.yml`) runs lint, tests with octocov, race and the difftest. Releases are made by tagpr.

## Writing

- Code comments, docs and commit messages are in English. Commits use Conventional Commits.
- Do not use the em dash, and do not use a colon or a hyphen to join clauses, in prose or comments. Syntax that needs them (Markdown, YAML keys, code, URLs) is fine.
- Comments explain why, not what the surrounding code already says.

# AGENTS.md

## What glibcollate is for

glibcollate orders strings exactly as glibc's `strcoll` does for a given locale and glibc version, in pure Go. Exactness is the whole point. A result that is usually right is worse than none, because a wrong order looks like any other order and nobody notices it.

This sets the rules for every change.

- Never reconstruct collation rules from observation, and never use `golang.org/x/text/collate` or ICU. Neither matches glibc.
- Never interpret the locale sources (`iso14651_t1_common` and the like) in Go. The tables are glibc's own compiled `LC_COLLATE`, produced by glibc's `localedef`. Its handling of `script`, `reorder-after` and per-section directions is too easy to get subtly wrong.
- The comparison is a port of glibc's C source, not a reimplementation.
- Any input where `Compare` and the real `strcoll_l` disagree is a bug. There is no list of known differences.

## Layout

```
glibcollate.go                 Collation, Register, Lookup. No dependencies.
glibc<ver>/<locale>/           One package per glibc version and locale (generated collation.go, hand-written tests)
internal/strcoll/              Port of string/strcoll_l.c and locale/weight.h, plus CheckBounded
internal/tables/lccollate_<h>/ Generated tables, named by the first 12 hex digits of the LC_COLLATE sha256
internal/localename/           Locale name normalization, same as glibc's _nl_normalize_codeset
devtools/                      Separate module (testcontainers etc.), replace => ../
  gen/                         Generator. Copies locale-archive and dpkg status out of an image
  difftest/                    Differential test. C harness on the unmodified image via testcontainers
```

## Rules that must hold

- **`Compare` returns exactly what `strcoll_l` returns.** The difftest checks the exact value, not only the sign. `Compare` never breaks ties by byte order. Callers do that themselves.
- **A published collation never changes meaning.** There is no unversioned name such as `EnUS`. A new glibc gets a new package (`glibc2_42/en_us_utf8`) even when its data is identical, in which case it imports the same `internal/tables` package.
- **Only imported collations are linked.** The root package never imports collation packages. Tables are static data (string constants and `int32` composite literals), and a collation package's `init` only calls `Register`. Do not build tables in `init`. Verify with `go tool nm` on a binary that imports only the root package.
- **The root module has zero dependencies and no cgo.** Anything that needs Docker, testcontainers or C lives in `devtools`. `CGO_ENABLED=0 go build ./...` must keep working for every GOOS/GOARCH.
- **`Compare` does not allocate.** `TestCompareDoesNotAllocate` guards it.
- **Input is read as C strings.** Bytes are looked up as they are, so invalid UTF-8 is ordered as glibc orders it. An embedded NUL ends the string. `at()` returns NUL past the end of the Go string.

## The port

- Keep `internal/strcoll` structured like the C source so that it can be checked line by line. Keep its quirks, for example `seq1.idxmax` is not reset between passes and `rule` is saved only from the first sequence of pass 0.
- `size_t` is `uint64` and `int32_t` is `int32`, so arithmetic wraps as it does in C on 64-bit Linux, also on 32-bit Go targets. The integer conversions are intentional, and gosec G115 is excluded for `internal/strcoll` in `.golangci.yml` for that reason.
- License (LGPL-2.1-or-later). Keep the FSF copyright block and the LGPL notice of the original file verbatim. LGPL 2.1 section 2(b) also requires a notice that the file was modified and the date, which is the `Modified by k1LoW on <date>` line. Update it when the port changes, and update the years in `_EXTRA_CREDITS` when a port comes from a newer glibc.
- strcoll reads only the multibyte items (NRULES, RULESETS, TABLEMB, WEIGHTMB, EXTRAMB, INDIRECTMB). The WC tables, the symbol hash and COLLSEQ serve wcscoll, regex and fnmatch.

## Reading the compiled data

- The `LC_COLLATE` item index follows the `_NL_COLLATE_*` enum in `locale/langinfo.h`, which has `GAP1` to `GAP3` after INDIRECTMB. `locale/categories.def` omits the gaps, so do not use its order. CODESET is item 18.
- The magic is `LIMAGIC(LC_COLLATE)`, which is `0x20051014 ^ 3`. The data is little endian on the images used so far.
- In `locale-archive`, names are normalized (`en_US.utf8`), and `LC_COLLATE` is record index 3 of `locrecent`. The generator scans the name hash table linearly instead of reimplementing glibc's hash.
- For glibc 2.41, the LC_COLLATE extracted from the archive is byte-identical to `localedef --no-archive -i en_US -c -f UTF-8`, and the amd64 and arm64 images give the same sha256.

## Reading past the end of a string

`findidx` has a branch for range entries that can step past the terminating NUL when a string ends in a truncated multibyte sequence. glibc then reads whatever memory follows, which the port cannot reproduce. `strcoll.CheckBounded` proves this cannot happen for a table. It requires that no sequence contains NUL and that every range differs from its start only in its last byte. The 2.41 en_US.UTF-8 table has 1,320 range entries and passes. Every collation package must have a `TestNeverReadsPastNUL` for its table, and the generator logs a warning when the check fails. If a new table fails it, stop and discuss before publishing the collation.

## Adding a glibc version or a locale

1. **Pick the image.** It must be Debian or Ubuntu based (the generator reads the libc6 version from `/var/lib/dpkg/status`), and it must have the locale in `/usr/lib/locale/locale-archive`. Pin it by digest. The image is used only to pin glibc and its compiled archive. When the locale is missing, the generator lists the locales the archive has.
2. **Check the C source.** Diff `string/strcoll_l.c` and `locale/weight.h` between the version the port came from (2.41) and the new one. If they differ, the new version needs its own port. Do not change the existing port in a way that alters the result for published collations. Install `glibc-source` in the image to get the exact tarball and `debian/patches`.
3. **Check the distro patches.** Grep `debian/patches` for `string/strcoll`, `locale/weight`, `locale/loadlocale`, `categories.def`, `ld-collate`, `localedata/locales/`. For 2.41-12+deb13u4, only `locale/check-unknown-symbols.diff` touches `ld-collate.c`, and it only adds a warning.
4. **Generate.** Add `IMAGE_GLIBC<ver>` to the `Makefile` and run the generator, for example `cd devtools && go run ./gen -image <image@digest> -locale en_US.UTF-8`. It writes `internal/tables/lccollate_<h>/table.go` and `glibc<ver>/<pkg>/collation.go`. Running it twice must give identical files.
5. **Write the regression tests.** Add `collation_test.go` to the new package with values measured on the real glibc of that image, never copied from another version. Include `TestNeverReadsPastNUL` for its table.
6. **Add the difftest target** to `targets` in `devtools/difftest/strcoll_test.go` and run `make difftest`.
7. **Update the docs.** Add the collation to the table in `README.md`.

## Tests

- `make test` runs unit and regression tests without Docker.
- `make difftest` builds `devtools/difftest/harness/strcoll.c` in a `debian:trixie` stage, copies the binary onto the unmodified target image, and compares about a million pairs. The harness prints `gnu_get_libc_version()`, and the test fails when it is not the expected version. The corpus has a fixed seed and no NUL bytes. When changing the harness or corpus, run a negative control once (compare against `C.UTF-8`) to confirm that the test can still fail.
- `make lint` lints both modules with the same `.golangci.yml`.
- CI (`.github/workflows/ci.yml`) runs lint, tests with octocov, race and the difftest. Releases are made by tagpr.

## Writing

- Code comments, docs and commit messages are in English. Commits use Conventional Commits.
- Do not use the em dash, and do not use a colon or a hyphen to join clauses, in prose or comments. Syntax that needs them (Markdown, YAML keys, code, URLs) is fine.
- Comments explain why, not what the surrounding code already says.

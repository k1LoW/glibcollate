/* Writes what towupper_l and towlower_l return for every wide character from
   0 to 0x10ffff, followed by the values in EXTRA, to stdout.  Each result is
   a pair of little-endian uint32, toupper first.  The first line of stdout is
   the glibc version that is actually loaded.  */
#define _GNU_SOURCE
#include <gnu/libc-version.h>
#include <locale.h>
#include <stdint.h>
#include <stdio.h>
#include <wctype.h>

/* Values past Unicode, which take the paths of wctrans_table_lookup that
   the Unicode range does not.  Keep in sync with extraWC in harness.go.  */
static const uint32_t extra[] = { 0x110000, 0x7fffffff, 0x80000000, 0xfffffffe, 0xffffffff };

static void
put (uint32_t v)
{
  uint8_t o[4] = { v, v >> 8, v >> 16, v >> 24 };
  fwrite (o, 1, 4, stdout);
}

static void
map (uint32_t wc, locale_t l)
{
  put (towupper_l (wc, l));
  put (towlower_l (wc, l));
}

int
main (int argc, char **argv)
{
  if (argc != 2)
    {
      fprintf (stderr, "usage: %s LOCALE\n", argv[0]);
      return 2;
    }
  locale_t l = newlocale (LC_ALL_MASK, argv[1], (locale_t) 0);
  if (l == (locale_t) 0)
    {
      perror ("newlocale");
      return 1;
    }
  printf ("%s\n", gnu_get_libc_version ());
  for (uint32_t wc = 0; wc <= 0x10ffff; wc++)
    map (wc, l);
  for (size_t i = 0; i < sizeof extra / sizeof extra[0]; i++)
    map (extra[i], l);
  return fflush (stdout) != 0;
}

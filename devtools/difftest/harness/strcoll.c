/* Reads pairs of strings from stdin and writes what strcoll_l returns for
   each pair to stdout.  Every string is a little-endian uint32 length followed
   by that many bytes, and every result is a little-endian int32.  The first
   line of stdout is the glibc version that is actually loaded.  */
#define _GNU_SOURCE
#include <gnu/libc-version.h>
#include <locale.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static char *
read_string (void)
{
  uint8_t h[4];
  if (fread (h, 1, 4, stdin) != 4)
    return NULL;
  uint32_t n = h[0] | h[1] << 8 | h[2] << 16 | (uint32_t) h[3] << 24;
  char *s = malloc (n + 1);
  if (s == NULL || fread (s, 1, n, stdin) != n)
    {
      fprintf (stderr, "short read\n");
      exit (1);
    }
  s[n] = '\0';
  return s;
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
  for (;;)
    {
      char *a = read_string ();
      if (a == NULL)
	break;
      char *b = read_string ();
      if (b == NULL)
	{
	  fprintf (stderr, "odd number of strings\n");
	  return 1;
	}
      int32_t r = strcoll_l (a, b, l);
      uint8_t o[4] = { r, r >> 8, r >> 16, (uint32_t) r >> 24 };
      fwrite (o, 1, 4, stdout);
      free (a);
      free (b);
    }
  return fflush (stdout) != 0;
}

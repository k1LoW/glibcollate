/* Copyright (C) 1996-2025 Free Software Foundation, Inc.
   This file is part of the GNU C Library.

   The GNU C Library is free software; you can redistribute it and/or
   modify it under the terms of the GNU Lesser General Public
   License as published by the Free Software Foundation; either
   version 2.1 of the License, or (at your option) any later version.

   The GNU C Library is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
   Lesser General Public License for more details.

   You should have received a copy of the GNU Lesser General Public
   License along with the GNU C Library; if not, see
   <https://www.gnu.org/licenses/>.  */

// Modified by k1LoW on 2026-10-08. This file is a Go port of locale/weight.h
// from glibc 2.41.
// The structure follows the C source so the two can be compared line by line.

package strcoll

// locfileAlign is LOCFILE_ALIGN (sizeof (int32_t)).
const locfileAlign = 4

func locfileAlignedP(x uint64) bool { return x&(locfileAlign-1) == 0 }

// at reads s as a NUL-terminated C string.
// Reading past the end yields the terminating NUL instead of panicking.
func at(s string, i int) byte {
	if i < len(s) {
		return s[i]
	}
	return 0
}

// le32 reads the int32 that the C code reads with *((const int32_t *) cp).
// The tables come from a little-endian host, which the generator checks.
func le32(b string, p int) int32 {
	return int32(uint32(b[p]) | uint32(b[p+1])<<8 | uint32(b[p+2])<<16 | uint32(b[p+3])<<24)
}

// findidx finds the index of the weight for the sequence at *cpp in s.
// length mirrors the size_t parameter; strcoll always passes -1.
func findidx(table *[256]int32, indirect []int32, extra string, s string, cpp *int, length uint64) int32 {
	i := table[at(s, *cpp)]
	*cpp++

	if i >= 0 {
		/* This is an index into the weight table.  Cool.  */
		return i
	}

	/* Oh well, more than one sequence starting with this byte.
	   Search for the correct one.  */
	cp := int(-i)
	usrc := *cpp
	length--
	for {
		/* The first thing is the index.  */
		i = le32(extra, cp)
		cp += 4

		/* Next is the length of the byte sequence.  These are always
		   short byte sequences so there is no reason to call any
		   function (even if they are inlined).  */
		nhere := uint64(extra[cp])
		cp++

		if i >= 0 {
			/* It is a single character.  If it matches we found our
			   index.  Note that at the end of each list there is an
			   entry of length zero which represents the single byte
			   sequence.  The first (and here only) byte was tested
			   already.  */
			var cnt uint64
			for cnt = 0; cnt < nhere && cnt < length; cnt++ {
				if extra[cp+int(cnt)] != at(s, usrc+int(cnt)) {
					break
				}
			}

			if cnt == nhere {
				/* Found it.  */
				*cpp += int(nhere)
				return i
			}

			/* Up to the next entry.  */
			cp += int(nhere)
			if !locfileAlignedP(1 + nhere) {
				cp += int(locfileAlign - (1+nhere)%locfileAlign)
			}
		} else {
			/* This is a range of characters.  First decide whether the
			   current byte sequence lies in the range.  */
			var cnt uint64
			var offset uint64

			for cnt = 0; cnt < nhere && cnt < length; cnt++ {
				if extra[cp+int(cnt)] != at(s, usrc+int(cnt)) {
					break
				}
			}

			if cnt != nhere {
				if cnt == length || extra[cp+int(cnt)] > at(s, usrc+int(cnt)) {
					/* Cannot be in this range.  */
					cp += 2 * int(nhere)
					if !locfileAlignedP(1 + 2*nhere) {
						cp += int(locfileAlign - (1+2*nhere)%locfileAlign)
					}
					continue
				}

				/* Test against the end of the range.  */
				for cnt = 0; cnt < nhere; cnt++ {
					if extra[cp+int(nhere)+int(cnt)] != at(s, usrc+int(cnt)) {
						break
					}
				}

				if cnt != nhere && extra[cp+int(nhere)+int(cnt)] < at(s, usrc+int(cnt)) {
					/* Cannot be in this range.  */
					cp += 2 * int(nhere)
					if !locfileAlignedP(1 + 2*nhere) {
						cp += int(locfileAlign - (1+2*nhere)%locfileAlign)
					}
					continue
				}

				/* This range matches the next characters.  Now find
				   the offset in the indirect table.  */
				for cnt = 0; extra[cp+int(cnt)] == at(s, usrc+int(cnt)); cnt++ {
				}

				for {
					offset <<= 8
					// The C code adds an int to a size_t, so a negative
					// difference wraps modulo 2^64.
					offset += uint64(int64(at(s, usrc+int(cnt))) - int64(extra[cp+int(cnt)]))
					cnt++
					if cnt >= nhere {
						break
					}
				}
			}

			*cpp += int(nhere)
			return indirect[uint64(int64(-i))+offset]
		}
	}
}

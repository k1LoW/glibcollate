/* Copyright (C) 2000-2025 Free Software Foundation, Inc.
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

// Modified by k1LoW on 2026-10-08. This file is a Go port of the mapping
// table lookup, wctrans_table_lookup, in wctype/wchar-lookup.h from glibc 2.41.
// The structure follows the C source so the two can be compared line by line.
// The bit and byte table lookups of the same file are not ported, since
// nothing in this module reads character classes or widths yet.

package wctype

/* Tables indexed by a wide character are compressed through the use
   of a multi-level lookup.  The compression effect comes from blocks
   that don't need particular data and from blocks that can share their
   data.  */

/* Bit tables are accessed by cutting wc in four blocks of bits:
   - the high 32-q-p bits,
   - the next q bits,
   - the next p bits,
   - the next 5 bits.

	    +------------------+-----+-----+-----+
     wc  =  +     32-q-p-5     |  q  |  p  |  5  |
	    +------------------+-----+-----+-----+

   p and q are variable.  For 16-bit Unicode it is sufficient to
   choose p and q such that q+p+5 <= 16.

   The table contains the following uint32_t words:
   - q+p+5,
   - s = upper exclusive bound for wc >> (q+p+5),
   - p+5,
   - 2^q-1,
   - 2^p-1,
   - 1st-level table: s offsets, pointing into the 2nd-level table,
   - 2nd-level table: k*2^q offsets, pointing into the 3rd-level table,
   - 3rd-level table: j*2^p words, each containing 32 bits of data.
*/

// word reads the uint32 that the C code reads with
// ((const uint32_t *) (table + off))[i]. The tables come from a little-endian
// host, which the generator checks.
func word(table string, off, i uint32) uint32 {
	p := uint64(off) + 4*uint64(i)
	return uint32(table[p]) | uint32(table[p+1])<<8 | uint32(table[p+2])<<16 | uint32(table[p+3])<<24
}

/* Mapping tables are similar to bit tables, except that the
   addressing unit is a single signed 32-bit word, containing the
   difference between the desired result and the argument, and no 5
   bits are used as a word index.  */

func wctransTableLookup(table string, wc uint32) uint32 {
	shift1 := word(table, 0, 0)
	index1 := wc >> shift1
	bound := word(table, 0, 1)
	if index1 < bound {
		lookup1 := word(table, 0, 5+index1)
		if lookup1 != 0 {
			shift2 := word(table, 0, 2)
			mask2 := word(table, 0, 3)
			index2 := (wc >> shift2) & mask2
			lookup2 := word(table, lookup1, index2)
			if lookup2 != 0 {
				mask3 := word(table, 0, 4)
				index3 := wc & mask3
				lookup3 := int32(word(table, lookup2, index3))
				return wc + uint32(lookup3)
			}
		}
	}
	return wc
}

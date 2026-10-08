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

// Modified by k1LoW on 2026-10-08. This file is a Go port of __towlower_l
// and __towupper_l in wctype/wcfuncs_l.c from glibc 2.41.
// The isw* class functions of the same file are not ported.

package wctype

/* LC_CTYPE specific:
   Hardwired indices for standard wide character translation mappings.  */
const (
	towToupper = 0
	towTolower = 1
)

// Table holds the LC_CTYPE items that towlower_l and towupper_l read.
type Table struct {
	// Maps are the items from _NL_CTYPE_MAP_OFFSET on, the first two of
	// them, toupper and tolower.
	Maps [2]string
}

func towlowerL(wc uint32, t *Table) uint32 {
	desc := t.Maps[towTolower]
	return wctransTableLookup(desc, wc)
}

func towupperL(wc uint32, t *Table) uint32 {
	desc := t.Maps[towToupper]
	return wctransTableLookup(desc, wc)
}

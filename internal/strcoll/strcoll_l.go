/* Copyright (C) 1995-2025 Free Software Foundation, Inc.
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

// Modified by k1LoW on 2026-10-08. This file is a Go port of
// string/strcoll_l.c (the narrow-character instantiation, __strcoll_l) from
// glibc 2.41.
// The structure follows the C source so the two can be compared line by line.
// Strings are read as NUL-terminated C strings. An embedded NUL ends the string
// as it does for strcoll, and reads past the end of a Go string yield NUL.

package strcoll

// Values of the rulesets bytes, from enum coll_sort_rule in locale/localeinfo.h.
const (
	sortForward  = 1 << 0
	sortBackward = 1 << 1
	sortPosition = 1 << 2
)

// sizeMax is ~0ul, the sentinel the C code stores in backw and backw_stop.
const sizeMax = ^uint64(0)

// Table holds the LC_COLLATE items that strcoll reads.
type Table struct {
	NRules     uint32      // _NL_COLLATE_NRULES
	Rulesets   string      // _NL_COLLATE_RULESETS
	TableMB    *[256]int32 // _NL_COLLATE_TABLEMB
	WeightMB   string      // _NL_COLLATE_WEIGHTMB
	ExtraMB    string      // _NL_COLLATE_EXTRAMB
	IndirectMB []int32     // _NL_COLLATE_INDIRECTMB
}

// Compare returns what strcoll_l returns for s1 and s2 under t.
func (t *Table) Compare(s1, s2 string) int {
	return Strcoll(s1, s2, t)
}

/* Track status while looking for sequences in a string.  */
type collSeq struct {
	len       int    /* Length of the current sequence.  */
	val       uint64 /* Position of the sequence relative to the previous non-ignored sequence.  */
	idxmax    uint64 /* Maximum index in sequences.  */
	idxcnt    uint64 /* Current count of indices.  */
	backw     uint64 /* Current Backward sequence index.  */
	backwStop uint64 /* Index where the backward sequences stop.  */
	us        int    /* The string.  */
	rule      byte   /* Saved rule for the first sequence.  */
	idx       int32  /* Index to weight of the current sequence.  */
	saveIdx   int32  /* Save looked up index of a forward sequence after the last backward sequence.  */
	backUs    int    /* Beginning of the backward sequence.  */
}

/* Get next sequence.  Traverse the string as required.  */
func getNextSeq(seq *collSeq, s string, nrules uint32, rulesets string,
	weights string, table *[256]int32,
	extra string, indirect []int32,
	pass uint32) {
	var val uint64
	seq.val = 0
	length := seq.len
	backwStop := seq.backwStop
	backw := seq.backw
	idxcnt := seq.idxcnt
	idxmax := seq.idxmax
	idx := seq.idx
	us := seq.us

	for length == 0 {
		val++
		if backwStop != sizeMax {
			/* There is something pushed.  */
			if backw == backwStop {
				/* The last pushed character was handled.  Continue
				   with forward characters.  */
				if idxcnt < idxmax {
					idx = seq.saveIdx
					backwStop = sizeMax
				} else {
					/* Nothing anymore.  The backward sequence ended with
					   the last sequence in the string.  Note that len is
					   still zero.  */
					idx = 0
					break
				}
			} else {
				/* XXX Traverse BACKW sequences from the beginning of
				   BACKW_STOP to get the next sequence.  Is there a quicker way
				   to do this?  */
				i := backwStop
				us = seq.backUs
				for i < backw {
					tmp := findidx(table, indirect, extra, s, &us, sizeMax)
					idx = tmp & 0xffffff
					i++
				}
				backw--
				us = seq.us
			}
		} else {
			backwStop = idxmax
			prevIdx := idx

			for at(s, us) != 0 {
				tmp := findidx(table, indirect, extra, s, &us, sizeMax)
				rule := byte(tmp >> 24)
				prevIdx = idx
				idx = tmp & 0xffffff
				idxcnt = idxmax
				idxmax++

				/* Save the rule for the first sequence.  */
				if idxcnt == 0 {
					seq.rule = rule
				}

				if rulesets[uint32(rule)*nrules+pass]&sortBackward == 0 {
					/* No more backward characters to push.  */
					break
				}
				idxcnt++
			}

			if backwStop >= idxcnt {
				/* No sequence at all or just one.  */
				if idxcnt == idxmax || backwStop > idxcnt {
					/* Note that len is still zero.  */
					break
				}

				backwStop = sizeMax
			} else {
				/* We pushed backward sequences.  If the stream ended with the
				   backward sequence, then we process the last sequence we
				   found.  Otherwise we process the sequence before the last
				   one since the last one was a forward sequence.  */
				seq.backUs = seq.us
				seq.us = us
				backw = idxcnt
				if idxmax > idxcnt {
					backw--
					seq.saveIdx = idx
					idx = prevIdx
				}
				if backw > backwStop {
					backw--
				}
			}
		}

		length = int(weights[idx])
		idx++
		/* Skip over indices of previous levels.  */
		for range pass {
			idx += int32(length)
			length = int(weights[idx])
			idx++
		}
	}

	/* Update the structure.  */
	seq.val = val
	seq.len = length
	seq.backwStop = backwStop
	seq.backw = backw
	seq.idxcnt = idxcnt
	seq.idxmax = idxmax
	seq.us = us
	seq.idx = idx
}

/* Compare two sequences.  */
func doCompare(seq1, seq2 *collSeq, position bool, weights string) int {
	seq1len := seq1.len
	seq2len := seq2.len
	val1 := seq1.val
	val2 := seq2.val
	idx1 := int(seq1.idx)
	idx2 := int(seq2.idx)
	result := 0

	/* Test for position if necessary.  */
	if position && val1 != val2 {
		if val1 > val2 {
			result = 1
		} else {
			result = -1
		}
		goto out
	}

	/* Compare the two sequences.  */
	for {
		if weights[idx1] != weights[idx2] {
			/* The sequences differ.  */
			result = int(weights[idx1]) - int(weights[idx2])
			goto out
		}

		/* Increment the offsets.  */
		idx1++
		idx2++

		seq1len--
		seq2len--
		if seq1len <= 0 || seq2len <= 0 {
			break
		}
	}

	if position && seq1len != seq2len {
		result = seq1len - seq2len
	}

out:
	seq1.len = seq1len
	seq2.len = seq2len
	seq1.idx = int32(idx1)
	seq2.idx = int32(idx2)
	return result
}

// strcmp is the C strcmp on NUL-terminated strings.
func strcmp(s1, s2 string) int {
	for i := 0; ; i++ {
		c1, c2 := at(s1, i), at(s2, i)
		if c1 != c2 {
			return int(c1) - int(c2)
		}
		if c1 == 0 {
			return 0
		}
	}
}

// Strcoll is __strcoll_l with the LC_COLLATE data in l.
func Strcoll(s1, s2 string, l *Table) int {
	nrules := l.NRules

	if nrules == 0 {
		return strcmp(s1, s2)
	}

	/* Catch empty strings.  */
	if at(s1, 0) == 0 || at(s2, 0) == 0 {
		return b2i(at(s1, 0) != 0) - b2i(at(s2, 0) != 0)
	}

	rulesets := l.Rulesets
	table := l.TableMB
	weights := l.WeightMB
	extra := l.ExtraMB
	indirect := l.IndirectMB

	result := 0
	var rule byte

	var seq1, seq2 collSeq
	seq1.len = 0
	seq1.idxmax = 0
	seq1.rule = 0
	seq2.len = 0
	seq2.idxmax = 0

	for pass := range nrules {
		seq1.idxcnt = 0
		seq1.idx = 0
		seq2.idx = 0
		seq1.backwStop = sizeMax
		seq1.backw = sizeMax
		seq2.idxcnt = 0
		seq2.backwStop = sizeMax
		seq2.backw = sizeMax

		/* We need the elements of the strings as unsigned values since they
		   are used as indices.  */
		seq1.us = 0
		seq2.us = 0

		/* We assume that if a rule has defined `position' in one section
		   this is true for all of them.  Please note that the localedef programs
		   makes sure that `position' is not used at the first level.  */

		position := rulesets[uint32(rule)*nrules+pass]&sortPosition != 0

		for {
			getNextSeq(&seq1, s1, nrules, rulesets, weights, table,
				extra, indirect, pass)
			getNextSeq(&seq2, s2, nrules, rulesets, weights, table,
				extra, indirect, pass)
			/* See whether any or both strings are empty.  */
			if seq1.len == 0 || seq2.len == 0 {
				if seq1.len == seq2.len {
					/* Both strings ended and are equal at this level.  Do a
					   byte-level comparison to ensure that we don't waste time
					   going through multiple passes for totally equal strings
					   before proceeding to subsequent passes.  */
					if pass == 0 && strcmp(s1, s2) == 0 {
						return result
					}
					break
				}

				/* This means one string is shorter than the other.  Find out
				   which one and return an appropriate value.  */
				if seq1.len == 0 {
					return -1
				}
				return 1
			}

			result = doCompare(&seq1, &seq2, position, weights)
			if result != 0 {
				return result
			}
		}

		rule = seq1.rule
	}

	return result
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

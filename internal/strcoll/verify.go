package strcoll

import "fmt"

// CheckBounded reports whether findidx can step past the terminating NUL of a
// string with this table. glibc reads memory past the NUL in that case, which
// this port cannot reproduce. It cannot happen when every byte sequence in the
// multibyte extra table is free of NUL and every range of characters differs
// from its start only in the last byte, because then the comparison stops at
// the NUL and rejects the range before the offset is computed.
func CheckBounded(t *Table) error {
	if t.NRules == 0 {
		return nil
	}
	extra := t.ExtraMB
	noNUL := func(b string) bool {
		for i := 0; i < len(b); i++ {
			if b[i] == 0 {
				return false
			}
		}
		return true
	}
	for lead, i := range t.TableMB {
		if i >= 0 {
			continue
		}
		cp := int(-i)
		for {
			idx := le32(extra, cp)
			nhere := int(extra[cp+4])
			cp += 5
			if idx >= 0 {
				if !noNUL(extra[cp : cp+nhere]) {
					return fmt.Errorf("sequence after byte %#x contains NUL", lead)
				}
				if nhere == 0 {
					break
				}
				cp += nhere
				if !locfileAlignedP(uint64(1 + nhere)) {
					cp += locfileAlign - (1+nhere)%locfileAlign
				}
				continue
			}
			start, end := extra[cp:cp+nhere], extra[cp+nhere:cp+2*nhere]
			if !noNUL(start) || !noNUL(end) {
				return fmt.Errorf("range after byte %#x contains NUL", lead)
			}
			if nhere > 0 && start[:nhere-1] != end[:nhere-1] {
				return fmt.Errorf("range %+q..%+q after byte %#x differs before its last byte", start, end, lead)
			}
			cp += 2 * nhere
			if !locfileAlignedP(uint64(1 + 2*nhere)) {
				cp += locfileAlign - (1+2*nhere)%locfileAlign
			}
		}
	}
	return nil
}

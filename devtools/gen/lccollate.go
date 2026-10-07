package main

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// LC_COLLATE item indices, from the _NL_COLLATE_* enum in locale/langinfo.h.
// categories.def omits the GAP items, so its order cannot be used.
const (
	itemNRules = iota
	itemRulesets
	itemTableMB
	itemWeightMB
	itemExtraMB
	itemIndirectMB
	itemGap1
	itemGap2
	itemGap3
	itemTableWC
	itemWeightWC
	itemExtraWC
	itemIndirectWC
	itemSymbHashSizeMB
	itemSymbTableMB
	itemSymbExtraMB
	itemCollSeqMB
	itemCollSeqWC
	itemCodeset
	numItems
)

// collateMagic is LIMAGIC (LC_COLLATE) from locale/localeinfo.h.
const collateMagic = 0x20051014 ^ lcCollate

// lcCollateData holds the items strcoll reads. The wide-character tables,
// the symbol hash and the collation sequence tables serve wcscoll, regex and
// fnmatch, so they are left out.
type lcCollateData struct {
	NRules     uint32
	Rulesets   []byte
	TableMB    [256]int32
	WeightMB   []byte
	ExtraMB    []byte
	IndirectMB []int32
	Codeset    string
}

// parseLCCollate reads a compiled LC_COLLATE file as _nl_intern_locale_data in
// locale/loadlocale.c does. Each item runs from its offset to the next one,
// so a table may carry the alignment padding that localedef appends.
func parseLCCollate(data []byte) (*lcCollateData, error) {
	le := binary.LittleEndian
	if len(data) < 8 {
		return nil, errors.New("LC_COLLATE too short")
	}
	if m := le.Uint32(data[0:]); m != collateMagic {
		return nil, fmt.Errorf("bad LC_COLLATE magic %#x, want %#x (big-endian data is not supported)", m, collateMagic)
	}
	nstrings := le.Uint32(data[4:])
	if nstrings < numItems || 8+4*int(nstrings) >= len(data) {
		return nil, fmt.Errorf("LC_COLLATE has %d items, want at least %d", nstrings, numItems)
	}
	offsets := make([]int, nstrings+1)
	for i := range int(nstrings) {
		offsets[i] = int(le.Uint32(data[8+4*i:]))
		if offsets[i] > len(data) {
			return nil, fmt.Errorf("item %d offset out of range", i)
		}
	}
	offsets[nstrings] = len(data)
	item := func(i int) []byte { return data[offsets[i]:offsets[i+1]] }
	int32s := func(b []byte) []int32 {
		out := make([]int32, len(b)/4)
		for i := range out {
			out[i] = int32(le.Uint32(b[4*i:])) //nolint:gosec // reinterprets the bits as the C code does
		}
		return out
	}

	d := &lcCollateData{
		NRules:     le.Uint32(item(itemNRules)),
		Rulesets:   item(itemRulesets),
		WeightMB:   item(itemWeightMB),
		ExtraMB:    item(itemExtraMB),
		IndirectMB: int32s(item(itemIndirectMB)),
		Codeset:    cstring(item(itemCodeset)),
	}
	if d.NRules == 0 {
		return d, nil
	}
	table := int32s(item(itemTableMB))
	if len(table) != 256 {
		return nil, fmt.Errorf("TABLEMB has %d entries, want 256", len(table))
	}
	copy(d.TableMB[:], table)
	return d, nil
}

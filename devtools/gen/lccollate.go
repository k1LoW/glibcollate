package main

import (
	"encoding/binary"
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

// parseLCCollate reads a compiled LC_COLLATE file.
func parseLCCollate(data []byte) (*lcCollateData, error) {
	le := binary.LittleEndian
	items, err := lcItems(data, "LC_COLLATE", collateMagic, numItems)
	if err != nil {
		return nil, err
	}
	item := func(i int) []byte { return items[i] }
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

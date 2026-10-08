package main

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// lcItems splits a compiled locale category file into its items as
// _nl_intern_locale_data in locale/loadlocale.c does. Each item runs from its
// offset to the next one, so a table may carry the alignment padding that
// localedef appends.
func lcItems(data []byte, category string, magic uint32, want int) ([][]byte, error) {
	le := binary.LittleEndian
	if len(data) < 8 {
		return nil, fmt.Errorf("%s too short", category)
	}
	if m := le.Uint32(data[0:]); m != magic {
		return nil, fmt.Errorf("bad %s magic %#x, want %#x (big-endian data is not supported)", category, m, magic)
	}
	nstrings := le.Uint32(data[4:])
	if nstrings < uint32(want) || 8+4*int(nstrings) >= len(data) { //nolint:gosec // want is a small constant
		return nil, fmt.Errorf("%s has %d items, want at least %d", category, nstrings, want)
	}
	offsets := make([]int, nstrings+1)
	for i := range int(nstrings) {
		offsets[i] = int(le.Uint32(data[8+4*i:]))
		if offsets[i] > len(data) {
			return nil, fmt.Errorf("%s item %d offset out of range", category, i)
		}
	}
	offsets[nstrings] = len(data)
	items := make([][]byte, nstrings)
	for i := range items {
		if offsets[i] > offsets[i+1] {
			return nil, errors.New(category + " items out of order")
		}
		items[i] = data[offsets[i]:offsets[i+1]]
	}
	return items, nil
}

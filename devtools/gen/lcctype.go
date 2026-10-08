package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// LC_CTYPE item indices, from the _NL_CTYPE_* enum in locale/langinfo.h.
// CODESET is an alias of _NL_CTYPE_CODESET_NAME and takes no index of its own.
const (
	itemCtypeMapNames  = 11 // _NL_CTYPE_MAP_NAMES
	itemCtypeMapOffset = 18 // _NL_CTYPE_MAP_OFFSET
)

// ctypeMagic is LIMAGIC (LC_CTYPE) from locale/localeinfo.h.
const ctypeMagic = 0x20090720 ^ lcCtype

// lcCtypeData holds the items towlower and towupper read.
type lcCtypeData struct {
	ToUpper []byte // item _NL_CTYPE_MAP_OFFSET + __TOW_toupper
	ToLower []byte // item _NL_CTYPE_MAP_OFFSET + __TOW_tolower
}

// parseLCCtype reads a compiled LC_CTYPE file.
func parseLCCtype(data []byte) (*lcCtypeData, error) {
	items, err := lcItems(data, "LC_CTYPE", ctypeMagic, itemCtypeMapOffset+1)
	if err != nil {
		return nil, err
	}
	// __TOW_toupper and __TOW_tolower are hardwired to 0 and 1 in
	// locale/localeinfo.h, and localedef writes the maps in the order of
	// their names, so the names confirm the indices.
	if !bytes.HasPrefix(items[itemCtypeMapNames], []byte("toupper\x00tolower\x00")) {
		return nil, fmt.Errorf("LC_CTYPE map names %q do not start with toupper and tolower", items[itemCtypeMapNames])
	}
	if len(items[itemCtypeMapOffset]) < 4 {
		return nil, fmt.Errorf("LC_CTYPE map offset item too short")
	}
	off := int(binary.LittleEndian.Uint32(items[itemCtypeMapOffset]))
	if off+1 >= len(items) {
		return nil, fmt.Errorf("LC_CTYPE map offset %d out of range of %d items", off, len(items))
	}
	return &lcCtypeData{ToUpper: items[off], ToLower: items[off+1]}, nil
}

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// Layout of locale/locarchive.h in glibc.
const (
	arMagic     = 0xde020109
	lcCollate   = 3  // __LC_COLLATE
	lcLast      = 13 // __LC_LAST
	headerSize  = 14 * 4
	namehashEnt = 3 * 4
)

// extractFromArchive returns the LC_COLLATE file stored in a locale-archive for
// the locale name, which must already be normalized (such as en_US.utf8).
// It scans the name hash table instead of probing it, since a linear scan
// gives the same answer and needs no copy of glibc's hash function.
func extractFromArchive(archive []byte, name string) ([]byte, error) {
	le := binary.LittleEndian
	if len(archive) < headerSize {
		return nil, errors.New("locale-archive too short")
	}
	if m := le.Uint32(archive[0:]); m != arMagic {
		return nil, fmt.Errorf("bad locale-archive magic %#x", m)
	}
	namehashOffset := le.Uint32(archive[8:])
	namehashSize := le.Uint32(archive[16:])
	stringOffset := le.Uint32(archive[20:])
	_ = stringOffset

	var names []string
	for i := range namehashSize {
		ent := archive[namehashOffset+i*namehashEnt:]
		nameOffset := le.Uint32(ent[4:])
		locrecOffset := le.Uint32(ent[8:])
		if locrecOffset == 0 {
			continue
		}
		n := cstring(archive[nameOffset:])
		names = append(names, n)
		if n != name {
			continue
		}
		rec := archive[locrecOffset+4+lcCollate*8:]
		off, size := le.Uint32(rec[0:]), le.Uint32(rec[4:])
		if uint64(off)+uint64(size) > uint64(len(archive)) {
			return nil, fmt.Errorf("LC_COLLATE record of %s out of range", name)
		}
		return archive[off : off+size], nil
	}
	return nil, fmt.Errorf("locale %s not in locale-archive (have %s)", name, strings.Join(names, ", "))
}

func cstring(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

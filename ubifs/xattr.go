package ubifs

import (
	"encoding/binary"
	"fmt"
)

// XAttrEntry represents an extended attribute
type XAttrEntry struct {
	Name  string
	Value []byte
	Inode uint64
}

// parseXEntry parses a UBIFS extended attribute entry
func (r *Reader) parseXEntry(data []byte) {
	if len(data) < 48 { return }
	
	inum := binary.LittleEndian.Uint64(data[32:40])
	nlen := binary.LittleEndian.Uint16(data[44:46])
	
	if int(nlen)+48 > len(data) { return }
	
	name := string(data[48:48+int(nlen)])
	parentInum := binary.LittleEndian.Uint32(data[24:28])
	
	entry := XAttrEntry{
		Name:  name,
		Inode: inum,
	}
	
	// Store xattr data
	if r.xattrs == nil {
		r.xattrs = make(map[uint64][]XAttrEntry)
	}
	r.xattrs[uint64(parentInum)] = append(r.xattrs[uint64(parentInum)], entry)
}

// GetXAttrs returns extended attributes for an inode
func (r *Reader) GetXAttrs(inum uint64) []XAttrEntry {
	if r.xattrs == nil { return nil }
	return r.xattrs[inum]
}

// FormatXAttrs returns a string representation of xattrs
func FormatXAttrs(attrs []XAttrEntry) string {
	if len(attrs) == 0 { return "" }
	s := ""
	for _, a := range attrs {
		s += fmt.Sprintf("    %s = %s\n", a.Name, string(a.Value))
	}
	return s
}

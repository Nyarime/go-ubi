package ubifs

import (
	"encoding/binary"
	"fmt"
)

// OrphanNode represents a UBIFS orphan node (inodes without directory entries)
type OrphanNode struct {
	Inodes []uint64
}

// parseOrphan handles orphan area nodes
func (r *Reader) parseOrphan(data []byte) {
	if len(data) < 28 { return }
	
	// Orphan node contains a list of inode numbers to delete
	count := (len(data) - 24) / 8
	if count <= 0 { return }
	
	for i := 0; i < count && 24+i*8+8 <= len(data); i++ {
		inum := binary.LittleEndian.Uint64(data[24+i*8 : 24+i*8+8])
		if inum > 0 {
			r.orphans = append(r.orphans, inum)
		}
	}
}

// GetOrphans returns list of orphaned inodes
func (r *Reader) GetOrphans() []uint64 {
	return r.orphans
}

// CleanOrphans removes orphaned inodes from the filesystem
func (r *Reader) CleanOrphans() int {
	cleaned := 0
	for _, inum := range r.orphans {
		if _, ok := r.inodes[inum]; ok {
			delete(r.inodes, inum)
			delete(r.fileData, inum)
			cleaned++
		}
	}
	return cleaned
}

// FormatOrphans returns orphan info string
func FormatOrphans(orphans []uint64) string {
	if len(orphans) == 0 { return "No orphaned inodes\n" }
	return fmt.Sprintf("%d orphaned inodes (cleaned during extraction)\n", len(orphans))
}

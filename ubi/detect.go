package ubi

import (
	"encoding/binary"
	"os"
)

// IsUBIImage checks if a file starts with UBI EC header magic
func IsUBIImage(path string) bool {
	f, err := os.Open(path)
	if err != nil { return false }
	defer f.Close()
	
	buf := make([]byte, 4)
	if _, err := f.Read(buf); err != nil { return false }
	
	return binary.BigEndian.Uint32(buf) == UBI_EC_HDR_MAGIC
}

// IsUBIData checks if data starts with UBI EC header magic
func IsUBIData(data []byte) bool {
	if len(data) < 4 { return false }
	return binary.BigEndian.Uint32(data) == UBI_EC_HDR_MAGIC
}

// FindUBIOffset searches for UBI magic in firmware data
func FindUBIOffset(data []byte) int {
	magic := []byte{0x55, 0x42, 0x49, 0x23} // "UBI#"
	for i := 0; i+4 <= len(data); i += 4 { // UBI headers are 4-byte aligned
		if data[i] == magic[0] && data[i+1] == magic[1] &&
			data[i+2] == magic[2] && data[i+3] == magic[3] {
			return i
		}
	}
	return -1
}

// FindUBIFSOffset searches for UBIFS magic in data
func FindUBIFSOffset(data []byte) int {
	// UBIFS node magic: 0x06101831 (little-endian)
	for i := 0; i+4 <= len(data); i++ {
		if data[i] == 0x31 && data[i+1] == 0x18 && data[i+2] == 0x10 && data[i+3] == 0x06 {
			return i
		}
	}
	return -1
}

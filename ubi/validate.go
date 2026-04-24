package ubi

import (
	"encoding/binary"
	"fmt"
	"os"
)

// ValidateUBIImage checks if a UBI image file is complete and valid
func ValidateUBIImage(path string) (*ValidationResult, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()

	info, _ := f.Stat()
	size := info.Size()

	result := &ValidationResult{
		FileSize: size,
	}

	// Read first EC header to get PEB size
	buf := make([]byte, UBI_EC_HDR_SIZE)
	if _, err := f.ReadAt(buf, 0); err != nil {
		return nil, fmt.Errorf("cannot read EC header: %v", err)
	}

	magic := binary.BigEndian.Uint32(buf[:4])
	if magic != UBI_EC_HDR_MAGIC {
		return nil, fmt.Errorf("not a UBI image (magic: 0x%X)", magic)
	}

	result.Version = buf[4]
	result.VIDHdrOff = int(binary.BigEndian.Uint32(buf[16:20]))
	result.DataOff = int(binary.BigEndian.Uint32(buf[20:24]))
	result.ImageSeq = binary.BigEndian.Uint32(buf[24:28])

	// Detect PEB size
	pebSize := 0
	for _, candidate := range []int{128*1024, 64*1024, 256*1024, 512*1024} {
		if int64(candidate) > size { continue }
		checkBuf := make([]byte, 4)
		f.ReadAt(checkBuf, int64(candidate))
		if binary.BigEndian.Uint32(checkBuf) == UBI_EC_HDR_MAGIC {
			pebSize = candidate
			break
		}
	}
	if pebSize == 0 { pebSize = 128 * 1024 }
	result.PEBSize = pebSize

	// Check alignment
	result.TotalPEBs = int(size) / pebSize
	result.Remainder = int(size) % pebSize
	result.BlockAligned = result.Remainder == 0

	// Count valid PEBs
	for i := 0; i < result.TotalPEBs; i++ {
		off := int64(i) * int64(pebSize)
		checkBuf := make([]byte, 4)
		f.ReadAt(checkBuf, off)
		if binary.BigEndian.Uint32(checkBuf) == UBI_EC_HDR_MAGIC {
			result.ValidPEBs++
		} else {
			result.BadPEBs++
		}
	}

	// Check image sequence consistency
	result.Consistent = result.ValidPEBs > 0 && result.BadPEBs < result.TotalPEBs/10

	return result, nil
}

// ValidationResult holds UBI image validation results
type ValidationResult struct {
	FileSize     int64
	PEBSize      int
	TotalPEBs    int
	ValidPEBs    int
	BadPEBs      int
	Remainder    int
	BlockAligned bool
	Consistent   bool
	Version      uint8
	ImageSeq     uint32
	VIDHdrOff    int
	DataOff      int
}

func (v *ValidationResult) String() string {
	s := fmt.Sprintf("  File:    %d bytes (%dMB)\n", v.FileSize, v.FileSize/1024/1024)
	s += fmt.Sprintf("  PEB:     %dKB\n", v.PEBSize/1024)
	s += fmt.Sprintf("  PEBs:    %d total, %d valid, %d bad\n", v.TotalPEBs, v.ValidPEBs, v.BadPEBs)
	if !v.BlockAligned {
		s += fmt.Sprintf("  ⚠️  NOT block aligned (remainder: %d bytes)\n", v.Remainder)
	}
	if !v.Consistent {
		s += "  ⚠️  Image may be corrupted or incomplete\n"
	}
	return s
}

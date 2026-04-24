package ubi

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// VolumeTableRecord represents a UBI volume table entry
type VolumeTableRecord struct {
	ReservedPEBs uint32
	Alignment    uint32
	DataPad      uint32
	VolType      uint8
	UpdateMarker uint8
	NameLen      uint16
	Name         [128]byte
	Flags        uint8
	Padding      [23]byte
	CRC          uint32
}

const (
	UBI_VTBL_RECORD_SIZE = 172
	UBI_INTERNAL_VOL_START = 0x7FFFEF00
	UBI_LAYOUT_VOL_ID = UBI_INTERNAL_VOL_START
)

// ParseVolumeTable reads the volume table from the layout volume
func (r *Reader) ParseVolumeTable() ([]VolumeTableRecord, error) {
	if r.image == nil {
		return nil, fmt.Errorf("call Parse() first")
	}

	// Layout volume is the internal volume with highest ID
	var layoutData []byte
	for id, vol := range r.image.Volumes {
		if id >= 0x7FFFEF00 {
			// This is an internal volume (layout volume)
			// Collect all LEBs
			for i := 0; i < len(vol.LEBs); i++ {
				if data, ok := vol.LEBs[i]; ok {
					layoutData = append(layoutData, data...)
				}
			}
			break
		}
	}

	// Also check volume 0 LEB 0 for vtbl records
	if len(layoutData) == 0 {
		// Try reading from beginning of image
		if len(r.image.Volumes) > 0 {
			for _, vol := range r.image.Volumes {
				if data, ok := vol.LEBs[0]; ok && len(data) >= UBI_VTBL_RECORD_SIZE {
					layoutData = data
					break
				}
			}
		}
	}

	if len(layoutData) < UBI_VTBL_RECORD_SIZE {
		return nil, fmt.Errorf("no volume table found")
	}

	var records []VolumeTableRecord
	for offset := 0; offset+UBI_VTBL_RECORD_SIZE <= len(layoutData); offset += UBI_VTBL_RECORD_SIZE {
		var rec VolumeTableRecord
		rec.ReservedPEBs = binary.BigEndian.Uint32(layoutData[offset : offset+4])
		rec.Alignment = binary.BigEndian.Uint32(layoutData[offset+4 : offset+8])
		rec.DataPad = binary.BigEndian.Uint32(layoutData[offset+8 : offset+12])
		rec.VolType = layoutData[offset+12]
		rec.UpdateMarker = layoutData[offset+13]
		rec.NameLen = binary.BigEndian.Uint16(layoutData[offset+14 : offset+16])
		copy(rec.Name[:], layoutData[offset+16:offset+144])
		rec.Flags = layoutData[offset+144]
		rec.CRC = binary.BigEndian.Uint32(layoutData[offset+168 : offset+172])

		if rec.ReservedPEBs > 0 && rec.NameLen > 0 {
			name := string(rec.Name[:rec.NameLen])
			records = append(records, rec)
			
			// Assign name to volume
			volIdx := len(records) - 1
			if vol, ok := r.image.Volumes[volIdx]; ok {
				vol.Name = name
			}
		}
	}

	return records, nil
}

// PrintVolumeTable prints a formatted volume table
func PrintVolumeTable(records []VolumeTableRecord) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %-4s %-20s %-8s %-6s %-10s\n", "ID", "Name", "Type", "PEBs", "Alignment"))
	sb.WriteString(fmt.Sprintf("  %-4s %-20s %-8s %-6s %-10s\n", "──", "────", "────", "────", "─────────"))

	for i, rec := range records {
		name := strings.TrimRight(string(rec.Name[:rec.NameLen]), "\x00")
		volType := "dynamic"
		if rec.VolType == UBI_VID_STATIC {
			volType = "static"
		}
		sb.WriteString(fmt.Sprintf("  %-4d %-20s %-8s %-6d %-10d\n",
			i, name, volType, rec.ReservedPEBs, rec.Alignment))
	}

	return sb.String()
}

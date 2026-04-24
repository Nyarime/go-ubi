package ubi

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
)

// Writer creates UBI images
type Writer struct {
	PEBSize   int
	MinIOSize int // Usually 2048 for NAND
	volumes   []writerVolume
}

type writerVolume struct {
	ID   int
	Name string
	Type uint8
	Data []byte
}

// NewWriter creates a UBI image writer
func NewWriter(pebSize, minIOSize int) *Writer {
	if pebSize == 0 { pebSize = 128 * 1024 }
	if minIOSize == 0 { minIOSize = 2048 }
	return &Writer{
		PEBSize:   pebSize,
		MinIOSize: minIOSize,
	}
}

// AddVolume adds a volume to the image
func (w *Writer) AddVolume(id int, name string, volType uint8, data []byte) {
	w.volumes = append(w.volumes, writerVolume{
		ID: id, Name: name, Type: volType, Data: data,
	})
}

// AddVolumeFromFile adds a volume from a file
func (w *Writer) AddVolumeFromFile(id int, name string, volType uint8, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	w.AddVolume(id, name, volType, data)
	return nil
}

// Write creates the UBI image file
func (w *Writer) Write(outputPath string) error {
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	lebSize := w.PEBSize - w.MinIOSize*2 // EC hdr + VID hdr space
	imageSeq := uint32(1)

	totalPEBs := 0

	// Write layout volume (volume table) first
	vtblData := w.buildVolumeTable()
	vtblPEBs := (len(vtblData) + lebSize - 1) / lebSize
	if vtblPEBs < 2 { vtblPEBs = 2 } // UBI requires at least 2 copies

	for i := 0; i < vtblPEBs; i++ {
		peb := w.buildPEB(UBI_LAYOUT_VOL_ID, uint8(UBI_VID_STATIC), i, imageSeq, vtblData, lebSize)
		out.Write(peb)
		totalPEBs++
	}

	// Write data volumes
	for _, vol := range w.volumes {
		numLEBs := (len(vol.Data) + lebSize - 1) / lebSize
		if numLEBs == 0 { numLEBs = 1 }

		for i := 0; i < numLEBs; i++ {
			start := i * lebSize
			end := start + lebSize
			if end > len(vol.Data) { end = len(vol.Data) }
			
			chunk := vol.Data[start:end]
			peb := w.buildPEB(uint32(vol.ID), vol.Type, i, imageSeq, chunk, lebSize)
			out.Write(peb)
			totalPEBs++
		}
	}

	fmt.Printf("  📦 UBI image written: %d PEBs, %dMB\n", totalPEBs, totalPEBs*w.PEBSize/1024/1024)
	return nil
}

func (w *Writer) buildPEB(volID uint32, volType uint8, lebNum int, imageSeq uint32, data []byte, lebSize int) []byte {
	peb := make([]byte, w.PEBSize)

	// EC Header
	ecHdr := peb[:64]
	binary.BigEndian.PutUint32(ecHdr[0:4], UBI_EC_HDR_MAGIC)
	ecHdr[4] = 1 // version
	binary.BigEndian.PutUint64(ecHdr[8:16], 0) // erase counter = 0 (new)
	binary.BigEndian.PutUint32(ecHdr[16:20], uint32(w.MinIOSize)) // VID hdr offset
	binary.BigEndian.PutUint32(ecHdr[20:24], uint32(w.MinIOSize*2)) // data offset
	binary.BigEndian.PutUint32(ecHdr[24:28], imageSeq)
	// CRC
	crc := crc32.Checksum(ecHdr[:60], ubiCRC32Table)
	binary.BigEndian.PutUint32(ecHdr[60:64], crc)

	// VID Header
	vidHdr := peb[w.MinIOSize : w.MinIOSize+64]
	binary.BigEndian.PutUint32(vidHdr[0:4], UBI_VID_HDR_MAGIC)
	vidHdr[4] = 1 // version
	vidHdr[5] = volType
	binary.BigEndian.PutUint32(vidHdr[8:12], volID)
	binary.BigEndian.PutUint32(vidHdr[12:16], uint32(lebNum))
	if volType == UBI_VID_STATIC {
		binary.BigEndian.PutUint32(vidHdr[20:24], uint32(len(data)))
	}
	// CRC
	vidCRC := crc32.Checksum(vidHdr[:60], ubiCRC32Table)
	binary.BigEndian.PutUint32(vidHdr[60:64], vidCRC)

	// Data
	dataOffset := w.MinIOSize * 2
	copy(peb[dataOffset:], data)

	return peb
}

func (w *Writer) buildVolumeTable() []byte {
	vtbl := make([]byte, UBI_VTBL_RECORD_SIZE*128) // Max 128 volumes

	for _, vol := range w.volumes {
		offset := vol.ID * UBI_VTBL_RECORD_SIZE
		if offset+UBI_VTBL_RECORD_SIZE > len(vtbl) { continue }

		numLEBs := (len(vol.Data) + (w.PEBSize - w.MinIOSize*2) - 1) / (w.PEBSize - w.MinIOSize*2)
		
		binary.BigEndian.PutUint32(vtbl[offset:offset+4], uint32(numLEBs)) // reserved PEBs
		binary.BigEndian.PutUint32(vtbl[offset+4:offset+8], 1)   // alignment
		vtbl[offset+12] = vol.Type
		
		nameBytes := []byte(vol.Name)
		if len(nameBytes) > 127 { nameBytes = nameBytes[:127] }
		binary.BigEndian.PutUint16(vtbl[offset+14:offset+16], uint16(len(nameBytes)))
		copy(vtbl[offset+16:offset+144], nameBytes)

		// CRC
		recCRC := crc32.Checksum(vtbl[offset:offset+168], ubiCRC32Table)
		binary.BigEndian.PutUint32(vtbl[offset+168:offset+172], recCRC)
	}

	return vtbl
}

package ubi

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
)

// Reader parses UBI images
type Reader struct {
	file    *os.File
	size    int64
	image   *Image
}

// NewReader creates a UBI reader from a file
func NewReader(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	
	r := &Reader{
		file: f,
		size: info.Size(),
	}
	
	return r, nil
}

// Close closes the reader
func (r *Reader) Close() error {
	return r.file.Close()
}

// Parse reads the UBI image structure
func (r *Reader) Parse() (img *Image, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("UBI parse panic: %v", rec)
		}
	}()
	return r.parseInternal()
}

func (r *Reader) parseInternal() (*Image, error) {
	img := &Image{
		Volumes: make(map[int]*Volume),
	}
	
	// Find PEB size by scanning for EC headers
	pebSize, err := r.detectPEBSize()
	if err != nil {
		return nil, fmt.Errorf("cannot detect PEB size: %w", err)
	}
	img.PEBSize = pebSize
	
	// Scan all PEBs
	numPEBs := int(r.size) / pebSize
	fmt.Printf("  UBI: %d PEBs, PEB size %dKB\n", numPEBs, pebSize/1024)
	
	for i := 0; i < numPEBs; i++ {
		offset := int64(i) * int64(pebSize)
		
		// Read EC header
		ec, err := r.readECHeader(offset)
		if err != nil {
			continue // Skip bad PEBs
		}
		
		if i == 0 {
			img.Version = ec.Version
			img.ImageSeq = ec.ImageSeq
			img.LEBSize = pebSize - int(ec.DataOff)
			img.MinIOSize = int(ec.VIDHdrOff)
		}
		
		// Read VID header
		vid, err := r.readVIDHeader(offset + int64(ec.VIDHdrOff))
		if err != nil {
			continue // Empty/free PEB
		}
		
		volID := int(vid.VolID)
		if volID > 127 {
			continue // Internal volume, skip
		}
		
		// Create volume if new
		if _, ok := img.Volumes[volID]; !ok {
			img.Volumes[volID] = &Volume{
				ID:      volID,
				Type:    vid.VolType,
				LEBs:    make(map[int][]byte),
				LEBSize: img.LEBSize,
			}
		}
		
		// Read LEB data
		dataOffset := offset + int64(ec.DataOff)
		dataSize := img.LEBSize
		if vid.VolType == UBI_VID_STATIC && vid.DataSize > 0 {
			dataSize = int(vid.DataSize)
		}
		
		if dataOffset+int64(dataSize) > r.size {
			dataSize = int(r.size - dataOffset)
		}
		if dataSize <= 0 { continue }
		data := make([]byte, dataSize)
		n, _ := r.file.ReadAt(data, dataOffset)
		if n < dataSize {
			data = data[:n]
		}
		
		img.Volumes[volID].LEBs[int(vid.LNum)] = data
	}
	
	fmt.Printf("  UBI: %d volumes found\n", len(img.Volumes))
	for id, vol := range img.Volumes {
		fmt.Printf("    Volume %d: %d LEBs (%s)\n", id, len(vol.LEBs),
			func() string {
				if vol.Type == UBI_VID_DYNAMIC { return "dynamic" }
				return "static"
			}())
	}
	
	r.image = img
	return img, nil
}

// ExtractVolume writes a volume's data to a file
func (r *Reader) ExtractVolume(volID int, outputPath string) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("volume extraction panic: %v", rec)
		}
	}()
	if r.image == nil {
		return fmt.Errorf("call Parse() first")
	}
	
	vol, ok := r.image.Volumes[volID]
	if !ok {
		return fmt.Errorf("volume %d not found", volID)
	}
	
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()
	
	// Sort LEBs by number and write sequentially
	lebNums := make([]int, 0, len(vol.LEBs))
	for n := range vol.LEBs {
		lebNums = append(lebNums, n)
	}
	sort.Ints(lebNums)
	
	totalWritten := 0
	for _, n := range lebNums {
		data := vol.LEBs[n]
		if len(data) == 0 { continue }
		written, err := out.Write(data)
		if err != nil {
			return err
		}
		totalWritten += written
	}
	
	fmt.Printf("  Extracted volume %d: %d bytes (%dMB)\n", volID, totalWritten, totalWritten/1024/1024)
	return nil
}

// ExtractAll extracts all volumes to a directory
func (r *Reader) ExtractAll(outputDir string) error {
	if r.image == nil {
		return fmt.Errorf("call Parse() first")
	}
	
	os.MkdirAll(outputDir, 0755)
	
	for id := range r.image.Volumes {
		outPath := fmt.Sprintf("%s/volume_%d.img", outputDir, id)
		if err := r.ExtractVolume(id, outPath); err != nil {
			fmt.Printf("  Warning: volume %d extraction failed: %v\n", id, err)
		}
	}
	
	return nil
}

// detectPEBSize finds the PEB size by looking for consecutive EC headers
func (r *Reader) detectPEBSize() (int, error) {
	// Common PEB sizes
	candidates := []int{
		128 * 1024,  // 128KB (most common)
		256 * 1024,  // 256KB
		64 * 1024,   // 64KB
		512 * 1024,  // 512KB
	}
	
	for _, size := range candidates {
		if int64(size) > r.size {
			continue
		}
		
		// Check if there's an EC header at offset 0 and at offset `size`
		ec0, err := r.readECHeader(0)
		if err != nil {
			continue
		}
		_ = ec0
		
		ec1, err := r.readECHeader(int64(size))
		if err != nil {
			continue
		}
		_ = ec1
		
		return size, nil
	}
	
	return 0, fmt.Errorf("no valid PEB size found")
}

// readECHeader reads an EC header at the given offset
func (r *Reader) readECHeader(offset int64) (*ECHeader, error) {
	var ec ECHeader
	
	buf := make([]byte, UBI_EC_HDR_SIZE)
	if offset < 0 || offset+int64(UBI_EC_HDR_SIZE) > r.size {
		return nil, fmt.Errorf("offset %d out of bounds (size %d)", offset, r.size)
	}
	if _, err := r.file.ReadAt(buf, offset); err != nil {
		return nil, err
	}
	
	ec.Magic = binary.BigEndian.Uint32(buf[0:4])
	if ec.Magic != UBI_EC_HDR_MAGIC {
		return nil, fmt.Errorf("bad EC magic at offset %d: 0x%X", offset, ec.Magic)
	}
	
	ec.Version = buf[4]
	ec.EC = binary.BigEndian.Uint64(buf[8:16])
	ec.VIDHdrOff = binary.BigEndian.Uint32(buf[16:20])
	ec.DataOff = binary.BigEndian.Uint32(buf[20:24])
	ec.ImageSeq = binary.BigEndian.Uint32(buf[24:28])
	ec.HdrCRC = binary.BigEndian.Uint32(buf[60:64])
	
	return &ec, nil
}

// readVIDHeader reads a VID header at the given offset
func (r *Reader) readVIDHeader(offset int64) (*VIDHeader, error) {
	if offset < 0 || offset+int64(UBI_VID_HDR_SIZE) > r.size {
		return nil, fmt.Errorf("VID header offset out of bounds")
	}
	var vid VIDHeader
	
	buf := make([]byte, UBI_VID_HDR_SIZE)
	if _, err := r.file.ReadAt(buf, offset); err != nil {
		return nil, err
	}
	
	vid.Magic = binary.BigEndian.Uint32(buf[0:4])
	if vid.Magic != UBI_VID_HDR_MAGIC {
		return nil, fmt.Errorf("bad VID magic at offset %d: 0x%X", offset, vid.Magic)
	}
	
	vid.Version = buf[4]
	vid.VolType = buf[5]
	vid.CopyFlag = buf[6]
	vid.Compat = buf[7]
	vid.VolID = binary.BigEndian.Uint32(buf[8:12])
	vid.LNum = binary.BigEndian.Uint32(buf[12:16])
	vid.DataSize = binary.BigEndian.Uint32(buf[20:24])
	vid.UsedEBs = binary.BigEndian.Uint32(buf[24:28])
	vid.DataPad = binary.BigEndian.Uint32(buf[28:32])
	vid.DataCRC = binary.BigEndian.Uint32(buf[32:36])
	vid.SqNum = binary.BigEndian.Uint64(buf[40:48])
	vid.HdrCRC = binary.BigEndian.Uint32(buf[60:64])
	
	return &vid, nil
}

// Ensure Reader implements io.Closer
var _ io.Closer = (*Reader)(nil)

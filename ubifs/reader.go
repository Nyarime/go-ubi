package ubifs

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Reader parses UBIFS images
type Reader struct {
	data     []byte
	lebSize  int
	lebCount int
	sb       *SuperBlock
	inodes   map[uint64]*InodeNode
	dentries map[uint64][]DentNode
	fileData map[uint64][]byte
	xattrs   map[uint64][]XAttrEntry
	orphans  []uint64
}

// NewReader creates a UBIFS reader from volume data
func NewReader(data []byte) (*Reader, error) {
	if len(data) < 64 {
		return nil, fmt.Errorf("data too short")
	}

	r := &Reader{
		data:     data,
		inodes:   make(map[uint64]*InodeNode),
		dentries: make(map[uint64][]DentNode),
		fileData: make(map[uint64][]byte),
	}

	return r, nil
}

// NewReaderFromFile creates a UBIFS reader from a file
func NewReaderFromFile(path string) (*Reader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewReader(data)
}

// Parse reads the UBIFS structure
func (r *Reader) Parse() error {
	// Detect LEB size by finding superblock
	if err := r.detectLEBSize(); err != nil {
		return err
	}

	fmt.Printf("  UBIFS: LEB size %dKB, %d LEBs\n", r.lebSize/1024, r.lebCount)

	// Scan all nodes
	nodeCount := 0
	for offset := 0; offset+24 < len(r.data); {
		// Look for UBIFS node magic
		if offset+4 > len(r.data) { break }
		
		magic := binary.LittleEndian.Uint32(r.data[offset:offset+4])
		if magic != UBIFS_NODE_MAGIC {
			offset += 8 // Align to 8 bytes
			continue
		}

		// Read common header
		if offset+24 > len(r.data) { break }
		
		var hdr CommonHeader
		hdr.Magic = magic
		hdr.CRC = binary.LittleEndian.Uint32(r.data[offset+4:offset+8])
		hdr.SqNum = binary.LittleEndian.Uint64(r.data[offset+8:offset+16])
		hdr.Len = binary.LittleEndian.Uint32(r.data[offset+16:offset+20])
		hdr.NodeType = r.data[offset+20]

		if hdr.Len == 0 || hdr.Len > 1024*1024 || int(hdr.Len)+offset > len(r.data) {
			offset += 8
			continue
		}

		nodeData := r.data[offset:offset+int(hdr.Len)]
		
		switch hdr.NodeType {
		case UBIFS_SB_NODE:
			r.parseSuperBlock(nodeData)
		case UBIFS_INO_NODE:
			r.parseInode(nodeData)
		case UBIFS_DENT_NODE:
			r.parseDent(nodeData)
		case UBIFS_DATA_NODE:
			r.parseDataNode(nodeData)
		case UBIFS_XENT_NODE:
			r.parseXEntry(nodeData)
		}

		nodeCount++
		offset += int(hdr.Len)
		// Align to 8 bytes
		if offset%8 != 0 {
			offset += 8 - (offset % 8)
		}
	}

	fmt.Printf("  UBIFS: %d nodes, %d inodes, %d dentries, %d data blocks\n",
		nodeCount, len(r.inodes), len(r.dentries), len(r.fileData))

	return nil
}

// Extract writes all files to output directory
func (r *Reader) Extract(outputDir string) error {
	os.MkdirAll(outputDir, 0755)

	// Build file tree starting from root inode (1)
	return r.extractDir(1, outputDir)
}

func (r *Reader) extractDir(inum uint64, path string) error {
	os.MkdirAll(path, 0755)

	entries, ok := r.dentries[inum]
	if !ok {
		return nil
	}

	for _, dent := range entries {
		name := string(dent.Name)
		if name == "." || name == ".." {
			continue
		}

		fullPath := filepath.Join(path, name)

		switch dent.DType {
		case UBIFS_ITYPE_DIR:
			r.extractDir(dent.INum, fullPath)
		case UBIFS_ITYPE_REG:
			if data, ok := r.fileData[dent.INum]; ok {
				os.WriteFile(fullPath, data, 0644)
			} else {
				// Create empty file
				os.WriteFile(fullPath, []byte{}, 0644)
			}
		case UBIFS_ITYPE_LNK:
			if inode, ok := r.inodes[dent.INum]; ok && inode.DataLen > 0 {
				// Symlink target is in inode data
				// For now, create a placeholder
				os.Symlink(".", fullPath)
			}
		}
	}

	return nil
}

func (r *Reader) detectLEBSize() error {
	// Common LEB sizes
	for _, size := range []int{126976, 253952, 129024, 65408, 130944, 258048} {
		if size < len(r.data) {
			r.lebSize = size
			r.lebCount = len(r.data) / size
			return nil
		}
	}
	// Default: try to find from superblock
	r.lebSize = 126976 // 124KB (128KB PEB - 2*512B headers)
	r.lebCount = len(r.data) / r.lebSize
	return nil
}

func (r *Reader) parseSuperBlock(data []byte) {
	if len(data) < 120 { return }
	sb := &SuperBlock{}
	sb.Header.NodeType = UBIFS_SB_NODE
	sb.MinIOSize = binary.LittleEndian.Uint32(data[28:32])
	sb.LEBSize = binary.LittleEndian.Uint32(data[32:36])
	sb.LEBCount = binary.LittleEndian.Uint32(data[36:40])
	
	if sb.LEBSize > 0 {
		r.lebSize = int(sb.LEBSize)
		r.lebCount = int(sb.LEBCount)
	}
	
	r.sb = sb
	fmt.Printf("  UBIFS Superblock: LEB %dKB, %d LEBs, minIO %d\n",
		sb.LEBSize/1024, sb.LEBCount, sb.MinIOSize)
}

func (r *Reader) parseInode(data []byte) {
	if len(data) < 100 { return }
	ino := &InodeNode{}
	copy(ino.Key[:], data[24:32])
	ino.Size = binary.LittleEndian.Uint64(data[48:56])
	ino.NLink = binary.LittleEndian.Uint32(data[80:84])
	ino.UID = binary.LittleEndian.Uint32(data[84:88])
	ino.GID = binary.LittleEndian.Uint32(data[88:92])
	ino.Mode = binary.LittleEndian.Uint32(data[92:96])
	ino.DataLen = binary.LittleEndian.Uint32(data[104:108])
	ino.Compr = binary.LittleEndian.Uint16(data[116:118])

	// Extract inode number from key
	inum := binary.LittleEndian.Uint32(data[24:28])
	r.inodes[uint64(inum)] = ino
}

func (r *Reader) parseDent(data []byte) {
	if len(data) < 48 { return }
	dent := DentNode{}
	copy(dent.Key[:], data[24:32])
	dent.INum = binary.LittleEndian.Uint64(data[32:40])
	dent.NLen = binary.LittleEndian.Uint16(data[44:46])
	dent.DType = data[46]

	if int(dent.NLen)+48 <= len(data) {
		dent.Name = make([]byte, dent.NLen)
		copy(dent.Name, data[48:48+int(dent.NLen)])
	}

	// Parent inode from key
	parentInum := binary.LittleEndian.Uint32(data[24:28])
	r.dentries[uint64(parentInum)] = append(r.dentries[uint64(parentInum)], dent)
}

func (r *Reader) parseDataNode(data []byte) {
	if len(data) < 44 { return }
	
	inum := binary.LittleEndian.Uint32(data[24:28])
	size := binary.LittleEndian.Uint32(data[36:40])
	compr := binary.LittleEndian.Uint16(data[40:42])

	if size == 0 || 44+int(size) > len(data) { return }
	
	compData := data[44:44+int(size)]
	
	var fileData []byte
	switch compr {
	case UBIFS_COMPR_NONE:
		fileData = make([]byte, len(compData))
		copy(fileData, compData)
	case UBIFS_COMPR_ZLIB:
		if decompressed, err := decompressZlib(compData); err == nil {
			fileData = decompressed
		}
	case UBIFS_COMPR_LZO:
		if decompressed, err := decompressLZO(compData); err == nil && len(decompressed) > 0 {
			fileData = decompressed
		} else {
			fileData = make([]byte, len(compData))
			copy(fileData, compData)
		}
	case UBIFS_COMPR_ZSTD:
		if decompressed, err := decompressZstdSimple(compData); err == nil {
			fileData = decompressed
		}
	case UBIFS_COMPR_LZ4:
		if decompressed, err := lz4BlockDecompress(compData); err == nil {
			fileData = decompressed
		} else {
			fileData = make([]byte, len(compData))
			copy(fileData, compData)
		}
	default:
		fileData = make([]byte, len(compData))
		copy(fileData, compData)
	}

	// Append to file data (files can span multiple data nodes)
	r.fileData[uint64(inum)] = append(r.fileData[uint64(inum)], fileData...)
}

func decompressZlib(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil { return nil, err }
	defer reader.Close()
	return io.ReadAll(reader)
}

// Ensure strings import is used
var _ = strings.TrimSpace

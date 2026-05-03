package ubifs

// UBIFS Magic & constants
const (
	UBIFS_NODE_MAGIC  = 0x06101831
	UBIFS_SB_NODE     = 0 // Superblock
	UBIFS_MST_NODE    = 1 // Master
	UBIFS_REF_NODE    = 2 // Reference
	UBIFS_INO_NODE    = 3 // Inode
	UBIFS_DATA_NODE   = 4 // Data
	UBIFS_DENT_NODE   = 5 // Directory entry
	UBIFS_XENT_NODE   = 6 // Extended attribute entry
	UBIFS_TRUN_NODE   = 7 // Truncation
	UBIFS_PAD_NODE    = 8 // Padding
	UBIFS_IDX_NODE    = 9 // Index
	UBIFS_CS_NODE     = 10 // Commit start
	
	// Compression types
	UBIFS_COMPR_NONE = 0
	UBIFS_COMPR_LZO  = 1
	UBIFS_COMPR_ZLIB = 2
	UBIFS_COMPR_ZSTD = 3
	UBIFS_COMPR_LZ4  = 4
	
	// Inode types
	UBIFS_ITYPE_REG  = 0 // Regular file
	UBIFS_ITYPE_DIR  = 1 // Directory
	UBIFS_ITYPE_LNK  = 2 // Symlink
	UBIFS_ITYPE_BLK  = 3 // Block device
	UBIFS_ITYPE_CHR  = 4 // Char device
	UBIFS_ITYPE_FIFO = 5 // FIFO
	UBIFS_ITYPE_SOCK = 6 // Socket
)

// CommonHeader is the common header for all UBIFS nodes
type CommonHeader struct {
	Magic    uint32 // UBIFS_NODE_MAGIC
	CRC      uint32 // CRC32 of node
	SqNum    uint64 // Sequence number
	Len      uint32 // Node length
	NodeType uint8  // Node type
	GroupType uint8
	Padding  [2]byte
}

// SuperBlock is the UBIFS superblock node
type SuperBlock struct {
	Header       CommonHeader
	KeyHash      uint8
	KeyFmt       uint8
	Flags        uint32
	MinIOSize    uint32
	LEBSize      uint32
	LEBCount     uint32
	MaxLEBCount  uint32
	MaxBudBytes  uint64
	LogLEBs      uint32
	LPTLEBs      uint32
	OrphLEBs     uint32
	JHeadCount   uint32
	FanOut       uint32
	LSaveCount   uint32
	FmtVersion   uint32
	DefaultCompr uint16
	RPSize       uint64
}

// InodeNode represents a UBIFS inode
type InodeNode struct {
	Header   CommonHeader
	Key      [8]byte
	CreatSqNum uint64
	Size     uint64
	ATime    int64
	CTime    int64
	MTime    int64
	NLink    uint32
	UID      uint32
	GID      uint32
	Mode     uint32
	Flags    uint32
	DataLen  uint32
	XAttrCnt uint32
	XAttrSize uint32
	XAttrNames uint32
	Compr    uint16
}

// DentNode represents a UBIFS directory entry
type DentNode struct {
	Header  CommonHeader
	Key     [8]byte
	INum    uint64 // Inode number
	Padding [4]byte
	NLen    uint16 // Name length
	DType   uint8  // Type (file/dir/symlink)
	Name    []byte // Name (variable length)
}

// DataNode represents a UBIFS data node
type DataNode struct {
	Header  CommonHeader
	Key     [8]byte
	Size    uint32
	Compr   uint16
	Data    []byte // Compressed data
}

// FileEntry represents an extracted file
type FileEntry struct {
	Path    string
	Size    uint64
	Mode    uint32
	IsDir   bool
	IsLink  bool
	LinkTarget string
	Data    []byte
}

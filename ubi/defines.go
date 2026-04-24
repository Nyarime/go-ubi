package ubi

// UBI Magic numbers
const (
	UBI_EC_HDR_MAGIC = 0x55424923 // "UBI#"
	UBI_VID_HDR_MAGIC = 0x55424921 // "UBI!"
	
	// Volume types
	UBI_VID_DYNAMIC = 1
	UBI_VID_STATIC  = 2
	
	// Default sizes
	UBI_EC_HDR_SIZE  = 64
	UBI_VID_HDR_SIZE = 64
)

// ECHeader is the Erase Counter header at the start of each PEB
type ECHeader struct {
	Magic      uint32 // UBI_EC_HDR_MAGIC
	Version    uint8
	Padding1   [3]byte
	EC         uint64 // Erase counter
	VIDHdrOff  uint32 // Offset to VID header
	DataOff    uint32 // Offset to data
	ImageSeq   uint32 // Image sequence number
	Padding2   [32]byte
	HdrCRC     uint32 // CRC32 of header
}

// VIDHeader is the Volume Identifier header
type VIDHeader struct {
	Magic      uint32 // UBI_VID_HDR_MAGIC
	Version    uint8
	VolType    uint8  // UBI_VID_DYNAMIC or UBI_VID_STATIC
	CopyFlag   uint8
	Compat     uint8
	VolID      uint32 // Volume ID
	LNum       uint32 // Logical eraseblock number
	Padding1   [4]byte
	DataSize   uint32 // Data size (for static volumes)
	UsedEBs    uint32 // Used logical eraseblocks (for static volumes)
	DataPad    uint32 // Data padding
	DataCRC    uint32 // CRC32 of data
	Padding2   [4]byte
	SqNum      uint64 // Sequence number
	Padding3   [12]byte
	HdrCRC     uint32 // CRC32 of header
}

// Volume represents a UBI volume
type Volume struct {
	ID      int
	Name    string
	Type    uint8
	LEBs    map[int][]byte // LEB number -> data
	LEBSize int
}

// Image represents a complete UBI image
type Image struct {
	Version   uint8
	PEBSize   int
	LEBSize   int
	MinIOSize int
	ImageSeq  uint32
	Volumes   map[int]*Volume
}

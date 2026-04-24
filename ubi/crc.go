package ubi

import (
	"encoding/binary"
	"hash/crc32"
)

// CRC32 table used by UBI (same as Linux kernel)
var ubiCRC32Table = crc32.MakeTable(crc32.IEEE)

// ValidateECHeader checks CRC of EC header
func ValidateECHeader(data []byte) bool {
	if len(data) < UBI_EC_HDR_SIZE { return false }
	
	// CRC is at offset 60, covers bytes 0-59
	storedCRC := binary.BigEndian.Uint32(data[60:64])
	
	// Zero out CRC field for calculation
	calcData := make([]byte, 64)
	copy(calcData, data[:64])
	binary.BigEndian.PutUint32(calcData[60:64], 0)
	
	computed := crc32.Checksum(calcData[:64], ubiCRC32Table)
	return computed == storedCRC
}

// ValidateVIDHeader checks CRC of VID header
func ValidateVIDHeader(data []byte) bool {
	if len(data) < UBI_VID_HDR_SIZE { return false }
	
	storedCRC := binary.BigEndian.Uint32(data[60:64])
	
	calcData := make([]byte, 64)
	copy(calcData, data[:64])
	binary.BigEndian.PutUint32(calcData[60:64], 0)
	
	computed := crc32.Checksum(calcData[:64], ubiCRC32Table)
	return computed == storedCRC
}

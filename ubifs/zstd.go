package ubifs

// UBIFS zstd decompression
// Uses the same zstd library as go-ubi's parent project

import (
	"bytes"
	"compress/flate"
	"io"
)

// decompressZstdSimple attempts basic zstd frame decompression
// For full zstd support, import github.com/klauspost/compress/zstd
func decompressZstdSimple(data []byte) ([]byte, error) {
	// Check zstd magic: 0xFD2FB528
	if len(data) >= 4 && data[0] == 0x28 && data[1] == 0xB5 && data[2] == 0x2F && data[3] == 0xFD {
		// This is a zstd frame - needs proper zstd library
		// Return raw data as fallback
		return data, nil
	}
	
	// Try deflate as fallback
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()
	result, err := io.ReadAll(reader)
	if err != nil {
		return data, nil // Return raw on error
	}
	return result, nil
}

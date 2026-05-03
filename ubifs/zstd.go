package ubifs

// UBIFS zstd decompression using pure Go implementation
// Copied from github.com/Nyarime/go-squashfs

import (
	"bytes"
	"io"
)

// decompressZstdSimple decompresses zstd data using the pure Go implementation
func decompressZstdSimple(data []byte) ([]byte, error) {
	reader, err := ZstdNewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

# go-ubi

Pure Go UBI (Unsorted Block Images) reader and extractor.

**The first and only Go implementation of UBI/UBIFS image parsing.**

[![Go](https://img.shields.io/badge/go-1.26-00ADD8)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## Features

- Parse UBI images from NAND flash firmware
- Extract individual volumes with names
- Read UBIFS filesystem (inodes, directories, files)
- Detect PEB size automatically
- Volume table parsing
- Wear leveling analysis
- CRC32 validation
- Zero external dependencies
- Cross-platform (Linux/macOS/Windows)

## Install

```bash
go install github.com/Nyarime/go-ubi/cmd/ubi-extract@latest
```

## Usage

### As a CLI tool

```bash
# Extract UBI image
ubi-extract firmware.ubi ./output

# Show UBI structure info
ubi-extract --info firmware.ubi

# Detect UBI in firmware binary
ubi-extract --detect firmware.bin
```

### As a library

```go
import (
    "github.com/Nyarime/go-ubi/ubi"
    "github.com/Nyarime/go-ubi/ubifs"
)

// Parse UBI image
reader, _ := ubi.NewReader("firmware.ubi")
defer reader.Close()

img, _ := reader.Parse()

// Get volume table
records, _ := reader.ParseVolumeTable()

// Analyze wear leveling
wear, _ := reader.AnalyzeWearLeveling()
fmt.Print(wear.String())

// Extract all volumes
reader.ExtractAll("./output")

// Parse UBIFS from extracted volume
ubifsReader, _ := ubifs.NewReaderFromFile("./output/rootfs.img")
ubifsReader.Parse()
ubifsReader.Extract("./rootfs")
```

### Detection helpers

```go
// Check if file is UBI
if ubi.IsUBIImage("firmware.bin") { ... }

// Find UBI offset in firmware
offset := ubi.FindUBIOffset(data)
```

## UBI Format

```
UBI Image
├── PEB 0: [EC Header (64B)] [VID Header (64B)] [Data (LEB)]
├── PEB 1: [EC Header (64B)] [VID Header (64B)] [Data (LEB)]
├── ...
└── PEB N: [EC Header (64B)] [VID Header (64B)] [Data (LEB)]

Volumes reconstructed by collecting LEBs with same Volume ID.
UBIFS filesystem lives inside a UBI volume.
```

## Status

- [x] EC Header parsing (magic, version, erase counter)
- [x] VID Header parsing (volume ID, LEB number, type)
- [x] Volume extraction (LEB reassembly)
- [x] PEB size auto-detection (64K/128K/256K/512K)
- [x] Volume table parsing (names, types, PEB counts)
- [x] UBIFS filesystem reading (superblock, inodes, dentries, data)
- [x] UBIFS file extraction (directory tree reconstruction)
- [x] Wear leveling analysis (EC statistics, health assessment)
- [x] CRC32 validation (EC header, VID header)
- [x] UBI/UBIFS detection in firmware binaries
- [x] LZO decompression (native Go)
- [x] ZSTD decompression (native Go, full implementation)
- [x] LZ4 decompression (native Go)
- [x] Extended attributes
- [x] Orphan handling

## License

MIT

## Credits

Built by [Naixi Networks](https://naixi.net) for the [Nyarc](https://nyarc.bbie.net) firmware security audit tool.

Inspired by [ubi_reader](https://github.com/onekey-sec/ubi_reader) (Python).

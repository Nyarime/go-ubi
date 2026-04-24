# go-ubi

Pure Go UBI (Unsorted Block Images) reader and extractor.

**The first and only Go implementation of UBI image parsing.**

## Features

- Parse UBI images from NAND flash firmware
- Extract individual volumes
- Detect PEB size automatically
- Zero external dependencies
- Cross-platform (Linux/macOS/Windows)

## Usage

### As a library

```go
import "github.com/Nyarime/go-ubi/ubi"

reader, _ := ubi.NewReader("firmware.ubi")
defer reader.Close()

img, _ := reader.Parse()
reader.ExtractAll("./output")
```

### As a CLI tool

```bash
go install github.com/Nyarime/go-ubi/cmd/ubi-extract@latest
ubi-extract firmware.ubi ./output
```

## UBI Format

UBI (Unsorted Block Images) is a flash translation layer for NAND flash, used extensively in embedded/IoT firmware (TP-Link, ASUS, D-Link, etc).

```
UBI Image
├── PEB 0: [EC Header] [VID Header] [Data (LEB)]
├── PEB 1: [EC Header] [VID Header] [Data (LEB)]
├── ...
└── PEB N: [EC Header] [VID Header] [Data (LEB)]

Volumes are reconstructed by collecting LEBs with the same Volume ID.
```

## Status

- [x] EC Header parsing
- [x] VID Header parsing  
- [x] Volume extraction
- [x] PEB size auto-detection
- [ ] UBIFS filesystem reading
- [ ] Volume table parsing
- [ ] Wear leveling info

## License

MIT

## Credits

Built by [Naixi Networks](https://naixi.net) for the [Nyarc](https://nyarc.bbie.net) firmware security audit tool.

Inspired by [ubi_reader](https://github.com/onekey-sec/ubi_reader) (Python).

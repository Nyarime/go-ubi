package main

import (
	"fmt"
	"os"
	"path/filepath"
	
	ubiPkg "github.com/Nyarime/go-ubi/ubi"
	"github.com/Nyarime/go-ubi/ubifs"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("go-ubi — Pure Go UBI/UBIFS reader")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  ubi-extract <ubi-image> [output-dir]")
		fmt.Println("  ubi-extract --info <ubi-image>")
		fmt.Println("  ubi-extract --detect <firmware>")
		os.Exit(1)
	}
	
	switch os.Args[1] {
	case "--info":
		if len(os.Args) < 3 { fmt.Println("Usage: ubi-extract --info <image>"); os.Exit(1) }
		showInfo(os.Args[2])
	case "--detect":
		if len(os.Args) < 3 { fmt.Println("Usage: ubi-extract --detect <firmware>"); os.Exit(1) }
		detectUBI(os.Args[2])
	case "--repack":
		if len(os.Args) < 4 { fmt.Println("Usage: ubi-extract --repack <dir> <output.ubi> [peb-size-kb]"); os.Exit(1) }
		pebSize := 128 * 1024
		if len(os.Args) > 4 { fmt.Sscanf(os.Args[4], "%d", &pebSize); pebSize *= 1024 }
		repack(os.Args[2], os.Args[3], pebSize)
	default:
		extract(os.Args[1], func() string {
			if len(os.Args) > 2 { return os.Args[2] }
			return "ubi-extracted"
		}())
	}
}

func showInfo(path string) {
	reader, err := ubiPkg.NewReader(path)
	if err != nil { fmt.Printf("❌ %v\n", err); os.Exit(1) }
	defer reader.Close()

	img, err := reader.Parse()
	if err != nil { fmt.Printf("❌ %v\n", err); os.Exit(1) }

	fmt.Printf("📦 UBI Image Info\n")
	fmt.Printf("  Version:  %d\n", img.Version)
	fmt.Printf("  PEB Size: %dKB\n", img.PEBSize/1024)
	fmt.Printf("  LEB Size: %dKB\n", img.LEBSize/1024)
	fmt.Printf("  Volumes:  %d\n\n", len(img.Volumes))

	records, err := reader.ParseVolumeTable()
	if err == nil && len(records) > 0 {
		fmt.Printf("Volume Table:\n")
		fmt.Print(ubiPkg.PrintVolumeTable(records))
	}
}

func detectUBI(path string) {
	data, err := os.ReadFile(path)
	if err != nil { fmt.Printf("❌ %v\n", err); os.Exit(1) }

	offset := ubiPkg.FindUBIOffset(data)
	if offset >= 0 {
		fmt.Printf("✅ UBI found at offset 0x%X (%d)\n", offset, offset)
	} else {
		fmt.Println("❌ No UBI image found")
	}

	ubifsOffset := ubiPkg.FindUBIFSOffset(data)
	if ubifsOffset >= 0 {
		fmt.Printf("✅ UBIFS found at offset 0x%X (%d)\n", ubifsOffset, ubifsOffset)
	}
}

func extract(input, outputDir string) {
	fmt.Printf("📦 Parsing UBI: %s\n", input)
	
	reader, err := ubiPkg.NewReader(input)
	if err != nil { fmt.Printf("❌ %v\n", err); os.Exit(1) }
	defer reader.Close()

	img, err := reader.Parse()
	if err != nil { fmt.Printf("❌ %v\n", err); os.Exit(1) }

	// Try to get volume names
	reader.ParseVolumeTable()

	os.MkdirAll(outputDir, 0755)

	// Extract all volumes
	for id, vol := range img.Volumes {
		volName := vol.Name
		if volName == "" { volName = fmt.Sprintf("volume_%d", id) }
		
		volPath := filepath.Join(outputDir, volName+".img")
		reader.ExtractVolume(id, volPath)

		// Try to parse as UBIFS
		volData, err := os.ReadFile(volPath)
		if err != nil { continue }

		if ubiPkg.FindUBIFSOffset(volData) == 0 {
			fmt.Printf("  📂 Volume %d (%s) contains UBIFS, extracting...\n", id, volName)
			ubifsReader, err := ubifs.NewReader(volData)
			if err != nil { continue }
			
			if err := ubifsReader.Parse(); err != nil {
				fmt.Printf("  ⚠️  UBIFS parse warning: %v\n", err)
			}
			
			ubifsDir := filepath.Join(outputDir, volName)
			if err := ubifsReader.Extract(ubifsDir); err != nil {
				fmt.Printf("  ⚠️  UBIFS extract warning: %v\n", err)
			} else {
				fmt.Printf("  ✅ UBIFS extracted to: %s\n", ubifsDir)
			}
		}
	}

	fmt.Printf("✅ Done: %s\n", outputDir)
}

func repack(inputDir, outputPath string, pebSize int) {
	fmt.Printf("📦 Creating UBI image from: %s\n", inputDir)
	
	writer := ubiPkg.NewWriter(pebSize, 2048)
	
	// Find volume images in directory
	entries, err := os.ReadDir(inputDir)
	if err != nil { fmt.Printf("❌ %v\n", err); os.Exit(1) }
	
	volID := 0
	for _, e := range entries {
		if e.IsDir() { continue }
		name := e.Name()
		path := filepath.Join(inputDir, name)
		
		// Remove .img extension for volume name
		volName := name
		if ext := filepath.Ext(name); ext == ".img" || ext == ".ubifs" {
			volName = name[:len(name)-len(ext)]
		}
		
		fmt.Printf("  + Volume %d: %s\n", volID, volName)
		if err := writer.AddVolumeFromFile(volID, volName, ubiPkg.UBI_VID_DYNAMIC, path); err != nil {
			fmt.Printf("  ⚠️  %v\n", err)
			continue
		}
		volID++
	}
	
	if err := writer.Write(outputPath); err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ UBI image created: %s\n", outputPath)
}

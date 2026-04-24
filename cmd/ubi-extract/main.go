package main

import (
	"fmt"
	"os"
	
	"github.com/Nyarime/go-ubi/ubi"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ubi-extract <ubi-image> [output-dir]")
		fmt.Println("  Extract UBI volumes from firmware image")
		os.Exit(1)
	}
	
	input := os.Args[1]
	outputDir := "ubi-extracted"
	if len(os.Args) > 2 {
		outputDir = os.Args[2]
	}
	
	fmt.Printf("📦 Parsing UBI image: %s\n", input)
	
	reader, err := ubi.NewReader(input)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()
	
	img, err := reader.Parse()
	if err != nil {
		fmt.Printf("❌ Parse error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("📊 UBI v%d, PEB %dKB, %d volumes\n", img.Version, img.PEBSize/1024, len(img.Volumes))
	
	if err := reader.ExtractAll(outputDir); err != nil {
		fmt.Printf("❌ Extract error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Extracted to: %s\n", outputDir)
}

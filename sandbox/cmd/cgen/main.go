package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <symbol-graph-json> [output-dir]", os.Args[0])
	}

	symbolGraphPath := os.Args[1]

	// Determine output directory
	outputDir := filepath.Dir(symbolGraphPath)
	if len(os.Args) > 2 {
		outputDir = os.Args[2]
	}

	data, err := os.ReadFile(symbolGraphPath)
	if err != nil {
		log.Fatal(err)
	}

	var root Root
	if err := json.Unmarshal(data, &root); err != nil {
		log.Fatal(err)
	}

	moduleName := root.Module.Name

	// Generate C shim implementation (.m file)
	implShim := generateMethodShim(root)
	implPath := filepath.Join(outputDir, fmt.Sprintf("%s.shim.m", moduleName))

	// If we're writing to stdout, print the implementation
	if len(os.Args) <= 2 {
		fmt.Println(implShim)
	} else {
		if err := os.WriteFile(implPath, []byte(implShim), 0644); err != nil {
			log.Fatalf("Failed to write implementation file: %v", err)
		}
		log.Printf("Generated implementation file: %s", implPath)
	}

	// Generate C header file (.h file) if we have an output directory
	if len(os.Args) > 2 {
		headerShim := generateHeaderFile(root)
		headerPath := filepath.Join(outputDir, fmt.Sprintf("%s.shim.h", moduleName))

		if err := os.WriteFile(headerPath, []byte(headerShim), 0644); err != nil {
			log.Fatalf("Failed to write header file: %v", err)
		}
		log.Printf("Generated header file: %s", headerPath)

		// Generate c-for-go configuration file using the enhanced version
		cfgPath := filepath.Join(outputDir, fmt.Sprintf("%s.yml", strings.ToLower(moduleName)))
		cfgContent := generateExtendedCForGoConfig(moduleName, headerPath)

		if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
			log.Fatalf("Failed to write c-for-go config: %v", err)
		}
		log.Printf("Generated c-for-go config: %s", cfgPath)
	}
}

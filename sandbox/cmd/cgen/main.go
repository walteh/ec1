package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <symbol-graph-json>", os.Args[0])
	}

	symbolGraphPath := os.Args[1]
	data, err := os.ReadFile(symbolGraphPath)
	if err != nil {
		log.Fatal(err)
	}

	var root Root
	if err := json.Unmarshal(data, &root); err != nil {
		log.Fatal(err)
	}

	// Generate C shim for methods
	methodShim := generateMethodShim(root)
	fmt.Println(methodShim)

	// TODO: Uncomment when ready to generate Go types
	/*
	outputDir := filepath.Dir(symbolGraphPath)
	if len(os.Args) > 2 {
		outputDir = os.Args[2]
	}

	// Generate Go type definitions
	goTypes := generateGoTypes(root)
	typesPath := filepath.Join(outputDir, fmt.Sprintf("%s_types.go", strings.ToLower(root.Module.Name)))
	if err := os.WriteFile(typesPath, []byte(goTypes), 0644); err != nil {
		log.Fatalf("Failed to write Go types: %v", err)
	}
	*/
}

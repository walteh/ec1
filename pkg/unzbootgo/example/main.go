package main

import (
	"fmt"
	"os"

	"github.com/walteh/ec1/pkg/unzbootgo"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input-efi-file> <output-kernel-file>\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	fmt.Printf("Extracting kernel from %s to %s\n", inputFile, outputFile)
	
	err := unzbootgo.ExtractKernel(inputFile, outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Kernel extraction successful")
} 
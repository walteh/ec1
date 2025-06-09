package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Foundation -framework AppKit
#include <Foundation/Foundation.h>
#include <AppKit/AppKit.h>

bool hasAppContext() {
    // Check if NSApp exists and is properly initialized
    NSApplication *app = [NSApplication sharedApplication];
    return app != nil && [app isRunning];
}

void printAppInfo() {
    NSApplication *app = [NSApplication sharedApplication];
    printf("NSApplication exists: %s\n", app ? "YES" : "NO");
    printf("NSApplication isRunning: %s\n", [app isRunning] ? "YES" : "NO");
    printf("NSApplication activationPolicy: %ld\n", (long)[app activationPolicy]);
}
*/
import "C"
import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("=== Current Process App Context ===")
	C.printAppInfo()

	fmt.Println("\n=== Testing VZ Context Inheritance ===")
	if C.hasAppContext() {
		fmt.Println("✓ This process has NSApplication context (inherited from terminal)")
		fmt.Println("✓ VZ framework should work here")
	} else {
		fmt.Println("✗ This process has NO NSApplication context")
		fmt.Println("✗ VZ framework will fail here")
	}

	// Test spawning a child process
	fmt.Println("\n=== Testing Child Process ===")
	if len(os.Args) > 1 && os.Args[1] == "child" {
		fmt.Println("Child process app context:")
		C.printAppInfo()
		return
	}

	// Spawn child to see if context is inherited
	cmd := exec.Command(os.Args[0], "child")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Child process error: %v\n", err)
		return
	}
	fmt.Print(string(output))
}

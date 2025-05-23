package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

var entitlements = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>com.apple.security.virtualization</key>
	<true/>
</dict>
</plist>`

var fileToSign = ""
var mode = ""

func main() {

	// flag.StringVar(&mode, "mode", "", "mode")
	// flag.Parse()

	mode = os.Args[1]
	fileToSign = os.Args[2]

	if mode == "run-after-signing" || mode == "just-sign" {

	} else {
		log.Fatal("invalid mode (use run-after-signing or just-sign)")
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	f, err := os.CreateTemp("", "*.entitlements")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()
	defer os.Remove(f.Name()) // clean up

	if _, err := f.WriteString(entitlements); err != nil {
		return fmt.Errorf("failed to write entitlements content: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if fileToSign == "" {
		return fmt.Errorf("fileToSign is required (use -file flag)")
	}

	cmd := exec.Command("codesign", "--entitlements", f.Name(), "-s", "-", fileToSign)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to do codesign(%q): %w",
			strings.Join(
				[]string{
					"codesign", "--entitlements", f.Name(), "-s", "-", fileToSign,
				},
				" ",
			),
			err,
		)
	}

	if mode == "run-after-signing" {
		// writeArgsToDownloads()

		testcmd := exec.Command(os.Args[2], os.Args[3:]...)
		testcmd.Stdout = os.Stdout
		testcmd.Stderr = os.Stderr
		testcmd.Stdin = os.Stdin

		return testcmd.Run()
	}

	return nil
}

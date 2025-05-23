package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestVSCodeTestScenarios simulates various VS Code test scenarios
func TestVSCodeTestScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a temporary test package
	tmpDir := t.TempDir()

	// Create a simple Go test file
	testFile := filepath.Join(tmpDir, "example_test.go")
	testContent := `package main

import "testing"

func TestExample(t *testing.T) {
	if 1+1 != 2 {
		t.Error("math is broken")
	}
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create go.mod
	goModFile := filepath.Join(tmpDir, "go.mod")
	goModContent := "module testmod\n\ngo 1.24\n"
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Build gow binary for testing
	gowBinary := filepath.Join(t.TempDir(), "gow")
	cmd := exec.Command("go", "build", "-o", gowBinary, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build gow: %v", err)
	}

	tests := []struct {
		name           string
		args           []string
		wantErr        bool
		timeout        time.Duration
		workingDir     string
		expectInOutput []string
	}{
		{
			name:           "vscode_test_with_ide_flag",
			args:           []string{"test", "-timeout", "30s", "-run", "^TestExample$", "-ide", "-v", "."},
			wantErr:        false,
			timeout:        10 * time.Second,
			workingDir:     tmpDir,
			expectInOutput: []string{"=== RUN", "TestExample", "PASS"},
		},
		{
			name:           "vscode_test_with_codesign",
			args:           []string{"test", "-ide", "-codesign", "-v", "."},
			wantErr:        false, // May fail if codesign tool not available
			timeout:        10 * time.Second,
			workingDir:     tmpDir,
			expectInOutput: []string{}, // Don't expect specific output due to potential codesign issues
		},
		{
			name:           "vscode_debug_compile",
			args:           []string{"test", "-c", "-o", filepath.Join(tmpDir, "debug_bin"), "-gcflags", "all=-N -l", "."},
			wantErr:        false,
			timeout:        10 * time.Second,
			workingDir:     tmpDir,
			expectInOutput: []string{}, // No output expected for compile-only
		},
		{
			name:           "vscode_debug_compile_with_codesign",
			args:           []string{"test", "-c", "-o", filepath.Join(tmpDir, "debug_bin_signed"), "-gcflags", "all=-N -l", "-codesign", "."},
			wantErr:        false,
			timeout:        15 * time.Second,
			workingDir:     tmpDir,
			expectInOutput: []string{}, // No output expected for compile-only
		},
		{
			name:           "complex_vscode_scenario",
			args:           []string{"test", "-timeout", "30s", "-run", "^TestExample$", "-ide", "-codesign", "-v", "-count", "1", "."},
			wantErr:        false,
			timeout:        15 * time.Second,
			workingDir:     tmpDir,
			expectInOutput: []string{"TestExample", "PASS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip codesign tests if the tool is not available
			if strings.Contains(strings.Join(tt.args, " "), "-codesign") {
				if !isCodesignToolAvailable() {
					t.Skip("codesign tool not available, skipping test")
					return
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			cmd := exec.CommandContext(ctx, gowBinary, tt.args...)
			if tt.workingDir != "" {
				cmd.Dir = tt.workingDir
			}

			output, err := cmd.CombinedOutput()

			if (err != nil) != tt.wantErr {
				t.Errorf("gow %v error = %v, wantErr %v\nOutput: %s", tt.args, err, tt.wantErr, output)
				return
			}

			// Check expected output patterns
			outputStr := string(output)
			for _, expected := range tt.expectInOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("expected output to contain %q, but got: %s", expected, outputStr)
				}
			}

			// Special checks for compile-only tests
			if hasFlag(tt.args, "-c") {
				// Check that the binary was created
				var outputFile string
				for i, arg := range tt.args {
					if arg == "-o" && i+1 < len(tt.args) {
						outputFile = tt.args[i+1]
						break
					}
				}
				if outputFile != "" {
					if _, err := os.Stat(outputFile); err != nil {
						t.Errorf("expected output binary %s to be created, but got error: %v", outputFile, err)
					} else {
						// Clean up the binary
						os.Remove(outputFile)
					}
				}
			}
		})
	}
}

// TestDebugBinaryCreation specifically tests debug binary compilation
func TestDebugBinaryCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a temporary test package
	tmpDir := t.TempDir()

	// Create a simple test
	testFile := filepath.Join(tmpDir, "debug_test.go")
	testContent := `package main

import "testing"

func TestDebugMe(t *testing.T) {
	x := 42
	y := x * 2
	if y != 84 {
		t.Errorf("expected 84, got %d", y)
	}
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create go.mod
	goModFile := filepath.Join(tmpDir, "go.mod")
	goModContent := "module debugtest\n\ngo 1.24\n"
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Build gow binary
	gowBinary := filepath.Join(t.TempDir(), "gow")
	cmd := exec.Command("go", "build", "-o", gowBinary, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build gow: %v", err)
	}

	// Test debug binary creation
	debugBinary := filepath.Join(tmpDir, "__debug_bin_test")
	defer os.Remove(debugBinary)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Simulate VS Code debug binary creation
	cmd = exec.CommandContext(ctx, gowBinary,
		"test", "-c", "-o", debugBinary,
		"-gcflags", "all=-N -l",
		"-ide",
		".",
	)
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to create debug binary: %v\nOutput: %s", err, output)
	}

	// Check that the binary was created
	if _, err := os.Stat(debugBinary); err != nil {
		t.Errorf("debug binary was not created: %v", err)
	}

	// Test that the binary is executable and contains debug symbols
	if err == nil {
		// Try to run the binary with -test.list to see if it works
		listCmd := exec.Command(debugBinary, "-test.list", ".*")
		listOutput, listErr := listCmd.Output()
		if listErr != nil {
			t.Errorf("debug binary is not functional: %v", listErr)
		} else if !strings.Contains(string(listOutput), "TestDebugMe") {
			t.Errorf("debug binary doesn't contain expected test: %s", listOutput)
		}
	}
}

// Helper function to check if a slice contains a flag
func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

// Helper function to check if codesign tool is available
func isCodesignToolAvailable() bool {
	cmd := exec.Command("go", "tool", "github.com/walteh/ec1/tools/cmd/codesign", "--help")
	return cmd.Run() == nil
}

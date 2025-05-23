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

func TestGowConfig_findSafeGo(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*GowConfig)
		wantErr bool
	}{
		{
			name: "finds go in PATH",
			setup: func(cfg *GowConfig) {
				// Use default config, should find go in PATH
			},
			wantErr: false,
		},
		{
			name: "uses cached executable",
			setup: func(cfg *GowConfig) {
				// Find the real go path first
				realGo, _ := cfg.findSafeGo()
				cfg.GoExecutable = realGo
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewGowConfig()
			tt.setup(cfg)

			goPath, err := cfg.findSafeGo()
			if (err != nil) != tt.wantErr {
				t.Errorf("findSafeGo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && goPath == "" {
				t.Error("findSafeGo() returned empty path")
			}

			// Verify it's actually go
			if !tt.wantErr {
				cmd := exec.Command(goPath, "version")
				output, err := cmd.Output()
				if err != nil {
					t.Errorf("failed to run go version: %v", err)
				}
				if !strings.Contains(string(output), "go version") {
					t.Errorf("unexpected go version output: %s", output)
				}
			}
		})
	}
}

func TestGowConfig_hasGotestsum(t *testing.T) {
	cfg := NewGowConfig()
	
	// This test depends on environment, but we can at least test the logic
	hasIt := cfg.hasGotestsum()
	t.Logf("hasGotestsum: %v", hasIt)
	
	// If we have gotestsum, verify it works
	if hasIt {
		cmd := exec.Command("gotestsum", "--version")
		if err := cmd.Run(); err != nil {
			t.Errorf("gotestsum available but doesn't work: %v", err)
		}
	}
}

func TestGowConfig_execSafeGo(t *testing.T) {
	cfg := NewGowConfig()
	ctx := context.Background()

	// Test basic go command
	err := cfg.execSafeGo(ctx, "version")
	if err != nil {
		t.Errorf("execSafeGo() failed: %v", err)
	}
}

func TestGowConfig_handleTest(t *testing.T) {
	// Create a temporary test package
	tmpDir := t.TempDir()
	
	// Create a simple Go file
	testFile := filepath.Join(tmpDir, "test_test.go")
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

	// Change to temp dir
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	cfg := NewGowConfig()
	cfg.WorkspaceRoot = tmpDir

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "basic test",
			args:    []string{"test", "."},
			wantErr: false,
		},
		{
			name:    "test with verbose",
			args:    []string{"test", "-v", "."},
			wantErr: false,
		},
		{
			name:    "test with force",
			args:    []string{"test", "-force", "."},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cfg.handleTest(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleTest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGowConfig_handleMod(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create go.mod
	goModFile := filepath.Join(tmpDir, "go.mod")
	goModContent := "module testmod\n\ngo 1.24\n"
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Change to temp dir
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	cfg := NewGowConfig()
	cfg.WorkspaceRoot = tmpDir

	tests := []struct {
		name      string
		args      []string
		wantErr   bool
		allowFail bool // Allow failure for tools that may not be available in test env
	}{
		{
			name:      "mod tidy",
			args:      []string{"mod", "tidy"},
			wantErr:   false,
			allowFail: true, // Task tool may not be available in temp test env
		},
		{
			name:    "unknown mod command",
			args:    []string{"mod", "unknown"},
			wantErr: true,
		},
		{
			name:    "missing subcommand",
			args:    []string{"mod"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cfg.handleMod(tt.args)
			if !tt.allowFail && (err != nil) != tt.wantErr {
				t.Errorf("handleMod() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.allowFail && err != nil {
				t.Logf("handleMod() failed as expected in test environment: %v", err)
			}
		})
	}
}

func TestParseWorkspaceModules(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "simple workspace",
			content: `go 1.24

use (
	.
	./tools
)
`,
			expected: []string{".", "./tools"},
		},
		{
			name: "workspace with comments",
			content: `go 1.24

use (
	. // main module
	./tools
	// ./disabled
)
`,
			expected: []string{".", "./tools"},
		},
		{
			name:     "empty workspace",
			content:  "go 1.24\n",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseWorkspaceModules(tt.content)
			
			if len(result) != len(tt.expected) {
				t.Errorf("parseWorkspaceModules() got %d modules, want %d", len(result), len(tt.expected))
				return
			}

			for i, module := range result {
				if module != tt.expected[i] {
					t.Errorf("parseWorkspaceModules() module[%d] = %q, want %q", i, module, tt.expected[i])
				}
			}
		})
	}
}

func TestNewGowConfig(t *testing.T) {
	cfg := NewGowConfig()

	if cfg.Verbose {
		t.Error("NewGowConfig() should not be verbose by default")
	}

	if cfg.MaxLines != 1000 {
		t.Errorf("NewGowConfig() MaxLines = %d, want 1000", cfg.MaxLines)
	}

	if len(cfg.ErrorsToSuppress) == 0 {
		t.Error("NewGowConfig() should have some errors to suppress")
	}

	if cfg.WorkspaceRoot == "" {
		t.Error("NewGowConfig() should set WorkspaceRoot")
	}
}

func TestFileExists(t *testing.T) {
	// Test with existing file
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if !fileExists(tmpFile) {
		t.Error("fileExists() should return true for existing file")
	}

	// Test with non-existing file
	if fileExists("/non/existing/file") {
		t.Error("fileExists() should return false for non-existing file")
	}
}

// Integration test to verify end-to-end behavior
func TestGowIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build gow binary
	gowBinary := filepath.Join(t.TempDir(), "gow")
	cmd := exec.Command("go", "build", "-o", gowBinary, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build gow: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		timeout time.Duration
	}{
		{
			name:    "version command",
			args:    []string{"version"},
			wantErr: false,
			timeout: 5 * time.Second,
		},
		{
			name:    "help command",
			args:    []string{"help"},
			wantErr: false,
			timeout: 5 * time.Second,
		},
		{
			name:    "gow help command",
			args:    []string{"gow-help"},
			wantErr: false,
			timeout: 5 * time.Second,
		},
		{
			name:    "env command",
			args:    []string{"env", "GOVERSION"},
			wantErr: false,
			timeout: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			cmd := exec.CommandContext(ctx, gowBinary, tt.args...)
			output, err := cmd.CombinedOutput()

			if (err != nil) != tt.wantErr {
				t.Errorf("gow %v error = %v, wantErr %v\nOutput: %s", tt.args, err, tt.wantErr, output)
			}

			// Basic sanity checks
			if !tt.wantErr {
				switch tt.args[0] {
				case "version":
					if !strings.Contains(string(output), "go version") {
						t.Errorf("version command should contain 'go version', got: %s", output)
					}
				case "gow-help":
					if !strings.Contains(string(output), "gow - High-performance") {
						t.Errorf("gow-help should contain gow description, got: %s", output)
					}
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkGowConfig_findSafeGo(b *testing.B) {
	cfg := NewGowConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cfg.findSafeGo()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseWorkspaceModules(b *testing.B) {
	content := `go 1.24

use (
	.
	./tools
	./pkg/module1
	./pkg/module2
	./pkg/module3
)
`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseWorkspaceModules(content)
	}
} 
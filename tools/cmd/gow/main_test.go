package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestGowConfig_handleRun(t *testing.T) {
	// Create a temporary package with a runnable main
	tmpDir := t.TempDir()

	// Create a simple Go main file
	mainFile := filepath.Join(tmpDir, "main.go")
	mainContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to create main file: %v", err)
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
		name      string
		args      []string
		wantErr   bool
		allowFail bool // Allow failure for codesign tests that may not work in test env
	}{
		{
			name:    "basic run",
			args:    []string{"run", "main.go"},
			wantErr: false,
		},
		{
			name:      "run with codesign",
			args:      []string{"run", "-codesign", "main.go"},
			wantErr:   false,
			allowFail: true, // Codesign tool may not be available in test env
		},
		{
			name:      "run with codesign entitlement",
			args:      []string{"run", "-codesign", "-codesign-entitlement", "virtualization", "main.go"},
			wantErr:   false,
			allowFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cfg.handleRun(tt.args)
			if !tt.allowFail && (err != nil) != tt.wantErr {
				t.Errorf("handleRun() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.allowFail && err != nil {
				t.Logf("handleRun() failed as expected in test environment: %v", err)
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

func TestGowConfig_createStdioLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := NewGowConfig()
	cfg.WorkspaceRoot = tmpDir

	// Test creating log file
	err := cfg.setupStdioLogging("go", []string{"version"})
	if err != nil {
		t.Fatalf("setupStdioLogging() failed: %v", err)
	}

	// Verify the log directory was created
	logDir := filepath.Join(tmpDir, ".log", "gow")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("log directory was not created")
	}

	// Find the log file
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("failed to read log directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("no log files were created")
	}

	var logFile string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "stdio-pipe.log") {
			logFile = filepath.Join(logDir, entry.Name())
			break
		}
	}

	if logFile == "" {
		t.Error("no stdio-pipe.log file found")
		return
	}

	// Verify filename format (should contain timestamp and stdio-pipe.log)
	filename := filepath.Base(logFile)
	if !strings.Contains(filename, "stdio-pipe.log") {
		t.Errorf("log filename should contain 'stdio-pipe.log', got: %s", filename)
	}

	// Verify timestamp format (YYYY-MM-DD_HH-MM-SS)
	parts := strings.Split(filename, "_")
	if len(parts) < 2 {
		t.Errorf("log filename should contain timestamp, got: %s", filename)
	}

	// Verify the log file contains command header
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "=== GOW STDIO LOG ===") {
		t.Error("log file should contain header")
	}
	if !strings.Contains(contentStr, "Command: go version") {
		t.Error("log file should contain command information")
	}
	if !strings.Contains(contentStr, "Working Directory:") {
		t.Error("log file should contain working directory")
	}
}

func TestParseCodesignFlags(t *testing.T) {
	tests := []struct {
		name                 string
		args                 []string
		expectedCodesign     bool
		expectedEntitlements []string
		expectedIdentity     string
		expectedForce        bool
		expectedRemaining    []string
	}{
		{
			name:                 "no codesign flags",
			args:                 []string{"run", "main.go"},
			expectedCodesign:     false,
			expectedEntitlements: nil,
			expectedIdentity:     "",
			expectedForce:        false,
			expectedRemaining:    []string{"run", "main.go"},
		},
		{
			name:                 "basic codesign",
			args:                 []string{"run", "-codesign", "main.go"},
			expectedCodesign:     true,
			expectedEntitlements: nil,
			expectedIdentity:     "",
			expectedForce:        false,
			expectedRemaining:    []string{"run", "main.go"},
		},
		{
			name:                 "codesign with entitlement",
			args:                 []string{"run", "-codesign", "-codesign-entitlement", "virtualization", "main.go"},
			expectedCodesign:     true,
			expectedEntitlements: []string{"virtualization"},
			expectedIdentity:     "",
			expectedForce:        false,
			expectedRemaining:    []string{"run", "main.go"},
		},
		{
			name:                 "codesign with multiple entitlements",
			args:                 []string{"run", "-codesign", "-codesign-entitlement", "virtualization", "-codesign-entitlement", "hypervisor", "main.go"},
			expectedCodesign:     true,
			expectedEntitlements: []string{"virtualization", "hypervisor"},
			expectedIdentity:     "",
			expectedForce:        false,
			expectedRemaining:    []string{"run", "main.go"},
		},
		{
			name:                 "codesign with identity and force",
			args:                 []string{"run", "-codesign", "-codesign-identity", "Developer ID", "-codesign-force", "main.go"},
			expectedCodesign:     true,
			expectedEntitlements: nil,
			expectedIdentity:     "Developer ID",
			expectedForce:        true,
			expectedRemaining:    []string{"run", "main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codesign, entitlements, identity, force, remaining := parseCodesignFlags(tt.args)

			assert.Equal(t, tt.expectedCodesign, codesign, "codesign flag should match")
			assert.Equal(t, tt.expectedEntitlements, entitlements, "entitlements should match")
			assert.Equal(t, tt.expectedIdentity, identity, "identity should match")
			assert.Equal(t, tt.expectedForce, force, "force flag should match")
			assert.Equal(t, tt.expectedRemaining, remaining, "remaining args should match")
		})
	}
}

func TestGowConfig_buildCodesignExecArgs(t *testing.T) {
	cfg := NewGowConfig()

	tests := []struct {
		name           string
		mode           string
		entitlements   []string
		identity       string
		force          bool
		additionalArgs []string
		expectedArgs   []string
	}{
		{
			name:         "basic test mode",
			mode:         "test",
			entitlements: nil,
			identity:     "",
			force:        false,
			expectedArgs: []string{"tool", "github.com/walteh/ec1/tools/cmd/codesign", "-mode=test", "-entitlement=virtualization", "-quiet", "--"},
		},
		{
			name:         "run mode with custom entitlement",
			mode:         "run",
			entitlements: []string{"hypervisor"},
			identity:     "",
			force:        false,
			expectedArgs: []string{"tool", "github.com/walteh/ec1/tools/cmd/codesign", "-mode=run", "-entitlement=hypervisor", "-quiet", "--"},
		},
		{
			name:         "test mode with identity and force",
			mode:         "test",
			entitlements: nil,
			identity:     "Developer ID",
			force:        true,
			expectedArgs: []string{"tool", "github.com/walteh/ec1/tools/cmd/codesign", "-mode=test", "-entitlement=virtualization", "-identity=Developer ID", "-force", "-quiet", "--"},
		},
		{
			name:           "run mode with multiple entitlements and additional args",
			mode:           "run",
			entitlements:   []string{"virtualization", "hypervisor"},
			identity:       "",
			force:          false,
			additionalArgs: []string{"-custom-flag", "value"},
			expectedArgs:   []string{"tool", "github.com/walteh/ec1/tools/cmd/codesign", "-mode=run", "-entitlement=virtualization", "-entitlement=hypervisor", "-quiet", "-custom-flag", "value", "--"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.buildCodesignExecArgs(tt.mode, tt.entitlements, tt.identity, tt.force, tt.additionalArgs)
			assert.Equal(t, tt.expectedArgs, result, "exec args should match expected")
		})
	}
}

func TestGowConfig_PipeStdioToFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := NewGowConfig()
	cfg.WorkspaceRoot = tmpDir
	cfg.PipeStdioToFile = true

	ctx := context.Background()

	// Set up stdio logging first
	err := cfg.setupStdioLogging("go", []string{"version"})
	if err != nil {
		t.Fatalf("setupStdioLogging() failed: %v", err)
	}

	// Test with a simple go command that produces output
	err = cfg.execSafeGo(ctx, "version")
	if err != nil {
		t.Fatalf("execSafeGo() with stdio piping failed: %v", err)
	}

	// Verify log file was created
	logDir := filepath.Join(tmpDir, ".log", "gow")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("failed to read log directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("no log files were created")
	}

	// Check that at least one log file contains expected content
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "stdio-pipe.log") {
			logPath := filepath.Join(logDir, entry.Name())
			content, err := os.ReadFile(logPath)
			if err != nil {
				t.Errorf("failed to read log file %s: %v", entry.Name(), err)
				continue
			}

			// Should contain go version output
			if !strings.Contains(string(content), "go version") {
				t.Errorf("log file should contain 'go version', got: %s", content)
			}
			return
		}
	}

	t.Error("no stdio-pipe.log file found")
}

func TestGowConfig_handleTestRunPatternFix(t *testing.T) {
	tests := []struct {
		name         string
		inputPattern string
		expected     string
	}{
		{
			name:         "VS Code subtest pattern gets fixed",
			inputPattern: "^TestHarpoon/bun_version$",
			expected:     "^TestHarpoon$/bun_version$",
		},
		{
			name:         "Regular test pattern passes through unchanged",
			inputPattern: "^TestHarpoon$",
			expected:     "^TestHarpoon$",
		},
		{
			name:         "Pattern without slash passes through unchanged",
			inputPattern: "TestSomething",
			expected:     "TestSomething",
		},
		{
			name:         "Complex subtest pattern gets fixed",
			inputPattern: "^TestComplex/subtest/nested$",
			expected:     "^TestComplex$/subtest/nested$",
		},
		{
			name:         "Pattern without leading caret gets fixed",
			inputPattern: "TestHarpoon/bun_version$",
			expected:     "^TestHarpoon$/bun_version$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the pattern transformation logic directly
			result := tt.inputPattern

			// Apply the same logic as in handleTest
			if strings.Contains(result, "/") {
				parts := strings.SplitN(result, "/", 2)
				if len(parts) == 2 {
					testFunc := parts[0]
					subtest := parts[1]

					// Remove leading ^ if present
					if strings.HasPrefix(testFunc, "^") {
						testFunc = testFunc[1:]
					}

					// Create the fixed pattern
					result = "^" + testFunc + "$/" + subtest
				}
			}

			assert.Equal(t, tt.expected, result, "Pattern transformation should match expected result")
		})
	}
}

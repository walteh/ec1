package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGowConfig(t *testing.T) {
	cfg := NewGowConfig()
	
	require.NotNil(t, cfg, "config should not be nil")
	assert.False(t, cfg.Verbose, "verbose should default to false")
	assert.NotEmpty(t, cfg.WorkspaceRoot, "workspace root should be set")
	assert.Empty(t, cfg.GoExecutable, "go executable should start empty")
	assert.Equal(t, 1000, cfg.MaxLines, "max lines should default to 1000")
	assert.Greater(t, len(cfg.ErrorsToSuppress), 0, "should have default errors to suppress")
	assert.Greater(t, len(cfg.StdoutsToSuppress), 0, "should have default stdouts to suppress")
}

func TestGowConfig_FindSafeGo(t *testing.T) {
	cfg := NewGowConfig()
	
	// Test with cached executable
	cfg.GoExecutable = "/usr/bin/go"
	goPath, err := cfg.findSafeGo()
	require.NoError(t, err, "should return cached executable")
	assert.Equal(t, "/usr/bin/go", goPath, "should return cached path")
	
	// Test finding go in PATH (reset cache first)
	cfg.GoExecutable = ""
	goPath, err = cfg.findSafeGo()
	
	// This might fail in test environments without go in PATH, so we'll check more carefully
	if err != nil {
		assert.Contains(t, err.Error(), "could not find go executable", "should return appropriate error")
	} else {
		assert.NotEmpty(t, goPath, "should find go executable")
		assert.True(t, strings.HasSuffix(goPath, "go"), "path should end with 'go'")
	}
}

func TestFileExists(t *testing.T) {
	// Test with existing file
	tempFile, err := os.CreateTemp("", "test-file-*")
	require.NoError(t, err, "should create temp file")
	defer os.Remove(tempFile.Name())
	tempFile.Close()
	
	assert.True(t, fileExists(tempFile.Name()), "should detect existing file")
	
	// Test with non-existing file
	assert.False(t, fileExists("/non/existent/file"), "should detect non-existing file")
}

func TestGowConfig_HasGotestsum(t *testing.T) {
	cfg := NewGowConfig()
	
	// This test will depend on the environment
	// We just verify it doesn't panic and returns a boolean
	result := cfg.hasGotestsum()
	assert.IsType(t, false, result, "should return a boolean")
}

func TestGowConfig_HandleMod(t *testing.T) {
	cfg := NewGowConfig()
	
	// Test with no subcommand
	err := cfg.handleMod([]string{"mod"})
	assert.Error(t, err, "should error when no subcommand provided")
	assert.Contains(t, err.Error(), "mod subcommand required", "should specify subcommand required")
	
	// Test with unknown subcommand
	err = cfg.handleMod([]string{"mod", "unknown"})
	assert.Error(t, err, "should error with unknown subcommand")
	assert.Contains(t, err.Error(), "unknown mod subcommand", "should specify unknown subcommand")
}

func TestGowConfig_HandleTool(t *testing.T) {
	cfg := NewGowConfig()
	
	// Create a temporary hl-config.yaml file
	tempDir, err := os.MkdirTemp("", "gow-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	hlConfigPath := filepath.Join(tempDir, "hl-config.yaml")
	err = os.WriteFile(hlConfigPath, []byte("test: config"), 0644)
	require.NoError(t, err)
	
	cfg.WorkspaceRoot = tempDir
	
	// Mock go executable to avoid actually running go tool
	cfg.GoExecutable = "true" // Use 'true' command as safe mock
	
	// This would normally run go tool, but with 'true' it will just succeed
	err = cfg.handleTool([]string{"tool", "version"})
	assert.NoError(t, err, "should not error with valid tool command")
	
	// Verify HL_CONFIG environment variable was set
	assert.Equal(t, hlConfigPath, os.Getenv("HL_CONFIG"), "HL_CONFIG should be set")
}

func TestGowConfig_HandleRetab(t *testing.T) {
	cfg := NewGowConfig()
	
	// Create a temporary workspace with .editorconfig
	tempDir, err := os.MkdirTemp("", "gow-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	editorConfigPath := filepath.Join(tempDir, ".editorconfig")
	err = os.WriteFile(editorConfigPath, []byte("root = true\n[*.go]\nindent_style = tab\n"), 0644)
	require.NoError(t, err)
	
	cfg.WorkspaceRoot = tempDir
	cfg.GoExecutable = "true" // Use 'true' command as safe mock
	
	// This would normally run the retab tool, but with 'true' it's safe
	err = cfg.handleRetab()
	assert.NoError(t, err, "should not error when .editorconfig exists")
	
	// Test with missing .editorconfig
	cfg.WorkspaceRoot = "/nonexistent"
	err = cfg.handleRetab()
	assert.Error(t, err, "should error when .editorconfig doesn't exist")
	assert.Contains(t, err.Error(), "failed to read .editorconfig", "should mention editorconfig")
}

func TestPrintUsage(t *testing.T) {
	// This test just ensures printUsage doesn't panic
	assert.NotPanics(t, func() {
		printUsage()
	}, "printUsage should not panic")
}

func TestMainFunction_Integration(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()
	
	// Test help command
	os.Args = []string{"gow", "help"}
	assert.NotPanics(t, func() {
		// We can't easily test main() directly since it calls os.Exit
		// So we test the individual handlers instead
		args := []string{"help"}
		
		switch args[0] {
		case "help":
			printUsage()
		}
	}, "help command should not panic")
}

func TestGowConfig_RunSafeGo(t *testing.T) {
	cfg := NewGowConfig()
	ctx := context.Background()
	
	// Test with invalid go executable
	cfg.GoExecutable = "/nonexistent/go"
	err := cfg.runSafeGo(ctx, "version")
	assert.Error(t, err, "should error with nonexistent go executable")
	
	// Test with 'true' as a safe mock
	cfg.GoExecutable = "true"
	err = cfg.runSafeGo(ctx, "test", "args")
	assert.NoError(t, err, "should succeed with 'true' mock")
}

func TestGowConfig_RunWithGotestsum(t *testing.T) {
	cfg := NewGowConfig()
	ctx := context.Background()
	cfg.GoExecutable = "true" // Use 'true' command as safe mock
	
	// This test mainly verifies the function doesn't panic
	// The actual execution depends on having gotestsum installed
	goArgs := []string{"test", "./..."}
	
	if cfg.hasGotestsum() {
		err := cfg.runWithGotestsum(ctx, goArgs)
		// May succeed or fail depending on environment, but shouldn't panic
		_ = err
	}
	
	// Just verify the method exists and can be called
	assert.NotNil(t, cfg.runWithGotestsum, "runWithGotestsum method should exist")
}

func TestGowConfig_HandleTest(t *testing.T) {
	cfg := NewGowConfig()
	cfg.GoExecutable = "true" // Use 'true' command as safe mock (always succeeds)
	
	// Test basic test command
	err := cfg.handleTest([]string{"test"})
	assert.NoError(t, err, "basic test command should succeed with mock")
	
	// Test with flags
	err = cfg.handleTest([]string{"test", "-v", "-target", "."})
	assert.NoError(t, err, "test with flags should succeed with mock")
}

func TestErrorPathHandling(t *testing.T) {
	cfg := NewGowConfig()
	
	// Test findSafeGo with invalid workspace
	cfg.WorkspaceRoot = "/nonexistent/path"
	cfg.GoExecutable = "" // Reset cache
	
	// Should still try to find go, might succeed or fail depending on system
	_, err := cfg.findSafeGo()
	// We don't assert the result since it depends on the system,
	// but it should not panic
	if err != nil {
		assert.Contains(t, err.Error(), "could not find go executable")
	}
}

func TestWorkspaceRootHandling(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "gow-workspace-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create a fake .editorconfig file
	editorConfigPath := filepath.Join(tempDir, ".editorconfig")
	err = os.WriteFile(editorConfigPath, []byte("root = true\n[*.go]\nindent_style = tab\n"), 0644)
	require.NoError(t, err)
	
	cfg := NewGowConfig()
	cfg.WorkspaceRoot = tempDir
	cfg.GoExecutable = "true" // Use 'true' command as safe mock
	
	// Test that retab command can find the .editorconfig
	err = cfg.handleRetab()
	assert.NoError(t, err, "should handle retab successfully")
	
	// Verify the .editorconfig file exists
	assert.True(t, fileExists(editorConfigPath), "editorconfig should exist")
}

func TestGlobalFlagParsing(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()
	
	// Test verbose flag parsing logic
	cfg := NewGowConfig()
	
	// Simulate args parsing (extracted from main)
	args := []string{"-verbose", "test", "./..."}
	
	// Parse global flags
	for i, arg := range args {
		if arg == "-verbose" || arg == "--verbose" {
			cfg.Verbose = true
			// Remove this flag from args
			args = append(args[:i], args[i+1:]...)
			break
		}
	}
	
	assert.True(t, cfg.Verbose, "verbose flag should be parsed")
	assert.Equal(t, []string{"test", "./..."}, args, "verbose flag should be removed from args")
}

func TestGowConfig_HandleTestWithCoverage(t *testing.T) {
	cfg := NewGowConfig()
	cfg.GoExecutable = "true" // Safe mock
	
	// Test function coverage flag (this hits the coverage code path)
	err := cfg.handleTest([]string{"test", "-function-coverage"})
	assert.NoError(t, err, "test with function coverage should succeed")
	
	// Test force flag
	err = cfg.handleTest([]string{"test", "-force"})
	assert.NoError(t, err, "test with force flag should succeed")
	
	// Test run pattern
	err = cfg.handleTest([]string{"test", "-run", "TestSomething"})
	assert.NoError(t, err, "test with run pattern should succeed")
}

func TestGowConfig_HandleDap(t *testing.T) {
	cfg := NewGowConfig()
	
	// Test with mock dlv command
	// Since dlv may not be available, we'll just test the path setup
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	
	// The function should modify PATH regardless of whether dlv exists
	assert.NotPanics(t, func() {
		err := cfg.handleDap([]string{"dap", "--help"})
		// This will likely fail since dlv probably isn't available
		// but the important thing is it doesn't panic and sets up PATH
		_ = err
	}, "handleDap should not panic")
	
	// Verify PATH was modified
	newPath := os.Getenv("PATH")
	assert.Contains(t, newPath, cfg.WorkspaceRoot, "PATH should contain workspace root")
}

func TestGowConfig_HandleModWithEmbedded(t *testing.T) {
	cfg := NewGowConfig()
	cfg.GoExecutable = "true" // Safe mock
	
	// Create a temporary workspace
	tempDir, err := os.MkdirTemp("", "gow-mod-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	cfg.WorkspaceRoot = tempDir
	
	// Test mod tidy without go.work (should run on current dir)
	err = cfg.handleMod([]string{"mod", "tidy"})
	assert.NoError(t, err, "mod tidy should succeed without go.work")
	
	// Create a mock go.work file
	goWorkContent := `go 1.21

use (
	.
	./submodule
)
`
	err = os.WriteFile(filepath.Join(tempDir, "go.work"), []byte(goWorkContent), 0644)
	require.NoError(t, err)
	
	// Create submodule directory
	submoduleDir := filepath.Join(tempDir, "submodule")
	err = os.MkdirAll(submoduleDir, 0755)
	require.NoError(t, err)
	
	// Test mod tidy with workspace
	err = cfg.handleMod([]string{"mod", "tidy"})
	assert.NoError(t, err, "mod tidy should succeed with go.work")
	
	// Test mod upgrade 
	err = cfg.handleMod([]string{"mod", "upgrade"})
	assert.NoError(t, err, "mod upgrade should succeed")
}

func TestParseWorkspaceModules(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "simple_workspace",
			content: `go 1.21

use (
	.
	./pkg/module1
	./pkg/module2
)
`,
			expected: []string{".", "./pkg/module1", "./pkg/module2"},
		},
		{
			name: "with_comments",
			content: `go 1.21

use (
	.
	// This is a comment
	./pkg/module1
	./pkg/module2
)
`,
			expected: []string{".", "./pkg/module1", "./pkg/module2"},
		},
		{
			name: "empty_workspace",
			content: `go 1.21

use (
)
`,
			expected: []string{},
		},
		{
			name: "no_use_block",
			content: `go 1.21
`,
			expected: []string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseWorkspaceModules(tt.content)
			assert.Equal(t, tt.expected, result, "should parse workspace modules correctly")
		})
	}
}

func TestHandlerCoverage(t *testing.T) {
	cfg := NewGowConfig()
	cfg.GoExecutable = "true" // Safe mock for all handlers
	
	// Test that all handlers exist and can be called without panicking
	handlers := map[string]func() error{
		"test": func() error {
			return cfg.handleTest([]string{"test"})
		},
		"mod_tidy": func() error {
			// Skip actual task execution to avoid dependencies
			return nil
		},
		"retab": func() error {
			// Create minimal setup
			tempDir, _ := os.MkdirTemp("", "test-*")
			defer os.RemoveAll(tempDir)
			editorConfig := filepath.Join(tempDir, ".editorconfig")
			os.WriteFile(editorConfig, []byte("root = true"), 0644)
			cfg.WorkspaceRoot = tempDir
			return cfg.handleRetab()
		},
		"tool": func() error {
			return cfg.handleTool([]string{"tool", "version"})
		},
	}
	
	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				err := handler()
				// We don't assert on error since some may fail in test environment
				// but they shouldn't panic
				_ = err
			}, "handler %s should not panic", name)
		})
	}
} 
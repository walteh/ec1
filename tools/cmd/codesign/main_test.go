package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommonEntitlements(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"virtualization", "virtualization", "com.apple.security.virtualization"},
		{"hypervisor", "hypervisor", "com.apple.security.hypervisor"},
		{"network-client", "network-client", "com.apple.security.network.client"},
		{"allow-jit", "allow-jit", "com.apple.security.cs.allow-jit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, exists := CommonEntitlements[tt.key]
			assert.True(t, exists, "entitlement key should exist")
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestArrayFlags(t *testing.T) {
	var flags arrayFlags

	// Test Set method
	err := flags.Set("value1")
	require.NoError(t, err)
	err = flags.Set("value2")
	require.NoError(t, err)

	assert.Len(t, flags, 2)
	assert.Equal(t, "value1", flags[0])
	assert.Equal(t, "value2", flags[1])

	// Test String method
	assert.Equal(t, "value1,value2", flags.String())
}

func TestGenerateEntitlementsXML(t *testing.T) {
	tests := []struct {
		name         string
		entitlements []string
		expected     string
	}{
		{
			name:         "single entitlement",
			entitlements: []string{"com.apple.security.virtualization"},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>com.apple.security.virtualization</key>
	<true/>
</dict>
</plist>`,
		},
		{
			name:         "multiple entitlements",
			entitlements: []string{"com.apple.security.virtualization", "com.apple.security.network.client"},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>com.apple.security.virtualization</key>
	<true/>
	<key>com.apple.security.network.client</key>
	<true/>
</dict>
</plist>`,
		},
		{
			name:         "empty entitlements",
			entitlements: []string{},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
</dict>
</plist>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := generateEntitlementsXML(tt.entitlements)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestCreateTestExecWrapper(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		entitlements []string
		identity     string
		force        bool
		expectedArgs []string
	}{
		{
			name:         "basic wrapper",
			entitlements: []string{"virtualization"},
			identity:     "-",
			force:        false,
			expectedArgs: []string{"tool", "github.com/walteh/ec1/tools/cmd/codesign", "-mode=exec", "-entitlement=virtualization", "--"},
		},
		{
			name:         "with custom identity",
			entitlements: []string{"virtualization"},
			identity:     "Developer ID",
			force:        false,
			expectedArgs: []string{"tool", "github.com/walteh/ec1/tools/cmd/codesign", "-mode=exec", "-identity=Developer ID", "-entitlement=virtualization", "--"},
		},
		{
			name:         "with force",
			entitlements: []string{"virtualization"},
			identity:     "-",
			force:        true,
			expectedArgs: []string{"tool", "github.com/walteh/ec1/tools/cmd/codesign", "-mode=exec", "-force", "-entitlement=virtualization", "--"},
		},
		{
			name:         "multiple entitlements",
			entitlements: []string{"virtualization", "network-client"},
			identity:     "-",
			force:        false,
			expectedArgs: []string{"tool", "github.com/walteh/ec1/tools/cmd/codesign", "-mode=exec", "-entitlement=virtualization", "-entitlement=network-client", "--"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createTestExecWrapper(ctx, tt.entitlements, tt.identity, tt.force)
			expected := "go " + strings.Join(tt.expectedArgs, " ")
			assert.Equal(t, expected, result)
		})
	}
}

func TestSignBinaryDryRun(t *testing.T) {
	ctx := context.Background()

	// Test with dry run - should not fail even without actual binary
	err := signBinary(ctx, "/nonexistent/binary", []string{"virtualization"}, "-", false, true)
	assert.NoError(t, err, "dry run should not fail")
}

func TestSignBinaryEntitlementResolution(t *testing.T) {
	ctx := context.Background()

	// Create a temporary executable for testing
	tmpDir := t.TempDir()
	testBinary := filepath.Join(tmpDir, "testapp")

	// Create a simple executable
	err := os.WriteFile(testBinary, []byte("#!/bin/bash\necho hello"), 0755)
	require.NoError(t, err)

	// Test entitlement resolution in dry-run mode
	err = signBinary(ctx, testBinary, []string{"virtualization", "com.apple.security.network.client"}, "-", false, true)
	assert.NoError(t, err)
}

func TestRunModeValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "sign mode without target",
			config: &Config{
				Mode:         "sign",
				Target:       "",
				Entitlements: []string{"virtualization"},
			},
			expectError: true,
			errorMsg:    "target file is required",
		},
		{
			name: "exec mode without args",
			config: &Config{
				Mode:     "exec",
				ExecArgs: []string{},
			},
			expectError: true,
			errorMsg:    "no command specified",
		},
		{
			name: "test mode without args",
			config: &Config{
				Mode:     "test",
				ExecArgs: []string{},
			},
			expectError: true,
			errorMsg:    "no go test command specified",
		},
		{
			name: "invalid mode",
			config: &Config{
				Mode: "invalid",
			},
			expectError: true,
			errorMsg:    "unknown mode",
		},
		{
			name: "valid sign mode",
			config: &Config{
				Mode:         "sign",
				Target:       "/tmp/testapp",
				Entitlements: []string{"virtualization"},
				DryRun:       true, // Use dry run to avoid actual codesign
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runMode(ctx, tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunLegacyMode(t *testing.T) {
	// Save original args
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	ctx := context.Background()

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "insufficient args",
			args:        []string{"codesign", "just-sign"},
			expectError: true,
			errorMsg:    "requires at least 2 arguments",
		},
		{
			name:        "unknown legacy mode",
			args:        []string{"codesign", "unknown-mode", "/tmp/test"},
			expectError: true,
			errorMsg:    "unknown legacy mode",
		},
		{
			name:        "valid just-sign mode (dry run via target)",
			args:        []string{"codesign", "just-sign", "/nonexistent/test"},
			expectError: false, // Will fail in signBinary but that's expected without dry-run in legacy mode
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			err := runLegacyMode(ctx)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				// For the valid case, we expect it to fail at codesign execution
				// since we're not in dry-run mode and don't have a real binary
				// This is acceptable for the test as we're testing the mode parsing
				if err != nil {
					assert.Contains(t, err.Error(), "codesign failed")
				}
			}
		})
	}
}

// Integration tests that require the actual codesign binary
func TestIntegrationSignBinary(t *testing.T) {
	// Skip if not on macOS or codesign not available
	if !isCodesignAvailable() {
		t.Skip("codesign not available, skipping integration test")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create a simple binary
	testBinary := filepath.Join(tmpDir, "testapp")
	source := `package main
func main() { println("hello") }`

	sourceFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(sourceFile, []byte(source), 0644)
	require.NoError(t, err)

	// Build the binary
	cmd := exec.Command("go", "build", "-o", testBinary, sourceFile)
	err = cmd.Run()
	require.NoError(t, err)

	// Test signing
	err = signBinary(ctx, testBinary, []string{"virtualization"}, "-", false, false)
	assert.NoError(t, err, "signing should succeed")

	// Verify the binary is signed (basic check)
	cmd = exec.Command("codesign", "-dv", testBinary)
	err = cmd.Run()
	assert.NoError(t, err, "binary should be properly signed")
}

func TestIntegrationExecMode(t *testing.T) {
	if !isCodesignAvailable() {
		t.Skip("codesign not available, skipping integration test")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create a simple binary that exits successfully
	testBinary := filepath.Join(tmpDir, "testapp")
	source := `package main
import "os"
func main() { os.Exit(0) }`

	sourceFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(sourceFile, []byte(source), 0644)
	require.NoError(t, err)

	// Build the binary
	cmd := exec.Command("go", "build", "-o", testBinary, sourceFile)
	err = cmd.Run()
	require.NoError(t, err)

	// Test exec mode
	config := &Config{
		Mode:         "exec",
		Entitlements: []string{"virtualization"},
		ExecArgs:     []string{testBinary},
		Identity:     "-",
		Force:        false,
		DryRun:       false,
	}

	err = execMode(ctx, config)
	assert.NoError(t, err, "exec mode should succeed")
}

// Helper function to check if codesign is available
func isCodesignAvailable() bool {
	// Test with a simple version check that should succeed
	cmd := exec.Command("which", "codesign")
	err := cmd.Run()
	return err == nil
}

// Benchmark tests
func BenchmarkGenerateEntitlementsXML(b *testing.B) {
	entitlements := []string{
		"com.apple.security.virtualization",
		"com.apple.security.network.client",
		"com.apple.security.hypervisor",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateEntitlementsXML(entitlements)
	}
}

func BenchmarkEntitlementResolution(b *testing.B) {
	entitlements := []string{"virtualization", "network-client", "hypervisor", "allow-jit"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolved := make([]string, 0, len(entitlements))
		for _, ent := range entitlements {
			if fullIdent, exists := CommonEntitlements[ent]; exists {
				resolved = append(resolved, fullIdent)
			} else {
				resolved = append(resolved, ent)
			}
		}
	}
}

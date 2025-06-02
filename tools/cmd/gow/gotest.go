package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// hasGotestsum checks if gotestsum is available
func (cfg *GowConfig) hasGotestsum() bool {
	// Check if we can run gotestsum via go tool
	goPath, err := cfg.findSafeGo()
	if err != nil {
		return false
	}

	cmd := exec.Command(goPath, "tool", "gotest.tools/gotestsum", "--version")
	cmd.Dir = cfg.WorkspaceRoot
	return cmd.Run() == nil
}

// runWithGotestsum runs tests using gotestsum from project tools
func (cfg *GowConfig) runWithGotestsum(ctx context.Context, goArgs []string) error {
	goPath, err := cfg.findSafeGo()
	if err != nil {
		return err
	}

	// Build gotestsum command - remove "test" from goArgs since gotestsum adds it
	testArgs := goArgs[1:] // Skip the "test" command

	args := []string{
		"tool", "gotest.tools/gotestsum",
		"--format", "pkgname",
		"--format-icons", "hivis",
		"--", // Separator for go test flags
	}
	args = append(args, testArgs...)

	cmd := exec.CommandContext(ctx, goPath, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	cmd.Dir = cfg.WorkspaceRoot

	return cmd.Run()
}

// handleTest processes test commands
func (cfg *GowConfig) handleTest(args []string) error {
	var functionCoverage bool
	var force bool
	var root bool
	var targetDir string
	var ide bool
	var codesign bool
	var codesignEntitlements arrayFlags
	var codesignIdentity string
	var codesignForce bool
	var isCompileOnly bool

	// Parse only gow-specific flags, pass everything else through
	var goArgs []string
	goArgs = append(goArgs, "test")

	// Skip "test" and process remaining args
	i := 1
	for i < len(args) {
		arg := args[i]

		switch arg {
		case "-function-coverage":
			functionCoverage = true
		case "-force":
			force = true
		case "-root":
			root = true
		case "-ide":
			ide = true
		case "-codesign":
			codesign = true
		case "-codesign-entitlement":
			// Handle -codesign-entitlement with next argument
			if i+1 < len(args) {
				codesignEntitlements = append(codesignEntitlements, args[i+1])
				i++ // Skip the entitlement value
			}
		case "-codesign-identity":
			// Handle -codesign-identity with next argument
			if i+1 < len(args) {
				codesignIdentity = args[i+1]
				i++ // Skip the identity value
			}
		case "-codesign-force":
			codesignForce = true
		case "-c":
			// Compile test binary only (used by DAP debugging)
			isCompileOnly = true
			goArgs = append(goArgs, arg)
		case "-target":
			// Handle -target with next argument
			if i+1 < len(args) {
				targetDir = args[i+1]
				i++ // Skip the target value
			}
		case "-run":
			// Handle -run with special processing for subtest patterns
			if i+1 < len(args) {
				runPattern := args[i+1]
				// Fix VS Code subtest patterns that incorrectly match multiple test functions
				// Pattern like "^TestHarpoon/bun_version$" should only match TestHarpoon, not TestHarpoonOCI
				if strings.Contains(runPattern, "/") {
					// Extract the test function name (part before the first "/")
					parts := strings.SplitN(runPattern, "/", 2)
					if len(parts) == 2 {
						testFunc := parts[0]
						subtest := parts[1]

						// Remove leading ^ if present
						if strings.HasPrefix(testFunc, "^") {
							testFunc = testFunc[1:]
						}

						// Create a more precise pattern that ensures exact test function match
						// Pattern: ^TestFunc$/^subtest$ ensures TestFunc is matched exactly, not as prefix
						fixedPattern := "^" + testFunc + "$/" + subtest
						if cfg.Verbose {
							fmt.Printf("🔧 Fixed -run pattern: %s -> %s\n", runPattern, fixedPattern)
						}
						goArgs = append(goArgs, "-run", fixedPattern)
						i++ // Skip the run pattern value
						break
					}
				}
				// If no "/" or pattern doesn't match expected format, pass through as-is
				goArgs = append(goArgs, "-run", runPattern)
				i++ // Skip the run pattern value
			}
		default:
			// Pass through all other arguments to go test
			goArgs = append(goArgs, arg)
		}
		i++
	}

	if root && os.Geteuid() != 0 {
		return fmt.Errorf("root is required for -root flag")
	}

	// For compile-only mode (debugging), skip gow enhancements and pass through directly
	if isCompileOnly {
		if cfg.Verbose {
			fmt.Printf("🔧 Debug mode: compiling test binary with go test -c\n")
		}

		ctx := context.Background()

		// Add codesign support for debug builds
		if codesign {
			// Run the compile first
			if err := cfg.execSafeGo(ctx, goArgs...); err != nil {
				return err
			}

			// Find the output binary and sign it
			for i, arg := range goArgs {
				if arg == "-o" && i+1 < len(goArgs) {
					outputFile := goArgs[i+1]
					if cfg.Verbose {
						fmt.Printf("🔐 Code signing debug binary: %s\n", outputFile)
					}

					// Use new codesign syntax
					signArgs := []string{"tool", "github.com/walteh/ec1/tools/cmd/codesign", "-mode=sign", "-target=" + outputFile}

					// Add entitlements if specified, otherwise use default
					if len(codesignEntitlements) > 0 {
						for _, ent := range codesignEntitlements {
							signArgs = append(signArgs, "-entitlement="+ent)
						}
					} else {
						signArgs = append(signArgs, "-entitlement=virtualization")
					}

					// Add identity if specified
					if codesignIdentity != "" {
						signArgs = append(signArgs, "-identity="+codesignIdentity)
					}

					// Add force if specified
					if codesignForce {
						signArgs = append(signArgs, "-force")
					}

					signCmd := exec.CommandContext(ctx, "go", signArgs...)
					signCmd.Dir = cfg.WorkspaceRoot
					signCmd.Stdout = stdout
					signCmd.Stderr = stderr

					return signCmd.Run()
				}
			}
		}

		return cfg.execSafeGo(ctx, goArgs...)
	}

	// Add gow-specific functionality for regular test runs
	if functionCoverage {
		coverDir, err := os.MkdirTemp("", "gow-coverage-*")
		if err != nil {
			return fmt.Errorf("failed to create temp coverage dir: %w", err)
		}
		defer os.RemoveAll(coverDir)

		coverFile := filepath.Join(coverDir, "coverage.out")
		goArgs = append(goArgs, "-coverprofile="+coverFile, "-covermode=atomic")

		defer func() {
			fmt.Println("================================================")
			fmt.Println("Function Coverage")
			fmt.Println("------------------------------------------------")

			goPath, err := cfg.findSafeGo()
			if err != nil {
				fmt.Printf("Error finding go executable: %v\n", err)
				return
			}

			coverCmd := exec.Command(goPath, "tool", "cover", "-func="+coverFile)
			coverCmd.Stdout = stdout
			coverCmd.Stderr = stderr
			coverCmd.Run()

			fmt.Println("================================================")
		}()
	}

	if force {
		goArgs = append(goArgs, "-count=1")
	}

	if codesign {
		// Use new codesign test mode
		execArgs := []string{"tool", "github.com/walteh/ec1/tools/cmd/codesign", "-mode=test"}

		// Add entitlements if specified, otherwise use default
		if len(codesignEntitlements) > 0 {
			for _, ent := range codesignEntitlements {
				execArgs = append(execArgs, "-entitlement="+ent)
			}
		} else {
			execArgs = append(execArgs, "-entitlement=virtualization")
		}

		// Add identity if specified
		if codesignIdentity != "" {
			execArgs = append(execArgs, "-identity="+codesignIdentity)
		}

		// Add force if specified
		if codesignForce {
			execArgs = append(execArgs, "-force")
		}

		execArgs = append(execArgs, "-quiet")

		execArgs = append(execArgs, "--")

		goArgs = append(goArgs, "-exec=go "+strings.Join(execArgs, " "))
	}

	// Add standard flags if not already present
	hasVet := false
	hasCover := false
	for _, arg := range goArgs {
		if strings.Contains(arg, "-vet") {
			hasVet = true
		}
		if strings.Contains(arg, "-cover") {
			hasCover = true
		}
	}

	if !hasVet {
		goArgs = append(goArgs, "-vet=all")
	}
	if !hasCover {
		goArgs = append(goArgs, "-cover")
	}

	// Add target directory if specified and no other targets present
	if targetDir != "" {
		goArgs = append(goArgs, targetDir)
	}

	ctx := context.Background()

	// For IDE mode, run raw go test directly (VS Code needs this format)
	if ide {
		if cfg.Verbose {
			fmt.Printf("🔧 Using raw go test for IDE compatibility\n")
		}
		return cfg.execSafeGo(ctx, goArgs...)
	}

	// Use gotestsum if available, otherwise fall back to raw go test
	if cfg.hasGotestsum() {
		if cfg.Verbose {
			fmt.Printf("🧪 Using gotestsum for enhanced test output\n")
		}
		return cfg.runWithGotestsum(ctx, goArgs)
	}

	if cfg.Verbose {
		fmt.Printf("🔧 Using raw go test (consider installing gotestsum for better output)\n")
	}
	return cfg.execSafeGo(ctx, goArgs...)
}
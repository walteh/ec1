package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// arrayFlags implements flag.Value for string slices
type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ",")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

// GowConfig holds configuration for the Go wrapper
type GowConfig struct {
	Verbose           bool
	WorkspaceRoot     string
	GoExecutable      string
	MaxLines          int
	ErrorsToSuppress  []string
	StdoutsToSuppress []string
}

// NewGowConfig creates a new configuration with defaults
func NewGowConfig() *GowConfig {
	workspaceRoot := findWorkspaceRoot()

	return &GowConfig{
		Verbose:       false,
		WorkspaceRoot: workspaceRoot,
		GoExecutable:  "",
		MaxLines:      1000,
		ErrorsToSuppress: []string{
			"plugin.proto#L122",
			"# github.com/lima-vm/lima/cmd/limactl",
			"ld: warning: ignoring duplicate libraries: '-lobjc'",
		},
		StdoutsToSuppress: []string{
			"invalid string just to have something here",
		},
	}
}

// findWorkspaceRoot finds the workspace root by looking for go.work or go.mod files
func findWorkspaceRoot() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return "."
	}

	// Start from current directory and walk up
	dir := currentDir
	for {
		// Check for go.work (workspace root)
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}

		// Check for go.mod as fallback
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Continue looking for go.work, but remember this as potential root
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root, fallback to current directory
			return currentDir
		}
		dir = parent
	}
}

// findSafeGo finds the real go binary, avoiding recursion with our wrapper
func (cfg *GowConfig) findSafeGo() (string, error) {
	if cfg.GoExecutable != "" {
		return cfg.GoExecutable, nil
	}

	// Remove our directory from PATH to avoid recursion
	path := os.Getenv("PATH")
	pathDirs := strings.Split(path, ":")

	// Filter out workspace root from PATH to avoid calling ourselves
	var filteredDirs []string
	for _, dir := range pathDirs {
		if dir != cfg.WorkspaceRoot {
			filteredDirs = append(filteredDirs, dir)
		}
	}

	// Look for go in the filtered PATH
	for _, dir := range filteredDirs {
		goPath := filepath.Join(dir, "go")
		if _, err := os.Stat(goPath); err == nil {
			cfg.GoExecutable = goPath
			return goPath, nil
		}
	}

	return "", fmt.Errorf("could not find go executable")
}

// execSafeGo executes the real go command with given arguments using exec.Command
func (cfg *GowConfig) execSafeGo(ctx context.Context, args ...string) error {
	goPath, err := cfg.findSafeGo()
	if err != nil {
		return err
	}

	if cfg.Verbose {
		fmt.Printf("executing go command: %s %v\n", goPath, args)
	}

	cmd := exec.CommandContext(ctx, goPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// replaceProcess replaces the current process with the go command (for true pass-through)
func (cfg *GowConfig) replaceProcess(args ...string) error {
	goPath, err := cfg.findSafeGo()
	if err != nil {
		return err
	}

	if cfg.Verbose {
		fmt.Printf("replacing process with go command: %s %v\n", goPath, args)
	}

	// Use syscall.Exec to replace the current process completely
	allArgs := append([]string{goPath}, args...)
	return syscall.Exec(goPath, allArgs, os.Environ())
}

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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Dir = cfg.WorkspaceRoot

	return cmd.Run()
}

// handleTest processes test commands
func (cfg *GowConfig) handleTest(args []string) error {
	var functionCoverage bool
	var force bool
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
		default:
			// Pass through all other arguments to go test
			goArgs = append(goArgs, arg)
		}
		i++
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
					signCmd.Stdout = os.Stdout
					signCmd.Stderr = os.Stderr

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
			coverCmd.Stdout = os.Stdout
			coverCmd.Stderr = os.Stderr
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

// handleMod processes mod commands with embedded task functionality
func (cfg *GowConfig) handleMod(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("mod subcommand required")
	}

	switch args[1] {
	case "tidy":
		return cfg.runEmbeddedModTidy()
	case "upgrade":
		return cfg.runEmbeddedModUpgrade()
	default:
		return fmt.Errorf("unknown mod subcommand: %s", args[1])
	}
}

// runEmbeddedModTidy runs go mod tidy across all workspace modules
func (cfg *GowConfig) runEmbeddedModTidy() error {
	ctx := context.Background()

	if cfg.Verbose {
		fmt.Println("🧹 Running optimized mod tidy via task system...")
	}

	// Use the project's task tool to run go-mod-tidy
	goPath, err := cfg.findSafeGo()
	if err != nil {
		return fmt.Errorf("could not find go executable: %w", err)
	}

	cmd := exec.CommandContext(ctx, goPath, "tool", "github.com/go-task/task/v3/cmd/task", "go-mod-tidy")
	cmd.Dir = cfg.WorkspaceRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// runEmbeddedModUpgrade runs go-mod-upgrade and then tidy
func (cfg *GowConfig) runEmbeddedModUpgrade() error {
	ctx := context.Background()

	if cfg.Verbose {
		fmt.Println("⬆️  Running optimized mod upgrade via task system...")
	}

	// Use the project's task tool to run go-mod-upgrade
	goPath, err := cfg.findSafeGo()
	if err != nil {
		return fmt.Errorf("could not find go executable: %w", err)
	}

	cmd := exec.CommandContext(ctx, goPath, "tool", "github.com/go-task/task/v3/cmd/task", "go-mod-upgrade")
	cmd.Dir = cfg.WorkspaceRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// parseWorkspaceModules extracts module paths from go.work content
func parseWorkspaceModules(content string) []string {
	modules := make([]string, 0)
	lines := strings.Split(content, "\n")
	inUseBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "use (" {
			inUseBlock = true
			continue
		}
		if line == ")" && inUseBlock {
			inUseBlock = false
			continue
		}

		if inUseBlock && line != "" && !strings.HasPrefix(line, "//") {
			// Handle inline comments by splitting on //
			parts := strings.Split(line, "//")
			module := strings.TrimSpace(parts[0])

			// Clean up the module path (remove quotes, whitespace, etc.)
			module = strings.Trim(module, "\t \"")
			if module != "" {
				modules = append(modules, module)
			}
		}
	}

	return modules
}

// handleRetab processes retab commands
func (cfg *GowConfig) handleRetab() error {
	ctx := context.Background()

	// Read .editorconfig
	editorConfigPath := filepath.Join(cfg.WorkspaceRoot, ".editorconfig")
	editorConfig, err := os.ReadFile(editorConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read .editorconfig: %w", err)
	}

	// Run retab tool with fmt subcommand
	retabArgs := []string{
		"tool", "github.com/walteh/retab/v2/cmd/retab",
		"fmt", // Add the fmt subcommand
		"--stdin", "--stdout",
		"--editorconfig-content=" + string(editorConfig),
		"--formatter=go", // Use auto formatter instead of "go fmt"
		"-",              // Dummy filename for stdin processing
	}

	return cfg.execSafeGo(ctx, retabArgs...)
}

// handleTool processes tool commands
func (cfg *GowConfig) handleTool(args []string) error {
	ctx := context.Background()

	// Set HL_CONFIG environment variable
	if hlConfig := filepath.Join(cfg.WorkspaceRoot, "hl-config.yaml"); fileExists(hlConfig) {
		os.Setenv("HL_CONFIG", hlConfig)
	}

	// Run the tool command
	toolArgs := append([]string{"tool"}, args[1:]...)
	return cfg.execSafeGo(ctx, toolArgs...)
}

// handleDap processes dap commands
func (cfg *GowConfig) handleDap(args []string) error {
	// Add our workspace to PATH for dlv to find go
	path := os.Getenv("PATH")
	newPath := cfg.WorkspaceRoot + ":" + path
	os.Setenv("PATH", newPath)

	dlvCmd := exec.Command("dlv", append([]string{"dap"}, args[1:]...)...)
	dlvCmd.Stdout = os.Stdout
	dlvCmd.Stderr = os.Stderr
	dlvCmd.Stdin = os.Stdin

	return dlvCmd.Run()
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// printUsage shows usage information
func printUsage() {
	fmt.Println("gow - High-performance drop-in replacement for go command")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gow [any-go-command]         True pass-through to go command")
	fmt.Println()
	fmt.Println("Enhanced commands:")
	fmt.Println("  gow test [flags] [target]    Enhanced test runner with project gotestsum")
	fmt.Println("  gow mod tidy                 Optimized mod tidy via project task system")
	fmt.Println("  gow mod upgrade              Optimized mod upgrade via project task system")
	fmt.Println("  gow tool [args...]           go tool with error suppression")
	fmt.Println("  gow retab                    Format code with retab tool")
	fmt.Println("  gow dap [args...]            Run delve in DAP mode")
	fmt.Println()
	fmt.Println("Test-specific flags:")
	fmt.Println("  -function-coverage           Enable function coverage reporting")
	fmt.Println("  -force                       Force re-running of tests")
	fmt.Println("  -ide                         IDE mode: raw test output (VS Code compatible)")
	fmt.Println("  -codesign                    Enable macOS code signing for virtualization")
	fmt.Println("  -codesign-entitlement <ent>  Add Apple entitlement (can be repeated)")
	fmt.Println("                               Common: virtualization, hypervisor, network-client")
	fmt.Println("  -codesign-identity <id>      Code signing identity (default: ad-hoc '-')")
	fmt.Println("  -codesign-force              Force re-signing even if already signed")
	fmt.Println("  -v                           Verbose output")
	fmt.Println("  -run pattern                 Run only tests matching pattern")
	fmt.Println("  -target dir                  Target directory (default: .)")
	fmt.Println()
	fmt.Println("Global flags:")
	fmt.Println("  -verbose                     Verbose gow output")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gow test -codesign ./pkg/vmnet                          # Basic signing with virtualization")
	fmt.Println("  gow test -codesign-entitlement hypervisor ./pkg/host    # Custom entitlement")
	fmt.Println("  gow test -codesign -function-coverage -v ./...          # Full enhanced testing")
	fmt.Println()
	fmt.Println("All other commands are passed through to the real go binary with zero overhead.")
	fmt.Println("Enhanced commands use project tools (gotestsum, task) for optimal performance.")
}

func main() {
	cfg := NewGowConfig()

	args := os.Args[1:]

	// Parse global flags
	for i, arg := range args {
		if arg == "-verbose" || arg == "--verbose" {
			cfg.Verbose = true
			// Remove this flag from args
			args = append(args[:i], args[i+1:]...)
			break
		}
	}

	if len(args) == 0 {
		printUsage()
		return
	}

	// Handle special commands that need enhanced functionality
	switch args[0] {
	case "test":
		if err := cfg.handleTest(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error running tests: %v\n", err)
			os.Exit(1)
		}

	case "mod":
		if len(args) > 1 && (args[1] == "tidy" || args[1] == "upgrade") {
			if err := cfg.handleMod(args); err != nil {
				fmt.Fprintf(os.Stderr, "Error with mod command: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Regular mod commands - pass through
			if err := cfg.replaceProcess(args...); err != nil {
				fmt.Fprintf(os.Stderr, "Error running go: %v\n", err)
				os.Exit(1)
			}
		}

	case "retab":
		if err := cfg.handleRetab(); err != nil {
			fmt.Fprintf(os.Stderr, "Error with retab: %v\n", err)
			os.Exit(1)
		}

	case "tool":
		if err := cfg.handleTool(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error with tool: %v\n", err)
			os.Exit(1)
		}

	case "dap":
		if err := cfg.handleDap(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error with dap: %v\n", err)
			os.Exit(1)
		}

	case "gow-help", "--gow-help":
		printUsage()

	default:
		// Default: pass through to go command by replacing the process
		if err := cfg.replaceProcess(args...); err != nil {
			fmt.Fprintf(os.Stderr, "Error running go: %v\n", err)
			os.Exit(1)
		}
	}
}

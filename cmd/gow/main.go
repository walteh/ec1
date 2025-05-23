package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
	workspaceRoot, _ := os.Getwd()

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
			"invalid string just to have something heree",
		},
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

	// Filter out workspace root from PATH
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

// runSafeGo executes the real go command with given arguments
func (cfg *GowConfig) runSafeGo(ctx context.Context, args ...string) error {
	goPath, err := cfg.findSafeGo()
	if err != nil {
		return err
	}

	if cfg.Verbose {
		fmt.Printf("running go command: %s %v\n", goPath, args)
	}

	cmd := exec.CommandContext(ctx, goPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// hasGotestsum checks if gotestsum is available
func (cfg *GowConfig) hasGotestsum() bool {
	_, err := exec.LookPath("gotestsum")
	return err == nil
}

// runWithGotestsum runs tests using gotestsum
func (cfg *GowConfig) runWithGotestsum(ctx context.Context, goArgs []string) error {
	goPath, err := cfg.findSafeGo()
	if err != nil {
		return err
	}

	// Build gotestsum command
	goTestCmd := strings.Join(append([]string{goPath}, goArgs...), " ")

	args := []string{
		"--format", "pkgname",
		"--format-icons", "hivis",
		"--raw-command", "--",
		"bash", "-c", goTestCmd,
	}

	cmd := exec.CommandContext(ctx, "gotestsum", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// handleTest processes test commands
func (cfg *GowConfig) handleTest(args []string) error {
	var functionCoverage bool
	var force bool
	var verbose bool
	var runPattern string
	var targetDir string

	// Simple flag parsing for test command
	testFlags := flag.NewFlagSet("test", flag.ExitOnError)
	testFlags.BoolVar(&functionCoverage, "function-coverage", false, "enable function coverage")
	testFlags.BoolVar(&force, "force", false, "force re-running tests")
	testFlags.BoolVar(&verbose, "v", false, "verbose output")
	testFlags.StringVar(&runPattern, "run", "", "run only tests matching pattern")
	testFlags.StringVar(&targetDir, "target", ".", "target directory")

	// Parse test-specific flags
	if len(args) > 1 {
		testFlags.Parse(args[1:])
	}

	// Build go test arguments
	var goArgs []string
	goArgs = append(goArgs, "test")

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

			coverCmd := exec.Command(cfg.GoExecutable, "tool", "cover", "-func="+coverFile)
			coverCmd.Stdout = os.Stdout
			coverCmd.Stderr = os.Stderr
			coverCmd.Run()

			fmt.Println("================================================")
		}()
	}

	if force {
		goArgs = append(goArgs, "-count=1")
	}

	if verbose {
		goArgs = append(goArgs, "-v")
	}

	if runPattern != "" {
		goArgs = append(goArgs, "-run="+runPattern)
	}

	// Add standard flags
	goArgs = append(goArgs, "-vet=all", "-cover")

	// Add remaining args as test targets
	remainingArgs := testFlags.Args()
	if len(remainingArgs) > 0 {
		goArgs = append(goArgs, remainingArgs...)
	} else if targetDir != "" {
		goArgs = append(goArgs, targetDir)
	}

	ctx := context.Background()

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
	return cfg.runSafeGo(ctx, goArgs...)
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
		fmt.Println("🧹 Running embedded mod tidy across workspace modules...")
	}

	// Ensure we have a valid go executable
	if _, err := cfg.findSafeGo(); err != nil {
		return fmt.Errorf("could not find go executable: %w", err)
	}

	// Read go.work to find all modules
	workspacePath := filepath.Join(cfg.WorkspaceRoot, "go.work")
	workspaceContent, err := os.ReadFile(workspacePath)
	if err != nil {
		if cfg.Verbose {
			fmt.Printf("📁 No go.work found, running tidy on current directory\n")
		}
		// If no workspace, just run in current directory
		return cfg.runSafeGo(ctx, "mod", "tidy", "-e")
	}

	// Parse modules from go.work
	modules := parseWorkspaceModules(string(workspaceContent))
	if cfg.Verbose {
		fmt.Printf("📦 Found %d modules in workspace\n", len(modules))
	}

	// Run go mod tidy on each module
	for _, module := range modules {
		moduleDir := filepath.Join(cfg.WorkspaceRoot, module)
		if cfg.Verbose {
			fmt.Printf("  🔧 Tidying %s\n", module)
		}

		cmd := exec.CommandContext(ctx, cfg.GoExecutable, "mod", "tidy", "-e")
		cmd.Dir = moduleDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to tidy module %s: %w", module, err)
		}
	}

	if cfg.Verbose {
		fmt.Println("✅ Mod tidy completed successfully!")
	}
	return nil
}

// runEmbeddedModUpgrade runs go-mod-upgrade and then tidy
func (cfg *GowConfig) runEmbeddedModUpgrade() error {
	ctx := context.Background()

	if cfg.Verbose {
		fmt.Println("⬆️  Running embedded mod upgrade...")
	}

	// Ensure we have a valid go executable
	if _, err := cfg.findSafeGo(); err != nil {
		return fmt.Errorf("could not find go executable: %w", err)
	}

	// First run go-mod-upgrade tool
	if err := cfg.runSafeGo(ctx, "tool", "go-mod-upgrade", "--force"); err != nil {
		return fmt.Errorf("go-mod-upgrade failed: %w", err)
	}

	// Then run tidy
	return cfg.runEmbeddedModTidy()
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
			// Clean up the module path (remove quotes, whitespace, etc.)
			module := strings.Trim(line, "\t \"")
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

	// Run retab tool
	retabArgs := []string{
		"tool", "github.com/walteh/retab/v2/cmd/retab",
		"--stdin", "--stdout",
		"--editorconfig-content=" + string(editorConfig),
		"--formatter=go fmt -",
	}

	return cfg.runSafeGo(ctx, retabArgs...)
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
	return cfg.runSafeGo(ctx, toolArgs...)
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
	fmt.Println("gow - Go wrapper with enhanced functionality")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gow test [flags] [target]    Run tests with enhanced features")
	fmt.Println("  gow mod tidy                 Run mod tidy via task")
	fmt.Println("  gow mod upgrade              Run mod upgrade via task")
	fmt.Println("  gow tool [args...]           Run go tool with error suppression")
	fmt.Println("  gow retab                    Format code with retab tool")
	fmt.Println("  gow dap [args...]            Run delve in DAP mode")
	fmt.Println("  gow [go-args...]             Pass through to go command")
	fmt.Println()
	fmt.Println("Test flags:")
	fmt.Println("  -function-coverage           Enable function coverage reporting")
	fmt.Println("  -force                       Force re-running of tests")
	fmt.Println("  -v                           Verbose output")
	fmt.Println("  -run pattern                 Run only tests matching pattern")
	fmt.Println("  -target dir                  Target directory (default: .)")
	fmt.Println()
	fmt.Println("Global flags:")
	fmt.Println("  -verbose                     Verbose output")
}

func main() {
	cfg := NewGowConfig()
	ctx := context.Background()

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

	switch args[0] {
	case "test":
		if err := cfg.handleTest(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error running tests: %v\n", err)
			os.Exit(1)
		}

	case "mod":
		if err := cfg.handleMod(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error with mod command: %v\n", err)
			os.Exit(1)
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

	case "help", "-h", "--help":
		printUsage()

	default:
		// Default: pass through to go
		if err := cfg.runSafeGo(ctx, args...); err != nil {
			fmt.Fprintf(os.Stderr, "Error running go: %v\n", err)
			os.Exit(1)
		}
	}
}

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/walteh/ec1/tools/cmd/codesign/codesigncmd"
)

func init() {
	if os.Args[1] == "codesign" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
		codesigncmd.CodeSignMain()
		os.Exit(0)
	}
}

// Global stdio writers that can be wrapped for logging
var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
	stdin  io.Reader = os.Stdin
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
	PipeStdioToFile   bool
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
		Verbose:         false,
		PipeStdioToFile: false,
		WorkspaceRoot:   workspaceRoot,
		GoExecutable:    "",
		MaxLines:        1000,
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

// setupStdioLogging wraps global stdio to pipe to log file
func (cfg *GowConfig) setupStdioLogging(command string, args []string) error {
	logDir := filepath.Join(cfg.WorkspaceRoot, ".log", "gow")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Use microseconds and process ID for unique timestamps
	timestamp := fmt.Sprintf("%s_%d", time.Now().Format("2006-01-02_15-04-05.000000"), os.Getpid())
	logFile := filepath.Join(logDir, fmt.Sprintf("%s_stdio-pipe.log", timestamp))

	file, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Write command header to log file
	header := fmt.Sprintf("=== GOW STDIO LOG ===\n")
	header += fmt.Sprintf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05.000000"))
	header += fmt.Sprintf("Process ID: %d\n", os.Getpid())
	header += fmt.Sprintf("Command: %s %s\n", command, strings.Join(args, " "))
	header += fmt.Sprintf("Working Directory: %s\n", cfg.WorkspaceRoot)
	header += fmt.Sprintf("=== OUTPUT START ===\n\n")

	if _, err := file.WriteString(header); err != nil {
		file.Close()
		return fmt.Errorf("failed to write log header: %w", err)
	}

	// Wrap global stdio
	stdout = io.MultiWriter(stdout, file)
	stderr = io.MultiWriter(stderr, file)
	stdin = io.TeeReader(stdin, file) // Also capture stdin input to log

	if cfg.Verbose {
		fmt.Printf("📝 Piping stdio to: %s\n", logFile)
	}

	return nil
}

// findSafeGo finds the real go binary, avoiding recursion with our wrapper
func (cfg *GowConfig) findSafeGo() (string, error) {
	if cfg.GoExecutable != "" {
		return cfg.GoExecutable, nil
	}

	// Remove our directory from PATH to avoid recursion
	path := os.Getenv("PATH")
	pathDirs := strings.Split(path, string(os.PathListSeparator))

	// Filter out workspace root from PATH to avoid calling ourselves
	var filteredDirs []string
	for _, dir := range pathDirs {

		if dir != cfg.WorkspaceRoot {
			filteredDirs = append(filteredDirs, dir)
		}

	}

	executable, err := os.Executable()
	if err != nil {
		fmt.Printf("error getting executable: %s\n", err)
	}

	// if i am a symlink, read the link
	if data, err := os.Readlink(executable); err == nil {
		executable = data
	}

	// Look for go in the filtered PATH
	for _, dir := range filteredDirs {
		goPath := filepath.Join(dir, "go")
		if _, err := os.Stat(goPath); err == nil {
			if data, err := os.Readlink(goPath); err == nil {
				if data == executable {
					// if 'go' is a symlink to the current binary, ignore it
					continue
				}
			}
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
	cmd.Env = os.Environ()
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cmd.Dir = pwd
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin

	return cmd.Run()
}

// replaceProcess replaces the current process with the go command (for true pass-through)
func (cfg *GowConfig) replaceProcess(args ...string) error {
	// If stdio piping is enabled, we can't use process replacement
	// because we need to control stdio, so fall back to execSafeGo
	if cfg.PipeStdioToFile {
		ctx := context.Background()
		return cfg.execSafeGo(ctx, args...)
	}

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
	cmd.Stdout = stdout
	cmd.Stderr = stderr

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
	cmd.Stdout = stdout
	cmd.Stderr = stderr

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
	fmt.Println("  gow run [flags] [files...]   Enhanced runner with codesign support")
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
	fmt.Println("  -pipe-stdio-to-file          Pipe all stdio to timestamped log file (./.log/gow/)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gow test -codesign ./pkg/vmnet                          # Basic test signing with virtualization")
	fmt.Println("  gow run -codesign ./cmd/myapp                           # Basic run signing with virtualization")
	fmt.Println("  gow test -codesign-entitlement hypervisor ./pkg/host    # Custom entitlement for tests")
	fmt.Println("  gow run -codesign-entitlement hypervisor ./cmd/vm       # Custom entitlement for run")
	fmt.Println("  gow test -codesign -function-coverage -v ./...          # Full enhanced testing")
	fmt.Println("  gow -pipe-stdio-to-file build ./cmd/myapp               # Build with stdio logging")
	fmt.Println()
	fmt.Println("All other commands are passed through to the real go binary with zero overhead.")
	fmt.Println("Enhanced commands use project tools (gotestsum, task) for optimal performance.")
}

// buildCodesignExecArgs builds the -exec arguments for codesigning binaries.
// This shared function is used by both test and run commands to generate consistent
// codesign arguments for the -exec flag.
func (cfg *GowConfig) buildCodesignExecArgs(mode string, entitlements []string, identity string, force bool, additionalArgs []string) []string {
	execArgs := []string{os.Args[0], "codesign", "-mode=" + mode}

	// Add entitlements if specified, otherwise use default
	if len(entitlements) > 0 {
		for _, ent := range entitlements {
			execArgs = append(execArgs, "-entitlement="+ent)
		}
	} else {
		execArgs = append(execArgs, "-entitlement=virtualization")
	}

	// Add identity if specified
	if identity != "" {
		execArgs = append(execArgs, "-identity="+identity)
	}

	// Add force if specified
	if force {
		execArgs = append(execArgs, "-force")
	}

	if !cfg.Verbose {
		execArgs = append(execArgs, "-quiet")
	}

	execArgs = append(execArgs, additionalArgs...)

	if mode == "sign" {
		execArgs = append(execArgs, "-target")
	} else {
		execArgs = append(execArgs, "--")
	}

	return execArgs
}

// parseCodesignFlags extracts codesign-related flags from args, returning the flags and remaining args.
// This shared function parses -codesign, -codesign-entitlement, -codesign-identity, and -codesign-force
// flags from the command line arguments, returning the parsed values and the remaining non-codesign arguments.
func parseCodesignFlags(args []string) (codesign bool, entitlements []string, identity string, force bool, remainingArgs []string) {
	i := 0
	for i < len(args) {
		arg := args[i]

		switch arg {
		case "-codesign":
			codesign = true
		case "-codesign-entitlement":
			// Handle -codesign-entitlement with next argument
			if i+1 < len(args) {
				entitlements = append(entitlements, args[i+1])
				i++ // Skip the entitlement value
			}
		case "-codesign-identity":
			// Handle -codesign-identity with next argument
			if i+1 < len(args) {
				identity = args[i+1]
				i++ // Skip the identity value
			}
		case "-codesign-force":
			force = true
		default:
			remainingArgs = append(remainingArgs, arg)
		}
		i++
	}
	return
}

// handleRun processes run commands with codesign support
func (cfg *GowConfig) handleRun(args []string) error {
	var root bool
	var targetArgs []string

	// Parse codesign flags
	codesign, codesignEntitlements, codesignIdentity, codesignForce, filteredArgs := parseCodesignFlags(args)

	// Parse remaining flags
	var goArgs []string
	goArgs = append(goArgs, "run")

	var codesignAdditionalArgs []string

	if codesign {
		// Use codesign run mode - create the exec command properly
		execArgs := cfg.buildCodesignExecArgs("test", codesignEntitlements, codesignIdentity, codesignForce, codesignAdditionalArgs)

		// Format the -exec flag correctly: -exec "go tool codesign ..."
		execCommand := strings.Join(execArgs, " ")
		goArgs = append(goArgs, "-exec="+execCommand)
	}

	// Skip "run" and process remaining args
	i := 1
	for i < len(filteredArgs) {
		arg := filteredArgs[i]

		switch arg {
		case "-root":
			root = true
		default:
			// Pass through all other arguments to go run
			goArgs = append(goArgs, arg)
		}
		i++
	}

	if root && os.Geteuid() != 0 {
		return fmt.Errorf("root is required for -root flag")
	}

	// Add target arguments if specified
	if len(targetArgs) > 0 {
		goArgs = append(goArgs, targetArgs...)
	}

	ctx := context.Background()

	if cfg.Verbose {
		fmt.Printf("🚀 Running go run with args: %v\n", goArgs)
	}

	return cfg.execSafeGo(ctx, goArgs...)
}

func main() {
	cfg := NewGowConfig()

	args := os.Args[1:]

	// Parse global flags
	var filteredArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-verbose" || arg == "--verbose" {
			cfg.Verbose = true
		} else if arg == "-pipe-stdio-to-file" || arg == "--pipe-stdio-to-file" {
			cfg.PipeStdioToFile = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	args = filteredArgs

	// Setup stdio logging if requested
	if cfg.PipeStdioToFile && len(args) > 0 {
		if err := cfg.setupStdioLogging("gow", args); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting up stdio logging: %v\n", err)
			os.Exit(1)
		}
	}

	if len(args) == 0 {
		printUsage()
		return
	}

	// Handle special commands that need enhanced functionality
	switch args[0] {
	case "build":
		if err := cfg.handleBuild(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error running go build: %v\n", err)
			os.Exit(1)
		}

	case "test":
		if err := cfg.handleTest(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error running tests: %v\n", err)
			os.Exit(1)
		}

	case "run":
		if err := cfg.handleRun(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error running go run: %v\n", err)
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

func (cfg *GowConfig) handleBuild(args []string) error {
	ctx := context.Background()

	realArgs := []string{}

	// check if codesign is in the args
	codesign := false
	output := ""
	for _, arg := range args {
		if arg == "-codesign" {
			codesign = true
			continue
		}
		if strings.HasPrefix(arg, "-o=") {
			output = strings.TrimPrefix(arg, "-o=")

		}
		realArgs = append(realArgs, arg)
	}

	codesignEntitlements := []string{"virtualization"}
	codesignIdentity := ""
	codesignForce := false
	codesignAdditionalArgs := []string{}

	err := cfg.execSafeGo(ctx, realArgs...)
	if err != nil {
		return err
	}

	if codesign {
		if output == "" {
			return fmt.Errorf("'-output=/your/output/path' syntax is required for -codesign flag")
		}

		execArgs := cfg.buildCodesignExecArgs("sign", codesignEntitlements, codesignIdentity, codesignForce, codesignAdditionalArgs)
		execArgs = append(execArgs, output)

		if cfg.Verbose {
			fmt.Printf("🚀 Running codesign with args: %s %s\n", execArgs[0], strings.Join(execArgs[1:], " "))
		}

		cmd := exec.Command(execArgs[0], execArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err := cmd.Run()

		if cfg.Verbose {
			fmt.Printf("🚀 Codesign output: %v\n", err)
		}

		if err != nil {
			return fmt.Errorf("failed to exec codesign: %v", err)
		}
	}

	return nil
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/lmittmann/tint"
	"gitlab.com/tozd/go/errors"

	slogctx "github.com/veqryn/slog-context"
)

// Common Apple entitlements for different use cases
var CommonEntitlements = map[string]string{
	"virtualization":                     "com.apple.security.virtualization",
	"hypervisor":                         "com.apple.security.hypervisor",
	"network-client":                     "com.apple.security.network.client",
	"network-server":                     "com.apple.security.network.server",
	"files-user-selected":                "com.apple.security.files.user-selected.read-write",
	"files-downloads":                    "com.apple.security.files.downloads.read-write",
	"device-audio":                       "com.apple.security.device.audio-input",
	"device-camera":                      "com.apple.security.device.camera",
	"allow-jit":                          "com.apple.security.cs.allow-jit",
	"allow-unsigned-executable":          "com.apple.security.cs.allow-unsigned-executable-memory",
	"disable-executable-page-protection": "com.apple.security.cs.disable-executable-page-protection",
	"disable-library-validation":         "com.apple.security.cs.disable-library-validation",
}

type Config struct {
	Mode         string
	Target       string
	Entitlements []string
	Identity     string
	Force        bool
	Verbose      bool
	DryRun       bool
	Quiet        bool
	DapListen    string
	// For exec mode
	ExecArgs []string
}

func main() {
	ctx := context.Background()

	var config Config
	var entitlementsFlag arrayFlags
	var showEntitlements bool

	flag.StringVar(&config.Mode, "mode", "sign", "Operation mode: sign, exec, test, detect")
	flag.StringVar(&config.Target, "target", "", "File or binary to sign (required for sign mode) or analyze (for detect mode)")
	flag.Var(&entitlementsFlag, "entitlement", "Entitlement to add (can be repeated). Use common names like 'virtualization' or full identifiers")
	flag.StringVar(&config.Identity, "identity", "-", "Code signing identity (default: ad-hoc signing with '-')")
	flag.BoolVar(&config.Force, "force", false, "Force re-signing even if already signed")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Show what would be done without executing")
	flag.BoolVar(&showEntitlements, "list-entitlements", false, "List common entitlements and exit")
	flag.StringVar(&config.DapListen, "dap-listen", "", "Listen address for dap mode")
	flag.BoolVar(&config.Quiet, "quiet", false, "Quiet output")
	flag.Parse()

	config.Entitlements = entitlementsFlag
	config.ExecArgs = flag.Args()

	// Set up logging with tint (matching project style)
	level := slog.LevelInfo
	if config.Verbose {
		level = slog.LevelDebug
	}
	if config.Quiet {
		level = slog.LevelError
	}

	tintHandler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: "2006-01-02 15:04 05.0000",
		AddSource:  false, // Keep simple for tool output
		NoColor:    false,
	})

	ctxHandler := slogctx.NewHandler(tintHandler, nil)
	logger := slog.New(ctxHandler)
	slog.SetDefault(logger)

	// Store logger in context and add tool info
	ctx = slogctx.NewCtx(ctx, logger)
	ctx = slogctx.With(ctx, "tool", "codesign")

	if showEntitlements {
		listCommonEntitlements()
		return
	}

	// Handle legacy mode support for backward compatibility
	if len(os.Args) >= 3 && (os.Args[1] == "run-after-signing" || os.Args[1] == "just-sign") {
		if err := runLegacyMode(ctx); err != nil {
			slogctx.Error(ctx, "Legacy mode failed", slogctx.Err(err))
			os.Exit(1)
		}
		return
	}

	if err := runMode(ctx, &config); err != nil {
		slogctx.Error(ctx, "Operation failed", slogctx.Err(err))
		os.Exit(1)
	}
}

// arrayFlags implements flag.Value for string slices
type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ",")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func listCommonEntitlements() {
	fmt.Println("Common Apple entitlements:")
	for name, identifier := range CommonEntitlements {
		fmt.Printf("  %-35s %s\n", name, identifier)
	}
	fmt.Println("\nUsage examples:")
	fmt.Println("  # Sign with virtualization entitlement")
	fmt.Println("  codesign -mode=sign -target=./myapp -entitlement=virtualization")
	fmt.Println("  ")
	fmt.Println("  # Sign and run tests with multiple entitlements")
	fmt.Println("  codesign -mode=test -entitlement=virtualization -entitlement=network-client -- go test ./...")
}

func runMode(ctx context.Context, config *Config) error {
	switch config.Mode {
	case "sign":
		return signMode(ctx, config)
	case "exec":
		return execMode(ctx, config)
	case "test":
		return testMode(ctx, config)
	case "detect":
		return detectMode(ctx, config)
	default:
		return errors.Errorf("unknown mode %q. Supported modes: sign, exec, test, detect", config.Mode)
	}
}

func signMode(ctx context.Context, config *Config) error {
	if config.Target == "" {
		return errors.New("target file is required for sign mode")
	}

	slog.InfoContext(ctx, "Starting sign mode",
		slog.String("target", config.Target),
		slog.Any("entitlements", config.Entitlements),
		slog.String("identity", config.Identity),
		slog.Bool("force", config.Force),
		slog.Bool("dry_run", config.DryRun))

	return signBinary(ctx, config.Target, config.Entitlements, config.Identity, config.Force, config.DryRun)
}

func execMode(ctx context.Context, config *Config) error {
	if len(config.ExecArgs) == 0 {
		return errors.New("no command specified for exec mode")
	}

	// First argument should be the binary to sign and execute
	binary := config.ExecArgs[0]
	args := config.ExecArgs[1:]

	slog.InfoContext(ctx, "Starting exec mode",
		slog.String("binary", binary),
		slog.Any("args", args),
		slog.Any("entitlements", config.Entitlements))

	// Sign the binary first
	if err := signBinary(ctx, binary, config.Entitlements, config.Identity, config.Force, config.DryRun); err != nil {
		return errors.Errorf("signing binary before execution: %w", err)
	}

	// Execute the signed binary
	if config.DryRun {
		slog.InfoContext(ctx, "Would execute", slog.String("command", strings.Join(config.ExecArgs, " ")))
		return nil
	}

	if err := syscall.Exec(binary, args, os.Environ()); err != nil {
		return errors.Errorf("executing signed binary: %w", err)
	}

	return nil
}

func testMode(ctx context.Context, config *Config) error {
	if len(config.ExecArgs) == 0 {
		return errors.New("no go test command specified for test mode")
	}

	slog.InfoContext(ctx, "Starting test mode",
		slog.Any("test_args", config.ExecArgs),
		slog.Any("entitlements", config.Entitlements))

	// Create a wrapper that will sign and execute test binaries
	// execWrapper := createTestExecWrapper(ctx, config.Entitlements, config.Identity, config.Force)

	if config.DryRun {
		slog.InfoContext(ctx, "Would run go test with signing",
			// slog.String("exec_wrapper", strings.Join(execWrapper, " ")),
			slog.Any("test_args", config.ExecArgs))
		return nil
	}

	binary := config.ExecArgs[0]
	args := config.ExecArgs[1:]

	// sign the binary
	if err := signBinary(ctx, binary, config.Entitlements, config.Identity, config.Force, config.DryRun); err != nil {
		return errors.Errorf("signing binary before execution: %w", err)
	}

	// wrapAddress := os.Getenv("GOW_DAP_WRAP_ADDRESS")

	// if config.DapListen != "" {
	// 	// create a wrapper script that will
	// 	tmpDir, err := os.MkdirTemp("", "gow-codesign-dap-test-wrapper-*")
	// 	if err != nil {
	// 		return errors.Errorf("creating temp directory: %w", err)
	// 	}
	// 	script := `
	// 	#!/bin/sh
	// 	# if tmpdir/start still exists, we just run the binary
	// 	if [ -f %[1]s/start ]; then
	// 		rm %[1]s/start
	// 		exec %[2] $@
	// 	else
	// 		go tool dap exec -listen=%[3]s %[2] $@
	// 	fi
	// 	`
	// 	err = os.WriteFile(filepath.Join(tmpDir, "run"), []byte(fmt.Sprintf(script, tmpDir, binary, config.DapListen)), 0755)
	// 	if err != nil {
	// 		return errors.Errorf("creating wrapper script: %w", err)
	// 	}
	// 	err = os.WriteFile(filepath.Join(tmpDir, "start"), []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
	// 	if err != nil {
	// 		return errors.Errorf("creating start file: %w", err)
	// 	}

	// 	binary = filepath.Join(tmpDir, "run")

	// 	// os.RemoveAll(tmpDir)

	// 	return fmt.Errorf("dap wrap address: %s", config.DapListen)
	// }

	if err := syscall.Exec(binary, args, os.Environ()); err != nil {
		return errors.Errorf("executing signed binary: %w", err)
	}

	return nil
}

func createTestExecWrapper(ctx context.Context, entitlements []string, identity string, force bool) []string {
	// Return a command that will sign and execute the test binary
	args := []string{"tool", "github.com/walteh/ec1/tools/cmd/codesign", "-mode=exec"}

	if identity != "-" {
		args = append(args, "-identity="+identity)
	}

	if force {
		args = append(args, "-force")
	}

	for _, ent := range entitlements {
		args = append(args, "-entitlement="+ent)
	}

	args = append(args, "--")

	return args
}

func signBinary(ctx context.Context, target string, entitlements []string, identity string, force bool, dryRun bool) error {
	// Resolve entitlements to full identifiers
	resolvedEntitlements := make([]string, 0, len(entitlements))
	for _, ent := range entitlements {
		if fullIdent, exists := CommonEntitlements[ent]; exists {
			resolvedEntitlements = append(resolvedEntitlements, fullIdent)
		} else {
			// Assume it's already a full identifier
			resolvedEntitlements = append(resolvedEntitlements, ent)
		}
	}

	slog.InfoContext(ctx, "Preparing to sign binary",
		slog.String("target", target),
		slog.Any("resolved_entitlements", resolvedEntitlements),
		slog.Bool("dry_run", dryRun))

	// Create entitlements file if we have any entitlements
	var entitlementsFile string
	if len(resolvedEntitlements) > 0 {
		content := generateEntitlementsXML(resolvedEntitlements)

		f, err := os.CreateTemp("", "codesign-*.entitlements")
		if err != nil {
			return errors.Errorf("creating temporary entitlements file: %w", err)
		}
		defer os.Remove(f.Name())

		if _, err := f.WriteString(content); err != nil {
			f.Close()
			return errors.Errorf("writing entitlements content: %w", err)
		}

		if err := f.Close(); err != nil {
			return errors.Errorf("closing entitlements file: %w", err)
		}

		entitlementsFile = f.Name()

		if dryRun {
			slog.InfoContext(ctx, "Would create entitlements file",
				slog.String("entitlements_file", entitlementsFile),
				slog.String("content", content))
		}
	}

	// Build codesign command
	args := []string{"codesign"}

	if entitlementsFile != "" {
		args = append(args, "--entitlements", entitlementsFile)
	}

	if force {
		args = append(args, "--force")
	}

	args = append(args, "-s", identity, target)

	slog.InfoContext(ctx, "Executing codesign", slog.Any("codesign_args", args))

	if dryRun {
		slog.InfoContext(ctx, "Would execute codesign", slog.String("command", strings.Join(args, " ")))
		return nil
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Errorf("codesign failed: %w\nOutput: %s", err, string(output))
	}

	slog.InfoContext(ctx, "Successfully signed binary",
		slog.String("target", target),
		slog.Any("entitlements", entitlements))

	return nil
}

func generateEntitlementsXML(entitlements []string) string {
	var sb strings.Builder

	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)

	for _, ent := range entitlements {
		sb.WriteString(fmt.Sprintf("\t<key>%s</key>\n\t<true/>\n", ent))
	}

	sb.WriteString("</dict>\n</plist>")

	return sb.String()
}

// runLegacyMode maintains backward compatibility with the old interface
func runLegacyMode(ctx context.Context) error {
	if len(os.Args) < 3 {
		return errors.New("legacy mode requires at least 2 arguments")
	}

	mode := os.Args[1]
	target := os.Args[2]
	execArgs := os.Args[2:] // Include target as first arg for exec

	slog.InfoContext(ctx, "Running in legacy mode",
		slog.String("legacy_mode", mode),
		slog.String("target", target),
		slog.Any("exec_args", execArgs))

	config := &Config{
		Target:       target,
		Entitlements: []string{"virtualization"}, // Default for backward compatibility
		Identity:     "-",
		Force:        false,
		ExecArgs:     execArgs,
	}

	switch mode {
	case "just-sign":
		return signMode(ctx, config)
	case "run-after-signing":
		return execMode(ctx, config)
	default:
		return errors.Errorf("unknown legacy mode %q", mode)
	}
}

func detectMode(ctx context.Context, config *Config) error {
	if config.Target == "" {
		return errors.New("target directory or file is required for detect mode")
	}

	slog.InfoContext(ctx, "Starting detect mode",
		slog.String("target", config.Target))

	detector := NewEntitlementDetector()

	// Determine if target is a file or directory
	var suggested []string
	var err error

	if strings.HasSuffix(config.Target, ".go") {
		suggested, err = detector.DetectFromFile(ctx, config.Target)
	} else {
		suggested, err = detector.DetectFromPackage(ctx, config.Target)
	}

	if err != nil {
		return errors.Errorf("detecting entitlements: %w", err)
	}

	slog.InfoContext(ctx, "Entitlement detection complete",
		slog.String("target", config.Target),
		slog.Any("suggested_entitlements", suggested))

	// Output results
	fmt.Printf("Suggested entitlements for %s:\n", config.Target)
	if len(suggested) == 0 {
		fmt.Println("  (none detected - default 'virtualization' recommended)")
	} else {
		for _, ent := range suggested {
			if fullIdent, exists := CommonEntitlements[ent]; exists {
				fmt.Printf("  %-25s -> %s\n", ent, fullIdent)
			} else {
				fmt.Printf("  %s\n", ent)
			}
		}
	}

	fmt.Println("\nUsage examples:")
	if len(suggested) > 0 {
		entArgs := make([]string, 0, len(suggested))
		for _, ent := range suggested {
			entArgs = append(entArgs, "-entitlement="+ent)
		}
		fmt.Printf("  codesign -mode=sign -target=./mybinary %s\n", strings.Join(entArgs, " "))
		fmt.Printf("  gow test -codesign %s ./pkg/mypackage\n", strings.Join(entArgs, " "))
	} else {
		fmt.Printf("  codesign -mode=sign -target=./mybinary -entitlement=virtualization\n")
		fmt.Printf("  gow test -codesign ./pkg/mypackage\n")
	}

	return nil
}

package main

import (
	"context"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	slogctx "github.com/veqryn/slog-context"
)

// EntitlementDetector analyzes Go source code to suggest required entitlements
type EntitlementDetector struct {
	// Map of import paths to required entitlements
	ImportToEntitlements map[string][]string
	// Map of build tags to required entitlements
	TagToEntitlements map[string][]string
}

// NewEntitlementDetector creates a detector with common Apple framework mappings
func NewEntitlementDetector() *EntitlementDetector {
	return &EntitlementDetector{
		ImportToEntitlements: map[string][]string{
			// VZ (Virtualization.framework)
			"github.com/walteh/vz":   {"virtualization"},
			"github.com/Code-Hex/vz": {"virtualization"},

			// Network frameworks that might need entitlements
			"net/http":                     {"network-client"},
			"net":                          {"network-client"},
			"github.com/gorilla/websocket": {"network-client"},

			// File system access
			"os":            {"files-user-selected"},
			"io/fs":         {"files-user-selected"},
			"path/filepath": {"files-user-selected"},

			// Audio/video frameworks
			"github.com/veandco/go-sdl2": {"device-audio", "device-camera"},
		},
		TagToEntitlements: map[string][]string{
			// Build tags that indicate entitlement needs
			"virtualization": {"virtualization", "hypervisor"},
			"hypervisor":     {"hypervisor"},
			"network":        {"network-client", "network-server"},
			"audio":          {"device-audio"},
			"camera":         {"device-camera"},
			"jit":            {"allow-jit"},
		},
	}
}

// DetectFromPackage analyzes a Go package directory and suggests entitlements
func (d *EntitlementDetector) DetectFromPackage(ctx context.Context, packageDir string) ([]string, error) {
	slogctx.Debug(ctx, "Detecting entitlements from package", "package_dir", packageDir)

	entitlements := make(map[string]bool)

	// Parse all Go files in the package
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packageDir, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			// Check imports
			for _, imp := range file.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				if ents, exists := d.ImportToEntitlements[importPath]; exists {
					for _, ent := range ents {
						entitlements[ent] = true
					}
					slogctx.Debug(ctx, "Found entitlement from import",
						"import", importPath,
						"entitlements", ents)
				}
			}

			// Check build tags in comments
			for _, cg := range file.Comments {
				for _, comment := range cg.List {
					if strings.HasPrefix(comment.Text, "//go:build") || strings.HasPrefix(comment.Text, "// +build") {
						d.checkBuildTags(ctx, comment.Text, entitlements)
					}
					// Check for custom entitlement comments
					if strings.Contains(comment.Text, "codesign:") {
						d.checkEntitlementComments(ctx, comment.Text, entitlements)
					}
				}
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(entitlements))
	for ent := range entitlements {
		result = append(result, ent)
	}

	slogctx.Info(ctx, "Detected entitlements from source code",
		"package_dir", packageDir,
		"entitlements", result)

	return result, nil
}

// checkBuildTags examines build tag comments for entitlement hints
func (d *EntitlementDetector) checkBuildTags(ctx context.Context, comment string, entitlements map[string]bool) {
	// Extract tags from build comments
	var tags []string

	if strings.HasPrefix(comment, "//go:build") {
		// Parse go:build syntax (Go 1.17+)
		tagPart := strings.TrimPrefix(comment, "//go:build")
		tagPart = strings.TrimSpace(tagPart)
		// Simple parsing - split on common operators
		tags = strings.FieldsFunc(tagPart, func(r rune) bool {
			return r == '&' || r == '|' || r == '!' || r == '(' || r == ')' || r == ' '
		})
	} else if strings.HasPrefix(comment, "// +build") {
		// Parse legacy +build syntax
		tagPart := strings.TrimPrefix(comment, "// +build")
		tagPart = strings.TrimSpace(tagPart)
		tags = strings.Fields(tagPart)
	}

	// Check each tag for entitlement mappings
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		if ents, exists := d.TagToEntitlements[tag]; exists {
			for _, ent := range ents {
				entitlements[ent] = true
			}
			slogctx.Debug(ctx, "Found entitlement from build tag",
				"tag", tag,
				"entitlements", ents)
		}
	}
}

// checkEntitlementComments looks for custom entitlement directives in comments
func (d *EntitlementDetector) checkEntitlementComments(ctx context.Context, comment string, entitlements map[string]bool) {
	// Look for comments like:
	// // codesign: virtualization, hypervisor
	// /* codesign:network-client */

	if idx := strings.Index(comment, "codesign:"); idx >= 0 {
		entPart := comment[idx+len("codesign:"):]
		// Clean up the comment markers
		entPart = strings.TrimSuffix(entPart, "*/")
		entPart = strings.TrimSpace(entPart)

		// Split by comma and clean each entitlement
		ents := strings.Split(entPart, ",")
		for _, ent := range ents {
			ent = strings.TrimSpace(ent)
			if ent != "" {
				entitlements[ent] = true
				slogctx.Debug(ctx, "Found entitlement from comment directive",
					"entitlement", ent)
			}
		}
	}
}

// DetectFromFile analyzes a single Go file for entitlements
func (d *EntitlementDetector) DetectFromFile(ctx context.Context, filename string) ([]string, error) {
	dir := filepath.Dir(filename)
	return d.DetectFromPackage(ctx, dir)
}

// SuggestForBinary examines the source that produced a binary and suggests entitlements
func (d *EntitlementDetector) SuggestForBinary(ctx context.Context, binaryPath, sourceDir string) ([]string, error) {
	slogctx.Info(ctx, "Suggesting entitlements for binary",
		"binary", binaryPath,
		"source_dir", sourceDir)

	// If no source dir provided, we can't analyze
	if sourceDir == "" {
		slogctx.Debug(ctx, "No source directory provided, using default entitlements")
		return []string{"virtualization"}, nil
	}

	detected, err := d.DetectFromPackage(ctx, sourceDir)
	if err != nil {
		slogctx.Warn(ctx, "Failed to detect entitlements from source",
			"error", err,
			"falling_back_to_defaults", true)
		return []string{"virtualization"}, nil
	}

	// If nothing detected, provide sensible defaults
	if len(detected) == 0 {
		slogctx.Debug(ctx, "No entitlements detected, using default virtualization entitlement")
		return []string{"virtualization"}, nil
	}

	return detected, nil
}

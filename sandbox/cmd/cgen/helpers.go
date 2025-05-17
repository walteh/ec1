package main

import (
	"fmt"
	"regexp"
	"strings"
)

// sanitizeUSR turns a USR into a valid C identifier
func sanitizeUSR(usr string) string {
	// Drop USR prefix if present
	usr = strings.TrimPrefix(usr, "c:@EA@")
	// Split type and member
	parts := strings.Split(usr, "@")
	if len(parts) == 2 {
		usr = parts[0] + "_" + parts[1]
	}
	// Replace invalid chars
	re := regexp.MustCompile(`[^A-Za-z0-9_]`)
	return re.ReplaceAllString(usr, "_")
}

// isCompletion checks if a symbol is a completion handler
func isCompletion(sig FunctionSignature) bool {
	for _, param := range sig.Parameters {
		// Check for completion handler patterns in parameter names or types
		if strings.Contains(strings.ToLower(param.Name), "completion") ||
			strings.Contains(strings.ToLower(param.Name), "handler") {
			return true
		}

		// Look for closure types in parameter declarations
		for _, frag := range param.DeclarationFragments {
			if strings.Contains(frag.Spelling, "->") ||
				strings.Contains(frag.Spelling, "@escaping") {
				return true
			}
		}
	}
	return false
}

// getReturnType determines the appropriate C return type for a Swift function
func getReturnType(sig FunctionSignature) string {
	// If returns contains a non-void tuple, map to void*
	for _, r := range sig.Returns {
		if r.Spelling != "()" && r.Spelling != "" {
			return "void*"
		}
	}
	return "void*"
}

// makeParams builds parameter declarations and argument lists for C function calls
func makeParams(sig FunctionSignature) (string, string) {
	ps := []string{"void* self"}
	as := []string{}
	for _, p := range sig.Parameters {
		name := p.InternalName
		if name == "" {
			name = p.Name
		}
		cname := sanitizeUSR(name)
		ps = append(ps, fmt.Sprintf("void* %s", cname))
		as = append(as, cname)
	}
	return strings.Join(ps, ", "), strings.Join(as, ", ")
}

// swiftTypeToCType converts a Swift type to a C type
func swiftTypeToCType(swiftType string) string {
	switch {
	case strings.Contains(swiftType, "Int8"):
		return "int8_t"
	case strings.Contains(swiftType, "UInt8"):
		return "uint8_t"
	case strings.Contains(swiftType, "Int16"):
		return "int16_t"
	case strings.Contains(swiftType, "UInt16"):
		return "uint16_t"
	case strings.Contains(swiftType, "Int32"):
		return "int32_t"
	case strings.Contains(swiftType, "UInt32"):
		return "uint32_t"
	case strings.Contains(swiftType, "Int64"):
		return "int64_t"
	case strings.Contains(swiftType, "UInt64"):
		return "uint64_t"
	case strings.Contains(swiftType, "Int"):
		return "int"
	case strings.Contains(swiftType, "UInt"):
		return "unsigned int"
	case strings.Contains(swiftType, "Float"):
		return "float"
	case strings.Contains(swiftType, "Double"):
		return "double"
	case strings.Contains(swiftType, "Bool"):
		return "bool"
	case strings.Contains(swiftType, "String"):
		return "const char*"
	default:
		return "void*" // Default to opaque pointer for complex types
	}
}

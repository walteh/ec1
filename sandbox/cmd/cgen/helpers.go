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
func isSwiftSymbol(selUSR string) bool {
	// we know ObjC USRs for methods look like "c:objc(...)pauseWithCompletionHandler:"
	return strings.HasPrefix(selUSR, "s:")
}

func isCompletion(sig FunctionSignature) bool {
	return len(sig.Parameters) > 0 && sig.Parameters[len(sig.Parameters)-1].Name == "completionHandler"
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

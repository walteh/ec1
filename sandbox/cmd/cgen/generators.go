package main

import (
	"fmt"
	"strings"
)

// generateMethodShim creates C shim code for Swift methods
func generateMethodShim(root Root) string {
	// Extract module name for imports
	moduleName := root.Module.Name

	// Start with header
	result := fmt.Sprintf(`#import <Foundation/Foundation.h>
#import <objc/message.h>
#import <objc/runtime.h>
#import <stdint.h>
#import <%s/%s.h>

#ifdef __cplusplus
extern "C" {
#endif

`, moduleName, moduleName)

	// Process all type symbols first
	typeDefinitions := ""
	for _, symbol := range root.Symbols {
		if isSwiftType(symbol.Kind.Identifier) {
			typeDefinitions += processType(symbol)
		}
	}
	
	// Add type definitions to the result if there are any
	if typeDefinitions != "" {
		result += "// Swift type definitions\n"
		result += typeDefinitions
		result += "\n"
	}

	// Process all method symbols
	for _, symbol := range root.Symbols {
		// Skip completion handlers
		if isCompletion(symbol.FunctionSignature) {
			continue
		}
		
		if symbol.Kind.Identifier == "swift.method" {
			result += processMethod(symbol)
		} else if symbol.Kind.Identifier == "swift.var" || symbol.Kind.Identifier == "swift.property" {
			result += processVariable(symbol)
		}
	}

	// Add footer
	result += `
#ifdef __cplusplus
}
#endif
`
	return result
}

// Check if a symbol represents a Swift type
func isSwiftType(kindIdentifier string) bool {
	return kindIdentifier == "swift.struct" ||
		kindIdentifier == "swift.class" ||
		kindIdentifier == "swift.enum" ||
		kindIdentifier == "swift.protocol"
}

// Process a type symbol into C type definition
func processType(symbol Symbol) string {
	typeName := symbol.Names.Title
	typeImpl := ""
	
	switch symbol.Kind.Identifier {
	case "swift.enum":
		typeImpl = processEnum(symbol)
	case "swift.struct":
		typeImpl = processStruct(symbol)
	case "swift.class", "swift.protocol":
		// For classes and protocols, we use opaque pointers in C
		typeImpl = fmt.Sprintf("// %s is a Swift %s\ntypedef void* %s;\n\n", 
			typeName, 
			symbol.Kind.Identifier[6:], 
			typeName)
	}
	
	return typeImpl
}

// Process a Swift enum into a C enum definition
func processEnum(symbol Symbol) string {
	typeName := symbol.Names.Title
	
	// Start with the enum definition
	enumImpl := fmt.Sprintf("// %s represents a Swift enum\ntypedef enum {\n", typeName)
	
	// Try to extract enum cases from Children
	caseFound := false
	if len(symbol.Children) > 0 {
		for i, child := range symbol.Children {
			if child.Kind.Identifier == "swift.enum.case" {
				caseFound = true
				// If we have a value, use it, otherwise use the index as the value
				caseName := fmt.Sprintf("%s_%s", typeName, child.Name)
				if child.Value != "" {
					enumImpl += fmt.Sprintf("    %s = %s,\n", caseName, child.Value)
				} else {
					enumImpl += fmt.Sprintf("    %s = %d,\n", caseName, i)
				}
			}
		}
	}
	
	// If no cases were found, add a placeholder
	if !caseFound {
		enumImpl += fmt.Sprintf("    %s_Unknown = 0,\n", typeName)
	}
	
	// Close the enum
	enumImpl += fmt.Sprintf("} %s;\n\n", typeName)
	
	return enumImpl
}

// Process a Swift struct into a C struct definition
func processStruct(symbol Symbol) string {
	typeName := symbol.Names.Title
	
	// Start with the struct definition
	structImpl := fmt.Sprintf("// %s represents a Swift struct\ntypedef struct {\n", typeName)
	
	// Try to extract field information from DeclarationFragments
	fieldsFound := false
	if len(symbol.DeclarationFragments) > 0 {
		inFields := false
		fieldName := ""
		fieldType := ""
		
		for _, frag := range symbol.DeclarationFragments {
			// Looking for field declarations between { and }
			if frag.Spelling == "{" {
				inFields = true
				continue
			} else if frag.Spelling == "}" {
				inFields = false
				continue
			}
			
			if inFields {
				switch frag.Kind {
				case "keyword", "text":
					// Skip keywords like var, let, etc.
					continue
				case "identifier":
					if fieldName == "" {
						fieldName = frag.Spelling
					} else if fieldType == "" {
						// If we have a name but no type yet, this identifier is likely a type
						fieldType = frag.Spelling
						cType := swiftTypeToCType(fieldType)
						structImpl += fmt.Sprintf("    %s %s;\n", cType, fieldName)
						fieldsFound = true
						// Reset for next field
						fieldName = ""
						fieldType = ""
					}
				}
			}
		}
	}
	
	// If no fields were found, use a generic opaque pointer
	if !fieldsFound {
		structImpl += "    void* _internal;\n"
	}
	
	// Close the struct
	structImpl += fmt.Sprintf("} %s;\n\n", typeName)
	
	return structImpl
}

// Process a method symbol into C shim code
func processMethod(symbol Symbol) string {
	usr := sanitizeUSR(symbol.Identifier.Precise)
	returnType := getReturnType(symbol.FunctionSignature)
	params, args := makeParams(symbol.FunctionSignature)

	// Check if we have enough path components
	if len(symbol.PathComponents) < 2 {
		// Skip this method if we don't have enough path components
		return ""
	}
	
	methodName := symbol.PathComponents[1]

	// Build the method implementation
	methodImpl := fmt.Sprintf("// Shim for Swift method: %s\n", symbol.Names.Title)
	methodImpl += fmt.Sprintf("%s %s(%s) {\n", returnType, usr, params)
	methodImpl += "    id obj = (__bridge id)self;\n"
	
	// Add function pointer type for objc_msgSend
	methodImpl += fmt.Sprintf("    typedef %s (*MsgFn)(id, SEL", returnType)
	if len(symbol.FunctionSignature.Parameters) > 0 {
		methodImpl += ", id" + strings.Repeat(", id", len(symbol.FunctionSignature.Parameters))
	}
	methodImpl += ");\n"
	methodImpl += "    MsgFn fn = (MsgFn)objc_msgSend;\n"

	// Call the method
	if len(symbol.FunctionSignature.Parameters) == 0 {
		if returnType == "void*" {
			methodImpl += fmt.Sprintf("    id rv = fn(obj, sel_getUid(\"%s\"));\n", methodName)
			methodImpl += "    return (__bridge_retained void*)rv;\n"
		} else {
			methodImpl += fmt.Sprintf("    fn(obj, sel_getUid(\"%s\"));\n", methodName)
			methodImpl += "    return NULL;\n"
		}
	} else {
		methodImpl += fmt.Sprintf("    id rv = fn(obj,\n")
		methodImpl += fmt.Sprintf("               sel_getUid(\"%s:\"),\n", methodName)
		methodImpl += fmt.Sprintf("               %s);\n", args)
		methodImpl += "    return (__bridge_retained void*)rv;\n"
	}
	
	methodImpl += "}\n\n"
	return methodImpl
}

// Process a variable or property symbol into C shim code
func processVariable(symbol Symbol) string {
	usr := sanitizeUSR(symbol.Identifier.Precise)
	
	// Check if we have enough path components
	if len(symbol.PathComponents) < 2 {
		// Skip this variable if we don't have enough path components
		return ""
	}
	
	propertyName := symbol.PathComponents[1]
	
	// Build getter function
	getterImpl := fmt.Sprintf("// Shim for Swift property getter: %s\n", symbol.Names.Title)
	getterImpl += fmt.Sprintf("void* %s_get(void* self) {\n", usr)
	getterImpl += "    id obj = (__bridge id)self;\n"
	getterImpl += "    typedef id (*MsgFn)(id, SEL);\n"
	getterImpl += "    MsgFn fn = (MsgFn)objc_msgSend;\n"
	getterImpl += fmt.Sprintf("    id rv = fn(obj, sel_getUid(\"%s\"));\n", propertyName)
	getterImpl += "    return (__bridge_retained void*)rv;\n"
	getterImpl += "}\n\n"
	
	// Build setter function if property is not readonly
	// Note: This is a simplification - we would need to check if property is writable
	setterImpl := fmt.Sprintf("// Shim for Swift property setter: %s\n", symbol.Names.Title)
	setterImpl += fmt.Sprintf("void %s_set(void* self, void* value) {\n", usr)
	setterImpl += "    id obj = (__bridge id)self;\n"
	setterImpl += "    id val = (__bridge id)value;\n"
	setterImpl += "    typedef void (*MsgFn)(id, SEL, id);\n"
	setterImpl += "    MsgFn fn = (MsgFn)objc_msgSend;\n"
	setterImpl += fmt.Sprintf("    fn(obj, sel_getUid(\"set%s:\"), val);\n", capitalizeFirst(propertyName))
	setterImpl += "}\n\n"
	
	return getterImpl + setterImpl
}

// Generate Go type definitions from Swift types
func generateGoTypes(root Root) string {
	packageName := strings.ToLower(root.Module.Name)
	
	result := fmt.Sprintf(`// Code generated by cgen; DO NOT EDIT.
package %s

import (
	"unsafe"
)

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework %s
// #include <%s/%s.h>
// #include "%s.shim.h"
import "C"

`, packageName, root.Module.Name, root.Module.Name, root.Module.Name, root.Module.Name)

	// Process all type symbols
	typeMap := make(map[string]bool)
	for _, symbol := range root.Symbols {
		if isSwiftType(symbol.Kind.Identifier) {
			typeName := symbol.Names.Title
			if typeMap[typeName] {
				continue // Skip duplicates
			}
			typeMap[typeName] = true
			
			result += processGoType(symbol)
		}
	}
	
	return result
}

// Process a type symbol into Go type definition with methods
func processGoType(symbol Symbol) string {
	typeName := symbol.Names.Title
	
	// Start with type definition
	typeImpl := fmt.Sprintf("// %s represents a Swift %s\n", typeName, symbol.Kind.Identifier[6:])
	typeImpl += fmt.Sprintf("type %s struct {\n", typeName)
	typeImpl += "    ptr unsafe.Pointer\n"
	typeImpl += "}\n\n"
	
	// TODO: Add factory methods and instance methods for this type
	// We would need to filter the symbols to find methods related to this type
	
	return typeImpl
}

// Helper function to capitalize the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

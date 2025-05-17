package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
)

// Root mirrors the top‑level symbol graph JSON.
type Root struct {
	Module  Module   `json:"module"`
	Symbols []Symbol `json:"symbols"`
}

// Module holds the module name.
type Module struct {
	Name string `json:"name"`
}

// Symbol represents an entry in the symbol graph.
type Symbol struct {
	Kind struct {
		Identifier string `json:"identifier"`
	} `json:"kind"`
	Identifier struct {
		Precise           string `json:"precise"`
		InterfaceLanguage string `json:"interfaceLanguage"`
	} `json:"identifier"`
	Names struct {
		Title string `json:"title"`
	} `json:"names"`
	PathComponents    []string          `json:"pathComponents"`
	FunctionSignature FunctionSignature `json:"functionSignature"`
}

// FunctionSignature contains parameters and return info.
type FunctionSignature struct {
	Parameters []Parameter `json:"parameters"`
	Returns    []Return    `json:"returns"`
}

// Parameter describes a function parameter.
type Parameter struct {
	Name                 string     `json:"name"`
	InternalName         string     `json:"internalName"`
	DeclarationFragments []Fragment `json:"declarationFragments"`
}

// Return describes a function return value.
type Return struct {
	Kind     string `json:"kind"`
	Spelling string `json:"spelling"`
}

// Fragment represents a token fragment in declarations.
type Fragment struct {
	Kind              string  `json:"kind"`
	Spelling          string  `json:"spelling"`
	PreciseIdentifier *string `json:"preciseIdentifier,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <symbol-graph-json>", os.Args[0])
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	var root Root
	if err := json.Unmarshal(data, &root); err != nil {
		log.Fatal(err)
	}

	// sanitizeUSR turns a USR into a valid C identifier
	sanitizeUSR := func(usr string) string {
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

	// Choose return type based on signature (default to void*)
	getReturnType := func(sig FunctionSignature) string {
		// If returns contains a non-void tuple, map to void*
		for _, r := range sig.Returns {
			if r.Spelling != "()" && r.Spelling != "" {
				return "void*"
			}
		}
		return "void*"
	}

	// Build parameter declaration list and argument list from signature
	makeParams := func(sig FunctionSignature) (string, string) {
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

	makeParams0 := func(sig FunctionSignature) string {
		p0, _ := makeParams(sig)
		return p0
	}

	makeParams1 := func(sig FunctionSignature) string {
		_, p1 := makeParams(sig)
		return p1
	}

	tmpl := template.Must(template.New("shim").Funcs(template.FuncMap{
		"sanitizeUSR":   sanitizeUSR,
		"getReturnType": getReturnType,
		"isCompletion": func(selUSR string) bool {
			// we know ObjC USRs for methods look like "c:objc(...)pauseWithCompletionHandler:"
			return strings.HasPrefix(selUSR, "s:")
		},
		"repeat": func(s string, count int) string {
			return strings.Repeat(s, count)
		},
		"makeParams0": makeParams0,
		"makeParams1": makeParams1,
	}).Parse(`
#import <Foundation/Foundation.h>
#import <objc/message.h>
#import <objc/runtime.h>
#import <{{.Module.Name}}/{{.Module.Name}}.h>

#ifdef __cplusplus
extern "C" {
#endif

{{- range .Symbols}}{{- if and (eq .Kind.Identifier "swift.method") (not (isCompletion .Identifier.Precise)) }}
// Shim for Swift method: {{ .Names.Title }}
{{ getReturnType .FunctionSignature }} {{ sanitizeUSR .Identifier.Precise }}({{ makeParams0 .FunctionSignature }}) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL{{ if gt (len .FunctionSignature.Parameters) 0 }}, id{{ repeat ", id" (len .FunctionSignature.Parameters) }}{{ end }}) -> {{ getReturnType .FunctionSignature }}
    typedef {{ getReturnType .FunctionSignature }} (*MsgFn)(id, SEL{{ if gt (len .FunctionSignature.Parameters) 0 }}, id{{ repeat ", id" (len .FunctionSignature.Parameters) }}{{ end }});
    MsgFn fn = (MsgFn)objc_msgSend;
    {{- $args := makeParams1 .FunctionSignature }}
    {{- if eq (len .FunctionSignature.Parameters) 0 }}
    {{ if eq (getReturnType .FunctionSignature) "void*" }}
    id rv = fn(obj, sel_getUid("{{ index .PathComponents 1 }}"));
    return (__bridge_retained void*)rv;
    {{- else }}
    fn(obj, sel_getUid("{{ index .PathComponents 1 }}"));
    return NULL;
    {{- end }}
    {{- else }}
    id rv = fn(obj,
               sel_getUid("{{ index .PathComponents 1 }}:"),
               {{ $args }});
    return (__bridge_retained void*)rv;
    {{- end }}
}

{{end}}{{end}}

#ifdef __cplusplus
}
#endif
`))

	if err := tmpl.Execute(os.Stdout, root); err != nil {
		log.Fatal(err)
	}
}

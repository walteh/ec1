package main

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
	Type              *TypeInfo         `json:"type,omitempty"`
}

// TypeInfo holds information about a type's structure
type TypeInfo struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
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

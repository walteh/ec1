package termlog

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/xlab/treeprint"
)

// StructToTree converts any Go struct (or value) to a styled ASCII tree.
func StructToTree(v interface{}) string {
	// Reflectively build a tree from the struct
	tree, err := treeprint.FromStruct(v) // auto‐map exported fields to branches  [oai_citation:6‡pkg.go.dev](https://pkg.go.dev/github.com/xlab/treeprint?utm_source=chatgpt.com)
	if err != nil {
		return fmt.Sprintf("(unable to render struct: %s) %+v", err, v)
	}

	// Style the tree block: padding, border, and color
	style := lipgloss.NewStyle(). // create a new style  [oai_citation:7‡github.com](https://github.com/charmbracelet/lipgloss?utm_source=chatgpt.com)
					Padding(1, 2). // 1 line vertical, 2 spaces horizontal

		// Border(lipgloss.ASCIIBorder()).   // simple ASCII border  [oai_citation:8‡pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/lipgloss?utm_source=chatgpt.com)
		Foreground(lipgloss.Color("245")) // gray text

	// Render the styled tree string, TTY‐aware
	return style.Render(tree.String()) // tree.String() gives the ASCII output  [oai_citation:9‡pkg.go.dev](https://pkg.go.dev/github.com/xlab/treeprint?utm_source=chatgpt.com)
}

// JSONToTree parses a JSON byte slice and returns a styled ASCII tree.
func JSONToTree(name string, data []byte) string {
	// Unmarshal into an empty interface
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil { // error on malformed JSON  [oai_citation:14‡pkg.go.dev](https://pkg.go.dev/encoding/json?utm_source=chatgpt.com)
		return fmt.Sprintf("(unable to render JSON: %s) %s", err, string(data))
	}

	// Start a new tree with a root labeled "JSON"
	root := treeprint.NewWithRoot("JSON") // custom root  [oai_citation:15‡pkg.go.dev](https://pkg.go.dev/github.com/xlab/treeprint?utm_source=chatgpt.com)

	// Recursively build branches for maps, arrays, and primitives
	attachMapToNode(root, raw)

	// Style and render as before
	style := lipgloss.NewStyle().
		Padding(1, 2).
		// Border(lipgloss.ASCIIBorder()).
		Foreground(lipgloss.Color("240"))
	return style.Render(root.String())
}

// attachMapToNode walks the JSON structure and adds branches/nodes.
func attachMapToNode(node treeprint.Tree, v interface{}) {
	switch vv := v.(type) {
	case map[string]interface{}:
		// For each key, create a branch and recurse
		for key, val := range vv {
			branch := node.AddBranch(key) // AddBranch(v) creates a new subtree  [oai_citation:16‡pkg.go.dev](https://pkg.go.dev/github.com/xlab/treeprint?utm_source=chatgpt.com)
			attachMapToNode(branch, val)
		}
	case []interface{}:
		// For arrays, index each element
		for i, item := range vv {
			branch := node.AddBranch(fmt.Sprintf("[%d]", i))
			attachMapToNode(branch, item)
		}
	default:
		// Leaf node: any primitive becomes a terminal node
		node.AddNode(fmt.Sprint(vv)) // AddNode(v) for scalar values  [oai_citation:17‡pkg.go.dev](https://pkg.go.dev/github.com/xlab/treeprint?utm_source=chatgpt.com)
	}
}

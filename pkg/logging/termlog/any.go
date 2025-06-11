package termlog

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

// StructToTree converts any Go struct (or value) to a stunning lipgloss tree.
func StructToTree(v interface{}, styles *Styles, render renderFunc) string {
	return StructToTreeWithTitle(v, "struct", styles, render)
}

// StructToTreeWithTitle creates a struct tree with a custom title
func StructToTreeWithTitle(v interface{}, title string, styles *Styles, render renderFunc) string {
	t := tree.Root("󰙅 " + title).
		RootStyle(styles.Tree.Root).
		EnumeratorStyle(styles.Tree.Branch).
		Enumerator(tree.RoundedEnumerator).
		ItemStyle(styles.Tree.Key)

	buildStructTreeNode(t, reflect.ValueOf(v), styles.Tree, render)

	return render(styles.Tree.Container, t.String())
}

type renderFunc func(s lipgloss.Style, str string) string

// JSONToTree parses a JSON byte slice and returns a stunning lipgloss tree.
func JSONToTree(keyName string, data []byte, styles *Styles, render renderFunc) string {
	// Unmarshal into an empty interface
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Sprintf("(unable to render JSON: %s) %s", err, string(data))
	}

	t := tree.Root("󰘦 " + keyName).
		RootStyle(styles.Tree.Root).
		EnumeratorStyle(styles.Tree.Branch).
		Enumerator(tree.RoundedEnumerator).
		ItemStyle(styles.Tree.Key)

	buildJSONTreeNode(t, raw, styles.Tree, render)

	return render(styles.Tree.Container, t.String())
}

// buildStructTreeNode recursively builds a lipgloss tree from a struct using reflection
func buildStructTreeNode(node *tree.Tree, v reflect.Value, styles TreeStyles, render renderFunc) {
	if !v.IsValid() {
		return
	}

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			node.Child(render(styles.Null, "nil"))
			return
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		numFields := v.NumField()

		for i := 0; i < numFields; i++ {
			field := t.Field(i)
			fieldValue := v.Field(i)

			// Skip unexported fields
			if !field.IsExported() {
				continue
			}

			if isSimpleType(fieldValue) {
				// Simple value - show as key: value
				keyStr := render(styles.Key, "󰌽 "+field.Name)
				valueStr := renderValue(fieldValue.Interface(), styles, render)
				node.Child(keyStr + ": " + valueStr)
			} else {
				// Complex value - create subtree
				keyStr := render(styles.Key, "󰌽 "+field.Name)
				subTree := tree.New().Root(keyStr).
					RootStyle(styles.Key).
					EnumeratorStyle(styles.Branch).
					Enumerator(tree.RoundedEnumerator).
					ItemStyle(styles.Key)
				buildStructTreeNode(subTree, fieldValue, styles, render)
				node.Child(subTree)
			}
		}

	case reflect.Slice, reflect.Array:
		length := v.Len()
		for i := 0; i < length; i++ {
			item := v.Index(i)
			indexStr := render(styles.Index, fmt.Sprintf("󰅪 [%d]", i))

			if isSimpleType(item) {
				valueStr := renderValue(item.Interface(), styles, render)
				node.Child(indexStr + ": " + valueStr)
			} else {
				subTree := tree.New().Root(indexStr).
					RootStyle(styles.Index).
					EnumeratorStyle(styles.Branch).
					Enumerator(tree.RoundedEnumerator).
					ItemStyle(styles.Key)
				buildStructTreeNode(subTree, item, styles, render)
				node.Child(subTree)
			}
		}

	case reflect.Map:
		keys := v.MapKeys()
		for _, key := range keys {
			mapValue := v.MapIndex(key)
			keyStr := render(styles.Key, "󰌽 "+fmt.Sprint(key.Interface()))

			if isSimpleType(mapValue) {
				valueStr := renderValue(mapValue.Interface(), styles, render)
				node.Child(keyStr + ": " + valueStr)
			} else {
				subTree := tree.New().Root(keyStr).
					RootStyle(styles.Key).
					EnumeratorStyle(styles.Branch).
					ItemStyle(styles.Key)
				buildStructTreeNode(subTree, mapValue, styles, render)
				node.Child(subTree)
			}
		}
	}
}

// buildJSONTreeNode recursively builds a lipgloss tree from JSON data
func buildJSONTreeNode(node *tree.Tree, v interface{}, styles TreeStyles, render renderFunc) {
	switch vv := v.(type) {
	case map[string]interface{}:
		for key, jsonVal := range vv {
			keyStr := render(styles.Key, "󰌽 "+key)

			if isSimpleJSONType(jsonVal) {
				valueStr := renderJSONValue(jsonVal, styles, render)
				node.Child(keyStr + ": " + valueStr)
			} else {
				subTree := tree.New().Root(keyStr).
					RootStyle(styles.Key).
					EnumeratorStyle(styles.Branch).
					Enumerator(tree.RoundedEnumerator).
					ItemStyle(styles.Key)
				buildJSONTreeNode(subTree, jsonVal, styles, render)
				node.Child(subTree)
			}
		}

	case []interface{}:
		for i, item := range vv {
			indexStr := render(styles.Index, fmt.Sprintf("󰅪 [%d]", i))

			if isSimpleJSONType(item) {
				valueStr := renderJSONValue(item, styles, render)
				node.Child(indexStr + ": " + valueStr)
			} else {
				subTree := tree.New().Root(indexStr).
					RootStyle(styles.Index).
					EnumeratorStyle(styles.Branch).
					Enumerator(tree.RoundedEnumerator).
					ItemStyle(styles.Key)
				buildJSONTreeNode(subTree, item, styles, render)
				node.Child(subTree)
			}
		}
	}
}

// isSimpleType checks if a reflect.Value represents a simple/primitive type
func isSimpleType(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	case reflect.Ptr:
		if v.IsNil() {
			return true
		}
		return isSimpleType(v.Elem())
	default:
		return false
	}
}

// isSimpleJSONType checks if a JSON value is a simple/primitive type
func isSimpleJSONType(v interface{}) bool {
	switch v.(type) {
	case string, bool, float64, nil:
		return true
	default:
		return false
	}
}

// renderValue renders a Go value with appropriate styling
func renderValue(v interface{}, styles TreeStyles, render renderFunc) string {
	if v == nil {
		return render(styles.Null, " nil")
	}

	switch vv := v.(type) {
	case string:
		return render(styles.String, fmt.Sprintf(` "%s"`, vv))
	case bool:
		if vv {
			return render(styles.Bool, "󰱒 true")
		} else {
			return render(styles.Bool, "󰰋 false")
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return render(styles.Number, "󰎠 "+fmt.Sprint(vv))
	default:
		return render(styles.Struct, "󰙅 "+fmt.Sprintf("(%T)", vv))
	}
}

// renderJSONValue renders a JSON value with appropriate styling
func renderJSONValue(v interface{}, styles TreeStyles, render renderFunc) string {
	switch vv := v.(type) {
	case nil:
		return render(styles.Null, " null")
	case string:
		return render(styles.String, fmt.Sprintf(` "%s"`, vv))
	case bool:
		if vv {
			return render(styles.Bool, "󰱒 true")
		} else {
			return render(styles.Bool, "󰰋 false")
		}
	case float64:
		return render(styles.Number, "󰎠 "+strconv.FormatFloat(vv, 'g', -1, 64))
	default:
		return render(styles.Struct, "󰙅 "+fmt.Sprint(vv))
	}
}

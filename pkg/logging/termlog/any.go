package termlog

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StructToTree converts any Go struct (or value) to a beautifully styled ASCII tree.
func StructToTree(v interface{}, styles *Styles, render renderFunc) string {
	// Build the tree content
	var b strings.Builder
	b.WriteString(render(styles.Tree.Root, "struct") + "\n")

	// Use reflection to walk the struct
	buildStructTree(&b, reflect.ValueOf(v), "", true, styles.Tree, render)

	// Wrap in container style
	return render(styles.Tree.Container, b.String())
}

type renderFunc func(s lipgloss.Style, str string) string

// JSONToTree parses a JSON byte slice and returns a beautifully styled ASCII tree.
func JSONToTree(name string, data []byte, styles *Styles, render renderFunc) string {

	// Unmarshal into an empty interface
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Sprintf("(unable to render JSON: %s) %s", err, string(data))
	}

	// Build the tree content
	var b strings.Builder
	b.WriteString(render(styles.Tree.Root, "JSON") + "\n")

	// Recursively build the tree
	buildJSONTree(&b, raw, "", true, styles.Tree, render)

	// Wrap in container style
	return render(styles.Tree.Container, b.String())
}

// buildStructTree recursively builds a colored tree from a struct using reflection
func buildStructTree(b *strings.Builder, v reflect.Value, prefix string, isLast bool, styles TreeStyles, render renderFunc) {
	if !v.IsValid() {
		return
	}

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
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

			isLastField := i == numFields-1
			connector := "├── "
			newPrefix := prefix + "│   "

			if isLastField {
				connector = "└── "
				newPrefix = prefix + "    "
			}

			branchStr := render(styles.Branch, connector)
			keyStr := render(styles.Key, field.Name)

			b.WriteString(prefix + branchStr + keyStr)

			// Add value inline for simple types, recurse for complex types
			if isSimpleType(fieldValue) {
				valueStr := renderValue(fieldValue.Interface(), styles, render)
				b.WriteString(": " + valueStr)
			}

			b.WriteString("\n")

			// Recurse for complex types
			if !isSimpleType(fieldValue) {
				buildStructTree(b, fieldValue, newPrefix, isLastField, styles, render)
			}
		}

	case reflect.Slice, reflect.Array:
		length := v.Len()
		for i := 0; i < length; i++ {
			isLastItem := i == length-1
			connector := "├── "
			newPrefix := prefix + "│   "

			if isLastItem {
				connector = "└── "
				newPrefix = prefix + "    "
			}

			branchStr := render(styles.Branch, connector)
			indexStr := render(styles.Index, fmt.Sprintf("[%d]", i))

			b.WriteString(prefix + branchStr + indexStr)

			item := v.Index(i)
			if isSimpleType(item) {
				valueStr := renderValue(item.Interface(), styles, render)
				b.WriteString(": " + valueStr)
			}

			b.WriteString("\n")

			if !isSimpleType(item) {
				buildStructTree(b, item, newPrefix, isLastItem, styles, render)
			}
		}

	case reflect.Map:
		keys := v.MapKeys()
		for i, key := range keys {
			isLastKey := i == len(keys)-1
			connector := "├── "
			newPrefix := prefix + "│   "

			if isLastKey {
				connector = "└── "
				newPrefix = prefix + "    "
			}

			branchStr := render(styles.Branch, connector)
			keyStr := render(styles.Key, fmt.Sprint(key.Interface()))

			b.WriteString(prefix + branchStr + keyStr)

			mapValue := v.MapIndex(key)
			if isSimpleType(mapValue) {
				valueStr := renderValue(mapValue.Interface(), styles, render)
				b.WriteString(": " + valueStr)
			}

			b.WriteString("\n")

			if !isSimpleType(mapValue) {
				buildStructTree(b, mapValue, newPrefix, isLastKey, styles, render)
			}
		}
	}
}

// buildJSONTree recursively builds a colored tree from JSON data
func buildJSONTree(b *strings.Builder, v interface{}, prefix string, isLast bool, styles TreeStyles, render renderFunc) {
	switch vv := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(vv))
		for k := range vv {
			keys = append(keys, k)
		}

		for i, key := range keys {
			isLastKey := i == len(keys)-1
			connector := "├── "
			newPrefix := prefix + "│   "

			if isLastKey {
				connector = "└── "
				newPrefix = prefix + "    "
			}

			branchStr := render(styles.Branch, connector)
			keyStr := render(styles.Key, key)

			b.WriteString(prefix + branchStr + keyStr)

			val := vv[key]
			if isSimpleJSONType(val) {
				valueStr := renderJSONValue(val, styles, render)
				b.WriteString(": " + valueStr)
			}

			b.WriteString("\n")

			if !isSimpleJSONType(val) {
				buildJSONTree(b, val, newPrefix, isLastKey, styles, render)
			}
		}

	case []interface{}:
		for i, item := range vv {
			isLastItem := i == len(vv)-1
			connector := "├── "
			newPrefix := prefix + "│   "

			if isLastItem {
				connector = "└── "
				newPrefix = prefix + "    "
			}

			branchStr := render(styles.Branch, connector)
			indexStr := render(styles.Index, fmt.Sprintf("[%d]", i))

			b.WriteString(prefix + branchStr + indexStr)

			if isSimpleJSONType(item) {
				valueStr := renderJSONValue(item, styles, render)
				b.WriteString(": " + valueStr)
			}

			b.WriteString("\n")

			if !isSimpleJSONType(item) {
				buildJSONTree(b, item, newPrefix, isLastItem, styles, render)
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
		return render(styles.Null, "nil")
	}

	switch vv := v.(type) {
	case string:
		return render(styles.String, fmt.Sprintf(`"%s"`, vv))
	case bool:
		return render(styles.Bool, strconv.FormatBool(vv))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return render(styles.Number, fmt.Sprint(vv))
	default:
		return render(styles.Struct, fmt.Sprintf("(%T)", vv))
	}
}

// renderJSONValue renders a JSON value with appropriate styling
func renderJSONValue(v interface{}, styles TreeStyles, render renderFunc) string {
	switch vv := v.(type) {
	case nil:
		return render(styles.Null, "null")
	case string:
		return render(styles.String, fmt.Sprintf(`"%s"`, vv))
	case bool:
		return render(styles.Bool, strconv.FormatBool(vv))
	case float64:
		return render(styles.Number, strconv.FormatFloat(vv, 'g', -1, 64))
	default:
		return render(styles.Struct, fmt.Sprint(vv))
	}
}

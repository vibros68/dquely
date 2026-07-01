package dquely

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Mutation serializes a struct pointer to a DGraph RDF N-Quad set mutation.
// input must be a non-nil pointer to a struct, and the struct must contain a field
// tagged `dquely:"uid"` (used to receive the created UID via SetUID after execution).
// Fields are mapped using the `dquely` struct tag as the predicate name.
// The blank node is the lowercased struct type name (e.g. *User → _:user).
// String fields are emitted first (in declaration order), then numeric/other fields,
// and dgraph.type is always appended last.
func Mutation(input any) (string, error) {
	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)
	if v.Kind() != reflect.Ptr {
		return "", fmt.Errorf("dquely: Mutation expects a pointer to struct, got %s", v.Kind())
	}
	v = v.Elem()
	t = t.Elem()
	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("dquely: Mutation expects a pointer to struct, got pointer to %s", v.Kind())
	}
	if !hasUIDField(t) {
		return "", fmt.Errorf("dquely: Mutation requires a field tagged dquely:\"uid\" in the struct")
	}

	typeName := t.Name()
	if dm, ok := input.(DgraphMutation); ok {
		typeName = dm.DgraphType()
	}
	blankNode := "_:" + strings.ToLower(typeName)

	var sb strings.Builder
	sb.WriteString("{\n  set {\n")

	// Strings and json-encoded fields first, then other kinds — each in struct declaration order.
	for _, stringPass := range []bool{true, false} {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			rawTag := field.Tag.Get("dquely")
			if rawTag == "-" {
				continue
			}
			predicate, isJSON, _ := parseTag(rawTag, field.Name)
			isString := field.Type.Kind() == reflect.String || isJSON
			if isString != stringPass {
				continue
			}
			val := v.Field(i)
			if val.IsZero() {
				continue
			}
			var valueStr string
			if isJSON {
				b, err := json.Marshal(val.Interface())
				if err != nil {
					return "", fmt.Errorf("dquely: failed to marshal field %s as JSON: %w", field.Name, err)
				}
				valueStr = strings.ReplaceAll(string(b), `"`, `\"`)
			} else {
				valueStr = escapeString(fmt.Sprintf("%v", val.Interface()))
			}
			sb.WriteString(fmt.Sprintf("    %s <%s> \"%s\" .\n", blankNode, predicate, valueStr))
		}
	}

	sb.WriteString(fmt.Sprintf("    %s <dgraph.type> \"%s\" .\n", blankNode, typeName))
	sb.WriteString("  }\n}")
	return sb.String(), nil
}

type DgraphMutation interface {
	DgraphType() string
}

// parseTag splits a raw dquely struct tag into the predicate name and options.
// Falls back to fieldName when the name part is empty.
// Returns isJSON=true when the "json" option is present.
// Returns isUnique=true when the "unique" option is present.
func parseTag(rawTag, fieldName string) (predicate string, isJSON bool, isUnique bool) {
	predicate = rawTag
	if idx := strings.Index(rawTag, ","); idx >= 0 {
		predicate = rawTag[:idx]
		for _, opt := range strings.Split(rawTag[idx+1:], ",") {
			switch opt {
			case "json":
				isJSON = true
			case "unique":
				isUnique = true
			}
		}
	}
	if predicate == "" {
		predicate = fieldName
	}
	return
}

// UniqueField holds the predicate name and current string value of a struct field
// tagged with the "unique" option (e.g. `dquely:"email,unique"`).
type UniqueField struct {
	Predicate string
	Value     string
}

// UniqueFields reflects over input and returns every field tagged with the "unique"
// option in its dquely tag. Returns an error if input is not a struct (or pointer to one).
func UniqueFields(input any) ([]UniqueField, error) {
	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("dquely: UniqueFields expects a struct, got %s", v.Kind())
	}
	var fields []UniqueField
	for i := 0; i < t.NumField(); i++ {
		rawTag := t.Field(i).Tag.Get("dquely")
		if rawTag == "-" {
			continue
		}
		predicate, _, isUnique := parseTag(rawTag, t.Field(i).Name)
		if !isUnique {
			continue
		}
		fields = append(fields, UniqueField{
			Predicate: predicate,
			Value:     fmt.Sprintf("%v", v.Field(i).Interface()),
		})
	}
	return fields, nil
}

// hasUIDField reports whether t contains a field whose dquely predicate is "uid".
func hasUIDField(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		rawTag := t.Field(i).Tag.Get("dquely")
		predicate, _, _ := parseTag(rawTag, t.Field(i).Name)
		if predicate == "uid" {
			return true
		}
	}
	return false
}

// BlankNodeName returns the blank-node key that Mutation() uses for the struct,
// i.e. the lowercased type name (or DgraphType() if the struct implements DgraphMutation).
// This key matches the entry in api.Response.Uids after a successful mutation.
// input may be a pointer or a value.
func BlankNodeName(input any) (string, error) {
	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("dquely: BlankNodeName expects a struct or pointer to struct, got %s", v.Kind())
	}
	typeName := t.Name()
	if dm, ok := input.(DgraphMutation); ok {
		typeName = dm.DgraphType()
	}
	return strings.ToLower(typeName), nil
}

// SetUIDs distributes UIDs from a DGraph mutation response into a struct and its
// direct nested structs.  The keys in uids are matched as follows:
//
//   - A nested pointer-to-struct field is matched by strings.ToLower(field.Name).
//   - A nested slice-of-struct element at index j is matched by
//     strings.ToLower(field.Name) + strconv.Itoa(j).
//   - Any key that does not match a field pattern is assumed to be the root struct's
//     UID and is applied via SetUID.
//
// input must be a non-nil pointer to a struct with a dquely:"uid" field.
func SetUIDs(input any, uids map[string]string) error {
	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("dquely: SetUIDs expects a pointer to struct, got %s", v.Kind())
	}
	v = v.Elem()
	t = t.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("dquely: SetUIDs expects a pointer to struct, got pointer to %s", v.Kind())
	}

	matched := make(map[string]bool, len(uids))

	// Match nested pointer and slice fields by their dquely tag predicate name.
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)
		ft := field.Type
		predicate, _, _ := parseTag(field.Tag.Get("dquely"), field.Name)
		if predicate == "uid" {
			continue
		}

		if ft.Kind() == reflect.Ptr && ft.Elem().Kind() == reflect.Struct {
			if uid, ok := uids[predicate]; ok {
				matched[predicate] = true
				if !fv.IsNil() {
					if err := SetUID(fv.Interface(), uid); err != nil {
						return err
					}
				}
			}
		} else if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Struct {
			if fv.IsNil() {
				continue
			}
			for j := 0; j < fv.Len(); j++ {
				key := predicate + strconv.Itoa(j)
				if uid, ok := uids[key]; ok {
					matched[key] = true
					elem := fv.Index(j)
					if elem.CanAddr() {
						if err := SetUID(elem.Addr().Interface(), uid); err != nil {
							return err
						}
					}
				}
			}
		}

	}

	// Any key not matched by a field is treated as the root struct's UID.
	for key, uid := range uids {
		if !matched[key] {
			if err := SetUID(input, uid); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

// SetUID writes uid into the field tagged `dquely:"uid"` on the struct that input
// points to. input must be a non-nil pointer to a struct.
func SetUID(input any, uid string) error {
	v := reflect.ValueOf(input)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("dquely: SetUID expects a pointer to struct, got %s", v.Kind())
	}
	v = v.Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		rawTag := t.Field(i).Tag.Get("dquely")
		predicate, _, _ := parseTag(rawTag, t.Field(i).Name)
		if predicate == "uid" {
			field := v.Field(i)
			if !field.CanSet() {
				return fmt.Errorf("dquely: uid field %q is not settable", t.Field(i).Name)
			}
			field.SetString(uid)
			return nil
		}
	}
	return fmt.Errorf("dquely: struct has no field tagged dquely:\"uid\"")
}

// structUID returns the value of the dquely:"uid" field in v, or "" if absent.
func structUID(v reflect.Value, t reflect.Type) string {
	for i := 0; i < t.NumField(); i++ {
		predicate, _, _ := parseTag(t.Field(i).Tag.Get("dquely"), t.Field(i).Name)
		if predicate == "uid" {
			return v.Field(i).String()
		}
	}
	return ""
}

// rootUID returns the dquely:"uid" value of the struct that input is or points to,
// or "" when it has no uid field or the field is empty. A non-empty result means
// the mutation is an update (addressing an existing node by <uid>) rather than an
// insert that creates a new node from a blank node.
func rootUID(input any) string {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	return structUID(v, v.Type())
}

// buildNquads writes raw N-quad lines for v into sb.
// For each node the output order is:
//  1. Primitive fields in struct declaration order, each ending with ".\n".
//  2. All nested-struct references (uid-based <uid> or blank-node _:x), each ending with ".\n".
//  3. dgraph.type triple (no trailing newline).
//  4. Recursive content for blank-node children, each preceded by "\n".
//
// When a nested struct has a non-empty uid its reference is "<uid>" and no content is emitted
// (the node already exists in DGraph).
func buildNquads(sb *strings.Builder, v reflect.Value, t reflect.Type, blankNode, typeName string, deep bool) error {
	// When this node already has a uid it must be addressed by <uid> so DGraph
	// updates the existing node instead of creating a new one from the blank node.
	subject := blankNode
	if uid := structUID(v, t); uid != "" {
		subject = fmt.Sprintf("<%s>", uid)
	}

	type nestedItem struct {
		ref         string // "<uid>" for existing nodes, "_:predicate" for new blank nodes
		blankNode   string // blank node name, only used when !skipContent
		v           reflect.Value
		t           reflect.Type
		typeName    string
		predicate   string
		skipContent bool // true when nested struct already has a uid
	}

	// Single pass in declaration order; nested struct fields are collected separately.
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		rawTag := field.Tag.Get("dquely")
		if rawTag == "-" {
			continue
		}
		predicate, isJSON, _ := parseTag(rawTag, field.Name)
		if predicate == "uid" {
			continue
		}
		ft := field.Type
		if (ft.Kind() == reflect.Ptr && ft.Elem().Kind() == reflect.Struct && hasUIDField(ft.Elem())) ||
			(ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Struct) ||
			(ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Ptr && ft.Elem().Elem().Kind() == reflect.Struct) {
			continue
		}
		fv := v.Field(i)
		if fv.IsZero() {
			continue
		}
		var valStr string
		if isJSON {
			b, err := json.Marshal(fv.Interface())
			if err != nil {
				return fmt.Errorf("dquely: failed to marshal field %s as JSON: %w", field.Name, err)
			}
			valStr = strings.ReplaceAll(string(b), `"`, `\"`)
		} else {
			dfv := fv
			if dfv.Kind() == reflect.Ptr {
				dfv = dfv.Elem()
			}
			if tv, ok := dfv.Interface().(time.Time); ok {
				valStr = tv.UTC().Format("2006-01-02T15:04:05")
			} else {
				valStr = escapeString(fmt.Sprintf("%v", dfv.Interface()))
			}
		}
		sb.WriteString(fmt.Sprintf("%s <%s> \"%s\" .\n", subject, predicate, valStr))
	}

	// Collect nested items in field declaration order.
	// Uid-based references are always emitted; blank-node references only when deep=true.
	var nestedItems []nestedItem
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		rawTag := field.Tag.Get("dquely")
		if rawTag == "-" {
			continue
		}
		predicate, _, _ := parseTag(rawTag, field.Name)
		if predicate == "uid" {
			continue
		}
		ft := field.Type
		fv := v.Field(i)

		if ft.Kind() == reflect.Ptr && ft.Elem().Kind() == reflect.Struct && hasUIDField(ft.Elem()) {
			if fv.IsNil() {
				continue
			}
			childV := fv.Elem()
			childT := ft.Elem()
			if uid := structUID(childV, childT); uid != "" {
				nestedItems = append(nestedItems, nestedItem{
					ref:         fmt.Sprintf("<%s>", uid),
					predicate:   predicate,
					skipContent: true,
				})
			} else if deep {
				bn := "_:" + predicate
				nestedItems = append(nestedItems, nestedItem{
					ref:       bn,
					blankNode: bn,
					v:         childV,
					t:         childT,
					typeName:  childT.Name(),
					predicate: predicate,
				})
			}
		} else if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Struct {
			if fv.IsNil() || fv.Len() == 0 {
				continue
			}
			childT := ft.Elem()
			for j := 0; j < fv.Len(); j++ {
				childV := fv.Index(j)
				if uid := structUID(childV, childT); uid != "" {
					nestedItems = append(nestedItems, nestedItem{
						ref:         fmt.Sprintf("<%s>", uid),
						predicate:   predicate,
						skipContent: true,
					})
				} else if deep {
					bn := fmt.Sprintf("_:%s%d", predicate, j)
					nestedItems = append(nestedItems, nestedItem{
						ref:       bn,
						blankNode: bn,
						v:         childV,
						t:         childT,
						typeName:  childT.Name(),
						predicate: predicate,
					})
				}
			}
		} else if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Ptr && ft.Elem().Elem().Kind() == reflect.Struct {
			if fv.IsNil() || fv.Len() == 0 {
				continue
			}
			childT := ft.Elem().Elem()
			for j := 0; j < fv.Len(); j++ {
				elemPtr := fv.Index(j)
				if elemPtr.IsNil() {
					continue
				}
				childV := elemPtr.Elem()
				if uid := structUID(childV, childT); uid != "" {
					nestedItems = append(nestedItems, nestedItem{
						ref:         fmt.Sprintf("<%s>", uid),
						predicate:   predicate,
						skipContent: true,
					})
				} else if deep {
					bn := fmt.Sprintf("_:%s%d", predicate, j)
					nestedItems = append(nestedItems, nestedItem{
						ref:       bn,
						blankNode: bn,
						v:         childV,
						t:         childT,
						typeName:  childT.Name(),
						predicate: predicate,
					})
				}
			}
		}
	}

	// All nested refs go before dgraph.type (uid-based and blank-node alike).
	for _, item := range nestedItems {
		sb.WriteString(fmt.Sprintf("%s <%s> %s .\n", subject, item.predicate, item.ref))
	}

	// dgraph.type is always the last triple for this node.
	sb.WriteString(fmt.Sprintf("%s <dgraph.type> \"%s\" .", subject, typeName))

	// Emit recursive content for blank-node children.
	for _, item := range nestedItems {
		if item.skipContent {
			continue
		}
		sb.WriteString("\n")
		if err := buildNquads(sb, item.v, item.t, item.blankNode, item.typeName, true); err != nil {
			return err
		}
	}

	return nil
}

// findUniquenessQuery recursively walks every nested struct field (both pointer-to-struct
// and slice-of-struct) and returns the first DGraph query block (variable "v") for a
// struct that has no uid but has at least one non-zero unique field.
// Returns an empty string when no such struct is found at any depth.
func findUniquenessQuery(v reflect.Value, t reflect.Type) string {
	for i := 0; i < t.NumField(); i++ {
		rawTag := t.Field(i).Tag.Get("dquely")
		if rawTag == "-" {
			continue
		}
		ft := t.Field(i).Type
		fv := v.Field(i)

		// Collect all child struct values for this field.
		type child struct {
			v reflect.Value
			t reflect.Type
		}
		var children []child
		if ft.Kind() == reflect.Ptr && ft.Elem().Kind() == reflect.Struct {
			if !fv.IsNil() {
				children = append(children, child{fv.Elem(), ft.Elem()})
			}
		} else if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Struct {
			childT := ft.Elem()
			for j := 0; j < fv.Len(); j++ {
				children = append(children, child{fv.Index(j), childT})
			}
		}

		for _, c := range children {
			if structUID(c.v, c.t) != "" {
				continue // existing node — no insert query needed
			}

			// Collect non-zero unique fields on this child.
			childTypeName := c.t.Name()
			var nonZeroUniques []struct {
				index     int
				predicate string
			}
			for j := 0; j < c.t.NumField(); j++ {
				rawChildTag := c.t.Field(j).Tag.Get("dquely")
				if rawChildTag == "-" {
					continue
				}
				pred, _, isUniq := parseTag(rawChildTag, c.t.Field(j).Name)
				if pred == "uid" || !isUniq || c.v.Field(j).IsZero() {
					continue
				}
				nonZeroUniques = append(nonZeroUniques, struct {
					index     int
					predicate string
				}{j, pred})
			}

			if len(nonZeroUniques) > 0 {
				var qb strings.Builder
				qb.WriteString("{\n")
				qb.WriteString(fmt.Sprintf("  v as var(func: type(%s))\n    @filter(", childTypeName))
				for k, f := range nonZeroUniques {
					if k > 0 {
						qb.WriteString(" OR ")
					}
					qb.WriteString(fmt.Sprintf("eq(%s, \"%s\")", f.predicate, escapeString(fmt.Sprintf("%v", c.v.Field(f.index).Interface()))))
				}
				qb.WriteString(")\n}\n")
				return qb.String()
			}

			// No unique fields on this child — recurse into its nested fields.
			if q := findUniquenessQuery(c.v, c.t); q != "" {
				return q
			}
		}
	}
	return ""
}

const FieldAll = "_all_"

// formatFieldValue returns the DQL literal representation of a struct field value.
// All values are wrapped in double quotes. JSON fields are json-encoded first.
// time.Time values are formatted as RFC3339 in UTC without timezone suffix.
func formatFieldValue(fv reflect.Value, isJSON bool) (string, error) {
	if isJSON {
		b, err := json.Marshal(fv.Interface())
		if err != nil {
			return "", err
		}
		return `"` + strings.ReplaceAll(string(b), `"`, `\"`) + `"`, nil
	}
	if fv.Kind() == reflect.Ptr {
		fv = fv.Elem()
	}
	if t, ok := fv.Interface().(time.Time); ok {
		return `"` + t.UTC().Format("2006-01-02T15:04:05") + `"`, nil
	}
	return `"` + escapeString(fmt.Sprintf("%v", fv.Interface())) + `"`, nil
}

package dquely

import (
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/dgo/v250/protos/api"
	"reflect"
	"strings"
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
				valueStr = fmt.Sprintf("%v", val.Interface())
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

// Upsert generates a DGraph upsert block that first queries for a node using matchExpr
// (any FilterExpr such as Eq, Gt, AllOfTerms, etc.), then updates only the specified
// updateFields using uid(var) as the subject.
// Update fields are matched by their `dquely` struct tag and emitted in the order given.
func Upsert(input any, matchExpr FilterExpr, updateFields ...string) (string, error) {
	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("dquely: Upsert expects a struct, got %s", v.Kind())
	}

	typeName := t.Name()
	if dm, ok := input.(DgraphMutation); ok {
		typeName = dm.DgraphType()
	}
	varName := strings.ToLower(typeName)

	// Build predicate → field index map using parseTag so options like ,unique are stripped.
	tagIndex := make(map[string]int, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		rawTag := t.Field(i).Tag.Get("dquely")
		if rawTag == "" || rawTag == "-" {
			continue
		}
		predicate, _, _ := parseTag(rawTag, t.Field(i).Name)
		tagIndex[predicate] = i
	}

	var sb strings.Builder
	sb.WriteString("upsert {\n  query {\n")
	sb.WriteString(fmt.Sprintf("    %s as var(func: %s)\n", varName, matchExpr.expr))
	sb.WriteString("  }\n  mutation {\n    set {\n")

	for _, field := range updateFields {
		idx, ok := tagIndex[field]
		if !ok {
			continue
		}
		val := v.Field(idx)
		if val.IsZero() {
			continue
		}
		sb.WriteString(fmt.Sprintf("      uid(%s) <%s> \"%v\" .\n", varName, field, val.Interface()))
	}

	sb.WriteString("    }\n  }\n}")
	return sb.String(), nil
}

// MutationTriple represents a single RDF triple in a DGraph mutation block.
type MutationTriple struct {
	isDelete  bool
	varRef    string
	predicate string
	value     string // pre-rendered: `"value"`, `val(a)`, `*`
}

// TripleSet creates a set triple with a literal value: uid(varRef) <predicate> "value" .
func TripleSet(varRef, predicate string, value any) MutationTriple {
	return MutationTriple{varRef: varRef, predicate: predicate, value: formatValue(value)}
}

// TripleSetVal creates a set triple with a val() reference: uid(varRef) <predicate> val(valVar) .
func TripleSetVal(varRef, predicate, valVar string) MutationTriple {
	return MutationTriple{varRef: varRef, predicate: predicate, value: fmt.Sprintf("val(%s)", valVar)}
}

// TripleDelete creates a delete triple with a wildcard: uid(varRef) <predicate> * .
func TripleDelete(varRef, predicate string) MutationTriple {
	return MutationTriple{isDelete: true, varRef: varRef, predicate: predicate, value: "*"}
}

// UpsertBlock builds a complete upsert block for any combination of set and delete triples.
// If q has a BlockVar set, the query renders as "blockVar as var(func: ...) { selects }".
// Otherwise it renders as "queryName(func: ...) { selects }".
func UpsertBlock(queryName string, q *DQuely, triples ...MutationTriple) string {
	funcExpr := ""
	for _, f := range q.filters {
		if f.isFuncPart {
			funcExpr = f.expr
			break
		}
	}

	var sb strings.Builder
	sb.WriteString("upsert {\n  query {\n")

	if q.blockVarName != "" {
		sb.WriteString(fmt.Sprintf("    %s as var(func: %s)", q.blockVarName, funcExpr))
		if len(q.selects) > 0 {
			sb.WriteString(" {\n")
			q.renderFields(&sb, "      ")
			sb.WriteString("    }\n")
		} else {
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(fmt.Sprintf("    %s(func: %s) {\n", queryName, funcExpr))
		q.renderFields(&sb, "      ")
		sb.WriteString("    }\n")
	}

	sb.WriteString("  }\n\n  mutation {\n")

	var setTriples, deleteTriples []MutationTriple
	for _, t := range triples {
		if t.isDelete {
			deleteTriples = append(deleteTriples, t)
		} else {
			setTriples = append(setTriples, t)
		}
	}

	if len(setTriples) > 0 {
		sb.WriteString("    set {\n")
		for _, t := range setTriples {
			sb.WriteString(fmt.Sprintf("      uid(%s) <%s> %s .\n", t.varRef, t.predicate, t.value))
		}
		sb.WriteString("    }\n")
	}

	if len(deleteTriples) > 0 {
		sb.WriteString("    delete {\n")
		for _, t := range deleteTriples {
			sb.WriteString(fmt.Sprintf("      uid(%s) <%s> %s .\n", t.varRef, t.predicate, t.value))
		}
		sb.WriteString("    }\n")
	}

	sb.WriteString("  }\n}")
	return sb.String()
}

// UpsertDelete generates a DGraph upsert block that queries nodes using matchExpr
// and deletes the specified predicates with a wildcard (*) value.
// varName is the variable assigned in the query block and referenced as uid(varName)
// in the delete mutation.
func UpsertDelete(varName string, matchExpr FilterExpr, fields ...string) (string, error) {
	var sb strings.Builder
	sb.WriteString("upsert {\n  query {\n")
	sb.WriteString(fmt.Sprintf("    %s as var(func: %s)\n", varName, matchExpr.expr))
	sb.WriteString("  }\n\n  mutation {\n    delete {\n")
	for _, field := range fields {
		sb.WriteString(fmt.Sprintf("      uid(%s) <%s> * .\n", varName, field))
	}
	sb.WriteString("    }\n  }\n}")
	return sb.String(), nil
}

// UpsertWithQuery generates a DGraph upsert block with a full named query block
// (as opposed to a bare var block). The query block is rendered using q and named
// queryName. varRef is the variable assigned inside the query body (e.g. "v as uid")
// and referenced as uid(varRef) in the mutation set.
// Update fields are matched by their `dquely` struct tag and emitted in the order given.
func UpsertWithQuery(queryName string, q *DQuely, varRef string, input any, updateFields ...string) (string, error) {
	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("dquely: UpsertWithQuery expects a struct, got %s", v.Kind())
	}

	// Build predicate → field index map using parseTag so options like ,unique are stripped.
	tagIndex := make(map[string]int, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		rawTag := t.Field(i).Tag.Get("dquely")
		if rawTag == "" || rawTag == "-" {
			continue
		}
		predicate, _, _ := parseTag(rawTag, t.Field(i).Name)
		tagIndex[predicate] = i
	}

	funcExpr := ""
	for _, f := range q.filters {
		if f.isFuncPart {
			funcExpr = f.expr
			break
		}
	}
	argsStr := ""
	if funcExpr != "" {
		argsStr = "func: " + funcExpr
	}

	var sb strings.Builder
	sb.WriteString("upsert {\n  query {\n")
	sb.WriteString(fmt.Sprintf("    %s(%s) {\n", queryName, argsStr))
	q.renderFields(&sb, "      ")
	sb.WriteString("    }\n")
	sb.WriteString("  }\n\n  mutation {\n    set {\n")

	for _, field := range updateFields {
		idx, ok := tagIndex[field]
		if !ok {
			continue
		}
		val := v.Field(idx)
		if val.IsZero() {
			continue
		}
		sb.WriteString(fmt.Sprintf("      uid(%s) <%s> \"%v\" .\n", varRef, field, val.Interface()))
	}

	sb.WriteString("    }\n  }\n}")
	return sb.String(), nil
}

// buildNquads writes raw N-quad lines for v into sb.
// Primitive fields are written first (strings/json pass, then others), then dgraph.type.
// In deep mode, nested pointer-to-struct and slice-of-struct fields are collected and
// emitted as blank-node references, followed by their recursive content.
func buildNquads(sb *strings.Builder, v reflect.Value, t reflect.Type, blankNode, typeName string, deep bool) error {
	type nestedItem struct {
		blankNode string
		v         reflect.Value
		t         reflect.Type
		typeName  string
		predicate string
	}
	var nestedItems []nestedItem

	// Two passes: strings/json first, then non-string primitives (nested structs skipped).
	for _, stringPass := range []bool{true, false} {
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
			if (ft.Kind() == reflect.Ptr && ft.Elem().Kind() == reflect.Struct) ||
				(ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Struct) {
				continue
			}
			isString := ft.Kind() == reflect.String || isJSON
			if isString != stringPass {
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
				valStr = fmt.Sprintf("%v", fv.Interface())
			}
			sb.WriteString(fmt.Sprintf("%s <%s> \"%s\" .\n", blankNode, predicate, valStr))
		}
	}

	sb.WriteString(fmt.Sprintf("%s <dgraph.type> \"%s\" .", blankNode, typeName))

	if !deep {
		return nil
	}

	// Collect nested items in field declaration order.
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

		if ft.Kind() == reflect.Ptr && ft.Elem().Kind() == reflect.Struct {
			if fv.IsNil() {
				continue
			}
			childT := ft.Elem()
			nestedItems = append(nestedItems, nestedItem{
				blankNode: "_:" + predicate,
				v:         fv.Elem(),
				t:         childT,
				typeName:  childT.Name(),
				predicate: predicate,
			})
		} else if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Struct {
			if fv.IsNil() || fv.Len() == 0 {
				continue
			}
			childT := ft.Elem()
			for j := 0; j < fv.Len(); j++ {
				nestedItems = append(nestedItems, nestedItem{
					blankNode: fmt.Sprintf("_:%s%d", predicate, j),
					v:         fv.Index(j),
					t:         childT,
					typeName:  childT.Name(),
					predicate: predicate,
				})
			}
		}
	}

	if len(nestedItems) == 0 {
		return nil
	}

	// Emit all blank-node references first.
	for _, item := range nestedItems {
		sb.WriteString(fmt.Sprintf("\n%s <%s> %s .", blankNode, item.predicate, item.blankNode))
	}

	// Emit nested content in the same order.
	for _, item := range nestedItems {
		sb.WriteString("\n")
		if err := buildNquads(sb, item.v, item.t, item.blankNode, item.typeName, true); err != nil {
			return err
		}
	}

	return nil
}

// ParseMutation inspects input (a non-nil pointer to a struct with a dquely:"uid" field)
// and produces a DGraph upsert-ready query string and a slice of api.Mutation objects.
//
// The behaviour depends on whether the struct carries fields tagged with the "unique" option
// and whether the uid field is populated:
//
//   - No unique fields (Case A — simple insert):
//     Returns an empty query string and a single mutation whose SetNquads contains the
//     full "{ set { … } }" block produced by Mutation(). No condition is set.
//
//   - Has unique fields, uid is empty (Case B — conditional insert):
//     Builds a query that assigns variable "v" to all nodes of the same dgraph.type whose
//     unique predicates match any of the non-zero unique field values (OR filter).
//     The mutation SetNquads contains raw N-quads using a blank node (e.g. _:user),
//     and the condition @if(eq(len(v), 0)) prevents the insert when a duplicate exists.
//
//   - Has unique fields, uid is non-empty (Case C — conditional update):
//     Builds a two-variable query:
//
//   - "u" selects the node with the given uid, filtered by dgraph.type (must be 1).
//
//   - "v" selects any other node of the same type whose unique predicates overlap with
//     the provided values, excluding the current uid (must be 0 for no duplicates).
//     SetNquads updates non-zero fields using the concrete uid reference (e.g. <0x1>).
//     DelNquads deletes zero-value fields: non-unique predicates first, unique ones after.
//     The condition @if(eq(len(v), 0) AND eq(len(u), 1)) ensures both invariants hold.
func ParseMutation(input any, deep ...bool) (string, []*api.Mutation, error) {
	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)
	if v.Kind() != reflect.Ptr {
		return "", nil, fmt.Errorf("dquely: ParseMutation expects a pointer to struct, got %s", v.Kind())
	}
	v = v.Elem()
	t = t.Elem()
	if v.Kind() != reflect.Struct {
		return "", nil, fmt.Errorf("dquely: ParseMutation expects a pointer to struct, got pointer to %s", v.Kind())
	}
	if !hasUIDField(t) {
		return "", nil, fmt.Errorf("dquely: ParseMutation requires a field tagged dquely:\"uid\" in the struct")
	}

	typeName := t.Name()
	if dm, ok := input.(DgraphMutation); ok {
		typeName = dm.DgraphType()
	}
	blankNode := "_:" + strings.ToLower(typeName)

	// Get uid value.
	uid := ""
	for i := 0; i < t.NumField(); i++ {
		predicate, _, _ := parseTag(t.Field(i).Tag.Get("dquely"), t.Field(i).Name)
		if predicate == "uid" {
			uid = v.Field(i).String()
			break
		}
	}

	// Collect metadata for all non-uid fields.
	type fieldMeta struct {
		index     int
		predicate string
		isJSON    bool
		isUnique  bool
	}
	var allFields []fieldMeta
	var uniqueFields []fieldMeta
	for i := 0; i < t.NumField(); i++ {
		rawTag := t.Field(i).Tag.Get("dquely")
		if rawTag == "-" {
			continue
		}
		predicate, isJSON, isUnique := parseTag(rawTag, t.Field(i).Name)
		if predicate == "uid" {
			continue
		}
		fm := fieldMeta{index: i, predicate: predicate, isJSON: isJSON, isUnique: isUnique}
		allFields = append(allFields, fm)
		if isUnique {
			uniqueFields = append(uniqueFields, fm)
		}
	}

	isDeep := len(deep) > 0 && deep[0]

	// Case A: No unique fields.
	if len(uniqueFields) == 0 {
		// Detect if any field is a nested struct (pointer or slice).
		hasNested := false
		for i := 0; i < t.NumField(); i++ {
			rawTag := t.Field(i).Tag.Get("dquely")
			if rawTag == "-" {
				continue
			}
			ft := t.Field(i).Type
			if (ft.Kind() == reflect.Ptr && ft.Elem().Kind() == reflect.Struct) ||
				(ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Struct) {
				hasNested = true
				break
			}
		}

		if !hasNested {
			// No nested structs — delegate to Mutation() for backwards-compatible output.
			mutStr, err := Mutation(input)
			if err != nil {
				return "", nil, err
			}
			return "", []*api.Mutation{{SetNquads: []byte(mutStr)}}, nil
		}

		// Has nested structs — generate raw N-quads with optional deep rendering.
		var sb strings.Builder
		if err := buildNquads(&sb, v, t, blankNode, typeName, isDeep); err != nil {
			return "", nil, err
		}
		return "", []*api.Mutation{{SetNquads: []byte(sb.String())}}, nil
	}

	// Non-zero unique fields used for query filter.
	var nonZeroUniques []fieldMeta
	for _, f := range uniqueFields {
		if !v.Field(f.index).IsZero() {
			nonZeroUniques = append(nonZeroUniques, f)
		}
	}

	// valueStr returns the string representation of a field value.
	valueStr := func(fm fieldMeta) (string, error) {
		fv := v.Field(fm.index)
		if fm.isJSON {
			b, err := json.Marshal(fv.Interface())
			if err != nil {
				return "", fmt.Errorf("dquely: failed to marshal field %s as JSON: %w",
					t.Field(fm.index).Name, err)
			}
			return strings.ReplaceAll(string(b), `"`, `\"`), nil
		}
		return fmt.Sprintf("%v", fv.Interface()), nil
	}

	// Case B: Insert (uid == "").
	if uid == "" {
		var qb strings.Builder
		qb.WriteString("{\n")
		qb.WriteString(fmt.Sprintf("  v as var(func: type(%s))\n    @filter(", typeName))
		for i, f := range nonZeroUniques {
			if i > 0 {
				qb.WriteString(" OR ")
			}
			val := fmt.Sprintf("%v", v.Field(f.index).Interface())
			qb.WriteString(fmt.Sprintf("eq(%s, \"%s\")", f.predicate, val))
		}
		qb.WriteString(")\n}")

		// Build raw N-quads: two passes (strings/json first, then others).
		var sb strings.Builder
		for _, stringPass := range []bool{true, false} {
			for _, fm := range allFields {
				isString := t.Field(fm.index).Type.Kind() == reflect.String || fm.isJSON
				if isString != stringPass {
					continue
				}
				fv := v.Field(fm.index)
				if fv.IsZero() {
					continue
				}
				val, err := valueStr(fm)
				if err != nil {
					return "", nil, err
				}
				sb.WriteString(fmt.Sprintf("%s <%s> \"%s\" .\n", blankNode, fm.predicate, val))
			}
		}
		sb.WriteString(fmt.Sprintf("%s <dgraph.type> \"%s\" .", blankNode, typeName))

		mu := &api.Mutation{
			SetNquads: []byte(sb.String()),
			Cond:      "@if(eq(len(v), 0))",
		}
		return qb.String(), []*api.Mutation{mu}, nil
	}

	// Case C: Update (uid != "").
	uidRef := fmt.Sprintf("<%s>", uid)

	// Build query with two-variable block.
	var qb strings.Builder
	qb.WriteString("{\n")
	qb.WriteString(fmt.Sprintf("  u as var(func: uid(%s)) @filter(type(%s))\n\n", uid, typeName))
	qb.WriteString(fmt.Sprintf("  v as var(func: type(%s))\n", typeName))
	if len(nonZeroUniques) >= 2 {
		// Tab-indented @filter for multiple unique conditions.
		qb.WriteString("\t@filter(\n\t  (")
		for i, f := range nonZeroUniques {
			if i > 0 {
				qb.WriteString(" OR ")
			}
			val := fmt.Sprintf("%v", v.Field(f.index).Interface())
			qb.WriteString(fmt.Sprintf("eq(%s, \"%s\")", f.predicate, val))
		}
		qb.WriteString(")\n")
		qb.WriteString(fmt.Sprintf("\t  AND NOT uid(%s)\n\t)\n}", uid))
	} else if len(nonZeroUniques) == 1 {
		// 4-space-indented @filter for a single unique condition.
		f := nonZeroUniques[0]
		val := fmt.Sprintf("%v", v.Field(f.index).Interface())
		qb.WriteString(fmt.Sprintf("    @filter(\n      eq(%s, \"%s\") AND NOT uid(%s)\n    )\n}", f.predicate, val, uid))
	} else {
		qb.WriteString("}")
	}

	// Build SetNquads: declaration order, non-uid non-zero fields, no dgraph.type.
	var setSB strings.Builder
	firstSet := true
	for _, fm := range allFields {
		fv := v.Field(fm.index)
		if fv.IsZero() {
			continue
		}
		val, err := valueStr(fm)
		if err != nil {
			return "", nil, err
		}
		if !firstSet {
			setSB.WriteByte('\n')
		}
		setSB.WriteString(fmt.Sprintf("%s <%s> \"%s\" .", uidRef, fm.predicate, val))
		firstSet = false
	}

	// Build DelNquads: non-unique zero fields first, then unique zero fields.
	var delSB strings.Builder
	firstDel := true
	for _, fm := range allFields {
		if fm.isUnique || !v.Field(fm.index).IsZero() {
			continue
		}
		if !firstDel {
			delSB.WriteByte('\n')
		}
		delSB.WriteString(fmt.Sprintf("%s <%s> * .", uidRef, fm.predicate))
		firstDel = false
	}
	for _, fm := range uniqueFields {
		if !v.Field(fm.index).IsZero() {
			continue
		}
		if !firstDel {
			delSB.WriteByte('\n')
		}
		delSB.WriteString(fmt.Sprintf("%s <%s> * .", uidRef, fm.predicate))
		firstDel = false
	}

	mu := &api.Mutation{
		SetNquads: []byte(setSB.String()),
		DelNquads: []byte(delSB.String()),
		Cond:      "@if(eq(len(v), 0) AND eq(len(u), 1))",
	}
	return qb.String(), []*api.Mutation{mu}, nil
}

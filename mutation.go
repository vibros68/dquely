package dquely

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Mutation serializes a struct to a DGraph RDF N-Quad set mutation.
// Fields are mapped using the `dquely` struct tag as the predicate name.
// The blank node is the lowercased struct type name (e.g. User → _:user).
// String fields are emitted first (in declaration order), then numeric/other fields,
// and dgraph.type is always appended last.
func Mutation(input any) (string, error) {
	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("dquely: Mutation expects a struct, got %s", v.Kind())
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
			predicate, isJSON := parseTag(rawTag, field.Name)
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
func parseTag(rawTag, fieldName string) (predicate string, isJSON bool) {
	predicate = rawTag
	if idx := strings.Index(rawTag, ","); idx >= 0 {
		predicate = rawTag[:idx]
		for _, opt := range strings.Split(rawTag[idx+1:], ",") {
			if opt == "json" {
				isJSON = true
			}
		}
	}
	if predicate == "" {
		predicate = fieldName
	}
	return
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

	// Build tag → field index map.
	tagIndex := make(map[string]int, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("dquely")
		if tag != "" && tag != "-" {
			tagIndex[tag] = i
		}
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

	tagIndex := make(map[string]int, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("dquely")
		if tag != "" && tag != "-" {
			tagIndex[tag] = i
		}
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

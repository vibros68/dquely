package dquely

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/dgraph-io/dgo/v250/protos/api"
)

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

	// Structs with nested pointer-to-struct or slice-of-struct fields always use the
	// buildNquads path regardless of unique fields.  Uniqueness for such structs must be
	// enforced by the caller (the nested graph is mutated atomically).
	hasNested := false
	for i := 0; i < t.NumField(); i++ {
		rawTag := t.Field(i).Tag.Get("dquely")
		if rawTag == "-" {
			continue
		}
		ft := t.Field(i).Type
		if (ft.Kind() == reflect.Ptr && ft.Elem().Kind() == reflect.Struct) ||
			(ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Struct) ||
			(ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Ptr && ft.Elem().Elem().Kind() == reflect.Struct) {
			hasNested = true
			break
		}
	}
	if hasNested {
		var sb strings.Builder
		if err := buildNquads(&sb, v, t, blankNode, typeName, isDeep); err != nil {
			return "", nil, err
		}
		nquads := []byte(sb.String())

		// No unique fields on root: recursively search nested fields for unique fields.
		if len(uniqueFields) == 0 {
			if isDeep {
				if q := findUniquenessQuery(v, t); q != "" {
					mu := &api.Mutation{
						SetNquads: nquads,
						Cond:      "@if(eq(len(v), 0))",
					}
					return q, []*api.Mutation{mu}, nil
				}
			}
			return "", []*api.Mutation{{SetNquads: nquads}}, nil
		}

		// Has unique fields and uid is empty (nested Case B): generate query+condition
		// for duplicate prevention while using the deep N-quads as the mutation body.
		if uid == "" {
			var nonZeroUniques []fieldMeta
			for _, f := range uniqueFields {
				if !v.Field(f.index).IsZero() {
					nonZeroUniques = append(nonZeroUniques, f)
				}
			}
			var qb strings.Builder
			qb.WriteString("{\n")
			qb.WriteString(fmt.Sprintf("  v as var(func: type(%s))\n    @filter(", typeName))
			for i, f := range nonZeroUniques {
				if i > 0 {
					qb.WriteString(" OR ")
				}
				val := escapeString(fmt.Sprintf("%v", v.Field(f.index).Interface()))
				qb.WriteString(fmt.Sprintf("eq(%s, \"%s\")", f.predicate, val))
			}
			if isDeep {
				qb.WriteString(")\n}\n")
			} else {
				qb.WriteString(")\n}")
			}
			mu := &api.Mutation{
				SetNquads: nquads,
				Cond:      "@if(eq(len(v), 0))",
			}
			return qb.String(), []*api.Mutation{mu}, nil
		}

		// uid is non-empty: uid-based update, no duplicate query needed.
		return "", []*api.Mutation{{SetNquads: nquads}}, nil
	}

	// Case A: No unique fields, no nested structs — raw N-quads, no condition.
	if len(uniqueFields) == 0 {
		var sb strings.Builder
		if err := buildNquads(&sb, v, t, blankNode, typeName, false); err != nil {
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
		return escapeString(fmt.Sprintf("%v", fv.Interface())), nil
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
			val := escapeString(fmt.Sprintf("%v", v.Field(f.index).Interface()))
			qb.WriteString(fmt.Sprintf("eq(%s, \"%s\")", f.predicate, val))
		}
		if isDeep {
			qb.WriteString(")\n}\n")
		} else {
			qb.WriteString(")\n}")
		}

		var sb strings.Builder
		if err := buildNquads(&sb, v, t, blankNode, typeName, false); err != nil {
			return "", nil, err
		}
		nquads := []byte(sb.String())

		mu := &api.Mutation{
			SetNquads: nquads,
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
			val := escapeString(fmt.Sprintf("%v", v.Field(f.index).Interface()))
			qb.WriteString(fmt.Sprintf("eq(%s, \"%s\")", f.predicate, val))
		}
		qb.WriteString(")\n")
		qb.WriteString(fmt.Sprintf("\t  AND NOT uid(%s)\n\t)\n}", uid))
	} else if len(nonZeroUniques) == 1 {
		// 4-space-indented @filter for a single unique condition.
		f := nonZeroUniques[0]
		val := escapeString(fmt.Sprintf("%v", v.Field(f.index).Interface()))
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

// ParseUpdate generates a DGraph conditional-mutation query and api.Mutation for
// updating an existing node identified by its uid field. The struct must have a
// non-empty field tagged dquely:"uid".
//
// Pass FieldAll ("_all_") to include all non-uid, non-zero fields in the update.
// Pass one or more predicate names to limit the update to those specific fields.
//
// The returned query string is wrapped in a "{}" block, suitable for use directly
// in an api.Request. The mutation carries:
//   - SetNquads: scalar values as typed literals; relationship fields as <uid> references.
//   - DelNquads: a wildcard delete for every relationship field that was requested, so
//     stale edges are cleared before the new references are written.
//   - Cond: "@if(eq(len(v), 1))" — only fires when exactly one node matches the uid
func ParseUpdate(input any, fields ...string) (string, []*api.Mutation, error) {
	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)
	if v.Kind() != reflect.Ptr {
		return "", nil, fmt.Errorf("dquely: ParseUpdate expects a pointer to struct, got %s", v.Kind())
	}
	v = v.Elem()
	t = t.Elem()
	if v.Kind() != reflect.Struct {
		return "", nil, fmt.Errorf("dquely: ParseUpdate expects a pointer to struct, got pointer to %s", v.Kind())
	}
	if !hasUIDField(t) {
		return "", nil, fmt.Errorf("dquely: ParseUpdate requires a field tagged dquely:\"uid\" in the struct")
	}

	typeName := t.Name()
	if dm, ok := input.(DgraphMutation); ok {
		typeName = dm.DgraphType()
	}
	uid := ""
	for i := 0; i < t.NumField(); i++ {
		predicate, _, _ := parseTag(t.Field(i).Tag.Get("dquely"), t.Field(i).Name)
		if predicate == "uid" {
			uid = v.Field(i).String()
			break
		}
	}
	if uid == "" {
		return "", nil, fmt.Errorf("dquely: ParseUpdate requires a non-empty uid field")
	}

	var fieldSet map[string]bool
	if !(len(fields) == 1 && fields[0] == FieldAll) {
		fieldSet = make(map[string]bool, len(fields))
		for _, f := range fields {
			fieldSet[f] = true
		}
	}

	// Single pass in struct declaration order.
	// Primitive fields go to setSB; relationship fields produce uid-references in setSB
	// and a wildcard delete in delSB.
	var setSB, delSB strings.Builder
	firstSet, firstDel := true, true

	appendSet := func(s string) {
		if !firstSet {
			setSB.WriteByte('\n')
		}
		setSB.WriteString(s)
		firstSet = false
	}
	appendDel := func(s string) {
		if !firstDel {
			delSB.WriteByte('\n')
		}
		delSB.WriteString(s)
		firstDel = false
	}

	// blankChild holds a []Struct element that has no uid and needs inline N-quad content.
	type blankChild struct {
		bn string
		v  reflect.Value
		t  reflect.Type
	}
	var blankChildren []blankChild

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
		if fieldSet != nil && !fieldSet[predicate] {
			continue
		}
		ft := field.Type
		fv := v.Field(i)

		if ft.Kind() == reflect.Ptr && ft.Elem().Kind() == reflect.Struct && hasUIDField(ft.Elem()) {
			if !fv.IsNil() {
				if childUID := structUID(fv.Elem(), ft.Elem()); childUID != "" {
					appendSet(fmt.Sprintf("uid(v) <%s> <%s> .", predicate, childUID))
				}
			}
			appendDel(fmt.Sprintf("uid(v) <%s> * .", predicate))
		} else if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Struct {
			childT := ft.Elem()
			for j := 0; j < fv.Len(); j++ {
				childV := fv.Index(j)
				if childUID := structUID(childV, childT); childUID != "" {
					appendSet(fmt.Sprintf("uid(v) <%s> <%s> .", predicate, childUID))
				} else {
					// Name blank nodes by predicate + index (e.g. _:staffs0), matching
					// the convention used by buildNquads so N-quads read consistently.
					bn := fmt.Sprintf("_:%s%d", predicate, j)
					appendSet(fmt.Sprintf("uid(v) <%s> %s .", predicate, bn))
					blankChildren = append(blankChildren, blankChild{bn, childV, childT})
				}
			}
			appendDel(fmt.Sprintf("uid(v) <%s> * .", predicate))
		} else if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Ptr && ft.Elem().Elem().Kind() == reflect.Struct {
			childT := ft.Elem().Elem()
			for j := 0; j < fv.Len(); j++ {
				elemPtr := fv.Index(j)
				if elemPtr.IsNil() {
					continue
				}
				if childUID := structUID(elemPtr.Elem(), childT); childUID != "" {
					appendSet(fmt.Sprintf("uid(v) <%s> <%s> .", predicate, childUID))
				}
			}
		} else {
			val, err := formatFieldValue(fv, isJSON)
			if err != nil {
				return "", nil, fmt.Errorf("dquely: field %s: %w", field.Name, err)
			}
			appendSet(fmt.Sprintf("uid(v) <%s> %s .", predicate, val))
		}
	}

	// Emit inline N-quad content for []Struct blank-node children (no uid).
	for _, bc := range blankChildren {
		for k := 0; k < bc.t.NumField(); k++ {
			cf := bc.t.Field(k)
			cRawTag := cf.Tag.Get("dquely")
			if cRawTag == "-" {
				continue
			}
			cPredicate, cIsJSON, _ := parseTag(cRawTag, cf.Name)
			if cPredicate == "uid" {
				continue
			}
			cft := cf.Type
			cfv := bc.v.Field(k)
			if cft.Kind() == reflect.Ptr && cft.Elem().Kind() == reflect.Struct {
				if !cfv.IsNil() {
					if nestedUID := structUID(cfv.Elem(), cft.Elem()); nestedUID != "" {
						appendSet(fmt.Sprintf("%s <%s> <%s> .", bc.bn, cPredicate, nestedUID))
					}
				}
			} else if cft.Kind() == reflect.Slice {
				// skip slices in child content for now
			} else {
				if cfv.IsZero() {
					continue
				}
				val, err := formatFieldValue(cfv, cIsJSON)
				if err != nil {
					return "", nil, fmt.Errorf("dquely: field %s: %w", cf.Name, err)
				}
				appendSet(fmt.Sprintf("%s <%s> %s .", bc.bn, cPredicate, val))
			}
		}
	}

	query := fmt.Sprintf("{\n  v as var(func: uid(%s))\n    @filter(type(%s))\n}", uid, typeName)
	mu := &api.Mutation{
		SetNquads: []byte(setSB.String()),
		DelNquads: []byte(delSB.String()),
		Cond:      "@if(eq(len(v), 1))",
	}
	return query, []*api.Mutation{mu}, nil
}

package dquely

import (
	"fmt"
	"strings"
)

type DgFilter interface {
	Query() string
	DgraphKey() string
}

// FilterExpr is a standalone filter expression for use with Or().
type FilterExpr struct {
	expr string
}

// Eq creates a standalone eq filter expression for use with Or().
func Eq(key string, value any) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("eq(%s, %s)", key, formatValue(value))}
}

func Gt(key any, value any) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("gt(%s, %s)", renderKey(key), renderValue(value))}
}

func Ge(key any, value any) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("ge(%s, %s)", renderKey(key), renderValue(value))}
}

func Le(key any, value any) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("le(%s, %s)", renderKey(key), renderValue(value))}
}

func Lt(key any, value any) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("lt(%s, %s)", renderKey(key), renderValue(value))}
}

// Val creates a val(varName) expression for use as a key or value in comparisons.
func Val(varName string) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("val(%s)", varName)}
}

// Count creates a count(predicate) expression for use as a key in comparisons.
func Count(predicate string) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("count(%s)", predicate)}
}

func Ngram(key string, value string) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("ngram(%s, %s)", key, formatValue(value))}
}

func AllOfText(key, value string) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("alloftext(%s, %s)", key, formatValue(value))}
}

func AnyOfText(key, value string) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("anyoftext(%s, %s)", key, formatValue(value))}
}

func AllOfTerms(key, value string) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("allofterms(%s, %s)", key, formatValue(value))}
}

func AnyOfTerms(key, value string) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("anyofterms(%s, %s)", key, formatValue(value))}
}

func Has(field string) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("has(%s)", field)}
}

func Uid(values ...any) FilterExpr {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%v", v)
	}
	return FilterExpr{expr: fmt.Sprintf("uid(%s)", strings.Join(parts, ", "))}
}

func UidIn(predicate string, values ...any) FilterExpr {
	if len(values) == 1 {
		switch v := values[0].(type) {
		case FilterExpr:
			// uid_in(predicate, uid(var) ) — trailing space before closing paren
			return FilterExpr{expr: fmt.Sprintf("uid_in(%s, %s )", predicate, v.expr)}
		default:
			return FilterExpr{expr: fmt.Sprintf("uid_in(%s, %v)", predicate, v)}
		}
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%v", v)
	}
	return FilterExpr{expr: fmt.Sprintf("uid_in(%s, [%s])", predicate, strings.Join(parts, ","))}
}

func Between(field string, from, to any) FilterExpr {
	return FilterExpr{expr: fmt.Sprintf("between(%s, %s, %s)", field, formatValue(from), formatValue(to))}
}

func Not(expr FilterExpr) FilterExpr {
	return FilterExpr{expr: "NOT " + expr.expr}
}

func Regexp(field, pattern string, flags ...string) FilterExpr {
	flag := ""
	if len(flags) > 0 {
		flag = flags[0]
	}
	return FilterExpr{expr: fmt.Sprintf("regexp(%s, /%s/%s)", field, pattern, flag)}
}

// ExpandAll is the DGraph predicate that expands all predicates of a node.
const ExpandAll = "expand(_all_)"

// ExpandAllBlock creates an expand(_all_) { fields... } inline select element.
func ExpandAllBlock(fields ...string) *DQuely {
	args := make([]any, len(fields))
	for i, f := range fields {
		args[i] = f
	}
	return NewDQL("").Select(args...).As(ExpandAll).Inline()
}

// renderKey renders the left operand of a comparison: predicate name or FilterExpr (val/count).
func renderKey(v any) string {
	switch e := v.(type) {
	case FilterExpr:
		return e.expr
	default:
		return fmt.Sprintf("%v", e)
	}
}

// renderValue renders the right operand: FilterExpr unquoted, strings quoted, numbers as-is.
func renderValue(v any) string {
	switch e := v.(type) {
	case FilterExpr:
		return e.expr
	default:
		return formatValue(e)
	}
}

func formatValue(v any) string {
	switch v.(type) {
	case string:
		return fmt.Sprintf(`"%v"`, v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

type filter struct {
	isFuncPart bool
	expr       string   // pre-rendered expression for simple filters
	orExprs    []string // non-empty for OR groups
}

type DQuely struct {
	dgKey        string   // block name used by Query(); also returned by DgraphKey()
	name         string   // set when used as a nested select element via As()
	isVar        bool     // renders as "var(func: ...)" block in Build
	condVar      string   // condition statement: "condVar as name(func: ...)" with no body
	blockVarName string   // variable prefix on the block itself: "blockVarName as var(func: ...) { ... }"
	varName      string   // variable assignment on nested select: "varName as name { ... }"
	queryArgs    []string // ordering/extra args for root and nested: "orderdesc: ...", "orderasc: ..."
	firstN       *int     // first: N — combined with queryArgs+offsetN into one field "(args)" group
	offsetN      *int     // offset: N — combined with queryArgs+firstN into one field "(args)" group
	inline       bool     // render nested select on one line: "name { field1 field2 }"
	cascade      bool     // adds @cascade directive before @filter / {
	groupBy      string   // adds @groupby(field) directive
	selects      []any
	filters      []filter
}

func NewDQL(dgKey string) *DQuely {
	return &DQuely{dgKey: dgKey}
}

// DgraphKey returns the block name used by Query() and for JSON response parsing.
func (d *DQuely) DgraphKey() string {
	return d.dgKey
}

// NewVar creates a DGraph var block: var(func: ...) { ... }.
// Variable blocks assign values to variables for use in subsequent query blocks.
func NewVar() *DQuely {
	return &DQuely{isVar: true}
}

// NewCondition creates a condition statement: "condVar as blockName(func: ...)".
// Condition statements have no body and are used to bind a variable to a set of UIDs.
func NewCondition(condVar, blockName string) *DQuely {
	return &DQuely{condVar: condVar, name: blockName}
}

func (d *DQuely) getInstance() *DQuely {
	clone := *d
	return &clone
}

func (d *DQuely) As(name string) *DQuely {
	clone := d.getInstance()
	clone.name = name
	return clone
}

func (d *DQuely) Select(elem ...any) *DQuely {
	clone := d.getInstance()
	clone.selects = append(clone.selects, elem...)
	return clone
}

// Assign sets a variable name for this node: "varName as name { ... }".
func (d *DQuely) Assign(varName string) *DQuely {
	clone := d.getInstance()
	clone.varName = varName
	return clone
}

// OrderDir specifies the ordering direction.
type OrderDir string

const (
	ASC  OrderDir = "asc"
	DESC OrderDir = "desc"
)

// Order adds an ordering directive. On root blocks it is included in the main args;
// on nested selects it is combined with First/Offset into one (order<dir>: expr, first: N, ...) group.
func (d *DQuely) Order(expr string, dir OrderDir) *DQuely {
	clone := d.getInstance()
	clone.queryArgs = append(clone.queryArgs, fmt.Sprintf("order%s: %s", dir, expr))
	return clone
}

// First adds a first: N pagination directive. Combined with Offset into one (first: N, offset: M) group
// on nested selects; appended to root args on root blocks.
func (d *DQuely) First(n int) *DQuely {
	clone := d.getInstance()
	clone.firstN = &n
	return clone
}

// Offset adds an offset: N pagination directive, combined with First into one (first: N, offset: M) group.
func (d *DQuely) Offset(n int) *DQuely {
	clone := d.getInstance()
	clone.offsetN = &n
	return clone
}

// BlockVar sets a variable prefix on the block itself: "varName as var(func: ...) { ... }".
func (d *DQuely) BlockVar(varName string) *DQuely {
	clone := d.getInstance()
	clone.blockVarName = varName
	return clone
}

// Inline marks this nested select to render on a single line: "name { field1 field2 }".
func (d *DQuely) Inline() *DQuely {
	clone := d.getInstance()
	clone.inline = true
	return clone
}

// GroupBy adds the @groupby(field) directive to this block or nested select.
func (d *DQuely) GroupBy(field string) *DQuely {
	clone := d.getInstance()
	clone.groupBy = field
	return clone
}

// Cascade adds the @cascade directive to this block or nested select.
// With @cascade, nodes that don't have all predicates specified in the query are removed.
// This can be useful in cases where some filter was applied or if nodes might not have all listed predicates.
func (d *DQuely) Cascade() *DQuely {
	clone := d.getInstance()
	clone.cascade = true
	return clone
}

// Uid sets func: uid(...) as the root function.
func (d *DQuely) Uid(values ...any) *DQuely {
	return d.Func(Uid(values...))
}

// Func sets any FilterExpr as the root function.
func (d *DQuely) Func(expr FilterExpr) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{isFuncPart: true, expr: expr.expr})
	return clone
}

// AllOfTerms sets func: allofterms(key, value) as the root function.
func (d *DQuely) AllOfTerms(key, value string) *DQuely {
	return d.Func(AllOfTerms(key, value))
}

// AnyOfTerms sets func: anyofterms(key, value) as the root function.
func (d *DQuely) AnyOfTerms(key, value string) *DQuely {
	return d.Func(AnyOfTerms(key, value))
}

// Between sets func: between(field, from, to) as the root function.
func (d *DQuely) Between(field string, from, to any) *DQuely {
	return d.Func(Between(field, from, to))
}

// Regexp sets func: regexp(field, /pattern/flags) as the root function.
func (d *DQuely) Regexp(field, pattern string, flags ...string) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{isFuncPart: true, expr: Regexp(field, pattern, flags...).expr})
	return clone
}

// Filter adds one or more FilterExpr to the @filter of this node (usable on root or nested selects).
func (d *DQuely) Filter(exprs ...FilterExpr) *DQuely {
	clone := d.getInstance()
	for _, e := range exprs {
		clone.filters = append(clone.filters, filter{expr: e.expr})
	}
	return clone
}

// Has sets func: has(field) as the root function.
func (d *DQuely) Has(field string) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{isFuncPart: true, expr: fmt.Sprintf("has(%s)", field)})
	return clone
}

// Type sets func: type(name) as the root function.
func (d *DQuely) Type(typeName any) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{isFuncPart: true, expr: fmt.Sprintf("type(%v)", typeName)})
	return clone
}

func (d *DQuely) Eq(key string, value any) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{expr: fmt.Sprintf("eq(%s, %s)", key, formatValue(value))})
	return clone
}

func (d *DQuely) Gt(key any, value any) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{expr: fmt.Sprintf("gt(%s, %s)", renderKey(key), renderValue(value))})
	return clone
}

func (d *DQuely) Ge(key any, value any) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{expr: fmt.Sprintf("ge(%s, %s)", renderKey(key), renderValue(value))})
	return clone
}

func (d *DQuely) Le(key any, value any) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{expr: fmt.Sprintf("le(%s, %s)", renderKey(key), renderValue(value))})
	return clone
}

func (d *DQuely) Lt(key any, value any) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{expr: fmt.Sprintf("lt(%s, %s)", renderKey(key), renderValue(value))})
	return clone
}

func (d *DQuely) Ngram(key string, value string) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{expr: fmt.Sprintf("ngram(%s, %s)", key, formatValue(value))})
	return clone
}

func (d *DQuely) AllOfText(key, value string) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{expr: fmt.Sprintf("alloftext(%s, %s)", key, formatValue(value))})
	return clone
}

func (d *DQuely) AnyOfText(key, value string) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{expr: fmt.Sprintf("anyoftext(%s, %s)", key, formatValue(value))})
	return clone
}

func (d *DQuely) Not(expr FilterExpr) *DQuely {
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{expr: "NOT " + expr.expr})
	return clone
}

// Or adds an AND-grouped OR filter: AND ( expr1 OR expr2 ... ).
func (d *DQuely) Or(exprs ...FilterExpr) *DQuely {
	orExprs := make([]string, len(exprs))
	for i, e := range exprs {
		orExprs[i] = e.expr
	}
	clone := d.getInstance()
	clone.filters = append(clone.filters, filter{orExprs: orExprs})
	return clone
}

func (d *DQuely) inlineFilter() string {
	var exprs []string
	for _, f := range d.filters {
		if !f.isFuncPart && len(f.orExprs) == 0 {
			exprs = append(exprs, f.expr)
		}
	}
	if len(exprs) == 0 {
		return ""
	}
	return " @filter(" + strings.Join(exprs, " AND ") + ")"
}

func (d *DQuely) renderFields(sb *strings.Builder, indent string) {
	for _, s := range d.selects {
		switch v := s.(type) {
		case string:
			sb.WriteString(indent + v + "\n")
		case *DQuely:
			prefix := ""
			if v.varName != "" {
				prefix = v.varName + " as "
			}
			fieldArgs := append([]string{}, v.queryArgs...)
			if v.firstN != nil {
				fieldArgs = append(fieldArgs, fmt.Sprintf("first: %d", *v.firstN))
			}
			if v.offsetN != nil {
				fieldArgs = append(fieldArgs, fmt.Sprintf("offset: %d", *v.offsetN))
			}
			fieldArgsStr := ""
			if len(fieldArgs) > 0 {
				fieldArgsStr = "(" + strings.Join(fieldArgs, ", ") + ")"
			}
			cascadeStr := ""
			if v.cascade {
				cascadeStr = " @cascade"
			}
			groupByStr := ""
			if v.groupBy != "" {
				groupByStr = fmt.Sprintf(" @groupby(%s)", v.groupBy)
			}
			if v.inline {
				var parts []string
				for _, s := range v.selects {
					if str, ok := s.(string); ok {
						parts = append(parts, str)
					}
				}
				sb.WriteString(indent + prefix + v.name + fieldArgsStr + cascadeStr + groupByStr + v.inlineFilter() + " { " + strings.Join(parts, " ") + " }\n")
			} else {
				sb.WriteString(indent + prefix + v.name + fieldArgsStr + cascadeStr + groupByStr + v.inlineFilter() + " {\n")
				v.renderFields(sb, indent+"  ")
				sb.WriteString(indent + "}\n")
			}
		}
	}
}

func (d *DQuely) renderCondition(sb *strings.Builder, blockName string) {
	funcExpr := ""
	for _, f := range d.filters {
		if f.isFuncPart {
			funcExpr = f.expr
		}
	}
	funcPart := ""
	if funcExpr != "" {
		funcPart = "func: " + funcExpr
	}
	sb.WriteString(fmt.Sprintf("  %s as %s(%s)\n", d.condVar, blockName, funcPart))
}

func (d *DQuely) renderBlock(sb *strings.Builder, blockName string) {
	// Separate func: filter from @filter expressions
	funcExpr := ""
	var atFilters []filter
	for _, f := range d.filters {
		if f.isFuncPart {
			funcExpr = f.expr
		} else {
			atFilters = append(atFilters, f)
		}
	}

	argsStr := ""
	if funcExpr != "" {
		argsStr = "func: " + funcExpr
	}
	if len(d.queryArgs) > 0 {
		if argsStr != "" {
			argsStr += ", "
		}
		argsStr += strings.Join(d.queryArgs, ", ")
	}
	if d.firstN != nil || d.offsetN != nil {
		var parts []string
		if d.firstN != nil {
			parts = append(parts, fmt.Sprintf("first: %d", *d.firstN))
		}
		if d.offsetN != nil {
			parts = append(parts, fmt.Sprintf("offset: %d", *d.offsetN))
		}
		if argsStr != "" {
			argsStr += ", "
		}
		argsStr += strings.Join(parts, ", ")
	}

	blockPrefix := ""
	if d.blockVarName != "" {
		blockPrefix = d.blockVarName + " as "
	}

	cascadeStr := ""
	if d.cascade {
		cascadeStr = " @cascade"
	}
	groupByStr := ""
	if d.groupBy != "" {
		groupByStr = fmt.Sprintf(" @groupby(%s)", d.groupBy)
	}

	switch {
	case len(atFilters) == 0:
		// No @filter
		sb.WriteString(fmt.Sprintf("  %s%s(%s)%s%s {\n", blockPrefix, blockName, argsStr, cascadeStr, groupByStr))

	case len(atFilters) == 1 && len(atFilters[0].orExprs) == 0:
		// Single simple filter → inline
		sb.WriteString(fmt.Sprintf("  %s%s(%s)%s%s @filter(%s) {\n", blockPrefix, blockName, argsStr, cascadeStr, groupByStr, atFilters[0].expr))

	default:
		// Multiple filters → multiline
		sb.WriteString(fmt.Sprintf("  %s%s(%s)%s%s\n", blockPrefix, blockName, argsStr, cascadeStr, groupByStr))
		sb.WriteString("  @filter(\n")
		for i, f := range atFilters {
			prefix := ""
			if i > 0 {
				prefix = "AND "
			}
			if len(f.orExprs) == 0 {
				sb.WriteString(fmt.Sprintf("    %s%s\n", prefix, f.expr))
			} else {
				sb.WriteString(fmt.Sprintf("    %s(\n", prefix))
				for j, oe := range f.orExprs {
					orPrefix := ""
					if j > 0 {
						orPrefix = "OR "
					}
					sb.WriteString(fmt.Sprintf("      %s%s\n", orPrefix, oe))
				}
				sb.WriteString("    )\n")
			}
		}
		sb.WriteString("  ) {\n")
	}

	d.renderFields(sb, "    ")
	sb.WriteString("  }\n")
}

// Query builds a single-query DQL string using dgKey as the block name.
func (d *DQuely) Query() string {
	var sb strings.Builder
	sb.WriteString("{\n")
	d.renderBlock(&sb, d.dgKey)
	sb.WriteString("}")
	return sb.String()
}

// Build combines multiple named queries into a single DQL string.
// Each query must have its block name set via As().
func Build(queries ...*DQuely) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	for i, q := range queries {
		if i > 0 {
			sb.WriteString("\n")
		}
		blockName := q.name
		if q.isVar {
			blockName = "var"
		}
		if q.condVar != "" {
			q.renderCondition(&sb, blockName)
		} else {
			q.renderBlock(&sb, blockName)
		}
	}
	sb.WriteString("}")
	return sb.String()
}

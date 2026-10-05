package vars

import "slices"

// Occurrence is a template found inside a value by Scan.
type Occurrence struct {
	// Path is the place of the string inside the scanned value: the fields
	// and indexes leading to it. It is empty if the value itself is the
	// string.
	Path []Segment
	// Src is the string as written.
	Src string
	// Template is the parsed template; nil if the string is malformed.
	Template *Template
	// Err is the syntax error of a malformed template; nil otherwise.
	Err *ParseError
}

// At returns the place of the occurrence as text: ".owner.tags[1]", or an
// empty string if the scanned value itself is the template.
func (o Occurrence) At() string {
	at := ""
	for _, seg := range o.Path {
		at += seg.String()
	}
	return at
}

// Scan finds the templates inside a value without evaluating them. It walks
// objects and arrays like Resolve does and returns every string that has
// expressions or is malformed; plain text is skipped. Occurrences come in a
// stable order: object fields sorted by key, array elements by index.
//
// Scan is the basis of static analysis: checking templates before a run and
// finding out which variables a value needs.
func Scan(value any) []Occurrence {
	var found []Occurrence
	scan(value, nil, &found)
	return found
}

func scan(value any, path []Segment, found *[]Occurrence) {
	switch v := value.(type) {
	case string:
		t, err := Parse(v)
		switch {
		case err != nil:
			// Parse returns only *ParseError.
			*found = append(*found, Occurrence{Path: slices.Clone(path), Src: v, Err: err.(*ParseError)})
		case !t.IsStatic():
			*found = append(*found, Occurrence{Path: slices.Clone(path), Src: v, Template: t})
		}
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			scan(v[key], append(path, Segment{Field: key}), found)
		}
	case []any:
		for i, item := range v {
			scan(item, append(path, Segment{Index: i, IsIndex: true}), found)
		}
	}
}

// VarNames returns the names of the variables a value refers to, sorted and
// without duplicates. Only the root name counts: "{{product.tags[0]}}" refers
// to the variable "product". Malformed templates are skipped.
func VarNames(value any) []string {
	return names(value, ExprVar)
}

// EnvNames returns the names of the env variables a value refers to, sorted
// and without duplicates. Malformed templates are skipped.
func EnvNames(value any) []string {
	return names(value, ExprEnv)
}

func names(value any, kind ExprKind) []string {
	var out []string
	for _, occ := range Scan(value) {
		if occ.Template == nil {
			continue
		}
		for _, e := range occ.Template.Exprs() {
			if e.Kind == kind && !slices.Contains(out, e.Name) {
				out = append(out, e.Name)
			}
		}
	}
	slices.Sort(out)
	return out
}

// IsReserved reports whether name cannot be used as a variable name because
// it starts a namespace of the template language: {{env.NAME}},
// {{random.email}}. A variable with such a name could never be read.
func IsReserved(name string) bool {
	return name == nsEnv || name == nsRandom
}

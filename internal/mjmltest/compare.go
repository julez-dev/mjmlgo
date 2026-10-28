package mjmltest

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// MaxMismatches is the maximum number of mismatches reported per document.
const MaxMismatches = 200

// Mismatch is a single structural difference between two normalized documents.
type Mismatch struct {
	// Path is an XPath-like location such as /html/body/table[1]/tr[1].
	Path string
	// Detail describes the difference.
	Detail string
}

func (m Mismatch) String() string {
	return m.Path + ": " + m.Detail
}

// CompareHTML parses both documents, compares their normalized trees and returns
// the differences found. The returned slice is empty when the documents are
// equivalent for mail rendering purposes.
func CompareHTML(want, got string) ([]Mismatch, error) {
	wantNode, err := ParseHTML(want)
	if err != nil {
		return nil, fmt.Errorf("parse reference HTML: %w", err)
	}

	gotNode, err := ParseHTML(got)
	if err != nil {
		return nil, fmt.Errorf("parse generated HTML: %w", err)
	}

	return Compare(wantNode, gotNode), nil
}

// Compare compares two normalized trees. want is the reference document, got the
// document under test.
func Compare(want, got *Node) []Mismatch {
	var mismatches []Mismatch

	compareNodes(&mismatches, want, got, "", 0)

	return mismatches
}

// compareNodes compares two nodes. index is the 1-based position of the node
// among its same-name siblings, used to build readable paths.
func compareNodes(mismatches *[]Mismatch, want, got *Node, path string, index int) {
	if len(*mismatches) >= MaxMismatches {
		return
	}

	if want == nil || got == nil {
		*mismatches = append(*mismatches, Mismatch{
			Path:   pathOrRoot(path),
			Detail: fmt.Sprintf("missing node (want %s, got %s)", describe(want), describe(got)),
		})
		return
	}

	path = childPath(path, want, got, index)

	if want.Kind != got.Kind {
		*mismatches = append(*mismatches, Mismatch{
			Path:   path,
			Detail: fmt.Sprintf("node kind mismatch: want %s, got %s", kindString(want), kindString(got)),
		})
		return
	}

	switch want.Kind {
	case KindDocument:
		// nothing to compare besides children
	case KindDoctype:
		if want.Text != got.Text {
			*mismatches = append(*mismatches, Mismatch{
				Path:   path,
				Detail: fmt.Sprintf("doctype mismatch: want %q, got %q", want.Text, got.Text),
			})
		}
		return
	case KindText, KindComment:
		if want.Text != got.Text {
			*mismatches = append(*mismatches, Mismatch{
				Path:   path,
				Detail: fmt.Sprintf("text mismatch: want %q, got %q", want.Text, got.Text),
			})
		}
		return
	case KindElement, KindConditional:
		if want.Name != got.Name {
			*mismatches = append(*mismatches, Mismatch{
				Path:   path,
				Detail: fmt.Sprintf("name mismatch: want %q, got %q", want.Name, got.Name),
			})
			return
		}

		compareAttrs(mismatches, want, got, path)

		if want.Sheet != got.Sheet {
			*mismatches = append(*mismatches, stylesheetMismatches(want.Sheet, got.Sheet, path)...)
		}
	}

	if len(want.Children) != len(got.Children) {
		*mismatches = append(*mismatches, Mismatch{
			Path:   path,
			Detail: fmt.Sprintf("child count mismatch: want %d, got %d", len(want.Children), len(got.Children)),
		})
	}

	limit := min(len(want.Children), len(got.Children))

	for i := range limit {
		index := sameNameIndex(want.Children, i)
		compareNodes(mismatches, want.Children[i], got.Children[i], path, index)
	}
}

func compareAttrs(mismatches *[]Mismatch, want, got *Node, path string) {
	wantAttrs := make(map[string]string, len(want.Attrs))
	for _, attr := range want.Attrs {
		wantAttrs[attr.Key] = attr.Val
	}

	gotAttrs := make(map[string]string, len(got.Attrs))
	for _, attr := range got.Attrs {
		gotAttrs[attr.Key] = attr.Val
	}

	keys := make([]string, 0, len(wantAttrs)+len(gotAttrs))
	for key := range wantAttrs {
		keys = append(keys, key)
	}

	for key := range gotAttrs {
		if _, ok := wantAttrs[key]; !ok {
			keys = append(keys, key)
		}
	}

	// Deterministic ordering so failure output is stable.
	sort.Strings(keys)

	for _, key := range keys {
		if len(*mismatches) >= MaxMismatches {
			return
		}

		wantVal, wantOK := wantAttrs[key]
		gotVal, gotOK := gotAttrs[key]

		switch {
		case !gotOK:
			*mismatches = append(*mismatches, Mismatch{
				Path:   path,
				Detail: fmt.Sprintf("attribute %q missing, want %q", key, wantVal),
			})
		case !wantOK:
			*mismatches = append(*mismatches, Mismatch{
				Path:   path,
				Detail: fmt.Sprintf("attribute %q unexpected, got %q", key, gotVal),
			})
		case wantVal != gotVal:
			*mismatches = append(*mismatches, Mismatch{
				Path:   path,
				Detail: fmt.Sprintf("attribute %q mismatch: want %q, got %q", key, wantVal, gotVal),
			})
		}
	}
}

// sameNameIndex returns the 1-based position of siblings[pos] among the siblings
// that share its name, so paths do not shift when unrelated siblings are added.
func sameNameIndex(siblings []*Node, pos int) int {
	index := 1

	for i := range pos {
		if siblings[i].Kind == siblings[pos].Kind && siblings[i].Name == siblings[pos].Name {
			index++
		}
	}

	return index
}

// childPath extends path with the node name and its 1-based position among
// siblings with the same name.
func childPath(path string, want, got *Node, index int) string {
	name := want.Name
	if name == "" {
		name = got.Name
	}

	switch want.Kind {
	case KindDocument:
		return "/#document"
	case KindElement:
		name = strings.ToLower(name)
	case KindConditional:
		name = "#conditional"
	case KindDoctype:
		name = "#doctype"
	case KindText:
		name = "#text"
	case KindComment:
		name = "#comment"
	}

	if index > 0 {
		return path + "/" + name + "[" + strconv.Itoa(index) + "]"
	}

	return path + "/" + name
}

func pathOrRoot(path string) string {
	if path == "" {
		return "/"
	}

	return path
}

func describe(n *Node) string {
	if n == nil {
		return "<nil>"
	}

	return kindString(n)
}

func kindString(n *Node) string {
	switch n.Kind {
	case KindDocument:
		return "#document"
	case KindDoctype:
		return "<!doctype " + n.Text + ">"
	case KindText:
		return strconv.Quote(n.Text)
	case KindComment:
		return "<!--" + n.Text + "-->"
	case KindConditional:
		return "#conditional [if " + n.Name + "]"
	case KindElement:
		return "<" + n.Name + ">"
	default:
		return "<unknown>"
	}
}

// stylesheetMismatches compares two canonical stylesheets rule by rule so the
// report points at the first rule that differs instead of dumping whole sheets.
func stylesheetMismatches(want, got, path string) []Mismatch {
	wantRules := splitStylesheetRules(want)
	gotRules := splitStylesheetRules(got)

	var mismatches []Mismatch

	limit := min(len(wantRules), len(gotRules))

	for i := range limit {
		if wantRules[i] == gotRules[i] {
			continue
		}

		mismatches = append(mismatches, Mismatch{
			Path:   path,
			Detail: fmt.Sprintf("stylesheet rule %d mismatch: want %q, got %q", i+1, wantRules[i], gotRules[i]),
		})

		if len(mismatches) >= MaxMismatches {
			return mismatches
		}
	}

	switch {
	case len(wantRules) > len(gotRules):
		mismatches = append(mismatches, Mismatch{
			Path:   path,
			Detail: fmt.Sprintf("stylesheet rule %d missing: want %q", len(gotRules)+1, wantRules[len(gotRules)]),
		})
	case len(gotRules) > len(wantRules):
		mismatches = append(mismatches, Mismatch{
			Path:   path,
			Detail: fmt.Sprintf("stylesheet rule %d unexpected: got %q", len(wantRules)+1, gotRules[len(wantRules)]),
		})
	}

	return mismatches
}

// splitStylesheetRules splits a canonical stylesheet into top-level rules.
func splitStylesheetRules(sheet string) []string {
	if sheet == "" {
		return nil
	}

	var (
		rules []string
		depth int
		start int
	)

	for i, char := range sheet {
		switch char {
		case '{':
			depth++
		case '}':
			depth--

			if depth == 0 {
				rules = append(rules, sheet[start:i+1])
				start = i + 1
			}
		}
	}

	if start < len(sheet) {
		if trailing := strings.TrimSpace(sheet[start:]); trailing != "" {
			rules = append(rules, trailing)
		}
	}

	return rules
}

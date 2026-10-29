package mjmlgo

import "strings"

// specificity is one selector's specificity vector in the form the CSS cascade
// compares it: the inline origin, then id, class/attribute and
// type/pseudo-class counts. A rule selector has 0 in the first slot; the
// element's own style attribute counts as {1, 0, 0, 0}, which is what makes an
// authored inline style outrank every rule that is not important.
type specificity [4]int

// splitTopLevel splits text on the separators that appear outside strings,
// brackets and parentheses. It is the one scanner the cascade needs: a selector
// list splits on the commas between selectors, a complex selector on the
// combinators between its compound selectors.
func splitTopLevel(text string, isSeparator func(byte) bool) []string {
	var (
		parts []string
		start int
		depth int
		quote byte
	)

	for i := 0; i < len(text); i++ {
		switch c := text[i]; {
		case quote != 0:
			switch c {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			if depth > 0 {
				depth--
			}
		case depth == 0 && isSeparator(c):
			parts = append(parts, strings.TrimSpace(text[start:i]))
			start = i + 1
		}
	}

	return append(parts, strings.TrimSpace(text[start:]))
}

// splitSelectorList splits a rule's selector text into its selectors, on the
// commas that separate them. Commas inside an attribute selector, a functional
// pseudo-class or a string do not separate selectors.
func splitSelectorList(text string) []string {
	return splitTopLevel(text, func(c byte) bool { return c == ',' })
}

// selectorSpecificity counts the specificity of one complex selector - a
// selector without top-level commas - the way CSS and the reference inliner do:
// one point per id, per class, per attribute selector and per type or
// pseudo-class, with :not() contributing the specificity of its argument
// instead of its own. The universal selector and the combinators add nothing.
//
// The count decides which declaration wins the cascade, so it only has to be as
// exact as the selector engine is: a selector this function cannot classify
// exactly - an unknown functional pseudo-class, say - is counted as one
// pseudo-class, which is the CSS rule for pseudo-classes it does not know.
func selectorSpecificity(selector string) specificity {
	var spec specificity

	for _, compound := range splitCompounds(selector) {
		spec = addCompoundSpecificity(spec, compound)
	}

	return spec
}

// splitCompounds splits a complex selector into its compound selectors on the
// combinators, ignoring anything inside brackets, parentheses or strings.
func splitCompounds(selector string) []string {
	parts := splitTopLevel(selector, func(c byte) bool {
		switch c {
		case ' ', '\t', '\n', '\r', '>', '+', '~':
			return true
		default:
			return false
		}
	})

	compounds := make([]string, 0, len(parts))

	for _, compound := range parts {
		if compound != "" {
			compounds = append(compounds, compound)
		}
	}

	return compounds
}

// addCompoundSpecificity adds the specificity of one compound selector to spec.
func addCompoundSpecificity(spec specificity, compound string) specificity {
	i := 0

	// ":root", "*" and a leading type or namespace-qualified type selector.
	if i < len(compound) && compound[i] == ':' {
		// No type selector before the pseudo-class.
	} else {
		start := i
		for i < len(compound) && isSelectorNameByte(compound[i]) {
			i++
		}

		if i < len(compound) && compound[i] == '|' {
			// Namespace prefix: the qualified name is what counts.
			i++
			start = i

			for i < len(compound) && isSelectorNameByte(compound[i]) {
				i++
			}
		}

		if name := compound[start:i]; name != "" && name != "*" {
			spec[3]++
		}
	}

	for i < len(compound) {
		switch compound[i] {
		case '#':
			spec[1]++
			i = skipSelectorName(compound, i+1)
		case '.':
			spec[2]++
			i = skipSelectorName(compound, i+1)
		case '[':
			spec[2]++
			i = skipBracketed(compound, i)
		case ':':
			i++
			if i < len(compound) && compound[i] == ':' {
				// A pseudo-element counts like a pseudo-class here.
				i++
			}

			start := i
			i = skipSelectorName(compound, i)

			name := strings.ToLower(compound[start:i])

			if i < len(compound) && compound[i] == '(' {
				end := skipBracketed(compound, i)
				argument := compound[i+1 : max(i+1, end-1)]

				if name == "not" {
					// :not() contributes the specificity of its most specific
					// argument instead of its own.
					var not specificity

					for _, alternative := range splitSelectorList(argument) {
						not = maxSpecificity(not, selectorSpecificity(alternative))
					}

					for entry := range spec {
						spec[entry] += not[entry]
					}
				} else {
					spec[3]++
				}

				i = end

				continue
			}

			spec[3]++
		default:
			i++
		}
	}

	return spec
}

// maxSpecificity returns the higher of two specificity vectors.
func maxSpecificity(a, b specificity) specificity {
	for i := range a {
		if a[i] != b[i] {
			if a[i] > b[i] {
				return a
			}

			return b
		}
	}

	return a
}

// skipSelectorName returns the index just past an identifier.
func skipSelectorName(s string, i int) int {
	if i < len(s) && s[i] == '\\' {
		i++
	}

	for i < len(s) && isSelectorNameByte(s[i]) {
		i++
	}

	return i
}

// skipBracketed returns the index just past the bracketed or parenthesised part
// of s that starts at i, honouring nesting and strings. An unterminated bracket
// ends at the end of s.
func skipBracketed(s string, i int) int {
	var (
		depth int
		quote byte
	)

	for ; i < len(s); i++ {
		switch c := s[i]; {
		case quote != 0:
			switch c {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}

	return len(s)
}

// isSelectorNameByte reports whether c can appear in a selector identifier. It
// is deliberately lenient: the exact character set only matters for counting.
func isSelectorNameByte(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '_', c == '\\':
		return true
	default:
		return false
	}
}

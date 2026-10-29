package component

import "strings"

// ruleBlockAtRules mirrors douceur's atRulesWithRulesBlock: an at-rule whose
// block holds nested rules rather than declarations.
var ruleBlockAtRules = map[string]bool{
	"@document":            true,
	"@font-feature-values": true,
	"@keyframes":           true,
	"@media":               true,
	"@supports":            true,
}

// sanitizeInlineCSS rewrites an inline stylesheet into the subset the stylesheet
// parser can consume without looping, recovering from malformed CSS the way the
// reference inliner's parser (postcss) does:
//
//   - a rule-level statement that never reaches a block (a ";" before any "{")
//     is dropped, and so is a block with an empty prelude;
//   - a stray "}" at top level is dropped;
//   - an unterminated block is closed at the end of the input;
//   - an unterminated comment or string drops everything from its start on.
//
// Well-formed CSS is returned byte for byte: a rule-level segment that carries no
// syntax of its own (whitespace, comments, the legacy CDO/CDC markers) is always
// copied through, so a document that parses today parses to the same stylesheet.
func sanitizeInlineCSS(input string) string {
	var out strings.Builder
	out.Grow(len(input))

	// openBlocks records, for each open block, whether it holds rules (true) or
	// declarations (false). No open block means top level, which is rule level.
	var openBlocks []bool

	for i := 0; i < len(input); {
		if len(openBlocks) > 0 && !openBlocks[len(openBlocks)-1] {
			end, closed, usable := scanDeclarations(input, i)
			out.WriteString(input[i:end])

			if !usable || !closed {
				closeOpenBlocks(&out, len(openBlocks))

				return out.String()
			}

			out.WriteByte('}')
			openBlocks = openBlocks[:len(openBlocks)-1]

			i = end + 1

			continue
		}

		terminator, at, usable := scanRuleLevel(input, i)
		segment := input[i:at]

		if !usable {
			out.WriteString(segment)
			closeOpenBlocks(&out, len(openBlocks))

			return out.String()
		}

		switch terminator {
		case '{':
			if isIgnorableRuleBytes(segment) {
				// `{ ... }` with no selector: skip the block whole.
				i = skipBlock(input, at)

				continue
			}

			out.WriteString(segment)
			out.WriteByte('{')
			openBlocks = append(openBlocks, embedsRules(segment))

			i = at + 1
		case ';':
			if isAtStatement(segment) {
				out.WriteString(segment)
				out.WriteByte(';')
			}

			i = at + 1
		case '}':
			// Whitespace and comments before the brace are layout, not junk; a
			// rule-level statement that never reached a block is junk.
			if isIgnorableRuleBytes(segment) {
				out.WriteString(segment)
			}

			// At top level a stray closing brace is dropped; inside an at-rule
			// block it closes the block the caller opened.
			if len(openBlocks) > 0 {
				out.WriteByte('}')
				openBlocks = openBlocks[:len(openBlocks)-1]
			}

			i = at + 1
		default:
			// End of input on a rule-level statement: an at-rule such as
			// "@import url(...)" and ignorable bytes are kept, anything else had
			// no block and is junk.
			if isAtStatement(segment) || isIgnorableRuleBytes(segment) {
				out.WriteString(segment)
			}

			i = len(input)
		}
	}

	// An unterminated rule list is closed at the end, like a declaration block.
	closeOpenBlocks(&out, len(openBlocks))

	return out.String()
}

// closeOpenBlocks terminates every block still open at the end of the input.
func closeOpenBlocks(out *strings.Builder, n int) {
	for range n {
		out.WriteByte('}')
	}
}

// scanRuleLevel finds the end of a rule-level segment. It returns the byte that
// terminated it ('{', ';', '}') or 0 at the end of the input, plus the index the
// segment ends at. usable is false when an unterminated comment or string stops
// the scan; at is then the index where that construct starts.
func scanRuleLevel(s string, start int) (terminator byte, at int, usable bool) {
	// openParens keeps a ';' inside url(...) or any other function from ending the
	// statement: "@import url(a;b);" must survive whole.
	openParens := 0

	for i := start; i < len(s); {
		c := s[i]

		switch {
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			next, ok := skipComment(s, i)
			if !ok {
				return 0, i, false
			}

			i = next
		case c == '"' || c == '\'':
			next, ok := skipString(s, i)
			if !ok {
				return 0, i, false
			}

			i = next
		case c == '(':
			openParens++

			i++
		case c == ')':
			if openParens > 0 {
				openParens--
			}

			i++
		case openParens == 0 && (c == '{' || c == ';' || c == '}'):
			return c, i, true
		default:
			i++
		}
	}

	return 0, len(s), true
}

// scanDeclarations finds the closing brace of a declaration block, honouring
// nested braces that appear in values. closed is false at the end of the input.
func scanDeclarations(s string, start int) (end int, closed, usable bool) {
	depth := 0

	for i := start; i < len(s); {
		c := s[i]

		switch {
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			next, ok := skipComment(s, i)
			if !ok {
				return i, false, false
			}

			i = next
		case c == '"' || c == '\'':
			next, ok := skipString(s, i)
			if !ok {
				return i, false, false
			}

			i = next
		case c == '{':
			depth++

			i++
		case c == '}':
			if depth == 0 {
				return i, true, true
			}

			depth--

			i++
		default:
			i++
		}
	}

	return len(s), false, true
}

// skipBlock returns the index just past the block that opens at start.
func skipBlock(s string, start int) int {
	depth := 0

	for i := start; i < len(s); {
		c := s[i]

		switch {
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			next, ok := skipComment(s, i)
			if !ok {
				return len(s)
			}

			i = next
		case c == '"' || c == '\'':
			next, ok := skipString(s, i)
			if !ok {
				return len(s)
			}

			i = next
		case c == '{':
			depth++

			i++
		case c == '}':
			depth--

			i++

			if depth == 0 {
				return i
			}
		default:
			i++
		}
	}

	return len(s)
}

// skipComment returns the index just past the comment that starts at start.
func skipComment(s string, start int) (int, bool) {
	if end := strings.Index(s[start+2:], "*/"); end >= 0 {
		return start + 2 + end + 2, true
	}

	return len(s), false
}

// skipString returns the index just past the quoted string that starts at start.
// A backslash escapes the next byte.
func skipString(s string, start int) (int, bool) {
	quote := s[start]

	for i := start + 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case quote:
			return i + 1, true
		}
	}

	return len(s), false
}

// stripComments removes comments from a prelude, for the checks that only look
// at its at-rule or its emptiness.
func stripComments(s string) string {
	var out strings.Builder

	for i := 0; i < len(s); {
		if s[i] == '/' && i+1 < len(s) && s[i+1] == '*' {
			next, ok := skipComment(s, i)
			if !ok {
				break
			}

			i = next

			continue
		}

		out.WriteByte(s[i])
		i++
	}

	return out.String()
}

// isIgnorableRuleBytes reports whether a rule-level segment carries no syntax of
// its own: whitespace, comments and the legacy CDO/CDC markers at most.
func isIgnorableRuleBytes(segment string) bool {
	rest := stripComments(segment)
	rest = strings.ReplaceAll(rest, "<!--", "")
	rest = strings.ReplaceAll(rest, "-->", "")

	return strings.TrimSpace(rest) == ""
}

// isAtStatement reports whether a rule-level statement is an at-rule, which may
// legitimately end with a semicolon ("@import", "@charset"). The check mirrors
// the stylesheet scanner, where only '@' followed by an identifier is an
// at-keyword: a bare or space-separated '@' ("@ x;") is ordinary bytes, and
// keeping it here would hand the parser the rule-level ';' it loops on.
func isAtStatement(segment string) bool {
	return isAtKeywordPrefix(strings.TrimSpace(stripComments(segment)))
}

// isAtKeywordPrefix reports whether s starts with a CSS at-keyword.
func isAtKeywordPrefix(s string) bool {
	if len(s) < 2 || s[0] != '@' {
		return false
	}

	switch c := s[1]; {
	case isIdentStartByte(c), c == '\\':
		return true
	case c == '-':
		return len(s) > 2 && (isIdentStartByte(s[2]) || s[2] == '-' || s[2] == '\\')
	default:
		return false
	}
}

// isIdentStartByte reports whether c can start a CSS identifier.
func isIdentStartByte(c byte) bool {
	switch {
	case c == '_':
		return true
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		return true
	case c >= 0x80:
		return true
	default:
		return false
	}
}

// atRuleName returns the at-keyword that starts a prelude, lowercased. The name
// ends at the first byte that cannot appear in one, so `@media(max-width:600px)`
// names `@media` rather than the whole condition: a missed name there would
// classify a rule-list block as declarations and let a rule-level `;` survive
// into the parser that loops on it.
func atRuleName(segment string) string {
	prelude := strings.TrimSpace(stripComments(segment))
	if !strings.HasPrefix(prelude, "@") {
		return ""
	}

	for i := 0; i < len(prelude); i++ {
		if !isCSSNameByte(prelude[i]) {
			return strings.ToLower(prelude[:i])
		}
	}

	return strings.ToLower(prelude)
}

// isCSSNameByte reports whether c can appear in an identifier or an at-keyword.
func isCSSNameByte(c byte) bool {
	switch {
	case c == '@' || c == '-':
		return true
	case c >= '0' && c <= '9':
		return true
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		return true
	default:
		return false
	}
}

// embedsRules reports whether the block opened by this prelude holds rules
// rather than declarations.
func embedsRules(segment string) bool {
	return ruleBlockAtRules[atRuleName(segment)]
}

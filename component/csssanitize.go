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
// Well-formed CSS is returned byte for byte unchanged, so a document that parses
// today parses to exactly the same stylesheet.
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
			if isEmptyPrelude(segment) {
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
			if isEmptyPrelude(segment) {
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
			// "@import url(...)" is kept, anything else had no block and is junk.
			if isAtStatement(segment) {
				out.WriteString(segment)
			}

			i = len(input)
		}
	}

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
		case c == '{' || c == ';' || c == '}':
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

// isEmptyPrelude reports whether a block prelude is empty once whitespace and
// comments are ignored.
func isEmptyPrelude(segment string) bool {
	return strings.TrimSpace(stripComments(segment)) == ""
}

// isAtStatement reports whether a rule-level statement is an at-rule, which may
// legitimately end with a semicolon ("@import", "@charset").
func isAtStatement(segment string) bool {
	return strings.HasPrefix(strings.TrimSpace(stripComments(segment)), "@")
}

// atRuleName returns the at-keyword that starts a prelude, lowercased.
func atRuleName(segment string) string {
	prelude := strings.TrimSpace(stripComments(segment))
	if !strings.HasPrefix(prelude, "@") {
		return ""
	}

	if end := strings.IndexAny(prelude, " \t\r\n{;"); end >= 0 {
		prelude = prelude[:end]
	}

	return strings.ToLower(prelude)
}

// embedsRules reports whether the block opened by this prelude holds rules
// rather than declarations.
func embedsRules(segment string) bool {
	return ruleBlockAtRules[atRuleName(segment)]
}

package component

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSanitizeInlineCSS pins the recovery rules applied to <mj-style inline>
// content before the stylesheet parser sees it. The reference inliner recovers
// from malformed CSS instead of failing, so each case names the recovery class
// and the bytes that survive it.
func TestSanitizeInlineCSS(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid stylesheet is untouched",
			input: ".a { color: red; }\n@media (max-width:600px) { .b { width: 10px } }\n.c, .d > .e { padding: 0 }",
			want:  ".a { color: red; }\n@media (max-width:600px) { .b { width: 10px } }\n.c, .d > .e { padding: 0 }",
		},
		{
			name:  "rule without a block is dropped",
			input: ".a color: red; .b { color: blue }",
			want:  " .b { color: blue }",
		},
		{
			name:  "stray closing brace is dropped",
			input: "} .a { color: red }",
			want:  " .a { color: red }",
		},
		{
			name:  "unterminated block is closed at end of input",
			input: `.a { color: red;`,
			want:  `.a { color: red;}`,
		},
		{
			name:  "unterminated comment is dropped",
			input: ".a { color: red } /* unterminated",
			want:  ".a { color: red } ",
		},
		{
			name:  "junk after a valid rule is dropped",
			input: ".a { color: red } .b font-weight: bold; }",
			want:  ".a { color: red } ",
		},
		{
			name:  "junk inside a rule list block is dropped",
			input: "@media (max-width:600px) { .a font-weight: bold; .b { width: 10px } }",
			want:  "@media (max-width:600px) { .b { width: 10px } }",
		},
		{
			name:  "at-rule name ends before the condition",
			input: "@media(max-width:600px){ .a font-weight: bold; .b { width: 10px } }",
			want:  "@media(max-width:600px){ .b { width: 10px } }",
		},
		{
			name:  "at-rule condition with a space is untouched",
			input: "@media screen and (max-width:600px){ .b { width: 10px } }",
			want:  "@media screen and (max-width:600px){ .b { width: 10px } }",
		},
		{
			name:  "statement at-rules keep their semicolon",
			input: "@import url(https://example.com/a.css); .a { color: red }",
			want:  "@import url(https://example.com/a.css); .a { color: red }",
		},
		{
			name:  "block with an empty prelude is dropped whole",
			input: `{ color: red } .a { color: blue }`,
			want:  " .a { color: blue }",
		},
		{
			name:  "braces and semicolons inside strings do not confuse the scan",
			input: `.a::after { content: "};{" } .b { color: red; }`,
			want:  `.a::after { content: "};{" } .b { color: red; }`,
		},
		{
			name:  "comments inside a prelude survive a following block",
			input: ".a /* note */ { color: red }",
			want:  ".a /* note */ { color: red }",
		},
		{
			name:  "bare at-sign is junk, not an at-rule",
			input: "@ x; .a { color: red }",
			want:  " .a { color: red }",
		},
		{
			name:  "at-sign alone is junk",
			input: "@;",
			want:  "",
		},
		{
			name:  "bare at-sign junk inside a rule list block is dropped",
			input: "@media y { @ x; .a { color: red } }",
			want:  "@media y { .a { color: red } }",
		},
		{
			name:  "vendor at-rules are at-rules",
			input: `@-webkit-keyframes x { from { opacity: 0 } }`,
			want:  `@-webkit-keyframes x { from { opacity: 0 } }`,
		},
		{
			name:  "unterminated rule list block is closed",
			input: "@media x { .a { color: red } ",
			want:  "@media x { .a { color: red } }",
		},
		{
			name:  "terminated trailing comment is kept",
			input: ".a { color: red } /* done */",
			want:  ".a { color: red } /* done */",
		},
		{
			name:  "semicolon inside a url does not end the statement",
			input: "@import url(a;b); .a { color: red }",
			want:  "@import url(a;b); .a { color: red }",
		},
		{
			name:  "legacy CDO and CDC are ignorable, not junk",
			input: ".a { color: red } -->",
			want:  ".a { color: red } -->",
		},
		{
			name:  "empty input stays empty",
			input: "",
			want:  "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.want, sanitizeInlineCSS(testCase.input))
		})
	}
}

// TestParseCSSRecoversFromMalformedCSS asserts a stylesheet the parser used to
// loop on is reduced to the rules the reference keeps, and that parseCSS returns
// instead of hanging. The input is the CI fuzz seed; see issue #10.
func TestParseCSSRecoversFromMalformedCSS(t *testing.T) {
	t.Parallel()

	var head MJMLHead

	// The reference (juice/postcss) drops this rule and applies nothing.
	sheet, err := head.parseCSS(".inline-class  font-weight: bold; }")
	require.NoError(t, err)
	require.Empty(t, sheet.Rules)
	require.Empty(t, sheet.MediaRules)

	// A malformed rule does not take its well-formed siblings with it. This is a
	// deliberate divergence: postcss drops the following rule too, which loses
	// valid styling; see testdata/README.md.
	sheet, err = head.parseCSS(`.inline-class font-weight: bold; } .ok { color: red }`)
	require.NoError(t, err)
	require.Len(t, sheet.Rules, 1)
	require.Equal(t, ".ok", sheet.Rules[0].Selectors)
	require.Equal(t, []Style{{Property: "color", Value: "red"}}, sheet.Rules[0].Declarations)

	// An unterminated block is valid CSS at the end of the input and keeps its
	// declarations, as it does in the reference when nothing precedes it.
	sheet, err = head.parseCSS(`.auto { text-decoration: underline;`)
	require.NoError(t, err)
	require.Len(t, sheet.Rules, 1)
	require.Equal(t, []Style{{Property: "text-decoration", Value: "underline"}}, sheet.Rules[0].Declarations)
}

// TestParseCSSMediaWithoutSpaceRecovers asserts the at-rule classification does
// not depend on a space after the at-keyword. Without it the rule-level-junk
// pass never runs inside the block and the stylesheet parser loops, so this case
// must return: go test's own timeout is the guard.
func TestParseCSSMediaWithoutSpaceRecovers(t *testing.T) {
	t.Parallel()

	var head MJMLHead

	sheet, err := head.parseCSS("@media(max-width:600px){ .a font-weight: bold; .b { width: 10px } }")
	require.NoError(t, err)
	require.Empty(t, sheet.Rules)
	require.Len(t, sheet.MediaRules, 1)
	require.Len(t, sheet.MediaRules[0].Rules, 1)
	require.Equal(t, ".b", sheet.MediaRules[0].Rules[0].Selectors)
}

// TestParseCSSDoesNotHangOnRuleLevelJunk asserts every shape that used to spin
// the stylesheet parser now returns. go test's own timeout is the guard: a
// regression shows up as a panic, not as a wrong value.
func TestParseCSSDoesNotHangOnRuleLevelJunk(t *testing.T) {
	t.Parallel()

	var head MJMLHead

	for _, source := range []string{
		".inline-class  font-weight: bold; }",
		"@ x;",
		"@;",
		"@1x;",
		"@media(max-width:600px){ .a font-weight: bold; .b { width: 10px } }",
		"@media y { @ x; .a { color: red } }",
		"@media x { .a { color: red } ",
		"} .a { color: red }",
		".a { color: red } .b font-weight: bold; }",
		".a { color: red",
		".a { color: red; /* unterminated",
	} {
		if _, err := head.parseCSS(source); err != nil {
			t.Fatalf("parseCSS(%q) returned %v, want a recovered stylesheet", source, err)
		}
	}
}

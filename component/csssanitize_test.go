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

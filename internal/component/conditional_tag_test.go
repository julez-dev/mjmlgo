package component

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mjmlMergeOracle is the regexp MJML itself uses in
// mjml-core/lib/helpers/mergeOutlookConditionnals.js. It is the independent
// source of truth for MergeOutlookConditionals.
var mjmlMergeOracle = regexp.MustCompile(`(<!\[endif]-->\s*?<!--\[if mso \| IE]>)`)

func TestMergeOutlookConditionals(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "adjacent blocks",
			input: `<!--[if mso | IE]><table><![endif]--><!--[if mso | IE]><tr><![endif]-->`,
		},
		{
			name:  "adjacent blocks with newline",
			input: "<!--[if mso | IE]><table><![endif]-->\n<!--[if mso | IE]><tr><![endif]-->",
		},
		{
			name:  "adjacent blocks with indentation",
			input: "<!--[if mso | IE]><table><![endif]-->  \n\t <!--[if mso | IE]><tr><![endif]-->",
		},
		{
			name: "chained blocks",
			input: `<!--[if mso | IE]><table><![endif]--><!--[if mso | IE]><tr><![endif]-->` +
				`<!--[if mso | IE]><td><![endif]-->`,
		},
		{
			name:  "different condition is not merged",
			input: `<!--[if mso | IE]><table><![endif]--><!--[if mso]><tr><![endif]-->`,
		},
		{
			name:  "content between blocks is not merged",
			input: `<!--[if mso | IE]><table><![endif]--><div>x</div><!--[if mso | IE]><tr><![endif]-->`,
		},
		{
			name:  "no conditional tags",
			input: `<html><body><div>x</div></body></html>`,
		},
		{
			name:  "open block only",
			input: `<!--[if mso | IE]><table><tr>`,
		},
		{
			name:  "trailing end marker",
			input: `<!--[if mso | IE]><table><![endif]-->`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t,
				mjmlMergeOracle.ReplaceAllString(testCase.input, ""),
				MergeOutlookConditionals(testCase.input),
			)
		})
	}
}

func TestConditionalTag(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `<!--[if mso | IE]><table><![endif]-->`, conditionalTag("<table>", false))
	assert.Equal(t, `<!--[if !mso | IE]><!--><table><!--<![endif]-->`, conditionalTag("<table>", true))
}

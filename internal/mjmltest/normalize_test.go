package mjmltest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseHTMLNormalization documents which differences are considered
// irrelevant and which ones are reported.
func TestParseHTMLNormalization(t *testing.T) {
	t.Parallel()

	equal := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "doctype case",
			a:    "<!DOCTYPE html><html><body></body></html>",
			b:    "<!doctype html><html><body></body></html>",
		},
		{
			name: "attribute order",
			a:    `<html><body><div width="600" height="20"></div></body></html>`,
			b:    `<html><body><div height="20" width="600"></div></body></html>`,
		},
		{
			name: "style declaration order",
			a:    `<html><body><div style="color:red;font-size:12px;"></div></body></html>`,
			b:    `<html><body><div style="font-size:12px;color:red"></div></body></html>`,
		},
		{
			name: "class token order",
			a:    `<html><body><div class="a b"></div></body></html>`,
			b:    `<html><body><div class="b a"></div></body></html>`,
		},
		{
			name: "formatting whitespace and comments",
			a:    "<html><body><p>\n  Hello   <b>world</b>\n</p><!-- note --></body></html>",
			b:    "<html><body><p>Hello <b>world</b></p></body></html>",
		},
		{
			name: "empty alt and style attributes",
			a:    `<html><body><img alt="" style="" src="a.png"></body></html>`,
			b:    `<html><body><img src="a.png"></body></html>`,
		},
		{
			name: "style whitespace and comments",
			a:    "<html><head><style>.a{color:red}</style></head></html>",
			b:    "<html><head><style>\n  /* layout */\n  .a {\n    color: red;\n  }\n</style></head></html>",
		},
		{
			name: "media query whitespace",
			a:    "<style>@media only screen and (min-width:480px){.a{width:50%}}</style>",
			b:    "<style>@media   only screen and (min-width:480px) { .a { width: 50%; } }</style>",
		},
		{
			name: "selector list whitespace",
			a:    "<style>.a,.b{color:red}</style>",
			b:    "<style>.a , .b { color: red; }</style>",
		},
		{
			name: "downlevel reveal wrapper on both sides",
			a:    `<!--[if !mso]><!--><meta charset="utf-8"><!--<![endif]-->`,
			b:    `<!--[if !mso]><!--><meta charset="utf-8"><!--<![endif]-->`,
		},
	}

	for _, testCase := range equal {
		t.Run("equal/"+testCase.name, func(t *testing.T) {
			t.Parallel()

			mismatches, err := CompareHTML(testCase.a, testCase.b)
			require.NoError(t, err)
			assert.Empty(t, mismatches, "documents should be considered equivalent")
		})
	}

	different := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "text content",
			a:    `<html><body><p>Hello</p></body></html>`,
			b:    `<html><body><p>Hello!</p></body></html>`,
		},
		{
			name: "style value",
			a:    `<html><body><div style="color:red;"></div></body></html>`,
			b:    `<html><body><div style="color:blue;"></div></body></html>`,
		},
		{
			name: "missing style declaration",
			a:    `<html><body><div style="color:red;text-align:left;"></div></body></html>`,
			b:    `<html><body><div style="color:red;"></div></body></html>`,
		},
		{
			name: "attribute value",
			a:    `<html><body><div bgcolor="#FFFFFF"></div></body></html>`,
			b:    `<html><body><div bgcolor="#FFF"></div></body></html>`,
		},
		{
			name: "style declaration value",
			a:    "<style>.a{color:red}</style>",
			b:    "<style>.a{color:blue}</style>",
		},
		{
			name: "style rule order",
			a:    "<style>.a{color:red}.b{color:blue}</style>",
			b:    "<style>.b{color:blue}.a{color:red}</style>",
		},
		{
			name: "style declaration order",
			a:    "<style>.a{margin:0;margin-top:5px}</style>",
			b:    "<style>.a{margin-top:5px;margin:0}</style>",
		},
		{
			name: "media query condition",
			a:    "<style>@media screen{.a{width:50%}}</style>",
			b:    "<style>@media print{.a{width:50%}}</style>",
		},
		{
			name: "downlevel reveal wrapper missing",
			a:    `<!--[if !mso]><!--><meta charset="utf-8"><!--<![endif]-->`,
			b:    `<meta charset="utf-8">`,
		},
		{
			name: "conditional comment payload",
			a:    `<!--[if mso]><table><tr><td>a</td></tr></table><![endif]-->`,
			b:    `<!--[if mso]><table><tr><td>b</td></tr></table><![endif]-->`,
		},
		{
			name: "condition",
			a:    `<!--[if mso]><table></table><![endif]-->`,
			b:    `<!--[if mso | IE]><table></table><![endif]-->`,
		},
	}

	for _, testCase := range different {
		t.Run("different/"+testCase.name, func(t *testing.T) {
			t.Parallel()

			mismatches, err := CompareHTML(testCase.a, testCase.b)
			require.NoError(t, err)
			assert.NotEmpty(t, mismatches, "documents should be considered different")
		})
	}
}

// TestParseHTMLStylesheet locks down how <style> bodies are canonicalized: the
// cascade is order sensitive, so rule and declaration order must survive, while
// formatting, comments and whitespace must not matter.
func TestParseHTMLStylesheet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		css      string
		expected string
	}{
		{
			name:     "qualified rule",
			css:      ".a { color: red; font-size: 12px !important }",
			expected: ".a{color:red;font-size:12px !important}",
		},
		{
			name:     "selector list",
			css:      "td , .b{color:red}",
			expected: "td,.b{color:red}",
		},
		{
			name:     "media query",
			css:      "@media only screen and (min-width:480px) { .a { width: 50%; } }",
			expected: "@media only screen and (min-width:480px){.a{width:50%}}",
		},
		{
			name:     "import",
			css:      `@import url(https://fonts.googleapis.com/css?family=Ubuntu:300,400,500,700);`,
			expected: "@import url(https://fonts.googleapis.com/css?family=Ubuntu:300,400,500,700)",
		},
		{
			name:     "comments dropped",
			css:      "/* header */\n.a /* inline */ { color: red }",
			expected: ".a{color:red}",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			node, err := ParseHTML("<style>" + testCase.css + "</style>")
			require.NoError(t, err)

			if !assert.Equal(t, testCase.expected, firstStylesheet(t, node)) {
				t.Log(Canonical(node))
			}
		})
	}
}

// firstStylesheet returns the canonical CSS of the first <style> element below node.
func firstStylesheet(t *testing.T, node *Node) string {
	t.Helper()

	if node.Kind == KindElement && node.Name == "style" {
		return node.Sheet
	}

	for _, child := range node.Children {
		if sheet := firstStylesheet(t, child); sheet != "" {
			return sheet
		}
	}

	return ""
}

func TestCanonical(t *testing.T) {
	t.Parallel()

	node, err := ParseHTML(`<!doctype html><html lang="en"><body><p style="color: red">Hi</p></body></html>`)
	require.NoError(t, err)

	const expected = `#document
  <!doctype html>
  <html lang="en">
    <head>
    <body>
      <p style="color:red;">
        #text Hi
`

	assert.Equal(t, expected, Canonical(node))
}

package component

import (
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/v2/node"
	"github.com/stretchr/testify/require"
)

// TestParseCSS asserts <mj-style inline> stylesheets are built with compiled
// selectors and lose nothing of their declarations.
func TestParseCSS(t *testing.T) {
	t.Parallel()

	var head MJMLHead

	t.Run("rules carry their declarations and a compiled selector", func(t *testing.T) {
		t.Parallel()

		sheet, err := head.parseCSS(`.a { color: red; font-size: 12px !important }`)
		require.NoError(t, err)
		require.Empty(t, sheet.MediaRules)
		require.Len(t, sheet.Rules, 1)

		rule := sheet.Rules[0]
		require.Equal(t, ".a", rule.Selectors)
		require.Equal(t, []Style{
			{Property: "color", Value: "red"},
			{Property: "font-size", Value: "12px", Important: true},
		}, rule.Declarations)

		sel, ok := rule.CompiledSelectors()
		require.True(t, ok)

		document := parseTestHTML(t, `<html><body><p class="a">x</p></body></html>`)
		require.Len(t, sel.Select(document), 1)
	})

	t.Run("selector lists stay one rule", func(t *testing.T) {
		t.Parallel()

		sheet, err := head.parseCSS(`.a, .b > .c { color: red }`)
		require.NoError(t, err)
		require.Len(t, sheet.Rules, 1)
		require.Equal(t, ".a, .b > .c", sheet.Rules[0].Selectors)

		sel, ok := sheet.Rules[0].CompiledSelectors()
		require.True(t, ok)

		document := parseTestHTML(t, `<html><body><p class="a">x</p><div class="b"><p class="c">y</p></div></body></html>`)
		require.Len(t, sel.Select(document), 2)
	})

	t.Run("media rules are built too", func(t *testing.T) {
		t.Parallel()

		sheet, err := head.parseCSS(`@media (max-width: 600px) { .b { width: 10px !important } } .a { color: red }`)
		require.NoError(t, err)

		require.Len(t, sheet.MediaRules, 1)
		require.Equal(t, "@media (max-width: 600px)", sheet.MediaRules[0].Condition)
		require.Len(t, sheet.MediaRules[0].Rules, 1)

		mediaRule := sheet.MediaRules[0].Rules[0]
		require.Equal(t, ".b", mediaRule.Selectors)
		require.Equal(t, []Style{{Property: "width", Value: "10px", Important: true}}, mediaRule.Declarations)

		_, ok := mediaRule.CompiledSelectors()
		require.True(t, ok)

		// The top-level rule after the media block is kept as well.
		require.Len(t, sheet.Rules, 1)
		require.Equal(t, ".a", sheet.Rules[0].Selectors)
	})

	t.Run("an unparsable selector does not fail the stylesheet", func(t *testing.T) {
		t.Parallel()

		// MJML tolerates a rule whose selectors do not compile; the inliner
		// skips it and styles the other rules as usual.
		sheet, err := head.parseCSS(`::unsupported { color: red } .a { color: blue }`)
		require.NoError(t, err)
		require.Len(t, sheet.Rules, 2)
		require.Equal(t, "::unsupported", sheet.Rules[0].Selectors)

		_, ok := sheet.Rules[0].CompiledSelectors()
		require.False(t, ok)

		_, ok = sheet.Rules[1].CompiledSelectors()
		require.True(t, ok)
	})

	t.Run("broken CSS is still an error", func(t *testing.T) {
		t.Parallel()

		_, err := head.parseCSS(`.a { color red }`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error parsing CSS")
	})
}

// TestParseCSSToInlineStyles asserts a parsed stylesheet survives being copied
// into the render context with its compiled selectors intact, which is how the
// inliner receives it.
func TestParseCSSToInlineStyles(t *testing.T) {
	t.Parallel()

	var head MJMLHead

	sheet, err := head.parseCSS(strings.Join([]string{
		`.a { color: red }`,
		`@media (max-width: 600px) { .b { color: blue } }`,
	}, "\n"))
	require.NoError(t, err)

	ctx := &RenderContext{InlineStyles: []Stylesheet{sheet}}
	require.Len(t, ctx.InlineStyles, 1)

	rule := &ctx.InlineStyles[0].Rules[0]

	sel, ok := rule.CompiledSelectors()
	require.True(t, ok)

	document := parseTestHTML(t, `<html><body><p class="a">x</p></body></html>`)
	require.Len(t, sel.Select(document), 1)
}

// renderHead renders an empty mj-head with ctx, so a test can assert the style
// blocks the head contributes for the CSS a component registered.
func renderHead(t *testing.T, ctx *RenderContext) string {
	t.Helper()

	ctx.Breakpoint = "480px"

	var output strings.Builder
	require.NoError(t, MJMLHead{}.Render(ctx, &output, &node.Node{Type: HeadTagName}))

	return output.String()
}

// TestHeadEmitsContributedCSS asserts the three contribution mechanisms a custom
// component has all reach the head, in a stable order: named component blocks in
// registration order, class rules inside the breakpoint media query, and the
// fluid-on-mobile rules for the registered classes.
func TestHeadEmitsContributedCSS(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{}
	ctx.AddComponentHeadStyle("mj-notice", func(breakpoint string) string {
		return "    @media only screen and (min-width:" + breakpoint + ") {\n        .mj-notice { padding: 16px !important; }\n    }\n"
	})
	ctx.AddMediaQuery("mj-notice-wide", []string{"width: 100% !important"})
	ctx.AddFluidOnMobileClass("mj-notice-fluid")

	head := renderHead(t, ctx)

	require.Contains(t, head,
		"    @media only screen and (min-width:480px) {\n        .mj-notice { padding: 16px !important; }\n    }\n")
	require.Contains(t, head, "@media only screen and (min-width:480px) {\n            .mj-notice-wide {\n                  width: 100% !important;")
	require.Contains(t, head, ".moz-text-html .mj-notice-wide {")
	require.Contains(t, head,
		"    @media only screen and (max-width:479px) {\n"+
			"        table.mj-notice-fluid { width: 100% !important; }\n"+
			"        td.mj-notice-fluid { width: auto !important; }\n"+
			"    }")

	// Rendering the same document shape twice must give the same bytes.
	second := renderHead(t, ctx)
	require.Equal(t, head, second)
}

// TestHeadEmitsMJMLImageFluidRulesUnchanged asserts mj-image's fluid-on-mobile
// block keeps the exact markup it had before component-contributed classes were
// supported, because built-in output is parity-locked.
func TestHeadEmitsMJMLImageFluidRulesUnchanged(t *testing.T) {
	t.Parallel()

	head := renderHead(t, &RenderContext{IncludeMobileFullWidthStyle: true})

	require.Contains(t, head,
		"    @media only screen and (max-width:479px) {\n"+
			"        table.mj-full-width-mobile { width: 100% !important; }\n"+
			"        td.mj-full-width-mobile { width: auto !important; }\n"+
			"    }")
}

// TestHeadOrdersMJMLImageFluidRulesBeforeCustomClasses asserts mj-image's own
// fluid class stays first when a custom component adds one, so built-in output
// keeps its position and the custom rule follows deterministically.
func TestHeadOrdersMJMLImageFluidRulesBeforeCustomClasses(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{IncludeMobileFullWidthStyle: true}
	ctx.AddFluidOnMobileClass("mj-notice-fluid")

	head := renderHead(t, ctx)

	require.Contains(t, head,
		"    @media only screen and (max-width:479px) {\n"+
			"        table.mj-full-width-mobile { width: 100% !important; }\n"+
			"        td.mj-full-width-mobile { width: auto !important; }\n"+
			"        table.mj-notice-fluid { width: 100% !important; }\n"+
			"        td.mj-notice-fluid { width: auto !important; }\n"+
			"    }")
}

package component

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// parseTestHTML parses document so a compiled selector can be run against it.
func parseTestHTML(t *testing.T, document string) *html.Node {
	t.Helper()

	node, err := html.Parse(strings.NewReader(document))
	require.NoError(t, err)

	return node
}

// TestRuleCompileSelectors asserts selectors are compiled once, with the rule,
// and that a rule whose selectors do not parse stays unusable instead of being
// retried on every render.
func TestRuleCompileSelectors(t *testing.T) {
	t.Parallel()

	document := parseTestHTML(t, `<html><body><p class="a">x</p><p class="b">y</p></body></html>`)

	t.Run("compiled selector selects nodes", func(t *testing.T) {
		t.Parallel()

		rule := Rule{Selectors: ".a, .b"}
		rule.CompileSelectors()

		sel, ok := rule.CompiledSelectors()
		require.True(t, ok)
		require.Len(t, sel.Select(document), 2)
	})

	t.Run("the compiled selector is reused", func(t *testing.T) {
		t.Parallel()

		rule := Rule{Selectors: ".a"}
		rule.CompileSelectors()

		first, ok := rule.CompiledSelectors()
		require.True(t, ok)

		second, ok := rule.CompiledSelectors()
		require.True(t, ok)

		// A second compile would return a different selector value.
		require.Same(t, first, second)
	})

	t.Run("an unusable selector reports false", func(t *testing.T) {
		t.Parallel()

		rule := Rule{Selectors: "!! not a selector"}
		rule.CompileSelectors()

		sel, ok := rule.CompiledSelectors()
		require.False(t, ok)
		require.Nil(t, sel)
	})

	t.Run("a rule built by hand is compiled on first use", func(t *testing.T) {
		t.Parallel()

		rule := Rule{Selectors: ".b"}

		sel, ok := rule.CompiledSelectors()
		require.True(t, ok)
		require.Len(t, sel.Select(document), 1)
	})

	t.Run("a copied rule carries the compiled selector", func(t *testing.T) {
		t.Parallel()

		// Stylesheets are stored and passed around by value, so the compiled
		// selector has to travel with the copy.
		rule := Rule{Selectors: ".a"}
		rule.CompileSelectors()

		copied := rule

		sel, ok := copied.CompiledSelectors()
		require.True(t, ok)
		require.Len(t, sel.Select(document), 1)
	})
}

// TestAddComponentHeadStyle asserts a component-level stylesheet is registered
// under its name: repeated registrations replace the block, keep its position
// and leave the other components' blocks alone.
func TestAddComponentHeadStyle(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{}

	ctx.AddComponentHeadStyle("mj-notice", func(string) string { return "first" })
	ctx.AddComponentHeadStyle("mj-panel", func(string) string { return "panel" })
	ctx.AddComponentHeadStyle("mj-notice", func(breakpoint string) string { return "second " + breakpoint })

	require.Len(t, ctx.ComponentHeadStyles, 2)
	require.Equal(t, "mj-notice", ctx.ComponentHeadStyles[0].Name)
	require.Equal(t, "second 480px", ctx.ComponentHeadStyles[0].Style("480px"))
	require.Equal(t, "mj-panel", ctx.ComponentHeadStyles[1].Name)
}

// TestAddMediaQuery asserts class rules enter the breakpoint media query in
// first-registration order and that a class registered twice keeps its first
// position with its latest rules.
func TestAddMediaQuery(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{}

	ctx.AddMediaQuery("mj-notice-wide", []string{"width: 100% !important"})
	ctx.AddMediaQuery("mj-notice-narrow", []string{"width: 50% !important"})
	ctx.AddMediaQuery("mj-notice-wide", []string{"width: 600px !important"})

	require.Equal(t, []string{"mj-notice-wide", "mj-notice-narrow"}, ctx.MJMLStyleOrder)
	require.Equal(t, []string{"width: 600px !important"}, ctx.MJMLStylesheet["mj-notice-wide"])
	require.Equal(t, []string{"width: 50% !important"}, ctx.MJMLStylesheet["mj-notice-narrow"])
}

// TestAddFluidOnMobileClass asserts the fluid-on-mobile class list is ordered and
// de-duplicated, so repeated instances contribute one rule set.
func TestAddFluidOnMobileClass(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{}

	ctx.AddFluidOnMobileClass("mj-notice-fluid")
	ctx.AddFluidOnMobileClass("")
	ctx.AddFluidOnMobileClass("mj-panel-fluid")
	ctx.AddFluidOnMobileClass("mj-notice-fluid")

	require.Equal(t, []string{"mj-notice-fluid", "mj-panel-fluid"}, ctx.fluidOnMobileClasses)
}

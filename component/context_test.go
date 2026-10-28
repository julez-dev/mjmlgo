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

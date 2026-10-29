package mjmlgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSplitSelectorList asserts a rule's selector text is split on the commas
// that separate selectors, and only on those.
func TestSplitSelectorList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		selector string
		want     []string
	}{
		{selector: ".a", want: []string{".a"}},
		{selector: ".a, .b", want: []string{".a", ".b"}},
		{selector: " .a ,.b , .c ", want: []string{".a", ".b", ".c"}},
		{selector: "a[href*=',']", want: []string{"a[href*=',']"}},
		{selector: `a[title="a, b"], .c`, want: []string{`a[title="a, b"]`, ".c"}},
		{selector: ".a:nth-child(2), .b", want: []string{".a:nth-child(2)", ".b"}},
		{selector: ":not(.a, .b) .c", want: []string{":not(.a, .b) .c"}},
		{selector: "", want: []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, splitSelectorList(tt.selector))
		})
	}
}

// TestSelectorSpecificity asserts the cascade compares selectors the way CSS and
// the reference inliner do. The inline origin ({1,0,0,0}) is asserted through the
// style attribute merge in TestStyleDecls.
func TestSelectorSpecificity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		selector string
		want     specificity
	}{
		{selector: "*", want: specificity{}},
		{selector: "p", want: specificity{0, 0, 0, 1}},
		{selector: ".a", want: specificity{0, 0, 1, 0}},
		{selector: ".a.b", want: specificity{0, 0, 2, 0}},
		{selector: "#id", want: specificity{0, 1, 0, 0}},
		{selector: "p.a", want: specificity{0, 0, 1, 1}},
		{selector: "div p.a", want: specificity{0, 0, 1, 2}},
		{selector: "div > p", want: specificity{0, 0, 0, 2}},
		{selector: "p + p", want: specificity{0, 0, 0, 2}},
		{selector: "p ~ p", want: specificity{0, 0, 0, 2}},
		{selector: "a[href]", want: specificity{0, 0, 1, 1}},
		{selector: "a[href^='http']", want: specificity{0, 0, 1, 1}},
		{selector: "p:first-child", want: specificity{0, 0, 0, 2}},
		{selector: "p:nth-child(2n+1)", want: specificity{0, 0, 0, 2}},
		{selector: "*:before", want: specificity{0, 0, 0, 1}},
		{selector: "p::first-line", want: specificity{0, 0, 0, 2}},
		{selector: "svg|rect", want: specificity{0, 0, 0, 1}},
		{selector: ".frame .target", want: specificity{0, 0, 2, 0}},
		// :not() contributes its most specific argument, not itself.
		{selector: "p:not(.a)", want: specificity{0, 0, 1, 1}},
		{selector: "p:not(#a, .b)", want: specificity{0, 1, 0, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, selectorSpecificity(tt.selector))
		})
	}
}

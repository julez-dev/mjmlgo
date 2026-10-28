package component

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUsedFonts(t *testing.T) {
	t.Parallel()

	newContext := func() *RenderContext {
		ctx := &RenderContext{}
		ctx.ensureFontRegistry()

		return ctx
	}

	t.Run("builtin fonts only when used, in registry order", func(t *testing.T) {
		t.Parallel()

		body := `<div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;"></div>` +
			`<div style="font-family:Lato, sans-serif;"></div>`

		require.Equal(t, []string{"Lato", "Ubuntu"}, newContext().usedFonts(body, nil))
	})

	t.Run("unused mj-font is not emitted", func(t *testing.T) {
		t.Parallel()

		ctx := newContext()
		ctx.addFont("Custom", "https://example.com/custom.css")

		body := `<div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;"></div>`

		require.Equal(t, []string{"Ubuntu"}, ctx.usedFonts(body, nil))
	})

	t.Run("used mj-font is emitted after the builtin fonts", func(t *testing.T) {
		t.Parallel()

		ctx := newContext()
		ctx.addFont("Custom", "https://example.com/custom.css")

		body := `<div style="font-family:Custom, sans-serif;"></div>` +
			`<div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;"></div>`

		require.Equal(t, []string{"Ubuntu", "Custom"}, ctx.usedFonts(body, nil))
	})

	t.Run("font usage in inline stylesheets is detected", func(t *testing.T) {
		t.Parallel()

		inline := []Stylesheet{{
			Rules: []Rule{{
				Selectors:    ".custom",
				Declarations: []Style{{Property: "font-family", Value: "Roboto, sans-serif"}},
			}},
		}}

		require.Equal(t, []string{"Roboto"}, newContext().usedFonts("", inline))
	})

	// MJML matches the quoted-segment regexp against the body, so a family name
	// only counts when it sits inside a quoted attribute after a font-family
	// declaration. Unquoted body text is not a usage.
	t.Run("unquoted font-family in body text is not a usage", func(t *testing.T) {
		t.Parallel()

		body := `<div>font-family:Ubuntu, sans-serif</div>`

		require.Empty(t, newContext().usedFonts(body, nil))
	})

	t.Run("body detection is case-insensitive", func(t *testing.T) {
		t.Parallel()

		body := `<div STYLE="FONT-FAMILY:uBuNtU, Helvetica, Arial, sans-serif;"></div>`

		require.Equal(t, []string{"Ubuntu"}, newContext().usedFonts(body, nil))
	})

	t.Run("inline stylesheet declaration is case-insensitive", func(t *testing.T) {
		t.Parallel()

		inline := []Stylesheet{{
			Rules: []Rule{{
				Selectors:    ".custom",
				Declarations: []Style{{Property: "FONT-FAMILY", Value: "lato, sans-serif"}},
			}},
		}}

		require.Equal(t, []string{"Lato"}, newContext().usedFonts("", inline))
	})

	// The reference regexp has no word boundary after the family name, so a
	// longer family that starts with a registered name counts as a usage.
	t.Run("family names match as substrings", func(t *testing.T) {
		t.Parallel()

		body := `<div style="font-family:Roboto Slab, serif;"></div>`

		require.Equal(t, []string{"Roboto"}, newContext().usedFonts(body, nil))
	})

	// A name that appears in the same quoted segment but before the declaration
	// does not match: the name has to follow `font-family:`.
	t.Run("name before the declaration in the same quote does not count", func(t *testing.T) {
		t.Parallel()

		body := `<div style="font-family:Verdana, sans-serif;" title="Ubuntu"></div>`

		require.Empty(t, newContext().usedFonts(body, nil))
	})

	t.Run("quoted body text pairs independently of attributes", func(t *testing.T) {
		t.Parallel()

		body := `<div>the "quick" brown <span style="font-family:Lato, sans-serif;">fox</span> "jumps"</div>`

		require.Equal(t, []string{"Lato"}, newContext().usedFonts(body, nil))
	})

	// The reference regexp anchors on quote characters rather than on attribute
	// values, so text that sits between a closing and an opening quote is a
	// candidate segment too. MJML 4.18.0 imports Lato for this document, so this
	// is faithful, not a bug to fix here.
	t.Run("text between two quote characters can match", func(t *testing.T) {
		t.Parallel()

		body := `<div style="font-family:Arial, sans-serif;">font-family: Lato, sans-serif</div>` +
			`<div style="font-family:Arial, sans-serif;"></div>`

		require.Equal(t, []string{"Lato"}, newContext().usedFonts(body, nil))
	})

	t.Run("a single quote is not enough for a match", func(t *testing.T) {
		t.Parallel()

		body := `<div style="font-family:Ubuntu, sans-serif;</div>`

		require.Empty(t, newContext().usedFonts(body, nil))
	})

	t.Run("unterminated quote is not a match", func(t *testing.T) {
		t.Parallel()

		body := `<div style="font-family:Ubuntu, sans-serif;`

		require.Empty(t, newContext().usedFonts(body, nil))
	})

	t.Run("inline stylesheet media rules are scanned", func(t *testing.T) {
		t.Parallel()

		inline := []Stylesheet{{
			MediaRules: []MediaRule{{
				Condition: "@media only screen and (max-width:480px)",
				Rules: []Rule{{
					Selectors:    ".mobile",
					Declarations: []Style{{Property: "font-family", Value: "Roboto, sans-serif"}},
				}},
			}},
		}}

		require.Equal(t, []string{"Roboto"}, newContext().usedFonts("", inline))
	})

	// MJML tests the inline stylesheet text with `font-family:[^;}]*<name>`, so
	// only font-family declarations (including vendor-prefixed ones) count; the
	// `font` shorthand carries no `font-family:` and is ignored.
	t.Run("inline font shorthand does not count", func(t *testing.T) {
		t.Parallel()

		inline := []Stylesheet{{
			Rules: []Rule{{
				Selectors:    ".custom",
				Declarations: []Style{{Property: "font", Value: "12px/1.5 Ubuntu, sans-serif"}},
			}},
		}}

		require.Empty(t, newContext().usedFonts("", inline))
	})

	t.Run("inline vendor prefixed font-family counts", func(t *testing.T) {
		t.Parallel()

		inline := []Stylesheet{{
			Rules: []Rule{{
				Selectors:    ".custom",
				Declarations: []Style{{Property: "-webkit-font-family", Value: "Ubuntu, sans-serif"}},
			}},
		}}

		require.Equal(t, []string{"Ubuntu"}, newContext().usedFonts("", inline))
	})

	t.Run("font without href is never emitted", func(t *testing.T) {
		t.Parallel()

		ctx := newContext()
		ctx.addFont("Broken", "")

		body := `<div style="font-family:Broken, Ubuntu, sans-serif;"></div>`

		require.Equal(t, []string{"Ubuntu"}, ctx.usedFonts(body, nil))
	})
}

func TestIndexFoldASCII(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		s      string
		substr string
		want   int
	}{
		{name: "exact", s: "font-family:Ubuntu", substr: "font-family:", want: 0},
		{name: "mixed case haystack", s: "FONT-FAMILY:Ubuntu", substr: "font-family:", want: 0},
		{name: "mixed case needle", s: "font-family:Ubuntu", substr: "FAMILY:", want: 5},
		{name: "both mixed", s: "Font-Family:Ubuntu", substr: "FONT-family:", want: 0},
		{name: "absent", s: "font-size:13px", substr: "font-family:", want: -1},
		{name: "empty needle", s: "anything", substr: "", want: 0},
		{name: "needle longer than haystack", s: "font", substr: "font-family:", want: -1},
		{name: "later occurrence", s: "abc font-family:Ubuntu", substr: "font-family:", want: 4},
		{name: "non-ASCII letters are not case folded", s: "FÖNT-FAMILY:", substr: "font-family:", want: -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.want, indexFoldASCII(test.s, test.substr))
		})
	}
}

// benchmarkBody builds a body the size of a large rendered document (24 KB)
// shaped like real MJML output: long quoted style attributes that carry a
// font-family declaration, table wrappers, and text containing quotes.
func benchmarkBody() string {
	families := []string{
		"Ubuntu, Helvetica, Arial, sans-serif",
		"Lato, Helvetica, Arial, sans-serif",
		"Custom, Helvetica, Arial, sans-serif",
		"Open Sans, Helvetica, Arial, sans-serif",
	}

	const block = `<table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="float:none;background:#ffffff;border:0;margin:0 auto;width:100%;"><tbody><tr><td align="left" style="font-size:0;padding:10px 25px;word-break:break-word;"><div style="font-family:%s;font-size:13px;line-height:1;text-align:left;color:#000000;">The "quick" brown fox <a href="https://example.com" style="color:#0a7d33;">jumps</a> over</div></td></tr></tbody></table>`

	var b strings.Builder
	b.Grow(24 << 10)

	for i := 0; b.Len() < 24<<10; i++ {
		b.WriteString(`<div style="font-family:`)
		b.WriteString(families[i%len(families)])
		b.WriteString(`;">`)
		b.WriteString(strings.Replace(block, "%s", families[(i+1)%len(families)], 1))
		b.WriteString(`</div>`)
	}

	return b.String()
}

// BenchmarkUsedFonts covers the font usage scan that runs once per render over
// the whole rendered body. It compiles no regexps and allocates only the small
// per-registry bookkeeping slices.
func BenchmarkUsedFonts(b *testing.B) {
	ctx := &RenderContext{}
	ctx.ensureFontRegistry()
	ctx.addFont("Custom", "https://example.com/custom.css")

	body := benchmarkBody()

	inline := []Stylesheet{{
		Rules: []Rule{{
			Selectors:    ".custom",
			Declarations: []Style{{Property: "font-family", Value: "Roboto, sans-serif"}},
		}},
	}}

	b.ReportAllocs()
	b.SetBytes(int64(len(body)))

	used := ctx.usedFonts(body, inline)
	if len(used) == 0 {
		b.Fatal("benchmark body does not use any registered font")
	}

	for b.Loop() {
		ctx.usedFonts(body, inline)
	}
}

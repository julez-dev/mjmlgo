package mjmlgo

// Tests for the inline stage: the two selector passes, the style declaration
// merge and the patching of the emitted markup. Tests for the whole rendering
// pipeline live in render_test.go.

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/component"
	"github.com/stretchr/testify/require"
)

func TestInlineCSS(t *testing.T) {
	t.Parallel()

	t.Run("adds declarations to matching nodes", func(t *testing.T) {
		t.Parallel()

		const input = `<html><body><p class="p-class">Hello</p></body></html>`

		var out bytes.Buffer
		require.NoError(t, inlineCSS(inlineContext(inlineSheet(".p-class",
			component.Style{Property: "font-size", Value: "22px", Important: true})),
			strings.NewReader(input), &out))

		require.Equal(t, `<html><body><p class="p-class" style="font-size:22px;">Hello</p></body></html>`, out.String())
	})

	t.Run("important overrides an existing declaration", func(t *testing.T) {
		t.Parallel()

		const input = `<html><body><p class="p-class" style="font-size: 20px;font-weight:300">Hello</p></body></html>`

		var out bytes.Buffer
		require.NoError(t, inlineCSS(inlineContext(inlineSheet(".p-class",
			component.Style{Property: "font-size", Value: "22px", Important: true},
			component.Style{Property: "font-weight", Value: "400"})),
			strings.NewReader(input), &out))

		// The existing declaration order is preserved, the important rule wins
		// and the plain rule loses against the node's own declaration.
		require.Equal(t, `<html><body><p class="p-class" style="font-size:22px;font-weight:300;">Hello</p></body></html>`, out.String())
	})

	t.Run("existing declaration order is preserved", func(t *testing.T) {
		t.Parallel()

		const input = `<html><body><p class="p-class" style="z-index:1;color:red">Hello</p></body></html>`

		var out bytes.Buffer
		require.NoError(t, inlineCSS(inlineContext(inlineSheet(".p-class",
			component.Style{Property: "border", Value: "0"},
			component.Style{Property: "color", Value: "green"})),
			strings.NewReader(input), &out))

		require.Equal(t, `<html><body><p class="p-class" style="z-index:1;color:red;border:0;">Hello</p></body></html>`, out.String())
	})

	t.Run("unmatched nodes are untouched", func(t *testing.T) {
		t.Parallel()

		// The markup is deliberately not canonical HTML (unquoted attribute
		// values, a bare <br>): a node no rule touches must not be reserialized.
		const input = `<html><body><p style=color:red class=other>a<br></p><div class=no-style>b</div></body></html>`

		var out bytes.Buffer
		require.NoError(t, inlineCSS(inlineContext(inlineSheet(".p-class",
			component.Style{Property: "font-size", Value: "22px"})),
			strings.NewReader(input), &out))

		require.Equal(t, input, out.String())
		require.NotContains(t, out.String(), "font-size")
	})

	t.Run("unparsable selectors are skipped", func(t *testing.T) {
		t.Parallel()

		const input = `<html><body><p class="p-class">Hello</p></body></html>`

		ctx := inlineContext(
			inlineSheet("!! not a selector", component.Style{Property: "color", Value: "red"}),
			inlineSheet(".p-class", component.Style{Property: "font-weight", Value: "bold"}),
		)

		var out bytes.Buffer
		require.NoError(t, inlineCSS(ctx, strings.NewReader(input), &out))

		require.Equal(t, `<html><body><p class="p-class" style="font-weight:bold;">Hello</p></body></html>`, out.String())
	})

	t.Run("layout declarations mirror to legacy attributes", func(t *testing.T) {
		t.Parallel()

		const input = `<html><body><table class="t"><tr><td class="c">x</td></tr></table><div class="d">y</div></body></html>`

		ctx := inlineContext(
			inlineSheet(".c",
				component.Style{Property: "width", Value: "50%"},
				component.Style{Property: "text-align", Value: "right"},
				component.Style{Property: "vertical-align", Value: "bottom"},
				component.Style{Property: "background-color", Value: "#000"}),
			inlineSheet(".d", component.Style{Property: "width", Value: "100px"}),
		)

		var out bytes.Buffer
		require.NoError(t, inlineCSS(ctx, strings.NewReader(input), &out))

		rendered := out.String()

		// px widths become numeric attributes, percentages stay as they are.
		require.Contains(t, rendered, `<td class="c" width="50%" align="right" valign="bottom" bgcolor="#000" style="width:50%;text-align:right;vertical-align:bottom;background-color:#000;">`)
		require.Contains(t, rendered, `<div class="d" width="100" style="width:100px;">`)
	})
}

// TestInlineCSSDeterminism asserts a render is byte-for-byte reproducible: the
// style attribute order used to come from Go map iteration, so two renders of
// the same document could differ.
func TestInlineCSSDeterminism(t *testing.T) {
	t.Parallel()

	input := `<html><body>` +
		`<p class="a b c" style="color:red;font-size:12px">x</p>` +
		`<div class="a" style="padding:0;margin:0">y</div>` +
		`</body></html>`

	ctx := inlineContext(inlineSheet(".a, .b, .c",
		component.Style{Property: "background-color", Value: "#fff"},
		component.Style{Property: "border", Value: "0"},
		component.Style{Property: "text-align", Value: "left"},
		component.Style{Property: "vertical-align", Value: "top"},
		component.Style{Property: "width", Value: "100px"},
		component.Style{Property: "font-weight", Value: "bold"},
	))

	var first bytes.Buffer
	require.NoError(t, inlineCSS(ctx, strings.NewReader(input), &first))

	for range 20 {
		var again bytes.Buffer
		require.NoError(t, inlineCSS(ctx, strings.NewReader(input), &again))
		require.Equal(t, first.String(), again.String())
	}
}

// inlineContext builds a render context with the given inline stylesheets.
func inlineContext(sheets ...component.Stylesheet) *component.RenderContext {
	return &component.RenderContext{InlineStyles: sheets}
}

// inlineSheet builds a stylesheet with one rule for selector and declarations.
// The selector is compiled the way the head parser compiles it.
func inlineSheet(selector string, declarations ...component.Style) component.Stylesheet {
	return component.Stylesheet{Rules: []component.Rule{newInlineRule(selector, declarations...)}}
}

func newInlineRule(selector string, declarations ...component.Style) component.Rule {
	rule := component.Rule{Selectors: selector, Declarations: declarations}
	rule.CompileSelectors()

	return rule
}

// TestStyleDecls locks down the inline style merge: declaration order is
// preserved, new properties are appended in application order, and the cascade
// picks the winning value by importance, then specificity, then source order.
func TestStyleDecls(t *testing.T) {
	t.Parallel()

	t.Run("appends new properties after the existing ones", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls
		require.NoError(t, parseStyleAttribute(&decls, "z-index: 1; color: red"))

		require.True(t, decls.set(ruleDeclaration("background-color", "#fff", 1)))
		require.True(t, decls.set(ruleDeclaration("border", "0", 2)))

		require.Equal(t, "z-index:1;color:red;background-color:#fff;border:0;", decls.String())
	})

	t.Run("a rule does not override the element's own style", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls
		require.NoError(t, parseStyleAttribute(&decls, "color:red"))

		require.False(t, decls.set(ruleDeclaration("color", "green", 1)))
		require.Equal(t, "color:red;", decls.String())
	})

	t.Run("an important declaration overrides the element's own style", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls
		require.NoError(t, parseStyleAttribute(&decls, "color:red;font-size:12px"))

		important := ruleDeclaration("color", "green", 1)
		important.priority = 2

		require.True(t, decls.set(important))
		require.Equal(t, "color:green;font-size:12px;", decls.String())
	})

	t.Run("the later rule wins at equal specificity", func(t *testing.T) {
		t.Parallel()

		first := ruleDeclaration("margin", "0", 1)
		second := ruleDeclaration("margin", "4px", 2)

		decls := styleDecls{first}
		require.True(t, decls.set(second))
		require.Equal(t, "margin:4px;", decls.String())
	})

	t.Run("the more specific rule wins regardless of source order", func(t *testing.T) {
		t.Parallel()

		specific := ruleDeclaration("color", "blue", 1)
		specific.specificity = specificity{0, 0, 2, 0}

		plain := ruleDeclaration("color", "red", 2)

		decls := styleDecls{specific}
		require.False(t, decls.set(plain))
		require.Equal(t, "color:blue;", decls.String())
	})

	t.Run("importance outranks specificity but not the element's own important style", func(t *testing.T) {
		t.Parallel()

		importantID := ruleDeclaration("color", "blue", 1)
		importantID.priority = 2
		importantID.specificity = specificity{0, 1, 0, 0}

		importantClass := ruleDeclaration("color", "green", 2)
		importantClass.priority = 2
		importantClass.specificity = specificity{0, 0, 1, 0}

		decls := styleDecls{importantID}
		require.False(t, decls.set(importantClass))
		require.Equal(t, "color:blue;", decls.String())

		var inline styleDecls
		require.NoError(t, parseStyleAttribute(&inline, "color:red !important"))
		require.False(t, inline.set(importantID))
		require.Equal(t, "color:red !important;", inline.String())
	})

	t.Run("a repeated property keeps its first position and the last value", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls
		require.NoError(t, parseStyleAttribute(&decls, "color:red;font-size:12px;color:blue"))

		require.Equal(t, "color:blue;font-size:12px;", decls.String())
	})

	t.Run("lookup reads the merged value", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls
		require.NoError(t, parseStyleAttribute(&decls, "width: 100px;"))

		value, ok := decls.lookup("width")
		require.True(t, ok)
		require.Equal(t, "100px", value)

		_, ok = decls.lookup("height")
		require.False(t, ok)
	})

	t.Run("empty declarations serialize to the empty string", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "", styleDecls(nil).String())
	})

	t.Run("an invalid declaration is rejected", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls

		require.Error(t, parseStyleAttribute(&decls, "color"))
		require.Error(t, parseStyleAttribute(&decls, "color:"))
	})

	t.Run("reset keeps the backing array", func(t *testing.T) {
		t.Parallel()

		decls := make(styleDecls, 0, 4)
		require.NoError(t, parseStyleAttribute(&decls, "color:red;font-size:12px"))

		buffer := &decls[0]
		decls.reset()

		require.Empty(t, decls)

		// Appending again reuses the same backing array, so no allocation happens
		// per matched node.
		require.NoError(t, parseStyleAttribute(&decls, "color:blue"))
		require.Same(t, buffer, &decls[0])
	})
}

// ruleDeclaration builds one class-selector declaration for the cascade tests:
// specificity {0, 0, 1, 0} - one class - at the given source position.
func ruleDeclaration(property, value string, order int) styleDecl {
	return styleDecl{
		property:    property,
		value:       value,
		specificity: specificity{0, 0, 1, 0},
		order:       order,
	}
}

// TestInlineCSSStylesRawContent asserts raw content sitting directly in a table
// body is styled in place: its shape is preserved exactly as the author wrote
// it, and both selector passes reach it, because the inline stage matches the
// emitted markup itself instead of a re-parsed document.
func TestInlineCSSStylesRawContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "bare row stays a row",
			raw:  `<tr><td class="cell">raw row</td></tr>`,
			want: `<tr><td class="cell" data-x="1" style="color:red;">raw row</td></tr>`,
		},
		{
			name: "nested table is not moved or reparented",
			raw:  `<table><tr><td class="cell">raw cell</td></tr></table>`,
			want: `<table><tr><td class="cell" data-x="1" style="color:red;">raw cell</td></tr></table>`,
		},
		{
			name: "generic markup is untouched around the styled node",
			raw:  `<div class="wrapper"><p class="cell">text</p></div>`,
			want: `<div class="wrapper"><p class="cell" data-x="1" style="color:red;">text</p></div>`,
		},
		{
			name: "a table child stays an element",
			raw:  `<caption class="cell">caption</caption>`,
			want: `<caption class="cell" data-x="1" style="color:red;">caption</caption>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			document := `<html><body><table><tbody>` +
				tt.raw +
				`<tr><td>content</td></tr>` +
				`</tbody></table></body></html>`

			ctx := inlineContext(inlineSheet(".cell", component.Style{Property: "color", Value: "red"}))
			ctx.HTMLAttributes = []component.HTMLAttributeSelector{
				{Path: ".cell", Attributes: []component.HTMLAttribute{{Name: "data-x", Value: "1"}}},
			}

			var out bytes.Buffer
			require.NoError(t, inlineCSS(ctx, strings.NewReader(document), &out))

			require.Equal(t, `<html><body><table><tbody>`+
				tt.want+
				`<tr><td>content</td></tr>`+
				`</tbody></table></body></html>`, out.String())
		})
	}
}

// TestInlineCSSMatchesAncestorsOfRawContent asserts a selector that depends on
// an ancestor outside a raw block matches inside it: the block is part of one
// document tree, so its ancestors are the elements that surround it.
func TestInlineCSSMatchesAncestorsOfRawContent(t *testing.T) {
	t.Parallel()

	const document = `<html><body><table class="frame"><tbody>` +
		`<div class="raw"><p class="cell">text</p></div>` +
		`</tbody></table></body></html>`

	ctx := inlineContext(inlineSheet(".frame .cell", component.Style{Property: "color", Value: "red"}))

	var out bytes.Buffer
	require.NoError(t, inlineCSS(ctx, strings.NewReader(document), &out))

	require.Equal(t, `<html><body><table class="frame"><tbody>`+
		`<div class="raw"><p class="cell" style="color:red;">text</p></div>`+
		`</tbody></table></body></html>`, out.String())
}

func TestApplyHTMLAttributes(t *testing.T) {
	t.Parallel()

	const document = `<html><head><title class="text">head</title></head><body>` +
		`<div class="text"><div>first</div></div>` +
		`<div class="text"><div>second</div></div>` +
		`</body></html>`

	emitted := buildEmittedDocument(document)

	selectors := []component.HTMLAttributeSelector{
		{
			Path:       ".text div",
			Attributes: []component.HTMLAttribute{{Name: "data-id", Value: "42"}},
		},
		{
			Path:       ".missing",
			Attributes: []component.HTMLAttribute{{Name: "data-missing", Value: "never"}},
		},
		{
			Path:       ".text div",
			Attributes: []component.HTMLAttribute{{Name: "data-extra", Value: "second"}},
		},
	}

	var applier styleApplier
	applier.applyHTMLAttributes(selectors, emitted.root)
	require.Len(t, applier.changed, 2)

	// The pass records no change when it sets the same values again.
	var again styleApplier
	again.applyHTMLAttributes(selectors, emitted.root)
	require.Empty(t, again.changed)

	var out bytes.Buffer
	require.NoError(t, emitted.write(&out, applier.changed))
	rendered := out.String()

	// Every match of the repeated selector gets both attributes.
	require.Equal(t, 2, strings.Count(rendered, `data-id="42"`))
	require.Equal(t, 2, strings.Count(rendered, `data-extra="second"`))

	// A selector matching nothing is a no-op.
	require.NotContains(t, rendered, "data-missing")

	// The pass only targets rendered body content, like MJML.
	require.Contains(t, rendered, `<title class="text">head</title>`)
}

// TestInlineCSSPatchesOnlyMatchedElements asserts the emitted markup is final
// except for the start tags a rule changed: a document with unusual author
// formatting keeps it byte for byte, so the inline stage can neither normalize
// nor reorder anything it was not asked to touch.
func TestInlineCSSPatchesOnlyMatchedElements(t *testing.T) {
	t.Parallel()

	const document = `<!doctype html>` +
		`<html><head><meta charset=utf-8 /><style>.picked{color:red}</style></head>` +
		`<body><p class='other' style="color: blue">a&nbsp;b<br></p>` +
		`<p class=picked>matched</p>` +
		`<p class=matched style="color: green">matched, rule loses</p>` +
		`<table><tr><td class=other width=100>x</td></tr></table></body></html>`

	ctx := inlineContext(inlineSheet(".picked, .matched", component.Style{Property: "color", Value: "red"}))

	var out bytes.Buffer
	require.NoError(t, inlineCSS(ctx, strings.NewReader(document), &out))

	// Only the start tag a declaration landed in is rewritten; everything else -
	// the lowercase doctype, the self-closing meta, the unquoted and
	// single-quoted attributes, the entity, the void br, the untouched table cell
	// and the matched element whose own style won - is untouched.
	require.Equal(t, strings.Replace(document,
		`<p class=picked>`, `<p class="picked" style="color:red;">`, 1), out.String())
}

// TestInlineCSSKeepsSelfClosingForm asserts a matched element the document
// emitted self-closed keeps that form, so VML and SVG shapes an author relies on
// are not rewritten into open tags.
func TestInlineCSSKeepsSelfClosingForm(t *testing.T) {
	t.Parallel()

	const document = `<html><body><v:image class="bg" src="x.png" /><img class="icon" src="i.png" /></body></html>`

	ctx := inlineContext(inlineSheet(".bg, .icon", component.Style{Property: "border", Value: "0"}))

	var out bytes.Buffer
	require.NoError(t, inlineCSS(ctx, strings.NewReader(document), &out))

	require.Equal(t, `<html><body>`+
		`<v:image class="bg" src="x.png" style="border:0;" />`+
		`<img class="icon" src="i.png" style="border:0;" />`+
		`</body></html>`, out.String())
}

// BenchmarkInlineStage measures the inline stage on the markup a render just
// emitted, so its cost can be read next to the end-to-end numbers of
// BenchmarkRenderMJML.
func BenchmarkInlineStage(b *testing.B) {
	for _, fixture := range []string{"inline-cascade.mjml", "head.mjml"} {
		path := filepath.Join(parityDir, fixture)

		input, err := os.Open(path)
		if err != nil {
			b.Fatal(err)
		}

		emitted, err := fixtureRenderer.renderMarkup(context.Background(), os.DirFS(filepath.Dir(path)), filepath.Base(path), input)
		require.NoError(b, err)
		require.NoError(b, input.Close())

		b.Run(fixture, func(b *testing.B) {
			b.SetBytes(int64(len(emitted.markup)))
			b.ReportAllocs()

			for b.Loop() {
				var out strings.Builder
				if err := inlineCSS(emitted.ctx, strings.NewReader(emitted.markup), &out); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestInlineCSSMatchesTheReferenceTree asserts the emitted markup is turned into
// the tree the reference inliner's parser builds (htmlparser2 in HTML mode), so
// selectors match what the reference matches: a block start closes an open
// paragraph, a list item closes the previous one, a table row and cell close
// their predecessor, and the self-closing flag is ignored outside foreign
// content.
func TestInlineCSSMatchesTheReferenceTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		document string
		sheet    component.Stylesheet
		want     string
	}{
		{
			name:     "a block start closes a paragraph",
			document: `<html><body><p>text<div class="inner">nested</div></body></html>`,
			sheet:    inlineSheet("p div, p > .inner", component.Style{Property: "color", Value: "red"}),
			want:     `<html><body><p>text<div class="inner">nested</div></body></html>`,
		},
		{
			name:     "a list item closes the previous one",
			document: `<html><body><ul><li class="item">first<li class="item">second</ul></body></html>`,
			sheet:    inlineSheet("li li", component.Style{Property: "color", Value: "red"}),
			want:     `<html><body><ul><li class="item">first<li class="item">second</ul></body></html>`,
		},
		{
			name:     "a row and a cell close their predecessor",
			document: `<html><body><table><tr><td>one<td>two<tr><td>three</table></body></html>`,
			sheet:    inlineSheet("tr tr td", component.Style{Property: "color", Value: "red"}),
			want:     `<html><body><table><tr><td>one<td>two<tr><td>three</table></body></html>`,
		},
		{
			name:     "a self-closed HTML element keeps the content that follows",
			document: `<html><body><div class="wrap" /><span class="inner">text</span></body></html>`,
			sheet:    inlineSheet(".wrap .inner", component.Style{Property: "color", Value: "red"}),
			want:     `<html><body><div class="wrap" /><span class="inner" style="color:red;">text</span></body></html>`,
		},
		{
			name:     "a self-closed element in foreign content is a leaf",
			document: `<html><body><svg><rect class="shape" /><circle class="shape" /></svg></body></html>`,
			sheet:    inlineSheet(".shape", component.Style{Property: "fill", Value: "#00f"}),
			want:     `<html><body><svg><rect class="shape" style="fill:#00f;" /><circle class="shape" style="fill:#00f;" /></svg></body></html>`,
		},
		{
			name:     "a self-closed shape does not contain the next shape",
			document: `<html><body><svg><rect class="shape" /><circle class="shape" /></svg></body></html>`,
			sheet:    inlineSheet(".shape .shape", component.Style{Property: "stroke", Value: "#000"}),
			want:     `<html><body><svg><rect class="shape" /><circle class="shape" /></svg></body></html>`,
		},
		{
			name:     "a self-closed cell is closed by the next cell",
			document: `<html><body><table><tr><td class="pad" /><td class="cell">x</td></tr></table></body></html>`,
			sheet:    inlineSheet("td td", component.Style{Property: "color", Value: "red"}),
			want:     `<html><body><table><tr><td class="pad" /><td class="cell">x</td></tr></table></body></html>`,
		},
		{
			name:     "a void element stays a leaf",
			document: `<html><body><div><img class="icon" src="i.png"><span class="next">x</span></div></body></html>`,
			sheet:    inlineSheet("img span", component.Style{Property: "color", Value: "red"}),
			want:     `<html><body><div><img class="icon" src="i.png"><span class="next">x</span></div></body></html>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			require.NoError(t, inlineCSS(inlineContext(tt.sheet), strings.NewReader(tt.document), &out))
			require.Equal(t, tt.want, out.String())
		})
	}
}

// TestInlineCSSStrayVoidEndTag asserts a stray end tag for a void element does
// not shift the patch positions: the bytes it occupies belong to the emitted
// markup, so the elements after it are still patched where they are.
func TestInlineCSSStrayVoidEndTag(t *testing.T) {
	t.Parallel()

	const document = `<html><body><div><br></br></div><p class="t">x</p></body></html>`

	ctx := inlineContext(inlineSheet(".t", component.Style{Property: "color", Value: "red"}))

	var out bytes.Buffer
	require.NoError(t, inlineCSS(ctx, strings.NewReader(document), &out))

	require.Equal(t, `<html><body><div><br></br></div><p class="t" style="color:red;">x</p></body></html>`, out.String())

	// The same for a stray </hr> and for a stray end tag of an element that was
	// never opened: whatever the tree ignores, the offsets keep counting.
	const stray = `<html><body><hr></hr><span class="t">y</span></div></body></html>`

	var second bytes.Buffer
	require.NoError(t, inlineCSS(ctx, strings.NewReader(stray), &second))

	require.Equal(t, `<html><body><hr></hr><span class="t" style="color:red;">y</span></div></body></html>`, second.String())
}

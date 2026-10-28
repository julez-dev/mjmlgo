package component

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bodyProbe is a custom component that records the container width the body
// handed it and writes fixed html, so the body tests can see the dispatch and
// the width propagation instead of a built-in component's markup.
type bodyProbe struct {
	html  string
	err   error
	width string
}

func (p *bodyProbe) Name() string { return "mj-probe" }

func (*bodyProbe) AllowedAttributes() map[string]AttributeValidator { return nil }

func (*bodyProbe) DefaultAttributes(*RenderContext) map[string]string { return nil }

func (p *bodyProbe) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	p.width = ctx.ContainerWidth

	if p.err != nil {
		return p.err
	}

	_, err := io.WriteString(w, p.html)

	return err
}

func renderMJMLBody(t *testing.T, ctx *RenderContext, n *node.Node) (string, error) {
	t.Helper()

	var body MJMLBody
	if err := InitComponent(ctx, body, n); err != nil {
		return "", err
	}

	var out strings.Builder
	err := body.Render(ctx, &out, n)

	return out.String(), err
}

// TestMJMLBodyDefaults asserts the body's only default is the document width,
// and that an author width survives it.
func TestMJMLBodyDefaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t, BodyTagName, MJMLBody{}.Name())

	t.Run("the document width defaults to 600px", func(t *testing.T) {
		t.Parallel()

		n := &node.Node{Type: BodyTagName}
		require.NoError(t, InitComponent(&RenderContext{}, MJMLBody{}, n))
		requireAttribute(t, n, "width", "600px")
	})

	t.Run("an author width is not overridden", func(t *testing.T) {
		t.Parallel()

		n := &node.Node{Type: BodyTagName, Attributes: attributes("width", "480px")}
		require.NoError(t, InitComponent(&RenderContext{}, MJMLBody{}, n))
		requireAttribute(t, n, "width", "480px")
	})
}

// TestMJMLBodyValidatesWidthAndBackground asserts the body accepts the two
// attributes it renders and rejects values the document validator must catch
// before rendering: a non-pixel width and an unknown colour.
func TestMJMLBodyValidatesWidthAndBackground(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		attrs []string
		want  string
	}{
		{name: "pixel width", attrs: []string{"width", "480px"}},
		{name: "hex background", attrs: []string{"background-color", "#ff0000"}},
		{name: "named background", attrs: []string{"background-color", "red"}},
		{name: "percentage width", attrs: []string{"width", "50%"}, want: "is not one of [px]"},
		{name: "em width", attrs: []string{"width", "10em"}, want: "is not one of [px]"},
		{name: "unknown colour", attrs: []string{"background-color", "chartreuseish"}, want: "not a valid color"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			n := &node.Node{Type: BodyTagName, Attributes: attributes(tt.attrs...)}

			err := InitComponent(&RenderContext{}, MJMLBody{}, n)
			if tt.want == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

// TestMJMLBodySeedsTheContainerWidth asserts the body publishes its width as the
// container width before any child renders, which is what every nested
// container and content component measures against.
func TestMJMLBodySeedsTheContainerWidth(t *testing.T) {
	t.Parallel()

	t.Run("an author width replaces the incoming container width", func(t *testing.T) {
		t.Parallel()

		probe := &bodyProbe{html: "probe"}
		ctx := contextWith(probe)
		ctx.ContainerWidth = "900px"

		n := &node.Node{
			Type:       BodyTagName,
			Attributes: attributes("width", "480px"),
			Children:   []*node.Node{{Type: "mj-probe"}},
		}

		out, err := renderMJMLBody(t, ctx, n)
		require.NoError(t, err)
		assert.Equal(t, "480px", probe.width, "the child renders at the body width")
		assert.Equal(t, "480px", ctx.ContainerWidth)
		assert.Contains(t, out, "probe")
	})

	t.Run("the default width seeds a context that has none", func(t *testing.T) {
		t.Parallel()

		probe := &bodyProbe{html: "probe"}
		ctx := contextWith(probe)
		n := &node.Node{Type: BodyTagName, Children: []*node.Node{{Type: "mj-probe"}}}

		_, err := renderMJMLBody(t, ctx, n)
		require.NoError(t, err)
		assert.Equal(t, "600px", probe.width)
		assert.Equal(t, "600px", ctx.ContainerWidth)
	})
}

// TestMJMLBodyBackground asserts a background colour reaches both the page body
// and the content column, and that a body without one emits no background
// declaration at all.
func TestMJMLBodyBackground(t *testing.T) {
	t.Parallel()

	t.Run("the colour styles the page and the content column", func(t *testing.T) {
		t.Parallel()

		n := &node.Node{Type: BodyTagName, Attributes: attributes("background-color", "#ff0000")}

		out, err := renderMJMLBody(t, &RenderContext{}, n)
		require.NoError(t, err)

		assert.Contains(t, out, `<body style="word-spacing:normal;background-color:#ff0000;">`)
		assert.Contains(t, out, `style="background-color:#ff0000;"`, "the content column repeats the colour")
	})

	t.Run("no colour leaves the page style alone", func(t *testing.T) {
		t.Parallel()

		out, err := renderMJMLBody(t, &RenderContext{}, &node.Node{Type: BodyTagName})
		require.NoError(t, err)

		assert.Contains(t, out, `<body style="word-spacing:normal;">`)
		assert.NotContains(t, out, "background-color")
	})
}

// TestMJMLBodyDocumentMetadata asserts the document-level head values land on the
// content column: title, language, direction and the author's css-class.
func TestMJMLBodyDocumentMetadata(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{Title: "Weekly digest", Language: "de", Direction: "rtl"}
	n := &node.Node{Type: BodyTagName, Attributes: attributes("css-class", "page")}

	out, err := renderMJMLBody(t, ctx, n)
	require.NoError(t, err)

	assert.Contains(t, out, `aria-label="Weekly digest"`)
	assert.Contains(t, out, `aria-roledescription="email"`)
	assert.Contains(t, out, `role="article"`)
	assert.Contains(t, out, `lang="de"`)
	assert.Contains(t, out, `dir="rtl"`)
	assert.Contains(t, out, `class="page"`)
}

// TestMJMLBodyPreviewText asserts the preview text is emitted as the hidden
// preheader before the content column, so clients show it before the body.
func TestMJMLBodyPreviewText(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{PreviewText: "Open me"}
	n := &node.Node{Type: BodyTagName}

	out, err := renderMJMLBody(t, ctx, n)
	require.NoError(t, err)

	preview := strings.Index(out, "Open me")
	content := strings.Index(out, `role="article"`)
	require.GreaterOrEqual(t, preview, 0)
	require.GreaterOrEqual(t, content, 0)
	assert.Less(t, preview, content, "the preview text comes before the content column")
	assert.Contains(t, out, "display:none")
}

// TestMJMLBodyRendersChildrenInDocumentOrder asserts raw children and component
// children render at their document position, and that raw content inside the
// body div is emitted without the table-body markers.
func TestMJMLBodyRendersChildrenInDocumentOrder(t *testing.T) {
	t.Parallel()

	probe := &bodyProbe{html: "<span>second</span>"}
	ctx := contextWith(probe)
	n := &node.Node{
		Type: BodyTagName,
		Children: []*node.Node{
			{Type: RawTagName, Content: "<div>first</div>"},
			{Type: "mj-probe"},
			{Type: RawTagName, Content: "<div>third</div>"},
		},
	}

	out, err := renderMJMLBody(t, ctx, n)
	require.NoError(t, err)

	first := strings.Index(out, "<div>first</div>")
	second := strings.Index(out, "<span>second</span>")
	third := strings.Index(out, "<div>third</div>")
	require.GreaterOrEqual(t, first, 0)
	require.GreaterOrEqual(t, second, 0)
	require.GreaterOrEqual(t, third, 0)
	assert.Less(t, first, second)
	assert.Less(t, second, third)
	assert.NotContains(t, out, RawBlockStartMarker, "body raw content sits outside a table body")
	assert.NotContains(t, out, RawBlockEndMarker)
}

// TestMJMLBodyDispatchesLayoutChildren asserts the body renders each layout
// element with its own renderer: a section, a wrapper - proven by the
// wrapper-only gap attribute being accepted - and a hero.
func TestMJMLBodyDispatchesLayoutChildren(t *testing.T) {
	t.Parallel()

	t.Run("section", func(t *testing.T) {
		t.Parallel()

		column := &node.Node{
			Type:     ColumnTagName,
			Children: []*node.Node{{Type: TextTagName, Content: "copy"}},
		}
		section := &node.Node{Type: SectionTagName, Children: []*node.Node{column}}
		column.Parent = section
		column.Children[0].Parent = column

		n := &node.Node{Type: BodyTagName, Children: []*node.Node{section}}

		out, err := renderMJMLBody(t, &RenderContext{}, n)
		require.NoError(t, err)
		assert.Contains(t, out, `<div style="margin:0px auto;max-width:600px;">`)
		assert.Contains(t, out, ">copy</div>")
	})

	t.Run("wrapper", func(t *testing.T) {
		t.Parallel()

		wrapper := &node.Node{Type: WrapperTagName, Attributes: attributes("gap", "10px")}
		n := &node.Node{Type: BodyTagName, Children: []*node.Node{wrapper}}

		out, err := renderMJMLBody(t, &RenderContext{}, n)
		require.NoError(t, err, "gap is a wrapper attribute; a section renderer would reject it")
		assert.NotEmpty(t, out)
	})

	t.Run("hero", func(t *testing.T) {
		t.Parallel()

		hero := &node.Node{Type: HeroTagName, Attributes: attributes(
			"mode", "fluid-height",
			"background-width", "600px",
			"background-height", "469px",
			"background-url", "https://example.com/bg.jpg",
		)}
		n := &node.Node{Type: BodyTagName, Children: []*node.Node{hero}}

		out, err := renderMJMLBody(t, &RenderContext{}, n)
		require.NoError(t, err)
		assert.Contains(t, out, "<v:image")
	})
}

// TestMJMLBodyReportsAChildNoComponentCanRender asserts the defensive branch:
// validation rejects every other child before rendering, so reaching it is a
// broken invariant and must be reported instead of dropped silently.
func TestMJMLBodyReportsAChildNoComponentCanRender(t *testing.T) {
	t.Parallel()

	n := &node.Node{Type: BodyTagName, Children: []*node.Node{{Type: "mj-mystery"}}}

	_, err := renderMJMLBody(t, &RenderContext{}, n)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no component for allowed child <mj-mystery> in <mj-body>")
}

// TestMJMLBodyWrapsCustomComponentErrors asserts a failing custom child is
// reported with the tag name and keeps the cause reachable, so a host can tell
// its own component failed from a renderer bug.
func TestMJMLBodyWrapsCustomComponentErrors(t *testing.T) {
	t.Parallel()

	cause := errors.New("probe failed")
	probe := &bodyProbe{err: cause}
	ctx := contextWith(probe)
	n := &node.Node{Type: BodyTagName, Children: []*node.Node{{Type: "mj-probe"}}}

	_, err := renderMJMLBody(t, ctx, n)
	require.Error(t, err)
	assert.ErrorIs(t, err, cause)
	assert.Contains(t, err.Error(), "failed to render <mj-probe>: probe failed")
}

package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCarouselNode builds an mj-carousel node with the given attributes and one
// mj-carousel-image child per attribute set.
func newCarouselNode(attributes []xml.Attr, images ...[]xml.Attr) *node.Node {
	carousel := &node.Node{Type: CarouselTagName, Attributes: attributes}

	for _, imageAttributes := range images {
		carousel.Children = append(carousel.Children, &node.Node{
			Type:       CarouselImageTagName,
			Parent:     carousel,
			Attributes: imageAttributes,
		})
	}

	return carousel
}

func TestMJMLCarousel(t *testing.T) {
	t.Parallel()

	t.Run("renders radios, thumbnails, images and the Outlook fallback", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{Breakpoint: "480px", ContainerWidth: "600px", MJMLStylesheet: make(map[string][]string)}

		carousel := newCarouselNode(
			[]xml.Attr{{Name: xml.Name{Local: "icon-width"}, Value: "32px"}},
			[]xml.Attr{{Name: xml.Name{Local: "src"}, Value: "https://example.com/one.png"}},
			[]xml.Attr{
				{Name: xml.Name{Local: "src"}, Value: "https://example.com/two.png"},
				{Name: xml.Name{Local: "thumbnails-src"}, Value: "https://example.com/two-thumb.png"},
				{Name: xml.Name{Local: "href"}, Value: "https://example.com/two"},
			},
		)

		var component MJMLCarousel
		require.NoError(t, InitComponent(ctx, component, carousel))

		var out strings.Builder
		require.NoError(t, component.Render(ctx, &out, carousel))
		html := out.String()

		// The whole carousel is hidden from Outlook, which gets the first image.
		assert.True(t, strings.HasPrefix(html, `<!--[if !mso]><!--><div class="mj-carousel">`))
		assert.Contains(t, html, `<!--[if mso]><div class="mj-carousel-image mj-carousel-image-1 " style="">`)
		assert.True(t, strings.HasSuffix(html, `<![endif]-->`))

		assert.Contains(t, html, `checked="checked" class="mj-carousel-radio mj-carousel-0000000000000000-radio mj-carousel-0000000000000000-radio-1" id="mj-carousel-0000000000000000-radio-1" name="mj-carousel-radio-0000000000000000"`)
		assert.NotContains(t, html, `class="mj-carousel-radio mj-carousel-0000000000000000-radio mj-carousel-0000000000000000-radio-2" checked`)

		assert.Contains(t, html, `class="mj-carousel-content mj-carousel-0000000000000000-content" style="display:table;width:100%;table-layout:fixed;text-align:center;font-size:0px;"`)
		assert.Contains(t, html, `class="mj-carousel-thumbnail mj-carousel-0000000000000000-thumbnail mj-carousel-0000000000000000-thumbnail-1 `)
		assert.Contains(t, html, `href="#2"`)
		assert.Contains(t, html, `src="https://example.com/two-thumb.png"`)
		assert.Contains(t, html, `<label for="mj-carousel-0000000000000000-radio-2">`)

		// Both arrow cells repeat the icon once per image.
		assert.Equal(t, 2, strings.Count(html, `class="mj-carousel-previous mj-carousel-previous-`))
		assert.Equal(t, 2, strings.Count(html, `class="mj-carousel-next mj-carousel-next-`))
		assert.Contains(t, html, `style="display:block;width:32px;height:auto;" width="32"`)

		// The second slide is hidden inline, the first is the visible one.
		assert.Contains(t, html, `class="mj-carousel-image mj-carousel-image-1 " style=""><img`)
		assert.Contains(t, html, `class="mj-carousel-image mj-carousel-image-2 " style="display:none;mso-hide:all;"`)
		assert.Contains(t, html, `width:600px;max-width:100%;height:auto;`)
	})
}

func TestMJMLCarouselThumbnails(t *testing.T) {
	t.Parallel()

	render := func(t *testing.T, mode string) string {
		t.Helper()

		ctx := &RenderContext{Breakpoint: "480px", ContainerWidth: "600px"}

		carousel := newCarouselNode(
			[]xml.Attr{{Name: xml.Name{Local: "thumbnails"}, Value: mode}},
			[]xml.Attr{{Name: xml.Name{Local: "src"}, Value: "https://example.com/one.png"}},
		)

		var component MJMLCarousel
		require.NoError(t, InitComponent(ctx, component, carousel))

		var out strings.Builder
		require.NoError(t, component.Render(ctx, &out, carousel))

		return out.String()
	}

	t.Run("visible is the default and uses inline-block", func(t *testing.T) {
		t.Parallel()

		html := render(t, "visible")

		assert.Contains(t, html, `class="mj-carousel-thumbnail `)
		assert.Contains(t, html, `overflow:hidden;width:110px;`)
		assert.Contains(t, html, `href="#1"`)
		assert.NotContains(t, html, `display:none;overflow:hidden`)
	})

	t.Run("hidden renders no thumbnails", func(t *testing.T) {
		t.Parallel()

		html := render(t, "hidden")

		assert.NotContains(t, html, `class="mj-carousel-thumbnail `)
		assert.Contains(t, html, `class="mj-carousel-images"`)
	})

	t.Run("supported hides the thumbnails inline for the client to reveal", func(t *testing.T) {
		t.Parallel()

		html := render(t, "supported")

		assert.Contains(t, html, `class="mj-carousel-thumbnail `)
		assert.Contains(t, html, `display:none;overflow:hidden;width:110px;`)
	})
}

func TestMJMLCarouselHeadStyle(t *testing.T) {
	t.Parallel()

	t.Run("registers one stylesheet per instance", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{Breakpoint: "480px", ContainerWidth: "600px"}

		var rendered []string

		for range 2 {
			carousel := newCarouselNode(
				nil,
				[]xml.Attr{{Name: xml.Name{Local: "src"}, Value: "https://example.com/one.png"}},
			)

			var component MJMLCarousel
			require.NoError(t, InitComponent(ctx, component, carousel))

			var out strings.Builder
			require.NoError(t, component.Render(ctx, &out, carousel))

			rendered = append(rendered, out.String())
		}

		// MJML's addComponentHeadSyle keeps every instance style, unlike the
		// per-component head styles of mj-navbar and mj-accordion.
		require.Len(t, ctx.InstanceComponentHeadStyles, 2)
		assert.Empty(t, ctx.ComponentHeadStyles)

		first := ctx.InstanceComponentHeadStyles[0].Style("480px")
		second := ctx.InstanceComponentHeadStyles[1].Style("480px")

		assert.Contains(t, first, ".mj-carousel-0000000000000000-icons-cell")
		assert.Contains(t, second, ".mj-carousel-0000000000000001-icons-cell")
		assert.NotEqual(t, first, second)

		// The generated ids are deterministic per render.
		assert.Contains(t, rendered[0], `id="mj-carousel-0000000000000000-radio-1"`)
		assert.Contains(t, rendered[1], `id="mj-carousel-0000000000000001-radio-1"`)
	})

	t.Run("generates the per-image rules, the media query and the fallback", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{Breakpoint: "480px", ContainerWidth: "600px"}

		carousel := newCarouselNode(
			[]xml.Attr{
				{Name: xml.Name{Local: "icon-width"}, Value: "32px"},
				{Name: xml.Name{Local: "tb-selected-border-color"}, Value: "#ccc"},
				{Name: xml.Name{Local: "tb-hover-border-color"}, Value: "#fead0d"},
			},
			[]xml.Attr{{Name: xml.Name{Local: "src"}, Value: "https://example.com/one.png"}},
			[]xml.Attr{{Name: xml.Name{Local: "src"}, Value: "https://example.com/two.png"}},
			[]xml.Attr{{Name: xml.Name{Local: "src"}, Value: "https://example.com/three.png"}},
		)

		var component MJMLCarousel
		require.NoError(t, InitComponent(ctx, component, carousel))
		require.NoError(t, component.Render(ctx, &strings.Builder{}, carousel))

		require.Len(t, ctx.InstanceComponentHeadStyles, 1)
		style := ctx.InstanceComponentHeadStyles[0].Style("480px")

		// The icon width comes from the attribute, colours are hex expanded.
		assert.Contains(t, style, "width: 32px !important;")
		assert.Contains(t, style, "border-color: #cccccc !important;")
		assert.Contains(t, style, "border-color: #fead0d !important;")

		// One selector per image, in image order.
		assert.Contains(t, style, "radio-1:checked + * + * + .mj-carousel-content .mj-carousel-image-1,")
		assert.Contains(t, style, "radio-3:checked + .mj-carousel-content .mj-carousel-image-3 {")
		assert.Contains(t, style, "radio-3:checked + .mj-carousel-content .mj-carousel-0000000000000000-thumbnail-3")

		// Arrows wrap around: image 1 shows next-2 and previous-3.
		assert.Contains(t, style, ".mj-carousel-next-2")
		assert.Contains(t, style, ".mj-carousel-previous-3")

		// No-input and Outlook fallbacks, including MJML's invalid media query.
		assert.Contains(t, style, ".mj-carousel noinput .mj-carousel-image-1 {")
		assert.Contains(t, style, "[owa] .mj-carousel-thumbnail {")
		assert.Contains(t, style, "@media screen yahoo {")
		assert.Contains(t, style, "radio-1:checked + *+ *+ .mj-carousel-content .mj-carousel-0000000000000000-thumbnail-1 {")
	})

	t.Run("an empty carousel has no stylesheet", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{Breakpoint: "480px", ContainerWidth: "600px"}
		carousel := newCarouselNode(nil)

		var component MJMLCarousel
		require.NoError(t, InitComponent(ctx, component, carousel))

		var out strings.Builder
		require.NoError(t, component.Render(ctx, &out, carousel))

		require.Len(t, ctx.InstanceComponentHeadStyles, 1)
		assert.Empty(t, ctx.InstanceComponentHeadStyles[0].Style("480px"))
		assert.Equal(t, `<!--[if !mso]><!--><div class="mj-carousel"><div class="mj-carousel-content mj-carousel-`+
			`0000000000000000-content" style="display:table;width:100%;table-layout:fixed;text-align:center;font-size:0px;">`+
			`<table style="caption-side:top;display:table-caption;table-layout:fixed;width:100%;" border="0" cellpadding="0" cellspacing="0" width="100%" role="presentation" class="mj-carousel-main"><tbody><tr>`+
			`<td class="mj-carousel-0000000000000000-icons-cell" style="font-size:0px;display:none;mso-hide:all;padding:0px;">`+
			`<div class="mj-carousel-previous-icons" style="display:none;mso-hide:all;"></div></td>`+
			`<td style="padding:0px;"><div class="mj-carousel-images"></div></td>`+
			`<td class="mj-carousel-0000000000000000-icons-cell" style="font-size:0px;display:none;mso-hide:all;padding:0px;">`+
			`<div class="mj-carousel-next-icons" style="display:none;mso-hide:all;"></div></td>`+
			`</tr></tbody></table></div></div><!--<![endif]-->`, out.String())
	})
}

func TestMJMLCarouselInvalidChildren(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{Breakpoint: "480px", ContainerWidth: "600px"}

	carousel := &node.Node{
		Type: CarouselTagName,
		Children: []*node.Node{
			{Type: TextTagName},
		},
	}

	var component MJMLCarousel
	require.NoError(t, InitComponent(ctx, component, carousel))

	err := component.Render(ctx, &strings.Builder{}, carousel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid child type mj-text in carousel")
}

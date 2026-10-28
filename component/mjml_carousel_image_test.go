package component

import (
	"encoding/xml"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCarouselImageNode(attributes ...xml.Attr) *node.Node {
	return &node.Node{Type: CarouselImageTagName, Attributes: attributes}
}

func TestMJMLCarouselImage(t *testing.T) {
	t.Parallel()

	t.Run("renders the slide with the carousel width", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{ContainerWidth: "600px"}
		image := newCarouselImageNode(
			xml.Attr{Name: xml.Name{Local: "src"}, Value: "https://example.com/one.png"},
			xml.Attr{Name: xml.Name{Local: "alt"}, Value: "First"},
			xml.Attr{Name: xml.Name{Local: "title"}, Value: "One"},
		)
		require.NoError(t, initCarouselImage(image))

		var component MJMLCarouselImage

		attributes := carouselAttributes{borderRadius: "8px", tbWidth: "90px"}

		assert.Equal(t, `<div class="mj-carousel-image mj-carousel-image-1 " style="">`+
			`<img alt="First" border="0" src="https://example.com/one.png" `+
			`style="border-radius:8px;display:block;width:600px;max-width:100%;height:auto;" `+
			`title="One" width="600" /></div>`,
			component.renderImage(ctx, image, 0, attributes))
	})

	t.Run("hides every slide but the first", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{ContainerWidth: "600px"}
		image := newCarouselImageNode(xml.Attr{Name: xml.Name{Local: "src"}, Value: "https://example.com/two.png"})
		require.NoError(t, initCarouselImage(image))

		var component MJMLCarouselImage

		rendered := component.renderImage(ctx, image, 1, carouselAttributes{borderRadius: "6px"})

		assert.Contains(t, rendered, `class="mj-carousel-image mj-carousel-image-2 " style="display:none;mso-hide:all;"`)
		assert.Contains(t, rendered, `style="border-radius:6px;display:block;width:600px;`)
	})

	t.Run("links the slide with a fixed target", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{ContainerWidth: "600px"}
		image := newCarouselImageNode(
			xml.Attr{Name: xml.Name{Local: "src"}, Value: "https://example.com/one.png"},
			xml.Attr{Name: xml.Name{Local: "href"}, Value: "https://example.com/one"},
			xml.Attr{Name: xml.Name{Local: "rel"}, Value: "noopener"},
			xml.Attr{Name: xml.Name{Local: "target"}, Value: "_self"},
		)
		require.NoError(t, initCarouselImage(image))

		var component MJMLCarouselImage

		// MJML always links the slide itself with target="_blank"; only the
		// thumbnail honours the target attribute.
		assert.Contains(t, component.renderImage(ctx, image, 0, carouselAttributes{}),
			`<a href="https://example.com/one" rel="noopener" target="_blank"><img`)
	})

	t.Run("own attributes win over the inherited ones", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{ContainerWidth: "600px"}
		image := newCarouselImageNode(
			xml.Attr{Name: xml.Name{Local: "src"}, Value: "https://example.com/one.png"},
			xml.Attr{Name: xml.Name{Local: "border-radius"}, Value: "15px"},
			xml.Attr{Name: xml.Name{Local: "tb-border"}, Value: "1px solid #000000"},
		)
		require.NoError(t, initCarouselImage(image))

		var component MJMLCarouselImage

		attributes := carouselAttributes{
			borderRadius:   "8px",
			tbBorder:       "2px solid transparent",
			tbBorderRadius: "6px",
			tbWidth:        "90px",
		}

		assert.Contains(t, component.renderImage(ctx, image, 0, attributes), `border-radius:15px;`)

		thumbnail := component.renderThumbnail(image, "0000000000000000", 0, "visible", attributes)
		assert.Contains(t, thumbnail, `border:1px solid #000000;`)
	})
}

func TestMJMLCarouselImageThumbnail(t *testing.T) {
	t.Parallel()

	attributes := carouselAttributes{
		tbBorder:       "2px solid transparent",
		tbBorderRadius: "6px",
		tbWidth:        "110px",
	}

	t.Run("renders the thumbnail link and image", func(t *testing.T) {
		t.Parallel()

		image := newCarouselImageNode(
			xml.Attr{Name: xml.Name{Local: "src"}, Value: "https://example.com/two.png"},
			xml.Attr{Name: xml.Name{Local: "alt"}, Value: "Second"},
		)
		require.NoError(t, initCarouselImage(image))

		var component MJMLCarouselImage

		assert.Equal(t, `<a class="mj-carousel-thumbnail mj-carousel-0000000000000000-thumbnail `+
			`mj-carousel-0000000000000000-thumbnail-2 " href="#2" `+
			`style="border:2px solid transparent;border-radius:6px;display:inline-block;overflow:hidden;width:110px;" `+
			`target="_blank"><label for="mj-carousel-0000000000000000-radio-2">`+
			`<img alt="Second" src="https://example.com/two.png" style="display:block;width:100%;height:auto;" width="110" />`+
			`</label></a>`, component.renderThumbnail(image, "0000000000000000", 1, "visible", attributes))
	})

	t.Run("prefers the thumbnails-src and suffixes css classes", func(t *testing.T) {
		t.Parallel()

		image := newCarouselImageNode(
			xml.Attr{Name: xml.Name{Local: "src"}, Value: "https://example.com/two.png"},
			xml.Attr{Name: xml.Name{Local: "thumbnails-src"}, Value: "https://example.com/two-thumb.png"},
			xml.Attr{Name: xml.Name{Local: "css-class"}, Value: "primary accent"},
			xml.Attr{Name: xml.Name{Local: "target"}, Value: "_self"},
		)
		require.NoError(t, initCarouselImage(image))

		var component MJMLCarouselImage

		rendered := component.renderThumbnail(image, "0000000000000000", 0, "visible", attributes)

		assert.Contains(t, rendered, `src="https://example.com/two-thumb.png"`)
		assert.Contains(t, rendered, `primary-thumbnail accent-thumbnail`)
		assert.Contains(t, rendered, `target="_self"`)
	})

	t.Run("supported mode hides the thumbnails inline", func(t *testing.T) {
		t.Parallel()

		image := newCarouselImageNode(xml.Attr{Name: xml.Name{Local: "src"}, Value: "https://example.com/one.png"})
		require.NoError(t, initCarouselImage(image))

		var component MJMLCarouselImage

		assert.Contains(t, component.renderThumbnail(image, "id", 0, "supported", attributes), `display:none;overflow:hidden;`)
		assert.Contains(t, component.renderThumbnail(image, "id", 0, "visible", attributes), `display:inline-block;overflow:hidden;`)
	})
}

func TestInitCarouselImage(t *testing.T) {
	t.Parallel()

	t.Run("applies the defaults", func(t *testing.T) {
		t.Parallel()

		image := newCarouselImageNode(xml.Attr{Name: xml.Name{Local: "src"}, Value: "https://example.com/one.png"})
		require.NoError(t, initCarouselImage(image))

		assert.Equal(t, "_blank", image.GetAttributeValueDefault("target"))
		assert.Equal(t, "", image.GetAttributeValueDefault("alt"))
	})

	t.Run("keeps explicit attributes", func(t *testing.T) {
		t.Parallel()

		image := newCarouselImageNode(
			xml.Attr{Name: xml.Name{Local: "target"}, Value: "_self"},
			xml.Attr{Name: xml.Name{Local: "alt"}, Value: "First"},
		)
		require.NoError(t, initCarouselImage(image))

		assert.Equal(t, "_self", image.GetAttributeValueDefault("target"))
		assert.Equal(t, "First", image.GetAttributeValueDefault("alt"))
	})

	t.Run("rejects invalid attributes", func(t *testing.T) {
		t.Parallel()

		image := newCarouselImageNode(xml.Attr{Name: xml.Name{Local: "border-radius"}, Value: "huge"})

		err := initCarouselImage(image)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to validate field border-radius in <mj-carousel-image>")
	})
}

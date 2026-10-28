package component

import (
	"strconv"

	"github.com/julez-dev/mjmlgo/node"
)

// MJMLCarouselImage renders mj-carousel-image, the only child of mj-carousel
// (node_modules/mjml-carousel/lib/CarouselImage.js). One image contributes a
// slide in the images table and an entry in the thumbnail strip; the hidden
// radio input in front of the content is rendered by mj-carousel.
//
// The image is never rendered on its own, so unlike the other components it does
// not implement Component: mj-carousel calls the parts and initCarouselImage
// applies the attribute defaults and validation.
type MJMLCarouselImage struct{}

func (i MJMLCarouselImage) Name() string {
	return "mj-carousel-image"
}

func (i MJMLCarouselImage) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"alt":              validateType("string"),
		"border-radius":    validateUnit([]string{"px", "%"}, true),
		"href":             validateType("string"),
		"rel":              validateType("string"),
		"src":              validateType("string"),
		"target":           validateType("string"),
		"tb-border":        validateType("string"),
		"tb-border-radius": validateUnit([]string{"px", "%"}, true),
		"thumbnails-src":   validateType("string"),
		"title":            validateType("string"),
	}
}

func (i MJMLCarouselImage) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"alt":    "",
		"target": "_blank",
	}
}

// carouselAttributes are the attributes mj-carousel passes down to its images.
// Like MJML's renderChildren, an attribute set on the image itself wins.
type carouselAttributes struct {
	borderRadius   string
	tbBorder       string
	tbBorderRadius string
	tbWidth        string
}

// inheritedAttribute returns the image's own attribute or the value inherited
// from the carousel.
func inheritedAttribute(n *node.Node, name, inherited string) string {
	if value, has := n.GetAttributeValue(name); has {
		return value
	}

	return inherited
}

// renderImage renders one slide: a div that is visible for the first image and
// hidden for the others, optionally wrapped in a link.
func (i MJMLCarouselImage) renderImage(ctx *RenderContext, n *node.Node, index int, attributes carouselAttributes) string {
	image := `<img ` + inlineAttributes{
		"title": n.GetAttributeValueDefault("title"),
		"src":   n.GetAttributeValueDefault("src"),
		"alt":   n.GetAttributeValueDefault("alt"),
		"style": inlineStyle{
			{Property: "border-radius", Value: inheritedAttribute(n, "border-radius", attributes.borderRadius)},
			{Property: "display", Value: "block"},
			{Property: "width", Value: ctx.ContainerWidth},
			{Property: "max-width", Value: "100%"},
			{Property: "height", Value: "auto"},
		}.InlineString(),
		"width":  strconv.Itoa(carouselIntValue(ctx.ContainerWidth)),
		"border": "0",
	}.InlineString() + ` />`

	if href := n.GetAttributeValueDefault("href"); href != "" {
		// MJML always links the slide with target="_blank", whatever the image's
		// own target says; only the thumbnail honours the attribute.
		image = `<a ` + inlineAttributes{
			"href":   href,
			"rel":    n.GetAttributeValueDefault("rel"),
			"target": "_blank",
		}.InlineString() + `>` + image + `</a>`
	}

	slideStyle := inlineStyle{}
	if index > 0 {
		slideStyle = inlineStyle{
			{Property: "display", Value: "none"},
			{Property: "mso-hide", Value: "all"},
		}
	}

	return `<div class="mj-carousel-image mj-carousel-image-` + strconv.Itoa(index+1) + ` ` +
		n.GetAttributeValueDefault("css-class") + `" style="` + slideStyle.InlineString() + `">` + image + `</div>`
}

// renderThumbnail renders one entry of the thumbnail strip. MJML hides the
// thumbnails case of the "supported" mode behind the display rule in the
// generated stylesheet, which is why the inline display differs between
// "visible" and "supported".
func (i MJMLCarouselImage) renderThumbnail(
	n *node.Node,
	carouselID string,
	index int,
	thumbnails string,
	attributes carouselAttributes,
) string {
	number := strconv.Itoa(index + 1)

	// MJML reads the mode from the image first and falls back to the carousel's
	// context. thumbnails is not an allowed attribute of mj-carousel-image, so
	// this only ever matters for input MJML itself would reject.
	if mode := n.GetAttributeValueDefault("thumbnails"); mode != "" {
		thumbnails = mode
	}

	display := "inline-block"
	if thumbnails == "supported" {
		display = "none"
	}

	src := n.GetAttributeValueDefault("src")
	if thumbnailsSrc := n.GetAttributeValueDefault("thumbnails-src"); thumbnailsSrc != "" {
		src = thumbnailsSrc
	}

	thumbnail := `<img ` + inlineAttributes{
		"style": inlineStyle{
			{Property: "display", Value: "block"},
			{Property: "width", Value: "100%"},
			{Property: "height", Value: "auto"},
		}.InlineString(),
		"src":   src,
		"alt":   n.GetAttributeValueDefault("alt"),
		"width": strconv.Itoa(carouselIntValue(attributes.tbWidth)),
	}.InlineString() + ` />`

	return `<a ` + inlineAttributes{
		"style": inlineStyle{
			{Property: "border", Value: inheritedAttribute(n, "tb-border", attributes.tbBorder)},
			{Property: "border-radius", Value: inheritedAttribute(n, "tb-border-radius", attributes.tbBorderRadius)},
			{Property: "display", Value: display},
			{Property: "overflow", Value: "hidden"},
			{Property: "width", Value: attributes.tbWidth},
		}.InlineString(),
		"href":   "#" + number,
		"target": n.GetAttributeValueDefault("target"),
		"class": "mj-carousel-thumbnail mj-carousel-" + carouselID + "-thumbnail mj-carousel-" + carouselID +
			"-thumbnail-" + number + " " + addSuffixToClasses(n.GetAttributeValueDefault("css-class"), "thumbnail"),
	}.InlineString() + `><label for="mj-carousel-` + carouselID + `-radio-` + number + `">` + thumbnail + `</label></a>`
}

// initCarouselImage applies the attribute defaults and validation of
// mj-carousel-image. The image is only ever rendered as part of mj-carousel, so
// it does not implement Component; this mirrors InitComponent.
func initCarouselImage(n *node.Node) error {
	image := MJMLCarouselImage{}

	if err := validateAttributes(image.AllowedAttributes(), n); err != nil {
		return err
	}

	for name, value := range image.DefaultAttributes(nil) {
		if _, has := n.GetAttributeValue(name); !has {
			n.SetAttribute(name, value)
		}
	}

	return nil
}

package component

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/julez-dev/mjmlgo/internal/node"
)

// MJMLCarousel renders mj-carousel. The markup and the generated
// `mj-carousel-<id>` stylesheet follow mjml-carousel 4.18.0
// (node_modules/mjml-carousel/lib/Carousel.js): a downlevel-revealed
// non-Outlook block holding one hidden radio per image, the optional
// thumbnails and the images table, plus an Outlook-only fallback that shows the
// first image.
//
// MJML gives every carousel a random 16-character hex id that ends up in ids,
// class names and the generated selectors. The id comes from the render's
// configured generator (see RenderContext.nextComponentID); the parity harness
// treats a 16-hex id as an opaque value.
type MJMLCarousel struct{}

func (c MJMLCarousel) Name() string {
	return "mj-carousel"
}

func (c MJMLCarousel) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"align":                      validateEnum([]string{"left", "center", "right"}),
		"border-radius":              validateUnit([]string{"px", "%"}, true),
		"container-background-color": validateColor(),
		"icon-width":                 validateUnit([]string{"px", "%"}, false),
		"left-icon":                  validateType("string"),
		"padding":                    validateUnit([]string{"px", "%"}, true),
		"padding-bottom":             validateUnit([]string{"px", "%"}, false),
		"padding-left":               validateUnit([]string{"px", "%"}, false),
		"padding-right":              validateUnit([]string{"px", "%"}, false),
		"padding-top":                validateUnit([]string{"px", "%"}, false),
		"right-icon":                 validateType("string"),
		"tb-border":                  validateType("string"),
		"tb-border-radius":           validateUnit([]string{"px", "%"}, false),
		"tb-hover-border-color":      validateColor(),
		"tb-selected-border-color":   validateColor(),
		"tb-width":                   validateUnit([]string{"px", "%"}, false),
		"thumbnails":                 validateEnum([]string{"visible", "hidden", "supported"}),
	}
}

func (c MJMLCarousel) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"align":                    "center",
		"border-radius":            "6px",
		"icon-width":               "44px",
		"left-icon":                "https://i.imgur.com/xTh3hln.png",
		"right-icon":               "https://i.imgur.com/os7o9kz.png",
		"thumbnails":               "visible",
		"tb-border":                "2px solid transparent",
		"tb-border-radius":         "6px",
		"tb-hover-border-color":    "#fead0d",
		"tb-selected-border-color": "#ccc",
	}
}

func (c MJMLCarousel) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	images := make([]*node.Node, 0, len(n.Children))

	for _, child := range n.Children {
		if child.Type != CarouselImageTagName {
			return fmt.Errorf("invalid child type %s in carousel, allowed types are: %v",
				child.Type, []string{CarouselImageTagName})
		}

		if err := initCarouselImage(child); err != nil {
			return err
		}

		images = append(images, child)
	}

	state := carousel{
		node:   n,
		id:     ctx.nextComponentID(),
		images: images,
	}

	ctx.addInstanceComponentHeadStyle(c.Name(), func(string) string {
		return state.headStyle()
	})

	_, _ = io.WriteString(w, msoConditionalTag(
		`<div class="mj-carousel">`+state.renderRadios()+state.renderContent(ctx)+`</div>`, true))
	_, _ = io.WriteString(w, state.renderFallback(ctx))

	return nil
}

// carousel is the resolved state of one mj-carousel render: the generated id,
// the carousel node and its image children in document order.
type carousel struct {
	node   *node.Node
	id     string
	images []*node.Node
}

// inherited returns the attributes that mj-carousel passes down to its image
// children. In MJML's renderChildren these are merged as
// {...attributes, ...child.attributes}, so an attribute set on the image itself
// wins.
func (c carousel) inherited() carouselAttributes {
	return carouselAttributes{
		borderRadius:   c.node.GetAttributeValueDefault("border-radius"),
		tbBorder:       c.node.GetAttributeValueDefault("tb-border"),
		tbBorderRadius: c.node.GetAttributeValueDefault("tb-border-radius"),
		tbWidth:        c.thumbnailsWidth(),
	}
}

// thumbnailsWidth resolves the thumbnail width like MJML's thumbnailsWidth().
//
// MJML computes Math.min([this.context.parentWidth / children.length, 110]) but
// nothing in MJML 4.18 ever puts parentWidth into the render context, so the
// expression always collapses to "110px" (lodash min ignores the NaN term).
// An explicit tb-width always wins.
func (c carousel) thumbnailsWidth() string {
	if width := c.node.GetAttributeValueDefault("tb-width"); width != "" {
		return width
	}

	return "110px"
}

// renderRadios renders the hidden radio inputs that drive the CSS-only carousel.
func (c carousel) renderRadios() string {
	var b strings.Builder

	for i := range c.images {
		number := strconv.Itoa(i + 1)

		attributes := inlineAttributes{
			"class": "mj-carousel-radio mj-carousel-" + c.id + "-radio mj-carousel-" + c.id + "-radio-" + number,
			"type":  "radio",
			"name":  "mj-carousel-radio-" + c.id,
			"id":    "mj-carousel-" + c.id + "-radio-" + number,
			"style": inlineStyle{
				{Property: "display", Value: "none"},
				{Property: "mso-hide", Value: "all"},
			}.InlineString(),
		}

		// MJML omits the attribute entirely for every radio but the first.
		if i == 0 {
			attributes["checked"] = "checked"
		}

		b.WriteString("<input " + attributes.InlineString() + " />")
	}

	return b.String()
}

// renderContent renders the carousel content div with the thumbnails and the
// images table.
func (c carousel) renderContent(ctx *RenderContext) string {
	var b strings.Builder

	b.WriteString(`<div class="mj-carousel-content mj-carousel-` + c.id + `-content" style="` + inlineStyle{
		{Property: "display", Value: "table"},
		{Property: "width", Value: "100%"},
		{Property: "table-layout", Value: "fixed"},
		{Property: "text-align", Value: "center"},
		{Property: "font-size", Value: "0px"},
	}.InlineString() + `">`)

	b.WriteString(c.renderThumbnails())
	b.WriteString(c.renderCarousel(ctx))
	b.WriteString(`</div>`)

	return b.String()
}

// renderThumbnails renders the thumbnail strip. MJML only emits it for the
// visible and supported modes; thumbnails="hidden" renders none at all.
func (c carousel) renderThumbnails() string {
	thumbnails := c.node.GetAttributeValueDefault("thumbnails")
	if thumbnails != "visible" && thumbnails != "supported" {
		return ""
	}

	var b strings.Builder

	for i, image := range c.images {
		var component MJMLCarouselImage

		b.WriteString(component.renderThumbnail(image, c.id, i, thumbnails, c.inherited()))
	}

	return b.String()
}

// renderCarousel renders the table with the previous/next controls and the
// images.
func (c carousel) renderCarousel(ctx *RenderContext) string {
	var b strings.Builder

	b.WriteString(`<table style="` + inlineStyle{
		{Property: "caption-side", Value: "top"},
		{Property: "display", Value: "table-caption"},
		{Property: "table-layout", Value: "fixed"},
		{Property: "width", Value: "100%"},
	}.InlineString() + `" border="0" cellpadding="0" cellspacing="0" width="100%" role="presentation" class="mj-carousel-main"><tbody><tr>`)
	b.WriteString(c.renderControls("previous", c.node.GetAttributeValueDefault("left-icon")))
	b.WriteString(`<td style="` + inlineStyle{{Property: "padding", Value: "0px"}}.InlineString() + `">`)
	b.WriteString(`<div class="mj-carousel-images">`)

	inherited := c.inherited()

	for i, image := range c.images {
		var component MJMLCarouselImage

		b.WriteString(component.renderImage(ctx, image, i, inherited))
	}

	b.WriteString(`</div></td>`)
	b.WriteString(c.renderControls("next", c.node.GetAttributeValueDefault("right-icon")))
	b.WriteString(`</tr></tbody></table>`)

	return b.String()
}

// renderControls renders one arrow cell: one label per image, so every radio has
// a "next" and a "previous" label that the generated CSS shows or hides.
func (c carousel) renderControls(direction, icon string) string {
	iconWidth := c.node.GetAttributeValueDefault("icon-width")

	style := inlineStyle{
		{Property: "font-size", Value: "0px"},
		{Property: "display", Value: "none"},
		{Property: "mso-hide", Value: "all"},
		{Property: "padding", Value: "0px"},
	}

	var b strings.Builder

	b.WriteString(`<td class="mj-carousel-` + c.id + `-icons-cell" style="` + style.InlineString() + `">`)
	b.WriteString(`<div class="mj-carousel-` + direction + `-icons" style="` + inlineStyle{
		{Property: "display", Value: "none"},
		{Property: "mso-hide", Value: "all"},
	}.InlineString() + `">`)

	for i := range c.images {
		number := strconv.Itoa(i + 1)

		b.WriteString(`<label for="mj-carousel-` + c.id + `-radio-` + number +
			`" class="mj-carousel-` + direction + ` mj-carousel-` + direction + `-` + number + `">`)
		b.WriteString(`<img ` + inlineAttributes{
			"src": icon,
			"alt": direction,
			"style": inlineStyle{
				{Property: "display", Value: "block"},
				{Property: "width", Value: iconWidth},
				{Property: "height", Value: "auto"},
			}.InlineString(),
			"width": strconv.Itoa(carouselIntValue(iconWidth)),
		}.InlineString() + ` />`)
		b.WriteString(`</label>`)
	}

	b.WriteString(`</div></td>`)

	return b.String()
}

// renderFallback renders the Outlook-only fallback: the first image, without any
// JavaScript or radio, so Outlook shows something.
func (c carousel) renderFallback(ctx *RenderContext) string {
	if len(c.images) == 0 {
		return ""
	}

	var component MJMLCarouselImage

	return msoConditionalTag(component.renderImage(ctx, c.images[0], 0, c.inherited()), false)
}

// headStyle builds the componentHeadStyle MJML generates for this carousel: the
// static rules, the per-image rules derived from the carousel id and the image
// count, and the no-input/Outlook fallback rules. It returns an empty string for
// a carousel without images, like MJML.
func (c carousel) headStyle() string {
	count := len(c.images)
	if count == 0 {
		return ""
	}

	iconWidth := c.node.GetAttributeValueDefault("icon-width")
	selectedBorderColor := expandHexShortcutDefault(c.node.GetAttributeValueDefault("tb-selected-border-color"))
	hoverBorderColor := expandHexShortcutDefault(c.node.GetAttributeValueDefault("tb-hover-border-color"))

	// Every rule below targets an element of the carousel content by position,
	// so the selectors repeat MJML's sibling combinators per image.
	hideImage := func(i int) string {
		// Any checked radio hides the images ...
		return ".mj-carousel-" + c.id + "-radio:checked " + carouselSiblings(i) + "+ .mj-carousel-content .mj-carousel-image"
	}
	checkedImage := func(i int) string {
		// ... and the checked radio shows its own image.
		return c.checkedSelector(count, i, ".mj-carousel-content .mj-carousel-image-"+strconv.Itoa(i+1))
	}
	nextArrow := func(i int) string {
		return c.checkedSelector(count, i, ".mj-carousel-content .mj-carousel-next-"+strconv.Itoa(carouselNextIndex(i, count)))
	}
	previousArrow := func(i int) string {
		return c.checkedSelector(count, i, ".mj-carousel-content .mj-carousel-previous-"+strconv.Itoa(carouselPreviousIndex(i, count)))
	}
	selectedThumbnail := func(i int) string {
		return c.checkedSelector(count, i, ".mj-carousel-content .mj-carousel-"+c.id+"-thumbnail-"+strconv.Itoa(i+1))
	}
	checkedThumbnail := func(i int) string {
		return c.checkedSelector(count, i, ".mj-carousel-content .mj-carousel-"+c.id+"-thumbnail")
	}
	hoveredImage := func(i int) string {
		return ".mj-carousel-" + c.id + "-thumbnail:hover " + carouselSiblings(count-i-1) +
			"+ .mj-carousel-main .mj-carousel-image"
	}
	hoveredThumbnail := func(i int) string {
		number := strconv.Itoa(i + 1)

		return ".mj-carousel-" + c.id + "-thumbnail-" + number + ":hover " + carouselSiblings(count-i-1) +
			"+ .mj-carousel-main .mj-carousel-image-" + number
	}

	var b strings.Builder

	b.WriteString("\n    .mj-carousel {\n" +
		"      -webkit-user-select: none;\n" +
		"      -moz-user-select: none;\n" +
		"      user-select: none;\n" +
		"    }\n\n")
	b.WriteString("    .mj-carousel-" + c.id + "-icons-cell {\n" +
		"      display: table-cell !important;\n" +
		"      width: " + iconWidth + " !important;\n" +
		"    }\n\n")
	b.WriteString("    .mj-carousel-radio,\n" +
		"    .mj-carousel-next,\n" +
		"    .mj-carousel-previous {\n" +
		"      display: none !important;\n" +
		"    }\n\n")
	b.WriteString("    .mj-carousel-thumbnail,\n" +
		"    .mj-carousel-next,\n" +
		"    .mj-carousel-previous {\n" +
		"      touch-action: manipulation;\n" +
		"    }\n\n")

	b.WriteString(carouselRule(carouselSelectors(count, hideImage), "display: none !important;"))
	b.WriteString(carouselRule(carouselSelectors(count, checkedImage), "display: block !important;"))

	// The checked image also shows its (wrapping) next and previous arrow.
	arrows := []string{".mj-carousel-previous-icons", ".mj-carousel-next-icons"}
	arrows = append(arrows, carouselSelectors(count, nextArrow)...)
	arrows = append(arrows, carouselSelectors(count, previousArrow)...)

	b.WriteString(carouselRule(arrows, "display: block !important;"))

	b.WriteString(carouselRule(carouselSelectors(count, selectedThumbnail),
		"border-color: "+selectedBorderColor+" !important;"))
	b.WriteString(carouselRule(carouselSelectors(count, checkedThumbnail), "display: inline-block !important;"))

	b.WriteString("    .mj-carousel-image img + div,\n" +
		"    .mj-carousel-thumbnail img + div {\n" +
		"      display: none !important;\n" +
		"    }\n\n")

	b.WriteString(carouselRule(carouselSelectors(count, hoveredImage), "display: none !important;"))
	b.WriteString("    .mj-carousel-thumbnail:hover {\n" +
		"      border-color: " + hoverBorderColor + " !important;\n" +
		"    }\n\n")
	b.WriteString(carouselRule(carouselSelectors(count, hoveredThumbnail), "display: block !important;"))

	// Fallback: clients without radio input support (noinput) keep showing the
	// first image, and Outlook drops the thumbnails entirely.
	b.WriteString("    .mj-carousel noinput {\n      display: block !important;\n    }\n\n" +
		"    .mj-carousel noinput .mj-carousel-image-1 {\n      display: block !important;\n    }\n\n" +
		"    .mj-carousel noinput .mj-carousel-arrows,\n" +
		"    .mj-carousel noinput .mj-carousel-thumbnails {\n      display: none !important;\n    }\n\n" +
		"    [owa] .mj-carousel-thumbnail {\n      display: none !important;\n    }\n\n")

	// MJML emits this media query verbatim, including the invalid "screen yahoo"
	// condition, which every client ignores.
	b.WriteString("    @media screen yahoo {\n" +
		"      .mj-carousel-" + c.id + "-icons-cell,\n" +
		"      .mj-carousel-previous-icons,\n" +
		"      .mj-carousel-next-icons {\n        display: none !important;\n      }\n\n" +
		"      .mj-carousel-" + c.id + "-radio-1:checked " + strings.Repeat("+ *", count-1) +
		"+ .mj-carousel-content .mj-carousel-" + c.id + "-thumbnail-1 {\n" +
		"        border-color: transparent;\n      }\n    }\n    ")

	return b.String()
}

// checkedSelector builds the selector for the element a checked radio reveals at
// image position i: the radio, the sibling combinators up to the content div and
// the target selector.
func (c carousel) checkedSelector(count, i int, target string) string {
	return ".mj-carousel-" + c.id + "-radio-" + strconv.Itoa(i+1) + ":checked " +
		carouselSiblings(count-i-1) + "+ " + target
}

// carouselRule renders one indented CSS rule whose selector list is written one
// selector per line.
func carouselRule(selectors []string, declarations string) string {
	return "    " + strings.Join(selectors, ",\n    ") + " {\n      " + declarations + "\n    }\n\n"
}

// carouselSelectors builds one selector per image.
func carouselSelectors(count int, build func(i int) string) []string {
	selectors := make([]string, count)

	for i := range count {
		selectors[i] = build(i)
	}

	return selectors
}

// carouselSiblings returns the sibling selectors MJML repeats for the n-th
// sibling. Callers append "+ <target>" themselves, like MJML's templates.
func carouselSiblings(n int) string {
	return strings.Repeat("+ * ", n)
}

// carouselNextIndex mirrors MJML's next-arrow index arithmetic for image i:
// ((i + 1 % count + count) % count) + 1.
func carouselNextIndex(i, count int) int {
	return (i+1%count+count)%count + 1
}

// carouselPreviousIndex mirrors MJML's previous-arrow index arithmetic for image
// i: ((i - 1 % count + count) % count) + 1.
func carouselPreviousIndex(i, count int) int {
	return (i-1%count+count)%count + 1
}

// carouselIntValue parses the integer prefix of a unit value like MJML's
// parseInt(value, 10): "44px" becomes 44 and "44.5px" also becomes 44.
func carouselIntValue(value string) int {
	end := 0

	if end < len(value) && (value[end] == '-' || value[end] == '+') {
		end++
	}

	start := end
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}

	if end == start {
		return 0
	}

	parsed, err := strconv.Atoi(value[:end])
	if err != nil {
		return 0
	}

	return parsed
}

// expandHexShortcutDefault expands a hex color the way MJML's color attribute
// type does (#ccc -> #cccccc) and returns the value unchanged when it is not a
// hex shortcut.
func expandHexShortcutDefault(value string) string {
	expanded, err := expandHexShortcut(value)
	if err != nil {
		return value
	}

	return expanded
}

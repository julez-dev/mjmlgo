package component

import (
	"fmt"
	"io"
	"slices"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLColumn struct{}

func (c MJMLColumn) Name() string {
	return "mj-column"
}

func (c MJMLColumn) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"background-color":       validateColor(),
		"border":                 validateType("string"),
		"border-bottom":          validateType("string"),
		"border-left":            validateType("string"),
		"border-radius":          validateUnit([]string{"px", "%"}, true),
		"border-right":           validateType("string"),
		"border-top":             validateType("string"),
		"direction":              validateEnum([]string{"ltr", "rtl"}),
		"inner-background-color": validateColor(),
		"padding-bottom":         validateUnit([]string{"px", "%"}, false),
		"padding-left":           validateUnit([]string{"px", "%"}, false),
		"padding-right":          validateUnit([]string{"px", "%"}, false),
		"padding-top":            validateUnit([]string{"px", "%"}, false),
		"padding":                validateUnit([]string{"px", "%"}, true),
		"inner-border":           validateType("string"),
		"inner-border-bottom":    validateType("string"),
		"inner-border-left":      validateType("string"),
		"inner-border-radius":    validateUnit([]string{"px", "%"}, true),
		"inner-border-right":     validateType("string"),
		"inner-border-top":       validateType("string"),
		"vertical-align":         validateEnum([]string{"top", "bottom", "middle"}),
		"width":                  validateUnit([]string{"px", "%"}, false),
	}
}

func (c MJMLColumn) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"direction":      "ltr",
		"vertical-align": "top",
	}
}

func (c MJMLColumn) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	className, err := getColumnClass(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get column class: %w", err)
	}

	className = className + " mj-outlook-group-fix"

	if v := n.GetAttributeValueDefault("css-class"); v != "" {
		className += " " + v
	}

	divAttr := inlineAttributes{
		"class": className,
	}

	divStyle := inlineStyle{
		{Property: "font-size", Value: "0px"},
		{Property: "text-align", Value: "left"},
		{Property: "direction", Value: n.GetAttributeValueDefault("direction")},
		{Property: "display", Value: "inline-block"},
		{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")},
	}

	divWidth, err := getMobileWidth(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get mobile width: %w", err)
	}

	divStyle = append(divStyle, Style{Property: "width", Value: divWidth})
	divAttr["style"] = divStyle.InlineString()

	childWidth, err := c.childContainerWidth(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get child container width: %w", err)
	}

	previousContainerWidth := ctx.ContainerWidth
	ctx.ContainerWidth = childWidth
	defer func() {
		ctx.ContainerWidth = previousContainerWidth
	}()

	_, _ = io.WriteString(w, "<div "+divAttr.InlineString()+">\n")
	if c.hasGutter(n) {
		if err := c.renderGutter(ctx, w, n); err != nil {
			return fmt.Errorf("failed to render column: %w", err)
		}
	} else {
		if err := c.renderColumn(ctx, w, n); err != nil {
			return fmt.Errorf("failed to render column: %w", err)
		}
	}

	_, _ = io.WriteString(w, "</div>\n")

	return nil
}

func (c MJMLColumn) renderColumn(ctx *RenderContext, w io.Writer, n *node.Node) error {
	tableAttr := inlineAttributes{
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
		"width":       "100%",
		"style":       c.tableStyle(n).InlineString(),
	}

	for _, child := range n.Children {
		if !slices.Contains(c.allowedChildren(), child.Type) {
			return fmt.Errorf("invalid child type %s in column, allowed types are: %v", child.Type, c.allowedChildren())
		}
	}

	_, _ = io.WriteString(w, "<table "+tableAttr.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tbody>\n")

	for _, child := range n.Children {
		if child.Type == RawTagName {
			var raw MJMLRaw
			if err := InitComponent(ctx, raw, child); err != nil {
				return err
			}
			// Mark the raw block so the CSS inliner can keep it at this position;
			// html.Parse would otherwise move non-table content out of <tbody>.
			_, _ = io.WriteString(w, RawBlockStartMarker)
			if err := raw.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render raw: %w", err)
			}
			_, _ = io.WriteString(w, RawBlockEndMarker)
			continue
		}

		_, _ = io.WriteString(w, "<tr>\n")
		switch child.Type {
		case DividerTagName:
			var divider MJMLDivider
			if err := InitComponent(ctx, divider, child); err != nil {
				return err
			}
			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := divider.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render divider: %w", err)
			}
		case SpacerTagName:
			var spacer MJMLSpacer
			if err := InitComponent(ctx, spacer, child); err != nil {
				return err
			}

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := spacer.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render spacer: %w", err)
			}
		case ImageTagName:
			var image MJMLImage
			if err := InitComponent(ctx, image, child); err != nil {
				return err
			}

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := image.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render image: %w", err)
			}
		case TextTagName:
			var text MJMLText
			if err := InitComponent(ctx, text, child); err != nil {
				return err
			}

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := text.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render text: %w", err)
			}
		case SocialTagName:
			var social MJMLSocial
			if err := InitComponent(ctx, social, child); err != nil {
				return err
			}

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := social.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render social: %w", err)
			}
		case TableTagName:
			var table MJMLTable
			if err := InitComponent(ctx, table, child); err != nil {
				return err
			}

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := table.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}
		case ButtonTagName:
			var button MJMLButton
			if err := InitComponent(ctx, button, child); err != nil {
				return err
			}

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := button.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render button: %w", err)
			}
		case NavbarTagName:
			var navbar MJMLNavbar
			if err := InitComponent(ctx, navbar, child); err != nil {
				return err
			}

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := navbar.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render navbar: %w", err)
			}
		case AccordionTagName:
			var accordion MJMLAccordion
			if err := InitComponent(ctx, accordion, child); err != nil {
				return err
			}

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := accordion.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render accordion: %w", err)
			}
		case CarouselTagName:
			var carousel MJMLCarousel
			if err := InitComponent(ctx, carousel, child); err != nil {
				return err
			}

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := carousel.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render carousel: %w", err)
			}
		}

		_, _ = io.WriteString(w, "</td>\n")
		_, _ = io.WriteString(w, "</tr>\n")
	}

	_, _ = io.WriteString(w, "</tbody>\n")
	_, _ = io.WriteString(w, "</table>\n")

	return nil
}

func (c MJMLColumn) renderGutter(ctx *RenderContext, w io.Writer, n *node.Node) error {
	tableAttr := inlineAttributes{
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
		"width":       "100%",
	}

	if c.hasBorderRadius(n) {
		tableAttr["style"] = inlineStyle{{Property: "border-collapse", Value: "separate"}}.InlineString()
	}

	tdStyle := inlineStyle{
		{Property: "padding", Value: n.GetAttributeValueDefault("padding")},
		{Property: "padding-top", Value: n.GetAttributeValueDefault("padding-top")},
		{Property: "padding-right", Value: n.GetAttributeValueDefault("padding-right")},
		{Property: "padding-bottom", Value: n.GetAttributeValueDefault("padding-bottom")},
		{Property: "padding-left", Value: n.GetAttributeValueDefault("padding-left")},
	}

	tdStyle = append(tdStyle, c.outerTableStyle(n)...)

	tdAttr := inlineAttributes{
		"style": tdStyle.InlineString(),
	}

	_, _ = io.WriteString(w, "<table "+tableAttr.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tbody>\n")
	_, _ = io.WriteString(w, "<tr>\n")
	_, _ = io.WriteString(w, "<td "+tdAttr.InlineString()+">\n")
	if err := c.renderColumn(ctx, w, n); err != nil {
		return err
	}
	_, _ = io.WriteString(w, "</td>\n")
	_, _ = io.WriteString(w, "</tr>\n")
	_, _ = io.WriteString(w, "</tbody>\n")
	_, _ = io.WriteString(w, "</table>\n")

	return nil
}

// childContainerWidth implements mj-column's getChildContext: children are
// rendered at the column width minus its paddings, borders and inner borders.
func (c MJMLColumn) childContainerWidth(ctx *RenderContext, n *node.Node) (string, error) {
	parentWidth, err := parseFloatWidth(ctx.ContainerWidth)
	if err != nil {
		return "", err
	}

	unit, width, err := getParsedWidth(n)
	if err != nil {
		return "", err
	}

	var containerWidth float64
	if unit == "%" {
		containerWidth = parentWidth * width / 100
	} else {
		containerWidth = width
	}

	paddingLeft, err := getShorthandAttrValue(n, "padding", "left")
	if err != nil {
		return "", err
	}

	paddingRight, err := getShorthandAttrValue(n, "padding", "right")
	if err != nil {
		return "", err
	}

	borderLeft := getShorthandBorderValue(n, "left", "border")
	borderRight := getShorthandBorderValue(n, "right", "border")
	innerBorderLeft := getShorthandBorderValue(n, "left", "inner-border")
	innerBorderRight := getShorthandBorderValue(n, "right", "inner-border")

	containerWidth -= float64(paddingLeft + paddingRight + borderLeft + borderRight + innerBorderLeft + innerBorderRight)

	return formatPixelWidth(containerWidth), nil
}

func (c MJMLColumn) tdAttribute(n *node.Node) inlineAttributes {
	attrs := inlineAttributes{
		"align": n.GetAttributeValueDefault("align"),
		"class": n.GetAttributeValueDefault("css-class"),
		"style": inlineStyle{
			{Property: "background", Value: n.GetAttributeValueDefault("container-background-color")},
			{Property: "font-size", Value: "0px"},
			{Property: "padding", Value: n.GetAttributeValueDefault("padding")},
			{Property: "padding-top", Value: n.GetAttributeValueDefault("padding-top")},
			{Property: "padding-right", Value: n.GetAttributeValueDefault("padding-right")},
			{Property: "padding-bottom", Value: n.GetAttributeValueDefault("padding-bottom")},
			{Property: "padding-left", Value: n.GetAttributeValueDefault("padding-left")},
			{Property: "word-break", Value: "break-word"},
		}.InlineString(),
	}

	return attrs
}

func (c MJMLColumn) tableStyle(n *node.Node) inlineStyle {
	if !c.hasGutter(n) {
		return c.outerTableStyle(n)
	}

	style := inlineStyle{
		{Property: "background-color", Value: n.GetAttributeValueDefault("inner-background-color")},
		{Property: "border", Value: n.GetAttributeValueDefault("inner-border")},
		{Property: "border-bottom", Value: n.GetAttributeValueDefault("inner-border-bottom")},
		{Property: "border-left", Value: n.GetAttributeValueDefault("inner-border-left")},
		{Property: "border-radius", Value: n.GetAttributeValueDefault("inner-border-radius")},
		{Property: "border-right", Value: n.GetAttributeValueDefault("inner-border-right")},
		{Property: "border-top", Value: n.GetAttributeValueDefault("inner-border-top")},
	}

	if c.hasInnerBorderRadius(n) {
		style = append(style, Style{Property: "border-collapse", Value: "separate"})
	}

	return style
}

// outerTableStyle is MJML's normal column table style. The gutter cell uses it
// too, so the column's own background and border survive the gutter wrapper.
func (c MJMLColumn) outerTableStyle(n *node.Node) inlineStyle {
	style := inlineStyle{
		{Property: "background-color", Value: n.GetAttributeValueDefault("background-color")},
		{Property: "border", Value: n.GetAttributeValueDefault("border")},
		{Property: "border-bottom", Value: n.GetAttributeValueDefault("border-bottom")},
		{Property: "border-left", Value: n.GetAttributeValueDefault("border-left")},
		{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
		{Property: "border-right", Value: n.GetAttributeValueDefault("border-right")},
		{Property: "border-top", Value: n.GetAttributeValueDefault("border-top")},
		{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")},
	}

	if c.hasBorderRadius(n) {
		style = append(style, Style{Property: "border-collapse", Value: "separate"})
	}

	return style
}

func (c MJMLColumn) hasBorderRadius(n *node.Node) bool {
	value, has := n.GetAttributeValue("border-radius")

	return has && value != ""
}

func (c MJMLColumn) hasInnerBorderRadius(n *node.Node) bool {
	value, has := n.GetAttributeValue("inner-border-radius")

	return has && value != ""
}

func (c MJMLColumn) hasGutter(n *node.Node) bool {
	var attrs = [...]string{
		"padding",
		"padding-top",
		"padding-right",
		"padding-bottom",
		"padding-left",
	}

	for _, attr := range attrs {
		_, has := n.GetAttributeValue(attr)
		if has {
			return true
		}
	}

	return false
}

func (c MJMLColumn) allowedChildren() []string {
	return []string{DividerTagName, SpacerTagName, ImageTagName,
		TextTagName, SocialTagName, TableTagName, ButtonTagName,
		NavbarTagName, AccordionTagName, CarouselTagName, RawTagName,
	}
}

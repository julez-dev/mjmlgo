package component

import (
	"fmt"
	"io"
	"slices"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLColumn struct{}

func (c MJMLColumn) allowedChildren() []string {
	return []string{
		SpacerTagName,
		ImageTagName,
		TextTagName,
		SocialTagName,
		DividerTagName,
		TableTagName,
	}
}

func (c MJMLColumn) applyDefaults(ctx *RenderContext, n *node.Node) {
	defaults := map[string]string{
		"vertical-align": "top",
		"direction":      ctx.Direction,
	}

	for key, value := range defaults {
		if _, ok := n.GetAttributeValue(key); !ok {
			n.SetAttribute(key, value)
		}
	}
}

func (c MJMLColumn) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	c.applyDefaults(ctx, n)

	className, err := getColumnClass(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get column class: %w", err)
	}

	className = className + " mj-outlook-group-fix"

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

	_, _ = io.WriteString(w, "<table "+tableAttr.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tbody>\n")

	for _, child := range n.Children {
		if !slices.Contains(c.allowedChildren(), child.Type) {
			return fmt.Errorf("invalid child type %s in column, allowed types are: %v", child.Type, c.allowedChildren())
		}

		_, _ = io.WriteString(w, "<tr>\n")
		switch child.Type {
		case DividerTagName:
			var divider MJMLDivider
			divider.applyDefaults(child)

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := divider.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render divider: %w", err)
			}
		case SpacerTagName:
			var spacer MJMLSpacer
			spacer.applyDefaults(child)

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := spacer.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render spacer: %w", err)
			}
		case ImageTagName:
			var image MJMLImage
			image.applyDefaults(child)

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := image.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render image: %w", err)
			}
		case TextTagName:
			var text MJMLText
			text.applyDefaults(child)

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := text.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render text: %w", err)
			}
		case SocialTagName:
			var social MJMLSocial
			social.applyDefaults(child)

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := social.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render social: %w", err)
			}
		case TableTagName:
			var table MJMLTable
			table.applyDefaults(child)

			_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
			if err := table.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
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

	tdStyle := inlineStyle{
		{Property: "padding", Value: n.GetAttributeValueDefault("padding")},
		{Property: "padding-top", Value: n.GetAttributeValueDefault("padding-top")},
		{Property: "padding-right", Value: n.GetAttributeValueDefault("padding-right")},
		{Property: "padding-bottom", Value: n.GetAttributeValueDefault("padding-bottom")},
		{Property: "padding-left", Value: n.GetAttributeValueDefault("padding-left")},
	}

	tdStyle = append(tdStyle, c.tableStyle(n)...)

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
	style := inlineStyle{}
	if c.hasGutter(n) {
		style = append(style, Style{Property: "background-color", Value: n.GetAttributeValueDefault("inner-background-color")})
		style = append(style, Style{Property: "border", Value: n.GetAttributeValueDefault("inner-border")})
		style = append(style, Style{Property: "border-bottom", Value: n.GetAttributeValueDefault("inner-border-bottom")})
		style = append(style, Style{Property: "border-left", Value: n.GetAttributeValueDefault("inner-border-left")})
		style = append(style, Style{Property: "border-radius", Value: n.GetAttributeValueDefault("inner-border-radius")})
		style = append(style, Style{Property: "border-right", Value: n.GetAttributeValueDefault("inner-border-right")})
		style = append(style, Style{Property: "border-top", Value: n.GetAttributeValueDefault("inner-border-top")})
	} else {
		style = append(style, Style{Property: "background-color", Value: n.GetAttributeValueDefault("background-color")})
		style = append(style, Style{Property: "border", Value: n.GetAttributeValueDefault("border")})
		style = append(style, Style{Property: "border-bottom", Value: n.GetAttributeValueDefault("border-bottom")})
		style = append(style, Style{Property: "border-left", Value: n.GetAttributeValueDefault("border-left")})
		style = append(style, Style{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")})
		style = append(style, Style{Property: "border-right", Value: n.GetAttributeValueDefault("border-right")})
		style = append(style, Style{Property: "border-top", Value: n.GetAttributeValueDefault("border-top")})
		style = append(style, Style{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")})
	}

	return style
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

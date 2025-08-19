package component

import (
	"fmt"
	"io"
	"strconv"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLGroup struct{}

func (g MJMLGroup) Name() string {
	return "mj-group"
}

func (g MJMLGroup) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"background-color": validateColor(),
		"direction":        validateEnum([]string{"ltr", "rtl"}),
		"vertical-align":   validateEnum([]string{"top", "bottom", "middle"}),
		"width":            validateUnit([]string{"px", "%"}, false),
	}
}

func (g MJMLGroup) DefaultAttributes(ctx *RenderContext) map[string]string {
	return map[string]string{
		"direction": ctx.Direction,
	}
}

func (g MJMLGroup) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	sibs := nonRawSiblings(n)
	parentWidth, err := strconv.Atoi(removeNonNumeric(ctx.ContainerWidth))
	if err != nil {
		return err
	}

	left, err := getShorthandAttrValue(n, "padding", "left")
	if err != nil {
		return err
	}

	right, err := getShorthandAttrValue(n, "padding", "right")
	if err != nil {
		return err
	}

	paddingSize := left + right

	var containerWidth string
	if v, has := n.GetAttributeValue("width"); has {
		containerWidth = v
	} else {
		containerWidth = fmt.Sprintf("%dpx", parentWidth/len(sibs))
	}

	parsedWidth, unit, err := parseWidth(containerWidth)
	if err != nil {
		return err
	}

	if unit == "%" {
		containerWidth = fmt.Sprintf("%dpx", (parentWidth*int(parsedWidth))/100-paddingSize)
	} else {
		containerWidth = fmt.Sprintf("%dpx", int(parsedWidth)-paddingSize)
	}

	groupWidth := containerWidth
	containerWidth = ctx.ContainerWidth

	getElementWidth := func(width string) (string, error) {
		if width == "" {
			parsedContainerWidth, err := strconv.Atoi(removeNonNumeric(containerWidth))
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("%dpx", parsedContainerWidth/len(sibs)), nil
		}

		parsedWidth, unit, err := parseWidth(width)
		if err != nil {
			return "", err
		}

		if unit == "%" {
			parsedGroupWidth, err := strconv.Atoi(removeNonNumeric(groupWidth))
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("%dpx", (100*int(parsedWidth))/parsedGroupWidth), nil
		}

		return fmt.Sprintf("%d%s", int(parsedWidth), unit), nil
	}

	columnClass, err := getColumnClass(ctx, n)
	if err != nil {
		return err
	}

	classesName := fmt.Sprintf("%s mj-outlook-group-fix", columnClass)

	if val, has := n.GetAttributeValue("css-class"); has {
		classesName += " " + val
	}

	styles, err := g.getStyles(ctx, n)
	if err != nil {
		return err
	}

	_, _ = io.WriteString(w, "<div "+inlineAttributes{
		"class": classesName,
		"style": styles["div"].InlineString(),
	}.InlineString()+">\n")

	tableAttr := inlineAttributes{
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
	}

	if val := n.GetAttributeValueDefault("background-color"); val != "" {
		tableAttr["bgcolor"] = val
	}

	_, _ = io.WriteString(w, "<!--[if mso | IE]><table "+tableAttr.InlineString()+"><tr><![endif]-->")

	for _, child := range n.Children {
		if child.Type != ColumnTagName {
			continue
		}

		child.SetAttribute("mobileWidth", "mobileWidth")

		childWidthAsPixel, err := getWidthAsPixel(ctx, child)
		if err != nil {
			return err
		}

		tdWidth, err := getElementWidth(childWidthAsPixel)
		if err != nil {
			return err
		}

		tdStyle := inlineStyle{
			{Property: "align", Value: child.GetAttributeValueDefault("align")},
			{Property: "vertical-align", Value: child.GetAttributeValueDefault("vertical-align")},
			{Property: "width", Value: tdWidth},
		}

		tdAttr := inlineAttributes{
			"style": tdStyle.InlineString(),
		}

		_, _ = io.WriteString(w, "<!--[if mso | IE]><td "+tdAttr.InlineString()+"><![endif]-->\n")

		var column MJMLColumn
		if err := InitComponent(ctx, column, child); err != nil {
			return err
		}
		if err := column.Render(ctx, w, child); err != nil {
			return err
		}

		_, _ = io.WriteString(w, "\n<!--[if mso | IE]></td><![endif]-->")
	}

	_, _ = io.WriteString(w, "<!--[if mso | IE]></tr></table><![endif]-->\n</div>")
	return nil
}

func (g MJMLGroup) getStyles(ctx *RenderContext, n *node.Node) (map[string]inlineStyle, error) {
	div := inlineStyle{
		{Property: "font-size", Value: "0"},
		{Property: "line-height", Value: "0"},
		{Property: "text-align", Value: "left"},
		{Property: "display", Value: "inline-block"},
		{Property: "width", Value: "100%"},
		{Property: "direction", Value: n.GetAttributeValueDefault("direction")},
		{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")},
		{Property: "background-color", Value: n.GetAttributeValueDefault("background-color")},
	}

	width, err := getWidthAsPixel(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("could not get width as pixel: %w", err)
	}

	tdOutlook := inlineStyle{
		{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")},
		{Property: "width", Value: width},
	}

	return map[string]inlineStyle{
		"div":       div,
		"tdOutlook": tdOutlook,
	}, nil
}

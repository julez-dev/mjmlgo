package component

import (
	"fmt"
	"io"
	"strconv"

	"github.com/julez-dev/mjmlgo/v2/node"
)

type MJMLGroup struct{}

func (g MJMLGroup) Name() string {
	return "mj-group"
}

var groupAllowedAttributes = map[string]AttributeValidator{
	"background-color": validateColor(),
	"direction":        validateEnum([]string{"ltr", "rtl"}),
	"vertical-align":   validateEnum([]string{"top", "bottom", "middle"}),
	"width":            validateUnit([]string{"px", "%"}, false),
}

func (g MJMLGroup) AllowedAttributes() map[string]AttributeValidator {
	return groupAllowedAttributes
}

func (g MJMLGroup) DefaultAttributes(ctx *RenderContext) map[string]string {
	return map[string]string{
		"direction": "ltr",
		//"vertical-align": "top",
	}
}

func (g MJMLGroup) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	sibs := nonRawSiblings(n)
	parentWidth, err := strconv.Atoi(RemoveNonNumeric(ctx.ContainerWidth))
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

	parsedWidth, unit, err := parseWidthFloat(containerWidth)
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
			parsedContainerWidth, err := strconv.Atoi(RemoveNonNumeric(containerWidth))
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("%dpx", parsedContainerWidth/len(sibs)), nil
		}

		parsedWidth, unit, err := parseWidthFloat(width)
		if err != nil {
			return "", err
		}

		if unit == "%" {
			parsedGroupWidth, err := strconv.Atoi(RemoveNonNumeric(groupWidth))
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("%dpx", (100*int(parsedWidth))/parsedGroupWidth), nil
		}

		return fmt.Sprintf("%d%s", int(parsedWidth), unit), nil
	}

	columnClass, err := g.columnClass(ctx, n)
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

	previousContainerWidth := ctx.ContainerWidth
	previousInGroup := ctx.InGroup
	previousDirection := ctx.ContainerDirection
	ctx.ContainerWidth = groupWidth
	ctx.InGroup = true
	ctx.ContainerDirection = n.GetAttributeValueDefault("direction")
	defer func() {
		ctx.ContainerWidth = previousContainerWidth
		ctx.InGroup = previousInGroup
		ctx.ContainerDirection = previousDirection
	}()

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
		if child.Type == RawTagName {
			if err := renderRawChild(ctx, w, child); err != nil {
				return err
			}

			continue
		}

		if child.Type != ColumnTagName {
			continue
		}

		child.SetAttribute("mobileWidth", "mobileWidth")

		var column MJMLColumn
		if err := InitComponent(ctx, column, child); err != nil {
			return err
		}

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

// columnClass is mj-group's own width class. Unlike a column's, it truncates the
// width in the class name while the rule keeps the exact one, which is what the
// reference implementation emits for a group that is not a whole number wide.
func (g MJMLGroup) columnClass(ctx *RenderContext, n *node.Node) (string, error) {
	unit, parsedWidth, err := getParsedWidth(n)
	if err != nil {
		return "", err
	}

	className := fmt.Sprintf("mj-column-px-%d", int(parsedWidth))
	if unit == "%" {
		className = fmt.Sprintf("mj-column-per-%d", int(parsedWidth))
	}

	ctx.AddMediaQuery(className, []string{
		fmt.Sprintf("width: %s%s !important", formatNumber(parsedWidth), unit),
		fmt.Sprintf("max-width: %s%s", formatNumber(parsedWidth), unit),
	})

	return className, nil
}

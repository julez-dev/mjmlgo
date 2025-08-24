package component

import (
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLHero struct{}

func (h MJMLHero) Name() string {
	return "mj-hero"
}

func (h MJMLHero) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"mode":                       validateEnum([]string{"fluid-height", "fixed-height"}),
		"height":                     validateUnit([]string{"px", "%"}, false),
		"background-url":             validateType("string"),
		"background-width":           validateUnit([]string{"px", "%"}, false),
		"background-height":          validateUnit([]string{"px", "%"}, false),
		"background-position":        validateType("string"),
		"border-radius":              validateUnit([]string{"px", "%"}, true),
		"container-background-color": validateColor(),
		"inner-background-color":     validateColor(),
		"inner-padding":              validateUnit([]string{"px", "%"}, true),
		"inner-padding-top":          validateUnit([]string{"px", "%"}, false),
		"inner-padding-bottom":       validateUnit([]string{"px", "%"}, false),
		"inner-padding-left":         validateUnit([]string{"px", "%"}, false),
		"inner-padding-right":        validateUnit([]string{"px", "%"}, false),
		"padding":                    validateUnit([]string{"px", "%"}, true),
		"padding-top":                validateUnit([]string{"px", "%"}, false),
		"padding-bottom":             validateUnit([]string{"px", "%"}, false),
		"padding-left":               validateUnit([]string{"px", "%"}, false),
		"padding-right":              validateUnit([]string{"px", "%"}, false),
		"background-color":           validateColor(),
		"vertical-align":             validateEnum([]string{"top", "middle", "bottom"}),
		"width":                      validateUnit([]string{"px", "%"}, false),
	}
}

func (h MJMLHero) DefaultAttributes(ctx *RenderContext) map[string]string {
	return map[string]string{
		"mode":                "fluid-height",
		"height":              "0px",
		"background-position": "center center",
		"padding":             "0px",
		"background-color":    "#ffffff",
		"vertical-align":      "top",
	}
}

func (h MJMLHero) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	containerWidth := ctx.ContainerWidth

	styles, err := h.getStyles(ctx, n)
	if err != nil {
		return err
	}

	_, _ = io.WriteString(w, "<!--[if mso | IE]>\n")
	_, _ = io.WriteString(w, "<table "+inlineAttributes{
		"align":       "center",
		"border":      "0",
		"cellpadding": "0",
		"role":        "presentation",
		"style":       styles["outlook-table"].InlineString(),
		"width":       RemoveNonNumeric(containerWidth),
	}.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tr>\n")
	_, _ = io.WriteString(w, "<td "+inlineAttributes{"style": styles["outlook-td"].InlineString()}.InlineString()+">\n")
	_, _ = io.WriteString(w, "<v:image "+inlineAttributes{
		"style":   styles["outlook-image"].InlineString(),
		"src":     n.GetAttributeValueDefault("background-url"),
		"xmlns:v": "urn:schemas-microsoft-com:vml",
	}.InlineString()+" />\n")
	_, _ = io.WriteString(w, "<![endif]-->\n")

	_, _ = io.WriteString(w, "<div "+inlineAttributes{
		"align": n.GetAttributeValueDefault("align"),
		"class": n.GetAttributeValueDefault("css-class"),
		"style": styles["div"].InlineString(),
	}.InlineString()+">\n")
	_, _ = io.WriteString(w, "<table "+inlineAttributes{
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
		"style":       styles["table"].InlineString(),
	}.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tbody>\n")
	_, _ = io.WriteString(w, "<tr "+inlineAttributes{"style": styles["tr"].InlineString()}.InlineString()+">\n")
	if err := h.renderMode(ctx, w, n); err != nil {
		return err
	}
	_, _ = io.WriteString(w, "</tr>\n</tbody>\n</table>\n</div>\n")
	_, _ = io.WriteString(w, "<!--[if mso | IE]>\n</td>\n</tr>\n</table>\n<![endif]-->\n")
	return nil
}

func (h MJMLHero) renderMode(ctx *RenderContext, w io.Writer, n *node.Node) error {
	styles, err := h.getStyles(ctx, n)
	if err != nil {
		return err
	}

	if n.GetAttributeValueDefault("mode") == "fluid-height" {
		_, _ = io.WriteString(w, "<td "+inlineAttributes{"style": styles["td-fluid"].InlineString()}.InlineString()+" />\n")
		_, _ = io.WriteString(w, "<td "+inlineAttributes{
			"background": n.GetAttributeValueDefault("background-url"),
			"style": inlineStyle{
				{Property: "background", Value: h.getBackground(n)},
				{Property: "background-position", Value: n.GetAttributeValueDefault("background-position")},
				{Property: "background-repeat", Value: "no-repeat"},
				{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
				{Property: "padding", Value: n.GetAttributeValueDefault("padding")},
				{Property: "padding-bottom", Value: n.GetAttributeValueDefault("padding-bottom")},
				{Property: "padding-left", Value: n.GetAttributeValueDefault("padding-left")},
				{Property: "padding-right", Value: n.GetAttributeValueDefault("padding-right")},
				{Property: "padding-top", Value: n.GetAttributeValueDefault("padding-top")},
				{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")},
			}.InlineString(),
		}.InlineString()+">\n")
		if err := h.renderContent(ctx, w, n); err != nil {
			return err
		}
		_, _ = io.WriteString(w, "</td>\n")
		_, _ = io.WriteString(w, "<td "+inlineAttributes{"style": styles["td-fluid"].InlineString()}.InlineString()+" />\n")
		return nil
	}

	paddingTop, err := getShorthandAttrValue(n, "padding", "top")
	if err != nil {
		return err
	}

	paddingBottom, err := getShorthandAttrValue(n, "padding", "bottom")
	if err != nil {
		return err
	}

	heightInt, err := strconv.Atoi(RemoveNonNumeric(n.GetAttributeValueDefault("height")))
	if err != nil {
		return err
	}

	height := heightInt - paddingTop - paddingBottom

	_, _ = io.WriteString(w, "<td "+inlineAttributes{
		"background": n.GetAttributeValueDefault("background-url"),
		"height":     fmt.Sprintf("%d", height),
		"style": inlineStyle{
			{Property: "background", Value: h.getBackground(n)},
			{Property: "background-position", Value: n.GetAttributeValueDefault("background-position")},
			{Property: "background-repeat", Value: "no-repeat"},
			{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
			{Property: "padding", Value: n.GetAttributeValueDefault("padding")},
			{Property: "padding-bottom", Value: n.GetAttributeValueDefault("padding-bottom")},
			{Property: "padding-left", Value: n.GetAttributeValueDefault("padding-left")},
			{Property: "padding-right", Value: n.GetAttributeValueDefault("padding-right")},
			{Property: "padding-top", Value: n.GetAttributeValueDefault("padding-top")},
			{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")},
			{Property: "height", Value: fmt.Sprintf("%dpx", height)},
		}.InlineString(),
	}.InlineString()+">\n")
	if err := h.renderContent(ctx, w, n); err != nil {
		return err
	}
	_, _ = io.WriteString(w, "</td>\n")

	return nil
}

func (h MJMLHero) renderContent(ctx *RenderContext, w io.Writer, n *node.Node) error {
	containerWidth := ctx.ContainerWidth

	styles, err := h.getStyles(ctx, n)
	if err != nil {
		return err
	}

	_, _ = io.WriteString(w, "<!--[if mso | IE]>\n")
	_, _ = io.WriteString(w, "<table "+inlineAttributes{
		"align":       n.GetAttributeValueDefault("align"),
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"style":       styles["outlook-inner-table"].InlineString(),
		"width":       RemoveNonNumeric(containerWidth),
	}.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tr>\n")
	_, _ = io.WriteString(w, "<td "+inlineAttributes{"style": styles["outlook-inner-td"].InlineString()}.InlineString()+">\n")
	_, _ = io.WriteString(w, "<![endif]-->\n")

	_, _ = io.WriteString(w, "<div "+inlineAttributes{
		"align": n.GetAttributeValueDefault("align"),
		"class": "mj-hero-content",
		"style": styles["inner-div"].InlineString(),
	}.InlineString()+">\n")
	_, _ = io.WriteString(w, "<table "+inlineAttributes{
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
		"style":       styles["inner-table"].InlineString(),
	}.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tbody>\n")
	_, _ = io.WriteString(w, "<tr>\n")
	_, _ = io.WriteString(w, "<td "+inlineAttributes{"style": styles["inner-td"].InlineString()}.InlineString()+">\n")
	_, _ = io.WriteString(w, "<table "+inlineAttributes{
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
		"style":       styles["inner-table"].InlineString(),
	}.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tbody>\n")

	paddingLeft, err := getShorthandAttrValue(n, "padding", "left")
	if err != nil {
		return err
	}

	paddingRight, err := getShorthandAttrValue(n, "padding", "right")
	if err != nil {
		return err
	}

	paddingSize := paddingLeft + paddingRight
	parsedWidth, unit, err := parseWidth(RemoveNonNumeric(containerWidth))
	if err != nil {
		return err
	}

	var currentContainerWidth string

	if unit == "%" {
		i, err := strconv.Atoi(RemoveNonNumeric(containerWidth))
		if err != nil {
			return err
		}
		currentContainerWidth = fmt.Sprintf("%dpx", (i*int(parsedWidth))/100-paddingSize)
	} else {
		currentContainerWidth = fmt.Sprintf("%dpx", int(parsedWidth)-paddingSize)
	}

	defer func(value string) {
		ctx.ContainerWidth = value
	}(ctx.ContainerWidth)

	ctx.ContainerWidth = currentContainerWidth

	for _, child := range n.Children {
		var childComponent Component

		switch child.Type {
		case TextTagName:
			childComponent = MJMLText{}
		case ImageTagName:
			childComponent = MJMLImage{}
		case ButtonTagName:
			childComponent = MJMLButton{}
		case DividerTagName:
			childComponent = MJMLDivider{}
		case SpacerTagName:
			childComponent = MJMLSpacer{}
		case SocialTagName:
			childComponent = MJMLSocial{}
		case TableTagName:
			childComponent = MJMLTable{}
		case RawTagName:
			childComponent = MJMLRaw{}
		}

		if childComponent == nil {
			return fmt.Errorf("unknown child type: %s for <mj-hero>", child.Type)
		}

		if err := InitComponent(ctx, childComponent, child); err != nil {
			return err
		}

		_, _ = io.WriteString(w, "<tr>\n")
		_, _ = io.WriteString(w, "<td "+inlineAttributes{
			"align":      child.GetAttributeValueDefault("align"),
			"background": child.GetAttributeValueDefault("container-background-color"),
			"class":      child.GetAttributeValueDefault("css-class"),
			"style": inlineStyle{
				{Property: "background", Value: child.GetAttributeValueDefault("container-background-color")},
				{Property: "font-size", Value: "0px"},
				{Property: "padding", Value: child.GetAttributeValueDefault("padding")},
				{Property: "padding-top", Value: child.GetAttributeValueDefault("padding-top")},
				{Property: "padding-right", Value: child.GetAttributeValueDefault("padding-right")},
				{Property: "padding-bottom", Value: child.GetAttributeValueDefault("padding-bottom")},
				{Property: "padding-left", Value: child.GetAttributeValueDefault("padding-left")},
				{Property: "word-break", Value: "break-word"},
			}.InlineString(),
		}.InlineString()+">\n")
		if err := childComponent.Render(ctx, w, child); err != nil {
			return err
		}

		_, _ = io.WriteString(w, "</td>\n</tr>\n")
	}

	_, _ = io.WriteString(w, "</tbody>\n</table>\n</td>\n</tr>\n</tbody>\n</table>\n</div>\n")
	_, _ = io.WriteString(w, "<!--[if mso | IE]>\n</td>\n</tr>\n</table>\n<![endif]-->\n")
	return nil
}

func (h MJMLHero) getStyles(ctx *RenderContext, n *node.Node) (map[string]inlineStyle, error) {
	containerWidth := ctx.ContainerWidth

	heightInt, err := strconv.Atoi(RemoveNonNumeric(n.GetAttributeValueDefault("background-height")))
	if err != nil {
		return nil, err
	}

	widthInt, err := strconv.Atoi(RemoveNonNumeric(n.GetAttributeValueDefault("background-width")))
	if err != nil {
		return nil, err
	}

	backgroundRatio := math.Round((float64(heightInt) / float64(widthInt)) * 100)

	width := n.GetAttributeValueDefault("background-width")
	if width == "" {
		width = containerWidth
	}

	m := map[string]inlineStyle{
		"div": {
			{Property: "margin", Value: "0 auto"},
			{Property: "max-width", Value: containerWidth},
		},
		"table": {
			{Property: "width", Value: "100%"},
		},
		"tr": {
			{Property: "vertical-align", Value: "top"},
		},
		"td-fluid": {
			{Property: "width", Value: "0.01%"},
			{Property: "padding-bottom", Value: fmt.Sprintf("%d%%", int(backgroundRatio))},
			{Property: "mso-padding-bottom-alt", Value: "0"},
		},
		"outlook-table": {
			{Property: "width", Value: containerWidth},
		},
		"outlook-td": {
			{Property: "line-height", Value: "0"},
			{Property: "font-size", Value: "0"},
			{Property: "mso-line-height-rule", Value: "exactly"},
		},
		"outlook-inner-table": {
			{Property: "width", Value: containerWidth},
		},
		"outlook-image": {
			{Property: "border", Value: "0"},
			{Property: "height", Value: n.GetAttributeValueDefault("background-height")},
			{Property: "mso-position-horizontal", Value: "center"},
			{Property: "position", Value: "absolute"},
			{Property: "top", Value: "0"},
			{Property: "width", Value: width},
			{Property: "z-index", Value: "-3"},
		},
		"outlook-inner-td": {
			{Property: "background-color", Value: n.GetAttributeValueDefault("inner-background-color")},
			{Property: "padding", Value: n.GetAttributeValueDefault("inner-padding")},
			{Property: "padding-bottom", Value: n.GetAttributeValueDefault("inner-padding-bottom")},
			{Property: "padding-left", Value: n.GetAttributeValueDefault("inner-padding-left")},
			{Property: "padding-right", Value: n.GetAttributeValueDefault("inner-padding-right")},
			{Property: "padding-top", Value: n.GetAttributeValueDefault("inner-padding-top")},
		},
		"inner-table": {
			{Property: "width", Value: "100%"},
			{Property: "margin", Value: "0px"},
		},
		"inner-div": {
			{Property: "background-color", Value: n.GetAttributeValueDefault("inner-background-color")},
			{Property: "background-color", Value: n.GetAttributeValueDefault("align")},
			{Property: "margin", Value: "0px auto"},
			{Property: "width", Value: n.GetAttributeValueDefault("width")},
		},
	}

	return m, nil
}

func (h MJMLHero) getBackground(n *node.Node) string {
	var parts []string

	parts = append(parts, n.GetAttributeValueDefault("background-color"))

	backURL, has := n.GetAttributeValue("background-url")
	if has {
		parts = append(parts, fmt.Sprintf("url('%s')", backURL))
		parts = append(parts, "no-repeat")
		parts = append(parts, fmt.Sprintf("%s / cover", n.GetAttributeValueDefault("background-position")))
	}

	return makeBackgroundString(parts)
}

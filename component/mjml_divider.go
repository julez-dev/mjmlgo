package component

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLDivider struct{}

func (d MJMLDivider) Name() string {
	return "mj-divider"
}

func (d MJMLDivider) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"border-color":               validateColor(),
		"border-style":               validateType("string"),
		"border-width":               validateUnit([]string{"px"}, false),
		"container-background-color": validateColor(),
		"padding":                    validateUnit([]string{"px", "%"}, true),
		"padding-bottom":             validateUnit([]string{"px", "%"}, false),
		"padding-left":               validateUnit([]string{"px", "%"}, false),
		"padding-right":              validateUnit([]string{"px", "%"}, false),
		"padding-top":                validateUnit([]string{"px", "%"}, false),
		"width":                      validateUnit([]string{"px", "%"}, false),
		"align":                      validateEnum([]string{"left", "center", "right"}),
	}
}

func (d MJMLDivider) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"border-color": "#000000",
		"border-style": "solid",
		"border-width": "4px",
		"padding":      "10px 25px",
		"width":        "100%",
		"align":        "center",
	}
}

func (d MJMLDivider) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	styles, err := d.getStyles(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get styles: %w", err)
	}

	_, _ = io.WriteString(w, "<p "+inlineAttributes{"style": styles["p"].InlineString()}.InlineString()+"></p>")
	if err := d.renderAfter(ctx, w, n); err != nil {
		return fmt.Errorf("failed to render after: %w", err)
	}

	return nil
}

func (d MJMLDivider) renderAfter(ctx *RenderContext, w io.Writer, n *node.Node) error {
	styles, err := d.getStyles(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get styles: %w", err)
	}

	outlookWidth, err := d.getOutlookWidth(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get outlook width: %w", err)
	}

	attr := inlineAttributes{
		"align":       n.GetAttributeValueDefault("align"),
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"style":       styles["outlook"].InlineString(),
		"role":        "presentation",
		"width":       outlookWidth,
	}

	_, _ = io.WriteString(w, "<!--[if mso | IE]><table "+attr.InlineString()+"><tr><td style=\"height:0;line-height:0;\">&nbsp;</td></tr></table><![endif]-->")
	return nil
}

func (d MJMLDivider) getStyles(ctx *RenderContext, n *node.Node) (map[string]inlineStyle, error) {
	var computeAlign = "0px auto"
	if n.GetAttributeValueDefault("align") == "left" {
		computeAlign = "0px"
	} else if n.GetAttributeValueDefault("align") == "right" {
		computeAlign = "0px 0px 0px auto"
	}

	var borderTopParts []string
	for _, field := range [...]string{"style", "width", "color"} {
		if v, has := n.GetAttributeValue("border-" + field); has {
			borderTopParts = append(borderTopParts, v)
		}
	}

	pStyle := inlineStyle{
		{Property: "border-top", Value: strings.Join(borderTopParts, " ")},
		{Property: "margin", Value: computeAlign},
		{Property: "width", Value: n.GetAttributeValueDefault("width")},
		{Property: "font-size", Value: "1px"},
	}

	outlookWidth, err := d.getOutlookWidth(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("failed to get outlook width: %w", err)
	}

	outlookStyle := inlineStyle{
		{Property: "border-top", Value: strings.Join(borderTopParts, " ")},
		{Property: "margin", Value: computeAlign},
		{Property: "width", Value: outlookWidth},
		{Property: "font-size", Value: "1px"},
	}

	return map[string]inlineStyle{
		"p":       pStyle,
		"outlook": outlookStyle,
	}, nil
}

func (d MJMLDivider) getOutlookWidth(ctx *RenderContext, n *node.Node) (string, error) {
	parsedContainerWidth, err := strconv.Atoi(RemoveNonNumeric(ctx.ContainerWidth))
	if err != nil {
		return "", err
	}

	left, err := getShorthandAttrValue(n, "padding", "left")
	if err != nil {
		return "", err
	}

	right, err := getShorthandAttrValue(n, "padding", "right")
	if err != nil {
		return "", err
	}

	paddingSize := left + right

	width := n.GetAttributeValueDefault("width")
	parsed, unit, err := parseWidth(width)
	if err != nil {
		return "", err
	}

	switch unit {
	case "%":
		effectiveWidth := parsedContainerWidth - paddingSize
		percentMultiplier := int(parsed) / 100
		return fmt.Sprintf("%dpx", effectiveWidth*percentMultiplier), nil
	case "px":
		return width, nil
	default:
		return fmt.Sprintf("%dpx", parsedContainerWidth-paddingSize), nil
	}
}

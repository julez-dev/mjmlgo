package component

import (
	"fmt"
	"io"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLTable struct{}

func (t MJMLTable) Name() string {
	return "mj-table"
}

func (t MJMLTable) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"align":                      validateEnum([]string{"left", "right", "center"}),
		"border":                     validateType("string"),
		"cellpadding":                validateType("number"),
		"cellspacing":                validateType("number"),
		"container-background-color": validateColor(),
		"color":                      validateColor(),
		"font-family":                validateType("string"),
		"font-size":                  validateUnit([]string{"px"}, false),
		"font-weight":                validateType("string"),
		"line-height":                validateUnit([]string{"px", "%"}, false),
		"padding-bottom":             validateUnit([]string{"px", "%"}, false),
		"padding-left":               validateUnit([]string{"px", "%"}, false),
		"padding-right":              validateUnit([]string{"px", "%"}, false),
		"padding-top":                validateUnit([]string{"px", "%"}, false),
		"padding":                    validateUnit([]string{"px", "%"}, true),
		"role":                       validateEnum([]string{"none", "presentation"}),
		"table-layout":               validateEnum([]string{"auto", "fixed", "initial", "inherit"}),
		"vertical-align":             validateEnum([]string{"top", "bottom", "middle"}),
		"width":                      validateUnit([]string{"px", "%"}, false),
	}
}

func (t MJMLTable) DefaultAttributes(ctx *RenderContext) map[string]string {
	return map[string]string{
		"align":        "left",
		"border":       "none",
		"cellpadding":  "0",
		"cellspacing":  "0",
		"color":        "#000000",
		"font-family":  "Ubuntu, Helvetica, Arial, sans-serif",
		"font-size":    "13px",
		"line-height":  "22px",
		"padding":      "10px 25px",
		"table-layout": "auto",
		"width":        "100%",
	}
}

func (t MJMLTable) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	attributeKeys := [...]string{"cellpadding", "cellspacing", "role"}
	tableAttributes := make(inlineAttributes)

	for _, key := range attributeKeys {
		if v, has := n.GetAttributeValue(key); has {
			tableAttributes[key] = v
		}
	}

	width, err := t.getWidth(n)
	if err != nil {
		return err
	}

	tableAttributes["width"] = width
	tableAttributes["border"] = "0"
	tableAttributes["style"] = t.getStyle(ctx, n).InlineString()

	_, _ = io.WriteString(w, "<table "+tableAttributes.InlineString()+">\n")
	_, _ = io.WriteString(w, n.Content)
	_, _ = io.WriteString(w, "</table>")

	return nil
}

func (t MJMLTable) getStyle(_ *RenderContext, n *node.Node) inlineStyle {
	return inlineStyle{
		{Property: "color", Value: n.GetAttributeValueDefault("color")},
		{Property: "font-family", Value: n.GetAttributeValueDefault("font-family")},
		{Property: "font-size", Value: n.GetAttributeValueDefault("font-size")},
		{Property: "line-height", Value: n.GetAttributeValueDefault("line-height")},
		{Property: "table-layout", Value: n.GetAttributeValueDefault("table-layout")},
		{Property: "width", Value: n.GetAttributeValueDefault("width")},
		{Property: "border", Value: n.GetAttributeValueDefault("border")},
	}
}

func (t MJMLTable) getWidth(n *node.Node) (string, error) {
	width := n.GetAttributeValueDefault("width")
	parsedWidth, unit, err := parseWidth(width)
	if err != nil {
		return "", err
	}

	if unit == "%" {
		return width, nil
	}

	return fmt.Sprintf("%d", int(parsedWidth)), nil
}

package component

import (
	"fmt"
	"io"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLButton struct{}

func (b MJMLButton) Name() string {
	return "mj-button"
}

func (b MJMLButton) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"align":                      validateEnum([]string{"left", "center", "right"}),
		"background-color":           validateColor(),
		"border-bottom":              validateType("string"),
		"border-left":                validateType("string"),
		"border-radius":              validateType("string"),
		"border-right":               validateType("string"),
		"border-top":                 validateType("string"),
		"border":                     validateType("string"),
		"color":                      validateColor(),
		"container-background-color": validateColor(),
		"font-family":                validateType("string"),
		"font-size":                  validateUnit([]string{"px"}, false),
		"font-style":                 validateType("string"),
		"font-weight":                validateType("string"),
		"height":                     validateUnit([]string{"px", "%"}, false),
		"href":                       validateType("string"),
		"name":                       validateType("string"),
		"title":                      validateType("string"),
		"inner-padding":              validateUnit([]string{"px", "%"}, true),
		"letter-spacing":             validateUnit([]string{"px", "em"}, false),
		"line-height":                validateUnit([]string{"px", "%", ""}, false),
		"padding-bottom":             validateUnit([]string{"px", "%"}, false),
		"padding-left":               validateUnit([]string{"px", "%"}, false),
		"padding-right":              validateUnit([]string{"px", "%"}, false),
		"padding-top":                validateUnit([]string{"px", "%"}, false),
		"padding":                    validateUnit([]string{"px", "%"}, true),
		"rel":                        validateType("string"),
		"target":                     validateType("string"),
		"text-decoration":            validateType("string"),
		"text-transform":             validateType("string"),
		"vertical-align":             validateEnum([]string{"top", "bottom", "middle"}),
		"text-align":                 validateEnum([]string{"left", "right", "center"}),
		"width":                      validateUnit([]string{"px", "%"}, false),
	}
}

func (b MJMLButton) DefaultAttributes(ctx *RenderContext) map[string]string {
	return map[string]string{
		"align":            "center",
		"background-color": "#414141",
		"border":           "none",
		"border-radius":    "3px",
		"color":            "#ffffff",
		"font-family":      "Ubuntu, Helvetica, Arial, sans-serif",
		"font-size":        "13px",
		"font-weight":      "normal",
		"inner-padding":    "10px 25px",
		"line-height":      "120%",
		"padding":          "10px 25px",
		"target":           "_blank",
		"text-decoration":  "none",
		"text-transform":   "none",
		"vertical-align":   "middle",
	}
}

func (b MJMLButton) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	styles, err := b.getStyles(ctx, n)
	if err != nil {
		return err
	}

	tag := "p"
	if _, ok := n.GetAttributeValue("href"); ok {
		tag = "a"
	}

	tableAttributes := inlineAttributes{
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
		"style":       styles["table"].InlineString(),
	}

	tdAttributes := inlineAttributes{
		"align":  "center",
		"role":   "presentation",
		"style":  styles["td"].InlineString(),
		"valign": n.GetAttributeValueDefault("vertical-align"),
	}

	if bgColor, ok := n.GetAttributeValue("background-color"); ok && bgColor != "none" {
		tdAttributes["bgcolor"] = bgColor
	}

	contentTagAttributes := inlineAttributes{
		"style": styles["content"].InlineString(),
	}

	if tag == "a" {
		contentTagAttributes["href"] = n.GetAttributeValueDefault("href")
		contentTagAttributes["name"] = n.GetAttributeValueDefault("name")
		contentTagAttributes["rel"] = n.GetAttributeValueDefault("rel")
		contentTagAttributes["title"] = n.GetAttributeValueDefault("title")
		contentTagAttributes["target"] = n.GetAttributeValueDefault("target")
	}

	fmt.Fprintf(w, "<table %s>", tableAttributes.InlineString())
	fmt.Fprint(w, "<tbody><tr>")
	fmt.Fprintf(w, "<td %s>", tdAttributes.InlineString())
	fmt.Fprintf(w, "<%s %s>", tag, contentTagAttributes.InlineString())
	fmt.Fprint(w, n.Content)
	fmt.Fprintf(w, "</%s>", tag)
	fmt.Fprint(w, "</td></tr></tbody></table>")

	return nil
}

func (b MJMLButton) getStyles(ctx *RenderContext, n *node.Node) (map[string]inlineStyle, error) {
	m := map[string]inlineStyle{
		"table": {
			{Property: "border-collapse", Value: "separate"},
			{Property: "width", Value: n.GetAttributeValueDefault("width")},
			{Property: "line-height", Value: "100%"},
		},
		"td": {
			{Property: "border", Value: n.GetAttributeValueDefault("border")},
			{Property: "border-bottom", Value: n.GetAttributeValueDefault("border-bottom")},
			{Property: "border-left", Value: n.GetAttributeValueDefault("border-left")},
			{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
			{Property: "border-right", Value: n.GetAttributeValueDefault("border-right")},
			{Property: "border-top", Value: n.GetAttributeValueDefault("border-top")},
			{Property: "cursor", Value: "auto"},
			{Property: "font-style", Value: n.GetAttributeValueDefault("font-style")},
			{Property: "height", Value: n.GetAttributeValueDefault("height")},
			{Property: "mso-padding-alt", Value: n.GetAttributeValueDefault("inner-padding")},
			{Property: "text-align", Value: n.GetAttributeValueDefault("text-align")},
			{Property: "background", Value: n.GetAttributeValueDefault("background-color")},
		},
		"content": {
			{Property: "display", Value: "inline-block"},
			{Property: "background", Value: n.GetAttributeValueDefault("background-color")},
			{Property: "color", Value: n.GetAttributeValueDefault("color")},
			{Property: "font-family", Value: n.GetAttributeValueDefault("font-family")},
			{Property: "font-size", Value: n.GetAttributeValueDefault("font-size")},
			{Property: "font-style", Value: n.GetAttributeValueDefault("font-style")},
			{Property: "font-weight", Value: n.GetAttributeValueDefault("font-weight")},
			{Property: "line-height", Value: n.GetAttributeValueDefault("line-height")},
			{Property: "letter-spacing", Value: n.GetAttributeValueDefault("letter-spacing")},
			{Property: "margin", Value: "0"},
			{Property: "text-decoration", Value: n.GetAttributeValueDefault("text-decoration")},
			{Property: "text-transform", Value: n.GetAttributeValueDefault("text-transform")},
			{Property: "padding", Value: n.GetAttributeValueDefault("inner-padding")},
			{Property: "mso-padding-alt", Value: "0px"},
			{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
		},
	}

	w, err := b.calculateAWidth(ctx, n)
	if err != nil {
		return nil, err
	}

	m["content"] = append(m["content"], Style{
		Property: "width",
		Value:    w,
	})

	return m, nil
}

func (b MJMLButton) calculateAWidth(ctx *RenderContext, n *node.Node) (string, error) {
	width, ok := n.GetAttributeValue("width")
	if !ok {
		return "", nil
	}

	parsedWidth, unit, err := parseWidth(width)
	if err != nil {
		return "", err
	}

	if unit != "px" {
		return "", nil
	}

	box, err := getBoxWidths(ctx, n)
	if err != nil {
		return "", err
	}

	innerPaddingLeft, err := getShorthandAttrValue(n, "inner-padding", "left")
	if err != nil {
		return "", err
	}

	innerPaddingRight, err := getShorthandAttrValue(n, "inner-padding", "right")
	if err != nil {
		return "", err
	}

	innerPaddings := innerPaddingLeft + innerPaddingRight

	return fmt.Sprintf("%dpx", int(parsedWidth)-innerPaddings-box["borders"]), nil
}

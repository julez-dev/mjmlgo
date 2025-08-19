package component

import (
	"fmt"
	"io"
	"strconv"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLImage struct{}

func (i MJMLImage) Name() string {
	return "mj-image"
}

func (i MJMLImage) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"alt":                        validateType("string"),
		"href":                       validateType("string"),
		"name":                       validateType("string"),
		"src":                        validateType("string"),
		"srcset":                     validateType("string"),
		"sizes":                      validateType("string"),
		"title":                      validateType("string"),
		"rel":                        validateType("string"),
		"align":                      validateEnum([]string{"left", "center", "right"}),
		"border":                     validateType("string"),
		"border-bottom":              validateType("string"),
		"border-left":                validateType("string"),
		"border-right":               validateType("string"),
		"border-top":                 validateType("string"),
		"border-radius":              validateUnit([]string{"px", "%"}, true),
		"container-background-color": validateColor(),
		"fluid-on-mobile":            validateType("boolean"),
		"padding":                    validateUnit([]string{"px", "%"}, true),
		"padding-bottom":             validateUnit([]string{"px", "%"}, false),
		"padding-left":               validateUnit([]string{"px", "%"}, false),
		"padding-right":              validateUnit([]string{"px", "%"}, false),
		"padding-top":                validateUnit([]string{"px", "%"}, false),
		"target":                     validateType("string"),
		"width":                      validateUnit([]string{"px"}, false),
		"height":                     validateUnit([]string{"px", "auto"}, false),
		"max-height":                 validateUnit([]string{"px", "%"}, false),
		"font-size":                  validateUnit([]string{"px"}, false),
		"usemap":                     validateType("string"),
	}
}

func (i MJMLImage) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"alt":       "",
		"align":     "center",
		"border":    "0",
		"height":    "auto",
		"padding":   "10px 25px",
		"target":    "_blank",
		"font-size": "13px",
	}
}

func (i MJMLImage) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	styles, err := i.getStyles(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get styles: %w", err)
	}

	tableAttr := inlineAttributes{
		"style":       styles["table"].InlineString(),
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
	}

	tdAttr := inlineAttributes{
		"style": styles["td"].InlineString(),
	}

	if _, has := n.GetAttributeValue("fluid-on-mobile"); has {
		tableAttr["class"] = "mj-full-width-mobile"
		tdAttr["class"] = "mj-full-width-mobile"
	}

	_, _ = io.WriteString(w, "<table "+tableAttr.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tbody>\n")
	_, _ = io.WriteString(w, "<tr>\n")
	_, _ = io.WriteString(w, "<td "+tdAttr.InlineString()+">\n")
	if err := i.renderImage(ctx, w, n, styles["img"]); err != nil {
		return fmt.Errorf("failed to render image: %w", err)
	}
	_, _ = io.WriteString(w, "</td>\n")
	_, _ = io.WriteString(w, "</tr>\n")
	_, _ = io.WriteString(w, "</tbody>\n")
	_, _ = io.WriteString(w, "</table>\n")

	return nil
}

func (i MJMLImage) renderImage(ctx *RenderContext, w io.Writer, n *node.Node, imgStyle inlineStyle) error {
	height := n.GetAttributeValueDefault("height")
	width, err := getContentWidth(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get content width: %w", err)
	}

	imgAttr := inlineAttributes{
		"src":    n.GetAttributeValueDefault("src"),
		"alt":    n.GetAttributeValueDefault("alt"),
		"srcset": n.GetAttributeValueDefault("srcset"),
		"sizes":  n.GetAttributeValueDefault("sizes"),
		"style":  imgStyle.InlineString(),
		"title":  n.GetAttributeValueDefault("title"),
		"width":  fmt.Sprintf("%d", width),
		"usemap": n.GetAttributeValueDefault("usemap"),
	}

	if height == "auto" {
		imgAttr["height"] = "auto"
	} else {
		heightAsInt, err := strconv.Atoi(removeNonNumeric(height))
		if err != nil {
			return fmt.Errorf("invalid height value: %w", err)
		}

		imgAttr["height"] = fmt.Sprintf("%d", heightAsInt)
	}

	img := fmt.Sprintf("<img %s />", imgAttr.InlineString())

	if href, ok := n.GetAttributeValue("href"); ok {
		aAttr := inlineAttributes{
			"href":   href,
			"target": n.GetAttributeValueDefault("target"),
			"rel":    n.GetAttributeValueDefault("rel"),
			"name":   n.GetAttributeValueDefault("name"),
			"title":  n.GetAttributeValueDefault("title"),
		}

		img = fmt.Sprintf("<a %s>%s</a>\n", aAttr.InlineString(), img)
		_, _ = io.WriteString(w, img)
		return nil
	}

	_, err = io.WriteString(w, img)
	return nil
}

func (i MJMLImage) getStyles(ctx *RenderContext, n *node.Node) (map[string]inlineStyle, error) {
	width, err := getContentWidth(ctx, n)
	if err != nil {
		return nil, err
	}

	var isFullWidth bool
	if v, ok := n.GetAttributeValue("full-width"); ok && v == "full-width" {
		isFullWidth = true
	}

	parsedWidth, unit, err := parseWidth(fmt.Sprintf("%dpx", width))
	if err != nil {
		return nil, fmt.Errorf("failed to parse width: %w", err)
	}

	imgStyle := inlineStyle{
		{Property: "border", Value: n.GetAttributeValueDefault("border")},
		{Property: "border-left", Value: n.GetAttributeValueDefault("border-left")},
		{Property: "border-right", Value: n.GetAttributeValueDefault("border-right")},
		{Property: "border-top", Value: n.GetAttributeValueDefault("border-top")},
		{Property: "border-bottom", Value: n.GetAttributeValueDefault("border-bottom")},
		{Property: "display", Value: "block"},
		{Property: "outline", Value: "none"},
		{Property: "text-decoration", Value: "none"},
		{Property: "height", Value: n.GetAttributeValueDefault("height")},
		{Property: "max-height", Value: n.GetAttributeValueDefault("max-height")},
		{Property: "width", Value: "100%"},
		{Property: "font-size", Value: n.GetAttributeValueDefault("font-size")},
	}

	tdStyle := inlineStyle{}

	tableStyle := inlineStyle{
		{Property: "border-collapse", Value: "collapse"},
		{Property: "border-spacing", Value: "0px"},
	}

	if isFullWidth {
		imgStyle = append(imgStyle, Style{Property: "min-width", Value: "100%"})
		imgStyle = append(imgStyle, Style{Property: "max-width", Value: "100%"})

		tableStyle = append(tableStyle, Style{Property: "min-width", Value: "100%"})
		tableStyle = append(tableStyle, Style{Property: "max-width", Value: "100%"})
		tableStyle = append(tableStyle, Style{Property: "width", Value: fmt.Sprintf("%d%s", int(parsedWidth), unit)})
	} else {
		tdStyle = append(tdStyle, Style{Property: "width", Value: fmt.Sprintf("%d%s", int(parsedWidth), unit)})
	}

	return map[string]inlineStyle{
		"img":   imgStyle,
		"td":    tdStyle,
		"table": tableStyle,
	}, nil
}

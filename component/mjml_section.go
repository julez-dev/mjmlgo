package component

import (
	"io"
	"strconv"
	"strings"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLSection struct {
	IsWrapper bool
}

func (r MJMLSection) Name() string {
	return "mj-section"
}

func (s MJMLSection) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"background-color":      validateColor(),
		"background-url":        validateType("string"),
		"background-repeat":     validateEnum([]string{"repeat", "no-repeat"}),
		"background-size":       validateType("string"),
		"background-position":   validateType("string"),
		"background-position-x": validateType("string"),
		"background-position-y": validateType("string"),
		"border":                validateType("string"),
		"border-bottom":         validateType("string"),
		"border-left":           validateType("string"),
		"border-radius":         validateType("string"),
		"border-right":          validateType("string"),
		"border-top":            validateType("string"),
		"direction":             validateEnum([]string{"ltr", "rtl"}),
		"full-width":            validateEnum([]string{"full-width", "false"}),
		"padding-bottom":        validateUnit([]string{"px", "%"}, false),
		"padding-left":          validateUnit([]string{"px", "%"}, false),
		"padding-right":         validateUnit([]string{"px", "%"}, false),
		"padding-top":           validateUnit([]string{"px", "%"}, false),
		"padding":               validateUnit([]string{"px", "%"}, true),
		"text-align":            validateEnum([]string{"left", "center", "right"}),
		"text-padding":          validateUnit([]string{"px", "%"}, true),
	}
}

func (s MJMLSection) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"background-repeat":   "repeat",
		"background-size":     "auto",
		"background-position": "top center",
		"direction":           "ltr",
		"padding":             "20px 0",
		"text-align":          "center",
		"text-padding":        "4px 4px 4px 0",
	}
}

func (s MJMLSection) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	if v, ok := n.GetAttributeValue("full-width"); ok && v == "full-width" {
		return s.renderFullWidth(ctx, w, n)
	}

	return s.renderSimple(ctx, w, n)
}

func (s MJMLSection) renderFullWidth(ctx *RenderContext, w io.Writer, n *node.Node) error {
	backgroundStyle := inlineStyle{}

	if bgColor, ok := n.GetAttributeValue("background-color"); ok {
		backgroundStyle = append(backgroundStyle, Style{Property: "background", Value: bgColor})
		backgroundStyle = append(backgroundStyle, Style{Property: "background-color", Value: bgColor})
	} else if backgroundURL, ok := n.GetAttributeValue("background-url"); ok {
		var backgroundDescParts []string

		backgroundDescParts = append(backgroundDescParts, "url('"+backgroundURL+"')")
		backgroundDescParts = append(backgroundDescParts, getBackgroundString(n))
		backgroundDescParts = append(backgroundDescParts, "/ "+n.GetAttributeValueDefault("background-size"))
		backgroundDescParts = append(backgroundDescParts, n.GetAttributeValueDefault("background-repeat"))

		backgroundStyle = append(backgroundStyle, Style{Property: "background", Value: makeBackgroundString(backgroundDescParts)})
		backgroundStyle = append(backgroundStyle, Style{Property: "background-position", Value: getBackgroundString(n)})
		backgroundStyle = append(backgroundStyle, Style{Property: "background-repeat", Value: n.GetAttributeValueDefault("background-repeat")})
		backgroundStyle = append(backgroundStyle, Style{Property: "background-size", Value: n.GetAttributeValueDefault("background-size")})
	}

	tableStyle := inlineStyle{
		{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
		{Property: "width", Value: "100%"},
	}

	tableStyle = append(tableStyle, backgroundStyle...)

	tableAttr := inlineAttributes{
		"align":       "center",
		"class":       n.GetAttributeValueDefault("css-class"),
		"background":  n.GetAttributeValueDefault("background-url"),
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
		"style":       tableStyle.InlineString(),
	}

	_, _ = io.WriteString(w, "<table "+tableAttr.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tbody>\n")
	_, _ = io.WriteString(w, "<tr>\n")
	_, _ = io.WriteString(w, "<td>")

	if err := s.renderSimple(ctx, w, n); err != nil {
		return err
	}

	_, _ = io.WriteString(w, "</td>\n")
	_, _ = io.WriteString(w, "</tr>\n")
	_, _ = io.WriteString(w, "</tbody>\n")
	_, _ = io.WriteString(w, "</table>\n")

	return nil
}

func (s MJMLSection) renderSimple(ctx *RenderContext, w io.Writer, n *node.Node) error {
	if err := s.renderBefore(ctx, w, n); err != nil {
		return err
	}

	if n.GetAttributeValueDefault("background-url") != "" {
		var b strings.Builder
		if err := s.renderSection(ctx, &b, n); err != nil {
			return err
		}

		if err := s.renderWithBackground(ctx, w, n, b.String()); err != nil {
			return err
		}
	} else {
		if err := s.renderSection(ctx, w, n); err != nil {
			return err
		}
	}

	if err := s.renderAfter(ctx, w, n); err != nil {
		return err
	}

	return nil
}

func (s MJMLSection) renderBefore(ctx *RenderContext, w io.Writer, n *node.Node) error {
	bgcolor, hasBackgroundColor := n.GetAttributeValue("background-color")

	attr := inlineAttributes{
		"align":       "center",
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"class":       addSuffixToClasses(n.GetAttributeValueDefault("css-class"), "outlook"),
		"role":        "presentation",
		"style":       inlineStyle{{Property: "width", Value: ctx.ContainerWidth}}.InlineString(),
		"width":       strings.TrimSuffix(ctx.ContainerWidth, "px"),
	}

	if hasBackgroundColor {
		attr["bgcolor"] = bgcolor
	}

	_, _ = io.WriteString(w, "<!--[if mso | IE]>")
	_, _ = io.WriteString(w, "<table "+attr.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tr>\n")
	_, _ = io.WriteString(w, "<td style=\"line-height:0px;font-size:0px;mso-line-height-rule:exactly;\">\n")
	_, _ = io.WriteString(w, "<![endif]-->\n")

	return nil
}

func (s MJMLSection) renderSection(ctx *RenderContext, w io.Writer, n *node.Node) error {
	var isFullWidth bool
	if v, ok := n.GetAttributeValue("full-width"); ok && v == "full-width" {
		isFullWidth = true
	}

	_, hasBackground := n.GetAttributeValue("background-url")

	backgroundStyle := inlineStyle{}

	if bgColor, ok := n.GetAttributeValue("background-color"); ok {
		backgroundStyle = append(backgroundStyle, Style{Property: "background", Value: bgColor})
		backgroundStyle = append(backgroundStyle, Style{Property: "background-color", Value: bgColor})
	} else if backgroundURL, ok := n.GetAttributeValue("background-url"); ok {
		var backgroundDescParts []string

		backgroundDescParts = append(backgroundDescParts, "url('"+backgroundURL+"')")
		backgroundDescParts = append(backgroundDescParts, getBackgroundString(n))
		backgroundDescParts = append(backgroundDescParts, "/ "+n.GetAttributeValueDefault("background-size"))
		backgroundDescParts = append(backgroundDescParts, n.GetAttributeValueDefault("background-repeat"))

		backgroundStyle = append(backgroundStyle, Style{Property: "background", Value: makeBackgroundString(backgroundDescParts)})
		backgroundStyle = append(backgroundStyle, Style{Property: "background-position", Value: getBackgroundString(n)})
		backgroundStyle = append(backgroundStyle, Style{Property: "background-repeat", Value: n.GetAttributeValueDefault("background-repeat")})
		backgroundStyle = append(backgroundStyle, Style{Property: "background-size", Value: n.GetAttributeValueDefault("background-size")})
	}

	divAttr := inlineAttributes{
		"class": n.GetAttributeValueDefault("css-class"),
	}

	divStyle := inlineStyle{
		{Property: "margin", Value: "0px auto"},
		{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
		{Property: "max-width", Value: ctx.ContainerWidth},
	}

	innerDivStyle := inlineStyle{
		{Property: "line-height", Value: "0"},
		{Property: "font-size", Value: "0"},
	}

	if !isFullWidth {
		divStyle = append(divStyle, backgroundStyle...)
	} else {
		delete(divAttr, "class")
	}

	divAttr["style"] = divStyle.InlineString()

	tableStyle := inlineStyle{
		{Property: "width", Value: "100%"},
		{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
	}

	if !isFullWidth {
		tableStyle = append(tableStyle, backgroundStyle...)
	}

	tableAttr := inlineAttributes{
		"align":       "center",
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
		"style":       tableStyle.InlineString(),
	}

	if !isFullWidth {
		tableAttr["background"] = n.GetAttributeValueDefault("background-url")
	}

	tdStyle := inlineStyle{
		{Property: "border", Value: n.GetAttributeValueDefault("border")},
		{Property: "border-bottom", Value: n.GetAttributeValueDefault("border-bottom")},
		{Property: "border-left", Value: n.GetAttributeValueDefault("border-left")},
		{Property: "border-right", Value: n.GetAttributeValueDefault("border-right")},
		{Property: "border-top", Value: n.GetAttributeValueDefault("border-top")},
		{Property: "direction", Value: n.GetAttributeValueDefault("direction")},
		{Property: "font-size", Value: "0px"},
		{Property: "padding", Value: n.GetAttributeValueDefault("padding")},
		{Property: "padding-bottom", Value: n.GetAttributeValueDefault("padding-bottom")},
		{Property: "padding-left", Value: n.GetAttributeValueDefault("padding-left")},
		{Property: "padding-right", Value: n.GetAttributeValueDefault("padding-right")},
		{Property: "padding-top", Value: n.GetAttributeValueDefault("padding-top")},
		{Property: "text-align", Value: n.GetAttributeValueDefault("text-align")},
	}

	_, _ = io.WriteString(w, "<div "+divAttr.InlineString()+">\n")
	if hasBackground {
		_, _ = io.WriteString(w, "<div "+innerDivStyle.InlineString()+">\n")
	}

	_, _ = io.WriteString(w, "<table "+tableAttr.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tbody>\n")
	_, _ = io.WriteString(w, "<tr>\n")
	_, _ = io.WriteString(w, "<td "+inlineAttributes{"style": tdStyle.InlineString()}.InlineString()+">\n")
	_, _ = io.WriteString(w, "<!--[if mso | IE]><table role=\"presentation\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><![endif]-->\n")

	if err := s.renderWrappedChildren(ctx, w, n); err != nil {
		return err
	}

	_, _ = io.WriteString(w, "<!--[if mso | IE]></table><![endif]-->\n")
	_, _ = io.WriteString(w, "</td>\n")
	_, _ = io.WriteString(w, "</tr>\n")
	_, _ = io.WriteString(w, "</tbody>\n")
	_, _ = io.WriteString(w, "</table>\n")

	if hasBackground {
		_, _ = io.WriteString(w, "</div>\n")
	}

	_, _ = io.WriteString(w, "</div>\n")
	return nil
}

func (s MJMLSection) renderWrappedChildren(ctx *RenderContext, w io.Writer, n *node.Node) error {
	if !s.IsWrapper {
		_, _ = io.WriteString(w, "<!--[if mso | IE]><tr><![endif]-->\n")

		for _, child := range n.Children {
			switch child.Type {
			case ColumnTagName:
				var column MJMLColumn
				if err := InitComponent(ctx, column, child); err != nil {
					return err
				}

				tdStyle := inlineStyle{
					{Property: "vertical-align", Value: child.GetAttributeValueDefault("vertical-align")},
				}

				width, err := getWidthAsPixel(ctx, child)
				if err != nil {
					return err
				}

				tdStyle = append(tdStyle, Style{Property: "width", Value: width})

				tdAttr := inlineAttributes{
					"align": child.GetAttributeValueDefault("align"),
					"class": addSuffixToClasses(child.GetAttributeValueDefault("css-class"), "outlook"),
					"style": tdStyle.InlineString(),
				}
				_, _ = io.WriteString(w, "<!--[if mso | IE]><td "+tdAttr.InlineString()+"><![endif]-->\n")
				if err := column.Render(ctx, w, child); err != nil {
					return err
				}
				_, _ = io.WriteString(w, "<!--[if mso | IE]></td><![endif]-->\n")
			case GroupTagName:
				var group MJMLGroup
				if err := InitComponent(ctx, group, child); err != nil {
					return err
				}
				if err := group.Render(ctx, w, child); err != nil {
					return err
				}
			}
		}

		_, _ = io.WriteString(w, "<!--[if mso | IE]></tr><![endif]-->\n")
		return nil
	}

	for _, child := range n.Children {
		attr := inlineAttributes{
			"align": child.GetAttributeValueDefault("align"),
			"width": ctx.ContainerWidth,
		}

		if class, has := child.GetAttributeValue("class"); has {
			attr["class"] = addSuffixToClasses(class, "outlook")
		}

		_, _ = io.WriteString(w, "<!--[if mso | IE]><tr>")
		_, _ = io.WriteString(w, "<td "+attr.InlineString()+">")
		_, _ = io.WriteString(w, "<![endif]-->")

		var sec MJMLSection
		if err := InitComponent(ctx, sec, child); err != nil {
			return err
		}
		if err := sec.Render(ctx, w, child); err != nil {
			return err
		}

		_, _ = io.WriteString(w, "<!--[if mso | IE]></td></tr><![endif]-->")
	}

	return nil
}

func (s MJMLSection) renderAfter(_ *RenderContext, w io.Writer, _ *node.Node) error {
	_, _ = io.WriteString(w, `<!--[if mso | IE]></td></tr></table><![endif]-->`)
	return nil
}

func (s MJMLSection) renderWithBackground(ctx *RenderContext, w io.Writer, n *node.Node, content string) error {
	var isFullWidth bool
	if v, ok := n.GetAttributeValue("full-width"); ok && v == "full-width" {
		isFullWidth = true
	}

	bgPosX, bgPosY := getBackgroundPosition(n)

	switch bgPosX {
	case "left":
		bgPosX = "0%"
	case "center":
		bgPosX = "50%"
	case "right":
		bgPosX = "100%"
	default:
		if !isPercentage(bgPosX) {
			bgPosX = "50%"
		}
	}

	switch bgPosY {
	case "top":
		bgPosY = "0%"
	case "center":
		bgPosY = "50%"
	case "bottom":
		bgPosY = "100%"
	default:
		if !isPercentage(bgPosY) {
			bgPosY = "0%"
		}
	}

	var (
		vOriginX string
		vPosX    string

		vOriginY string
		vPosY    string
	)

	for _, v := range []string{"x", "y"} {
		var isX bool
		if v == "x" {
			isX = true
		}

		var isBgRepeat bool
		if v, ok := n.GetAttributeValue("background-repeat"); ok && v == "repeat" {
			isBgRepeat = true
		}

		var (
			pos string
		)

		if isX {
			pos = bgPosX
		} else {
			pos = bgPosY
		}

		var (
			posFloat    float64
			originFloat float64
		)

		if isPercentage(pos) {
			percentageValue := strings.TrimSuffix(pos, "%")
			parsedInt, err := strconv.Atoi(percentageValue)
			if err != nil {
				return err
			}

			decimal := float64(parsedInt) / 100.0

			if isBgRepeat {
				posFloat = decimal
				originFloat = decimal
			} else {
				posFloat = (-50 + decimal*100) / 100
				originFloat = (-50 + decimal*100) / 100
			}
		} else if isBgRepeat {
			if isX {
				originFloat = 0.5
				posFloat = 0.5
			} else {
				originFloat = 0
				posFloat = 0
			}
		} else {
			if isX {
				originFloat = 0
				posFloat = 0
			} else {
				originFloat = -0.5
				posFloat = -0.5
			}
		}

		if isX {
			vOriginX = strconv.FormatFloat(originFloat, 'f', -1, 64)
			vPosX = strconv.FormatFloat(posFloat, 'f', -1, 64)
		} else {
			vOriginY = strconv.FormatFloat(originFloat, 'f', -1, 64)
			vPosY = strconv.FormatFloat(posFloat, 'f', -1, 64)
		}
	}

	// If background size is either cover or contain, we tell VML to keep the aspect
	// and fill the entire element.
	vSizeAttr := inlineAttributes{}
	if n.GetAttributeValueDefault("background-size") == "cover" || n.GetAttributeValueDefault("background-size") == "contain" {
		vSizeAttr["size"] = "1,1"

		if n.GetAttributeValueDefault("background-size") == "cover" {
			vSizeAttr["aspect"] = "atleast"
		} else {
			vSizeAttr["aspect"] = "atmost"
		}
	} else if n.GetAttributeValueDefault("background-size") != "auto" {
		splits := strings.Split(n.GetAttributeValueDefault("background-size"), " ")

		if len(splits) == 1 {
			vSizeAttr["size"] = n.GetAttributeValueDefault("background-size")
			vSizeAttr["aspect"] = "atmost"
		} else {
			vSizeAttr["size"] = strings.Join(splits, ",")
		}
	}

	vmlType := "title"

	if n.GetAttributeValueDefault("background-repeat") == "no-repeat" {
		vmlType = "frame"
	}

	if n.GetAttributeValueDefault("background-size") == "auto" {
		vmlType = "title"
		vOriginX, vPosX, vOriginY, vPosY = "0.5", "0.5", "0", "0"
	}

	rectAttr := inlineAttributes{
		"xmlns:v": "urn:schemas-microsoft-com:vml",
		"fill":    "true",
		"stroke":  "false",
		"style":   inlineStyle{{Property: "width", Value: ctx.ContainerWidth}}.InlineString(),
	}

	if isFullWidth {
		rectAttr["style"] = inlineStyle{{Property: "mso-width-percent", Value: "1000"}}.InlineString()
	}

	fillAttr := inlineAttributes{
		"origin":   vOriginX + ", " + vOriginY,
		"position": vPosX + ", " + vPosY,
		"src":      n.GetAttributeValueDefault("background-url"),
		"color":    n.GetAttributeValueDefault("background-color"),
		"type":     vmlType,
	}

	for k, v := range vSizeAttr {
		fillAttr[k] = v
	}

	_, _ = io.WriteString(w, "<!--[if mso | IE]>\n")
	_, _ = io.WriteString(w, "<v:rect "+rectAttr.InlineString()+">\n")
	_, _ = io.WriteString(w, "<v:fill "+fillAttr.InlineString()+" />\n")
	_, _ = io.WriteString(w, "<v:textbox style=\"mso-fit-shape-to-text:true\" inset=\"0,0,0,0\"><![endif]-->\n")
	_, _ = io.WriteString(w, content)
	_, _ = io.WriteString(w, "<!--[if mso | IE]></v:textbox></v:rect><![endif]-->\n")

	return nil
}

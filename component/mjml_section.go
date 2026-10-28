package component

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLSection struct {
	IsWrapper bool
}

func (s MJMLSection) Name() string {
	if s.IsWrapper {
		return WrapperTagName
	}

	return SectionTagName
}

func (s MJMLSection) AllowedAttributes() map[string]AttributeValidator {
	attributes := map[string]AttributeValidator{
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
		"full-width":            validateEnum([]string{"full-width", "false", ""}),
		"padding-bottom":        validateUnit([]string{"px", "%"}, false),
		"padding-left":          validateUnit([]string{"px", "%"}, false),
		"padding-right":         validateUnit([]string{"px", "%"}, false),
		"padding-top":           validateUnit([]string{"px", "%"}, false),
		"padding":               validateUnit([]string{"px", "%"}, true),
		"text-align":            validateEnum([]string{"left", "center", "right"}),
		"text-padding":          validateUnit([]string{"px", "%"}, true),
	}

	if s.IsWrapper {
		attributes["gap"] = validateUnit([]string{"px"}, false)
	}

	return attributes
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

	// wrong width
	attr := inlineAttributes{
		"align":       "center",
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"class":       addSuffixToClasses(n.GetAttributeValueDefault("css-class"), "outlook"),
		"role":        "presentation",
	}

	width, err := parseIntWidth(ctx.ContainerWidth)
	if err != nil {
		return err
	}

	attr["width"] = fmt.Sprintf("%d", width)

	gap := sectionGap(n)
	style := inlineStyle{{Property: "width", Value: ctx.ContainerWidth}}
	if gap != "" && !isFirstSibling(n) {
		style = append(style, Style{Property: "padding-top", Value: gap})
	}
	attr["style"] = style.InlineString()

	// A wrapper gap replaces the background colour on the Outlook seat: the gap
	// is rendered by the section itself so the colour would fill the gap.
	if hasBackgroundColor && gap == "" {
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

	containerWidth := ctx.ContainerWidth
	borderRadius, hasBorderRadius := n.GetAttributeValue("border-radius")
	hasBorderRadius = hasBorderRadius && borderRadius != ""

	box, err := getBoxWidths(ctx, n)
	if err != nil {
		return err
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
	}

	if hasBorderRadius {
		divStyle = append(divStyle, Style{Property: "overflow", Value: "hidden"})
	}

	if gap := sectionGap(n); gap != "" && !isFirstSibling(n) {
		divStyle = append(divStyle, Style{Property: "margin-top", Value: gap})
	}

	divStyle = append(divStyle, Style{Property: "max-width", Value: containerWidth})

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
	}

	if !isFullWidth {
		tableStyle = append(tableStyle, backgroundStyle...)
	}

	if hasBorderRadius {
		tableStyle = append(tableStyle, Style{Property: "border-collapse", Value: "separate"})
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
		{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
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

	previousContainerWidth := ctx.ContainerWidth
	ctx.ContainerWidth = fmt.Sprintf("%dpx", box["box"])
	defer func() {
		ctx.ContainerWidth = previousContainerWidth
	}()

	channelWidth := containerWidth
	if !s.IsWrapper {
		channelWidth = ctx.ContainerWidth
	}

	if err := s.renderWrappedChildren(ctx, w, n, channelWidth); err != nil {
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

func (s MJMLSection) renderWrappedChildren(ctx *RenderContext, w io.Writer, n *node.Node, wrapperWidth string) error {
	if !s.IsWrapper {
		_, _ = io.WriteString(w, "<!--[if mso | IE]><tr><![endif]-->\n")

		for _, child := range n.Children {
			if child.Type == RawTagName {
				// Section raw content is emitted inside the section cell, not
				// directly in a table body, so it needs no markers.
				if err := renderRawChild(ctx, w, child); err != nil {
					return err
				}

				continue
			}

			switch child.Type {
			case ColumnTagName:
				var column MJMLColumn
				if err := InitComponent(ctx, column, child); err != nil {
					return err
				}

				width, err := getWidthAsPixel(ctx, child)
				if err != nil {
					return err
				}

				if err := renderOutlookSeat(w, child, width); err != nil {
					return err
				}
				if err := column.Render(ctx, w, child); err != nil {
					return err
				}
				_, _ = io.WriteString(w, "<!--[if mso | IE]></td><![endif]-->\n")
			case GroupTagName:
				var group MJMLGroup
				if err := InitComponent(ctx, group, child); err != nil {
					return err
				}

				width, err := getWidthAsPixel(ctx, child)
				if err != nil {
					return err
				}

				if err := renderOutlookSeat(w, child, width); err != nil {
					return err
				}
				if err := group.Render(ctx, w, child); err != nil {
					return err
				}
				_, _ = io.WriteString(w, "<!--[if mso | IE]></td><![endif]-->\n")
			}
		}

		_, _ = io.WriteString(w, "<!--[if mso | IE]></tr><![endif]-->\n")
		return nil
	}

	for _, child := range n.Children {
		if child.Type == RawTagName {
			// Wrapper raw content is emitted between the section cells, not
			// directly in a table body, so it needs no markers.
			if err := renderRawChild(ctx, w, child); err != nil {
				return err
			}

			continue
		}

		attr := inlineAttributes{
			"align": child.GetAttributeValueDefault("align"),
			"class": addSuffixToClasses(child.GetAttributeValueDefault("css-class"), "outlook"),
			"width": wrapperWidth,
		}

		var childComponent Component

		switch child.Type {
		case SectionTagName:
			childComponent = MJMLSection{}
		case HeroTagName:
			childComponent = MJMLHero{}
		default:
			continue
		}

		_, _ = io.WriteString(w, "<!--[if mso | IE]><tr>")
		_, _ = io.WriteString(w, "<td "+attr.InlineString()+">")
		_, _ = io.WriteString(w, "<![endif]-->")

		if err := InitComponent(ctx, childComponent, child); err != nil {
			return err
		}
		if err := childComponent.Render(ctx, w, child); err != nil {
			return err
		}

		_, _ = io.WriteString(w, "<!--[if mso | IE]></td></tr><![endif]-->")
	}

	return nil
}

// renderOutlookSeat emits the conditional <td> that reserves a column's (or
// group's) width in the section's Outlook table.
func renderOutlookSeat(w io.Writer, child *node.Node, width string) error {
	tdStyle := inlineStyle{
		{Property: "vertical-align", Value: child.GetAttributeValueDefault("vertical-align")},
		{Property: "width", Value: width},
	}

	tdAttr := inlineAttributes{
		"align": child.GetAttributeValueDefault("align"),
		"class": addSuffixToClasses(child.GetAttributeValueDefault("css-class"), "outlook"),
		"style": tdStyle.InlineString(),
	}

	_, err := io.WriteString(w, "<!--[if mso | IE]><td "+tdAttr.InlineString()+"><![endif]-->\n")

	return err
}

// sectionGap returns the wrapper gap that applies to n: only direct children of
// an <mj-wrapper> inherit its gap, matching MJML's section child context.
func sectionGap(n *node.Node) string {
	if n.Parent == nil || n.Parent.Type != WrapperTagName {
		return ""
	}

	gap, _ := n.Parent.GetAttributeValue("gap")

	return gap
}

// isFirstSibling reports whether n is the first child of its parent. MJML uses
// props.index for this, which counts raw children too.
func isFirstSibling(n *node.Node) bool {
	if n.Parent == nil || len(n.Parent.Children) == 0 {
		return true
	}

	return n.Parent.Children[0] == n
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

		var pos string

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

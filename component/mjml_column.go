package component

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/julez-dev/mjmlgo/v2/node"
)

type MJMLColumn struct{}

func (c MJMLColumn) Name() string {
	return "mj-column"
}

var columnAllowedAttributes = map[string]AttributeValidator{
	"background-color":       validateColor(),
	"border":                 validateType("string"),
	"border-bottom":          validateType("string"),
	"border-left":            validateType("string"),
	"border-radius":          validateType("string"),
	"border-right":           validateType("string"),
	"border-top":             validateType("string"),
	"direction":              validateEnum([]string{"ltr", "rtl"}),
	"inner-background-color": validateColor(),
	"padding-bottom":         validateUnit([]string{"px", "%"}, false),
	"padding-left":           validateUnit([]string{"px", "%"}, false),
	"padding-right":          validateUnit([]string{"px", "%"}, false),
	"padding-top":            validateUnit([]string{"px", "%"}, false),
	"padding":                validateUnit([]string{"px", "%"}, true),
	"inner-border":           validateType("string"),
	"inner-border-bottom":    validateType("string"),
	"inner-border-left":      validateType("string"),
	"inner-border-radius":    validateType("string"),
	"inner-border-right":     validateType("string"),
	"inner-border-top":       validateType("string"),
	"vertical-align":         validateEnum([]string{"top", "bottom", "middle"}),
	"width":                  validateUnit([]string{"px", "%"}, false),
}

func (c MJMLColumn) AllowedAttributes() map[string]AttributeValidator {
	return columnAllowedAttributes
}

func (c MJMLColumn) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"direction":      "ltr",
		"vertical-align": "top",
	}
}

func (c MJMLColumn) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	className, err := c.columnClass(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get column class: %w", err)
	}

	className += " mj-outlook-group-fix"

	if v := n.GetAttributeValueDefault("css-class"); v != "" {
		className += " " + v
	}

	divStyle := inlineStyle{
		{Property: "font-size", Value: "0px"},
		{Property: "text-align", Value: "left"},
		{Property: "direction", Value: n.GetAttributeValueDefault("direction")},
		{Property: "display", Value: "inline-block"},
		{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")},
	}

	divWidth, err := c.mobileWidth(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get mobile width: %w", err)
	}

	divStyle = append(divStyle, Style{Property: "width", Value: divWidth})

	gutterStyle, err := c.mobileGutterStyle(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get gutter padding: %w", err)
	}

	divStyle = append(divStyle, gutterStyle...)

	divAttr := inlineAttributes{
		"class": className,
		"style": divStyle.InlineString(),
	}

	childWidth, err := c.childContainerWidth(ctx, n)
	if err != nil {
		return fmt.Errorf("failed to get child container width: %w", err)
	}

	previousContainerWidth := ctx.ContainerWidth
	ctx.ContainerWidth = childWidth
	defer func() {
		ctx.ContainerWidth = previousContainerWidth
	}()

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

	for _, child := range n.Children {
		if !allowsChild(ctx, ColumnTagName, child.Type) {
			return NewError(
				KindDisallowedChild,
				append(elementPathSegments(n), child.Type),
				"",
				child.Type,
				fmt.Errorf("allowed children: %s", allowedChildList(ctx, ColumnTagName)),
			)
		}
	}

	_, _ = io.WriteString(w, "<table "+tableAttr.InlineString()+">\n")
	_, _ = io.WriteString(w, "<tbody>\n")

	for _, child := range n.Children {
		if child.Type == RawTagName {
			if err := renderRawChildInTableBody(ctx, w, child); err != nil {
				return err
			}

			continue
		}

		component, ok := childComponent(ctx, child.Type)
		if !ok {
			// Unreachable while ValidateDocument runs first: every other child
			// of mj-column is a disallowed-child error before rendering. Report
			// the broken invariant as internal, not as author input.
			return fmt.Errorf("no component for allowed child <%s> in <mj-column>", child.Type)
		}

		if err := InitComponent(ctx, component, child); err != nil {
			return err
		}

		_, _ = io.WriteString(w, "<tr>\n")
		_, _ = io.WriteString(w, "<td "+c.tdAttribute(child).InlineString()+">\n")
		if err := component.Render(ctx, w, child); err != nil {
			return fmt.Errorf("failed to render <%s>: %w", child.Type, err)
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

	if c.hasBorderRadius(n) {
		tableAttr["style"] = inlineStyle{{Property: "border-collapse", Value: "separate"}}.InlineString()
	}

	tdStyle := inlineStyle{
		{Property: "padding", Value: n.GetAttributeValueDefault("padding")},
		{Property: "padding-top", Value: n.GetAttributeValueDefault("padding-top")},
		{Property: "padding-right", Value: n.GetAttributeValueDefault("padding-right")},
		{Property: "padding-bottom", Value: n.GetAttributeValueDefault("padding-bottom")},
		{Property: "padding-left", Value: n.GetAttributeValueDefault("padding-left")},
	}

	tdStyle = append(tdStyle, c.outerTableStyle(n)...)

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

// childContainerWidth implements mj-column's getChildContext: children are
// rendered at the column width minus its paddings, borders and inner borders.
func (c MJMLColumn) childContainerWidth(ctx *RenderContext, n *node.Node) (string, error) {
	parentWidth, err := parseFloatWidth(ctx.ContainerWidth)
	if err != nil {
		return "", err
	}

	unit, width, err := getParsedWidth(n)
	if err != nil {
		return "", err
	}

	var containerWidth float64
	if unit == "%" {
		containerWidth = parentWidth * width / 100
	} else {
		containerWidth = width
	}

	paddingLeft, err := getShorthandAttrValue(n, "padding", "left")
	if err != nil {
		return "", err
	}

	paddingRight, err := getShorthandAttrValue(n, "padding", "right")
	if err != nil {
		return "", err
	}

	borderLeft := getShorthandBorderValue(n, "left", "border")
	borderRight := getShorthandBorderValue(n, "right", "border")
	innerBorderLeft := getShorthandBorderValue(n, "left", "inner-border")
	innerBorderRight := getShorthandBorderValue(n, "right", "inner-border")

	containerWidth -= float64(paddingLeft + paddingRight + borderLeft + borderRight + innerBorderLeft + innerBorderRight)

	return formatPixelWidth(containerWidth), nil
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
	if !c.hasGutter(n) {
		return c.outerTableStyle(n)
	}

	style := inlineStyle{
		{Property: "background-color", Value: n.GetAttributeValueDefault("inner-background-color")},
		{Property: "border", Value: n.GetAttributeValueDefault("inner-border")},
		{Property: "border-bottom", Value: n.GetAttributeValueDefault("inner-border-bottom")},
		{Property: "border-left", Value: n.GetAttributeValueDefault("inner-border-left")},
		{Property: "border-radius", Value: n.GetAttributeValueDefault("inner-border-radius")},
		{Property: "border-right", Value: n.GetAttributeValueDefault("inner-border-right")},
		{Property: "border-top", Value: n.GetAttributeValueDefault("inner-border-top")},
	}

	if c.hasInnerBorderRadius(n) {
		style = append(style, Style{Property: "border-collapse", Value: "separate"})
	}

	return style
}

// outerTableStyle is MJML's normal column table style. The gutter cell uses it
// too, so the column's own background and border survive the gutter wrapper.
func (c MJMLColumn) outerTableStyle(n *node.Node) inlineStyle {
	style := inlineStyle{
		{Property: "background-color", Value: n.GetAttributeValueDefault("background-color")},
		{Property: "border", Value: n.GetAttributeValueDefault("border")},
		{Property: "border-bottom", Value: n.GetAttributeValueDefault("border-bottom")},
		{Property: "border-left", Value: n.GetAttributeValueDefault("border-left")},
		{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
		{Property: "border-right", Value: n.GetAttributeValueDefault("border-right")},
		{Property: "border-top", Value: n.GetAttributeValueDefault("border-top")},
		{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")},
	}

	if c.hasBorderRadius(n) {
		style = append(style, Style{Property: "border-collapse", Value: "separate"})
	}

	return style
}

func (c MJMLColumn) hasBorderRadius(n *node.Node) bool {
	value, has := n.GetAttributeValue("border-radius")

	return has && value != ""
}

func (c MJMLColumn) hasInnerBorderRadius(n *node.Node) bool {
	value, has := n.GetAttributeValue("inner-border-radius")

	return has && value != ""
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

// columnProps is MJML's props for the child it renders: the position among every
// child of its parent - raw elements included, which is how MJML counts them -
// and the number of non-raw siblings a width is divided among.
type columnProps struct {
	index   int
	first   bool
	last    bool
	sibling int
	nonRaw  int
}

func columnPropsFor(n *node.Node) columnProps {
	props := columnProps{nonRaw: len(nonRawSiblings(n)), sibling: 1}

	if n.Parent == nil {
		props.first, props.last = true, true

		return props
	}

	props.sibling = len(n.Parent.Children)

	for i, child := range n.Parent.Children {
		if child == n {
			props.index = i

			break
		}
	}

	props.first = props.index == 0
	props.last = props.index+1 == props.sibling

	return props
}

// hasColumnGutter reports whether the enclosing mj-section handed this column a
// gutter to share with its siblings. MJML treats an empty gutter as no gutter.
func hasColumnGutter(ctx *RenderContext) bool {
	return ctx.Gutter != ""
}

// columnClass registers and returns the classes that place this column: its
// width class, preceded by the class carrying its share of the gutter, if the
// section has one. A gutter column is narrower than its declared width, so its
// width class is derived from the reduced width.
func (c MJMLColumn) columnClass(ctx *RenderContext, n *node.Node) (string, error) {
	unit, parsedWidth, err := getParsedWidth(n)
	if err != nil {
		return "", err
	}

	if hasColumnGutter(ctx) {
		unit, parsedWidth, err = c.desktopWidth(ctx, n)
		if err != nil {
			return "", err
		}
	}

	normalized := parsedWidth
	if unit == "px" {
		normalized = normalizePxValue(parsedWidth)
	}

	width := formatNumber(normalized)

	widthClass := "mj-column-px-" + width
	if unit == "%" {
		widthClass = "mj-column-per-" + width
	}

	// The class name spells the width with dashes, the rule keeps the point.
	widthClass = strings.ReplaceAll(widthClass, ".", "-")

	ctx.AddMediaQuery(widthClass, []string{
		fmt.Sprintf("width: %s%s !important", width, unit),
		fmt.Sprintf("max-width: %s%s", width, unit),
	})

	if !hasColumnGutter(ctx) {
		return widthClass, nil
	}

	gutterClass, err := c.desktopGutterClassName(ctx, n)
	if err != nil {
		return "", err
	}

	// Group columns carry their padding inline, so they need no extra rule.
	if !ctx.InGroup {
		padding, err := c.desktopPadding(ctx, n)
		if err != nil {
			return "", err
		}

		ctx.AddMediaQuery(gutterClass, []string{"padding: " + padding + " !important"})
	}

	return gutterClass + " " + widthClass, nil
}

// mobileWidth is the width a column takes when the columns stack: 100% of the
// width their container gives them, except inside a group, which does not stack.
func (c MJMLColumn) mobileWidth(ctx *RenderContext, n *node.Node) (string, error) {
	if n.GetAttributeValueDefault("mobileWidth") != "mobileWidth" {
		return "100%", nil
	}

	if ctx.InGroup && hasColumnGutter(ctx) {
		unit, parsedWidth, err := c.desktopWidth(ctx, n)
		if err != nil {
			return "", err
		}

		if unit == "%" {
			return formatNumber(parsedWidth) + "%", nil
		}

		containerWidth, err := parseIntPrefix(ctx.ContainerWidth)
		if err != nil {
			return "", err
		}

		return formatNumber(normalizeUnitValue(parsedWidth/containerWidth*100)) + "%", nil
	}

	props := columnPropsFor(n)

	width, hasWidth := n.GetAttributeValue("width")
	if !hasWidth {
		return fmt.Sprintf("%d%%", 100/props.nonRaw), nil
	}

	parsedWidth, unit, err := parseWidth(width)
	if err != nil {
		return "", err
	}

	if unit == "%" {
		return width, nil
	}

	containerWidth, err := parseIntPrefix(ctx.ContainerWidth)
	if err != nil {
		return "", err
	}

	return formatNumber(normalizeUnitValue(parsedWidth/containerWidth*100)) + "%", nil
}

// desktopWidth is the width a column takes next to its siblings, with its share
// of the gutter taken out. Whole pixels are distributed so the columns still add
// up to the container.
func (c MJMLColumn) desktopWidth(ctx *RenderContext, n *node.Node) (string, float64, error) {
	unit, parsedWidth, err := getParsedWidth(n)
	if err != nil {
		return "", 0, err
	}

	if !hasColumnGutter(ctx) {
		if unit == "px" {
			return unit, normalizePxValue(parsedWidth), nil
		}

		return unit, parsedWidth, nil
	}

	gutter, err := c.normalizeGutterValue(ctx, unit)
	if err != nil {
		return "", 0, err
	}

	props := columnPropsFor(n)
	reduction := gutter * float64(props.sibling-1) / float64(props.sibling)

	reduced := math.Max(0, normalizeUnitValue(parsedWidth-reduction))

	if unit != "px" {
		return unit, reduced, nil
	}

	floorWidth := math.Floor(reduced)
	fractional := reduced - floorWidth
	extraPixels := math.Max(0, math.Min(float64(props.sibling), math.Round(float64(props.sibling)*fractional)))

	if float64(props.index) < extraPixels {
		floorWidth++
	}

	return unit, floorWidth, nil
}

// normalizeGutterValue converts the container's gutter into targetUnit.
func (c MJMLColumn) normalizeGutterValue(ctx *RenderContext, targetUnit string) (float64, error) {
	if ctx.Gutter == "" {
		return 0, nil
	}

	parsedWidth, unit, err := parseWidthFloat(ctx.Gutter)
	if err != nil {
		return 0, err
	}

	if unit == targetUnit {
		return parsedWidth, nil
	}

	containerWidth, err := parseFloatWidth(ctx.ContainerWidth)
	if err != nil {
		return 0, err
	}

	switch {
	case targetUnit == "%" && unit == "px":
		return parsedWidth / containerWidth * 100, nil
	case targetUnit == "px" && unit == "%":
		return containerWidth * parsedWidth / 100, nil
	default:
		return parsedWidth, nil
	}
}

// desktopGutterClassName names the rule that carries this column's share of the
// gutter. The name spells out the sibling count, the position, the unit and the
// gutter itself, so two sections that need the same padding share one rule.
func (c MJMLColumn) desktopGutterClassName(ctx *RenderContext, n *node.Node) (string, error) {
	gutterUnit, err := c.desktopUnit(n)
	if err != nil {
		return "", err
	}

	gutter, err := c.normalizeGutterValue(ctx, gutterUnit)
	if err != nil {
		return "", err
	}

	gutterUnitToken := gutterUnit
	if gutterUnit == "%" {
		gutterUnitToken = "per"
	}

	directionToken := ""
	if ctx.ContainerDirection == "rtl" {
		directionToken = "-rtl"
	}

	normalizedGutter := gutter
	if gutterUnit == "px" {
		normalizedGutter = normalizePxValue(gutter)
	}

	gutterToken := strings.ReplaceAll(formatNumber(normalizeUnitValue(normalizedGutter)), ".", "-")

	props := columnPropsFor(n)

	return fmt.Sprintf(
		"mj-column-gutter-%d-%d-%s-%s%s",
		props.sibling, props.index+1, gutterUnitToken, gutterToken, directionToken,
	), nil
}

func (c MJMLColumn) desktopUnit(n *node.Node) (string, error) {
	unit, _, err := getParsedWidth(n)

	return unit, err
}

// paddingValues is one side of a gutter share, in the unit the caller asked for.
type paddingValues struct {
	top    float64
	right  float64
	bottom float64
	left   float64
}

// desktopPaddingValues splits the gutter between neighbouring columns on
// desktop: the outer edges get nothing, and each inside edge gets the half that
// faces it. Pixels are split so the two halves are whole numbers.
func (c MJMLColumn) desktopPaddingValues(ctx *RenderContext, n *node.Node, unit string) (paddingValues, error) {
	gutter, err := c.normalizeGutterValue(ctx, unit)
	if err != nil {
		return paddingValues{}, err
	}

	props := columnPropsFor(n)
	if props.sibling == 1 {
		return paddingValues{}, nil
	}

	halfLeading, halfTrailing := gutter/2, gutter/2
	if unit == "px" {
		normalized := normalizePxValue(gutter)
		halfLeading, halfTrailing = math.Ceil(normalized/2), math.Floor(normalized/2)
	}

	if ctx.ContainerDirection == "rtl" {
		return paddingValues{
			right: pickWhen(props.first, 0, halfTrailing),
			left:  pickWhen(props.last, 0, halfLeading),
		}, nil
	}

	return paddingValues{
		right: pickWhen(props.last, 0, halfLeading),
		left:  pickWhen(props.first, 0, halfTrailing),
	}, nil
}

// mobilePaddingValues stacks the gutter between columns instead of beside them:
// the space goes above and below, and never on the outer edges.
func (c MJMLColumn) mobilePaddingValues(ctx *RenderContext, n *node.Node) (paddingValues, error) {
	gutter, err := c.normalizeGutterValue(ctx, "%")
	if err != nil {
		return paddingValues{}, err
	}

	props := columnPropsFor(n)
	half := gutter / 2

	return paddingValues{
		top:    pickWhen(props.first, 0, half),
		bottom: pickWhen(props.last, 0, half),
	}, nil
}

func (c MJMLColumn) desktopPadding(ctx *RenderContext, n *node.Node) (string, error) {
	unit, err := c.desktopUnit(n)
	if err != nil {
		return "", err
	}

	values, err := c.desktopPaddingValues(ctx, n, unit)
	if err != nil {
		return "", err
	}

	return formatPaddingValues(values, unit), nil
}

func (c MJMLColumn) mobilePadding(ctx *RenderContext, n *node.Node) (string, error) {
	values, err := c.mobilePaddingValues(ctx, n)
	if err != nil {
		return "", err
	}

	return formatPaddingValues(values, "%"), nil
}

// mobileGutterStyle is the padding a gutter column carries inline, which is what
// makes the margin collapse on small screens without another media query.
func (c MJMLColumn) mobileGutterStyle(ctx *RenderContext, n *node.Node) (inlineStyle, error) {
	if !hasColumnGutter(ctx) {
		return nil, nil
	}

	// Group columns do not stack, so they keep their desktop padding.
	if ctx.InGroup {
		padding, err := c.desktopPadding(ctx, n)
		if err != nil {
			return nil, err
		}

		return inlineStyle{{Property: "padding", Value: padding}}, nil
	}

	padding, err := c.mobilePadding(ctx, n)
	if err != nil {
		return nil, err
	}

	return inlineStyle{{Property: "padding", Value: padding}}, nil
}

// outlookGutterStyle is the same split in whole pixels, for the Outlook seat.
func (c MJMLColumn) outlookGutterStyle(ctx *RenderContext, n *node.Node) (inlineStyle, error) {
	if !hasColumnGutter(ctx) {
		return nil, nil
	}

	values, err := c.desktopPaddingValues(ctx, n, "px")
	if err != nil {
		return nil, err
	}

	return inlineStyle{{Property: "padding", Value: formatPaddingValues(values, "px")}}, nil
}

// tdOutlookStyle is the style of the Outlook seat the enclosing section draws
// around this column.
func (c MJMLColumn) tdOutlookStyle(ctx *RenderContext, n *node.Node) (inlineStyle, error) {
	style := inlineStyle{{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")}}

	width, err := getWidthAsPixel(ctx, n)
	if err != nil {
		return nil, err
	}

	style = append(style, Style{Property: "width", Value: width})

	gutter, err := c.outlookGutterStyle(ctx, n)
	if err != nil {
		return nil, err
	}

	return append(style, gutter...), nil
}

// formatPaddingValues renders a padding quadruple, rounding pixels and trimming
// unit values the way MJML does.
func formatPaddingValues(values paddingValues, unit string) string {
	if unit == "px" {
		return fmt.Sprintf(
			"%spx %spx %spx %spx",
			formatNumber(normalizePxValue(values.top)),
			formatNumber(normalizePxValue(values.right)),
			formatNumber(normalizePxValue(values.bottom)),
			formatNumber(normalizePxValue(values.left)),
		)
	}

	return fmt.Sprintf(
		"%s%s %s%s %s%s %s%s",
		formatNumber(normalizeUnitValue(values.top)), unit,
		formatNumber(normalizeUnitValue(values.right)), unit,
		formatNumber(normalizeUnitValue(values.bottom)), unit,
		formatNumber(normalizeUnitValue(values.left)), unit,
	)
}

// pickWhen returns a when condition holds and b otherwise.
func pickWhen(condition bool, a, b float64) float64 {
	if condition {
		return a
	}

	return b
}

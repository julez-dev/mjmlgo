package component

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/julez-dev/mjmlgo/node"
)

type inlineStyle []Style

func (s inlineStyle) InlineString() string {
	if len(s) == 0 {
		return ""
	}

	b := strings.Builder{}
	for _, style := range s {
		if style.Value == "" {
			continue // Skip empty styles
		}

		b.WriteString(style.Property)
		b.WriteString(":")
		b.WriteString(style.Value)
		if style.Important {
			b.WriteString("!important")
		}
		b.WriteString(";")
	}

	return b.String()
}

type inlineAttributes map[string]string

func (ia inlineAttributes) InlineString() string {
	if len(ia) == 0 {
		return ""
	}

	var result string
	for key, value := range ia {
		if value == "" {
			continue
		}

		result += fmt.Sprintf("%s=\"%s\" ", key, value)
	}
	return strings.TrimSpace(result)
}

var isPercentageRegex = regexp.MustCompile(`^\d+(\.\d+)?%$`)

func isPercentage(s string) bool {
	return isPercentageRegex.MatchString(s)
}

func getBackgroundString(n *node.Node) string {
	x, y := getBackgroundPosition(n)

	return x + " " + y
}

func getBackgroundPosition(n *node.Node) (string, string) {
	x, y := parseBackgroundPosition(n.GetAttributeValueDefault("background-position"))

	if x1, ok := n.GetAttributeValue("background-position-x"); ok {
		x = x1
	}

	if y1, ok := n.GetAttributeValue("background-position-y"); ok {
		y = y1
	}

	return x, y
}

func parseBackgroundPosition(position string) (string, string) {
	if position == "" {
		return "center", "center"
	}

	posSplit := strings.Split(position, " ")

	if len(posSplit) == 1 {
		var val = posSplit[0]

		if slices.Contains([]string{"top", "bottom"}, val) {
			return "center", val
		}

		return val, "center"
	}

	if len(posSplit) == 2 {
		val1, val2 := posSplit[0], posSplit[1]

		if slices.Contains([]string{"top", "bottom"}, val1) || (val1 == "center" && slices.Contains([]string{"left", "right"}, val2)) {
			return val2, val1
		}

		return val1, val2
	}

	// more than 2 values is not supported, let's treat as default value
	return "center", "top"
}

func addSuffixToClasses(classes string, suffix string) string {
	if classes == "" {
		return ""
	}

	classSlice := strings.Split(classes, " ")

	for i, c := range classSlice {
		classSlice[i] = c + "-" + suffix
	}

	return strings.Join(classSlice, " ")
}

// valueAndUnitRegex is a compiled regular expression to extract the numeric
// part and the unit from a width string
// It captures a numeric part (digits, dots, commas) and a non-numeric unit part.
var valueAndUnitRegex = regexp.MustCompile(`^([\d.,]*)(\D*)$`)

// parseWidth parses a string representation of a width (e.g., "100px", "50.5%")
// into a numeric value and its unit.
func parseWidth(width string) (float64, string, error) {
	// extract the numeric string and unit using the regex
	matches := valueAndUnitRegex.FindStringSubmatch(width)

	if len(matches) < 3 {
		return 0, "", fmt.Errorf("invalid width format: %s", width)
	}
	numericStr := matches[1]
	unit := matches[2]

	// parse the numeric string into a float64
	var parsedValue float64
	if numericStr != "" {
		var err error
		parsedValue, err = strconv.ParseFloat(numericStr, 64)
		if err != nil {
			return 0, "", fmt.Errorf("could not parse numeric value from '%s': %w", width, err)
		}
	}

	parsedValue = math.Trunc(parsedValue)

	// default the unit to "px" if it was not present.
	if unit == "" {
		unit = "px"
	}

	return parsedValue, unit, nil
}

func removeNonNumeric(s string) string {
	var builder strings.Builder

	builder.Grow(len(s))

	for _, r := range s {
		if unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func getContentWidth(ctx *RenderContext, n *node.Node) (int, error) {
	widthStr, ok := n.GetAttributeValue("width")

	var width int

	if !ok {
		width = math.MaxInt
	} else {
		p, err := strconv.Atoi(removeNonNumeric(widthStr))
		if err != nil {
			width = math.MaxInt
		} else {
			width = p
		}
	}

	widths, err := getBoxWidths(ctx, n)
	if err != nil {
		return 0, fmt.Errorf("failed to get box widths: %w", err)
	}

	return min(width, widths["box"]), nil
}

func getBoxWidths(ctx *RenderContext, n *node.Node) (map[string]int, error) {
	parsedWidth, err := strconv.Atoi(removeNonNumeric(ctx.ContainerWidth))
	if err != nil {
		return nil, fmt.Errorf("invalid container width: %w", err)
	}

	paddingLeft, err := getShorthandAttrValue(n, "padding", "left")
	if err != nil {
		return nil, fmt.Errorf("failed to get padding left: %w", err)
	}
	paddingRight, err := getShorthandAttrValue(n, "padding", "right")
	if err != nil {
		return nil, fmt.Errorf("failed to get padding right: %w", err)
	}

	paddings := paddingLeft + paddingRight

	borderRight := getShorthandBorderValue(n, "right", "border")
	borderLeft := getShorthandBorderValue(n, "left", "border")
	borders := borderRight + borderLeft

	return map[string]int{
		"totalWidth": parsedWidth,
		"borders":    borders,
		"paddings":   paddings,
		"box":        parsedWidth - paddings - borders,
	}, nil
}

func getShorthandBorderValue(n *node.Node, direction, attribute string) int {
	if attribute == "" {
		attribute = "border"
	}

	borderDirection := n.GetAttributeValueDefault(attribute + "-" + direction)
	border := n.GetAttributeValueDefault(attribute)

	if borderDirection != "" {
		return borderParser(borderDirection)
	}
	if border != "" {
		return borderParser(border)
	}

	return 0
}

func getShorthandAttrValue(n *node.Node, attr string, direction string) (int, error) {
	attributeDirection := n.GetAttributeValueDefault(attr + "-" + direction)
	attribute := n.GetAttributeValueDefault(attr)

	if attributeDirection != "" {
		p, err := strconv.Atoi(removeNonNumeric(attributeDirection))
		if err != nil {
			return 0, err
		}
		return p, nil
	}

	if attribute == "" {
		return 0, nil
	}

	p, err := shorthandParser(attribute, direction)
	if err != nil {
		return 0, err
	}

	return p, nil
}

var whitespaceRegex = regexp.MustCompile(`\s+`)

func shorthandParser(cssValue, direction string) (int, error) {
	splitCSSValue := strings.SplitN(whitespaceRegex.ReplaceAllString(strings.TrimSpace(cssValue), " "), " ", 4)

	dirs := make(map[string]int)

	switch len(splitCSSValue) {
	case 2:
		dirs["top"] = 0
		dirs["bottom"] = 0
		dirs["left"] = 1
		dirs["right"] = 1
	case 3:
		dirs["top"] = 0
		dirs["left"] = 1
		dirs["right"] = 1
		dirs["bottom"] = 2
	case 4:
		dirs["top"] = 0
		dirs["right"] = 1
		dirs["bottom"] = 2
		dirs["left"] = 3
	case 1:
		p, err := strconv.Atoi(removeNonNumeric(cssValue))
		if err != nil {
			return 0, err
		}
		return p, nil
	}

	p, err := strconv.Atoi(removeNonNumeric(splitCSSValue[dirs[direction]]))
	if err != nil {
		return 0, err
	}

	return p, nil
}

var borderWidthRegex = regexp.MustCompile(`(?:(?:^| )(\d+))`)

// borderParser extracts the integer value for a border's width from a CSS
// border shorthand string (e.g., "1px solid black").
func borderParser(border string) int {
	matches := borderWidthRegex.FindStringSubmatch(border)

	if len(matches) < 2 {
		return 0
	}

	width, err := strconv.Atoi(removeNonNumeric(matches[1]))
	if err != nil {
		return 0
	}

	return width
}

func nonRawSiblings(n *node.Node) []*node.Node {
	if n.Parent == nil {
		return nil
	}

	var sibs []*node.Node
	for _, child := range n.Parent.Children {
		if child.Type != RawTagName /* && child != n*/ {
			sibs = append(sibs, child)
		}
	}
	return sibs
}

func getColumnClass(ctx *RenderContext, n *node.Node) (string, error) {
	var class string

	unit, parsedWidth, err := getParsedWidth(n)
	if err != nil {
		return "", err
	}

	formatted := strings.ReplaceAll(strconv.FormatFloat(parsedWidth, 'f', -1, 64), ".", "-")

	if unit == "%" {
		class = fmt.Sprintf("mj-column-per-%s", formatted)
	} else {
		class = fmt.Sprintf("mj-column-px-%s", formatted)
	}

	ctx.MJMLStylesheet[class] = []string{
		fmt.Sprintf("width: %s%s !important", formatted, unit),
		fmt.Sprintf("max-width: %s%s", formatted, unit),
	}

	return class, nil
}

func getParsedWidth(n *node.Node) (string, float64, error) {
	numberSiblings := len(nonRawSiblings(n))

	var width string
	if w, ok := n.GetAttributeValue("width"); ok {
		width = w
	} else {
		width = fmt.Sprintf("%d%%", 100/numberSiblings)
	}

	parsedWidth, unit, err := parseWidth(width)
	if err != nil {
		return "", 0, err
	}

	return unit, parsedWidth, nil
}

func getWidthAsPixel(ctx *RenderContext, n *node.Node) (string, error) {
	containerWidth := strings.TrimSuffix(ctx.ContainerWidth, "px")

	unit, width, err := getParsedWidth(n)
	if err != nil {
		return "", err
	}

	floatAsStr := func(floatValue float64) string {
		return strings.TrimSuffix(fmt.Sprintf("%.2f", floatValue), ".00")
	}

	if unit == "%" {
		cWidthFloat, err := strconv.ParseFloat(containerWidth, 64)
		if err != nil {
			return "", err
		}

		p := (cWidthFloat * width) / 100.0

		return fmt.Sprintf("%spx", floatAsStr(p)), nil
	}

	return fmt.Sprintf("%spx", floatAsStr(width)), nil
}

func getMobileWidth(ctx *RenderContext, n *node.Node) (string, error) {
	containerWidth := removeNonNumeric(ctx.ContainerWidth)
	numberSiblings := len(nonRawSiblings(n))
	width, hasWidth := n.GetAttributeValue("width")
	mobileWidth := n.GetAttributeValueDefault("mobileWidth")

	if mobileWidth != "mobileWidth" {
		return "100%", nil
	}

	if !hasWidth {
		r := 100 / numberSiblings
		return fmt.Sprintf("%d%%", r), nil
	}

	parsedWidth, unit, err := parseWidth(width)
	if err != nil {
		return "", err
	}

	if unit == "%" {
		return width, nil
	} else {
		// Convert pixel width to percentage based on container width
		// containerWidth as float
		pf, err := strconv.ParseFloat(containerWidth, 64)
		if err != nil {
			return "", fmt.Errorf("invalid container width: %w", err)
		}

		percentage := (parsedWidth / pf) * 100
		return fmt.Sprintf("%.2f%%", percentage), nil
	}
}

// makeBackgroundString filters out empty strings from a slice
// and then joins the remaining elements into a single, space-separated string.
func makeBackgroundString(parts []string) string {
	var filteredParts []string
	for _, part := range parts {
		if part != "" {
			filteredParts = append(filteredParts, part)
		}
	}

	return strings.Join(filteredParts, " ")
}

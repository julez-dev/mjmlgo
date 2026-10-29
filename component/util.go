package component

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/julez-dev/mjmlgo/v2/node"
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

		if strings.HasPrefix(style.Value, "#") {
			r, err := expandHexShortcut(style.Value)
			if err == nil {
				style.Value = r
			}
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
func expandHexShortcut(hexCode string) (string, error) {
	// 1. Ensure the code starts with '#'
	if !strings.HasPrefix(hexCode, "#") {
		return "", fmt.Errorf("invalid format: hex code must start with '#'")
	}

	// 2. Get the characters after '#'
	chars := hexCode[1:]
	length := len(chars)

	// 3. Validate that all characters are valid hex digits
	for _, r := range chars {
		if !unicode.Is(unicode.Hex_Digit, r) {
			return "", fmt.Errorf("invalid character: '%c' is not a hexadecimal digit", r)
		}
	}

	// 4. Expand based on length
	switch length {
	case 3, 4: // This is a shortcut (#RGB or #RGBA)
		var builder strings.Builder
		builder.WriteString("#")
		// Double each character (e.g., F -> FF)
		for _, char := range chars {
			builder.WriteRune(char)
			builder.WriteRune(char)
		}
		return builder.String(), nil
	case 6, 8: // This is already a full-length code (#RRGGBB or #RRGGBBAA)
		// Just return it in a consistent case
		return hexCode, nil
	default: // Any other length is invalid
		return "", fmt.Errorf("invalid length: must have 3, 4, 6, or 8 characters after '#'")
	}
}

type inlineAttributes map[string]string

func (ia inlineAttributes) InlineString() string {
	if len(ia) == 0 {
		return ""
	}

	// An element carries a handful of attributes, so the keys fit on the stack:
	// sorting them in place gives the deterministic order the output needs
	// without a heap allocation per element.
	var stack [16]string

	keys := stack[:0]
	for key := range ia {
		keys = append(keys, key)
	}

	sortStrings(keys)

	var b strings.Builder
	for _, key := range keys {
		value := ia[key]
		if value == "" {
			continue
		}

		if key == "bgcolor" || key == "color" {
			if expanded, err := expandHexShortcut(value); err == nil {
				value = expanded
			}
		}

		if b.Len() > 0 {
			b.WriteByte(' ')
		}

		b.WriteString(key)
		b.WriteString(`="`)
		b.WriteString(value)
		b.WriteByte('"')
	}

	return b.String()
}

// sortStrings sorts keys in place. It exists instead of slices.Sort so that the
// caller's stack-allocated key buffer does not escape through a generic
// call; the slices are a handful of strings long, so insertion sort wins.
func sortStrings(keys []string) {
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
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

	// Classes are separated by single spaces, and the separator is preserved as
	// it was: a doubled space keeps its empty class.
	var b strings.Builder
	b.Grow(len(classes) + 8)

	for {
		next := strings.IndexByte(classes, ' ')
		if next < 0 {
			b.WriteString(classes)
			b.WriteByte('-')
			b.WriteString(suffix)

			return b.String()
		}

		b.WriteString(classes[:next])
		b.WriteByte('-')
		b.WriteString(suffix)
		b.WriteByte(' ')
		classes = classes[next+1:]
	}
}

// valueAndUnitRegex is a compiled regular expression to extract the numeric
// part and the unit from a width string
// It captures a numeric part (digits, dots, commas) and a non-numeric unit part.
var valueAndUnitRegex = regexp.MustCompile(`^([\d.,]*)(\D*)$`)

// parseWidth parses a string representation of a width (e.g., "100px", "50.5%")
// into a numeric value and its unit, truncating the value.
func parseWidth(width string) (float64, string, error) {
	return parseWidthOptions(width, false)
}

// parseWidthFloat keeps a fractional percentage, the way MJML's widthParser does
// when a caller passes parseFloatToInt: false. Pixel widths still truncate:
// widthParser always reads those with parseInt, whatever the caller asked for, so
// "200.7px" is 200.
func parseWidthFloat(width string) (float64, string, error) {
	return parseWidthOptions(width, true)
}

func parseWidthOptions(width string, keepPercentageFloat bool) (float64, string, error) {
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

	// default the unit to "px" if it was not present.
	if unit == "" {
		unit = "px"
	}

	if unit != "%" || !keepPercentageFloat {
		parsedValue = math.Trunc(parsedValue)
	}

	return parsedValue, unit, nil
}

// parseIntWidth reads a leading integer from a CSS length the way JavaScript's
// parseInt does, stopping at the first non-digit. Unlike RemoveNonNumeric it
// handles fractional widths such as "174.24px" correctly (174).
func parseIntWidth(s string) (int, error) {
	s = strings.TrimSpace(s)

	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}

	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}

	if i == start {
		return 0, fmt.Errorf("invalid width: %q", s)
	}

	return strconv.Atoi(s[:i])
}

// parseFloatWidth reads a leading floating point value from a CSS length,
// stopping at the unit. It mirrors parseFloat("600px") / parseFloat("174.24px").
func parseFloatWidth(s string) (float64, error) {
	s = strings.TrimSpace(s)

	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}

	start := i
	seenDot := false
	for i < len(s) {
		c := s[i]
		if c >= '0' && c <= '9' {
			i++
			continue
		}
		if c == '.' && !seenDot {
			seenDot = true
			i++
			continue
		}
		break
	}

	if i == start {
		return 0, fmt.Errorf("invalid width: %q", s)
	}

	return strconv.ParseFloat(s[:i], 64)
}

// formatNumber renders a float like JavaScript's Number#toString for the values
// used here: no trailing zeros and no exponent.
func formatNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// formatPixelWidth formats a numeric pixel value as "<value>px".
func formatPixelWidth(v float64) string {
	return formatNumber(v) + "px"
}

func RemoveNonNumeric(s string) string {
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
		p, err := strconv.Atoi(RemoveNonNumeric(widthStr))
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
	parsedWidth, err := parseIntWidth(ctx.ContainerWidth)
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
		p, err := strconv.Atoi(RemoveNonNumeric(attributeDirection))
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

func shorthandParser(cssValue, direction string) (int, error) {
	// Whitespace runs separate the shorthands, and at most four are meaningful:
	// anything past the fourth belongs to the last value, like SplitN did.
	var parts [4]string

	count := splitWhitespace(cssValue, parts[:])

	if count <= 1 {
		p, err := strconv.Atoi(RemoveNonNumeric(cssValue))
		if err != nil {
			return 0, err
		}

		return p, nil
	}

	p, err := strconv.Atoi(RemoveNonNumeric(parts[shorthandIndex(direction, count)]))
	if err != nil {
		return 0, err
	}

	return p, nil
}

// shorthandIndex is the position the value of direction occupies in a CSS
// shorthand with count values: one value applies to all sides, two split
// vertical and horizontal, three and four follow the CSS order.
func shorthandIndex(direction string, count int) int {
	switch count {
	case 2:
		switch direction {
		case "top", "bottom":
			return 0
		default:
			return 1
		}
	case 3:
		switch direction {
		case "top":
			return 0
		case "bottom":
			return 2
		default:
			return 1
		}
	default: // 4, or a longer value whose tail is part of the fourth
		switch direction {
		case "right":
			return 1
		case "bottom":
			return 2
		case "left":
			return 3
		default:
			return 0
		}
	}
}

// splitWhitespace splits s on runs of whitespace, ignoring leading and trailing
// whitespace. It fills at most len(dst) parts and returns how many it wrote; the
// last part keeps any remainder, so a value with more parts than dst loses
// nothing.
func splitWhitespace(s string, dst []string) int {
	n := 0
	i := 0

	for {
		for i < len(s) && isSpaceByte(s[i]) {
			i++
		}

		if i == len(s) {
			return n
		}

		start := i
		for i < len(s) && !isSpaceByte(s[i]) {
			i++
		}

		if n == len(dst)-1 {
			// Everything left, including the separators, belongs to the last
			// part: a trailing space is trimmed, as before.
			dst[n] = strings.TrimRight(s[start:], " \t\n\r\f\v")
			return n + 1
		}

		dst[n] = s[start:i]
		n++
	}
}

func isSpaceByte(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	}

	return false
}

var borderWidthRegex = regexp.MustCompile(`(?:(?:^| )(\d+))`)

// borderParser extracts the integer value for a border's width from a CSS
// border shorthand string (e.g., "1px solid black").
func borderParser(border string) int {
	matches := borderWidthRegex.FindStringSubmatch(border)

	if len(matches) < 2 {
		return 0
	}

	width, err := strconv.Atoi(RemoveNonNumeric(matches[1]))
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

func getParsedWidth(n *node.Node) (string, float64, error) {
	numberSiblings := len(nonRawSiblings(n))

	var width string
	if w, ok := n.GetAttributeValue("width"); ok {
		width = w
	} else {
		// MJML divides as a JavaScript number, so three equal columns are
		// 33.333333333333336% wide and get a class name to match.
		width = formatNumber(100/float64(numberSiblings)) + "%"
	}

	parsedWidth, unit, err := parseWidthFloat(width)
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

	if unit == "%" {
		cWidthFloat, err := strconv.ParseFloat(containerWidth, 64)
		if err != nil {
			return "", err
		}

		return formatPixelWidth(normalizePxValue((cWidthFloat * width) / 100.0)), nil
	}

	return formatPixelWidth(normalizePxValue(width)), nil
}

// normalizePxValue rounds a pixel value the way MJML does: Math.round, because a
// fractional CSS pixel in an email is noise.
func normalizePxValue(v float64) float64 {
	return math.Round(v)
}

// normalizeUnitValue trims a unit value to six decimals, matching MJML's
// Number(parseFloat(value).toFixed(6)), so a width percentage does not carry
// floating point dust into the markup.
func normalizeUnitValue(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}

// parseIntPrefix reads the leading integer of a CSS length the way JavaScript's
// parseInt(containerWidth, 10) does: "533.99px" is 533, not 533.99.
func parseIntPrefix(s string) (float64, error) {
	s = strings.TrimSpace(s)

	end := 0
	if end < len(s) && (s[end] == '+' || s[end] == '-') {
		end++
	}

	start := end
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}

	if end == start {
		return 0, fmt.Errorf("invalid width: %q", s)
	}

	return strconv.ParseFloat(s[:end], 64)
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

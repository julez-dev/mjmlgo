package component

import (
	"encoding/xml"
	"fmt"
	"strconv"
)

type RenderContext struct {
	GlobalTextAttributes []xml.Attr
	GlobalAllAttributes  []xml.Attr
	InlineStyles         []Stylesheet

	MJMLStylesheet              map[string][]string
	IncludeMobileFullWidthStyle bool
	ContainerWidth              string
	PreviewText                 string
	Breakpoint                  string

	Language  string
	Direction string
}

func (c RenderContext) makeLowerBreakpoint() string {
	parsed, err := strconv.Atoi(removeNonNumeric(c.Breakpoint))
	if err != nil {
		return c.Breakpoint
	}

	return fmt.Sprintf("%dpx", parsed-1)
}

// Stylesheet is the top-level structure for our parsed CSS.
type Stylesheet struct {
	Rules      []Rule      `json:"rules"`
	MediaRules []MediaRule `json:"media_rules"`
}

// MediaRule represents an @media block.
type MediaRule struct {
	Condition string `json:"condition"`
	Rules     []Rule `json:"rules"`
}

// Rule represents a standard CSS rule with selectors and declarations.
type Rule struct {
	Selectors    string  `json:"selectors"`
	Declarations []Style `json:"declarations"`
}

// Style represents a single CSS property-value pair.
type Style struct {
	Property  string `json:"property"`
	Value     string `json:"value"`
	Important bool   `json:"important"`
}

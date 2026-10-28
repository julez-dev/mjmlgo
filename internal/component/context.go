package component

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"sync/atomic"
)

// AttributeMap holds a set of MJML attribute values keyed by attribute name.
type AttributeMap = map[string]string

type RenderContext struct {
	InlineStyles []Stylesheet
	Fonts        map[string]string
	FontOrder    []string
	// BodyContent is the rendered body HTML, used to detect which registered
	// web fonts the document actually uses.
	BodyContent string

	// AttributeDefaults holds per-tag defaults from <mj-attributes>, keyed by
	// tag name. The <mj-all> defaults live under the "mj-all" key.
	AttributeDefaults map[string]AttributeMap
	// Classes holds the attributes of each <mj-class>, keyed by class name.
	Classes map[string]AttributeMap
	// ClassDefaults holds the per-tag defaults declared inside a <mj-class>,
	// keyed by class name and then by tag name.
	ClassDefaults map[string]map[string]AttributeMap
	// HTMLAttributes holds the <mj-html-attributes> selectors in declaration
	// order. Selectors with the same path are merged in place.
	HTMLAttributes []HTMLAttributeSelector

	MJMLStylesheet map[string][]string
	MJMLStyleOrder []string
	// ComponentHeadStyles holds the named component-level <style> blocks (navbar,
	// accordion, ...) in first-registration order. They are keyed by component
	// name, so multiple instances of the same component register their style
	// only once, like MJML's addHeadStyle.
	ComponentHeadStyles []ComponentHeadStyle
	// InstanceComponentHeadStyles holds component stylesheets that depend on the
	// rendered instance rather than only on the breakpoint (mj-carousel, whose
	// CSS embeds the carousel id and the number of images). MJML keeps these in
	// an array without deduplication and emits them before ComponentHeadStyles.
	InstanceComponentHeadStyles []ComponentHeadStyle
	// IDGenerator produces the opaque ids MJML assigns to components that need
	// them (navbar hamburger labels, carousels). It is set once per render by
	// the renderer, so a nil generator falls back to DefaultIDGenerator.
	IDGenerator                 func() string
	IncludeMobileFullWidthStyle bool
	ForceOWADesktop             bool
	ContainerWidth              string
	PreviewText                 string
	Title                       string
	Breakpoint                  string

	Language  string
	Direction string

	// accordionFontFamily and elementFontFamily carry the contextual font
	// inheritance of the currently rendered accordion/element. Title and text
	// fall back to them when they do not set font-family themselves.
	accordionFontFamily string
	elementFontFamily   string
}

// HTMLAttribute is one attribute set by <mj-html-attribute>.
type HTMLAttribute struct {
	Name  string
	Value string
}

// HTMLAttributeSelector is one <mj-selector> target and the attributes it sets.
type HTMLAttributeSelector struct {
	Path       string
	Attributes []HTMLAttribute
}

// addHTMLAttribute records a selector attribute, merging repeated paths and
// attribute names the same way MJML's htmlAttributes object does.
func (c *RenderContext) addHTMLAttribute(path, name, value string) {
	for i := range c.HTMLAttributes {
		if c.HTMLAttributes[i].Path != path {
			continue
		}

		for j := range c.HTMLAttributes[i].Attributes {
			if c.HTMLAttributes[i].Attributes[j].Name == name {
				c.HTMLAttributes[i].Attributes[j].Value = value
				return
			}
		}

		c.HTMLAttributes[i].Attributes = append(c.HTMLAttributes[i].Attributes, HTMLAttribute{Name: name, Value: value})

		return
	}

	c.HTMLAttributes = append(c.HTMLAttributes, HTMLAttributeSelector{
		Path:       path,
		Attributes: []HTMLAttribute{{Name: name, Value: value}},
	})
}

// addMJMLStyle records a generated component stylesheet in insertion order.
func (c *RenderContext) addMJMLStyle(className string, rules []string) {
	if c.MJMLStylesheet == nil {
		c.MJMLStylesheet = make(map[string][]string)
	}

	if _, exists := c.MJMLStylesheet[className]; !exists {
		c.MJMLStyleOrder = append(c.MJMLStyleOrder, className)
	}

	c.MJMLStylesheet[className] = rules
}

// orderedMJMLStyles returns the generated stylesheet classes in insertion
// order. Contexts populated without an order fall back to sorted keys.
func (c RenderContext) orderedMJMLStyles() []string {
	if len(c.MJMLStyleOrder) > 0 {
		return c.MJMLStyleOrder
	}

	classes := make([]string, 0, len(c.MJMLStylesheet))
	for className := range c.MJMLStylesheet {
		classes = append(classes, className)
	}

	slices.Sort(classes)

	return classes
}

func (c RenderContext) makeLowerBreakpoint() string {
	parsed, err := strconv.Atoi(RemoveNonNumeric(c.Breakpoint))
	if err != nil {
		return c.Breakpoint
	}

	return fmt.Sprintf("%dpx", parsed-1)
}

// ComponentHeadStyle is one component-generated stylesheet. Style is rendered
// with the document breakpoint and emitted verbatim into the head.
type ComponentHeadStyle struct {
	Name  string
	Style func(breakpoint string) string
}

// addComponentHeadStyle records a component stylesheet, replacing an existing
// registration for the same component name. Registration order is preserved.
// It mirrors MJML's addHeadStyle, which is keyed by component name.
func (c *RenderContext) addComponentHeadStyle(name string, style func(breakpoint string) string) {
	for i := range c.ComponentHeadStyles {
		if c.ComponentHeadStyles[i].Name == name {
			c.ComponentHeadStyles[i].Style = style
			return
		}
	}

	c.ComponentHeadStyles = append(c.ComponentHeadStyles, ComponentHeadStyle{Name: name, Style: style})
}

// addInstanceComponentHeadStyle records a component stylesheet that is specific
// to one rendered instance. Unlike addComponentHeadStyle it never merges, which
// matches MJML's addComponentHeadSyle: every instance contributes its own block.
func (c *RenderContext) addInstanceComponentHeadStyle(name string, style func(breakpoint string) string) {
	c.InstanceComponentHeadStyles = append(c.InstanceComponentHeadStyles, ComponentHeadStyle{Name: name, Style: style})
}

// renderedComponentHeadStyles renders the instance-scoped component stylesheets
// (mj-carousel) with the document breakpoint. They are emitted before the named
// ones, like MJML's [...componentsHeadStyles, ...headStyles].
func (c RenderContext) renderedInstanceComponentHeadStyles() []string {
	lines := make([]string, 0, len(c.InstanceComponentHeadStyles))
	for _, componentStyle := range c.InstanceComponentHeadStyles {
		lines = append(lines, componentStyle.Style(c.Breakpoint))
	}

	return lines
}

// renderedComponentHeadStyles renders the named component stylesheets (navbar,
// accordion, ...) with the document breakpoint.
func (c RenderContext) renderedComponentHeadStyles() []string {
	lines := make([]string, 0, len(c.ComponentHeadStyles))
	for _, componentStyle := range c.ComponentHeadStyles {
		lines = append(lines, componentStyle.Style(c.Breakpoint))
	}

	return lines
}

// nextComponentID returns the next id for a component that MJML identifies
// with a random value (navbar hamburger labels, carousels).
func (c *RenderContext) nextComponentID() string {
	if c.IDGenerator != nil {
		return c.IDGenerator()
	}

	return DefaultIDGenerator()
}

// fallbackIDCounter keeps generated ids distinct in the (practically
// impossible) event that crypto/rand fails.
var fallbackIDCounter atomic.Uint64

// DefaultIDGenerator returns the MJML-shaped id used when no generator is
// configured: 16 lowercase hex characters from crypto/rand. MJML itself uses a
// random value here, so documents are not byte-for-byte reproducible unless a
// caller injects its own generator.
func DefaultIDGenerator() string {
	var id [8]byte
	if _, err := rand.Read(id[:]); err != nil {
		binary.BigEndian.PutUint64(id[:], fallbackIDCounter.Add(1))
	}

	return hex.EncodeToString(id[:])
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

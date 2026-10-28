package component

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"sync/atomic"

	"github.com/ericchiang/css"
)

// AttributeMap holds a set of MJML attribute values keyed by attribute name.
type AttributeMap = map[string]string

// RenderContext is the state of one render: what the document's head configured,
// what the renderer passes down the component tree and what components
// accumulate for the head. The renderer creates one per document; every
// component in that document receives the same pointer.
//
// The exported fields are frozen API: their meaning and their type do not change
// (see the package documentation). A component that narrows a field for its
// children - the container width, the contextual font family - restores it
// before it returns.
type RenderContext struct {
	// InlineStyles holds the stylesheets the document asked to inline - every
	// <mj-style inline> block - in document order. The renderer applies them to
	// the finished HTML.
	InlineStyles []Stylesheet
	// Fonts maps a font family name to the URL it is imported from. It is seeded
	// with the built-in web fonts unless FontsConfigured reports a caller-supplied
	// registry; <mj-font> adds to it.
	Fonts map[string]string
	// FontOrder lists font families in emission order. Built-in and declared
	// fonts keep their registration order; a caller-supplied registry is sorted by
	// family name, because a Go map has no order of its own.
	FontOrder []string
	// FontsConfigured reports that Fonts was supplied by the caller and must
	// not be extended with the built-in web fonts. Fonts declared by the
	// document with <mj-font> are still added on top of it.
	FontsConfigured bool
	// BodyContent is the rendered body HTML, used to detect which registered
	// web fonts the document actually uses.
	BodyContent string
	// CustomComponents holds the custom components the renderer was configured
	// with, keyed by tag name. Validation and rendering look a tag up here after
	// the built-in tables, so a registered tag is a first-class element.
	//
	// The map belongs to the renderer and every render of that renderer shares
	// it, including concurrent ones: components must not modify it. A nil or
	// empty map is a renderer without custom components.
	CustomComponents map[string]Component

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

	// MJMLStylesheet maps a generated class name to the CSS rules components
	// registered for it.
	MJMLStylesheet map[string][]string
	// MJMLStyleOrder lists those class names in first-registration order; the head
	// emits them in this order, so the cascade is deterministic.
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
	IDGenerator func() string
	// IncludeMobileFullWidthStyle reports that the document contains an mj-image,
	// which adds MJML's mobile-full-width rule to the head.
	IncludeMobileFullWidthStyle bool
	// ForceOWADesktop is set by <mjml owa="desktop"> and makes the generated
	// conditional CSS target desktop Outlook.
	ForceOWADesktop bool
	// PrinterSupport repeats the generated media queries in an
	// `@media only print` block, like MJML's printerSupport option.
	PrinterSupport bool
	// ContainerWidth is the width content may occupy at the current nesting
	// depth, such as "600px". Containers narrow it for their children and restore
	// it afterwards; the renderer seeds it from the body width.
	ContainerWidth string
	// PreviewText is the text of <mj-preview>, emitted as the hidden preheader of
	// the body.
	PreviewText string
	// Title is the text of <mj-title>, used as the aria-label of the body div.
	Title string
	// Breakpoint is the document's media-query breakpoint, such as "480px".
	// <mj-breakpoint> overrides the default.
	Breakpoint string

	// Language is the document language from <mjml lang>, emitted as the lang
	// attribute of the body.
	Language string
	// Direction is the document direction from <mjml dir>, emitted as the dir
	// attribute of the body.
	Direction string

	// accordionFontFamily and elementFontFamily carry the contextual font
	// inheritance of the currently rendered accordion/element. Title and text
	// fall back to them when they do not set font-family themselves.
	accordionFontFamily string
	elementFontFamily   string
	// fluidOnMobileClasses holds the class names custom components registered
	// through AddFluidOnMobileClass, in registration order. mj-image's own class
	// stays behind the IncludeMobileFullWidthStyle flag.
	fluidOnMobileClasses []string
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

// AddMediaQuery contributes className to the document's breakpoint media query.
// It is the mechanism the built-in columns and groups use for their width
// classes: the rules are emitted inside
// `@media only screen and (min-width: <breakpoint>)` for className, and repeated
// in the Thunderbird, print and desktop-Outlook variants MJML generates. A class
// is emitted in first-registration order and registered once, so two instances
// of a component contribute one block; registering a class again replaces its
// rules.
func (c *RenderContext) AddMediaQuery(className string, declarations []string) {
	c.addMJMLStyle(className, declarations)
}

// AddComponentHeadStyle contributes a named stylesheet to the document head,
// rendered with the document breakpoint. It is the mechanism the built-in
// navbar, accordion and image components use. The block is keyed by name: a
// component registering the same name again - its second instance, for example -
// replaces the previous block and keeps its position, so the head carries one
// deterministic block per component.
func (c *RenderContext) AddComponentHeadStyle(name string, style func(breakpoint string) string) {
	c.addComponentHeadStyle(name, style)
}

// AddFluidOnMobileClass makes the head emit MJML's fluid-on-mobile rule for
// className: inside the media query below the breakpoint,
// `table.<className>` gets width:100% !important and `td.<className>` width:auto
// !important. It is the rule mj-image's fluid-on-mobile attribute uses, exposed
// so a custom component can opt its own class in. Classes are emitted in
// first-registration order, deduplicated, and after mj-image's own class when
// the document has one.
func (c *RenderContext) AddFluidOnMobileClass(className string) {
	if className == "" || slices.Contains(c.fluidOnMobileClasses, className) {
		return
	}

	c.fluidOnMobileClasses = append(c.fluidOnMobileClasses, className)
}

// fluidOnMobileClassList is the ordered, de-duplicated list of classes that
// receive the fluid-on-mobile rule: mj-image's fixed class first when the
// document has an image, then the classes custom components registered.
func (c RenderContext) fluidOnMobileClassList() []string {
	if !c.IncludeMobileFullWidthStyle && len(c.fluidOnMobileClasses) == 0 {
		return nil
	}

	classes := make([]string, 0, 1+len(c.fluidOnMobileClasses))

	if c.IncludeMobileFullWidthStyle {
		classes = append(classes, "mj-full-width-mobile")
	}

	for _, className := range c.fluidOnMobileClasses {
		if !slices.Contains(classes, className) {
			classes = append(classes, className)
		}
	}

	return classes
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

	// compiled is the selector list compiled when the stylesheet was built, and
	// compiledSet records that compilation ran. A rule whose selector text does
	// not compile keeps compiled == nil: the inliner skips such a rule, which is
	// how unparsable selector lists were always treated.
	compiled    *css.Selector
	compiledSet bool
}

// CompileSelectors compiles the rule's selector list and caches the result. The
// stylesheet builders call it once, when the stylesheet is constructed, so
// rendering never parses selectors.
func (r *Rule) CompileSelectors() {
	r.compiled, _ = css.Parse(r.Selectors)
	r.compiledSet = true
}

// CompiledSelectors returns the rule's compiled selector list and reports
// whether the rule is usable. A rule whose selectors do not compile reports
// false and must be skipped. Rules that were never compiled - stylesheets built
// by hand rather than by the head parser - are compiled on first use.
func (r *Rule) CompiledSelectors() (*css.Selector, bool) {
	if !r.compiledSet {
		r.CompileSelectors()
	}

	return r.compiled, r.compiled != nil
}

// Style represents a single CSS property-value pair.
type Style struct {
	Property  string `json:"property"`
	Value     string `json:"value"`
	Important bool   `json:"important"`
}

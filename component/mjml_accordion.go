package component

import (
	"fmt"
	"io"
	"slices"

	"github.com/julez-dev/mjmlgo/node"
)

// accordionChildrenAttributes are the attributes mj-accordion and
// mj-accordion-element pass down to their children (mjml-accordion
// handleMissingChildren/renderChildren).
var accordionChildrenAttributes = [...]string{
	"border",
	"icon-align",
	"icon-width",
	"icon-height",
	"icon-position",
	"icon-wrapped-url",
	"icon-wrapped-alt",
	"icon-unwrapped-url",
	"icon-unwrapped-alt",
}

// MJMLAccordion renders mj-accordion (node_modules/mjml-accordion/lib/Accordion.js).
type MJMLAccordion struct{}

func (a MJMLAccordion) Name() string {
	return "mj-accordion"
}

var accordionAllowedAttributes = map[string]AttributeValidator{
	"container-background-color": validateColor(),
	"border":                     validateType("string"),
	"font-family":                validateType("string"),
	"icon-align":                 validateEnum([]string{"top", "middle", "bottom"}),
	"icon-width":                 validateUnit([]string{"px", "%"}, false),
	"icon-height":                validateUnit([]string{"px", "%"}, false),
	"icon-wrapped-url":           validateType("string"),
	"icon-wrapped-alt":           validateType("string"),
	"icon-unwrapped-url":         validateType("string"),
	"icon-unwrapped-alt":         validateType("string"),
	"icon-position":              validateEnum([]string{"left", "right"}),
	"padding-bottom":             validateUnit([]string{"px", "%"}, false),
	"padding-left":               validateUnit([]string{"px", "%"}, false),
	"padding-right":              validateUnit([]string{"px", "%"}, false),
	"padding-top":                validateUnit([]string{"px", "%"}, false),
	"padding":                    validateUnit([]string{"px", "%"}, true),
}

func (a MJMLAccordion) AllowedAttributes() map[string]AttributeValidator {
	return accordionAllowedAttributes
}

func (a MJMLAccordion) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"border":             "2px solid black",
		"font-family":        "Ubuntu, Helvetica, Arial, sans-serif",
		"icon-align":         "middle",
		"icon-wrapped-url":   "https://i.imgur.com/bIXv1bk.png",
		"icon-wrapped-alt":   "+",
		"icon-unwrapped-url": "https://i.imgur.com/w4uTygT.png",
		"icon-unwrapped-alt": "-",
		"icon-position":      "right",
		"icon-height":        "32px",
		"icon-width":         "32px",
		"padding":            "10px 25px",
	}
}

func (a MJMLAccordion) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	ctx.addComponentHeadStyle(a.Name(), func(string) string {
		return accordionHeadStyle
	})

	tableStyle := inlineStyle{
		{Property: "width", Value: "100%"},
		{Property: "border-collapse", Value: "collapse"},
		{Property: "border", Value: n.GetAttributeValueDefault("border")},
		{Property: "border-bottom", Value: "none"},
		{Property: "font-family", Value: n.GetAttributeValueDefault("font-family")},
	}

	tableAttr := inlineAttributes{
		"cellspacing": "0",
		"cellpadding": "0",
		"class":       "mj-accordion",
		"style":       tableStyle.InlineString(),
	}

	previousFont := ctx.accordionFontFamily
	ctx.accordionFontFamily = n.GetAttributeValueDefault("font-family")
	defer func() { ctx.accordionFontFamily = previousFont }()

	_, _ = io.WriteString(w, `<table `+tableAttr.InlineString()+`><tbody>`)

	for _, child := range n.Children {
		if child.Type == RawTagName {
			// Accordion children sit directly in the accordion's <tbody>.
			if err := renderRawChildInTableBody(ctx, w, child); err != nil {
				return err
			}

			continue
		}

		if child.Type != AccordionElementTagName {
			continue
		}

		inheritAttributes(child, n, accordionChildrenAttributes[:])

		var element MJMLAccordionElement
		if err := InitComponent(ctx, element, child); err != nil {
			return err
		}
		if err := element.Render(ctx, w, child); err != nil {
			return fmt.Errorf("failed to render accordion element: %w", err)
		}
	}

	_, _ = io.WriteString(w, `</tbody></table>`)

	return nil
}

// inheritAttributes copies the given attributes from parent to child unless the
// child sets them itself. This mirrors renderChildren's
// {...parentAttributes, ...childAttributes} merge.
func inheritAttributes(child, parent *node.Node, names []string) {
	for _, name := range names {
		value := parent.GetAttributeValueDefault(name)
		if _, has := child.GetAttributeValue(name); !has {
			child.SetAttribute(name, value)
		}
	}
}

// MJMLAccordionElement renders mj-accordion-element.
type MJMLAccordionElement struct{}

func (e MJMLAccordionElement) Name() string {
	return "mj-accordion-element"
}

var accordionElementAllowedAttributes = map[string]AttributeValidator{
	"background-color":   validateColor(),
	"border":             validateType("string"),
	"font-family":        validateType("string"),
	"icon-align":         validateEnum([]string{"top", "middle", "bottom"}),
	"icon-width":         validateUnit([]string{"px", "%"}, false),
	"icon-height":        validateUnit([]string{"px", "%"}, false),
	"icon-wrapped-url":   validateType("string"),
	"icon-wrapped-alt":   validateType("string"),
	"icon-unwrapped-url": validateType("string"),
	"icon-unwrapped-alt": validateType("string"),
	"icon-position":      validateEnum([]string{"left", "right"}),
}

func (e MJMLAccordionElement) AllowedAttributes() map[string]AttributeValidator {
	return accordionElementAllowedAttributes
}

func (e MJMLAccordionElement) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{}
}

func (e MJMLAccordionElement) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	tdStyle := inlineStyle{
		{Property: "padding", Value: "0px"},
		{Property: "background-color", Value: n.GetAttributeValueDefault("background-color")},
	}

	labelStyle := inlineStyle{
		{Property: "font-size", Value: "13px"},
		{Property: "font-family", Value: n.GetAttributeValueDefault("font-family")},
	}

	previousFont := ctx.elementFontFamily
	ctx.elementFontFamily = n.GetAttributeValueDefault("font-family")
	defer func() { ctx.elementFontFamily = previousFont }()

	_, _ = io.WriteString(w, `<tr class="`+n.GetAttributeValueDefault("css-class")+`">`)
	_, _ = io.WriteString(w, `<td style="`+tdStyle.InlineString()+`">`)
	_, _ = io.WriteString(w, `<label class="mj-accordion-element" style="`+labelStyle.InlineString()+`">`)
	_, _ = io.WriteString(w, conditionalTag(`<input class="mj-accordion-checkbox" type="checkbox" style="display:none;" />`, true))
	_, _ = io.WriteString(w, `<div>`)

	hasTitle := slices.ContainsFunc(n.Children, func(child *node.Node) bool {
		return child.Type == AccordionTitleTagName
	})
	hasText := slices.ContainsFunc(n.Children, func(child *node.Node) bool {
		return child.Type == AccordionTextTagName
	})

	if !hasTitle {
		if err := e.renderSyntheticChild(ctx, w, n, AccordionTitleTagName); err != nil {
			return err
		}
	}

	for _, child := range n.Children {
		switch child.Type {
		case AccordionTitleTagName, AccordionTextTagName:
			inheritAttributes(child, n, accordionChildrenAttributes[:])

			if err := renderAccordionChild(ctx, w, child); err != nil {
				return err
			}
		case RawTagName:
			if err := renderRawChild(ctx, w, child); err != nil {
				return err
			}
		}
	}

	if !hasText {
		if err := e.renderSyntheticChild(ctx, w, n, AccordionTextTagName); err != nil {
			return err
		}
	}

	_, _ = io.WriteString(w, `</div></label></td></tr>`)

	return nil
}

func (e MJMLAccordionElement) renderSyntheticChild(ctx *RenderContext, w io.Writer, parent *node.Node, tag string) error {
	child := &node.Node{
		Type:   tag,
		Parent: parent,
	}

	for _, name := range accordionChildrenAttributes {
		child.SetAttribute(name, parent.GetAttributeValueDefault(name))
	}

	return renderAccordionChild(ctx, w, child)
}

func renderAccordionChild(ctx *RenderContext, w io.Writer, n *node.Node) error {
	switch n.Type {
	case AccordionTitleTagName:
		var title MJMLAccordionTitle
		if err := InitComponent(ctx, title, n); err != nil {
			return err
		}
		if err := title.Render(ctx, w, n); err != nil {
			return fmt.Errorf("failed to render accordion title: %w", err)
		}
	case AccordionTextTagName:
		var text MJMLAccordionText
		if err := InitComponent(ctx, text, n); err != nil {
			return err
		}
		if err := text.Render(ctx, w, n); err != nil {
			return fmt.Errorf("failed to render accordion text: %w", err)
		}
	}

	return nil
}

// MJMLAccordionTitle renders mj-accordion-title.
type MJMLAccordionTitle struct{}

func (t MJMLAccordionTitle) Name() string {
	return "mj-accordion-title"
}

var accordionTitleAllowedAttributes = map[string]AttributeValidator{
	"background-color": validateColor(),
	"color":            validateColor(),
	"font-size":        validateUnit([]string{"px"}, false),
	"font-family":      validateType("string"),
	"font-weight":      validateType("string"),
	"padding-bottom":   validateUnit([]string{"px", "%"}, false),
	"padding-left":     validateUnit([]string{"px", "%"}, false),
	"padding-right":    validateUnit([]string{"px", "%"}, false),
	"padding-top":      validateUnit([]string{"px", "%"}, false),
	"padding":          validateUnit([]string{"px", "%"}, true),
}

func (t MJMLAccordionTitle) AllowedAttributes() map[string]AttributeValidator {
	return accordionTitleAllowedAttributes
}

func (t MJMLAccordionTitle) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"font-size": "13px",
		"padding":   "16px",
	}
}

func (t MJMLAccordionTitle) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	tdStyle := inlineStyle{
		{Property: "width", Value: "100%"},
		{Property: "background-color", Value: n.GetAttributeValueDefault("background-color")},
		{Property: "color", Value: n.GetAttributeValueDefault("color")},
		{Property: "font-size", Value: n.GetAttributeValueDefault("font-size")},
		{Property: "font-family", Value: resolveAccordionFontFamily(ctx, n)},
		{Property: "font-weight", Value: n.GetAttributeValueDefault("font-weight")},
		{Property: "padding", Value: n.GetAttributeValueDefault("padding")},
		{Property: "padding-bottom", Value: n.GetAttributeValueDefault("padding-bottom")},
		{Property: "padding-left", Value: n.GetAttributeValueDefault("padding-left")},
		{Property: "padding-right", Value: n.GetAttributeValueDefault("padding-right")},
		{Property: "padding-top", Value: n.GetAttributeValueDefault("padding-top")},
	}

	tableStyle := inlineStyle{
		{Property: "width", Value: "100%"},
		{Property: "border-bottom", Value: n.GetAttributeValueDefault("border")},
	}

	titleCell := `<td class="` + n.GetAttributeValueDefault("css-class") + `" style="` + tdStyle.InlineString() + `">` + n.Content + `</td>`

	iconCellStyle := inlineStyle{
		{Property: "padding", Value: "16px"},
		{Property: "background", Value: n.GetAttributeValueDefault("background-color")},
		{Property: "vertical-align", Value: n.GetAttributeValueDefault("icon-align")},
	}

	imgStyle := inlineStyle{
		{Property: "display", Value: "none"},
		{Property: "width", Value: n.GetAttributeValueDefault("icon-width")},
		{Property: "height", Value: n.GetAttributeValueDefault("icon-height")},
	}

	iconCell := conditionalTag(
		`<td class="mj-accordion-ico" style="`+iconCellStyle.InlineString()+`">`+
			`<img src="`+n.GetAttributeValueDefault("icon-wrapped-url")+`" alt="`+n.GetAttributeValueDefault("icon-wrapped-alt")+`" class="mj-accordion-more" style="`+imgStyle.InlineString()+`" />`+
			`<img src="`+n.GetAttributeValueDefault("icon-unwrapped-url")+`" alt="`+n.GetAttributeValueDefault("icon-unwrapped-alt")+`" class="mj-accordion-less" style="`+imgStyle.InlineString()+`" />`+
			`</td>`, true)

	content := titleCell + iconCell
	if n.GetAttributeValueDefault("icon-position") != "right" {
		content = iconCell + titleCell
	}

	_, _ = io.WriteString(w, `<div class="mj-accordion-title"><table cellspacing="0" cellpadding="0" style="`+tableStyle.InlineString()+`"><tbody><tr>`+content+`</tr></tbody></table></div>`)

	return nil
}

// MJMLAccordionText renders mj-accordion-text.
type MJMLAccordionText struct{}

func (t MJMLAccordionText) Name() string {
	return "mj-accordion-text"
}

var accordionTextAllowedAttributes = map[string]AttributeValidator{
	"background-color": validateColor(),
	"font-size":        validateUnit([]string{"px"}, false),
	"font-family":      validateType("string"),
	"font-weight":      validateType("string"),
	"letter-spacing":   validateUnitWithNegative([]string{"px", "em"}, false),
	"line-height":      validateUnit([]string{"px", "%", ""}, false),
	"color":            validateColor(),
	"padding-bottom":   validateUnit([]string{"px", "%"}, false),
	"padding-left":     validateUnit([]string{"px", "%"}, false),
	"padding-right":    validateUnit([]string{"px", "%"}, false),
	"padding-top":      validateUnit([]string{"px", "%"}, false),
	"padding":          validateUnit([]string{"px", "%"}, true),
}

func (t MJMLAccordionText) AllowedAttributes() map[string]AttributeValidator {
	return accordionTextAllowedAttributes
}

func (t MJMLAccordionText) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"font-size":   "13px",
		"line-height": "1",
		"padding":     "16px",
	}
}

func (t MJMLAccordionText) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	tdStyle := inlineStyle{
		{Property: "background", Value: n.GetAttributeValueDefault("background-color")},
		{Property: "font-size", Value: n.GetAttributeValueDefault("font-size")},
		{Property: "font-family", Value: resolveAccordionFontFamily(ctx, n)},
		{Property: "font-weight", Value: n.GetAttributeValueDefault("font-weight")},
		{Property: "letter-spacing", Value: n.GetAttributeValueDefault("letter-spacing")},
		{Property: "line-height", Value: n.GetAttributeValueDefault("line-height")},
		{Property: "color", Value: n.GetAttributeValueDefault("color")},
		{Property: "padding", Value: n.GetAttributeValueDefault("padding")},
		{Property: "padding-bottom", Value: n.GetAttributeValueDefault("padding-bottom")},
		{Property: "padding-left", Value: n.GetAttributeValueDefault("padding-left")},
		{Property: "padding-right", Value: n.GetAttributeValueDefault("padding-right")},
		{Property: "padding-top", Value: n.GetAttributeValueDefault("padding-top")},
	}

	tableStyle := inlineStyle{
		{Property: "width", Value: "100%"},
		{Property: "border-bottom", Value: n.GetAttributeValueDefault("border")},
	}

	_, _ = io.WriteString(w, `<div class="mj-accordion-content"><table cellspacing="0" cellpadding="0" style="`+tableStyle.InlineString()+`"><tbody><tr>`+
		`<td class="`+n.GetAttributeValueDefault("css-class")+`" style="`+tdStyle.InlineString()+`">`+n.Content+`</td>`+
		`</tr></tbody></table></div>`)

	return nil
}

// resolveAccordionFontFamily mirrors AccordionTitle/AccordionText
// resolveFontFamily: an explicitly written font-family wins, otherwise the
// enclosing element, then the accordion, then the (undefined) default.
func resolveAccordionFontFamily(ctx *RenderContext, n *node.Node) string {
	if accordionHasExplicitFontFamily(ctx, n) {
		return n.GetAttributeValueDefault("font-family")
	}

	if ctx.elementFontFamily != "" {
		return ctx.elementFontFamily
	}

	return ctx.accordionFontFamily
}

// accordionHasExplicitFontFamily distinguishes a font-family written on the
// element itself from one injected by <mj-attributes>/<mj-class> (MJML checks
// props.rawAttrs, which only contains attributes from the document).
func accordionHasExplicitFontFamily(ctx *RenderContext, n *node.Node) bool {
	value, has := n.GetAttributeValue("font-family")
	if !has {
		return false
	}

	combined := make(AttributeMap)
	mergeAttributes(combined, ctx.AttributeDefaults[AllTagName])
	mergeAttributes(combined, ctx.AttributeDefaults[n.Type])
	mergeAttributes(combined, classAttributes(ctx, n.GetAttributeValueDefault("mj-class")))
	mergeAttributes(combined, classDefaultsForTag(ctx, inheritedMjClasses(n), n.Type))

	defaultValue, hasDefault := combined["font-family"]

	return !hasDefault || defaultValue != value
}

// inheritedMjClasses returns the nearest ancestor mj-class value, matching the
// parentMjClass propagation of setAttributeDefaults.
func inheritedMjClasses(n *node.Node) string {
	for parent := n.Parent; parent != nil; parent = parent.Parent {
		if value, has := parent.GetAttributeValue("mj-class"); has {
			return value
		}
	}

	return ""
}

const accordionHeadStyle = `noinput.mj-accordion-checkbox { display: block !important; }
@media yahoo, only screen and (min-width:0) {
  .mj-accordion-element { display: block; }
  input.mj-accordion-checkbox, .mj-accordion-less { display: none !important; }
  input.mj-accordion-checkbox+* .mj-accordion-title { cursor: pointer; touch-action: manipulation; -webkit-user-select: none; -moz-user-select: none; user-select: none; }
  input.mj-accordion-checkbox+* .mj-accordion-content { overflow: hidden; display: none; }
  input.mj-accordion-checkbox+* .mj-accordion-more { display: block !important; }
  input.mj-accordion-checkbox:checked+* .mj-accordion-content { display: block; }
  input.mj-accordion-checkbox:checked+* .mj-accordion-more { display: none !important; }
  input.mj-accordion-checkbox:checked+* .mj-accordion-less { display: block !important; }
}
.moz-text-html input.mj-accordion-checkbox+* .mj-accordion-title { cursor: auto; touch-action: auto; -webkit-user-select: auto; -moz-user-select: auto; user-select: auto; }
.moz-text-html input.mj-accordion-checkbox+* .mj-accordion-content { overflow: hidden; display: block; }
.moz-text-html input.mj-accordion-checkbox+* .mj-accordion-ico { display: none; }
@goodbye { @gmail }`

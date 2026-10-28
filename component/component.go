package component

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/julez-dev/mjmlgo/node"
)

// Component is one MJML element that knows how to render itself. The renderer
// looks a component up by tag name, validates the element's author attributes
// with InitComponent and then calls Render, which writes the element's HTML to
// w.
//
// It is the custom-component contract: a host implements it and registers the
// component on a renderer with mjmlgo.WithComponents. A registered tag is a
// first-class element - it is validated like a built-in one, receives the
// defaults declared in mj-attributes and mj-class, and renders at its document
// position with Render. The node the renderer hands to Render is the one the
// built-in components receive: attribute defaults are already applied and the
// children are the parsed child elements. A component that implements it is a
// leaf unless it also implements ChildComponent.
type Component interface {
	// Name is the element's tag name, for example "mj-badge". It must be
	// non-empty, must not be registered twice on one renderer and must not
	// shadow a built-in MJML element.
	Name() string
	// AllowedAttributes maps every attribute the component accepts to its
	// validator. An author attribute that is not in the map is rejected; the
	// universal css-class and mj-class attributes are accepted without an entry.
	// A nil validator accepts every value.
	AllowedAttributes() map[string]AttributeValidator
	// DefaultAttributes returns the values to apply for attributes the author
	// did not write. They are applied after author attributes have been
	// validated, like a built-in component's defaults.
	DefaultAttributes(ctx *RenderContext) map[string]string
	// Render writes the element's HTML to w. Returning an error stops the
	// render.
	Render(ctx *RenderContext, w io.Writer, n *node.Node) error
}

// ChildComponent is a Component that accepts child elements. A component that
// does not implement it is a leaf: any child element under its tag is rejected
// as a disallowed child.
//
// AllowedChildren returns the child tag names the component accepts, in the
// order they are reported in error messages. Declaring children makes the tag a
// container for validation. Render receives the children and is responsible for
// them: it can look each one up in RenderContext.CustomComponents and render it,
// but the built-in container renderers do not descend into a custom tag. That
// dispatch, together with head-style contribution, is ticket T4 (issue #5).
type ChildComponent interface {
	Component
	AllowedChildren() []string
}

// universalAttributes are accepted on every component regardless of its own
// allowedAttributes table. MJML's validator uses the same whitelist.
var universalAttributes = map[string]struct{}{
	"css-class": {},
	"mj-class":  {},
}

// internalSyntheticAttributes are attributes this implementation writes onto a
// node while rendering (never input): mj-group marks its columns with
// mobileWidth so getMobileWidth can see the group context. Document validation
// still rejects them in input, because they are not in any allowedAttributes
// table.
var internalSyntheticAttributes = map[string]struct{}{
	"mobileWidth": {},
}

// InitComponent prepares n for rendering with comp: it validates the attributes
// the author wrote and then sets the component's default for every attribute the
// element does not already have. It is the first thing every body-component
// renderer calls for an element.
func InitComponent(ctx *RenderContext, comp Component, n *node.Node) error {
	// Validate the attributes the author wrote, before defaults are applied.
	// Applying defaults first used to hide both unknown attributes and invalid
	// author values behind the component's own default for the same key.
	if err := validateComponentAttributes(comp, n); err != nil {
		return err
	}

	for key, value := range comp.DefaultAttributes(ctx) {
		if _, has := n.GetAttributeValue(key); !has {
			n.SetAttribute(key, value)
		}
	}

	return nil
}

func validateComponentAttributes(comp Component, n *node.Node) error {
	allowed := comp.AllowedAttributes()

	for _, attribute := range n.Attributes {
		name := attribute.Name.Local

		if _, internal := internalSyntheticAttributes[name]; internal {
			continue
		}

		if _, universal := universalAttributes[name]; universal {
			continue
		}

		validator, ok := allowed[name]
		if !ok {
			// Unknown author attributes are rejected by ValidateDocument before
			// rendering. An unknown attribute can still reach a component here
			// because MJML passes a parent's whole attribute map down to its
			// children; those inherited extras are rendering context, not input.
			continue
		}

		if err := runValidator(validator, attribute.Value); err != nil {
			return NewError(KindInvalidValue, elementPathSegments(n), name, attribute.Value, err)
		}
	}

	return nil
}

// runValidator applies an attribute validator. A nil validator accepts every
// value: a custom component may leave one out of its table instead.
func runValidator(validator AttributeValidator, value string) error {
	if validator == nil {
		return nil
	}

	return validator(value)
}

func validateAttributes(allowed map[string]AttributeValidator, n *node.Node) error {
	for _, attribute := range n.Attributes {
		name := attribute.Name.Local

		if _, internal := internalSyntheticAttributes[name]; internal {
			continue
		}

		if _, universal := universalAttributes[name]; universal {
			continue
		}

		validator, ok := allowed[name]
		if !ok {
			return NewError(KindUnknownAttribute, elementPathSegments(n), name, attribute.Value, nil)
		}

		if err := runValidator(validator, attribute.Value); err != nil {
			return NewError(KindInvalidValue, elementPathSegments(n), name, attribute.Value, err)
		}
	}

	return nil
}

// validationDependencies is MJML's dependency table, copied from
// node_modules/mjml-preset-core/lib/dependencies.js. It is the single authority
// for which element may contain which child. mj-attributes and mj-class accept
// any component tag; that regex entry, custom components and their own child
// lists are handled in allowsChild.
var validationDependencies = map[string][]string{
	MJMLTagName:             {BodyTagName, HeadTagName, RawTagName},
	AccordionTagName:        {AccordionElementTagName, RawTagName},
	AccordionElementTagName: {AccordionTitleTagName, AccordionTextTagName, RawTagName},
	AccordionTitleTagName:   {},
	AccordionTextTagName:    {},
	AttributesTagName:       {},
	BodyTagName:             {RawTagName, SectionTagName, WrapperTagName, HeroTagName},
	ButtonTagName:           {},
	CarouselTagName:         {CarouselImageTagName},
	CarouselImageTagName:    {},
	ColumnTagName:           {AccordionTagName, ButtonTagName, CarouselTagName, DividerTagName, ImageTagName, RawTagName, SocialTagName, SpacerTagName, TableTagName, TextTagName, NavbarTagName},
	HTMLAttributeTagName:    {},
	HTMLAttributesTagName:   {SelectorTagName},
	DividerTagName:          {},
	GroupTagName:            {ColumnTagName, RawTagName},
	HeadTagName:             {AttributesTagName, BreakpointTagName, HTMLAttributesTagName, FontTagName, PreviewTagName, StyleTagName, TitleTagName, RawTagName},
	HeroTagName:             {AccordionTagName, ButtonTagName, CarouselTagName, DividerTagName, ImageTagName, SocialTagName, SpacerTagName, TableTagName, TextTagName, NavbarTagName, RawTagName},
	ImageTagName:            {},
	NavbarTagName:           {NavbarLinkTagName, RawTagName},
	RawTagName:              {},
	SectionTagName:          {ColumnTagName, GroupTagName, RawTagName},
	SelectorTagName:         {HTMLAttributeTagName},
	SocialTagName:           {SocialElementTagName, RawTagName},
	SocialElementTagName:    {},
	SpacerTagName:           {},
	TableTagName:            {},
	TextTagName:             {},
	WrapperTagName:          {HeroTagName, RawTagName, SectionTagName},
}

// componentTags are tags backed by a component (or by an MJML head handler).
// They are what mj-attributes and mj-class accept as nested per-tag defaults.
var componentTags = map[string]struct{}{
	MJMLTagName: {}, HeadTagName: {}, BodyTagName: {}, RawTagName: {},
	AttributesTagName: {}, BreakpointTagName: {}, FontTagName: {}, PreviewTagName: {},
	StyleTagName: {}, TitleTagName: {}, HTMLAttributesTagName: {}, SelectorTagName: {},
	HTMLAttributeTagName: {},
	SectionTagName:       {}, ColumnTagName: {}, WrapperTagName: {}, GroupTagName: {},
	SpacerTagName: {}, ImageTagName: {}, SocialTagName: {}, SocialElementTagName: {},
	DividerTagName: {}, TableTagName: {}, ButtonTagName: {}, HeroTagName: {}, TextTagName: {},
	NavbarTagName: {}, NavbarLinkTagName: {}, AccordionTagName: {},
	AccordionElementTagName: {}, AccordionTitleTagName: {}, AccordionTextTagName: {},
	CarouselTagName: {}, CarouselImageTagName: {},
}

// knownTags is every tag this implementation recognises, including the
// component-less head tags.
var knownTags = func() map[string]struct{} {
	tags := make(map[string]struct{}, len(componentTags)+4)

	for tag := range componentTags {
		tags[tag] = struct{}{}
	}

	for tag, children := range validationDependencies {
		tags[tag] = struct{}{}
		for _, child := range children {
			tags[child] = struct{}{}
		}
	}

	for _, tag := range []string{AllTagName, ClassTagName} {
		tags[tag] = struct{}{}
	}

	return tags
}()

// ValidateDocument checks the parsed tree against MJML's dependency table and
// every component's allowedAttributes before anything is rendered. It is the
// fail-fast counterpart of mjml-validator's soft warnings: unknown elements,
// disallowed children, unknown attributes and invalid attribute values are hard
// errors that carry the element path.
func ValidateDocument(ctx *RenderContext, root *node.Node) error {
	return validateDocumentNode(ctx, root, nil)
}

func validateDocumentNode(ctx *RenderContext, n, parent *node.Node) error {
	if !IsBuiltinTag(n.Type) {
		if _, custom := customComponent(ctx, n.Type); !custom {
			return NewError(KindUnknownElement, elementPathSegments(n), "", n.Type, nil)
		}
	}

	if parent != nil && !allowsChild(ctx, parent.Type, n.Type) {
		return NewError(
			KindDisallowedChild,
			elementPathSegments(n),
			"",
			n.Type,
			fmt.Errorf("allowed children: %s", allowedChildList(ctx, parent.Type)),
		)
	}

	if err := validateDocumentAttributes(ctx, n); err != nil {
		return err
	}

	if err := validateMjClassReference(ctx, n); err != nil {
		return err
	}

	for _, child := range n.Children {
		if err := validateDocumentNode(ctx, child, n); err != nil {
			return err
		}
	}

	return nil
}

func allowsChild(ctx *RenderContext, parent, child string) bool {
	if parentComponent, custom := customComponent(ctx, parent); custom {
		return allowsCustomComponentChild(parentComponent, child)
	}

	switch parent {
	case AttributesTagName:
		// <mj-attributes> accepts mj-all, mj-class and any component tag.
		if child == AllTagName || child == ClassTagName {
			return true
		}

		return acceptsPerTagDefaults(ctx, child)
	case ClassTagName:
		// Nested per-tag defaults inside <mj-class>.
		return acceptsPerTagDefaults(ctx, child)
	}

	if _, custom := customComponent(ctx, child); custom {
		// A custom component is a leaf, so it sits where content is placed, not
		// where a layout element is expected.
		_, contentParent := customComponentParents[parent]

		return contentParent
	}

	return slices.Contains(validationDependencies[parent], child)
}

func allowedChildList(ctx *RenderContext, parent string) string {
	if parentComponent, custom := customComponent(ctx, parent); custom {
		container, isContainer := parentComponent.(ChildComponent)
		if !isContainer || len(container.AllowedChildren()) == 0 {
			return "none"
		}

		return strings.Join(container.AllowedChildren(), ", ")
	}

	switch parent {
	case AttributesTagName, ClassTagName:
		return "any component tag"
	}

	children := validationDependencies[parent]
	if len(children) == 0 {
		return "none"
	}

	return strings.Join(children, ", ")
}

func validateDocumentAttributes(ctx *RenderContext, n *node.Node) error {
	validators, enforce := documentAttributeValidators(ctx, n.Type)
	if !enforce {
		return nil
	}

	path := elementPathSegments(n)

	for _, attribute := range n.Attributes {
		name := attribute.Name.Local

		if _, universal := universalAttributes[name]; universal {
			continue
		}

		validator, ok := validators[name]
		if !ok {
			return NewError(KindUnknownAttribute, path, name, attribute.Value, nil)
		}

		if err := runValidator(validator, attribute.Value); err != nil {
			return NewError(KindInvalidValue, path, name, attribute.Value, err)
		}
	}

	return nil
}

// documentAttributeValidators returns the allowedAttributes table for one tag.
// A custom component's table comes first, so a registered tag is validated
// exactly like a built-in element. The second result is false for mj-all and
// mj-class, whose attributes are intentionally free-form because they are
// applied to whichever components use them (MJML does not validate them
// either).
func documentAttributeValidators(ctx *RenderContext, tag string) (map[string]AttributeValidator, bool) {
	if comp, custom := customComponent(ctx, tag); custom {
		return comp.AllowedAttributes(), true
	}

	switch tag {
	case AccordionTagName:
		return MJMLAccordion{}.AllowedAttributes(), true
	case AccordionElementTagName:
		return MJMLAccordionElement{}.AllowedAttributes(), true
	case AccordionTitleTagName:
		return MJMLAccordionTitle{}.AllowedAttributes(), true
	case AccordionTextTagName:
		return MJMLAccordionText{}.AllowedAttributes(), true
	case BodyTagName:
		return MJMLBody{}.AllowedAttributes(), true
	case ButtonTagName:
		return MJMLButton{}.AllowedAttributes(), true
	case CarouselTagName:
		return MJMLCarousel{}.AllowedAttributes(), true
	case CarouselImageTagName:
		return MJMLCarouselImage{}.AllowedAttributes(), true
	case ColumnTagName:
		return MJMLColumn{}.AllowedAttributes(), true
	case DividerTagName:
		return MJMLDivider{}.AllowedAttributes(), true
	case GroupTagName:
		return MJMLGroup{}.AllowedAttributes(), true
	case HeroTagName:
		return MJMLHero{}.AllowedAttributes(), true
	case ImageTagName:
		return MJMLImage{}.AllowedAttributes(), true
	case NavbarTagName:
		return MJMLNavbar{}.AllowedAttributes(), true
	case NavbarLinkTagName:
		return MJMLNavbarLink{}.AllowedAttributes(), true
	case RawTagName:
		return MJMLRaw{}.AllowedAttributes(), true
	case SectionTagName:
		return MJMLSection{}.AllowedAttributes(), true
	case WrapperTagName:
		return MJMLSection{IsWrapper: true}.AllowedAttributes(), true
	case SocialTagName:
		return MJMLSocial{}.AllowedAttributes(), true
	case SocialElementTagName:
		return MJMLSocialElement{}.AllowedAttributes(), true
	case SpacerTagName:
		return MJMLSpacer{}.AllowedAttributes(), true
	case TableTagName:
		return MJMLTable{}.AllowedAttributes(), true
	case TextTagName:
		return MJMLText{}.AllowedAttributes(), true
	case MJMLTagName:
		return MJML{}.AllowedAttributes(), true
	case BreakpointTagName:
		return map[string]AttributeValidator{"width": validateUnit([]string{"px"}, false)}, true
	case FontTagName:
		return map[string]AttributeValidator{
			"name": validateType("string"),
			"href": validateType("string"),
		}, true
	case StyleTagName:
		return map[string]AttributeValidator{"inline": validateType("string")}, true
	case SelectorTagName:
		// MJML reads path from the raw attributes even though the component has
		// no allowedAttributes table.
		return map[string]AttributeValidator{"path": validateType("string")}, true
	case HTMLAttributeTagName:
		// MJML reads name from the raw attributes even though the component has
		// no allowedAttributes table.
		return map[string]AttributeValidator{"name": validateType("string")}, true
	case HeadTagName, AttributesTagName, PreviewTagName, TitleTagName, HTMLAttributesTagName:
		return map[string]AttributeValidator{}, true
	default:
		return nil, false
	}
}

// validateMjClassReference enforces that every class listed in a component's
// mj-class attribute was declared in <mj-attributes>. MJML silently applies
// nothing for an undeclared class; we fail fast instead.
func validateMjClassReference(ctx *RenderContext, n *node.Node) error {
	value, has := n.GetAttributeValue("mj-class")
	if !has || value == "" || insideHead(n) {
		return nil
	}

	for _, name := range strings.Split(value, " ") {
		if name == "" {
			continue
		}

		if _, declared := ctx.Classes[name]; !declared {
			return NewError(KindUndefinedMJClass, elementPathSegments(n), "mj-class", name, nil)
		}
	}

	return nil
}

func insideHead(n *node.Node) bool {
	for current := n; current != nil; current = current.Parent {
		if current.Type == HeadTagName {
			return true
		}
	}

	return false
}

// elementPathSegments renders the ancestry of n as data: "mj-body",
// "mj-section", "mj-column", "mj-text". The root <mjml> element is omitted so
// paths point at the part of the document an author edits.
func elementPathSegments(n *node.Node) []string {
	var parts []string

	for current := n; current != nil; current = current.Parent {
		parts = append(parts, current.Type)
	}

	slices.Reverse(parts)

	if len(parts) > 1 && parts[0] == MJMLTagName {
		parts = parts[1:]
	}

	return parts
}

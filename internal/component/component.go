package component

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/julez-dev/mjmlgo/internal/node"
)

type Component interface {
	Name() string
	AllowedAttributes() map[string]validateAttributeFunc
	DefaultAttributes(ctx *RenderContext) map[string]string
	Render(ctx *RenderContext, w io.Writer, n *node.Node) error
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

		if err := validator(attribute.Value); err != nil {
			return fmt.Errorf("failed to validate field %s in <%s>: %w", name, comp.Name(), err)
		}
	}

	return nil
}

func validateAttributes(allowed map[string]validateAttributeFunc, componentName string, n *node.Node) error {
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
			return fmt.Errorf("%w: attribute %q is not allowed on <%s>", ErrValidation, name, componentName)
		}

		if err := validator(attribute.Value); err != nil {
			return fmt.Errorf("failed to validate field %s in <%s>: %w", name, componentName, err)
		}
	}

	return nil
}

// validationDependencies is MJML's dependency table, copied from
// node_modules/mjml-preset-core/lib/dependencies.js. It is the single authority
// for which element may contain which child. mj-attributes and mj-class accept
// any component tag; that regex entry is handled in allowsChild.
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
	if _, known := knownTags[n.Type]; !known {
		return fmt.Errorf("%w: %s: unknown element <%s>", ErrValidation, elementPath(n), n.Type)
	}

	if parent != nil && !allowsChild(parent.Type, n.Type) {
		return fmt.Errorf("%w: %s: <%s> is not allowed inside <%s> (allowed children: %s)",
			ErrValidation, elementPath(n), n.Type, parent.Type, allowedChildList(parent.Type))
	}

	if err := validateDocumentAttributes(n); err != nil {
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

func allowsChild(parent, child string) bool {
	switch parent {
	case AttributesTagName:
		// <mj-attributes> accepts mj-all, mj-class and any component tag.
		if child == AllTagName || child == ClassTagName {
			return true
		}

		_, isComponent := componentTags[child]
		return isComponent
	case ClassTagName:
		// Nested per-tag defaults inside <mj-class>.
		_, isComponent := componentTags[child]
		return isComponent
	}

	return slices.Contains(validationDependencies[parent], child)
}

func allowedChildList(parent string) string {
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

func validateDocumentAttributes(n *node.Node) error {
	validators, enforce := documentAttributeValidators(n.Type)
	if !enforce {
		return nil
	}

	path := elementPath(n)

	for _, attribute := range n.Attributes {
		name := attribute.Name.Local

		if _, universal := universalAttributes[name]; universal {
			continue
		}

		validator, ok := validators[name]
		if !ok {
			return fmt.Errorf("%w: %s: attribute %q is not allowed on <%s>", ErrValidation, path, name, n.Type)
		}

		if err := validator(attribute.Value); err != nil {
			return fmt.Errorf("%w: %s: attribute %q in <%s>: %w", ErrValidation, path, name, n.Type, err)
		}
	}

	return nil
}

// documentAttributeValidators returns the allowedAttributes table for one tag.
// The second result is false for mj-all and mj-class, whose attributes are
// intentionally free-form because they are applied to whichever components use
// them (MJML does not validate them either).
func documentAttributeValidators(tag string) (map[string]validateAttributeFunc, bool) {
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
		return map[string]validateAttributeFunc{"width": validateUnit([]string{"px"}, false)}, true
	case FontTagName:
		return map[string]validateAttributeFunc{
			"name": validateType("string"),
			"href": validateType("string"),
		}, true
	case StyleTagName:
		return map[string]validateAttributeFunc{"inline": validateType("string")}, true
	case SelectorTagName:
		// MJML reads path from the raw attributes even though the component has
		// no allowedAttributes table.
		return map[string]validateAttributeFunc{"path": validateType("string")}, true
	case HTMLAttributeTagName:
		// MJML reads name from the raw attributes even though the component has
		// no allowedAttributes table.
		return map[string]validateAttributeFunc{"name": validateType("string")}, true
	case HeadTagName, AttributesTagName, PreviewTagName, TitleTagName, HTMLAttributesTagName:
		return map[string]validateAttributeFunc{}, true
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
			return fmt.Errorf("%w: %s: mj-class %q is not declared in <mj-attributes>", ErrValidation, elementPath(n), name)
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

// elementPath renders the ancestry of n as "mj-body > mj-section > mj-column >
// mj-text". The root <mjml> element is omitted so paths point at the part of the
// document an author edits.
func elementPath(n *node.Node) string {
	var parts []string

	for current := n; current != nil; current = current.Parent {
		parts = append(parts, current.Type)
	}

	slices.Reverse(parts)

	if len(parts) > 1 && parts[0] == MJMLTagName {
		parts = parts[1:]
	}

	return strings.Join(parts, " > ")
}

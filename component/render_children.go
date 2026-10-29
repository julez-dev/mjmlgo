package component

import (
	"fmt"
	"io"

	"github.com/julez-dev/mjmlgo/v2/node"
)

// contentComponents maps the tag of every built-in content component to the
// component that renders it. They are the children the built-in content
// containers dispatch to (mj-body, mj-column and mj-hero), so they are also the
// built-in tags a custom container may declare in ChildComponent.AllowedChildren.
var contentComponents = map[string]Component{
	AccordionTagName: MJMLAccordion{},
	ButtonTagName:    MJMLButton{},
	CarouselTagName:  MJMLCarousel{},
	DividerTagName:   MJMLDivider{},
	ImageTagName:     MJMLImage{},
	NavbarTagName:    MJMLNavbar{},
	RawTagName:       MJMLRaw{},
	SocialTagName:    MJMLSocial{},
	SpacerTagName:    MJMLSpacer{},
	TableTagName:     MJMLTable{},
	TextTagName:      MJMLText{},
}

// IsContentTag reports whether tag is a built-in content component: one a custom
// container may declare as a child, because RenderChildren can render it. The
// layout components (mj-section, mj-wrapper, mj-group, mj-column) and the head
// components are not content tags; a custom component occupies a content
// position, and its children do too.
func IsContentTag(tag string) bool {
	_, content := contentComponents[tag]

	return content
}

// RenderChildren renders every child of n into w, in document order. It is how a
// custom container renders the children it accepted: each child is dispatched to
// the built-in component that renders its tag or to the custom component
// registered for it, with its author attributes validated and its defaults
// applied first - exactly what the built-in content containers do for their
// children. A child outside the container's AllowedChildren is a
// KindDisallowedChild error carrying the element path, so a component that
// renders without prior validation still enforces its own contract.
//
// The component owns all markup around the children. It may render them straight
// into w, or into a buffer of its own first so every child is rendered before
// the wrapper is written:
//
//	var children strings.Builder
//	if err := component.RenderChildren(ctx, &children, n); err != nil {
//		return err
//	}
//	// ... write the wrapper and children.String() ...
//
// An mj-raw child is rendered verbatim, the way the built-in containers render raw
// content outside a table body (mj-body, mj-head, mj-section, mj-wrapper,
// mj-group, mj-accordion-element, mj-social and mj-navbar). A container whose
// children sit directly inside a <tbody> calls RenderChildrenInTableBody instead,
// which states that position in the markup with the raw-block markers.
func RenderChildren(ctx *RenderContext, w io.Writer, n *node.Node) error {
	return renderChildList(ctx, w, n, renderRawChild)
}

// RenderChildrenInTableBody is RenderChildren for a container whose children sit
// directly inside a <tbody> - the position the built-in column and hero put their
// children in. It wraps an mj-raw child in the raw-block markers, which state that
// position in the markup and are stripped by the renderer before the caller sees
// the document.
//
// The markers are internal transport, not a styling mechanism: the inline stage
// matches selectors against the emitted markup itself and styles a raw block where
// it stands, so it needs no protection from any parser and both helpers produce
// the same document. Choosing the wrong helper is therefore a documentation
// mistake, not a styling difference - but use this one when your children really
// do sit in a table body, because that is the contract the markers state.
func RenderChildrenInTableBody(ctx *RenderContext, w io.Writer, n *node.Node) error {
	return renderChildList(ctx, w, n, renderRawChildInTableBody)
}

// renderChildList is the shared child loop of RenderChildren and
// RenderChildrenInTableBody. renderRaw is the entry point for an mj-raw child and
// is the only difference between the two, so the child-list check and the
// dispatch cannot drift apart.
func renderChildList(
	ctx *RenderContext,
	w io.Writer,
	n *node.Node,
	renderRaw func(*RenderContext, io.Writer, *node.Node) error,
) error {
	var container ChildComponent

	parent, isCustom := customComponent(ctx, n.Type)
	if isCustom {
		container, _ = parent.(ChildComponent)
	}

	for _, child := range n.Children {
		if isCustom && (container == nil || !allowsCustomComponentChild(container, child.Type)) {
			return NewError(
				KindDisallowedChild,
				elementPathSegments(child),
				"",
				child.Type,
				fmt.Errorf("allowed children: %s", allowedChildList(ctx, n.Type)),
			)
		}

		if child.Type == RawTagName {
			if err := renderRaw(ctx, w, child); err != nil {
				return err
			}

			continue
		}

		comp, ok := childComponent(ctx, child.Type)
		if !ok {
			// Unreachable while registration rejects a declared child nothing
			// can render and ValidateDocument enforces that declaration; report
			// the broken invariant instead of dropping the child.
			return fmt.Errorf("no component for child <%s> in <%s>", child.Type, n.Type)
		}

		if err := InitComponent(ctx, comp, child); err != nil {
			return err
		}

		if err := comp.Render(ctx, w, child); err != nil {
			return fmt.Errorf("failed to render <%s>: %w", child.Type, err)
		}
	}

	return nil
}

// childComponent resolves the component that renders one child of a custom
// container: a built-in content component first, then a component registered on
// this renderer. A custom component can never shadow a built-in one, so the
// order only documents the intent. It is the dispatch table the content
// dispatchers share - mj-column, mj-hero and the RenderChildren helpers - so a
// new content component is listed once. mj-body keeps its own switch because it
// also dispatches the layout children (mj-section, mj-wrapper, mj-hero), which
// contentComponents deliberately excludes.
func childComponent(ctx *RenderContext, tag string) (Component, bool) {
	if builtin, ok := contentComponents[tag]; ok {
		return builtin, true
	}

	return customComponent(ctx, tag)
}

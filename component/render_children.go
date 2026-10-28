package component

import (
	"fmt"
	"io"
	"strings"

	"github.com/julez-dev/mjmlgo/node"
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
// An mj-raw child is rendered the way a built-in container renders raw content
// that sits directly in a table body: inside the raw-block markers, which the
// renderer strips from the finished document. A container may put its children in
// a <tbody>, and the marker is what keeps html.Parse from foster-parenting raw
// markup out of the table; the inliner parses such a block as a fragment at its
// position, so the limits the built-in containers have for raw content apply here
// too (a selector that needs an ancestor outside the block cannot match inside
// it).
func RenderChildren(ctx *RenderContext, w io.Writer, n *node.Node) error {
	var container ChildComponent
	if parent, custom := customComponent(ctx, n.Type); custom {
		container, _ = parent.(ChildComponent)
	}

	for _, child := range n.Children {
		if container != nil && !allowsCustomComponentChild(container, child.Type) {
			return NewError(
				KindDisallowedChild,
				elementPathSegments(child),
				"",
				child.Type,
				fmt.Errorf("allowed children: %s", strings.Join(container.AllowedChildren(), ", ")),
			)
		}

		if child.Type == RawTagName {
			if err := renderRawChildInTableBody(ctx, w, child); err != nil {
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
// order only documents the intent. It is the single dispatch table every
// container uses - the built-in ones and RenderChildren - so a new content
// component is listed once.
func childComponent(ctx *RenderContext, tag string) (Component, bool) {
	if builtin, ok := contentComponents[tag]; ok {
		return builtin, true
	}

	return customComponent(ctx, tag)
}

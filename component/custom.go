package component

import "slices"

// customComponentParents are the built-in containers that accept a custom
// component as a child. They are the content containers, where an author would
// otherwise place a content component such as mj-text, mj-image or mj-button. A
// custom component - leaf or container - occupies a content position, so it does
// not replace a layout element: mj-section, mj-wrapper and mj-group keep their
// fixed child lists.
var customComponentParents = map[string]struct{}{
	BodyTagName:   {},
	ColumnTagName: {},
	HeroTagName:   {},
}

// customComponent returns the custom component registered for tag.
func customComponent(ctx *RenderContext, tag string) (Component, bool) {
	comp, ok := ctx.CustomComponents[tag]

	return comp, ok
}

// allowsCustomComponentChild reports whether the custom component parent
// declares child as one of its children. A component that does not implement
// ChildComponent is a leaf and accepts none.
func allowsCustomComponentChild(parent Component, child string) bool {
	container, ok := parent.(ChildComponent)
	if !ok {
		return false
	}

	return slices.Contains(container.AllowedChildren(), child)
}

// acceptsPerTagDefaults reports whether tag may carry per-tag defaults inside
// mj-attributes, or nested per-tag defaults inside mj-class. That is every tag
// backed by a component or an MJML head handler - componentTags - plus a custom
// component registered on this renderer. It is narrower than IsBuiltinTag, which
// also covers mj-all, mj-class and the document elements.
func acceptsPerTagDefaults(ctx *RenderContext, tag string) bool {
	if _, builtin := componentTags[tag]; builtin {
		return true
	}

	_, custom := customComponent(ctx, tag)

	return custom
}

// IsBuiltinTag reports whether tag is one of the elements this package
// implements: every body and head component, mj-all, mj-class, mj-include and
// the document elements. A custom component must not use such a tag, because
// registration would shadow the built-in element instead of extending the
// language.
func IsBuiltinTag(tag string) bool {
	_, builtin := knownTags[tag]

	return builtin
}

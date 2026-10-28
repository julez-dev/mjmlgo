// Package component renders the MJML component tree: every body and head
// component of MJML 4.18, the per-render state the renderer accumulates from the
// document head, the validation applied to a parsed document before it is
// rendered, and the contract a host implements to add a component of its own.
//
// The package is exported because the root mjmlgo package and custom components
// share it. Part of what it exports is frozen API: the render context fields
// keep their meaning and their type, methods may be added and fields may not
// change meaning. See docs/adr/0004-exported-internals-freeze.md for the exact
// frozen set and for why exporting beat a facade.
//
// # What is API
//
// The supported surface is what an embedder needs to inspect a render or to
// build a component against:
//
//   - RenderContext and its exported fields: the state a component reads while
//     rendering, and the structure the parser fills from the head
//     (mj-attributes, mj-class, mj-font, mj-html-attributes, styles, preview,
//     title, breakpoint, language and direction). CustomComponents holds the
//     components the renderer was configured with.
//   - The types those fields are declared with: AttributeMap, Stylesheet, Rule,
//     MediaRule, Style, HTMLAttributeSelector, HTMLAttribute,
//     ComponentHeadStyle and Component.
//   - Component, ChildComponent and AttributeValidator, the custom-component
//     contract (see below).
//   - InitComponent, which validates the attributes the author wrote and then
//     fills in the component's defaults.
//   - ValidateDocument, which checks tag names, nesting, attribute values and
//     mj-class references against MJML's own tables before rendering starts.
//   - Error, ErrorKind and NewError, the structured failure every validation,
//     parse and component-registration site constructs. Callers inspect Kind,
//     Path, Attribute and Value; errors.Is keeps matching ErrValidation.
//   - IsBuiltinTag, which reports whether a tag is already implemented here
//     (every component, the head handlers, mj-all, mj-class and mj-include) and
//     so cannot be used by a custom component.
//   - DefaultIDGenerator and the TagName constants.
//
// # Custom components
//
// A host adds an element without patching this repository by implementing
// Component and registering it on a renderer (mjmlgo.WithComponents):
//
//   - Name is the tag. It must not be empty, must not be registered twice on the
//     same renderer and must not shadow a built-in element; such a renderer is
//     rejected with a *Error of kind KindInvalidComponent.
//   - AllowedAttributes declares every accepted attribute with its validator.
//     An undeclared author attribute and a value its validator rejects are
//     validation errors carrying the element path, exactly as for a built-in
//     component. css-class and mj-class are universal and need no entry, and a
//     nil validator accepts every value.
//   - DefaultAttributes returns the values applied when the author wrote none,
//     after author attributes have been validated. Per-tag defaults from
//     mj-attributes and attributes from mj-class are applied before those, so a
//     custom tag takes part in the document's default cascade.
//   - Render writes the element's HTML from the render context and the node.
//     Both are the values a built-in component receives, so a custom component
//     needs no access to package internals.
//
// A custom component is a leaf unless it also implements ChildComponent, whose
// AllowedChildren lists the child tags it accepts; a component without it
// rejects every child. A custom tag is accepted where content goes - as a
// direct child of mj-body, mj-column or mj-hero - and not where a layout
// element is expected. Registration is per renderer: there is no global
// registry, and two renderers in one process may know different tags.
//
// Rendering a ChildComponent's children is that component's own job: the
// built-in container renderers do not descend into a custom tag. Container
// support, including head-style contribution, is ticket T4 (issue #5).
// example/customcomponent is the in-repo example of the whole contract.
//
// # What is not API
//
// The concrete components - MJML, MJMLSection, MJMLColumn, ... - are exported
// only so the renderer can dispatch to them. Their markup, their default
// attribute tables and their call order are implementation details; they change
// whenever matching the reference implementation requires it.
//
// ErrValidation is still exported as the sentinel the element, attribute,
// value, child and mj-class kinds match with errors.Is; ErrMJMLBadlyFormatted, MergeOutlookConditionals,
// RemoveNonNumeric, RawBlockStartMarker, RawBlockEndMarker and
// StripRawBlockMarkers are exported to wire the packages together. They are not
// meant to be called from outside, and may be changed or unexported.
//
// # What user code must not touch
//
// Rendering keeps private markers on nodes and in the HTML it emits. They are
// internal conveniences, not part of the MJML language, and user code must not
// set, read or depend on them:
//
//   - The parser protects the content of raw-content elements (mj-text,
//     mj-button, mj-raw, ...) with CDATA sections so arbitrary markup survives
//     encoding/xml. The protection is removed again while parsing and never
//     appears in node.Node.Content.
//   - mj-group marks its child columns with the synthetic attribute
//     mobileWidth="mobileWidth" so the column renderer knows it is inside a
//     group. Document validation rejects that attribute on input; only the
//     group renderer may set it.
//   - Raw content emitted directly inside a table body is wrapped in
//     RawBlockStartMarker / RawBlockEndMarker so the CSS inliner can keep it
//     out of the HTML round-trip. The renderer strips the markers from the
//     final document; they are not output.
//   - node.Node.Parent is parser bookkeeping. Components receive the node to
//     render and must walk Children instead of ascending Parent; see package
//     node.
//
// A RenderContext also carries unexported state: the two font-family values
// that accordion titles and texts inherit from the element they are rendered
// in. That is contextual rendering state, not configuration.
package component

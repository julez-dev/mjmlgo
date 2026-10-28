// Package component renders the MJML component tree: every body and head
// component of MJML 4.18, the per-render state the renderer accumulates from the
// document head, and the validation applied to a parsed document before it is
// rendered.
//
// The package is exported because the root mjmlgo package and (from ticket T3,
// issue #4) custom components share it. Part of what it exports is frozen API: the render
// context fields keep their meaning and their type, methods may be added and
// fields may not change meaning. See docs/adr/0004-exported-internals-freeze.md
// for the exact frozen set and for why exporting beat a facade.
//
// # What is API
//
// The supported surface is what an embedder needs to inspect a render or to
// build a component against:
//
//   - RenderContext and its exported fields: the state a component reads while
//     rendering, and the structure the parser fills from the head
//     (mj-attributes, mj-class, mj-font, mj-html-attributes, styles, preview,
//     title, breakpoint, language and direction).
//   - The types those fields are declared with: AttributeMap, Stylesheet, Rule,
//     MediaRule, Style, HTMLAttributeSelector, HTMLAttribute and
//     ComponentHeadStyle.
//   - InitComponent, which validates the attributes the author wrote and then
//     fills in the component's defaults.
//   - ValidateDocument, which checks tag names, nesting, attribute values and
//     mj-class references against MJML's own tables before rendering starts.
//   - Error, ErrorKind and NewError, the structured failure every validation and
//     parse site constructs. Callers inspect Kind, Path, Attribute and Value;
//     errors.Is keeps matching ErrValidation.
//   - DefaultIDGenerator and the TagName constants.
//
// InitComponent and ValidateDocument are safe to call, but their signatures are
// not frozen: the custom-component contract (T3) and structured errors (T5) are
// expected to change them.
//
// # What is not API
//
// The concrete components - MJML, MJMLSection, MJMLColumn, ... - are exported
// only so the renderer can dispatch to them. Their markup, their default
// attribute tables and their call order are implementation details; they change
// whenever matching the reference implementation requires it.
//
// ErrValidation is still exported as the sentinel every validation kind matches
// with errors.Is; ErrMJMLBadlyFormatted, MergeOutlookConditionals,
// RemoveNonNumeric, RawBlockStartMarker, RawBlockEndMarker and
// StripRawBlockMarkers are exported to wire the packages together. They are not
// meant to be called from outside, and may be changed or unexported.
//
// The Component interface is the shape a component has, and is exported so the
// renderer can hold one, but it is not yet an extension point: its
// AllowedAttributes method returns an unexported validator type, so code outside
// this package cannot implement it. The public custom-component contract is a
// separate ticket (T3, issue #4) and will replace that method with an exported
// type.
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

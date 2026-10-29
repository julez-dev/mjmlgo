// Package node holds the element tree the parser produces and the renderer
// walks: one Node per MJML element, with its attributes, its children and - for
// a raw-content element such as mj-text or mj-raw - its unparsed content.
//
// The package is exported so custom components can receive a node from the
// renderer. Its exported shape is frozen: exported fields keep their meaning
// and their type; methods may be added.
//
// # What is API
//
// The supported surface is the Node type, its fields and its attribute
// accessors. Components read a node through those; user code that builds a tree
// by hand may write to them.
//
//   - Type is the element's tag name, for example "mj-section".
//   - Attributes holds the attributes in document order, exactly as the parser
//     read them. Rendering appends component defaults, so after a render the
//     slice can hold more attributes than the author wrote.
//   - Content is the unparsed text or markup of a raw-content element (mj-text,
//     mj-button, mj-table, mj-raw, ...); it is empty for containers.
//   - Children holds the child elements in document order. Text and raw content
//     live in Content, not as child nodes.
//   - SetAttribute sets an attribute, replacing its value or appending it.
//   - GetAttributeValue returns an attribute's value and whether the node has
//     it; GetAttributeValueDefault returns the empty string when it does not.
//
// # What is not API
//
// Parent is bookkeeping the parser fills in so it can close elements. User code
// must not set it, and no caller may rely on it: a hand-built tree has no
// parents, a subtree moved to another parent carries a stale one, and matching
// MJML means walking Children downwards. Components must use the children they
// are handed rather than ascending Parent.
//
// The parser and the renderers also keep private markers on nodes; user code
// must not set, read or depend on them:
//
//   - Attributes may carry an implementation-owned synthetic attribute:
//     mj-group marks each of its columns with mobileWidth="mobileWidth" so the
//     column renderer can see the group context. It is never input - document
//     validation rejects it - and it is not part of the MJML language.
//   - Raw content is protected with CDATA sections while the document is
//     parsed, so arbitrary markup survives encoding/xml unchanged. That
//     protection is a parser detail: it never appears in Content, and nothing
//     outside the parser may introduce or inspect it.
package node

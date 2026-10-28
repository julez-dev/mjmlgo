---
status: accepted
---

# Hard validation is an intentional divergence

The reference implementation validates softly: it logs warnings for unknown
tags, unknown or invalid attributes, invalid values and invalid nesting, and then
renders the document anyway, so a typo reaches the recipient as a partially
correct email. This implementation validates hard instead. Unknown elements,
attributes that are not allowed on their element, values that do not match
MJML's types, disallowed nesting, undeclared `mj-class` references, malformed
XML and a document without a root are all errors; rendering stops and no partial
output is written. Every error carries the offending element's path.

There is one validation mode. The reference's `validationLevel` setting is not
implemented, because a mode that lets known-bad input through reproduces the
failure mode this project exists to avoid.

## The contract is `testdata/invalid/`

The divergence is pinned by fixtures, not by prose: each document MJML tolerates
but this implementation rejects lives in `testdata/invalid/` as `<name>.mjml`
next to `<name>.mjml.error.txt`, which holds the error text (or a stable
substring of it) that rendering must return. `TestMJMLInvalid` requires the
renderer to fail and the error to contain that text. Adding a validation rule
without a fixture is incomplete work; changing an error message means updating
its fixture.

Invalid documents are deliberately not part of the parity corpus: their output
under the reference is exactly what we refuse to produce, so comparing the two
would be pointless.

## Failures are structured, and the kinds are closed

Every validation and parse site returns a `*Error{Kind, Path, Attribute, Value,
Err}` through one constructor. `Path` is the element path as data, not text, so a
caller can locate a failure without parsing a message; `Err` keeps the underlying
cause for `errors.Is`/`errors.As`. The kinds are a closed set: unknown element,
unknown attribute, invalid value, disallowed child, unknown include type,
undefined `mj-class` and parse failure.

Two mapping decisions follow from that closed set:

- Include failures are parse failures. A cycle, an escaped path, a missing
  include file and a document without `mj-body` all describe a document that
  could not be assembled into a renderable tree; the specific sentinel
  (`ErrIncludeCycle`, `ErrIncludePath`, ...) remains reachable through `Err`.
  Only the include *type* has a kind of its own, because it is the one include
  failure that is about the author's value rather than the document assembly.
- A missing required attribute (an `mj-selector` without `path`, an
  `mj-font` without `name`) is an invalid value, not an unknown attribute; the
  attribute is known, its value is absent. Template nesting violations such as
  `mj-html-attributes` outside `mj-head` are disallowed children.

CSS parsing inside `<mj-style>` is not one of these failures: it is a stylesheet
parse with no MJML element path, so it keeps its own error instead of inventing a
kind for it.

`errors.Is` keeps working for the older sentinels, including `ErrValidation` for
the element, attribute, value, child and `mj-class` kinds (an unknown include
type keeps its `ErrIncludeType` sentinel), so migrating a caller to the
structured error does not require a flag day. Error text remains part of the tested surface and is
deliberately regenerated when it changes, but callers are told to branch on
`Kind`, not on the message.

## Consequences

- Some documents the reference renders are rejected here. That is the intended
  behaviour, and the invalid corpus is where each case is recorded.
- The strictness extends to the input syntax: the parser is strict XML, so
  mismatched tags and comments that are not well-formed XML are errors even
  where the reference's lenient HTML parser accepts them.
- Error texts are part of the tested surface. They are compared by substring, so
  they can gain context, but a fixture must be updated whenever the stable part
  changes.
- A user migrating from MJML may see errors where MJML only warned. The error
  message names the element path and the rule, so the fix is straightforward.

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

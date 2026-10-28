---
status: accepted
---

# Inline CSS on the emitted markup, not on a re-parsed document

`<mj-style inline>` and `<mj-html-attributes>` are applied to an element tree
built from the markup the components just emitted, and the document is written
out once: every start tag a pass did not change keeps the exact bytes its
component produced, and a matched start tag is patched in place. There is no
whole-document `html.Parse`/`html.Render` round-trip and no fallback path.

This replaces the previous pipeline, which rendered HTML to a string, parsed the
whole document, matched selectors against the parsed tree, applied
`<mj-html-attributes>` to the body subtree, and re-serialized everything. That
round-trip was ~65% of the inline stage on `input_big.mjml` and, because it
normalized the document, it also rewrote bytes it was not asked to touch.

## The tree

The tree is a record of what was emitted, not a reconstruction of what a browser
would build: `buildEmittedDocument` tokenizes the emitted markup and records the
ancestor/sibling relations plus the byte span of every start tag. No implied
elements, no table fix-ups, no parser-inserted tags, no text or comment nodes
(the selector engine's matchers count element nodes only).

Two consequences are deliberate:

- a self-closing tag is a leaf, because the renderer writes `<img ... />` and
  `<td ... />` that way and the reference inliner's parser (htmlparser2, through
  juice) is equally tolerant;
- an end tag closes the nearest matching open element and is otherwise ignored,
  so author raw content with unbalanced markup nests the way it was written.

A raw block inside a table body is therefore styled in place, in the same tree as
everything else: an author's `<div>` in a `<tbody>` is an ancestor of its own
children, so both the block's own selectors and selectors that depend on the
elements around it match. The raw-block markers, the placeholder machinery and
the per-block fragment parse that existed only to survive `html.Parse` are gone.

## What "during emission" means here, and what it costs

The components still emit markup as text; the tree is built by tokenizing that
markup once, at the end of emission, instead of parsing it as a browser would.
Applying declarations "during emission" therefore means: the document is emitted
once and written out once, and no rule-driven re-serialization of the document
happens at any point.

That leftover tokenizer pass is why the ticket's timing target is only partly
met. On `input_big.mjml` the round-trip cost ~0.54 ms of a 0.82 ms inline stage;
the stage is now ~0.61 ms, so ~0.21 ms is recovered and the rest came back as the
tree build the selector passes need (about half of the new stage is tokenizing,
a third is matching). Removing the tokenizer as well means having components emit
elements instead of text - a change to every component and template, out of scope
here. The numbers live in `benchmarks/BUDGETS.md`.

## Reproducing the cascade evidence

`testdata/parity/inline-cascade.mjml` is the adversarial fixture for the cascade
(two classes, both `!important`, conflicting shorthand and longhand). To see what
the pre-change build did with it, render it against the reference from a checkout
of the parent commit:

```bash
cp testdata/parity/inline-cascade.mjml <parent-checkout>/testdata/parity/
cd <parent-checkout>
MJML_BIN=$PWD/../<this-checkout>/node_modules/.bin/mjml \
  go test ./ -run 'TestMJMLParity/testdata/parity/inline-cascade'
```

It reports three mismatches - the source-order `margin`, the
importance-versus-specificity `bgcolor` and the mirrored `align` attribute. All
three pass now.

## The cascade

Declarations are collected per matched element and materialized afterwards,
because the cascade compares declarations with each other. Importance raises the
first specificity entry, then the specificity vector is compared, then the
declaration's position, where a later declaration wins - the tuple the reference
inliner sorts by. The element's own style attribute cascades as `{1,0,0,0}`, which
is what makes an authored inline style outrank every rule that is not important.

A rule that lists several selectors is applied once per selector, so each match
carries the specificity of the selector that actually matched. The mirrored
`align`/`valign`/`bgcolor` attributes follow the winning declaration and replace a
value the element carried before, like the reference's
`setAttributesOnTableElements`.

## Output

Serialization is the emitted markup plus the patched start tags, so the output of
a document that inlines CSS is formatted like the output of one that does not. The
previous round-trip turned `<!doctype html>` into `<!DOCTYPE html>`, void elements
into `<meta .../>`, `&nbsp;` into a raw byte, `{ if item < 5 }` into
`{ if item &lt; 5 }`, and expanded the renderer's self-closing `<td ... />` cells
into an open/close pair. Those were `html.Parse`/`html.Render` artifacts: on the
fixtures that move, the reference output keeps the emitted form (lowercase
doctype, four meta elements without `/>` in `head.mjml`, 247 `&nbsp;` entities and
no raw non-breaking spaces in `input_big.mjml`), and only the reference's own
`<o:AllowPNG/>` and author raw content still carry `/>`. Every fixture's document
is unchanged; only this serialization detail moved.

A self-closing `<td ... />` inside a `<tr>` is an open tag to every HTML parser,
so the emitted form and the expanded form build the same tree - the parity
harness parses both to the same document. A consumer that needs the reference's
own text formatting is out of luck anyway: the reference pretty-prints.

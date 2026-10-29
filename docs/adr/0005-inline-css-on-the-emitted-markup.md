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
round-trip was ~65% of the inline stage on the largest fixture and, because it
normalized the document, it also rewrote bytes it was not asked to touch.

## The tree

The tree is a record of what was emitted, built by tokenizing the emitted markup
and recording the ancestor/sibling relations plus the byte span of every start
tag. It is not a browser's reconstruction of the document: no implied elements,
no table fix-ups, no foster parenting, no text or comment nodes (the selector
engine's matchers count element nodes only).

It does, however, follow the tree the **reference inliner** builds, because that
is the tree the reference's selectors match. juice 10 parses the document with
htmlparser2 in HTML mode (`juice/lib/cheerio.js` sets `_useHtmlParser2` and
leaves `xmlMode` off), so this tree reproduces htmlparser2 8.0.2's rules:

- the `openImpliesClose` table: a `<div>` closes an open `<p>`, an `<li>` closes
  the previous `<li>`, a `<tr>` closes a preceding `<tr>`, a `<td>` a preceding
  `<td>`, `<dd>`/`<dt>`, `<option>`/`<optgroup>`, `<tbody>`/`<tfoot>` versus the
  table sections, and the rest of that table verbatim. Without these, markup such
  as `<p>text<div class="inner">` would keep the `div` inside the paragraph and a
  `p div` rule would style content the reference never styles;
- the self-closing flag is **ignored** on HTML elements: `<div class="wrap" />`
  stays open, so what follows it is inside the div and `.wrap .inner` matches,
  exactly as the reference sees it. The flag is honoured only inside foreign
  content (`<svg>`, `<math>`), because that is where htmlparser2 honours it - the
  renderer's self-closed `<td ... />` cells therefore stay open and are closed by
  the next cell or row, as in the reference;
- htmlparser2's void-element list decides what is a leaf (`br`, `img`, `meta`,
  `input`, `hr`, `col`, `link`, `param`, `source`, `track`, `wbr`, plus the
  legacy `basefont`, `command`, `frame`, `isindex`, `keygen`);
- an end tag closes the nearest matching open element and is otherwise ignored,
  so author raw content with unbalanced markup nests the way it was written.

A raw block inside a table body is therefore styled in place, in the same tree as
everything else: an author's `<div>` in a `<tbody>` is an ancestor of its own
children, so both the block's own selectors and selectors that depend on the
elements around it match. The placeholder machinery and the per-block fragment
parse that existed only to survive `html.Parse` are gone.

The raw-block markers survive as container transport, and only as that:
`component.RenderChildrenInTableBody` wraps raw children in them so a container
states where its children sit, and the renderer strips them before the caller
sees the document. No parser needs protecting any more, so they are comments the
inline stage ignores rather than something it has to restore.

## Residual limits

- A stray `</p>` or `</br>` in author raw content makes htmlparser2 materialize
  an empty element; this tree has no node for it, because there is no emitted
  markup to carry one. Only a rule matching an empty `p`/`br` in that situation
  could tell the difference, and no fixture has one.
- htmlparser2 applies no HTML5 scope checks in its implied-close table, so the
  table above is the whole rule set - there is nothing further to emulate.
- The selector engine (`github.com/ericchiang/css`) applies `+` and `~` in both
  directions, so `li + li` styles both items where the reference styles the
  second. That is a property of the engine, not of this tree: it reproduces on
  the parent commit with the pre-change pipeline, and fixing it means replacing
  or wrapping the matcher, which the ticket puts out of scope.

## What "during emission" means here, and what it costs

The components still emit markup as text; the tree is built by tokenizing that
markup once, at the end of emission, instead of parsing it as a browser would.
Applying declarations "during emission" therefore means: the document is emitted
once and written out once, and no rule-driven re-serialization of the document
happens at any point.

That leftover tokenizer pass is why the ticket's timing target is only partly
met. On the largest fixture the round-trip cost ~0.54 ms of a 0.82 ms inline stage;
the stage is now ~0.61 ms, so ~0.21 ms is recovered and the rest came back as the
tree build the selector passes need (about half of the new stage is tokenizing,
a third is matching). Removing the tokenizer as well means having components emit
elements instead of text - a change to every component and template, out of scope
here. The numbers come from `BenchmarkInlineStage` and `BenchmarkRenderMJML`.

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

The same probe works for the parser fidelity fixtures: on the parent commit
`testdata/parity/inline-implied-end-tags.mjml` and
`testdata/parity/inline-self-closed-html.mjml` fail (three and one mismatch), and
both pass with the implied-close table and the HTML-mode self-closing rule.

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
no raw non-breaking spaces in that document), and only the reference's own
`<o:AllowPNG/>` and author raw content still carry `/>`. Every fixture's document
is unchanged; only this serialization detail moved.

A self-closing `<td ... />` inside a `<tr>` is an open tag to every HTML parser,
so the emitted form and the expanded form build the same tree - the parity
harness parses both to the same document. A consumer that needs the reference's
own text formatting is out of luck anyway: the reference pretty-prints.

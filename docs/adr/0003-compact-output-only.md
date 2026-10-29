---
status: accepted
---

# Compact output only

The renderer emits compact HTML and offers no output-formatting option. MJML's
`beautify` and `minify` configuration keys are not implemented, not mapped onto
any renderer option, and are out of scope permanently; the equivalent CLI flags
are only presentation switches and do not change the document's meaning.

## Why

- Formatting is not part of the contract. The parity harness normalizes
  whitespace, attribute order, class tokens and CSS formatting before comparing,
  so a pretty-printed and a compact rendering of the same component tree are
  equal by definition. Paying for a formatter in the renderer would buy nothing
  the tests can see.
- It would sit on the hot path. The renderer writes output straight to the
  caller's `io.Writer`, and the render benchmarks are measured on that pipeline. Indenting every element means buffering, extra
  allocation and a second pass over a document that is already built.
- It can change how a mail client renders the result. Whitespace between
  inline-level nodes and inside table cells is layout-relevant in HTML email, so
  line breaks and indentation inserted by a formatter are a rendering risk that
  compact output does not take.
- Determinism. One output form is byte-reproducible for a given document and id
  generator, which makes fixtures, diffing and caching straightforward.

## Consequences

- Users who want readable HTML format the output afterwards, with their own tool
  and their own whitespace policy. The renderer never reformats a document.
- A fixture that records `--config.beautify` or `--config.minify` in its
  `.cli-args` sidecar fails as unmappable rather than being compared under a
  different configuration; such a fixture should not be added.
- The compact form is the only output the benchmarks, fuzz targets and parity
  comparisons cover, so there is no untested formatting path.

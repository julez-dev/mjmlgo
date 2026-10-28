# Upstream fixtures

MJML documents extracted from the upstream MJML test suite and rendered by both
implementations in `TestMJMLParity`.

| | |
| --- | --- |
| Source | https://github.com/mjmlio/mjml |
| Commit | `40ddfbb4c04e13dd9572aa0d5b1c58826a163289` |
| Extraction | `extract.mjs` (prints `{name, source, mjml}` JSON lines) |
| Curation | Documents from `packages/*/test/*.test.js` that MJML 4.18 renders without validation warnings. Documents that rely on illegal attributes (`mj-section gutter`) or unknown tags (`mj-head-component-with-function-style`), and documents using `mj-include` or JS interpolation, were excluded. |

File names are `<package-without-mjml-prefix>--<test-name>--<index>.mjml`.

These fixtures intentionally cover behaviour the hand-written corpus does not:
accordion font inheritance, wrapper `gap`, section gutter, table
`cellspacing`/width, social icon height/alignment, navbar icon padding,
`mj-html-attributes`, HTML comments, column/wrapper border-radius and
hero padding.

To refresh after a MJML upgrade, re-run the extraction, re-apply the curation
rule, update the commit above and re-record the parity baseline.

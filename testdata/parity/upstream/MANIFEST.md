# Upstream fixtures

MJML documents extracted from the upstream MJML test suite and rendered by both
implementations in `TestMJMLParity`. They are regenerated for each MJML major, so
this corpus describes the release the renderer targets.

| | |
| --- | --- |
| Source | https://github.com/mjmlio/mjml |
| Commit | `1c434fc` (tag `v5.4.1`) |
| Extraction | `extract.mjs` (prints `{name, source, mjml}` JSON lines) |
| Curation | Documents from `packages/*/test/*.test.js` that MJML 5.4 renders without validation warnings. Documents relying on unknown tags (`mj-head-component-with-function-style`), on `mj-include`, or on JS interpolation are excluded. 45 of the 46 documents the extractor finds survive that rule. |

File names are `<package-without-mjml-prefix>--<test-name>--<index>.mjml`.

Local edits: `parser-xml--preprocessors--00.mjml` declares the `mj-class` it
references. MJML only warns about an undeclared class, while this implementation
rejects it, so the declaration was added without changing the rendered HTML.

These fixtures intentionally cover behaviour the hand-written corpus does not:
accordion font inheritance and padding, wrapper `gap`, the fourteen section
`gutter` cases, table `cellspacing`/width, social icon height/alignment, navbar
icon padding, `mj-html-attributes`, HTML comments, `htmlmin:ignore`, column and
wrapper border-radius, and template-syntax sanitisation.

To refresh after a MJML upgrade, re-run the extraction, re-apply the curation
rule, update the commit above and run the parity test; every difference it reports
must be reviewed.

## License

The documents here are extracted from MJML's test suite, which is MIT licensed:

> Copyright (c) 2016 Mailjet SAS, https://mjml.io

They remain under that license. Everything else in this repository is under the
MIT license in [`../../../LICENSE`](../../../LICENSE).

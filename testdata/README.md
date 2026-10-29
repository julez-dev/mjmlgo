# Test corpora

Two corpora, both driven by fixtures on disk:

| Directory | Purpose | Test |
| --- | --- | --- |
| `parity/` | Documents that must render **identically** to the reference MJML implementation. | `TestMJMLParity` |
| `invalid/` | Documents that must be **rejected** with a specific error. | `TestMJMLInvalid` |

The reference implementation is the real `mjml` CLI; its version is pinned
project-wide in `../package.json`.

## Parity corpus (`parity/`)

Every `.mjml` file below `parity/` (including subdirectories) is rendered by this
package and by the reference MJML CLI, then compared structurally:

- elements, attributes (inline styles included) and text content must match;
- attributes are sorted, CSS declaration order inside a `style` attribute is
  normalized, class tokens are sorted and whitespace is collapsed, so formatting
  noise does not matter; empty `style`, `class` and `alt` attributes are treated
  as absent (they carry no information);
- Outlook conditional comments are parsed and compared as part of the tree.
  Downlevel-reveal wrappers (`<!--[if !mso]><!-->`) have no payload but are kept,
  so their presence and position are compared too;
- every other comment is compared **byte for byte** at its document position,
  spacing inside the delimiters included. The reference implementation preserves
  author comments verbatim (`<!--     comment with 5 spaces     -->` stays as
  written), so a dropped or re-spaced comment is a real divergence;
- `<style>` bodies are parsed as CSS and compared rule by rule. Rule **and**
  declaration order are preserved because the cascade depends on them;
  formatting and comments are dropped.

`parity/upstream/` holds fixtures extracted from the upstream MJML test suite;
see `parity/upstream/MANIFEST.md` for provenance and the curation rule.

### Malformed inline CSS

Only `<mj-style inline="inline">` bodies are parsed (for the inliner). The
stylesheet parser loops forever on a rule-level `;` that never reaches a block,
so a recovery pass normalises the input first: a statement with no block and a
block with an empty prelude are dropped, a stray top-level `}` is dropped, an
unterminated block is closed at the end of the input, and an unterminated comment
drops the rest. Well-formed CSS is passed through byte for byte, so
`parity/style-malformed-comment.mjml` and every well-formed fixture match the
reference.

Recovery is **best effort and deliberately divergent**. Observation against the
pinned CLI, not a fixture: for `.inline-class font-weight: bold; } .ok { color: red }`
postcss reads everything up to `{` as one selector, so the junk and `.ok` merge
into an uncompilable selector that juice then discards - the valid rule is lost
with the invalid one. This implementation drops only the malformed statement and
keeps `.ok`, because losing valid styling silently is worse than keeping it. No
fixture asserts the reference's recovery for mixed invalid/valid input;
`component/csssanitize_test.go` pins ours. What is not negotiable is that a
document renders, or fails with a `*component.Error`, and never hangs.

### Per-fixture reference flags (`<fixture>.cli-args`)

Some features only show up under a non-default reference configuration
(`keepComments`, `fonts`, `ignoreIncludes`, `printerSupport`). A fixture is
rendered with reference flags by writing them to a sibling sidecar:
`testdata/parity/printer-support.mjml.cli-args` configures
`testdata/parity/printer-support.mjml`.

The sidecar format is **one reference CLI argument per line**. Blank lines and
lines starting with `#` are ignored; every other line is passed to the reference
CLI unchanged, with surrounding whitespace trimmed. One argument per line keeps
values that contain spaces (a URL in the `-c.fonts` JSON, for example)
unambiguous:

```text
# comments are dropped, so the fixture proves the option
-c.keepComments=false
```

The harness maps the flags it understands onto this package's renderer options,
so the reference and this implementation are compared under the same
configuration. An unknown or unmappable flag fails the test instead of silently
comparing the reference's configured output against our defaults. The mapping is
deliberately small:

| Reference flag | Renderer option |
| --- | --- |
| `-c.keepComments=<value>` | `WithComments` |
| `-c.fonts=<json object>` | `WithFonts` |
| `-c.ignoreIncludes=<value>` | `WithIgnoreIncludes` |
| `-c.printerSupport=<value>` | `WithPrinterSupport` |

`-c.fonts` must be a JSON **object**. That is the only value the CLI turns into a
registry: `null`, an empty value and `false` reach mjml-core as the string the
CLI saw, and `[]` as an array, so the registry stops being a name-to-URL map.
`<mj-font>` then throws (`Cannot create property 'Unused Font' on string
'null'`), and a used family is imported once per character of the string
(`<link href="n">`, `<link href="u">`, ...). A sidecar that asks for such a value
cannot be mapped onto an option and fails the test; omit the flag for the default
registry, and use `-c.fonts={}` to replace the defaults with nothing.

The registry then reaches this package as a Go map, which has no order of its
own; a custom registry is therefore emitted in ascending family-name order (see
`WithFonts`). A fixture that uses several custom families must list them in the
sidecar in alphabetical order so both sides agree.

`--config.<name>=<value>` is accepted as well. The value semantics follow the
reference CLI, which is not uniform: `keepComments` is off only for the literal
`false`, while `ignoreIncludes` and `printerSupport` are handed to mjml-core as
strings and every non-empty string is truthy - so `-c.printerSupport=false`
*enables* the option in the reference run, and sidecars use `true`.

`keepComments` is the one option that changes the AST: the reference parser
represents each author comment as an `mj-raw` node at its document position and
drops them at parse time when the option is off. Comments inside ending tags
(`mj-text`, `mj-table`, `mj-raw`, `mj-button`, ...) are part of that content and
are never affected. Outlook conditional comments the renderer generates itself
are emitted after parsing and are never affected either. The fixture
`comments-keep-false.mjml` covers both halves.

Because a comment is an `mj-raw` node, it is also subject to the same rules:
a comment inside a component that does not accept `mj-raw` (`mj-carousel`,
`mj-html-attributes`, ...) is a validation error while comments are on, and
only `-c.keepComments=false` drops it early enough to be accepted. Comments are
parsed as XML comments, so a comment that is not well-formed XML - one
containing `--`, for instance - is rejected even though the reference's lenient
HTML parser tolerates it; this is the same strictness that rejects mismatched
tags.

### Configuring the MJML version

```json
"devDependencies": {
  "mjml": "4.18.0"
}
```

Install once with `npm install` (Node.js >= 18), then `go test ./...`.

| Variable | Effect |
| --- | --- |
| `MJML_VERSION` | Use another MJML version for this run, e.g. `MJML_VERSION=4.15.3 go test ./...`. Resolved through `npx`, so `node_modules` is optional. |
| `MJML_BIN` | Use an existing CLI executable instead, e.g. `MJML_BIN=$(which mjml)`. |
| `MJML_SKIP` | Skip the parity tests (useful where Node.js is unavailable). |

`go test -short ...` skips them too. A resolved version that differs from the pin
fails the test instead of silently comparing against the wrong reference.

#### Bumping MJML

```bash
npm install --save-dev --save-exact mjml@<new-version>
go test ./ -run TestMJMLParity -v   # every difference is reported with its path
```

### Zero tolerance

Every fixture must match MJML exactly; there is no allowlist of accepted
differences and no baseline file to re-record. A structural difference fails the
test with the element path and the expected/actual value.

When bumping MJML, run the suite and review the failures: each one is a real
reference change. If it is intended, fix the implementation (or add a focused
fixture for the new behaviour); if the new output is only a formatting change,
widen the CSS/HTML normalizer in `internal/mjmltest` deliberately, with a test.

### Failure artifacts

Failing fixtures write both documents to `.failures/` (git-ignored):

- `<name>.want.html` / `<name>.got.html` — raw HTML from MJML and from this package;
- `<name>.want.canonical.txt` / `<name>.got.canonical.txt` — the normalized trees
  that are actually compared, diffable with any diff tool.

## Invalid corpus (`invalid/`)

This implementation validates hard and fails fast, while MJML only warns and
keeps rendering. Documents that MJML tolerates but we reject therefore live
here, not in the parity corpus.

Each fixture needs a sibling `<name>.mjml.error.txt` containing the error text
(or a stable substring of it) that the renderer must return:

```text
testdata/invalid/invalid-color.mjml
testdata/invalid/invalid-color.mjml.error.txt   ->  chartreuseish not a valid color string
```

## Adding fixtures

- Parity: drop a `.mjml` file under `parity/`. It must match the reference
  implementation immediately. Keep fixtures small and focused: one fixture per
  feature area makes a failure easy to interpret.
- A large, multi-feature fixture is useful as an end-to-end performance document,
  but it must be synthetic: never commit a customer or production document. Add
  focused fixtures instead of growing one.
- Invalid: add the fixture, its `.error.txt`, and an entry in the
  `invalidFailures` table in `render_test.go`. The `.error.txt` is a stable
  substring of the message; the table is the structured contract the fixture must
  meet (`errors.As` to `*Error`, the expected kind, a non-empty element path, the
  attribute and value where applicable, and the sentinel `errors.Is` must still
  find). A fixture without an entry fails the test, and an entry without a
  fixture fails too.
- Include targets go into a `partials/` directory next to the fixture that
  includes them. Both corpora render each fixture through the same entry point a
  caller uses — the fixture's own directory is the root of the `fs.FS` — so
  relative includes resolve the same way the reference CLI resolves them from the
  fixture path. `partials/` directories are skipped when fixtures are collected:
  they are read by `mj-include`, not rendered on their own. The parity fixture
  `parity/includes/main.mjml` covers a head include, a full-document include,
  nested includes, `type="html"` and `type="css"`; the error cases
  (cycle, path escaping the corpus, missing file, unknown type) live in
  `invalid/include-*.mjml` with their include targets in `invalid/partials/`.

## Benchmarks

`BenchmarkRenderMJML` in `../render_test.go` measures the full pipeline per
fixture, so the numbers describe realistic documents rather than toy inputs:

```sh
go test ./ -run XXX -bench BenchmarkRenderMJML -benchmem -count=5
```

Benchmarks are a local tool; CI does not run them.

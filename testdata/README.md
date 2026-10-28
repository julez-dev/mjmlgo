# Test corpora

Two corpora, both driven by fixtures on disk:

| Directory | Purpose | Test |
| --- | --- | --- |
| `parity/` | Documents that must render **identically** to the reference MJML implementation. | `TestMJMLParity` |
| `invalid/` | Documents that must be **rejected** with a specific error. | `TestMJMLInvalid` |

The reference implementation is the real `mjml` CLI; its version is pinned
project-wide in `../package.json`. See `../PLAN.md` for the overall roadmap.

## Parity corpus (`parity/`)

Every `.mjml` file below `parity/` (including subdirectories) is rendered by this
package and by the reference MJML CLI, then compared structurally:

- elements, attributes (inline styles included) and text content must match;
- attributes are sorted, CSS declaration order inside a `style` attribute is
  normalized, class tokens are sorted and whitespace is collapsed, so formatting
  noise does not matter;
- comments are ignored except Outlook conditional comments, whose payload
  (`<table><tr><td>` MSO markup) is parsed and compared as part of the tree;
- `<style>` bodies are parsed as CSS and compared rule by rule. Rule **and**
  declaration order are preserved because the cascade depends on them;
  formatting and comments are dropped.

`parity/upstream/` holds fixtures extracted from the upstream MJML test suite;
see `parity/upstream/MANIFEST.md` for provenance and the curation rule.

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
go test ./ -run TestMJMLParity -update   # re-record the baseline
git diff testdata/parity-baseline.json   # review every HTML change
```

### Baseline of known divergences

`parity-baseline.json` records the accepted differences per fixture, scoped to
the MJML version it was recorded with:

```json
{
  "mjmlVersion": "4.18.0",
  "fixtures": {
    "testdata/parity/simple.mjml": {
      "mismatches": [
        {
          "path": "/#document/html[1]/body[1]/div[1]",
          "detail": "attribute \"role\" missing, want \"article\""
        }
      ]
    },
    "testdata/parity/raw.mjml": {
      "error": "failed to render column: invalid child type mj-raw in column, ..."
    }
  }
}
```

The test compares the *set* of differences, not just their count:

- an unrecorded difference (or a new render error) fails — no regressions;
- a recorded difference that disappeared also fails, so fixes must update the
  baseline and thereby shrink it.

Fixtures without an entry must match MJML exactly. Re-record with:

```bash
go test ./ -run TestMJMLParity -update
```

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

- Parity: drop a `.mjml` file under `parity/` and re-record the baseline. Keep
  fixtures small and focused: one fixture per feature area keeps each baseline
  entry interpretable. Fixtures that cannot render yet are still valuable — they
  record the missing feature.
- Invalid: add the fixture plus its `.error.txt`.

`mj-include` works because the reference CLI runs on the fixture path itself, so
relative includes resolve as usual. The Go implementation is being reworked to
resolve includes through a caller-provided `fs.FS` (see `PLAN.md`, M2).

## Benchmarks

`../benchmarks/BUDGETS.md` holds the end-to-end budget table
(`BenchmarkRenderMJML`); CI reports benchmark results without gating on them.

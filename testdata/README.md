# MJML parity tests

`TestMJMLParity` (in `../render_test.go`) renders every `.mjml` file below this
directory twice:

1. with this Go implementation (`RenderMJML`), and
2. with the **reference JavaScript implementation** of MJML, using the real
   `mjml` CLI.

Both HTML documents are parsed, normalized and compared structurally, so
formatting, attribute order and CSS declaration order do not matter, while
elements, attributes (including inline styles), text content and Outlook
conditional markup do.

## Configuring the MJML version

The reference version is pinned project-wide in `../package.json`:

```json
"devDependencies": {
  "mjml": "4.18.0"
}
```

Install it once with `npm install` (Node.js >= 18, `node_modules/` is ignored by
git) and run the tests with `go test ./...`.

Overrides:

| Variable      | Effect                                                                 |
| ------------- | ---------------------------------------------------------------------- |
| `MJML_VERSION`| Use another MJML version for this run, e.g. `MJML_VERSION=4.15.3 go test ./...`. Resolved through `npx`, so it works even if `node_modules` is absent. |
| `MJML_BIN`    | Use an existing CLI executable instead, e.g. `MJML_BIN=$(which mjml)`.  |
| `MJML_SKIP`   | Skip the parity tests entirely (useful where Node.js is unavailable).   |

`go test -short ...` also skips them. If the resolved version does not match the
pinned one, the test fails instead of silently comparing against the wrong
reference.

### Bumping MJML

```bash
npm install --save-dev --save-exact mjml@<new-version>
go test ./ -run TestMJMLParity -update   # re-record the baseline
git diff testdata/parity-baseline.json   # review every HTML change
```

The baseline records the MJML version it was created with. Comparing against a
different version fails until the baseline is re-recorded, so an accidental
version drift cannot go unnoticed.

## Baseline of known divergences

This implementation is not yet byte-for-byte identical to MJML. The accepted
differences per fixture are recorded in [`parity-baseline.json`](parity-baseline.json):

```json
{
  "mjmlVersion": "4.18.0",
  "fixtures": {
    "testdata/simple.mjml": {
      "mismatches": [
        {
          "path": "/#document/html[1]/body[1]/div[1]",
          "detail": "attribute \"role\" missing, want \"article\""
        }
      ]
    },
    "testdata/raw.mjml": {
      "error": "failed to render column: invalid child type mj-raw in column, ..."
    }
  }
}
```

The test compares the *set* of differences, not just their count:

- a new difference (or a new render error) fails the test — no regressions;
- a recorded difference that is gone also fails the test, so fixing the
  implementation requires updating the baseline and thereby shrinking it.

Regenerate after intentional changes with:

```bash
go test ./ -run TestMJMLParity -update
```

Fixtures without an entry must match MJML exactly. To check current progress:

```bash
go test ./ -run TestMJMLParity -update && git diff --stat testdata/parity-baseline.json
```

### Failure artifacts

When a fixture fails, both documents are written to `.failures/` (git-ignored)
for inspection:

- `<name>.want.html` / `<name>.got.html` — raw HTML from MJML and from this package;
- `<name>.want.canonical.txt` / `<name>.got.canonical.txt` — the normalized trees
  that are actually compared, diffable with any diff tool.

## Adding fixtures

Drop a `.mjml` file anywhere below `testdata/` (subdirectories are supported)
and re-record the baseline. Keep fixtures small and focused: one fixture per
feature area makes a baseline entry easy to interpret. Fixtures that currently
cannot be rendered at all are still valuable — they record the missing feature.

Note that `mj-include` works because the reference CLI runs on the fixture path
itself, so relative includes resolve as usual.

---
status: accepted
---

# The reference implementation is the MJML npm CLI

MJML has no normative specification, so the `mjml` npm package is the only
definition of what a correct document looks like. This project therefore treats
the reference JavaScript CLI as the oracle and measures itself against it: the
harness renders each fixture with the CLI, renders the same fixture with this
implementation, and compares the two as normalized trees. A green unit test is
not evidence of parity; a diff against `./node_modules/.bin/mjml <fixture> -s`
is.

## Version pinning

The reference version is pinned exactly in `package.json`
(`devDependencies.mjml`, currently `4.18.0`), because MJML changes its output
between releases and a parity claim is only meaningful for one version. The
harness resolves the CLI in this order:

1. `MJML_BIN`, an existing executable (no version resolution, no download);
2. the local `node_modules/.bin/mjml`, when its package version equals the pin;
3. `npx --yes mjml@<version>`, when Node.js is available;
4. otherwise the parity tests skip with instructions (or `MJML_SKIP` skips them
   deliberately).

`MJML_VERSION` overrides the pin for a single run, e.g. `MJML_VERSION=4.15.3 go
test ./...`, and is used when evaluating a bump. A resolved version that differs
from the requested one fails the test: comparing against the wrong reference is
worse than not comparing.

## Comparison rules

Both documents are parsed into the same normalized tree before comparing, so
formatting is not part of the contract:

- attributes are sorted by name; declarations inside a `style` attribute are
  parsed and sorted; class tokens are sorted; whitespace is collapsed; empty
  `style`, `class` and `alt` attributes are treated as absent because they carry
  no information;
- Outlook conditional comments (`<!--[if mso]>...<![endif]-->`) are parsed and
  compared as part of the tree, and downlevel-reveal wrappers are kept, so their
  presence and position are compared;
- every other comment is compared byte for byte at its document position, since
  the reference preserves author comments verbatim;
- `<style>` bodies are parsed as CSS and compared rule by rule with rule and
  declaration order preserved, because the cascade depends on both; formatting
  and comments inside the CSS are dropped;
- generated ids (carousel, navbar hamburger) are normalized as a safety net,
  because the reference uses random values; deterministic renders use
  `WithIDGenerator` instead.

Per-fixture reference flags live in a `<fixture>.cli-args` sidecar, so the
reference and this implementation are compared under the same configuration. The
harness maps only the flags it understands onto renderer options; an unknown or
unmappable flag fails the fixture instead of comparing the reference's configured
output against our defaults.

## Consequences

- Parity tests need a Node.js toolchain, and CI installs the pinned reference
  before running them. Offline runs need a warm npm/npx cache or a local
  `node_modules`.
- There is no allowlist and no baseline file. A structural difference is a defect
  to fix, never an expectation to re-record; re-recording an expectation to
  explain away a divergence is itself a defect.
- If a difference turns out to be formatting only, the normalizer is widened
  deliberately and with a test of its own - not silently.
- The pin is bumped by installing the new version and running the parity suite,
  then fixing every difference it reports. Widening the pin without doing that
  invalidates the parity claim.

# mjmlgo: plan of record

Goal: a complete, fast, well-tested Go implementation of MJML 4.18.0, verified
against the reference JavaScript implementation.

## Acceptance criteria

| Area | Definition of done |
| --- | --- |
| Parity | Every fixture in `testdata/parity/` renders structurally identical HTML to MJML 4.18.0 (normalized DOM **and** normalized CSS). `testdata/parity-baseline.json` shrinks to empty and is then deleted. |
| Components | Every MJML 4.18 component renders: body (`mj-body`, `mj-section`, `mj-wrapper`, `mj-column`, `mj-group`, `mj-text`, `mj-image`, `mj-button`, `mj-divider`, `mj-spacer`, `mj-table`, `mj-raw`, `mj-social`, `mj-social-element`, `mj-navbar`, `mj-navbar-link`, `mj-accordion`, `mj-accordion-element`, `mj-accordion-title`, `mj-accordion-text`, `mj-carousel`, `mj-carousel-image`, `mj-hero`) and head (`mj-head`, `mj-attributes` with `mj-all`/`mj-class`/per-tag, `mj-breakpoint`, `mj-font`, `mj-html-attributes`, `mj-preview`, `mj-style`, `mj-title`, `mj-raw`), plus parser-level `mj-include` and `mj-raw position="file-start"`. `mj-map` does not exist in MJML 4.18 and is out of scope. |
| Validation | Fail-fast, hard errors: unknown tags, unknown/invalid attributes, invalid nesting (driven by MJML's own `dependencies.js`), missing required children. Structured errors carry the element path and attribute. This is an intentional divergence from MJML's soft validation; invalid input is covered by `testdata/invalid/`. |
| API | `NewRenderer(Options)` + `Render(ctx, fs.FS, name, io.Writer)`; `Options` carries `IDGenerator`, custom components, and feature toggles; `RenderString`/`RenderFile`/`RenderBytes` conveniences; `RenderMJML` kept as a deprecated wrapper. Compact output only (no beautifier). |
| Performance | Benchmarks committed with a documented budget table; CI reports but does not gate. |
| Coverage | Reported in CI (no numeric gate). Quality is enforced by parity, CSS comparison, fuzz targets and component unit tests. |

## Measured starting point (MJML 4.18.0, `HEAD` before the rewrite)

`input_big.mjml` (23,930 B) end to end: **2.85 ms, 3.74 MB, 34,253 allocs**.

| Stage | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `parse` | 372,179 | 332,070 | 3,681 |
| component render | 1,113,938 | 1,721,780 | 19,379 |
| `inlineCSS` | 873,642 | 1,088,996 | 10,688 |

Top allocation sites: inline-style string concat in a loop (29.4% of objects),
`html.Parse` for CSS inlining (16%), whole-output conditional-comment
`ReplaceAllString` (13.2% of space), per-node `InitComponent` maps (18.5%).

Coverage: 50.7% own tests, 78.2% combined. `renderWithBackground` 0%. No fuzz,
no benchmarks, no golden files. The harness ignored `<style>` bodies, so
11,762 B of generated CSS was unverified.

Known divergences at the start: 214 records / 15 root causes (see
`testdata/parity-baseline.json`), plus one render error (`mj-raw` in
`mj-column`). 89 records came from a single bug: `mj-text` reading
`text-align` instead of `align`.

## Decisions

| Decision | Choice |
| --- | --- |
| Reference version | MJML 4.18.0 pinned in `package.json`; `MJML_VERSION`/`MJML_BIN`/`MJML_SKIP` override per run. Baseline is version-scoped. |
| Parity comparison | Normalized DOM: attributes sorted, style declarations sorted, class tokens sorted, whitespace collapsed, comments dropped except Outlook conditionals (parsed and compared). `<style>` bodies compared as normalized CSS (rule order preserved, declarations in order, whitespace/comments normalized, `@media` nesting preserved). |
| Corpus | `testdata/parity/` hand-written fixtures plus a curated, attributed subset of upstream MJML test inputs; `testdata/invalid/` fail-fast error fixtures. |
| Error model | Fail fast, no partial output. |
| Includes | `mj-include` resolves through the caller's `fs.FS` entry point (`fs.Glob` for patterns); no implicit filesystem or network access. |
| Package layout | Implementation moves to `internal/` (`internal/component`, `internal/node`, parser/renderer). Public surface: root package only. |
| Inliner | Own inliner emitting into the component tree, no `html.Parse`/`html.Render` round-trip; juice-compatible specificity and `!important`; fast path when nothing needs inlining. Focused dependencies allowed case by case with a written justification. |
| Determinism | `Options.IDGenerator` (default random) for carousel/navbar ids; the harness also normalizes generated ids as a safety net. |
| Extensibility | Public custom-component interface via `Options`, with a small stable contract shared by internal and user components. |
| Output formatting | Compact HTML only. |

## Status

- **M0 done** - this document.
- **M1 done** - parity/invalid corpora split, CSS comparison in the harness,
  31 curated upstream fixtures with provenance, benchmarks plus budget table,
  fuzz targets, CI workflow, coverage reporting.
- **First fix batch done** - `mj-text` alignment, body accessibility wrapper
  (role/aria), `mj-head` component name, stray brace in the font style block,
  debug output in the hero and social renderers, hex shorthand expansion with
  case preservation, and removal of the conditional-comment merging pass (which
  also removed a whole-output regexp). Divergences: 334 -> 288.

## Roadmap

| Milestone | Content |
| --- | --- |
| M0 | This document. |
| M1 | Harness v2: parity/invalid corpora, CSS comparison, curated upstream fixtures with manifest, benchmarks with budget table, fuzz targets, CI workflow, coverage reporting. |
| M2 | Internal restructure (`internal/`), new public API (`Renderer`, `Options`, `Result`, structured `Error`), `fs.FS` input, `mj-include`, public custom-component interface, JS-parity validation core. |
| M3 | Inliner on the component tree, parser/emitter performance work, budget table updated from measurements. |
| M4 | Missing components and head features: navbar, accordion, carousel, `mj-class`/per-tag attributes, `mj-html-attributes`, font registry with usage filtering, `owa`, `mj-raw position="file-start"`, `mj-wrapper gap`. |
| M5 | Divergence backlog to zero, delete `testdata/parity-baseline.json`, docs (`README.md`, `testdata/README.md`), final review. |

Every milestone keeps `go test ./...` green; commits are one component family or
one harness capability at a time.

## Divergence backlog (from the starting baseline)

Root causes, with the site responsible, in the order they should be fixed
(cheapest first; several are one-line bugs):

1. `mj-text` uses attribute `text-align` instead of `align` (`component/mjml_text.go`) - 89 records.
2. Missing `role="article"` / `aria-roledescription="email"` on the body div (`component/mjml_body.go`) - 22.
3. Section `border-radius` handling: `overflow:hidden` on the inner div, `border-collapse:separate` on the table, radius on the cell (`component/mjml_section.go`) - 18.
4. `ctx.ContainerWidth` not reduced to the section content width before rendering children, so divider widths are wrong (`component/mjml_section.go`, `component/mjml_divider.go`) - 14.
5. MSO column seat uses the section width instead of the column width (`component/mjml_section.go`) - 8.
6. Social: missing `text-align` on the text cell, unconditional icon `height`, debug attribute `mjml-social-test` (`component/mjml_social*.go`) - 15.
7. Hex colours not expanded/case-preserved (`#fff` vs `#ffffff`) (`component/util.go`) - 7.
8. `mj-group` children not wrapped in the MSO `<td>` seat (`component/mjml_section.go`) - 5.
9. Stray `}` per font in `head-style-section.tmpl` (head content leaks into body) - 4.
10. Divider outlook width truncates percentages (`int(p)/100`) - 2.
11. Hero MSO conditional table missing `cellspacing="0"`; debug `fmt.Println` in the hero renderer - 3.
12. `mj-image` `border-radius`, gutter column background colour - 2.
13. `mj-raw` as a child of `mj-column`/`mj-section`/`mj-group` - 1 render error, and the
    reference CLI cannot render `mj-raw`-bearing fixtures until this is fixed.
14. Font registry and usage filtering (default web fonts, no unconditional import) - 24.
15. `owa="desktop"` CSS, dead `printSupport` branches - not visible to the old harness.
16. Outlook conditional comment grouping: MJML emits one conditional comment per
    contiguous MSO fragment (e.g. `</td><td ...>`), while components here emit one
    per tag. Functionally equivalent in Outlook, structurally different. Requires
    changing conditional emission in section, column, group, hero and social.
17. `mj-table width="auto"` is rejected by validation (MJML allows it).

## Budgets

Budgets are recorded next to the benchmarks (`benchmarks/BUDGETS.md`) and
updated when a measurement changes materially. They describe per-fixture
end-to-end cost and per-stage cost; CI prints the comparison without failing.

## Out of scope

- MJML 5.x behaviour (separate migration).
- MJML CLI options that only affect presentation: `--config.beautify`,
  `--config.minify`, `--config.filePath`, watch mode.
- The MJML "skeleton" HTML customisation hook and `juiceOptions` passthrough are
  deferred; revisit if a consumer needs them.

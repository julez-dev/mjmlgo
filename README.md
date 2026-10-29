# mjmlgo

A pure-Go renderer for [MJML](https://mjml.io), the email markup language. It
targets **MJML 5** (currently 5.4.1): an `.mjml` document becomes the HTML an
email client can display, the same document the reference JavaScript
implementation produces — without Node.js, a browser, or CGO at runtime.

```go
var out bytes.Buffer
if err := mjmlgo.Render(&out, src); err != nil {
	return err
}
```

There is also a command line compiler, [covered below](#command-line).

| Version | Implements | Import path |
| --- | --- | --- |
| v2 — this one | MJML 5.4.1 | `github.com/julez-dev/mjmlgo/v2` |
| v1 | MJML 4.18.0 | `github.com/julez-dev/mjmlgo` |

The two lines are independent: **v1 implements MJML 4**, and stays at its own tag
for documents that must keep rendering exactly as MJML 4 rendered them.

## What it is

- **Parity, not approximation.** Every fixture in `testdata/parity` (105 documents,
  including ones derived from the upstream MJML test suite) is rendered by this
  package and by the pinned `mjml` CLI, then compared element by element:
  attributes, inline styles, rule order, and comments (byte for byte). No
  allowlist, no recorded baselines. See [`testdata/README.md`](testdata/README.md).
- **No runtime dependencies.** A Go program that renders email needs no Node.js
  toolchain, no `node_modules`, and no subprocess.
- **Fail-fast validation.** A document with an unknown element, a bad attribute
  value, or a child in the wrong place is an error with a structured value
  attached, never partially rendered HTML.
- **Writer-based output.** The caller supplies the `io.Writer`, so a
  `bytes.Buffer` reused across renders keeps its capacity and a server writes
  straight to its response body or a file.
- **Concurrency-safe renderers.** `Renderer` holds configuration only; all
  per-render state lives in the call.

Pinning one release is deliberate: `package.json` fixes the reference version and
the tests assert it, so the suite never compares against a different one by
accident.

## Install

```sh
go get github.com/julez-dev/mjmlgo/v2
go install github.com/julez-dev/mjmlgo/v2/cmd/mjmlgo@latest
```

The module path carries the major version, so v1 (MJML 4) and v2 (MJML 5) can be
used by the same program, or even the same package.

Requires Go 1.27 or newer.

## Library

Render a document from memory:

```go
import (
	"bytes"

	"github.com/julez-dev/mjmlgo/v2"
)

func render(src []byte) ([]byte, error) {
	var out bytes.Buffer

	if err := mjmlgo.Render(&out, src); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}
```

Configure a reusable renderer with the reference implementation's knobs
(`WithComments`, `WithFonts`, `WithoutIncludes`, `WithPrinterSupport`), or
with `WithIDGenerator` for deterministic output and `WithComponents` for custom
components:

```go
renderer := mjmlgo.NewRenderer(
	mjmlgo.WithComments(false),
	mjmlgo.WithFonts([]mjmlgo.Font{
		{Family: "Inter", Href: "https://example.test/inter.css"},
	}),
)

var out bytes.Buffer
err := renderer.RenderFile(ctx, &out, "templates/welcome.mjml")
```

`RenderFile` reads the document from disk and scopes `mj-include` to the file's
own directory. `RenderFS` takes any `fs.FS` plus a name inside it, so documents
can come from an embedded filesystem, a zip, or a test fixture:

```go
embedFS, _ := fs.Sub(assets, "templates")

err := renderer.RenderFS(ctx, &out, embedFS, "welcome.mjml")
```

A document that uses `mj-include` needs one of those two: `Renderer.Render`
returns `ErrIncludeNoFS` rather than silently reading the process's working
directory. `WithoutIncludes()` is the way to ask for MJML 5's own default
instead — the element is dropped and the file it names is never opened:

```go
renderer := mjmlgo.NewRenderer(mjmlgo.WithoutIncludes())
```

### Errors

Validation and parse failures are `*mjmlgo.Error`. Inspect the fields instead of
matching message text:

```go
var invalid *mjmlgo.Error

if errors.As(err, &invalid) {
	log.Printf("%s at %s (attribute %s=%q)",
		invalid.Kind, strings.Join(invalid.Path, " > "), invalid.Attribute, invalid.Value)
}
```

`errors.Is` still finds the package sentinels (`ErrValidation`,
`ErrParsingFailed`, `ErrIncludeCycle`, …). I/O problems that are not about the
document — an unreadable file, a cancelled context, a nil writer — keep their
plain errors.

### Custom components

Implement `component.Component` (tag, allowed attributes with validators,
defaults, render), optionally `component.ChildComponent` for its children, and
register it on a renderer with `WithComponents`. A registered tag then behaves
like a built-in one: same attribute validation, same default cascade, same
`mj-attributes`/`mj-class` handling. Registration is per renderer, never global.
[`example/customcomponent`](example/customcomponent) is the worked example.

## Command line

```sh
mjmlgo newsletter.mjml                  # compact HTML on stdout
mjmlgo 'mail/*.mjml' -o dist            # one .html per input
mjmlgo -i -c filePath=templates < page.mjml
mjmlgo -v 'mail/**/*.mjml'              # validate only, no output
```

| Option | Meaning |
| --- | --- |
| `-r, --read <file...>` | Compile file(s); glob patterns, `**` included |
| `-i, --stdin` | Compile a document from standard input |
| `-v, --validate <file...>` | Validate without writing output |
| `-s, --stdout` | Write HTML to stdout (the default) |
| `-o, --output <path>` | Output file, or an existing directory |
| `--noStdoutFileComment` | Omit the `<!-- FILE: … -->` header on stdout |
| `-c, --config <key=value>` | `fonts`, `keepComments`, `filePath`, `allowIncludes`, `ignoreIncludes`, `includePath`; also `--config.<key>=<value>` |
| `--json` | One JSON record per input, for tooling |
| `-h, --help`, `-V, --version` | |

Exit codes match the reference CLI: `0` success, `1` command-line or validation
error, `2` when some input failed to render. A run in which *every* input failed
counts as a command-line error, which is also what the reference CLI does.
Failures are reported per file and the remaining inputs are still written.

## Supported MJML

The full 5.4.1 component set:

- **Layout** — `mj-body`, `mj-section`, `mj-wrapper`, `mj-group`, `mj-column`,
  `mj-hero`
- **Content** — `mj-text`, `mj-button`, `mj-image`, `mj-divider`, `mj-spacer`,
  `mj-table`, `mj-raw`, `mj-social`, `mj-social-element`, `mj-navbar`,
  `mj-navbar-link`, `mj-accordion` (+ `-element`, `-title`, `-text`),
  `mj-carousel` (+ `-image`)
- **Head** — `mj-head`, `mj-attributes` (`mj-all`, `mj-class`),
  `mj-html-attributes` (`mj-selector`), `mj-breakpoint`, `mj-font`,
  `mj-preview`, `mj-style` (including `inline="inline"` for juice-style CSS
  inlining), `mj-title`
- **Composition** — `mj-include` for `type="mjml"`, `"html"` and `"css"`
  (opt-in, see below), author comments, Outlook conditional comments, media
  queries, web fonts, printer styles
- **Layout details** — `gutter` on `mj-section`, which sizes the space between a
  section's columns and carries it into the responsive rules

Everything is emitted compact. There is no pretty-printer: formatting belongs to
whatever displays or transports the HTML.

## Differences from the reference implementation

| | reference `mjml` | mjmlgo |
| --- | --- | --- |
| Output | beautified by default | compact only; no `beautify`/`minify` |
| Invalid documents | warnings on stderr, HTML still produced | error, no output |
| `-v/--validate` | lists every warning | reports the first failure per file |
| `mj-include` | off unless `allowIncludes`; a denied path is a comment in the HTML plus a warning | same default; a path outside the filesystem root is an error |
| `mj-include` root | may climb with `../` up to `includePath` | confined to the `fs.FS` root |
| `--config.sanitizeStyles` | protects template syntax through CSS minification | accepted and ignored: there is no minifier to protect it from |
| Custom components | JavaScript packages, `.mjmlconfig` | Go types registered per renderer |
| Watch mode, MJML 3 migration | watch mode and `--migrate` | neither |
| Malformed XML (a comment containing `--`, say) | tolerated by the lenient parser | rejected |

The `mjml` CLI's `--config.beautify`, `--config.minify` and unknown config keys
are rejected with an explanation rather than silently ignored.

## Tests

The reference implementation is a test dependency, pinned in `package.json`. Its
version is read back from the CLI at test time, so a mismatch fails instead of
comparing against the wrong oracle.

```sh
npm install          # once, Node.js >= 18; fetches the pinned mjml CLI
go test ./...        # includes the parity suite
```

| Variable | Effect |
| --- | --- |
| `MJML_VERSION` | Use another MJML version for this run (resolved through `npx`) |
| `MJML_BIN` | Use an existing CLI executable instead |
| `MJML_SKIP` | Skip the reference comparison entirely (no Node.js needed) |

`go test -short ./...` skips the parity suite too. Fuzz targets `FuzzParse` and
`FuzzRenderMJML` live in the same package. [`testdata/README.md`](testdata/README.md)
documents both corpora, the per-fixture reference flags, and how to add fixtures.

## Performance

Rendering [`testdata/parity/simple-hero.mjml`](testdata/parity/simple-hero.mjml)
(603 bytes in, 4.3 KB out) through the library costs 42 µs, 62 KB of allocation
and 507 allocations on a 32-core Linux machine with Go 1.27.

End to end, one CLI process rendering the largest public fixture
(`button-complex.mjml`, 5.3 KB), median of ten invocations:

| | wall clock | max RSS |
| --- | --- | --- |
| `mjmlgo button-complex.mjml -o out.html` | 2.3 ms | 11 MB |
| `mjml button-complex.mjml -o out.html` | 181 ms | 181 MB |

Reproduce the library numbers with
`go test ./ -run XXX -bench BenchmarkRenderMJML -benchmem`, which runs the full
pipeline over every parity fixture. The CLI figures are wall clock around the
process and its peak RSS, so they include startup; a long-lived library caller
sees only the rendering cost.

## License

MIT — see [LICENSE](LICENSE).

The fixtures in [`testdata/parity/upstream`](testdata/parity/upstream) were
extracted from MJML's test suite and remain under MJML's MIT license
(Copyright (c) 2016 Mailjet SAS, https://mjml.io); see the manifest there.

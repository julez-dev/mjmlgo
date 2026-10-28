// Package mjmlgo renders MJML documents to HTML.
//
// The implementation targets MJML 4.18.0 exactly: for the supported feature
// set it produces the same document as the reference JavaScript
// implementation. Output is compact (never pretty-printed) and validation is
// fail-fast: unknown tags, invalid attributes or nesting, and missing required
// children are reported as errors instead of being warned about and ignored.
// Invalid input never produces partial output.
//
// Use NewRenderer to configure a renderer and Render, RenderString,
// RenderBytes or RenderFile to render a document:
//
//	renderer := mjmlgo.NewRenderer()
//	html, err := renderer.RenderString(ctx, os.DirFS("templates"), "welcome.mjml")
//
// Rendering reads the input through the provided fs.FS, so callers decide
// where documents come from and the renderer performs no implicit filesystem
// or network access. A Renderer is safe for concurrent use.
//
// mj-include is resolved through the same fs.FS: paths are relative to the
// directory of the file that contains the include, and type="html", type="css"
// and the default type="mjml" are supported. Include cycles, paths that would
// leave the filesystem and unknown include types are errors.
//
// The tree the renderer walks and the components it dispatches to live in the
// public node and component packages, so that custom components can receive the
// same values. Their exported shape is frozen; see their package documentation
// and docs/adr/0004-exported-internals-freeze.md.
//
// RenderMJML is kept for existing callers and is deprecated in favour of the
// Renderer API; it has no filesystem, so documents with mj-include are rejected.
package mjmlgo

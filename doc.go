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
// RenderMJML is kept for existing callers and is deprecated in favour of the
// Renderer API.
package mjmlgo

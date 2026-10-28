package mjmlgo_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/julez-dev/mjmlgo"
	"github.com/julez-dev/mjmlgo/component"
	"github.com/julez-dev/mjmlgo/example/customcomponent"
	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// renderWithComponents renders document with a renderer that registered the
// given components and returns the HTML.
func renderWithComponents(t *testing.T, document string, components ...component.Component) string {
	t.Helper()

	fsys := fstest.MapFS{"doc.mjml": &fstest.MapFile{Data: []byte(document)}}

	rendered, err := mjmlgo.NewRenderer(mjmlgo.WithComponents(components...)).
		RenderString(context.Background(), fsys, "doc.mjml")
	require.NoError(t, err)

	return rendered
}

// bodyContent returns the rendered content of the body div, so a test can assert
// exactly what a document produced without pinning the head boilerplate that the
// parity corpus already covers.
func bodyContent(t *testing.T, rendered string) string {
	t.Helper()

	const bodyDiv = `<div aria-roledescription="email" dir="auto" lang="und" role="article">` + "\n"

	start := strings.Index(rendered, bodyDiv)
	require.GreaterOrEqual(t, start, 0, "rendered document has no body div")

	rest := rendered[start+len(bodyDiv):]

	end := strings.Index(rest, "</div>\n</body>")
	require.GreaterOrEqual(t, end, 0, "rendered document has no closing body div")

	return rest[:end]
}

// TestWithComponentsRendersCustomComponent asserts the example component's
// output exactly, at its document position: mj-badge sits directly in mj-body,
// where a content leaf is allowed, and the component's own defaults are applied.
func TestWithComponentsRendersCustomComponent(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-badge text="Body" /></mj-body></mjml>`

	assert.Equal(t,
		`<table role="presentation" border="0" cellpadding="0" cellspacing="0">`+
			`<tbody><tr><td align="center" style="background-color:#1f2937;border-radius:4px;color:#ffffff;`+
			`font-family:Helvetica,Arial,sans-serif;font-size:13px;line-height:1;padding:8px 12px;">`+
			`Body</td></tr></tbody></table>`,
		bodyContent(t, renderWithComponents(t, document, customcomponent.Badge{})),
	)
}

// TestCustomComponentInColumn asserts a custom component is a content child of
// mj-column: it gets the column's cell wrapper, exactly like a built-in leaf.
func TestCustomComponentInColumn(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-section><mj-column>` +
		`<mj-badge text="New" background-color="#fde68a" color="#111827" />` +
		`</mj-column></mj-section></mj-body></mjml>`

	rendered := renderWithComponents(t, document, customcomponent.Badge{})

	assert.Contains(t, rendered, "<tr>\n<td style=\"font-size:0px;word-break:break-word;\">\n"+
		`<table role="presentation" border="0" cellpadding="0" cellspacing="0">`+
		`<tbody><tr><td align="center" style="background-color:#fde68a;border-radius:4px;color:#111827;`+
		`font-family:Helvetica,Arial,sans-serif;font-size:13px;line-height:1;padding:8px 12px;">`+
		"New</td></tr></tbody></table></td>\n</tr>")
}

// TestCustomComponentReceivesAttributeDefaults asserts the document-level
// default layers reach a custom tag: per-tag defaults from mj-attributes and
// attributes from a referenced mj-class are applied before Render runs, and the
// component's own default fills what neither layer set.
func TestCustomComponentReceivesAttributeDefaults(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-head><mj-attributes>` +
		`<mj-badge text="Default label" />` +
		`<mj-class name="accent" background-color="#fde68a" />` +
		`</mj-attributes></mj-head><mj-body><mj-section><mj-column>` +
		`<mj-badge mj-class="accent" />` +
		`</mj-column></mj-section></mj-body></mjml>`

	rendered := renderWithComponents(t, document, customcomponent.Badge{})

	assert.Contains(t, rendered,
		`<table role="presentation" border="0" cellpadding="0" cellspacing="0">`+
			`<tbody><tr><td align="center" style="background-color:#fde68a;border-radius:4px;color:#ffffff;`+
			`font-family:Helvetica,Arial,sans-serif;font-size:13px;line-height:1;padding:8px 12px;">`+
			"Default label</td></tr></tbody></table>")
}

// TestCustomComponentValidationErrors asserts every validation failure a custom
// tag can cause is a structured error carrying the element path.
func TestCustomComponentValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		document  string
		kind      component.ErrorKind
		path      []string
		attribute string
		value     string
	}{
		{
			name: "undeclared attribute",
			document: `<mjml><mj-body><mj-section><mj-column>` +
				`<mj-badge text="New" made-up="value" />` +
				`</mj-column></mj-section></mj-body></mjml>`,
			kind:      component.KindUnknownAttribute,
			path:      []string{"mj-body", "mj-section", "mj-column", "mj-badge"},
			attribute: "made-up",
			value:     "value",
		},
		{
			name: "invalid declared value",
			document: `<mjml><mj-body><mj-section><mj-column>` +
				`<mj-badge text="New" background-color="chartreuseish" />` +
				`</mj-column></mj-section></mj-body></mjml>`,
			kind:      component.KindInvalidValue,
			path:      []string{"mj-body", "mj-section", "mj-column", "mj-badge"},
			attribute: "background-color",
			value:     "chartreuseish",
		},
		{
			name: "disallowed child of a leaf",
			document: `<mjml><mj-body><mj-section><mj-column>` +
				`<mj-badge text="New"><mj-text>inside</mj-text></mj-badge>` +
				`</mj-column></mj-section></mj-body></mjml>`,
			kind:  component.KindDisallowedChild,
			path:  []string{"mj-body", "mj-section", "mj-column", "mj-badge", "mj-text"},
			value: "mj-text",
		},
		{
			name: "custom component where a layout element is expected",
			document: `<mjml><mj-body><mj-section>` +
				`<mj-badge text="New" />` +
				`</mj-section></mj-body></mjml>`,
			kind:  component.KindDisallowedChild,
			path:  []string{"mj-body", "mj-section", "mj-badge"},
			value: "mj-badge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{"doc.mjml": &fstest.MapFile{Data: []byte(tt.document)}}

			_, err := mjmlgo.NewRenderer(mjmlgo.WithComponents(customcomponent.Badge{})).
				RenderString(context.Background(), fsys, "doc.mjml")
			require.Error(t, err)

			var structured *component.Error
			require.ErrorAs(t, err, &structured)
			assert.Equal(t, tt.kind, structured.Kind)
			assert.Equal(t, tt.path, structured.Path)
			assert.Equal(t, tt.attribute, structured.Attribute)
			assert.Equal(t, tt.value, structured.Value)
		})
	}
}

// namedComponent is a component whose only interesting property is its tag, for
// the registration tests.
type namedComponent struct {
	tag string
}

func (c namedComponent) Name() string { return c.tag }

func (namedComponent) AllowedAttributes() map[string]component.AttributeValidator { return nil }

func (namedComponent) DefaultAttributes(*component.RenderContext) map[string]string { return nil }

func (namedComponent) Render(*component.RenderContext, io.Writer, *node.Node) error { return nil }

// TestWithComponentsRejectsInvalidRegistration asserts a renderer refuses a
// component it cannot register, with a structured error instead of a plain one.
// The failure is about the renderer's configuration, so it applies to every
// document, including one that never uses the tag.
func TestWithComponentsRejectsInvalidRegistration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		components []component.Component
		value      string
	}{
		{
			name:       "duplicate tag",
			components: []component.Component{customcomponent.Badge{}, customcomponent.Badge{}},
			value:      customcomponent.BadgeTagName,
		},
		{
			name:       "tag shadowing a built-in element",
			components: []component.Component{namedComponent{tag: component.TextTagName}},
			value:      component.TextTagName,
		},
		{
			name:       "empty tag",
			components: []component.Component{namedComponent{tag: ""}},
		},
		{
			name:       "nil component",
			components: []component.Component{nil},
		},
	}

	const document = `<mjml><mj-body><mj-section><mj-column>` +
		`<mj-text>built-in only</mj-text>` +
		`</mj-column></mj-section></mj-body></mjml>`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{"doc.mjml": &fstest.MapFile{Data: []byte(document)}}

			_, err := mjmlgo.NewRenderer(mjmlgo.WithComponents(tt.components...)).
				RenderString(context.Background(), fsys, "doc.mjml")
			require.Error(t, err)

			var structured *component.Error
			require.ErrorAs(t, err, &structured)
			assert.Equal(t, component.KindInvalidComponent, structured.Kind)
			assert.Equal(t, []string{component.MJMLTagName}, structured.Path)
			assert.Equal(t, tt.value, structured.Value)
			assert.NotErrorIs(t, err, component.ErrValidation, "registration is not document validation")
		})
	}
}

// TestCustomComponentsArePerRenderer asserts there is no global registry: the
// same document is accepted by a configured renderer and rejected by an
// unconfigured one in the same process.
func TestCustomComponentsArePerRenderer(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-section><mj-column>` +
		`<mj-badge text="New" />` +
		`</mj-column></mj-section></mj-body></mjml>`

	assert.Contains(t, renderWithComponents(t, document, customcomponent.Badge{}), ">New</td>")

	fsys := fstest.MapFS{"doc.mjml": &fstest.MapFile{Data: []byte(document)}}

	_, err := mjmlgo.NewRenderer().RenderString(context.Background(), fsys, "doc.mjml")
	require.Error(t, err)

	var structured *component.Error
	require.ErrorAs(t, err, &structured)
	assert.Equal(t, component.KindUnknownElement, structured.Kind)
	assert.Equal(t, "mj-badge", structured.Value)
}

// childListComponent is a ChildComponent whose only job is to declare its
// children: rendering them is the container work of ticket T4, so its Render
// writes a marker instead.
type childListComponent struct{}

func (childListComponent) Name() string { return "mj-panel" }

func (childListComponent) AllowedAttributes() map[string]component.AttributeValidator {
	return map[string]component.AttributeValidator{"title": func(string) error { return nil }}
}

func (childListComponent) DefaultAttributes(*component.RenderContext) map[string]string { return nil }

func (childListComponent) AllowedChildren() []string {
	return []string{component.TextTagName}
}

func (childListComponent) Render(_ *component.RenderContext, w io.Writer, n *node.Node) error {
	_, err := fmt.Fprintf(w, `<div class="mj-panel">%s</div>`, n.GetAttributeValueDefault("title"))

	return err
}

// TestCustomComponentAllowedChildren asserts a custom component's declared child
// list is enforced in both directions.
func TestCustomComponentAllowedChildren(t *testing.T) {
	t.Parallel()

	t.Run("a declared child is accepted", func(t *testing.T) {
		t.Parallel()

		const document = `<mjml><mj-body><mj-section><mj-column>` +
			`<mj-panel title="Panel"><mj-text>inside</mj-text></mj-panel>` +
			`</mj-column></mj-section></mj-body></mjml>`

		rendered := renderWithComponents(t, document, childListComponent{})
		assert.Contains(t, rendered, `<div class="mj-panel">Panel</div>`)
	})

	t.Run("an undeclared child is rejected", func(t *testing.T) {
		t.Parallel()

		const document = `<mjml><mj-body><mj-section><mj-column>` +
			`<mj-panel title="Panel"><mj-image src="a.png" /></mj-panel>` +
			`</mj-column></mj-section></mj-body></mjml>`

		fsys := fstest.MapFS{"doc.mjml": &fstest.MapFile{Data: []byte(document)}}

		_, err := mjmlgo.NewRenderer(mjmlgo.WithComponents(childListComponent{})).
			RenderString(context.Background(), fsys, "doc.mjml")
		require.Error(t, err)

		var structured *component.Error
		require.ErrorAs(t, err, &structured)
		assert.Equal(t, component.KindDisallowedChild, structured.Kind)
		assert.Equal(t, []string{"mj-body", "mj-section", "mj-column", "mj-panel", "mj-image"}, structured.Path)
		assert.Equal(t, "mj-image", structured.Value)
	})
}

// TestComponentRenderErrorIsPropagated asserts a custom component's render
// failure stops the render and reaches the caller.
func TestComponentRenderErrorIsPropagated(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{"doc.mjml": &fstest.MapFile{Data: []byte(
		`<mjml><mj-body><mj-section><mj-column><mj-badge text="New" /></mj-column></mj-section></mj-body></mjml>`,
	)}}

	_, err := mjmlgo.NewRenderer(mjmlgo.WithComponents(failingComponent{})).
		RenderString(context.Background(), fsys, "doc.mjml")
	require.Error(t, err)
	assert.ErrorIs(t, err, errCustomRender)
}

var errCustomRender = errors.New("custom render failed")

// failingComponent is a custom component whose Render always fails.
type failingComponent struct{}

func (failingComponent) Name() string { return customcomponent.BadgeTagName }

// AllowedAttributes also covers the nil-validator rule: the entry accepts every
// value instead of panicking.
func (failingComponent) AllowedAttributes() map[string]component.AttributeValidator {
	return map[string]component.AttributeValidator{"text": nil}
}

func (failingComponent) DefaultAttributes(*component.RenderContext) map[string]string {
	return nil
}

func (failingComponent) Render(*component.RenderContext, io.Writer, *node.Node) error {
	return errCustomRender
}

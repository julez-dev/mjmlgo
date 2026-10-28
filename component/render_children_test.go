package component

import (
	"io"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// textHTML is the exact markup mj-text renders for content with its default
// attributes (mj-text's own style; the padding default belongs to the <td> its
// container wraps it in). It is written out rather than produced by the
// component so a change to mj-text's defaults fails this expectation.
func textHTML(content string) string {
	return `<div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:1;` +
		`text-align:left;color:#000000;">` + content + "</div>\n"
}

// renderingComponent is a custom component that writes html and nothing else, so
// the dispatch tests can see exactly which child rendered.
type renderingComponent struct {
	tag  string
	html string
}

func (c renderingComponent) Name() string { return c.tag }

func (renderingComponent) AllowedAttributes() map[string]AttributeValidator { return nil }

func (renderingComponent) DefaultAttributes(*RenderContext) map[string]string { return nil }

func (c renderingComponent) Render(_ *RenderContext, w io.Writer, n *node.Node) error {
	_, err := io.WriteString(w, c.html)

	return err
}

// TestRenderChildrenRendersBuiltInChildren asserts a container's children are
// rendered by the built-in components in document order, with their defaults
// applied to the node in place, exactly as the built-in content containers do.
func TestRenderChildrenRendersBuiltInChildren(t *testing.T) {
	t.Parallel()

	first := &node.Node{Type: TextTagName, Content: "One"}
	second := &node.Node{Type: TextTagName, Content: "Two"}
	container := &node.Node{Type: "mj-notice", Children: []*node.Node{first, second}}

	var output strings.Builder
	require.NoError(t, RenderChildren(&RenderContext{}, &output, container))

	assert.Equal(t, textHTML("One")+textHTML("Two"), output.String())
	assert.Equal(t, "13px", first.GetAttributeValueDefault("font-size"), "defaults are applied in place")
}

// TestRenderChildrenRendersRegisteredComponents asserts children that are custom
// components render through the registry the render context carries.
func TestRenderChildrenRendersRegisteredComponents(t *testing.T) {
	t.Parallel()

	ctx := contextWith(renderingComponent{tag: "mj-chip", html: `<span class="mj-chip">One</span>`})
	container := &node.Node{
		Type:     "mj-notice",
		Children: []*node.Node{{Type: "mj-chip", Content: "One"}},
	}

	var output strings.Builder
	require.NoError(t, RenderChildren(ctx, &output, container))

	assert.Equal(t, `<span class="mj-chip">One</span>`, output.String())
}

// containerComponent is a custom container that renders its children with
// RenderChildren and nothing else, so the dispatch tests can see it recurse.
type containerComponent struct {
	tag string
}

func (c containerComponent) Name() string { return c.tag }

func (containerComponent) AllowedAttributes() map[string]AttributeValidator { return nil }

func (containerComponent) DefaultAttributes(*RenderContext) map[string]string { return nil }

func (containerComponent) AllowedChildren() []string { return []string{TextTagName} }

func (c containerComponent) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	return RenderChildren(ctx, w, n)
}

// TestRenderChildrenRendersNestedContainers asserts a container child that is
// itself a container renders its own children, so containers nest.
func TestRenderChildrenRendersNestedContainers(t *testing.T) {
	t.Parallel()

	ctx := contextWith(containerComponent{tag: "mj-panel"})
	inner := &node.Node{Type: "mj-panel", Children: []*node.Node{{Type: TextTagName, Content: "Inside"}}}
	outer := &node.Node{Type: "mj-notice", Children: []*node.Node{inner}}

	var output strings.Builder
	require.NoError(t, RenderChildren(ctx, &output, outer))

	assert.Equal(t, textHTML("Inside"), output.String())
}

// TestRenderChildrenRendersRawChildren asserts mj-raw children are emitted at
// their document position, like the built-in content containers do.
func TestRenderChildrenRendersRawChildren(t *testing.T) {
	t.Parallel()

	container := &node.Node{
		Type: "mj-notice",
		Children: []*node.Node{
			{Type: RawTagName, Content: "\n  <div class=\"raw\">raw</div>\n"},
		},
	}

	var output strings.Builder
	require.NoError(t, RenderChildren(&RenderContext{}, &output, container))

	assert.Equal(t, `<div class="raw">raw</div>`, output.String())
}

// TestRenderChildrenReportsAnUnrenderableChild asserts a child no component can
// render is a broken invariant, reported instead of silently dropped.
// Registration rejects such a declared child before any document renders (see
// componentTable), so this is defensive.
func TestRenderChildrenReportsAnUnrenderableChild(t *testing.T) {
	t.Parallel()

	container := &node.Node{Type: "mj-notice", Children: []*node.Node{{Type: "mj-unknown"}}}

	var output strings.Builder
	err := RenderChildren(&RenderContext{}, &output, container)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no component for child <mj-unknown> in <mj-notice>")
}

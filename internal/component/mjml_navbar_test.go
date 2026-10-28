package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/internal/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMJMLNavbar(t *testing.T) {
	t.Parallel()

	t.Run("renders hamburger links and component style", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{Breakpoint: "480px", MJMLStylesheet: make(map[string][]string), IDGenerator: testIDGenerator()}

		navbar := &node.Node{
			Type: NavbarTagName,
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "hamburger"}, Value: "hamburger"},
				{Name: xml.Name{Local: "base-url"}, Value: "https://example.com"},
				{Name: xml.Name{Local: "align"}, Value: "left"},
				{Name: xml.Name{Local: "ico-open"}, Value: "OPEN"},
			},
		}
		link := &node.Node{
			Type:   NavbarLinkTagName,
			Parent: navbar,
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "href"}, Value: "/docs"},
			},
			Content: "Docs",
		}
		navbar.Children = []*node.Node{link}

		var component MJMLNavbar
		require.NoError(t, InitComponent(ctx, component, navbar))
		require.NoError(t, InitComponent(ctx, MJMLNavbarLink{}, link))

		var out strings.Builder
		require.NoError(t, component.Render(ctx, &out, navbar))
		html := out.String()

		assert.Contains(t, html, `id="0000000000000000"`)
		assert.Contains(t, html, `for="0000000000000000"`)
		assert.Contains(t, html, `class="mj-menu-trigger"`)
		assert.Contains(t, html, `align="left"`)
		assert.Contains(t, html, `>OPEN</span>`)
		assert.Contains(t, html, `<div class="mj-inline-links" style="">`)
		assert.Contains(t, html, `<table role="presentation" border="0" cellpadding="0" cellspacing="0" align="left"><tr>`)
		assert.Contains(t, html, `href="https://example.com/docs"`)
		assert.Contains(t, html, `class="mj-link"`)
		assert.Contains(t, html, `>Docs</a>`)

		require.Len(t, ctx.ComponentHeadStyles, 1)
		assert.Equal(t, "mj-navbar", ctx.ComponentHeadStyles[0].Name)
		style := ctx.ComponentHeadStyles[0].Style("480px")
		assert.Contains(t, style, "noinput.mj-menu-checkbox")
		assert.Contains(t, style, "max-width:479px")
	})

	t.Run("registers its head style once for multiple navbars", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{Breakpoint: "480px", MJMLStylesheet: make(map[string][]string)}

		for range 2 {
			navbar := &node.Node{Type: NavbarTagName}
			var component MJMLNavbar
			require.NoError(t, InitComponent(ctx, component, navbar))
			require.NoError(t, component.Render(ctx, &strings.Builder{}, navbar))
		}

		require.Len(t, ctx.ComponentHeadStyles, 1)
	})
}

func TestMJMLNavbarLinkCssClass(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{Breakpoint: "480px", MJMLStylesheet: make(map[string][]string)}
	navbar := &node.Node{Type: NavbarTagName}
	link := &node.Node{
		Type:   NavbarLinkTagName,
		Parent: navbar,
		Attributes: []xml.Attr{
			{Name: xml.Name{Local: "href"}, Value: "/x"},
			{Name: xml.Name{Local: "css-class"}, Value: "primary accent"},
		},
		Content: "Link",
	}
	navbar.Children = []*node.Node{link}

	var component MJMLNavbarLink
	require.NoError(t, InitComponent(ctx, component, link))

	var out strings.Builder
	require.NoError(t, component.Render(ctx, &out, link))

	assert.Contains(t, out.String(), `class="mj-link primary accent"`)
	assert.Contains(t, out.String(), `class="primary-outlook accent-outlook"`)
}

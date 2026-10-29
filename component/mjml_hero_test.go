package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/v2/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLHeroOutlookTable(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{ContainerWidth: "600px"}
	n := &node.Node{
		Attributes: []xml.Attr{
			{Name: xml.Name{Local: "mode"}, Value: "fluid-height"},
			{Name: xml.Name{Local: "background-width"}, Value: "600px"},
			{Name: xml.Name{Local: "background-height"}, Value: "469px"},
			{Name: xml.Name{Local: "background-url"}, Value: "https://example.com/bg.jpg"},
		},
	}

	var hero MJMLHero
	require.NoError(t, InitComponent(ctx, hero, n))

	b := strings.Builder{}
	require.NoError(t, hero.Render(ctx, &b, n))

	out := b.String()
	require.Contains(t, out, `<!--[if mso | IE]>`)
	require.Contains(t, out, `cellspacing="0"`)
}

// renderHeroWithChild renders a fluid-height hero whose only child is child and
// returns the rendered output.
func renderHeroWithChild(t *testing.T, child *node.Node) string {
	t.Helper()

	hero := &node.Node{
		Type: HeroTagName,
		Attributes: []xml.Attr{
			{Name: xml.Name{Local: "mode"}, Value: "fluid-height"},
			{Name: xml.Name{Local: "background-width"}, Value: "600px"},
			{Name: xml.Name{Local: "background-height"}, Value: "469px"},
		},
	}
	child.Parent = hero
	hero.Children = []*node.Node{child}

	ctx := &RenderContext{ContainerWidth: "600px"}

	var component MJMLHero
	require.NoError(t, InitComponent(ctx, component, hero))

	b := strings.Builder{}
	require.NoError(t, component.Render(ctx, &b, hero))

	return b.String()
}

// TestMJMLHeroRawChild locks down where a raw child lands: hero children are
// wrapped in a row/cell, but raw content must be emitted bare at its document
// position - and, because that position is directly inside a <tbody>, wrapped in
// the raw-block markers the CSS inliner needs to keep it there.
func TestMJMLHeroRawChild(t *testing.T) {
	t.Parallel()

	t.Run("raw content is bare and marked", func(t *testing.T) {
		t.Parallel()

		raw := &node.Node{Type: RawTagName, Content: `<div class="raw">raw child</div>`}

		out := renderHeroWithChild(t, raw)

		// The markers and the content follow the content <tbody> directly: a
		// content child would sit one <tr><td> further in.
		require.Contains(t, out, "<tbody>\n"+RawBlockStartMarker+`<div class="raw">raw child</div>`+RawBlockEndMarker)
	})

	t.Run("content child keeps the row and cell and is not marked", func(t *testing.T) {
		t.Parallel()

		text := &node.Node{Type: TextTagName, Content: "hero text"}

		out := renderHeroWithChild(t, text)

		require.NotContains(t, out, RawBlockStartMarker)
		require.NotContains(t, out, RawBlockEndMarker)
		require.Contains(t, out, "<tr>\n<td ")
	})
}

package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
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

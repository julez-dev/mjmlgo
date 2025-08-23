package component

import (
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLDividerRender(t *testing.T) {
	t.Parallel()
	ctx := &RenderContext{
		ContainerWidth: "600px",
	}
	n := &node.Node{}

	var divider MJMLDivider
	err := InitComponent(ctx, divider, n)
	require.NoError(t, err)

	b := strings.Builder{}
	err = divider.Render(ctx, &b, n)
	require.NoError(t, err)

	want := `<p style="border-top:solid 4px #000000;margin:0px auto;width:100%;font-size:1px;"></p><!--[if mso | IE]><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-top:solid 4px #000000;margin:0px auto;width:550px;font-size:1px;" width="550px"><tr><td style="height:0;line-height:0;">&nbsp;</td></tr></table><![endif]-->`
	require.Equal(t, want, b.String())
}

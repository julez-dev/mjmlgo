package component

import (
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLSpacerRender(t *testing.T) {
	t.Parallel()
	ctx := &RenderContext{}
	n := &node.Node{}

	var spacer MJMLSpacer
	err := InitComponent(ctx, spacer, n)
	require.NoError(t, err)

	b := strings.Builder{}
	err = spacer.Render(ctx, &b, n)
	require.NoError(t, err)

	require.Equal(t, `<div style="height:20px;line-height:20px;">&#8202;</div>`, b.String())
}

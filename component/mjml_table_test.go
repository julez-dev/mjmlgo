package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLTable(t *testing.T) {
	t.Parallel()

	t.Run("width-px", func(t *testing.T) {
		ctx := &RenderContext{}
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "width"}, Value: "200px"},
			},
			Content: "<tr></tr>",
		}

		var table MJMLTable
		err := InitComponent(ctx, table, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = table.Render(ctx, &b, n)
		require.NoError(t, err)

		require.Equal(t, "<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" style=\"color:#000000;font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:22px;table-layout:auto;width:200px;border:none;\" width=\"200\">\n<tr></tr></table>", b.String())
	})

	t.Run("width-percentage", func(t *testing.T) {
		ctx := &RenderContext{}
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "width"}, Value: "80%"},
			},
			Content: "<tr></tr>",
		}

		var table MJMLTable
		err := InitComponent(ctx, table, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = table.Render(ctx, &b, n)
		require.NoError(t, err)

		require.Equal(t, "<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" style=\"color:#000000;font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:22px;table-layout:auto;width:80%;border:none;\" width=\"80%\">\n<tr></tr></table>", b.String())
	})
}

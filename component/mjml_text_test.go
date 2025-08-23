package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLText(t *testing.T) {
	t.Parallel()

	t.Run("no-height", func(t *testing.T) {
		ctx := &RenderContext{}
		n := &node.Node{
			Content: "My Text",
		}

		var text MJMLText
		err := InitComponent(ctx, text, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = text.Render(ctx, &b, n)
		require.NoError(t, err)

		require.Equal(t, "<div style=\"font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:1;color:#000000;\">My Text</div>\n", b.String())
	})

	t.Run("with-height", func(t *testing.T) {
		ctx := &RenderContext{}
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "height"}, Value: "100px"},
			},
			Content: "My Text",
		}

		var text MJMLText
		err := InitComponent(ctx, text, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = text.Render(ctx, &b, n)
		require.NoError(t, err)

		require.Equal(t, "<!--[if mso | IE]><table role=\"presentation\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><tr><td height=\"100px\" style=\"vertical-align:top;height:100px;\"><![endif]--><div style=\"font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:1;color:#000000;height:100px;\">My Text</div>\n<!--[if mso | IE]></td></tr></table><![endif]-->", b.String())
	})
}

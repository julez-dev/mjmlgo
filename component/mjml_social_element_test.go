package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLSocialElement(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{}
	n := &node.Node{
		Attributes: []xml.Attr{
			{Name: xml.Name{Local: "name"}, Value: "facebook"},
			{Name: xml.Name{Local: "href"}, Value: "https://mjml.io/"},
			{Name: xml.Name{Local: "icon-size"}, Value: "20px"},
		},
		Content: "Facebook",
	}

	var socialElement MJMLSocialElement
	err := InitComponent(ctx, socialElement, n)
	require.NoError(t, err)

	b := strings.Builder{}
	err = socialElement.Render(ctx, &b, n)
	require.NoError(t, err)

	require.Equal(t, "<tr >\n<td style=\"padding:4px;vertical-align:middle;\"><table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"background:#3b5998;border-radius:3px;width:20px;\"><tbody><tr>\n<td style=\"font-size:0;vertical-align:middle;width:20px;height:20px;\"><a href=\"https://www.facebook.com/sharer/sharer.php?u=https://mjml.io/\" target=\"_blank\"><img height=\"20\" src=\"https://www.mailjet.com/images/theme/v1/icons/ico-social/facebook.png\" style=\"border-radius:3px;display:block;\" width=\"20\" /></a></tr></tbody></table></td>\n <td style=\"vertical-align:middle;padding:4px 4px 4px 0;\"><a href=\"https://www.facebook.com/sharer/sharer.php?u=https://mjml.io/\" style=\"color:#000000;font-size:13px;font-family:Ubuntu, Helvetica, Arial, sans-serif;line-height:1;text-decoration:none;\" target=\"_blank\">Facebook</a></td>\n</tr>\n", b.String())

}

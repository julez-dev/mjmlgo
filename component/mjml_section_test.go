package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLSection(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		parent := &node.Node{
			Type: "mj-body",
		}

		n := &node.Node{
			Type:   "mj-section",
			Parent: parent,
		}
		parent.Children = append(parent.Children, n)
		n.Children = append(n.Children, &node.Node{
			Type:   "mj-section",
			Parent: n,
		})

		var section MJMLSection
		err := InitComponent(ctx, section, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = section.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<!--[if mso | IE]><table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:600px;\" width=\"600\">\n<tr>\n<td style=\"line-height:0px;font-size:0px;mso-line-height-rule:exactly;\">\n<![endif]-->\n<div style=\"margin:0px auto;max-width:600px;\">\n<table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:100%;\">\n<tbody>\n<tr>\n<td style=\"direction:ltr;font-size:0px;padding:20px 0;text-align:center;\">\n<!--[if mso | IE]><table role=\"presentation\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><![endif]-->\n<!--[if mso | IE]><tr><![endif]-->\n<!--[if mso | IE]></tr><![endif]-->\n<!--[if mso | IE]></table><![endif]-->\n</td>\n</tr>\n</tbody>\n</table>\n</div>\n<!--[if mso | IE]></td></tr></table><![endif]-->"
		require.Equal(t, want, b.String())
	})

	t.Run("full-width", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		parent := &node.Node{
			Type: "mj-body",
		}

		n := &node.Node{
			Type:   "mj-section",
			Parent: parent,
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "full-width"}, Value: "full-width"},
			},
		}
		parent.Children = append(parent.Children, n)
		n.Children = append(n.Children, &node.Node{
			Type:   "mj-section",
			Parent: n,
		})

		var section MJMLSection
		err := InitComponent(ctx, section, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = section.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:100%;\">\n<tbody>\n<tr>\n<td><!--[if mso | IE]><table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:600px;\" width=\"600\">\n<tr>\n<td style=\"line-height:0px;font-size:0px;mso-line-height-rule:exactly;\">\n<![endif]-->\n<div style=\"margin:0px auto;max-width:600px;\">\n<table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:100%;\">\n<tbody>\n<tr>\n<td style=\"direction:ltr;font-size:0px;padding:20px 0;text-align:center;\">\n<!--[if mso | IE]><table role=\"presentation\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><![endif]-->\n<!--[if mso | IE]><tr><![endif]-->\n<!--[if mso | IE]></tr><![endif]-->\n<!--[if mso | IE]></table><![endif]-->\n</td>\n</tr>\n</tbody>\n</table>\n</div>\n<!--[if mso | IE]></td></tr></table><![endif]--></td>\n</tr>\n</tbody>\n</table>\n"
		require.Equal(t, want, b.String())
	})

	t.Run("is-wrapper", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		parent := &node.Node{
			Type: "mj-body",
		}

		n := &node.Node{
			Type:       "mj-section",
			Parent:     parent,
			Attributes: []xml.Attr{},
		}
		parent.Children = append(parent.Children, n)
		n.Children = append(n.Children, &node.Node{
			Type:   "mj-section",
			Parent: n,
		})

		var section MJMLSection
		section.IsWrapper = true
		err := InitComponent(ctx, section, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = section.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<!--[if mso | IE]><table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:600px;\" width=\"600\">\n<tr>\n<td style=\"line-height:0px;font-size:0px;mso-line-height-rule:exactly;\">\n<![endif]-->\n<div style=\"margin:0px auto;max-width:600px;\">\n<table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:100%;\">\n<tbody>\n<tr>\n<td style=\"direction:ltr;font-size:0px;padding:20px 0;text-align:center;\">\n<!--[if mso | IE]><table role=\"presentation\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><![endif]-->\n<!--[if mso | IE]><tr><td width=\"600px\"><![endif]--><!--[if mso | IE]><table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:600px;\" width=\"600\">\n<tr>\n<td style=\"line-height:0px;font-size:0px;mso-line-height-rule:exactly;\">\n<![endif]-->\n<div style=\"margin:0px auto;max-width:600px;\">\n<table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:100%;\">\n<tbody>\n<tr>\n<td style=\"direction:ltr;font-size:0px;padding:20px 0;text-align:center;\">\n<!--[if mso | IE]><table role=\"presentation\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><![endif]-->\n<!--[if mso | IE]><tr><![endif]-->\n<!--[if mso | IE]></tr><![endif]-->\n<!--[if mso | IE]></table><![endif]-->\n</td>\n</tr>\n</tbody>\n</table>\n</div>\n<!--[if mso | IE]></td></tr></table><![endif]--><!--[if mso | IE]></td></tr><![endif]--><!--[if mso | IE]></table><![endif]-->\n</td>\n</tr>\n</tbody>\n</table>\n</div>\n<!--[if mso | IE]></td></tr></table><![endif]-->"
		require.Equal(t, want, b.String())
	})
}

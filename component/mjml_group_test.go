package component

import (
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLGroup(t *testing.T) {
	t.Parallel()

	t.Run("width-px", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		parent := &node.Node{
			Type: "mj-section",
		}

		n := &node.Node{
			Type:   "mj-section",
			Parent: parent,
		}
		parent.Children = append(parent.Children, n)
		n.Children = append(n.Children, &node.Node{
			Type:   "mj-column",
			Parent: n,
		})
		n.Children = append(n.Children, &node.Node{
			Type:   "mj-column",
			Parent: n,
		})

		var group MJMLGroup
		err := InitComponent(ctx, group, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = group.Render(ctx, &b, n)
		require.NoError(t, err)

		require.Equal(t, "<div class=\"mj-column-per-100 mj-outlook-group-fix\" style=\"font-size:0;line-height:0;text-align:left;display:inline-block;width:100%;\">\n<!--[if mso | IE]><table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\"><tr><![endif]--><!--[if mso | IE]><td style=\"vertical-align:top;width:300px;\"><![endif]-->\n<div class=\"mj-column-per-50 mj-outlook-group-fix\" style=\"font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:50%;\">\n<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"vertical-align:top;\" width=\"100%\">\n<tbody>\n</tbody>\n</table>\n</div>\n\n<!--[if mso | IE]></td><![endif]--><!--[if mso | IE]><td style=\"vertical-align:top;width:300px;\"><![endif]-->\n<div class=\"mj-column-per-50 mj-outlook-group-fix\" style=\"font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:50%;\">\n<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"vertical-align:top;\" width=\"100%\">\n<tbody>\n</tbody>\n</table>\n</div>\n\n<!--[if mso | IE]></td><![endif]--><!--[if mso | IE]></tr></table><![endif]-->\n</div>", b.String())
	})
}

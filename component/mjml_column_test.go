package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLColumnRender(t *testing.T) {
	t.Parallel()

	t.Run("no-gutter", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		p := &node.Node{}
		n := &node.Node{
			Parent: p,
		}
		p.Children = append(p.Children, n)
		p.Children = append(p.Children, &node.Node{
			Parent: n,
		})

		var column MJMLColumn
		err := InitComponent(ctx, column, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = column.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<div class=\"mj-column-per-50 mj-outlook-group-fix\" style=\"font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;\">\n<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"vertical-align:top;\" width=\"100%\">\n<tbody>\n</tbody>\n</table>\n</div>\n"
		require.Equal(t, want, b.String())
	})

	t.Run("with-gutter", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		p := &node.Node{}
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "padding"}, Value: "11px"},
			},
			Parent: p,
		}
		p.Children = append(p.Children, n)
		p.Children = append(p.Children, &node.Node{
			Parent: n,
		})

		var column MJMLColumn
		err := InitComponent(ctx, column, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = column.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<div class=\"mj-column-per-50 mj-outlook-group-fix\" style=\"font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;\">\n<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" width=\"100%\">\n<tbody>\n<tr>\n<td style=\"padding:11px;vertical-align:top;\">\n<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" width=\"100%\">\n<tbody>\n</tbody>\n</table>\n</td>\n</tr>\n</tbody>\n</table>\n</div>\n"
		require.Equal(t, want, b.String())
	})

	t.Run("no-gutter-mobile-width", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		p := &node.Node{}
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "mobileWidth"}, Value: "mobileWidth"},
			},
			Parent: p,
		}
		p.Children = append(p.Children, n)
		p.Children = append(p.Children, &node.Node{
			Parent: n,
		})

		var column MJMLColumn
		err := InitComponent(ctx, column, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = column.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<div class=\"mj-column-per-50 mj-outlook-group-fix\" style=\"font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:50%;\">\n<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"vertical-align:top;\" width=\"100%\">\n<tbody>\n</tbody>\n</table>\n</div>\n"
		require.Equal(t, want, b.String())
	})
}

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

	t.Run("raw-between-children", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		parent := &node.Node{}
		n := &node.Node{Parent: parent}
		parent.Children = append(parent.Children, n)

		n.Children = append(n.Children,
			&node.Node{Type: TextTagName, Parent: n, Content: "first text"},
			&node.Node{Type: RawTagName, Parent: n, Content: `<div class="raw-between">raw markup</div>`},
			&node.Node{Type: TextTagName, Parent: n, Content: "second text"},
		)

		var column MJMLColumn
		err := InitComponent(ctx, column, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = column.Render(ctx, &b, n)
		require.NoError(t, err)

		out := b.String()
		tbody := strings.Index(out, "<tbody>")
		require.NotEqual(t, -1, tbody, "missing <tbody>")

		firstTR := strings.Index(out[tbody:], "<tr>")
		require.NotEqual(t, -1, firstTR, "missing first row")
		firstTR += tbody

		firstClose := strings.Index(out[firstTR:], "</tr>")
		require.NotEqual(t, -1, firstClose, "missing first row close")
		firstClose += firstTR

		raw := strings.Index(out, `<div class="raw-between">raw markup</div>`)
		require.NotEqual(t, -1, raw, "missing raw markup")

		secondTR := strings.Index(out[firstClose:], "<tr>")
		require.NotEqual(t, -1, secondTR, "missing second row")
		secondTR += firstClose

		require.Less(t, tbody, firstTR, "first <tr> must follow <tbody>")
		require.Less(t, firstClose, raw, "raw markup must follow the first row")
		require.Less(t, raw, secondTR, "raw markup must precede the second row")
	})
}

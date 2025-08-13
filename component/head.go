package component

import (
	"fmt"
	"io"

	"github.com/julez-dev/mjmlgo/node"
)

const defaultHeadStyle = `<style type="text/css">
  #outlook a {
    padding: 0;
  }

  body {
    margin: 0;
    padding: 0;
    -webkit-text-size-adjust: 100%;
    -ms-text-size-adjust: 100%;
  }

  table,
  td {
    border-collapse: collapse;
    mso-table-lspace: 0pt;
    mso-table-rspace: 0pt;
  }

  img {
    border: 0;
    height: auto;
    line-height: 100%;
    outline: none;
    text-decoration: none;
    -ms-interpolation-mode: bicubic;
  }

  p {
    display: block;
    margin: 13px 0;
  }
</style>
<!--[if mso]>
      <noscript>
      <xml>
      <o:OfficeDocumentSettings>
        <o:AllowPNG/>
        <o:PixelsPerInch>96</o:PixelsPerInch>
      </o:OfficeDocumentSettings>
      </xml>
      </noscript>
      <![endif]-->
<!--[if lte mso 11]>
      <style type="text/css">
        .mj-outlook-group-fix { width:100% !important; }
      </style>
      <![endif]-->
`

type Head struct{}

func (h Head) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	_, _ = io.WriteString(w, `<head>`)
	for _, child := range n.Children {
		switch child.Type {
		case AttibutesTagName:
			var attr Attributes
			if err := attr.Render(ctx, w, child); err != nil {
				return err
			}
		case PreviewTagName:
			ctx.PreviewText = child.Content
		case TitleTagName:
			_, _ = io.WriteString(w, fmt.Sprintf("<title>%s</title>", child.Content))
		}
	}

	_, _ = io.WriteString(w, `<!--[if !mso]><!-->
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
<!--<![endif]-->`)

	_, _ = io.WriteString(w, `<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">`)
	_, _ = io.WriteString(w, `<meta name="viewport" content="width=device-width, initial-scale=1">`)
	_, _ = io.WriteString(w, defaultHeadStyle)

	_, _ = io.WriteString(w, `</head>`)

	return nil
}

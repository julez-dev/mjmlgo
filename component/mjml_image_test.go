package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLImage(t *testing.T) {
	t.Parallel()

	t.Run("width-px", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
		}
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "width"}, Value: "300px"},
				{Name: xml.Name{Local: "src"}, Value: "https://www.online-image-editor.com//styles/2014/images/example_image.png"},
			},
		}

		var image MJMLImage
		err := InitComponent(ctx, image, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = image.Render(ctx, &b, n)
		require.NoError(t, err)

		require.Equal(t, "<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"border-collapse:collapse;border-spacing:0px;\">\n<tbody>\n<tr>\n<td style=\"width:300px;\">\n<img height=\"auto\" src=\"https://www.online-image-editor.com//styles/2014/images/example_image.png\" style=\"border:0;display:block;outline:none;text-decoration:none;height:auto;width:100%;font-size:13px;\" width=\"300\" /></td>\n</tr>\n</tbody>\n</table>\n", b.String())
	})

	t.Run("fluid-on-mobile", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
		}

		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "width"}, Value: "300px"},
				{Name: xml.Name{Local: "height"}, Value: "250px"},
				{Name: xml.Name{Local: "fluid-on-mobile"}, Value: "true"},
				{Name: xml.Name{Local: "src"}, Value: "https://www.online-image-editor.com//styles/2014/images/example_image.png"},
			},
		}

		var image MJMLImage
		err := InitComponent(ctx, image, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = image.Render(ctx, &b, n)
		require.NoError(t, err)

		require.Equal(t, "<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" class=\"mj-full-width-mobile\" role=\"presentation\" style=\"border-collapse:collapse;border-spacing:0px;\">\n<tbody>\n<tr>\n<td class=\"mj-full-width-mobile\" style=\"width:300px;\">\n<img height=\"250\" src=\"https://www.online-image-editor.com//styles/2014/images/example_image.png\" style=\"border:0;display:block;outline:none;text-decoration:none;height:250px;width:100%;font-size:13px;\" width=\"300\" /></td>\n</tr>\n</tbody>\n</table>\n", b.String())
	})
}

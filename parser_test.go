package mjmlgo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("mj-raw", func(t *testing.T) {
		const input = `<mjml>
			<mj-body>
				<mj-raw>Raw </div></mj-raw>
			</mj-body>
		</mjml>
		`

		n, err := parse(strings.NewReader(input))
		require.NoError(t, err)

		var rawContent string
		for _, v := range n.Children {
			if v.Type == "mj-body" {
				for _, v2 := range v.Children {
					if v2.Type == "mj-raw" {
						rawContent = v2.Content
					}
				}
			}
		}

		require.Equal(t, "Raw </div>", rawContent)
	})

	t.Run("mj-end-tags", func(t *testing.T) {
		const input = `<mjml><mj-text><h1>Test</h1></mj-text></mjml>`

		n, err := parse(strings.NewReader(input))
		require.NoError(t, err)

		var rawContent string
		for _, v := range n.Children {
			if v.Type == "mj-text" {
				rawContent = v.Content
			}
		}

		require.Equal(t, "<h1>Test</h1>", rawContent)
	})
}

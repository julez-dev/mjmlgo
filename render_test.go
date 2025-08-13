package mjmlgo

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	input, err := os.ReadFile("input_s.mjml")
	require.NoError(t, err)

	out, err := RenderMJML(bytes.NewReader(input))
	require.NoError(t, err)
	t.Log(out)
}

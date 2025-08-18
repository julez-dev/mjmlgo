package mjmlgo

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	input, err := os.ReadFile("input_big.mjml")
	require.NoError(t, err)

	out, err := RenderMJML(bytes.NewReader(input))
	require.NoError(t, err)

	os.WriteFile("out.html", []byte(out), 0644)
	_ = out
	//t.Log(out)
}

package mjmlgo

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	input, err := os.ReadFile("input_s.mjml")
	require.NoError(t, err)

	node, err := parse(bytes.NewReader(input))
	require.NoError(t, err)

	_ = node
	// spew.Dump(node)
}

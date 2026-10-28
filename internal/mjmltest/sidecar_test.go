package mjmltest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestArgs locks down the sidecar format: one reference CLI argument per line,
// blank lines and comments ignored, surrounding whitespace trimmed.
func TestArgs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fixture := filepath.Join(dir, "fixture.mjml")

	require.NoError(t, os.WriteFile(fixture, []byte("<mjml />"), 0o644))
	require.NoError(t, os.WriteFile(fixture+SidecarSuffix, []byte(
		"# reference flags for this fixture\n"+
			"\n"+
			"  -c.keepComments=false  \n"+
			"-c.printerSupport=true\n",
	), 0o644))

	assert.Equal(t, []string{"-c.keepComments=false", "-c.printerSupport=true"}, Args(t, fixture))

	// A fixture without a sidecar is not an error: it uses the reference
	// defaults.
	assert.Nil(t, Args(t, filepath.Join(dir, "other.mjml")))
}

// TestParseConfig locks down the flag-to-option mapping, including the
// reference CLI's uneven boolean handling.
func TestParseConfig(t *testing.T) {
	t.Parallel()

	t.Run("keepComments-honours-only-the-literal-false", func(t *testing.T) {
		t.Parallel()

		config, err := ParseConfig("f.mjml", []string{"-c.keepComments=false"})
		require.NoError(t, err)
		require.NotNil(t, config.KeepComments)
		assert.False(t, *config.KeepComments)

		config, err = ParseConfig("f.mjml", []string{"-c.keepComments=true"})
		require.NoError(t, err)
		require.NotNil(t, config.KeepComments)
		assert.True(t, *config.KeepComments)
	})

	t.Run("raw-string-options-are-truthy", func(t *testing.T) {
		t.Parallel()

		// mjml-core receives the string from the CLI, so "false" enables the
		// option in the reference implementation. Sidecar fixtures must not
		// pretend otherwise.
		config, err := ParseConfig("f.mjml", []string{"-c.printerSupport=false", "-c.ignoreIncludes=0"})
		require.NoError(t, err)
		require.NotNil(t, config.PrinterSupport)
		require.NotNil(t, config.IgnoreIncludes)
		assert.True(t, *config.PrinterSupport)
		assert.True(t, *config.IgnoreIncludes)
	})

	t.Run("long-spelling", func(t *testing.T) {
		t.Parallel()

		config, err := ParseConfig("f.mjml", []string{"--config.printerSupport=true"})
		require.NoError(t, err)
		require.NotNil(t, config.PrinterSupport)
		assert.True(t, *config.PrinterSupport)
	})

	t.Run("fonts", func(t *testing.T) {
		t.Parallel()

		config, err := ParseConfig("f.mjml", []string{`-c.fonts={"A":"https://example.com/a.css"}`})
		require.NoError(t, err)
		assert.True(t, config.FontsSet)
		assert.Equal(t, map[string]string{"A": "https://example.com/a.css"}, config.Fonts)

		// An empty object replaces the defaults with nothing, which is not the
		// same as leaving the flag out.
		config, err = ParseConfig("f.mjml", []string{"-c.fonts={}"})
		require.NoError(t, err)
		assert.True(t, config.FontsSet)
		assert.Empty(t, config.Fonts)
	})

	t.Run("malformed", func(t *testing.T) {
		t.Parallel()

		for _, args := range [][]string{
			{"-c.unknown=true"},
			{"-c.keepComments"},
			{"--config.=true"},
			{"keepComments=false"},
			{"-c.fonts={not json}"},
			// Anything but a JSON object reaches mjml-core as a string, where
			// the registry is unusable (per-character links, or a throw on
			// <mj-font>). Omit the flag for the defaults.
			{"-c.fonts=null"},
			{"-c.fonts="},
			{"-c.fonts=[]"},
			{"-c.fonts=false"},
		} {
			_, err := ParseConfig("f.mjml", args)
			require.Error(t, err, "args %v must be rejected", args)
		}
	})
}

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConfigAcceptsKnownKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfg, err := parseConfig([]configPair{
		{key: "fonts", value: `{"Comic Sans MS":"https://example.test/font"}`},
		{key: "keepComments", value: "false"},
		{key: "filePath", value: dir},
	})
	require.NoError(t, err)
	require.Len(t, cfg.options, 2)
	require.Equal(t, dir, cfg.root)
}

func TestParseConfigRejectsUnknownKeys(t *testing.T) {
	t.Parallel()

	for _, pair := range []configPair{
		{key: "beautify", value: "true"},
		{key: "minify", value: "true"},
		{key: "minifyOptions", value: "{}"},
		{key: "juiceOptions", value: "{}"},
	} {
		_, err := parseConfig([]configPair{pair})
		require.Error(t, err, "key %s", pair.key)
	}

	_, err := parseConfig([]configPair{{key: "frobnicate", value: "1"}})
	require.ErrorContains(t, err, "unknown option")
}

func TestParseConfigValidatesValues(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))

	_, err := parseConfig([]configPair{{key: "fonts", value: "not json"}})
	require.ErrorContains(t, err, "config.fonts")

	_, err = parseConfig([]configPair{{key: "keepComments", value: "maybe"}})
	require.ErrorContains(t, err, "config.keepComments")

	_, err = parseConfig([]configPair{{key: "filePath", value: filepath.Join(file, "missing")}})
	require.ErrorContains(t, err, "config.filePath")

	_, err = parseConfig([]configPair{{key: "filePath", value: file}})
	require.ErrorContains(t, err, "config.filePath")
}

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo"
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

func TestParseConfigKeepsFontOrder(t *testing.T) {
	t.Parallel()

	cfg, err := parseConfig([]configPair{{key: "fonts", value: `{"Zeta":"https://z.test/z","Alpha":"https://a.test/a"}`}})
	require.NoError(t, err)
	require.Len(t, cfg.options, 1)

	// The object's text order reaches WithFonts, which emits in that order.
	var out bytes.Buffer

	document := []byte(`<mjml><mj-body><mj-section><mj-column>` +
		`<mj-text font-family="Alpha">a</mj-text>` +
		`<mj-text font-family="Zeta">z</mj-text>` +
		`</mj-column></mj-section></mj-body></mjml>`)

	require.NoError(t, mjmlgo.NewRenderer(cfg.options...).Render(context.Background(), &out, document))

	html := out.String()
	zeta := strings.Index(html, "https://z.test/z")
	alpha := strings.Index(html, "https://a.test/a")

	require.Positive(t, zeta, "the registry is used")
	require.Positive(t, alpha)
	require.Less(t, zeta, alpha, "the JSON order decides, not the alphabet")
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

	for _, value := range []string{"null", "[]", `"family"`, `{"a":"b"} trailing`, `{"a":1}`} {
		_, err := parseConfig([]configPair{{key: "fonts", value: value}})
		require.Error(t, err, "value %s", value)
	}

	_, err = parseConfig([]configPair{{key: "keepComments", value: "maybe"}})
	require.ErrorContains(t, err, "config.keepComments")

	_, err = parseConfig([]configPair{{key: "filePath", value: filepath.Join(file, "missing")}})
	require.ErrorContains(t, err, "config.filePath")

	_, err = parseConfig([]configPair{{key: "filePath", value: file}})
	require.ErrorContains(t, err, "config.filePath")
}

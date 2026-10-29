package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeFile writes a file with content, creating the directory above it.
func writeFile(t *testing.T, path, content string) string {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestExpandInputs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	first := writeFile(t, filepath.Join(dir, "b.mjml"), "")
	second := writeFile(t, filepath.Join(dir, "a.mjml"), "")
	nested := writeFile(t, filepath.Join(dir, "sub", "c.mjml"), "")

	// A literal path is kept even if it does not exist: reading settles that,
	// and the error should name the file the user asked for.
	files, err := expandInputs([]string{filepath.Join(dir, "missing.mjml")})
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(dir, "missing.mjml")}, files)

	files, err = expandInputs([]string{filepath.Join(dir, "*.mjml")})
	require.NoError(t, err)
	require.Equal(t, []string{second, first}, files, "glob matches are sorted")

	files, err = expandInputs([]string{filepath.Join(dir, "**", "*.mjml")})
	require.NoError(t, err)
	require.Equal(t, []string{second, first, nested}, files, "** matches any number of directories")

	files, err = expandInputs([]string{filepath.Join(dir, "nothing-*.mjml")})
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestMatchSegments(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		pattern, path string
		want          bool
	}{
		{"**/*.mjml", "a.mjml", true},
		{"**/*.mjml", "x/y/a.mjml", true},
		{"**/*.mjml", "x/y/a.html", false},
		{"a/**/b.mjml", "a/b.mjml", true},
		{"a/**/b.mjml", "a/x/y/b.mjml", true},
		{"a/**/b.mjml", "c/b.mjml", false},
		{"*.mjml", "a.mjml", true},
		{"*.mjml", "x/a.mjml", false},
	} {
		got := matchSegments(strings.Split(tc.pattern, "/"), strings.Split(tc.path, "/"))
		require.Equal(t, tc.want, got, "pattern %q path %q", tc.pattern, tc.path)
	}
}

func TestResolveOutputs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")
	require.NoError(t, os.Mkdir(outDir, 0o755))

	one := []input{{file: filepath.Join(dir, "a.mjml")}}
	many := []input{{file: filepath.Join(dir, "a.mjml")}, {file: filepath.Join(dir, "b.mjml")}}

	// A single input takes the path as given.
	targets, err := resolveOutputs(filepath.Join(dir, "named.html"), one)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(dir, "named.html")}, targets)

	// An existing directory collects one .html file per input.
	targets, err = resolveOutputs(outDir, one)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(outDir, "a.html")}, targets)

	targets, err = resolveOutputs(outDir, many)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(outDir, "a.html"), filepath.Join(outDir, "b.html")}, targets)

	// An empty output path drops the files in the working directory, which is
	// what the reference CLI does with -o ''.
	targets, err = resolveOutputs("", many)
	require.NoError(t, err)
	require.Equal(t, []string{"a.html", "b.html"}, targets)

	// Several inputs need a directory, and the CLI never creates one.
	_, err = resolveOutputs(filepath.Join(dir, "named.html"), many)
	require.ErrorContains(t, err, "Multiple input files")

	_, err = resolveOutputs(filepath.Join(dir, "missing", "a.html"), one)
	require.ErrorContains(t, err, "Output directory doesn't exist")

	_, err = resolveOutputs(filepath.Join(dir, "missing")+string(filepath.Separator), one)
	require.ErrorContains(t, err, "Output directory doesn't exist")

	// Standard input has no name to derive one from, so it needs a file.
	targets, err = resolveOutputs(filepath.Join(dir, "stdin.html"), []input{{}})
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(dir, "stdin.html")}, targets)

	_, err = resolveOutputs(outDir, []input{{}})
	require.ErrorContains(t, err, "must name a file")
}

func TestHTMLName(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ in, want string }{
		{"a.mjml", "a.html"},
		{"dir/a.mjml", "a.html"},
		{"a.MJML", "a.html"},
		{"a.html", "a.html.html"},
		{"no-extension", "no-extension.html"},
		{"a.b.mjml", "a.b.html"},
	} {
		require.Equal(t, tc.want, htmlName(tc.in), "input %q", tc.in)
	}
}

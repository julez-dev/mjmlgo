package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/julez-dev/mjmlgo/v2"
	"github.com/stretchr/testify/require"
)

// hero is a fixture without generated ids, so its rendering is deterministic.
const hero = "../../testdata/parity/simple-hero.mjml"

// document is a small document with an include, for the include tests.
const document = `<mjml><mj-body><mj-section><mj-column><mj-include path="partial.mjml" /></mj-column></mj-section></mj-body></mjml>`

// result is what one command line produced.
type result struct {
	code   int
	stdout string
	stderr string
}

// runCommand runs the command the way the process would, with stdin as its
// standard input.
func runCommand(t *testing.T, stdin string, args ...string) result {
	t.Helper()

	var stdout, stderr bytes.Buffer

	code := run(args, bytes.NewBufferString(stdin), &stdout, &stderr)

	return result{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

// renderHero is what the library emits for the fixture, which the command must
// reproduce and only wrap.
func renderHero(t *testing.T) string {
	t.Helper()

	var out bytes.Buffer
	renderer := mjmlgo.NewRenderer()

	require.NoError(t, renderer.RenderFile(context.Background(), &out, hero))

	return out.String()
}

func TestRunWritesStdout(t *testing.T) {
	t.Parallel()

	want := renderHero(t)

	got := runCommand(t, "", hero)
	require.Equal(t, 0, got.code)
	require.Equal(t, "<!-- FILE: "+hero+" -->\n"+want+"\n", got.stdout)
	require.Empty(t, got.stderr)

	got = runCommand(t, "", "--noStdoutFileComment", hero)
	require.Equal(t, 0, got.code)
	require.Equal(t, want+"\n", got.stdout)

	got = runCommand(t, "", "-s", hero)
	require.Equal(t, 0, got.code)
	require.Equal(t, want+"\n", got.stdout[len("<!-- FILE: "+hero+" -->\n"):])
}

func TestRunWritesOutputFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	output := filepath.Join(dir, "named.html")

	got := runCommand(t, "", hero, "-o", output)
	require.Equal(t, 0, got.code)
	require.Empty(t, got.stdout)

	written, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, renderHero(t), string(written), "a file gets the document without the stdout wrapper")

	// An existing directory collects <name>.html.
	got = runCommand(t, "", hero, "-o", dir)
	require.Equal(t, 0, got.code)

	written, err = os.ReadFile(filepath.Join(dir, "simple-hero.html"))
	require.NoError(t, err)
	require.Equal(t, renderHero(t), string(written))
}

func TestRunCompilesStandardInput(t *testing.T) {
	t.Parallel()

	want := renderHero(t)
	src, err := os.ReadFile(hero)
	require.NoError(t, err)

	got := runCommand(t, string(src), "-i")
	require.Equal(t, 0, got.code)
	require.Equal(t, want+"\n", got.stdout, "a document without a file name announces none")

	output := filepath.Join(t.TempDir(), "stdin.html")

	got = runCommand(t, string(src), "-i", "-o", output)
	require.Equal(t, 0, got.code)

	written, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, want, string(written))
}

func TestRunResolvesIncludes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "partial.mjml"), "<mj-text>included</mj-text>")
	doc := writeFile(t, filepath.Join(dir, "doc.mjml"), document)

	// Relative to the document itself.
	got := runCommand(t, "", doc)
	require.Equal(t, 0, got.code)
	require.Contains(t, got.stdout, "included")

	// Relative to -c filePath, for a document that lives elsewhere.
	elsewhere := writeFile(t, filepath.Join(t.TempDir(), "doc.mjml"), document)

	got = runCommand(t, "", elsewhere, "-c", "filePath="+dir)
	require.Equal(t, 0, got.code)
	require.Contains(t, got.stdout, "included")

	// And for a document read from standard input, where the working directory
	// has nothing to do with the partials.
	got = runCommand(t, document, "-i", "-c", "filePath="+dir)
	require.Equal(t, 0, got.code)
	require.Contains(t, got.stdout, "included")
}

func TestRunValidates(t *testing.T) {
	t.Parallel()

	got := runCommand(t, "", "-v", hero)
	require.Equal(t, 0, got.code)
	require.Empty(t, got.stdout)
	require.Empty(t, got.stderr)

	dir := t.TempDir()
	invalid := writeFile(t, filepath.Join(dir, "bad.mjml"), "<mjml><mj-body><mj-foo /></mj-body></mjml>")

	got = runCommand(t, "", "-v", invalid)
	require.Equal(t, 1, got.code)
	require.Empty(t, got.stdout)
	require.Contains(t, got.stderr, "File: "+invalid)
	require.Contains(t, got.stderr, "mj-foo")
	require.Contains(t, got.stderr, "Validation failed")
}

func TestRunExitCodes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	invalid := writeFile(t, filepath.Join(dir, "bad.mjml"), "<mjml><mj-body><mj-foo /></mj-body></mjml>")

	// Everything failed: the reference CLI calls that a command-line error.
	got := runCommand(t, "", invalid)
	require.Equal(t, 1, got.code)
	require.Contains(t, got.stderr, "Input file(s) failed to render")

	// Some failed: a render failure, and the documents that did work are still
	// written.
	got = runCommand(t, "", invalid, hero)
	require.Equal(t, 2, got.code)
	require.Contains(t, got.stdout, "<html")
	require.Contains(t, got.stderr, "File: "+invalid)

	// Usage problems.
	got = runCommand(t, "")
	require.Equal(t, 1, got.code)
	require.Contains(t, got.stderr, "No input argument received")

	got = runCommand(t, "", "-i", "-v", hero)
	require.Equal(t, 1, got.code)
	require.Contains(t, got.stderr, "Too many input arguments received")

	got = runCommand(t, "", "-s", "-o", "out.html", hero)
	require.Equal(t, 1, got.code)
	require.Contains(t, got.stderr, "Too many output arguments received")

	got = runCommand(t, "", "--frobnicate", hero)
	require.Equal(t, 1, got.code)
	require.Contains(t, got.stderr, "unknown option")

	got = runCommand(t, "", filepath.Join(dir, "nothing-*.mjml"))
	require.Equal(t, 1, got.code)
	require.Contains(t, got.stderr, "No input files found")

	got = runCommand(t, "", filepath.Join(dir, "missing.mjml"))
	require.Equal(t, 1, got.code)
	require.Contains(t, got.stderr, "cannot read file")
}

func TestRunJSONReports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	invalid := writeFile(t, filepath.Join(dir, "bad.mjml"), "<mjml><mj-body><mj-foo /></mj-body></mjml>")

	// Validation's report is the only output, so it goes to stdout.
	got := runCommand(t, "", "--json", "-v", hero, invalid)
	require.Equal(t, 1, got.code)

	lines := bytes.Split(bytes.TrimSuffix([]byte(got.stdout), []byte("\n")), []byte("\n"))
	require.Len(t, lines, 2)

	var first jsonRecord
	require.NoError(t, json.Unmarshal(lines[0], &first))
	require.True(t, first.OK)
	require.Equal(t, hero, first.File)

	var second jsonRecord
	require.NoError(t, json.Unmarshal(lines[1], &second))
	require.False(t, second.OK)
	require.Equal(t, "unknown-element", second.Error.Kind)

	// In compile mode HTML owns stdout, so the report goes to stderr.
	got = runCommand(t, "", "--json", invalid)
	require.Equal(t, 1, got.code)
	require.Empty(t, got.stdout)
	require.Contains(t, got.stderr, `"kind":"unknown-element"`)
}

func TestRunVersionAndHelp(t *testing.T) {
	t.Parallel()

	got := runCommand(t, "", "-V")
	require.Equal(t, 0, got.code)
	require.Contains(t, got.stdout, "mjmlgo ")

	got = runCommand(t, "", "--help")
	require.Equal(t, 0, got.code)
	require.Contains(t, got.stdout, "Usage: mjmlgo")
	require.Contains(t, got.stdout, "--noStdoutFileComment")
}

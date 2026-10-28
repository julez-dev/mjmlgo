package mjmlgo

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/component"
	"github.com/julez-dev/mjmlgo/internal/mjmltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baselinePath holds the recorded HTML divergences from the reference MJML
// implementation. It is version scoped: bumping MJML requires re-recording it.
const baselinePath = "testdata/parity-baseline.json"

// parityDir holds the fixtures that must render identically to MJML.
const parityDir = "testdata/parity"

// invalidDir holds fixtures that must be rejected with the error recorded in
// the sibling .error.txt file.
const invalidDir = "testdata/invalid"

var updateBaseline = flag.Bool("update", false, "re-record the MJML parity baseline in "+baselinePath)

// parityBaseline records the known differences between this implementation and
// the reference MJML implementation for a specific MJML version.
type parityBaseline struct {
	// MJMLVersion is the reference version the baseline was recorded against.
	MJMLVersion string `json:"mjmlVersion"`

	// Fixtures maps fixture paths to the divergences expected for them. Fixtures
	// that match the reference implementation are not listed.
	Fixtures map[string]fixtureDivergence `json:"fixtures"`
}

type fixtureDivergence struct {
	// Error is the expected render error, if the fixture cannot be rendered at
	// all by this implementation.
	Error string `json:"error,omitempty"`

	// Mismatches are the expected structural differences in the generated HTML.
	Mismatches []mismatchRecord `json:"mismatches,omitempty"`
}

type mismatchRecord struct {
	Path   string `json:"path"`
	Detail string `json:"detail"`
}

func (m mismatchRecord) String() string {
	return m.Path + ": " + m.Detail
}

func newMismatchRecords(mismatches []mjmltest.Mismatch) []mismatchRecord {
	if len(mismatches) == 0 {
		return nil
	}

	records := make([]mismatchRecord, len(mismatches))
	for i, mismatch := range mismatches {
		records[i] = mismatchRecord{Path: mismatch.Path, Detail: mismatch.Detail}
	}

	return records
}

// TestMJMLParity renders every fixture in testdata with both this package and
// the reference JavaScript MJML implementation and compares the resulting HTML
// documents structurally.
//
// The reference version is pinned in package.json; see testdata/README.md.
// Divergences that are known and accepted are recorded in the baseline file, so
// the test fails when HTML output regresses or when a recorded divergence is
// fixed (both are changes worth reviewing). Re-record with:
//
//	go test ./ -run TestMJMLParity -update
func TestMJMLParity(t *testing.T) {
	// The reference implementation is the real mjml CLI, so this needs Node.js.
	if testing.Short() {
		t.Skip("skipping parity test against the reference MJML implementation in short mode")
	}

	fixtures := mjmlFixtures(t, parityDir)
	require.NotEmpty(t, fixtures, "no .mjml fixtures found in %s", parityDir)

	cli := mjmltest.NewCLI(t)
	t.Logf("reference implementation: MJML %s (%s)", cli.Version, cli.Source)

	baseline := loadBaseline(t, cli.Version)

	reference := cli.RenderAll(t, fixtures)

	updated := parityBaseline{
		MJMLVersion: cli.Version,
		Fixtures:    make(map[string]fixtureDivergence, len(baseline.Fixtures)),
	}

	for _, fixture := range fixtures {
		input, err := os.ReadFile(fixture)
		require.NoError(t, err)

		want, ok := reference[fixture]
		require.True(t, ok, "no reference output for %s", fixture)

		got, renderErr := RenderMJML(bytes.NewReader(input))

		var actual fixtureDivergence

		if renderErr != nil {
			actual.Error = renderErr.Error()
		} else {
			mismatches, err := mjmltest.CompareHTML(want, got)
			require.NoError(t, err)
			actual.Mismatches = newMismatchRecords(mismatches)
		}

		if actual.Error != "" || len(actual.Mismatches) > 0 {
			updated.Fixtures[fixture] = actual
		}

		expected := baseline.Fixtures[fixture]

		if *updateBaseline {
			continue
		}

		t.Run(filepath.ToSlash(fixture), func(t *testing.T) {
			if diff := compareDivergences(expected, actual); diff != "" {
				if renderErr == nil && len(actual.Mismatches) > 0 {
					writeFailureArtifacts(t, fixture, want, got)
				}

				t.Fatalf(
					"HTML for %s diverges from MJML %s in a way that is not recorded in %s:\n%s\n"+
						"Fix the implementation, or re-record the baseline with: go test ./ -run TestMJMLParity -update",
					fixture, cli.Version, baselinePath, diff,
				)
			}
		})
	}

	if *updateBaseline {
		writeBaseline(t, updated)
		t.Logf("recorded parity baseline for MJML %s in %s", updated.MJMLVersion, baselinePath)
	}
}

// loadBaseline reads the recorded divergences. A missing or stale baseline is
// not fatal: it only means every divergence is currently "new".
func loadBaseline(t testing.TB, version string) parityBaseline {
	t.Helper()

	baseline := parityBaseline{
		MJMLVersion: version,
		Fixtures:    map[string]fixtureDivergence{},
	}

	raw, err := os.ReadFile(baselinePath)
	if err != nil {
		if *updateBaseline {
			return baseline
		}

		t.Fatalf("read %s: %v (record it with: go test ./ -run TestMJMLParity -update)", baselinePath, err)
	}

	require.NoError(t, json.Unmarshal(raw, &baseline), "parse %s", baselinePath)

	if *updateBaseline {
		return baseline
	}

	if baseline.MJMLVersion != version {
		t.Fatalf(
			"%s was recorded against MJML %s but the reference implementation is MJML %s; "+
				"review the HTML changes and re-record with: go test ./ -run TestMJMLParity -update",
			baselinePath, baseline.MJMLVersion, version,
		)
	}

	return baseline
}

func writeBaseline(t testing.TB, baseline parityBaseline) {
	t.Helper()

	if baseline.Fixtures == nil {
		baseline.Fixtures = map[string]fixtureDivergence{}
	}

	raw, err := json.MarshalIndent(baseline, "", "  ")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(baselinePath, append(raw, '\n'), 0o644))
}

// compareDivergences describes how actual divergences differ from the expected
// ones. An empty string means they are identical.
func compareDivergences(expected, actual fixtureDivergence) string {
	var b strings.Builder

	if expected.Error != actual.Error {
		switch {
		case expected.Error == "":
			fmt.Fprintf(&b, "  unrecorded render error: %s\n", actual.Error)
		case actual.Error == "":
			fmt.Fprintf(&b, "  recorded render error is gone: %s\n", expected.Error)
		default:
			fmt.Fprintf(&b, "  render error changed:\n    want: %s\n    got:  %s\n", expected.Error, actual.Error)
		}
	}

	expectedSet := make(map[string]struct{}, len(expected.Mismatches))
	for _, mismatch := range expected.Mismatches {
		expectedSet[mismatch.String()] = struct{}{}
	}

	actualSet := make(map[string]struct{}, len(actual.Mismatches))
	for _, mismatch := range actual.Mismatches {
		actualSet[mismatch.String()] = struct{}{}

		if _, ok := expectedSet[mismatch.String()]; !ok {
			fmt.Fprintf(&b, "  unrecorded difference: %s\n", mismatch)
		}
	}

	for _, mismatch := range expected.Mismatches {
		if _, ok := actualSet[mismatch.String()]; !ok {
			fmt.Fprintf(&b, "  recorded difference is fixed: %s\n", mismatch)
		}
	}

	return b.String()
}

// writeFailureArtifacts stores the reference and generated HTML plus their
// normalized forms so failures can be inspected with a normal diff tool.
func writeFailureArtifacts(t *testing.T, fixture, want, got string) {
	t.Helper()

	dir := filepath.Join("testdata", ".failures")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Logf("could not create failure artifact directory: %v", err)
		return
	}

	name := strings.TrimSuffix(filepath.Base(fixture), filepath.Ext(fixture))

	artifacts := map[string]string{
		name + ".want.html":          want,
		name + ".got.html":           got,
		name + ".want.canonical.txt": mjmltest.CanonicalMust(want),
		name + ".got.canonical.txt":  mjmltest.CanonicalMust(got),
	}

	for file, content := range artifacts {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
			t.Logf("could not write failure artifact %s: %v", file, err)
			continue
		}
	}

	t.Logf("wrote failure artifacts to %s", dir)
}

// TestMJMLInvalid asserts that malformed or invalid documents are rejected with
// a clear error. This implementation validates hard (fail fast), while MJML
// itself only warns and keeps rendering, so these fixtures are deliberately not
// part of the parity corpus.
func TestMJMLInvalid(t *testing.T) {
	t.Parallel()

	fixtures := mjmlFixtures(t, invalidDir)
	require.NotEmpty(t, fixtures, "no .mjml fixtures found in %s", invalidDir)

	for _, fixture := range fixtures {
		t.Run(filepath.ToSlash(fixture), func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(fixture)
			require.NoError(t, err)

			html, err := RenderMJML(bytes.NewReader(input))

			expectedPath := fixture + ".error.txt"
			expected, readErr := os.ReadFile(expectedPath)
			require.NoError(t, readErr, "fixture %s needs a %s with the expected error", fixture, filepath.Base(expectedPath))

			require.Error(t, err, "expected %s to be rejected, rendered:\n%s", fixture, html)
			assert.Contains(t, err.Error(), strings.TrimSpace(string(expected)))
		})
	}
}

// mjmlFixtures returns all .mjml files below dir, sorted for stable output.
func mjmlFixtures(t testing.TB, dir string) []string {
	t.Helper()

	var fixtures []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path != dir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}

			return nil
		}

		if filepath.Ext(path) == ".mjml" {
			fixtures = append(fixtures, path)
		}

		return nil
	})
	require.NoError(t, err)

	slices.Sort(fixtures)

	return fixtures
}

func TestRender(t *testing.T) {
	input, err := os.ReadFile("input_s.mjml")
	require.NoError(t, err)

	out, err := RenderMJML(bytes.NewReader(input))
	require.NoError(t, err)

	os.WriteFile("out.html", []byte(out), 0o644)
	_ = out
	//t.Log(out)
}

func TestInlineCSS(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		const input = `<html><body><p class="p-class">Hello</p></body></html>`
		ctx := &component.RenderContext{
			InlineStyles: []component.Stylesheet{
				{
					Rules: []component.Rule{
						{
							Selectors: ".p-class",
							Declarations: []component.Style{
								{Property: "font-size", Value: "22px", Important: true},
							}},
					},
				},
			},
		}

		var out bytes.Buffer
		err := inlineCSS(ctx, strings.NewReader(input), &out)
		require.NoError(t, err)

		require.Equal(t, "<html><head></head><body><p class=\"p-class\" style=\"font-size:22px;\">Hello</p></body></html>", out.String())
	})

	t.Run("important-override", func(t *testing.T) {
		const input = `<html><body><p class="p-class" style="font-size: 20px;font-weight:300">Hello</p></body></html>`
		ctx := &component.RenderContext{
			InlineStyles: []component.Stylesheet{
				{
					Rules: []component.Rule{
						{
							Selectors: ".p-class",
							Declarations: []component.Style{
								{Property: "font-size", Value: "22px", Important: true},
								{Property: "font-weight", Value: "400"},
							}},
					},
				},
			},
		}

		var out bytes.Buffer
		err := inlineCSS(ctx, strings.NewReader(input), &out)
		require.NoError(t, err)

		randomOrder := []string{
			"<html><head></head><body><p class=\"p-class\" style=\"font-size:22px;font-weight:300;\">Hello</p></body></html>",
			"<html><head></head><body><p class=\"p-class\" style=\"font-weight:300;font-size:22px;\">Hello</p></body></html>",
		}

		assert.True(t, slices.Contains(randomOrder, out.String()), "returned value should be one of the possible")
	})
}

// BenchmarkRenderMJML measures the full pipeline (parse, render, inline CSS,
// output post-processing) per fixture. The fixture set is the parity corpus, so
// the numbers describe realistic documents rather than toy inputs.
//
// Budgets live in benchmarks/BUDGETS.md.
func BenchmarkRenderMJML(b *testing.B) {
	for _, fixture := range mjmlFixtures(b, parityDir) {
		name := strings.TrimPrefix(filepath.ToSlash(fixture), parityDir+"/")

		b.Run(name, func(b *testing.B) {
			input, err := os.ReadFile(fixture)
			if err != nil {
				b.Fatal(err)
			}

			if _, err := RenderMJML(bytes.NewReader(input)); err != nil {
				// Fixtures that do not render yet are tracked by the parity
				// baseline, not by the benchmarks.
				b.Skipf("fixture does not render: %v", err)
			}

			b.ReportAllocs()
			b.SetBytes(int64(len(input)))

			for b.Loop() {
				if _, err := RenderMJML(bytes.NewReader(input)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkParityComparison isolates the test harness cost (normalize both
// documents and compare them), which gates how large the parity corpus can get.
func BenchmarkParityComparison(b *testing.B) {
	for _, fixture := range mjmlFixtures(b, parityDir) {
		name := strings.TrimPrefix(filepath.ToSlash(fixture), parityDir+"/")

		b.Run(name, func(b *testing.B) {
			input, err := os.ReadFile(fixture)
			if err != nil {
				b.Fatal(err)
			}

			html, err := RenderMJML(bytes.NewReader(input))
			if err != nil {
				b.Skipf("fixture does not render: %v", err)
			}

			b.ReportAllocs()

			for b.Loop() {
				if _, err := mjmltest.CompareHTML(html, html); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// FuzzRenderMJML asserts that the renderer never panics on arbitrary input and
// that anything it accepts produces a well-formed HTML document. Malformed input
// is expected to return an error (this implementation fails fast).
func FuzzRenderMJML(f *testing.F) {
	f.Add(`<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section></mj-body></mjml>`)
	f.Add(`<mjml><mj-body></mj-body></mjml>`)
	f.Add(`<mjml lang="de" dir="rtl"><mj-body><mj-section><mj-column><mj-text font-size="12px">x</mj-text></mj-column></mj-section></mj-body></mjml>`)
	f.Add(`<mjml><mj-head><mj-attributes><mj-all font-family="Arial" /></mj-attributes></mj-head><mj-body /></mjml>`)
	f.Add(`<mjml><mj-body><mj-section><mj-column><mj-raw><div>x</div></mj-raw></mj-column></mj-section></mj-body></mjml>`)
	f.Add(`<mjml><mj-body><mj-section><mj-column><mj-text`)
	f.Add(``)

	for _, fixture := range []string{"testdata/parity/simple.mjml", "testdata/parity/content.mjml", "testdata/parity/head.mjml"} {
		input, err := os.ReadFile(fixture)
		if err != nil {
			f.Fatal(err)
		}

		f.Add(string(input))
	}

	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 1<<20 {
			t.Skip("input too large for a fuzz seed")
		}

		html, err := RenderMJML(strings.NewReader(source))
		if err != nil {
			return
		}

		if html == "" {
			t.Fatalf("rendered empty output for input %q", source)
		}

		if !strings.Contains(strings.ToLower(html), "<!doctype html>") {
			t.Fatalf("output is not an HTML document: %q", html)
		}
	})
}

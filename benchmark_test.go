package mjmlgo

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/internal/mjmltest"
)

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

// BenchmarkParse isolates the MJML parser.
func BenchmarkParse(b *testing.B) {
	for _, fixture := range mjmlFixtures(b, parityDir) {
		name := strings.TrimPrefix(filepath.ToSlash(fixture), parityDir+"/")

		b.Run(name, func(b *testing.B) {
			input, err := os.ReadFile(fixture)
			if err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.SetBytes(int64(len(input)))

			for b.Loop() {
				if _, err := parse(bytes.NewReader(input)); err != nil {
					b.Skipf("fixture does not parse: %v", err)
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

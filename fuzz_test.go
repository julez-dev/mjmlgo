package mjmlgo

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

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

// FuzzParse asserts the parser never panics and either returns an error or a
// root node; it deliberately does not require MJML validity.
func FuzzParse(f *testing.F) {
	f.Add(`<mjml><mj-body><mj-raw><div></mj-raw></mj-body></mjml>`)
	f.Add(`<mjml><mj-body><mj-text><h1>Test</h1></mj-text></mj-body></mjml>`)
	f.Add(`<mjml><mj-body><mj-section attr="value"><mj-column /></mj-section></mj-body></mjml>`)
	f.Add(`<mjml></mjml><mjml></mjml>`)
	f.Add(`<mjml><mj-body>`)
	f.Add(`not mjml`)

	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 1<<20 {
			t.Skip("input too large for a fuzz seed")
		}

		root, err := parse(bytes.NewReader([]byte(source)))
		if err != nil {
			return
		}

		if root == nil {
			t.Fatalf("parse returned nil root without an error for input %q", source)
		}
	})
}

package mjmltest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// SidecarSuffix names the file that records the reference CLI flags a fixture
// must be rendered with. It sits next to the fixture: simple.mjml is configured
// by simple.mjml.cli-args. A fixture without a sidecar uses the reference
// defaults, which is the common case.
const SidecarSuffix = ".cli-args"

// RenderConfig is the reference renderer configuration a fixture's sidecar asks
// for. A nil pointer means "not set": the option keeps its reference default.
// It is the harness's mapping of the reference CLI flags onto the options the Go
// renderer exposes, so both sides are compared under the same configuration.
type RenderConfig struct {
	// KeepComments is MJML's keepComments option (default true).
	KeepComments *bool
	// IgnoreIncludes is MJML's ignoreIncludes option (default false).
	IgnoreIncludes *bool
	// PrinterSupport is MJML's printerSupport option (default false).
	PrinterSupport *bool
	// Fonts replaces the web-font registry, in the order the sidecar's JSON
	// object lists the families: the reference implementation emits a registry
	// in insertion order, so the sidecar's order is part of what is compared. It
	// is never nil: an empty object replaces the defaults with nothing, and a
	// fixture that wants the defaults omits the flag.
	Fonts []Font
	// FontsSet reports that the sidecar configures fonts, so an empty registry
	// is distinguishable from "not configured".
	FontsSet bool
}

// Font is one entry of the web-font registry a sidecar configures, in the order
// the JSON object lists it.
type Font struct {
	Family string
	Href   string
}

// Args returns the raw reference CLI arguments recorded for fixture, or nil when
// the fixture has no sidecar. The arguments are passed to the reference CLI
// unchanged.
//
// The sidecar format is one argument per line: blank lines and lines starting
// with '#' are ignored, and every other line is passed through with surrounding
// whitespace trimmed. One argument per line keeps values that contain spaces -
// a font URL in the JSON of -c.fonts, for instance - unambiguous.
func Args(t testing.TB, fixture string) []string {
	t.Helper()

	path := fixture + SidecarSuffix

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		t.Fatalf("read fixture sidecar %s: %v", path, err)
	}

	var args []string

	for line := range strings.SplitSeq(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		args = append(args, line)
	}

	return args
}

// Config returns the renderer configuration recorded for fixture. A flag the
// harness does not know is a test failure: silently ignoring it would compare
// the reference run under one configuration against our defaults.
func Config(t testing.TB, fixture string) RenderConfig {
	t.Helper()

	config, err := ParseConfig(fixture, Args(t, fixture))
	if err != nil {
		t.Fatalf("%v", err)
	}

	return config
}

// ParseConfig maps the raw reference CLI arguments of fixture onto a
// RenderConfig. fixture only names the sidecar in error messages.
//
// Only the options the renderer exposes can be configured here. The reference
// CLI accepts both -c.<name>=<value> and --config.<name>=<value>; both spellings
// describe the same mjml-core option and both are accepted.
//
// The value semantics follow the reference CLI, which is not uniform:
//
//   - keepComments: mjml-cli converts only the literal string "false" to the
//     boolean false; every other value leaves comments on.
//   - ignoreIncludes and printerSupport: the CLI hands the raw string to
//     mjml-core, where a non-empty string is truthy - so "false" enables them.
//     Sidecars therefore use "true" to switch them on.
//   - fonts: JSON, parsed by the CLI. Only an object is a usable registry; every
//     other value reaches mjml-core as the string the CLI saw. See parseFonts.
func ParseConfig(fixture string, args []string) (RenderConfig, error) {
	var config RenderConfig

	for _, arg := range args {
		name, value, ok := splitConfigFlag(arg)
		if !ok {
			return RenderConfig{}, fmt.Errorf(
				"%s: %q is not a -c.<name>=<value> or --config.<name>=<value> reference flag; "+
					"the harness cannot map it onto a renderer option",
				fixture+SidecarSuffix, arg,
			)
		}

		switch name {
		case "keepComments":
			keep := value != "false"
			config.KeepComments = &keep
		case "ignoreIncludes":
			ignore := truthy(value)
			config.IgnoreIncludes = &ignore
		case "printerSupport":
			printer := truthy(value)
			config.PrinterSupport = &printer
		case "fonts":
			fonts, err := parseFonts(value)
			if err != nil {
				return RenderConfig{}, fmt.Errorf("%s: -c.fonts=%s: %w", fixture+SidecarSuffix, value, err)
			}

			config.Fonts = fonts
			config.FontsSet = true
		default:
			return RenderConfig{}, fmt.Errorf(
				"%s: unknown reference config option %q; add a mapping in mjmltest.ParseConfig "+
					"and a renderer option before using it in a fixture",
				fixture+SidecarSuffix, name,
			)
		}
	}

	return config, nil
}

// splitConfigFlag splits a -c.<name>=<value> reference flag into its parts. The
// long spelling --config.<name>=<value> is accepted as well.
func splitConfigFlag(arg string) (name, value string, ok bool) {
	rest, found := strings.CutPrefix(arg, "-c.")
	if !found {
		rest, found = strings.CutPrefix(arg, "--config.")
	}

	if !found {
		return "", "", false
	}

	name, value, found = strings.Cut(rest, "=")
	if !found || name == "" {
		return "", "", false
	}

	return name, value, true
}

// truthy reports whether the reference implementation treats value as a set
// option: mjml-core receives the raw string from the CLI, and every non-empty
// string is truthy in JavaScript - including "false".
func truthy(value string) bool {
	return value != ""
}

// parseFonts decodes the JSON font registry of -c.fonts, keeping the order the
// object lists its families in. That order is what the reference implementation
// emits, so it is part of the comparison.
//
// Only a JSON object is a usable registry. The CLI parses nothing else into an
// object: null, "", false and friends reach mjml-core as the string the CLI saw,
// and [] as an array, so the registry stops being a name-to-URL map. <mj-font>
// registration then throws ("Cannot create property 'Unused Font' on string
// 'null'") or the emitted links are built from characters and attribute names.
// A sidecar asking for that would compare the reference under a broken
// configuration against one of our options, so it fails instead. Omit the flag
// to keep the default registry and use {} to replace it with nothing.
func parseFonts(value string) ([]Font, error) {
	decoder := json.NewDecoder(strings.NewReader(value))

	fonts, err := decodeFontObject(decoder)
	if err != nil {
		return nil, fmt.Errorf("decode JSON font registry: %w", err)
	}

	return fonts, nil
}

// errNotFontObject is the diagnosis for every -c.fonts value that is not a JSON
// object. The reference CLI hands such a value to mjml-core unchanged, where the
// registry stops being a name-to-URL map: <mj-font> registration throws ("Cannot
// create property 'Unused Font' on string 'null'") and emitted links are built
// from the characters of the value.
var errNotFontObject = errors.New(
	"not a JSON object of family to URL: the reference CLI hands anything else to mjml-core, " +
		"where the font registry is unusable - omit the flag for the default registry, use {} for none")

// decodeFontObject reads one JSON object as an ordered font list. An empty object
// is an empty registry, which is not the same as no registry.
func decodeFontObject(decoder *json.Decoder) ([]Font, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, errNotFontObject
	}

	fonts := []Font{}

	for decoder.More() {
		family, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		name, ok := family.(string)
		if !ok {
			return nil, errors.New("object key is not a string")
		}

		var href string
		if err := decoder.Decode(&href); err != nil {
			return nil, fmt.Errorf("family %q: %w", name, err)
		}

		fonts = append(fonts, Font{Family: name, Href: href})
	}

	// Consume the closing brace.
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}

	return fonts, nil
}

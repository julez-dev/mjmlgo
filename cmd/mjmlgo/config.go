package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/julez-dev/mjmlgo/v2"
)

// renderConfig is what the --config assignments add to a render.
type renderConfig struct {
	// root overrides the directory mj-include paths resolve against. Empty
	// means the directory of the document itself.
	root    string
	options []mjmlgo.Option
}

// parseConfig turns --config assignments into renderer options. The reference
// CLI forwards anything it is given to mjml-core; this renderer has fewer knobs,
// so an unknown key is an error rather than silently ignored.
func parseConfig(pairs []configPair) (renderConfig, error) {
	var cfg renderConfig

	for _, pair := range pairs {
		switch pair.key {
		case "fonts":
			fonts, err := parseFonts(pair.value)
			if err != nil {
				return cfg, err
			}

			cfg.options = append(cfg.options, mjmlgo.WithFonts(fonts))
		case "keepComments":
			keep, err := strconv.ParseBool(pair.value)
			if err != nil {
				return cfg, fmt.Errorf("config.keepComments: %q is not a boolean", pair.value)
			}

			cfg.options = append(cfg.options, mjmlgo.WithComments(keep))
		case "filePath":
			info, err := os.Stat(pair.value)
			if err != nil || !info.IsDir() {
				return cfg, fmt.Errorf("config.filePath: %q is not a directory", pair.value)
			}

			cfg.root = pair.value
		case "beautify", "minify", "minifyOptions":
			return cfg, fmt.Errorf("config.%s: output is always compact, so there is nothing to %s", pair.key, pair.key)
		default:
			return cfg, fmt.Errorf("config.%s: unknown option (supported: fonts, keepComments, filePath)", pair.key)
		}
	}

	return cfg, nil
}

// parseFonts reads the web-font registry as a JSON object, the form the
// reference CLI accepts for config.fonts. The object's own order is the order the
// families are emitted in, so it is read with a token stream rather than decoded
// into a Go map, which would lose it.
func parseFonts(value string) ([]mjmlgo.Font, error) {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()

	fonts, err := decodeFontObject(decoder)
	if err != nil {
		return nil, fmt.Errorf("config.fonts: %q: %w", value, err)
	}

	// Anything after the object means the value was not one.
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("config.fonts: %q: trailing data after the object", value)
	}

	return fonts, nil
}

// decodeFontObject reads one JSON object as an ordered font list.
func decodeFontObject(decoder *json.Decoder) ([]mjmlgo.Font, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, errors.New("not a JSON object of family to URL")
	}

	var fonts []mjmlgo.Font

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

		fonts = append(fonts, mjmlgo.Font{Family: name, Href: href})
	}

	// Consume the closing brace.
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}

	return fonts, nil
}

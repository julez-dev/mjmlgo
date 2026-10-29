package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/julez-dev/mjmlgo"
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
// reference CLI accepts for config.fonts.
func parseFonts(value string) (map[string]string, error) {
	fonts := map[string]string{}

	if err := json.Unmarshal([]byte(value), &fonts); err != nil {
		return nil, fmt.Errorf(`config.fonts: %q is not a JSON object of family to URL: %w`, value, err)
	}

	return fonts, nil
}

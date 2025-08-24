package component

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var ErrValidation = errors.New("failed validation")

var (
	// Regex for different color formats
	rgbaRegex = regexp.MustCompile(`(?i)^rgba\(\s*\d{1,3}\s*,\s*\d{1,3}\s*,\s*\d{1,3}\s*,\s*\d(\.\d+)?\s*\)$`)
	rgbRegex  = regexp.MustCompile(`(?i)^rgb\(\s*\d{1,3}\s*,\s*\d{1,3}\s*,\s*\d{1,3}\s*\)$`)
	hexRegex  = regexp.MustCompile(`^#([0-9a-fA-F]{3}){1,2}$`)

	// Regex for expanding shorthand hex values
	shorthandHexRegex = regexp.MustCompile(`^#([0-9a-fA-F])([0-9a-fA-F])([0-9a-fA-F])$`)
)

var cssColorNames = map[string]struct{}{
	"aliceblue": {}, "antiquewhite": {}, "aqua": {}, "aquamarine": {}, "azure": {}, "beige": {}, "bisque": {}, "black": {},
	"blanchedalmond": {}, "blue": {}, "blueviolet": {}, "brown": {}, "burlywood": {}, "cadetblue": {}, "chartreuse": {},
	"chocolate": {}, "coral": {}, "cornflowerblue": {}, "cornsilk": {}, "crimson": {}, "cyan": {}, "darkblue": {}, "darkcyan": {},
	"darkgoldenrod": {}, "darkgray": {}, "darkgreen": {}, "darkgrey": {}, "darkkhaki": {}, "darkmagenta": {}, "darkolivegreen": {},
	"darkorange": {}, "darkorchid": {}, "darkred": {}, "darksalmon": {}, "darkseagreen": {}, "darkslateblue": {}, "darkslategray": {},
	"darkslategrey": {}, "darkturquoise": {}, "darkviolet": {}, "deeppink": {}, "deepskyblue": {}, "dimgray": {}, "dimgrey": {},
	"dodgerblue": {}, "firebrick": {}, "floralwhite": {}, "forestgreen": {}, "fuchsia": {}, "gainsboro": {}, "ghostwhite": {},
	"gold": {}, "goldenrod": {}, "gray": {}, "green": {}, "greenyellow": {}, "grey": {}, "honeydew": {}, "hotpink": {}, "indianred": {},
	"indigo": {}, "ivory": {}, "khaki": {}, "lavender": {}, "lavenderblush": {}, "lawngreen": {}, "lemonchiffon": {}, "lightblue": {},
	"lightcoral": {}, "lightcyan": {}, "lightgoldenrodyellow": {}, "lightgray": {}, "lightgreen": {}, "lightgrey": {}, "lightpink": {},
	"lightsalmon": {}, "lightseagreen": {}, "lightskyblue": {}, "lightslategray": {}, "lightslategrey": {}, "lightsteelblue": {},
	"lightyellow": {}, "lime": {}, "limegreen": {}, "linen": {}, "magenta": {}, "maroon": {}, "mediumaquamarine": {}, "mediumblue": {},
	"mediumorchid": {}, "mediumpurple": {}, "mediumseagreen": {}, "mediumslateblue": {}, "mediumspringgreen": {}, "mediumturquoise": {},
	"mediumvioletred": {}, "midnightblue": {}, "mintcream": {}, "mistyrose": {}, "moccasin": {}, "navajowhite": {}, "navy": {},
	"oldlace": {}, "olive": {}, "olivedrab": {}, "orange": {}, "orangered": {}, "orchid": {}, "palegoldenrod": {}, "palegreen": {},
	"paleturquoise": {}, "palevioletred": {}, "papayawhip": {}, "peachpuff": {}, "peru": {}, "pink": {}, "plum": {}, "powderblue": {},
	"purple": {}, "rebeccapurple": {}, "red": {}, "rosybrown": {}, "royalblue": {}, "saddlebrown": {}, "salmon": {}, "sandybrown": {},
	"seagreen": {}, "seashell": {}, "sienna": {}, "silver": {}, "skyblue": {}, "slateblue": {}, "slategray": {}, "slategrey": {},
	"snow": {}, "springgreen": {}, "steelblue": {}, "tan": {}, "teal": {}, "thistle": {}, "tomato": {}, "turquoise": {}, "violet": {},
	"wheat": {}, "white": {}, "whitesmoke": {}, "yellow": {}, "yellowgreen": {},
}

type validateAttributeFunc func(value string) error

func validateEnum(valid []string) validateAttributeFunc {
	return func(value string) error {
		if value == "" {
			return nil
		}
		if !slices.Contains(valid, value) {
			return fmt.Errorf("%w: %s not one of %v", ErrValidation, value, valid)
		}
		return nil
	}
}

func validateColor() validateAttributeFunc {
	return func(value string) error {
		if value == "" {
			return nil
		}
		if rgbaRegex.MatchString(value) || rgbRegex.MatchString(value) || hexRegex.MatchString(value) {
			return nil
		}

		if _, ok := cssColorNames[strings.ToLower(value)]; ok {
			return nil
		}

		return fmt.Errorf("%w: %s not a valid color string", ErrValidation, value)
	}
}

func validateUnit(validUnits []string, isMultipleValues bool) validateAttributeFunc {
	return func(value string) error {
		if value == "" {
			return nil
		}

		var splits []string
		if isMultipleValues {
			splits = strings.Split(value, " ")
		} else {
			if strings.Contains(strings.TrimSpace(value), " ") {
				return fmt.Errorf("%w: did not expect attribute value with multiple units in %s", ErrValidation, value)
			}

			splits = append(splits, value)
		}

		for _, s := range splits {
			if s == "0" {
				continue
			}

			matches := valueAndUnitRegex.FindStringSubmatch(s)
			if len(matches) < 3 {
				return fmt.Errorf("%w: unable to pares CSS unit from: %s", ErrValidation, s)
			}

			if !slices.Contains(validUnits, matches[2]) {
				return fmt.Errorf("%w: unit %q from value %q is not one of %v", ErrValidation, matches[2], s, validUnits)
			}
		}
		return nil
	}
}

func validateType(expectedType string) validateAttributeFunc {
	return func(value string) error {
		if value == "" {
			return nil
		}

		if expectedType == "string" {
			return nil
		}

		if expectedType == "number" {
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return fmt.Errorf("%w: value %q is not a valid number", ErrValidation, value)
			}
		}

		if expectedType == "boolean" {
			if value != "true" && value != "false" {
				return fmt.Errorf("%w: value %q is not a boolean", ErrValidation, value)
			}
		}

		return nil
	}
}

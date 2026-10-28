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
	// Regex for different color formats. These mirror the type matchers of
	// mjml-core/lib/types/color.js, including the fact that the rgb/rgba forms
	// are not anchored there.
	rgbaRegex = regexp.MustCompile(`(?i)rgba\(\d{1,3},\s?\d{1,3},\s?\d{1,3},\s?\d(\.\d{1,3})?\)`)
	rgbRegex  = regexp.MustCompile(`(?i)rgb\(\d{1,3},\s?\d{1,3},\s?\d{1,3}\)`)
	hexRegex  = regexp.MustCompile(`^#([0-9a-fA-F]{3}){1,2}$`)

	// Regex for expanding shorthand hex values
	shorthandHexRegex = regexp.MustCompile(`^#([0-9a-fA-F])([0-9a-fA-F])([0-9a-fA-F])$`)

	// unitValueRegex splits a single unit value into its optional sign, numeric
	// part and unit. It mirrors the permissive number part of MJML's unit type,
	// which accepts any sequence of digits, dots and commas.
	unitValueRegex = regexp.MustCompile(`^(-?)([\d.,]+)([A-Za-z%]*)$`)

	// integerRegex mirrors MJML's integer type, which is an unanchored \d+.
	integerRegex = regexp.MustCompile(`\d+`)
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
	// CSS-wide keywords accepted by MJML's colour type.
	"inherit": {}, "transparent": {},
}

type validateAttributeFunc func(value string) error

func validateEnum(valid []string) validateAttributeFunc {
	return func(value string) error {
		if !slices.Contains(valid, value) {
			return fmt.Errorf("%s not one of %v", value, valid)
		}
		return nil
	}
}

func validateColor() validateAttributeFunc {
	return func(value string) error {
		if rgbaRegex.MatchString(value) || rgbRegex.MatchString(value) || hexRegex.MatchString(value) {
			return nil
		}

		if _, ok := cssColorNames[strings.ToLower(value)]; ok {
			return nil
		}

		return fmt.Errorf("%s not a valid color string", value)
	}
}

// validateUnit validates a unit(units) or unit(units){1,4} attribute. Multiple
// values are space separated and, like MJML's shorthand types, limited to one
// to four values.
func validateUnit(validUnits []string, isMultipleValues bool) validateAttributeFunc {
	maxValues := 1
	if isMultipleValues {
		maxValues = 4
	}

	return validateUnitValues(validUnits, 1, maxValues, false)
}

// validateUnitWithNegative is the unitWithNegative type: identical to unit,
// except that a negative numeric part is allowed. MJML uses it for
// letter-spacing on text, button, navbar-link and accordion-text.
func validateUnitWithNegative(validUnits []string, isMultipleValues bool) validateAttributeFunc {
	maxValues := 1
	if isMultipleValues {
		maxValues = 4
	}

	return validateUnitValues(validUnits, 1, maxValues, true)
}

func validateUnitValues(validUnits []string, minValues, maxValues int, allowNegative bool) validateAttributeFunc {
	return func(value string) error {
		if value == "" {
			return errors.New("empty value is not a valid unit")
		}

		values := strings.Split(value, " ")
		if len(values) < minValues || len(values) > maxValues {
			return fmt.Errorf("%q has %d value(s), want %d to %d", value, len(values), minValues, maxValues)
		}

		for _, s := range values {
			if s == "" {
				return fmt.Errorf("empty value in %q", value)
			}

			if s == "0" {
				continue
			}

			if s == "auto" {
				if !slices.Contains(validUnits, "auto") || len(values) != 1 {
					return fmt.Errorf("unit %q from value %q is not one of %v", s, value, validUnits)
				}

				continue
			}

			matches := unitValueRegex.FindStringSubmatch(s)
			if matches == nil {
				return fmt.Errorf("unable to parse CSS unit from: %s", s)
			}

			negative, number, unit := matches[1], matches[2], matches[3]
			if negative != "" && !allowNegative {
				return fmt.Errorf("negative value %q is not allowed", s)
			}

			if number == "" {
				return fmt.Errorf("unable to parse CSS unit from: %s", s)
			}

			if !slices.Contains(validUnits, unit) {
				return fmt.Errorf("unit %q from value %q is not one of %v", unit, s, validUnits)
			}
		}

		return nil
	}
}

func validateInteger() validateAttributeFunc {
	return func(value string) error {
		if !integerRegex.MatchString(value) {
			return fmt.Errorf("value %q is not an integer", value)
		}
		return nil
	}
}

func validateType(expectedType string) validateAttributeFunc {
	return func(value string) error {
		switch expectedType {
		case "string":
			return nil
		case "boolean":
			if !strings.EqualFold(value, "true") && !strings.EqualFold(value, "false") {
				return fmt.Errorf("value %q is not a boolean", value)
			}
		case "number":
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return fmt.Errorf("value %q is not a valid number", value)
			}
		case "integer":
			return validateInteger()(value)
		}

		return nil
	}
}

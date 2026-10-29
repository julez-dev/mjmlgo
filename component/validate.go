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

// AttributeValidator checks one attribute value written by the document author.
// It returns nil when the value is acceptable and an error describing why it is
// not otherwise. The error text is embedded in the structured *Error the
// validator site returns, so it should name what was rejected and why.
//
// A nil validator accepts every value; leave an attribute out of the
// AllowedAttributes map to reject it instead.
type AttributeValidator func(value string) error

func validateEnum(valid []string) AttributeValidator {
	return func(value string) error {
		if !slices.Contains(valid, value) {
			return fmt.Errorf("%s not one of %v", value, valid)
		}
		return nil
	}
}

func validateColor() AttributeValidator {
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
func validateUnit(validUnits []string, isMultipleValues bool) AttributeValidator {
	maxValues := 1
	if isMultipleValues {
		maxValues = 4
	}

	return validateUnitValues(validUnits, 1, maxValues, false)
}

// validateUnitWithNegative is the unitWithNegative type: identical to unit,
// except that a negative numeric part is allowed. MJML uses it for
// letter-spacing on text, button, navbar-link and accordion-text.
func validateUnitWithNegative(validUnits []string, isMultipleValues bool) AttributeValidator {
	maxValues := 1
	if isMultipleValues {
		maxValues = 4
	}

	return validateUnitValues(validUnits, 1, maxValues, true)
}

func validateUnitValues(validUnits []string, minValues, maxValues int, allowNegative bool) AttributeValidator {
	return func(value string) error {
		if value == "" {
			return errors.New("empty value is not a valid unit")
		}

		// The values are space separated, like MJML's shorthand types. Count
		// them without splitting so the success path allocates nothing: a space
		// always starts a new value, including the empty values produced by
		// leading, trailing and repeated spaces.
		valueCount := strings.Count(value, " ") + 1
		if valueCount < minValues || valueCount > maxValues {
			return fmt.Errorf("%q has %d value(s), want %d to %d", value, valueCount, minValues, maxValues)
		}

		for start := 0; ; {
			end := strings.IndexByte(value[start:], ' ')

			var s string

			if end < 0 {
				s = value[start:]
			} else {
				end += start
				s = value[start:end]
			}

			if s == "" {
				return fmt.Errorf("empty value in %q", value)
			}

			if s != "0" {
				if s == "auto" {
					if !slices.Contains(validUnits, "auto") || valueCount != 1 {
						return fmt.Errorf("unit %q from value %q is not one of %v", s, value, validUnits)
					}
				} else {
					negative, unit, ok := parseUnitValue(s)
					if !ok {
						return fmt.Errorf("unable to parse CSS unit from: %s", s)
					}

					if negative && !allowNegative {
						return fmt.Errorf("negative value %q is not allowed", s)
					}

					if !slices.Contains(validUnits, unit) {
						return fmt.Errorf("unit %q from value %q is not one of %v", unit, s, validUnits)
					}
				}
			}

			if end < 0 {
				break
			}

			start = end + 1
		}

		return nil
	}
}

// parseUnitValue splits one unit value into its optional sign, numeric part and
// unit without allocating. It mirrors MJML's permissive number part, which
// accepts any sequence of digits, dots and commas, and requires the unit to
// consist of ASCII letters and percent signs up to the end of the value.
func parseUnitValue(s string) (negative bool, unit string, ok bool) {
	i := 0
	if i < len(s) && s[i] == '-' {
		negative = true
		i++
	}

	numberStart := i
	for i < len(s) {
		c := s[i]
		if (c < '0' || c > '9') && c != '.' && c != ',' {
			break
		}

		i++
	}

	if i == numberStart {
		return false, "", false
	}

	unitStart := i
	for i < len(s) {
		c := s[i]
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && c != '%' {
			return false, "", false
		}

		i++
	}

	return negative, s[unitStart:], true
}

// The type validators are stateless, so each type is one shared value rather
// than a fresh closure per call. validateType hands them out; the hoisted
// AllowedAttributes tables then hold them for the process lifetime.
var (
	acceptAllValue AttributeValidator = func(string) error { return nil }

	booleanValue AttributeValidator = func(value string) error {
		if !strings.EqualFold(value, "true") && !strings.EqualFold(value, "false") {
			return fmt.Errorf("value %q is not a boolean", value)
		}

		return nil
	}

	numberValue AttributeValidator = func(value string) error {
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("value %q is not a valid number", value)
		}

		return nil
	}

	// integerValue mirrors MJML's integer type, which is an unanchored \d+: the
	// value is accepted when it contains at least one ASCII digit.
	integerValue AttributeValidator = func(value string) error {
		for i := 0; i < len(value); i++ {
			if c := value[i]; c >= '0' && c <= '9' {
				return nil
			}
		}

		return fmt.Errorf("value %q is not an integer", value)
	}
)

func validateType(expectedType string) AttributeValidator {
	switch expectedType {
	case "boolean":
		return booleanValue
	case "number":
		return numberValue
	case "integer":
		return integerValue
	default:
		// "string" accepts every value; MJML's unknown types are permissive too.
		return acceptAllValue
	}
}

package component

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

var ErrValidation = errors.New("failed validation")

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

		if !strings.HasPrefix(value, "#") || (len(value) != 4 && len(value) != 7) {
			return fmt.Errorf("%w: %s not a valid color string", ErrValidation, value)
		}

		return nil
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

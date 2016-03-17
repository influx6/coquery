package utils

import (
	"regexp"
	"strconv"
)

// intdigits defines a regexp for matching ints.
var intdigits = regexp.MustCompile("[\\d]+")

// fldigits defines a regexp for matching floats.
var fldigits = regexp.MustCompile("[\\d\\.]+")

// IsDigits returns true/false if the string is all digits.
func IsDigits(fl string) bool {
	return intdigits.MatchString(fl) || fldigits.MatchString(fl)
}

// nodigits defines a regexp for matching non-digits.
var nodigits = regexp.MustCompile("[^\\d\\.]+")

// DigitsOnly removes all non-digits characters in a string.
func DigitsOnly(fl string) string {
	return nodigits.ReplaceAllString(fl, "")
}

// ParseFloat parses a string into a float if fails returns the default value 0.
func ParseFloat(fl string) (float64, bool) {
	if fldigits.MatchString(fl) {
		fll, _ := strconv.ParseFloat(DigitsOnly(fl), 64)
		return fll, true
	}
	return 0, false
}

// ParseInt parses a string into a int if fails returns the default value 0.
func ParseInt(fl string) (int, bool) {
	if intdigits.MatchString(fl) {
		fll, _ := strconv.Atoi(DigitsOnly(fl))
		return fll, true
	}
	return 0, false
}

// ParseIntBase16 parses a string into a int using base16.
func ParseIntBase16(fl string) int {
	fll, _ := strconv.ParseInt(fl, 16, 64)
	return int(fll)
}

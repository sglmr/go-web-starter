package validator

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/exp/constraints"
)

//	Validaton checks

var RgxEmail = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// NotBlank checks if a string is not empty or composed only of whitespace.
func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// MinRunes checks if a string contains at least a minimum number of runes (characters).
func MinRunes(value string, n int) bool {
	return utf8.RuneCountInString(value) >= n
}

// MaxRunes checks if a string contains no more than a maximum number of runes.
func MaxRunes(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

// Between checks if a value is within a specified range (inclusive). It's a
// generic function that can be used with any ordered type, such as numbers or
// strings.
func Between[T constraints.Ordered](value, min, max T) bool {
	return value >= min && value <= max
}

// Matches checks if a string matches a given regular expression.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// In checks if a value is present in a given list of allowed values.
func In[T comparable](value T, safelist ...T) bool {
	for i := range safelist {
		if value == safelist[i] {
			return true
		}
	}
	return false
}

// AllIn checks if all values in a slice are present in a given list of allowed
// values.
func AllIn[T comparable](values []T, safelist ...T) bool {
	for i := range values {
		if !In(values[i], safelist...) {
			return false
		}
	}
	return true
}

// NotIn returns true when the value is not in the blocklist of values.
func NotIn[T comparable](value T, blocklist ...T) bool {
	for i := range blocklist {
		if value == blocklist[i] {
			return false
		}
	}
	return true
}

// NoDuplicates returns true when there are no duplicates in the values
func NoDuplicates[T comparable](values []T) bool {
	uniqueValues := make(map[T]bool)

	for _, value := range values {
		uniqueValues[value] = true
	}

	return len(values) == len(uniqueValues)
}

// IsEmail returns true when the string value passes an email regular expression pattern.
func IsEmail(value string) bool {
	if len(value) > 254 {
		return false
	}

	return RgxEmail.MatchString(value)
}

// IsURL returns true if the value is a valid URL
func IsURL(value string) bool {
	u, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}

	return u.Scheme != "" && u.Host != ""
}

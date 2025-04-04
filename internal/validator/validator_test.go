package validator

import (
	"regexp"
	"testing"
)

func TestValidatorValid(t *testing.T) {
	tests := []struct {
		name     string
		errors   map[string]string
		expected bool
	}{
		{
			name:     "no errors",
			errors:   map[string]string{},
			expected: true,
		},
		{
			name:     "nil errors",
			errors:   nil,
			expected: true,
		},
		{
			name:     "with errors",
			errors:   map[string]string{"field": "error message"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Validator{Errors: tt.errors}
			if got := v.Valid(); got != tt.expected {
				t.Errorf("Valid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidatorHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		errors   map[string]string
		expected bool
	}{
		{
			name:     "no errors",
			errors:   map[string]string{},
			expected: false,
		},
		{
			name:     "nil errors",
			errors:   nil,
			expected: false,
		},
		{
			name:     "with errors",
			errors:   map[string]string{"field": "error message"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Validator{Errors: tt.errors}
			if got := v.HasErrors(); got != tt.expected {
				t.Errorf("HasErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidatorAddError(t *testing.T) {
	tests := []struct {
		name         string
		initialState map[string]string
		key          string
		message      string
		wantState    map[string]string
	}{
		{
			name:         "add to nil map",
			initialState: nil,
			key:          "field",
			message:      "error message",
			wantState:    map[string]string{"field": "error message"},
		},
		{
			name:         "add to empty map",
			initialState: map[string]string{},
			key:          "field",
			message:      "error message",
			wantState:    map[string]string{"field": "error message"},
		},
		{
			name:         "add new key to map with existing errors",
			initialState: map[string]string{"existing": "error"},
			key:          "field",
			message:      "error message",
			wantState:    map[string]string{"existing": "error", "field": "error message"},
		},
		{
			name:         "don't overwrite existing key",
			initialState: map[string]string{"field": "original error"},
			key:          "field",
			message:      "new error message",
			wantState:    map[string]string{"field": "original error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Validator{Errors: tt.initialState}
			v.AddError(tt.key, tt.message)

			// Check if maps have same length
			if len(v.Errors) != len(tt.wantState) {
				t.Errorf("After AddError(), Errors map has %d entries, want %d", len(v.Errors), len(tt.wantState))
			}

			// Check if all expected keys exist with correct values
			for k, wantMsg := range tt.wantState {
				if gotMsg, exists := v.Errors[k]; !exists {
					t.Errorf("After AddError(), Errors map is missing key %q", k)
				} else if gotMsg != wantMsg {
					t.Errorf("After AddError(), Errors[%q] = %q, want %q", k, gotMsg, wantMsg)
				}
			}
		})
	}
}

func TestValidatorCheck(t *testing.T) {
	tests := []struct {
		name         string
		initialState map[string]string
		key          string
		ok           bool
		message      string
		wantState    map[string]string
	}{
		{
			name:         "check passes, nil map",
			initialState: nil,
			key:          "field",
			ok:           true,
			message:      "error message",
			wantState:    map[string]string{},
		},
		{
			name:         "check passes, existing map",
			initialState: map[string]string{"existing": "error"},
			key:          "field",
			ok:           true,
			message:      "error message",
			wantState:    map[string]string{"existing": "error"},
		},
		{
			name:         "check fails, nil map",
			initialState: nil,
			key:          "field",
			ok:           false,
			message:      "error message",
			wantState:    map[string]string{"field": "error message"},
		},
		{
			name:         "check fails, existing map",
			initialState: map[string]string{"existing": "error"},
			key:          "field",
			ok:           false,
			message:      "error message",
			wantState:    map[string]string{"existing": "error", "field": "error message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Validator{Errors: tt.initialState}
			v.Check(tt.key, tt.ok, tt.message)

			// Check if maps have same length
			if len(v.Errors) != len(tt.wantState) {
				t.Errorf("After Check(), Errors map has %d entries, want %d", len(v.Errors), len(tt.wantState))
			}

			// Check if all expected keys exist with correct values
			for k, wantMsg := range tt.wantState {
				if gotMsg, exists := v.Errors[k]; !exists {
					t.Errorf("After Check(), Errors map is missing key %q", k)
				} else if gotMsg != wantMsg {
					t.Errorf("After Check(), Errors[%q] = %q, want %q", k, gotMsg, wantMsg)
				}
			}
		})
	}
}

func TestNotBlank(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "empty string",
			value:    "",
			expected: false,
		},
		{
			name:     "whitespace only",
			value:    "   \t\n",
			expected: false,
		},
		{
			name:     "non-empty string",
			value:    "hello",
			expected: true,
		},
		{
			name:     "string with whitespace",
			value:    "  hello  ",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NotBlank(tt.value); got != tt.expected {
				t.Errorf("NotBlank(%q) = %v, want %v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestMinRunes(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		n        int
		expected bool
	}{
		{
			name:     "empty string, n=0",
			value:    "",
			n:        0,
			expected: true,
		},
		{
			name:     "empty string, n=1",
			value:    "",
			n:        1,
			expected: false,
		},
		{
			name:     "string with exactly n runes",
			value:    "hello",
			n:        5,
			expected: true,
		},
		{
			name:     "string with more than n runes",
			value:    "hello world",
			n:        5,
			expected: true,
		},
		{
			name:     "string with less than n runes",
			value:    "hi",
			n:        5,
			expected: false,
		},
		{
			name:     "string with multibyte characters",
			value:    "こんにちは", // 5 runes
			n:        5,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MinRunes(tt.value, tt.n); got != tt.expected {
				t.Errorf("MinRunes(%q, %d) = %v, want %v", tt.value, tt.n, got, tt.expected)
			}
		})
	}
}

func TestMaxRunes(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		n        int
		expected bool
	}{
		{
			name:     "empty string, n=0",
			value:    "",
			n:        0,
			expected: true,
		},
		{
			name:     "empty string, n=1",
			value:    "",
			n:        1,
			expected: true,
		},
		{
			name:     "string with exactly n runes",
			value:    "hello",
			n:        5,
			expected: true,
		},
		{
			name:     "string with less than n runes",
			value:    "hi",
			n:        5,
			expected: true,
		},
		{
			name:     "string with more than n runes",
			value:    "hello world",
			n:        5,
			expected: false,
		},
		{
			name:     "string with multibyte characters",
			value:    "こんにちは世界", // 7 runes
			n:        5,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaxRunes(tt.value, tt.n); got != tt.expected {
				t.Errorf("MaxRunes(%q, %d) = %v, want %v", tt.value, tt.n, got, tt.expected)
			}
		})
	}
}

func TestBetween(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		min      int
		max      int
		expected bool
	}{
		{
			name:     "value equal to min",
			value:    5,
			min:      5,
			max:      10,
			expected: true,
		},
		{
			name:     "value equal to max",
			value:    10,
			min:      5,
			max:      10,
			expected: true,
		},
		{
			name:     "value between min and max",
			value:    7,
			min:      5,
			max:      10,
			expected: true,
		},
		{
			name:     "value less than min",
			value:    3,
			min:      5,
			max:      10,
			expected: false,
		},
		{
			name:     "value greater than max",
			value:    15,
			min:      5,
			max:      10,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Between(tt.value, tt.min, tt.max); got != tt.expected {
				t.Errorf("Between(%d, %d, %d) = %v, want %v", tt.value, tt.min, tt.max, got, tt.expected)
			}
		})
	}

	// Test with float64
	t.Run("float64 values", func(t *testing.T) {
		if got := Between(7.5, 5.0, 10.0); got != true {
			t.Errorf("Between(7.5, 5.0, 10.0) = %v, want true", got)
		}

		if got := Between(3.5, 5.0, 10.0); got != false {
			t.Errorf("Between(3.5, 5.0, 10.0) = %v, want false", got)
		}
	})

	// Test with string
	t.Run("string values", func(t *testing.T) {
		if got := Between("b", "a", "c"); got != true {
			t.Errorf("Between(\"b\", \"a\", \"c\") = %v, want true", got)
		}

		if got := Between("d", "a", "c"); got != false {
			t.Errorf("Between(\"d\", \"a\", \"c\") = %v, want false", got)
		}
	})
}

func TestMatches(t *testing.T) {
	rxDigitsOnly := regexp.MustCompile(`^\d+$`)

	tests := []struct {
		name     string
		value    string
		rx       *regexp.Regexp
		expected bool
	}{
		{
			name:     "matching pattern",
			value:    "12345",
			rx:       rxDigitsOnly,
			expected: true,
		},
		{
			name:     "non-matching pattern",
			value:    "abc123",
			rx:       rxDigitsOnly,
			expected: false,
		},
		{
			name:     "empty string",
			value:    "",
			rx:       rxDigitsOnly,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Matches(tt.value, tt.rx); got != tt.expected {
				t.Errorf("Matches(%q, %v) = %v, want %v", tt.value, tt.rx, got, tt.expected)
			}
		})
	}
}

func TestIn(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		safelist []string
		expected bool
	}{
		{
			name:     "value in safelist",
			value:    "apple",
			safelist: []string{"banana", "apple", "orange"},
			expected: true,
		},
		{
			name:     "value not in safelist",
			value:    "grape",
			safelist: []string{"banana", "apple", "orange"},
			expected: false,
		},
		{
			name:     "empty safelist",
			value:    "apple",
			safelist: []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := In(tt.value, tt.safelist...); got != tt.expected {
				t.Errorf("In(%q, %v) = %v, want %v", tt.value, tt.safelist, got, tt.expected)
			}
		})
	}

	// Test with int type
	t.Run("int values", func(t *testing.T) {
		if got := In(5, 1, 3, 5, 7); got != true {
			t.Errorf("In(5, [1,3,5,7]) = %v, want true", got)
		}

		if got := In(6, 1, 3, 5, 7); got != false {
			t.Errorf("In(6, [1,3,5,7]) = %v, want false", got)
		}
	})
}

func TestAllIn(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		safelist []string
		expected bool
	}{
		{
			name:     "all values in safelist",
			values:   []string{"apple", "banana"},
			safelist: []string{"banana", "apple", "orange"},
			expected: true,
		},
		{
			name:     "some values not in safelist",
			values:   []string{"apple", "grape"},
			safelist: []string{"banana", "apple", "orange"},
			expected: false,
		},
		{
			name:     "empty values",
			values:   []string{},
			safelist: []string{"banana", "apple", "orange"},
			expected: true,
		},
		{
			name:     "empty safelist",
			values:   []string{"apple", "banana"},
			safelist: []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AllIn(tt.values, tt.safelist...); got != tt.expected {
				t.Errorf("AllIn(%v, %v) = %v, want %v", tt.values, tt.safelist, got, tt.expected)
			}
		})
	}
}

func TestNotIn(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		blocklist []string
		expected  bool
	}{
		{
			name:      "value not in blocklist",
			value:     "grape",
			blocklist: []string{"banana", "apple", "orange"},
			expected:  true,
		},
		{
			name:      "value in blocklist",
			value:     "apple",
			blocklist: []string{"banana", "apple", "orange"},
			expected:  false,
		},
		{
			name:      "empty blocklist",
			value:     "apple",
			blocklist: []string{},
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NotIn(tt.value, tt.blocklist...); got != tt.expected {
				t.Errorf("NotIn(%q, %v) = %v, want %v", tt.value, tt.blocklist, got, tt.expected)
			}
		})
	}
}

func TestNoDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected bool
	}{
		{
			name:     "no duplicates",
			values:   []string{"apple", "banana", "orange"},
			expected: true,
		},
		{
			name:     "with duplicates",
			values:   []string{"apple", "banana", "apple", "orange"},
			expected: false,
		},
		{
			name:     "empty slice",
			values:   []string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NoDuplicates(tt.values); got != tt.expected {
				t.Errorf("NoDuplicates(%v) = %v, want %v", tt.values, got, tt.expected)
			}
		})
	}

	// Test with int type
	t.Run("int values", func(t *testing.T) {
		if got := NoDuplicates([]int{1, 2, 3, 4}); got != true {
			t.Errorf("NoDuplicates([1,2,3,4]) = %v, want true", got)
		}

		if got := NoDuplicates([]int{1, 2, 3, 2, 4}); got != false {
			t.Errorf("NoDuplicates([1,2,3,2,4]) = %v, want false", got)
		}
	})
}

func TestIsEmail(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "valid email",
			value:    "user@example.com",
			expected: true,
		},
		{
			name:     "valid email with subdomain",
			value:    "user@sub.example.com",
			expected: true,
		},
		{
			name:     "valid email with plus",
			value:    "user+tag@example.com",
			expected: true,
		},
		{
			name:     "invalid email - no @",
			value:    "userexample.com",
			expected: false,
		},
		{
			name:     "invalid email - no domain",
			value:    "user@",
			expected: false,
		},
		{
			name:     "invalid email - no username",
			value:    "@example.com",
			expected: false,
		},
		{
			name:     "invalid email - spaces",
			value:    "user @example.com",
			expected: false,
		},
		{
			name:     "email exceeding max length",
			value:    "a" + string(make([]byte, 250)) + "@example.com", // over 254 chars
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEmail(tt.value); got != tt.expected {
				t.Errorf("IsEmail(%q) = %v, want %v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "valid URL with http",
			value:    "http://example.com",
			expected: true,
		},
		{
			name:     "valid URL with https",
			value:    "https://example.com",
			expected: true,
		},
		{
			name:     "valid URL with path",
			value:    "https://example.com/path",
			expected: true,
		},
		{
			name:     "valid URL with query",
			value:    "https://example.com/path?query=value",
			expected: true,
		},
		{
			name:     "invalid URL - no scheme",
			value:    "example.com",
			expected: false,
		},
		{
			name:     "invalid URL - no host",
			value:    "http://",
			expected: false,
		},
		{
			name:     "invalid URL - malformed",
			value:    "http:/example.com",
			expected: false,
		},
		{
			name:     "invalid URL - spaces",
			value:    "http://example. com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsURL(tt.value); got != tt.expected {
				t.Errorf("IsURL(%q) = %v, want %v", tt.value, got, tt.expected)
			}
		})
	}
}

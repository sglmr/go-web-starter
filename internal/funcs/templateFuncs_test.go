package funcs

import (
	"fmt"
	"html/template"
	"net/url"
	"testing"
	"time"
)

// TestSlugify runs a series of tests on the slugify function
func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Hello_World", "hello_world"},
		{"Hello-World", "hello-world"},
		{"Hello World 123", "hello-world-123"},
		{"Hello   World", "hello---world"},
		{"HELLO world", "hello-world"},
		{"Hello, World!", "hello-world"},
		{"Héllö Wörld", "hll-wrld"},
		{"", ""},
		{"---", "---"},
		{"Special@#$Characters", "specialcharacters"},
		{"Mixed 123 & Symbols!", "mixed-123--symbols"},
		{"  Leading and trailing spaces  ", "--leading-and-trailing-spaces--"},
		{"123 Start with number", "123-start-with-number"},
		{"CamelCaseExample", "camelcaseexample"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got, want := slugify(tc.input), tc.want; got != want {
				t.Errorf("got %q, wanted %q", got, want)
			}
		})
	}
}

// TestFormatTime tests the formatTime function.
func TestFormatTime(t *testing.T) {
	t.Parallel()

	// A known time for consistent test results.
	knownTime := time.Date(2023, 10, 26, 10, 30, 0, 0, time.UTC)

	testCases := []struct {
		name   string
		format string
		time   time.Time
		want   string
	}{
		{
			name:   "RFC3339 format",
			format: time.RFC3339,
			time:   knownTime,
			want:   "2023-10-26T10:30:00Z",
		},
		{
			name:   "Custom date format",
			format: "2006-01-02",
			time:   knownTime,
			want:   "2023-10-26",
		},
		{
			name:   "Custom time format",
			format: "15:04:05",
			time:   knownTime,
			want:   "10:30:00",
		},
		{
			name:   "Empty format string",
			format: "",
			time:   knownTime,
			want:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := formatTime(tc.format, tc.time), tc.want; got != want {
				t.Errorf("formatTime(%q, %v) = %q; want %q", tc.format, tc.time, got, want)
			}
		})
	}
}

// TestSafeHTML tests the safeHTML function.
func TestSafeHTML(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want template.HTML
	}{
		{
			name: "Simple string",
			in:   "Hello, World!",
			want: "Hello, World!",
		},
		{
			name: "String with HTML",
			in:   "<p>This is a paragraph.</p>",
			want: "<p>This is a paragraph.</p>",
		},
		{
			name: "Empty string",
			in:   "",
			want: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := safeHTML(tc.in), tc.want; got != want {
				t.Errorf("safeHTML(%q) = %q; want %q", tc.in, got, want)
			}
		})
	}
}

// TestFormatInt tests the formatInt function.
func TestFormatInt(t *testing.T) {
	testCases := []struct {
		name    string
		in      any
		want    string
		wantErr bool
	}{
		{name: "int", in: 12345, want: "12,345", wantErr: false},
		{name: "int64", in: int64(9876543210), want: "9,876,543,210", wantErr: false},
		{name: "string number", in: "54321", want: "54,321", wantErr: false},
		{name: "zero", in: 0, want: "0", wantErr: false},
		{name: "negative int", in: -1000, want: "-1,000", wantErr: false},
		{name: "invalid string", in: "abc", want: "", wantErr: true},
		{name: "invalid type float", in: 123.45, want: "", wantErr: true},
		{name: "invalid type bool", in: true, want: "", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := formatInt(tc.in)

			if (err != nil) != tc.wantErr {
				t.Fatalf("formatInt(%v) returned error %v; wantErr is %v", tc.in, err, tc.wantErr)
			}

			if !tc.wantErr {
				if want := tc.want; got != want {
					t.Errorf("formatInt(%v) = %q; want %q", tc.in, got, want)
				}
			}
		})
	}
}

// TestFormatFloat tests the formatFloat function.
func TestFormatFloat(t *testing.T) {
	testCases := []struct {
		name string
		f    float64
		dp   int
		want string
	}{
		{name: "2 decimal places", f: 12345.6789, dp: 2, want: "12,345.68"},
		{name: "0 decimal places", f: 12345.6789, dp: 0, want: "12,346"},
		{name: "5 decimal places", f: 123.45, dp: 5, want: "123.45000"},
		{name: "negative float", f: -987.654, dp: 1, want: "-987.7"},
		{name: "zero", f: 0.0, dp: 2, want: "0.00"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := formatFloat(tc.f, tc.dp), tc.want; got != want {
				t.Errorf("formatFloat(%f, %d) = %q; want %q", tc.f, tc.dp, got, want)
			}
		})
	}
}

// TestYesno tests the yesno function.
func TestYesno(t *testing.T) {
	testCases := []struct {
		name string
		in   bool
		want string
	}{
		{name: "true", in: true, want: "Yes"},
		{name: "false", in: false, want: "No"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := yesno(tc.in), tc.want; got != want {
				t.Errorf("yesno(%v) = %q; want %q", tc.in, got, want)
			}
		})
	}
}

// TestURLSetParam tests the urlSetParam function.
func TestURLSetParam(t *testing.T) {
	baseURL, _ := url.Parse("https://example.com/path")

	testCases := []struct {
		name  string
		u     *url.URL
		key   string
		value any
		want  string
	}{
		{
			name:  "add new param to clean url",
			u:     baseURL,
			key:   "q",
			value: "search",
			want:  "https://example.com/path?q=search",
		},
		{
			name:  "add new param to existing query",
			u:     mustParseURL("https://example.com/path?page=1"),
			key:   "sort",
			value: "asc",
			want:  "https://example.com/path?page=1&sort=asc",
		},
		{
			name:  "update existing param",
			u:     mustParseURL("https://example.com/path?page=1"),
			key:   "page",
			value: 2,
			want:  "https://example.com/path?page=2",
		},
		{
			name:  "value with special characters",
			u:     baseURL,
			key:   "redirect",
			value: "https://other.com/login?a=b",
			want:  "https://example.com/path?redirect=https%3A%2F%2Fother.com%2Flogin%3Fa%3Db",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure original URL is not mutated
			originalURLStr := tc.u.String()

			gotURL := urlSetParam(tc.u, tc.key, tc.value)

			if got, want := gotURL.String(), tc.want; got != want {
				t.Errorf("urlSetParam() got = %q, want %q", got, want)
			}

			if got, want := tc.u.String(), originalURLStr; got != want {
				t.Errorf("original URL was mutated: got %q, want %q", got, want)
			}
		})
	}
}

// TestURLDelParam tests the urlDelParam function.
func TestURLDelParam(t *testing.T) {
	testCases := []struct {
		name string
		u    *url.URL
		key  string
		want string
	}{
		{
			name: "delete existing param",
			u:    mustParseURL("https://example.com/path?page=1&sort=asc"),
			key:  "sort",
			want: "https://example.com/path?page=1",
		},
		{
			name: "delete the only param",
			u:    mustParseURL("https://example.com/path?q=search"),
			key:  "q",
			want: "https://example.com/path",
		},
		{
			name: "delete non-existent param",
			u:    mustParseURL("https://example.com/path?page=1"),
			key:  "filter",
			want: "https://example.com/path?page=1",
		},
		{
			name: "delete from url with no params",
			u:    mustParseURL("https://example.com/path"),
			key:  "any",
			want: "https://example.com/path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure original URL is not mutated
			originalURLStr := tc.u.String()

			gotURL := urlDelParam(tc.u, tc.key)

			if got, want := gotURL.String(), tc.want; got != want {
				t.Errorf("urlDelParam() got = %q, want %q", got, want)
			}

			if got, want := tc.u.String(), originalURLStr; got != want {
				t.Errorf("original URL was mutated: got %q, want %q", got, want)
			}
		})
	}
}

// TestToInt64 tests the toInt64 function.
func TestToInt64(t *testing.T) {
	testCases := []struct {
		name    string
		in      any
		want    int64
		wantErr bool
	}{
		{name: "int", in: 123, want: 123, wantErr: false},
		{name: "int8", in: int8(123), want: 123, wantErr: false},
		{name: "int16", in: int16(123), want: 123, wantErr: false},
		{name: "int32", in: int32(123), want: 123, wantErr: false},
		{name: "int64", in: int64(123), want: 123, wantErr: false},
		{name: "uint", in: uint(123), want: 123, wantErr: false},
		{name: "uint8", in: uint8(123), want: 123, wantErr: false},
		{name: "uint16", in: uint16(123), want: 123, wantErr: false},
		{name: "uint32", in: uint32(123), want: 123, wantErr: false},
		{name: "string number", in: "456", want: 456, wantErr: false},
		{name: "negative string number", in: "-789", want: -789, wantErr: false},
		{name: "invalid string", in: "abc", want: 0, wantErr: true},
		{name: "unsupported type float", in: 123.45, want: 0, wantErr: true},
		{name: "unsupported type bool", in: false, want: 0, wantErr: true},
		{name: "unsupported type nil", in: nil, want: 0, wantErr: true},
		{name: "error message check", in: 1.2, want: 0, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := toInt64(tc.in)

			if (err != nil) != tc.wantErr {
				t.Fatalf("toInt64(%v) returned error %v; wantErr is %v", tc.in, err, tc.wantErr)
			}

			if !tc.wantErr {
				if want := tc.want; got != want {
					t.Errorf("toInt64(%v) = %d; want %d", tc.in, got, want)
				}
			} else if tc.name == "error message check" {
				// Also check the error message for one case to be thorough
				expectedErr := fmt.Sprintf("unable to convert type %T to int", tc.in)
				if got, want := err.Error(), expectedErr; got != want {
					t.Errorf("toInt64(%v) error message = %q; want %q", tc.in, got, want)
				}
			}
		})
	}
}

// mustParseURL is a helper function for tests that require a valid *url.URL.
// It panics if the URL cannot be parsed, which is acceptable in a test setup.
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(fmt.Sprintf("failed to parse URL %q: %v", rawURL, err))
	}
	return u
}

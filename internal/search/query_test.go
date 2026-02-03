package search

import "testing"

func TestIsSimplePattern(t *testing.T) {
	tests := []struct {
		pattern string
		want    bool
	}{
		{"hello", true},
		{"hello world", true},
		{"hello-world", true},
		{"hello_world", true},
		{"hello123", true},
		{"", true},

		// Regex metacharacters
		{"hello.*world", false},
		{"hello.+world", false},
		{"[abc]", false},
		{"(foo|bar)", false},
		{"foo{2}", false},
		{"hello\\d", false},
		{"^start", false},
		{"end$", false},
		{"a+b", false},
		{"a?b", false},
	}

	for _, tt := range tests {
		got := IsSimplePattern(tt.pattern)
		if got != tt.want {
			t.Errorf("IsSimplePattern(%q) = %v, want %v", tt.pattern, got, tt.want)
		}
	}
}

func TestEscapeQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"hello world", "hello world"},
		{"hello.world", "hello\\.world"},
		{"user@host", "user\\@host"},
		{"price$100", "price\\$100"},
		{"a+b=c", "a\\+b\\=c"},
	}

	for _, tt := range tests {
		got := EscapeQuery(tt.input)
		if got != tt.want {
			t.Errorf("EscapeQuery(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

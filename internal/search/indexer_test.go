package search

import "testing"

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"empty", "", false},
		{"text", "hello world\nfoo bar", false},
		{"binary", "hello\x00world", true},
		{"binary at start", "\x00hello", true},
		{"long text", string(make([]byte, 1000)), true}, // null bytes
		{"long text no null", func() string {
			b := make([]byte, 1000)
			for i := range b {
				b[i] = 'a'
			}
			return string(b)
		}(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinary(tt.content)
			if got != tt.want {
				t.Errorf("isBinary(%q...) = %v, want %v", tt.content[:min(10, len(tt.content))], got, tt.want)
			}
		})
	}
}

func TestParentDir(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/foo/bar.txt", "/foo"},
		{"/bar.txt", "/"},
		{"/a/b/c/d.txt", "/a/b/c"},
		{"/", "/"},
	}

	for _, tt := range tests {
		got := parentDir(tt.path)
		if got != tt.want {
			t.Errorf("parentDir(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

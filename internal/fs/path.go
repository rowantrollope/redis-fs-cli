package fs

import (
	"path"
	"strings"
)

// NormalizePath converts a path to a clean, absolute form.
// It resolves ".", "..", multiple slashes, and trailing slashes (except for root).
func NormalizePath(p string) string {
	if p == "" {
		return "/"
	}
	// Clean the path (resolves . and ..)
	cleaned := path.Clean(p)
	if cleaned == "." {
		return "/"
	}
	// Ensure leading slash
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned
}

// ResolvePath resolves a potentially relative path against the current working directory.
func ResolvePath(cwd, p string) string {
	if p == "" {
		return NormalizePath(cwd)
	}
	if strings.HasPrefix(p, "/") {
		return NormalizePath(p)
	}
	return NormalizePath(cwd + "/" + p)
}

// ParentPath returns the parent directory of the given path.
// Returns "/" for root and first-level paths.
func ParentPath(p string) string {
	p = NormalizePath(p)
	if p == "/" {
		return "/"
	}
	parent := path.Dir(p)
	if parent == "." {
		return "/"
	}
	return parent
}

// BaseName returns the final component of the path.
func BaseName(p string) string {
	p = NormalizePath(p)
	if p == "/" {
		return "/"
	}
	return path.Base(p)
}

// SplitPath returns the parent directory and the base name.
func SplitPath(p string) (string, string) {
	return ParentPath(p), BaseName(p)
}

// JoinPath joins path components into a normalized path.
func JoinPath(parts ...string) string {
	return NormalizePath(strings.Join(parts, "/"))
}

// IsRoot returns true if the path is the root directory.
func IsRoot(p string) bool {
	return NormalizePath(p) == "/"
}

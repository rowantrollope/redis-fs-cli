package cli

import (
	"fmt"
	"strings"
)

const maxPromptPathLen = 30

// BuildPrompt generates the dynamic prompt string.
// Format: redis-fs:volume:path>
func BuildPrompt(volume, cwd string, color bool) string {
	path := truncatePath(cwd, maxPromptPathLen)
	if color {
		// Green prompt
		return fmt.Sprintf("\033[32mredis-fs:%s:%s>\033[0m ", volume, path)
	}
	return fmt.Sprintf("redis-fs:%s:%s> ", volume, path)
}

// truncatePath shortens a path if it exceeds maxLen.
// e.g., /very/long/nested/path → /.../nested/path
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}

	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}

	// Try to keep the last 2 components
	suffix := parts[len(parts)-2] + "/" + parts[len(parts)-1]
	truncated := "/.../" + suffix
	if len(truncated) <= maxLen {
		return truncated
	}

	// Just keep the last component
	return "/.../" + parts[len(parts)-1]
}

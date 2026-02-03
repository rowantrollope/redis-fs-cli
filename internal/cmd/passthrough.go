package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// handlePassthrough executes a command via redis-cli subprocess.
func (r *Router) handlePassthrough(ctx context.Context, tokens []string) error {
	redisCLI, err := exec.LookPath("redis-cli")
	if err != nil {
		return fmt.Errorf("redis-cli not found on PATH (exit code 127)")
	}

	args := r.Config.RedisCLIArgs()
	args = append(args, tokens...)

	cmd := exec.CommandContext(ctx, redisCLI, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = r.Formatter.Writer
	cmd.Stderr = r.Formatter.ErrWriter

	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("redis-cli exited with code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("redis-cli: %w", err)
	}
	return nil
}

// handlePassthroughRaw executes raw tokens via redis-cli and returns output as string.
func (r *Router) handlePassthroughRaw(ctx context.Context, tokens []string) (string, error) {
	redisCLI, err := exec.LookPath("redis-cli")
	if err != nil {
		return "", fmt.Errorf("redis-cli not found on PATH")
	}

	args := r.Config.RedisCLIArgs()
	args = append(args, tokens...)

	cmd := exec.CommandContext(ctx, redisCLI, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("redis-cli: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

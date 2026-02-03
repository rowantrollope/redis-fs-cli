package cmd

import (
	"context"
	"fmt"
)

func (r *Router) handleClear(ctx context.Context, args []string) error {
	// ANSI escape to clear screen and move cursor to top-left
	fmt.Fprint(r.Formatter.Writer, "\033[2J\033[H")
	return nil
}

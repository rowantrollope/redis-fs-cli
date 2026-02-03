package cmd

import (
	"context"
	"fmt"
)

func (r *Router) handleChmod(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("chmod: missing operand")
	}

	mode := args[0]
	for _, arg := range args[1:] {
		path := r.ResolvePath(arg)
		if err := r.Client.Chmod(ctx, path, mode); err != nil {
			return err
		}
	}
	return nil
}

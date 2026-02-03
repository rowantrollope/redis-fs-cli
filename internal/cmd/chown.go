package cmd

import (
	"context"
	"fmt"
)

func (r *Router) handleChown(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("chown: missing operand")
	}

	owner := args[0]
	for _, arg := range args[1:] {
		path := r.ResolvePath(arg)
		if err := r.Client.Chown(ctx, path, owner); err != nil {
			return err
		}
	}
	return nil
}

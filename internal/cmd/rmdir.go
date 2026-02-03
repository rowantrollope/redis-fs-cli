package cmd

import (
	"context"
	"fmt"
)

func (r *Router) handleRmdir(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("rmdir: missing operand")
	}

	for _, arg := range args {
		path := r.ResolvePath(arg)
		if err := r.Client.Rmdir(ctx, path); err != nil {
			return err
		}
	}
	return nil
}

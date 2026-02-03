package cmd

import (
	"context"
	"fmt"
)

func (r *Router) handleTouch(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("touch: missing file operand")
	}

	for _, arg := range args {
		path := r.ResolvePath(arg)
		if err := r.Client.Touch(ctx, path); err != nil {
			return err
		}
	}
	return nil
}

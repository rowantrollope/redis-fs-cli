package cmd

import (
	"context"
	"fmt"
)

func (r *Router) handleMv(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("mv: missing operand")
	}

	src := r.ResolvePath(args[0])
	dst := r.ResolvePath(args[1])

	return r.Client.Move(ctx, src, dst)
}

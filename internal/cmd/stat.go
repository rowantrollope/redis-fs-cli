package cmd

import (
	"context"
	"fmt"
)

func (r *Router) handleStat(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("stat: missing operand")
	}

	for _, arg := range args {
		path := r.ResolvePath(arg)
		meta, err := r.Client.Stat(ctx, path)
		if err != nil {
			return err
		}
		if meta == nil {
			return fmt.Errorf("stat: cannot stat '%s': No such file or directory", path)
		}
		r.Formatter.PrintStat(path, meta)
	}
	return nil
}

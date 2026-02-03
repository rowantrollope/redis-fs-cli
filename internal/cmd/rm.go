package cmd

import (
	"context"
	"fmt"

	flag "github.com/spf13/pflag"
)

func (r *Router) handleRm(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("rm", flag.ContinueOnError)
	recursive := fs.BoolP("recursive", "r", false, "Remove directories and their contents recursively")
	force := fs.BoolP("force", "f", false, "Ignore nonexistent files")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("rm: missing operand")
	}

	for _, arg := range fs.Args() {
		path := r.ResolvePath(arg)

		if *recursive {
			err := r.Client.RemoveRecursive(ctx, path)
			if err != nil && !*force {
				return err
			}
		} else {
			err := r.Client.Remove(ctx, path)
			if err != nil && !*force {
				return err
			}
		}
	}
	return nil
}

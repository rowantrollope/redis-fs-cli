package cmd

import (
	"context"
	"fmt"

	flag "github.com/spf13/pflag"
)

func (r *Router) handleLn(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("ln", flag.ContinueOnError)
	symbolic := fs.BoolP("symbolic", "s", false, "Create a symbolic link")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if !*symbolic {
		return fmt.Errorf("ln: hard links not supported; use ln -s")
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("ln: missing operand")
	}

	target := fs.Arg(0)
	linkPath := r.ResolvePath(fs.Arg(1))

	return r.Client.Symlink(ctx, target, linkPath)
}

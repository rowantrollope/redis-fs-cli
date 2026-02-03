package cmd

import (
	"context"
	"fmt"

	flag "github.com/spf13/pflag"
)

func (r *Router) handleCp(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("cp", flag.ContinueOnError)
	recursive := fs.BoolP("recursive", "r", false, "Copy directories recursively")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("cp: missing operand")
	}

	src := r.ResolvePath(fs.Arg(0))
	dst := r.ResolvePath(fs.Arg(1))

	if *recursive {
		return r.Client.CopyRecursive(ctx, src, dst)
	}
	return r.Client.CopyFile(ctx, src, dst)
}

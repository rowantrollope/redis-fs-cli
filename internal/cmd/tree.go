package cmd

import (
	"context"

	flag "github.com/spf13/pflag"
)

func (r *Router) handleTree(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("tree", flag.ContinueOnError)
	maxDepth := fs.IntP("level", "L", 0, "Max display depth (0 = unlimited)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	path := "."
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	}
	path = r.ResolvePath(path)

	entry, dirCount, fileCount, err := r.Client.Tree(ctx, path, *maxDepth)
	if err != nil {
		return err
	}

	r.Formatter.PrintTree(entry, dirCount, fileCount)
	return nil
}

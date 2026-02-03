package cmd

import (
	"context"

	flag "github.com/spf13/pflag"
)

func (r *Router) handleLs(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	long := fs.BoolP("long", "l", false, "Long listing format")
	all := fs.BoolP("all", "a", false, "Show hidden entries")
	if err := fs.Parse(args); err != nil {
		return err
	}

	path := "."
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	}
	path = r.ResolvePath(path)

	if *long {
		entries, err := r.Client.ReadDirWithMeta(ctx, path)
		if err != nil {
			return err
		}
		r.Formatter.PrintLsLong(entries, *all)
	} else {
		entries, err := r.Client.ReadDirWithMeta(ctx, path)
		if err != nil {
			return err
		}
		r.Formatter.PrintLs(entries, *all)
	}
	return nil
}

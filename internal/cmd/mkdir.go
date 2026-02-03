package cmd

import (
	"context"
	"fmt"

	flag "github.com/spf13/pflag"
)

func (r *Router) handleMkdir(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("mkdir", flag.ContinueOnError)
	parents := fs.BoolP("parents", "p", false, "Create parent directories as needed")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("mkdir: missing operand")
	}

	for _, arg := range fs.Args() {
		path := r.ResolvePath(arg)
		if err := r.Client.Mkdir(ctx, path, *parents); err != nil {
			return err
		}
	}
	return nil
}

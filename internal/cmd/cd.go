package cmd

import (
	"context"
	"fmt"

	"github.com/rowantrollope/redis-fs-cli/internal/fs"
)

func (r *Router) handleCd(ctx context.Context, args []string) error {
	var target string
	if len(args) == 0 {
		target = "/"
	} else if args[0] == "-" {
		if r.State.PrevDir == "" {
			return fmt.Errorf("cd: OLDPWD not set")
		}
		target = r.State.PrevDir
	} else {
		target = r.ResolvePath(args[0])
	}

	target = fs.NormalizePath(target)

	isDir, err := r.Client.IsDir(ctx, target)
	if err != nil {
		return err
	}
	if !isDir {
		// Check if it exists but is not a dir
		exists, err := r.Client.Exists(ctx, target)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("cd: %s: Not a directory", target)
		}
		return fmt.Errorf("cd: %s: No such file or directory", target)
	}

	r.State.PrevDir = r.State.Cwd
	r.State.Cwd = target
	return nil
}

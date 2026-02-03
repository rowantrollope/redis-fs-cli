package cmd

import (
	"context"
	"fmt"
)

func (r *Router) handleFind(ctx context.Context, args []string) error {
	// find uses POSIX-style -name and -type flags (single dash, not pflag-compatible)
	// Parse manually.
	path := "."
	namePattern := ""
	typeFilter := ""

	i := 0
	for i < len(args) {
		switch args[i] {
		case "-name":
			i++
			if i >= len(args) {
				return fmt.Errorf("find: -name requires an argument")
			}
			namePattern = args[i]
		case "-type":
			i++
			if i >= len(args) {
				return fmt.Errorf("find: -type requires an argument")
			}
			typeFilter = args[i]
		default:
			if args[i][0] != '-' && path == "." {
				path = args[i]
			} else {
				return fmt.Errorf("find: unknown option: %s", args[i])
			}
		}
		i++
	}

	path = r.ResolvePath(path)

	entries, err := r.Client.Find(ctx, path, namePattern, typeFilter)
	if err != nil {
		return err
	}

	if r.Formatter.JSON {
		var result []map[string]interface{}
		for _, e := range entries {
			entry := map[string]interface{}{
				"path": e.Path,
				"type": string(e.Meta.Type),
			}
			result = append(result, entry)
		}
		return r.Formatter.PrintJSON(result)
	}

	for _, e := range entries {
		fmt.Fprintln(r.Formatter.Writer, e.Path)
	}
	return nil
}

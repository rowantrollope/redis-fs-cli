package cmd

import (
	"context"
	"fmt"
)

func (r *Router) handleCat(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("cat: missing file operand")
	}

	for _, arg := range args {
		path := r.ResolvePath(arg)
		content, err := r.Client.ReadFile(ctx, path)
		if err != nil {
			return err
		}
		r.Formatter.Printf("%s", content)
		// Add newline if content doesn't end with one
		if len(content) > 0 && content[len(content)-1] != '\n' {
			r.Formatter.Println()
		}
	}
	return nil
}

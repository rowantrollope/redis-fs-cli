package cmd

import (
	"context"
	"strings"
)

func (r *Router) handleEcho(ctx context.Context, args []string) error {
	// Without redirect, just print
	r.Formatter.Println(strings.Join(args, " "))
	return nil
}

func (r *Router) handleEchoRedirect(ctx context.Context, args []string, redirect *Redirect) error {
	content := strings.Join(args, " ")
	path := r.ResolvePath(redirect.Path)

	if redirect.Append {
		return r.Client.AppendFile(ctx, path, content)
	}
	return r.Client.WriteFile(ctx, path, content)
}

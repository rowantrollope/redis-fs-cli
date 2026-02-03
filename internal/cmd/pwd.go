package cmd

import "context"

func (r *Router) handlePwd(ctx context.Context, args []string) error {
	r.Formatter.Println(r.State.Cwd)
	return nil
}

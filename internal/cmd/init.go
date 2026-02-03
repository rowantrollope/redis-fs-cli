package cmd

import "context"

func (r *Router) handleInit(ctx context.Context, args []string) error {
	err := r.Client.Init(ctx)
	if err != nil {
		return err
	}
	r.Formatter.Printf("Volume '%s' initialized\n", r.State.Volume)
	return nil
}

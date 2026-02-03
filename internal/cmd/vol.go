package cmd

import (
	"context"
	"fmt"
	"strings"
)

func (r *Router) handleVol(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("vol: usage: vol list|switch|create|info")
	}

	subcmd := strings.ToLower(args[0])
	subargs := args[1:]

	switch subcmd {
	case "list":
		return r.volList(ctx)
	case "switch":
		if len(subargs) == 0 {
			return fmt.Errorf("vol switch: missing volume name")
		}
		return r.volSwitch(ctx, subargs[0])
	case "create":
		if len(subargs) == 0 {
			return fmt.Errorf("vol create: missing volume name")
		}
		return r.volCreate(ctx, subargs[0])
	case "info":
		return r.volInfo(ctx)
	default:
		return fmt.Errorf("vol: unknown subcommand '%s'", subcmd)
	}
}

func (r *Router) volList(ctx context.Context) error {
	volumes, err := r.Client.ListVolumes(ctx)
	if err != nil {
		return err
	}

	if r.Formatter.JSON {
		return r.Formatter.PrintJSON(volumes)
	}

	for _, vol := range volumes {
		marker := "  "
		if vol == r.State.Volume {
			marker = "* "
		}
		fmt.Fprintf(r.Formatter.Writer, "%s%s\n", marker, vol)
	}
	return nil
}

func (r *Router) volSwitch(ctx context.Context, name string) error {
	// Verify volume root exists
	r.Client.SetVolume(name)
	exists, err := r.Client.Exists(ctx, "/")
	if err != nil {
		r.Client.SetVolume(r.State.Volume)
		return err
	}
	if !exists {
		r.Client.SetVolume(r.State.Volume)
		return fmt.Errorf("vol switch: volume '%s' does not exist (use 'vol create %s')", name, name)
	}

	r.State.Volume = name
	r.State.Cwd = "/"
	r.State.PrevDir = ""
	return nil
}

func (r *Router) volCreate(ctx context.Context, name string) error {
	// Save current volume
	prev := r.Client.Volume
	r.Client.SetVolume(name)

	err := r.Client.Init(ctx)
	if err != nil {
		r.Client.SetVolume(prev)
		return err
	}

	// Switch to it
	r.State.Volume = name
	r.State.Cwd = "/"
	r.State.PrevDir = ""

	r.Formatter.Printf("Volume '%s' created and active\n", name)
	return nil
}

func (r *Router) volInfo(ctx context.Context) error {
	if r.Formatter.JSON {
		return r.Formatter.PrintJSON(map[string]string{
			"volume": r.State.Volume,
			"cwd":    r.State.Cwd,
		})
	}

	r.Formatter.Printf("Volume: %s\n", r.State.Volume)
	r.Formatter.Printf("CWD:    %s\n", r.State.Cwd)
	return nil
}

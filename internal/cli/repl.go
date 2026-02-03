package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
	"github.com/rowantrollope/redis-fs-cli/internal/cmd"
	"github.com/rowantrollope/redis-fs-cli/internal/config"
	"github.com/rowantrollope/redis-fs-cli/internal/fs"
	"github.com/rowantrollope/redis-fs-cli/internal/output"
)

// REPL is the interactive read-eval-print loop.
type REPL struct {
	Router    *cmd.Router
	Client    *fs.Client
	Config    *config.Config
	Formatter *output.Formatter
}

// NewREPL creates a new REPL instance.
func NewREPL(router *cmd.Router, client *fs.Client, cfg *config.Config, formatter *output.Formatter) *REPL {
	return &REPL{
		Router:    router,
		Client:    client,
		Config:    cfg,
		Formatter: formatter,
	}
}

// Run starts the interactive REPL loop.
func (r *REPL) Run(ctx context.Context) error {
	completer := NewCompleter(r.Router, r.Client)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          BuildPrompt(r.Router.State.Volume, r.Router.State.Cwd, r.Config.ShouldColor()),
		HistoryFile:     r.Config.HistoryFile,
		HistoryLimit:    10000,
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return fmt.Errorf("readline init: %w", err)
	}
	defer rl.Close()

	for {
		// Update prompt each iteration (cwd may have changed)
		rl.SetPrompt(BuildPrompt(r.Router.State.Volume, r.Router.State.Cwd, r.Config.ShouldColor()))

		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			continue
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle exit/quit
		lower := strings.ToLower(line)
		if lower == "exit" || lower == "quit" {
			return nil
		}

		if execErr := r.Router.Execute(ctx, line); execErr != nil {
			r.Formatter.Errorf("%s\n", execErr)
		}
	}
}

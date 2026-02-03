package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/rowantrollope/redis-fs-cli/internal/config"
	"github.com/rowantrollope/redis-fs-cli/internal/fs"
	"github.com/rowantrollope/redis-fs-cli/internal/output"
)

// State holds the current session state.
type State struct {
	Cwd     string
	PrevDir string
	Volume  string
}

// Router dispatches commands to the appropriate handler.
type Router struct {
	Client    *fs.Client
	Config    *config.Config
	Formatter *output.Formatter
	State     *State
	handlers  map[string]Handler
}

// Handler is a function that handles a command.
type Handler func(ctx context.Context, args []string) error

// NewRouter creates a command router with all registered handlers.
func NewRouter(client *fs.Client, cfg *config.Config, formatter *output.Formatter) *Router {
	r := &Router{
		Client:    client,
		Config:    cfg,
		Formatter: formatter,
		State: &State{
			Cwd:    "/",
			Volume: cfg.Volume,
		},
		handlers: make(map[string]Handler),
	}
	r.registerHandlers()
	return r
}

func (r *Router) registerHandlers() {
	r.handlers["ls"] = r.handleLs
	r.handlers["pwd"] = r.handlePwd
	r.handlers["cd"] = r.handleCd
	r.handlers["mkdir"] = r.handleMkdir
	r.handlers["rmdir"] = r.handleRmdir
	r.handlers["touch"] = r.handleTouch
	r.handlers["cat"] = r.handleCat
	r.handlers["echo"] = r.handleEcho
	r.handlers["rm"] = r.handleRm
	r.handlers["cp"] = r.handleCp
	r.handlers["mv"] = r.handleMv
	r.handlers["stat"] = r.handleStat
	r.handlers["find"] = r.handleFind
	r.handlers["grep"] = r.handleGrep
	r.handlers["ln"] = r.handleLn
	r.handlers["chmod"] = r.handleChmod
	r.handlers["chown"] = r.handleChown
	r.handlers["tree"] = r.handleTree
	r.handlers["vol"] = r.handleVol
	r.handlers["init"] = r.handleInit
	r.handlers["help"] = r.handleHelp
	r.handlers["clear"] = r.handleClear
	r.handlers["index"] = r.handleIndex
	r.handlers["reindex"] = r.handleReindex
	r.handlers["vector-search"] = r.handleVectorSearch
}

// Execute runs a parsed command line.
func (r *Router) Execute(ctx context.Context, line string) error {
	tokens, redirect, err := Tokenize(line)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return nil
	}

	cmd := strings.ToLower(tokens[0])
	args := tokens[1:]

	// Handle echo with redirect specially
	if cmd == "echo" && redirect != nil {
		return r.handleEchoRedirect(ctx, args, redirect)
	}

	// Check for redirect on any other command (not supported)
	if redirect != nil && cmd != "echo" {
		return fmt.Errorf("redirect not supported for command: %s", cmd)
	}

	handler, ok := r.handlers[cmd]
	if ok {
		return handler(ctx, args)
	}

	// Passthrough to redis-cli
	return r.handlePassthrough(ctx, tokens)
}

// IsBuiltin returns true if the command is a built-in FS command.
func (r *Router) IsBuiltin(cmd string) bool {
	_, ok := r.handlers[strings.ToLower(cmd)]
	return ok
}

// CommandNames returns all registered command names.
func (r *Router) CommandNames() []string {
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}

// ResolvePath resolves a path relative to cwd.
func (r *Router) ResolvePath(path string) string {
	if path == "" {
		return r.State.Cwd
	}
	return fs.ResolvePath(r.State.Cwd, path)
}

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/redis/go-redis/v9"
	"github.com/rowantrollope/redis-fs-cli/internal/cli"
	"github.com/rowantrollope/redis-fs-cli/internal/cmd"
	"github.com/rowantrollope/redis-fs-cli/internal/config"
	"github.com/rowantrollope/redis-fs-cli/internal/fs"
	"github.com/rowantrollope/redis-fs-cli/internal/embedding"
	"github.com/rowantrollope/redis-fs-cli/internal/output"
	"github.com/rowantrollope/redis-fs-cli/internal/search"
	flag "github.com/spf13/pflag"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	cfg := config.DefaultConfig()

	// Custom flag set to avoid os.Exit on parse error
	flags := flag.NewFlagSet("redis-fs-cli", flag.ContinueOnError)
	flags.SetInterspersed(false) // Stop parsing at first non-flag arg (the command)
	cfg.RegisterFlags(flags)
	showVersion := flags.Bool("version", false, "Show version and exit")

	// Parse flags; remaining args are the single-command
	if err := flags.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 2
	}
	cfg.Args = flags.Args()

	if *showVersion {
		fmt.Printf("redis-fs-cli %s\n", version)
		return 0
	}

	// Set up color
	if !cfg.ShouldColor() {
		color.NoColor = true
	}

	formatter := output.NewFormatter(cfg.JSON, cfg.ShouldColor())

	// Connect to Redis
	ctx := context.Background()
	rdb := redis.NewClient(cfg.RedisOptions())

	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot connect to Redis at %s: %s\n", cfg.Addr(), err)
		return 1
	}
	defer rdb.Close()

	// Detect search capability
	cfg.SearchAvailable = search.DetectSearch(ctx, rdb)

	// Create FS client
	fsClient := fs.NewClient(rdb, cfg.Volume)

	// Wire search indexer if available
	if cfg.SearchAvailable {
		indexer := search.NewIndexer(rdb, cfg.Volume)
		if cfg.EmbeddingAPIKey != "" {
			embCfg := &embedding.Config{
				APIKey:  cfg.EmbeddingAPIKey,
				BaseURL: cfg.EmbeddingAPIURL,
				Model:   cfg.EmbeddingModel,
				Dim:     cfg.EmbeddingDim,
			}
			indexer.SetEmbedder(embedding.NewClient(embCfg), cfg.EmbeddingDim)
		}
		fsClient.SetObserver(indexer)
	}

	// Auto-init volume root
	if err := fsClient.Init(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize volume: %s\n", err)
		return 1
	}

	// Create router
	router := cmd.NewRouter(fsClient, cfg, formatter)

	// Single-command mode
	if len(cfg.Args) > 0 {
		line := strings.Join(cfg.Args, " ")
		if err := router.Execute(ctx, line); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return 1
		}
		return 0
	}

	// Interactive REPL mode
	repl := cli.NewREPL(router, fsClient, cfg, formatter)
	if err := repl.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 1
	}
	return 0
}

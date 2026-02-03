package cmd

import (
	"context"
	"fmt"

	"github.com/rowantrollope/redis-fs-cli/internal/search"
)

func (r *Router) handleIndex(ctx context.Context, args []string) error {
	if !r.Config.SearchAvailable {
		return fmt.Errorf("index: search not available (requires Redis 8.0+ or Redis Stack with RediSearch)")
	}

	if len(args) == 0 {
		return fmt.Errorf("index: usage: index <status|create|drop|info>")
	}

	mgr := search.NewIndexManager(r.Client.Redis(), r.State.Volume)

	switch args[0] {
	case "status":
		return r.indexStatus(ctx, mgr)
	case "create":
		return r.indexCreate(ctx, mgr)
	case "drop":
		return r.indexDrop(ctx, mgr)
	case "info":
		return r.indexInfo(ctx, mgr)
	default:
		return fmt.Errorf("index: unknown subcommand '%s' (use status, create, drop, or info)", args[0])
	}
}

func (r *Router) indexStatus(ctx context.Context, mgr *search.IndexManager) error {
	exists, err := mgr.IndexExists(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(r.Formatter.Writer, "Search available: yes\n")
	fmt.Fprintf(r.Formatter.Writer, "Index name: %s\n", mgr.IndexName())

	if !exists {
		fmt.Fprintf(r.Formatter.Writer, "Index exists: no\n")
		fmt.Fprintf(r.Formatter.Writer, "Run 'reindex' to create the index and populate it.\n")
		return nil
	}

	fmt.Fprintf(r.Formatter.Writer, "Index exists: yes\n")

	info, err := mgr.IndexInfo(ctx)
	if err != nil {
		return nil
	}

	if numDocs, ok := info["num_docs"]; ok {
		fmt.Fprintf(r.Formatter.Writer, "Indexed documents: %v\n", numDocs)
	}
	if indexing, ok := info["indexing"]; ok {
		fmt.Fprintf(r.Formatter.Writer, "Indexing in progress: %v\n", indexing)
	}

	return nil
}

func (r *Router) indexCreate(ctx context.Context, mgr *search.IndexManager) error {
	exists, err := mgr.IndexExists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("index: index '%s' already exists (use 'index drop' first)", mgr.IndexName())
	}

	withVector := r.Config.EmbeddingAPIKey != ""
	dim := r.Config.EmbeddingDim
	if dim == 0 {
		dim = 1536
	}

	if err := mgr.CreateIndex(ctx, withVector, dim); err != nil {
		return err
	}

	fmt.Fprintf(r.Formatter.Writer, "Created index '%s'", mgr.IndexName())
	if withVector {
		fmt.Fprintf(r.Formatter.Writer, " (with vector field, dim=%d)", dim)
	}
	fmt.Fprintln(r.Formatter.Writer)
	return nil
}

func (r *Router) indexDrop(ctx context.Context, mgr *search.IndexManager) error {
	exists, err := mgr.IndexExists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("index: index '%s' does not exist", mgr.IndexName())
	}

	if err := mgr.DropIndex(ctx); err != nil {
		return err
	}
	fmt.Fprintf(r.Formatter.Writer, "Dropped index '%s'\n", mgr.IndexName())
	return nil
}

func (r *Router) indexInfo(ctx context.Context, mgr *search.IndexManager) error {
	info, err := mgr.IndexInfo(ctx)
	if err != nil {
		return err
	}

	for key, val := range info {
		fmt.Fprintf(r.Formatter.Writer, "%s: %v\n", key, val)
	}
	return nil
}

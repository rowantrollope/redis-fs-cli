package cmd

import (
	"context"
	"fmt"

	"github.com/rowantrollope/redis-fs-cli/internal/embedding"
	"github.com/rowantrollope/redis-fs-cli/internal/search"
	flag "github.com/spf13/pflag"
)

func (r *Router) handleReindex(ctx context.Context, args []string) error {
	if !r.Config.SearchAvailable {
		return fmt.Errorf("reindex: search not available (requires Redis 8.0+ or Redis Stack with RediSearch)")
	}

	fset := flag.NewFlagSet("reindex", flag.ContinueOnError)
	drop := fset.Bool("drop", false, "Drop and recreate index before reindexing")
	status := fset.Bool("status", false, "Show indexing status")
	if err := fset.Parse(args); err != nil {
		return err
	}

	indexer := search.NewIndexer(r.Client.Redis(), r.State.Volume)

	// Configure embedding client if API key is set
	withVector := r.Config.EmbeddingAPIKey != ""
	dim := r.Config.EmbeddingDim
	if dim == 0 {
		dim = 1536
	}

	if withVector {
		embCfg := &embedding.Config{
			APIKey:  r.Config.EmbeddingAPIKey,
			BaseURL: r.Config.EmbeddingAPIURL,
			Model:   r.Config.EmbeddingModel,
			Dim:     r.Config.EmbeddingDim,
		}
		indexer.SetEmbedder(embedding.NewClient(embCfg), dim)
	}

	if *status {
		return r.reindexStatus(ctx, indexer)
	}

	root := "/"
	if fset.NArg() > 0 {
		root = r.ResolvePath(fset.Arg(0))
	}

	opts := search.ReindexOptions{
		Drop: *drop,
		Root: root,
		Progress: func(indexed int, path string) {
			fmt.Fprintf(r.Formatter.Writer, "\r  indexed %d files... %s", indexed, path)
		},
	}

	walker := r.makeFileWalker()

	var count int
	var err error
	if withVector {
		count, err = search.ReindexWithVector(ctx, r.Client.Redis(), indexer, walker, opts, dim)
	} else {
		count, err = search.Reindex(ctx, r.Client.Redis(), indexer, walker, opts)
	}
	if err != nil {
		return err
	}

	fmt.Fprintf(r.Formatter.Writer, "\nIndexed %d files\n", count)
	return nil
}

func (r *Router) reindexStatus(ctx context.Context, indexer *search.Indexer) error {
	mgr := indexer.Manager()
	exists, err := mgr.IndexExists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Fprintf(r.Formatter.Writer, "No index exists. Run 'reindex' to create one.\n")
		return nil
	}

	info, err := mgr.IndexInfo(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(r.Formatter.Writer, "Index: %s\n", mgr.IndexName())
	if numDocs, ok := info["num_docs"]; ok {
		fmt.Fprintf(r.Formatter.Writer, "Documents: %v\n", numDocs)
	}
	if indexing, ok := info["indexing"]; ok {
		fmt.Fprintf(r.Formatter.Writer, "Indexing: %v\n", indexing)
	}
	if hashErr, ok := info["hash_indexing_failures"]; ok {
		fmt.Fprintf(r.Formatter.Writer, "Indexing failures: %v\n", hashErr)
	}

	return nil
}

// makeFileWalker returns a FileWalker that uses the fs.Client to walk the tree.
func (r *Router) makeFileWalker() search.FileWalker {
	return func(ctx context.Context, root string) ([]search.FileEntry, error) {
		entries, err := r.Client.Find(ctx, root, "", "f")
		if err != nil {
			return nil, err
		}

		var files []search.FileEntry
		for _, entry := range entries {
			content, err := r.Client.ReadFile(ctx, entry.Path)
			if err != nil {
				continue
			}
			files = append(files, search.FileEntry{
				Path:    entry.Path,
				Content: content,
				MTime:   entry.Meta.MTime,
				Size:    entry.Meta.Size,
			})
		}
		return files, nil
	}
}

package search

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ReindexOptions controls reindex behavior.
type ReindexOptions struct {
	Drop     bool   // drop and recreate index before reindexing
	Root     string // root path to reindex (default "/")
	Progress func(indexed int, path string)
}

// FileEntry represents a file to be indexed during reindex.
type FileEntry struct {
	Path    string
	Content string
	MTime   int64
	Size    int64
}

// FileWalker is a function that walks the filesystem and returns files.
type FileWalker func(ctx context.Context, root string) ([]FileEntry, error)

// Reindex rebuilds the index for all files under root.
func Reindex(ctx context.Context, rdb *redis.Client, indexer *Indexer, walker FileWalker, opts ReindexOptions) (int, error) {
	mgr := indexer.Manager()

	if opts.Root == "" {
		opts.Root = "/"
	}

	if opts.Drop {
		exists, err := mgr.IndexExists(ctx)
		if err != nil {
			return 0, err
		}
		if exists {
			if err := mgr.DropIndex(ctx); err != nil {
				return 0, fmt.Errorf("reindex: failed to drop index: %w", err)
			}
		}

		// Clean up existing idx keys
		if err := cleanIdxKeys(ctx, rdb, mgr.IdxPrefix()); err != nil {
			return 0, fmt.Errorf("reindex: failed to clean idx keys: %w", err)
		}
	}

	// Ensure index exists
	withVector := false // will be set by caller when embeddings configured
	if err := mgr.EnsureIndex(ctx, withVector, 1536); err != nil {
		return 0, fmt.Errorf("reindex: failed to create index: %w", err)
	}

	// Walk and index
	files, err := walker(ctx, opts.Root)
	if err != nil {
		return 0, fmt.Errorf("reindex: walk failed: %w", err)
	}

	indexed := 0
	for _, f := range files {
		if err := indexer.IndexFile(ctx, f.Path, f.Content, f.MTime, f.Size); err != nil {
			// Log but continue
			if opts.Progress != nil {
				opts.Progress(indexed, fmt.Sprintf("error: %s: %v", f.Path, err))
			}
			continue
		}
		indexed++
		if opts.Progress != nil {
			opts.Progress(indexed, f.Path)
		}
	}

	return indexed, nil
}

// ReindexWithVector is like Reindex but creates the index with vector support
// and generates embeddings for each file using the indexer's embedding client.
func ReindexWithVector(ctx context.Context, rdb *redis.Client, indexer *Indexer, walker FileWalker, opts ReindexOptions, dim int) (int, error) {
	mgr := indexer.Manager()

	if opts.Root == "" {
		opts.Root = "/"
	}

	if opts.Drop {
		exists, err := mgr.IndexExists(ctx)
		if err != nil {
			return 0, err
		}
		if exists {
			if err := mgr.DropIndex(ctx); err != nil {
				return 0, fmt.Errorf("reindex: failed to drop index: %w", err)
			}
		}

		if err := cleanIdxKeys(ctx, rdb, mgr.IdxPrefix()); err != nil {
			return 0, fmt.Errorf("reindex: failed to clean idx keys: %w", err)
		}
	}

	if err := mgr.EnsureIndex(ctx, true, dim); err != nil {
		return 0, fmt.Errorf("reindex: failed to create index: %w", err)
	}

	files, err := walker(ctx, opts.Root)
	if err != nil {
		return 0, fmt.Errorf("reindex: walk failed: %w", err)
	}

	indexed := 0
	for _, f := range files {
		if err := indexer.IndexFileWithEmbedding(ctx, f.Path, f.Content); err != nil {
			if opts.Progress != nil {
				opts.Progress(indexed, fmt.Sprintf("error: %s: %v", f.Path, err))
			}
			continue
		}
		indexed++
		if opts.Progress != nil {
			opts.Progress(indexed, f.Path)
		}
	}

	return indexed, nil
}

func cleanIdxKeys(ctx context.Context, rdb *redis.Client, prefix string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			rdb.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

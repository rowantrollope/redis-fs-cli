package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/rowantrollope/redis-fs-cli/internal/embedding"
	"github.com/rowantrollope/redis-fs-cli/internal/search"
	flag "github.com/spf13/pflag"
)

func (r *Router) handleVectorSearch(ctx context.Context, args []string) error {
	if !r.Config.SearchAvailable {
		return fmt.Errorf("vector-search: search not available (requires Redis 8.0+ or Redis Stack with RediSearch)")
	}

	if r.Config.EmbeddingAPIKey == "" {
		return fmt.Errorf("vector-search: embedding API key not configured (set EMBEDDING_API_KEY or use --embedding-api-key)")
	}

	fset := flag.NewFlagSet("vector-search", flag.ContinueOnError)
	topK := fset.Int("top", 10, "Number of results to return")
	textFilter := fset.String("filter", "", "Full-text filter to narrow results")
	if err := fset.Parse(args); err != nil {
		return err
	}

	if fset.NArg() < 1 {
		return fmt.Errorf("vector-search: usage: vector-search [--top N] [--filter text] \"query\" [path]")
	}

	query := fset.Arg(0)
	dirFilter := "/"
	if fset.NArg() > 1 {
		dirFilter = r.ResolvePath(fset.Arg(1))
	}

	// Check index exists
	mgr := search.NewIndexManager(r.Client.Redis(), r.State.Volume)
	exists, err := mgr.IndexExists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("vector-search: no index exists. Run 'reindex' first")
	}

	// Create embedding client and embed the query
	embCfg := &embedding.Config{
		APIKey:  r.Config.EmbeddingAPIKey,
		BaseURL: r.Config.EmbeddingAPIURL,
		Model:   r.Config.EmbeddingModel,
		Dim:     r.Config.EmbeddingDim,
	}
	embClient := embedding.NewClient(embCfg)

	queryVec, err := embClient.Embed(ctx, query)
	if err != nil {
		return fmt.Errorf("vector-search: failed to embed query: %w", err)
	}

	// Perform hybrid search
	opts := search.HybridSearchOptions{
		QueryVector: queryVec,
		TextFilter:  *textFilter,
		DirFilter:   dirFilter,
		TopK:        *topK,
	}

	results, err := search.SearchHybrid(ctx, r.Client.Redis(), mgr.IndexName(), opts)
	if err != nil {
		return fmt.Errorf("vector-search: %w", err)
	}

	if len(results) == 0 {
		fmt.Fprintln(r.Formatter.Writer, "No results found.")
		return nil
	}

	for i, result := range results {
		similarity := 1.0 - result.Score // cosine distance to similarity
		fmt.Fprintf(r.Formatter.Writer, "%d. %s (similarity: %.4f)\n", i+1, result.Path, similarity)

		// Show content snippet (first 200 chars)
		snippet := result.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		snippet = strings.ReplaceAll(snippet, "\n", " ")
		fmt.Fprintf(r.Formatter.Writer, "   %s\n\n", snippet)
	}

	return nil
}

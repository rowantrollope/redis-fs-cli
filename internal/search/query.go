package search

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/rowantrollope/redis-fs-cli/internal/embedding"
)

// SearchResult holds a single search result from FT.SEARCH.
type SearchResult struct {
	Path    string
	Content string
	Score   float64
}

// IsSimplePattern returns true if the pattern can be used as a full-text query
// (no regex metacharacters that would require scan-based matching).
func IsSimplePattern(pattern string) bool {
	for _, ch := range pattern {
		switch ch {
		case '[', ']', '(', ')', '{', '}', '|', '+', '\\', '^', '$', '?':
			return false
		}
	}
	// Also check for regex-specific sequences
	if strings.Contains(pattern, ".*") || strings.Contains(pattern, ".+") {
		return false
	}
	return true
}

// EscapeQuery escapes special characters in a RediSearch query string.
func EscapeQuery(s string) string {
	special := []string{",", ".", "<", ">", "{", "}", "[", "]",
		"\"", "'", ":", ";", "!", "@", "#", "$", "%",
		"^", "&", "*", "(", ")", "-", "+", "=", "~"}
	result := s
	for _, ch := range special {
		result = strings.ReplaceAll(result, ch, "\\"+ch)
	}
	return result
}

// SearchFullText performs a full-text search using FT.SEARCH.
func SearchFullText(ctx context.Context, rdb *redis.Client, indexName, pattern, dirFilter string, limit int) ([]SearchResult, error) {
	query := EscapeQuery(pattern)

	if dirFilter != "" && dirFilter != "/" {
		escapedDir := strings.ReplaceAll(dirFilter, "/", "\\/")
		query = fmt.Sprintf("@dir:{%s*} %s", escapedDir, query)
	}

	args := []interface{}{
		"FT.SEARCH", indexName, query,
		"RETURN", "2", "path", "content",
		"LIMIT", "0", fmt.Sprintf("%d", limit),
	}

	result, err := rdb.Do(ctx, args...).Result()
	if err != nil {
		return nil, fmt.Errorf("FT.SEARCH: %w", err)
	}

	return parseSearchResults(result)
}

// HybridSearchOptions controls hybrid vector + full-text search.
type HybridSearchOptions struct {
	QueryText   string    // text to embed for vector search
	QueryVector []float32 // pre-computed query embedding
	TextFilter  string    // optional full-text filter terms
	DirFilter   string    // optional directory filter
	TopK        int       // number of results to return
}

// SearchHybrid performs a hybrid search combining vector KNN with optional
// full-text filtering and directory scoping.
func SearchHybrid(ctx context.Context, rdb *redis.Client, indexName string, opts HybridSearchOptions) ([]SearchResult, error) {
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	// Build the pre-filter query
	preFilter := "*"
	var filters []string

	if opts.DirFilter != "" && opts.DirFilter != "/" {
		escapedDir := strings.ReplaceAll(opts.DirFilter, "/", "\\/")
		filters = append(filters, fmt.Sprintf("@dir:{%s*}", escapedDir))
	}

	if opts.TextFilter != "" {
		filters = append(filters, EscapeQuery(opts.TextFilter))
	}

	if len(filters) > 0 {
		preFilter = "(" + strings.Join(filters, " ") + ")"
	}

	// Build KNN query
	query := fmt.Sprintf("%s=>[KNN %d @embedding $vec AS vector_score]",
		preFilter, opts.TopK)

	vecBytes := embedding.Float32ToBytes(opts.QueryVector)

	args := []interface{}{
		"FT.SEARCH", indexName, query,
		"RETURN", "3", "path", "content", "vector_score",
		"SORTBY", "vector_score",
		"LIMIT", "0", fmt.Sprintf("%d", opts.TopK),
		"PARAMS", "2", "vec", string(vecBytes),
		"DIALECT", "2",
	}

	result, err := rdb.Do(ctx, args...).Result()
	if err != nil {
		return nil, fmt.Errorf("FT.SEARCH hybrid: %w", err)
	}

	return parseSearchResultsWithScore(result)
}

func parseSearchResults(result interface{}) ([]SearchResult, error) {
	slice, ok := result.([]interface{})
	if !ok || len(slice) < 1 {
		return nil, nil
	}

	// First element is total count
	total, ok := slice[0].(int64)
	if !ok || total == 0 {
		return nil, nil
	}

	var results []SearchResult

	// Results are: key, [field, value, field, value, ...], key, [...], ...
	i := 1
	for i < len(slice) {
		if i+1 >= len(slice) {
			break
		}

		// skip key name
		i++

		// parse field-value pairs
		fields, ok := slice[i].([]interface{})
		if !ok {
			i++
			continue
		}

		sr := SearchResult{}
		for j := 0; j+1 < len(fields); j += 2 {
			key, _ := fields[j].(string)
			val, _ := fields[j+1].(string)
			switch key {
			case "path":
				sr.Path = val
			case "content":
				sr.Content = val
			}
		}

		if sr.Path != "" {
			results = append(results, sr)
		}
		i++
	}

	return results, nil
}

func parseSearchResultsWithScore(result interface{}) ([]SearchResult, error) {
	slice, ok := result.([]interface{})
	if !ok || len(slice) < 1 {
		return nil, nil
	}

	total, ok := slice[0].(int64)
	if !ok || total == 0 {
		return nil, nil
	}

	var results []SearchResult

	i := 1
	for i < len(slice) {
		if i+1 >= len(slice) {
			break
		}

		// skip key name
		i++

		fields, ok := slice[i].([]interface{})
		if !ok {
			i++
			continue
		}

		sr := SearchResult{}
		for j := 0; j+1 < len(fields); j += 2 {
			key, _ := fields[j].(string)
			val, _ := fields[j+1].(string)
			switch key {
			case "path":
				sr.Path = val
			case "content":
				sr.Content = val
			case "vector_score":
				sr.Score, _ = strconv.ParseFloat(val, 64)
			}
		}

		if sr.Path != "" {
			results = append(results, sr)
		}
		i++
	}

	return results, nil
}

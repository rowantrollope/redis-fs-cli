package search

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// IndexManager handles FT index lifecycle (create, drop, info).
type IndexManager struct {
	rdb    *redis.Client
	volume string
}

// NewIndexManager creates a new IndexManager for the given volume.
func NewIndexManager(rdb *redis.Client, volume string) *IndexManager {
	return &IndexManager{rdb: rdb, volume: volume}
}

// IndexName returns the FT index name for this volume.
func (m *IndexManager) IndexName() string {
	return fmt.Sprintf("fsidx:%s", m.volume)
}

// IdxPrefix returns the key prefix for index HASH keys.
func (m *IndexManager) IdxPrefix() string {
	return fmt.Sprintf("fs:%s:idx:", m.volume)
}

// CreateIndex creates the FT index. If withVector is true, includes the embedding field.
func (m *IndexManager) CreateIndex(ctx context.Context, withVector bool, dim int) error {
	args := []interface{}{
		"FT.CREATE", m.IndexName(),
		"ON", "HASH",
		"PREFIX", "1", m.IdxPrefix(),
		"SCHEMA",
		"content", "TEXT", "WEIGHT", "1.0",
		"path", "TAG", "SEPARATOR", "/",
		"dir", "TAG", "SEPARATOR", "/",
		"filename", "TEXT", "WEIGHT", "0.5",
		"mtime", "NUMERIC", "SORTABLE",
		"size", "NUMERIC", "SORTABLE",
	}

	if withVector {
		args = append(args,
			"embedding", "VECTOR", "HNSW", "6",
			"TYPE", "FLOAT32",
			"DIM", dim,
			"DISTANCE_METRIC", "COSINE",
		)
	}

	_, err := m.rdb.Do(ctx, args...).Result()
	if err != nil {
		return fmt.Errorf("FT.CREATE: %w", err)
	}
	return nil
}

// DropIndex drops the FT index (does not delete the underlying HASH keys).
func (m *IndexManager) DropIndex(ctx context.Context) error {
	_, err := m.rdb.Do(ctx, "FT.DROPINDEX", m.IndexName()).Result()
	if err != nil {
		return fmt.Errorf("FT.DROPINDEX: %w", err)
	}
	return nil
}

// IndexExists checks if the FT index exists.
func (m *IndexManager) IndexExists(ctx context.Context) (bool, error) {
	_, err := m.rdb.Do(ctx, "FT.INFO", m.IndexName()).Result()
	if err != nil {
		if isIndexNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IndexInfo returns raw FT.INFO result for display.
func (m *IndexManager) IndexInfo(ctx context.Context) (map[string]interface{}, error) {
	result, err := m.rdb.Do(ctx, "FT.INFO", m.IndexName()).Result()
	if err != nil {
		return nil, fmt.Errorf("FT.INFO: %w", err)
	}
	return parseInfoResult(result), nil
}

// EnsureIndex creates the index if it doesn't already exist.
func (m *IndexManager) EnsureIndex(ctx context.Context, withVector bool, dim int) error {
	exists, err := m.IndexExists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return m.CreateIndex(ctx, withVector, dim)
}

// SetVolume updates the volume for this manager.
func (m *IndexManager) SetVolume(volume string) {
	m.volume = volume
}

func isIndexNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "Unknown index name" || msg == "Unknown Index name"
}

// parseInfoResult converts FT.INFO result (flat key-value list) into a map.
func parseInfoResult(result interface{}) map[string]interface{} {
	info := make(map[string]interface{})
	slice, ok := result.([]interface{})
	if !ok {
		return info
	}
	for i := 0; i+1 < len(slice); i += 2 {
		key, ok := slice[i].(string)
		if !ok {
			continue
		}
		info[key] = slice[i+1]
	}
	return info
}

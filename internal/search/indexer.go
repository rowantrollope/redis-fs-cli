package search

import (
	"context"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rowantrollope/redis-fs-cli/internal/embedding"
)

// Indexer maintains the search index in sync with file mutations.
// It implements fs.FileObserver.
type Indexer struct {
	rdb       *redis.Client
	mgr       *IndexManager
	volume    string
	embedder  *embedding.Client
	embedDim  int
}

// NewIndexer creates a new Indexer for the given volume.
func NewIndexer(rdb *redis.Client, volume string) *Indexer {
	return &Indexer{
		rdb:    rdb,
		mgr:    NewIndexManager(rdb, volume),
		volume: volume,
	}
}

// SetEmbedder configures the embedding client for vector indexing.
func (idx *Indexer) SetEmbedder(client *embedding.Client, dim int) {
	idx.embedder = client
	idx.embedDim = dim
}

// SetVolume updates the volume for this indexer.
func (idx *Indexer) SetVolume(volume string) {
	idx.volume = volume
	idx.mgr.SetVolume(volume)
}

// Manager returns the underlying IndexManager.
func (idx *Indexer) Manager() *IndexManager {
	return idx.mgr
}

// HasEmbedder returns true if an embedding client is configured.
func (idx *Indexer) HasEmbedder() bool {
	return idx.embedder != nil
}

// EmbedDim returns the configured embedding dimension.
func (idx *Indexer) EmbedDim() int {
	return idx.embedDim
}

// Embedder returns the embedding client.
func (idx *Indexer) Embedder() *embedding.Client {
	return idx.embedder
}

// OnFileWrite indexes a file after it is written or updated.
func (idx *Indexer) OnFileWrite(ctx context.Context, filePath, content string) error {
	if isBinary(content) {
		return nil
	}

	exists, err := idx.mgr.IndexExists(ctx)
	if err != nil || !exists {
		return nil
	}

	if err := idx.indexFileContent(ctx, filePath, content); err != nil {
		return err
	}

	// Generate embedding asynchronously
	if idx.embedder != nil {
		go idx.asyncEmbed(filePath, content)
	}

	return nil
}

// OnFileRemove removes a file from the index.
func (idx *Indexer) OnFileRemove(ctx context.Context, filePath string) error {
	key := idx.idxKey(filePath)
	idx.rdb.Del(ctx, key)
	return nil
}

// OnFileMove updates the index when a file is moved.
func (idx *Indexer) OnFileMove(ctx context.Context, oldPath, newPath string) error {
	oldKey := idx.idxKey(oldPath)
	newKey := idx.idxKey(newPath)

	// Check if old entry exists
	exists, err := idx.rdb.Exists(ctx, oldKey).Result()
	if err != nil || exists == 0 {
		return nil
	}

	// Get old content and re-index at new path
	content, err := idx.rdb.HGet(ctx, oldKey, "content").Result()
	if err != nil {
		return nil
	}

	pipe := idx.rdb.TxPipeline()
	pipe.Del(ctx, oldKey)
	pipe.HSet(ctx, newKey, map[string]interface{}{
		"content":  content,
		"path":     newPath,
		"dir":      parentDir(newPath),
		"filename": path.Base(newPath),
		"mtime":    strconv.FormatInt(time.Now().Unix(), 10),
		"size":     strconv.Itoa(len(content)),
	})
	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}

	// Re-embed at new path asynchronously
	if idx.embedder != nil {
		go idx.asyncEmbed(newPath, content)
	}

	return nil
}

// IndexFile indexes a single file with the given content and metadata.
// Used by reindex for bulk indexing.
func (idx *Indexer) IndexFile(ctx context.Context, filePath, content string, mtime int64, size int64) error {
	if isBinary(content) {
		return nil
	}
	return idx.indexFileContent(ctx, filePath, content)
}

// IndexFileWithEmbedding indexes a file and also generates and stores its embedding.
// Used by reindex when embeddings are configured.
func (idx *Indexer) IndexFileWithEmbedding(ctx context.Context, filePath, content string) error {
	if isBinary(content) {
		return nil
	}

	if err := idx.indexFileContent(ctx, filePath, content); err != nil {
		return err
	}

	if idx.embedder == nil || content == "" {
		return nil
	}

	vec, err := idx.embedder.Embed(ctx, content)
	if err != nil {
		return fmt.Errorf("embed %s: %w", filePath, err)
	}

	key := idx.idxKey(filePath)
	_, err = idx.rdb.HSet(ctx, key, "embedding", embedding.Float32ToBytes(vec)).Result()
	if err != nil {
		return fmt.Errorf("store embedding %s: %w", filePath, err)
	}

	return nil
}

func (idx *Indexer) indexFileContent(ctx context.Context, filePath, content string) error {
	key := idx.idxKey(filePath)
	fields := map[string]interface{}{
		"content":  content,
		"path":     filePath,
		"dir":      parentDir(filePath),
		"filename": path.Base(filePath),
		"mtime":    strconv.FormatInt(time.Now().Unix(), 10),
		"size":     strconv.Itoa(len(content)),
	}

	_, err := idx.rdb.HSet(ctx, key, fields).Result()
	if err != nil {
		return fmt.Errorf("index file %s: %w", filePath, err)
	}
	return nil
}

// asyncEmbed generates an embedding asynchronously and stores it.
func (idx *Indexer) asyncEmbed(filePath, content string) {
	if content == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vec, err := idx.embedder.Embed(ctx, content)
	if err != nil {
		log.Printf("async embed %s: %v", filePath, err)
		return
	}

	key := idx.idxKey(filePath)
	_, err = idx.rdb.HSet(ctx, key, "embedding", embedding.Float32ToBytes(vec)).Result()
	if err != nil {
		log.Printf("store embedding %s: %v", filePath, err)
	}
}

func (idx *Indexer) idxKey(filePath string) string {
	return fmt.Sprintf("fs:%s:idx:%s", idx.volume, filePath)
}

func parentDir(filePath string) string {
	dir := path.Dir(filePath)
	if dir == "." {
		return "/"
	}
	return dir
}

// isBinary checks if content appears to be binary by looking for null bytes.
func isBinary(content string) bool {
	check := content
	if len(check) > 512 {
		check = check[:512]
	}
	return strings.ContainsRune(check, '\x00')
}

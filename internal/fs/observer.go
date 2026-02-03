package fs

import "context"

// FileObserver receives notifications when files are mutated.
// Implementations can use this to maintain search indexes, caches, etc.
type FileObserver interface {
	OnFileWrite(ctx context.Context, path, content string) error
	OnFileRemove(ctx context.Context, path string) error
	OnFileMove(ctx context.Context, oldPath, newPath string) error
}

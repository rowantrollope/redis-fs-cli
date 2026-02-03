package fs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const maxSymlinkDepth = 40

// Client provides filesystem operations backed by Redis.
type Client struct {
	rdb      *redis.Client
	keys     *KeyGen
	Volume   string
	observer FileObserver
}

// NewClient creates a new filesystem client.
func NewClient(rdb *redis.Client, volume string) *Client {
	return &Client{
		rdb:    rdb,
		keys:   NewKeyGen(volume),
		Volume: volume,
	}
}

// SetVolume switches the active volume.
func (c *Client) SetVolume(volume string) {
	c.Volume = volume
	c.keys = NewKeyGen(volume)
}

// SetObserver registers a FileObserver for mutation notifications.
func (c *Client) SetObserver(obs FileObserver) {
	c.observer = obs
}

// Keys returns the key generator (for use by search indexing).
func (c *Client) Keys() *KeyGen {
	return c.keys
}

// Redis returns the underlying Redis client.
func (c *Client) Redis() *redis.Client {
	return c.rdb
}

// --- Init ---

// Init bootstraps the volume root directory if it doesn't exist.
func (c *Client) Init(ctx context.Context) error {
	metaKey := c.keys.Meta("/")
	// Use HSETNX to make it idempotent
	created, err := c.rdb.HSetNX(ctx, metaKey, "type", "dir").Result()
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	if created {
		meta := NewDirMeta("0755")
		_, err := c.rdb.HSet(ctx, metaKey, meta.ToMap()).Result()
		if err != nil {
			return fmt.Errorf("init: %w", err)
		}
	}
	return nil
}

// --- Stat / Exists ---

// Stat returns metadata for a path. Returns nil, nil if not found.
func (c *Client) Stat(ctx context.Context, path string) (*Metadata, error) {
	m, err := c.rdb.HGetAll(ctx, c.keys.Meta(path)).Result()
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	if len(m) == 0 {
		return nil, nil
	}
	return MetaFromMap(m), nil
}

// Exists checks if a path exists.
func (c *Client) Exists(ctx context.Context, path string) (bool, error) {
	n, err := c.rdb.Exists(ctx, c.keys.Meta(path)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// IsDir checks if the path is an existing directory.
func (c *Client) IsDir(ctx context.Context, path string) (bool, error) {
	t, err := c.rdb.HGet(ctx, c.keys.Meta(path), "type").Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return t == string(TypeDir), nil
}

// --- ReadDir ---

// ReadDir returns the child entry names of a directory.
func (c *Client) ReadDir(ctx context.Context, path string) ([]string, error) {
	members, err := c.rdb.SMembers(ctx, c.keys.Dir(path)).Result()
	if err != nil {
		return nil, fmt.Errorf("readdir: %w", err)
	}
	return members, nil
}

// ReadDirWithMeta returns child names with metadata (for ls -l).
func (c *Client) ReadDirWithMeta(ctx context.Context, dirPath string) ([]DirEntry, error) {
	children, err := c.ReadDir(ctx, dirPath)
	if err != nil {
		return nil, err
	}

	if len(children) == 0 {
		return nil, nil
	}

	// Pipeline HGETALL for all children
	pipe := c.rdb.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(children))
	for i, child := range children {
		childPath := JoinPath(dirPath, child)
		cmds[i] = pipe.HGetAll(ctx, c.keys.Meta(childPath))
	}
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("readdir meta: %w", err)
	}

	entries := make([]DirEntry, 0, len(children))
	for i, child := range children {
		m, _ := cmds[i].Result()
		meta := MetaFromMap(m)
		entries = append(entries, DirEntry{
			Name: child,
			Meta: meta,
		})
	}
	return entries, nil
}

// DirEntry is a directory listing entry with metadata.
type DirEntry struct {
	Name string
	Meta *Metadata
}

// --- Mkdir ---

// Mkdir creates a directory. If parents is true, creates intermediate directories.
func (c *Client) Mkdir(ctx context.Context, path string, parents bool) error {
	path = NormalizePath(path)
	if path == "/" {
		return nil
	}

	if parents {
		return c.mkdirParents(ctx, path)
	}

	// Check parent exists and is a directory
	parent := ParentPath(path)
	isDir, err := c.IsDir(ctx, parent)
	if err != nil {
		return err
	}
	if !isDir {
		return fmt.Errorf("mkdir: cannot create directory '%s': No such file or directory", path)
	}

	// Check target doesn't already exist
	exists, err := c.Exists(ctx, path)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("mkdir: cannot create directory '%s': File exists", path)
	}

	return c.createDir(ctx, path)
}

func (c *Client) mkdirParents(ctx context.Context, path string) error {
	parts := strings.Split(path, "/")
	current := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		current += "/" + part
		exists, err := c.Exists(ctx, current)
		if err != nil {
			return err
		}
		if !exists {
			if err := c.createDir(ctx, current); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) createDir(ctx context.Context, path string) error {
	parent, base := SplitPath(path)
	meta := NewDirMeta("0755")

	pipe := c.rdb.TxPipeline()
	pipe.HSet(ctx, c.keys.Meta(path), meta.ToMap())
	pipe.SAdd(ctx, c.keys.Dir(parent), base)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return nil
}

// --- Rmdir ---

// Rmdir removes an empty directory.
func (c *Client) Rmdir(ctx context.Context, path string) error {
	path = NormalizePath(path)
	if path == "/" {
		return fmt.Errorf("rmdir: cannot remove root directory")
	}

	isDir, err := c.IsDir(ctx, path)
	if err != nil {
		return err
	}
	if !isDir {
		return fmt.Errorf("rmdir: failed to remove '%s': Not a directory", path)
	}

	count, err := c.rdb.SCard(ctx, c.keys.Dir(path)).Result()
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("rmdir: failed to remove '%s': Directory not empty", path)
	}

	parent, base := SplitPath(path)
	pipe := c.rdb.TxPipeline()
	pipe.Del(ctx, c.keys.Meta(path))
	pipe.Del(ctx, c.keys.Dir(path))
	pipe.Del(ctx, c.keys.Xattr(path))
	pipe.SRem(ctx, c.keys.Dir(parent), base)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("rmdir: %w", err)
	}
	return nil
}

// --- Touch ---

// Touch creates a file or updates timestamps.
func (c *Client) Touch(ctx context.Context, path string) error {
	path = NormalizePath(path)
	exists, err := c.Exists(ctx, path)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	nowStr := strconv.FormatInt(now, 10)

	if exists {
		_, err := c.rdb.HSet(ctx, c.keys.Meta(path), "mtime", nowStr, "atime", nowStr).Result()
		return err
	}

	// New file
	parent := ParentPath(path)
	isDir, err := c.IsDir(ctx, parent)
	if err != nil {
		return err
	}
	if !isDir {
		return fmt.Errorf("touch: cannot touch '%s': No such file or directory", path)
	}

	_, base := SplitPath(path)
	meta := NewFileMeta("0644", 0)

	pipe := c.rdb.TxPipeline()
	pipe.Set(ctx, c.keys.Data(path), "", 0)
	pipe.HSet(ctx, c.keys.Meta(path), meta.ToMap())
	pipe.SAdd(ctx, c.keys.Dir(parent), base)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("touch: %w", err)
	}
	return nil
}

// --- Cat (ReadFile) ---

// ReadFile returns the content of a file.
func (c *Client) ReadFile(ctx context.Context, path string) (string, error) {
	path = NormalizePath(path)

	meta, err := c.Stat(ctx, path)
	if err != nil {
		return "", err
	}
	if meta == nil {
		return "", fmt.Errorf("cat: %s: No such file or directory", path)
	}
	if meta.Type == TypeDir {
		return "", fmt.Errorf("cat: %s: Is a directory", path)
	}

	// Follow symlinks
	if meta.Type == TypeSymlink {
		resolved, err := c.ResolveSymlink(ctx, path, 0)
		if err != nil {
			return "", err
		}
		path = resolved
	}

	data, err := c.rdb.Get(ctx, c.keys.Data(path)).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("cat: %w", err)
	}

	// Update atime
	now := strconv.FormatInt(time.Now().Unix(), 10)
	c.rdb.HSet(ctx, c.keys.Meta(path), "atime", now)

	return data, nil
}

// --- WriteFile (echo >) ---

// WriteFile sets file content (truncate/overwrite).
func (c *Client) WriteFile(ctx context.Context, path, content string) error {
	path = NormalizePath(path)

	exists, err := c.Exists(ctx, path)
	if err != nil {
		return err
	}

	now := strconv.FormatInt(time.Now().Unix(), 10)
	size := strconv.Itoa(len(content))

	if exists {
		meta, err := c.Stat(ctx, path)
		if err != nil {
			return err
		}
		if meta != nil && meta.Type == TypeDir {
			return fmt.Errorf("echo: %s: Is a directory", path)
		}
		pipe := c.rdb.TxPipeline()
		pipe.Set(ctx, c.keys.Data(path), content, 0)
		pipe.HSet(ctx, c.keys.Meta(path), "size", size, "mtime", now)
		_, err = pipe.Exec(ctx)
		if err != nil {
			return err
		}
		c.notifyWrite(ctx, path, content)
		return nil
	}

	// New file
	parent := ParentPath(path)
	isDir, err := c.IsDir(ctx, parent)
	if err != nil {
		return err
	}
	if !isDir {
		return fmt.Errorf("echo: %s: No such file or directory", ParentPath(path))
	}

	_, base := SplitPath(path)
	meta := NewFileMeta("0644", int64(len(content)))

	pipe := c.rdb.TxPipeline()
	pipe.Set(ctx, c.keys.Data(path), content, 0)
	pipe.HSet(ctx, c.keys.Meta(path), meta.ToMap())
	pipe.SAdd(ctx, c.keys.Dir(parent), base)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("echo: %w", err)
	}
	c.notifyWrite(ctx, path, content)
	return nil
}

// --- AppendFile (echo >>) ---

// AppendFile appends content to a file.
func (c *Client) AppendFile(ctx context.Context, path, content string) error {
	path = NormalizePath(path)

	exists, err := c.Exists(ctx, path)
	if err != nil {
		return err
	}

	if !exists {
		return c.WriteFile(ctx, path, content)
	}

	meta, err := c.Stat(ctx, path)
	if err != nil {
		return err
	}
	if meta != nil && meta.Type == TypeDir {
		return fmt.Errorf("echo: %s: Is a directory", path)
	}

	now := strconv.FormatInt(time.Now().Unix(), 10)

	pipe := c.rdb.TxPipeline()
	pipe.Append(ctx, c.keys.Data(path), content)
	strlenCmd := pipe.StrLen(ctx, c.keys.Data(path))
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("echo: %w", err)
	}

	newSize := strconv.FormatInt(strlenCmd.Val(), 10)
	_, err = c.rdb.HSet(ctx, c.keys.Meta(path), "size", newSize, "mtime", now).Result()
	if err != nil {
		return err
	}

	// Re-index with full content
	if c.observer != nil {
		fullContent, readErr := c.rdb.Get(ctx, c.keys.Data(path)).Result()
		if readErr == nil {
			c.notifyWrite(ctx, path, fullContent)
		}
	}
	return nil
}

// --- Remove ---

// Remove removes a file or empty directory.
func (c *Client) Remove(ctx context.Context, path string) error {
	path = NormalizePath(path)
	if path == "/" {
		return fmt.Errorf("rm: cannot remove root directory")
	}

	meta, err := c.Stat(ctx, path)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("rm: cannot remove '%s': No such file or directory", path)
	}
	if meta.Type == TypeDir {
		return fmt.Errorf("rm: cannot remove '%s': Is a directory", path)
	}

	parent, base := SplitPath(path)
	pipe := c.rdb.TxPipeline()
	pipe.Del(ctx, c.keys.Meta(path))
	pipe.Del(ctx, c.keys.Data(path))
	pipe.Del(ctx, c.keys.Xattr(path))
	pipe.SRem(ctx, c.keys.Dir(parent), base)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("rm: %w", err)
	}
	c.notifyRemove(ctx, path)
	return nil
}

// RemoveRecursive removes a file or directory recursively.
func (c *Client) RemoveRecursive(ctx context.Context, path string) error {
	path = NormalizePath(path)
	if path == "/" {
		return fmt.Errorf("rm: cannot remove root directory")
	}

	meta, err := c.Stat(ctx, path)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("rm: cannot remove '%s': No such file or directory", path)
	}

	if meta.Type != TypeDir {
		return c.Remove(ctx, path)
	}

	// DFS traversal
	children, err := c.ReadDir(ctx, path)
	if err != nil {
		return err
	}
	for _, child := range children {
		childPath := JoinPath(path, child)
		if err := c.RemoveRecursive(ctx, childPath); err != nil {
			return err
		}
	}

	// Remove the directory itself
	parent, base := SplitPath(path)
	pipe := c.rdb.TxPipeline()
	pipe.Del(ctx, c.keys.Meta(path))
	pipe.Del(ctx, c.keys.Dir(path))
	pipe.Del(ctx, c.keys.Xattr(path))
	pipe.SRem(ctx, c.keys.Dir(parent), base)
	_, err = pipe.Exec(ctx)
	return err
}

// --- Copy ---

// CopyFile copies a single file.
func (c *Client) CopyFile(ctx context.Context, src, dst string) error {
	src = NormalizePath(src)
	dst = NormalizePath(dst)

	// Check if dst is an existing directory
	dstMeta, err := c.Stat(ctx, dst)
	if err != nil {
		return err
	}
	if dstMeta != nil && dstMeta.Type == TypeDir {
		dst = JoinPath(dst, BaseName(src))
	}

	srcMeta, err := c.Stat(ctx, src)
	if err != nil {
		return err
	}
	if srcMeta == nil {
		return fmt.Errorf("cp: cannot stat '%s': No such file or directory", src)
	}
	if srcMeta.Type == TypeDir {
		return fmt.Errorf("cp: -r not specified; omitting directory '%s'", src)
	}

	data, err := c.rdb.Get(ctx, c.keys.Data(src)).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("cp: %w", err)
	}

	dstParent := ParentPath(dst)
	isDir, err := c.IsDir(ctx, dstParent)
	if err != nil {
		return err
	}
	if !isDir {
		return fmt.Errorf("cp: cannot create '%s': No such file or directory", dst)
	}

	now := time.Now().Unix()
	nowStr := strconv.FormatInt(now, 10)
	newMeta := *srcMeta
	newMeta.CTime = now
	newMeta.MTime = now
	newMeta.ATime = now

	_, dstBase := SplitPath(dst)
	pipe := c.rdb.TxPipeline()
	pipe.Set(ctx, c.keys.Data(dst), data, 0)
	pipe.HSet(ctx, c.keys.Meta(dst), newMeta.ToMap())
	pipe.SAdd(ctx, c.keys.Dir(dstParent), dstBase)
	// Update src atime
	pipe.HSet(ctx, c.keys.Meta(src), "atime", nowStr)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("cp: %w", err)
	}
	c.notifyWrite(ctx, dst, data)
	return nil
}

// CopyRecursive copies a file or directory recursively.
func (c *Client) CopyRecursive(ctx context.Context, src, dst string) error {
	src = NormalizePath(src)
	dst = NormalizePath(dst)

	srcMeta, err := c.Stat(ctx, src)
	if err != nil {
		return err
	}
	if srcMeta == nil {
		return fmt.Errorf("cp: cannot stat '%s': No such file or directory", src)
	}

	// Check if dst is an existing directory
	dstMeta, err := c.Stat(ctx, dst)
	if err != nil {
		return err
	}
	if dstMeta != nil && dstMeta.Type == TypeDir {
		dst = JoinPath(dst, BaseName(src))
	}

	if srcMeta.Type != TypeDir {
		return c.CopyFile(ctx, src, dst)
	}

	// Create destination directory
	if err := c.Mkdir(ctx, dst, true); err != nil {
		return err
	}

	// Recursively copy children
	children, err := c.ReadDir(ctx, src)
	if err != nil {
		return err
	}
	for _, child := range children {
		srcChild := JoinPath(src, child)
		dstChild := JoinPath(dst, child)
		if err := c.CopyRecursive(ctx, srcChild, dstChild); err != nil {
			return err
		}
	}
	return nil
}

// --- Move ---

// Move moves/renames a file or directory.
func (c *Client) Move(ctx context.Context, src, dst string) error {
	src = NormalizePath(src)
	dst = NormalizePath(dst)

	srcMeta, err := c.Stat(ctx, src)
	if err != nil {
		return err
	}
	if srcMeta == nil {
		return fmt.Errorf("mv: cannot stat '%s': No such file or directory", src)
	}

	// Check if dst is an existing directory
	dstMeta, err := c.Stat(ctx, dst)
	if err != nil {
		return err
	}
	if dstMeta != nil && dstMeta.Type == TypeDir {
		dst = JoinPath(dst, BaseName(src))
	}

	dstParent := ParentPath(dst)
	isDir, err := c.IsDir(ctx, dstParent)
	if err != nil {
		return err
	}
	if !isDir {
		return fmt.Errorf("mv: cannot move '%s' to '%s': No such file or directory", src, dst)
	}

	if srcMeta.Type == TypeDir {
		return c.moveDir(ctx, src, dst)
	}

	return c.moveFile(ctx, src, dst)
}

func (c *Client) moveFile(ctx context.Context, src, dst string) error {
	srcParent, srcBase := SplitPath(src)
	dstParent, dstBase := SplitPath(dst)

	pipe := c.rdb.TxPipeline()
	pipe.Rename(ctx, c.keys.Meta(src), c.keys.Meta(dst))
	pipe.Rename(ctx, c.keys.Data(src), c.keys.Data(dst))
	pipe.SRem(ctx, c.keys.Dir(srcParent), srcBase)
	pipe.SAdd(ctx, c.keys.Dir(dstParent), dstBase)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("mv: %w", err)
	}

	// Best-effort rename xattr
	c.rdb.Rename(ctx, c.keys.Xattr(src), c.keys.Xattr(dst))
	c.notifyMove(ctx, src, dst)
	return nil
}

func (c *Client) moveDir(ctx context.Context, src, dst string) error {
	// For directories, we need to recursively rename all children
	if err := c.CopyRecursive(ctx, src, dst); err != nil {
		return err
	}
	return c.RemoveRecursive(ctx, src)
}

// --- Symlink ---

// Symlink creates a symbolic link.
func (c *Client) Symlink(ctx context.Context, target, linkPath string) error {
	linkPath = NormalizePath(linkPath)

	exists, err := c.Exists(ctx, linkPath)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("ln: '%s': File exists", linkPath)
	}

	parent := ParentPath(linkPath)
	isDir, err := c.IsDir(ctx, parent)
	if err != nil {
		return err
	}
	if !isDir {
		return fmt.Errorf("ln: '%s': No such file or directory", linkPath)
	}

	_, base := SplitPath(linkPath)
	meta := NewSymlinkMeta(target)

	pipe := c.rdb.TxPipeline()
	pipe.HSet(ctx, c.keys.Meta(linkPath), meta.ToMap())
	pipe.SAdd(ctx, c.keys.Dir(parent), base)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("ln: %w", err)
	}
	return nil
}

// ResolveSymlink follows symlinks to the final target.
func (c *Client) ResolveSymlink(ctx context.Context, path string, depth int) (string, error) {
	if depth >= maxSymlinkDepth {
		return "", fmt.Errorf("too many levels of symbolic links: %s", path)
	}

	meta, err := c.Stat(ctx, path)
	if err != nil {
		return "", err
	}
	if meta == nil {
		return path, nil
	}
	if meta.Type != TypeSymlink {
		return path, nil
	}

	target := meta.LinkTarget
	if !strings.HasPrefix(target, "/") {
		target = JoinPath(ParentPath(path), target)
	}

	return c.ResolveSymlink(ctx, target, depth+1)
}

// --- Chmod / Chown ---

// Chmod changes the mode of a path.
func (c *Client) Chmod(ctx context.Context, path, mode string) error {
	path = NormalizePath(path)
	exists, err := c.Exists(ctx, path)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("chmod: cannot access '%s': No such file or directory", path)
	}
	_, err = c.rdb.HSet(ctx, c.keys.Meta(path), "mode", mode).Result()
	return err
}

// Chown changes the uid and/or gid of a path.
func (c *Client) Chown(ctx context.Context, path, owner string) error {
	path = NormalizePath(path)
	exists, err := c.Exists(ctx, path)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("chown: cannot access '%s': No such file or directory", path)
	}

	fields := map[string]interface{}{}
	parts := strings.SplitN(owner, ":", 2)
	if parts[0] != "" {
		fields["uid"] = parts[0]
	}
	if len(parts) > 1 && parts[1] != "" {
		fields["gid"] = parts[1]
	}

	if len(fields) == 0 {
		return fmt.Errorf("chown: invalid owner: '%s'", owner)
	}

	_, err = c.rdb.HSet(ctx, c.keys.Meta(path), fields).Result()
	return err
}

// --- Find ---

// FindEntry represents a result from find.
type FindEntry struct {
	Path string
	Meta *Metadata
}

// Find recursively walks the tree from root, optionally filtering by name glob and type.
func (c *Client) Find(ctx context.Context, root string, namePattern string, typeFilter string) ([]FindEntry, error) {
	root = NormalizePath(root)
	var results []FindEntry
	err := c.findWalk(ctx, root, namePattern, typeFilter, &results)
	return results, err
}

func (c *Client) findWalk(ctx context.Context, path, namePattern, typeFilter string, results *[]FindEntry) error {
	meta, err := c.Stat(ctx, path)
	if err != nil {
		return err
	}
	if meta == nil {
		return nil
	}

	if matchesFind(path, meta, namePattern, typeFilter) {
		*results = append(*results, FindEntry{Path: path, Meta: meta})
	}

	if meta.Type == TypeDir {
		children, err := c.ReadDir(ctx, path)
		if err != nil {
			return err
		}
		for _, child := range children {
			childPath := JoinPath(path, child)
			if err := c.findWalk(ctx, childPath, namePattern, typeFilter, results); err != nil {
				return err
			}
		}
	}
	return nil
}

func matchesFind(path string, meta *Metadata, namePattern, typeFilter string) bool {
	if typeFilter != "" {
		switch typeFilter {
		case "f":
			if meta.Type != TypeFile {
				return false
			}
		case "d":
			if meta.Type != TypeDir {
				return false
			}
		case "l":
			if meta.Type != TypeSymlink {
				return false
			}
		}
	}
	if namePattern != "" {
		base := BaseName(path)
		matched, _ := matchGlob(namePattern, base)
		if !matched {
			return false
		}
	}
	return true
}

// matchGlob performs simple glob matching (supports *, ?).
func matchGlob(pattern, name string) (bool, error) {
	return globMatch(pattern, name), nil
}

func globMatch(pattern, name string) bool {
	if pattern == "*" {
		return true
	}
	px, nx := 0, 0
	starPx, starNx := -1, -1
	for nx < len(name) {
		if px < len(pattern) && (pattern[px] == '?' || pattern[px] == name[nx]) {
			px++
			nx++
		} else if px < len(pattern) && pattern[px] == '*' {
			starPx = px
			starNx = nx
			px++
		} else if starPx >= 0 {
			px = starPx + 1
			starNx++
			nx = starNx
		} else {
			return false
		}
	}
	for px < len(pattern) && pattern[px] == '*' {
		px++
	}
	return px == len(pattern)
}

// --- Volume ---

// ListVolumes scans for volume root meta keys.
func (c *Client) ListVolumes(ctx context.Context) ([]string, error) {
	var volumes []string
	var cursor uint64
	for {
		keys, nextCursor, err := c.rdb.Scan(ctx, cursor, VolumeRootPattern(), 100).Result()
		if err != nil {
			return nil, fmt.Errorf("vol list: %w", err)
		}
		for _, key := range keys {
			// Extract volume name from fs:{vol}:meta:/
			parts := strings.SplitN(key, ":", 4)
			if len(parts) >= 3 {
				volumes = append(volumes, parts[1])
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return volumes, nil
}

// --- Tree ---

// TreeEntry represents a node in a tree listing.
type TreeEntry struct {
	Name     string
	Path     string
	Type     EntryType
	Children []TreeEntry
}

// Tree builds a tree structure for a path.
func (c *Client) Tree(ctx context.Context, root string, maxDepth int) (*TreeEntry, int, int, error) {
	root = NormalizePath(root)
	meta, err := c.Stat(ctx, root)
	if err != nil {
		return nil, 0, 0, err
	}
	if meta == nil {
		return nil, 0, 0, fmt.Errorf("tree: '%s': No such file or directory", root)
	}

	entry := &TreeEntry{
		Name: BaseName(root),
		Path: root,
		Type: meta.Type,
	}

	dirCount, fileCount := 0, 0
	if meta.Type == TypeDir {
		if err := c.buildTree(ctx, root, entry, 1, maxDepth, &dirCount, &fileCount); err != nil {
			return nil, 0, 0, err
		}
	} else {
		fileCount = 1
	}

	return entry, dirCount, fileCount, nil
}

func (c *Client) buildTree(ctx context.Context, path string, entry *TreeEntry, depth, maxDepth int, dirCount, fileCount *int) error {
	if maxDepth > 0 && depth > maxDepth {
		return nil
	}

	children, err := c.ReadDir(ctx, path)
	if err != nil {
		return err
	}

	for _, childName := range children {
		childPath := JoinPath(path, childName)
		childMeta, err := c.Stat(ctx, childPath)
		if err != nil {
			return err
		}
		if childMeta == nil {
			continue
		}

		childEntry := TreeEntry{
			Name: childName,
			Path: childPath,
			Type: childMeta.Type,
		}

		if childMeta.Type == TypeDir {
			*dirCount++
			if err := c.buildTree(ctx, childPath, &childEntry, depth+1, maxDepth, dirCount, fileCount); err != nil {
				return err
			}
		} else {
			*fileCount++
		}

		entry.Children = append(entry.Children, childEntry)
	}
	return nil
}

// --- Grep ---

// GrepResult holds a match result from grep.
type GrepResult struct {
	Path    string
	Line    int
	Content string
}

// --- Observer helpers ---

func (c *Client) notifyWrite(ctx context.Context, path, content string) {
	if c.observer != nil {
		c.observer.OnFileWrite(ctx, path, content)
	}
}

func (c *Client) notifyRemove(ctx context.Context, path string) {
	if c.observer != nil {
		c.observer.OnFileRemove(ctx, path)
	}
}

func (c *Client) notifyMove(ctx context.Context, oldPath, newPath string) {
	if c.observer != nil {
		c.observer.OnFileMove(ctx, oldPath, newPath)
	}
}

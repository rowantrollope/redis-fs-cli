package fs

import (
	"fmt"
	"strconv"
	"time"
)

// EntryType represents the type of a filesystem entry.
type EntryType string

const (
	TypeDir     EntryType = "dir"
	TypeFile    EntryType = "file"
	TypeSymlink EntryType = "symlink"
)

// Metadata holds the filesystem metadata for an entry.
type Metadata struct {
	Type       EntryType
	Mode       string
	UID        string
	GID        string
	Size       int64
	CTime      int64 // creation time (unix timestamp)
	MTime      int64 // modification time
	ATime      int64 // access time
	LinkTarget string
}

// NewDirMeta creates metadata for a new directory.
func NewDirMeta(mode string) *Metadata {
	now := time.Now().Unix()
	if mode == "" {
		mode = "0755"
	}
	return &Metadata{
		Type:  TypeDir,
		Mode:  mode,
		UID:   "0",
		GID:   "0",
		CTime: now,
		MTime: now,
		ATime: now,
	}
}

// NewFileMeta creates metadata for a new file.
func NewFileMeta(mode string, size int64) *Metadata {
	now := time.Now().Unix()
	if mode == "" {
		mode = "0644"
	}
	return &Metadata{
		Type:  TypeFile,
		Mode:  mode,
		UID:   "0",
		GID:   "0",
		Size:  size,
		CTime: now,
		MTime: now,
		ATime: now,
	}
}

// NewSymlinkMeta creates metadata for a new symlink.
func NewSymlinkMeta(target string) *Metadata {
	now := time.Now().Unix()
	return &Metadata{
		Type:       TypeSymlink,
		Mode:       "0777",
		UID:        "0",
		GID:        "0",
		CTime:      now,
		MTime:      now,
		ATime:      now,
		LinkTarget: target,
	}
}

// ToMap converts metadata to a map for HSET.
func (m *Metadata) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type":  string(m.Type),
		"mode":  m.Mode,
		"uid":   m.UID,
		"gid":   m.GID,
		"size":  strconv.FormatInt(m.Size, 10),
		"ctime": strconv.FormatInt(m.CTime, 10),
		"mtime": strconv.FormatInt(m.MTime, 10),
		"atime": strconv.FormatInt(m.ATime, 10),
	}
	if m.LinkTarget != "" {
		result["link_target"] = m.LinkTarget
	}
	return result
}

// MetaFromMap parses a Redis hash map into Metadata.
func MetaFromMap(m map[string]string) *Metadata {
	if m == nil {
		return nil
	}
	size, _ := strconv.ParseInt(m["size"], 10, 64)
	ctime, _ := strconv.ParseInt(m["ctime"], 10, 64)
	mtime, _ := strconv.ParseInt(m["mtime"], 10, 64)
	atime, _ := strconv.ParseInt(m["atime"], 10, 64)

	return &Metadata{
		Type:       EntryType(m["type"]),
		Mode:       m["mode"],
		UID:        m["uid"],
		GID:        m["gid"],
		Size:       size,
		CTime:      ctime,
		MTime:      mtime,
		ATime:      atime,
		LinkTarget: m["link_target"],
	}
}

// FormatTime formats a unix timestamp for display.
func FormatTime(ts int64) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(ts, 0).Format("Jan _2 15:04")
}

// FormatSize formats a file size for display.
func FormatSize(size int64) string {
	return fmt.Sprintf("%d", size)
}

// ModeString returns a POSIX-style mode string like "drwxr-xr-x".
func (m *Metadata) ModeString() string {
	var prefix byte
	switch m.Type {
	case TypeDir:
		prefix = 'd'
	case TypeSymlink:
		prefix = 'l'
	default:
		prefix = '-'
	}

	mode, err := strconv.ParseUint(m.Mode, 8, 32)
	if err != nil {
		return string(prefix) + "rwxr-xr-x"
	}

	perms := [9]byte{'-', '-', '-', '-', '-', '-', '-', '-', '-'}
	bits := []byte{'r', 'w', 'x'}
	for i := 0; i < 9; i++ {
		if mode&(1<<uint(8-i)) != 0 {
			perms[i] = bits[i%3]
		}
	}

	return string(prefix) + string(perms[:])
}

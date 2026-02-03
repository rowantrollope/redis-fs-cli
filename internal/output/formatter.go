package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/rowantrollope/redis-fs-cli/internal/fs"
)

// Formatter handles text/JSON/colored output.
type Formatter struct {
	Writer    io.Writer
	ErrWriter io.Writer
	JSON      bool
	Color     bool
}

// NewFormatter creates a new output formatter.
func NewFormatter(jsonMode, colorMode bool) *Formatter {
	return &Formatter{
		Writer:    os.Stdout,
		ErrWriter: os.Stderr,
		JSON:      jsonMode,
		Color:     colorMode,
	}
}

// Printf prints formatted text to stdout.
func (f *Formatter) Printf(format string, args ...interface{}) {
	fmt.Fprintf(f.Writer, format, args...)
}

// Println prints a line to stdout.
func (f *Formatter) Println(args ...interface{}) {
	fmt.Fprintln(f.Writer, args...)
}

// Errorf prints a formatted error message to stderr.
func (f *Formatter) Errorf(format string, args ...interface{}) {
	if f.Color {
		c := color.New(color.FgRed)
		c.Fprintf(f.ErrWriter, format, args...)
	} else {
		fmt.Fprintf(f.ErrWriter, format, args...)
	}
}

// PrintJSON outputs a value as JSON.
func (f *Formatter) PrintJSON(v interface{}) error {
	enc := json.NewEncoder(f.Writer)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// FormatDirName formats a directory name with color.
func (f *Formatter) FormatDirName(name string) string {
	if f.Color {
		return color.New(color.FgBlue, color.Bold).Sprint(name)
	}
	return name
}

// FormatSymlinkName formats a symlink name with color.
func (f *Formatter) FormatSymlinkName(name string) string {
	if f.Color {
		return color.New(color.FgCyan).Sprint(name)
	}
	return name
}

// FormatFileName formats a file name (default color).
func (f *Formatter) FormatFileName(name string) string {
	return name
}

// FormatEntryName formats an entry name based on its type.
func (f *Formatter) FormatEntryName(name string, entryType fs.EntryType) string {
	switch entryType {
	case fs.TypeDir:
		return f.FormatDirName(name)
	case fs.TypeSymlink:
		return f.FormatSymlinkName(name)
	default:
		return f.FormatFileName(name)
	}
}

// --- ls output ---

// PrintLs prints a simple directory listing.
func (f *Formatter) PrintLs(entries []fs.DirEntry, showAll bool) {
	// Sort entries alphabetically
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	if f.JSON {
		var names []string
		for _, e := range entries {
			if !showAll && strings.HasPrefix(e.Name, ".") {
				continue
			}
			names = append(names, e.Name)
		}
		f.PrintJSON(names)
		return
	}

	for _, e := range entries {
		if !showAll && strings.HasPrefix(e.Name, ".") {
			continue
		}
		if e.Meta != nil {
			fmt.Fprintln(f.Writer, f.FormatEntryName(e.Name, e.Meta.Type))
		} else {
			fmt.Fprintln(f.Writer, e.Name)
		}
	}
}

// PrintLsLong prints a detailed directory listing (ls -l).
func (f *Formatter) PrintLsLong(entries []fs.DirEntry, showAll bool) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	if f.JSON {
		var result []map[string]interface{}
		for _, e := range entries {
			if !showAll && strings.HasPrefix(e.Name, ".") {
				continue
			}
			entry := map[string]interface{}{
				"name": e.Name,
			}
			if e.Meta != nil {
				entry["type"] = string(e.Meta.Type)
				entry["mode"] = e.Meta.Mode
				entry["uid"] = e.Meta.UID
				entry["gid"] = e.Meta.GID
				entry["size"] = e.Meta.Size
				entry["mtime"] = e.Meta.MTime
			}
			result = append(result, entry)
		}
		f.PrintJSON(result)
		return
	}

	for _, e := range entries {
		if !showAll && strings.HasPrefix(e.Name, ".") {
			continue
		}
		if e.Meta == nil {
			fmt.Fprintf(f.Writer, "?????????? ? ? ? ? %s\n", e.Name)
			continue
		}
		name := f.FormatEntryName(e.Name, e.Meta.Type)
		if e.Meta.Type == fs.TypeSymlink && e.Meta.LinkTarget != "" {
			name = name + " -> " + e.Meta.LinkTarget
		}
		fmt.Fprintf(f.Writer, "%s %s %s %6s %s %s\n",
			e.Meta.ModeString(),
			e.Meta.UID,
			e.Meta.GID,
			fs.FormatSize(e.Meta.Size),
			fs.FormatTime(e.Meta.MTime),
			name,
		)
	}
}

// --- stat output ---

// PrintStat prints file/directory metadata.
func (f *Formatter) PrintStat(path string, meta *fs.Metadata) {
	if f.JSON {
		result := map[string]interface{}{
			"path":  path,
			"type":  string(meta.Type),
			"mode":  meta.Mode,
			"uid":   meta.UID,
			"gid":   meta.GID,
			"size":  meta.Size,
			"ctime": meta.CTime,
			"mtime": meta.MTime,
			"atime": meta.ATime,
		}
		if meta.LinkTarget != "" {
			result["link_target"] = meta.LinkTarget
		}
		f.PrintJSON(result)
		return
	}

	fmt.Fprintf(f.Writer, "  File: %s\n", path)
	fmt.Fprintf(f.Writer, "  Type: %s\n", meta.Type)
	fmt.Fprintf(f.Writer, "  Mode: %s (%s)\n", meta.ModeString(), meta.Mode)
	fmt.Fprintf(f.Writer, "   UID: %s\n", meta.UID)
	fmt.Fprintf(f.Writer, "   GID: %s\n", meta.GID)
	fmt.Fprintf(f.Writer, "  Size: %d\n", meta.Size)
	fmt.Fprintf(f.Writer, " CTime: %s\n", fs.FormatTime(meta.CTime))
	fmt.Fprintf(f.Writer, " MTime: %s\n", fs.FormatTime(meta.MTime))
	fmt.Fprintf(f.Writer, " ATime: %s\n", fs.FormatTime(meta.ATime))
	if meta.LinkTarget != "" {
		fmt.Fprintf(f.Writer, "  Link: %s\n", meta.LinkTarget)
	}
}

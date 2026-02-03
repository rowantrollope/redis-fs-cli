package output

import (
	"fmt"
	"io"
	"sort"

	"github.com/rowantrollope/redis-fs-cli/internal/fs"
)

// PrintTree renders a tree structure with Unicode box-drawing characters.
func (f *Formatter) PrintTree(entry *fs.TreeEntry, dirCount, fileCount int) {
	if f.JSON {
		f.PrintJSON(treeToJSON(entry))
		return
	}

	// Print root name
	fmt.Fprintln(f.Writer, f.FormatEntryName(entry.Name, entry.Type))

	// Print children
	printTreeChildren(f.Writer, f, entry.Children, "")

	// Summary
	fmt.Fprintf(f.Writer, "\n%d directories, %d files\n", dirCount, fileCount)
}

func printTreeChildren(w io.Writer, f *Formatter, children []fs.TreeEntry, prefix string) {
	// Sort children
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})

	for i, child := range children {
		isLast := i == len(children)-1

		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}

		name := f.FormatEntryName(child.Name, child.Type)
		fmt.Fprintf(w, "%s%s%s\n", prefix, connector, name)

		if child.Type == fs.TypeDir && len(child.Children) > 0 {
			printTreeChildren(w, f, child.Children, prefix+childPrefix)
		}
	}
}

func treeToJSON(entry *fs.TreeEntry) interface{} {
	result := map[string]interface{}{
		"name": entry.Name,
		"type": string(entry.Type),
	}
	if len(entry.Children) > 0 {
		children := make([]interface{}, len(entry.Children))
		for i, child := range entry.Children {
			children[i] = treeToJSON(&child)
		}
		result["children"] = children
	}
	return result
}

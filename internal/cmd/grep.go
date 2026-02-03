package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/rowantrollope/redis-fs-cli/internal/fs"
	"github.com/rowantrollope/redis-fs-cli/internal/search"
	flag "github.com/spf13/pflag"
)

func (r *Router) handleGrep(ctx context.Context, args []string) error {
	fset := flag.NewFlagSet("grep", flag.ContinueOnError)
	recursive := fset.BoolP("recursive", "r", false, "Search directories recursively")
	ignoreCase := fset.BoolP("ignore-case", "i", false, "Case insensitive matching")
	lineNumbers := fset.BoolP("line-number", "n", false, "Show line numbers")
	noIndex := fset.Bool("no-index", false, "Force scan-based search (skip index)")
	if err := fset.Parse(args); err != nil {
		return err
	}

	if fset.NArg() < 2 {
		return fmt.Errorf("grep: usage: grep [-r] [-i] [-n] [--no-index] pattern path")
	}

	pattern := fset.Arg(0)
	path := r.ResolvePath(fset.Arg(1))

	if *ignoreCase {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("grep: invalid pattern: %s", err)
	}

	meta, err := r.Client.Stat(ctx, path)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("grep: %s: No such file or directory", path)
	}

	// Try index-accelerated path for recursive directory grep
	if meta.Type == fs.TypeDir && *recursive && !*noIndex {
		if r.tryIndexedGrep(ctx, re, fset.Arg(0), path, *lineNumbers, *ignoreCase) {
			return nil
		}
	}

	// Fall back to scan-based grep
	if meta.Type == fs.TypeDir {
		if !*recursive {
			return fmt.Errorf("grep: %s: Is a directory", path)
		}
		return r.grepDir(ctx, re, path, *lineNumbers)
	}

	return r.grepFile(ctx, re, path, "", *lineNumbers)
}

// tryIndexedGrep attempts to use FT.SEARCH for grep. Returns true if successful.
func (r *Router) tryIndexedGrep(ctx context.Context, re *regexp.Regexp, rawPattern, dirPath string, lineNumbers, ignoreCase bool) bool {
	if !r.Config.SearchAvailable {
		return false
	}

	// Only use index for simple literal patterns
	if !search.IsSimplePattern(rawPattern) {
		return false
	}

	mgr := search.NewIndexManager(r.Client.Redis(), r.State.Volume)
	exists, err := mgr.IndexExists(ctx)
	if err != nil || !exists {
		return false
	}

	results, err := search.SearchFullText(ctx, r.Client.Redis(), mgr.IndexName(), rawPattern, dirPath, 10000)
	if err != nil {
		return false
	}

	// Post-filter with regex for exact line-level matching
	for _, result := range results {
		lines := strings.Split(result.Content, "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				display := result.Path + ":"
				if lineNumbers {
					display += fmt.Sprintf("%d:", i+1)
				}
				display += line
				fmt.Fprintln(r.Formatter.Writer, display)
			}
		}
	}

	return true
}

func (r *Router) grepFile(ctx context.Context, re *regexp.Regexp, path, prefix string, lineNumbers bool) error {
	content, err := r.Client.ReadFile(ctx, path)
	if err != nil {
		return err
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if re.MatchString(line) {
			display := ""
			if prefix != "" {
				display = prefix + ":"
			}
			if lineNumbers {
				display += fmt.Sprintf("%d:", i+1)
			}
			display += line
			fmt.Fprintln(r.Formatter.Writer, display)
		}
	}
	return nil
}

func (r *Router) grepDir(ctx context.Context, re *regexp.Regexp, dirPath string, lineNumbers bool) error {
	entries, err := r.Client.Find(ctx, dirPath, "", "f")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := r.grepFile(ctx, re, entry.Path, entry.Path, lineNumbers); err != nil {
			// Continue on individual file errors
			continue
		}
	}
	return nil
}

package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/rowantrollope/redis-fs-cli/internal/fs"
	flag "github.com/spf13/pflag"
)

func (r *Router) handleGrep(ctx context.Context, args []string) error {
	fset := flag.NewFlagSet("grep", flag.ContinueOnError)
	recursive := fset.BoolP("recursive", "r", false, "Search directories recursively")
	ignoreCase := fset.BoolP("ignore-case", "i", false, "Case insensitive matching")
	lineNumbers := fset.BoolP("line-number", "n", false, "Show line numbers")
	if err := fset.Parse(args); err != nil {
		return err
	}

	if fset.NArg() < 2 {
		return fmt.Errorf("grep: usage: grep [-r] [-i] [-n] pattern path")
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

	if meta.Type == fs.TypeDir {
		if !*recursive {
			return fmt.Errorf("grep: %s: Is a directory", path)
		}
		return r.grepDir(ctx, re, path, *lineNumbers)
	}

	return r.grepFile(ctx, re, path, "", *lineNumbers)
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

package cli

import (
	"context"
	"sort"
	"strings"

	"github.com/chzyer/readline"
	"github.com/rowantrollope/redis-fs-cli/internal/cmd"
	"github.com/rowantrollope/redis-fs-cli/internal/fs"
)

// Common Redis commands for completion
var commonRedisCommands = []string{
	"GET", "SET", "DEL", "EXISTS", "KEYS", "SCAN", "TYPE",
	"HGET", "HSET", "HGETALL", "HDEL", "HMSET",
	"SADD", "SMEMBERS", "SREM", "SCARD",
	"LPUSH", "RPUSH", "LRANGE", "LLEN",
	"ZADD", "ZRANGE", "ZRANGEBYSCORE",
	"EXPIRE", "TTL", "PERSIST",
	"INFO", "DBSIZE", "FLUSHDB", "PING",
	"SELECT", "CONFIG", "CLIENT",
	"SUBSCRIBE", "PUBLISH",
	"MULTI", "EXEC", "DISCARD",
}

// NewCompleter creates a tab completer for the REPL.
func NewCompleter(router *cmd.Router, fsClient *fs.Client) *Completer {
	return &Completer{
		router:   router,
		fsClient: fsClient,
	}
}

// Completer provides tab completion for the REPL.
type Completer struct {
	router   *cmd.Router
	fsClient *fs.Client
}

// Do implements readline.AutoCompleter.
func (c *Completer) Do(line []rune, pos int) ([][]rune, int) {
	lineStr := string(line[:pos])
	parts := strings.Fields(lineStr)

	// Complete command name
	if len(parts) == 0 || (len(parts) == 1 && !strings.HasSuffix(lineStr, " ")) {
		prefix := ""
		if len(parts) == 1 {
			prefix = parts[0]
		}
		return c.completeCommand(prefix), len(prefix)
	}

	// Complete path argument
	partial := ""
	if !strings.HasSuffix(lineStr, " ") {
		partial = parts[len(parts)-1]
	}

	// Skip flag-like args
	if strings.HasPrefix(partial, "-") {
		return nil, 0
	}

	return c.completePath(partial), len(partial)
}

func (c *Completer) completeCommand(prefix string) [][]rune {
	var candidates []string

	// FS commands
	for _, name := range c.router.CommandNames() {
		if strings.HasPrefix(name, strings.ToLower(prefix)) {
			candidates = append(candidates, name)
		}
	}

	// Common Redis commands
	for _, name := range commonRedisCommands {
		if strings.HasPrefix(strings.ToUpper(name), strings.ToUpper(prefix)) {
			candidates = append(candidates, name)
		}
	}

	sort.Strings(candidates)

	result := make([][]rune, len(candidates))
	for i, c := range candidates {
		suffix := c[len(prefix):]
		result[i] = []rune(suffix + " ")
	}
	return result
}

func (c *Completer) completePath(partial string) [][]rune {
	ctx := context.Background()

	// Determine the directory to list and the prefix to match
	dir := c.router.State.Cwd
	prefix := partial

	if strings.Contains(partial, "/") {
		lastSlash := strings.LastIndex(partial, "/")
		dirPart := partial[:lastSlash+1]
		prefix = partial[lastSlash+1:]
		dir = fs.ResolvePath(c.router.State.Cwd, dirPart)
	}

	children, err := c.fsClient.ReadDirWithMeta(ctx, dir)
	if err != nil {
		return nil
	}

	var candidates [][]rune
	for _, child := range children {
		if strings.HasPrefix(child.Name, prefix) {
			suffix := child.Name[len(prefix):]
			if child.Meta != nil && child.Meta.Type == fs.TypeDir {
				suffix += "/"
			} else {
				suffix += " "
			}
			candidates = append(candidates, []rune(suffix))
		}
	}
	return candidates
}

// Ensure Completer satisfies the readline.AutoCompleter interface.
var _ readline.AutoCompleter = (*Completer)(nil)

package cmd

import (
	"context"
	"fmt"
)

var commandHelp = map[string]string{
	"ls":    "ls [path] [-l] [-a]       List directory contents",
	"pwd":   "pwd                       Print working directory",
	"cd":    "cd [path]                 Change directory (cd - for previous)",
	"mkdir": "mkdir [-p] path           Create directory (-p for parents)",
	"rmdir": "rmdir path                Remove empty directory",
	"touch": "touch path                Create file or update timestamps",
	"cat":   "cat path                  Display file contents",
	"echo":  "echo \"text\" > path        Write to file (> or >> for append)",
	"rm":    "rm [-r] [-f] path         Remove file or directory",
	"cp":    "cp [-r] src dst           Copy file or directory",
	"mv":    "mv src dst                Move/rename file or directory",
	"stat":  "stat path                 Display file metadata",
	"find":  "find [path] [-name pat] [-type f|d|l]  Find files",
	"grep":  "grep [-r] [-i] [-n] pattern path       Search file contents",
	"ln":    "ln -s target link         Create symbolic link",
	"chmod": "chmod mode path           Change file mode",
	"chown": "chown uid:gid path        Change file owner",
	"tree":  "tree [path] [-L depth]    Display directory tree",
	"vol":   "vol list|switch|create|info  Volume management",
	"init":  "init                      Initialize volume root",
	"help":  "help [command]            Show this help",
	"clear": "clear                     Clear the terminal",
	"exit":  "exit / quit               Exit the REPL",
}

func (r *Router) handleHelp(ctx context.Context, args []string) error {
	if len(args) > 0 {
		cmd := args[0]
		if help, ok := commandHelp[cmd]; ok {
			fmt.Fprintln(r.Formatter.Writer, help)
		} else {
			fmt.Fprintf(r.Formatter.Writer, "No help available for '%s'\n", cmd)
		}
		return nil
	}

	fmt.Fprintln(r.Formatter.Writer, "redis-fs-cli — POSIX-like filesystem on Redis")
	fmt.Fprintln(r.Formatter.Writer, "")
	fmt.Fprintln(r.Formatter.Writer, "Filesystem commands:")
	for _, cmd := range []string{"ls", "pwd", "cd", "mkdir", "rmdir", "touch", "cat", "echo",
		"rm", "cp", "mv", "stat", "find", "grep", "ln", "chmod", "chown", "tree"} {
		fmt.Fprintf(r.Formatter.Writer, "  %s\n", commandHelp[cmd])
	}
	fmt.Fprintln(r.Formatter.Writer, "")
	fmt.Fprintln(r.Formatter.Writer, "Volume commands:")
	fmt.Fprintf(r.Formatter.Writer, "  %s\n", commandHelp["vol"])
	fmt.Fprintf(r.Formatter.Writer, "  %s\n", commandHelp["init"])
	fmt.Fprintln(r.Formatter.Writer, "")
	fmt.Fprintln(r.Formatter.Writer, "Other:")
	fmt.Fprintf(r.Formatter.Writer, "  %s\n", commandHelp["help"])
	fmt.Fprintf(r.Formatter.Writer, "  %s\n", commandHelp["clear"])
	fmt.Fprintf(r.Formatter.Writer, "  %s\n", commandHelp["exit"])
	fmt.Fprintln(r.Formatter.Writer, "")
	fmt.Fprintln(r.Formatter.Writer, "Any unrecognized command is passed through to redis-cli.")
	return nil
}

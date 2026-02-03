# redis-fs-cli

A command-line tool that provides a POSIX-like filesystem interface on top of Redis. Browse, create, and manage files and directories stored entirely in Redis using familiar shell commands like `ls`, `cd`, `cat`, `cp`, and more.

## Features

- Interactive shell with tab completion, command history, and a dynamic prompt
- Single-command mode for scripting and automation
- Multiple independent volumes (namespaced filesystems)
- Symbolic links with chain resolution
- POSIX-style permissions and ownership
- JSON output mode for programmatic use
- Transparent passthrough to `redis-cli` for native Redis commands
- TLS support

## Installation

### Build from source

Requires Go 1.21+.

```bash
git clone https://github.com/yourorg/redis-fs-cli.git
cd redis-fs-cli
go build -o redis-fs-cli ./cmd/redis-fs-cli
```

## Quick Start

```bash
# Connect to local Redis and start interactive shell
redis-fs-cli

# Connect to a specific host/port
redis-fs-cli -h myredis.example.com -p 6380

# Run a single command
redis-fs-cli ls /

# Use a Redis URI
redis-fs-cli -u redis://user:pass@host:6379/0
```

## Connection Options

| Flag | Description | Default |
|------|-------------|---------|
| `-h, --host` | Redis server hostname | `127.0.0.1` |
| `-p, --port` | Redis server port | `6379` |
| `-s, --socket` | Unix socket path | |
| `-a, --password` | Authentication password | |
| `-n, --db` | Database number | `0` |
| `-u, --uri` | Redis URI (`redis://...`) | |
| `--tls` | Enable TLS | `false` |
| `--cacert` | CA certificate file for TLS | |
| `--cert` | Client certificate file for TLS | |
| `--key` | Client key file for TLS | |
| `--volume` | Active volume name | `main` |
| `--json` | Enable JSON output | `false` |
| `--no-color` | Disable colored output | `false` |

The password can also be set via the `REDISCLI_AUTH` environment variable.

## Commands

### Navigation

```bash
pwd                  # Print working directory
cd /path/to/dir      # Change directory
cd -                 # Switch to previous directory
cd                   # Go to root
ls                   # List current directory
ls -l                # Long format with permissions, size, timestamps
ls -a /some/path     # Show all entries at a specific path
```

### File Operations

```bash
touch myfile.txt              # Create an empty file
echo "hello world" > file.txt # Write text to a file
echo "more text" >> file.txt  # Append text to a file
cat file.txt                  # Display file contents
cp source.txt dest.txt        # Copy a file
cp -r srcdir/ dstdir/         # Copy a directory recursively
mv old.txt new.txt            # Move or rename
rm file.txt                   # Remove a file
rm -r mydir                   # Remove a directory recursively
rm -rf mydir                  # Force remove (ignore if missing)
```

### Directory Operations

```bash
mkdir newdir          # Create a directory
mkdir -p a/b/c        # Create nested directories
rmdir emptydir        # Remove an empty directory
```

### Search and Inspect

```bash
stat file.txt                     # Show detailed metadata
find / -name "*.txt"              # Find files by name pattern
find /data -type d                # Find directories only
find /data -type f -name "log*"   # Combine filters
tree                              # Display directory tree
tree /data -L 2                   # Tree with depth limit
grep "pattern" file.txt           # Search in a file
grep -r "TODO" /src               # Recursive search
grep -in "error" log.txt          # Case-insensitive with line numbers
```

### Permissions and Ownership

```bash
chmod 0755 script.sh      # Set file mode (octal)
chown 1000:1000 file.txt  # Set uid:gid
chown 1000: file.txt      # Set uid only
chown :1000 file.txt      # Set gid only
```

### Symbolic Links

```bash
ln -s /target/path linkname   # Create a symbolic link
cat linkname                  # Reading follows the link
stat linkname                 # Shows link metadata
```

### Volume Management

Volumes are independent, namespaced filesystems within the same Redis database.

```bash
vol list             # List all volumes (* marks current)
vol create staging   # Create and switch to a new volume
vol switch main      # Switch to an existing volume
vol info             # Show current volume and working directory
```

### Other

```bash
init           # Reinitialize the current volume root
help           # Show all available commands
help ls        # Show help for a specific command
clear          # Clear the terminal
exit           # Exit the shell (also: quit)
```

### Redis Passthrough

Any command not recognized as a filesystem command is forwarded to `redis-cli` with your connection settings:

```bash
redis-fs:main:/> PING
PONG
redis-fs:main:/> SET mykey myvalue
OK
redis-fs:main:/> KEYS fs:*
...
```

## Interactive Shell

When started without a command argument, `redis-fs-cli` launches an interactive REPL with:

- **Dynamic prompt** showing the current volume and path: `redis-fs:main:/data>`
- **Tab completion** for commands and filesystem paths
- **Command history** persisted to `~/.redis-fs-cli_history` (configurable via `REDIS_FS_HISTORY`)
- **Quoted arguments** and escape sequences

## JSON Output

Use `--json` for machine-readable output:

```bash
redis-fs-cli --json stat /myfile.txt
redis-fs-cli --json ls /
```

## How Data is Stored in Redis

Each volume uses a set of namespaced keys:

| Key Pattern | Type | Purpose |
|---|---|---|
| `fs:{volume}:meta:{path}` | Hash | Entry metadata (type, mode, size, timestamps, ...) |
| `fs:{volume}:data:{path}` | String | File content |
| `fs:{volume}:dir:{path}` | Set | Child entry names for directories |
| `fs:{volume}:xattr:{path}` | Hash | Extended attributes |

## Environment Variables

| Variable | Description |
|---|---|
| `REDISCLI_AUTH` | Redis authentication password |
| `REDIS_FS_VOLUME` | Default volume name (default: `main`) |
| `REDIS_FS_HISTORY` | History file path (default: `~/.redis-fs-cli_history`) |
| `NO_COLOR` | Disable colored output when set |

## License

See [LICENSE](LICENSE) for details.

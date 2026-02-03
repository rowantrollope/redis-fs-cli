package config

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
	flag "github.com/spf13/pflag"
)

// Config holds all connection and runtime configuration.
type Config struct {
	Host     string
	Port     int
	Socket   string
	Password string
	DB       int
	URI      string

	TLS    bool
	CACert string
	Cert   string
	Key    string

	Volume  string
	JSON    bool
	NoColor bool
	Color   bool

	HistoryFile string

	// Search / indexing
	SearchAvailable bool   // set at startup, not a flag
	EmbeddingAPIKey string
	EmbeddingAPIURL string
	EmbeddingModel  string
	EmbeddingDim    int

	// Remaining args after flag parsing (single-command mode)
	Args []string
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	histFile := home + "/.redis-fs-cli_history"
	if env := os.Getenv("REDIS_FS_HISTORY"); env != "" {
		histFile = env
	}

	volume := "main"
	if env := os.Getenv("REDIS_FS_VOLUME"); env != "" {
		volume = env
	}

	password := ""
	if env := os.Getenv("REDISCLI_AUTH"); env != "" {
		password = env
	}

	embeddingKey := os.Getenv("EMBEDDING_API_KEY")
	embeddingURL := os.Getenv("EMBEDDING_API_URL")
	if embeddingURL == "" {
		embeddingURL = "https://api.openai.com/v1"
	}
	embeddingModel := os.Getenv("EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = "text-embedding-3-small"
	}
	embeddingDim := 1536
	if env := os.Getenv("EMBEDDING_DIM"); env != "" {
		if d, err := strconv.Atoi(env); err == nil && d > 0 {
			embeddingDim = d
		}
	}

	return &Config{
		Host:            "127.0.0.1",
		Port:            6379,
		DB:              0,
		Password:        password,
		Volume:          volume,
		HistoryFile:     histFile,
		EmbeddingAPIKey: embeddingKey,
		EmbeddingAPIURL: embeddingURL,
		EmbeddingModel:  embeddingModel,
		EmbeddingDim:    embeddingDim,
	}
}

// RegisterFlags registers CLI flags on the given flag set.
func (c *Config) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVarP(&c.Host, "host", "h", c.Host, "Server hostname")
	fs.IntVarP(&c.Port, "port", "p", c.Port, "Server port")
	fs.StringVarP(&c.Socket, "socket", "s", c.Socket, "Unix socket path")
	fs.StringVarP(&c.Password, "password", "a", c.Password, "Password")
	fs.IntVarP(&c.DB, "db", "n", c.DB, "Database number")
	fs.StringVarP(&c.URI, "uri", "u", c.URI, "Server URI (redis://...)")

	fs.BoolVar(&c.TLS, "tls", false, "Enable TLS")
	fs.StringVar(&c.CACert, "cacert", "", "CA certificate file")
	fs.StringVar(&c.Cert, "cert", "", "Client certificate file")
	fs.StringVar(&c.Key, "key", "", "Client key file")

	fs.BoolVar(&c.JSON, "json", false, "JSON output mode")
	fs.BoolVar(&c.NoColor, "no-color", false, "Disable colors")
	fs.BoolVar(&c.Color, "color", false, "Force colors")
	fs.StringVar(&c.Volume, "volume", c.Volume, "Filesystem volume name")

	fs.StringVar(&c.EmbeddingAPIKey, "embedding-api-key", c.EmbeddingAPIKey, "API key for embedding model")
	fs.StringVar(&c.EmbeddingAPIURL, "embedding-api-url", c.EmbeddingAPIURL, "Base URL for embedding API")
	fs.StringVar(&c.EmbeddingModel, "embedding-model", c.EmbeddingModel, "Embedding model name")
	fs.IntVar(&c.EmbeddingDim, "embedding-dim", c.EmbeddingDim, "Embedding vector dimension")
}

// RedisOptions builds a go-redis Options from the config.
func (c *Config) RedisOptions() *redis.Options {
	if c.URI != "" {
		opts, err := redis.ParseURL(c.URI)
		if err == nil {
			if c.DB != 0 {
				opts.DB = c.DB
			}
			return opts
		}
	}

	addr := c.Host + ":" + strconv.Itoa(c.Port)
	opts := &redis.Options{
		Addr:     addr,
		Password: c.Password,
		DB:       c.DB,
	}

	if c.Socket != "" {
		opts.Network = "unix"
		opts.Addr = c.Socket
	}

	if c.TLS {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: false,
		}
	}

	return opts
}

// RedisCLIArgs returns the connection arguments to pass to redis-cli for passthrough.
func (c *Config) RedisCLIArgs() []string {
	var args []string
	if c.URI != "" {
		args = append(args, "-u", c.URI)
	} else {
		if c.Socket != "" {
			args = append(args, "-s", c.Socket)
		} else {
			if c.Host != "127.0.0.1" {
				args = append(args, "-h", c.Host)
			}
			if c.Port != 6379 {
				args = append(args, "-p", strconv.Itoa(c.Port))
			}
		}
	}
	if c.Password != "" {
		args = append(args, "-a", c.Password)
	}
	if c.DB != 0 {
		args = append(args, "-n", strconv.Itoa(c.DB))
	}
	if c.TLS {
		args = append(args, "--tls")
		if c.CACert != "" {
			args = append(args, "--cacert", c.CACert)
		}
		if c.Cert != "" {
			args = append(args, "--cert", c.Cert)
		}
		if c.Key != "" {
			args = append(args, "--key", c.Key)
		}
	}
	return args
}

// Addr returns a display-friendly connection address.
func (c *Config) Addr() string {
	if c.Socket != "" {
		return c.Socket
	}
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// ShouldColor returns true if color output should be enabled.
func (c *Config) ShouldColor() bool {
	if c.NoColor {
		return false
	}
	if c.Color {
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return true
}

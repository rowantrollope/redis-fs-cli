package embedding

// Config holds embedding API configuration.
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
	Dim     int
}

// IsConfigured returns true if an API key is set.
func (c *Config) IsConfigured() bool {
	return c.APIKey != ""
}

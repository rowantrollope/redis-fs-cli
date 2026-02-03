package search

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// DetectSearch checks if Redis search (FT.*) commands are available.
// Returns true for Redis 8.0+ (built-in) or older Redis with RediSearch module.
func DetectSearch(ctx context.Context, rdb *redis.Client) bool {
	_, err := rdb.Do(ctx, "FT._LIST").Result()
	return err == nil
}

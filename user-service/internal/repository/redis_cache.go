package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9" // Using go-redis
	"github.com/zhandarbeks/petstore-final-project/user-service/internal/domain" // Adjust import path
)

// redisUserCache is the Redis implementation of UserCache
type redisUserCache struct {
	client *redis.Client
	prefix string // e.g., "usercache:" to namespace keys
}

// NewRedisUserCache creates a new instance of redisUserCache.
func NewRedisUserCache(ctx context.Context, addr, password string, db int, keyPrefix string) (UserCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // no password set
		DB:       db,       // use default DB
	})

	// Ping to check connection
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Printf("Error connecting to Redis: %v", err)
		return nil, err
	}
	log.Println("Successfully connected to Redis!")

	if keyPrefix == "" {
		keyPrefix = "user:" // Default prefix
	}

	return &redisUserCache{
		client: rdb,
		prefix: keyPrefix,
	}, nil
}

// Close closes the Redis client connection.
func (c *redisUserCache) Close() error {
	if c.client != nil {
		log.Println("Closing Redis client connection...")
		return c.client.Close()
	}
	return nil
}

func (c *redisUserCache) userKey(id string) string {
	return fmt.Sprintf("%s%s", c.prefix, id)
}

// GetUser retrieves a user from the cache.
func (c *redisUserCache) GetUser(ctx context.Context, id string) (*domain.User, error) {
	key := c.userKey(id)
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("user not found in cache") // Or return a specific cache miss error
		}
		log.Printf("Error getting user from Redis cache (key: %s): %v", key, err)
		return nil, err
	}

	var user domain.User
	err = json.Unmarshal([]byte(val), &user)
	if err != nil {
		log.Printf("Error unmarshalling user data from Redis (key: %s): %v", key, err)
		// Potentially delete the malformed key from cache
		// c.client.Del(ctx, key)
		return nil, err
	}
	return &user, nil
}

// SetUser stores a user in the cache with an expiration.
func (c *redisUserCache) SetUser(ctx context.Context, id string, user *domain.User, expiration time.Duration) error {
	key := c.userKey(id)
	data, err := json.Marshal(user)
	if err != nil {
		log.Printf("Error marshalling user data for Redis cache (key: %s): %v", key, err)
		return err
	}

	err = c.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		log.Printf("Error setting user in Redis cache (key: %s): %v", key, err)
		return err
	}
	return nil
}

// DeleteUser removes a user from the cache.
func (c *redisUserCache) DeleteUser(ctx context.Context, id string) error {
	key := c.userKey(id)
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		// If key doesn't exist, Del doesn't error, it returns 0.
		// So, this error is likely a connection issue or similar.
		log.Printf("Error deleting user from Redis cache (key: %s): %v", key, err)
		return err
	}
	return nil
}

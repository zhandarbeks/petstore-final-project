package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/domain" // Adjust import path
)

type redisAdoptionCache struct {
	client *redis.Client
	prefix string // e.g., "adoptioncache:"
}

// NewRedisAdoptionCache creates a new instance of redisAdoptionCache.
func NewRedisAdoptionCache(ctx context.Context, addr, password string, db int, keyPrefix string) (AdoptionCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Printf("Adoption Service | Error connecting to Redis: %v", err)
		return nil, err
	}
	log.Println("Adoption Service | Successfully connected to Redis!")

	if keyPrefix == "" {
		keyPrefix = "adoptionapp:" // Default prefix for adoption application cache keys
	}

	return &redisAdoptionCache{
		client: rdb,
		prefix: keyPrefix,
	}, nil
}

func (c *redisAdoptionCache) Close() error {
	if c.client != nil {
		log.Println("Adoption Service | Closing Redis client connection...")
		return c.client.Close()
	}
	return nil
}

func (c *redisAdoptionCache) appKey(id string) string {
	return fmt.Sprintf("%s%s", c.prefix, id)
}

func (c *redisAdoptionCache) GetAdoptionApplication(ctx context.Context, id string) (*domain.AdoptionApplication, error) {
	key := c.appKey(id)
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("adoption application not found in cache")
		}
		log.Printf("Adoption Service | Error getting application from Redis cache (key: %s): %v", key, err)
		return nil, err
	}

	var app domain.AdoptionApplication
	err = json.Unmarshal([]byte(val), &app)
	if err != nil {
		log.Printf("Adoption Service | Error unmarshalling application data from Redis (key: %s): %v", key, err)
		return nil, err
	}
	return &app, nil
}

func (c *redisAdoptionCache) SetAdoptionApplication(ctx context.Context, id string, app *domain.AdoptionApplication, expiration time.Duration) error {
	key := c.appKey(id)
	data, err := json.Marshal(app)
	if err != nil {
		log.Printf("Adoption Service | Error marshalling application data for Redis cache (key: %s): %v", key, err)
		return err
	}

	err = c.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		log.Printf("Adoption Service | Error setting application in Redis cache (key: %s): %v", key, err)
		return err
	}
	return nil
}

func (c *redisAdoptionCache) DeleteAdoptionApplication(ctx context.Context, id string) error {
	key := c.appKey(id)
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Adoption Service | Error deleting application from Redis cache (key: %s): %v", key, err)
		return err
	}
	return nil
}
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/domain" // Adjust import path
)

type redisPetCache struct {
	client *redis.Client
	prefix string // e.g., "petcache:"
}

// NewRedisPetCache creates a new instance of redisPetCache.
func NewRedisPetCache(ctx context.Context, addr, password string, db int, keyPrefix string) (PetCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Printf("Pet Service | Error connecting to Redis: %v", err)
		return nil, err
	}
	log.Println("Pet Service | Successfully connected to Redis!")

	if keyPrefix == "" {
		keyPrefix = "pet:" // Default prefix for pet cache keys
	}

	return &redisPetCache{
		client: rdb,
		prefix: keyPrefix,
	}, nil
}

func (c *redisPetCache) Close() error {
	if c.client != nil {
		log.Println("Pet Service | Closing Redis client connection...")
		return c.client.Close()
	}
	return nil
}

func (c *redisPetCache) petKey(id string) string {
	return fmt.Sprintf("%s%s", c.prefix, id)
}

func (c *redisPetCache) GetPet(ctx context.Context, id string) (*domain.Pet, error) {
	key := c.petKey(id)
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("pet not found in cache")
		}
		log.Printf("Pet Service | Error getting pet from Redis cache (key: %s): %v", key, err)
		return nil, err
	}

	var pet domain.Pet
	err = json.Unmarshal([]byte(val), &pet)
	if err != nil {
		log.Printf("Pet Service | Error unmarshalling pet data from Redis (key: %s): %v", key, err)
		return nil, err
	}
	return &pet, nil
}

func (c *redisPetCache) SetPet(ctx context.Context, id string, pet *domain.Pet, expiration time.Duration) error {
	key := c.petKey(id)
	data, err := json.Marshal(pet)
	if err != nil {
		log.Printf("Pet Service | Error marshalling pet data for Redis cache (key: %s): %v", key, err)
		return err
	}

	err = c.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		log.Printf("Pet Service | Error setting pet in Redis cache (key: %s): %v", key, err)
		return err
	}
	return nil
}

func (c *redisPetCache) DeletePet(ctx context.Context, id string) error {
	key := c.petKey(id)
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Pet Service | Error deleting pet from Redis cache (key: %s): %v", key, err)
		return err
	}
	return nil
}

// Example for caching lists (implement if needed)
/*
func (c *redisPetCache) listCacheKey(filters map[string]interface{}, page, limit int) string {
	// Create a stable key based on filters, page, and limit
	// This can be complex; consider hashing or a canonical string representation.
	// For simplicity, a basic example:
	var filterParts []string
	for k, v := range filters {
		filterParts = append(filterParts, fmt.Sprintf("%s=%v", k, v))
	}
	sort.Strings(filterParts) // Ensure order for consistency
	return fmt.Sprintf("%slist:%s:page%d:limit%d", c.prefix, strings.Join(filterParts, "&"), page, limit)
}

func (c *redisPetCache) SetListedPets(ctx context.Context, cacheKey string, pets []*domain.Pet, expiration time.Duration) error {
	data, err := json.Marshal(pets)
	if err != nil {
		log.Printf("Pet Service | Error marshalling listed pets for Redis cache (key: %s): %v", cacheKey, err)
		return err
	}
	err = c.client.Set(ctx, cacheKey, data, expiration).Err()
	if err != nil {
		log.Printf("Pet Service | Error setting listed pets in Redis cache (key: %s): %v", cacheKey, err)
	}
	return err
}

func (c *redisPetCache) GetListedPets(ctx context.Context, cacheKey string) ([]*domain.Pet, error) {
	val, err := c.client.Get(ctx, cacheKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("listed pets not found in cache")
		}
		log.Printf("Pet Service | Error getting listed pets from Redis cache (key: %s): %v", cacheKey, err)
		return nil, err
	}
	var pets []*domain.Pet
	err = json.Unmarshal([]byte(val), &pets)
	if err != nil {
		log.Printf("Pet Service | Error unmarshalling listed pets from Redis (key: %s): %v", cacheKey, err)
		return nil, err
	}
	return pets, nil
}

func (c *redisPetCache) DeleteListedPets(ctx context.Context, cacheKey string) error {
	// More robust: delete all keys matching a pattern if filters change, e.g., using SCAN and DEL.
	// For simplicity, just deleting a specific key.
	err := c.client.Del(ctx, cacheKey).Err()
	if err != nil {
		log.Printf("Pet Service | Error deleting listed pets from Redis cache (key: %s): %v", cacheKey, err)
	}
	return err
}
*/
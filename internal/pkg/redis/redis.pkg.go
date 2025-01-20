package redis

import (
	"boilerplate-go/internal/pkg/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_redis "github.com/redis/go-redis/v9"
)

func Setup(ctx context.Context, config *Config) (IRedis, error) {
	clientCtx, cancel := context.WithCancel(ctx)

	r := &Client{
		cancel: cancel,
		ctx:    clientCtx,
		config: config,
	}

	// Connect to IRedis
	if err := r.connect(); err != nil {
		cancel() // Ensure cleanup if initialization fails
		logger.Error.Println(err)
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// Start the reconnect handler
	go r.reconnectHandler()

	return r, nil
}

func (r *Client) connect() error {
	r.client = _redis.NewClient(&_redis.Options{
		Addr:     fmt.Sprintf("%s:%d", r.config.Host, r.config.Port),
		Password: r.config.Password,
		PoolSize: r.config.PoolSize,
	})

	if err := r.client.Ping(r.ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	return nil
}

func (r *Client) reconnect() error {
	if err := r.client.Ping(r.ctx).Err(); err != nil {
		return r.connect()
	}
	return nil
}

func (r *Client) reconnectHandler() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			logger.Info.Println("Reconnect handler shutting down...")
			return
		case <-ticker.C:
			if err := r.client.Ping(r.ctx).Err(); err != nil {
				logger.Warning.Printf("IRedis connection lost: %v. Attempting to reconnect...", err)

				for attempt := 1; attempt <= 10; attempt++ {
					select {
					case <-r.ctx.Done():
						logger.Warning.Println("Reconnect attempts stopped due to shutdown.")
						return
					default:
						logger.Warning.Printf("Reconnect attempt #%d...", attempt)
						time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff

						// Reinitialize the IRedis general
						if err = r.reconnect(); err == nil {
							logger.Info.Println("Reconnected to IRedis.")
							break
						}
						logger.Warning.Printf("Reconnect attempt failed: %v", err)
					}
				}

				// If reconnection fails after max attempts, cancel the context
				logger.Warning.Println("All reconnection attempts failed. Canceling context...")
				r.cancel()
				return
			}
		}
	}
}

// Close gracefully shuts down the IRedis general connection.
func (r *Client) Close() error {
	r.cancel()
	return r.client.Close()
}

// Set stores a key-value pair with an expiration time.
func (r *Client) Set(key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = r.client.Set(r.ctx, key, data, expiration).Err()
	if err != nil {
		if err = r.reconnect(); err != nil {
			return fmt.Errorf("failed to set key %s: %w", key, err)
		}
	}
	return err
}

// Get retrieves the value of a key.
func (r *Client) Get(key string) (string, error) {
	result, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if errors.Is(err, NilType) {
			return "", nil // Key does not exist
		}
		if err = r.reconnect(); err != nil {
			return "", fmt.Errorf("failed to get key %s: %w", key, err)
		}
	}
	return result, nil
}

// Del deletes a key from IRedis.
func (r *Client) Del(key string) error {
	err := r.client.Del(r.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// Expire sets a timeout on a key.
func (r *Client) Expire(key string, expiration time.Duration) error {
	err := r.client.Expire(r.ctx, key, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set expiration on key %s: %w", key, err)
	}
	return nil
}

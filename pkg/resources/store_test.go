package resources

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryConfigStore(t *testing.T) {
	store := NewInMemoryConfigStore()
	testConfigStore(t, store)
}

func TestRedisConfigStore(t *testing.T) {
	// Skip if Redis is not available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skip("Redis not available")
	}
	client.Close()

	store, err := NewRedisConfigStore(RedisConfigStoreConfig{
		Address:   "localhost:6379",
		KeyPrefix: "test:config:",
	})
	require.NoError(t, err)
	defer store.Close()

	testConfigStore(t, store)
}

func testConfigStore(t *testing.T, store interface {
	Load(ctx context.Context, serviceType ServiceType) (*ServiceConfig, error)
	Save(ctx context.Context, config ServiceConfig) error
	List(ctx context.Context) ([]ServiceConfig, error)
	Watch(ctx context.Context) (<-chan ServiceConfig, error)
}) {
	ctx := context.Background()

	// Test Save and Load
	t.Run("Save and Load", func(t *testing.T) {
		config := ServiceConfig{
			Type:    "test-service",
			Version: "1.0.0",
			Timeout: 5 * time.Second,
		}

		err := store.Save(ctx, config)
		require.NoError(t, err)

		loaded, err := store.Load(ctx, config.Type)
		require.NoError(t, err)
		assert.Equal(t, config.Type, loaded.Type)
		assert.Equal(t, config.Version, loaded.Version)
		assert.Equal(t, config.Timeout, loaded.Timeout)
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		configs := []ServiceConfig{
			{
				Type:    "service-1",
				Version: "1.0.0",
			},
			{
				Type:    "service-2",
				Version: "1.0.0",
			},
		}

		for _, cfg := range configs {
			err := store.Save(ctx, cfg)
			require.NoError(t, err)
		}

		list, err := store.List(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), len(configs))

		found := make(map[ServiceType]bool)
		for _, cfg := range list {
			found[cfg.Type] = true
		}
		for _, cfg := range configs {
			assert.True(t, found[cfg.Type], "Config %s not found in list", cfg.Type)
		}
	})

	// Test Watch
	t.Run("Watch", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		watch, err := store.Watch(ctx)
		require.NoError(t, err)

		config := ServiceConfig{
			Type:    "watched-service",
			Version: "1.0.0",
		}

		done := make(chan bool)
		go func() {
			select {
			case updated := <-watch:
				assert.Equal(t, config.Type, updated.Type)
				assert.Equal(t, config.Version, updated.Version)
				// No Environment field to check
				done <- true
			case <-time.After(5 * time.Second):
				t.Error("Watch timeout")
				done <- false
			}
		}()

		err = store.Save(ctx, config)
		require.NoError(t, err)

		success := <-done
		assert.True(t, success)
	})
}

func TestRedisConfigStore_InvalidConnection(t *testing.T) {
	store, err := NewRedisConfigStore(RedisConfigStoreConfig{
		Address:   "localhost:12345", // Invalid port
		KeyPrefix: "test:config:",
	})
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	_, err = store.Load(ctx, "test")
	assert.Error(t, err)

	err = store.Save(ctx, ServiceConfig{Type: "test"})
	assert.Error(t, err)

	_, err = store.List(ctx)
	assert.Error(t, err)
}

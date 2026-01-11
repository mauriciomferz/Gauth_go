package revocation

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// EmergencyRevocationOracle provides sub-second revocation via centralized broadcast
type EmergencyRevocationOracle struct {
	redis       *redis.ClusterClient
	subscribers map[string]chan *RevocationEvent
	mu          sync.RWMutex
	logger      Logger
}

// RevocationEvent represents a PoA revocation
type RevocationEvent struct {
	PoAID     string    `json:"poa_id"`
	Principal string    `json:"principal"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
	TTL       int64     `json:"ttl"`
	EventID   string    `json:"event_id"`
}

// NewEmergencyOracle creates a new emergency revocation oracle
func NewEmergencyOracle(redisAddrs []string, logger Logger) (*EmergencyRevocationOracle, error) {
	if len(redisAddrs) == 0 {
		return nil, fmt.Errorf("at least one Redis address required")
	}

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:           redisAddrs,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        100,
		MinIdleConns:    10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis cluster ping failed: %w", err)
	}

	oracle := &EmergencyRevocationOracle{
		redis:       rdb,
		subscribers: make(map[string]chan *RevocationEvent),
		logger:      logger,
	}

	logger.Info("Emergency Revocation Oracle initialized successfully")
	return oracle, nil
}

// EmergencyRevoke immediately suspends a PoA across all validators
func (o *EmergencyRevocationOracle) EmergencyRevoke(ctx context.Context, event *RevocationEvent) error {
	start := time.Now()

	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.TTL == 0 {
		event.TTL = 86400
	}

	o.logger.Infof("Initiating emergency revocation for PoA %s (event: %s)", event.PoAID, event.EventID)

	key := fmt.Sprintf("revoked:%s", event.PoAID)
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := o.redis.Set(ctx, key, eventJSON, time.Duration(event.TTL)*time.Second).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	if err := o.redis.Publish(ctx, "agentauth:revocations", eventJSON).Err(); err != nil {
		o.logger.Errorf("Redis pub/sub publish failed (non-fatal): %v", err)
	}

	successCount := o.broadcastToSubscribers(event)
	o.logger.Infof("Broadcasted to %d/%d local subscribers", successCount, len(o.subscribers))

	totalDuration := time.Since(start)
	o.logger.Infof("✅ Emergency revocation completed in %v (PoA: %s)", totalDuration, event.PoAID)

	return nil
}

func (o *EmergencyRevocationOracle) broadcastToSubscribers(event *RevocationEvent) int {
	o.mu.RLock()
	defer o.mu.RUnlock()

	successCount := 0
	for subscriberID, ch := range o.subscribers {
		select {
		case ch <- event:
			successCount++
		case <-time.After(100 * time.Millisecond):
			o.logger.Warnf("Slow subscriber: %s", subscriberID)
		}
	}
	return successCount
}

// IsRevoked checks if a PoA is currently revoked
func (o *EmergencyRevocationOracle) IsRevoked(ctx context.Context, poaID string) (bool, *RevocationEvent, error) {
	key := fmt.Sprintf("revoked:%s", poaID)
	data, err := o.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("redis get failed: %w", err)
	}

	var event RevocationEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return false, nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return true, &event, nil
}

// Subscribe allows validators to receive real-time revocation events
func (o *EmergencyRevocationOracle) Subscribe(subscriberID string) <-chan *RevocationEvent {
	o.mu.Lock()
	defer o.mu.Unlock()

	ch := make(chan *RevocationEvent, 100)
	o.subscribers[subscriberID] = ch
	o.logger.Infof("New subscriber: %s (total: %d)", subscriberID, len(o.subscribers))

	return ch
}

// Unsubscribe removes a validator from the broadcast list
func (o *EmergencyRevocationOracle) Unsubscribe(subscriberID string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if ch, exists := o.subscribers[subscriberID]; exists {
		close(ch)
		delete(o.subscribers, subscriberID)
		o.logger.Infof("Unsubscribed: %s (remaining: %d)", subscriberID, len(o.subscribers))
	}
}

// StartRedisPubSub listens to Redis Pub/Sub for cluster-wide revocation broadcasts
func (o *EmergencyRevocationOracle) StartRedisPubSub(ctx context.Context) error {
	pubsub := o.redis.Subscribe(ctx, "agentauth:revocations")
	defer func() { _ = pubsub.Close() }()

	if _, err := pubsub.Receive(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to Redis Pub/Sub: %w", err)
	}

	o.logger.Info("Started Redis Pub/Sub listener for revocations")
	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			o.logger.Info("Stopping Redis Pub/Sub listener")
			return nil
		case msg, ok := <-ch:
			if !ok {
				return fmt.Errorf("pub/sub channel closed")
			}

			var event RevocationEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				o.logger.Errorf("Failed to unmarshal revocation event: %v", err)
				continue
			}

			o.logger.Infof("Received revocation via Pub/Sub: PoA=%s", event.PoAID)
			o.broadcastToSubscribers(&event)
		}
	}
}

// Close gracefully shuts down the oracle
func (o *EmergencyRevocationOracle) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	for subscriberID, ch := range o.subscribers {
		close(ch)
		o.logger.Infof("Closed subscriber: %s", subscriberID)
	}
	o.subscribers = make(map[string]chan *RevocationEvent)

	if err := o.redis.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}

	o.logger.Info("Emergency Revocation Oracle shut down successfully")
	return nil
}

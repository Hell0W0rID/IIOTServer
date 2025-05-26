package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// MessageClient defines the messaging interface
type MessageClient interface {
	Connect() error
	Disconnect() error
	Publish(topic string, data interface{}) error
	Subscribe(topic string, handler MessageHandler) error
	Unsubscribe(topic string) error
}

// MessageHandler defines message handling function
type MessageHandler func(topic string, data []byte) error

// RedisMessageClient implements MessageClient using Redis Streams
type RedisMessageClient struct {
	client      *redis.Client
	subscribers map[string]MessageHandler
	logger      *logrus.Logger
	mutex       sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewRedisMessageClient creates a new Redis message client
func NewRedisMessageClient(addr, password string, db int, logger *logrus.Logger) *RedisMessageClient {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &RedisMessageClient{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		subscribers: make(map[string]MessageHandler),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Connect establishes connection to Redis
func (r *RedisMessageClient) Connect() error {
	err := r.client.Ping(r.ctx).Err()
	if err != nil {
		r.logger.Errorf("Failed to connect to Redis: %v", err)
		return err
	}
	
	r.logger.Info("Connected to Redis message bus")
	return nil
}

// Disconnect closes the Redis connection
func (r *RedisMessageClient) Disconnect() error {
	r.cancel()
	return r.client.Close()
}

// Publish sends a message to a topic
func (r *RedisMessageClient) Publish(topic string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = r.client.XAdd(r.ctx, &redis.XAddArgs{
		Stream: topic,
		Values: map[string]interface{}{
			"data":      string(jsonData),
			"timestamp": fmt.Sprint(r.client.Time(r.ctx).Val().UnixMilli()),
		},
	}).Err()

	if err != nil {
		r.logger.Errorf("Failed to publish message to topic %s: %v", topic, err)
		return err
	}

	r.logger.Debugf("Published message to topic: %s", topic)
	return nil
}

// Subscribe subscribes to a topic with a message handler
func (r *RedisMessageClient) Subscribe(topic string, handler MessageHandler) error {
	r.mutex.Lock()
	r.subscribers[topic] = handler
	r.mutex.Unlock()

	go r.listenToStream(topic)
	
	r.logger.Infof("Subscribed to topic: %s", topic)
	return nil
}

// Unsubscribe removes subscription from a topic
func (r *RedisMessageClient) Unsubscribe(topic string) error {
	r.mutex.Lock()
	delete(r.subscribers, topic)
	r.mutex.Unlock()

	r.logger.Infof("Unsubscribed from topic: %s", topic)
	return nil
}

// listenToStream listens for messages on a Redis stream
func (r *RedisMessageClient) listenToStream(topic string) {
	consumerGroup := "edgex-consumer-group"
	consumerName := "edgex-consumer"

	// Create consumer group if it doesn't exist
	r.client.XGroupCreateMkStream(r.ctx, topic, consumerGroup, "0")

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			streams, err := r.client.XReadGroup(r.ctx, &redis.XReadGroupArgs{
				Group:    consumerGroup,
				Consumer: consumerName,
				Streams:  []string{topic, ">"},
				Count:    1,
				Block:    0,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					continue
				}
				r.logger.Errorf("Error reading from stream %s: %v", topic, err)
				continue
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					r.handleMessage(topic, message)
				}
			}
		}
	}
}

// handleMessage processes incoming messages
func (r *RedisMessageClient) handleMessage(topic string, message redis.XMessage) {
	r.mutex.RLock()
	handler, exists := r.subscribers[topic]
	r.mutex.RUnlock()

	if !exists {
		return
	}

	if data, ok := message.Values["data"].(string); ok {
		err := handler(topic, []byte(data))
		if err != nil {
			r.logger.Errorf("Error handling message from topic %s: %v", topic, err)
		}
	}
}

// MessageTopics defines common message topics
var MessageTopics = struct {
	Events      string
	Commands    string
	Metadata    string
	Metrics     string
}{
	Events:   "edgex.events",
	Commands: "edgex.commands",
	Metadata: "edgex.metadata",
	Metrics:  "edgex.metrics",
}
package stream

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RishiKendai/aegis/internal/preprocess"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Consumer struct {
	client              *redis.Client
	streamKey           string
	consumerGroup       string
	consumerName        string
	preprocessSvc       *preprocess.Service
	retryHandler        *RetryHandler
	retentionDuration   time.Duration
	pelRecoveryInterval time.Duration
	cleanupInterval     time.Duration
	lastPELCheck        time.Time
	lastCleanup         time.Time
}

func NewConsumer(
	client *redis.Client,
	streamKey string,
	consumerGroup string,
	consumerName string,
	preprocessSvc *preprocess.Service,
	retryHandler *RetryHandler,
	retentionDuration time.Duration,
) *Consumer {
	return &Consumer{
		client:              client,
		streamKey:           streamKey,
		consumerGroup:       consumerGroup,
		consumerName:        consumerName,
		preprocessSvc:       preprocessSvc,
		retryHandler:        retryHandler,
		retentionDuration:   retentionDuration,
		pelRecoveryInterval: 30 * time.Second,
		cleanupInterval:     1 * time.Hour,
		lastPELCheck:        time.Now(),
		lastCleanup:         time.Now(),
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	if err := c.createConsumerGroup(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to create consumer group, may be already exists")
	}

	// Recover PEL messages on startup (handle crash recovery)
	log.Info().Msg("Recovering PEL messages on startup")
	if err := c.recoverPEL(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to recover PEL messages on startup")
	}
	c.lastPELCheck = time.Now()

	// Start cleanup goroutine (background periodic cleanup)
	go c.runCleanupPeriodically(ctx)
	log.Info().
		Dur("cleanup_interval", c.cleanupInterval).
		Dur("retention", c.retentionDuration).
		Msg("Started cleanup goroutine")

	// Start consuming
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.consume(ctx); err != nil {
				log.Error().Err(err).Msg("Error consuming messages")
				time.Sleep(1 * time.Second) // Brief pause before retrying
			}
		}
	}
}

func (c *Consumer) createConsumerGroup(ctx context.Context) error {
	// MKSTREAM will create the stream if it doesn't exist
	err := c.client.XGroupCreateMkStream(ctx, c.streamKey, c.consumerGroup, "$").Err()
	if err != nil {
		if strings.Contains(err.Error(), "BUSYGROUP") {
			log.Debug().
				Str("group", c.consumerGroup).
				Msg("Consumer group already exists")
			return nil
		}
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	log.Info().
		Str("group", c.consumerGroup).
		Str("stream", c.streamKey).
		Msg("Created new consumer group (will only read new messages)")
	return nil
}

// recovers pending messages from the Pending Entry List
func (c *Consumer) recoverPEL(ctx context.Context) error {
	// Read pending messages for this consumer group
	pending, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: c.streamKey,
		Group:  c.consumerGroup,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil // No pending messages
		}
		return fmt.Errorf("failed to get pending messages: %w", err)
	}

	if len(pending) == 0 {
		return nil
	}

	log.Debug().Int("count", len(pending)).Msg("Found pending messages in PEL")

	// Claim pending messages that are idle for more than 1 minute
	minIdleTime := 1 * time.Minute
	messageIDs := make([]string, 0, len(pending))
	for _, p := range pending {
		if p.Idle >= minIdleTime {
			messageIDs = append(messageIDs, p.ID)
		}
	}

	if len(messageIDs) == 0 {
		return nil
	}

	log.Info().
		Int("claimable", len(messageIDs)).
		Msg("Attempting to claim idle pending messages")

	// Claim the messages
	claimed, err := c.client.XClaim(ctx, &redis.XClaimArgs{
		Stream:   c.streamKey,
		Group:    c.consumerGroup,
		Consumer: c.consumerName,
		MinIdle:  minIdleTime,
		Messages: messageIDs,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to claim messages: %w", err)
	}

	if len(claimed) == 0 {
		return nil
	}

	log.Info().
		Int("claimed", len(claimed)).
		Msg("Successfully claimed PEL messages, processing")

	// Process claimed messages
	for _, msg := range claimed {
		if err := c.processMessage(ctx, &msg); err != nil {
			log.Error().
				Err(err).
				Str("message_id", msg.ID).
				Msg("Failed to process claimed PEL message")
		}
	}

	return nil
}

func (c *Consumer) consume(ctx context.Context) error {
	// Periodically check for PEL messages (every 30 seconds)
	if time.Since(c.lastPELCheck) > c.pelRecoveryInterval {
		if err := c.recoverPEL(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to recover PEL messages")
		}
		c.lastPELCheck = time.Now()
	}

	streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.consumerGroup,
		Consumer: c.consumerName,
		Streams:  []string{c.streamKey, ">"},
		Count:    10,          // Read up to 10 messages at a time
		Block:    time.Second, // Block for 1 second if no messages
	}).Result()

	if err == redis.Nil {
		return nil // No messages available
	}
	if err != nil {
		return fmt.Errorf("failed to read from stream: %w", err)
	}

	// Process each stream
	for _, stream := range streams {
		if stream.Stream != c.streamKey {
			continue
		}

		for _, msg := range stream.Messages {
			if err := c.processMessage(ctx, &msg); err != nil {
				log.Error().
					Err(err).
					Str("message_id", msg.ID).
					Msg("Failed to process message")
				// Error handling is done in processMessage via retry handler
			}
		}
	}

	return nil
}

// processMessage processes a single message
func (c *Consumer) processMessage(ctx context.Context, msg *redis.XMessage) error {
	// Parse message fields
	fields := make(map[string]string)
	for key, val := range msg.Values {
		if value, ok := val.(string); ok {
			fields[key] = value
		}
	}

	streamMsg := &StreamMessage{
		ID:     msg.ID,
		Fields: fields,
	}

	// Parse submission
	submission, err := ParseSubmission(streamMsg)
	if err != nil {
		log.Error().Err(err).Str("message_id", msg.ID).Msg("Failed to parse submission")
		// Acknowledge bad messages to avoid reprocessing
		c.acknowledge(ctx, msg.ID)
		return err
	}

	// Convert fields to map[string]interface{} for death queue
	fieldsMap := make(map[string]interface{})
	for k, v := range fields {
		fieldsMap[k] = v
	}

	// Process with retry logic
	err = c.retryHandler.RetryWithBackoff(ctx, func() error {
		return c.preprocessSvc.ProcessSubmission(ctx, submission)
	}, msg.ID, fieldsMap)

	if err != nil {
		// Already sent to death queue by retry handler
		return err
	}

	// Acknowledge successful processing
	return c.acknowledge(ctx, msg.ID)
}

// removes messages older than retention duration
func (c *Consumer) cleanupOldMessages(ctx context.Context) error {
	// Calculate the minimum ID to keep (messages older than this will be deleted)
	cutoffTime := time.Now().Add(-c.retentionDuration)
	minID := fmt.Sprintf("%d-0", cutoffTime.UnixMilli())

	// Use XTrimMinID to remove old messages
	trimmed, err := c.client.XTrimMinID(ctx, c.streamKey, minID).Result()
	if err != nil {
		return fmt.Errorf("failed to trim stream: %w", err)
	}

	if trimmed > 0 {
		log.Debug().
			Int64("trimmed", trimmed).
			Dur("retention", c.retentionDuration).
			Str("cutoff_time", cutoffTime.Format(time.RFC3339)).
			Msg("Cleaned up old messages from stream")
	}

	return nil
}

// runs cleanup every hour
func (c *Consumer) runCleanupPeriodically(ctx context.Context) {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	// Run initial cleanup after startup
	if err := c.cleanupOldMessages(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to run initial cleanup")
	}
	c.lastCleanup = time.Now()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Cleanup goroutine shutting down")
			return
		case <-ticker.C:
			if err := c.cleanupOldMessages(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to cleanup old messages")
			}
			c.lastCleanup = time.Now()
		}
	}
}

func (c *Consumer) acknowledge(ctx context.Context, messageID string) error {
	err := c.client.XAck(ctx, c.streamKey, c.consumerGroup, messageID).Err()
	if err != nil {
		log.Error().Err(err).Str("message_id", messageID).Msg("Failed to acknowledge message")
		return err
	}

	log.Debug().
		Str("message_id", messageID).
		Msg("Message acknowledged")

	return nil
}

package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"hound-todo/shared/logging"
)

const (
	queueName = "text.commands"
)

// TextMessage matches the format published by ingress-svc
type TextMessage struct {
	UserID         string `json:"user_id"`
	CommandText    string `json:"command_text"`
	MessageSid     string `json:"message_sid"`
	IdempotencyKey string `json:"idempotency_key"`
}

// Handler is called for each message received
type Handler func(ctx context.Context, msg *TextMessage) error

// Consumer consumes messages from RabbitMQ
type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  *logging.Logger
}

// New creates a new RabbitMQ consumer
func New(rabbitMQURL string, logger *logging.Logger) (*Consumer, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare the queue (idempotent - won't fail if it exists)
	_, err = ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Set prefetch count to process one message at a time
	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &Consumer{
		conn:    conn,
		channel: ch,
		logger:  logger,
	}, nil
}

// Start begins consuming messages and calls the handler for each one
func (c *Consumer) Start(ctx context.Context, handler Handler) error {
	msgs, err := c.channel.Consume(
		queueName,
		"",    // consumer tag (auto-generated)
		false, // auto-ack (we'll manually ack after processing)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	c.logger.Info("Waiting for messages on queue: %s", queueName)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Consumer shutting down")
			return ctx.Err()

		case delivery, ok := <-msgs:
			if !ok {
				return fmt.Errorf("channel closed")
			}

			// Parse the message
			var msg TextMessage
			if err := json.Unmarshal(delivery.Body, &msg); err != nil {
				c.logger.Error("Failed to parse message: %v", err)
				// Reject without requeue - bad message format
				delivery.Nack(false, false)
				continue
			}

			c.logger.Info("Received message from %s: %s", msg.UserID, msg.CommandText)

			// Process the message
			if err := handler(ctx, &msg); err != nil {
				c.logger.Error("Failed to process message: %v", err)
				// Requeue for retry
				delivery.Nack(false, true)
				continue
			}

			// Acknowledge successful processing
			delivery.Ack(false)
		}
	}
}

// Close cleanly shuts down the consumer
func (c *Consumer) Close() error {
	if err := c.channel.Close(); err != nil {
		return err
	}
	return c.conn.Close()
}

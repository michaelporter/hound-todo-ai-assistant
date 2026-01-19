package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// Queue names
	AudioProcessingQueue = "audio.processing"
	TextCommandsQueue    = "text.commands"
)

// AudioMessage represents a voice memo message for transcription
type AudioMessage struct {
	UserID         string `json:"user_id"`
	MediaURL       string `json:"media_url"`
	MessageSid     string `json:"message_sid"`
	IdempotencyKey string `json:"idempotency_key"`
}

// TextMessage represents a text command message
type TextMessage struct {
	UserID         string `json:"user_id"`
	CommandText    string `json:"command_text"`
	MessageSid     string `json:"message_sid"`
	IdempotencyKey string `json:"idempotency_key"`
}

// Publisher handles RabbitMQ message publishing
type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// New creates a new RabbitMQ publisher
func New(url string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	p := &Publisher{
		conn:    conn,
		channel: ch,
	}

	// Declare queues
	if err := p.declareQueues(); err != nil {
		p.Close()
		return nil, err
	}

	return p, nil
}

func (p *Publisher) declareQueues() error {
	queues := []string{AudioProcessingQueue, TextCommandsQueue}
	for _, q := range queues {
		_, err := p.channel.QueueDeclare(
			q,     // name
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", q, err)
		}
	}
	return nil
}

// PublishAudio publishes an audio processing message
func (p *Publisher) PublishAudio(ctx context.Context, msg *AudioMessage) error {
	return p.publish(ctx, AudioProcessingQueue, msg)
}

// PublishText publishes a text command message
func (p *Publisher) PublishText(ctx context.Context, msg *TextMessage) error {
	return p.publish(ctx, TextCommandsQueue, msg)
}

func (p *Publisher) publish(ctx context.Context, queue string, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(
		ctx,
		"",    // exchange (default)
		queue, // routing key (queue name)
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message to %s: %w", queue, err)
	}

	return nil
}

// Close gracefully shuts down the publisher
func (p *Publisher) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

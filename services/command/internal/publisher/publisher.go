package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	SMSRepliesQueue = "sms.replies"
)

// SMSReply represents a message to be sent back to the user
type SMSReply struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
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

	// Declare the replies queue
	_, err = ch.QueueDeclare(
		SMSRepliesQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return p, nil
}

// PublishReply publishes an SMS reply message
func (p *Publisher) PublishReply(ctx context.Context, userID, message string) error {
	reply := &SMSReply{
		UserID:  userID,
		Message: message,
	}

	body, err := json.Marshal(reply)
	if err != nil {
		return fmt.Errorf("failed to marshal reply: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(
		ctx,
		"",              // exchange (default)
		SMSRepliesQueue, // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish reply: %w", err)
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

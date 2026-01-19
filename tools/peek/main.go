package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	defaultRabbitMQAPI = "http://localhost:15672/api"
	defaultUser        = "hound"
	defaultPass        = "hound_dev"
)

type Message struct {
	Payload         string `json:"payload"`
	PayloadEncoding string `json:"payload_encoding"`
	MessageCount    int    `json:"message_count"`
	Redelivered     bool   `json:"redelivered"`
	Properties      struct {
		ContentType  string `json:"content_type"`
		DeliveryMode int    `json:"delivery_mode"`
	} `json:"properties"`
}

type QueueInfo struct {
	Name     string `json:"name"`
	Messages int    `json:"messages"`
	State    string `json:"state"`
}

func main() {
	listCmd := flag.Bool("list", false, "List all queues")
	queueName := flag.String("queue", "", "Queue to peek (e.g., text.commands, audio.processing)")
	count := flag.Int("count", 10, "Number of messages to peek")
	flag.Parse()

	apiURL := getEnvOrDefault("RABBITMQ_API", defaultRabbitMQAPI)
	user := getEnvOrDefault("RABBITMQ_USER", defaultUser)
	pass := getEnvOrDefault("RABBITMQ_PASS", defaultPass)

	client := &rabbitClient{apiURL: apiURL, user: user, pass: pass}

	if *listCmd {
		if err := client.listQueues(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *queueName == "" {
		fmt.Println("Usage:")
		fmt.Println("  peek -list                    List all queues")
		fmt.Println("  peek -queue text.commands     Peek at messages in queue")
		fmt.Println("  peek -queue text.commands -count 5")
		fmt.Println("")
		fmt.Println("Queues:")
		client.listQueues()
		return
	}

	if err := client.peekMessages(*queueName, *count); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

type rabbitClient struct {
	apiURL string
	user   string
	pass   string
}

func (c *rabbitClient) listQueues() error {
	req, err := http.NewRequest("GET", c.apiURL+"/queues", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.user, c.pass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var queues []QueueInfo
	if err := json.NewDecoder(resp.Body).Decode(&queues); err != nil {
		return err
	}

	fmt.Printf("%-25s %s\n", "QUEUE", "MESSAGES")
	fmt.Println(strings.Repeat("-", 35))
	for _, q := range queues {
		fmt.Printf("%-25s %d\n", q.Name, q.Messages)
	}
	return nil
}

func (c *rabbitClient) peekMessages(queue string, count int) error {
	// Use ack_requeue_true to peek without consuming
	body := fmt.Sprintf(`{"count":%d,"ackmode":"ack_requeue_true","encoding":"auto"}`, count)

	url := fmt.Sprintf("%s/queues/%%2F/%s/get", c.apiURL, queue)
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.user, c.pass)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("queue '%s' not found", queue)
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var messages []Message
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return err
	}

	if len(messages) == 0 {
		fmt.Printf("Queue '%s' is empty\n", queue)
		return nil
	}

	fmt.Printf("Queue: %s (%d messages shown)\n", queue, len(messages))
	fmt.Println(strings.Repeat("=", 60))

	for i, msg := range messages {
		fmt.Printf("\n[Message %d]\n", i+1)

		// Try to pretty-print JSON payload
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Payload), &payload); err == nil {
			prettyJSON, _ := json.MarshalIndent(payload, "", "  ")
			fmt.Println(string(prettyJSON))
		} else {
			fmt.Println(msg.Payload)
		}

		if i < len(messages)-1 {
			fmt.Println(strings.Repeat("-", 40))
		}
	}
	fmt.Println()
	return nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

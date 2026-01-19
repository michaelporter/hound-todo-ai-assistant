package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"hound-todo/services/ingress/internal/publisher"
	"hound-todo/shared/idempotency"
	"hound-todo/shared/logging"

	"github.com/twilio/twilio-go/client"
)

// Publisher defines the interface for message publishing
type Publisher interface {
	PublishAudio(ctx context.Context, msg *publisher.AudioMessage) error
	PublishText(ctx context.Context, msg *publisher.TextMessage) error
}

// WebhookHandler handles Twilio SMS/MMS webhooks
type WebhookHandler struct {
	pub             Publisher
	logger          *logging.Logger
	twilioAuthToken string
	webhookURL      string
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(pub Publisher, logger *logging.Logger, twilioAuthToken string) *WebhookHandler {
	return &WebhookHandler{
		pub:             pub,
		logger:          logger,
		twilioAuthToken: twilioAuthToken,
	}
}

// SetWebhookURL sets the URL used for signature validation
func (h *WebhookHandler) SetWebhookURL(url string) {
	h.webhookURL = url
}

// HandleSMS handles incoming SMS/MMS webhooks from Twilio
func (h *WebhookHandler) HandleSMS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("Failed to parse form: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Extract Twilio webhook data
	from := r.FormValue("From")
	body := r.FormValue("Body")
	messageSid := r.FormValue("MessageSid")
	numMedia := r.FormValue("NumMedia")
	mediaURL := r.FormValue("MediaUrl0")

	// Validate required fields
	if from == "" || messageSid == "" {
		h.logger.Error("Missing required fields: From=%s, MessageSid=%s", from, messageSid)
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Validate Twilio signature if auth token is configured
	if h.twilioAuthToken != "" {
		if !h.validateSignature(r) {
			h.logger.Error("Invalid Twilio signature for message %s", messageSid)
			http.Error(w, "Invalid signature", http.StatusForbidden)
			return
		}
	}

	// Generate idempotency key from message SID
	idemKey := idempotency.GenerateKey(messageSid)

	ctx := r.Context()

	// Determine message type and publish to appropriate queue
	if numMedia != "" && numMedia != "0" && mediaURL != "" {
		// Voice memo / MMS with media
		if err := h.handleAudioMessage(ctx, from, mediaURL, messageSid, idemKey); err != nil {
			h.logger.Error("Failed to publish audio message: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		h.logger.Info("Published audio message from %s (sid: %s)", from, messageSid)
	} else if body != "" {
		// Text command
		if err := h.handleTextMessage(ctx, from, body, messageSid, idemKey); err != nil {
			h.logger.Error("Failed to publish text message: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		h.logger.Info("Published text message from %s (sid: %s)", from, messageSid)
	} else {
		h.logger.Error("Empty message from %s (sid: %s)", from, messageSid)
		http.Error(w, "Empty message", http.StatusBadRequest)
		return
	}

	// Return empty TwiML response
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?><Response></Response>")
}

func (h *WebhookHandler) handleAudioMessage(ctx context.Context, from, mediaURL, messageSid, idemKey string) error {
	msg := &publisher.AudioMessage{
		UserID:         from,
		MediaURL:       mediaURL,
		MessageSid:     messageSid,
		IdempotencyKey: idemKey,
	}
	return h.pub.PublishAudio(ctx, msg)
}

func (h *WebhookHandler) handleTextMessage(ctx context.Context, from, body, messageSid, idemKey string) error {
	msg := &publisher.TextMessage{
		UserID:         from,
		CommandText:    strings.TrimSpace(body),
		MessageSid:     messageSid,
		IdempotencyKey: idemKey,
	}
	return h.pub.PublishText(ctx, msg)
}

func (h *WebhookHandler) validateSignature(r *http.Request) bool {
	signature := r.Header.Get("X-Twilio-Signature")
	if signature == "" {
		return false
	}

	// Build the URL for validation
	url := h.webhookURL
	if url == "" {
		// Fallback: construct URL from request
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		url = fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)
	}

	// Use Twilio's request validator
	validator := client.NewRequestValidator(h.twilioAuthToken)

	// Convert form values to map
	params := make(map[string]string)
	for key, values := range r.PostForm {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	return validator.Validate(url, params, signature)
}

// HandleHealth handles health check requests
func (h *WebhookHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"healthy"}`)
}

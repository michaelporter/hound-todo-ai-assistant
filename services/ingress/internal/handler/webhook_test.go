package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hound-todo/services/ingress/internal/publisher"
	"hound-todo/shared/logging"
)

// mockPublisher records published messages for test assertions
type mockPublisher struct {
	audioMessages []*publisher.AudioMessage
	textMessages  []*publisher.TextMessage
	audioErr      error
	textErr       error
}

func (m *mockPublisher) PublishAudio(ctx context.Context, msg *publisher.AudioMessage) error {
	if m.audioErr != nil {
		return m.audioErr
	}
	m.audioMessages = append(m.audioMessages, msg)
	return nil
}

func (m *mockPublisher) PublishText(ctx context.Context, msg *publisher.TextMessage) error {
	if m.textErr != nil {
		return m.textErr
	}
	m.textMessages = append(m.textMessages, msg)
	return nil
}

func newTestHandler(pub *mockPublisher, authToken string) *WebhookHandler {
	logger := logging.New("test")
	return NewWebhookHandler(pub, logger, authToken)
}

func makeFormRequest(values url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/sms", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// =============================================================================
// Text Message Tests
// =============================================================================

func TestHandleSMS_TextMessage(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("Body", "buy groceries tomorrow")
	form.Set("MessageSid", "SM123456789")

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "<Response></Response>") {
		t.Errorf("expected TwiML response, got %s", rec.Body.String())
	}

	if len(mock.textMessages) != 1 {
		t.Fatalf("expected 1 text message, got %d", len(mock.textMessages))
	}

	msg := mock.textMessages[0]
	if msg.UserID != "+15551234567" {
		t.Errorf("expected UserID +15551234567, got %s", msg.UserID)
	}
	if msg.CommandText != "buy groceries tomorrow" {
		t.Errorf("expected CommandText 'buy groceries tomorrow', got %s", msg.CommandText)
	}
	if msg.MessageSid != "SM123456789" {
		t.Errorf("expected MessageSid SM123456789, got %s", msg.MessageSid)
	}
	if msg.IdempotencyKey == "" {
		t.Error("expected IdempotencyKey to be set")
	}
}

func TestHandleSMS_TextMessage_TrimsWhitespace(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("Body", "  buy milk  ")
	form.Set("MessageSid", "SM123")

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if mock.textMessages[0].CommandText != "buy milk" {
		t.Errorf("expected trimmed text, got '%s'", mock.textMessages[0].CommandText)
	}
}

// =============================================================================
// Media/Audio Message Tests
// =============================================================================

func TestHandleSMS_MediaMessage(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("From", "+15559876543")
	form.Set("Body", "")
	form.Set("MessageSid", "SMmedia123")
	form.Set("NumMedia", "1")
	form.Set("MediaUrl0", "https://api.twilio.com/media/ME123")

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(mock.audioMessages) != 1 {
		t.Fatalf("expected 1 audio message, got %d", len(mock.audioMessages))
	}
	if len(mock.textMessages) != 0 {
		t.Errorf("expected 0 text messages, got %d", len(mock.textMessages))
	}

	msg := mock.audioMessages[0]
	if msg.UserID != "+15559876543" {
		t.Errorf("expected UserID +15559876543, got %s", msg.UserID)
	}
	if msg.MediaURL != "https://api.twilio.com/media/ME123" {
		t.Errorf("expected MediaURL, got %s", msg.MediaURL)
	}
	if msg.MessageSid != "SMmedia123" {
		t.Errorf("expected MessageSid SMmedia123, got %s", msg.MessageSid)
	}
}

func TestHandleSMS_MediaMessage_WithBodyPrefersMedia(t *testing.T) {
	// When both media and body are present, media takes precedence
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("Body", "some text")
	form.Set("MessageSid", "SM123")
	form.Set("NumMedia", "1")
	form.Set("MediaUrl0", "https://api.twilio.com/media/ME123")

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(mock.audioMessages) != 1 {
		t.Errorf("expected 1 audio message, got %d", len(mock.audioMessages))
	}
	if len(mock.textMessages) != 0 {
		t.Errorf("expected 0 text messages (media takes precedence), got %d", len(mock.textMessages))
	}
}

func TestHandleSMS_MediaMessage_NumMediaZero(t *testing.T) {
	// NumMedia=0 should be treated as text message
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("Body", "just text")
	form.Set("MessageSid", "SM123")
	form.Set("NumMedia", "0")

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(mock.textMessages) != 1 {
		t.Errorf("expected 1 text message, got %d", len(mock.textMessages))
	}
	if len(mock.audioMessages) != 0 {
		t.Errorf("expected 0 audio messages, got %d", len(mock.audioMessages))
	}
}

// =============================================================================
// Validation Error Tests
// =============================================================================

func TestHandleSMS_MissingFrom(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("Body", "test message")
	form.Set("MessageSid", "SM123")

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	if len(mock.textMessages) != 0 || len(mock.audioMessages) != 0 {
		t.Error("expected no messages to be published")
	}
}

func TestHandleSMS_MissingMessageSid(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("Body", "test message")

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSMS_EmptyMessage(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("MessageSid", "SM123")
	form.Set("Body", "")
	// No media either

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSMS_MethodNotAllowed(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "")

	req := httptest.NewRequest(http.MethodGet, "/webhooks/sms", nil)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

// =============================================================================
// Publisher Error Tests
// =============================================================================

func TestHandleSMS_TextPublishError(t *testing.T) {
	mock := &mockPublisher{
		textErr: errors.New("connection refused"),
	}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("Body", "test message")
	form.Set("MessageSid", "SM123")

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleSMS_AudioPublishError(t *testing.T) {
	mock := &mockPublisher{
		audioErr: errors.New("connection refused"),
	}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("MessageSid", "SM123")
	form.Set("NumMedia", "1")
	form.Set("MediaUrl0", "https://api.twilio.com/media/ME123")

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// Signature Validation Tests
// =============================================================================

func TestHandleSMS_MissingSignature_WhenAuthTokenSet(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "my-auth-token")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("Body", "test message")
	form.Set("MessageSid", "SM123")

	req := makeFormRequest(form)
	// No X-Twilio-Signature header
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	if len(mock.textMessages) != 0 {
		t.Error("expected no messages to be published when signature is missing")
	}
}

func TestHandleSMS_InvalidSignature(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "my-auth-token")
	handler.SetWebhookURL("https://example.com/webhooks/sms")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("Body", "test message")
	form.Set("MessageSid", "SM123")

	req := makeFormRequest(form)
	req.Header.Set("X-Twilio-Signature", "invalid-signature")
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestHandleSMS_NoAuthToken_SkipsValidation(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "") // empty auth token

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("Body", "test message")
	form.Set("MessageSid", "SM123")

	req := makeFormRequest(form)
	// No signature header, but that's OK because auth token is empty
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d (validation skipped), got %d", http.StatusOK, rec.Code)
	}
}

// =============================================================================
// Health Endpoint Tests
// =============================================================================

func TestHandleHealth(t *testing.T) {
	handler := newTestHandler(&mockPublisher{}, "")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HandleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	expected := `{"status":"healthy"}`
	if rec.Body.String() != expected {
		t.Errorf("expected body %s, got %s", expected, rec.Body.String())
	}
}

// =============================================================================
// Response Format Tests
// =============================================================================

func TestHandleSMS_ResponseContentType(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("Body", "test")
	form.Set("MessageSid", "SM123")

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/xml" {
		t.Errorf("expected Content-Type text/xml, got %s", contentType)
	}
}

func TestHandleSMS_TwiMLResponse(t *testing.T) {
	mock := &mockPublisher{}
	handler := newTestHandler(mock, "")

	form := url.Values{}
	form.Set("From", "+15551234567")
	form.Set("Body", "test")
	form.Set("MessageSid", "SM123")

	req := makeFormRequest(form)
	rec := httptest.NewRecorder()

	handler.HandleSMS(rec, req)

	expected := `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`
	if rec.Body.String() != expected {
		t.Errorf("expected TwiML response %s, got %s", expected, rec.Body.String())
	}
}

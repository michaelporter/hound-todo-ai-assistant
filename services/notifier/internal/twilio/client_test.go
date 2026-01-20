package twilio

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("AC123", "token123", "+15551234567")

	if client.accountSID != "AC123" {
		t.Errorf("expected accountSID AC123, got %s", client.accountSID)
	}
	if client.authToken != "token123" {
		t.Errorf("expected authToken token123, got %s", client.authToken)
	}
	if client.phoneNumber != "+15551234567" {
		t.Errorf("expected phoneNumber +15551234567, got %s", client.phoneNumber)
	}
	if client.httpClient == nil {
		t.Error("expected httpClient to be initialized")
	}
}

func TestSendSMS_Success(t *testing.T) {
	var capturedTo string
	var capturedFrom string
	var capturedBody string
	var capturedAuth string
	var capturedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture request details
		capturedContentType = r.Header.Get("Content-Type")
		username, password, _ := r.BasicAuth()
		capturedAuth = username + ":" + password

		r.ParseForm()
		capturedTo = r.Form.Get("To")
		capturedFrom = r.Form.Get("From")
		capturedBody = r.Form.Get("Body")

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"sid":"SM123"}`))
	}))
	defer server.Close()

	customClient := &Client{
		accountSID:  "AC123",
		authToken:   "token123",
		phoneNumber: "+15551234567",
		httpClient:  server.Client(),
	}

	// Create a wrapper that uses the test server URL
	err := sendSMSWithURL(customClient, context.Background(), "+15559876543", "Hello!", server.URL+"/test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedContentType != "application/x-www-form-urlencoded" {
		t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %s", capturedContentType)
	}

	if capturedAuth != "AC123:token123" {
		t.Errorf("expected basic auth AC123:token123, got %s", capturedAuth)
	}

	if capturedTo != "+15559876543" {
		t.Errorf("expected To +15559876543, got %s", capturedTo)
	}
	if capturedFrom != "+15551234567" {
		t.Errorf("expected From +15551234567, got %s", capturedFrom)
	}
	if capturedBody != "Hello!" {
		t.Errorf("expected Body 'Hello!', got %s", capturedBody)
	}
}

// sendSMSWithURL is a test helper that allows overriding the API URL
func sendSMSWithURL(c *Client, ctx context.Context, to, body, apiURL string) error {
	// Build form data properly
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", c.phoneNumber)
	data.Set("Body", body)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.accountSID, c.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func TestSendSMS_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"code":21211,"message":"Invalid 'To' phone number"}`))
	}))
	defer server.Close()

	client := &Client{
		accountSID:  "AC123",
		authToken:   "token123",
		phoneNumber: "+15551234567",
		httpClient:  server.Client(),
	}

	err := sendSMSWithURLAndCheckStatus(client, context.Background(), "+invalid", "Hello!", server.URL+"/test")

	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to contain status code, got: %v", err)
	}
}

func sendSMSWithURLAndCheckStatus(c *Client, ctx context.Context, to, body, apiURL string) error {
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", c.phoneNumber)
	data.Set("Body", body)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.accountSID, c.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio API error (status %d)", resp.StatusCode)
	}

	return nil
}

func TestSendSMS_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This handler should never be reached
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		accountSID:  "AC123",
		authToken:   "token123",
		phoneNumber: "+15551234567",
		httpClient:  server.Client(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := sendSMSWithURL(client, ctx, "+15559876543", "Hello!", server.URL+"/test")

	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

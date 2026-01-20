package twilio

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	twilioAPIBase = "https://api.twilio.com/2010-04-01"
)

// Client handles sending SMS via Twilio API
type Client struct {
	accountSID  string
	authToken   string
	phoneNumber string
	httpClient  *http.Client
}

// NewClient creates a new Twilio client
func NewClient(accountSID, authToken, phoneNumber string) *Client {
	return &Client{
		accountSID:  accountSID,
		authToken:   authToken,
		phoneNumber: phoneNumber,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendSMS sends an SMS message to the specified phone number
func (c *Client) SendSMS(ctx context.Context, to, body string) error {
	// Build request URL
	apiURL := fmt.Sprintf("%s/Accounts/%s/Messages.json", twilioAPIBase, c.accountSID)

	// Build form data
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", c.phoneNumber)
	data.Set("Body", body)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.SetBasicAuth(c.accountSID, c.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error details
	respBody, _ := io.ReadAll(resp.Body)

	// Check status
	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateKey creates an idempotency key from a source string
// This is used to prevent duplicate processing of the same request
func GenerateKey(source string) string {
	hash := sha256.Sum256([]byte(source))
	return fmt.Sprintf("idem_%s", hex.EncodeToString(hash[:16]))
}

// Example: Twilio message SID -> idempotency key
// GenerateKey("SM1234567890abcdef") -> "idem_a1b2c3d4e5f6..."

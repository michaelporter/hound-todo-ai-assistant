package logging

import (
	"log"
	"os"
)

// Logger wraps standard logging with structured fields
// TODO: Replace with proper structured logging (zap, zerolog)
type Logger struct {
	serviceName string
}

// New creates a new logger for a service
func New(serviceName string) *Logger {
	return &Logger{
		serviceName: serviceName,
	}
}

// Info logs an informational message
func (l *Logger) Info(msg string, args ...interface{}) {
	log.Printf("[%s] INFO: "+msg, append([]interface{}{l.serviceName}, args...)...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	log.Printf("[%s] ERROR: "+msg, append([]interface{}{l.serviceName}, args...)...)
}

// Fatal logs a fatal error and exits
func (l *Logger) Fatal(err error) {
	log.Printf("[%s] FATAL: %v", l.serviceName, err)
	os.Exit(1)
}

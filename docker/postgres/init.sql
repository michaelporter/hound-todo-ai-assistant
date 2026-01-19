-- =============================================================================
-- Hound Todo App - PostgreSQL Initialization
-- =============================================================================
-- This script runs on first container startup to create the required databases

-- Create the three databases as per ARCHITECTURE.md
CREATE DATABASE todo_db;
CREATE DATABASE audit_db;
CREATE DATABASE transcription_db;

-- Grant privileges to the hound user
GRANT ALL PRIVILEGES ON DATABASE todo_db TO hound;
GRANT ALL PRIVILEGES ON DATABASE audit_db TO hound;
GRANT ALL PRIVILEGES ON DATABASE transcription_db TO hound;

-- =============================================================================
-- todo_db schema
-- =============================================================================
\c todo_db

CREATE TABLE IF NOT EXISTS todos (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_todos_user_id ON todos(user_id);
CREATE INDEX idx_todos_status ON todos(status);
CREATE INDEX idx_todos_user_status ON todos(user_id, status);

-- Idempotency keys table to prevent duplicate operations
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key VARCHAR(255) PRIMARY KEY,
    response JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- audit_db schema (append-only event log)
-- =============================================================================
\c audit_db

CREATE TABLE IF NOT EXISTS audit_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id BIGINT NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_entity ON audit_events(entity_type, entity_id);
CREATE INDEX idx_audit_user ON audit_events(user_id);
CREATE INDEX idx_audit_created ON audit_events(created_at);

-- =============================================================================
-- transcription_db schema
-- =============================================================================
\c transcription_db

CREATE TABLE IF NOT EXISTS transcriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    audio_url TEXT,
    raw_text TEXT NOT NULL,
    parsed_action TEXT,
    twilio_message_sid VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transcriptions_user ON transcriptions(user_id);
CREATE INDEX idx_transcriptions_twilio ON transcriptions(twilio_message_sid);

-- GAuth Database Initialization Script

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create database schema
CREATE SCHEMA IF NOT EXISTS gauth;

-- Set search path
SET search_path TO gauth, public;

-- Create tables for token storage
CREATE TABLE IF NOT EXISTS tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_value TEXT NOT NULL UNIQUE,
    client_id VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    scopes TEXT[],
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB
);

-- Create tables for authorization grants
CREATE TABLE IF NOT EXISTS grants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    grant_id VARCHAR(255) NOT NULL UNIQUE,
    client_id VARCHAR(255) NOT NULL,
    scopes TEXT[],
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    used_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB
);

-- Create tables for audit logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    client_id VARCHAR(255),
    user_id VARCHAR(255),
    resource VARCHAR(255),
    action VARCHAR(100),
    result VARCHAR(50),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_tokens_client_id ON tokens(client_id);
CREATE INDEX IF NOT EXISTS idx_tokens_expires_at ON tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_tokens_created_at ON tokens(created_at);
CREATE INDEX IF NOT EXISTS idx_grants_client_id ON grants(client_id);
CREATE INDEX IF NOT EXISTS idx_grants_expires_at ON grants(expires_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_client_id ON audit_logs(client_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- Create user for application
CREATE USER gauth_app WITH PASSWORD 'secure_app_password';
GRANT USAGE ON SCHEMA gauth TO gauth_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA gauth TO gauth_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA gauth TO gauth_app;

-- Set default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA gauth GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO gauth_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA gauth GRANT USAGE, SELECT ON SEQUENCES TO gauth_app;
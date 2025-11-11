-- Migration: Create subscriptions table
-- RFC-0111 Subscription Flow Storage (Steps I-VIII)
-- Version: 002
-- Created: 2025-11-11

CREATE TABLE IF NOT EXISTS subscriptions (
    -- Primary key
    id VARCHAR(255) PRIMARY KEY,
    
    -- Status tracking
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Step I: Owner's Authorizer Identity Proof
    owners_authorizer_identity JSONB,
    
    -- Step II: Owner's Authorizer Authorization Proof
    commercial_register_entry JSONB,
    authorization_proof JSONB,
    
    -- Step III: Client Owner Identity Proof
    client_owner_identity JSONB,
    
    -- Step IV: Client Owner Authorization Proof
    client_owner_auth_proof JSONB,
    
    -- Step V: Client Authorization
    client_authorization_grant JSONB,
    
    -- Step VI: Resource Owner Identity Proof
    resource_owner_identity JSONB,
    
    -- Step VII: Resource Owner Authorization Proof
    resource_owner_auth_proof JSONB,
    
    -- Step VIII: Resource Server Authorization
    resource_server_auth JSONB,
    
    -- Complete authorization chain
    authorization_chain JSONB,
    
    -- Indexes for common queries
    CONSTRAINT status_valid CHECK (status IN ('pending', 'awaiting_identity', 'awaiting_auth_proof', 
        'awaiting_client_owner', 'awaiting_client', 'awaiting_resource', 'completed', 'failed'))
);

-- Create indexes for performance
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_created_at ON subscriptions(created_at DESC);
CREATE INDEX idx_subscriptions_updated_at ON subscriptions(updated_at DESC);
CREATE INDEX idx_subscriptions_client_id ON subscriptions((client_authorization_grant->>'client_id'));
CREATE INDEX idx_subscriptions_resource_owner ON subscriptions((resource_owner_identity->>'subject_id'));
CREATE INDEX idx_subscriptions_completed ON subscriptions(id) WHERE status = 'completed';

-- Add comments
COMMENT ON TABLE subscriptions IS 'RFC-0111 Subscription Flow (one-off enrollment, Steps I-VIII)';
COMMENT ON COLUMN subscriptions.id IS 'Subscription ID (e.g., sub_1234567890)';
COMMENT ON COLUMN subscriptions.status IS 'Current step in subscription flow';
COMMENT ON COLUMN subscriptions.owners_authorizer_identity IS 'Step I: PVP-verified identity of owners authorizer';
COMMENT ON COLUMN subscriptions.authorization_proof IS 'Step II: Commercial register proof';
COMMENT ON COLUMN subscriptions.client_owner_identity IS 'Step III: PVP-verified identity of client owner';
COMMENT ON COLUMN subscriptions.client_authorization_grant IS 'Step V: Client authorization with PoA credential';
COMMENT ON COLUMN subscriptions.resource_owner_identity IS 'Step VI: PVP-verified identity of resource owner';
COMMENT ON COLUMN subscriptions.resource_server_auth IS 'Step VIII: Resource server authorization';
COMMENT ON COLUMN subscriptions.authorization_chain IS 'Complete 3-level authorization chain';

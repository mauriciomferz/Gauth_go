-- CIBA Authentication Requests Table
CREATE TABLE IF NOT EXISTS ciba_auth_requests (
    auth_req_id VARCHAR(255) PRIMARY KEY,
    status VARCHAR(50) NOT NULL, -- pending, completed, expired, denied
    client_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255), -- populated after user identification/approval
    scope TEXT,
    login_hint TEXT,
    binding_message TEXT,
    is_consent_required BOOLEAN DEFAULT TRUE,
    access_token VARCHAR(4096),
    id_token VARCHAR(4096),
    refresh_token VARCHAR(4096),
    expires_in INTEGER,
    interval INTEGER DEFAULT 5,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ciba_requests_status ON ciba_auth_requests(status);

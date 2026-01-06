CREATE TABLE IF NOT EXISTS policies (
    id TEXT PRIMARY KEY,
    version INTEGER NOT NULL DEFAULT 1,
    content JSONB NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_policies_version_active ON policies(active, version DESC);
CREATE INDEX IF NOT EXISTS idx_policies_version ON policies(version);
CREATE INDEX IF NOT EXISTS idx_policies_updated_at ON policies(updated_at);

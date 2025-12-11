-- Fix Resilience Subsystem Schema Mismatches

-- 1. circuit_breakers
ALTER TABLE circuit_breakers 
    RENAME COLUMN name TO breaker_name;

ALTER TABLE circuit_breakers
    ADD COLUMN IF NOT EXISTS half_open_max_requests INTEGER;

-- 2. rate_limiters
ALTER TABLE rate_limiters
    RENAME COLUMN name TO limiter_name;

ALTER TABLE rate_limiters
    ADD COLUMN IF NOT EXISTS last_request_at TIMESTAMP WITH TIME ZONE;

-- 3. retry_policies
ALTER TABLE retry_policies
    RENAME COLUMN name TO policy_name;

ALTER TABLE retry_policies
    ADD COLUMN IF NOT EXISTS retryable_errors TEXT[],
    ADD COLUMN IF NOT EXISTS total_retries BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS successful_retries BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS failed_retries BIGINT DEFAULT 0;

-- 4. bulkheads
ALTER TABLE bulkheads
    RENAME COLUMN name TO bulkhead_name;

ALTER TABLE bulkheads
    ADD COLUMN IF NOT EXISTS peak_concurrent INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_timeout BIGINT DEFAULT 0;

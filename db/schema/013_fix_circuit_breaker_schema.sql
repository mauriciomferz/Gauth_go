-- Fix Circuit Breaker Missing Columns

ALTER TABLE circuit_breakers
    ADD COLUMN IF NOT EXISTS consecutive_failures INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS consecutive_successes INTEGER DEFAULT 0;

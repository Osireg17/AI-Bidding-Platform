-- === CONTEXT ===
-- Purpose: Create the auctions table for the auction-service.
-- This is the single source of truth for auction state.
--
-- === DATA / STATE ===
-- One row per auction. Status transitions: pending -> active -> ending_soon -> closed.
-- Prices stored as DOUBLE PRECISION (float64 in Go) for MVP simplicity.
-- Trade-off: real financial systems would use INTEGER cents to avoid floating-point errors.

CREATE TABLE IF NOT EXISTS auctions (
    id            TEXT PRIMARY KEY,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    start_price   DOUBLE PRECISION NOT NULL,
    current_price DOUBLE PRECISION NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    winner_bot_id TEXT NOT NULL DEFAULT '',
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index on status for the scheduler queries (FindExpiredActive, FindEndingSoon).
CREATE INDEX IF NOT EXISTS idx_auctions_status ON auctions (status);

-- Index on end_time for finding expired and ending-soon auctions.
CREATE INDEX IF NOT EXISTS idx_auctions_end_time ON auctions (end_time);

-- Composite index for the most common scheduler query pattern.
CREATE INDEX IF NOT EXISTS idx_auctions_status_end_time ON auctions (status, end_time);

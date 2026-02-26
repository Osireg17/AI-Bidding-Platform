CREATE TABLE IF NOT EXISTS auctions (
    id            BIGSERIAL PRIMARY KEY,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    start_price   DOUBLE PRECISION NOT NULL,
    current_price DOUBLE PRECISION NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    winner_bot_id BIGINT NOT NULL DEFAULT 0,
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

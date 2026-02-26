CREATE TABLE IF NOT EXISTS auction_snapshots (
    auction_id BIGINT PRIMARY KEY,
    title TEXT NOT NULL,
    start_price DOUBLE PRECISION NOT NULL,
    status TEXT NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
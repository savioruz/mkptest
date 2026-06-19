BEGIN;

CREATE TABLE IF NOT EXISTS studios (
    id VARCHAR(36) PRIMARY KEY,
    cinema_id VARCHAR(36) NOT NULL REFERENCES cinemas(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    total_seats INT NOT NULL DEFAULT 0,
    row_count INT NOT NULL DEFAULT 0,
    cols_per_row INT NOT NULL DEFAULT 0,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now(),
    modified_at TIMESTAMPTZ DEFAULT now(),
    created_by VARCHAR(36) NOT NULL,
    modified_by VARCHAR(36) NOT NULL,
    CONSTRAINT uq_studio_name_per_cinema UNIQUE (cinema_id, name)
);

CREATE INDEX idx_studios_cinema_id ON studios(cinema_id);

COMMIT;

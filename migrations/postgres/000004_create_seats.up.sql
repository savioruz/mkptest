BEGIN;

CREATE TABLE IF NOT EXISTS seats (
    id VARCHAR(36) PRIMARY KEY,
    studio_id VARCHAR(36) NOT NULL REFERENCES studios(id) ON DELETE CASCADE,
    seat_label VARCHAR(8) NOT NULL,
    row_label VARCHAR(4) DEFAULT NULL,
    seat_number INT DEFAULT NULL,
    seat_type VARCHAR(16) NOT NULL DEFAULT 'regular',
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now(),
    modified_at TIMESTAMPTZ DEFAULT now(),
    created_by VARCHAR(36) NOT NULL,
    modified_by VARCHAR(36) NOT NULL,
    CONSTRAINT uq_seat_label_per_studio UNIQUE (studio_id, seat_label)
);

CREATE INDEX idx_seats_studio_id ON seats(studio_id);

COMMIT;

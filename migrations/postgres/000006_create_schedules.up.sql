BEGIN;

-- Required for the GiST exclusion constraint that mixes equality (studio_id)
-- with range overlap (tstzrange) to prevent overlapping showtimes per studio.
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS schedules (
    id VARCHAR(36) PRIMARY KEY,
    movie_id VARCHAR(36) NOT NULL REFERENCES movies(id) ON DELETE RESTRICT,
    studio_id VARCHAR(36) NOT NULL REFERENCES studios(id) ON DELETE RESTRICT,
    show_date DATE NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    price NUMERIC(12, 2) NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'scheduled',
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now(),
    modified_at TIMESTAMPTZ DEFAULT now(),
    created_by VARCHAR(36) NOT NULL,
    modified_by VARCHAR(36) NOT NULL,
    CONSTRAINT chk_schedule_time CHECK (end_time > start_time)
);

ALTER TABLE schedules
    ADD CONSTRAINT no_studio_time_overlap
    EXCLUDE USING gist (
        studio_id WITH =,
        tstzrange(start_time, end_time) WITH &&
    ) WHERE (status <> 'cancelled');

CREATE INDEX idx_schedules_movie_id ON schedules(movie_id);
CREATE INDEX idx_schedules_studio_id ON schedules(studio_id);
CREATE INDEX idx_schedules_show_date ON schedules(show_date);
CREATE INDEX idx_schedules_status ON schedules(status);

COMMIT;

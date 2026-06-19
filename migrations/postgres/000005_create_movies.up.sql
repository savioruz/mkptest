BEGIN;

CREATE TABLE IF NOT EXISTS movies (
    id VARCHAR(36) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT DEFAULT NULL,
    duration_min INT NOT NULL,
    genre VARCHAR(100) DEFAULT NULL,
    rating VARCHAR(10) DEFAULT NULL,
    poster_url VARCHAR(512) DEFAULT NULL,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now(),
    modified_at TIMESTAMPTZ DEFAULT now(),
    created_by VARCHAR(36) NOT NULL,
    modified_by VARCHAR(36) NOT NULL
);

CREATE INDEX idx_movies_title ON movies(title);

COMMIT;

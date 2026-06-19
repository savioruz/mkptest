BEGIN;

CREATE TABLE IF NOT EXISTS cinemas (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    city VARCHAR(255) NOT NULL,
    address TEXT DEFAULT NULL,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now(),
    modified_at TIMESTAMPTZ DEFAULT now(),
    created_by VARCHAR(36) NOT NULL,
    modified_by VARCHAR(36) NOT NULL
);

CREATE INDEX idx_cinemas_city ON cinemas(city);
CREATE INDEX idx_cinemas_name ON cinemas(name);

COMMIT;

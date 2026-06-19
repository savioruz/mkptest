BEGIN;

CREATE TABLE IF NOT EXISTS booking_seats (
    id VARCHAR(36) PRIMARY KEY,
    booking_id VARCHAR(36) NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    schedule_id VARCHAR(36) NOT NULL REFERENCES schedules(id) ON DELETE RESTRICT,
    seat_id VARCHAR(36) NOT NULL REFERENCES seats(id) ON DELETE RESTRICT,
    status VARCHAR(16) NOT NULL DEFAULT 'held',
    price NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now(),
    modified_at TIMESTAMPTZ DEFAULT now(),
    created_by VARCHAR(36) NOT NULL,
    modified_by VARCHAR(36) NOT NULL
);

CREATE UNIQUE INDEX uq_active_seat_per_schedule
    ON booking_seats (schedule_id, seat_id)
    WHERE status = 'booked';

CREATE INDEX idx_booking_seats_booking_id ON booking_seats(booking_id);
CREATE INDEX idx_booking_seats_schedule_id ON booking_seats(schedule_id);

COMMIT;

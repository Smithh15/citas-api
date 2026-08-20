CREATE TABLE availability (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    doctor_id    UUID NOT NULL REFERENCES doctor_profiles(id) ON DELETE CASCADE,
    day_of_week  SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6), -- 0=domingo
    start_time   TIME NOT NULL,
    end_time     TIME NOT NULL,
    CHECK (end_time > start_time)
);

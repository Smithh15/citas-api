CREATE TABLE doctor_profiles (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    specialty             VARCHAR(100) NOT NULL,
    default_slot_minutes  INT NOT NULL DEFAULT 30,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

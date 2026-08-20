CREATE TYPE appointment_status AS ENUM ('pending', 'confirmed', 'cancelled', 'completed');

CREATE TABLE appointments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id   UUID NOT NULL REFERENCES users(id),
    doctor_id    UUID NOT NULL REFERENCES doctor_profiles(id),
    slot_start   TIMESTAMPTZ NOT NULL,
    slot_end     TIMESTAMPTZ NOT NULL,
    status       appointment_status NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (slot_end > slot_start)
);

-- EL CONSTRAINT QUE RESUELVE LA CONCURRENCIA:
-- Solo puede existir una cita activa (pending o confirmed) por doctor+horario.
-- Postgres lo garantiza a nivel de motor, sin importar cuántas peticiones lleguen a la vez.
CREATE UNIQUE INDEX idx_unique_active_slot
    ON appointments (doctor_id, slot_start)
    WHERE status IN ('pending', 'confirmed');

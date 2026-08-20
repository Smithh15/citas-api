-- name: GetAvailableSlots :many
WITH slots AS (
    -- Las columnas start_time/end_time no tienen timezone: representan hora local
    -- del consultorio (America/Bogota), no UTC. Sin el AT TIME ZONE explícito, Postgres
    -- las interpretaría como UTC al convertirlas a timestamptz (bug detectado en pruebas).
    SELECT (generate_series(
        (sqlc.arg(target_date)::date + a.start_time) AT TIME ZONE 'America/Bogota',
        ((sqlc.arg(target_date)::date + a.end_time) AT TIME ZONE 'America/Bogota') - (dp.default_slot_minutes || ' minutes')::interval,
        (dp.default_slot_minutes || ' minutes')::interval
    ))::timestamptz AS slot_start
    FROM availability a
    JOIN doctor_profiles dp ON dp.id = a.doctor_id
    WHERE a.doctor_id = sqlc.arg(doctor_id)
      AND a.day_of_week = EXTRACT(DOW FROM sqlc.arg(target_date)::date)
)
SELECT s.slot_start
FROM slots s
WHERE NOT EXISTS (
    SELECT 1 FROM appointments ap
    WHERE ap.doctor_id = sqlc.arg(doctor_id)
      AND ap.slot_start = s.slot_start
      AND ap.status IN ('pending', 'confirmed')
)
ORDER BY s.slot_start;

-- name: CreateAppointment :one
INSERT INTO appointments (patient_id, doctor_id, slot_start, slot_end, status)
VALUES ($1, $2, $3, $4, 'pending')
ON CONFLICT (doctor_id, slot_start) WHERE status IN ('pending', 'confirmed')
DO NOTHING
RETURNING *;

-- name: GetAppointmentByID :one
SELECT * FROM appointments WHERE id = $1;

-- name: CancelAppointment :one
UPDATE appointments
SET status = 'cancelled'
WHERE id = $1 AND status IN ('pending', 'confirmed')
RETURNING *;

-- name: ReleaseExpiredPendingAppointments :many
UPDATE appointments
SET status = 'cancelled'
WHERE status = 'pending'
  AND created_at < now() - (sqlc.arg(hold_minutes)::int * interval '1 minute')
RETURNING id, doctor_id, slot_start;

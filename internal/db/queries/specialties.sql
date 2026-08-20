-- name: GetOrCreateSpecialty :one
INSERT INTO specialties (name) VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: AddDoctorSpecialty :exec
INSERT INTO doctor_specialties (doctor_id, specialty_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetDoctorSpecialties :many
SELECT s.* FROM specialties s
JOIN doctor_specialties ds ON ds.specialty_id = s.id
WHERE ds.doctor_id = $1;

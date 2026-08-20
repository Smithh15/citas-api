-- name: CreateDoctorProfile :one
INSERT INTO doctor_profiles (user_id, specialty)
VALUES ($1, $2)
RETURNING *;

-- name: GetDoctorProfileByUserID :one
SELECT * FROM doctor_profiles
WHERE user_id = $1;

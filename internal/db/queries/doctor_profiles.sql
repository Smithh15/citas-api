-- name: CreateDoctorProfile :one
INSERT INTO doctor_profiles (user_id)
VALUES ($1)
RETURNING *;

-- name: GetDoctorProfileByUserID :one
SELECT * FROM doctor_profiles
WHERE user_id = $1;

-- name: CreateAvailability :one
INSERT INTO availability (doctor_id, day_of_week, start_time, end_time)
VALUES ($1, $2, $3, $4)
RETURNING *;

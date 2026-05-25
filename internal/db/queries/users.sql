-- name: CreateUser :one
INSERT INTO users (email, password, full_name)
VALUES ($1, $2, $3)
RETURNING id, email, password, full_name, created_at;

-- name: GetUserByEmail :one
SELECT id, email, password, full_name, created_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password, full_name, created_at
FROM users
WHERE id = $1;
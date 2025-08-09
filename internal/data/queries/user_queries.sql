--
-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = ?;
--
-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = ?;
--
-- name: CreateUser :one
INSERT INTO users (name, email, password_hash, activated)
VALUES (?, ?, ?, ?)
RETURNING *;
--
-- name: UpdateUserPasswordByEmail :execrows
UPDATE users
SET password_hash = ?
WHERE email = ?;

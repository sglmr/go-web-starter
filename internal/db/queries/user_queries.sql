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
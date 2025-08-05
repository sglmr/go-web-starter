-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = ?;
-- name: CreateUser :one
INSERT INTO users (
    name,
    email,
    password
) VALUES (
    $1, $2, $3
)
RETURNING id, name, email, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, name, email, password, created_at, updated_at 
FROM users 
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, name, email, password, created_at, updated_at 
FROM users 
WHERE email = $1 LIMIT 1;

-- name: UpdateUser :one
UPDATE users
SET 
    name = COALESCE(sqlc.narg('name'), name),
    email = COALESCE(sqlc.narg('email'), email),
    password = COALESCE(sqlc.narg('password'), password)
WHERE id = $1
RETURNING id, name, email, updated_at;

-- name: DeleteUser :exec
DELETE FROM users 
WHERE id = $1;
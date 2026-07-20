-- name: CreateUser :one
INSERT INTO users (email, password_hash, is_admin)
VALUES ($1, $2, $3)
    RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: SetUserWhatsmeowJID :exec
UPDATE users SET whatsmeow_jid = $2 WHERE id = $1;

-- name: ListPairedUsers :many
SELECT * FROM users WHERE whatsmeow_jid IS NOT NULL;

-- name: SetUserActive :exec
UPDATE users SET is_active = $2 WHERE id = $1;

-- name: SetUserAdmin :exec
UPDATE users SET is_admin = $2 WHERE id = $1;

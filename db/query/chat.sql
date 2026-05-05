-- name: CreateSession :one
INSERT INTO chat_sessions (customer_id, status)
VALUES ($1, 'WAITING_USER_REPLY')
    RETURNING *;

-- name: GetActiveSessionByPhone :one
SELECT cs.*
FROM chat_sessions cs
         JOIN customers c ON cs.customer_id = c.id
WHERE c.phone_number = $1 AND cs.status != 'RESOLVED'
LIMIT 1;

-- name: UpdateSessionStatus :exec
UPDATE chat_sessions
SET status = $2, last_interaction_at = NOW()
WHERE id = $1;

-- name: InsertMessage :one
INSERT INTO message_history (session_id, sender_type, content)
VALUES ($1, $2, $3)
    RETURNING *;

-- name: GetSessionMessages :many
SELECT * FROM message_history
WHERE session_id = $1
ORDER BY created_at ASC;

-- name: CleanSessions :exec
UPDATE chat_sessions SET status = 'RESOLVED'
WHERE status IN ('WAITING_USER_REPLY', 'AI_HANDLING')
AND last_interaction_at < NOW() - INTERVAL '24 hours';

-- name: GetCustomerByPhone :one
SELECT * FROM customers WHERE phone_number = $1 LIMIT 1;

-- name: CreateCustomer :one
INSERT INTO customers (phone_number, name)
VALUES ($1, $2)
    RETURNING *;
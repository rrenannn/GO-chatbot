package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	db "github.com/rrenannn/GO-chatbot/db/sqlc"
)

type RepositoryInterface interface {
	GetActiveSessionByPhone(ctx context.Context, phone string) (db.ChatSession, error)
	CreateSession(ctx context.Context, customerID uuid.UUID) (db.ChatSession, error)
	UpdateSessionStatus(ctx context.Context, sessionID uuid.UUID, status db.SessionStatus) error
	InsertMessage(ctx context.Context, sessionID uuid.UUID, senderType string, content string) (db.MessageHistory, error)
	GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]db.MessageHistory, error)
	CleanSessions(ctx context.Context) error
}

type chatRepo struct {
	conn *sql.DB
	q    *db.Queries
}

func NewRepository(conn *sql.DB, q *db.Queries) RepositoryInterface {
	return &chatRepo{conn: conn, q: q}
}

func (r *chatRepo) GetActiveSessionByPhone(ctx context.Context, phone string) (db.ChatSession, error) {
	return r.q.GetActiveSessionByPhone(ctx, phone)
}

func (r *chatRepo) CreateSession(ctx context.Context, customerID uuid.UUID) (db.ChatSession, error) {
	return r.q.CreateSession(ctx, uuid.NullUUID{UUID: customerID, Valid: true})
}

func (r *chatRepo) UpdateSessionStatus(ctx context.Context, sessionID uuid.UUID, status db.SessionStatus) error {
	return r.q.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
		ID:     sessionID,
		Status: db.NullSessionStatus{Valid: true, SessionStatus: status},
	})
}

func (r *chatRepo) InsertMessage(ctx context.Context, sessionID uuid.UUID, senderType string, content string) (db.MessageHistory, error) {
	return r.q.InsertMessage(ctx, db.InsertMessageParams{
		SessionID:  uuid.NullUUID{UUID: sessionID, Valid: true},
		SenderType: sql.NullString{String: senderType, Valid: true},
		Content:    content,
	})
}

func (r *chatRepo) GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]db.MessageHistory, error) {
	return r.q.GetSessionMessages(ctx, uuid.NullUUID{UUID: sessionID, Valid: true})
}

func (r *chatRepo) CleanSessions(ctx context.Context) error {
	return r.q.CleanSessions(ctx)
}

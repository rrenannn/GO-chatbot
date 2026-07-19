package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	db "github.com/rrenannn/GO-chatbot/db/sqlc"
)

type ChatRepository interface {
	GetActiveSessionByPhone(ctx context.Context, userID uuid.UUID, phone string) (db.ChatSession, error)
	CreateSession(ctx context.Context, customerID uuid.UUID) (db.ChatSession, error)
	UpdateSessionStatus(ctx context.Context, sessionID uuid.UUID, status db.SessionStatus) error
	InsertMessage(ctx context.Context, sessionID uuid.UUID, senderType string, content string) (db.MessageHistory, error)
	GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]db.MessageHistory, error)
	CleanSessions(ctx context.Context) error
	GetCustomerByPhone(ctx context.Context, userID uuid.UUID, phone string) (db.Customer, error)
	CreateCustomer(ctx context.Context, userID uuid.UUID, phone string, name string) (db.Customer, error)

	CreateUser(ctx context.Context, email string, passwordHash string) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	SetUserWhatsmeowJID(ctx context.Context, userID uuid.UUID, jid string) error
	ListPairedUsers(ctx context.Context) ([]db.User, error)
}

type chatRepo struct {
	q *db.Queries
}

func NewChatRepository(dbConn *sql.DB) ChatRepository {
	return &chatRepo{
		q: db.New(dbConn), // db.New() do sqlc aceita *sql.DB nativamente
	}
}

func (r *chatRepo) GetActiveSessionByPhone(ctx context.Context, userID uuid.UUID, phone string) (db.ChatSession, error) {
	return r.q.GetActiveSessionByPhone(ctx, db.GetActiveSessionByPhoneParams{
		UserID:      uuid.NullUUID{UUID: userID, Valid: true},
		PhoneNumber: phone,
	})
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

func (r *chatRepo) GetCustomerByPhone(ctx context.Context, userID uuid.UUID, phone string) (db.Customer, error) {
	return r.q.GetCustomerByPhone(ctx, db.GetCustomerByPhoneParams{
		UserID:      uuid.NullUUID{UUID: userID, Valid: true},
		PhoneNumber: phone,
	})
}

func (r *chatRepo) CreateCustomer(ctx context.Context, userID uuid.UUID, phone string, name string) (db.Customer, error) {
	return r.q.CreateCustomer(ctx, db.CreateCustomerParams{
		UserID:      uuid.NullUUID{UUID: userID, Valid: true},
		PhoneNumber: phone,
		Name:        name,
	})
}

func (r *chatRepo) CreateUser(ctx context.Context, email string, passwordHash string) (db.User, error) {
	return r.q.CreateUser(ctx, db.CreateUserParams{Email: email, PasswordHash: passwordHash})
}

func (r *chatRepo) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return r.q.GetUserByEmail(ctx, email)
}

func (r *chatRepo) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *chatRepo) SetUserWhatsmeowJID(ctx context.Context, userID uuid.UUID, jid string) error {
	return r.q.SetUserWhatsmeowJID(ctx, db.SetUserWhatsmeowJIDParams{
		ID:           userID,
		WhatsmeowJid: sql.NullString{String: jid, Valid: jid != ""},
	})
}

func (r *chatRepo) ListPairedUsers(ctx context.Context) ([]db.User, error) {
	return r.q.ListPairedUsers(ctx)
}

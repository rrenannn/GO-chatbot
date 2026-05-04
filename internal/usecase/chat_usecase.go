package usecase

import "go.mau.fi/whatsmeow"

type ChatUseCase interface {
}

type chatUseCase struct {
	repo   *db.Queries
	client *whatsmeow.Client
}

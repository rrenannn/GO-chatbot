package whatsapp

import (
	"context"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func NewWhatsappClient() (*whatsmeow.Client, error) {
	ctx := context.Background()
	dbLog := waLog.Stdout("Database", "DEBUG", true)

	container, err := sqlstore.New(ctx, "sqlite3", "file:sessions.db?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no banco do whatsmeow: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	client.EnableAutoReconnect = true
	client.AutoTrustIdentity = true

	return client, nil
}

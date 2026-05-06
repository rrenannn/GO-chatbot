package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

func NewWhatsAppClient() (*whatsmeow.Client, error) {
	ctx := context.Background()
	dbLog := waLog.Stdout("Database", "DEBUG", true)

	container, err := sqlstore.New(ctx, "sqlite", "file:/app/data/sessions.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbLog)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no banco do whatsmeow: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	client.EnableAutoReconnect = true
	client.AutoTrustIdentity = true

	return client, nil
}

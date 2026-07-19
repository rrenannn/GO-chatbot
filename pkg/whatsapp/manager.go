package whatsapp

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

// Manager mantém um cliente whatsmeow isolado por usuário, todos apoiados no
// mesmo banco sqlite (o whatsmeow suporta múltiplos dispositivos num único
// Container). Cada usuário só enxerga o próprio pareamento/sessão.
type Manager struct {
	container *sqlstore.Container

	mu      sync.Mutex
	clients map[uuid.UUID]*whatsmeow.Client

	// OnEvent é chamado para todo evento recebido por qualquer cliente gerenciado.
	OnEvent func(userID uuid.UUID, client *whatsmeow.Client, evt interface{})
	// OnPaired é chamado quando o pareamento de um usuário é concluído, para persistir o JID.
	OnPaired func(userID uuid.UUID, jid string)
	// OnLoggedOut é chamado quando o usuário desconecta o aparelho pelo próprio
	// WhatsApp (fora do nosso controle), para limpar o JID salvo no banco.
	OnLoggedOut func(userID uuid.UUID)
}

func NewManager() (*Manager, error) {
	ctx := context.Background()
	dbLog := waLog.Stdout("Database", "WARN", true)

	container, err := sqlstore.New(ctx, "sqlite", "file:/app/data/sessions.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbLog)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no banco do whatsmeow: %w", err)
	}

	return &Manager{
		container: container,
		clients:   make(map[uuid.UUID]*whatsmeow.Client),
	}, nil
}

// GetClient retorna (criando se necessário) o cliente whatsmeow de um usuário.
// existingJID deve ser o JID salvo no banco (vazio se o usuário nunca pareou).
func (m *Manager) GetClient(userID uuid.UUID, existingJID string) (*whatsmeow.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.clients[userID]; ok {
		return c, nil
	}

	ctx := context.Background()
	var device *store.Device

	if existingJID != "" {
		jid, err := types.ParseJID(existingJID)
		if err != nil {
			return nil, fmt.Errorf("JID salvo inválido: %w", err)
		}
		device, err = m.container.GetDevice(ctx, jid)
		if err != nil {
			return nil, fmt.Errorf("falha ao carregar dispositivo: %w", err)
		}
	}
	if device == nil {
		device = m.container.NewDevice()
	}

	clientLog := waLog.Stdout("Client", "WARN", true)
	client := whatsmeow.NewClient(device, clientLog)
	client.EnableAutoReconnect = true
	client.AutoTrustIdentity = true

	client.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.PairSuccess:
			if client.Store.ID != nil && m.OnPaired != nil {
				m.OnPaired(userID, client.Store.ID.String())
			}

		case *events.LoggedOut:
			// O whatsmeow NÃO limpa Store.ID sozinho quando o aparelho é
			// desconectado pelo celular, então sem isso o app continuaria achando
			// que a sessão está válida para sempre. Apaga o dispositivo e derruba
			// o cliente do cache: a próxima vez que esse usuário abrir a tela de
			// QR, um dispositivo novo (sem ID) é criado, permitindo um novo pareamento.
			go func() {
				if err := m.container.DeleteDevice(context.Background(), client.Store); err != nil {
					fmt.Println("🚨 Erro ao remover dispositivo deslogado:", userID, err)
				}
			}()

			m.mu.Lock()
			delete(m.clients, userID)
			m.mu.Unlock()

			if m.OnLoggedOut != nil {
				m.OnLoggedOut(userID)
			}
		}

		if m.OnEvent != nil {
			m.OnEvent(userID, client, evt)
		}
	})

	m.clients[userID] = client

	// Se o dispositivo já estava pareado (carregado de uma sessão anterior),
	// reconecta imediatamente em vez de esperar por uma requisição de QR code.
	if client.Store.ID != nil {
		go func() {
			if err := client.Connect(); err != nil {
				fmt.Println("🚨 Erro ao reconectar cliente existente:", userID, err)
			}
		}()
	}

	return client, nil
}

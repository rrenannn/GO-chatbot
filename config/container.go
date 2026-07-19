package config

import (
	"context"
	"database/sql"
	"log"

	"github.com/google/uuid"
	"github.com/rrenannn/GO-chatbot/internal/delivery/http"
	wpp "github.com/rrenannn/GO-chatbot/internal/delivery/whatsapp"
	"github.com/rrenannn/GO-chatbot/internal/repository"
	"github.com/rrenannn/GO-chatbot/internal/usecase"
	"github.com/rrenannn/GO-chatbot/internal/worker"
	"github.com/rrenannn/GO-chatbot/pkg/database"
	"github.com/rrenannn/GO-chatbot/pkg/whatsapp"
)

type ContainerDI struct {
	Config    Config
	Conn      *sql.DB
	WaManager *whatsapp.Manager

	// Domínio de Chat
	ChatRepo    repository.ChatRepository
	ChatUC      usecase.ChatUseCase
	HttpHandler *http.HttpHandler
}

func NewContainerDI(config Config) *ContainerDI {
	container := &ContainerDI{Config: config}

	container.db()
	container.buildRepositories()
	container.buildUseCases()
	container.buildClients()
	container.buildHandlers()
	container.reconnectPairedUsers()

	// Opcional: Inicia o worker de limpeza em background
	container.startWorkers()

	return container
}

func (c *ContainerDI) db() {
	dbConfig := database.Config{
		Host:        c.Config.DBHost,
		Port:        c.Config.DBPort,
		User:        c.Config.DBUser,
		Password:    c.Config.DBPassword,
		Database:    c.Config.DBDatabase,
		SSLMode:     c.Config.DBSSLMode,
		Driver:      c.Config.DBDriver,
		Environment: c.Config.Environment,
	}

	c.Conn = database.NewConnection(&dbConfig)
}

func (c *ContainerDI) buildRepositories() {
	c.ChatRepo = repository.NewChatRepository(c.Conn)
}

func (c *ContainerDI) buildUseCases() {
	c.ChatUC = usecase.NewChatUseCase(c.ChatRepo)
}

func (c *ContainerDI) buildClients() {
	manager, err := whatsapp.NewManager()
	if err != nil {
		panic("Erro ao inicializar Whatsmeow: " + err.Error())
	}

	waHandler := wpp.NewWhatsAppHandler(c.ChatUC)
	manager.OnEvent = waHandler.HandleEvent
	manager.OnPaired = func(userID uuid.UUID, jid string) {
		if err := c.ChatRepo.SetUserWhatsmeowJID(context.Background(), userID, jid); err != nil {
			log.Println("🚨 Erro ao salvar JID do usuário:", err)
		}
	}

	c.WaManager = manager
}

func (c *ContainerDI) buildHandlers() {
	c.HttpHandler = http.NewHttpHandler(c.ChatUC, c.ChatRepo, c.WaManager, c.Config.JWTSecret, c.Config.AdminAPIKey)
}

// reconnectPairedUsers restaura, no boot, a conexão de quem já havia pareado
// o WhatsApp antes (sem isso, o usuário só reconectaria ao abrir a tela de QR).
func (c *ContainerDI) reconnectPairedUsers() {
	users, err := c.ChatRepo.ListPairedUsers(context.Background())
	if err != nil {
		log.Println("🚨 Erro ao listar usuários pareados:", err)
		return
	}

	for _, u := range users {
		if _, err := c.WaManager.GetClient(u.ID, u.WhatsmeowJid.String); err != nil {
			log.Println("🚨 Erro ao reconectar usuário", u.ID, ":", err)
		}
	}
}

func (c *ContainerDI) startWorkers() {
	worker.StartSessionCleaner(c.ChatRepo)
	log.Println("Worker de limpeza de sessões inicializado.")
}

// Close garante que as conexões externas sejam encerradas no shutdown
func (c *ContainerDI) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
}

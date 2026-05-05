package config

import (
	"context"
	"database/sql"
	"log"

	"github.com/rrenannn/GO-chatbot/internal/delivery/http"
	wpp "github.com/rrenannn/GO-chatbot/internal/delivery/whatsapp"
	"github.com/rrenannn/GO-chatbot/internal/repository"
	"github.com/rrenannn/GO-chatbot/internal/usecase"
	"github.com/rrenannn/GO-chatbot/internal/worker"
	"github.com/rrenannn/GO-chatbot/pkg/database"
	"github.com/rrenannn/GO-chatbot/pkg/llm"
	"github.com/rrenannn/GO-chatbot/pkg/whatsapp"
	"go.mau.fi/whatsmeow"
)

type ContainerDI struct {
	Config   Config
	Conn     *sql.DB
	WaClient *whatsmeow.Client
	AIClient llm.AIClient

	// Domínio de Chat
	ChatRepo    repository.ChatRepository
	ChatUC      usecase.ChatUseCase
	HttpHandler *http.HttpHandler
}

func NewContainerDI(config Config) *ContainerDI {
	container := &ContainerDI{Config: config}

	container.db()
	container.buildClients()
	container.buildRepositories()
	container.buildUseCases()
	container.buildHandlers()

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

func (c *ContainerDI) buildClients() {
	// 1. Constrói cliente Whatsmeow
	waClient, err := whatsapp.NewWhatsAppClient()
	if err != nil {
		panic("Erro ao inicializar Whatsmeow: " + err.Error())
	}
	c.WaClient = waClient

	// 2. Constrói cliente Gemini
	aiClient, err := llm.NewGeminiClient(context.Background(), c.Config.GeminiAPIKey)
	if err != nil {
		panic("Erro ao inicializar Gemini: " + err.Error())
	}
	c.AIClient = aiClient
}

func (c *ContainerDI) buildRepositories() {
	c.ChatRepo = repository.NewChatRepository(c.Conn)
}

func (c *ContainerDI) buildUseCases() {
	c.ChatUC = usecase.NewChatUseCase(c.ChatRepo, c.WaClient, c.AIClient)
}

func (c *ContainerDI) buildHandlers() {
	// Instancia o handler do Echo
	c.HttpHandler = http.NewHttpHandler(c.ChatUC, c.WaClient)

	// Registra o listener de eventos do WhatsApp diretamente no client
	wpp.NewWhatsAppHandler(c.WaClient, c.ChatUC)
}

func (c *ContainerDI) startWorkers() {
	worker.StartSessionCleaner(c.ChatRepo)
	log.Println("Worker de limpeza de sessões inicializado.")
}

// Close garante que as conexões externas sejam encerradas no shutdown
func (c *ContainerDI) Close() {
	if c.WaClient != nil {
		c.WaClient.Disconnect()
	}
	if c.Conn != nil {
		c.Conn.Close()
	}
}

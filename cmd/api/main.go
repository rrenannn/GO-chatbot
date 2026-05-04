package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rrenannn/GO-chatbot/internal/delivery/http"
	wpp "github.com/rrenannn/GO-chatbot/internal/delivery/whatsapp"
	"github.com/rrenannn/GO-chatbot/internal/repository"
	"github.com/rrenannn/GO-chatbot/internal/usecase"
	"github.com/rrenannn/GO-chatbot/internal/worker"
	"github.com/rrenannn/GO-chatbot/pkg/llm"
	"github.com/rrenannn/GO-chatbot/pkg/whatsapp"
)

func main() {
	ctx := context.Background()

	dbUrl := os.Getenv("DATABASE_URL")

	dbConn, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("Falha ao abrir o banco de dados: %v", err)
	}
	defer dbConn.Close()

	if err := dbConn.Ping(); err != nil {
		log.Fatalf("Falha ao conectar fisicamente no banco de dados: %v", err)
	}
	log.Println("Conectado ao PostgreSQL com sucesso!")

	dbConn.SetMaxOpenConns(25)
	dbConn.SetMaxIdleConns(25)

	waClient, err := whatsapp.NewWhatsAppClient()
	if err != nil {
		log.Fatalf("Falha ao inicializar Whatsmeow: %v", err)
	}
	defer waClient.Disconnect()

	geminiKey := os.Getenv("GEMINI_API_KEY")
	aiClient, err := llm.NewGeminiClient(ctx, geminiKey)
	if err != nil {
		log.Fatalf("Falha ao inicializar Gemini: %v", err)
	}

	chatRepo := repository.NewChatRepository(dbConn)
	chatUC := usecase.NewChatUseCase(chatRepo, waClient, aiClient)
	wpp.NewWhatsappHandler(waClient)

	worker.StartSessionCleaner(chatRepo)

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	http.NewEchoHandler(e, chatUC, waClient)

	if waClient.Store.ID != nil {
		err = waClient.Connect()
		if err != nil {
			log.Fatalf("Falha ao conectar WhatsApp: %v", err)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		log.Printf("Servidor Echo rodando na porta %s", port)
		if err := e.Start(":" + port); err != nil {
			e.Logger.Fatal("Erro no servidor HTTP")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Desligando o servidor...")

	waClient.Disconnect()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	log.Println("Servidor finalizado.")
}

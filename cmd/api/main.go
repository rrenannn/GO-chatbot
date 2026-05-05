package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rrenannn/GO-chatbot/config"
)

func main() {
	// 1. Carrega as variáveis de ambiente
	cfg := config.NewConfig()

	// 2. Inicializa o Container de Dependências
	container := config.NewContainerDI(cfg)
	defer container.Close() // Garante que o banco e o WhatsApp desconectem ao final

	// 3. Configura e inicializa o Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Registra as rotas da API vindas do Container
	container.HttpHandler.RegisterRoutes(e)

	// 4. Conecta o WhatsApp (Se já tiver sessão salva)
	if container.WaClient.Store.ID != nil {
		log.Println("Sessão do WhatsApp encontrada. Conectando...")
		if err := container.WaClient.Connect(); err != nil {
			log.Fatalf("Falha ao conectar WhatsApp: %v", err)
		}
	}

	port := cfg.ServerPort
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

	if err := e.Shutdown(context.Background()); err != nil {
		e.Logger.Fatal(err)
	}

	log.Println("Servidor finalizado com sucesso.")
}

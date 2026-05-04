package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rrenannn/GO-chatbot/internal/usecase"
	"go.mau.fi/whatsmeow"
)

type HttpHandler struct {
	chatUC   usecase.ChatUseCase
	waClient *whatsmeow.Client
}

func NewEchoHandler(e *echo.Echo, uc usecase.ChatUseCase, waClient *whatsmeow.Client) {
	handler := &HttpHandler{
		chatUC:   uc,
		waClient: waClient,
	}

	// Suas rotas da API
	api := e.Group("/api/v1")

	// Rota que o seu sistema de vendas chama para iniciar o pós-venda
	api.POST("/trigger-post-sale", handler.TriggerPostSale)

	// Rota que o Front-end vai consumir via EventSource para ler o QR Code
	api.GET("/whatsapp/qr", handler.StreamQRCode)
}

func (h *HttpHandler) StreamQRCode(c echo.Context) error {
	// Configura os headers para SSE
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")

	// Se já estiver logado, avisa o front e encerra
	if h.waClient.Store.ID != nil {
		fmt.Fprintf(c.Response(), "data: {\"status\": \"CONNECTED\"}\n\n")
		c.Response().Flush()
		return nil
	}

	// Pega o canal de eventos do QR Code
	qrChan, _ := h.waClient.GetQRChannel(context.Background())
	err := h.waClient.Connect()
	if err != nil {
		return err
	}

	// Fica escutando o canal e enviando para o front
	for evt := range qrChan {
		if evt.Event == "code" {
			// Envia o código em formato JSON
			fmt.Fprintf(c.Response(), "data: {\"status\": \"QR_CODE\", \"code\": \"%s\"}\n\n", evt.Code)
			c.Response().Flush()
		} else if evt.Event == "success" {
			fmt.Fprintf(c.Response(), "data: {\"status\": \"CONNECTED\"}\n\n")
			c.Response().Flush()
			break
		}
	}

	return nil
}

func (h *HttpHandler) TriggerPostSale(c echo.Context) error {
	// Implementação do gatilho (recebe JSON com phone e customer_id e chama h.chatUC.TriggerPostSale)
	return c.JSON(http.StatusOK, map[string]string{"message": "Atendimento iniciado"})
}

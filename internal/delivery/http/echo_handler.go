package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/rrenannn/GO-chatbot/internal/usecase"
	"go.mau.fi/whatsmeow"
)

type HttpHandler struct {
	chatUC   usecase.ChatUseCase
	waClient *whatsmeow.Client
}

func NewHttpHandler(uc usecase.ChatUseCase, waClient *whatsmeow.Client) *HttpHandler {
	return &HttpHandler{
		chatUC:   uc,
		waClient: waClient,
	}
}

func (h *HttpHandler) RegisterRoutes(e *echo.Echo) {
	api := e.Group("/api/v1")

	api.POST("/trigger-post-sale", h.TriggerPostSale)
	api.GET("/whatsapp/qr", h.StreamQRCode)
}

var (
	isConnecting bool
	mu           sync.Mutex
)

func (h *HttpHandler) StreamQRCode(c echo.Context) error {
	// Headers SSE
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")

	// 1. Já está autenticado no banco? Retorna conectado.
	if h.waClient.Store.ID != nil {
		fmt.Fprintf(c.Response(), "data: {\"status\": \"CONNECTED\"}\n\n")
		c.Response().Flush()
		return nil
	}

	// 2. REGRA DE OURO: Pegar o canal SEMPRE antes de conectar
	qrChan, _ := h.waClient.GetQRChannel(context.Background())

	// 3. Conecta apenas se o WebSocket estiver fechado
	// O Whatsmeow já lida internamente com o controle de concorrência
	if !h.waClient.IsConnected() {
		err := h.waClient.Connect()
		if err != nil {
			fmt.Println("🚨 Erro ao conectar WhatsApp:", err)
			fmt.Fprintf(c.Response(), "data: {\"status\": \"ERROR\"}\n\n")
			c.Response().Flush()
			return nil
		}
	}

	// 4. Loop SSE com Select (Lida com fechamento de aba do navegador)
	for {
		select {
		case evt := <-qrChan:
			switch evt.Event {
			case "code":
				// Envia o código para o front-end renderizar
				fmt.Fprintf(c.Response(), "data: {\"status\": \"QR_CODE\", \"code\": \"%s\"}\n\n", evt.Code)
				c.Response().Flush()

			case "success":
				// Pareamento concluído com sucesso
				fmt.Fprintf(c.Response(), "data: {\"status\": \"CONNECTED\"}\n\n")
				c.Response().Flush()
				return nil

			case "timeout", "error":
				// O QR Code expirou ou houve erro na rede
				fmt.Fprintf(c.Response(), "data: {\"status\": \"ERROR\"}\n\n")
				c.Response().Flush()
				return nil
			}

		case <-c.Request().Context().Done():
			// O cliente atualizou a página ou fechou a aba do navegador
			// Interrompemos o loop para não gastar memória do EC2
			fmt.Println("⚠️ Conexão SSE encerrada pelo navegador.")
			return nil
		}
	}
}

type PostSaleRequest struct {
	Phone      string `json:"phone"`
	CustomerID string `json:"customer_id"`
}

func (h *HttpHandler) TriggerPostSale(c echo.Context) error {
	req := new(PostSaleRequest)

	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	// Como é um teste, se o customer_id vier vazio, criamos um UUID zerado para não quebrar o banco
	if req.CustomerID == "" {
		req.CustomerID = "00000000-0000-0000-0000-000000000000"
	}

	err := h.chatUC.TriggerPostSale(c.Request().Context(), req.Phone, req.CustomerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Atendimento iniciado"})
}

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

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

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
	api.POST("/broadcast", h.Broadcast)
}

// qrSharedState guarda o último status de pareamento conhecido. O whatsmeow só
// permite obter o canal de QR code uma única vez por processo, então em vez de
// chamar GetQRChannel a cada conexão SSE (o que trava conexões subsequentes num
// canal nil), um único goroutine observa o canal e todas as requisições SSE
// leem esse estado compartilhado.
type qrSharedState struct {
	mu      sync.Mutex
	status  string // "" (aguardando) | "QR_CODE" | "CONNECTED" | "ERROR"
	code    string
	version int
}

var (
	qrShared     = &qrSharedState{}
	qrWatcherOne sync.Once
)

func (s *qrSharedState) set(status, code string) {
	s.mu.Lock()
	s.status = status
	s.code = code
	s.version++
	s.mu.Unlock()
}

func (s *qrSharedState) snapshot() (status, code string, version int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status, s.code, s.version
}

// ensureQRWatcher inicia (uma única vez por processo) o goroutine que possui o
// canal de QR code do whatsmeow e conecta o cliente.
func (h *HttpHandler) ensureQRWatcher() {
	qrWatcherOne.Do(func() {
		qrChan, err := h.waClient.GetQRChannel(context.Background())
		if err != nil {
			fmt.Println("🚨 Erro ao obter canal de QR code:", err)
			qrShared.set("ERROR", "")
			return
		}

		if !h.waClient.IsConnected() {
			if err := h.waClient.Connect(); err != nil {
				fmt.Println("🚨 Erro ao conectar WhatsApp:", err)
				qrShared.set("ERROR", "")
				return
			}
		}

		go func() {
			for evt := range qrChan {
				switch evt.Event {
				case "code":
					qrShared.set("QR_CODE", evt.Code)
				case "success":
					qrShared.set("CONNECTED", "")
					return
				case "timeout", "error":
					qrShared.set("ERROR", "")
					return
				}
			}
		}()
	})
}

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

	h.ensureQRWatcher()

	// 2. Faz polling do estado compartilhado e envia só quando ele muda,
	// permitindo múltiplas abas/reconexões sem recriar o canal do whatsmeow.
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	lastVersion := -1
	for {
		select {
		case <-ticker.C:
			status, code, version := qrShared.snapshot()
			if version == lastVersion {
				continue
			}
			lastVersion = version

			switch status {
			case "QR_CODE":
				fmt.Fprintf(c.Response(), "data: {\"status\": \"QR_CODE\", \"code\": \"%s\"}\n\n", code)
				c.Response().Flush()
			case "CONNECTED":
				fmt.Fprintf(c.Response(), "data: {\"status\": \"CONNECTED\"}\n\n")
				c.Response().Flush()
				return nil
			case "ERROR":
				fmt.Fprintf(c.Response(), "data: {\"status\": \"ERROR\"}\n\n")
				c.Response().Flush()
				return nil
			}

		case <-c.Request().Context().Done():
			// O cliente atualizou a página ou fechou a aba do navegador
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

type BroadcastContactRequest struct {
	Phone string `json:"phone"`
	Name  string `json:"name"`
}

type BroadcastRequest struct {
	Contacts []BroadcastContactRequest `json:"contacts"`
	Message  string                    `json:"message"`
}

var phoneDigitsRe = regexp.MustCompile(`\D`)

// Broadcast envia a mensagem (com suporte à variável {{nome}}) para vários
// contatos via SSE, reportando o progresso de cada envio em tempo real
// (com delay aleatório entre eles).
func (h *HttpHandler) Broadcast(c echo.Context) error {
	req := new(BroadcastRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Mensagem vazia"})
	}

	contacts := make([]usecase.BroadcastContact, 0, len(req.Contacts))
	seen := map[string]bool{}
	for _, raw := range req.Contacts {
		p := phoneDigitsRe.ReplaceAllString(raw.Phone, "")
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		contacts = append(contacts, usecase.BroadcastContact{Phone: p, Name: strings.TrimSpace(raw.Name)})
	}

	if len(contacts) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Nenhum número válido informado"})
	}
	if len(contacts) > usecase.MaxBroadcastRecipients {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Máximo de %d destinatários por disparo", usecase.MaxBroadcastRecipients),
		})
	}

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")

	sendEvent := func(payload map[string]interface{}) {
		b, _ := json.Marshal(payload)
		fmt.Fprintf(c.Response(), "data: %s\n\n", b)
		c.Response().Flush()
	}

	err := h.chatUC.BroadcastMessage(c.Request().Context(), contacts, message, func(result usecase.BroadcastResult, sent, total int) {
		sendEvent(map[string]interface{}{
			"status":  "PROGRESS",
			"phone":   result.Phone,
			"success": result.Success,
			"error":   result.Error,
			"sent":    sent,
			"total":   total,
		})
	})

	if err != nil {
		sendEvent(map[string]interface{}{"status": "ERROR", "error": err.Error()})
		return nil
	}

	sendEvent(map[string]interface{}{"status": "DONE"})
	return nil
}

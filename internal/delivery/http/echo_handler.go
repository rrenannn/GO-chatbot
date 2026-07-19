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

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rrenannn/GO-chatbot/internal/auth"
	"github.com/rrenannn/GO-chatbot/internal/repository"
	"github.com/rrenannn/GO-chatbot/internal/usecase"
	"github.com/rrenannn/GO-chatbot/pkg/whatsapp"
	"go.mau.fi/whatsmeow"
)

type HttpHandler struct {
	chatUC      usecase.ChatUseCase
	repo        repository.ChatRepository
	waManager   *whatsapp.Manager
	jwtSecret   string
	adminAPIKey string

	qrMu    sync.Mutex
	qrState map[uuid.UUID]*qrSharedState
}

func NewHttpHandler(chatUC usecase.ChatUseCase, repo repository.ChatRepository, waManager *whatsapp.Manager, jwtSecret, adminAPIKey string) *HttpHandler {
	return &HttpHandler{
		chatUC:      chatUC,
		repo:        repo,
		waManager:   waManager,
		jwtSecret:   jwtSecret,
		adminAPIKey: adminAPIKey,
		qrState:     make(map[uuid.UUID]*qrSharedState),
	}
}

func (h *HttpHandler) RegisterRoutes(e *echo.Echo) {
	api := e.Group("/api/v1")

	api.POST("/auth/login", h.Login)
	api.POST("/admin/users", h.CreateUser)

	authed := api.Group("", h.authMiddleware)
	authed.POST("/trigger-post-sale", h.TriggerPostSale)
	authed.GET("/whatsapp/qr", h.StreamQRCode)
	authed.POST("/broadcast", h.Broadcast)
}

// authMiddleware aceita o token tanto no header Authorization: Bearer quanto
// numa query string (?token=...), já que EventSource não permite headers
// customizados no navegador.
func (h *HttpHandler) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token := c.QueryParam("token")
		if authHeader := c.Request().Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if token == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Não autenticado"})
		}

		userID, err := auth.ParseToken(h.jwtSecret, token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Sessão inválida, faça login novamente"})
		}

		c.Set("userID", userID)
		return next(c)
	}
}

func (h *HttpHandler) userID(c echo.Context) uuid.UUID {
	return c.Get("userID").(uuid.UUID)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *HttpHandler) Login(c echo.Context) error {
	req := new(LoginRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	user, err := h.repo.GetUserByEmail(c.Request().Context(), strings.TrimSpace(strings.ToLower(req.Email)))
	if err != nil || !auth.CheckPassword(req.Password, user.PasswordHash) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "E-mail ou senha inválidos"})
	}

	token, err := auth.IssueToken(h.jwtSecret, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Erro ao gerar sessão"})
	}

	return c.JSON(http.StatusOK, map[string]string{"token": token})
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CreateUser não é exposto no front — é usado manualmente (ex: via curl) pelo
// administrador para cadastrar quem pode acessar o painel, protegido pela
// chave ADMIN_API_KEY.
func (h *HttpHandler) CreateUser(c echo.Context) error {
	if h.adminAPIKey == "" || c.Request().Header.Get("X-Admin-Key") != h.adminAPIKey {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Não autorizado"})
	}

	req := new(CreateUserRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "E-mail obrigatório e senha com pelo menos 8 caracteres"})
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Erro ao gerar senha"})
	}

	user, err := h.repo.CreateUser(c.Request().Context(), email, hash)
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "Não foi possível criar (e-mail já existe?)"})
	}

	return c.JSON(http.StatusOK, map[string]string{"id": user.ID.String(), "email": user.Email})
}

// qrSharedState guarda o último status de pareamento conhecido de UM usuário.
// O whatsmeow só permite obter o canal de QR code uma única vez por tentativa
// de conexão, então em vez de chamar GetQRChannel a cada conexão SSE, um único
// goroutine por usuário observa o canal e todas as abas/reconexões desse
// usuário leem esse estado compartilhado.
type qrSharedState struct {
	mu      sync.Mutex
	status  string // "" (aguardando) | "QR_CODE" | "CONNECTED" | "ERROR"
	code    string
	version int
	active  bool // true enquanto um watcher está de fato tentando parear
}

func (h *HttpHandler) getQRState(userID uuid.UUID) *qrSharedState {
	h.qrMu.Lock()
	defer h.qrMu.Unlock()
	s, ok := h.qrState[userID]
	if !ok {
		s = &qrSharedState{}
		h.qrState[userID] = s
	}
	return s
}

// ensureQRWatcher garante que exista um goroutine tentando parear o cliente do
// usuário. Pode iniciar uma NOVA tentativa sempre que a anterior terminar
// (sucesso, erro ou timeout do WhatsApp) — sem isso, uma vez que o QR
// expirasse a conexão nunca mais tentaria de novo, exigindo reiniciar o app.
func (h *HttpHandler) ensureQRWatcher(state *qrSharedState, client *whatsmeow.Client) {
	state.mu.Lock()
	if state.active {
		state.mu.Unlock()
		return
	}
	state.active = true
	state.status = ""
	state.code = ""
	state.version++
	state.mu.Unlock()

	finish := func(status string) {
		state.mu.Lock()
		state.status = status
		state.code = ""
		state.version++
		state.active = false
		state.mu.Unlock()
	}

	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		fmt.Println("🚨 Erro ao obter canal de QR code:", err)
		finish("ERROR")
		return
	}

	if !client.IsConnected() {
		if err := client.Connect(); err != nil {
			fmt.Println("🚨 Erro ao conectar WhatsApp:", err)
			finish("ERROR")
			return
		}
	}

	go func() {
		for evt := range qrChan {
			switch evt.Event {
			case "code":
				state.mu.Lock()
				state.status = "QR_CODE"
				state.code = evt.Code
				state.version++
				state.mu.Unlock()
			case "success":
				finish("CONNECTED")
				return
			case "timeout", "error":
				finish("ERROR")
				return
			}
		}
	}()
}

func (h *HttpHandler) StreamQRCode(c echo.Context) error {
	userID := h.userID(c)

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")

	user, err := h.repo.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		fmt.Fprintf(c.Response(), "data: {\"status\": \"ERROR\"}\n\n")
		c.Response().Flush()
		return nil
	}

	client, err := h.waManager.GetClient(userID, user.WhatsmeowJid.String)
	if err != nil {
		fmt.Println("🚨 Erro ao obter cliente WhatsApp:", err)
		fmt.Fprintf(c.Response(), "data: {\"status\": \"ERROR\"}\n\n")
		c.Response().Flush()
		return nil
	}

	// 1. Já está autenticado? Retorna conectado.
	if client.Store.ID != nil {
		fmt.Fprintf(c.Response(), "data: {\"status\": \"CONNECTED\"}\n\n")
		c.Response().Flush()
		return nil
	}

	state := h.getQRState(userID)
	h.ensureQRWatcher(state, client)

	// 2. Faz polling do estado compartilhado e envia só quando ele muda,
	// permitindo múltiplas abas/reconexões sem recriar o canal do whatsmeow.
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	lastVersion := -1
	for {
		select {
		case <-ticker.C:
			state.mu.Lock()
			status, code, version := state.status, state.code, state.version
			state.mu.Unlock()

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
			fmt.Println("⚠️ Conexão SSE encerrada pelo navegador.")
			return nil
		}
	}
}

type PostSaleRequest struct {
	Phone string `json:"phone"`
}

func (h *HttpHandler) TriggerPostSale(c echo.Context) error {
	userID := h.userID(c)

	req := new(PostSaleRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	user, err := h.repo.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Usuário não encontrado"})
	}

	client, err := h.waManager.GetClient(userID, user.WhatsmeowJid.String)
	if err != nil || client.Store.ID == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "WhatsApp não conectado"})
	}

	if err := h.chatUC.TriggerPostSale(c.Request().Context(), client, userID, req.Phone); err != nil {
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
	userID := h.userID(c)

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

	user, err := h.repo.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Usuário não encontrado"})
	}

	client, err := h.waManager.GetClient(userID, user.WhatsmeowJid.String)
	if err != nil || client.Store.ID == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "WhatsApp não conectado"})
	}

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")

	sendEvent := func(payload map[string]interface{}) {
		b, _ := json.Marshal(payload)
		fmt.Fprintf(c.Response(), "data: %s\n\n", b)
		c.Response().Flush()
	}

	err = h.chatUC.BroadcastMessage(c.Request().Context(), client, contacts, message, func(result usecase.BroadcastResult, sent, total int) {
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

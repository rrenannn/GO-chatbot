package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rrenannn/GO-chatbot/internal/usecase"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

type WhatsappHandler struct {
	client  *whatsmeow.Client
	usecase usecase.ChatUseCase
}

func NewWhatsAppHandler(client *whatsmeow.Client, usecase usecase.ChatUseCase) *WhatsappHandler {
	handler := &WhatsappHandler{client: client, usecase: usecase}

	client.AddEventHandler(handler.EventHandler)
	fmt.Println("✅ Listener do WhatsApp registrado com sucesso!")
	return handler
}

func (h *WhatsappHandler) EventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if time.Since(v.Info.Timestamp) > 2*time.Minute {
			fmt.Println("⏳ Ignorando mensagem antiga de:", v.Info.Sender.User)
			return
		}
		// Evita responder a si mesmo ou a mensagens de status
		if v.Info.IsFromMe || v.Info.IsGroup {
			return
		}

		// Extrai o texto da mensagem
		msgText := ""
		if v.Message.GetConversation() != "" {
			msgText = v.Message.GetConversation()
		} else if v.Message.ExtendedTextMessage != nil {
			msgText = v.Message.ExtendedTextMessage.GetText()
		}

		if strings.TrimSpace(msgText) != "" {
			fmt.Println("📩 MENSAGEM RECEBIDA -> De:", v.Info.Sender.User, "| Texto:", msgText)

			// Chama o caso de uso em uma Goroutine separada para não travar o listener do WhatsApp
			go func() {
				err := h.usecase.ProcessIncomingMessage(context.Background(), v.Info.Sender, msgText)
				if err != nil {
					fmt.Println("🚨 Erro no UseCase:", err)
				}
			}()
		}

	case *events.Disconnected:
		fmt.Println("Whatsmeow desconectado. O AutoReconnect está ativado, tentando reestabelecer...")

	case *events.LoggedOut:
		fmt.Println("Atenção: O aparelho foi deslogado (QR Code revogado).")
		// Aqui você dispararia um alerta para o Sentry/Slack para escanear de novo
	}
}

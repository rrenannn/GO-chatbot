package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

type WhatsappHandler struct {
	client *whatsmeow.Client
	// usecase usecase.ChatUseCase
}

func NewWhatsappHandler(client *whatsmeow.Client) *WhatsappHandler {
	handler := &WhatsappHandler{client: client}
	client.AddEventHandler(handler.EventHandler)
	return handler
}

func (h *WhatsappHandler) EventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
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

		if msgText != "" {
			// Chama a regra de negócio (Onde a mágica do banco + IA acontece)
			err := h.usecase.ProcessIncomingMessage(context.Background(), v.Info.Sender.User, msgText)
			if err != nil {
				fmt.Println("Erro ao processar mensagem:", err)
			}
		}

	case *events.Disconnected:
		fmt.Println("Whatsmeow desconectado. O AutoReconnect está ativado, tentando reestabelecer...")

	case *events.LoggedOut:
		fmt.Println("Atenção: O aparelho foi deslogado (QR Code revogado).")
		// Aqui você dispararia um alerta para o Sentry/Slack para escanear de novo
	}
}

package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rrenannn/GO-chatbot/internal/usecase"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// WhatsappHandler processa eventos recebidos de QUALQUER cliente whatsmeow
// gerenciado (um por usuário logado). É usado como callback do
// pkg/whatsapp.Manager, que informa de qual usuário (ownerID) veio o evento.
type WhatsappHandler struct {
	usecase usecase.ChatUseCase
}

func NewWhatsAppHandler(usecase usecase.ChatUseCase) *WhatsappHandler {
	return &WhatsappHandler{usecase: usecase}
}

func (h *WhatsappHandler) HandleEvent(ownerID uuid.UUID, client *whatsmeow.Client, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if v.Message == nil {
			return
		}

		if time.Since(v.Info.Timestamp) > 2*time.Minute {
			fmt.Println("⏳ Ignorando mensagem antiga de:", v.Info.Sender.User)
			return
		}

		// Evita responder a si mesmo ou a mensagens de status
		if v.Info.IsFromMe || v.Info.IsGroup {
			return
		}

		msgText := ""
		if v.Message.GetConversation() != "" {
			msgText = v.Message.GetConversation()
		} else if v.Message.ExtendedTextMessage != nil {
			msgText = v.Message.ExtendedTextMessage.GetText()
		}

		if strings.TrimSpace(msgText) != "" {
			fmt.Println("📩 MENSAGEM RECEBIDA -> De:", v.Info.Sender.User, "| Texto:", msgText)

			// Roda em goroutine separada para não travar o listener do WhatsApp
			go func() {
				err := h.usecase.ProcessIncomingMessage(context.Background(), client, ownerID, v.Info.Sender, msgText)
				if err != nil {
					fmt.Println("🚨 Erro no UseCase:", err)
				}
			}()
		}

	case *events.Disconnected:
		fmt.Println("Whatsmeow desconectado (usuário", ownerID, "). O AutoReconnect está ativado, tentando reestabelecer...")

	case *events.LoggedOut:
		fmt.Println("Atenção: o aparelho do usuário", ownerID, "foi deslogado (QR Code revogado).")
	}
}

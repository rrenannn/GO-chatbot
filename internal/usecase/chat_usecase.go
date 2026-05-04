package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	db "github.com/rrenannn/GO-chatbot/db/sqlc"
	"github.com/rrenannn/GO-chatbot/internal/repository"
	"github.com/rrenannn/GO-chatbot/pkg/llm"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type ChatUseCase interface {
	TriggerPostSale(ctx context.Context, phone string, customerID string) error
	ProcessIncomingMessage(ctx context.Context, sender types.JID, messageText string) error
}

type chatUseCase struct {
	repo     repository.ChatRepository
	client   *whatsmeow.Client
	aiClient llm.AIClient
}

func NewChatUseCase(repo repository.ChatRepository, waClient *whatsmeow.Client, aiClient llm.AIClient) ChatUseCase {
	return &chatUseCase{
		repo:     repo,
		client:   waClient,
		aiClient: aiClient,
	}
}

func (uc *chatUseCase) TriggerPostSale(ctx context.Context, phone string, customerID string) error {
	// Cria a sessão no banco com status WAITING_USER_REPLY
	// Nota: Em produção, faça parse correto do customerID para UUID
	session, err := uc.repo.CreateSession(ctx, parseUUID(customerID))
	if err != nil {
		return err
	}

	msgContent := "Olá! Vimos que seu pedido chegou. Está tudo certo com ele ou houve alguma avaria?"

	// Salva a mensagem do bot no histórico
	_, err = uc.repo.InsertMessage(ctx, session.ID, "BOT", msgContent)
	if err != nil {
		return err
	}

	// Monta o JID (Identificador do WhatsApp)
	targetJID := types.NewJID(phone, types.DefaultUserServer)

	// Dispara via Whatsmeow
	_, err = uc.client.SendMessage(ctx, targetJID, &waProto.Message{
		Conversation: proto.String(msgContent),
	})

	return err
}

func (uc *chatUseCase) ProcessIncomingMessage(ctx context.Context, sender types.JID, messageText string) error {
	phone := sender.User

	// 1. Busca a sessão ativa
	session, err := uc.repo.GetActiveSessionByPhone(ctx, phone)
	if err != nil {
		// Se não tem sessão ativa, o bot ignora a mensagem (não é pós-venda)
		return nil
	}

	// 2. Se um humano já assumiu, o bot fica calado
	if session.Status.SessionStatus == db.SessionStatusHUMANHANDLING {
		return nil
	}

	// 3. Salva a resposta do cliente no banco
	_, err = uc.repo.InsertMessage(ctx, session.ID, "USER", messageText)
	if err != nil {
		return fmt.Errorf("erro ao salvar msg do usuario: %w", err)
	}

	// 4. Se estava aguardando, muda o status para atendimento com IA
	if session.Status.SessionStatus == db.SessionStatusWAITINGUSERREPLY {
		err = uc.repo.UpdateSessionStatus(ctx, session.ID, db.SessionStatusAIHANDLING)
		if err != nil {
			return err
		}
	}

	// 5. Busca o histórico para dar contexto à IA
	history, err := uc.repo.GetSessionMessages(ctx, session.ID)
	if err != nil {
		return err
	}

	systemPrompt := `Você é o assistente de pós-venda. 
Sua função é descobrir se o produto chegou bem. 
Seja amigável e curto.
REGRAS:
1. Se o cliente relatar avaria, peça desculpas e diga que vai transferir para um humano. Retorne EXATAMENTE e APENAS a string: [TRANSBORDO_NECESSARIO]
2. Se o cliente tiver uma dúvida complexa ou estiver irritado, retorne EXATAMENTE: [TRANSBORDO_NECESSARIO]
3. Se o cliente disser que chegou tudo bem, agradeça e encerre o assunto.`

	// 6. Envia para a IA (Pseudo-código)
	// aiResponse := uc.aiClient.AskWithContext(history, systemPrompt)
	aiResponse, err := uc.aiClient.AskWithContext(ctx, history, systemPrompt)
	if err != nil {
		// Tratar erro (ex: enviar mensagem padrão de "estamos com instabilidade")
		fmt.Errorf("erro ao encaminhar para o gemini :%w", err)
		_, errS := uc.client.SendMessage(ctx, sender, &waProto.Message{
			Conversation: proto.String("Estamos com instabilidade, aguarde um momento e tente novamente."),
		})
		if errS != nil {
			fmt.Errorf("erro ao enviar mensagem: %v", err)
		}
		return err
	}

	// 7. Avalia a resposta da IA (Regra de Transbordo)
	if strings.Contains(aiResponse, "[TRANSBORDO_NECESSARIO]") {
		// Pausa o bot
		err := uc.repo.UpdateSessionStatus(ctx, session.ID, db.SessionStatusHUMANHANDLING)
		if err != nil {
			fmt.Errorf("erro ao atualizar status da sessao: %w", err)
		}

		handoffMsg := "Vou transferir você para um de nossos especialistas. Só um instante!"

		uc.repo.InsertMessage(ctx, session.ID, "BOT", handoffMsg)
		uc.client.SendMessage(ctx, sender, &waProto.Message{
			Conversation: proto.String(handoffMsg),
		})

		// Aqui você chamaria um serviço de notificação (WebSocket para seu front, Slack, etc.)
		return nil
	}

	// 8. Se a IA resolveu continuar a conversa, envia a resposta dela
	_, errI := uc.repo.InsertMessage(ctx, session.ID, "BOT", aiResponse)
	if errI != nil {
		fmt.Printf("erro ao salvar mensagem no banco: %v", err)
	}

	_, errS := uc.client.SendMessage(ctx, sender, &waProto.Message{
		Conversation: proto.String(aiResponse),
	})
	if errS != nil {
		fmt.Errorf("erro ao enviar mensagem: %v", err)
	}

	return err
}

// Função auxiliar para parse de UUID (omitida para brevidade)
func parseUUID(id string) uuid.UUID {
	parsed, err := uuid.Parse(id)
	if err != nil {
		log.Fatalf("UUID inválido: %v", err)
	}
	return parsed
}

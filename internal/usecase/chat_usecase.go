package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	db "github.com/rrenannn/GO-chatbot/db/sqlc"
	"github.com/rrenannn/GO-chatbot/internal/repository"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

const (
	// Limites de segurança para reduzir risco de banimento por spam
	MaxBroadcastRecipients = 200
	MinDelaySeconds        = 5
	MaxDelaySeconds        = 15
)

type BroadcastContact struct {
	Phone string
	Name  string
}

type BroadcastResult struct {
	Phone   string `json:"phone"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type ChatUseCase interface {
	TriggerPostSale(ctx context.Context, phone string, customerID string) error
	ProcessIncomingMessage(ctx context.Context, sender types.JID, messageText string) error
	BroadcastMessage(ctx context.Context, contacts []BroadcastContact, messageTemplate string, onProgress func(BroadcastResult, int, int)) error
}

// renderMessage substitui a variável {{nome}} pelo nome do contato (ou remove
// o placeholder de forma limpa quando o contato não tem nome cadastrado).
func renderMessage(template string, name string) string {
	if name == "" {
		return strings.TrimSpace(strings.ReplaceAll(template, "{{nome}}", ""))
	}
	return strings.ReplaceAll(template, "{{nome}}", name)
}

type chatUseCase struct {
	repo   repository.ChatRepository
	client *whatsmeow.Client
}

func NewChatUseCase(repo repository.ChatRepository, waClient *whatsmeow.Client) ChatUseCase {
	return &chatUseCase{
		repo:   repo,
		client: waClient,
	}
}

func (uc *chatUseCase) TriggerPostSale(ctx context.Context, phone string, customerID string) error {
	fmt.Println("🚀 Iniciando disparo ativo para:", phone)

	customer, err := uc.repo.GetCustomerByPhone(ctx, phone)
	if err != nil {
		fmt.Println("👤 Novo cliente (Ativo)! Cadastrando...")
		customer, err = uc.repo.CreateCustomer(ctx, phone, "Cliente Pós-Venda")
		if err != nil {
			return fmt.Errorf("erro ao criar cliente no disparo: %w", err)
		}
	}

	session, err := uc.repo.CreateSession(ctx, customer.ID)
	if err != nil {
		return err
	}

	msgContent := "Olá 💖\nSeja bem-vindo(a) ao atendimento da Gocase!\nMe chamo Gabi e vou te ajudar da forma mais rápida possível 😊\nComo posso te ajudar hoje? "

	_, err = uc.repo.InsertMessage(ctx, session.ID, "BOT", msgContent)
	if err != nil {
		return err
	}

	targetJID := types.NewJID(phone, types.DefaultUserServer)
	_, err = uc.client.SendMessage(ctx, targetJID, &waProto.Message{
		Conversation: proto.String(msgContent),
	})

	return err
}

func (uc *chatUseCase) ProcessIncomingMessage(ctx context.Context, sender types.JID, messageText string) error {
	rawPhone := sender.User

	searchPhone := rawPhone
	if rawPhone == "93583361220718" {
		searchPhone = "5511945097706"
	}

	fmt.Println("📩 Mensagem recebida! JID RAW:", rawPhone, "| Texto:", messageText)

	if len(rawPhone) >= 4 && rawPhone[:4] == "5511" && len(rawPhone) == 12 {
		searchPhone = "55119" + rawPhone[4:]
		fmt.Println("🔄 Número formatado (DDD 11 compensado):", searchPhone)
	}

	customer, err := uc.repo.GetCustomerByPhone(ctx, searchPhone)
	if err != nil {
		fmt.Println("👤 Novo cliente detectado! Cadastrando automaticamente...")
		customer, err = uc.repo.CreateCustomer(ctx, searchPhone, "Cliente WhatsApp")
		if err != nil {
			return fmt.Errorf("erro ao criar cliente: %w", err)
		}
	}

	session, err := uc.repo.GetActiveSessionByPhone(ctx, searchPhone)
	if err != nil {
		fmt.Println("🆕 Nenhuma sessão ativa. Criando nova conversa...")
		session, err = uc.repo.CreateSession(ctx, customer.ID)
		if err != nil {
			return fmt.Errorf("erro ao criar sessao: %w", err)
		}

		uc.repo.UpdateSessionStatus(ctx, session.ID, db.SessionStatusHUMANHANDLING)
	}

	_, err = uc.repo.InsertMessage(ctx, session.ID, "USER", messageText)
	if err != nil {
		return fmt.Errorf("erro ao salvar msg do usuario: %w", err)
	}

	// Sem IA: as mensagens recebidas são apenas registradas para atendimento humano.
	return nil
}

// BroadcastMessage envia a mesma mensagem para uma lista de números com um
// intervalo aleatório entre cada envio, reduzindo o risco de banimento por spam.
// onProgress é chamado após cada tentativa de envio (pode ser nil).
func (uc *chatUseCase) BroadcastMessage(ctx context.Context, contacts []BroadcastContact, messageTemplate string, onProgress func(BroadcastResult, int, int)) error {
	if len(contacts) == 0 {
		return fmt.Errorf("nenhum número informado")
	}
	if len(contacts) > MaxBroadcastRecipients {
		return fmt.Errorf("limite de %d destinatários por disparo excedido", MaxBroadcastRecipients)
	}

	total := len(contacts)
	for i, contact := range contacts {
		result := BroadcastResult{Phone: contact.Phone}

		targetJID := types.NewJID(contact.Phone, types.DefaultUserServer)
		_, err := uc.client.SendMessage(ctx, targetJID, &waProto.Message{
			Conversation: proto.String(renderMessage(messageTemplate, contact.Name)),
		})

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			fmt.Println("🚨 Erro ao enviar disparo em massa para", contact.Phone, ":", err)
		} else {
			result.Success = true
		}

		if onProgress != nil {
			onProgress(result, i+1, total)
		}

		// Aguarda um intervalo aleatório antes do próximo envio (menos entre o último e o fim)
		if i < total-1 {
			delay := MinDelaySeconds + rand.Intn(MaxDelaySeconds-MinDelaySeconds+1)
			select {
			case <-time.After(time.Duration(delay) * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

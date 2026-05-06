package usecase

import (
	"context"
	"fmt"
	"strings"

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

	if strings.HasPrefix(rawPhone, "5511") && len(rawPhone) == 12 {
		searchPhone = "55119" + rawPhone[4:]
		fmt.Println("🔄 Número formatado (DDD 11 compensado):", searchPhone)
	}

	customer, err := uc.repo.GetCustomerByPhone(ctx, searchPhone)
	if err != nil {
		fmt.Println("👤 Novo cliente detectado! Cadastrando automaticamente...")
		// Se não achou, cria o cliente com um nome genérico
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

		uc.repo.UpdateSessionStatus(ctx, session.ID, db.SessionStatusAIHANDLING)
	}

	if session.Status.SessionStatus == db.SessionStatusHUMANHANDLING {
		return nil
	}

	_, err = uc.repo.InsertMessage(ctx, session.ID, "USER", messageText)
	if err != nil {
		return fmt.Errorf("erro ao salvar msg do usuario: %w", err)
	}

	if session.Status.SessionStatus == db.SessionStatusWAITINGUSERREPLY {
		err = uc.repo.UpdateSessionStatus(ctx, session.ID, db.SessionStatusAIHANDLING)
		if err != nil {
			fmt.Println("erro ao salvar msg do usuario:", err)
			return err
		}
	}

	fmt.Println("📚 4. Buscando histórico da conversa...")
	history, err := uc.repo.GetSessionMessages(ctx, session.ID)
	if err != nil {
		fmt.Println("🚨 ERRO ao buscar histórico:", err)
		return err
	}

	systemPrompt := `Você é a Gabi, assistente virtual de atendimento da Gocase.
Seja muito amigável, demonstre empatia e use emojis.

FLUXO DE ATENDIMENTO OBRIGATÓRIO (SIGA À RISCA):
PASSO 1: O cliente relata um problema. Você DEVE pedir o "Número do pedido" e o "E-mail cadastrado". Não aplique a solução ainda.
PASSO 2: O cliente fornece os dados (Pedido e E-mail). Você DEVE obrigatoriamente ler o histórico da conversa para lembrar qual foi o problema relatado no PASSO 1, e então aplicar a SOLUÇÃO correspondente da lista abaixo. NUNCA pergunte qual é o problema novamente se ele já informou antes!

SOLUÇÕES PARA OS 10 PROBLEMAS (Aplique no PASSO 2):
1. Pedido atrasado: Diga que o pedido já foi enviado e está em rota, mas houve um pequeno atraso logístico na transportadora. Dê a nova previsão: 08/05.
2. Pedido marcado como entregue mas não recebido: Peça para ele verificar se vizinhos ou a portaria receberam. Se não acharem, diga que abrirá uma solicitação com a transportadora (prazo de 3 dias úteis).
3. Endereço errado: Se o pedido ainda não foi enviado, diga que atualizou com sucesso. Se já foi enviado, diga que não é possível alterar diretamente e inicie o transbordo.
4. Código de rastreio inválido: Diga que o pedido foi enviado recentemente e o código leva algumas horas para atualizar no sistema da transportadora.
5. Produto sem personalização: Peça desculpas pelo erro na produção. Diga que o time analisará e enviará orientações em 24h úteis.
6. Capinha no modelo errado: Peça desculpas pela divergência. Diga que o caso foi pra análise prioritária e o time entrará em contato em 24h úteis.
7. Garrafa riscada: Peça desculpas. Diga que será tratado como avaria e resolvido prioritariamente em 24h úteis.
8. Cliente perdeu prazo de troca: Diga que o prazo oficial encerrou, mas que você registrará o caso para uma análise especial da equipe.
9. Problema na recuperação de senha: Diga que você enviou um novo link e peça para ele verificar a caixa de spam/lixo eletrônico.
10. Troca recusada sem explicação: Peça desculpas pela falta de clareza. Diga que a troca foi recusada com base nas políticas, mas que você solicitará uma reanálise.

REGRA DE TRANSBORDO (HUMANO):
Se o cliente ficar muito irritado, usar palavrões, ou disser que a sua solução não resolveu (ex: "isso é um absurdo", "precisava pra amanhã", "vou ficar no prejuízo", "ninguém resolve"), você DEVE parar de tentar resolver e retornar EXATAMENTE e APENAS a string: [TRANSBORDO_NECESSARIO].`

	aiResponse, err := uc.aiClient.AskWithContext(ctx, history, systemPrompt)
	if err != nil {
		fmt.Println("🚨 ERRO NA IA:", err)
		_, errS := uc.client.SendMessage(ctx, sender.ToNonAD(), &waProto.Message{
			Conversation: proto.String("Estamos com instabilidade, aguarde um momento e tente novamente."),
		})
		if errS != nil {
			fmt.Printf("erro ao enviar mensagem: %v", err)
		}
		return err
	}

	if strings.Contains(aiResponse, "[TRANSBORDO_NECESSARIO]") {
		// Pausa o bot
		err := uc.repo.UpdateSessionStatus(ctx, session.ID, db.SessionStatusHUMANHANDLING)
		if err != nil {
			fmt.Printf("erro ao atualizar status da sessao: %s", err)
		}

		handoffMsg := "Vou transferir você para um de nossos especialistas. Só um instante!"

		uc.repo.InsertMessage(ctx, session.ID, "BOT", handoffMsg)
		uc.client.SendMessage(ctx, sender.ToNonAD(), &waProto.Message{
			Conversation: proto.String(handoffMsg),
		})

		return nil
	}

	_, errI := uc.repo.InsertMessage(ctx, session.ID, "BOT", aiResponse)
	if errI != nil {
		fmt.Printf("erro ao salvar mensagem no banco: %v", err)
	}

	_, errS := uc.client.SendMessage(ctx, sender.ToNonAD(), &waProto.Message{
		Conversation: proto.String(aiResponse),
	})
	if errS != nil {
		fmt.Println("🚨 Erro ao encaminhar para o gemini:", errS)
	}

	return err
}

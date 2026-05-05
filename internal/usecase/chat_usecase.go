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

	msgContent := "Olá 💖\nSeja bem-vindo(a) ao atendimento da Gocase!\nMe chamo Gabi e vou te ajudar da forma mais rápida possível 😊\nComo posso te ajudar hoje? "

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
	rawPhone := sender.User

	// Se for um grupo, ignora
	//if sender.Server != types.DefaultUserServer {
	//	return nil
	//}

	searchPhone := rawPhone
	if rawPhone == "93583361220718" {
		searchPhone = "5511945097706"
	}

	fmt.Println("📩 Mensagem recebida! JID RAW:", rawPhone, "| Texto:", messageText)

	if strings.HasPrefix(rawPhone, "5511") && len(rawPhone) == 12 {
		// Reconstrói o número colocando o 9 de volta (5511 + 9 + resto do numero)
		searchPhone = "55119" + rawPhone[4:]
		fmt.Println("🔄 Número formatado (DDD 11 compensado):", searchPhone)
	}

	// 1. Busca a sessão ativa
	session, err := uc.repo.GetActiveSessionByPhone(ctx, searchPhone)
	if err != nil {
		fmt.Println("⚠️ Sessão não encontrada para o número:", searchPhone, "Ignorando mensagem.")
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
			fmt.Println("erro ao salvar msg do usuario:", err)
			return err
		}
	}

	// 5. Busca o histórico para dar contexto à IA
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

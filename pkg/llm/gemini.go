package llm

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	db "github.com/rrenannn/GO-chatbot/db/sqlc"
	"google.golang.org/api/option"
)

type AIClient interface {
	AskWithContext(ctx context.Context, history []db.MessageHistory, systemPrompt string) (string, error)
}

type geminiClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewGeminiClient(ctx context.Context, apiKey string) (AIClient, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	// flash é perfeito para chatbots: rápido e barato
	model := client.GenerativeModel("gemini-2.5-flash")

	return &geminiClient{
		client: client,
		model:  model,
	}, nil
}

func (g *geminiClient) AskWithContext(ctx context.Context, history []db.MessageHistory, systemPrompt string) (string, error) {
	g.model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	chat := g.model.StartChat()

	// 3. Preenche o histórico da conversa a partir do banco de dados
	// Ignoramos a última mensagem (que é a atual), pois a enviaremos no final
	var chatHistory []*genai.Content
	for i := 0; i < len(history)-1; i++ {
		msg := history[i]
		role := "user"
		if msg.SenderType.String == "BOT" {
			role = "model" // O Gemini usa 'model' para si mesmo e 'user' para o humano
		}

		chatHistory = append(chatHistory, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(msg.Content)},
		})
	}
	chat.History = chatHistory

	// 4. Pega a mensagem atual (a última que o usuário mandou)
	lastMessage := history[len(history)-1].Content

	// 5. Envia para o Gemini
	resp, err := chat.SendMessage(ctx, genai.Text(lastMessage))
	if err != nil {
		return "", fmt.Errorf("erro ao chamar api do gemini: %w", err)
	}

	// 6. Extrai o texto da resposta
	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			responseText += string(text)
		}
	}

	return responseText, nil
}

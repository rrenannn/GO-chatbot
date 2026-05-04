package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/rrenannn/GO-chatbot/internal/repository"
)

func StartSessionCleaner(repo repository.RepositoryInterface) {
	ticker := time.NewTicker(1 * time.Hour) // Roda a cada 1 hora
	go func() {
		for range ticker.C {
			ctx := context.Background()

			err := repo.CleanSessions(ctx)
			if err != nil {
				fmt.Println("Erro ao limpar sessões antigas:", err)
			} else {
				fmt.Println("Worker: Limpeza de sessões inativas concluída.")
			}
		}
	}()
}

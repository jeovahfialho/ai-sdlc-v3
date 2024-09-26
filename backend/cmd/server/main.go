package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"backend-ai-sdlc/internal/api"
	"backend-ai-sdlc/internal/claude"
	"backend-ai-sdlc/internal/storage"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erro ao carregar o arquivo .env")
	}

	// Inicializa o armazenamento
	store := storage.NewMemoryStorage()

	// Pega a chave API do ambiente
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		log.Fatal("CLAUDE_API_KEY não está definida")
	}

	// Inicializa o cliente Claude
	claudeClient := claude.NewClient(apiKey)

	// Configura o CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Porta correta do frontend
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "X-Requested-With"},
		AllowCredentials: true, // Permitir credenciais, se necessário
		Debug:            true,
	})

	// Configura os handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/chat", api.NewChatHandler(store, claudeClient))
	mux.HandleFunc("/messages", api.GetMessagesHandler(store))
	mux.HandleFunc("/readFile", api.ReadFileContentHandler)
	mux.HandleFunc("/downloadProject", api.DownloadProjectHandler) // Nova rota

	// Aplica o middleware CORS
	handler := c.Handler(mux)

	// Inicia o servidor
	port := ":8080"
	fmt.Printf("Server running on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, handler))
}

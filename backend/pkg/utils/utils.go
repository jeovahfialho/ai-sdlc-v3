package utils

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"backend-ai-sdlc/internal/models"
)

const frontendURL = "http://localhost:58195/events"

func EnhancePrompt(prompt string) string {
	// Implemente a lógica para melhorar o prompt aqui
	return prompt + " (Enhanced)"
}

func ProcessResponse(response string) string {
	// Implemente a lógica para processar a resposta aqui
	return response + " (Processed)"
}

func SendToFrontend(response models.ChatResponse) {
	jsonResponse, _ := json.Marshal(response)
	resp, err := http.Post(frontendURL, "application/json", bytes.NewBuffer(jsonResponse))
	if err != nil {
		log.Printf("Error sending to frontend: %v", err)
	}
	defer resp.Body.Close()
}

package models

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GetMessagesResponse struct {
	ConversationID string    `json:"conversation_id"`
	Messages       []Message `json:"messages"`
}

type Conversation struct {
	ID             string
	Steps          []Step
	ProjectCreated bool
}

type Step struct {
	Number   int    `json:"number"`
	Input    string `json:"input"`
	Response string `json:"response"`
}

type ChatRequest struct {
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message"`
	IsConfirmation bool   `json:"is_confirmation"`
}

type ChatResponse struct {
	ConversationID       string `json:"conversation_id"`
	Message              string `json:"message"`
	StepNumber           int    `json:"step_number"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
}

// FrontendResponse pode ser o mesmo que ChatResponse se a estrutura for idêntica
type FrontendResponse ChatResponse

// Novas estruturas para suportar as funcionalidades de WebSocket e conteúdo de arquivos

type WebSocketMessage struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ProjectStructure struct {
	Structure map[string]interface{} `json:"structure"`
}

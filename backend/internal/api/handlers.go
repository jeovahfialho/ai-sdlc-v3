package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"backend-ai-sdlc/internal/claude"
	"backend-ai-sdlc/internal/models"
	"backend-ai-sdlc/internal/storage"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Permite todas as origens. Ajuste conforme necessário para produção.
	},
}

var confirmationSteps = map[int]bool{
	1:  true,
	2:  true,
	7:  true,
	8:  true,
	13: true,
}

type WebSocketMessage struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

type Progress struct {
	Percentage int    `json:"percentage"`
	Message    string `json:"message"`
}

// Estrutura para a resposta com o conteúdo do arquivo
type FileContentResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func DownloadProjectHandler(w http.ResponseWriter, r *http.Request) {
	projectName := r.URL.Query().Get("project")
	if projectName == "" {
		http.Error(w, "Missing project name", http.StatusBadRequest)
		log.Println("Error: Missing project name")
		return
	}

	// Ajuste o diretório base conforme necessário
	baseDir := "/Users/jeovahsimoes/Documents/vtkl/ai-sdlc-v3/backend/cmd/server"
	projectPath := filepath.Join(baseDir, projectName)
	log.Printf("Project path: %s", projectPath)

	// Verifica se o diretório do projeto existe
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		http.Error(w, "Project not found", http.StatusNotFound)
		log.Printf("Error: Project not found at path %s", projectPath)
		return
	}

	// Crie um buffer para armazenar o arquivo ZIP
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Função para adicionar arquivos ao ZIP
	addFiles := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error walking through the directory: %v", err)
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(projectPath, path)
		if err != nil {
			log.Printf("Error getting relative path: %v", err)
			return err
		}
		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			log.Printf("Error creating zip file entry: %v", err)
			return err
		}
		fsFile, err := os.Open(path)
		if err != nil {
			log.Printf("Error opening file: %v", err)
			return err
		}
		defer fsFile.Close()
		_, err = io.Copy(zipFile, fsFile)
		if err != nil {
			log.Printf("Error copying file content to zip: %v", err)
		}
		return err
	}

	// Percorra o diretório do projeto e adicione os arquivos ao ZIP
	err := filepath.Walk(projectPath, addFiles)
	if err != nil {
		http.Error(w, "Error creating ZIP file", http.StatusInternalServerError)
		log.Printf("Error walking the project directory: %v", err)
		return
	}

	// Fecha o zipWriter para finalizar o arquivo ZIP
	err = zipWriter.Close()
	if err != nil {
		http.Error(w, "Error finalizing ZIP file", http.StatusInternalServerError)
		log.Printf("Error closing zipWriter: %v", err)
		return
	}

	// Configure os cabeçalhos para download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", projectName))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

	// Envie o arquivo ZIP
	if _, err := buf.WriteTo(w); err != nil {
		http.Error(w, "Error sending ZIP file", http.StatusInternalServerError)
		log.Printf("Error writing buffer to response: %v", err)
		return
	}

	log.Printf("Successfully sent the ZIP file for project: %s", projectName)
}

// Função para ler o conteúdo do arquivo no disco
func ReadFileContentHandler(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	projectName := r.URL.Query().Get("project")

	log.Printf("Received request for file: %s in project: %s", filePath, projectName)

	if filePath == "" || projectName == "" {
		log.Printf("Missing file path or project name")
		http.Error(w, "Missing file path or project name", http.StatusBadRequest)
		return
	}

	// Ajuste o diretório base conforme necessário
	baseDir := "/Users/jeovahsimoes/Documents/vtkl/ai-sdlc-v3/backend/cmd/server"

	// Construa o caminho completo do arquivo
	fullPath := filepath.Join(baseDir, projectName, filePath)

	log.Printf("Attempting to read file from: %s", fullPath)

	// Verifique se o arquivo existe
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		log.Printf("File does not exist: %s", fullPath)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Printf("Error reading file: %v", err)
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	log.Printf("File read successfully: %s", fullPath)

	response := FileContentResponse{
		Path:    filePath,
		Content: string(content),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetMessagesHandler(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		conversationID := r.URL.Query().Get("conversation_id")
		if conversationID == "" {
			http.Error(w, "Missing conversation_id", http.StatusBadRequest)
			return
		}

		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

		conversation, exists := store.GetConversation(conversationID)
		if !exists {
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}

		messages := getMessagesFromSteps(conversation.Steps, limit)

		response := models.GetMessagesResponse{
			ConversationID: conversationID,
			Messages:       messages,
		}

		sendJSONResponse(w, response)
	}
}

func NewChatHandler(store storage.Storage, claudeClient *claude.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error upgrading to WebSocket: %v", err)
			return
		}
		defer conn.Close()

		for {
			var chatReq models.ChatRequest
			err := conn.ReadJSON(&chatReq)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}

			log.Printf("Received chat request: %+v", chatReq)

			conv, exists := store.GetOrCreateConversation(chatReq.ConversationID)
			if !exists {
				conv = &models.Conversation{
					ID:    chatReq.ConversationID,
					Steps: []models.Step{},
				}
				store.UpdateConversation(conv)
				log.Printf("Created new conversation with ID: %s", chatReq.ConversationID)
			} else {
				log.Printf("Retrieved existing conversation with ID: %s", chatReq.ConversationID)
			}

			currentStep := len(conv.Steps)
			log.Printf("Current step: %d", currentStep)

			var claudeResponse string
			if chatReq.IsConfirmation {
				claudeResponse, err = handleConfirmation(chatReq.Message, currentStep, conv, claudeClient, store, conn)
			} else {
				claudeResponse, err = processNormalMessage(chatReq.Message, currentStep, conv, claudeClient, conn)
			}

			if err != nil {
				log.Printf("Error processing message: %v", err)
				sendWebSocketError(conn, "Error processing message")
				continue
			}

			currentStep++
			newStep := models.Step{
				Number:   currentStep,
				Input:    chatReq.Message,
				Response: claudeResponse,
			}
			conv.Steps = append(conv.Steps, newStep)

			store.UpdateConversation(conv)
			log.Printf("Updated conversation with ID: %s, new step count: %d", chatReq.ConversationID, len(conv.Steps))

			chatResponse := models.ChatResponse{
				ConversationID:       chatReq.ConversationID,
				Message:              claudeResponse,
				StepNumber:           currentStep,
				RequiresConfirmation: confirmationSteps[currentStep+1],
			}

			log.Printf("Sending response: %+v", chatResponse)
			sendWebSocketMessage(conn, "chat_response", chatResponse)
		}
	}
}

func handleConfirmation(answer string, currentStep int, conv *models.Conversation, claudeClient *claude.Client, store storage.Storage, conn *websocket.Conn) (string, error) {
	log.Printf("Handling confirmation for step %d with answer: %s", currentStep, answer)

	switch currentStep {
	case 1:
		if answer == "YES" {
			log.Println("Confirmation received for step 1. Processing step 2.")
			return processStep2(conv, claudeClient, store, conn)
		}
		return "I understand. Let's revise the JSON structure. What would you like to change?", nil
	case 2:
		if answer == "YES" {
			log.Println("Confirmation received for step 2. Moving to next step.")
			return "Great! The file contents have been generated. Let's move on to the next step.", nil
		}
		return "I see. What part of the generated file contents would you like to modify?", nil
	case 7, 8, 13:
		if answer == "YES" {
			return fmt.Sprintf("Excellent! We'll continue with the next phase. (Step %d completed)", currentStep), nil
		}
		return fmt.Sprintf("No problem. Let's revisit step %d. What needs adjustment?", currentStep), nil
	default:
		log.Printf("Unexpected confirmation for step %d", currentStep)
		return "", fmt.Errorf("unexpected confirmation for step %d", currentStep)
	}
}

func processNormalMessage(message string, currentStep int, conv *models.Conversation, claudeClient *claude.Client, conn *websocket.Conn) (string, error) {
	enhancedPrompt := enhancePrompt(message, currentStep+1)
	response, err := claudeClient.GetResponse(enhancedPrompt)
	if err != nil {
		return "", fmt.Errorf("error getting response from Claude: %v", err)
	}

	// Se estamos no passo 1, enviamos a estrutura JSON para o frontend
	if currentStep == 0 {
		sendWebSocketMessage(conn, "project_structure", response)
	}

	return response, nil
}

func enhancePrompt(originalPrompt string, currentStep int) string {
	currentTime := time.Now().Format(time.RFC3339)

	switch currentStep {
	case 1:
		promptTemplate := `Based on the following description of a project, provide a simplified JSON representation of the project structure.
		Description:
		%s
		Please respond with a JSON object containing the project structure. The backend must be implemented in the specified backend technology (e.g., Go, Node.js, Python), and the frontend must be implemented in the specified frontend technology (e.g., React, Vue, Angular). Include both backend and frontend if applicable, with nested objects representing directories and arrays for files.

		Ensure to include the following files:
		1. Any necessary configuration files for package management (e.g., package.json for the frontend, go.mod for Go in the backend, requirements.txt for Python, etc.), representing them as regular files.
		2. A Dockerfile for both the backend and frontend. The Dockerfile must be suitable for running the backend technology (e.g., Go, Node.js, Python) and the frontend technology (e.g., React, Vue, Angular).
		3. A docker-compose.yml file to set up the project environment, including separate services for the backend and frontend.

		Be sure that the Dockerfile for the backend and frontend properly sets up the runtime environment, installs dependencies, and runs the application according to best practices for the specified technology.

		Keep the structure as simple as possible while accurately representing the project. Provide only the JSON object, with no additional text or explanations.`

		// Format the template with the original prompt
		return fmt.Sprintf(promptTemplate, originalPrompt)
	case 2:
		return fmt.Sprintf(`Based on the JSON file structure provided in the previous step, generate the content for all files in the structure. Follow these rules:

		1. Use the project name and structure from the JSON as context.
		2. Generate appropriate content for each file, including code, configuration, or documentation as needed.
		3. For code files, provide complete, functional code that follows best practices for the respective language or framework.
		4. For configuration files, include realistic and relevant settings.
		5. For documentation files like README.md, provide comprehensive information about the project structure, setup, and usage.
		6. Use comments in the code to explain complex logic or important details.
		7. Ensure consistency across all files in terms of naming conventions, coding style, and overall architecture.
		8. If there are dependencies between files, ensure they are properly referenced and imported.

		Present each file's content in the following format:

		--- [File Path] ---
		[File Content]

		--- [Next File Path] ---
		[Next File Content]

		...and so on for all files in the structure.

		The JSON structure from the previous step was:

		%s

		Please generate the content for all files based on this structure.`, originalPrompt)
	default:
		return fmt.Sprintf(`Interaction #%d:
		This question was asked at %s.
		- Additional instructions: 
		1. Continue to respond based on previous interactions.
		2. Provide a clear and objective response, and include additional examples if useful.
		
		User question: %s`, currentStep, currentTime, originalPrompt)
	}
}

func generateAndSaveFileContent(claudeClient *claude.Client, appName, filePath string, conn *websocket.Conn, totalFiles int, filesProcessed *int) error {

	prompt := fmt.Sprintf(`Generate the content for the file: %s

	Use the following rules:
	1. Provide complete, functional code that follows best practices for the respective language or framework.
	2. If it's a Dockerfile for the backend, make sure it installs dependencies (e.g., go.mod), compiles the code, and runs the backend service.
	3. If it's a Dockerfile for the frontend, ensure it installs the necessary frontend dependencies (e.g., Node modules), builds the frontend, and serves the application.
	4. For configuration files like go.mod and package.json, ensure they use the project name "chat-app-maker" instead of generic placeholders like "github.com/your_username/your_project".
	5. Ensure consistency in naming conventions, coding style, and architecture across all files.

	Please generate only the content of the file, without any additional explanations or file path indicators.`, filePath)

	response, err := claudeClient.GetResponse(prompt)
	if err != nil {
		return fmt.Errorf("error getting response from Claude for file %s: %v", filePath, err)
	}

	if err := saveFileToDisk(appName, filePath, response); err != nil {
		return fmt.Errorf("error saving file to disk: %v", err)
	}

	fileContent := models.FileContent{
		Path:    filePath,
		Content: response,
	}
	sendWebSocketMessage(conn, "file_content", fileContent)

	*filesProcessed++
	percentage := (*filesProcessed * 100) / totalFiles
	sendProgressUpdate(conn, percentage, fmt.Sprintf("Generating: %s", filePath))

	log.Printf("Sent file content for %s to frontend", filePath)

	return nil
}

func processStep2(conv *models.Conversation, claudeClient *claude.Client, store storage.Storage, conn *websocket.Conn) (string, error) {
	jsonStructure := conv.Steps[0].Response
	var projectStructure map[string]interface{}
	err := json.Unmarshal([]byte(jsonStructure), &projectStructure)
	if err != nil {
		return "", fmt.Errorf("error parsing JSON structure: %v", err)
	}

	appName := "chat-app-maker"

	fileStructure := generateFileList(projectStructure)

	sendWebSocketMessage(conn, "project_structure", fileStructure)
	sendWebSocketMessage(conn, "status_update", "Generating project files...")

	totalFiles := countFiles(fileStructure)
	filesProcessed := 0

	var processFiles func(structure map[string]interface{}, path string) error
	processFiles = func(structure map[string]interface{}, path string) error {
		for key, value := range structure {
			newPath := path + "/" + key

			switch v := value.(type) {
			case []string:
				// Processar lista de arquivos
				for _, file := range v {
					filePath := newPath + "/" + file
					// Se for um diretório, adicione uma barra no final
					if !strings.Contains(file, ".") && file != "Dockerfile" {
						filePath += "/"
					}

					// Se for um diretório, não gere conteúdo
					if strings.HasSuffix(filePath, "/") {
						if err := saveFileToDisk(appName, filePath, ""); err != nil {
							return fmt.Errorf("error saving directory to disk: %v", err)
						}
						filesProcessed++
						if totalFiles > 0 { // Verificação para evitar divisão por zero
							percentage := (filesProcessed * 100) / totalFiles
							sendProgressUpdate(conn, percentage, fmt.Sprintf("Creating directory: %s", filePath))
						}
						continue
					}

					// Geração de conteúdo do arquivo
					err := generateAndSaveFileContent(claudeClient, appName, filePath, conn, totalFiles, &filesProcessed)
					if err != nil {
						return err
					}
				}
			case map[string]interface{}:
				// Caso o map esteja vazio, tratar como arquivo
				if len(v) == 0 && (strings.Contains(key, ".") || key == "Dockerfile") {
					filePath := newPath
					err := generateAndSaveFileContent(claudeClient, appName, filePath, conn, totalFiles, &filesProcessed)
					if err != nil {
						return err
					}
				} else {
					// Recursão para processar diretórios aninhados
					err := processFiles(v, newPath)
					if err != nil {
						return err
					}
				}
			case []interface{}:
				// Quando o arquivo for do tipo "App.js": [] ou "Dockerfile": [] ou similar, tratar como arquivo regular
				if len(v) == 0 && (strings.Contains(key, ".") || key == "Dockerfile" || key == "go.mod" || key == "main.go" || key == "docker-compose.yml") {
					// Este é um arquivo, mesmo que o array esteja vazio
					filePath := newPath
					err := generateAndSaveFileContent(claudeClient, appName, filePath, conn, totalFiles, &filesProcessed)
					if err != nil {
						return err
					}
				}
			default:
				// Tratamento especial para arquivos com objetos vazios {}
				if key == "Dockerfile" || key == "package.json" || key == "App.js" || key == "index.js" || key == "go.mod" || key == "main.go" || key == "docker-compose.yml" || key == ".gitignore" {
					filePath := newPath
					err := generateAndSaveFileContent(claudeClient, appName, filePath, conn, totalFiles, &filesProcessed)
					if err != nil {
						return err
					}
				} else {
					log.Printf("Unexpected value type for key: %s", key)
				}
			}
		}
		return nil
	}

	sendProgressUpdate(conn, 0, "Starting file generation...")
	err = processFiles(fileStructure, "")
	if err != nil {
		return "", err
	}
	sendProgressUpdate(conn, 100, "File generation complete!")

	newStep := models.Step{
		Number:   2,
		Input:    "Generate file contents",
		Response: "File contents generated and sent to frontend",
	}
	conv.Steps = append(conv.Steps, newStep)
	store.UpdateConversation(conv)

	sendWebSocketMessage(conn, "status_update", "Is this the structure you were expecting? Please confirm with YES or NO.")

	return "Great! I've generated the content for all files based on the JSON structure. The files have been sent to the frontend for display.", nil
}

func generateFileList(structure map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range structure {
		switch v := value.(type) {
		case map[string]interface{}:
			result[key] = generateFileList(v)
		case []interface{}:
			files := make([]string, 0)
			for _, item := range v {
				if file, ok := item.(string); ok {
					files = append(files, file)
				}
			}
			result[key] = files
		default:
			result[key] = v
		}
	}
	return result
}

func sendProgressUpdate(conn *websocket.Conn, percentage int, message string) {
	progress := Progress{Percentage: percentage, Message: message}
	sendWebSocketMessage(conn, "progress_update", progress)
}

func countFiles(structure map[string]interface{}) int {
	count := 0
	for key, value := range structure {
		switch v := value.(type) {
		case []string:
			// Contar cada arquivo no array
			count += len(v)
		case map[string]interface{}:
			// Se for um map vazio, contar como arquivo (ex.: "go.mod": {})
			if len(v) == 0 && (strings.Contains(key, ".") || key == "Dockerfile") {
				count++
			} else {
				// Recursão para processar diretórios aninhados
				count += countFiles(v)
			}
		case []interface{}:
			// Quando for um arquivo representado por array vazio (ex.: "Dockerfile": [])
			if len(v) == 0 && (strings.Contains(key, ".") || key == "Dockerfile" || key == "go.mod" || key == "package.json") {
				count++
			}
		default:
			// Se o arquivo for identificado por uma chave (ex.: "Dockerfile", "go.mod")
			if key == "Dockerfile" || key == "go.mod" || key == "package.json" || key == "App.js" || key == "index.js" || key == "docker-compose.yml" || key == ".gitignore" {
				count++
			}
		}
	}
	return count
}

func getMessagesFromSteps(steps []models.Step, limit int) []models.Message {
	if limit > 0 && limit < len(steps) {
		steps = steps[len(steps)-limit:]
	}

	var messages []models.Message
	for _, step := range steps {
		messages = append(messages,
			models.Message{Role: "user", Content: step.Input},
			models.Message{Role: "assistant", Content: step.Response},
		)
	}
	return messages
}

func sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func sendWebSocketMessage(conn *websocket.Conn, messageType string, content interface{}) {
	message := WebSocketMessage{
		Type:    messageType,
		Content: content,
	}
	if err := conn.WriteJSON(message); err != nil {
		log.Printf("Error sending WebSocket message: %v", err)
	}
}

func sendWebSocketError(conn *websocket.Conn, errorMessage string) {
	sendWebSocketMessage(conn, "error", errorMessage)
}

// Função para salvar o conteúdo do arquivo no sistema de arquivos
func saveFileToDisk(appName, filePath, content string) error {
	fullPath := filepath.Join(appName, filePath)
	dir := filepath.Dir(fullPath)

	// Cria todos os diretórios necessários
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	// Se o caminho terminar com uma barra, é um diretório
	if strings.HasSuffix(filePath, "/") {
		// Não precisa fazer nada, pois o diretório já foi criado
		log.Printf("Directory created: %s", fullPath)
		return nil
	}

	// Se não for um diretório, escreve o conteúdo do arquivo
	if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	log.Printf("File %s saved successfully.", fullPath)
	return nil
}

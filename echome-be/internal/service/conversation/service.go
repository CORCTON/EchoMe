package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
)

// ConversationService implements domain.ConversationService
type ConversationService struct {
	aiService        domain.AIService
	characterService domain.CharacterService
	sessionService   domain.SessionService
	messageRepo      domain.MessageRepository
}

// NewConversationService creates a new conversation service
func NewConversationService(
	aiService domain.AIService,
	characterService domain.CharacterService,
	sessionService domain.SessionService,
	messageRepo domain.MessageRepository,
) *ConversationService {
	return &ConversationService{
		aiService:        aiService,
		characterService: characterService,
		sessionService:   sessionService,
		messageRepo:      messageRepo,
	}
}

// StartVoiceConversation starts a voice conversation session
func (s *ConversationService) StartVoiceConversation(ctx context.Context, req *domain.VoiceConversationRequest) error {
	log.Printf("Starting voice conversation for session %s with character %s", req.SessionID, req.CharacterID)

	// Validate input
	if req.SessionID == uuid.Nil {
		return WrapError(ErrCodeInvalidInput, "Session ID is required", nil)
	}
	if req.CharacterID == uuid.Nil {
		return WrapError(ErrCodeInvalidInput, "Character ID is required", nil)
	}
	if req.WebSocketConn == nil {
		return WrapError(ErrCodeInvalidInput, "WebSocket connection is required", nil)
	}

	// Get character information
	character, err := s.characterService.GetCharacterByID(req.CharacterID)
	if err != nil {
		log.Printf("Failed to get character %s: %v", req.CharacterID, err)
		return WrapError(ErrCodeCharacterNotFound, "Character not found", err)
	}

	// Get voice configuration for the character
	voiceConfig, err := s.GetCharacterVoiceConfig(req.CharacterID)
	if err != nil {
		log.Printf("Failed to get voice config for character %s: %v", req.CharacterID, err)
		return WrapError(ErrCodeConfigurationError, "Failed to get voice configuration", err)
	}

	// Validate session exists
	session, err := s.sessionService.GetSessionByID(req.SessionID)
	if err != nil {
		log.Printf("Failed to get session %s: %v", req.SessionID, err)
		return WrapError(ErrCodeSessionNotFound, "Session not found", err)
	}

	log.Printf("Voice conversation started for session %s, character: %s", session.ID, character.Name)

	// Handle the voice conversation flow
	return s.handleVoiceConversationFlow(ctx, req.WebSocketConn, voiceConfig, session)
}

// handleVoiceConversationFlow manages the complete voice conversation pipeline
func (s *ConversationService) handleVoiceConversationFlow(ctx context.Context, clientWS *websocket.Conn, voiceConfig *domain.VoiceConfig, session *domain.Session) error {
	log.Printf("Starting voice conversation flow for session %s", session.ID)

	// Use WaitGroup to manage goroutines
	var wg sync.WaitGroup
	wg.Add(1)

	// Channel for communication between goroutines
	textChan := make(chan string, 10)
	audioChan := make(chan []byte, 10)
	errorChan := make(chan error, 5)

	// Goroutine to handle the complete voice pipeline
	go func() {
		defer wg.Done()
		defer close(textChan)
		defer close(audioChan)

		for {
			select {
			case <-ctx.Done():
				log.Printf("Context cancelled for session %s", session.ID)
				return
			default:
				// Read message from client
				messageType, data, err := clientWS.ReadMessage()
				if err != nil {
					log.Printf("Error reading from client WebSocket: %v", err)
					errorChan <- err
					return
				}

				switch messageType {
				case websocket.BinaryMessage:
					// Handle audio input - process through ASR -> AI -> TTS pipeline
					err := s.processAudioMessage(ctx, data, voiceConfig, session, clientWS)
					if err != nil {
						log.Printf("Error processing audio message: %v", err)
						errorChan <- err
					}

				case websocket.TextMessage:
					// Handle text input - process through AI -> TTS pipeline
					var textMsg map[string]interface{}
					if err := json.Unmarshal(data, &textMsg); err != nil {
						log.Printf("Error unmarshaling text message: %v", err)
						continue
					}

					if text, ok := textMsg["text"].(string); ok {
						err := s.processTextInput(ctx, text, voiceConfig, session, clientWS)
						if err != nil {
							log.Printf("Error processing text input: %v", err)
							errorChan <- err
						}
					}

				case websocket.CloseMessage:
					log.Printf("Client closed connection for session %s", session.ID)
					return
				}
			}
		}
	}()

	// Wait for completion or error
	select {
	case err := <-errorChan:
		log.Printf("Voice conversation error for session %s: %v", session.ID, err)
		return err
	case <-ctx.Done():
		log.Printf("Voice conversation cancelled for session %s", session.ID)
		return ctx.Err()
	}
}

// processAudioMessage handles audio input through ASR -> AI -> TTS pipeline
func (s *ConversationService) processAudioMessage(ctx context.Context, audioData []byte, voiceConfig *domain.VoiceConfig, session *domain.Session, clientWS *websocket.Conn) error {
	log.Printf("Processing audio message for session %s", session.ID)

	// For now, we'll simulate ASR processing since the current ASR implementation
	// is designed for direct WebSocket handling. In a real implementation,
	// we would need to refactor the ASR to work with byte arrays.

	// TODO: Implement proper ASR processing
	// This is a placeholder - in reality we would:
	// 1. Send audioData to ASR service
	// 2. Get transcribed text
	// 3. Process through AI
	// 4. Convert AI response to speech
	// 5. Send audio back to client

	// For now, send a message indicating audio was received
	response := map[string]interface{}{
		"type":       "audio_received",
		"message":    "Audio received and processing started",
		"session_id": session.ID,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal audio response: %w", err)
	}

	return clientWS.WriteMessage(websocket.TextMessage, responseData)
}

// processTextInput handles text input through AI -> TTS pipeline
func (s *ConversationService) processTextInput(ctx context.Context, text string, voiceConfig *domain.VoiceConfig, session *domain.Session, clientWS *websocket.Conn) error {
	log.Printf("Processing text input for session %s: %s", session.ID, text)

	// Save user message
	userMessage := &domain.Message{
		ID:        uuid.New(),
		SessionID: session.ID,
		Content:   text,
		Sender:    "user",
		Timestamp: time.Now(),
	}

	if err := s.messageRepo.Save(userMessage); err != nil {
		log.Printf("Failed to save user message: %v", err)
		// Continue processing even if save fails
	}

	// Get session history for context
	sessionHistory, err := s.messageRepo.GetBySessionID(session.ID)
	if err != nil {
		log.Printf("Failed to get session history: %v", err)
		sessionHistory = []*domain.Message{} // Use empty history if fetch fails
	}

	// Create AI request with character context
	aiRequest := &domain.AIRequest{
		UserInput:        text,
		CharacterContext: voiceConfig.Character,
		SessionHistory:   sessionHistory,
		Language:         voiceConfig.Language,
	}

	// Generate AI response
	aiResponse, err := s.generateAIResponse(ctx, aiRequest)
	if err != nil {
		log.Printf("Failed to generate AI response: %v", err)
		return s.sendErrorResponse(clientWS, "Failed to generate AI response", session.ID)
	}

	// Save AI message
	aiMessage := &domain.Message{
		ID:        uuid.New(),
		SessionID: session.ID,
		Content:   aiResponse.Text,
		Sender:    "ai",
		Timestamp: time.Now(),
	}

	if err := s.messageRepo.Save(aiMessage); err != nil {
		log.Printf("Failed to save AI message: %v", err)
		// Continue processing even if save fails
	}

	// Send text response to client
	textResponse := map[string]interface{}{
		"type":       "text_response",
		"text":       aiResponse.Text,
		"message_id": aiMessage.ID,
		"session_id": session.ID,
		"timestamp":  aiMessage.Timestamp,
	}

	responseData, err := json.Marshal(textResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal text response: %w", err)
	}

	if err := clientWS.WriteMessage(websocket.TextMessage, responseData); err != nil {
		return fmt.Errorf("failed to send text response: %w", err)
	}

	// TODO: Convert AI response to speech using TTS
	// This would involve:
	// 1. Using the character's voice configuration
	// 2. Calling TTS service with the AI response text
	// 3. Sending the generated audio back to the client

	log.Printf("Text processing completed for session %s", session.ID)
	return nil
}

// generateAIResponse generates AI response using the AI service
func (s *ConversationService) generateAIResponse(ctx context.Context, req *domain.AIRequest) (*domain.AIResponse, error) {
	// Build character context string
	characterContext := ""
	if req.CharacterContext != nil {
		characterContext = fmt.Sprintf("你是%s。%s。%s",
			req.CharacterContext.Name,
			req.CharacterContext.Description,
			req.CharacterContext.Persona)
	}

	// Generate response using existing AI service
	responseText, err := s.aiService.GenerateResponse(ctx, req.UserInput, characterContext)
	if err != nil {
		return nil, fmt.Errorf("AI service error: %w", err)
	}

	return &domain.AIResponse{
		Text: responseText,
		Metadata: map[string]interface{}{
			"character_id": req.CharacterContext.ID,
			"language":     req.Language,
			"timestamp":    time.Now(),
		},
	}, nil
}

// ProcessTextMessage processes a text message and returns AI response
func (s *ConversationService) ProcessTextMessage(ctx context.Context, req *domain.TextMessageRequest) (*domain.TextMessageResponse, error) {
	log.Printf("Processing text message for session %s", req.SessionID)

	// Validate input
	if req.SessionID == uuid.Nil {
		return nil, WrapError(ErrCodeInvalidInput, "Session ID is required", nil)
	}
	if req.CharacterID == uuid.Nil {
		return nil, WrapError(ErrCodeInvalidInput, "Character ID is required", nil)
	}
	if req.UserInput == "" {
		return nil, WrapError(ErrCodeInvalidInput, "User input is required", nil)
	}

	// Get character information
	character, err := s.characterService.GetCharacterByID(req.CharacterID)
	if err != nil {
		return nil, WrapError(ErrCodeCharacterNotFound, "Character not found", err)
	}

	// Get session history for context
	sessionHistory, err := s.messageRepo.GetBySessionID(req.SessionID)
	if err != nil {
		log.Printf("Failed to get session history: %v", err)
		sessionHistory = []*domain.Message{} // Use empty history if fetch fails
	}

	// Create AI request
	aiRequest := &domain.AIRequest{
		UserInput:        req.UserInput,
		CharacterContext: character,
		SessionHistory:   sessionHistory,
		Language:         "zh-CN", // Default language
	}

	// Generate AI response
	aiResponse, err := s.generateAIResponse(ctx, aiRequest)
	if err != nil {
		return nil, WrapError(ErrCodeAIGenerationFailed, "Failed to generate AI response", err)
	}

	// Save user message
	userMessage := &domain.Message{
		ID:        uuid.New(),
		SessionID: req.SessionID,
		Content:   req.UserInput,
		Sender:    "user",
		Timestamp: time.Now(),
	}

	if err := s.messageRepo.Save(userMessage); err != nil {
		log.Printf("Failed to save user message: %v", err)
		// Don't fail the request if message save fails, just log it
	}

	// Save AI message
	aiMessage := &domain.Message{
		ID:        uuid.New(),
		SessionID: req.SessionID,
		Content:   aiResponse.Text,
		Sender:    "ai",
		Timestamp: time.Now(),
	}

	if err := s.messageRepo.Save(aiMessage); err != nil {
		log.Printf("Failed to save AI message: %v", err)
		// Don't fail the request if message save fails, just log it
	}

	log.Printf("Successfully processed text message for session %s, generated response length: %d", req.SessionID, len(aiResponse.Text))

	return &domain.TextMessageResponse{
		Response:  aiResponse.Text,
		MessageID: aiMessage.ID,
		Timestamp: aiMessage.Timestamp,
	}, nil
}

// GetCharacterVoiceConfig retrieves voice configuration for a character
func (s *ConversationService) GetCharacterVoiceConfig(characterID uuid.UUID) (*domain.VoiceConfig, error) {
	character, err := s.characterService.GetCharacterByID(characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character: %w", err)
	}

	// Create default voice configuration if character doesn't have one
	voiceConfig := &domain.VoiceConfig{
		Character: character,
		ASRConfig: domain.ASRConfig{
			Model:      "paraformer-realtime-v2",
			Format:     "pcm",
			SampleRate: 16000,
		},
		TTSConfig: domain.TTSConfig{
			Model:          "qwen-tts-realtime",
			Voice:          "Cherry", // Default voice
			ResponseFormat: "pcm",
			SampleRate:     24000,
			Mode:           "server_commit",
		},
		Language: "zh-CN",
	}

	// Use character's voice configuration if available
	if character.VoiceConfig != nil {
		voiceConfig.TTSConfig.Voice = character.VoiceConfig.Voice
		voiceConfig.TTSConfig.SampleRate = int(character.VoiceConfig.SpeechRate * 24000) // Adjust sample rate based on speech rate
		voiceConfig.Language = character.VoiceConfig.Language

		// Apply custom TTS parameters if available
		if character.VoiceConfig.CustomParams != nil {
			// Here we could apply custom parameters to the TTS config
			// For now, we'll just log them
			log.Printf("Character %s has custom voice params: %v", character.Name, character.VoiceConfig.CustomParams)
		}
	}

	return voiceConfig, nil
}

// sendErrorResponse sends an error response to the client
func (s *ConversationService) sendErrorResponse(clientWS *websocket.Conn, errorMsg string, sessionID uuid.UUID) error {
	errorResponse := map[string]interface{}{
		"type":       "error",
		"message":    errorMsg,
		"session_id": sessionID,
		"timestamp":  time.Now(),
	}

	responseData, err := json.Marshal(errorResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal error response: %w", err)
	}

	return clientWS.WriteMessage(websocket.TextMessage, responseData)
}

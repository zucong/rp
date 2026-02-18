package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/zucong/rp/config"
	"github.com/zucong/rp/llm"
	"github.com/zucong/rp/models"
	"github.com/zucong/rp/services"
)

type ChatHandler struct {
	db        *sqlx.DB
	llmClient *llm.Client
	cfgStore  *config.Store
}

func NewChatHandler(db *sqlx.DB, llmClient *llm.Client, cfgStore *config.Store) *ChatHandler {
	return &ChatHandler{
		db:        db,
		llmClient: llmClient,
		cfgStore:  cfgStore,
	}
}

type SendMessageRequest struct {
	Content string `json:"content"`
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user's participant in this room
	var userParticipant models.RoomParticipant
	err = h.db.Get(&userParticipant, `
		SELECT rp.*, c.name as character_name FROM room_participants rp
		JOIN characters c ON rp.character_id = c.id
		WHERE rp.room_id = ? AND rp.is_user = true LIMIT 1`, roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no user participant found in room"})
		return
	}

	// Store user message
	result, err := h.db.Exec(
		"INSERT INTO messages (room_id, participant_id, content) VALUES (?, ?, ?)",
		roomID, userParticipant.ID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	msgID, _ := result.LastInsertId()

	// Broadcast user message to all clients
	userMessageData := map[string]interface{}{
		"type": "message",
		"message": map[string]interface{}{
			"id":                 msgID,
			"room_id":            roomID,
			"participant_id":     userParticipant.ID,
			"participant_name":   userParticipant.CharacterName,
			"participant_avatar": "",
			"content":            req.Content,
			"is_ai":              false,
			"created_at":         time.Now().Format(time.RFC3339),
		},
	}
	userMessageJSON, _ := json.Marshal(userMessageData)
	log.Printf("[Chat] Broadcasting user message: %s", string(userMessageJSON))
	broadcastToRoom(roomID, string(userMessageJSON))

	// Trigger orchestrator and AI responses in background
	go h.processAIResponses(roomID, userParticipant.ID, req.Content, msgID)

	c.JSON(http.StatusOK, gin.H{"status": "message sent"})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// isWordInText checks if a word appears as a whole word in text
func isWordInText(text, word string) bool {
	if word == "" {
		return false
	}
	// Normalize text: replace punctuation with spaces
	normalized := text
	for _, r := range ".,!?;:\"'()[]{}" {
		normalized = strings.ReplaceAll(normalized, string(r), " ")
	}
	normalized = strings.ToLower(strings.TrimSpace(normalized))
	word = strings.ToLower(word)

	// Check word boundaries
	words := strings.Fields(normalized)
	for _, w := range words {
		if w == word {
			return true
		}
	}
	return false
}

func (h *ChatHandler) processAIResponses(roomID, userParticipantID int64, userMessage string, userMessageID int64) {
	log.Printf("[AI] Processing AI responses for room %d", roomID)

	// Initialize decision recorder
	recorder := services.NewDecisionRecorder(h.db, userMessageID, roomID)

	// Parse @ and ! mentions
	forceInclude, forceExclude := parseMentions(userMessage)
	log.Printf("[AI] Force include: %v, Force exclude: %v", forceInclude, forceExclude)

	// Record mention parsing
	recorder.RecordParseMentions(userMessage, forceInclude, forceExclude)

	// Get AI participants
	var aiParticipants []struct {
		models.RoomParticipant
		CharacterName   string `db:"character_name"`
		CharacterPrompt string `db:"character_prompt"`
	}
	err := h.db.Select(&aiParticipants, `
		SELECT rp.*, c.name as character_name, c.prompt as character_prompt
		FROM room_participants rp
		JOIN characters c ON rp.character_id = c.id
		WHERE rp.room_id = ? AND rp.participant_type = 'ai'`, roomID)
	if err != nil {
		log.Printf("[AI] Failed to get AI participants: %v", err)
		return
	}
	log.Printf("[AI] Found %d AI participants", len(aiParticipants))
	if len(aiParticipants) == 0 {
		return
	}

	// Convert to RoomParticipant slice for selection functions
	participants := make([]models.RoomParticipant, len(aiParticipants))
	charNames := make([]string, len(aiParticipants))
	for i, p := range aiParticipants {
		participants[i] = p.RoomParticipant
		participants[i].CharacterName = p.CharacterName
		charNames[i] = p.CharacterName
	}

	// First apply forceInclude/forceExclude to respect user commands
	preSelectedIDs := mergeSelections(nil, forceInclude, forceExclude, participants)
	log.Printf("[AI] After force include/exclude: %v", preSelectedIDs)

	// Record force include if applicable
	if len(forceInclude) > 0 {
		recorder.RecordForceInclude(forceInclude, preSelectedIDs, preSelectedIDs)
	}

	// Record force exclude if applicable
	if len(forceExclude) > 0 {
		allIDs := make([]int64, len(participants))
		for i, p := range participants {
			allIDs[i] = p.ID
		}
		removedIDs := make([]int64, 0)
		for _, id := range allIDs {
			found := false
			for _, selectedID := range preSelectedIDs {
				if selectedID == id {
					found = true
					break
				}
			}
			if !found {
				removedIDs = append(removedIDs, id)
			}
		}
		recorder.RecordForceExclude(forceExclude, removedIDs, preSelectedIDs)
	}

	// Always use LLM selection if there are 2+ AI participants
	var selectedIDs []int64
	if len(participants) < 2 {
		// Only 1 AI participant, skip LLM selection
		selectedIDs = preSelectedIDs
		log.Printf("[AI] Only 1 AI participant, skipping LLM selection")
		recorder.RecordCharacterSelection(charNames, selectedIDs, []int64{})
	} else {
		// 2+ AI participants, use LLM to select
		selectedIDs = h.selectCharactersWithDecisions(roomID, participants, userMessage, userMessageID, recorder, charNames)
		log.Printf("[AI] LLM selected %d characters", len(selectedIDs))
	}

	// Merge with force include/exclude (again to ensure consistency)
	finalIDs := mergeSelections(selectedIDs, forceInclude, forceExclude, participants)
	log.Printf("[AI] Final %d characters to generate responses", len(finalIDs))

	// Record final character selection
	excludedIDs := make([]int64, 0)
	for _, p := range participants {
		found := false
		for _, id := range finalIDs {
			if id == p.ID {
				found = true
				break
			}
		}
		if !found {
			excludedIDs = append(excludedIDs, p.ID)
		}
	}
	recorder.RecordCharacterSelection(charNames, finalIDs, excludedIDs)

	// Generate responses in parallel
	var wg sync.WaitGroup
	for _, pid := range finalIDs {
		wg.Add(1)
		go func(participantID int64) {
			defer wg.Done()
			h.generateResponse(roomID, participantID, userMessageID, recorder)
		}(pid)
	}
	wg.Wait()
	log.Printf("[AI] All responses generated")
}

func parseMentions(message string) (include, exclude []string) {
	includeRe := regexp.MustCompile(`@(\S+)`)
	excludeRe := regexp.MustCompile(`!(\S+)`)

	for _, m := range includeRe.FindAllStringSubmatch(message, -1) {
		include = append(include, m[1])
	}
	for _, m := range excludeRe.FindAllStringSubmatch(message, -1) {
		exclude = append(exclude, m[1])
	}
	return
}

func (h *ChatHandler) selectCharactersWithDecisions(roomID int64, participants []models.RoomParticipant, message string, messageID int64, recorder *services.DecisionRecorder, charNames []string) []int64 {
	if len(participants) == 0 {
		return nil
	}

	// Get fresh config for LLM call
	cfg, err := h.cfgStore.Get()
	if err != nil {
		log.Printf("[Orchestrator] Failed to get config: %v", err)
		return []int64{participants[0].ID}
	}
	h.llmClient.UpdateConfig(cfg)

	// Get character details for all participants
	var participantDetails []struct {
		ID              int64  `db:"id"`
		CharacterName   string `db:"character_name"`
		CharacterPrompt string `db:"character_prompt"`
	}
	err = h.db.Select(&participantDetails, `
		SELECT rp.id, c.name as character_name, c.prompt as character_prompt
		FROM room_participants rp
		JOIN characters c ON rp.character_id = c.id
		WHERE rp.room_id = ? AND rp.participant_type = 'ai'`, roomID)
	if err != nil {
		log.Printf("[Orchestrator] Failed to get participant details: %v", err)
		return []int64{participants[0].ID}
	}

	// Build character list for LLM
	var charList strings.Builder
	for _, p := range participantDetails {
		charList.WriteString(fmt.Sprintf("- %s (ID: %d)\n", p.CharacterName, p.ID))
	}

	// LLM Intent Analysis: understand who the user is addressing
	intentPrompt := fmt.Sprintf(`You are analyzing a user's message in a group chat to determine which character(s) they are addressing.

Available characters:
%s

User message: "%s"

Analyze the user's intent:
1. Is the user DIRECTLY ADDRESSING a specific character? (e.g., "Alice, how are you?", "What do you think, Bob?")
2. Is the user asking a QUESTION that implies a specific character should answer?
3. Is the user speaking to the GROUP in general?

IMPORTANT: Names mentioned as EXAMPLES or REFERENCES (like "you too, like Alice") do NOT mean that character should reply. The user is talking TO someone ABOUT another character.

Reply in this exact format:
INTENT: direct | question | group
CHARACTERS: ID1,ID2 (or "none")

Examples:
- "Alice, how are you?" → INTENT: direct\nCHARACTERS: 3
- "What do you think?" → INTENT: question\nCHARACTERS: (select 1-2 most relevant)
- "You are like Alice, aren't you?" → INTENT: question\nCHARACTERS: (current speaker, not Alice)
- "How is everyone?" → INTENT: group\nCHARACTERS: 3,4`, charList.String(), message)

	intentMessages := []llm.Message{
		{Role: "system", Content: intentPrompt},
	}

	// Create logged client for intent analysis
	intentLogger := services.NewLoggedClient(h.llmClient, h.db, &services.LLMCallMetadata{
		MessageID: messageID,
		RoomID:    roomID,
		CallType:  "intent_analysis",
	})
	intentResponse, err := intentLogger.Complete(intentMessages, cfg.DefaultModel, 0.1, 100)
	if err != nil {
		log.Printf("[Orchestrator] Intent analysis failed: %v", err)
		if recorder != nil {
			recorder.RecordIntentAnalysis(message, charNames, []int64{participants[0].ID}, "error")
		}
		return []int64{participants[0].ID}
	}

	intentResponse = strings.TrimSpace(strings.ToLower(intentResponse))
	log.Printf("[Orchestrator] Intent response: %s", intentResponse)

	// Parse intent response
	var selectedIDs []int64
	var intent string
	lines := strings.Split(intentResponse, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "intent:") {
			intent = strings.TrimSpace(strings.TrimPrefix(line, "intent:"))
		}
		if strings.HasPrefix(line, "characters:") {
			charPart := strings.TrimPrefix(line, "characters:")
			charPart = strings.TrimSpace(charPart)
			if charPart != "" && charPart != "none" {
				for _, idStr := range strings.Split(charPart, ",") {
					id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
					if err == nil {
						selectedIDs = append(selectedIDs, id)
					}
				}
			}
		}
	}

	if len(selectedIDs) > 0 {
		log.Printf("[Orchestrator] LLM selected: %v", selectedIDs)
		if recorder != nil {
			recorder.RecordIntentAnalysis(message, charNames, selectedIDs, intent)
		}
		return selectedIDs
	}

	// Fallback: general topic-based selection
	fallbackPrompt := fmt.Sprintf(`Select 1-2 characters most relevant to reply to: "%s"

Characters:
%s

Reply with IDs only: 3 or 3,4`, message, charList.String())

	fallbackMessages := []llm.Message{
		{Role: "system", Content: fallbackPrompt},
	}

	// Create logged client for fallback selection
	fallbackLogger := services.NewLoggedClient(h.llmClient, h.db, &services.LLMCallMetadata{
		MessageID: messageID,
		RoomID:    roomID,
		CallType:  "fallback_selection",
	})
	response, err := fallbackLogger.Complete(fallbackMessages, cfg.DefaultModel, 0.1, 50)
	if err != nil {
		log.Printf("[Orchestrator] LLM call failed: %v", err)
		return []int64{participants[0].ID}
	}

	response = strings.TrimSpace(strings.ToLower(response))
	log.Printf("[Orchestrator] LLM response: %s", response)

	if response == "none" || response == "" {
		return nil
	}

	// Parse comma-separated IDs
	var selected []int64
	for _, part := range strings.Split(response, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			continue
		}
		// Validate ID belongs to a participant
		for _, p := range participants {
			if p.ID == id {
				selected = append(selected, id)
				break
			}
		}
	}

	if len(selected) == 0 {
		if recorder != nil {
			recorder.RecordFallbackSelection(message, charNames, []int64{participants[0].ID})
		}
		return []int64{participants[0].ID}
	}

	if recorder != nil {
		recorder.RecordFallbackSelection(message, charNames, selected)
	}
	return selected
}

func mergeSelections(selected []int64, forceInclude, forceExclude []string, participants []models.RoomParticipant) []int64 {
	idSet := make(map[int64]bool)

	// If no selected provided, default to all participants
	if len(selected) == 0 {
		for _, p := range participants {
			idSet[p.ID] = true
		}
	} else {
		// Add orchestrator selections
		for _, id := range selected {
			idSet[id] = true
		}
	}

	// Helper function to check if a name matches a participant
	nameMatches := func(query, fullName string) bool {
		queryLower := strings.ToLower(strings.TrimSpace(query))
		nameLower := strings.ToLower(fullName)
		parts := strings.Fields(nameLower)
		if len(parts) == 0 {
			return false
		}
		firstName := parts[0]
		lastName := ""
		if len(parts) > 1 {
			lastName = parts[len(parts)-1]
		}

		// Remove common titles from query
		titles := []string{"mr.", "ms.", "mrs.", "miss", "dr.", "prof."}
		for _, title := range titles {
			if strings.HasPrefix(queryLower, title+" ") {
				queryLower = strings.TrimSpace(strings.TrimPrefix(queryLower, title))
				break
			}
		}

		// Check various match types
		return nameLower == queryLower || // Full name exact match
			firstName == queryLower || // First name match
			lastName == queryLower || // Last name match
			strings.HasPrefix(firstName, queryLower) || // Partial first name
			strings.HasPrefix(lastName, queryLower) || // Partial last name
			strings.Contains(nameLower, queryLower) // Contains query
	}

	// Add force includes
	for _, name := range forceInclude {
		for _, p := range participants {
			if nameMatches(name, p.CharacterName) {
				idSet[p.ID] = true
			}
		}
	}

	// Remove force excludes
	for _, name := range forceExclude {
		for _, p := range participants {
			if nameMatches(name, p.CharacterName) {
				delete(idSet, p.ID)
			}
		}
	}

	var result []int64
	for id := range idSet {
		result = append(result, id)
	}
	return result
}

func (h *ChatHandler) generateResponse(roomID, participantID int64, messageID int64, recorder *services.DecisionRecorder) {
	log.Printf("[AI] Starting response generation for participant %d in room %d", participantID, roomID)

	// Get participant with character details
	var p struct {
		models.RoomParticipant
		CharacterName   string  `db:"character_name"`
		CharacterAvatar string  `db:"character_avatar"`
		Prompt          string  `db:"prompt"`
		ModelName       string  `db:"model_name"`
		Temperature     float64 `db:"temperature"`
		MaxTokens       int     `db:"max_tokens"`
	}
	err := h.db.Get(&p, `
		SELECT rp.*, c.name as character_name, c.avatar as character_avatar, c.prompt, c.model_name, c.temperature, c.max_tokens
		FROM room_participants rp
		JOIN characters c ON rp.character_id = c.id
		WHERE rp.id = ?`, participantID)
	if err != nil {
		log.Printf("[AI] Failed to get participant: %v", err)
		return
	}
	log.Printf("[AI] Character: %s, Model: %s", p.CharacterName, p.ModelName)

	// Get room info
	var room models.Room
	err = h.db.Get(&room, "SELECT id, name, description, setting, created_at, updated_at FROM rooms WHERE id = ?", roomID)
	if err != nil {
		log.Printf("[AI] Failed to get room: %v", err)
		return
	}

	// Get user persona (the human player this AI is responding to)
	var userPersona string
	var userChar struct {
		Name   string `db:"name"`
		Prompt string `db:"prompt"`
	}
	err = h.db.Get(&userChar, `
		SELECT c.name, c.prompt
		FROM room_participants rp
		JOIN characters c ON rp.character_id = c.id
		WHERE rp.room_id = ? AND rp.participant_type = 'human' AND rp.is_user = true
		LIMIT 1`, roomID)
	if err != nil {
		userPersona = "A user is speaking to you."
	} else {
		userPersona = fmt.Sprintf("Name: %s\n%s", userChar.Name, userChar.Prompt)
	}

	// Build role-aware context - pass current character name
	contextMessages := h.buildContext(roomID, p.CharacterName)

	// Get fresh config for API call
	cfg, err := h.cfgStore.Get()
	if err != nil {
		log.Printf("[AI] Failed to get config: %v", err)
		if recorder != nil {
			recorder.RecordResponseGeneration(participantID, "", 0)
		}
		return
	}
	h.llmClient.UpdateConfig(cfg)

	// Build three-section system prompt
	aiPersona := fmt.Sprintf("You are %s.\n%s", p.CharacterName, p.Prompt)
	setting := room.Setting
	if setting == "" {
		setting = "No specific setting defined."
	}

	mergedSystem := fmt.Sprintf("[AI Persona]\n%s\n\n[Setting]\n%s\n\n[User Persona]\n%s",
		aiPersona, setting, userPersona)

	// Extract conversation messages from contextMessages
	var conversationMessages []contextMessage
	for _, cm := range contextMessages {
		if cm.Role != "system" {
			conversationMessages = append(conversationMessages, cm)
		}
	}
	messages := []llm.Message{
		{Role: "system", Content: mergedSystem},
	}
	for _, cm := range conversationMessages {
		messages = append(messages, llm.Message{Role: cm.Role, Content: cm.Content})
	}
	log.Printf("[AI] Sending %d messages to LLM", len(messages))

	// Create logged client for response generation
	responseLogger := services.NewLoggedClient(h.llmClient, h.db, &services.LLMCallMetadata{
		MessageID: messageID,
		RoomID:    roomID,
		CallType:  "response_generation",
	})
	response, err := responseLogger.Complete(messages, p.ModelName, p.Temperature, p.MaxTokens)
	if err != nil {
		log.Printf("[AI] LLM call failed: %v", err)
		if recorder != nil {
			recorder.RecordResponseGeneration(participantID, p.CharacterName, 0)
		}
		return
	}
	log.Printf("[AI] Got response: %s", response[:min(len(response), 50)])

	// Record successful response generation
	if recorder != nil {
		recorder.RecordResponseGeneration(participantID, p.CharacterName, 0)
	}

	// Store response
	result, err := h.db.Exec(
		"INSERT INTO messages (room_id, participant_id, content) VALUES (?, ?, ?)",
		roomID, participantID, response)
	if err != nil {
		log.Printf("[AI] Failed to store response: %v", err)
		return
	}

	msgID, _ := result.LastInsertId()

	// Broadcast to all connected clients
	messageData := map[string]interface{}{
		"type": "message",
		"message": map[string]interface{}{
			"id":                 msgID,
			"room_id":            roomID,
			"participant_id":     participantID,
			"participant_name":   p.CharacterName,
			"participant_avatar": p.CharacterAvatar,
			"content":            response,
			"is_ai":              true,
			"created_at":         time.Now().Format(time.RFC3339),
		},
	}
	messageJSON, _ := json.Marshal(messageData)
	log.Printf("[AI] Broadcasting message: %s", string(messageJSON[:min(len(messageJSON), 100)]))
	broadcastToRoom(roomID, string(messageJSON))
}

type contextMessage struct {
	Role    string
	Content string
}

func (h *ChatHandler) buildContext(roomID int64, characterName string) []contextMessage {
	// Get recent messages with participant type
	var messages []struct {
		Name            string `db:"character_name"`
		Content         string `db:"content"`
		ParticipantType string `db:"participant_type"`
	}
	err := h.db.Select(&messages, `
		SELECT c.name as character_name, m.content, rp.participant_type
		FROM messages m
		JOIN room_participants rp ON m.participant_id = rp.id
		JOIN characters c ON rp.character_id = c.id
		WHERE m.room_id = ?
		ORDER BY m.created_at DESC
		LIMIT 20`, roomID)
	var result []contextMessage

	if err != nil {
		log.Printf("[Context] Failed to get messages: %v", err)
		return result
	}

	// Reverse to get chronological order
	// Role-aware: current character's messages are "assistant", others are "user"
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]

		if m.ParticipantType == "ai" {
			// Use case-insensitive matching for character names
			if strings.EqualFold(m.Name, characterName) {
				// Current character's own message - assistant role
				// Strip legacy name prefix if present
				content := m.Content
				prefix := m.Name + ": "
				if strings.HasPrefix(content, prefix) {
					content = strings.TrimPrefix(content, prefix)
				}
				result = append(result, contextMessage{Role: "assistant", Content: content})
			} else {
				// Other AI character's message - user role with name prefix
				result = append(result, contextMessage{Role: "user", Content: fmt.Sprintf("%s: %s", m.Name, m.Content)})
			}
		} else {
			// Human user message - user role with name prefix
			result = append(result, contextMessage{Role: "user", Content: fmt.Sprintf("%s: %s", m.Name, m.Content)})
		}
	}

	return result
}

// SSE events for real-time updates
var sseClients = make(map[int64][]chan string)
var sseMu sync.RWMutex

func broadcastToRoom(roomID int64, message string) {
	sseMu.RLock()
	clients := sseClients[roomID]
	sseMu.RUnlock()

	for _, ch := range clients {
		select {
		case ch <- message:
		default:
		}
	}
}

func (h *ChatHandler) Events(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ch := make(chan string, 10)
	sseMu.Lock()
	sseClients[roomID] = append(sseClients[roomID], ch)
	sseMu.Unlock()

	defer func() {
		sseMu.Lock()
		clients := sseClients[roomID]
		for i, client := range clients {
			if client == ch {
				sseClients[roomID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
		sseMu.Unlock()
		close(ch)
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-ch:
			if !ok {
				return false
			}
			// Write raw SSE data without event type for default onmessage handler
			fmt.Fprintf(w, "data: %s\n\n", msg)
			c.Writer.Flush()
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

type EditMessageRequest struct {
	Content string `json:"content"`
}

func (h *ChatHandler) EditMessage(c *gin.Context) {
	msgID, err := strconv.ParseInt(c.Param("msgId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	var req EditMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get message to check if it's from a user (not AI)
	var msg struct {
		ParticipantID   int64  `db:"participant_id"`
		RoomID          int64  `db:"room_id"`
		ParticipantType string `db:"participant_type"`
	}
	err = h.db.Get(&msg, `
		SELECT m.participant_id, m.room_id, rp.participant_type
		FROM messages m
		JOIN room_participants rp ON m.participant_id = rp.id
		WHERE m.id = ?`, msgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}

	// Update message (allow editing both user and AI messages)
	_, err = h.db.Exec(
		"UPDATE messages SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		req.Content, msgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast edit event
	editData := map[string]interface{}{
		"type":       "message_edited",
		"message_id": msgID,
		"content":    req.Content,
	}
	editJSON, _ := json.Marshal(editData)
	broadcastToRoom(msg.RoomID, string(editJSON))

	c.Status(http.StatusNoContent)
}

func (h *ChatHandler) Regenerate(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	// Get user's last message in this room
	var lastUserMsg struct {
		ID            int64 `db:"id"`
		ParticipantID int64 `db:"participant_id"`
	}
	err = h.db.Get(&lastUserMsg, `
		SELECT m.id, m.participant_id FROM messages m
		JOIN room_participants rp ON m.participant_id = rp.id
		WHERE m.room_id = ? AND rp.is_user = true
		ORDER BY m.created_at DESC LIMIT 1`, roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no user message found"})
		return
	}

	// Get all AI messages after user's last message
	var aiMessages []struct {
		ID int64 `db:"id"`
	}
	err = h.db.Select(&aiMessages, `
		SELECT m.id FROM messages m
		JOIN room_participants rp ON m.participant_id = rp.id
		WHERE m.room_id = ? AND rp.participant_type = 'ai' AND m.created_at > (
			SELECT created_at FROM messages WHERE id = ?
		)`, roomID, lastUserMsg.ID)
	if err != nil {
		log.Printf("[Regenerate] Failed to get AI messages: %v", err)
	}

	// Delete AI messages and broadcast delete events
	for _, msg := range aiMessages {
		_, err := h.db.Exec("DELETE FROM messages WHERE id = ?", msg.ID)
		if err != nil {
			log.Printf("[Regenerate] Failed to delete message %d: %v", msg.ID, err)
			continue
		}
		// Broadcast delete event
		deleteData := map[string]interface{}{
			"type":       "message_deleted",
			"message_id": msg.ID,
		}
		deleteJSON, _ := json.Marshal(deleteData)
		broadcastToRoom(roomID, string(deleteJSON))
	}

	log.Printf("[Regenerate] Deleted %d AI messages, triggering regeneration", len(aiMessages))

	// Get user's last message content
	var userContent string
	err = h.db.Get(&userContent, "SELECT content FROM messages WHERE id = ?", lastUserMsg.ID)
	if err != nil {
		userContent = ""
	}

	// Trigger AI responses in background
	go h.processAIResponses(roomID, lastUserMsg.ParticipantID, userContent, lastUserMsg.ID)

	c.JSON(http.StatusOK, gin.H{"status": "regenerating", "deleted_count": len(aiMessages)})
}

func (h *ChatHandler) DeleteMessage(c *gin.Context) {
	msgID, err := strconv.ParseInt(c.Param("msgId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	// Get message info before deleting
	var msg struct {
		ParticipantID   int64  `db:"participant_id"`
		RoomID          int64  `db:"room_id"`
		ParticipantType string `db:"participant_type"`
	}
	err = h.db.Get(&msg, `
		SELECT m.participant_id, m.room_id, rp.participant_type
		FROM messages m
		JOIN room_participants rp ON m.participant_id = rp.id
		WHERE m.id = ?`, msgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}

	// Delete message
	_, err = h.db.Exec("DELETE FROM messages WHERE id = ?", msgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast delete event
	deleteData := map[string]interface{}{
		"type":       "message_deleted",
		"message_id": msgID,
	}
	deleteJSON, _ := json.Marshal(deleteData)
	broadcastToRoom(msg.RoomID, string(deleteJSON))

	c.Status(http.StatusNoContent)
}

func (h *ChatHandler) GetLLMLogs(c *gin.Context) {
	msgID, err := strconv.ParseInt(c.Param("msgId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	logs, err := services.GetLogsForMessage(h.db, msgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (h *ChatHandler) GetDecisions(c *gin.Context) {
	msgID, err := strconv.ParseInt(c.Param("msgId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	decisions, err := services.GetDecisionsForMessage(h.db, msgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"decisions": decisions})
}

package services

import (
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/zucong/rp/llm"
	"github.com/zucong/rp/models"
)

// LLMCallMetadata contains context for the LLM call
type LLMCallMetadata struct {
	MessageID int64
	RoomID    int64
	CallType  string // intent_analysis, fallback_selection, response_generation
}

// LoggedClient wraps llm.Client to record all API calls
type LoggedClient struct {
	client   *llm.Client
	db       *sqlx.DB
	metadata *LLMCallMetadata
}

// NewLoggedClient creates a new logging wrapper around llm.Client
func NewLoggedClient(client *llm.Client, db *sqlx.DB, metadata *LLMCallMetadata) *LoggedClient {
	return &LoggedClient{
		client:   client,
		db:       db,
		metadata: metadata,
	}
}

// Complete wraps the original Complete method with logging
func (lc *LoggedClient) Complete(messages []llm.Message, model string, temperature float64, maxTokens int) (string, error) {
	response, _, err := lc.CompleteWithLogID(messages, model, temperature, maxTokens)
	return response, err
}

// CompleteWithLogID wraps Complete and returns the LLM call log ID for decision tracking
func (lc *LoggedClient) CompleteWithLogID(messages []llm.Message, model string, temperature float64, maxTokens int) (string, int64, error) {
	start := time.Now()

	// Serialize request
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": temperature,
		"max_tokens":  maxTokens,
	})

	// Make the actual call
	response, err := lc.client.Complete(messages, model, temperature, maxTokens)

	latency := time.Since(start).Milliseconds()

	// Prepare log entry
	log := models.LLMCallLog{
		MessageID:   lc.metadata.MessageID,
		RoomID:      lc.metadata.RoomID,
		CallType:    lc.metadata.CallType,
		ModelName:   model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		RequestBody: string(reqBody),
		LatencyMs:   latency,
	}

	if err != nil {
		log.ErrorMessage = err.Error()
	} else {
		log.ResponseBody = response
		// Note: token counts would need to be parsed from actual API response
		// which requires modifying the llm.Client to return usage info
	}

	// Sync write to get the ID
	logID := lc.saveLogSync(&log)

	return response, logID, err
}

func (lc *LoggedClient) saveLog(log *models.LLMCallLog) {
	lc.saveLogSync(log)
}

func (lc *LoggedClient) saveLogSync(log *models.LLMCallLog) int64 {
	result, err := lc.db.Exec(`
		INSERT INTO llm_call_logs (
			message_id, room_id, call_type, model_name, temperature, max_tokens,
			request_body, response_body, prompt_tokens, completion_tokens,
			latency_ms, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, log.MessageID, log.RoomID, log.CallType, log.ModelName, log.Temperature,
		log.MaxTokens, log.RequestBody, log.ResponseBody, log.PromptTokens,
		log.CompletionTokens, log.LatencyMs, log.ErrorMessage)

	if err != nil {
		// Log error but don't fail the main flow
		println("Failed to save LLM call log:", err.Error())
		return 0
	}

	logID, _ := result.LastInsertId()
	return logID
}

// GetLogsForMessage retrieves all LLM call logs for a specific message
func GetLogsForMessage(db *sqlx.DB, messageID int64) ([]models.LLMCallLog, error) {
	var logs []models.LLMCallLog
	err := db.Select(&logs, `
		SELECT * FROM llm_call_logs
		WHERE message_id = ?
		ORDER BY created_at ASC
	`, messageID)
	return logs, err
}

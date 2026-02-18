package services

import (
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"github.com/zucong/rp/models"
)

// DecisionRecorder tracks orchestrator decision steps
type DecisionRecorder struct {
	db        *sqlx.DB
	messageID int64
	roomID    int64
	stepOrder int
}

// NewDecisionRecorder creates a new recorder for a message
func NewDecisionRecorder(db *sqlx.DB, messageID, roomID int64) *DecisionRecorder {
	return &DecisionRecorder{
		db:        db,
		messageID: messageID,
		roomID:    roomID,
		stepOrder: 0,
	}
}

// RecordParseMentions records the mention parsing step
func (dr *DecisionRecorder) RecordParseMentions(userMessage string, forceInclude, forceExclude []string) error {
	input, _ := json.Marshal(map[string]string{
		"user_message": userMessage,
	})
	output, _ := json.Marshal(map[string][]string{
		"force_include": forceInclude,
		"force_exclude": forceExclude,
	})

	return dr.recordStep("parse_mentions", string(input), string(output), 0,
		"Parsed @mentions for force include and !mentions for force exclude")
}

// RecordIntentAnalysis records the intent analysis step
func (dr *DecisionRecorder) RecordIntentAnalysis(userMessage string, availableChars []string, selectedIDs []int64, intent string) error {
	input, _ := json.Marshal(map[string]interface{}{
		"user_message":    userMessage,
		"available_chars": availableChars,
	})
	output, _ := json.Marshal(map[string]interface{}{
		"intent":       intent,
		"selected_ids": selectedIDs,
	})

	return dr.recordStep("intent_analysis", string(input), string(output), 0,
		"LLM analyzed user intent to determine which characters should reply")
}

// RecordFallbackSelection records the fallback selection step
func (dr *DecisionRecorder) RecordFallbackSelection(userMessage string, availableChars []string, selectedIDs []int64) error {
	input, _ := json.Marshal(map[string]interface{}{
		"user_message":    userMessage,
		"available_chars": availableChars,
	})
	output, _ := json.Marshal(map[string][]int64{
		"selected_ids": selectedIDs,
	})

	return dr.recordStep("fallback_selection", string(input), string(output), 0,
		"Fallback LLM selection used because intent analysis did not produce valid results")
}

// RecordForceInclude records when characters are force-included
func (dr *DecisionRecorder) RecordForceInclude(names []string, addedIDs []int64, finalIDs []int64) error {
	input, _ := json.Marshal(map[string][]string{
		"force_include_names": names,
	})
	output, _ := json.Marshal(map[string]interface{}{
		"added_ids": addedIDs,
		"final_ids": finalIDs,
	})

	return dr.recordStep("apply_force_include", string(input), string(output), 0,
		"Applied @mentions to force include specified characters")
}

// RecordForceExclude records when characters are force-excluded
func (dr *DecisionRecorder) RecordForceExclude(names []string, removedIDs []int64, finalIDs []int64) error {
	input, _ := json.Marshal(map[string][]string{
		"force_exclude_names": names,
	})
	output, _ := json.Marshal(map[string]interface{}{
		"removed_ids": removedIDs,
		"final_ids":   finalIDs,
	})

	return dr.recordStep("apply_force_exclude", string(input), string(output), 0,
		"Applied !mentions to force exclude specified characters")
}

// RecordCharacterSelection records the final character selection
func (dr *DecisionRecorder) RecordCharacterSelection(allParticipants []string, selectedIDs []int64, excludedIDs []int64) error {
	input, _ := json.Marshal(map[string][]string{
		"all_participants": allParticipants,
	})
	output, _ := json.Marshal(map[string]interface{}{
		"selected_ids": selectedIDs,
		"excluded_ids": excludedIDs,
	})

	return dr.recordStep("character_selection", string(input), string(output), 0,
		"Final character selection after all filters applied")
}

// RecordResponseGeneration records that a response was generated for a character
func (dr *DecisionRecorder) RecordResponseGeneration(characterID int64, characterName string, llmCallLogID int64) error {
	input, _ := json.Marshal(map[string]interface{}{
		"character_id":   characterID,
		"character_name": characterName,
	})
	output, _ := json.Marshal(map[string]string{
		"status": "generated",
	})

	return dr.recordStep("response_generation", string(input), string(output), llmCallLogID,
		"Generated AI response for character")
}

// recordStep is the internal method to save a decision step
func (dr *DecisionRecorder) recordStep(stepType, inputData, outputData string, llmCallLogID int64, reason string) error {
	dr.stepOrder++

	_, err := dr.db.Exec(`
		INSERT INTO orchestrator_decisions (
			message_id, room_id, step_order, step_type,
			input_data, output_data, llm_call_log_id, reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, dr.messageID, dr.roomID, dr.stepOrder, stepType,
		inputData, outputData, llmCallLogID, reason)

	return err
}

// GetDecisionsForMessage retrieves all decision steps for a message
func GetDecisionsForMessage(db *sqlx.DB, messageID int64) ([]models.OrchestratorDecision, error) {
	var decisions []models.OrchestratorDecision
	err := db.Select(&decisions, `
		SELECT * FROM orchestrator_decisions
		WHERE message_id = ?
		ORDER BY step_order ASC
	`, messageID)
	return decisions, err
}

// GetDecisionsWithLLMLogs retrieves decisions with their associated LLM call details
type DecisionWithLLMLog struct {
	models.OrchestratorDecision
	LLMCallLog *models.LLMCallLog `db:"-" json:"llm_call_log,omitempty"`
}

func GetDecisionsWithLLMLogs(db *sqlx.DB, messageID int64) ([]DecisionWithLLMLog, error) {
	decisions, err := GetDecisionsForMessage(db, messageID)
	if err != nil {
		return nil, err
	}

	result := make([]DecisionWithLLMLog, len(decisions))
	for i, d := range decisions {
		result[i] = DecisionWithLLMLog{
			OrchestratorDecision: d,
		}

		// Fetch associated LLM call if exists
		if d.LLMCallLogID > 0 {
			var llmLog models.LLMCallLog
			err := db.Get(&llmLog, "SELECT * FROM llm_call_logs WHERE id = ?", d.LLMCallLogID)
			if err == nil {
				result[i].LLMCallLog = &llmLog
			}
		}
	}

	return result, nil
}

package models

import "time"

type Character struct {
	ID             int64     `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Avatar         string    `json:"avatar" db:"avatar"`
	Prompt         string    `json:"prompt" db:"prompt"`
	IsUserPlayable bool      `json:"is_user_playable" db:"is_user_playable"`
	ModelName      string    `json:"model_name" db:"model_name"`
	Temperature    float64   `json:"temperature" db:"temperature"`
	MaxTokens      int       `json:"max_tokens" db:"max_tokens"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type Room struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Setting     string    `json:"setting" db:"setting"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type RoomParticipant struct {
	ID               int64     `json:"id" db:"id"`
	RoomID           int64     `json:"room_id" db:"room_id"`
	CharacterID      int64     `json:"character_id" db:"character_id"`
	CharacterName    string    `json:"character_name" db:"character_name"`
	CharacterAvatar  string    `json:"character_avatar" db:"character_avatar"`
	ParticipantType  string    `json:"participant_type" db:"participant_type"`
	IsUser           bool      `json:"is_user" db:"is_user"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

type Message struct {
	ID              int64     `json:"id" db:"id"`
	RoomID          int64     `json:"room_id" db:"room_id"`
	ParticipantID   int64     `json:"participant_id" db:"participant_id"`
	ParticipantName string    `json:"participant_name" db:"participant_name"`
	ParticipantAvatar string  `json:"participant_avatar" db:"participant_avatar"`
	Content         string    `json:"content" db:"content"`
	IsAI            bool      `json:"is_ai" db:"is_ai"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type Summary struct {
	ID        int64     `json:"id" db:"id"`
	RoomID    int64     `json:"room_id" db:"room_id"`
	Content   string    `json:"content" db:"content"`
	MsgFrom   int64     `json:"message_from" db:"message_from"`
	MsgTo     int64     `json:"message_to" db:"message_to"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Config struct {
	APIEndpoint   string  `json:"api_endpoint" db:"api_endpoint"`
	APIKey        string  `json:"api_key" db:"api_key"`
	DefaultModel  string  `json:"default_model" db:"default_model"`
}

type LLMCallLog struct {
	ID                int64     `json:"id" db:"id"`
	MessageID         int64     `json:"message_id" db:"message_id"`
	RoomID            int64     `json:"room_id" db:"room_id"`
	CallType          string    `json:"call_type" db:"call_type"`
	ModelName         string    `json:"model_name" db:"model_name"`
	Temperature       float64   `json:"temperature" db:"temperature"`
	MaxTokens         int       `json:"max_tokens" db:"max_tokens"`
	RequestBody       string    `json:"request_body" db:"request_body"`
	ResponseBody      string    `json:"response_body" db:"response_body"`
	PromptTokens      int       `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens  int       `json:"completion_tokens" db:"completion_tokens"`
	LatencyMs         int64     `json:"latency_ms" db:"latency_ms"`
	ErrorMessage      string    `json:"error_message" db:"error_message"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

type OrchestratorDecision struct {
	ID            int64     `json:"id" db:"id"`
	MessageID     int64     `json:"message_id" db:"message_id"`
	RoomID        int64     `json:"room_id" db:"room_id"`
	StepOrder     int       `json:"step_order" db:"step_order"`
	StepType      string    `json:"step_type" db:"step_type"`
	InputData     string    `json:"input_data" db:"input_data"`
	OutputData    string    `json:"output_data" db:"output_data"`
	LLMCallLogID  int64     `json:"llm_call_log_id" db:"llm_call_log_id"`
	Reason        string    `json:"reason" db:"reason"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

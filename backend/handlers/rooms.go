package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/zucong/rp/models"
)

type RoomHandler struct {
	db *sqlx.DB
}

func NewRoomHandler(db *sqlx.DB) *RoomHandler {
	return &RoomHandler{db: db}
}

func (h *RoomHandler) List(c *gin.Context) {
	query := `
		SELECT r.id, r.name, r.description, r.setting, r.created_at, r.updated_at,
			(SELECT COUNT(*) FROM room_participants WHERE room_id = r.id) as participant_count,
			(SELECT MAX(created_at) FROM messages WHERE room_id = r.id) as last_activity
		FROM rooms r
		ORDER BY r.created_at DESC
	`
	var rooms []struct {
		models.Room
		ParticipantCount int     `json:"participant_count" db:"participant_count"`
		LastActivity     *string `json:"last_activity" db:"last_activity"`
	}
	err := h.db.Select(&rooms, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rooms)
}

func (h *RoomHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var room models.Room
	err = h.db.Get(&room, "SELECT id, name, description, setting, created_at, updated_at FROM rooms WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	c.JSON(http.StatusOK, room)
}

func (h *RoomHandler) Create(c *gin.Context) {
	var room models.Room
	if err := c.ShouldBindJSON(&room); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if room.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	result, err := h.db.NamedExec(
		`INSERT INTO rooms (name, description, setting)
		VALUES (:name, :description, :setting)`,
		&room,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	room.ID = id
	c.JSON(http.StatusCreated, room)
}

func (h *RoomHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var room models.Room
	if err := c.ShouldBindJSON(&room); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	room.ID = id
	_, err = h.db.NamedExec(
		`UPDATE rooms SET
			name = :name,
			description = :description,
			setting = :setting,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = :id`,
		&room,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, room)
}

func (h *RoomHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	_, err = h.db.Exec("DELETE FROM rooms WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Participant management
func (h *RoomHandler) ListParticipants(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	query := `
		SELECT
			rp.id,
			rp.room_id,
			rp.character_id,
			c.name as character_name,
			c.avatar as character_avatar,
			rp.participant_type,
			rp.is_user,
			rp.created_at
		FROM room_participants rp
		JOIN characters c ON rp.character_id = c.id
		WHERE rp.room_id = ?
	`
	var participants []models.RoomParticipant
	err = h.db.Select(&participants, query, roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, participants)
}

func (h *RoomHandler) AddParticipant(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var input struct {
		CharacterID     int64  `json:"character_id"`
		ParticipantType string `json:"participant_type"`
		IsUser          bool   `json:"is_user"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = h.db.Exec(
		"INSERT INTO room_participants (room_id, character_id, participant_type, is_user) VALUES (?, ?, ?, ?)",
		roomID, input.CharacterID, input.ParticipantType, input.IsUser,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *RoomHandler) RemoveParticipant(c *gin.Context) {
	participantID, err := strconv.ParseInt(c.Param("pid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}

	_, err = h.db.Exec("DELETE FROM room_participants WHERE id = ?", participantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *RoomHandler) ListMessages(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	query := `
		SELECT
			m.id,
			m.room_id,
			m.participant_id,
			c.name as participant_name,
			c.avatar as participant_avatar,
			m.content,
			rp.participant_type = 'ai' as is_ai,
			m.created_at
		FROM messages m
		JOIN room_participants rp ON m.participant_id = rp.id
		JOIN characters c ON rp.character_id = c.id
		WHERE m.room_id = ?
		ORDER BY m.created_at ASC
	`
	var messages []models.Message
	err = h.db.Select(&messages, query, roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (h *RoomHandler) ResetChat(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	_, err = h.db.Exec("DELETE FROM messages WHERE room_id = ?", roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

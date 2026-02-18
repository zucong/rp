package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/zucong/rp/models"
)

type CharacterHandler struct {
	db *sqlx.DB
}

func NewCharacterHandler(db *sqlx.DB) *CharacterHandler {
	return &CharacterHandler{db: db}
}

func (h *CharacterHandler) List(c *gin.Context) {
	var characters []models.Character
	err := h.db.Select(&characters, "SELECT * FROM characters ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, characters)
}

func (h *CharacterHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var character models.Character
	err = h.db.Get(&character, "SELECT * FROM characters WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "character not found"})
		return
	}

	c.JSON(http.StatusOK, character)
}

func (h *CharacterHandler) Create(c *gin.Context) {
	var character models.Character
	if err := c.ShouldBindJSON(&character); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if character.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	result, err := h.db.NamedExec(
		`INSERT INTO characters (name, avatar, prompt, is_user_playable, model_name, temperature, max_tokens)
		VALUES (:name, :avatar, :prompt, :is_user_playable, :model_name, :temperature, :max_tokens)`,
		&character,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	character.ID = id
	c.JSON(http.StatusCreated, character)
}

func (h *CharacterHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var character models.Character
	if err := c.ShouldBindJSON(&character); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	character.ID = id
	_, err = h.db.NamedExec(
		`UPDATE characters SET
			name = :name,
			avatar = :avatar,
			prompt = :prompt,
			is_user_playable = :is_user_playable,
			model_name = :model_name,
			temperature = :temperature,
			max_tokens = :max_tokens,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = :id`,
		&character,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, character)
}

func (h *CharacterHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check if character is in use
	var count int
	err = h.db.Get(&count, "SELECT COUNT(*) FROM room_participants WHERE character_id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "character is in use in one or more rooms"})
		return
	}

	_, err = h.db.Exec("DELETE FROM characters WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

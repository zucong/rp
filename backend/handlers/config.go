package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/zucong/rp/models"
)

type ConfigHandler struct {
	db *sqlx.DB
}

func NewConfigHandler(db *sqlx.DB) *ConfigHandler {
	return &ConfigHandler{db: db}
}

func (h *ConfigHandler) Get(c *gin.Context) {
	var cfg models.Config
	err := h.db.Get(&cfg, "SELECT api_endpoint, api_key, default_model FROM config WHERE id = 1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (h *ConfigHandler) Update(c *gin.Context) {
	var cfg models.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.Exec(
		"UPDATE config SET api_endpoint = ?, api_key = ?, default_model = ? WHERE id = 1",
		cfg.APIEndpoint, cfg.APIKey, cfg.DefaultModel,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cfg)
}

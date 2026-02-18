package config

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/zucong/rp/models"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
	LLM struct {
		APIEndpoint  string `yaml:"api_endpoint"`
		APIKey       string `yaml:"api_key"`
		DefaultModel string `yaml:"default_model"`
	} `yaml:"llm"`
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
}

var GlobalConfig AppConfig

func LoadConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &GlobalConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Get() (*models.Config, error) {
	var cfg models.Config
	err := s.db.Get(&cfg, "SELECT api_endpoint, api_key, default_model FROM config WHERE id = 1")
	return &cfg, err
}

func (s *Store) Update(cfg *models.Config) error {
	_, err := s.db.Exec(
		"UPDATE config SET api_endpoint = ?, api_key = ?, default_model = ? WHERE id = 1",
		cfg.APIEndpoint, cfg.APIKey, cfg.DefaultModel,
	)
	return err
}

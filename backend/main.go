package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/zucong/rp/config"
	"github.com/zucong/rp/db"
	"github.com/zucong/rp/handlers"
	"github.com/zucong/rp/llm"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Initialize database with config path
	if err := db.Init(config.GlobalConfig.Database.Path); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize config store
	cfgStore := config.NewStore(db.DB)

	// Get initial config for LLM client
	cfg, err := cfgStore.Get()
	if err != nil {
		log.Fatal("Failed to get config:", err)
	}

	// Initialize LLM client
	llmClient := llm.NewClient(cfg)

	// Setup router
	r := gin.Default()

	// CORS
	r.Use(cors.Default())

	// Static files (frontend)
	r.Static("/static", "./frontend/dist/assets")
	r.StaticFile("/", "./frontend/dist/index.html")

	// API routes
	api := r.Group("/api")
	{
		// Characters
		charHandler := handlers.NewCharacterHandler(db.DB)
		api.GET("/characters", charHandler.List)
		api.GET("/characters/:id", charHandler.Get)
		api.POST("/characters", charHandler.Create)
		api.PUT("/characters/:id", charHandler.Update)
		api.DELETE("/characters/:id", charHandler.Delete)

		// Rooms
		roomHandler := handlers.NewRoomHandler(db.DB)
		api.GET("/rooms", roomHandler.List)
		api.GET("/rooms/:id", roomHandler.Get)
		api.POST("/rooms", roomHandler.Create)
		api.PUT("/rooms/:id", roomHandler.Update)
		api.DELETE("/rooms/:id", roomHandler.Delete)
		api.GET("/rooms/:id/participants", roomHandler.ListParticipants)
		api.POST("/rooms/:id/participants", roomHandler.AddParticipant)
		api.DELETE("/rooms/:id/participants/:pid", roomHandler.RemoveParticipant)
		api.GET("/rooms/:id/messages", roomHandler.ListMessages)
		api.DELETE("/rooms/:id/messages", roomHandler.ResetChat)

		// Config
		configHandler := handlers.NewConfigHandler(db.DB)
		api.GET("/config", configHandler.Get)
		api.PUT("/config", configHandler.Update)

		// Chat
		chatHandler := handlers.NewChatHandler(db.DB, llmClient, cfgStore)
		api.POST("/rooms/:id/chat", chatHandler.SendMessage)
		api.GET("/rooms/:id/events", chatHandler.Events)
		api.PUT("/messages/:msgId", chatHandler.EditMessage)
		api.DELETE("/messages/:msgId", chatHandler.DeleteMessage)
		api.POST("/rooms/:id/regenerate", chatHandler.Regenerate)
		api.GET("/messages/:msgId/llm-logs", chatHandler.GetLLMLogs)
		api.GET("/messages/:msgId/decisions", chatHandler.GetDecisions)
	}

	// Use port from config
	addr := config.GlobalConfig.Server.Host + ":" + fmt.Sprintf("%d", config.GlobalConfig.Server.Port)
	log.Println("Server starting on", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

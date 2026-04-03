package main

import (
	"log"

	"github.com/gin-gonic/gin"

	authHandler "github.com/Zeropeepo/neknow-bot/internal/auth/handler"
	authRepo "github.com/Zeropeepo/neknow-bot/internal/auth/repository"
	authService "github.com/Zeropeepo/neknow-bot/internal/auth/service"

	botHandler "github.com/Zeropeepo/neknow-bot/internal/bot/handler"
	botRepo "github.com/Zeropeepo/neknow-bot/internal/bot/repository"
	botService "github.com/Zeropeepo/neknow-bot/internal/bot/service"

	fileHandler "github.com/Zeropeepo/neknow-bot/internal/files/handler"
	fileRepository "github.com/Zeropeepo/neknow-bot/internal/files/repository"
	fileService "github.com/Zeropeepo/neknow-bot/internal/files/service"

	chatHandler "github.com/Zeropeepo/neknow-bot/internal/chat/handler"
	chatRepo "github.com/Zeropeepo/neknow-bot/internal/chat/repository"
	chatService "github.com/Zeropeepo/neknow-bot/internal/chat/service"
	"github.com/Zeropeepo/neknow-bot/pkg/ai"

	"github.com/Zeropeepo/neknow-bot/pkg/config"
	"github.com/Zeropeepo/neknow-bot/pkg/database"
	"github.com/Zeropeepo/neknow-bot/pkg/middleware"
	"github.com/Zeropeepo/neknow-bot/pkg/queue"
	"github.com/Zeropeepo/neknow-bot/pkg/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect postgres: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	// Auth
	aRepo := authRepo.NewAuthRepository(db)
	aSvc := authService.NewAuthService(aRepo, cfg)
	aHdlr := authHandler.NewAuthHandler(aSvc)

	// Bots
	botRepo := botRepo.NewBotRepository(db)
	botSvc := botService.NewBotService(botRepo, cfg)
	botHdlr := botHandler.NewBotHandler(botSvc)

	// Chat
	ragClient := ai.NewRAGClient(cfg)
	convRepo := chatRepo.NewConversationRepository(db)
	msgRepo := chatRepo.NewMessageRepository(db)
	cService := chatService.NewChatService(convRepo, msgRepo, botRepo, ragClient)
	cHandler := chatHandler.NewChatHandler(cService)

	// Storage
	minioStorage, err := storage.NewMinIOStorage(cfg)
	if err != nil {
		log.Fatalf("failed to initialize MinIO: %v", err)
	}

	// Queue
	rabbitMQ, err := queue.NewRabbitMQ(cfg)
	if err != nil {
		log.Fatalf("failed to initialize RabbitMQ: %v", err)
	}

	// Files
	fileRepo := fileRepository.NewFileRepository(db)
	fileSvc := fileService.NewFileService(fileRepo, botRepo, minioStorage, rabbitMQ, cfg.MinIO.Bucket)
	fileHdlr := fileHandler.NewFileHandler(fileSvc)

	r := gin.Default()
	r.RedirectTrailingSlash = false

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", aHdlr.Register)
			auth.POST("/login", aHdlr.Login)
			auth.POST("/refresh", aHdlr.RefreshToken)
		}

		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(aSvc))
		{
			protected.GET("/me", aHdlr.Me)

			bots := protected.Group("/bots")
			{
				bots.POST("", botHdlr.Create)
				bots.GET("", botHdlr.GetAll)
				bots.GET("/:id", botHdlr.GetByID)
				bots.PUT("/:id", botHdlr.Update)
				bots.DELETE("/:id", botHdlr.Delete)

				bots.POST("/:id/files", fileHdlr.Upload)
				bots.GET("/:id/files", fileHdlr.GetAll)
				bots.GET("/:id/files/:file_id", fileHdlr.GetByID)
				bots.DELETE("/:id/files/:file_id", fileHdlr.Delete)

				bots.POST("/:id/conversations", cHandler.CreateConversation)
				bots.GET("/:id/conversations", cHandler.GetConversations)
				bots.GET("/:id/conversations/:conv_id", cHandler.GetConversation)
				bots.PUT("/:id/conversations/:conv_id", cHandler.UpdateConversation)
				bots.POST("/:id/conversations/:conv_id/messages", cHandler.SendMessage)
				bots.DELETE("/:id/conversations/:conv_id", cHandler.DeleteConversation)
			}

		}
	}

	if err := r.Run(":" + cfg.App.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

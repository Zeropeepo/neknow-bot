package main

import (
	"log"

	"github.com/gin-gonic/gin"

	authHandler "github.com/Zeropeepo/neknow-bot/internal/auth/handler"
	authRepo    "github.com/Zeropeepo/neknow-bot/internal/auth/repository"
	authService "github.com/Zeropeepo/neknow-bot/internal/auth/service"

	botHandler "github.com/Zeropeepo/neknow-bot/internal/bot/handler"
	botRepo    "github.com/Zeropeepo/neknow-bot/internal/bot/repository"
	botService "github.com/Zeropeepo/neknow-bot/internal/bot/service"

	fileHandler    "github.com/Zeropeepo/neknow-bot/internal/files/handler"
    fileRepository "github.com/Zeropeepo/neknow-bot/internal/files/repository"
    fileService    "github.com/Zeropeepo/neknow-bot/internal/files/service"
    
	"github.com/Zeropeepo/neknow-bot/pkg/queue"
    "github.com/Zeropeepo/neknow-bot/pkg/storage"
	"github.com/Zeropeepo/neknow-bot/pkg/config"
	"github.com/Zeropeepo/neknow-bot/pkg/database"
	"github.com/Zeropeepo/neknow-bot/pkg/middleware"
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
	aSvc  := authService.NewAuthService(aRepo, cfg)
	aHdlr := authHandler.NewAuthHandler(aSvc)

	// Bots
	botRepo := botRepo.NewBotRepository(db)
	botSvc := botService.NewBotService(botRepo, cfg)
	botHdlr := botHandler.NewBotHandler(botSvc)

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

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", aHdlr.Register)
			auth.POST("/login",    aHdlr.Login)
			auth.POST("/refresh",  aHdlr.RefreshToken)
		}

		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(aSvc))
		{
			protected.GET("/me", aHdlr.Me)

			bots := protected.Group("/bots")
			{
				bots.POST("/", botHdlr.Create)
				bots.GET("/", botHdlr.GetAll)
				bots.GET("/:id", botHdlr.GetByID)
				bots.PUT("/:id", botHdlr.Update)
				bots.DELETE("/:id", botHdlr.Delete)

				bots.POST("/:id/files", fileHdlr.Upload)
				bots.GET("/:id/files", fileHdlr.GetAll)
				bots.GET("/:id/files/:file_id", fileHdlr.GetByID)
				bots.DELETE("/:id/files/:file_id", fileHdlr.Delete)
			}
		}
	}

	if err := r.Run(":" + cfg.App.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
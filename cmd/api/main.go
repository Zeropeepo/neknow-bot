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
			}
		}
	}

	if err := r.Run(":" + cfg.App.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

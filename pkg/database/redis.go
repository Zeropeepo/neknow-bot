package database

import(
	"context"
	"fmt"

	"github.com/Zeropeepo/neknow-bot/pkg/config"
	"github.com/Zeropeepo/neknow-bot/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewRedis(cfg *config.Config) *redis.Client{
	client := redis.NewClient(&redis.Options{
		Addr:		fmt.Sprint("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: 	cfg.Redis.Password,
		DB:	0,

	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("failed to connect to redis", zap.Error(err))
	}
	
	logger.Info("Sucessfully connected to redis")
	return client
}
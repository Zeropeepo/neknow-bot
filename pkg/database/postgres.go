package database

import (
	"fmt"

	"github.com/Zeropeepo/neknow-bot/pkg/config"
	"github.com/Zeropeepo/neknow-bot/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewPostgres(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Jakarta",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
	)

	gormCfg := &gorm.Config{}

	if cfg.App.Env == "production"{
		gormCfg.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	}

	db, err := gorm.Open(postgres.Open(dsn), gormCfg)
	if err != nil{
		logger.Fatal("failed connect to postgres", zap.Error(err))
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil{
		logger.Fatal("failed to get sql.DB", zap.Error(err))
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	logger.Info("postgres connected successfully")
	return db, nil
}
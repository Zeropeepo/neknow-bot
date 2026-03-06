package logger
import(
	"os"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger
func Init(env string){
	var cfg zap.Config

	if env == "production"{
		cfg = zap.NewProductionConfig()
	} else{
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	var err error
	log, err = cfg.Build()
	if err != nil{
		os.Exit(1)
	}
}



func Get() *zap.Logger{
	if log == nil{
		log, _ = zap.NewDevelopment()
	}
	return log
}

func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}
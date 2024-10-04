package log

var (
	level  Level
	logger *Logger
)

func SetDefault(l *Logger) {
	logger = l
}

func Debug(message string, a ...any) error {
	return logger.Log(LevelDebug, message, a)
}

func Info(message string, a ...any) error {
	return logger.Log(LevelInfo, message, a)
}

func Warn(message string, a ...any) error {
	return logger.Log(LevelWarn, message, a)
}

func Error(message string, a ...any) error {
	return logger.Log(LevelError, message, a)
}

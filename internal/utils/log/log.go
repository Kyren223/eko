package log

var (
	level  Level
	logger *Logger
)

func SetDefault(l *Logger) {
	logger = l
}

func Debug(message string, a ...any) error {
	return logger.Debug(message, a...)
}

func Info(message string, a ...any) error {
	return logger.Info(message, a...)
}

func Warn(message string, a ...any) error {
	return logger.Warn(message, a...)
}

func Error(message string, a ...any) error {
	return logger.Error(message, a...)
}

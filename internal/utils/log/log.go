package log

var logger *Logger

func SetDefault(l *Logger) {
	logger = l
}

func Debug(message string, a ...any) {
	_ = logger.Debug(message, a...)
}

func Info(message string, a ...any) {
	_ = logger.Info(message, a...)
}

func Warn(message string, a ...any) {
	_ = logger.Warn(message, a...)
}

func Error(message string, a ...any) {
	_ = logger.Error(message, a...)
}

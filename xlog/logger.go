package xlog

import (
	"fmt"
	"io"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DefaultHeader = "[${level}]${prefix}[${short_file}:${line}]"
)

var (
	globalZapLogger *zap.Logger
	globalMutex     sync.RWMutex
	headerFormat    string = DefaultHeader
)

// Logger defines the logging interface
type Logger interface {
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Warn(args ...interface{})
	Warnf(template string, args ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(template string, args ...interface{})
	With(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	Name() string
}

type zapLogger struct {
	zap   *zap.Logger
	sugar *zap.SugaredLogger
	name  string
}

// Initialize global zap logger
func init() {
	initGlobalLogger()
}

func initGlobalLogger() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	logger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize zap logger: %v", err))
	}

	globalMutex.Lock()
	globalZapLogger = logger
	globalMutex.Unlock()
}

func SetHeader(name string) {
	globalMutex.Lock()
	headerFormat = name
	globalMutex.Unlock()
}

// Logger interface implementation
func (l *zapLogger) Debug(args ...interface{}) {
	l.sugar.Debug(args...)
}

func (l *zapLogger) Debugf(template string, args ...interface{}) {
	l.sugar.Debugf(template, args...)
}

func (l *zapLogger) Info(args ...interface{}) {
	l.sugar.Info(args...)
}

func (l *zapLogger) Infof(template string, args ...interface{}) {
	l.sugar.Infof(template, args...)
}

func (l *zapLogger) Warn(args ...interface{}) {
	l.sugar.Warn(args...)
}

func (l *zapLogger) Warnf(template string, args ...interface{}) {
	l.sugar.Warnf(template, args...)
}

func (l *zapLogger) Error(args ...interface{}) {
	l.sugar.Error(args...)
}

func (l *zapLogger) Errorf(template string, args ...interface{}) {
	l.sugar.Errorf(template, args...)
}

func (l *zapLogger) Fatal(args ...interface{}) {
	l.sugar.Fatal(args...)
}

func (l *zapLogger) Fatalf(template string, args ...interface{}) {
	l.sugar.Fatalf(template, args...)
}

func (l *zapLogger) With(key string, value interface{}) Logger {
	newLogger := l.zap.With(zap.Any(key, value))
	return &zapLogger{
		zap:   newLogger,
		sugar: newLogger.Sugar(),
		name:  l.name,
	}
}

func (l *zapLogger) WithFields(fields map[string]interface{}) Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	newLogger := l.zap.With(zapFields...)
	return &zapLogger{
		zap:   newLogger,
		sugar: newLogger.Sugar(),
		name:  l.name,
	}
}

func (l *zapLogger) Name() string {
	return l.name
}

// Factory functions
func NewLogger(name string) Logger {
	globalMutex.RLock()
	namedLogger := globalZapLogger.Named(name)
	globalMutex.RUnlock()

	return &zapLogger{
		zap:   namedLogger,
		sugar: namedLogger.Sugar(),
		name:  name,
	}
}

func WithChildName(name string, parent Logger) Logger {
	if zl, ok := parent.(*zapLogger); ok {
		childZap := zl.zap.Named(name)
		childName := fmt.Sprintf("%s-%s", zl.name, name)

		return &zapLogger{
			zap:   childZap,
			sugar: childZap.Sugar(),
			name:  childName,
		}
	}

	// Fallback for non-zap loggers
	return NewLogger(fmt.Sprintf("%s-%s", parent.Name(), name))
}

// Setup file and console logging
func SetupFileLogging(baseDir, fileName string) error {
	logFile, err := CreateLogFile(baseDir, fileName)
	if err != nil {
		return err
	}

	// Create multi-writer for both console and file
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Configure zap with multi-writer
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(multiWriter),
		zapcore.DebugLevel,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	globalMutex.Lock()
	if globalZapLogger != nil {
		globalZapLogger.Sync()
	}
	globalZapLogger = logger
	globalMutex.Unlock()

	return nil
}

// Get the underlying zap logger for advanced usage
func GetZapLogger() *zap.Logger {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalZapLogger
}

// Sync flushes any buffered log entries
func Sync() error {
	globalMutex.RLock()
	logger := globalZapLogger
	globalMutex.RUnlock()

	if logger != nil {
		return logger.Sync()
	}
	return nil
}

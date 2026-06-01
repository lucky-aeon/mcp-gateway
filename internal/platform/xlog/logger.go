package xlog

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

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
	sinkDispatcher  *asyncSinkDispatcher
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
	zap    *zap.Logger
	sugar  *zap.SugaredLogger
	name   string
	fields map[string]interface{}
}

type Entry struct {
	Timestamp time.Time
	Level     string
	Logger    string
	Message   string
	Fields    map[string]interface{}
}

type Sink interface {
	Write(Entry)
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
	if sinkDispatcher == nil {
		sinkDispatcher = newAsyncSinkDispatcher(1024)
	}
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
	l.emit("debug", fmt.Sprint(args...))
}

func (l *zapLogger) Debugf(template string, args ...interface{}) {
	l.sugar.Debugf(template, args...)
	l.emit("debug", fmt.Sprintf(template, args...))
}

func (l *zapLogger) Info(args ...interface{}) {
	l.sugar.Info(args...)
	l.emit("info", fmt.Sprint(args...))
}

func (l *zapLogger) Infof(template string, args ...interface{}) {
	l.sugar.Infof(template, args...)
	l.emit("info", fmt.Sprintf(template, args...))
}

func (l *zapLogger) Warn(args ...interface{}) {
	l.sugar.Warn(args...)
	l.emit("warn", fmt.Sprint(args...))
}

func (l *zapLogger) Warnf(template string, args ...interface{}) {
	l.sugar.Warnf(template, args...)
	l.emit("warn", fmt.Sprintf(template, args...))
}

func (l *zapLogger) Error(args ...interface{}) {
	l.sugar.Error(args...)
	l.emit("error", fmt.Sprint(args...))
}

func (l *zapLogger) Errorf(template string, args ...interface{}) {
	l.sugar.Errorf(template, args...)
	l.emit("error", fmt.Sprintf(template, args...))
}

func (l *zapLogger) Fatal(args ...interface{}) {
	l.sugar.Fatal(args...)
}

func (l *zapLogger) Fatalf(template string, args ...interface{}) {
	l.sugar.Fatalf(template, args...)
}

func (l *zapLogger) With(key string, value interface{}) Logger {
	newLogger := l.zap.With(zap.Any(key, value))
	fields := cloneFields(l.fields)
	fields[key] = value
	return &zapLogger{
		zap:    newLogger,
		sugar:  newLogger.Sugar(),
		name:   l.name,
		fields: fields,
	}
}

func (l *zapLogger) WithFields(fields map[string]interface{}) Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	newLogger := l.zap.With(zapFields...)
	fieldsCopy := cloneFields(l.fields)
	for k, v := range fields {
		fieldsCopy[k] = v
	}
	return &zapLogger{
		zap:    newLogger,
		sugar:  newLogger.Sugar(),
		name:   l.name,
		fields: fieldsCopy,
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
		zap:    namedLogger,
		sugar:  namedLogger.Sugar(),
		name:   name,
		fields: map[string]interface{}{},
	}
}

func WithChildName(name string, parent Logger) Logger {
	if zl, ok := parent.(*zapLogger); ok {
		childZap := zl.zap.Named(name)
		childName := fmt.Sprintf("%s-%s", zl.name, name)

		return &zapLogger{
			zap:    childZap,
			sugar:  childZap.Sugar(),
			name:   childName,
			fields: cloneFields(zl.fields),
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

func RegisterSink(sink Sink) {
	if sink == nil {
		return
	}
	globalMutex.Lock()
	if sinkDispatcher == nil {
		sinkDispatcher = newAsyncSinkDispatcher(1024)
	}
	sinkDispatcher.add(sink)
	globalMutex.Unlock()
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

func (l *zapLogger) emit(level, message string) {
	if l == nil {
		return
	}
	globalMutex.RLock()
	dispatcher := sinkDispatcher
	globalMutex.RUnlock()
	if dispatcher == nil {
		return
	}
	dispatcher.publish(Entry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Logger:    l.name,
		Message:   message,
		Fields:    cloneFields(l.fields),
	})
}

func cloneFields(fields map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		out[k] = v
	}
	return out
}

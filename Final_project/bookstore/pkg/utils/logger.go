// pkg/utils/logger.go

package utils

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
}

type LogConfig struct {
	LogDir     string
	LogFile    string
	Debug      bool
	TimeFormat string
}

func NewLogger(config LogConfig) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// Open log file
	logPath := filepath.Join(config.LogDir, config.LogFile)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Create multi-writer for both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Create formatters for different log levels
	timeFormat := config.TimeFormat
	if timeFormat == "" {
		timeFormat = "2006/01/02 15:04:05"
	}

	flags := log.Ldate | log.Ltime
	if config.Debug {
		flags |= log.Llongfile
	}

	return &Logger{
		infoLogger: log.New(multiWriter,
			fmt.Sprintf("\033[36m%s\033[0m ", "INFO "),
			flags,
		),
		errorLogger: log.New(multiWriter,
			fmt.Sprintf("\033[31m%s\033[0m ", "ERROR"),
			flags,
		),
		debugLogger: log.New(multiWriter,
			fmt.Sprintf("\033[33m%s\033[0m ", "DEBUG"),
			flags,
		),
	}, nil
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.log(l.infoLogger, format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.log(l.errorLogger, format, v...)
}

func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(l.debugLogger, format, v...)
}

func (l *Logger) log(logger *log.Logger, format string, v ...interface{}) {
	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if ok {
		// Extract just the file name without the full path
		file = filepath.Base(file)
		prefix := fmt.Sprintf("[%s:%d] ", file, line)
		logger.Printf(prefix+format, v...)
	} else {
		logger.Printf(format, v...)
	}
}

// HTTP request logging middleware
func (l *Logger) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response wrapper to capture the status code
		wrapped := wrapResponseWriter(w)

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		l.Info("HTTP %s %s %d %v",
			r.Method,
			r.URL.Path,
			wrapped.status,
			duration,
		)
	})
}

// responseWriter wrapper to capture status code
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(buf []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(buf)
}

// Helper functions for common logging patterns

func (l *Logger) LogAPIRequest(method, path string, status int, duration time.Duration) {
	l.Info("API Request: %s %s - Status: %d - Duration: %v",
		method, path, status, duration)
}

func (l *Logger) LogDatabaseOperation(operation, entity string, duration time.Duration, err error) {
	if err != nil {
		l.Error("DB Operation: %s %s - Error: %v - Duration: %v",
			operation, entity, err, duration)
	} else {
		l.Debug("DB Operation: %s %s - Duration: %v",
			operation, entity, duration)
	}
}

func (l *Logger) LogReportGeneration(reportType string, start, end time.Time, err error) {
	if err != nil {
		l.Error("Report Generation: %s (%v to %v) - Error: %v",
			reportType, start.Format("2006-01-02"), end.Format("2006-01-02"), err)
	} else {
		l.Info("Report Generation: %s (%v to %v) - Success",
			reportType, start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
}

// Transaction logging
func (l *Logger) LogTransaction(transactionType string, details map[string]interface{}, err error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Transaction: %s - ", transactionType))

	for k, v := range details {
		sb.WriteString(fmt.Sprintf("%s: %v, ", k, v))
	}

	if err != nil {
		l.Error("%s - Error: %v", sb.String(), err)
	} else {
		l.Info("%s - Success", sb.String())
	}
}

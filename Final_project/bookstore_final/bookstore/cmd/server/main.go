package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"bookstore/internal/handlers"
	"bookstore/internal/reports"
	"bookstore/internal/store"
	"bookstore/pkg/utils"
)

const (
	defaultPort       = 8080
	defaultLogDir     = "logs"
	defaultReportsDir = "output-reports"

	// Currently 24 * time.Minute; adjust as you like (e.g. 24 * time.Hour).
	reportInterval = 24 * time.Minute
)

func main() {
	// Command-line flags
	port := flag.Int("port", defaultPort, "Server port number")

	authorsPath := flag.String("authors", "authors.json", "Path to authors file")
	booksPath := flag.String("books", "books.json", "Path to books file")
	customersPath := flag.String("customers", "customers.json", "Path to customers file")
	ordersPath := flag.String("orders", "orders.json", "Path to orders file")
	reportsPath := flag.String("reports", "reports.json", "Path to reports file")

	logDir := flag.String("logdir", defaultLogDir, "Directory for log files")
	reportsDir := flag.String("reportsdir", defaultReportsDir, "Directory for sales reports")
	flag.Parse()

	// Ensure directories exist
	if err := ensureDir(*logDir); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}
	if err := ensureDir(*reportsDir); err != nil {
		log.Fatalf("Failed to create reports directory: %v", err)
	}

	// Initialize logger
	logger, err := utils.NewLogger(utils.LogConfig{
		LogDir:     *logDir,
		LogFile:    "bookstore.log",
		Debug:      true,
		TimeFormat: "2006/01/02 15:04:05",
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize stores
	bookStore, err := store.NewBookStore(*booksPath)
	if err != nil {
		logger.Error("Failed to initialize book store: %v", err)
		os.Exit(1)
	}

	authorStore, err := store.NewAuthorStore(*authorsPath)
	if err != nil {
		logger.Error("Failed to initialize author store: %v", err)
		os.Exit(1)
	}

	customerStore, err := store.NewCustomerStore(*customersPath)
	if err != nil {
		logger.Error("Failed to initialize customer store: %v", err)
		os.Exit(1)
	}

	orderStore, err := store.NewOrderStore(*ordersPath)
	if err != nil {
		logger.Error("Failed to initialize order store: %v", err)
		os.Exit(1)
	}

	reportStore, err := store.NewReportStore(*reportsPath)
	if err != nil {
		logger.Error("Failed to initialize report store: %v", err)
		os.Exit(1)
	}

	// Handlers
	bookHandler := handlers.NewBookHandler(bookStore)

	// IMPORTANT: Pass BOTH authorStore (which implements interfaces.AuthorStore)
	// AND bookStore (which implements interfaces.BookStore) to the new AuthorHandler
	authorHandler := handlers.NewAuthorHandler(authorStore, bookStore)

	customerHandler := handlers.NewCustomerHandler(customerStore)
	orderHandler := handlers.NewOrderHandler(orderStore, bookStore)
	reportHandler := handlers.NewReportHandler(reportStore)

	// Sales reporter
	salesReporter, err := reports.NewSalesReporter(
		orderStore,
		reportStore,
		*reportsDir,
		reportInterval,
	)
	if err != nil {
		logger.Error("Failed to initialize sales reporter: %v", err)
		os.Exit(1)
	}

	// Router
	router := mux.NewRouter()
	apiRouter := router.PathPrefix("/api").Subrouter()

	// Register all handlers
	bookHandler.RegisterRoutes(apiRouter)
	authorHandler.RegisterRoutes(apiRouter)
	customerHandler.RegisterRoutes(apiRouter)
	orderHandler.RegisterRoutes(apiRouter)
	reportHandler.RegisterRoutes(apiRouter)

	// CORS
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Simple ping route
	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	}).Methods("GET")

	// Setup HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      corsMiddleware.Handler(logger.HTTPMiddleware(router)),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  80 * time.Second,
	}

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start sales reporter in a goroutine
	go func() {
		logger.Info("Starting sales reporter...")
		if err := salesReporter.Start(ctx); err != nil {
			logger.Error("Sales reporter failed: %v", err)
		}
	}()

	// Start the server in another goroutine
	go func() {
		logger.Info("Starting server on port %d...", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed: %v", err)
			os.Exit(1)
		}
	}()

	// Listen for OS signals (Ctrl+C, etc.)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	logger.Info("Received signal: %v. Initiating shutdown...", sig)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop sales reporter
	salesReporter.Stop()

	// Shutdown the server gracefully
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed: %v", err)
	}

	logger.Info("Server shutdown complete")
}

func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}
	return nil
}
func initDB() (*sql.DB, error) {
    connStr := "postgres://user:password@localhost:5432/bookstore?sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }
    // Ping to verify
    if err = db.Ping(); err != nil {
        return nil, err
    }
    return db, nil
}

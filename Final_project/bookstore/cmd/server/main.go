
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

	_ "github.com/lib/pq"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"bookstore/internal/auth"
	"bookstore/internal/handlers"
	"bookstore/internal/reports"
	"bookstore/internal/store"
	"bookstore/pkg/utils"
)

const (
	defaultPort       = 8080
	defaultLogDir     = "logs"
	defaultReportsDir = "output-reports"

	reportInterval = 24 * time.Minute
)

func main() {
	
	port := flag.Int("port", defaultPort, "Server port number")
	logDir := flag.String("logdir", defaultLogDir, "Directory for log files")
	reportsDir := flag.String("reportsdir", defaultReportsDir, "Directory for sales reports")
	flag.Parse()


	if err := ensureDir(*logDir); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}
	if err := ensureDir(*reportsDir); err != nil {
		log.Fatalf("Failed to create reports directory: %v", err)
	}

	
	logger, err := utils.NewLogger(utils.LogConfig{
		LogDir:     *logDir,
		LogFile:    "bookstore.log",
		Debug:      true,
		TimeFormat: "2006/01/02 15:04:05",
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	
	db, err := initDB()
	if err != nil {
		logger.Error("Failed to connect to Postgres: %v", err)
		os.Exit(1)
	}


	if err := migrateDB(db); err != nil {
		logger.Error("Failed to run migrations: %v", err)
		os.Exit(1)
	}

	
	bookStore, _ := store.NewPostgresBookStore(db)
	authorStore, _ := store.NewPostgresAuthorStore(db)
	customerStore, _ := store.NewPostgresCustomerStore(db)
	orderStore, _ := store.NewPostgresOrderStore(db)
	reportStore, _ := store.NewPostgresReportStore(db)

	
	bookHandler := handlers.NewBookHandler(bookStore)
	authorHandler := handlers.NewAuthorHandler(authorStore, bookStore)
	customerHandler := handlers.NewCustomerHandler(customerStore)
	orderHandler := handlers.NewOrderHandler(orderStore, bookStore)
	reportHandler := handlers.NewReportHandler(reportStore)

	
	jwtManager := auth.NewJWTManager("MY_SECRET_KEY", 24*time.Hour)
	authHandler := handlers.NewAuthHandler(jwtManager)

	
	salesReporter, err := reports.NewSalesReporter(
		orderStore,
		reportStore,
		*reportsDir,
		reportInterval,
		bookStore,
	)
	if err != nil {
		logger.Error("Failed to initialize sales reporter: %v", err)
		os.Exit(1)
	}


	router := mux.NewRouter()
	apiRouter := router.PathPrefix("/api").Subrouter()

	
	authHandler.RegisterRoutes(apiRouter)

	
	jwtMiddleware := auth.NewAuthMiddleware(jwtManager)

	
	bookHandler.RegisterRoutes(apiRouter, jwtMiddleware.Middleware)
	authorHandler.RegisterRoutes(apiRouter, jwtMiddleware.Middleware)
	customerHandler.RegisterRoutes(apiRouter, jwtMiddleware.Middleware)
	orderHandler.RegisterRoutes(apiRouter, jwtMiddleware.Middleware)
	reportHandler.RegisterRoutes(apiRouter, jwtMiddleware.Middleware)

	
	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	}).Methods("GET")

	
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      corsMiddleware.Handler(logger.HTTPMiddleware(router)),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  80 * time.Second,
	}

	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		logger.Info("Starting sales reporter...")
		if err := salesReporter.Start(ctx); err != nil {
			logger.Error("Sales reporter failed: %v", err)
		}
	}()

	go func() {
		logger.Info("Starting server on port %d...", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed: %v", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	logger.Info("Received signal: %v. Initiating shutdown...", sig)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	
	salesReporter.Stop()

	
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed: %v", err)
	}
	logger.Info("Server shutdown complete")
}

func initDB() (*sql.DB, error) {
	connStr := "postgres://postgres:secret@localhost:5432/bookstore?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func migrateDB(db *sql.DB) error {
	schema := `
    CREATE TABLE IF NOT EXISTS authors (
        id SERIAL PRIMARY KEY,
        first_name TEXT NOT NULL,
        last_name TEXT NOT NULL,
        bio TEXT
    );

    CREATE TABLE IF NOT EXISTS books (
        id SERIAL PRIMARY KEY,
        title TEXT NOT NULL,
        author_id INT NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
        published_at TIMESTAMP,
        price NUMERIC(12,2) NOT NULL,
        stock INT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS customers (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        email TEXT NOT NULL,
        street TEXT,
        city TEXT,
        state TEXT,
        postal_code TEXT,
        country TEXT,
        created_at TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS orders (
        id SERIAL PRIMARY KEY,
        customer_id INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
        total_price NUMERIC(12,2),
        created_at TIMESTAMP,
        status TEXT
    );

    CREATE TABLE IF NOT EXISTS order_items (
        id SERIAL PRIMARY KEY,
        order_id INT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
        book_id INT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
        quantity INT NOT NULL
    );
    `
	_, err := db.Exec(schema)
	return err
}

func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}
	return nil
}

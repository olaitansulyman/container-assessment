package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	port := envOrDefault("PORT", "8080")
	databaseName := envOrDefault("MONGO_DATABASE", "muchtodo")
	mongoURI := os.Getenv("MONGO_URI")

	if mongoURI == "" {
		log.Fatal("MONGO_URI must be set")
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(
		startupCtx,
		options.Client().
			ApplyURI(mongoURI).
			SetServerSelectionTimeout(5*time.Second),
	)
	if err != nil {
		log.Fatalf("connect to MongoDB: %v", err)
	}

	defer func() {
		ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()

		if err := client.Disconnect(ctx); err != nil {
			log.Printf("disconnect MongoDB: %v", err)
		}
	}()

	if err := client.Ping(startupCtx, nil); err != nil {
		log.Fatalf("MongoDB is not reachable: %v", err)
	}

	repository := NewMongoTodoRepository(
		client.Database(databaseName).Collection("todos"),
	)
	handler := NewAPIHandler(repository)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.health)
	mux.HandleFunc("GET /ready", handler.ready)
	mux.HandleFunc("GET /todos", handler.listTodos)
	mux.HandleFunc("POST /todos", handler.createTodo)
	mux.HandleFunc("GET /todos/{id}", handler.getTodo)
	mux.HandleFunc("PUT /todos/{id}", handler.updateTodo)
	mux.HandleFunc("DELETE /todos/{id}", handler.deleteTodo)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("MuchTodo API listening on :%s", port)
		serverErrors <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server: %v", err)
		}
	case <-stop:
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s",
			r.Method,
			r.URL.Path,
			time.Since(started).Round(time.Millisecond),
		)
	})
}

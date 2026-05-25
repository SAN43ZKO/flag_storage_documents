package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"storage_files/internal/config"
	"storage_files/internal/db"
	"storage_files/internal/handler"
	"storage_files/internal/repository"
	"storage_files/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	// Миграции
	migrationsDir := "migrations"
	if err := db.RunMigrations(ctx, pool, migrationsDir); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	// Убедимся, что папка uploads существует
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatalf("create upload dir: %v", err)
	}

	fileRepo := repository.NewFileRepo(pool)
	fileSvc := service.NewFileService(fileRepo, cfg.UploadDir, cfg.PublicURL)
	fileHandler := handler.NewFileHandler(fileSvc)

	mux := http.NewServeMux()
	fileHandler.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("starting server on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down...")
	shutdownCtx, cancelShut := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShut()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("stopped")
}

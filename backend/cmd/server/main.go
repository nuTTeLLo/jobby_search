package main

import (
	"fmt"
	"job-tracker-backend/internal/config"
	"job-tracker-backend/internal/domain"
	"job-tracker-backend/internal/handler"
	"job-tracker-backend/internal/repository"
	"job-tracker-backend/internal/service"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

func main() {
	cfg := config.Load()

	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	db, err := gorm.Open(sqlite.Open("file:"+cfg.DatabasePath+"?_pragma=foreign_keys=ON&driver=sqlite"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&domain.Job{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	jobService := service.NewJobService(jobRepo, cfg.MCPServerURL)
	jobHandler := handler.NewJobHandler(jobService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Mount("/api/jobs", jobHandler.Routes())

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

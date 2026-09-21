package main

import (
	"fmt"
	"job-tracker-backend/internal/auth"
	"job-tracker-backend/internal/config"
	"job-tracker-backend/internal/domain"
	"job-tracker-backend/internal/handler"
	appMiddleware "job-tracker-backend/internal/middleware"
	"job-tracker-backend/internal/repository"
	"job-tracker-backend/internal/service"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET env var must be set")
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate with user_id nullable first to allow backfill
	if err := db.AutoMigrate(&domain.User{}, &domain.Job{}, &domain.Attachment{}, &domain.DiscoveredJob{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed nuttello@gmail.com and backfill existing jobs to that user
	if err := seedAndBackfill(db, cfg); err != nil {
		log.Fatalf("Failed to seed user and backfill jobs: %v", err)
	}

	// Jobs applied to before applied_at existed have no stamp, so seed one from
	// created_at — for jobs added by the application workflow the two are the
	// same day. Only fills nulls, so it is a no-op on every later boot.
	if err := db.Exec(
		"UPDATE jobs SET applied_at = created_at WHERE status = ? AND applied_at IS NULL",
		string(domain.StatusApplied),
	).Error; err != nil {
		log.Fatalf("Failed to backfill jobs.applied_at: %v", err)
	}

	// Now enforce NOT NULL on user_id
	if err := db.Exec("ALTER TABLE jobs ALTER COLUMN user_id SET NOT NULL").Error; err != nil {
		log.Fatalf("Failed to add NOT NULL constraint to jobs.user_id: %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	discoveredRepo := repository.NewDiscoveredJobRepository(db)
	jobService := service.NewJobService(jobRepo, cfg.MCPServerURL)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiration)
	discoveredService := service.NewDiscoveredJobService(discoveredRepo)
	jobHandler := handler.NewJobHandler(jobService, cfg.JWTSecret)
	attachmentHandler := handler.NewAttachmentHandler(jobService, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(authService)
	discoveredHandler := handler.NewDiscoveredJobHandler(discoveredService)

	authMW := appMiddleware.Authenticate(cfg.JWTSecret)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   strings.Split(cfg.AllowedOrigins, ","),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "Content-Disposition"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Public auth routes
	r.Mount("/api/auth", authHandler.PublicRoutes())

	// Viewing an attachment in a browser tab is a plain navigation, which can
	// carry no Authorization header, so this route authenticates itself from a
	// short-lived single-attachment token in the query string.
	r.Get("/api/attachments/{id}/view", attachmentHandler.ViewAttachment)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMW)
		r.Post("/api/auth/change-password", authHandler.ChangePassword)
		r.Mount("/api/jobs", jobHandler.Routes())
		r.Mount("/api/jobs/{id}/attachments", attachmentHandler.Routes())
		r.Mount("/api/discovered-jobs", discoveredHandler.Routes())
	})

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func seedAndBackfill(db *gorm.DB, cfg *config.Config) error {
	const seedEmail = "nuttello@gmail.com"

	// Check if seed user already exists
	var existing domain.User
	err := db.Where("email = ?", seedEmail).First(&existing).Error
	if err == nil {
		// User already exists — backfill any jobs that may still have empty user_id
		db.Exec("UPDATE jobs SET user_id = ? WHERE user_id IS NULL OR user_id = ''", existing.ID)
		return nil
	}

	// First boot: require seed password
	if cfg.SeedUserPassword == "" {
		log.Fatal("SEED_USER_PASSWORD env var must be set on first boot to create the initial user account")
	}

	hash, err := auth.HashPassword(cfg.SeedUserPassword)
	if err != nil {
		return fmt.Errorf("failed to hash seed password: %w", err)
	}

	seedUser := &domain.User{
		Email:        seedEmail,
		PasswordHash: hash,
	}
	if err := db.Create(seedUser).Error; err != nil {
		return fmt.Errorf("failed to create seed user: %w", err)
	}

	// Backfill all existing jobs to this user
	db.Exec("UPDATE jobs SET user_id = ? WHERE user_id IS NULL OR user_id = ''", seedUser.ID)
	log.Printf("Seeded user %s (id: %s) and backfilled existing jobs", seedEmail, seedUser.ID)
	return nil
}

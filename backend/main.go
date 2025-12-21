package main

import (
	"log"
	"net/http"
	"time"

	"github.com/GHutch55/fragments/backend/api/v1/database"
	"github.com/GHutch55/fragments/backend/api/v1/handlers"
	"github.com/GHutch55/fragments/backend/api/v1/middleware"
	"github.com/GHutch55/fragments/backend/config"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

func main() {
	log.Println("1. Starting application...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("2. Config loaded successfully")

	// JWT secret is required - fail fast if not provided
	jwtSecret := cfg.JWTSecret
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters long")
	}

	pool, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("5. Database connected successfully")

	// Create middleware and handlers
	authMiddleware := middleware.NewAuthMiddleware(pool, jwtSecret)
	userHandler := &handlers.UserHandler{DB: pool}
	snippetHandler := &handlers.SnippetHandler{DB: pool}
	folderHandler := &handlers.FolderHandler{DB: pool}
	authHandler := handlers.NewAuthHandler(pool, authMiddleware)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.RequestSize(10 << 20))     // 10 mb limit
	r.Use(chimiddleware.Timeout(60 * time.Second)) // 1 minute timeout
	r.Use(chimiddleware.Compress(5))
	r.Use(chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173", "127.0.0.1:5555", "fragments-7gas.onrender.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/", handlers.HomeHandler)
	r.Get("/health", handlers.HealthHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/", handlers.ApiInfoHandler)

		// Auth routes with rate limiting
		r.Route("/auth", func(r chi.Router) {
			// Rate limit auth endpoints: 5 requests per minute per IP
			r.Use(httprate.LimitByIP(5, 1*time.Minute))

			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)

			// Protected auth routes (no rate limiting needed - already authenticated)
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireAuth)
				r.Get("/me", authHandler.Me)
				r.Post("/change-password", authHandler.ChangePassword)
			})
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)

			// User routes - restricted to own user only
			r.Route("/users", func(r chi.Router) {
				r.Get("/me", userHandler.GetCurrentUser)
				r.Put("/me", userHandler.UpdateCurrentUser)
				r.Delete("/me", userHandler.DeleteCurrentUser)
			})

			// Snippet routes
			r.Route("/snippets", func(r chi.Router) {
				r.Post("/", snippetHandler.CreateSnippet)
				r.Get("/{id}", snippetHandler.GetSnippet)
				r.Get("/", snippetHandler.GetSnippets)
				r.Delete("/{id}", snippetHandler.DeleteSnippet)
				r.Put("/{id}", snippetHandler.UpdateSnippet)
			})

			r.Route("/folders", func(r chi.Router) {
				r.Post("/", folderHandler.CreateFolder)
				r.Get("/{id}", folderHandler.GetFolder)
				r.Get("/", folderHandler.GetFolders)
				r.Delete("/{id}", folderHandler.DeleteFolder)
				r.Put("/{id}", folderHandler.UpdateFolder)
			})
		})
	})

	log.Printf("Starting server on port %s", cfg.Port)
	err = http.ListenAndServe(":"+cfg.Port, r)
	if err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

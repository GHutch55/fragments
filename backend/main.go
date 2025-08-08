package main

import (
	"log"
	"net/http"

	"github.com/GHutch55/fragments/backend/api/v1/database"
	"github.com/GHutch55/fragments/backend/api/v1/handlers"
	"github.com/GHutch55/fragments/backend/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create handler with database connection
	userHandler := &handlers.UserHandler{DB: db}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/", handlers.HomeHandler)
	r.Get("/health", handlers.HealthHandler)

	r.Route("/api", func(r chi.Router) {
		r.Get("/", handlers.ApiInfoHandler)

		// User routes
		r.Route("/users", func(r chi.Router) {
			r.Post("/", userHandler.CreateUser)
			r.Get("/{id}", userHandler.GetUser)
			r.Get("/", userHandler.GetUsers)

			// Future user routes:
			// r.Put("/{id}", userHandler.UpdateUser)  // PUT /api/users/123
			// r.Delete("/{id}", userHandler.DeleteUser) // DELETE /api/users/123
		})

		// Future routes for other resources:
		// r.Route("/snippets", func(r chi.Router) { ... })
		// r.Route("/folders", func(r chi.Router) { ... })
		// r.Route("/tags", func(r chi.Router) { ... })
	})

	log.Printf("Starting server on port %s", cfg.Port)
	err = http.ListenAndServe(":"+cfg.Port, r)
	if err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

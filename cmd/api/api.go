package main

import (
	"log"
	"log/slog"
	"net/http"
	"time"

	repo "github.com/JeffreyOmoakah/Auth-session.git/internal/adapters/postgresql/sqlc"
	"github.com/JeffreyOmoakah/Auth-session.git/internal/users"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
)

// api.go
type application struct {
    config config
    db     *pgxpool.Pool
    redis  *redis.Client   // added
    logger *slog.Logger
}

type config struct {
    addr  string
    db    dbConfig
    redis redisConfig         // added
}

type dbConfig struct {
    dsn string
}

type redisConfig struct {
    addr     string
    password string
}

func (app *application) mount() http.Handler {
    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.Timeout(60 * time.Second))

    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("all good"))
    })

    userRepo := repo.New(app.db)
    userService := users.NewService(userRepo, app.db, app.redis, app.logger)  // redis injected
    userHandler := users.NewHandler(userService, app.logger)

    r.Route("/v1", func(r chi.Router) {
        r.Route("/auth-sessions", func(r chi.Router) {
            r.Post("/signup", userHandler.Signup)
            r.Post("/login", userHandler.Login)
            r.Delete("/logout", userHandler.Logout)  // was missing from routes
        })
    })
    
    // Protected routes go here — everything inside gets the middleware
    r.Group(func(r chi.Router) {
        r.Use(userHandler.SessionMiddleware)
        r.Get("/me", userHandler.GetMe)
    })
        

    return r
}

func (app *application) run(h http.Handler) error {
    srv := &http.Server{
        Addr:         app.config.addr,
        Handler:      h,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  time.Minute,
    }

    log.Printf("Starting server on %s", app.config.addr)
    return srv.ListenAndServe()
}
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/JeffreyOmoakah/Auth-session.git/internal/env"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// main.go
func main() {
    _ = godotenv.Load()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    port := env.GetString("PORT", "3000")
    cfg := config{
        addr: ":" + port,
        db: dbConfig{
            dsn: env.GetString("DATABASE_URL", ""),
        },
        redis: redisConfig{
            addr:     env.GetString("REDIS_ADDR", "localhost:6379"),
            password: env.GetString("REDIS_PASSWORD", ""),
        },
    }

    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    // Postgres pool
    pool, err := pgxpool.New(ctx, cfg.db.dsn)
    if err != nil {
        logger.Error("unable to connect to database pool", "error", err)
        os.Exit(1)
    }
    defer pool.Close()
    logger.Info("database connection pool established")


    // Redis client
    rdb := redis.NewClient(&redis.Options{
        Addr:     cfg.redis.addr,
        Password: cfg.redis.password,
    })
    if err := rdb.Ping(ctx).Err(); err != nil {
        logger.Error("unable to connect to redis", "error", err)
        os.Exit(1)
    }
    defer rdb.Close()
    logger.Info("redis connection established")

    api := application{
        config: cfg,
        db:     pool,
        redis:  rdb,        // now part of the app
        logger: logger,
    }

    logger.Info("server starting", "addr", cfg.addr)
    if err := api.run(api.mount()); err != nil {
        logger.Error("server crashed", "error", err)
        os.Exit(1)
    }
}
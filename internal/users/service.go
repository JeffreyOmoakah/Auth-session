package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	repo "github.com/JeffreyOmoakah/Auth-session.git/internal/adapters/postgresql/sqlc"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrCredentialsRequired = errors.New("credentials required")
)

type svc struct {
	repo *repo.Queries
	db   *pgxpool.Pool
	redis *redis.Client
	logger *slog.Logger
}

type Service interface {
	Signup(ctx context.Context, tempSignupReq createSignupReq ) (repo.User, error)
	Login(ctx context.Context, req loginReq) (string, error)  // returns sessionID not user
	Logout(ctx context.Context, sessionID string) error
    ValidateSession(ctx context.Context, sessionID string) (repo.User, error)
}

func NewService(repo *repo.Queries, db *pgxpool.Pool, redis *redis.Client, logger *slog.Logger) Service {
	return &svc{
		repo: repo,
		db:   db,
		redis:  redis,   
        logger: logger,
	}
}

func generateSessionID() (string, error) {
    b := make([]byte, 32)
    _, err := rand.Read(b)
    if err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}

func (s *svc) createSession(ctx context.Context, user repo.GetUserByEmailRow) (string, error) {
    sessionID, err := generateSessionID()
    if err != nil {
        return "", err
    }

    session := sessions{
        ID:        sessionID,
        UserID:    user.ID.String(),
        Email:     user.Email,
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(24 * time.Hour),
    }

    data, err := json.Marshal(session)
    if err != nil {
        return "", err
    }

    // Key pattern:  session:<sessionID>
    err = s.redis.Set(ctx, "session:"+sessionID, data, 24*time.Hour).Err()
    if err != nil {
        return "", err
    }

    return sessionID, nil
}

func (s *svc) ValidateSession(ctx context.Context, sessionID string) (repo.User, error) {
    data, err := s.redis.Get(ctx, "session:"+sessionID).Bytes()
    if err == redis.Nil {
        return repo.User{}, errors.New("session not found or expired")
    }
    if err != nil {
        return repo.User{}, fmt.Errorf("redis error: %w", err)
    }

    var session sessions
    if err := json.Unmarshal(data, &session); err != nil {
        return repo.User{}, err
    }

    if time.Now().After(session.ExpiresAt) {
        s.redis.Del(ctx, "session:"+sessionID)
        return repo.User{}, errors.New("session expired")
    }

    return repo.User{
        Email: session.Email,
    }, nil
}

func (s *svc) Logout(ctx context.Context, sessionID string) error {
    return s.redis.Del(ctx, "session:"+sessionID).Err()
}

func (s *svc) Signup(ctx context.Context, tempSignupReq createSignupReq) (repo.User, error) {
	// VALIDATE THE PAYLOAD 
	if tempSignupReq.Email == "" || tempSignupReq.Password == "" {
        return repo.User{}, ErrCredentialsRequired
    }
	// Hash Password 
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempSignupReq.Password), bcrypt.DefaultCost)
    if err != nil {
        return repo.User{}, fmt.Errorf("failed to hash password: %w", err)
    }
	
	tx, err := s.db.Begin(ctx)
	if err != nil { 
		return repo.User{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.repo.WithTx(tx)
	
	
	// LOOK IF USER ALREADY EXISTS 
	row, err := qtx.CreateUser(ctx, repo.CreateUserParams{
        Email:    tempSignupReq.Email,
        Password: string(hashedPassword),
    })
	
	if err != nil {
        return repo.User{}, err 
    }
    
    newUser := repo.User{
            ID:        row.ID,
            Email:     row.Email,
            Password:  string(hashedPassword),
            CreatedAt: row.CreatedAt,
        }
    // Commit
    if err := tx.Commit(ctx); err != nil {
            return repo.User{}, err
        }
    
    return newUser, nil
}

func (s *svc) Login(ctx context.Context, req loginReq) (string, error) {
    row, err := s.repo.GetUserByEmail(ctx, req.Email)
    if err != nil {
        return "", errors.New("invalid email or password")
    }

    err = bcrypt.CompareHashAndPassword([]byte(row.Password), []byte(req.Password))
    if err != nil {
        return "", errors.New("invalid email or password")
    }

    // Credentials valid — create session and return the ID
    sessionID, err := s.createSession(ctx, row)
    if err != nil {
        return "", fmt.Errorf("failed to create session: %w", err)
    }

    return sessionID, nil
}

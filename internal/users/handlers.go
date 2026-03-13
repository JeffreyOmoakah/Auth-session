package users

import (
	"log/slog"
	"net/http"

	"github.com/JeffreyOmoakah/Auth-session.git/internal/json"
)

type handler struct {
    service Service
    logger  *slog.Logger
}

func NewHandler(service Service, logger *slog.Logger) *handler {
    return &handler{
        service: service,
        logger:  logger,   
    }
}

func (h *handler) Signup(w http.ResponseWriter, r *http.Request) {
    var tempSignupReq createSignupReq
    if err := json.Read(r, &tempSignupReq); err != nil {
        h.logger.Error("failed to read signup body", "error", err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    newUser, err := h.service.Signup(r.Context(), tempSignupReq)
    if err != nil {
        if err == ErrCredentialsRequired {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        h.logger.Error("signup failed", "error", err)
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }

    json.Write(w, http.StatusCreated, newUser)
}

func (h *handler) Login(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request body
    var req loginReq
    if err := json.Read(r, &req); err != nil {
        h.logger.Error("failed to read login body", "error", err)
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    // 2. Call service — returns sessionID not a user
    sessionID, err := h.service.Login(r.Context(), req)
    if err != nil {
        h.logger.Warn("login attempt failed", "email", req.Email, "error", err)
        http.Error(w, "invalid email or password", http.StatusUnauthorized)
        return
    }

    // 3. Set the session cookie — this is the entire point
    http.SetCookie(w, &http.Cookie{
        Name:     "session_id",
        Value:    sessionID,
        HttpOnly: true,                       // JS cannot read this
        Secure:   true,                       // HTTPS only
        SameSite: http.SameSiteStrictMode,    // blocks CSRF
        Path:     "/",
        MaxAge:   86400,                      // 24 hours, matches Redis TTL
    })

    // 4. Return minimal response — never return the session ID in the body
    json.Write(w, http.StatusOK, map[string]string{
        "message": "login successful",
    })
}

func (h *handler) Logout(w http.ResponseWriter, r *http.Request) {
    // 1. Extract the cookie
    cookie, err := r.Cookie("session_id")
    if err != nil {
        http.Error(w, "no active session", http.StatusUnauthorized)
        return
    }

    // 2. Delete from Redis via service
    if err := h.service.Logout(r.Context(), cookie.Value); err != nil {
        h.logger.Error("logout failed", "error", err)
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }

    // 3. Clear the cookie on the client
    http.SetCookie(w, &http.Cookie{
        Name:     "session_id",
        Value:    "",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        Path:     "/",
        MaxAge:   -1,    // tells browser to delete it immediately
    })

    json.Write(w, http.StatusOK, map[string]string{
        "message": "logged out",
    })
}

func (h *handler) GetMe(w http.ResponseWriter, r *http.Request) {
    session, ok := GetSessionFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    json.Write(w, http.StatusOK, map[string]string{
        "user_id": session.UserID,
        "email":   session.Email,
    })
}
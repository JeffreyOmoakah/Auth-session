package users

import (
	"context"
	"net/http"
)

type contextKey string

const sessionContextKey contextKey = "session"

func (h *handler) SessionMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Extract cookie
        cookie, err := r.Cookie("session_id")
        if err != nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        // 2. Validate session against Redis
        session, err := h.service.ValidateSession(r.Context(), cookie.Value)
        if err != nil {
            // Clear the dead cookie on the client
            http.SetCookie(w, &http.Cookie{
                Name:     "session_id",
                Value:    "",
                HttpOnly: true,
                Secure:   true,
                SameSite: http.SameSiteStrictMode,
                Path:     "/",
                MaxAge:   -1,
            })
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        // 3. Inject session into context
        ctx := context.WithValue(r.Context(), sessionContextKey, session)

        // 4. Pass to next handler
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// GetSessionFromContext — handlers call this to pull session out
func GetSessionFromContext(ctx context.Context) (sessions, bool) {
    session, ok := ctx.Value(sessionContextKey).(sessions)
    return session, ok
}
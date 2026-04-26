package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/service"
)

type AuthContextUseCase interface {
	Authenticate(ctx context.Context, token string) (models.User, models.UserSession, error)
}

type authContextKey string

const userContextKey authContextKey = "auth_user"

type AuthMiddleware struct {
	useCase AuthContextUseCase
}

func NewAuthMiddleware(useCase AuthContextUseCase) *AuthMiddleware {
	return &AuthMiddleware{useCase: useCase}
}

func (m *AuthMiddleware) RequireAuthenticated(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authUser, _, err := m.authenticateRequest(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), userContextKey, authUser)))
	}
}

func (m *AuthMiddleware) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authUser, _, err := m.authenticateRequest(r)
		if err != nil {
			if acceptsHTML(r) {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		if authUser.Role != "admin" {
			if acceptsHTML(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), userContextKey, authUser)))
	}
}

func (m *AuthMiddleware) authenticateRequest(r *http.Request) (models.User, models.UserSession, error) {
	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" && allowsQueryToken(r) {
		token = r.URL.Query().Get("access_token")
	}
	if token == "" {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			token = cookie.Value
		}
	}

	user, session, err := m.useCase.Authenticate(r.Context(), token)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			return models.User{}, models.UserSession{}, service.ErrUnauthorized
		}
		return models.User{}, models.UserSession{}, err
	}

	return user, session, nil
}

func userFromContext(ctx context.Context) (models.User, bool) {
	user, ok := ctx.Value(userContextKey).(models.User)
	return user, ok
}

func acceptsHTML(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	return accept == "" ||
		accept == "*/*" ||
		strings.Contains(accept, "text/html") ||
		r.Header.Get("Sec-Fetch-Dest") == "document"
}

func allowsQueryToken(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}

	switch r.URL.Path {
	case "/api/v1/labels/qr", "/labels/print", "/labels/pdf":
		return true
	default:
		return false
	}
}

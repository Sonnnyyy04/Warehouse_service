package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/service"
)

type AuthUseCase interface {
	Login(ctx context.Context, login, password string) (models.User, models.UserSession, error)
	Authenticate(ctx context.Context, token string) (models.User, models.UserSession, error)
	Logout(ctx context.Context, token string) error
}

type AuthHandler struct {
	useCase AuthUseCase
}

func NewAuthHandler(useCase AuthUseCase) *AuthHandler {
	return &AuthHandler{useCase: useCase}
}

type loginRequest struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User      models.User `json:"user"`
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
}

func (h *AuthHandler) APILogin(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req loginRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	login := req.Login
	if login == "" {
		login = req.Email
	}

	user, session, err := h.useCase.Login(ctx, login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		User:      user,
		Token:     session.Token,
		ExpiresAt: session.ExpiresAt,
	})
}

func (h *AuthHandler) APIGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	authUser, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]models.User{"user": authUser})
}

func (h *AuthHandler) APILogout(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.useCase.Logout(ctx, token); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func extractBearerToken(headerValue string) string {
	if headerValue == "" {
		return ""
	}

	parts := strings.SplitN(headerValue, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/service"
)

const sessionCookieName = "warehouse_session"

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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

func (h *AuthHandler) WebLoginPage(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.New("login").Parse(loginPageTemplate))

	data := struct {
		Error string
	}{
		Error: r.URL.Query().Get("error"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.Execute(w, data); err != nil {
		http.Error(w, "failed to render login page", http.StatusInternalServerError)
	}
}

func (h *AuthHandler) WebLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/login?error=invalid+form", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	login := r.FormValue("login")
	if login == "" {
		login = r.FormValue("email")
	}

	user, session, err := h.useCase.Login(ctx, login, r.FormValue("password"))
	if err != nil {
		http.Redirect(w, r, "/login?error=invalid+credentials", http.StatusSeeOther)
		return
	}

	if user.Role != "admin" {
		http.Redirect(w, r, "/login?error=admin+access+required", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *AuthHandler) WebLogout(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_ = h.useCase.Logout(ctx, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
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

const loginPageTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Warehouse Login</title>
  <style>
    :root {
      color-scheme: light;
      --ink: #172033;
      --muted: #637083;
      --paper: #ffffff;
      --line: #d8dde6;
      --accent: #0f766e;
      --danger: #b42318;
      --bg: linear-gradient(180deg, #f2f0ea 0%, #eef6f4 100%);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      background: var(--bg);
      color: var(--ink);
      font-family: Arial, sans-serif;
      padding: 20px;
    }
    .card {
      width: 100%;
      max-width: 420px;
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: 28px;
      box-shadow: 0 18px 40px rgba(23, 32, 51, 0.08);
    }
    h1 {
      margin: 0 0 10px;
      font-size: 30px;
    }
    p {
      margin: 0;
      color: var(--muted);
      line-height: 1.5;
    }
    label {
      display: block;
      margin: 16px 0 8px;
      font-size: 14px;
      font-weight: 700;
    }
    input {
      width: 100%;
      min-height: 48px;
      border: 1px solid var(--line);
      border-radius: 14px;
      padding: 0 14px;
      font: inherit;
    }
    button {
      width: 100%;
      min-height: 48px;
      border: none;
      border-radius: 14px;
      margin-top: 18px;
      background: var(--accent);
      color: white;
      font: inherit;
      font-weight: 700;
      cursor: pointer;
    }
    .error {
      margin-top: 14px;
      padding: 12px 14px;
      border-radius: 14px;
      border: 1px solid #fecdd3;
      background: #fff1f2;
      color: var(--danger);
      font-weight: 700;
    }
    .hint {
      margin-top: 14px;
      font-size: 14px;
    }
  </style>
</head>
<body>
  <form class="card" method="post" action="/login">
    <h1>Вход администратора</h1>
    <p>Веб-панель администрирования склада доступна только после входа.</p>

    <label for="login">Логин</label>
    <input id="login" type="text" name="login" placeholder="admin" required />

    <label for="password">Пароль</label>
    <input id="password" type="password" name="password" placeholder="admin123" required />

    <button type="submit">Войти</button>

    {{if .Error}}
    <div class="error">{{.Error}}</div>
    {{end}}

    <p class="hint">Demo admin: <strong>admin</strong> / <strong>admin123</strong></p>
  </form>
</body>
</html>`

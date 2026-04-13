package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
)

type AuthUserRepository interface {
	GetByLogin(ctx context.Context, login string) (models.User, error)
	GetByID(ctx context.Context, id int64) (models.User, error)
}

type AuthSessionRepository interface {
	Create(ctx context.Context, token string, userID int64, expiresAt time.Time) (models.UserSession, error)
	GetByToken(ctx context.Context, token string) (models.UserSession, error)
	Touch(ctx context.Context, token string, lastSeenAt time.Time) error
	DeleteByToken(ctx context.Context, token string) error
}

type AuthService struct {
	userRepo        AuthUserRepository
	sessionRepo     AuthSessionRepository
	sessionDuration time.Duration
}

func NewAuthService(userRepo AuthUserRepository, sessionRepo AuthSessionRepository) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		sessionRepo:     sessionRepo,
		sessionDuration: 30 * 24 * time.Hour,
	}
}

func (s *AuthService) Login(ctx context.Context, login, password string) (models.User, models.UserSession, error) {
	login = strings.TrimSpace(strings.ToLower(login))
	password = strings.TrimSpace(password)
	if login == "" || password == "" {
		return models.User{}, models.UserSession{}, ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.User{}, models.UserSession{}, ErrInvalidCredentials
		}
		return models.User{}, models.UserSession{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return models.User{}, models.UserSession{}, ErrInvalidCredentials
	}

	token, err := generateSessionToken()
	if err != nil {
		return models.User{}, models.UserSession{}, err
	}

	session, err := s.sessionRepo.Create(ctx, token, user.ID, time.Now().Add(s.sessionDuration))
	if err != nil {
		return models.User{}, models.UserSession{}, err
	}

	return sanitizeUser(user), session, nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (models.User, models.UserSession, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return models.User{}, models.UserSession{}, ErrUnauthorized
	}

	session, err := s.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.User{}, models.UserSession{}, ErrUnauthorized
		}
		return models.User{}, models.UserSession{}, err
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.sessionRepo.DeleteByToken(ctx, token)
		return models.User{}, models.UserSession{}, ErrUnauthorized
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.User{}, models.UserSession{}, ErrUnauthorized
		}
		return models.User{}, models.UserSession{}, err
	}

	_ = s.sessionRepo.Touch(ctx, token, time.Now())

	return sanitizeUser(user), session, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrUnauthorized
	}

	if err := s.sessionRepo.DeleteByToken(ctx, token); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUnauthorized
		}
		return err
	}

	return nil
}

func sanitizeUser(user models.User) models.User {
	user.PasswordHash = ""
	return user
}

func generateSessionToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(tokenBytes), nil
}

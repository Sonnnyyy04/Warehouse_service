package models

import "time"

type UserSession struct {
	ID         int64     `json:"id"`
	Token      string    `json:"token"`
	UserID     int64     `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

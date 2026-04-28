package models

type User struct {
	ID           int64  `json:"id"`
	Login        string `json:"login"`
	Email        string `json:"email"`
	FullName     string `json:"full_name"`
	Role         string `json:"role"`
	IsSuperAdmin bool   `json:"is_super_admin"`
	PasswordHash string `json:"-"`
}

package models

type UserSummary struct {
	ID       int64  `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

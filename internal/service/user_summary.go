package service

import "Warehouse_service/internal/models"

func UserSummaryFromUser(user models.User) *models.UserSummary {
	return &models.UserSummary{
		ID:           user.ID,
		Login:        user.Login,
		FullName:     user.FullName,
		Role:         user.Role,
		IsSuperAdmin: user.IsSuperAdmin,
	}
}

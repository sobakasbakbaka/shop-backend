package utils

import (
	"errors"
	"gobackend/models"

	"gorm.io/gorm"
)

func GetUserIDBySessionToken(db *gorm.DB, token string) (uint, error) {
    var session models.Session
    if err := db.Where("token = ?", token).First(&session).Error; err != nil {
        return 0, errors.New("invalid or expired session token")
    }
    return session.UserID, nil
}
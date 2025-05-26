package models

import (
	"time"
)

type CartItem struct {
	ID uint `gorm:"primaryKey"`
	UserID uint `gorm:"not null"`
	ProductID uint `gorm:"not null"`
	Quantity uint `gorm:"default:1"`
	Product Product `gorm:"foreignKey:ProductID"`
	CreatedAt time.Time
}
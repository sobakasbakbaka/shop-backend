package models

import (
	"time"
)

type CartItem struct {
	ID uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"not null" json:"user_id"`
	ProductID uint `gorm:"not null" json:"product_id"`
	Quantity uint `gorm:"default:1" json:"quantity"`
	Product Product `gorm:"foreignKey:ProductID" json:"products"`
	CreatedAt time.Time `json:"created_at"`
}
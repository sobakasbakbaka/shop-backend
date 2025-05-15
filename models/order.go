package models

import "time"

type Order struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"not null" json:"user_id"`
	User      User       `gorm:"foreignKey:UserID" json:"-"`
	Total     float64    `json:"total"`
	Products  []Product  `gorm:"many2many:order_products;" json:"products"`
	CreatedAt time.Time  `json:"created_at"`
}
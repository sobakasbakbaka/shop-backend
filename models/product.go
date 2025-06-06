package models

import "time"

type Product struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ImageURL  string  `json:"image_url"`
	Description string `json:"description"`
	Stock     uint      `json:"stock"`
	CreatedAt time.Time `json:"created_at"`
}
package models

import "time"

type Order struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	UserID    uint        `gorm:"not null" json:"user_id"`
	User      User        `gorm:"foreignKey:UserID" json:"-"`
	Total     float64     `json:"total"`
	Items     []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
	CreatedAt time.Time   `json:"created_at"`
	FirstName string	`json:"first_name"`
	LastName  string 	`json:"last_name"`
	Street   string 	`json:"street"`
	City      string 	`json:"city"`
	Phone 	string	`json:"phone"`
}
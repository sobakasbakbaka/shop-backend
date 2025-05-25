package models

import "time"

type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Email	string `gorm:"unique;not null" json:"email"`
	PasswordHash string `gorm:"not null" json:"-"`
	UserName string `gorm:"not null" json:"user_name"`
	Role    string `gorm:"default:user" json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
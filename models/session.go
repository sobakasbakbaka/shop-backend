package models

import "time"

type Session struct {
    ID        uint      `gorm:"primaryKey"`
    Token     string    `gorm:"uniqueIndex"`
    UserID    uint
    CreatedAt time.Time
}
package models

type OrderItem struct {
	ID        uint    `gorm:"primaryKey"`
	OrderID   uint    `gorm:"not null"`
	ProductID uint    `gorm:"not null"`
	Product   Product
	Quantity  uint    `gorm:"not null"`
	Price     float64 `gorm:"not null"`
}
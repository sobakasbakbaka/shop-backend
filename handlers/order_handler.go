package handlers

import (
	"gobackend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	DB *gorm.DB
}

func NewOrderHandler(db *gorm.DB) *OrderHandler {
	return &OrderHandler{DB: db}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	userID := userIDInterface.(uint)

	var input struct {
		Total float64 `json:"total"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	order := models.Order{
		UserID: userID,
		Total:  input.Total,
	}

	if err := h.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать заказ"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Заказ создан", "order_id": order.ID})
}
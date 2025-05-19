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
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	userIDFloat, ok := userIDRaw.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Невалидный user_id"})
		return
	}
	userID := uint(userIDFloat)

	var input struct {
		ProductIDs []uint `json:"product_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil || len(input.ProductIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нужно передать product_ids"})
		return
	}

	var products []models.Product
	if err := h.DB.Where("id IN ?", input.ProductIDs).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения товаров"})
		return
	}

	var total float64
	for _, p := range products {
		total += p.Price
	}

	order := models.Order{
		UserID:   userID,
		Products: products,
		Total:    total,
	}

	if err := h.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания заказа"})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) GetMyOrders(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userID := uint(userIDRaw.(float64))

	var orders []models.Order
	if err := h.DB.Preload("Products").Where("user_id = ?", userID).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения заказов"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

func (h *OrderHandler) GetAllOrders(c *gin.Context) {
	var orders []models.Order
	if err := h.DB.Preload("Products").Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения заказов"})
		return
	}

	c.JSON(http.StatusOK, orders)
}
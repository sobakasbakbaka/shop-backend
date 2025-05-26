package handlers

import (
	"gobackend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CartHandler struct {
	DB *gorm.DB
}

func NewCartHandler(db *gorm.DB) *CartHandler {
	return &CartHandler{DB: db}
}

func (h *CartHandler) AddToCart(c *gin.Context) {
	var input struct {
		PtoductID uint `json:"product_id"`
		Quantity uint `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Quantity == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "неверный ввод"})
		return
	}

	userIDRaw, _ := c.Get("user_id")
	userID := uint(userIDRaw.(float64))

	var item models.CartItem
	err := h.DB.
		Where("user_id = ? AND product_id = ?", userID, input.PtoductID).
		First(&item).
		Error
	if err == nil {
		item.Quantity += input.Quantity
		h.DB.Save(&item)
	} else {
		h.DB.Create(&models.CartItem{
			UserID: userID,
			ProductID: input.PtoductID,
			Quantity: input.Quantity,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Товар добавлен в корзину"})
}

func (h *CartHandler) GetCart(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userId := uint(userIDRaw.(float64))

	var items []models.CartItem
	h.DB.Preload("Product").Where("user_id = ?", userId).Find(&items)

	c.JSON(http.StatusOK, items)
}

func (h *CartHandler) RemoveFromCart(c *gin.Context) {
	productID := c.Param("product_id")
	userIDRaw, _ := c.Get("user_id")
	userId := uint(userIDRaw.(float64))

	h.DB.Where("user_id = ? AND product_id = ?", userId, productID).Delete(&models.CartItem{})

	c.JSON(http.StatusOK, gin.H{"message": "Товар удален"})
}

func (h *CartHandler) ClearCart(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userId := uint(userIDRaw.(float64))

	h.DB.Where("user_id = ?", userId).Delete(&models.CartItem{})

	c.JSON(http.StatusOK, gin.H{"message": "Корзина отчищена"})
}
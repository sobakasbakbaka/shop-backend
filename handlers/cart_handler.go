package handlers

import (
	"gobackend/models"
	"gobackend/utils"
	"net/http"
	"strconv"

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

func (h *CartHandler) UpdateQuantity(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	productID, err := strconv.ParseUint(c.Param("product_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный product_id"})
		return
	}

	var input struct {
		Quantity uint `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Quantity == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректное количество"})
		return
	}

	var cartItem models.CartItem
	err = h.DB.Where("user_id = ? AND product_id = ?", userID, productID).First(&cartItem).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Товар не найден в корзине"})
		return
	}

	cartItem.Quantity = input.Quantity
	if err := h.DB.Save(&cartItem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Количество обновлено"})
}

func (h *CartHandler) Checkout(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var cartItems []models.CartItem
	if err := h.DB.Preload("Product").Where("user_id = ?", userID).Find(&cartItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка полученяи корзины"})
		return
	}
	if len(cartItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Корзина пуста"})
		return
	}

	total := 0.0
	for _, item := range cartItems {
		total += float64(item.Quantity) * item.Product.Price
	}

	order := models.Order{
		UserID: userID,
		Total: total,
	}
	if err := h.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания заказа"})
		return
	}

	for _, item := range cartItems {
		h.DB.Model(&order).Association("Products").Append(&item.Product)
	}

	h.DB.Where("user_id = ?", userID).Delete(&models.CartItem{})

	c.JSON(http.StatusOK, gin.H{"message": "Заказ создан", "order_id": order.ID})
}
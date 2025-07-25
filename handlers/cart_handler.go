package handlers

import (
	"encoding/json"
	"errors"
	"gobackend/models"
	"gobackend/utils"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/stripe/stripe-go/v78"
	checkoutsession "github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/webhook"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CartHandler struct {
	DB *gorm.DB
}

func NewCartHandler(db *gorm.DB) *CartHandler {
	return &CartHandler{DB: db}
}

func (h *CartHandler) AddToCart(c *gin.Context) {
	var input struct {
		ProductID uint `json:"product_id"`
		Quantity  uint `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Quantity == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "неверный ввод"})
		return
	}

	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var product models.Product
	if err := h.DB.First(&product, input.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Продукт не найден"})
		return
	}

	var item models.CartItem
	err = h.DB.Where("user_id = ? AND product_id = ?", userID, input.ProductID).First(&item).Error

	var newQuantity uint
	if err == nil {
		newQuantity = item.Quantity + input.Quantity
	} else {
		newQuantity = input.Quantity
	}

	if newQuantity > product.Stock {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Недостаточно товара на складе"})
		return
	}

	if err == nil {
		item.Quantity = newQuantity
		if err := h.DB.Save(&item).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления корзины"})
			return
		}
	} else {
		newItem := models.CartItem{
			UserID:    userID,
			ProductID: input.ProductID,
			Quantity:  input.Quantity,
		}
		if err := h.DB.Create(&newItem).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка добавления в корзину"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Товар добавлен в корзину"})
}

func (h *CartHandler) GetCart(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var items []models.CartItem
	if err := h.DB.Preload("Product").Where("user_id = ?", userID).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения корзины"})
		return
	}

	var total float64
	for _, item := range items {
		total += float64(item.Quantity) * item.Product.Price
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
	})
}

func (h *CartHandler) RemoveFromCart(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	productID := c.Param("product_id")
	h.DB.Where("user_id = ? AND product_id = ?", userID, productID).Delete(&models.CartItem{})

	c.JSON(http.StatusOK, gin.H{"message": "Товар удален"})
}

func (h *CartHandler) ClearCart(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.DB.Where("user_id = ?", userID).Delete(&models.CartItem{})

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

func (h *CartHandler) HandleStripeWebhook(c *gin.Context) {
	const MaxBodyBytes = int64(65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Println("Ошибка чтения тела запроса:", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Ошибка чтения тела запроса"})
		return
	}

	//TODO: Для отладки без проверки подписи (например, в Insomnia/Postman)
	if os.Getenv("DISABLE_STRIPE_SIGNATURE_CHECK") == "true" {
		var event stripe.Event
		if err := json.Unmarshal(payload, &event); err != nil {
			log.Println("Ошибка парсинга события Stripe:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка парсинга события"})
			return
		}
		h.handleEvent(c, event)
		return
	}

	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	sigHeader := c.GetHeader("Stripe-Signature")

	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, endpointSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		log.Println("Ошибка проверки подписи Stripe:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка проверки подписи Stripe"})
		return
	}

	h.handleEvent(c, event)
}

func (h *CartHandler) handleEvent(c *gin.Context, event stripe.Event) {
	log.Printf("Получено событие от Stripe: %s", event.Type)

	if event.Type == "checkout.session.completed" {
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Println("Ошибка парсинга session:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка парсинга session"})
			return
		}

		log.Printf("Stripe session ID: %s", session.ID)
		log.Printf("ClientReferenceID: %s", session.ClientReferenceID)
		log.Printf("PaymentStatus: %s", session.PaymentStatus)

		if session.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
			log.Println("Оплата не завершена, заказ не создается")
			c.Status(http.StatusOK)
			return
		}

		uid64, err := strconv.ParseUint(session.ClientReferenceID, 10, 64)
		if err != nil {
			log.Println("Invalid user ID в session.ClientReferenceID:", session.ClientReferenceID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		userID := uint(uid64)

		log.Printf("Создание заказа для user_id=%d\n", userID)

		err = h.DB.Transaction(func(tx *gorm.DB) error {
			return CreateOrderFromCart(tx, userID, session.Metadata)
		})
		if err != nil {
			log.Println("Ошибка создания заказа из webhook:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания заказа"})
			return
		}

		log.Println("Заказ успешно создан")
	}

	c.Status(http.StatusOK)
}

func CreateOrderFromCart(tx *gorm.DB, userID uint, metadata map[string]string) error {
	var cartItems []models.CartItem
	if err := tx.Preload("Product").Where("user_id = ?", userID).Find(&cartItems).Error; err != nil {
		return err
	}

	if len(cartItems) == 0 {
		return errors.New("корзина пуста")
	}

	var total float64
	var orderItems []models.OrderItem

	for _, item := range cartItems {
		var product models.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&product, item.ProductID).Error; err != nil {
			return err
		}

		if item.Quantity > product.Stock {
			return errors.New("недостаточно товара на складе: " + product.Name)
		}

		product.Stock -= item.Quantity
		if err := tx.Save(&product).Error; err != nil {
			return err
		}

		total += float64(item.Quantity) * product.Price
		orderItems = append(orderItems, models.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     product.Price,
		})
	}

	order := models.Order{
		UserID:    userID,
		FirstName: metadata["first_name"],
		LastName:  metadata["last_name"],
		Street:   metadata["street"],
		City:      metadata["city"],
		Phone: 		metadata["phone"],
		Items:     orderItems,
		Total:     total,
	}

	if err := tx.Create(&order).Error; err != nil {
		return err
	}

	if err := tx.Where("user_id = ?", userID).Delete(&models.CartItem{}).Error; err != nil {
		return err
	}

	return nil
}

func (h *CartHandler) CreateCheckoutSession(c *gin.Context) {
	tokenString, err := c.Cookie("token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session token not found"})
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["user_id"] == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user_id in token"})
		return
	}
	userID := uint(userIDFloat)

	var input models.Checkout
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ввод"})
		return
	}

	var cartItems []models.CartItem
	if err := h.DB.Preload("Product").Where("user_id = ?", userID).Find(&cartItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении корзины"})
		return
	}

	if len(cartItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Корзина пуста"})
		return
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stripe secret key is not configured"})
		return
	}

	var lineItems []*stripe.CheckoutSessionLineItemParams
	for _, item := range cartItems {
		priceCents := int64(item.Product.Price * 100)
		if priceCents <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректная цена в корзине"})
			return
		}
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String("usd"),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(item.Product.Name),
				},
				UnitAmount: stripe.Int64(priceCents),
			},
			Quantity: stripe.Int64(int64(item.Quantity)),
		})
	}

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems:          lineItems,
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:         stripe.String("http://localhost:3000/order?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:          stripe.String("http://localhost:3000/cart"),
		ClientReferenceID:  stripe.String(strconv.Itoa(int(userID))),
		Metadata: map[string]string{
			"first_name": input.FirstName,
			"last_name":  input.LastName,
			"city":       input.City,
			"street":    input.Street,
			"phone": input.Phone,
		},
	}

	session, err := checkoutsession.New(params)
	if err != nil {
		log.Printf("Ошибка создания сессии Stripe: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка Stripe при создании сессии"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessionId": session.ID})
}
package main

import (
	"gobackend/config"
	"gobackend/db"
	"gobackend/handlers"
	"gobackend/middleware"
	"gobackend/models"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()
	
	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatal("Ошибка подключения к БД", err)
	}

	err = database.AutoMigrate(
		&models.Product{}, 
		&models.User{}, 
		&models.Order{},
		&models.CartItem{},
	)
	if err != nil {
		log.Fatal("Ошибка миграции БД", err)
	}
	
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
	}))

	productHandler := handlers.NewProductHandler(database)
	authHandler := handlers.NewAuthHandler(database)
	orderHandler := handlers.NewOrderHandler(database)
	cartHandler := handlers.NewCartHandler(database)

	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)
	r.GET("/logout", authHandler.Logout)
	
	auth := r.Group("/")
	auth.Use(middleware.AuthRequired())

	auth.GET("/me", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		var user models.User

		if err := database.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить данные пользователя"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"email": user.Email,
			"created_at": user.CreatedAt,
			"role": user.Role,
			"user_name": user.UserName,
		})
	})

	r.GET("/products", productHandler.GetProducts)
	r.GET("/products/:id", productHandler.GetProductByID)

	auth.POST("/products/:id/image", productHandler.UploadProductImage)

	admin := auth.Group("/")
	admin.Use(middleware.AdminOnly())

	admin.POST("/products", productHandler.CreateProduct)
	admin.PUT("/products/:id", productHandler.UpdateProduct)
	admin.DELETE("/products/:id", productHandler.DeleteProduct)
	
	
	auth.POST("/orders", orderHandler.CreateOrder)
	auth.GET("/orders/mine", orderHandler.GetMyOrders)
	admin.GET("/orders", orderHandler.GetAllOrders)

	cart := r.Group("/cart", middleware.AuthRequired())
	{
		cart.POST("/add", cartHandler.AddToCart)
		cart.GET("/", cartHandler.GetCart)
		cart.DELETE("/:product_id", cartHandler.RemoveFromCart)
		cart.DELETE("/", cartHandler.ClearCart)
	}

	err = r.Run(":" + cfg.ServerPort)
	if err != nil {
		log.Fatal("Ошибка запуска сервера", err)
	}
}
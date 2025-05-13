package main

import (
	"gobackend/config"
	"gobackend/db"
	"gobackend/handlers"
	"gobackend/middleware"
	"gobackend/models"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()
	
	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatal("Ошибка подключения к БД", err)
	}

	err = database.AutoMigrate(&models.Product{}, &models.User{})
	if err != nil {
		log.Fatal("Ошибка миграции БД", err)
	}
	
	r := gin.Default()
	productHandler := handlers.NewProductHandler(database)
	authHandler := handlers.NewAuthHandler(database)

	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)

	r.GET("/me", middleware.AuthRequired(), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		var user models.User
		if err := database.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить данные пользователя"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"email": user.Email, "created_at": user.CreatedAt})
	})

	r.GET("/products", productHandler.GetProducts)
	r.POST("/products", productHandler.CreateProduct)
	r.GET("/products/:id", productHandler.GetProductByID)
	r.PUT("/products/:id", productHandler.UpdateProduct)
	r.DELETE("/products/:id", productHandler.DeleteProduct)
	r.POST("/products/:id/image", productHandler.UploadProductImage)

	err = r.Run(":" + cfg.ServerPort)
	if err != nil {
		log.Fatal("Ошибка запуска сервера", err)
	}
}
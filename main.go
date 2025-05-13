package main

import (
	"gobackend/config"
	"gobackend/db"
	"gobackend/handlers"
	"gobackend/models"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()
	
	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatal("Ошибка подключения к БД", err)
	}

	err = database.AutoMigrate(&models.Product{})
	if err != nil {
		log.Fatal("Ошибка миграции БД", err)
	}
	
	r := gin.Default()
	productHandler := handlers.NewProductHandler(database)

	r.GET("/products", productHandler.GetProducts)
	r.POST("/products", productHandler.CreateProduct)

	err = r.Run(":" + cfg.ServerPort)
	if err != nil {
		log.Fatal("Ошибка запуска сервера", err)
	}
}
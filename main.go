package main

import (
	"gobackend/config"
	"gobackend/db"
	"gobackend/handlers"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()
	
	database, err := db.Connect(cfg)

	if err != nil {
		log.Fatal("Ошибка подключения к БД", err)
	}
	defer database.Close()

	r := gin.Default()

	productHandler := handlers.NewProductHandler(database)

	r.GET("/products", productHandler.GetProducts)
	r.POST("/products", productHandler.CreateProduct)

	r.Run(":" + cfg.ServerPort)
}
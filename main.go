package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Ошибка загрузки файла .env", err)
	}

	host := os.Getenv("DB_HOSTNAME")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
	host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		log.Fatal("Не удалось подключиться к БД", err)
	}

	defer db.Close()

	err = db.Ping()

	if err != nil {
		log.Fatal("БД не отвечает", err)
	}

	fmt.Println("Успешно подключено к БД")

	r := gin.Default()

	r.GET("/products", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, name, price FROM products")	

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при выполнении запроса"})
			return
		}

		defer rows.Close()

		var products []map[string]interface{}
		for rows.Next() {
			var id int
			var name string
			var price float64

			rows.Scan(&id, &name, &price)
			
			products = append(products, map[string]interface{}{
				"id": id,
				"name": name,
				"price": price,
			})
		}

		c.JSON(http.StatusOK, products)
	})

	serverPort := os.Getenv("SERVER_PORT")
	r.Run(":" + serverPort)
}
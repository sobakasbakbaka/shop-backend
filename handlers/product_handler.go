package handlers

import (
	"database/sql"
	"gobackend/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	DB *sql.DB
}

func NewProductHandler(db *sql.DB) *ProductHandler {
	return &ProductHandler{DB: db}
}

func (h *ProductHandler) GetProducts(c *gin.Context) {
	rows, err := h.DB.Query("SELECT id, name, price FROM products")

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса к БД"})
		return
	}

	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		rows.Scan(&p.ID, &p.Name, &p.Price)
		products = append(products, p)
	}

	c.JSON(http.StatusOK, products)
}
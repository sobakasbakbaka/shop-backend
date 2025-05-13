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

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var p models.Product

	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный JSON"})
		return
	}

	query := `INSERT INTO products (name, price) VALUES ($1, $2) RETURNING id`

	err := h.DB.QueryRow(query, p.Name, p.Price).Scan(&p.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка добавления товара"})
		return
	}

	c.JSON(http.StatusCreated, p)
}
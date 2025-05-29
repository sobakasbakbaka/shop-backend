package handlers

import (
	"bytes"
	"context"
	"fmt"
	"gobackend/models"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProductHandler struct {
	DB *gorm.DB
}

func NewProductHandler(db *gorm.DB) *ProductHandler {
	return &ProductHandler{DB: db}
}

func (h *ProductHandler) GetProducts(c *gin.Context) {
	var products []models.Product
	if err := h.DB.Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения продуктов"})
		return
	}

	c.JSON(http.StatusOK, products)
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невалидный JSON"})
		return
	}

	if err := h.DB.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания продукта"})
		return
	}
	
	c.JSON(http.StatusCreated, product)
}

func (h *ProductHandler) GetProductByID(c *gin.Context) {
	id := c.Param("id")
	var product models.Product
	if err := h.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Продукт не найден"})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	var product models.Product
	if err := h.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Продукт не найден"})
		return
	}

	var input models.Product
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невалидный JSON"})
		return
	}

	product.Name = input.Name
	product.Price = input.Price
	product.Stock = input.Stock
	product.Description = input.Description

	if err := h.DB.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления продукта"})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	var product models.Product

	if err := h.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Продукт не найден"})
		return
	}

	if err := h.DB.Delete(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления продукта"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Продукт удален"})
}

func (h *ProductHandler) UploadProductImage(c *gin.Context) {
	id := c.Param("id")

	var product models.Product
	if err := h.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Товар не найден"})
		return
	}

	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не передан"})
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Можно загружать только изображения"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось открыть файл"})
		return
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка чтения файла"})
		return
	}

	r2Endpoint := os.Getenv("S3_ENDPOINT")
	r2Bucket := os.Getenv("S3_BUCKET")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	publicDomain := os.Getenv("S3_PUBLIC_DOMAIN")

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               r2Endpoint,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		log.Println("Ошибка конфигурации R2:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка инициализации хранилища"})
		return
	}

	client := s3.NewFromConfig(cfg)
	key := fmt.Sprintf("%s/main.jpg", id)

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(r2Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		log.Println("Ошибка загрузки в R2:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка загрузки изображения"})
		return
	}

	imageURL := fmt.Sprintf("%s/%s", publicDomain, key)

	h.DB.Model(&product).Update("image_url", imageURL)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Изображение загружено",
		"image_url": imageURL,
	})
}
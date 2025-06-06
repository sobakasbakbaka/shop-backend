package handlers

import (
	"errors"
	"gobackend/models"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		UserName string `json:"user_name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невалидный JSON"})
		return
	}

	var existing models.User
	err := h.DB.Where("email = ?", input.Email).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пользователь с таким email уже существует"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке пользователя"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка хеширования пароля"})
		return
	}

	user := models.User{
		Email:        input.Email,
		PasswordHash: string(hash),
		UserName: input.UserName,
	}

	if err := h.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания пользователя"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания токена"})
		return
	}

	c.SetCookie("token", tokenString, 3600*24*3, "/", "", false, true)

	c.JSON(http.StatusCreated, gin.H{"message": "Пользователь зарегистрирован и авторизован"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невалидный JSON"})
		return
	}

	var user models.User
	if err := h.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role": user.Role,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания токена"})
		return
	}

	c.SetCookie(
		"token",
		tokenString,
		3600*24*3,
		"/",
		os.Getenv("CLIENT_ADRESS"),
		false, //HTTP true if https
		true, //HTTPOnly
	)

	c.JSON(http.StatusOK, gin.H{"message": "Авторизация успешна"})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie(
		"token",
		"", 
		-1, 
		"/", 
		os.Getenv("CLIENT_ADRESS"), 
		false,
		true,
	)
}
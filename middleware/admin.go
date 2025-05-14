package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context)  {
		userRole, exists := c.Get("user_role")
		if !exists || userRole != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Доступ запрещен"})
			c.Abort()
			return
		}
		c.Next()
	}
}